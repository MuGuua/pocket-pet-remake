package npc

import "context"

// Repository 补充后台 NPC / 场景分布配置管理所需的持久化接口。
// 这样管理台的实体位置、菜单项和动作提示都能直接回写数据库配置表。
type AdminRepository interface {
	ListEntitiesForAdmin(ctx context.Context, query AdminEntityListQuery) (*AdminEntityList, error)
	FindAdminEntityDetailByEntityID(ctx context.Context, entityID uint64) (*AdminEntityDetail, error)
	CreateEntityForAdmin(ctx context.Context, input AdminCreateEntityInput) (*AdminEntityDetail, error)
	UpdateEntityForAdmin(ctx context.Context, entityID uint64, input AdminUpdateEntityInput) (*AdminEntityDetail, error)
	DeleteEntityForAdmin(ctx context.Context, entityID uint64) error
	ListMenuEntriesForAdmin(ctx context.Context, query AdminMenuEntryListQuery) (*AdminMenuEntryList, error)
	FindAdminMenuEntryDetail(ctx context.Context, entityID uint64, entryID string) (*AdminMenuEntryDetail, error)
	CreateMenuEntryForAdmin(ctx context.Context, input AdminCreateMenuEntryInput) (*AdminMenuEntryDetail, error)
	UpdateMenuEntryForAdmin(ctx context.Context, entityID uint64, entryID string, input AdminUpdateMenuEntryInput) (*AdminMenuEntryDetail, error)
	DeleteMenuEntryForAdmin(ctx context.Context, entityID uint64, entryID string) error
}
