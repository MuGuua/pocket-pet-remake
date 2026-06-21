package teststub

import (
	"context"
	"testing"

	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/progression"
)

// TestAddExpLevelUpRefillsHPAndVigor 验证升级后服务端会把当前血量与精力补满。
func TestAddExpLevelUpRefillsHPAndVigor(t *testing.T) {
	playerRepo := NewPlayerRepository()
	progressionRepo := NewProgressionRepository(playerRepo)
	progressionService := progression.NewService(progressionRepo)
	if err := progressionService.RefreshRuntimeCache(context.Background()); err != nil {
		t.Fatalf("RefreshRuntimeCache() error = %v", err)
	}
	svc := player.NewService(playerRepo, nil, progressionService)

	playerRepo.mu.Lock()
	stored := playerRepo.players[DemoPlayerID]
	stored.HP = 30
	stored.Vigor = 20
	playerRepo.players[DemoPlayerID] = stored
	playerRepo.mu.Unlock()

	grantResult, err := svc.AddExp(t.Context(), DemoPlayerID, 100)
	if err != nil {
		t.Fatalf("AddExp() error = %v", err)
	}
	if grantResult.LevelUpCount == 0 {
		t.Fatal("LevelUpCount = 0, want > 0")
	}
	profileAfter := grantResult.Profile
	if profileAfter == nil {
		t.Fatal("Profile is nil")
	}
	if profileAfter.HP != profileAfter.HPMax {
		t.Fatalf("HP = %d, want full HPMax %d after level up", profileAfter.HP, profileAfter.HPMax)
	}
	if profileAfter.Vigor != profileAfter.VigorMax {
		t.Fatalf("Vigor = %d, want full VigorMax %d after level up", profileAfter.Vigor, profileAfter.VigorMax)
	}
}
