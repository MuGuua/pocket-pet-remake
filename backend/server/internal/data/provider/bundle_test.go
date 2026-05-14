package provider

import (
	"testing"

	"pocket-pet-remake/server/internal/config"
)

func TestNewConfiguredBundleDefaultsToMemory(t *testing.T) {
	cfg := config.Config{
		DemoAccount:    "demo",
		DemoPassword:   "demo123",
		DemoPlayerName: "DemoTrainer",
		DemoAccountID:  1,
		DemoPlayerID:   10001,
	}

	bundle, err := NewConfiguredBundle(cfg, Dependencies{})
	if err != nil {
		t.Fatalf("NewConfiguredBundle() error = %v", err)
	}
	if bundle.Accounts == nil || bundle.Players == nil || bundle.Pets == nil || bundle.World == nil || bundle.WSTokens == nil {
		t.Fatal("NewConfiguredBundle() returned an incomplete memory bundle")
	}
}

func TestNewConfiguredBundleRequiresExternalDeps(t *testing.T) {
	cfg := config.Config{RepositoryMode: config.RepositoryModePostgresRedis}

	_, err := NewConfiguredBundle(cfg, Dependencies{})
	if err == nil {
		t.Fatal("NewConfiguredBundle() error = nil, want dependency validation error")
	}
}
