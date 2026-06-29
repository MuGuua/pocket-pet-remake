package postgres

import (
	"context"

	"pocket-pet-remake/server/internal/module/bag"
)

// buildRuntimeRepairPreview 为背包内损坏装备组装修复弹窗预览数据。
func buildRuntimeRepairPreview(
	ctx context.Context,
	db DBTX,
	playerID uint64,
	isDamaged bool,
) (*bag.RuntimeRepairPreview, error) {
	if !isDamaged {
		return nil, nil
	}
	cost, err := loadRuntimeRepairCost(ctx, db)
	if err != nil {
		return nil, err
	}
	if cost == nil {
		return &bag.RuntimeRepairPreview{CanRepair: false}, nil
	}
	ownedQuantity, err := loadRuntimeBagOwnedItemQuantity(ctx, db, playerID, bag.ContainerTypeBag, cost.CostItemID)
	if err != nil {
		return nil, err
	}
	costItemName, err := loadItemNameByID(ctx, db, cost.CostItemID)
	if err != nil {
		return nil, err
	}
	canRepair := ownedQuantity >= cost.CostQuantity
	return &bag.RuntimeRepairPreview{
		CanRepair:         canRepair,
		CostItemID:        cost.CostItemID,
		CostItemName:      costItemName,
		CostQuantity:      cost.CostQuantity,
		OwnedCostQuantity: ownedQuantity,
	}, nil
}
