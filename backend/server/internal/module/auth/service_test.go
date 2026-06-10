package auth_test

import (
	"context"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/auth"
	"pocket-pet-remake/server/internal/teststub"
)

func TestLoginAndConsumeWSToken(t *testing.T) {
	signer := auth.NewHMACSigner("secret", time.Hour)
	service := auth.NewService(teststub.NewAccountRepository(), teststub.NewWSTokenRepository(), signer, time.Minute)

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
