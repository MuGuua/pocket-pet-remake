package httptransport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/auth"
	"pocket-pet-remake/server/internal/teststub"
)

func TestLoginHandler(t *testing.T) {
	signer := auth.NewHMACSigner("secret", time.Hour)
	handler := NewLoginHandler(auth.NewService(teststub.NewAccountRepository(), teststub.NewWSTokenRepository(), signer, time.Minute))

	body, err := json.Marshal(map[string]string{
		"account":   "demo",
		"password":  "demo123",
		"device_id": "device-1",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["msg"] != "success" {
		t.Fatalf("payload[msg] = %v, want success", payload["msg"])
	}
	if payload["data"] == nil {
		t.Fatal("payload[data] should not be nil")
	}
}
