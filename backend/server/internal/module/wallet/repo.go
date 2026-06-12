package wallet

import "context"

// Repository 定义后台钱包管理需要的最小持久化能力。
// 列表、详情和增量调整都统一经过该接口，保证货币修改一定会留下流水。
type Repository interface {
	ListForAdmin(ctx context.Context, query AdminListQuery) (*AdminWalletList, error)
	FindAdminDetailByPlayerID(ctx context.Context, playerID uint64) (*AdminWalletDetail, error)
	AdjustForAdmin(ctx context.Context, playerID uint64, input AdminAdjustInput) (*AdminWalletDetail, error)
	AdjustRuntime(ctx context.Context, playerID uint64, input RuntimeAdjustInput) (*RuntimeAdjustResult, error)
	GetRuntimeSnapshot(ctx context.Context, playerID uint64) (*Snapshot, error)
}
