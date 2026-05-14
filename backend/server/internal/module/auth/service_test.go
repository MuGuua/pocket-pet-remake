package auth_test

import (
	"context"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/data/memory"
	"pocket-pet-remake/server/internal/module/auth"
)

func TestLoginAndConsumeWSToken(t *testing.T) {
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
	service := auth.NewService(accountRepo, wsTokenRepo, signer, cfg.WSTokenTTL)

	result, err := service.Login(context.Background(), "demo", "demo123", "device-1")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.PlayerID != 10001 {
		t.Fatalf("PlayerID = %d, want 10001", result.PlayerID)
	}
	if result.WSToken == "" {
		t.Fatal("WSToken should not be empty")
	}
	if result.AccessJWT == "" {
		t.Fatal("AccessJWT should not be empty")
	}

	principal, err := service.ConsumeWSToken(context.Background(), result.WSToken, "device-1")
	if err != nil {
		t.Fatalf("ConsumeWSToken() error = %v", err)
	}
	if principal.PlayerID != result.PlayerID {
		t.Fatalf("principal.PlayerID = %d, want %d", principal.PlayerID, result.PlayerID)
	}

	if _, err := service.ConsumeWSToken(context.Background(), result.WSToken, "device-1"); err == nil {
		t.Fatal("second ConsumeWSToken() should fail")
	}
}
