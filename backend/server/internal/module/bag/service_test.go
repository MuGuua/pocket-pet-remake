package bag

import (
	"context"
	"errors"
	"testing"
	"time"
)

// useItemRateLimitRepo 仅实现 UseRuntimeItem，供冷却测试复用。
type useItemRateLimitRepo struct {
	calls int
}

func (r *useItemRateLimitRepo) ListForAdmin(context.Context, AdminListQuery) (*AdminItemList, error) {
	panic("unexpected ListForAdmin call")
}

func (r *useItemRateLimitRepo) FindAdminDetailByRecordID(context.Context, uint64) (*AdminItemDetail, error) {
	panic("unexpected FindAdminDetailByRecordID call")
}

func (r *useItemRateLimitRepo) CreateForAdmin(context.Context, AdminCreateItemInput) (*AdminItemDetail, error) {
	panic("unexpected CreateForAdmin call")
}

func (r *useItemRateLimitRepo) UpdateForAdmin(context.Context, uint64, AdminUpdateItemInput) (*AdminItemDetail, error) {
	panic("unexpected UpdateForAdmin call")
}

func (r *useItemRateLimitRepo) DeleteForAdmin(context.Context, uint64) error {
	panic("unexpected DeleteForAdmin call")
}

func (r *useItemRateLimitRepo) ListRuntimeContainer(context.Context, uint64, string) (*RuntimeContainerSnapshot, error) {
	panic("unexpected ListRuntimeContainer call")
}

func (r *useItemRateLimitRepo) TransferRuntimeItem(context.Context, uint64, string, string, uint32, uint64) (*RuntimeTransferResult, error) {
	panic("unexpected TransferRuntimeItem call")
}

func (r *useItemRateLimitRepo) SortRuntimeContainer(context.Context, uint64, string) (*RuntimeSortResult, error) {
	panic("unexpected SortRuntimeContainer call")
}

func (r *useItemRateLimitRepo) MoveRuntimeItem(context.Context, uint64, string, uint32, uint32, uint64) (*RuntimeMoveResult, error) {
	panic("unexpected MoveRuntimeItem call")
}

func (r *useItemRateLimitRepo) GrantRuntimeItem(context.Context, uint64, string, uint64, uint64, string, uint64, string, uint64) (*RuntimeGrantResult, error) {
	panic("unexpected GrantRuntimeItem call")
}

func (r *useItemRateLimitRepo) UseRuntimeItem(context.Context, uint64, string, uint32, uint64, uint64, uint64) (*RuntimeUseResult, error) {
	r.calls++
	return &RuntimeUseResult{ItemID: 3004, UsedQuantity: 1}, nil
}

func (r *useItemRateLimitRepo) ConsumeRuntimeItemStack(context.Context, uint64, string, uint32, uint64, string, uint64) (*RuntimeContainerSnapshot, error) {
	panic("unexpected ConsumeRuntimeItemStack call")
}

func (r *useItemRateLimitRepo) PlayerHasEverOwnedItem(context.Context, uint64, uint64) (bool, error) {
	panic("unexpected PlayerHasEverOwnedItem call")
}

func (r *useItemRateLimitRepo) RecordUniqueItemObtained(context.Context, uint64, uint64, string, uint64) error {
	panic("unexpected RecordUniqueItemObtained call")
}

func TestUseRuntimeItemRateLimitBlocksSecondCallWithinOneSecond(t *testing.T) {
	repo := &useItemRateLimitRepo{}
	service := NewService(repo)
	ctx := context.Background()
	const playerID uint64 = 10001

	if _, err := service.UseRuntimeItem(ctx, playerID, ContainerTypeBag, 1, 1, 0, 0); err != nil {
		t.Fatalf("first UseRuntimeItem() error = %v, want nil", err)
	}
	if repo.calls != 1 {
		t.Fatalf("repo.calls = %d, want 1", repo.calls)
	}

	if _, err := service.UseRuntimeItem(ctx, playerID, ContainerTypeBag, 2, 1, 0, 0); !errors.Is(err, ErrUseItemTooFast) {
		t.Fatalf("second UseRuntimeItem() error = %v, want ErrUseItemTooFast", err)
	}
	if repo.calls != 1 {
		t.Fatalf("repo.calls after blocked call = %d, want 1", repo.calls)
	}
}

func TestUseRuntimeItemRateLimitAllowsCallAfterCooldown(t *testing.T) {
	repo := &useItemRateLimitRepo{}
	service := NewService(repo)
	ctx := context.Background()
	const playerID uint64 = 10002

	if _, err := service.UseRuntimeItem(ctx, playerID, ContainerTypeBag, 1, 1, 0, 0); err != nil {
		t.Fatalf("first UseRuntimeItem() error = %v, want nil", err)
	}

	service.useItemMu.Lock()
	service.useItemLast[playerID] = time.Now().Add(-RuntimeUseItemCooldown)
	service.useItemMu.Unlock()

	if _, err := service.UseRuntimeItem(ctx, playerID, ContainerTypeBag, 2, 1, 0, 0); err != nil {
		t.Fatalf("second UseRuntimeItem() after cooldown error = %v, want nil", err)
	}
	if repo.calls != 2 {
		t.Fatalf("repo.calls = %d, want 2", repo.calls)
	}
}