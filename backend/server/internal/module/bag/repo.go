package bag

import "context"

// Repository 定义后台背包管理所需的最小持久化能力。
// 这里暂时只承载管理后台 CRUD，不直接耦合玩家端 WebSocket 背包流程。
type Repository interface {
	ListForAdmin(ctx context.Context, query AdminListQuery) (*AdminItemList, error)
	FindAdminDetailByRecordID(ctx context.Context, recordID uint64) (*AdminItemDetail, error)
	CreateForAdmin(ctx context.Context, input AdminCreateItemInput) (*AdminItemDetail, error)
	UpdateForAdmin(ctx context.Context, recordID uint64, input AdminUpdateItemInput) (*AdminItemDetail, error)
	DeleteForAdmin(ctx context.Context, recordID uint64) error
	ListRuntimeContainer(ctx context.Context, playerID uint64, containerType string) (*RuntimeContainerSnapshot, error)
	TransferRuntimeItem(ctx context.Context, playerID uint64, fromContainerType string, toContainerType string, fromSlotIndex uint32, quantity uint64) (*RuntimeTransferResult, error)
	SortRuntimeContainer(ctx context.Context, playerID uint64, containerType string) (*RuntimeSortResult, error)
	MoveRuntimeItem(ctx context.Context, playerID uint64, containerType string, fromSlotIndex uint32, toSlotIndex uint32, quantity uint64) (*RuntimeMoveResult, error)
	GrantRuntimeItem(ctx context.Context, playerID uint64, containerType string, itemID uint64, quantity uint64, reasonType string, reasonRefID uint64, operatorType string, operatorID uint64) (*RuntimeGrantResult, error)
	UseRuntimeItem(ctx context.Context, playerID uint64, containerType string, slotIndex uint32, quantity uint64, targetPetUID uint64, targetPlayerID uint64, targetItemUID string) (*RuntimeUseResult, error)
	DropRuntimeItem(ctx context.Context, playerID uint64, containerType string, slotIndex uint32, itemUID string, quantity uint64) (*RuntimeDropResult, error)
	ConsumeRuntimeItemStack(ctx context.Context, playerID uint64, containerType string, slotIndex uint32, quantity uint64, reasonType string, reasonRefID uint64) (*RuntimeContainerSnapshot, error)
	// PlayerHasEverOwnedItem 判断玩家是否已获得过指定道具，供唯一战斗掉落去重。
	PlayerHasEverOwnedItem(ctx context.Context, playerID uint64, itemID uint64) (bool, error)
	// RecordUniqueItemObtained 记录玩家首次获得唯一道具，重复写入会被忽略。
	RecordUniqueItemObtained(ctx context.Context, playerID uint64, itemID uint64, reasonType string, reasonRefID uint64) error
}
