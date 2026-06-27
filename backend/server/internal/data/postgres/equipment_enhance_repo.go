package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/wallet"
)

const runtimeEnhanceInstanceQuery = `
SELECT
  COALESCE(pes.equip_slot, ''),
  ei.item_uid,
  ei.item_id,
  ei.enhance_level,
  ei.state,
  idf.item_name,
  iee.can_enhance,
  iee.max_enhance_level,
  iee.appearance_skin_id,
  iee.appearance_only,
  iee.base_hp,
  iee.base_mana,
  iee.base_atk,
  iee.base_def,
  iee.base_spd,
  COALESCE(iee.base_stats_json, '{}'::jsonb),
  COALESCE(iee.enhance_per_level_stats_json, '{}'::jsonb)
FROM equipment_instance ei
JOIN item_definition idf ON idf.item_id = ei.item_id
JOIN item_equipment_extra iee ON iee.item_id = ei.item_id
LEFT JOIN player_equipment_slot pes ON pes.item_uid = ei.item_uid AND pes.player_id = ei.player_id
WHERE ei.item_uid = $1
  AND ei.player_id = $2
LIMIT 1
`

const runtimeEnhanceInstanceInBagQuery = `
SELECT 1
FROM player_container_item pci
WHERE pci.player_id = $1
  AND pci.container_type = $2
  AND pci.item_uid = $3
LIMIT 1
`

const runtimeEnhanceCostQuery = `
SELECT cost_item_id, cost_quantity
FROM equipment_enhance_cost
WHERE target_level = $1
  AND status = 1
LIMIT 1
`

const runtimeEnhanceSuccessRateQuery = `
SELECT success_rate_pct
FROM equipment_enhance_success_config
WHERE target_level = $1
  AND status = 1
LIMIT 1
`

const runtimeUpdateEnhanceLevelQuery = `
UPDATE equipment_instance
SET enhance_level = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE item_uid = $1
  AND player_id = $3
`

type runtimeEnhanceInstanceRow struct {
	runtimeEquippedRow
	State           string
	CanEnhance      bool
	MaxEnhanceLevel uint32
}

// EnhanceInstance 消耗强化材料并尝试提升背包内未佩戴装备实例的强化等级。
func (r *EquipmentRepository) EnhanceInstance(
	ctx context.Context,
	playerID uint64,
	itemUID string,
	costItemID uint64,
) (*equipment.EnhanceResult, error) {
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
		return nil, err
	}
	if !instanceRow.CanEnhance || instanceRow.MaxEnhanceLevel == 0 {
		return nil, equipment.ErrEquipmentEnhanceNotAllowed
	}
	oldLevel := instanceRow.EnhanceLevel
	if oldLevel >= instanceRow.MaxEnhanceLevel {
		return nil, equipment.ErrEquipmentEnhanceMaxLevel
	}
	targetLevel := oldLevel + 1

	cost, err := loadRuntimeEnhanceCost(ctx, tx, instanceRow.ItemID, targetLevel)
	if err != nil {
		return nil, err
	}
	if cost == nil {
		return nil, equipment.ErrEquipmentEnhanceConfigMissing
	}
	resolvedCostItemID := cost.CostItemID
	if costItemID > 0 {
		resolvedCostItemID = costItemID
	}
	if err := validateEnhanceMaterialItemID(ctx, tx, resolvedCostItemID); err != nil {
		return nil, err
	}
	if cost.CostGoldCopper > 0 {
		if err := ensureRuntimeWalletAffordableInTx(ctx, tx, playerID, cost.CostGoldCopper); err != nil {
			return nil, err
		}
	}
	if err := consumeBagMaterialsByItemIDInTx(ctx, tx, playerID, bag.ContainerTypeBag, resolvedCostItemID, cost.CostQuantity, "player_equipment_enhance"); err != nil {
		return nil, err
	}
	var walletSnapshot *wallet.Snapshot
	if cost.CostGoldCopper > 0 {
		adjustedWallet, err := adjustRuntimeWalletInTx(ctx, tx, playerID, wallet.RuntimeAdjustInput{
			ChangeTotalCopper: -int64(cost.CostGoldCopper),
			ReasonType:        "player_equipment_enhance",
			ReasonRefID:       0,
			OperatorType:      "player",
			OperatorID:        playerID,
		})
		if err != nil {
			return nil, err
		}
		walletSnapshot = &adjustedWallet
	}

	successRatePct, err := loadRuntimeEnhanceSuccessRate(ctx, tx, targetLevel)
	if err != nil {
		return nil, err
	}
	rollPct, success, err := equipment.RollEnhanceSuccess(successRatePct)
	if err != nil {
		return nil, err
	}

	newLevel := oldLevel
	if success {
		newLevel = targetLevel
		if _, err := tx.ExecContext(ctx, runtimeUpdateEnhanceLevelQuery, normalizedUID, newLevel, playerID); err != nil {
			return nil, err
		}
		instanceRow.EnhanceLevel = newLevel
	}

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
	return &equipment.EnhanceResult{
		Success:     success,
		OldLevel:    oldLevel,
		NewLevel:    newLevel,
		RatePct:     successRatePct,
		RollPct:     rollPct,
		Item:        itemSnapshot,
		AllEquipped: allEquipped,
		Wallet:      walletSnapshot,
	}, nil
}

func validateEnhanceInstanceUnequippedInBag(ctx context.Context, tx *sql.Tx, playerID uint64, row *runtimeEnhanceInstanceRow) error {
	if row == nil {
		return equipment.ErrEquipmentNotFound
	}
	if strings.TrimSpace(row.EquipSlot) != "" {
		return equipment.ErrEquipmentEnhanceEquipped
	}
	if strings.TrimSpace(row.State) != "bag" {
		return equipment.ErrEquipmentEnhanceEquipped
	}
	inBag, err := isEquipmentInstanceInBagInTx(ctx, tx, playerID, row.ItemUID)
	if err != nil {
		return err
	}
	if !inBag {
		return equipment.ErrEquipmentEnhanceEquipped
	}
	return nil
}

func isEquipmentInstanceInBagInTx(ctx context.Context, tx *sql.Tx, playerID uint64, itemUID string) (bool, error) {
	var marker int
	err := tx.QueryRowContext(ctx, runtimeEnhanceInstanceInBagQuery, playerID, bag.ContainerTypeBag, itemUID).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func loadRuntimeEnhanceInstanceRow(ctx context.Context, tx *sql.Tx, itemUID string, playerID uint64) (*runtimeEnhanceInstanceRow, error) {
	var row runtimeEnhanceInstanceRow
	var enhanceLevel int64
	var maxEnhanceLevel int64
	var baseHP, baseMana, baseATK, baseDEF, baseSPD int64
	err := tx.QueryRowContext(ctx, runtimeEnhanceInstanceQuery, itemUID, playerID).Scan(
		&row.EquipSlot,
		&row.ItemUID,
		&row.ItemID,
		&enhanceLevel,
		&row.State,
		&row.ItemName,
		&row.CanEnhance,
		&maxEnhanceLevel,
		&row.AppearanceSkinID,
		&row.AppearanceOnly,
		&baseHP,
		&baseMana,
		&baseATK,
		&baseDEF,
		&baseSPD,
		&row.BaseStatsJSON,
		&row.EnhancePerLevelStatsJSON,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.EnhanceLevel = uint32(enhanceLevel)
	row.MaxEnhanceLevel = uint32(maxEnhanceLevel)
	row.BaseHP = uint32(baseHP)
	row.BaseMana = uint32(baseMana)
	row.BaseATK = uint32(baseATK)
	row.BaseDEF = uint32(baseDEF)
	row.BaseSPD = uint32(baseSPD)
	return &row, nil
}

func loadRuntimeEnhanceCost(ctx context.Context, db DBTX, itemID uint64, targetLevel uint32) (*equipment.EnhanceCost, error) {
	var cost equipment.EnhanceCost
	var costItemID int64
	var costQuantity int64
	err := db.QueryRowContext(ctx, runtimeEnhanceCostQuery, targetLevel).Scan(&costItemID, &costQuantity)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	cost.TargetLevel = targetLevel
	cost.CostItemID = uint64(costItemID)
	cost.CostQuantity = uint64(costQuantity)
	goldConfig, err := loadRuntimeEnhanceGoldCostConfigByItemID(ctx, db, itemID)
	if err != nil {
		return nil, err
	}
	if goldConfig != nil {
		cost.CostGoldCopper = equipment.CalculateEnhanceGoldCost(targetLevel, *goldConfig)
	}
	return &cost, nil
}

func loadRuntimeEnhanceSuccessRate(ctx context.Context, db DBTX, targetLevel uint32) (uint32, error) {
	var successRatePct int64
	err := db.QueryRowContext(ctx, runtimeEnhanceSuccessRateQuery, targetLevel).Scan(&successRatePct)
	if errors.Is(err, sql.ErrNoRows) {
		return equipment.DefaultEnhanceSuccessRate(targetLevel), nil
	}
	if err != nil {
		return 0, err
	}
	return uint32(successRatePct), nil
}

func buildRuntimeItemSnapshotFromEnhanceRow(row *runtimeEnhanceInstanceRow) (equipment.RuntimeEquippedItem, error) {
	if row == nil {
		return equipment.RuntimeEquippedItem{}, equipment.ErrEquipmentNotFound
	}
	template, err := row.toPieceTemplate()
	if err != nil {
		return equipment.RuntimeEquippedItem{}, err
	}
	return equipment.ToRuntimeEquippedItem(template, row.ItemUID, row.ItemID, row.ItemName, "", ""), nil
}

// consumeBagMaterialsByItemIDInTx 按 item_id 跨格子扣减可堆叠材料，优先消耗 slot_index 较小的格子。
func consumeBagMaterialsByItemIDInTx(
	ctx context.Context,
	tx *sql.Tx,
	playerID uint64,
	containerType string,
	costItemID uint64,
	needQuantity uint64,
	reasonType string,
) error {
	if needQuantity == 0 {
		return nil
	}
	rows, err := loadTransferTargetRows(ctx, tx, playerID, containerType)
	if err != nil {
		return err
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].SlotIndex < rows[j].SlotIndex
	})

	remaining := needQuantity
	for _, row := range rows {
		if row.ItemID != costItemID {
			continue
		}
		if strings.TrimSpace(row.ItemUID) != "" {
			continue
		}
		if row.Quantity == 0 {
			continue
		}
		consumeQty := row.Quantity
		if consumeQty > remaining {
			consumeQty = remaining
		}
		if consumeQty == row.Quantity {
			if _, err := tx.ExecContext(ctx, runtimeTransferDeleteItemQuery, row.RecordID); err != nil {
				return err
			}
			if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
				PlayerID:      playerID,
				ContainerType: containerType,
				SlotIndex:     row.SlotIndex,
				ChangeType:    "equipment_enhance_remove",
				ItemID:        row.ItemID,
				ItemUID:       row.ItemUID,
				BeforeQty:     row.Quantity,
				ChangeQty:     -int64(consumeQty),
				AfterQty:      0,
				ReasonType:    reasonType,
				OperatorType:  "player",
				OperatorID:    playerID,
			}); err != nil {
				return err
			}
		} else {
			afterQty := row.Quantity - consumeQty
			if err := updateTransferItemQuantity(ctx, tx, row.RecordID, afterQty); err != nil {
				return err
			}
			if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
				PlayerID:      playerID,
				ContainerType: containerType,
				SlotIndex:     row.SlotIndex,
				ChangeType:    "equipment_enhance_reduce",
				ItemID:        row.ItemID,
				ItemUID:       row.ItemUID,
				BeforeQty:     row.Quantity,
				ChangeQty:     -int64(consumeQty),
				AfterQty:      afterQty,
				ReasonType:    reasonType,
				OperatorType:  "player",
				OperatorID:    playerID,
			}); err != nil {
				return err
			}
		}
		remaining -= consumeQty
		if remaining == 0 {
			return nil
		}
	}
	return equipment.ErrEquipmentEnhanceMaterialInsufficient
}

const runtimeEnhanceMaterialSubTypeQuery = `
SELECT COALESCE(item_sub_type, '')
FROM item_definition
WHERE item_id = $1
  AND is_enabled = TRUE
LIMIT 1
`

// validateEnhanceMaterialItemID 校验所选材料属于强化材料子分类。
func validateEnhanceMaterialItemID(ctx context.Context, db DBTX, costItemID uint64) error {
	if costItemID == 0 {
		return equipment.ErrEquipmentEnhanceMaterialInvalid
	}
	var itemSubType string
	err := db.QueryRowContext(ctx, runtimeEnhanceMaterialSubTypeQuery, costItemID).Scan(&itemSubType)
	if errors.Is(err, sql.ErrNoRows) {
		return equipment.ErrEquipmentEnhanceMaterialInvalid
	}
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(itemSubType), bag.ItemSubTypeEquipmentEnhance) {
		return equipment.ErrEquipmentEnhanceMaterialInvalid
	}
	return nil
}

// ensureRuntimeWalletAffordableInTx 校验玩家钱包总铜币是否足够支付强化铜币消耗。
func ensureRuntimeWalletAffordableInTx(ctx context.Context, tx *sql.Tx, playerID uint64, needCopper uint64) error {
	if needCopper == 0 {
		return nil
	}
	var totalCopper uint64
	err := tx.QueryRowContext(ctx, runtimeWalletQuery, playerID).Scan(&totalCopper)
	if errors.Is(err, sql.ErrNoRows) {
		return equipment.ErrEquipmentEnhanceWalletInsufficient
	}
	if err != nil {
		return err
	}
	if totalCopper < needCopper {
		return equipment.ErrEquipmentEnhanceWalletInsufficient
	}
	return nil
}

// adjustRuntimeWalletInTx 在同一数据库事务内扣减强化铜币并写入货币流水。
func adjustRuntimeWalletInTx(ctx context.Context, tx *sql.Tx, playerID uint64, input wallet.RuntimeAdjustInput) (wallet.Snapshot, error) {
	input = input.Normalize()
	if playerID == 0 || input.ChangeTotalCopper == 0 || input.ReasonType == "" || input.OperatorType == "" {
		return wallet.Snapshot{}, wallet.ErrInvalidRuntimeAdjustInput
	}
	var beforeTotal uint64
	if err := tx.QueryRowContext(ctx, runtimeWalletQuery, playerID).Scan(&beforeTotal); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return wallet.Snapshot{}, wallet.ErrWalletNotFound
		}
		return wallet.Snapshot{}, err
	}
	var (
		afterTotal uint64
		version    uint64
		createdAt  sql.NullTime
		updatedAt  sql.NullTime
	)
	if err := tx.QueryRowContext(ctx, adjustWalletTotalQuery, playerID, input.ChangeTotalCopper).Scan(&afterTotal, &version, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return wallet.Snapshot{}, equipment.ErrEquipmentEnhanceWalletInsufficient
		}
		return wallet.Snapshot{}, err
	}
	if _, err := tx.ExecContext(ctx, insertCurrencyChangeLogQuery, playerID, beforeTotal, input.ChangeTotalCopper, afterTotal, input.ReasonType, input.ReasonRefID, input.OperatorType, input.OperatorID); err != nil {
		return wallet.Snapshot{}, err
	}
	return buildWalletSnapshot(afterTotal), nil
}
