package wstransport

import (
	"context"
	"testing"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/battle"
	"pocket-pet-remake/server/internal/module/monster"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/reward"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/teststub"
)

// TestApplyBattleResultSideEffectsSkipsDuplicateGrantButSyncsProfile 验证重复结算不会二次发奖，
// 但仍会带回当前玩家档案，供客户端刷新经验与等级展示。
func TestApplyBattleResultSideEffectsSkipsDuplicateGrantButSyncsProfile(t *testing.T) {
	t.Parallel()

	playerService := teststub.NewTestPlayerService()
	battleRepo := teststub.NewBattleRepository()
	handler := NewBattleHandler(nil, playerService, nil, nil, nil, nil, nil, nil, nil, battle.NewService(nil), battleRepo)

	ctx := context.Background()
	result := &battle.ResultSnapshot{
		BattleID:        42,
		BattleType:      battle.BattleTypePVE,
		Win:             true,
		RewardGold:      12,
		RewardPlayerExp: 28,
	}

	firstSettlement, err := handler.applyBattleResultSideEffects(ctx, nil, teststub.DemoPlayerID, result, nil)
	if err != nil {
		t.Fatalf("first applyBattleResultSideEffects() error = %v", err)
	}
	if firstSettlement == nil || firstSettlement.PlayerProfile == nil {
		t.Fatal("first settlement profile = nil, want granted profile snapshot")
	}
	if len(firstSettlement.GrantedRewards) == 0 {
		t.Fatal("first settlement granted rewards = 0, want positive reward entries")
	}

	beforeExp := firstSettlement.PlayerProfile.Exp
	secondSettlement, err := handler.applyBattleResultSideEffects(ctx, nil, teststub.DemoPlayerID, result, nil)
	if err != nil {
		t.Fatalf("second applyBattleResultSideEffects() error = %v", err)
	}
	if secondSettlement == nil || !secondSettlement.rewardsAlreadyGranted {
		t.Fatal("second settlement should be sync-only without duplicate grant")
	}
	if secondSettlement.PlayerProfile == nil {
		t.Fatal("second settlement profile = nil, want current profile for client sync")
	}
	if secondSettlement.PlayerProfile.Exp != beforeExp {
		t.Fatalf("second settlement exp = %d, want unchanged %d", secondSettlement.PlayerProfile.Exp, beforeExp)
	}
	if len(secondSettlement.GrantedRewards) != 0 {
		t.Fatalf("second settlement granted rewards = %d, want 0 duplicate entries", len(secondSettlement.GrantedRewards))
	}
}

// TestBattlePopupRewardsFromResultRequiresGrantedSettlement 验证未真正发奖时不会伪造弹窗奖励。
func TestBattlePopupRewardsFromResultRequiresGrantedSettlement(t *testing.T) {
	t.Parallel()

	result := &battle.ResultSnapshot{
		Win:             true,
		RewardGold:      10,
		RewardPlayerExp: 20,
	}

	if rewards := battlePopupRewardsFromResult(result, nil); len(rewards) != 0 {
		t.Fatalf("nil settlement rewards = %#v, want empty popup list", rewards)
	}

	emptySettlement := &battleSettlement{}
	if rewards := battlePopupRewardsFromResult(result, emptySettlement); len(rewards) != 0 {
		t.Fatalf("empty settlement rewards = %#v, want empty popup list", rewards)
	}

	grantedSettlement := &battleSettlement{
		GrantedRewards: []reward.Entry{{Type: "exp", Value: 20}},
	}
	if rewards := battlePopupRewardsFromResult(result, grantedSettlement); len(rewards) != 1 {
		t.Fatalf("granted settlement rewards = %#v, want one popup reward", rewards)
	}

	syncSettlement := &battleSettlement{rewardsAlreadyGranted: true}
	if rewards := battlePopupRewardsFromResult(result, syncSettlement); len(rewards) != 0 {
		t.Fatalf("sync settlement rewards = %#v, want empty popup on duplicate sync", rewards)
	}
}

// TestBuildBattleGrantEntriesSkipsOwnedUniqueItems 验证唯一物品已获得后不再进入发奖列表，经验与铜币仍保留。
func TestBuildBattleGrantEntriesSkipsOwnedUniqueItems(t *testing.T) {
	t.Parallel()

	bagRepo := teststub.NewBagRepository()
	bagService := bag.NewService(bagRepo)
	handler := NewBattleHandler(nil, nil, nil, bagService, nil, nil, nil, nil, nil, nil, nil)

	ctx := context.Background()
	uniqueItemID := uint64(5101)
	if err := bagRepo.RecordUniqueItemObtained(ctx, teststub.DemoPlayerID, uniqueItemID, "battle_reward", 1); err != nil {
		t.Fatalf("RecordUniqueItemObtained() error = %v", err)
	}

	result := &battle.ResultSnapshot{
		Win:             true,
		RewardGold:      12,
		RewardPlayerExp: 28,
		DropItems: []battle.DropReward{
			{ItemID: uniqueItemID, Quantity: 1, GrantOnce: true},
			{ItemID: 3101, Quantity: 2, GrantOnce: false},
		},
	}
	entries, err := handler.buildBattleGrantEntries(ctx, teststub.DemoPlayerID, result)
	if err != nil {
		t.Fatalf("buildBattleGrantEntries() error = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3 (gold + exp + repeatable item)", len(entries))
	}
	for _, entry := range entries {
		if entry.Type == "item" && entry.ItemID == uniqueItemID {
			t.Fatalf("unique item %d should be skipped, got entries=%#v", uniqueItemID, entries)
		}
	}
}

// TestEnsureNextBattleIDAvoidsReusingPersistedBattleID 验证服务启动后会跳过 battle_record 已占用的 battle_id。
func TestEnsureNextBattleIDAvoidsReusingPersistedBattleID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	battleRepo := teststub.NewBattleRepository()
	if _, err := battleRepo.CreateRewardRecord(ctx, battle.RewardRecord{
		BattleID:   70123,
		PlayerID:   teststub.DemoPlayerID,
		BattleType: battle.BattleTypePVE,
	}); err != nil {
		t.Fatalf("CreateRewardRecord() error = %v", err)
	}

	playerService := teststub.NewTestPlayerService()
	petService := pet.NewService(teststub.NewPetRepository(), nil, teststub.NewMonsterRepository(), nil)
	monsterService := monster.NewService(teststub.NewMonsterRepository(), nil, petService)
	battleService := battle.NewService(monsterService)
	if err := battleService.EnsureNextBattleID(ctx, battleRepo); err != nil {
		t.Fatalf("EnsureNextBattleID() error = %v", err)
	}
	profile, err := playerService.GetProfile(ctx, teststub.DemoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	start, err := battleService.StartPVE(ctx, profile, nil, world.Entity{EntityID: 90001, EntityType: 2, Name: "GuideNPC"})
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if start.BattleID <= 70123 {
		t.Fatalf("start.BattleID = %d, want greater than persisted max 70123", start.BattleID)
	}
}
