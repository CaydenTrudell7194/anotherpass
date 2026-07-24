//go:build cgo

package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"forward-panel/internal/config"
	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
)

func TestRechargeCreationIsIdempotent(t *testing.T) {
	if err := model.InitDatabase(filepath.Join(t.TempDir(), "recharge-create.db")); err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "payer", Password: "unused", Status: "active"}
	if err := model.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	ConfigurePayments(&config.Config{PublicURL: "https://panel.example", EpayGateway: "https://pay.example", EpayPID: "merchant-1", EpayKey: "secret", EpayType: "alipay"})
	t.Cleanup(func() { ConfigurePayments(&config.Config{}) })
	router := gin.New()
	router.POST("/recharge", func(c *gin.Context) { c.Set("user_id", user.ID); CreateRecharge(c) })
	request := func() *httptest.ResponseRecorder {
		body, _ := json.Marshal(gin.H{"provider": "epay", "amount_cents": 1234})
		req := httptest.NewRequest(http.MethodPost, "/recharge", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "recharge-once")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res
	}
	first, second := request(), request()
	if first.Code != http.StatusCreated || second.Code != http.StatusOK {
		t.Fatalf("unexpected statuses %d/%d: %s %s", first.Code, second.Code, first.Body.String(), second.Body.String())
	}
	var count int64
	model.DB.Model(&model.RechargeOrder{}).Count(&count)
	if count != 1 {
		t.Fatalf("created %d recharge orders", count)
	}
}

func TestEpayCallbackValidationAndReplay(t *testing.T) {
	if err := model.InitDatabase(filepath.Join(t.TempDir(), "payment.db")); err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "recharge-user", Password: "unused", Status: "active", BalanceCents: 200}
	if err := model.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	order := model.RechargeOrder{
		TradeNo: "local-trade-1", UserID: user.ID, Provider: model.RechargeProviderEpay,
		AmountCents: 1234, Status: model.RechargeStatusPending,
	}
	if err := model.DB.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	ConfigurePayments(&config.Config{PublicURL: "https://panel.example", EpayGateway: "https://pay.example", EpayPID: "merchant-1", EpayKey: "secret"})
	t.Cleanup(func() { ConfigurePayments(&config.Config{}) })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/notify", EpayNotify)
	values := url.Values{
		"pid": {"merchant-1"}, "trade_status": {"TRADE_SUCCESS"}, "out_trade_no": {order.TradeNo},
		"trade_no": {"provider-trade-1"}, "money": {"12.34"}, "sign_type": {"MD5"},
	}

	values.Set("sign", "bad")
	if response := paymentCallbackRequest(router, values); response.Code != http.StatusOK || response.Body.String() != "fail" {
		t.Fatalf("invalid signature response: %d %q", response.Code, response.Body.String())
	}
	assertRechargeState(t, user.ID, order.ID, 200, model.RechargeStatusPending, 0)

	values.Set("sign", md5Signature(values, "secret"))
	if response := paymentCallbackRequest(router, values); response.Code != http.StatusOK || response.Body.String() != "success" {
		t.Fatalf("valid callback response: %d %q", response.Code, response.Body.String())
	}
	assertRechargeState(t, user.ID, order.ID, 1434, model.RechargeStatusPaid, 1)

	if response := paymentCallbackRequest(router, values); response.Code != http.StatusOK || response.Body.String() != "success" {
		t.Fatalf("replay response: %d %q", response.Code, response.Body.String())
	}
	assertRechargeState(t, user.ID, order.ID, 1434, model.RechargeStatusPaid, 1)
}

func paymentCallbackRequest(router http.Handler, values url.Values) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func assertRechargeState(t *testing.T, userID, orderID uint, balance int64, status string, ledgerCount int64) {
	t.Helper()
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		t.Fatal(err)
	}
	var order model.RechargeOrder
	if err := model.DB.First(&order, orderID).Error; err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := model.DB.Model(&model.BalanceLedger{}).Where("kind = ?", model.LedgerKindRecharge).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if user.BalanceCents != balance || order.Status != status || count != ledgerCount {
		t.Fatalf("unexpected recharge state: balance=%d status=%s ledger=%d", user.BalanceCents, order.Status, count)
	}
	if ledgerCount > 0 {
		var ledger model.BalanceLedger
		if err := model.DB.Where("kind = ?", model.LedgerKindRecharge).First(&ledger).Error; err != nil {
			t.Fatal(err)
		}
		if ledger.DeltaCents != 1234 || ledger.BalanceAfterCents != balance {
			t.Fatalf("unexpected recharge ledger: %+v", ledger)
		}
	}
}
