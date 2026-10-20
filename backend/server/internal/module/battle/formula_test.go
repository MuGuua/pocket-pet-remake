package battle

import "testing"

// TestCalculatePocketDamageMatchesExcelSample 锁定新表样例行（杰西卡徐 vs 无名麒麟）的四组结果。
func TestCalculatePocketDamageMatchesExcelSample(t *testing.T) {
	base := pocketDamageInput{
		AttackerPanel:     100000,
		SkillMult:         30,
		CritDmg:           2000,
		SkillCritAdd:      0,
		AntiCrit:          1000,
		RevSkill:          0,
		SkillRes:          50,
		DefenderPanel:     500000,
		Guard:             700,
		TalentDmgPct:      20,
		TalentReducePct:   30,
		AntiClassPct:      31,
		ElementAdvPct:     30,
		ElementPenaltyPct: 15,
		// 新表样例本身就按暴击结果取值，这里显式标记本次结算为暴击。
		IsCrit: true,
	}

	cases := []struct {
		name     string
		isAOE    bool
		adv      bool
		dis      bool
		expected int32
	}{
		{name: "单体克制", isAOE: false, adv: true, expected: 3213371},
		{name: "单体被克", isAOE: false, dis: true, expected: 2101050},
		{name: "群体克制", isAOE: true, adv: true, expected: 682841},
		{name: "群体被克", isAOE: true, dis: true, expected: 446473},
	}

	for _, tc := range cases {
		input := base
		input.IsAOE = tc.isAOE
		input.ElementAdvantaged = tc.adv
		input.ElementDisadvantaged = tc.dis
		got := calculatePocketDamage(input)
		if got != tc.expected {
			t.Fatalf("%s damage = %d, want %d", tc.name, got, tc.expected)
		}
	}
}

func TestCalculatePocketDamageNumeratorMatchesExcel(t *testing.T) {
	input := pocketDamageInput{
		AttackerPanel: 100000,
		SkillMult:     30,
		CritDmg:       2000,
		AntiCrit:      1000,
		SkillRes:      50,
		DefenderPanel: 500000,
		IsCrit:        true,
	}
	got := calculatePocketDamageNumerator(input)
	if got != 14500000 {
		t.Fatalf("numerator = %d, want 14500000", got)
	}
}

func TestScaledPanelBaseUsesAttackPctWhenSkillMultMissing(t *testing.T) {
	got := scaledPanelBase(pocketDamageInput{
		AttackerPanel:  200,
		AttackScalePct: 135,
	})
	if got != 270 {
		t.Fatalf("scaled panel = %d, want 270", got)
	}
}

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

// TestCritMultiplierPctFollowsCritFlag 锁定「未暴击固定 100、暴击走爆伤链」这条核心规则。
// 旧实现在这里无条件套用爆伤，导致暴击率属性形同虚设。
func TestCritMultiplierPctFollowsCritFlag(t *testing.T) {
	input := pocketDamageInput{CritDmg: 2000, SkillCritAdd: 0, AntiCrit: 1000}

	if got := critMultiplierPct(input); got != 100 {
		t.Fatalf("non-crit multiplier = %d, want 100", got)
	}

	input.IsCrit = true
	if got := critMultiplierPct(input); got != 1000 {
		t.Fatalf("crit multiplier = %d, want 1000", got)
	}
}

// TestCritMultiplierPctFloorsAt150 锁定「抗爆伤最多把实际爆伤压到 150%」。
func TestCritMultiplierPctFloorsAt150(t *testing.T) {
	input := pocketDamageInput{CritDmg: 200, AntiCrit: 1000, IsCrit: true}
	if got := critMultiplierPct(input); got != 150 {
		t.Fatalf("crit multiplier = %d, want floor 150", got)
	}
}

// TestCritMultiplierPctCapsAt2000 锁定爆伤上限 2000%。
func TestCritMultiplierPctCapsAt2000(t *testing.T) {
	input := pocketDamageInput{CritDmg: 5000, SkillCritAdd: 500, IsCrit: true}
	if got := critMultiplierPct(input); got != 2000 {
		t.Fatalf("crit multiplier = %d, want cap 2000", got)
	}
}

// TestNonCritDamageIsLowerThanCrit 证明暴击判定真正影响最终伤害，而不是只影响事件文案。
func TestNonCritDamageIsLowerThanCrit(t *testing.T) {
	base := pocketDamageInput{
		AttackerPanel: 100000,
		SkillMult:     30,
		CritDmg:       2000,
		AntiCrit:      1000,
		SkillRes:      50,
		DefenderPanel: 500000,
	}
	normal := calculatePocketDamage(base)

	critInput := base
	critInput.IsCrit = true
	crit := calculatePocketDamage(critInput)

	if normal >= crit {
		t.Fatalf("normal damage = %d, crit damage = %d; want normal < crit", normal, crit)
	}
}

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

func TestEffectiveStatsPanelAndSpeedModifiers(t *testing.T) {
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
	if stats.Guard != 12 {
		t.Fatalf("stats.Guard = %d, want 12", stats.Guard)
	}
}

func TestStatusDerivedModifiersAffectSpeedAndCritThenExpire(t *testing.T) {
	actor := &actorRuntime{
		hp:          40,
		hpMax:       40,
		spd:         20,
		critRatePct: 8,
		statuses:    map[uint32]*statusRuntime{},
	}

	if !actor.applyStatus(StatusSlow, 2, 50) {
		t.Fatal("apply slow status = false, want true")
	}
	if !actor.applyStatus(StatusCritBoost, 2, 25) {
		t.Fatal("apply crit boost status = false, want true")
	}

	actorStats := actor.effectiveStats()
	if actorStats.Speed != 10 {
		t.Fatalf("actorStats.Speed = %d, want 10", actorStats.Speed)
	}
	if actorStats.CritRatePct != 33 {
		t.Fatalf("actorStats.CritRatePct = %d, want 33", actorStats.CritRatePct)
	}

	battle := &activeBattle{}
	battle.expireActorStatuses(actor)
	battle.expireActorStatuses(actor)

	if actor.effectiveStats().Speed != 20 {
		t.Fatalf("expired actor speed = %d, want 20", actor.effectiveStats().Speed)
	}
	if actor.effectiveStats().CritRatePct != 8 {
		t.Fatalf("expired crit rate = %d, want 8", actor.effectiveStats().CritRatePct)
	}
}

func TestResolveActorTurnEndStatusTicksIncludesCurse(t *testing.T) {
	actor := &actorRuntime{
		actorID:      1,
		name:         "Target",
		hp:           30,
		hpMax:        30,
		lifestealPct: 50,
		statuses:     map[uint32]*statusRuntime{},
	}
	if !actor.applyStatus(StatusCurse, 2, 6) {
		t.Fatal("apply curse status = false, want true")
	}
	if !actor.applyStatus(StatusBleed, 2, 5) {
		t.Fatal("apply bleed status = false, want true")
	}
	battle := &activeBattle{
		allies: []*actorRuntime{actor},
	}

	events := battle.resolveActorTurnEndStatusTicks(actor)
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].StateID != StatusBleed {
		t.Fatalf("events[0].StateID = %d, want StatusBleed", events[0].StateID)
	}
	if events[0].Value != 5 {
		t.Fatalf("events[0].Value = %d, want 5", events[0].Value)
	}
	if events[1].StateID != StatusCurse {
		t.Fatalf("events[1].StateID = %d, want StatusCurse", events[1].StateID)
	}
	if events[1].Value != 6 {
		t.Fatalf("events[1].Value = %d, want 6", events[1].Value)
	}
	for _, event := range events {
		if event.EventType == EventTypeHeal {
			t.Fatalf("events = %#v, passive status damage should not trigger lifesteal", events)
		}
	}
	if actor.hp != 19 {
		t.Fatalf("actor.hp = %d, want 19", actor.hp)
	}
}

func TestResolveRoundTicksStatusAtEachActorTurnEnd(t *testing.T) {
	fastActor := &actorRuntime{
		actorID:    10,
		actorType:  PlayerActorType,
		unitClass:  ActorUnitClassPet,
		name:       "Fast",
		hp:         80,
		hpMax:      80,
		atk:        8,
		def:        4,
		spd:        30,
		mana:       5,
		critDmgPct: 150,
		skillIDs:   []uint32{DefaultAttackSkillID},
		statuses:   map[uint32]*statusRuntime{},
	}
	slowActor := &actorRuntime{
		actorID:    20,
		actorType:  EnemyActorType,
		unitClass:  ActorUnitClassMonster,
		name:       "Slow",
		hp:         80,
		hpMax:      80,
		atk:        8,
		def:        4,
		spd:        10,
		mana:       5,
		critDmgPct: 150,
		skillIDs:   []uint32{DefaultAttackSkillID},
		statuses:   map[uint32]*statusRuntime{},
	}
	if !fastActor.applyStatus(StatusBleed, 2, 3) {
		t.Fatal("apply fast bleed status = false, want true")
	}
	if !slowActor.applyStatus(StatusBleed, 2, 4) {
		t.Fatal("apply slow bleed status = false, want true")
	}
	battle := &activeBattle{
		battleID:      90001,
		round:         1,
		phase:         PhaseCommand,
		allies:        []*actorRuntime{fastActor},
		enemies:       []*actorRuntime{slowActor},
		plannedActs:   map[uint64]ActionRequest{},
		pendingActors: []uint64{},
		stateHistory:  []StateSnapshot{},
	}

	state, result := battle.resolveRound()
	if result != nil {
		t.Fatalf("result = %#v, want battle still running", result)
	}

	fastUseIndex := -1
	fastTickIndex := -1
	slowUseIndex := -1
	for index, event := range state.Events {
		if event.EventType == EventTypeUseSkill && event.SourceID == fastActor.actorID {
			fastUseIndex = index
		}
		if event.EventType == EventTypeStatusTick && event.TargetID == fastActor.actorID && event.StateID == StatusBleed {
			fastTickIndex = index
		}
		if event.EventType == EventTypeUseSkill && event.SourceID == slowActor.actorID {
			slowUseIndex = index
		}
	}
	if fastUseIndex == -1 || fastTickIndex == -1 || slowUseIndex == -1 {
		t.Fatalf("events = %#v, want fast use, fast tick, slow use", state.Events)
	}
	if !(fastUseIndex < fastTickIndex && fastTickIndex < slowUseIndex) {
		t.Fatalf("events = %#v, want fast bleed tick immediately after fast unit turn before slow unit acts", state.Events)
	}
}

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
	dodgeEvents := battle.resolveDamageSkill(dodgeAttacker, dodgeTarget, DefaultAttackSkillID, skillDef{AttackPct: 100}, true, true, true)
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
	lifestealEvents := battle.resolveDamageSkill(lifestealAttacker, lifestealTarget, DefaultAttackSkillID, skillDef{AttackPct: 100}, false, false, true)
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
		actorID:      6,
		name:         "Counterer",
		hp:           40,
		hpMax:        40,
		atk:          16,
		mana:         8,
		counterPct:   100,
		lifestealPct: 50,
		critDmgPct:   150,
		statuses:     map[uint32]*statusRuntime{},
	}
	counterEvents := battle.resolveDamageSkill(counterAttacker, counterTarget, DefaultAttackSkillID, skillDef{AttackPct: 100}, true, false, true)
	var sawCounter bool
	var sawCounterLifesteal bool
	for _, event := range counterEvents {
		if event.EventType == EventTypeCounter && event.SourceID == counterTarget.actorID && event.TargetID == counterAttacker.actorID {
			sawCounter = true
		}
		if event.EventType == EventTypeHeal && event.SourceID == counterTarget.actorID {
			sawCounterLifesteal = true
		}
	}
	if !sawCounter {
		t.Fatalf("counterEvents = %#v, want counter event", counterEvents)
	}
	if sawCounterLifesteal {
		t.Fatalf("counterEvents = %#v, passive counter should not trigger lifesteal", counterEvents)
	}
	if counterAttacker.hp >= 40 {
		t.Fatalf("counterAttacker.hp = %d, want reduced by counter", counterAttacker.hp)
	}
}

func TestPassiveComboReviveAndControlImmunity(t *testing.T) {
	battle := &activeBattle{battleID: 80002, round: 1}

	comboAttacker := &actorRuntime{
		actorID:      11,
		name:         "Combo",
		hp:           20,
		hpMax:        40,
		atk:          18,
		mana:         10,
		comboPct:     100,
		lifestealPct: 50,
		critDmgPct:   150,
		statuses:     map[uint32]*statusRuntime{},
	}
	comboTarget := &actorRuntime{
		actorID:  12,
		name:     "Target",
		hp:       60,
		hpMax:    60,
		statuses: map[uint32]*statusRuntime{},
	}
	comboEvents := battle.resolveDamageSkill(comboAttacker, comboTarget, DefaultAttackSkillID, skillDef{AttackPct: 100}, false, true, true)
	var sawCombo bool
	damageCount := 0
	lifestealCount := 0
	for _, event := range comboEvents {
		if event.EventType == EventTypeCombo {
			sawCombo = true
		}
		if event.EventType == EventTypeDamage {
			damageCount++
		}
		if event.EventType == EventTypeHeal && event.SourceID == comboAttacker.actorID {
			lifestealCount++
		}
	}
	if !sawCombo || damageCount < 2 {
		t.Fatalf("comboEvents = %#v, want combo event and at least two damage events", comboEvents)
	}
	if lifestealCount != 1 {
		t.Fatalf("comboEvents = %#v, want only active attack lifesteal, got %d heal events", comboEvents, lifestealCount)
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
	reviveEvents := battle.resolveDamageSkill(reviveAttacker, reviveTarget, DefaultAttackSkillID, skillDef{AttackPct: 100}, false, false, true)
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

// TestCritChancePctSubtractsTargetCritResist 锁定「攻击者暴击率 - 目标抗致命」的判定口径。
func TestCritChancePctSubtractsTargetCritResist(t *testing.T) {
	cases := []struct {
		name     string
		attacker uint32
		target   uint32
		want     uint32
	}{
		{name: "抗致命抵消", attacker: 60, target: 20, want: 40},
		{name: "抗致命更高时归零", attacker: 20, target: 60, want: 0},
		{name: "暴击率封顶 100", attacker: 150, target: 10, want: 100},
	}
	for _, tc := range cases {
		got := critChancePct(effectiveStats{CritRatePct: tc.attacker}, effectiveStats{CritResistPct: tc.target})
		if got != tc.want {
			t.Fatalf("%s: critChancePct = %d, want %d", tc.name, got, tc.want)
		}
	}
}

// TestDamageAgainstRespectsAllowCrit 锁定「技能未开启 allow_crit 时不暴击」。
func TestDamageAgainstRespectsAllowCrit(t *testing.T) {
	attacker := &actorRuntime{
		actorID:     1,
		hp:          1000,
		hpMax:       1000,
		atk:         1000,
		critRatePct: 100,
		critDmgPct:  2000,
	}
	target := &actorRuntime{
		actorID: 2,
		hp:      100000,
		hpMax:   100000,
		def:     10,
	}
	// 暴击率 100 但 allow_crit=false，结果必须不是暴击。
	_, crit := attacker.damageAgainst(target, skillDef{SkillMult: 30, AllowCrit: false}, 70002, 1)
	if crit {
		t.Fatal("crit = true, want false when allow_crit is disabled")
	}
}

// TestDamageAgainstCritsWhenChanceIsFull 验证暴击率 100 时必然暴击，且伤害高于非暴击。
func TestDamageAgainstCritsWhenChanceIsFull(t *testing.T) {
	attacker := &actorRuntime{
		actorID:     1,
		hp:          1000,
		hpMax:       1000,
		atk:         1000,
		critRatePct: 100,
		critDmgPct:  2000,
	}
	target := &actorRuntime{
		actorID: 2,
		hp:      100000,
		hpMax:   100000,
		def:     10,
	}
	skill := skillDef{SkillMult: 30, AllowCrit: true}

	critDamage, crit := attacker.damageAgainst(target, skill, 70003, 1)
	if !crit {
		t.Fatal("crit = false, want true when crit rate is 100")
	}

	attacker.critRatePct = 0
	normalDamage, normalCrit := attacker.damageAgainst(target, skill, 70003, 1)
	if normalCrit {
		t.Fatal("crit = true, want false when crit rate is 0")
	}
	if critDamage <= normalDamage {
		t.Fatalf("crit damage = %d, normal damage = %d; want crit > normal", critDamage, normalDamage)
	}
}

func TestBuildPocketDamageInputMapsRuntimeActors(t *testing.T) {
	attacker := &actorRuntime{
		unitClass:             ActorUnitClassPet,
		atk:                   100000,
		critDmgPct:            2000,
		reverseSkillResistPct: 0,
		talentDmgPct:          20,
		elementAdvPct:         30,
	}
	target := &actorRuntime{
		def:                500000,
		guard:              700,
		skillResistPct:     50,
		critDmgResistPct:   1000,
		characterResistPct: 10,
		petResistPct:       31,
		talentReducePct:    30,
		elementPenaltyPct:  15,
	}
	input := buildPocketDamageInput(attacker, target, skillDef{SkillMult: 30, TargetRule: targetEnemySingle})
	if input.AntiClassPct != 31 {
		t.Fatalf("AntiClassPct = %d, want 31", input.AntiClassPct)
	}
	if input.Guard != 700 {
		t.Fatalf("Guard = %d, want 700", input.Guard)
	}
}
