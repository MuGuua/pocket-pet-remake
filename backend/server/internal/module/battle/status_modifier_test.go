package battle

import "testing"

func TestHolyRepentanceAppliesFullDebuffProfile(t *testing.T) {
	actor := &actorRuntime{
		hp:                       100,
		hpMax:                    100,
		hitPct:                   120,
		dodgeRatePct:             80,
		critDmgPct:               300,
		physicalResistPct:        40,
		skillResistPct:           35,
		reversePhysicalResistPct: 30,
		reverseSkillResistPct:    25,
		spd:                      100,
		statuses:                 map[uint32]*statusRuntime{},
	}
	profile := defaultStatusProfile(StatusHolyRepentance)
	if !actor.applyStatusWithProfile(StatusHolyRepentance, 3, 0, profile) {
		t.Fatal("applyStatusWithProfile() = false, want true")
	}
	if actor.applyStatusWithProfile(StatusHolyRepentance, 3, 0, profile) {
		t.Fatal("second apply should fail due to non-stackable")
	}
	stats := actor.effectiveStats()
	if stats.HitPct != 70 {
		t.Fatalf("HitPct = %d, want 70", stats.HitPct)
	}
	if stats.DodgePct != 30 {
		t.Fatalf("DodgePct = %d, want 30", stats.DodgePct)
	}
	if stats.CritDmgPct != 100 {
		t.Fatalf("CritDmgPct = %d, want 100 after -200 penalty floor", stats.CritDmgPct)
	}
	if stats.PhysicalResistPct != 15 || stats.SkillResistPct != 10 {
		t.Fatalf("resists = %d/%d, want 15/10", stats.PhysicalResistPct, stats.SkillResistPct)
	}
	if stats.ReversePhysicalResistPct != 5 || stats.ReverseSkillResistPct != 0 {
		t.Fatalf("reverse resists = %d/%d, want 5/0", stats.ReversePhysicalResistPct, stats.ReverseSkillResistPct)
	}
}

func TestElectrifiedAppliesSpeedAndAccuracyDebuff(t *testing.T) {
	actor := &actorRuntime{
		hp:           100,
		hpMax:        100,
		spd:          100,
		hitPct:       90,
		dodgeRatePct: 70,
		statuses:     map[uint32]*statusRuntime{},
	}
	profile := defaultStatusProfile(StatusElectrified)
	if !actor.applyStatusWithProfile(StatusElectrified, 3, 0, profile) {
		t.Fatal("applyStatusWithProfile() = false, want true")
	}
	stats := actor.effectiveStats()
	if stats.Speed != 60 {
		t.Fatalf("Speed = %d, want 60 (-40%%)", stats.Speed)
	}
	if stats.HitPct != 50 || stats.DodgePct != 30 {
		t.Fatalf("hit/dodge = %d/%d, want 50/30", stats.HitPct, stats.DodgePct)
	}
}

func TestResolveSignatureStatusApplyChanceUsesHigherBranch(t *testing.T) {
	if got := resolveSignatureStatusApplyChance(75, 325, 275); got != 100 {
		t.Fatalf("chance = %d, want 100 when power diff >= 50", got)
	}
	if got := resolveSignatureStatusApplyChance(75, 0, 0); got != 75 {
		t.Fatalf("chance = %d, want 75 when only probability set", got)
	}
}

func TestBloodConfusionForcesConfusionWithoutActionBlock(t *testing.T) {
	actor := &actorRuntime{hp: 100, hpMax: 100, statuses: map[uint32]*statusRuntime{}}
	profile := defaultStatusProfile(StatusBloodConfusion)
	if !actor.applyStatusWithProfile(StatusBloodConfusion, 2, 0, profile) {
		t.Fatal("apply failed")
	}
	if !actor.isConfused() {
		t.Fatal("blood confusion should mark actor confused")
	}
	if _, _, blocked := actor.actionBlockedStatus(); blocked {
		t.Fatal("blood confusion should not block action, only redirect target")
	}
}

// TestParalysisHalvesDefensiveStats 锁定《战斗系统规则》6 中「麻痹使物理抗性、技能抗性、
// 抗爆伤、命中、闪避各降低 50%」的规则，同时保留无法行动的效果。
func TestParalysisHalvesDefensiveStats(t *testing.T) {
	actor := &actorRuntime{
		hp:                  100,
		hpMax:               100,
		hitPct:              80,
		dodgeRatePct:        60,
		critDmgPct:          400,
		physicalResistPct:   60,
		skillResistPct:      50,
		statuses:            map[uint32]*statusRuntime{},
		globalMultiplierPct: 100,
	}
	profile := defaultStatusProfile(StatusParalysis)
	if !actor.applyStatusWithProfile(StatusParalysis, 2, 0, profile) {
		t.Fatal("applyStatusWithProfile() = false, want true")
	}

	stats := actor.effectiveStats()
	if stats.HitPct != 40 {
		t.Fatalf("HitPct = %d, want 40 (80 × 50%%)", stats.HitPct)
	}
	if stats.DodgePct != 30 {
		t.Fatalf("DodgePct = %d, want 30 (60 × 50%%)", stats.DodgePct)
	}
	if stats.CritDmgPct != 200 {
		t.Fatalf("CritDmgPct = %d, want 200 (400 × 50%%)", stats.CritDmgPct)
	}
	if stats.PhysicalResistPct != 30 {
		t.Fatalf("PhysicalResistPct = %d, want 30 (60 × 50%%)", stats.PhysicalResistPct)
	}
	if stats.SkillResistPct != 25 {
		t.Fatalf("SkillResistPct = %d, want 25 (50 × 50%%)", stats.SkillResistPct)
	}
	// 麻痹仍然阻止行动，这条不能因为补丁属性削弱而丢失。
	if _, _, blocked := actor.actionBlockedStatus(); !blocked {
		t.Fatal("paralysis should still block action")
	}
}

// TestParalysisRestoresStatsAfterExpiry 验证状态到期后派生属性恢复，避免削弱被永久写入。
func TestParalysisRestoresStatsAfterExpiry(t *testing.T) {
	actor := &actorRuntime{
		hp:                  100,
		hpMax:               100,
		hitPct:              80,
		critDmgPct:          400,
		physicalResistPct:   60,
		statuses:            map[uint32]*statusRuntime{},
		globalMultiplierPct: 100,
	}
	if !actor.applyStatusWithProfile(StatusParalysis, 1, 0, defaultStatusProfile(StatusParalysis)) {
		t.Fatal("apply failed")
	}
	if stats := actor.effectiveStats(); stats.HitPct != 40 {
		t.Fatalf("while paralysed HitPct = %d, want 40", stats.HitPct)
	}

	delete(actor.statuses, StatusParalysis)
	actor.refreshStatusDerivedModifiers()

	stats := actor.effectiveStats()
	if stats.HitPct != 80 || stats.CritDmgPct != 400 || stats.PhysicalResistPct != 60 {
		t.Fatalf("stats after expiry = hit %d / critdmg %d / physres %d, want 80/400/60",
			stats.HitPct, stats.CritDmgPct, stats.PhysicalResistPct)
	}
}
