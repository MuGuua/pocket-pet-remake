package pet

import (
	"testing"

	"pocket-pet-remake/server/internal/module/skill"
)

func TestApplyPersistentPassiveBonusesToPetUpdatesDisplayedStats(t *testing.T) {
	item := Pet{
		HP:       100,
		HPMax:    100,
		ATK:      40,
		SPD:      100,
		MANA:     30,
		SkillIDs: []uint32{3001, 3002, 3003},
	}

	applied := applyPersistentPassiveBonusesToPet(&item, func(skillID uint32) (skill.RuntimeDefinition, bool) {
		switch skillID {
		case 3001:
			return skill.RuntimeDefinition{
				SkillID:        3001,
				SkillName:      "迅捷之心",
				SkillType:      "support",
				ActivationMode: skill.ActivationModePassive,
				SpeedPct:       50,
			}, true
		case 3002:
			return skill.RuntimeDefinition{
				SkillID:        3002,
				SkillName:      "强壮之躯",
				SkillType:      "support",
				ActivationMode: skill.ActivationModePassive,
				HealPct:        20,
			}, true
		case 3003:
			return skill.RuntimeDefinition{
				SkillID:        3003,
				SkillName:      "致命本能",
				SkillType:      "support",
				ActivationMode: skill.ActivationModePassive,
				CritBoostPct:   15,
			}, true
		default:
			return skill.RuntimeDefinition{}, false
		}
	})

	if !applied {
		t.Fatal("applyPersistentPassiveBonusesToPet() = false, want true")
	}
	if item.SPD != 150 {
		t.Fatalf("SPD = %d, want 150", item.SPD)
	}
	if item.HPMax != 120 || item.HP != 120 {
		t.Fatalf("HP/HPMax = %d/%d, want 120/120", item.HP, item.HPMax)
	}
	if item.CritRatePct != 15 {
		t.Fatalf("CritRatePct = %d, want 15", item.CritRatePct)
	}
}

func TestApplyPersistentPassiveBonusesToLineupPetUpdatesBattleBaseStats(t *testing.T) {
	item := LineupPet{
		HP:       80,
		HPMax:    80,
		ATK:      60,
		SPD:      100,
		MANA:     50,
		SkillIDs: []uint32{3101, 3102},
	}

	applied := applyPersistentPassiveBonusesToLineupPet(&item, func(skillID uint32) (skill.RuntimeDefinition, bool) {
		switch skillID {
		case 3101:
			return skill.RuntimeDefinition{
				SkillID:        3101,
				SkillName:      "强力本能",
				SkillType:      "support",
				ActivationMode: skill.ActivationModePassive,
				AttackPct:      25,
			}, true
		case 3102:
			return skill.RuntimeDefinition{
				SkillID:        3102,
				SkillName:      "魔心回响",
				SkillType:      "support",
				ActivationMode: skill.ActivationModePassive,
				ManaPct:        40,
			}, true
		default:
			return skill.RuntimeDefinition{}, false
		}
	})

	if !applied {
		t.Fatal("applyPersistentPassiveBonusesToLineupPet() = false, want true")
	}
	if item.ATK != 75 {
		t.Fatalf("ATK = %d, want 75", item.ATK)
	}
	if item.MANA != 70 {
		t.Fatalf("MANA = %d, want 70", item.MANA)
	}
}

func TestApplyPersistentPassiveBonusesToPetUsesExplicitAttributeBonusConfig(t *testing.T) {
	item := Pet{
		HP:       100,
		HPMax:    100,
		ATK:      40,
		SPD:      100,
		MANA:     30,
		SkillIDs: []uint32{3201, 3202},
	}

	applied := applyPersistentPassiveBonusesToPet(&item, func(skillID uint32) (skill.RuntimeDefinition, bool) {
		switch skillID {
		case 3201:
			return skill.RuntimeDefinition{
				SkillID:          3201,
				SkillName:        "任意名字也能生效",
				SkillType:        "support",
				ActivationMode:   skill.ActivationModePassive,
				PassiveAttrKey:   skill.PassiveAttrKeySPD,
				PassiveAttrMode:  skill.PassiveAttrModePercent,
				PassiveAttrValue: 50,
			}, true
		case 3202:
			return skill.RuntimeDefinition{
				SkillID:          3202,
				SkillName:        "固定攻击被动",
				SkillType:        "support",
				ActivationMode:   skill.ActivationModePassive,
				PassiveAttrKey:   skill.PassiveAttrKeyATK,
				PassiveAttrMode:  skill.PassiveAttrModeFlat,
				PassiveAttrValue: 12,
			}, true
		default:
			return skill.RuntimeDefinition{}, false
		}
	})

	if !applied {
		t.Fatal("applyPersistentPassiveBonusesToPet() = false, want true")
	}
	if item.SPD != 150 {
		t.Fatalf("SPD = %d, want 150", item.SPD)
	}
	if item.ATK != 52 {
		t.Fatalf("ATK = %d, want 52", item.ATK)
	}
}

func TestReconcileDisplayedAdminStatsToRawRestoresUntouchedBaseStatsAfterPassiveSkillRemoval(t *testing.T) {
	rawDetail := &AdminPetDetail{
		HP:             100,
		HPMax:          100,
		ATK:            40,
		SPD:            100,
		MANA:           30,
		SkillIDs:       []uint32{3301},
		InnateSkillIDs: []uint32{},
		NormalSkillIDs: []uint32{3301},
		AdminPetCombatStats: AdminPetCombatStats{
			CritRatePct: 0,
		},
	}
	input := AdminUpdatePetInput{
		PetID:          101,
		HP:             120,
		HPMax:          120,
		ATK:            40,
		SPD:            100,
		MANA:           30,
		SkillIDs:       []uint32{},
		InnateSkillIDs: []uint32{},
		NormalSkillIDs: []uint32{},
		AdminPetCombatStats: AdminPetCombatStats{
			CritRatePct: 15,
		},
	}

	reconcileDisplayedAdminStatsToRaw(&input, rawDetail, func(detail *AdminPetDetail) {
		if detail == nil {
			return
		}
		detail.HP = 120
		detail.HPMax = 120
		detail.CritRatePct = 15
	})

	if input.HP != 100 || input.HPMax != 100 {
		t.Fatalf("HP/HPMax = %d/%d, want 100/100", input.HP, input.HPMax)
	}
	if input.CritRatePct != 0 {
		t.Fatalf("CritRatePct = %d, want 0", input.CritRatePct)
	}
}

func TestReconcileDisplayedAdminStatsToRawKeepsUserModifiedDisplayedStat(t *testing.T) {
	rawDetail := &AdminPetDetail{
		HP:             100,
		HPMax:          100,
		ATK:            40,
		SPD:            100,
		MANA:           30,
		SkillIDs:       []uint32{3302},
		InnateSkillIDs: []uint32{},
		NormalSkillIDs: []uint32{3302},
		AdminPetCombatStats: AdminPetCombatStats{
			CritRatePct: 0,
		},
	}
	input := AdminUpdatePetInput{
		PetID:          101,
		HP:             150,
		HPMax:          150,
		ATK:            40,
		SPD:            100,
		MANA:           30,
		SkillIDs:       []uint32{},
		InnateSkillIDs: []uint32{},
		NormalSkillIDs: []uint32{},
		AdminPetCombatStats: AdminPetCombatStats{
			CritRatePct: 20,
		},
	}

	reconcileDisplayedAdminStatsToRaw(&input, rawDetail, func(detail *AdminPetDetail) {
		if detail == nil {
			return
		}
		detail.HP = 120
		detail.HPMax = 120
		detail.CritRatePct = 15
	})

	if input.HP != 150 || input.HPMax != 150 {
		t.Fatalf("HP/HPMax = %d/%d, want keep user value 150/150", input.HP, input.HPMax)
	}
	if input.CritRatePct != 20 {
		t.Fatalf("CritRatePct = %d, want keep user value 20", input.CritRatePct)
	}
}
