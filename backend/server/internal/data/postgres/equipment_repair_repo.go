package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/equipment"
)

const runtimeRepairCostQuery = `
SELECT cost_item_id, cost_quantity
FROM equipment_repair_cost
WHERE status = 1
ORDER BY cost_item_id ASC
LIMIT 1
`

const runtimeClearEquipmentDamagedQuery = `
UPDATE equipment_instance
SET is_damaged = FALSE,
    updated_at = CURRENT_TIMESTAMP
WHERE item_uid = $1
  AND player_id = $2
  AND is_damaged = TRUE
`

// RepairInstance 消耗修复宝石并清除装备损坏状态。
func (r *EquipmentRepository) RepairInstance(
	ctx context.Context,
	playerID uint64,
	itemUID string,
) (*equipment.RepairResult, error) {
	normalizedUID := strings.TrimSpace(itemUID)
	if normalizedUID == "" {
		return nil, equipment.ErrEquipmentNotFound
	}
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	instanceRow, err := loadRuntimeEnhanceInstanceRow(ctx, tx, normalizedUID, playerID)
	if err != nil {
		return nil, err
	}
	if instanceRow == nil {
		return nil, equipment.ErrEquipmentNotFound
	}
	if err := validateEnhanceInstanceUnequippedInBag(ctx, tx, playerID, instanceRow); err != nil {
		if errors.Is(err, equipment.ErrEquipmentEnhanceEquipped) {
			return nil, equipment.ErrEquipmentRepairEquipped
		}
		return nil, err
	}
	if !instanceRow.IsDamaged {
		return nil, equipment.ErrEquipmentRepairNotDamaged
	}

	cost, err := loadRuntimeRepairCost(ctx, tx)
	if err != nil {
		return nil, err
	}
	if cost == nil {
		return nil, equipment.ErrEquipmentRepairConfigMissing
	}
	if err := validateRepairMaterialItemID(ctx, tx, cost.CostItemID); err != nil {
		return nil, err
	}
	if err := consumeBagMaterialsByItemIDInTx(ctx, tx, playerID, bag.ContainerTypeBag, cost.CostItemID, cost.CostQuantity, "player_equipment_repair"); err != nil {
		if errors.Is(err, equipment.ErrEquipmentEnhanceMaterialInsufficient) {
			return nil, equipment.ErrEquipmentRepairMaterialInsufficient
		}
		return nil, err
	}

	result, err := tx.ExecContext(ctx, runtimeClearEquipmentDamagedQuery, normalizedUID, playerID)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, equipment.ErrEquipmentRepairNotDamaged
	}
	instanceRow.IsDamaged = false

	itemSnapshot, err := buildRuntimeItemSnapshotFromEnhanceRow(instanceRow)
	if err != nil {
		return nil, err
	}
	allEquipped, err := loadEquippedRuntimeItemsInTx(ctx, tx, playerID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &equipment.RepairResult{
		Item:        itemSnapshot,
		AllEquipped: allEquipped,
	}, nil
}

func loadRuntimeRepairCost(ctx context.Context, db DBTX) (*equipment.RepairCost, error) {
	var cost equipment.RepairCost
	var costItemID int64
	var costQuantity int64
	err := db.QueryRowContext(ctx, runtimeRepairCostQuery).Scan(&costItemID, &costQuantity)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	cost.CostItemID = uint64(costItemID)
	cost.CostQuantity = uint64(costQuantity)
	return &cost, nil
}

const runtimeRepairMaterialSubTypeQuery = `
SELECT COALESCE(item_sub_type, '')
FROM item_definition
WHERE item_id = $1
  AND is_enabled = TRUE
LIMIT 1
`

func validateRepairMaterialItemID(ctx context.Context, db DBTX, costItemID uint64) error {
	if costItemID == 0 {
		return equipment.ErrEquipmentRepairMaterialInvalid
	}
	var itemSubType string
	err := db.QueryRowContext(ctx, runtimeRepairMaterialSubTypeQuery, costItemID).Scan(&itemSubType)
	if errors.Is(err, sql.ErrNoRows) {
		return equipment.ErrEquipmentRepairMaterialInvalid
	}
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(itemSubType), bag.ItemSubTypeEquipmentRepair) {
		return equipment.ErrEquipmentRepairMaterialInvalid
	}
	return nil
}
