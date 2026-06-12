package item

import "context"

// Repository 定义后台物品模板管理需要的最小持久化能力。
// 当前先覆盖统一模板主表 CRUD，后续若补装备扩展表可在这里继续扩展专属接口。
type Repository interface {
	ListForAdmin(ctx context.Context, query AdminListQuery) (*AdminItemList, error)
	FindAdminDetailByItemID(ctx context.Context, itemID uint64) (*AdminItemDetail, error)
	CreateForAdmin(ctx context.Context, input AdminUpsertItemInput) (*AdminItemDetail, error)
	UpdateForAdmin(ctx context.Context, itemID uint64, input AdminUpsertItemInput) (*AdminItemDetail, error)
	DeleteForAdmin(ctx context.Context, itemID uint64) error
}
