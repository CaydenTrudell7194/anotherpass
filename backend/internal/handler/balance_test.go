//go:build cgo

package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
)

func setupBalanceTest(t *testing.T, balance int64) (*gin.Engine, model.User, model.User, model.ServicePlan) {
	t.Helper()
	if err := model.InitDatabase(filepath.Join(t.TempDir(), "commerce.db")); err != nil {
		t.Fatal(err)
	}
	buyer := model.User{Username: "balance-buyer", Password: "unused", Status: "active", RuleLimit: 1, BalanceCents: balance}
	admin := model.User{Username: "balance-admin", Password: "unused", Status: "active", IsAdmin: true}
	if err := model.DB.Create(&buyer).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.DB.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	plan := model.ServicePlan{Name: "Balance monthly", PriceCents: 1000, DurationDays: 30, RuleLimit: 20, Enabled: true}
	if err := model.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/orders/balance", func(c *gin.Context) { c.Set("user_id", buyer.ID); CreateBalanceOrder(c) })
	router.POST("/admin/users/:id/balance-adjustments", func(c *gin.Context) { c.Set("user_id", admin.ID); AdminAdjustBalance(c) })
	return router, buyer, admin, plan
}

func balanceRequest(router http.Handler, method, path, key string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestBalanceOrderSuccessAndReplay(t *testing.T) {
	router, buyer, _, plan := setupBalanceTest(t, 2500)
	body := []byte(fmt.Sprintf(`{"plan_id":%d}`, plan.ID))
	first := balanceRequest(router, http.MethodPost, "/orders/balance", "purchase-1", body)
	if first.Code != http.StatusCreated {
		t.Fatalf("purchase: %d %s", first.Code, first.Body.String())
	}
	var firstOrder model.Order
	if err := json.Unmarshal(first.Body.Bytes(), &firstOrder); err != nil {
		t.Fatal(err)
	}
	replay := balanceRequest(router, http.MethodPost, "/orders/balance", "purchase-1", body)
	if replay.Code != http.StatusCreated {
		t.Fatalf("replay: %d %s", replay.Code, replay.Body.String())
	}
	var replayOrder model.Order
	if err := json.Unmarshal(replay.Body.Bytes(), &replayOrder); err != nil {
		t.Fatal(err)
	}
	var updated model.User
	if err := model.DB.First(&updated, buyer.ID).Error; err != nil {
		t.Fatal(err)
	}
	var ledgerCount int64
	model.DB.Model(&model.BalanceLedger{}).Count(&ledgerCount)
	if firstOrder.ID != replayOrder.ID || updated.BalanceCents != 1500 || updated.RuleLimit != plan.RuleLimit || updated.ExpireAt.Before(time.Now().AddDate(0, 0, 29)) || ledgerCount != 1 {
		t.Fatalf("unexpected replay state: orders=%d/%d balance=%d rule_limit=%d ledger=%d", firstOrder.ID, replayOrder.ID, updated.BalanceCents, updated.RuleLimit, ledgerCount)
	}
}

func TestBalanceOrderKeyConflictAndInsufficientFunds(t *testing.T) {
	router, buyer, _, plan := setupBalanceTest(t, 1000)
	body := []byte(fmt.Sprintf(`{"plan_id":%d}`, plan.ID))
	if response := balanceRequest(router, http.MethodPost, "/orders/balance", "purchase-conflict", body); response.Code != http.StatusCreated {
		t.Fatalf("purchase: %d %s", response.Code, response.Body.String())
	}
	other := model.ServicePlan{Name: "Other", PriceCents: 1, DurationDays: 1, RuleLimit: 2, Enabled: true}
	if err := model.DB.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	conflictBody := []byte(fmt.Sprintf(`{"plan_id":%d}`, other.ID))
	if response := balanceRequest(router, http.MethodPost, "/orders/balance", "purchase-conflict", conflictBody); response.Code != http.StatusConflict {
		t.Fatalf("expected key conflict, got %d %s", response.Code, response.Body.String())
	}
	if response := balanceRequest(router, http.MethodPost, "/orders/balance", "purchase-insufficient", conflictBody); response.Code != http.StatusConflict {
		t.Fatalf("expected insufficient funds, got %d %s", response.Code, response.Body.String())
	}
	var updated model.User
	model.DB.First(&updated, buyer.ID)
	if updated.BalanceCents != 0 {
		t.Fatalf("balance changed after failed purchase: %d", updated.BalanceCents)
	}
}

func TestAdminBalanceAdjustments(t *testing.T) {
	router, buyer, _, _ := setupBalanceTest(t, 100)
	path := fmt.Sprintf("/admin/users/%d/balance-adjustments", buyer.ID)
	body := []byte(`{"target_balance_cents":500,"reason":"support credit"}`)
	if response := balanceRequest(router, http.MethodPost, path, "adjust-1", body); response.Code != http.StatusOK {
		t.Fatalf("adjustment: %d %s", response.Code, response.Body.String())
	}
	if response := balanceRequest(router, http.MethodPost, path, "adjust-1", body); response.Code != http.StatusOK {
		t.Fatalf("adjustment replay: %d %s", response.Code, response.Body.String())
	}
	if response := balanceRequest(router, http.MethodPost, path, "adjust-1", []byte(`{"target_balance_cents":400,"reason":"support credit"}`)); response.Code != http.StatusConflict {
		t.Fatalf("expected adjustment key conflict, got %d %s", response.Code, response.Body.String())
	}
	if response := balanceRequest(router, http.MethodPost, path, "adjust-2", []byte(`{"target_balance_cents":500,"reason":"no change"}`)); response.Code != http.StatusConflict {
		t.Fatalf("expected no-change conflict, got %d %s", response.Code, response.Body.String())
	}
	var updated model.User
	model.DB.First(&updated, buyer.ID)
	var ledgerCount int64
	model.DB.Model(&model.BalanceLedger{}).Count(&ledgerCount)
	if updated.BalanceCents != 500 || ledgerCount != 1 {
		t.Fatalf("unexpected adjustment state: balance=%d ledger=%d", updated.BalanceCents, ledgerCount)
	}
	var ledger model.BalanceLedger
	if err := model.DB.Where("user_id = ?", buyer.ID).First(&ledger).Error; err != nil || ledger.DeltaCents != 400 || ledger.BalanceAfterCents != 500 {
		t.Fatalf("target adjustment ledger is wrong: %+v, err=%v", ledger, err)
	}
	if err := model.DB.Model(&model.BalanceLedger{}).Where("user_id = ?", buyer.ID).Update("note", "tampered").Error; err == nil {
		t.Fatal("expected immutable ledger update to fail")
	}
	if err := model.DB.Where("user_id = ?", buyer.ID).Delete(&model.BalanceLedger{}).Error; err == nil {
		t.Fatal("expected immutable ledger delete to fail")
	}
}

func TestZeroPricePlanCannotUseBalancePurchase(t *testing.T) {
	router, _, _, _ := setupBalanceTest(t, 1000)
	plan := model.ServicePlan{Name: "Free", PriceCents: 0, DurationDays: 30, RuleLimit: 20, Enabled: true}
	if err := model.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	body := []byte(fmt.Sprintf(`{"plan_id":%d}`, plan.ID))
	response := balanceRequest(router, http.MethodPost, "/orders/balance", "free-plan", body)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	var count int64
	model.DB.Model(&model.Order{}).Where("plan_id = ?", plan.ID).Count(&count)
	if count != 0 {
		t.Fatalf("free plan created %d orders", count)
	}
}
