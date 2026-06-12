package battle

import (
	"context"
	"testing"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/world"
)

func TestServiceSubmitCaptureRequiresMonsterService(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()
	profile := &player.Profile{PlayerID: 10001, Name: "DemoTrainer", Level: 8, SceneID: 1, PosX: 8, PosY: 6, SkillIDs: []uint32{1101, 1001}}
	lineup := []pet.LineupPet{
		{PetUID: 20001, PetID: 101, Level: 5, HP: 90, HPMax: 90, ATK: 10, DEF: 10, SPD: 8, MANA: 10, SkillIDs: []uint32{1001}},
	}
	enemy := world.Entity{EntityID: 90006, EntityType: 2, Pos: world.Vec2i{X: 10, Y: 6}, Name: "GuideNPC"}
	start, err := svc.StartPVE(ctx, profile, lineup, enemy)
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}

	outcome, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:     start.BattleID,
		Round:        start.Round,
		ActionType:   ActionTypeCapture,
		ActorID:      start.Allies[0].ActorID,
		TargetID:     start.Enemies[0].ActorID,
		ItemID:       2001,
		BagSlotIndex: 1,
	})
	if err != nil {
		t.Fatalf("SubmitAction() error = %v", err)
	}
	if outcome.Response.Accepted || outcome.Response.Reason != "capture unavailable" {
		t.Fatalf("expected capture unavailable, got %#v", outcome.Response)
	}
}
