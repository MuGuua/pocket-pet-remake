package httptransport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/data/memory"
	"pocket-pet-remake/server/internal/module/auth"
)

func TestLoginHandler(t *testing.T) {
	cfg := config.Config{
		JWTSecret:      "secret",
		AccessTokenTTL: time.Hour,
		WSTokenTTL:     time.Minute,
		DemoAccount:    "demo",
		DemoPassword:   "demo123",
		DemoPlayerName: "DemoTrainer",
		DemoAccountID:  1,
		DemoPlayerID:   10001,
	}

	accountRepo := memory.NewAccountRepository(cfg)
	wsTokenRepo := memory.NewWSTokenRepository()
	signer := auth.NewHMACSigner(cfg.JWTSecret, cfg.AccessTokenTTL)
	handler := NewLoginHandler(auth.NewService(accountRepo, wsTokenRepo, signer, cfg.WSTokenTTL))

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
