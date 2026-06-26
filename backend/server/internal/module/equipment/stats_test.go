package equipment

import (
	"testing"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/progression"
)

// TestBuildRecalcResultAddsEquipmentATK 验证成长层裸装攻击与装备固定攻击会叠加到最终 ATK。
func TestBuildRecalcResultAddsEquipmentATK(t *testing.T) {
	ctx := RecalcContext{
		Progression: progression.ProgressionState{
			BaseCombat: progression.BaseCombatStats{ATK: 52},
		},
		CombatBonus: progression.CombatBonus{},
		Caps:        pet.CombatStatCaps{},
	}
	templates := []EquippedPieceTemplate{
		{
			EquipSlot: EquipSlotWeapon,
			BaseATK:   100,
		},
	}

	result := BuildRecalcResult(ctx, templates, &player.Profile{})
	if result.ATK != 152 {
		t.Fatalf("ATK = %d, want 152 (52 base + 100 weapon)", result.ATK)
	}
}
