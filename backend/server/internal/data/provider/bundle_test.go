package provider

import (
	"testing"

	"pocket-pet-remake/server/internal/config"
)

func TestNewConfiguredBundleRequiresExternalDeps(t *testing.T) {
	cfg := config.Config{}

	_, err := NewConfiguredBundle(cfg, Dependencies{})
	if err == nil {
		t.Fatal("NewConfiguredBundle() error = nil, want dependency validation error")
	}
}
