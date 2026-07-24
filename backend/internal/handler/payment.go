package handler

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"forward-panel/internal/config"
	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	paymentConfig   = &config.Config{}
	paymentClient   = &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	errRechargeRate = errors.New("recharge rate limited")
)

func ConfigurePayments(cfg *config.Config) {
	paymentConfig = cfg
}

func epayConfigured() bool {
	return paymentConfig.PublicURL != "" && paymentConfig.EpayGateway != "" && paymentConfig.EpayPID != "" && paymentConfig.EpayKey != ""
}

func codepayConfigured() bool {
	return paymentConfig.PublicURL != "" && paymentConfig.CodepayCreateURL != "" && paymentConfig.CodepayMerchantID != "" && paymentConfig.CodepayKey != ""
}

func ListRechargeProviders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		model.RechargeProviderEpay:    epayConfigured(),
		model.RechargeProviderCodepay: codepayConfigured(),
	})
}

func CreateRecharge(c *gin.Context) {
	key, ok := operationKey(c)
	if !ok {
		return
	}
	var input struct {
		Provider    string `json:"provider"`
		AmountCents int64  `json:"amount_cents"`
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || decoder.Decode(&struct{}{}) != io.EOF || input.AmountCents < 100 || input.AmountCents > maxBalanceAmount || !providerConfigured(input.Provider) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "充值参数或支付渠道无效"})
		return
	}

	userID := c.GetUint("user_id")
	requestHash := hashRequest(fmt.Sprintf("recharge-create:%d:%s:%d", userID, input.Provider, input.AmountCents))
	var existing model.RechargeOrder
	if err := model.DB.Where("user_id = ? AND idempotency_key = ?", userID, key).First(&existing).Error; err == nil {
		if existing.UserID != userID || existing.RequestHash != requestHash {
			c.JSON(http.StatusConflict, gin.H{"error": "Idempotency-Key已用于其他请求"})
			return
		}
		if existing.PayURL == "" {
			c.JSON(http.StatusConflict, gin.H{"error": "充值订单正在处理或渠道创建失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"trade_no": existing.TradeNo, "pay_url": existing.PayURL})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建充值订单失败"})
		return
	}
	order := model.RechargeOrder{
		UserID: userID, Provider: input.Provider, AmountCents: input.AmountCents,
		Status: model.RechargeStatusPending, IdempotencyKey: key, RequestHash: requestHash, CreatedAt: time.Now(),
	}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		lock := tx.Model(&model.User{}).Where("id = ?", userID).UpdateColumn("id", gorm.Expr("id"))
		if lock.Error != nil {
			return lock.Error
		}
		if lock.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		var recent int64
		if err := tx.Model(&model.RechargeOrder{}).Where("user_id = ? AND created_at >= ?", userID, time.Now().Add(-time.Hour)).Count(&recent).Error; err != nil {
			return err
		}
		if recent >= 10 {
			return errRechargeRate
		}
		for attempt := 0; attempt < 3; attempt++ {
			tradeNo, err := randomTradeNo()
			if err != nil {
				return err
			}
			order.TradeNo = tradeNo
			if err = tx.Create(&order).Error; err == nil {
				return nil
			}
			order.ID = 0
		}
		return errors.New("could not allocate trade number")
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在或已删除"})
		return
	}
	if errors.Is(err, errRechargeRate) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "充值请求过于频繁，请稍后再试"})
		return
	}
	if err != nil {
		if loadErr := model.DB.Where("user_id = ? AND idempotency_key = ?", userID, key).First(&existing).Error; loadErr == nil {
			if existing.UserID != userID || existing.RequestHash != requestHash {
				c.JSON(http.StatusConflict, gin.H{"error": "Idempotency-Key已用于其他请求"})
				return
			}
			if existing.PayURL != "" {
				c.JSON(http.StatusOK, gin.H{"trade_no": existing.TradeNo, "pay_url": existing.PayURL})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建充值订单失败"})
		return
	}

	payURL, err := createPayURL(order)
	if err != nil {
		model.DB.Model(&order).Update("status", model.RechargeStatusFailed)
		c.JSON(http.StatusBadGateway, gin.H{"error": "支付渠道暂不可用"})
		return
	}
	if err := model.DB.Model(&order).Update("pay_url", payURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存支付链接失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"trade_no": order.TradeNo, "pay_url": payURL})
}

func ListRechargeOrders(c *gin.Context) {
	var orders []model.RechargeOrder
	if err := model.DB.Where("user_id = ?", c.GetUint("user_id")).Order("id desc").Limit(commerceListLimit(c)).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询充值订单失败"})
		return
	}
	c.JSON(http.StatusOK, orders)
}

func EpayNotify(c *gin.Context) {
	if !epayConfigured() || c.Request.ParseForm() != nil {
		paymentResult(c, false)
		return
	}
	values := c.Request.Form
	valid := paramsPresent(values, "pid", "trade_status", "out_trade_no", "trade_no", "money", "sign") &&
		values.Get("pid") == paymentConfig.EpayPID && values.Get("trade_status") == "TRADE_SUCCESS" && verifyMD5(values, paymentConfig.EpayKey)
	if valid {
		valid = creditRecharge(model.RechargeProviderEpay, values.Get("out_trade_no"), values.Get("money"), values.Get("trade_no")) == nil
	}
	paymentResult(c, valid)
}

func CodepayNotify(c *gin.Context) {
	if !codepayConfigured() || c.Request.ParseForm() != nil {
		paymentResult(c, false)
		return
	}
	values := c.Request.Form
	valid := paramsPresent(values, "merchant_id", "out_trade_no", "amount", "trade_no", "status", "timestamp", "nonce", "sign") &&
		values.Get("merchant_id") == paymentConfig.CodepayMerchantID && values.Get("status") == "success" && verifyHMAC(values, paymentConfig.CodepayKey)
	if valid {
		valid = creditRecharge(model.RechargeProviderCodepay, values.Get("out_trade_no"), values.Get("amount"), values.Get("trade_no")) == nil
	}
	paymentResult(c, valid)
}

func PaymentReturn(c *gin.Context) {
	if paymentConfig.PublicURL == "" {
		c.Status(http.StatusNotFound)
		return
	}
	c.Redirect(http.StatusFound, strings.TrimRight(paymentConfig.PublicURL, "/")+"/plans")
}

func providerConfigured(provider string) bool {
	return (provider == model.RechargeProviderEpay && epayConfigured()) || (provider == model.RechargeProviderCodepay && codepayConfigured())
}

func randomTradeNo() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func createPayURL(order model.RechargeOrder) (string, error) {
	notifyURL := strings.TrimRight(paymentConfig.PublicURL, "/") + "/api/payment/" + order.Provider + "/notify"
	returnURL := strings.TrimRight(paymentConfig.PublicURL, "/") + "/api/payment/" + order.Provider + "/return"
	amount := formatCents(order.AmountCents)
	if order.Provider == model.RechargeProviderEpay {
		values := url.Values{
			"pid": {paymentConfig.EpayPID}, "out_trade_no": {order.TradeNo},
			"type": {paymentConfig.EpayType}, "notify_url": {notifyURL}, "return_url": {returnURL}, "name": {"Account recharge"}, "money": {amount},
		}
		values.Set("sign", md5Signature(values, paymentConfig.EpayKey))
		values.Set("sign_type", "MD5")
		gateway, err := url.Parse(paymentConfig.EpayGateway)
		if err != nil {
			return "", err
		}
		if !strings.HasSuffix(strings.TrimRight(gateway.Path, "/"), "/submit.php") {
			gateway.Path = strings.TrimRight(gateway.Path, "/") + "/submit.php"
		}
		gateway.RawQuery = values.Encode()
		gateway.Fragment = ""
		return gateway.String(), nil
	}

	values := url.Values{
		"merchant_id": {paymentConfig.CodepayMerchantID}, "out_trade_no": {order.TradeNo}, "amount": {amount},
		"notify_url": {notifyURL}, "return_url": {returnURL}, "timestamp": {strconv.FormatInt(time.Now().Unix(), 10)},
	}
	nonce, err := randomTradeNo()
	if err != nil {
		return "", err
	}
	values.Set("nonce", nonce)
	values.Set("sign", hmacSignature(values, paymentConfig.CodepayKey))
	request, err := http.NewRequest(http.MethodPost, paymentConfig.CodepayCreateURL, strings.NewReader(values.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := paymentClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", errors.New("payment provider rejected request")
	}
	var result struct {
		PayURL string `json:"pay_url"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&result); err != nil {
		return "", err
	}
	parsed, err := url.Parse(result.PayURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return "", errors.New("payment provider returned an invalid URL")
	}
	return result.PayURL, nil
}

func creditRecharge(provider, tradeNo, amount, providerTradeNo string) error {
	amountCents, err := parseCents(amount)
	if err != nil || tradeNo == "" || providerTradeNo == "" {
		return errors.New("invalid callback")
	}
	var order model.RechargeOrder
	if err := model.DB.Where("trade_no = ? AND provider = ?", tradeNo, provider).First(&order).Error; err != nil || order.AmountCents != amountCents {
		return errors.New("order mismatch")
	}
	if order.Status == model.RechargeStatusPaid {
		if order.ProviderTradeNo == providerTradeNo {
			return nil
		}
		return errors.New("provider trade mismatch")
	}
	if order.Status != model.RechargeStatusPending {
		return errors.New("invalid order status")
	}

	err = model.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		result := tx.Model(&model.RechargeOrder{}).Where("id = ? AND status = ?", order.ID, model.RechargeStatusPending).Updates(map[string]interface{}{
			"status": model.RechargeStatusPaid, "provider_trade_no": providerTradeNo, "paid_at": now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			var current model.RechargeOrder
			if err := tx.First(&current, order.ID).Error; err != nil {
				return err
			}
			if current.Status == model.RechargeStatusPaid && current.ProviderTradeNo == providerTradeNo {
				return nil
			}
			return errors.New("order already processed")
		}
		result = tx.Model(&model.User{}).Where("id = ? AND balance_cents <= ?", order.UserID, int64(math.MaxInt64)-order.AmountCents).
			UpdateColumn("balance_cents", gorm.Expr("balance_cents + ?", order.AmountCents))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("balance overflow or user missing")
		}
		var user model.User
		if err := tx.Select("balance_cents").First(&user, order.UserID).Error; err != nil {
			return err
		}
		ledger := model.BalanceLedger{
			UserID: order.UserID, DeltaCents: order.AmountCents, BalanceAfterCents: user.BalanceCents,
			Kind: model.LedgerKindRecharge, OperationKey: "recharge:" + provider + ":" + tradeNo,
			RequestHash: hashRequest(fmt.Sprintf("recharge:%s:%s:%d", provider, tradeNo, order.AmountCents)),
			Note:        "Account recharge", CreatedAt: now,
		}
		return tx.Create(&ledger).Error
	})
	if err == nil {
		return nil
	}
	// A concurrent identical callback may have committed after our initial read.
	var current model.RechargeOrder
	if reloadErr := model.DB.First(&current, order.ID).Error; reloadErr == nil && current.Status == model.RechargeStatusPaid && current.ProviderTradeNo == providerTradeNo {
		return nil
	}
	return err
}

func paymentResult(c *gin.Context, success bool) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	if success {
		c.String(http.StatusOK, "success")
		return
	}
	c.String(http.StatusOK, "fail")
}

func formatCents(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func parseCents(value string) (int64, error) {
	parts := strings.Split(value, ".")
	if len(parts) > 2 || len(parts) == 0 || parts[0] == "" || (len(parts) == 2 && (len(parts[1]) == 0 || len(parts[1]) > 2)) {
		return 0, errors.New("invalid amount")
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole < 0 || whole > maxBalanceAmount/100 {
		return 0, errors.New("invalid amount")
	}
	fraction := int64(0)
	if len(parts) == 2 {
		text := parts[1]
		if len(text) == 1 {
			text += "0"
		}
		fraction, err = strconv.ParseInt(text, 10, 64)
		if err != nil {
			return 0, errors.New("invalid amount")
		}
	}
	cents := whole*100 + fraction
	if cents > maxBalanceAmount {
		return 0, errors.New("invalid amount")
	}
	return cents, nil
}

func sortedParams(values url.Values, excluded ...string) string {
	exclude := make(map[string]bool, len(excluded))
	for _, key := range excluded {
		exclude[key] = true
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		if !exclude[key] && values.Get(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	return strings.Join(parts, "&")
}

func paramsPresent(values url.Values, keys ...string) bool {
	for _, key := range keys {
		if values.Get(key) == "" {
			return false
		}
	}
	return true
}

func md5Signature(values url.Values, key string) string {
	sum := md5.Sum([]byte(sortedParams(values, "sign", "sign_type") + key))
	return hex.EncodeToString(sum[:])
}

func verifyMD5(values url.Values, key string) bool {
	provided, err := hex.DecodeString(values.Get("sign"))
	expected, expectedErr := hex.DecodeString(md5Signature(values, key))
	return err == nil && expectedErr == nil && hmac.Equal(provided, expected)
}

func hmacSignature(values url.Values, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(sortedParams(values, "sign")))
	return hex.EncodeToString(mac.Sum(nil))
}

func verifyHMAC(values url.Values, key string) bool {
	provided, err := hex.DecodeString(values.Get("sign"))
	expected, expectedErr := hex.DecodeString(hmacSignature(values, key))
	return err == nil && expectedErr == nil && hmac.Equal(provided, expected)
}
