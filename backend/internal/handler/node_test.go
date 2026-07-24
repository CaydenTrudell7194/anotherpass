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
)

func TestNodeEnrollmentCodeCanOnlyBeUsedOnce(t *testing.T) {
	if err := model.InitDatabase(fmt.Sprintf("file:node-enroll-%d?mode=memory&cache=shared", time.Now().UnixNano())); err != nil {
		t.Fatalf("init database: %v", err)
	}
	group := model.DeviceGroup{Name: "direct", Type: model.DeviceGroupEntryForceDirect}
	if err := model.DB.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	node := model.Node{Name: "node-1", DeviceGroupID: group.ID, Token: "old-token", Status: "offline"}
	if err := model.DB.Create(&node).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/nodes/:id/setup", GetNodeSetup)
	router.POST("/node/enroll", EnrollNode)

	setupReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/nodes/%d/setup", node.ID), nil)
	setupRes := httptest.NewRecorder()
	router.ServeHTTP(setupRes, setupReq)
	if setupRes.Code != http.StatusOK {
		t.Fatalf("setup failed: %s", setupRes.Body.String())
	}
	var setup struct {
		Code string `json:"enroll_code"`
	}
	if err := json.Unmarshal(setupRes.Body.Bytes(), &setup); err != nil || setup.Code == "" {
		t.Fatalf("invalid setup response: %s", setupRes.Body.String())
	}

	payload, _ := json.Marshal(map[string]string{"code": setup.Code})
	for attempt := 1; attempt <= 2; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/node/enroll", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if attempt == 1 && res.Code != http.StatusOK {
			t.Fatalf("first enrollment failed: %s", res.Body.String())
		}
		if attempt == 2 && res.Code != http.StatusUnauthorized {
			t.Fatalf("reused code returned %d: %s", res.Code, res.Body.String())
		}
	}
}
