package npc

import "context"

type Repository interface {
	ListMenuEntriesByEntityID(ctx context.Context, entityID uint64) ([]MenuEntry, error)
	FindActionResult(ctx context.Context, entityID uint64, entryID string) (*ActionResult, error)
	ListShopGoodsByEntityID(ctx context.Context, entityID uint64) ([]ShopGood, error)
	ShopGoodExists(ctx context.Context, entityID uint64, itemID uint64) (bool, error)
	ListEntitiesForAdmin(ctx context.Context, query AdminEntityListQuery) (*AdminEntityList, error)
	ListWorldScenesForAdmin(ctx context.Context) ([]AdminWorldSceneSummary, error)
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
