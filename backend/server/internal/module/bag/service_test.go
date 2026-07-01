package bag

import (
	"context"
	"errors"
	"io"
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

func (r *useItemRateLimitRepo) UseRuntimeItem(context.Context, uint64, string, uint32, uint64, uint64, uint64, string) (*RuntimeUseResult, error) {
	r.calls++
	return &RuntimeUseResult{ItemID: 3004, UsedQuantity: 1}, nil
}

func (r *useItemRateLimitRepo) DropRuntimeItem(context.Context, uint64, string, uint32, string, uint64) (*RuntimeDropResult, error) {
	panic("unexpected DropRuntimeItem call")
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

// dropRuntimeItemRepo 复用 useItemRateLimitRepo 的未使用方法，只记录 DropRuntimeItem 入参。
type dropRuntimeItemRepo struct {
	useItemRateLimitRepo
	calls         int
	playerID      uint64
	containerType string
	slotIndex     uint32
	itemUID       string
	quantity      uint64
}

func (r *dropRuntimeItemRepo) DropRuntimeItem(_ context.Context, playerID uint64, containerType string, slotIndex uint32, itemUID string, quantity uint64) (*RuntimeDropResult, error) {
	r.calls++
	r.playerID = playerID
	r.containerType = containerType
	r.slotIndex = slotIndex
	r.itemUID = itemUID
	r.quantity = quantity
	return &RuntimeDropResult{
		ContainerType: containerType,
		SlotIndex:     slotIndex,
		ItemUID:       itemUID,
		ItemID:        1001,
		ItemName:      "测试物品",
		DroppedQty:    quantity,
	}, nil
}

// listRuntimeContainerRetryRepo 用于模拟远端 PostgreSQL 首次读快照时断开连接。
// 该查询是只读快照读取，服务层允许对瞬时网络错误重试一次。
type listRuntimeContainerRetryRepo struct {
	useItemRateLimitRepo
	calls int
	errs  []error
}

func (r *listRuntimeContainerRetryRepo) ListRuntimeContainer(_ context.Context, _ uint64, containerType string) (*RuntimeContainerSnapshot, error) {
	r.calls++
	if len(r.errs) >= r.calls && r.errs[r.calls-1] != nil {
		return nil, r.errs[r.calls-1]
	}
	return &RuntimeContainerSnapshot{
		ContainerType: containerType,
		Capacity:      24,
		MaxCapacity:   48,
		Items:         []RuntimeItemSnapshot{},
	}, nil
}

func TestUseRuntimeItemRateLimitBlocksSecondCallWithinOneSecond(t *testing.T) {
	repo := &useItemRateLimitRepo{}
	service := NewService(repo)
	ctx := context.Background()
	const playerID uint64 = 10001

	if _, err := service.UseRuntimeItem(ctx, playerID, ContainerTypeBag, 1, 1, 0, 0, ""); err != nil {
		t.Fatalf("first UseRuntimeItem() error = %v, want nil", err)
	}
	if repo.calls != 1 {
		t.Fatalf("repo.calls = %d, want 1", repo.calls)
	}

	if _, err := service.UseRuntimeItem(ctx, playerID, ContainerTypeBag, 2, 1, 0, 0, ""); !errors.Is(err, ErrUseItemTooFast) {
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

	if _, err := service.UseRuntimeItem(ctx, playerID, ContainerTypeBag, 1, 1, 0, 0, ""); err != nil {
		t.Fatalf("first UseRuntimeItem() error = %v, want nil", err)
	}

	service.useItemMu.Lock()
	service.useItemLast[playerID] = time.Now().Add(-RuntimeUseItemCooldown)
	service.useItemMu.Unlock()

	if _, err := service.UseRuntimeItem(ctx, playerID, ContainerTypeBag, 2, 1, 0, 0, ""); err != nil {
		t.Fatalf("second UseRuntimeItem() after cooldown error = %v, want nil", err)
	}
	if repo.calls != 2 {
		t.Fatalf("repo.calls = %d, want 2", repo.calls)
	}
}

func TestListRuntimeContainerRetriesTransientEOFOnce(t *testing.T) {
	repo := &listRuntimeContainerRetryRepo{errs: []error{io.ErrUnexpectedEOF}}
	service := NewService(repo)

	result, err := service.ListRuntimeContainer(context.Background(), 10001, ContainerTypeBag)
	if err != nil {
		t.Fatalf("ListRuntimeContainer() error = %v, want nil after transient retry", err)
	}
	if repo.calls != 2 {
		t.Fatalf("repo.calls = %d, want 2 after one transient retry", repo.calls)
	}
	if result == nil || result.ContainerType != ContainerTypeBag {
		t.Fatalf("result = %#v, want normalized bag snapshot", result)
	}
}

func TestListRuntimeContainerDoesNotRetryBusinessError(t *testing.T) {
	repo := &listRuntimeContainerRetryRepo{errs: []error{ErrContainerNotFound}}
	service := NewService(repo)

	if _, err := service.ListRuntimeContainer(context.Background(), 10001, ContainerTypeBag); !errors.Is(err, ErrContainerNotFound) {
		t.Fatalf("ListRuntimeContainer() error = %v, want ErrContainerNotFound", err)
	}
	if repo.calls != 1 {
		t.Fatalf("repo.calls = %d, want 1 because business errors must not retry", repo.calls)
	}
}

func TestDropRuntimeItemNormalizesContainerAndItemUID(t *testing.T) {
	repo := &dropRuntimeItemRepo{}
	service := NewService(repo)

	result, err := service.DropRuntimeItem(context.Background(), 10001, " bag ", 0, " eq_10001 ", 1)
	if err != nil {
		t.Fatalf("DropRuntimeItem() error = %v, want nil", err)
	}
	if result == nil || result.ItemUID != "eq_10001" {
		t.Fatalf("DropRuntimeItem() result = %#v, want trimmed item_uid", result)
	}
	if repo.calls != 1 {
		t.Fatalf("repo.calls = %d, want 1", repo.calls)
	}
	if repo.containerType != ContainerTypeBag || repo.itemUID != "eq_10001" || repo.slotIndex != 0 || repo.quantity != 1 {
		t.Fatalf("repo args = container=%q uid=%q slot=%d qty=%d, want normalized item_uid drop", repo.containerType, repo.itemUID, repo.slotIndex, repo.quantity)
	}
}

func TestDropRuntimeItemRejectsMissingLocatorAndQuantity(t *testing.T) {
	repo := &dropRuntimeItemRepo{}
	service := NewService(repo)

	if _, err := service.DropRuntimeItem(context.Background(), 10001, ContainerTypeBag, 0, "", 1); !errors.Is(err, ErrInvalidTransferQuantity) {
		t.Fatalf("DropRuntimeItem() missing locator error = %v, want ErrInvalidTransferQuantity", err)
	}
	if _, err := service.DropRuntimeItem(context.Background(), 10001, ContainerTypeBag, 1, "", 0); !errors.Is(err, ErrInvalidTransferQuantity) {
		t.Fatalf("DropRuntimeItem() zero quantity error = %v, want ErrInvalidTransferQuantity", err)
	}
	if repo.calls != 0 {
		t.Fatalf("repo.calls = %d, want 0 when validation fails", repo.calls)
	}
}
