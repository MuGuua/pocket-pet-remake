package battle

import (
	"context"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/world"
)

func mustFindSnapshotByUnitClass(t *testing.T, actors []ActorSnapshot, unitClass uint32) ActorSnapshot {
	t.Helper()
	for _, actor := range actors {
		if actor.UnitClass == unitClass {
			return actor
		}
	}
	t.Fatalf("missing actor snapshot for unit class %d", unitClass)
	return ActorSnapshot{}
}

func TestSubmitBasicAttackWithoutLoadoutSkill(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()
	profile := &player.Profile{
		PlayerID: 10001,
		Name:     "DemoTrainer",
		Level:    8,
		SceneID:  1,
		PosX:     8,
		PosY:     6,
		SkillIDs: []uint32{1101},
	}
	enemy := world.Entity{EntityID: 90001, EntityType: 2, Pos: world.Vec2i{X: 10, Y: 6}, Name: "GuideNPC"}

	start, err := svc.StartPVE(ctx, profile, nil, enemy, EmptyCharacterBattleSkillInput())
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	character := mustFindSnapshotByUnitClass(t, start.Allies, ActorUnitClassCharacter)
	if len(character.SkillIDs) != 1 || character.SkillIDs[0] != 1101 {
		t.Fatalf("character.SkillIDs = %#v, want only character skill 1101", character.SkillIDs)
	}
	for _, skill := range character.Skills {
		if skill.SkillID == DefaultAttackSkillID || skill.IsBasicAttack {
			t.Fatalf("character.Skills = %#v, want basic attack excluded from snapshot", character.Skills)
		}
	}
	for _, skill := range start.Enemies[0].Skills {
		if skill.SkillID == DefaultAttackSkillID || skill.IsBasicAttack {
			t.Fatalf("enemy.Skills = %#v, want basic attack excluded from snapshot", start.Enemies[0].Skills)
		}
	}

	outcome, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    character.ActorID,
		SkillID:    DefaultAttackSkillID,
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(basic attack) error = %v", err)
	}
	if !outcome.Response.Accepted {
		t.Fatalf("outcome.Response = %#v, want accepted basic attack", outcome.Response)
	}
}

func TestServiceSubmitActionHealTargetsAlly(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()
	profile := &player.Profile{PlayerID: 10001, Name: "DemoTrainer", Level: 8, SceneID: 1, PosX: 8, PosY: 6, SkillIDs: []uint32{1101, 1001}}
	lineup := []pet.LineupPet{
		// Ally one uses a larger HP pool so the stronger post-formula enemy damage
		// still leaves room for the follow-up heal assertion in round two.
		{PetUID: 20001, PetID: 101, Level: 5, HP: 90, HPMax: 90, ATK: 10, DEF: 10, SPD: 8, MANA: 10, SkillIDs: []uint32{1001, 1002}},
		{PetUID: 20002, PetID: 102, Level: 4, HP: 28, HPMax: 30, ATK: 12, DEF: 11, SPD: 9, MANA: 20, SkillIDs: []uint32{1001, 1003}},
	}
	// 选用双怪配置避免人物+宠物三人首轮直接秒杀目标，确保治疗断言能落到下一轮。
	enemy := world.Entity{EntityID: 90006, EntityType: 2, Pos: world.Vec2i{X: 10, Y: 6}, Name: "GuideNPC"}

	start, err := svc.StartPVE(ctx, profile, lineup, enemy, EmptyCharacterBattleSkillInput())
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if len(start.Allies) != 3 {
		t.Fatalf("len(start.Allies) = %d, want 3", len(start.Allies))
	}
	healerPet := start.Allies[2]
	targetActor := start.Allies[0]
	if len(healerPet.Skills) != 1 || healerPet.Skills[0].TargetType != "ally_single" {
		t.Fatalf("healer pet skills = %#v, want heal skill with ally_single target", healerPet.Skills)
	}

	outcomeOne, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
		SkillID:    start.Allies[0].SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(round1 ally1) error = %v", err)
	}
	if outcomeOne.State != nil {
		t.Fatalf("outcomeOne.State = %#v, want nil until all round intents arrive", outcomeOne.State)
	}

	outcomeOneB, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[1].ActorID,
		SkillID:    start.Allies[1].SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(round1 pet attack) error = %v", err)
	}
	if outcomeOneB.State != nil {
		t.Fatalf("outcomeOneB.State = %#v, want nil until all round intents arrive", outcomeOneB.State)
	}

	outcomeTwo, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    healerPet.ActorID,
		SkillID:    healerPet.SkillIDs[0],
		TargetID:   targetActor.ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(round1 ally2 heal) error = %v", err)
	}
	if outcomeTwo.State == nil {
		t.Fatalf("outcomeTwo.State = nil, want round result state")
	}
	if outcomeTwo.State.Round != 2 {
		t.Fatalf("outcomeTwo.State.Round = %d, want 2", outcomeTwo.State.Round)
	}

	var sawEnemyDamageOnTargetPet bool
	for _, actor := range outcomeTwo.State.Actors {
		if actor.ActorID == targetActor.ActorID && actor.HP == 0 {
			t.Fatalf("target actor hp = 0, want alive after first round for follow-up heal test")
		}
	}
	for _, event := range outcomeTwo.State.Events {
		if event.EventType == EventTypeDamage && event.TargetID == targetActor.ActorID {
			sawEnemyDamageOnTargetPet = true
			break
		}
	}
	if !sawEnemyDamageOnTargetPet {
		t.Fatalf("expected enemy damage on target pet in round one, got %#v", outcomeTwo.State.Events)
	}

	outcomeThree, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      outcomeTwo.State.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
		SkillID:    start.Allies[0].SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(round2 character) error = %v", err)
	}
	if outcomeThree.State != nil {
		t.Fatalf("outcomeThree.State = %#v, want nil until all round two intents arrive", outcomeThree.State)
	}

	outcomeThreeB, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      outcomeTwo.State.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[1].ActorID,
		SkillID:    start.Allies[1].SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(round2 pet attack) error = %v", err)
	}
	if outcomeThreeB.State != nil {
		t.Fatalf("outcomeThreeB.State = %#v, want nil until final round two intent arrives", outcomeThreeB.State)
	}

	outcomeFour, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      outcomeTwo.State.Round,
		ActionType: ActionTypeSkill,
		ActorID:    healerPet.ActorID,
		SkillID:    healerPet.SkillIDs[0],
		TargetID:   targetActor.ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(round2 ally2 heal) error = %v", err)
	}
	if outcomeFour.State == nil {
		t.Fatalf("outcomeFour.State = nil, want state snapshot")
	}

	var sawHeal bool
	for _, event := range outcomeFour.State.Events {
		if event.EventType == EventTypeHeal && event.SourceID == healerPet.ActorID && event.TargetID == targetActor.ActorID {
			sawHeal = true
			if event.Value <= 0 {
				t.Fatalf("heal event value = %d, want positive", event.Value)
			}
		}
	}
	if !sawHeal {
		t.Fatalf("expected heal event in round two, got %#v", outcomeFour.State.Events)
	}
}

func TestServiceClientOwnedAutoBattleAndTimeout(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()
	profile := &player.Profile{PlayerID: 10001, Name: "DemoTrainer", Level: 8, SceneID: 1, PosX: 8, PosY: 6, SkillIDs: []uint32{1101, 1001}}
	lineup := []pet.LineupPet{
		{PetUID: 20001, PetID: 101, Level: 5, HP: 120, HPMax: 120, ATK: 12, DEF: 10, SPD: 30, MANA: 12, SkillIDs: []uint32{1001, 1002}},
		{PetUID: 20002, PetID: 102, Level: 4, HP: 30, HPMax: 30, ATK: 11, DEF: 11, SPD: 9, MANA: 18, SkillIDs: []uint32{1001, 1003}},
	}
	enemy := world.Entity{EntityID: 90001, EntityType: 2, Pos: world.Vec2i{X: 10, Y: 6}, Name: "GuideNPC"}

	start, err := svc.StartPVE(ctx, profile, lineup, enemy, EmptyCharacterBattleSkillInput())
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if start.CommandDeadlineMS != 0 {
		t.Fatalf("start.CommandDeadlineMS = %d, want zero because client owns command countdown", start.CommandDeadlineMS)
	}

	autoOutcome, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:          start.BattleID,
		Round:             start.Round,
		ActionType:        ActionTypeSetAuto,
		AutoBattleEnabled: true,
	})
	if err != nil {
		t.Fatalf("SubmitAction(set auto) error = %v", err)
	}
	if !autoOutcome.Response.Accepted {
		t.Fatalf("autoOutcome.Response.Accepted = false, reason=%q", autoOutcome.Response.Reason)
	}
	if autoOutcome.State != nil {
		t.Fatalf("autoOutcome.State = %#v, want nil because server ignores client auto toggle", autoOutcome.State)
	}

	svc = NewService(nil)
	start, err = svc.StartPVE(ctx, profile, lineup, enemy, EmptyCharacterBattleSkillInput())
	if err != nil {
		t.Fatalf("StartPVE(timeout case) error = %v", err)
	}
	firstOutcome, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
		SkillID:    start.Allies[0].SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(first action) error = %v", err)
	}
	if !firstOutcome.Response.Accepted {
		t.Fatalf("firstOutcome.Response.Accepted = false, reason=%q", firstOutcome.Response.Reason)
	}
	if firstOutcome.State != nil {
		t.Fatalf("firstOutcome.State = %#v, want nil until all client round intents arrive", firstOutcome.State)
	}

	battle := svc.activeByPlayer[profile.PlayerID]
	if battle == nil {
		t.Fatal("battle = nil, want active battle")
	}
	battle.commandDeadline = time.Now().Add(-time.Second)

	timeoutOutcome, err := svc.ProgressAuto(ctx, profile.PlayerID)
	if err != nil {
		t.Fatalf("ProgressAuto() error = %v", err)
	}
	if timeoutOutcome != nil {
		t.Fatalf("timeoutOutcome = %#v, want nil because server no longer owns command timeout", timeoutOutcome)
	}
	var finalOutcome *ActionOutcome
	for actorIndex := 1; actorIndex < len(start.Allies); actorIndex++ {
		outcome, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
			BattleID:   start.BattleID,
			Round:      start.Round,
			ActionType: ActionTypeSkill,
			ActorID:    start.Allies[actorIndex].ActorID,
			SkillID:    start.Allies[actorIndex].SkillIDs[0],
			TargetID:   start.Enemies[0].ActorID,
		})
		if err != nil {
			t.Fatalf("SubmitAction(remaining action %d) error = %v", actorIndex, err)
		}
		finalOutcome = outcome
	}
	if finalOutcome == nil || finalOutcome.State == nil {
		t.Fatalf("finalOutcome = %#v, want current round state after all client intents arrive", finalOutcome)
	}
	if finalOutcome.State.AutoBattleEnabled {
		t.Fatal("finalOutcome.State.AutoBattleEnabled = true, want false because server does not persist client auto state")
	}
	if finalOutcome.State.CommandDeadlineMS != 0 {
		t.Fatalf("finalOutcome.State.CommandDeadlineMS = %d, want zero", finalOutcome.State.CommandDeadlineMS)
	}
}

func TestServiceResolveDisconnectSinglePlayerBattleFailsWithoutRewards(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()
	profile := &player.Profile{PlayerID: 10001, Name: "DemoTrainer", Level: 8, SceneID: 1, PosX: 8, PosY: 6, SkillIDs: []uint32{1101, 1001}}
	lineup := []pet.LineupPet{
		{PetUID: 20001, PetID: 101, Level: 5, HP: 120, HPMax: 120, ATK: 12, DEF: 10, SPD: 30, MANA: 12, SkillIDs: []uint32{1001, 1002}},
	}
	enemy := world.Entity{EntityID: 90001, EntityType: 2, Pos: world.Vec2i{X: 10, Y: 6}, Name: "GuideNPC"}

	start, err := svc.StartPVE(ctx, profile, lineup, enemy, EmptyCharacterBattleSkillInput())
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	result := svc.ResolveDisconnect(ctx, profile.PlayerID)
	if result == nil {
		t.Fatal("ResolveDisconnect() = nil, want failed battle result")
	}
	if result.Win {
		t.Fatal("result.Win = true, want false after single-player disconnect")
	}
	if result.BattleID != start.BattleID {
		t.Fatalf("result.BattleID = %d, want %d", result.BattleID, start.BattleID)
	}
	if result.RewardGold != 0 || result.RewardPlayerExp != 0 || len(result.DropItems) != 0 || len(result.AttrRewards) != 0 {
		t.Fatalf("result rewards = gold:%d exp:%d drops:%d attrs:%d, want no rewards", result.RewardGold, result.RewardPlayerExp, len(result.DropItems), len(result.AttrRewards))
	}
	if len(result.PetResults) != 1 || result.PetResults[0].ExpGained != 0 {
		t.Fatalf("result.PetResults = %#v, want no pet exp reward", result.PetResults)
	}
	if _, _, ok := svc.GetActiveSnapshot(ctx, profile.PlayerID); ok {
		t.Fatal("GetActiveSnapshot() ok = true, want battle removed after disconnect failure")
	}
}

func TestServiceAllTargetSkillHitsMultipleEnemies(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()
	profile := &player.Profile{PlayerID: 10001, Name: "DemoTrainer", Level: 8, SceneID: 1, PosX: 8, PosY: 6, SkillIDs: []uint32{1101, 1001}}
	lineup := []pet.LineupPet{
		{PetUID: 20001, PetID: 101, Level: 5, HP: 120, HPMax: 120, ATK: 12, DEF: 10, SPD: 30, MANA: 12, SkillIDs: []uint32{1001, 1002}},
		{PetUID: 20002, PetID: 102, Level: 4, HP: 30, HPMax: 30, ATK: 11, DEF: 11, SPD: 9, MANA: 18, SkillIDs: []uint32{1001, 1003}},
	}
	enemy := world.Entity{EntityID: 90004, EntityType: 2, Pos: world.Vec2i{X: 10, Y: 6}, Name: "GuideNPC"}

	start, err := svc.StartPVE(ctx, profile, lineup, enemy, EmptyCharacterBattleSkillInput())
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if len(start.Enemies) != 2 {
		t.Fatalf("len(start.Enemies) = %d, want 2", len(start.Enemies))
	}
	damagePet := start.Allies[1]
	if damagePet.Skills[0].TargetType != "enemy_all" {
		t.Fatalf("damage pet skills = %#v, want burst skill target type enemy_all", damagePet.Skills)
	}
	if damagePet.Skills[0].AnimationKey != "burst" || damagePet.Skills[0].CastColor == "" || damagePet.Skills[0].ImpactColor == "" || !damagePet.Skills[0].Projectile {
		t.Fatalf("damage pet skill visuals = %#v, want burst animation metadata", damagePet.Skills[0])
	}

	_, err = svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    damagePet.ActorID,
		SkillID:    1002,
		TargetID:   0,
	})
	if err != nil {
		t.Fatalf("SubmitAction(all target) error = %v", err)
	}
	outcome, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[2].ActorID,
		SkillID:    1001,
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(second ally) error = %v", err)
	}
	outcome, err = svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
		SkillID:    start.Allies[0].SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(character ally) error = %v", err)
	}
	if outcome.State == nil {
		t.Fatal("outcome.State = nil, want resolved round state")
	}

	damagedTargets := map[uint64]bool{}
	for _, event := range outcome.State.Events {
		if event.EventType == EventTypeDamage && (event.TargetID == start.Enemies[0].ActorID || event.TargetID == start.Enemies[1].ActorID) {
			damagedTargets[event.TargetID] = true
		}
	}
	if len(damagedTargets) < 2 {
		t.Fatalf("damagedTargets = %#v, enemies=(%d,%d), events=%#v", damagedTargets, start.Enemies[0].ActorID, start.Enemies[1].ActorID, outcome.State.Events)
	}
}

func TestBuildPlayerCharacterActorUsesProfileATK(t *testing.T) {
	profile := &player.Profile{
		PlayerID: 40001,
		Name:     "SoloHero",
		ATK:      152,
		DEF:      20,
		HP:       200,
		HPMax:    200,
	}
	actor := buildPlayerCharacterActor(profile, PlayerActorType, EmptyCharacterBattleSkillInput())
	if actor == nil {
		t.Fatal("buildPlayerCharacterActor() = nil, want runtime actor")
	}
	if actor.atk != 152 {
		t.Fatalf("actor.atk = %d, want 152 from equipped profile", actor.atk)
	}
	stats := actor.effectiveStats()
	if stats.Attack != 152 {
		t.Fatalf("effectiveStats().Attack = %d, want 152", stats.Attack)
	}
}

func TestBuildPlayerCharacterActorIncludesRequestedAttributes(t *testing.T) {
	profile := &player.Profile{PlayerID: 40001, Name: "SoloHero"}
	actor := buildPlayerCharacterActor(profile, PlayerActorType, EmptyCharacterBattleSkillInput())
	if actor == nil {
		t.Fatal("buildPlayerCharacterActor() = nil, want runtime actor")
	}
	if actor.unitClass != ActorUnitClassCharacter {
		t.Fatalf("actor.unitClass = %d, want %d", actor.unitClass, ActorUnitClassCharacter)
	}
	if actor.hp == 0 || actor.spirit == 0 || actor.hitPct == 0 || actor.critRatePct == 0 {
		t.Fatalf("actor core stats = %#v, want non-zero hp/spirit/hit/crit", actor)
	}
	if actor.physicalResistPct == 0 || actor.skillResistPct == 0 || actor.sleepResistPct == 0 || actor.critDmgResistPct == 0 {
		t.Fatalf("actor resistances = %#v, want seeded character resistances", actor)
	}
	if len(actor.skillIDs) != 2 || actor.skillIDs[0] != DefaultCharacterSkillID {
		t.Fatalf("actor.skillIDs = %#v, want default character skills", actor.skillIDs)
	}
}

func TestExecuteDecisionFallsBackWhenSpiritInsufficient(t *testing.T) {
	actor := &actorRuntime{
		actorID:              1,
		actorType:            PlayerActorType,
		unitClass:            ActorUnitClassCharacter,
		ownerPlayerID:        10001,
		name:                 "SoloHero",
		hp:                   100,
		hpMax:                100,
		spirit:               0,
		spiritMax:            100,
		atk:                  20,
		def:                  10,
		spd:                  12,
		mana:                 18,
		hitPct:               10,
		skillIDs:             []uint32{DefaultAttackSkillID, 1002},
		critRatePct:          10,
		critDmgPct:           150,
		statuses:             map[uint32]*statusRuntime{},
		globalMultiplierPct:  100,
		attackMultiplierPct:  100,
		defenseMultiplierPct: 100,
		speedMultiplierPct:   100,
		manaMultiplierPct:    100,
	}
	target := &actorRuntime{
		actorID:              2,
		actorType:            EnemyActorType,
		unitClass:            ActorUnitClassMonster,
		name:                 "Training Dummy",
		hp:                   100,
		hpMax:                100,
		spirit:               100,
		spiritMax:            100,
		def:                  8,
		spd:                  6,
		statuses:             map[uint32]*statusRuntime{},
		globalMultiplierPct:  100,
		attackMultiplierPct:  100,
		defenseMultiplierPct: 100,
		speedMultiplierPct:   100,
		manaMultiplierPct:    100,
	}
	battle := &activeBattle{
		battleID: 70001,
		round:    1,
		allies:   []*actorRuntime{actor},
		enemies:  []*actorRuntime{target},
	}

	events := battle.executeDecision(turnDecision{
		actor: actor,
		action: ActionRequest{
			SkillID:  1002,
			TargetID: target.actorID,
		},
	})
	if len(events) == 0 {
		t.Fatal("len(events) = 0, want use-skill event")
	}
	if events[0].SkillID != DefaultAttackSkillID {
		t.Fatalf("events[0].SkillID = %d, want fallback default attack", events[0].SkillID)
	}
	if actor.spirit != 0 {
		t.Fatalf("actor.spirit = %d, want unchanged 0 after fallback", actor.spirit)
	}
}

func TestExecuteDecisionRetargetsSingleAttackWhenTargetDead(t *testing.T) {
	// 构造“提交时选中目标、出手时目标已死亡”的权威战斗态，验证服务端会重新选择存活敌人。
	actor := &actorRuntime{
		actorID:              1,
		actorType:            PlayerActorType,
		unitClass:            ActorUnitClassCharacter,
		ownerPlayerID:        10001,
		name:                 "SoloHero",
		hp:                   100,
		hpMax:                100,
		spirit:               100,
		spiritMax:            100,
		atk:                  30,
		def:                  10,
		spd:                  12,
		mana:                 18,
		hitPct:               100,
		skillIDs:             []uint32{DefaultAttackSkillID},
		critRatePct:          10,
		critDmgPct:           150,
		statuses:             map[uint32]*statusRuntime{},
		globalMultiplierPct:  100,
		attackMultiplierPct:  100,
		defenseMultiplierPct: 100,
		speedMultiplierPct:   100,
		manaMultiplierPct:    100,
	}
	deadTarget := &actorRuntime{
		actorID:              2,
		actorType:            EnemyActorType,
		unitClass:            ActorUnitClassMonster,
		name:                 "Dead Dummy",
		hp:                   0,
		hpMax:                100,
		statuses:             map[uint32]*statusRuntime{},
		globalMultiplierPct:  100,
		attackMultiplierPct:  100,
		defenseMultiplierPct: 100,
		speedMultiplierPct:   100,
		manaMultiplierPct:    100,
	}
	livingTarget := &actorRuntime{
		actorID:              3,
		actorType:            EnemyActorType,
		unitClass:            ActorUnitClassMonster,
		name:                 "Living Dummy",
		hp:                   100,
		hpMax:                100,
		statuses:             map[uint32]*statusRuntime{},
		globalMultiplierPct:  100,
		attackMultiplierPct:  100,
		defenseMultiplierPct: 100,
		speedMultiplierPct:   100,
		manaMultiplierPct:    100,
	}
	battle := &activeBattle{
		battleID: 70002,
		round:    1,
		allies:   []*actorRuntime{actor},
		enemies:  []*actorRuntime{deadTarget, livingTarget},
	}

	events := battle.executeDecision(turnDecision{
		actor: actor,
		action: ActionRequest{
			SkillID:  DefaultAttackSkillID,
			TargetID: deadTarget.actorID,
		},
	})
	if len(events) == 0 {
		t.Fatal("len(events) = 0, want use-skill event after retarget")
	}
	if events[0].TargetID != livingTarget.actorID {
		t.Fatalf("events[0].TargetID = %d, want living target %d", events[0].TargetID, livingTarget.actorID)
	}
	for _, event := range events {
		if event.TargetID == deadTarget.actorID {
			t.Fatalf("event targeted dead actor: %#v", event)
		}
	}
}

func TestExecuteDecisionMultiTargetUsesLivingTargetsAtCastTime(t *testing.T) {
	// 多目标技能不信任客户端预选目标，出手瞬间应只从当前存活敌人中抽取命中目标。
	actor := &actorRuntime{
		actorID:              1,
		actorType:            PlayerActorType,
		unitClass:            ActorUnitClassPet,
		ownerPlayerID:        10001,
		name:                 "VolleyPet",
		hp:                   100,
		hpMax:                100,
		spirit:               100,
		spiritMax:            100,
		atk:                  30,
		def:                  10,
		spd:                  12,
		mana:                 18,
		hitPct:               100,
		skillIDs:             []uint32{DefaultAttackSkillID, 1004},
		critRatePct:          10,
		critDmgPct:           150,
		statuses:             map[uint32]*statusRuntime{},
		globalMultiplierPct:  100,
		attackMultiplierPct:  100,
		defenseMultiplierPct: 100,
		speedMultiplierPct:   100,
		manaMultiplierPct:    100,
	}
	deadTarget := &actorRuntime{
		actorID:              2,
		actorType:            EnemyActorType,
		unitClass:            ActorUnitClassMonster,
		name:                 "Dead Dummy",
		hp:                   0,
		hpMax:                100,
		statuses:             map[uint32]*statusRuntime{},
		globalMultiplierPct:  100,
		attackMultiplierPct:  100,
		defenseMultiplierPct: 100,
		speedMultiplierPct:   100,
		manaMultiplierPct:    100,
	}
	livingTarget := &actorRuntime{
		actorID:              3,
		actorType:            EnemyActorType,
		unitClass:            ActorUnitClassMonster,
		name:                 "Living Dummy",
		hp:                   100,
		hpMax:                100,
		statuses:             map[uint32]*statusRuntime{},
		globalMultiplierPct:  100,
		attackMultiplierPct:  100,
		defenseMultiplierPct: 100,
		speedMultiplierPct:   100,
		manaMultiplierPct:    100,
	}
	battle := &activeBattle{
		battleID: 70003,
		round:    1,
		allies:   []*actorRuntime{actor},
		enemies:  []*actorRuntime{deadTarget, livingTarget},
	}

	events := battle.executeDecision(turnDecision{
		actor: actor,
		action: ActionRequest{
			SkillID:  1004,
			TargetID: deadTarget.actorID,
		},
	})
	if len(events) == 0 {
		t.Fatal("len(events) = 0, want use-skill event after living target selection")
	}
	if events[0].TargetID != livingTarget.actorID {
		t.Fatalf("events[0].TargetID = %d, want living target %d", events[0].TargetID, livingTarget.actorID)
	}
	for _, event := range events {
		if event.TargetID == deadTarget.actorID {
			t.Fatalf("event targeted dead actor: %#v", event)
		}
	}
}

func TestAdjustStatusChancePctUsesSpecificResistance(t *testing.T) {
	battle := &activeBattle{}
	target := &actorRuntime{
		curseResistPct: 5,
	}

	if got := battle.adjustStatusChancePct(10, target, StatusCurse); got != 5 {
		t.Fatalf("adjustStatusChancePct(curse) = %d, want 5", got)
	}
	if got := resolveControlApplyChance(35, 0, target.sealResistPct); got != 35 {
		t.Fatalf("resolveControlApplyChance probability = %d, want 35 ignoring resist", got)
	}
	if got := resolveControlApplyChance(0, 350, 300); got != 100 {
		t.Fatalf("resolveControlApplyChance power = %d, want 100", got)
	}
}

func TestServiceMultiTargetSkillHitsConfiguredEnemyCount(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()
	profile := &player.Profile{PlayerID: 10001, Name: "DemoTrainer", Level: 8, SceneID: 1, PosX: 8, PosY: 6, SkillIDs: []uint32{1101, 1001}}
	lineup := []pet.LineupPet{
		// 第一只宠物显式携带双目标技能，用来验证服务端会在出手时按存活敌人补足命中目标。
		{PetUID: 20001, PetID: 101, Level: 5, HP: 120, HPMax: 120, ATK: 12, DEF: 10, SPD: 30, MANA: 12, SkillIDs: []uint32{1001, 1004}},
		{PetUID: 20002, PetID: 102, Level: 4, HP: 30, HPMax: 30, ATK: 11, DEF: 11, SPD: 9, MANA: 18, SkillIDs: []uint32{1001, 1003}},
	}
	enemy := world.Entity{EntityID: 90004, EntityType: 2, Pos: world.Vec2i{X: 10, Y: 6}, Name: "GuideNPC"}

	start, err := svc.StartPVE(ctx, profile, lineup, enemy, EmptyCharacterBattleSkillInput())
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if len(start.Enemies) != 2 {
		t.Fatalf("len(start.Enemies) = %d, want 2", len(start.Enemies))
	}
	multiTargetPet := start.Allies[1]
	if len(multiTargetPet.Skills) != 1 || multiTargetPet.Skills[0].TargetType != "enemy_multi" {
		t.Fatalf("pet skills = %#v, want volley skill target type enemy_multi", multiTargetPet.Skills)
	}
	if multiTargetPet.Skills[0].TargetCount != 2 {
		t.Fatalf("multiTargetPet.Skills[0].TargetCount = %d, want 2", multiTargetPet.Skills[0].TargetCount)
	}
	if multiTargetPet.Skills[0].AnimationKey != "volley" || multiTargetPet.Skills[0].CastColor == "" || multiTargetPet.Skills[0].ImpactColor == "" || !multiTargetPet.Skills[0].Projectile {
		t.Fatalf("multiTargetPet.Skills[0] visuals = %#v, want volley animation metadata", multiTargetPet.Skills[0])
	}

	_, err = svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    multiTargetPet.ActorID,
		SkillID:    1004,
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(multi target) error = %v", err)
	}
	outcome, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[2].ActorID,
		SkillID:    1001,
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(second ally) error = %v", err)
	}
	outcome, err = svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
		SkillID:    start.Allies[0].SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(character ally) error = %v", err)
	}
	if outcome.State == nil {
		t.Fatal("outcome.State = nil, want resolved round state")
	}

	damagedTargets := map[uint64]bool{}
	for _, event := range outcome.State.Events {
		if event.EventType == EventTypeDamage && (event.TargetID == start.Enemies[0].ActorID || event.TargetID == start.Enemies[1].ActorID) {
			damagedTargets[event.TargetID] = true
		}
	}
	if len(damagedTargets) < 2 {
		t.Fatalf("damagedTargets = %#v, enemies=(%d,%d), events=%#v", damagedTargets, start.Enemies[0].ActorID, start.Enemies[1].ActorID, outcome.State.Events)
	}
}

func TestServiceStartPVPWaitsForBothPlayers(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()
	challenger := &player.Profile{PlayerID: 10001, Name: "DemoTrainer", Level: 8, SceneID: 1, PosX: 8, PosY: 6}
	defender := &player.Profile{PlayerID: 10002, Name: "RivalTrainer", Level: 8, SceneID: 1, PosX: 9, PosY: 6}
	challengerLineup := []pet.LineupPet{
		{PetUID: 20001, PetID: 101, Level: 5, HP: 120, HPMax: 120, ATK: 12, DEF: 10, SPD: 30, MANA: 12, SkillIDs: []uint32{1001, 1002}},
	}
	defenderLineup := []pet.LineupPet{
		{PetUID: 21001, PetID: 101, Level: 5, HP: 110, HPMax: 110, ATK: 12, DEF: 10, SPD: 28, MANA: 12, SkillIDs: []uint32{1001, 1002}},
	}

	start, err := svc.StartPVP(ctx, challenger, challengerLineup, defender, defenderLineup)
	if err != nil {
		t.Fatalf("StartPVP() error = %v", err)
	}
	if start.BattleType != BattleTypePVP {
		t.Fatalf("start.BattleType = %d, want %d", start.BattleType, BattleTypePVP)
	}
	if len(start.ParticipantPlayerIDs) != 2 {
		t.Fatalf("len(start.ParticipantPlayerIDs) = %d, want 2", len(start.ParticipantPlayerIDs))
	}
	if len(start.PendingActorIDs) != 2 {
		t.Fatalf("len(start.PendingActorIDs) = %d, want 2", len(start.PendingActorIDs))
	}
	if start.Enemies[0].OwnerPlayerID != defender.PlayerID {
		t.Fatalf("start.Enemies[0].OwnerPlayerID = %d, want %d", start.Enemies[0].OwnerPlayerID, defender.PlayerID)
	}

	outcomeOne, err := svc.SubmitAction(ctx, challenger.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
		SkillID:    1001,
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(challenger) error = %v", err)
	}
	if outcomeOne.State != nil {
		t.Fatalf("outcomeOne.State = %#v, want nil until defender round intent arrives", outcomeOne.State)
	}

	outcomeTwo, err := svc.SubmitAction(ctx, defender.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Enemies[0].ActorID,
		SkillID:    1001,
		TargetID:   start.Allies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(defender) error = %v", err)
	}
	if outcomeTwo.State == nil {
		t.Fatal("outcomeTwo.State = nil, want resolved shared state")
	}
	if outcomeTwo.State.Round != 2 {
		t.Fatalf("outcomeTwo.State.Round = %d, want 2", outcomeTwo.State.Round)
	}
	if len(outcomeTwo.State.ParticipantPlayerIDs) != 2 {
		t.Fatalf("len(outcomeTwo.State.ParticipantPlayerIDs) = %d, want 2", len(outcomeTwo.State.ParticipantPlayerIDs))
	}
}

func TestServiceStartPVEIncludesCharacterWithoutLineup(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()
	profile := &player.Profile{
		PlayerID: 50001,
		Name:     "SoloHero",
		Level:    6,
		SceneID:  1,
		PosX:     4,
		PosY:     5,
		SkillIDs: []uint32{1101, 1001},
	}
	enemy := world.Entity{EntityID: 90001, EntityType: 2, Pos: world.Vec2i{X: 5, Y: 5}, Name: "GuideNPC"}

	start, err := svc.StartPVE(ctx, profile, nil, enemy, EmptyCharacterBattleSkillInput())
	if err != nil {
		t.Fatalf("StartPVE(no lineup) error = %v", err)
	}
	if len(start.Allies) != 1 {
		t.Fatalf("len(start.Allies) = %d, want 1 character actor", len(start.Allies))
	}
	character := mustFindSnapshotByUnitClass(t, start.Allies, ActorUnitClassCharacter)
	if character.ActorID != profile.PlayerID {
		t.Fatalf("character.ActorID = %d, want %d", character.ActorID, profile.PlayerID)
	}
	if start.ActiveActorID != character.ActorID {
		t.Fatalf("start.ActiveActorID = %d, want %d", start.ActiveActorID, character.ActorID)
	}
	if start.ActivePetUID != 0 {
		t.Fatalf("start.ActivePetUID = %d, want 0 for character turn", start.ActivePetUID)
	}
}

func TestResolveRoundSkipsUnactedAlliesWhenEnemiesEliminated(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()
	profile := &player.Profile{
		PlayerID: 60001,
		Name:     "BurstHero",
		Level:    20,
		SceneID:  1,
		PosX:     8,
		PosY:     6,
		SkillIDs: []uint32{DefaultCharacterSkillID},
		ATK:      200,
		SPD:      30,
	}
	lineup := []pet.LineupPet{
		{PetUID: 70001, PetID: 101, Level: 10, HP: 80, HPMax: 80, ATK: 12, DEF: 10, SPD: 5, MANA: 20, SkillIDs: []uint32{1002}},
	}
	enemy := world.Entity{EntityID: 90001, EntityType: 2, Pos: world.Vec2i{X: 10, Y: 6}, Name: "WeakMob"}

	start, err := svc.StartPVE(ctx, profile, lineup, enemy, EmptyCharacterBattleSkillInput())
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if len(start.Allies) != 2 {
		t.Fatalf("len(start.Allies) = %d, want character + pet", len(start.Allies))
	}
	character := mustFindSnapshotByUnitClass(t, start.Allies, ActorUnitClassCharacter)
	petActor := mustFindSnapshotByUnitClass(t, start.Allies, ActorUnitClassPet)

	outcomeA, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    character.ActorID,
		SkillID:    DefaultCharacterSkillID,
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(character) error = %v", err)
	}
	if outcomeA.State != nil {
		t.Fatalf("outcomeA.State = %#v, want nil until pet round intent arrives", outcomeA.State)
	}

	outcomeB, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    petActor.ActorID,
		SkillID:    1002,
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(pet) error = %v", err)
	}
	if outcomeB.Result == nil || !outcomeB.Result.Win {
		t.Fatalf("outcomeB.Result = %#v, want immediate win", outcomeB.Result)
	}
	if outcomeB.State == nil {
		t.Fatal("outcomeB.State = nil, want resolved round state")
	}
	for _, event := range outcomeB.State.Events {
		if event.EventType == EventTypeUseSkill && event.SourceID == petActor.ActorID {
			t.Fatalf("events = %#v, pet should not consume skill after enemies eliminated", outcomeB.State.Events)
		}
	}
	for _, actorState := range outcomeB.State.Actors {
		if actorState.ActorID != petActor.ActorID {
			continue
		}
		if actorState.Spirit != petActor.Spirit {
			t.Fatalf("pet spirit = %d, want unchanged %d", actorState.Spirit, petActor.Spirit)
		}
	}
}
