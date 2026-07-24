//go:build cgo

package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCommerceTest(t *testing.T) (*gin.Engine, model.User, model.ServicePlan) {
	t.Helper()
	var err error
	model.DB, err = gorm.Open(sqlite.Open(fmt.Sprintf("file:commerce-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.DB.AutoMigrate(&model.User{}, &model.UserGroup{}, &model.ServicePlan{}, &model.Order{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "buyer", Password: "unused", Status: "active", RuleLimit: 1}
	if err := model.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	plan := model.ServicePlan{Name: "Monthly", PriceCents: 1000, DurationDays: 30, RuleLimit: 20, Enabled: true}
	if err := model.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/orders", func(c *gin.Context) { c.Set("user_id", user.ID); CreateOrder(c) })
	router.POST("/orders/:id/approve", func(c *gin.Context) { c.Set("user_id", uint(99)); AdminApproveOrder(c) })
	return router, user, plan
}

func TestCommerceOrderSnapshotsAndApprovalIsIdempotent(t *testing.T) {
	router, user, plan := setupCommerceTest(t)
	body, _ := json.Marshal(gin.H{"plan_id": plan.ID, "user_note": "bank transfer"})
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body)))
	if res.Code != http.StatusCreated {
		t.Fatalf("create order: %d %s", res.Code, res.Body.String())
	}
	var order model.Order
	if err := json.Unmarshal(res.Body.Bytes(), &order); err != nil {
		t.Fatal(err)
	}
	if err := model.DB.Model(&model.ServicePlan{}).Where("id = ?", plan.ID).Updates(map[string]interface{}{"name": "Changed", "rule_limit": 99}).Error; err != nil {
		t.Fatal(err)
	}

	approve := func() {
		res = httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/orders/%d/approve", order.ID), bytes.NewBufferString(`{}`)))
		if res.Code != http.StatusOK {
			t.Fatalf("approve order: %d %s", res.Code, res.Body.String())
		}
	}
	approve()
	var approvedUser model.User
	if err := model.DB.First(&approvedUser, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	firstExpiry := approvedUser.ExpireAt
	if approvedUser.RuleLimit != 20 || firstExpiry.Before(time.Now().AddDate(0, 0, 29)) {
		t.Fatalf("snapshot benefits not applied: rule_limit=%d expire_at=%s", approvedUser.RuleLimit, firstExpiry)
	}
	approve()
	if err := model.DB.First(&approvedUser, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !approvedUser.ExpireAt.Equal(firstExpiry) {
		t.Fatalf("second approval extended expiry: %s != %s", approvedUser.ExpireAt, firstExpiry)
	}
}

func TestCommercePendingOrderLimitAndReferencedPlanDelete(t *testing.T) {
	router, _, plan := setupCommerceTest(t)
	body := []byte(fmt.Sprintf(`{"plan_id":%d}`, plan.ID))
	for i := 0; i < 3; i++ {
		res := httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body)))
		if res.Code != http.StatusCreated {
			t.Fatalf("order %d: %d %s", i, res.Code, res.Body.String())
		}
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body)))
	if res.Code != http.StatusConflict {
		t.Fatalf("expected pending limit conflict, got %d: %s", res.Code, res.Body.String())
	}

	deleteRouter := gin.New()
	deleteRouter.DELETE("/plans/:id", AdminDeleteServicePlan)
	res = httptest.NewRecorder()
	deleteRouter.ServeHTTP(res, httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/plans/%d", plan.ID), nil))
	if res.Code != http.StatusConflict {
		t.Fatalf("expected referenced plan conflict, got %d: %s", res.Code, res.Body.String())
	}
}
