package battle

import (
	"testing"

	skillmod "pocket-pet-remake/server/internal/module/skill"
)

func TestApplyPassiveSkillDefStrongAndLifesteal(t *testing.T) {
	actor := &actorRuntime{hp: 100, hpMax: 100}
	applyPassiveSkillDef(actor, skillDef{Name: "强壮B", SkillType: "support", HealPct: 20})
	applyPassiveSkillDef(actor, skillDef{Name: "嗜血B", SkillType: "support", CritBoostPct: 18})

	if actor.hpMax != 120 {
		t.Fatalf("hpMax = %d, want 120", actor.hpMax)
	}
	if actor.hp != 120 {
		t.Fatalf("hp = %d, want scaled 120", actor.hp)
	}
	if actor.lifestealPct != 18 {
		t.Fatalf("lifestealPct = %d, want 18", actor.lifestealPct)
	}
}

func TestApplySkillPassivesFromSkillList(t *testing.T) {
	prevResolver := runtimeSkillResolver
	defer func() { runtimeSkillResolver = prevResolver }()
	runtimeSkillResolver = func(skillID uint32) (skillmod.RuntimeDefinition, bool) {
		if skillID != 20001 {
			return skillmod.RuntimeDefinition{}, false
		}
		return skillmod.RuntimeDefinition{
			SkillID:        20001,
			SkillName:      "强壮B",
			SkillType:      "support",
			ActivationMode: skillmod.ActivationModePassive,
			HealPct:        15,
		}, true
	}

	actor := &actorRuntime{hp: 80, hpMax: 80, skillIDs: []uint32{20001}}
	if !applySkillPassives(actor) {
		t.Fatal("applySkillPassives() = false, want true")
	}
	if actor.hpMax != 92 {
		t.Fatalf("hpMax = %d, want 92", actor.hpMax)
	}
}

func TestSkillIDsForClientSnapshotSkipsPassiveSkills(t *testing.T) {
	prevResolver := runtimeSkillResolver
	defer func() { runtimeSkillResolver = prevResolver }()
	runtimeSkillResolver = func(skillID uint32) (skillmod.RuntimeDefinition, bool) {
		if skillID != 20001 {
			return skillmod.RuntimeDefinition{}, false
		}
		return skillmod.RuntimeDefinition{
			SkillID:        20001,
			SkillName:      "反伤体质",
			SkillType:      "support",
			ActivationMode: skillmod.ActivationModePassive,
		}, true
	}

	filtered := skillIDsForClientSnapshot([]uint32{DefaultAttackSkillID, 20001, 1002})
	if len(filtered) != 1 || filtered[0] != 1002 {
		t.Fatalf("skillIDsForClientSnapshot() = %#v, want only active skill 1002", filtered)
	}
}

func TestApplySkillPassivesWithoutPersistentStatsKeepsPanelStatsButRetainsCombatEffects(t *testing.T) {
	prevResolver := runtimeSkillResolver
	defer func() { runtimeSkillResolver = prevResolver }()
	runtimeSkillResolver = func(skillID uint32) (skillmod.RuntimeDefinition, bool) {
		switch skillID {
		case 21001:
			return skillmod.RuntimeDefinition{
				SkillID:        21001,
				SkillName:      "迅捷之心",
				SkillType:      "support",
				ActivationMode: skillmod.ActivationModePassive,
				SpeedPct:       50,
			}, true
		case 21002:
			return skillmod.RuntimeDefinition{
				SkillID:        21002,
				SkillName:      "嗜血之牙",
				SkillType:      "support",
				ActivationMode: skillmod.ActivationModePassive,
				CritBoostPct:   18,
			}, true
		default:
			return skillmod.RuntimeDefinition{}, false
		}
	}

	actor := &actorRuntime{
		spd:      150,
		skillIDs: []uint32{21001, 21002},
	}
	if !applySkillPassivesWithoutPersistentStats(actor) {
		t.Fatal("applySkillPassivesWithoutPersistentStats() = false, want true")
	}
	if actor.speedMultiplierPct != 0 {
		t.Fatalf("speedMultiplierPct = %d, want 0 because panel stats already include passive speed", actor.speedMultiplierPct)
	}
	if actor.lifestealPct != 18 {
		t.Fatalf("lifestealPct = %d, want 18", actor.lifestealPct)
	}
}

func TestBuildPocketDamageInputUsesManaPanel(t *testing.T) {
	attacker := &actorRuntime{atk: 100, mana: 40}
	target := &actorRuntime{def: 10, guard: 50}
	skill := skillDef{SkillMult: 90, ManaPct: 90}
	input := buildPocketDamageInput(attacker, target, skill)
	if input.AttackerPanel != 40 {
		t.Fatalf("AttackerPanel = %d, want mana panel 40", input.AttackerPanel)
	}
}

func TestBuildPocketDamageInputPrefersAttackWhenMixedPanels(t *testing.T) {
	attacker := &actorRuntime{atk: 200, mana: 20, spd: 30}
	target := &actorRuntime{def: 10, guard: 50}
	skill := skillDef{AttackPct: 135, ManaPct: 55, SpeedPct: 35}
	input := buildPocketDamageInput(attacker, target, skill)
	if input.AttackerPanel != 200 {
		t.Fatalf("AttackerPanel = %d, want attack panel 200", input.AttackerPanel)
	}
}

func TestCalculateHealAmountUsesManaForHealSkill(t *testing.T) {
	caster := effectiveStats{MaxHP: 200, Mana: 50}
	heal := calculateHealAmount(caster, skillDef{SkillType: "heal", HealPct: 70})
	if heal != 350 {
		t.Fatalf("heal = %d, want 350", heal)
	}
}

func TestCalculateCompositeSkillDamageTrueHit(t *testing.T) {
	attacker := effectiveStats{Attack: 100, Mana: 50, Speed: 80}
	target := effectiveStats{Guard: 500, Defense: 200}
	skill := skillDef{
		SkillType:     "attack",
		AttackPct:     400,
		ManaPct:       2000,
		SpeedPct:      2000,
		IgnoreDefense: true,
	}
	raw := calculateCompositePanelRaw(attacker, skill)
	if raw != 3000 {
		t.Fatalf("raw = %d, want 3000", raw)
	}
	damage := calculateCompositeSkillDamage(attacker, target, skill, true)
	if damage != 3000 {
		t.Fatalf("damage = %d, want true damage 3000", damage)
	}
}

func TestSkillDefGuaranteedHitAndComposite(t *testing.T) {
	skill := skillDef{Name: "圣技·心灵审判", SkillType: "attack", AttackPct: 400, ManaPct: 2000, SpeedPct: 2000}
	if !skill.isGuaranteedHit() {
		t.Fatal("signature skill should be guaranteed hit")
	}
	if !skill.usesCompositePanelDamage() {
		t.Fatal("signature skill should use composite panel damage")
	}
}
