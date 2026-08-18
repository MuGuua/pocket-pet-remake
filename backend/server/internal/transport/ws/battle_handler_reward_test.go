package wstransport

import (
	"context"
	"errors"
	"testing"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/battle"
	"pocket-pet-remake/server/internal/module/monster"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/reward"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/protocol"
	"pocket-pet-remake/server/internal/teststub"
)

var errForcedPetBattleProgress = errors.New("forced pet battle progress failure")

// failingBattlePetRepository 在奖励已落库后模拟宠物战斗进度写回失败，用于验证防重复记录不会被误删。
type failingBattlePetRepository struct {
	*teststub.PetRepository
}

// UpdatePetHPByUID 固定返回错误，模拟奖励事务成功后的宠物血量同步故障。
func (r *failingBattlePetRepository) UpdatePetHPByUID(
	_ context.Context,
	_ uint64,
	_ uint64,
	_ uint32,
) (pet.Pet, error) {
	return pet.Pet{}, errForcedPetBattleProgress
}

// TestApplyBattleResultSideEffectsSkipsDuplicateGrantButSyncsProfile 验证重复结算不会二次发奖，
// 但仍会带回当前玩家档案，供客户端刷新经验与等级展示。
func TestApplyBattleResultSideEffectsSkipsDuplicateGrantButSyncsProfile(t *testing.T) {
	t.Parallel()

	playerService := teststub.NewTestPlayerService()
	battleRepo := teststub.NewBattleRepository()
	handler := NewBattleHandler(nil, playerService, nil, nil, nil, nil, nil, nil, nil, battle.NewService(nil), battleRepo, nil, nil)

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

// TestApplyBattleResultSideEffectsKeepsRewardRecordAfterPostGrantFailure 验证正式奖励落库后，
// 即使宠物进度同步失败也不会删除 battle_record，重试同一场战斗时不会再次增加玩家经验。
func TestApplyBattleResultSideEffectsKeepsRewardRecordAfterPostGrantFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	playerService := teststub.NewTestPlayerService()
	battleRepo := teststub.NewBattleRepository()
	petRepo := &failingBattlePetRepository{PetRepository: teststub.NewPetRepository()}
	petService := pet.NewService(petRepo, nil, nil, nil)
	handler := NewBattleHandler(nil, playerService, petService, nil, nil, nil, nil, nil, nil, battle.NewService(nil), battleRepo, nil, nil)

	profileBefore, err := playerService.GetProfile(ctx, teststub.DemoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() before error = %v", err)
	}
	result := &battle.ResultSnapshot{
		BattleID:        43,
		BattleType:      battle.BattleTypePVE,
		Win:             true,
		RewardPlayerExp: 10,
		PetResults: []battle.PetResult{
			{PetUID: 20001, HP: 1},
		},
	}

	firstSettlement, firstErr := handler.applyBattleResultSideEffects(ctx, nil, teststub.DemoPlayerID, result, nil)
	if !errors.Is(firstErr, errForcedPetBattleProgress) {
		t.Fatalf("first applyBattleResultSideEffects() error = %v, want %v", firstErr, errForcedPetBattleProgress)
	}
	if firstSettlement == nil || !firstSettlement.rewardGrantCommitted {
		t.Fatal("first settlement should mark the persisted reward as committed")
	}
	profileAfterFirst, err := playerService.GetProfile(ctx, teststub.DemoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() after first settlement error = %v", err)
	}
	if profileAfterFirst.Exp != profileBefore.Exp+10 {
		t.Fatalf("profile exp after first settlement = %d, want %d", profileAfterFirst.Exp, profileBefore.Exp+10)
	}

	secondSettlement, secondErr := handler.applyBattleResultSideEffects(ctx, nil, teststub.DemoPlayerID, result, nil)
	if secondErr != nil {
		t.Fatalf("second applyBattleResultSideEffects() error = %v", secondErr)
	}
	if secondSettlement == nil || !secondSettlement.rewardsAlreadyGranted || !secondSettlement.rewardGrantCommitted {
		t.Fatal("second settlement should load sync data without reopening reward grant")
	}
	profileAfterSecond, err := playerService.GetProfile(ctx, teststub.DemoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() after second settlement error = %v", err)
	}
	if profileAfterSecond.Exp != profileAfterFirst.Exp {
		t.Fatalf("profile exp after retry = %d, want unchanged %d", profileAfterSecond.Exp, profileAfterFirst.Exp)
	}
}

// TestPushBattleResultAfterSideEffectsUsesErrorPush 验证战斗结算错误使用 ERROR_PUSH，
// 并根据背包容量不足和奖励已提交后的同步异常展示不同提示框文案。
func TestPushBattleResultAfterSideEffectsUsesErrorPush(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		settlement *battleSettlement
		grantErr   error
		wantMsg    string
	}{
		{
			name:       "bag capacity full",
			settlement: &battleSettlement{},
			grantErr:   bag.ErrContainerCapacityFull,
			wantMsg:    "背包空间不足，战斗物品奖励未能发放，请清理背包后重试。",
		},
		{
			name:       "post grant sync failure",
			settlement: &battleSettlement{rewardGrantCommitted: true},
			grantErr:   errForcedPetBattleProgress,
			wantMsg:    "战斗奖励已发放，但结算数据同步异常，请重新登录刷新。",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn := &fakeConn{id: "battle-error-test"}
			handler := NewBattleHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, battle.NewService(nil), nil, nil, nil)
			result := &battle.ResultSnapshot{BattleID: 44, BattleType: battle.BattleTypePVE, Win: true}

			if err := handler.pushBattleResultAfterSideEffects(context.Background(), conn, teststub.DemoPlayerID, result, test.settlement, test.grantErr); err != nil {
				t.Fatalf("pushBattleResultAfterSideEffects() error = %v", err)
			}
			if len(conn.packets) != 2 {
				t.Fatalf("len(conn.packets) = %d, want battle result and error push", len(conn.packets))
			}
			if conn.packets[0].Cmd != protocol.CmdBattleResultPush {
				t.Fatalf("first packet cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdBattleResultPush)
			}
			if conn.packets[1].Cmd != protocol.CmdErrorPush {
				t.Fatalf("second packet cmd = %d, want %d", conn.packets[1].Cmd, protocol.CmdErrorPush)
			}
			var errorPush protocol.ErrorPush
			if err := protocol.UnmarshalBody(conn.packets[1].Body, &errorPush); err != nil {
				t.Fatalf("UnmarshalBody() error = %v", err)
			}
			if errorPush.Msg != test.wantMsg {
				t.Fatalf("error push msg = %q, want %q", errorPush.Msg, test.wantMsg)
			}
		})
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
		t.Fatalf("nil settlement rewards = %#v, want no uncommitted popup rewards", rewards)
	}

	emptySettlement := &battleSettlement{}
	if rewards := battlePopupRewardsFromResult(result, emptySettlement); len(rewards) != 0 {
		t.Fatalf("empty settlement rewards = %#v, want no uncommitted popup rewards", rewards)
	}

	grantedSettlement := &battleSettlement{
		GrantedRewards: []reward.Entry{{Type: "exp", Value: 20}},
	}
	if rewards := battlePopupRewardsFromResult(result, grantedSettlement); len(rewards) != 1 {
		t.Fatalf("granted settlement rewards = %#v, want one popup reward", rewards)
	}

	committedSettlement := &battleSettlement{rewardGrantCommitted: true}
	if rewards := battlePopupRewardsFromResult(result, committedSettlement); len(rewards) != 2 {
		t.Fatalf("committed settlement rewards = %#v, want snapshot fallback popup rewards", rewards)
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
	handler := NewBattleHandler(nil, nil, nil, bagService, nil, nil, nil, nil, nil, battle.NewService(nil), teststub.NewBattleRepository(), nil, nil)

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
	start, err := battleService.StartPVE(ctx, profile, nil, world.Entity{EntityID: 90001, EntityType: 2, Name: "GuideNPC"}, battle.EmptyCharacterBattleSkillInput())
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if start.BattleID <= 70123 {
		t.Fatalf("start.BattleID = %d, want greater than persisted max 70123", start.BattleID)
	}
}
