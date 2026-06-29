package reward

import (
	"context"
	"errors"
	"testing"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/wallet"
)

func TestGrantRuntimeRewardsRollsBackGrantedItemsWhenLaterItemFails(t *testing.T) {
	t.Parallel()

	bagRepo := &rewardTestBagRepo{failGrantOnCall: 2}
	walletRepo := &rewardTestWalletRepo{}
	service := NewService(bag.NewService(bagRepo), nil, nil, nil, wallet.NewService(walletRepo))

	_, err := service.GrantRuntimeRewards(context.Background(), GrantInput{
		PlayerID:     10001,
		ReasonType:   "item_use_reward",
		ReasonRefID:  3004,
		OperatorType: "player",
		OperatorID:   10001,
		Rewards: []Entry{
			{Type: "gold", Value: 2},
			{Type: "item", ItemID: 2001, Count: 1},
			{Type: "item", ItemID: 2002, Count: 1},
		},
	})
	if !errors.Is(err, bag.ErrContainerCapacityFull) {
		t.Fatalf("GrantRuntimeRewards() error = %v, want ErrContainerCapacityFull", err)
	}
	if bagRepo.grantCalls != 2 {
		t.Fatalf("bagRepo.grantCalls = %d, want 2", bagRepo.grantCalls)
	}
	if len(bagRepo.consumeCalls) != 1 {
		t.Fatalf("len(bagRepo.consumeCalls) = %d, want 1", len(bagRepo.consumeCalls))
	}
	if bagRepo.consumeCalls[0].slotIndex != 1 || bagRepo.consumeCalls[0].quantity != 1 {
		t.Fatalf("bagRepo.consumeCalls[0] = %#v, want rollback slot 1 qty 1", bagRepo.consumeCalls[0])
	}
	if walletRepo.adjustCalls != 0 {
		t.Fatalf("walletRepo.adjustCalls = %d, want 0 because gold should not run before failing items", walletRepo.adjustCalls)
	}
}

func TestGrantRuntimeRewardsRollsBackItemsWhenWalletGrantFails(t *testing.T) {
	t.Parallel()

	bagRepo := &rewardTestBagRepo{}
	walletRepo := &rewardTestWalletRepo{failPositiveAdjust: true}
	service := NewService(bag.NewService(bagRepo), nil, nil, nil, wallet.NewService(walletRepo))

	_, err := service.GrantRuntimeRewards(context.Background(), GrantInput{
		PlayerID:     10001,
		ReasonType:   "item_use_reward",
		ReasonRefID:  3004,
		OperatorType: "player",
		OperatorID:   10001,
		Rewards: []Entry{
			{Type: "item", ItemID: 2001, Count: 1},
			{Type: "gold", Value: 2},
		},
	})
	if !errors.Is(err, errRewardTestWalletAdjustFailed) {
		t.Fatalf("GrantRuntimeRewards() error = %v, want errRewardTestWalletAdjustFailed", err)
	}
	if bagRepo.grantCalls != 1 {
		t.Fatalf("bagRepo.grantCalls = %d, want 1", bagRepo.grantCalls)
	}
	if len(bagRepo.consumeCalls) != 1 {
		t.Fatalf("len(bagRepo.consumeCalls) = %d, want 1", len(bagRepo.consumeCalls))
	}
	if walletRepo.adjustCalls != 1 {
		t.Fatalf("walletRepo.adjustCalls = %d, want 1", walletRepo.adjustCalls)
	}
}

func TestGrantRuntimeRewardsGoldValueMeansCopper(t *testing.T) {
	t.Parallel()

	walletRepo := &rewardTestWalletRepo{totalCopper: 100}
	service := NewService(nil, nil, nil, nil, wallet.NewService(walletRepo))

	result, err := service.GrantRuntimeRewards(context.Background(), GrantInput{
		PlayerID:     10001,
		ReasonType:   "battle_reward",
		ReasonRefID:  42,
		OperatorType: "system",
		OperatorID:   10001,
		Rewards: []Entry{
			{Type: "gold", Value: 20},
		},
	})
	if err != nil {
		t.Fatalf("GrantRuntimeRewards() error = %v", err)
	}
	if result == nil || result.Wallet == nil {
		t.Fatal("result wallet = nil, want granted wallet snapshot")
	}
	if result.Wallet.TotalCopper != 120 {
		t.Fatalf("result.Wallet.TotalCopper = %d, want 120", result.Wallet.TotalCopper)
	}
	if walletRepo.lastChangeCopper != 20 {
		t.Fatalf("walletRepo.lastChangeCopper = %d, want 20", walletRepo.lastChangeCopper)
	}
}

type rewardTestBagRepo struct {
	grantCalls      int
	failGrantOnCall int
	consumeCalls    []rewardTestConsumeCall
}

type rewardTestConsumeCall struct {
	slotIndex uint32
	quantity  uint64
}

func (r *rewardTestBagRepo) ListForAdmin(context.Context, bag.AdminListQuery) (*bag.AdminItemList, error) {
	panic("unexpected ListForAdmin call")
}

func (r *rewardTestBagRepo) FindAdminDetailByRecordID(context.Context, uint64) (*bag.AdminItemDetail, error) {
	panic("unexpected FindAdminDetailByRecordID call")
}

func (r *rewardTestBagRepo) CreateForAdmin(context.Context, bag.AdminCreateItemInput) (*bag.AdminItemDetail, error) {
	panic("unexpected CreateForAdmin call")
}

func (r *rewardTestBagRepo) UpdateForAdmin(context.Context, uint64, bag.AdminUpdateItemInput) (*bag.AdminItemDetail, error) {
	panic("unexpected UpdateForAdmin call")
}

func (r *rewardTestBagRepo) DeleteForAdmin(context.Context, uint64) error {
	panic("unexpected DeleteForAdmin call")
}

func (r *rewardTestBagRepo) ListRuntimeContainer(context.Context, uint64, string) (*bag.RuntimeContainerSnapshot, error) {
	panic("unexpected ListRuntimeContainer call")
}

func (r *rewardTestBagRepo) TransferRuntimeItem(context.Context, uint64, string, string, uint32, uint64) (*bag.RuntimeTransferResult, error) {
	panic("unexpected TransferRuntimeItem call")
}

func (r *rewardTestBagRepo) SortRuntimeContainer(context.Context, uint64, string) (*bag.RuntimeSortResult, error) {
	panic("unexpected SortRuntimeContainer call")
}

func (r *rewardTestBagRepo) MoveRuntimeItem(context.Context, uint64, string, uint32, uint32, uint64) (*bag.RuntimeMoveResult, error) {
	panic("unexpected MoveRuntimeItem call")
}

func (r *rewardTestBagRepo) GrantRuntimeItem(_ context.Context, _ uint64, _ string, itemID uint64, quantity uint64, _ string, _ uint64, _ string, _ uint64) (*bag.RuntimeGrantResult, error) {
	r.grantCalls++
	if r.failGrantOnCall > 0 && r.grantCalls == r.failGrantOnCall {
		return nil, bag.ErrContainerCapacityFull
	}
	return &bag.RuntimeGrantResult{
		ContainerType: bag.ContainerTypeBag,
		ItemID:        itemID,
		ItemName:      "test_item",
		GrantedQty:    quantity,
		SlotIndex:     uint32(r.grantCalls),
	}, nil
}

func (r *rewardTestBagRepo) UseRuntimeItem(context.Context, uint64, string, uint32, uint64, uint64, uint64, string) (*bag.RuntimeUseResult, error) {
	panic("unexpected UseRuntimeItem call")
}

func (r *rewardTestBagRepo) DropRuntimeItem(context.Context, uint64, string, uint32, string, uint64) (*bag.RuntimeDropResult, error) {
	panic("unexpected DropRuntimeItem call")
}

func (r *rewardTestBagRepo) ConsumeRuntimeItemStack(_ context.Context, _ uint64, _ string, slotIndex uint32, quantity uint64, _ string, _ uint64) (*bag.RuntimeContainerSnapshot, error) {
	r.consumeCalls = append(r.consumeCalls, rewardTestConsumeCall{
		slotIndex: slotIndex,
		quantity:  quantity,
	})
	return &bag.RuntimeContainerSnapshot{ContainerType: bag.ContainerTypeBag}, nil
}

func (r *rewardTestBagRepo) PlayerHasEverOwnedItem(context.Context, uint64, uint64) (bool, error) {
	return false, nil
}

func (r *rewardTestBagRepo) RecordUniqueItemObtained(context.Context, uint64, uint64, string, uint64) error {
	return nil
}

var errRewardTestWalletAdjustFailed = errors.New("wallet adjust failed")

type rewardTestWalletRepo struct {
	totalCopper        uint64
	adjustCalls        int
	lastChangeCopper   int64
	failPositiveAdjust bool
}

func (r *rewardTestWalletRepo) ListForAdmin(context.Context, wallet.AdminListQuery) (*wallet.AdminWalletList, error) {
	panic("unexpected ListForAdmin call")
}

func (r *rewardTestWalletRepo) FindAdminDetailByPlayerID(context.Context, uint64) (*wallet.AdminWalletDetail, error) {
	panic("unexpected FindAdminDetailByPlayerID call")
}

func (r *rewardTestWalletRepo) AdjustForAdmin(context.Context, uint64, wallet.AdminAdjustInput) (*wallet.AdminWalletDetail, error) {
	panic("unexpected AdjustForAdmin call")
}

func (r *rewardTestWalletRepo) AdjustRuntime(_ context.Context, _ uint64, input wallet.RuntimeAdjustInput) (*wallet.RuntimeAdjustResult, error) {
	r.adjustCalls++
	r.lastChangeCopper = input.ChangeTotalCopper
	if input.ChangeTotalCopper > 0 && r.failPositiveAdjust {
		return nil, errRewardTestWalletAdjustFailed
	}
	next := int64(r.totalCopper) + input.ChangeTotalCopper
	if next < 0 {
		return nil, wallet.ErrInvalidRuntimeAdjustInput
	}
	r.totalCopper = uint64(next)
	return &wallet.RuntimeAdjustResult{
		Wallet: wallet.Snapshot{
			TotalCopper: r.totalCopper,
			Gold:        r.totalCopper / wallet.CopperPerGold,
			Silver:      (r.totalCopper % wallet.CopperPerGold) / wallet.CopperPerSilver,
			Copper:      r.totalCopper % wallet.CopperPerSilver,
		},
	}, nil
}

func (r *rewardTestWalletRepo) GetRuntimeSnapshot(context.Context, uint64) (*wallet.Snapshot, error) {
	return &wallet.Snapshot{
		TotalCopper: r.totalCopper,
		Gold:        r.totalCopper / wallet.CopperPerGold,
		Silver:      (r.totalCopper % wallet.CopperPerGold) / wallet.CopperPerSilver,
		Copper:      r.totalCopper % wallet.CopperPerSilver,
	}, nil
}
