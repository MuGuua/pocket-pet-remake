package battle

import "testing"

// This test locks the stat-by-stat breakdown so future skill extensions do not
// silently drop one coefficient from the final base damage sum.
func TestCalculateBaseDamageIncludesConfiguredStatParts(t *testing.T) {
	attacker := effectiveStats{
		Attack:    20,
		Mana:      16,
		Defense:   15,
		Speed:     10,
		CurrentHP: 40,
		MaxHP:     40,
	}
	target := effectiveStats{
		CurrentHP: 50,
		MaxHP:     80,
	}
	skill := skillDef{
		AttackPct:          100,
		ManaPct:            50,
		DefensePct:         50,
		SpeedPct:           30,
		TargetCurrentHPPct: 20,
		FixedDamage:        7,
	}

	breakdown := calculateBaseDamage(attacker, target, skill)

	if breakdown.AttackPart != 20 {
		t.Fatalf("AttackPart = %d, want 20", breakdown.AttackPart)
	}
	if breakdown.DefensePart != 7 {
		t.Fatalf("DefensePart = %d, want 7", breakdown.DefensePart)
	}
	if breakdown.ManaPart != 8 {
		t.Fatalf("ManaPart = %d, want 8", breakdown.ManaPart)
	}
	if breakdown.SpeedPart != 3 {
		t.Fatalf("SpeedPart = %d, want 3", breakdown.SpeedPart)
	}
	if breakdown.CurrentHPPart != 10 {
		t.Fatalf("CurrentHPPart = %d, want 10", breakdown.CurrentHPPart)
	}
	if breakdown.FixedPart != 7 {
		t.Fatalf("FixedPart = %d, want 7", breakdown.FixedPart)
	}
	if breakdown.Total != 55 {
		t.Fatalf("Total = %d, want 55", breakdown.Total)
	}
}

// This test verifies the defense branch that the design doc calls out most
// explicitly: 90% cap, ignore-defense override, and modifier stacking.
func TestCalculateDefenseReductionHonorsCapsAndModifiers(t *testing.T) {
	target := effectiveStats{Defense: 10000}

	reduction := calculateDefenseReduction(target, skillDef{})
	if reduction != 0.90 {
		t.Fatalf("reduction = %.2f, want 0.90 cap", reduction)
	}

	ignoreDefense := calculateDefenseReduction(target, skillDef{IgnoreDefense: true})
	if ignoreDefense != 0 {
		t.Fatalf("ignoreDefense = %.2f, want 0", ignoreDefense)
	}

	armorBreakAndVulnerability := calculateDefenseReduction(
		effectiveStats{Defense: 100},
		skillDef{ArmorBreakPct: 50, VulnerabilityPct: 10},
	)
	if armorBreakAndVulnerability != 0 {
		t.Fatalf("armorBreakAndVulnerability = %.4f, want 0 after vulnerability clamp", armorBreakAndVulnerability)
	}
}

// This test keeps malformed crit configuration data from bypassing the
// recommended upper and lower bounds in the battle design doc.
func TestClampCritValues(t *testing.T) {
	if got := clampCritRatePct(180); got != 100 {
		t.Fatalf("clampCritRatePct(180) = %d, want 100", got)
	}
	if got := clampCritDmgPct(50); got != 100 {
		t.Fatalf("clampCritDmgPct(50) = %d, want 100", got)
	}
	if got := clampCritDmgPct(5000); got != 2000 {
		t.Fatalf("clampCritDmgPct(5000) = %d, want 2000", got)
	}
}

// This test protects the current healing rule: percent-based recovery plus
// optional fixed value, with a floor of 1 so support skills never "whiff".
func TestCalculateHealAmountRespectsPercentAndFloor(t *testing.T) {
	caster := effectiveStats{MaxHP: 80}

	heal := calculateHealAmount(caster, skillDef{HealPct: 25, FixedHeal: 3})
	if heal != 23 {
		t.Fatalf("heal = %d, want 23", heal)
	}

	minimumHeal := calculateHealAmount(caster, skillDef{})
	if minimumHeal != 1 {
		t.Fatalf("minimumHeal = %d, want 1", minimumHeal)
	}
}

// This test verifies the two new extension points needed by the formula phase:
// effective stat multipliers and post-defense block reduction.
func TestEffectiveStatsAndBlockReduction(t *testing.T) {
	actor := &actorRuntime{
		atk:                  20,
		def:                  10,
		spd:                  8,
		mana:                 12,
		globalMultiplierPct:  120,
		attackMultiplierPct:  150,
		defenseMultiplierPct: 100,
		speedMultiplierPct:   50,
		manaMultiplierPct:    200,
		attackFlatBonus:      4,
	}

	stats := actor.effectiveStats()
	if stats.Attack != 43 {
		t.Fatalf("stats.Attack = %d, want 43", stats.Attack)
	}
	if stats.Speed != 5 {
		t.Fatalf("stats.Speed = %d, want 5", stats.Speed)
	}
	if stats.Mana != 29 {
		t.Fatalf("stats.Mana = %d, want 29", stats.Mana)
	}

	blockReduction := calculateBlockReduction(
		effectiveStats{Attack: 10},
		effectiveStats{GenericBlockPct: 5, PetBlockPct: 18},
	)
	if blockReduction != 0.18 {
		t.Fatalf("blockReduction = %.2f, want 0.18", blockReduction)
	}

	damage := calculateFinalDamage(100, 0.20, blockReduction)
	if damage != 66 {
		t.Fatalf("damage = %d, want 66", damage)
	}
}

// This test verifies that runtime battle statuses now feed back into the
// formula layer instead of only existing as UI markers.
func TestStatusDerivedModifiersAffectFormulaAndExpireCleanly(t *testing.T) {
	actor := &actorRuntime{
		hp:          40,
		hpMax:       40,
		spd:         20,
		critRatePct: 8,
		statuses:    map[uint32]*statusRuntime{},
	}
	target := &actorRuntime{
		hp:       40,
		hpMax:    40,
		def:      100,
		statuses: map[uint32]*statusRuntime{},
	}

	if !actor.applyStatus(StatusSlow, 2, 50) {
		t.Fatal("apply slow status = false, want true")
	}
	if !actor.applyStatus(StatusCritBoost, 2, 25) {
		t.Fatal("apply crit boost status = false, want true")
	}
	if !target.applyStatus(StatusVulnerability, 2, 15) {
		t.Fatal("apply vulnerability status = false, want true")
	}
	if !target.applyStatus(StatusArmorBreak, 2, 0) {
		t.Fatal("apply armor break status = false, want true")
	}

	actorStats := actor.effectiveStats()
	if actorStats.Speed != 10 {
		t.Fatalf("actorStats.Speed = %d, want 10", actorStats.Speed)
	}
	if actorStats.CritRatePct != 33 {
		t.Fatalf("actorStats.CritRatePct = %d, want 33", actorStats.CritRatePct)
	}

	targetStats := target.effectiveStats()
	if !targetStats.ArmorBroken {
		t.Fatal("targetStats.ArmorBroken = false, want true")
	}
	if targetStats.VulnerabilityPct != 15 {
		t.Fatalf("targetStats.VulnerabilityPct = %d, want 15", targetStats.VulnerabilityPct)
	}

	reduction := calculateDefenseReduction(targetStats, skillDef{})
	if reduction != 0 {
		t.Fatalf("reduction = %.2f, want 0 when armor break is active", reduction)
	}

	battle := &activeBattle{
		allies:  []*actorRuntime{actor},
		enemies: []*actorRuntime{target},
	}
	battle.expireRoundStatuses()
	battle.expireRoundStatuses()

	if actor.effectiveStats().Speed != 20 {
		t.Fatalf("expired actor speed = %d, want 20", actor.effectiveStats().Speed)
	}
	if actor.effectiveStats().CritRatePct != 8 {
		t.Fatalf("expired crit rate = %d, want 8", actor.effectiveStats().CritRatePct)
	}
	if target.effectiveStats().ArmorBroken {
		t.Fatal("expired armor break still active")
	}
	if target.effectiveStats().VulnerabilityPct != 0 {
		t.Fatalf("expired vulnerability = %d, want 0", target.effectiveStats().VulnerabilityPct)
	}
}

// This test covers the newly added persistent-damage status so future control
// system work does not accidentally drop curse ticks from round-end resolution.
func TestResolveStatusTicksIncludesCurse(t *testing.T) {
	actor := &actorRuntime{
		actorID:  1,
		name:     "Target",
		hp:       30,
		hpMax:    30,
		statuses: map[uint32]*statusRuntime{},
	}
	if !actor.applyStatus(StatusCurse, 2, 6) {
		t.Fatal("apply curse status = false, want true")
	}
	battle := &activeBattle{
		allies: []*actorRuntime{actor},
	}

	events := battle.resolveStatusTicks()
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].StateID != StatusCurse {
		t.Fatalf("events[0].StateID = %d, want StatusCurse", events[0].StateID)
	}
	if events[0].Value != 6 {
		t.Fatalf("events[0].Value = %d, want 6", events[0].Value)
	}
	if actor.hp != 24 {
		t.Fatalf("actor.hp = %d, want 24", actor.hp)
	}
}

// This test verifies that the broader control-state family now influences turn
// resolution and not only the old stun path.
func TestActionBlockedStatusAndConfusionTargeting(t *testing.T) {
	actor := &actorRuntime{
		actorID:  100,
		hp:       40,
		hpMax:    40,
		name:     "Actor",
		statuses: map[uint32]*statusRuntime{},
	}
	if !actor.applyStatus(StatusSleep, 1, 0) {
		t.Fatal("apply sleep status = false, want true")
	}
	statusID, _, blocked := actor.actionBlockedStatus()
	if !blocked || statusID != StatusSleep {
		t.Fatalf("blocked = %v, statusID = %d, want sleep block", blocked, statusID)
	}

	confused := &actorRuntime{
		actorID:  200,
		hp:       40,
		hpMax:    40,
		name:     "Confused",
		statuses: map[uint32]*statusRuntime{},
	}
	ally := &actorRuntime{actorID: 201, hp: 30, hpMax: 30, name: "Ally", statuses: map[uint32]*statusRuntime{}}
	enemy := &actorRuntime{actorID: 301, hp: 30, hpMax: 30, name: "Enemy", statuses: map[uint32]*statusRuntime{}}
	if !confused.applyStatus(StatusConfusion, 1, 0) {
		t.Fatal("apply confusion status = false, want true")
	}
	battle := &activeBattle{
		battleID: 70123,
		round:    2,
		allies:   []*actorRuntime{confused, ally},
		enemies:  []*actorRuntime{enemy},
	}

	target := battle.resolveDecisionTarget(confused, enemy.actorID, skillDef{TargetRule: targetEnemySingle})
	if target == nil {
		t.Fatal("confusion target = nil, want some living target")
	}
	if target.actorID == confused.actorID {
		t.Fatal("confusion targeted self, want another unit")
	}
}

// This test verifies the first batch of passive hooks: dodge, lifesteal, and
// counter all run on the authoritative server inside one damage resolution.
func TestResolveDamageSkillPassiveHooks(t *testing.T) {
	battle := &activeBattle{battleID: 80001, round: 1}

	dodgeAttacker := &actorRuntime{
		actorID:    1,
		name:       "Attacker",
		hp:         30,
		hpMax:      40,
		atk:        15,
		mana:       10,
		critDmgPct: 150,
		statuses:   map[uint32]*statusRuntime{},
	}
	dodgeTarget := &actorRuntime{
		actorID:  2,
		name:     "DodgeTarget",
		hp:       30,
		hpMax:    30,
		dodgePct: 100,
		statuses: map[uint32]*statusRuntime{},
	}
	dodgeEvents := battle.resolveDamageSkill(dodgeAttacker, dodgeTarget, DefaultAttackSkillID, skillDef{AttackPct: 100}, true, true)
	if len(dodgeEvents) != 1 || dodgeEvents[0].EventType != EventTypeDodge {
		t.Fatalf("dodgeEvents = %#v, want single dodge event", dodgeEvents)
	}

	lifestealAttacker := &actorRuntime{
		actorID:      3,
		name:         "Lifesteal",
		hp:           20,
		hpMax:        40,
		atk:          20,
		mana:         10,
		lifestealPct: 50,
		critDmgPct:   150,
		statuses:     map[uint32]*statusRuntime{},
	}
	lifestealTarget := &actorRuntime{
		actorID:  4,
		name:     "Victim",
		hp:       40,
		hpMax:    40,
		statuses: map[uint32]*statusRuntime{},
	}
	lifestealEvents := battle.resolveDamageSkill(lifestealAttacker, lifestealTarget, DefaultAttackSkillID, skillDef{AttackPct: 100}, false, false)
	var sawLifesteal bool
	for _, event := range lifestealEvents {
		if event.EventType == EventTypeHeal && event.SourceID == lifestealAttacker.actorID && event.TargetID == lifestealAttacker.actorID {
			sawLifesteal = true
		}
	}
	if !sawLifesteal {
		t.Fatalf("lifestealEvents = %#v, want self-heal event", lifestealEvents)
	}
	if lifestealAttacker.hp <= 20 {
		t.Fatalf("lifestealAttacker.hp = %d, want greater than 20", lifestealAttacker.hp)
	}

	counterAttacker := &actorRuntime{
		actorID:    5,
		name:       "Striker",
		hp:         40,
		hpMax:      40,
		atk:        18,
		mana:       10,
		critDmgPct: 150,
		statuses:   map[uint32]*statusRuntime{},
	}
	counterTarget := &actorRuntime{
		actorID:    6,
		name:       "Counterer",
		hp:         40,
		hpMax:      40,
		atk:        16,
		mana:       8,
		counterPct: 100,
		critDmgPct: 150,
		statuses:   map[uint32]*statusRuntime{},
	}
	counterEvents := battle.resolveDamageSkill(counterAttacker, counterTarget, DefaultAttackSkillID, skillDef{AttackPct: 100}, true, false)
	var sawCounter bool
	for _, event := range counterEvents {
		if event.EventType == EventTypeCounter && event.SourceID == counterTarget.actorID && event.TargetID == counterAttacker.actorID {
			sawCounter = true
		}
	}
	if !sawCounter {
		t.Fatalf("counterEvents = %#v, want counter event", counterEvents)
	}
	if counterAttacker.hp >= 40 {
		t.Fatalf("counterAttacker.hp = %d, want reduced by counter", counterAttacker.hp)
	}
}

// This test locks the second passive batch: combo adds one extra attack,
// revive cancels a death once, and control immunity blocks new control states.
func TestPassiveComboReviveAndControlImmunity(t *testing.T) {
	battle := &activeBattle{battleID: 80002, round: 1}

	comboAttacker := &actorRuntime{
		actorID:    11,
		name:       "Combo",
		hp:         40,
		hpMax:      40,
		atk:        18,
		mana:       10,
		comboPct:   100,
		critDmgPct: 150,
		statuses:   map[uint32]*statusRuntime{},
	}
	comboTarget := &actorRuntime{
		actorID:  12,
		name:     "Target",
		hp:       60,
		hpMax:    60,
		statuses: map[uint32]*statusRuntime{},
	}
	comboEvents := battle.resolveDamageSkill(comboAttacker, comboTarget, DefaultAttackSkillID, skillDef{AttackPct: 100}, false, true)
	var sawCombo bool
	damageCount := 0
	for _, event := range comboEvents {
		if event.EventType == EventTypeCombo {
			sawCombo = true
		}
		if event.EventType == EventTypeDamage {
			damageCount++
		}
	}
	if !sawCombo || damageCount < 2 {
		t.Fatalf("comboEvents = %#v, want combo event and at least two damage events", comboEvents)
	}

	reviveAttacker := &actorRuntime{
		actorID:    13,
		name:       "Finisher",
		hp:         40,
		hpMax:      40,
		atk:        40,
		mana:       20,
		critDmgPct: 150,
		statuses:   map[uint32]*statusRuntime{},
	}
	reviveTarget := &actorRuntime{
		actorID:     14,
		name:        "Reviver",
		hp:          15,
		hpMax:       40,
		revivePct:   100,
		reviveHPPct: 50,
		critDmgPct:  150,
		statuses:    map[uint32]*statusRuntime{},
	}
	reviveEvents := battle.resolveDamageSkill(reviveAttacker, reviveTarget, DefaultAttackSkillID, skillDef{AttackPct: 100}, false, false)
	var sawRevive bool
	var sawDefeat bool
	for _, event := range reviveEvents {
		if event.EventType == EventTypeRevive {
			sawRevive = true
		}
		if event.EventType == EventTypeDefeat {
			sawDefeat = true
		}
	}
	if !sawRevive || sawDefeat {
		t.Fatalf("reviveEvents = %#v, want revive without defeat", reviveEvents)
	}
	if reviveTarget.hp == 0 {
		t.Fatal("reviveTarget.hp = 0, want restored hp")
	}

	immuneTarget := &actorRuntime{
		actorID:       15,
		name:          "Immune",
		hp:            30,
		hpMax:         30,
		controlImmune: true,
		statuses:      map[uint32]*statusRuntime{},
	}
	if immuneTarget.applyStatus(StatusSleep, 1, 0) {
		t.Fatal("applyStatus sleep succeeded on control-immune target, want false")
	}
	if immuneTarget.hasStatus(StatusSleep) {
		t.Fatal("control-immune target still has sleep status")
	}
}

// This test preserves the design choice that pure fixed-damage skills behave
// like non-crit true damage even if the caster has a guaranteed crit rate.
func TestPureFixedDamageDoesNotCrit(t *testing.T) {
	attacker := &actorRuntime{
		actorID:     1,
		hp:          30,
		hpMax:       30,
		critRatePct: 100,
		critDmgPct:  500,
	}
	target := &actorRuntime{
		actorID: 2,
		hp:      30,
		hpMax:   30,
	}

	damage, crit := attacker.damageAgainst(target, skillDef{
		FixedDamage: 10,
		AllowCrit:   true,
	}, 70001, 1)

	if crit {
		t.Fatalf("crit = true, want false for pure fixed damage")
	}
	if damage != 10 {
		t.Fatalf("damage = %d, want 10", damage)
	}
}
