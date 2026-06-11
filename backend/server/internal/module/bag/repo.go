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
}
