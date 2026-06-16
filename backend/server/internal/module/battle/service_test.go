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

	start, err := svc.StartPVE(ctx, profile, lineup, enemy)
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if len(start.Allies) != 3 {
		t.Fatalf("len(start.Allies) = %d, want 3", len(start.Allies))
	}
	healerPet := start.Allies[2]
	targetActor := start.Allies[0]
	if len(healerPet.Skills) != 2 || healerPet.Skills[1].TargetType != "ally_single" {
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
	if outcomeOne.State == nil || len(outcomeOne.State.PendingActorIDs) != 2 {
		t.Fatalf("outcomeOne.State.PendingActorIDs = %#v, want two remaining allies", outcomeOne.State)
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
	if outcomeOneB.State == nil || len(outcomeOneB.State.PendingActorIDs) != 1 {
		t.Fatalf("outcomeOneB.State.PendingActorIDs = %#v, want one remaining ally", outcomeOneB.State)
	}

	outcomeTwo, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    healerPet.ActorID,
		SkillID:    healerPet.SkillIDs[1],
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
	if outcomeThree.State == nil || len(outcomeThree.State.PendingActorIDs) != 2 {
		t.Fatalf("outcomeThree pending = %#v, want two allies still pending", outcomeThree.State)
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
	if outcomeThreeB.State == nil || len(outcomeThreeB.State.PendingActorIDs) != 1 {
		t.Fatalf("outcomeThreeB pending = %#v, want one ally still pending", outcomeThreeB.State)
	}

	outcomeFour, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      outcomeTwo.State.Round,
		ActionType: ActionTypeSkill,
		ActorID:    healerPet.ActorID,
		SkillID:    healerPet.SkillIDs[1],
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

func TestServiceAutoBattleAndTimeoutProgress(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()
	profile := &player.Profile{PlayerID: 10001, Name: "DemoTrainer", Level: 8, SceneID: 1, PosX: 8, PosY: 6, SkillIDs: []uint32{1101, 1001}}
	lineup := []pet.LineupPet{
		{PetUID: 20001, PetID: 101, Level: 5, HP: 120, HPMax: 120, ATK: 12, DEF: 10, SPD: 30, MANA: 12, SkillIDs: []uint32{1001, 1002}},
		{PetUID: 20002, PetID: 102, Level: 4, HP: 30, HPMax: 30, ATK: 11, DEF: 11, SPD: 9, MANA: 18, SkillIDs: []uint32{1001, 1003}},
	}
	enemy := world.Entity{EntityID: 90001, EntityType: 2, Pos: world.Vec2i{X: 10, Y: 6}, Name: "GuideNPC"}

	start, err := svc.StartPVE(ctx, profile, lineup, enemy)
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if start.CommandDeadlineMS == 0 {
		t.Fatal("start.CommandDeadlineMS = 0, want non-zero server deadline")
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
	if autoOutcome.State == nil {
		t.Fatal("autoOutcome.State = nil, want resolved state")
	}
	if !autoOutcome.State.AutoBattleEnabled {
		t.Fatal("autoOutcome.State.AutoBattleEnabled = false, want true")
	}
	if autoOutcome.State.Frame <= start.Frame {
		t.Fatalf("autoOutcome.State.Frame = %d, want progress beyond start frame %d", autoOutcome.State.Frame, start.Frame)
	}

	svc = NewService(nil)
	start, err = svc.StartPVE(ctx, profile, lineup, enemy)
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
	if firstOutcome.State == nil || len(firstOutcome.State.PendingActorIDs) != 2 {
		t.Fatalf("firstOutcome.State = %#v, want two pending actors", firstOutcome.State)
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
	if timeoutOutcome == nil || timeoutOutcome.State == nil {
		t.Fatalf("timeoutOutcome = %#v, want resolved state", timeoutOutcome)
	}
	if !timeoutOutcome.State.AutoBattleEnabled {
		t.Fatal("timeoutOutcome.State.AutoBattleEnabled = false, want true after command timeout")
	}
	if timeoutOutcome.State.Frame <= firstOutcome.State.Frame {
		t.Fatalf("timeoutOutcome.State.Frame = %d, want progress beyond queued frame %d", timeoutOutcome.State.Frame, firstOutcome.State.Frame)
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

	start, err := svc.StartPVE(ctx, profile, lineup, enemy)
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if len(start.Enemies) != 2 {
		t.Fatalf("len(start.Enemies) = %d, want 2", len(start.Enemies))
	}
	damagePet := start.Allies[1]
	if damagePet.Skills[1].TargetType != "enemy_all" {
		t.Fatalf("damage pet skills = %#v, want second skill target type enemy_all", damagePet.Skills)
	}
	if damagePet.Skills[1].AnimationKey != "burst" || damagePet.Skills[1].CastColor == "" || damagePet.Skills[1].ImpactColor == "" || !damagePet.Skills[1].Projectile {
		t.Fatalf("damage pet skill visuals = %#v, want burst animation metadata", damagePet.Skills[1])
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

func TestBuildPlayerCharacterActorIncludesRequestedAttributes(t *testing.T) {
	profile := &player.Profile{PlayerID: 40001, Name: "SoloHero"}
	actor := buildPlayerCharacterActor(profile, PlayerActorType)
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

func TestAdjustStatusChancePctUsesSpecificResistance(t *testing.T) {
	battle := &activeBattle{}
	target := &actorRuntime{
		confusionResistPct: 15,
		sealResistPct:      20,
		curseResistPct:     5,
	}

	if got := battle.adjustStatusChancePct(35, target, StatusSeal); got != 15 {
		t.Fatalf("adjustStatusChancePct(seal) = %d, want 15", got)
	}
	if got := battle.adjustStatusChancePct(10, target, StatusCurse); got != 5 {
		t.Fatalf("adjustStatusChancePct(curse) = %d, want 5", got)
	}
	if got := battle.adjustStatusChancePct(10, target, StatusConfusion); got != 0 {
		t.Fatalf("adjustStatusChancePct(confusion) = %d, want 0 after resist clamp", got)
	}
}

func TestServiceMultiTargetSkillHitsConfiguredEnemyCount(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()
	profile := &player.Profile{PlayerID: 10001, Name: "DemoTrainer", Level: 8, SceneID: 1, PosX: 8, PosY: 6, SkillIDs: []uint32{1101, 1001}}
	lineup := []pet.LineupPet{
		// 第一只宠物显式携带双目标技能，用来验证服务端会以用户指定目标为首，
		// 再自动补足剩余存活敌方单位，而不是把多目标选择权交给客户端。
		{PetUID: 20001, PetID: 101, Level: 5, HP: 120, HPMax: 120, ATK: 12, DEF: 10, SPD: 30, MANA: 12, SkillIDs: []uint32{1001, 1004}},
		{PetUID: 20002, PetID: 102, Level: 4, HP: 30, HPMax: 30, ATK: 11, DEF: 11, SPD: 9, MANA: 18, SkillIDs: []uint32{1001, 1003}},
	}
	enemy := world.Entity{EntityID: 90004, EntityType: 2, Pos: world.Vec2i{X: 10, Y: 6}, Name: "GuideNPC"}

	start, err := svc.StartPVE(ctx, profile, lineup, enemy)
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if len(start.Enemies) != 2 {
		t.Fatalf("len(start.Enemies) = %d, want 2", len(start.Enemies))
	}
	multiTargetPet := start.Allies[1]
	if len(multiTargetPet.Skills) != 2 || multiTargetPet.Skills[1].TargetType != "enemy_multi" {
		t.Fatalf("pet skills = %#v, want second skill target type enemy_multi", multiTargetPet.Skills)
	}
	if multiTargetPet.Skills[1].TargetCount != 2 {
		t.Fatalf("multiTargetPet.Skills[1].TargetCount = %d, want 2", multiTargetPet.Skills[1].TargetCount)
	}
	if multiTargetPet.Skills[1].AnimationKey != "volley" || multiTargetPet.Skills[1].CastColor == "" || multiTargetPet.Skills[1].ImpactColor == "" || !multiTargetPet.Skills[1].Projectile {
		t.Fatalf("multiTargetPet.Skills[1] visuals = %#v, want volley animation metadata", multiTargetPet.Skills[1])
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
	if outcomeOne.State == nil || len(outcomeOne.State.PendingActorIDs) != 1 {
		t.Fatalf("outcomeOne.State.PendingActorIDs = %#v, want one defender actor pending", outcomeOne.State)
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

	start, err := svc.StartPVE(ctx, profile, nil, enemy)
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
