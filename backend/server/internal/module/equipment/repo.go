package equipment

import (
	"context"

	"pocket-pet-remake/server/internal/module/player"
)

// Repository 定义装备模板 Admin 与运行时佩戴接口。
type Repository interface {
	ListForAdmin(ctx context.Context, query AdminListQuery) (*AdminEquipmentList, error)
	FindForAdminByItemID(ctx context.Context, itemID uint64) (*AdminEquipmentDetail, error)
	CreateForAdmin(ctx context.Context, input AdminUpsertEquipmentInput) (*AdminEquipmentDetail, error)
	UpdateForAdmin(ctx context.Context, itemID uint64, input AdminUpsertEquipmentInput) (*AdminEquipmentDetail, error)
	DeleteForAdmin(ctx context.Context, itemID uint64) error

	ListEquipped(ctx context.Context, playerID uint64) ([]RuntimeEquippedItem, error)
	EquipFromBagSlot(ctx context.Context, playerID uint64, containerType string, bagSlotIndex uint32, recalc RecalcContext, currentProfile *player.Profile) (*EquipFromBagResult, error)
	UnequipSlot(ctx context.Context, playerID uint64, equipSlot string, containerType string, recalc RecalcContext, currentProfile *player.Profile) (*UnequipSlotResult, error)
	EnhanceInstance(ctx context.Context, playerID uint64, itemUID string) (*EnhanceResult, error)
}
