//go:build cgo

package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
)

func TestToggleForwardRuleEnforcesOwnership(t *testing.T) {
	if err := model.InitDatabase("file::memory:?cache=shared"); err != nil {
		t.Fatalf("init database: %v", err)
	}
	owner := model.User{Username: "owner", Password: "unused", Status: "active"}
	attacker := model.User{Username: "attacker", Password: "unused", Status: "active"}
	if err := model.DB.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.DB.Create(&attacker).Error; err != nil {
		t.Fatal(err)
	}
	rule := model.ForwardRule{UserID: owner.ID, Name: "rule", ListenPort: 10000, TargetAddr: "127.0.0.1", TargetPort: 80, Protocol: "tcp", Enabled: true}
	if err := model.DB.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/rules/:id/toggle", func(c *gin.Context) {
		c.Set("user_id", attacker.ID)
		c.Set("is_admin", false)
		ToggleForwardRule(c)
	})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/rules/%d/toggle", rule.ID), nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", res.Code, res.Body.String())
	}
	var stored model.ForwardRule
	if err := model.DB.First(&stored, rule.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !stored.Enabled {
		t.Fatal("attacker changed another user's rule")
	}
}
