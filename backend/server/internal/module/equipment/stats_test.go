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

// TestBuildRecalcResultAddsAllocatedAttrsAndMultipleEquipment 验证人物最终公式会同时接收
// 成长加点转化值与多件装备加成，避免后续只读取单件装备或漏掉加点重算。
func TestBuildRecalcResultAddsAllocatedAttrsAndMultipleEquipment(t *testing.T) {
	ctx := RecalcContext{
		Progression: progression.ProgressionState{
			BaseCombat: progression.BaseCombatStats{
				HPMax: 100,
				ATK:   20,
				DEF:   10,
				SPD:   5,
				MANA:  8,
			},
		},
		CombatBonus: progression.CombatBonus{
			HPMax: 50,
			ATK:   6,
			MANA:  4,
		},
		Caps: pet.CombatStatCaps{},
	}
	templates := []EquippedPieceTemplate{
		{
			EquipSlot: EquipSlotWeapon,
			BaseATK:   30,
			BaseSPD:   2,
		},
		{
			EquipSlot: EquipSlotClothes,
			BaseHP:    80,
			BaseDEF:   12,
		},
		{
			EquipSlot:      EquipSlotCostume,
			AppearanceOnly: true,
			BaseATK:        999,
		},
	}

	result := BuildRecalcResult(ctx, templates, &player.Profile{})
	if result.HPMax != 230 {
		t.Fatalf("HPMax = %d, want 230 (100 base + 50 allocated + 80 armor)", result.HPMax)
	}
	if result.ATK != 56 {
		t.Fatalf("ATK = %d, want 56 (20 base + 6 allocated + 30 weapon)", result.ATK)
	}
	if result.DEF != 22 {
		t.Fatalf("DEF = %d, want 22 (10 base + 12 armor)", result.DEF)
	}
	if result.SPD != 7 {
		t.Fatalf("SPD = %d, want 7 (5 base + 2 weapon)", result.SPD)
	}
	if result.MANA != 12 {
		t.Fatalf("MANA = %d, want 12 (8 base + 4 allocated)", result.MANA)
	}
}
