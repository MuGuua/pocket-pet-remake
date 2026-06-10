package battle

import (
	"context"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/world"
)

func TestServiceSubmitActionHealTargetsAlly(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	profile := &player.Profile{PlayerID: 10001, Name: "DemoTrainer", Level: 8, SceneID: 1, PosX: 8, PosY: 6}
	lineup := []pet.LineupPet{
		// Ally one uses a larger HP pool so the stronger post-formula enemy damage
		// still leaves room for the follow-up heal assertion in round two.
		{PetUID: 20001, PetID: 101, Level: 5, HP: 90, HPMax: 90, ATK: 10, DEF: 10, SPD: 8, MANA: 10, SkillIDs: []uint32{1001, 1002}},
		{PetUID: 20002, PetID: 102, Level: 4, HP: 28, HPMax: 30, ATK: 12, DEF: 11, SPD: 9, MANA: 20, SkillIDs: []uint32{1001, 1003}},
	}
	enemy := world.Entity{EntityID: 90001, EntityType: 2, Pos: world.Vec2i{X: 10, Y: 6}, Name: "GuideNPC"}

	start, err := svc.StartPVE(ctx, profile, lineup, enemy)
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if len(start.Allies) != 2 {
		t.Fatalf("len(start.Allies) = %d, want 2", len(start.Allies))
	}
	if len(start.Allies[1].Skills) != 2 || start.Allies[1].Skills[1].TargetType != "ally_single" {
		t.Fatalf("second ally skills = %#v, want heal skill with ally_single target", start.Allies[1].Skills)
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
	if outcomeOne.State == nil || len(outcomeOne.State.PendingActorIDs) != 1 {
		t.Fatalf("outcomeOne.State.PendingActorIDs = %#v, want one remaining ally", outcomeOne.State)
	}

	outcomeTwo, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[1].ActorID,
		SkillID:    start.Allies[1].SkillIDs[1],
		TargetID:   start.Allies[0].ActorID,
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

	var allyOneHPAfterEnemy uint32
	for _, actor := range outcomeTwo.State.Actors {
		if actor.ActorID == start.Allies[0].ActorID {
			allyOneHPAfterEnemy = actor.HP
			break
		}
	}
	if allyOneHPAfterEnemy >= start.Allies[0].HP {
		t.Fatalf("allyOneHPAfterEnemy = %d, want less than %d after enemy action", allyOneHPAfterEnemy, start.Allies[0].HP)
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
		t.Fatalf("SubmitAction(round2 ally1) error = %v", err)
	}
	if outcomeThree.State == nil || len(outcomeThree.State.PendingActorIDs) != 1 {
		t.Fatalf("outcomeThree pending = %#v, want second ally still pending", outcomeThree.State)
	}

	outcomeFour, err := svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      outcomeTwo.State.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[1].ActorID,
		SkillID:    start.Allies[1].SkillIDs[1],
		TargetID:   start.Allies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(round2 ally2 heal) error = %v", err)
	}
	if outcomeFour.State == nil {
		t.Fatalf("outcomeFour.State = nil, want state snapshot")
	}

	var sawHeal bool
	for _, event := range outcomeFour.State.Events {
		if event.EventType == EventTypeHeal && event.SourceID == start.Allies[1].ActorID && event.TargetID == start.Allies[0].ActorID {
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
	svc := NewService()
	ctx := context.Background()
	profile := &player.Profile{PlayerID: 10001, Name: "DemoTrainer", Level: 8, SceneID: 1, PosX: 8, PosY: 6}
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
	if autoOutcome.State.Round != 2 {
		t.Fatalf("autoOutcome.State.Round = %d, want 2", autoOutcome.State.Round)
	}

	svc = NewService()
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
	if firstOutcome.State == nil || len(firstOutcome.State.PendingActorIDs) != 1 {
		t.Fatalf("firstOutcome.State = %#v, want one pending actor", firstOutcome.State)
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
	if timeoutOutcome.State.Round != 2 {
		t.Fatalf("timeoutOutcome.State.Round = %d, want 2", timeoutOutcome.State.Round)
	}
}

func TestServiceAllTargetSkillHitsMultipleEnemies(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	profile := &player.Profile{PlayerID: 10001, Name: "DemoTrainer", Level: 8, SceneID: 1, PosX: 8, PosY: 6}
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
	if start.Allies[0].Skills[1].TargetType != "enemy_all" {
		t.Fatalf("start.Allies[0].Skills[1].TargetType = %q, want enemy_all", start.Allies[0].Skills[1].TargetType)
	}

	_, err = svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
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
		ActorID:    start.Allies[1].ActorID,
		SkillID:    1001,
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(second ally) error = %v", err)
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

func TestServiceMultiTargetSkillHitsConfiguredEnemyCount(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	profile := &player.Profile{PlayerID: 10001, Name: "DemoTrainer", Level: 8, SceneID: 1, PosX: 8, PosY: 6}
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
	if len(start.Allies[0].Skills) != 2 || start.Allies[0].Skills[1].TargetType != "enemy_multi" {
		t.Fatalf("first ally skills = %#v, want second skill target type enemy_multi", start.Allies[0].Skills)
	}
	if start.Allies[0].Skills[1].TargetCount != 2 {
		t.Fatalf("start.Allies[0].Skills[1].TargetCount = %d, want 2", start.Allies[0].Skills[1].TargetCount)
	}

	_, err = svc.SubmitAction(ctx, profile.PlayerID, ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
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
		ActorID:    start.Allies[1].ActorID,
		SkillID:    1001,
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("SubmitAction(second ally) error = %v", err)
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
	svc := NewService()
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
