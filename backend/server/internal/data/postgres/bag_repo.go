package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"pocket-pet-remake/server/internal/module/bag"
)

// BagRepository 把后台背包与仓库管理映射到新的 player_container_item 持久化结构。
// 后台改动都会直接进入数据库真实格子数据，不再停留在旧版 player_item 的汇总结构里。
type BagRepository struct {
	db DBTX
}

// NewBagRepository 构造后台容器仓储。
func NewBagRepository(db DBTX) *BagRepository {
	return &BagRepository{db: db}
}

const adminBagListBaseQuery = `
SELECT
  pci.id,
  pci.player_id,
  p.name,
  pci.container_type,
  pci.slot_index,
  pci.item_id,
  COALESCE(pci.item_uid, ''),
  COALESCE(idf.item_name, ''),
  COALESCE(idf.item_type, ''),
  pci.quantity,
  pci.is_bound,
  pci.created_at,
  pci.updated_at
FROM player_container_item pci
JOIN player p ON p.id = pci.player_id
LEFT JOIN item_definition idf ON idf.item_id = pci.item_id
`

const adminBagDetailQuery = `
SELECT
  pci.id,
  pci.player_id,
  p.name,
  pci.container_type,
  pci.slot_index,
  pci.item_id,
  COALESCE(pci.item_uid, ''),
  COALESCE(idf.item_name, ''),
  COALESCE(idf.item_type, ''),
  pci.quantity,
  pci.is_bound,
  pci.expire_at,
  pci.created_at,
  pci.updated_at
FROM player_container_item pci
JOIN player p ON p.id = pci.player_id
LEFT JOIN item_definition idf ON idf.item_id = pci.item_id
WHERE pci.id = $1
LIMIT 1
`

const adminBagPlayerExistsQuery = `
SELECT COUNT(1)
FROM player
WHERE id = $1 AND status = 1
`

const adminBagItemDefinitionExistsQuery = `
SELECT COUNT(1)
FROM item_definition
WHERE item_id = $1
`

const insertAdminBagItemQuery = `
INSERT INTO player_container_item (
  player_id,
  container_type,
  slot_index,
  item_id,
  item_uid,
  quantity,
  is_bound
) VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7)
RETURNING id
`

const updateAdminBagItemQuery = `
UPDATE player_container_item
SET player_id = $2,
    container_type = $3,
    slot_index = $4,
    item_id = $5,
    item_uid = NULLIF($6, ''),
    quantity = $7,
    is_bound = $8
WHERE id = $1
`

const deleteAdminBagItemQuery = `
DELETE FROM player_container_item
WHERE id = $1
`

const runtimeContainerMetaQuery = `
SELECT
  capacity,
  max_capacity
FROM player_container
WHERE player_id = $1
  AND container_type = $2
LIMIT 1
`

const runtimeContainerItemsQuery = `
SELECT
  pci.slot_index,
  pci.item_id,
  COALESCE(pci.item_uid, ''),
  pci.quantity,
  pci.is_bound,
  pci.expire_at,
  COALESCE(idf.item_name, ''),
  COALESCE(idf.item_type, ''),
  COALESCE(idf.item_sub_type, ''),
  COALESCE(idf.quality, 1),
  COALESCE(idf.icon, ''),
  COALESCE(eq.enhance_level, 0)
FROM player_container_item pci
LEFT JOIN item_definition idf ON idf.item_id = pci.item_id
LEFT JOIN equipment_instance eq ON eq.item_uid = pci.item_uid
WHERE pci.player_id = $1
  AND pci.container_type = $2
ORDER BY pci.slot_index ASC
`

const runtimeTransferSourceItemQuery = `
SELECT
  pci.id,
  pci.slot_index,
  pci.item_id,
  COALESCE(pci.item_uid, ''),
  pci.quantity,
  pci.is_bound,
  pci.expire_at,
  COALESCE(idf.max_stack, 1),
  COALESCE(idf.can_store, TRUE)
FROM player_container_item pci
JOIN item_definition idf ON idf.item_id = pci.item_id
WHERE pci.player_id = $1
  AND pci.container_type = $2
  AND pci.slot_index = $3
LIMIT 1
`

const runtimeTransferContainerMetaQuery = `
SELECT capacity, max_capacity
FROM player_container
WHERE player_id = $1
  AND container_type = $2
LIMIT 1
`

const runtimeTransferTargetRowsQuery = `
SELECT
  id,
  slot_index,
  item_id,
  COALESCE(item_uid, ''),
  quantity,
  is_bound
FROM player_container_item
WHERE player_id = $1
  AND container_type = $2
ORDER BY slot_index ASC
`

const runtimeTransferUpdateQuantityQuery = `
UPDATE player_container_item
SET quantity = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
`

const runtimeTransferDeleteItemQuery = `
DELETE FROM player_container_item
WHERE id = $1
`

const runtimeTransferInsertItemQuery = `
INSERT INTO player_container_item (
  player_id,
  container_type,
  slot_index,
  item_id,
  item_uid,
  quantity,
  is_bound,
  expire_at
) VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8)
`

const runtimeTransferUpdateEquipmentStateQuery = `
UPDATE equipment_instance
SET state = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE item_uid = $1
`

const runtimeUpdateSlotIndexQuery = `
UPDATE player_container_item
SET slot_index = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
`

const runtimeSortRowsQuery = `
SELECT
  pci.id,
  pci.slot_index,
  pci.item_id,
  COALESCE(pci.item_uid, ''),
  pci.quantity,
  COALESCE(idf.sort_weight, 0),
  COALESCE(idf.quality, 1)
FROM player_container_item pci
LEFT JOIN item_definition idf ON idf.item_id = pci.item_id
WHERE pci.player_id = $1
  AND pci.container_type = $2
ORDER BY pci.slot_index ASC
`

const runtimeGrantItemDefinitionQuery = `
SELECT
  COALESCE(item_name, ''),
  COALESCE(max_stack, 1),
  COALESCE(bind_type, 'none')
FROM item_definition
WHERE item_id = $1
LIMIT 1
`

const runtimeUseItemQuery = `
SELECT
  pci.id,
  pci.slot_index,
  pci.item_id,
  COALESCE(pci.item_uid, ''),
  pci.quantity,
  pci.is_bound,
  pci.expire_at,
  COALESCE(idf.item_type, ''),
  COALESCE(idf.usable, FALSE),
  COALESCE(idf.target_type, ''),
  COALESCE(idf.effect_type, ''),
  COALESCE(idf.effect_value, 0),
  COALESCE(idf.effect_params_json, '{}'::jsonb),
  COALESCE(ife.expand_target, ''),
  COALESCE(ife.expand_slots, 0)
FROM player_container_item pci
JOIN item_definition idf ON idf.item_id = pci.item_id
LEFT JOIN item_functional_extra ife ON ife.item_id = idf.item_id
WHERE pci.player_id = $1
  AND pci.container_type = $2
  AND pci.slot_index = $3
LIMIT 1
`

const runtimePetUseTargetQuery = `
SELECT
  pp.id,
  pp.pet_id,
  pp.level,
  pp.exp,
  pp.quality,
  pp.hp,
  pp.hp_max,
  pp.atk,
  pp.def,
  pp.spd,
  pp.skill_ids,
  EXISTS(SELECT 1 FROM player_lineup pl WHERE pl.player_id = pp.player_id AND pl.pet_uid = pp.id) AS in_lineup
FROM player_pet pp
WHERE pp.player_id = $1
  AND pp.id = $2
LIMIT 1
`

const runtimeUpdatePetHPByUIDQuery = `
UPDATE player_pet
SET hp = LEAST($3, hp_max)
WHERE player_id = $1 AND id = $2
`

const runtimeContainerCapacityDetailQuery = `
SELECT capacity, max_capacity
FROM player_container
WHERE player_id = $1
  AND container_type = $2
LIMIT 1
`

const runtimeUpdateContainerCapacityQuery = `
UPDATE player_container
SET capacity = $3,
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE player_id = $1
  AND container_type = $2
`

const insertContainerExpandLogQuery = `
INSERT INTO container_expand_log (
  player_id,
  container_type,
  before_capacity,
  expand_slots,
  after_capacity,
  item_id,
  reason_type,
  reason_ref_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

const insertItemChangeLogQuery = `
INSERT INTO item_change_log (
  player_id,
  container_type,
  slot_index,
  change_type,
  item_id,
  item_uid,
  before_qty,
  change_qty,
  after_qty,
  reason_type,
  reason_ref_id,
  operator_type,
  operator_id
) VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, $9, $10, $11, $12, $13)
`

func (r *BagRepository) ListForAdmin(ctx context.Context, query bag.AdminListQuery) (*bag.AdminItemList, error) {
	query = query.Normalize()
	conditions := make([]string, 0, 5)
	args := make([]any, 0, 7)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.RecordID > 0 {
		conditions = append(conditions, "pci.id = "+nextArg(query.RecordID))
	}
	if query.PlayerID > 0 {
		conditions = append(conditions, "pci.player_id = "+nextArg(query.PlayerID))
	}
	if query.ContainerType != "" {
		conditions = append(conditions, "pci.container_type = "+nextArg(query.ContainerType))
	}
	if query.ItemID > 0 {
		conditions = append(conditions, "pci.item_id = "+nextArg(query.ItemID))
	}
	if query.ItemUID != "" {
		conditions = append(conditions, "pci.item_uid = "+nextArg(query.ItemUID))
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}
	countQuery := `
SELECT COUNT(1)
FROM player_container_item pci
JOIN player p ON p.id = pci.player_id
LEFT JOIN item_definition idf ON idf.item_id = pci.item_id
` + whereClause
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	listQuery := adminBagListBaseQuery + whereClause + fmt.Sprintf("\nORDER BY pci.id DESC\nLIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]bag.AdminItemSummary, 0)
	for rows.Next() {
		itemValue, err := scanAdminBagSummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, itemValue)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &bag.AdminItemList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *BagRepository) FindAdminDetailByRecordID(ctx context.Context, recordID uint64) (*bag.AdminItemDetail, error) {
	itemValue, err := scanAdminBagDetailRow(r.db.QueryRowContext(ctx, adminBagDetailQuery, recordID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return itemValue, nil
}

func (r *BagRepository) CreateForAdmin(ctx context.Context, input bag.AdminCreateItemInput) (*bag.AdminItemDetail, error) {
	if ok, err := r.playerExists(ctx, input.PlayerID); err != nil {
		return nil, err
	} else if !ok {
		return nil, bag.ErrBagItemNotFound
	}
	if ok, err := r.itemExists(ctx, input.ItemID); err != nil {
		return nil, err
	} else if !ok {
		return nil, bag.ErrBagItemNotFound
	}
	var recordID int64
	if err := r.db.QueryRowContext(ctx, insertAdminBagItemQuery,
		input.PlayerID,
		input.ContainerType,
		input.SlotIndex,
		input.ItemID,
		input.ItemUID,
		input.Quantity,
		input.IsBound,
	).Scan(&recordID); err != nil {
		if isPlayerContainerItemUniqueViolation(err) {
			return nil, bag.ErrBagItemConflict
		}
		return nil, err
	}
	return r.FindAdminDetailByRecordID(ctx, uint64(recordID))
}

func (r *BagRepository) UpdateForAdmin(ctx context.Context, recordID uint64, input bag.AdminUpdateItemInput) (*bag.AdminItemDetail, error) {
	if ok, err := r.playerExists(ctx, input.PlayerID); err != nil {
		return nil, err
	} else if !ok {
		return nil, bag.ErrBagItemNotFound
	}
	if ok, err := r.itemExists(ctx, input.ItemID); err != nil {
		return nil, err
	} else if !ok {
		return nil, bag.ErrBagItemNotFound
	}
	result, err := r.db.ExecContext(ctx, updateAdminBagItemQuery,
		recordID,
		input.PlayerID,
		input.ContainerType,
		input.SlotIndex,
		input.ItemID,
		input.ItemUID,
		input.Quantity,
		input.IsBound,
	)
	if err != nil {
		if isPlayerContainerItemUniqueViolation(err) {
			return nil, bag.ErrBagItemConflict
		}
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, bag.ErrBagItemNotFound
	}
	return r.FindAdminDetailByRecordID(ctx, recordID)
}

func (r *BagRepository) DeleteForAdmin(ctx context.Context, recordID uint64) error {
	result, err := r.db.ExecContext(ctx, deleteAdminBagItemQuery, recordID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return bag.ErrBagItemNotFound
	}
	return nil
}

func (r *BagRepository) ListRuntimeContainer(ctx context.Context, playerID uint64, containerType string) (*bag.RuntimeContainerSnapshot, error) {
	var snapshot bag.RuntimeContainerSnapshot
	snapshot.ContainerType = containerType
	if err := r.db.QueryRowContext(ctx, runtimeContainerMetaQuery, playerID, containerType).Scan(&snapshot.Capacity, &snapshot.MaxCapacity); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, runtimeContainerItemsQuery, playerID, containerType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]bag.RuntimeItemSnapshot, 0)
	for rows.Next() {
		var (
			value    bag.RuntimeItemSnapshot
			expireAt sql.NullTime
		)
		if err := rows.Scan(
			&value.SlotIndex,
			&value.ItemID,
			&value.ItemUID,
			&value.Quantity,
			&value.IsBound,
			&expireAt,
			&value.ItemName,
			&value.ItemType,
			&value.ItemSubType,
			&value.Quality,
			&value.Icon,
			&value.EnhanceLevel,
		); err != nil {
			return nil, err
		}
		if expireAt.Valid {
			expireAtValue := expireAt.Time
			value.ExpireAt = &expireAtValue
		}
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	snapshot.Items = items
	snapshot.UsedSlots = uint32(len(items))
	return &snapshot, nil
}

func (r *BagRepository) TransferRuntimeItem(ctx context.Context, playerID uint64, fromContainerType string, toContainerType string, fromSlotIndex uint32, quantity uint64) (*bag.RuntimeTransferResult, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	sourceRow, err := loadTransferSourceRow(ctx, tx, playerID, fromContainerType, fromSlotIndex)
	if err != nil {
		return nil, err
	}
	if sourceRow == nil {
		return nil, bag.ErrContainerItemNotFound
	}
	if quantity > sourceRow.Quantity {
		return nil, bag.ErrInvalidTransferQuantity
	}
	if sourceRow.ItemUID != "" && quantity != sourceRow.Quantity {
		return nil, bag.ErrInvalidTransferQuantity
	}
	if toContainerType == bag.ContainerTypeWarehouse && !sourceRow.CanStore {
		return nil, bag.ErrItemCannotStore
	}

	targetCapacity, err := loadTransferContainerCapacity(ctx, tx, playerID, toContainerType)
	if err != nil {
		return nil, err
	}
	if targetCapacity == 0 {
		return nil, bag.ErrContainerNotFound
	}
	targetRows, err := loadTransferTargetRows(ctx, tx, playerID, toContainerType)
	if err != nil {
		return nil, err
	}

	remainingQuantity := quantity
	targetSlotIndex := uint32(0)
	if sourceRow.ItemUID == "" && sourceRow.MaxStack > 1 {
		for _, targetRow := range targetRows {
			if targetRow.ItemID != sourceRow.ItemID || targetRow.ItemUID != "" || targetRow.IsBound != sourceRow.IsBound {
				continue
			}
			if targetRow.Quantity >= sourceRow.MaxStack {
				continue
			}
			available := sourceRow.MaxStack - targetRow.Quantity
			moveQuantity := minUint64(available, remainingQuantity)
			if err := updateTransferItemQuantity(ctx, tx, targetRow.RecordID, targetRow.Quantity+moveQuantity); err != nil {
				return nil, err
			}
			if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
				PlayerID:      playerID,
				ContainerType: toContainerType,
				SlotIndex:     targetRow.SlotIndex,
				ChangeType:    "transfer_in_merge",
				ItemID:        sourceRow.ItemID,
				ItemUID:       sourceRow.ItemUID,
				BeforeQty:     targetRow.Quantity,
				ChangeQty:     int64(moveQuantity),
				AfterQty:      targetRow.Quantity + moveQuantity,
				ReasonType:    buildTransferReasonType(fromContainerType, toContainerType),
				OperatorType:  "player",
				OperatorID:    playerID,
			}); err != nil {
				return nil, err
			}
			targetSlotIndex = targetRow.SlotIndex
			remainingQuantity -= moveQuantity
			break
		}
	}

	if remainingQuantity > 0 {
		targetSlotIndex = firstEmptySlotIndex(targetRows, targetCapacity)
		if targetSlotIndex == 0 {
			return nil, bag.ErrContainerCapacityFull
		}
		if _, err := tx.ExecContext(ctx, runtimeTransferInsertItemQuery,
			playerID,
			toContainerType,
			targetSlotIndex,
			sourceRow.ItemID,
			sourceRow.ItemUID,
			remainingQuantity,
			sourceRow.IsBound,
			sourceRow.ExpireAt,
		); err != nil {
			if isPlayerContainerItemUniqueViolation(err) {
				return nil, bag.ErrContainerCapacityFull
			}
			return nil, err
		}
		if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
			PlayerID:      playerID,
			ContainerType: toContainerType,
			SlotIndex:     targetSlotIndex,
			ChangeType:    "transfer_in_add",
			ItemID:        sourceRow.ItemID,
			ItemUID:       sourceRow.ItemUID,
			BeforeQty:     0,
			ChangeQty:     int64(remainingQuantity),
			AfterQty:      remainingQuantity,
			ReasonType:    buildTransferReasonType(fromContainerType, toContainerType),
			OperatorType:  "player",
			OperatorID:    playerID,
		}); err != nil {
			return nil, err
		}
	}

	if quantity == sourceRow.Quantity {
		if _, err := tx.ExecContext(ctx, runtimeTransferDeleteItemQuery, sourceRow.RecordID); err != nil {
			return nil, err
		}
		if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
			PlayerID:      playerID,
			ContainerType: fromContainerType,
			SlotIndex:     fromSlotIndex,
			ChangeType:    "transfer_out_remove",
			ItemID:        sourceRow.ItemID,
			ItemUID:       sourceRow.ItemUID,
			BeforeQty:     sourceRow.Quantity,
			ChangeQty:     -int64(quantity),
			AfterQty:      0,
			ReasonType:    buildTransferReasonType(fromContainerType, toContainerType),
			OperatorType:  "player",
			OperatorID:    playerID,
		}); err != nil {
			return nil, err
		}
	} else {
		if err := updateTransferItemQuantity(ctx, tx, sourceRow.RecordID, sourceRow.Quantity-quantity); err != nil {
			return nil, err
		}
		if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
			PlayerID:      playerID,
			ContainerType: fromContainerType,
			SlotIndex:     fromSlotIndex,
			ChangeType:    "transfer_out_reduce",
			ItemID:        sourceRow.ItemID,
			ItemUID:       sourceRow.ItemUID,
			BeforeQty:     sourceRow.Quantity,
			ChangeQty:     -int64(quantity),
			AfterQty:      sourceRow.Quantity - quantity,
			ReasonType:    buildTransferReasonType(fromContainerType, toContainerType),
			OperatorType:  "player",
			OperatorID:    playerID,
		}); err != nil {
			return nil, err
		}
	}

	if sourceRow.ItemUID != "" {
		if _, err := tx.ExecContext(ctx, runtimeTransferUpdateEquipmentStateQuery, sourceRow.ItemUID, toContainerType); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &bag.RuntimeTransferResult{
		MovedItemID:       sourceRow.ItemID,
		MovedItemUID:      sourceRow.ItemUID,
		MovedQuantity:     quantity,
		FromContainerType: fromContainerType,
		ToContainerType:   toContainerType,
		FromSlotIndex:     fromSlotIndex,
		ToSlotIndex:       targetSlotIndex,
	}, nil
}

func (r *BagRepository) SortRuntimeContainer(ctx context.Context, playerID uint64, containerType string) (*bag.RuntimeSortResult, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	capacity, err := loadTransferContainerCapacity(ctx, tx, playerID, containerType)
	if err != nil {
		return nil, err
	}
	if capacity == 0 {
		return nil, bag.ErrContainerNotFound
	}
	rows, err := loadSortRows(ctx, tx, playerID, containerType)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return &bag.RuntimeSortResult{ContainerType: containerType, Sorted: true}, nil
	}

	sort.SliceStable(rows, func(left int, right int) bool {
		if rows[left].SortWeight != rows[right].SortWeight {
			return rows[left].SortWeight > rows[right].SortWeight
		}
		if rows[left].Quality != rows[right].Quality {
			return rows[left].Quality > rows[right].Quality
		}
		if rows[left].ItemID != rows[right].ItemID {
			return rows[left].ItemID < rows[right].ItemID
		}
		if rows[left].ItemUID != rows[right].ItemUID {
			return rows[left].ItemUID < rows[right].ItemUID
		}
		return rows[left].SlotIndex < rows[right].SlotIndex
	})

	tempBase := capacity + 1000
	for index, row := range rows {
		if err := updateTransferSlotIndex(ctx, tx, row.RecordID, tempBase+uint32(index)); err != nil {
			return nil, err
		}
	}
	for index, row := range rows {
		if err := updateTransferSlotIndex(ctx, tx, row.RecordID, uint32(index+1)); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &bag.RuntimeSortResult{ContainerType: containerType, Sorted: true}, nil
}

func (r *BagRepository) MoveRuntimeItem(ctx context.Context, playerID uint64, containerType string, fromSlotIndex uint32, toSlotIndex uint32, quantity uint64) (*bag.RuntimeMoveResult, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	capacity, err := loadTransferContainerCapacity(ctx, tx, playerID, containerType)
	if err != nil {
		return nil, err
	}
	if capacity == 0 {
		return nil, bag.ErrContainerNotFound
	}
	if fromSlotIndex == toSlotIndex || fromSlotIndex > capacity || toSlotIndex > capacity {
		return nil, bag.ErrInvalidContainerMove
	}

	sourceRow, err := loadTransferSourceRow(ctx, tx, playerID, containerType, fromSlotIndex)
	if err != nil {
		return nil, err
	}
	if sourceRow == nil || quantity > sourceRow.Quantity {
		return nil, bag.ErrInvalidContainerMove
	}
	if sourceRow.ItemUID != "" && quantity != sourceRow.Quantity {
		return nil, bag.ErrInvalidContainerMove
	}
	targetRow, err := loadTransferSourceRow(ctx, tx, playerID, containerType, toSlotIndex)
	if err != nil {
		return nil, err
	}

	if targetRow == nil {
		if quantity == sourceRow.Quantity {
			if err := updateTransferSlotIndex(ctx, tx, sourceRow.RecordID, toSlotIndex); err != nil {
				return nil, err
			}
			if err := insertMoveOutInLogs(ctx, tx, playerID, containerType, sourceRow.ItemID, sourceRow.ItemUID, fromSlotIndex, toSlotIndex, sourceRow.Quantity, "container_move_full"); err != nil {
				return nil, err
			}
		} else {
			if err := updateTransferItemQuantity(ctx, tx, sourceRow.RecordID, sourceRow.Quantity-quantity); err != nil {
				return nil, err
			}
			if _, err := tx.ExecContext(ctx, runtimeTransferInsertItemQuery,
				playerID,
				containerType,
				toSlotIndex,
				sourceRow.ItemID,
				sourceRow.ItemUID,
				quantity,
				sourceRow.IsBound,
				sourceRow.ExpireAt,
			); err != nil {
				return nil, err
			}
			if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
				PlayerID:      playerID,
				ContainerType: containerType,
				SlotIndex:     fromSlotIndex,
				ChangeType:    "container_split_out",
				ItemID:        sourceRow.ItemID,
				ItemUID:       sourceRow.ItemUID,
				BeforeQty:     sourceRow.Quantity,
				ChangeQty:     -int64(quantity),
				AfterQty:      sourceRow.Quantity - quantity,
				ReasonType:    "container_move",
				OperatorType:  "player",
				OperatorID:    playerID,
			}); err != nil {
				return nil, err
			}
			if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
				PlayerID:      playerID,
				ContainerType: containerType,
				SlotIndex:     toSlotIndex,
				ChangeType:    "container_split_in",
				ItemID:        sourceRow.ItemID,
				ItemUID:       sourceRow.ItemUID,
				BeforeQty:     0,
				ChangeQty:     int64(quantity),
				AfterQty:      quantity,
				ReasonType:    "container_move",
				OperatorType:  "player",
				OperatorID:    playerID,
			}); err != nil {
				return nil, err
			}
		}
	} else if sourceRow.ItemUID == "" && targetRow.ItemUID == "" && sourceRow.ItemID == targetRow.ItemID && sourceRow.IsBound == targetRow.IsBound && sourceRow.MaxStack > 1 {
		available := sourceRow.MaxStack - targetRow.Quantity
		if available == 0 {
			return nil, bag.ErrInvalidContainerMove
		}
		moveQuantity := minUint64(quantity, available)
		if err := updateTransferItemQuantity(ctx, tx, targetRow.RecordID, targetRow.Quantity+moveQuantity); err != nil {
			return nil, err
		}
		if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
			PlayerID:      playerID,
			ContainerType: containerType,
			SlotIndex:     toSlotIndex,
			ChangeType:    "container_merge_in",
			ItemID:        sourceRow.ItemID,
			ItemUID:       sourceRow.ItemUID,
			BeforeQty:     targetRow.Quantity,
			ChangeQty:     int64(moveQuantity),
			AfterQty:      targetRow.Quantity + moveQuantity,
			ReasonType:    "container_move",
			OperatorType:  "player",
			OperatorID:    playerID,
		}); err != nil {
			return nil, err
		}
		if moveQuantity == sourceRow.Quantity {
			if _, err := tx.ExecContext(ctx, runtimeTransferDeleteItemQuery, sourceRow.RecordID); err != nil {
				return nil, err
			}
			if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
				PlayerID:      playerID,
				ContainerType: containerType,
				SlotIndex:     fromSlotIndex,
				ChangeType:    "container_merge_out_remove",
				ItemID:        sourceRow.ItemID,
				ItemUID:       sourceRow.ItemUID,
				BeforeQty:     sourceRow.Quantity,
				ChangeQty:     -int64(moveQuantity),
				AfterQty:      0,
				ReasonType:    "container_move",
				OperatorType:  "player",
				OperatorID:    playerID,
			}); err != nil {
				return nil, err
			}
		} else {
			if err := updateTransferItemQuantity(ctx, tx, sourceRow.RecordID, sourceRow.Quantity-moveQuantity); err != nil {
				return nil, err
			}
			if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
				PlayerID:      playerID,
				ContainerType: containerType,
				SlotIndex:     fromSlotIndex,
				ChangeType:    "container_merge_out_reduce",
				ItemID:        sourceRow.ItemID,
				ItemUID:       sourceRow.ItemUID,
				BeforeQty:     sourceRow.Quantity,
				ChangeQty:     -int64(moveQuantity),
				AfterQty:      sourceRow.Quantity - moveQuantity,
				ReasonType:    "container_move",
				OperatorType:  "player",
				OperatorID:    playerID,
			}); err != nil {
				return nil, err
			}
		}
	} else {
		if quantity != sourceRow.Quantity || targetRow.ItemUID != "" && targetRow.Quantity != 1 {
			return nil, bag.ErrInvalidContainerMove
		}
		tempSlotIndex := capacity + 1000
		if err := updateTransferSlotIndex(ctx, tx, sourceRow.RecordID, tempSlotIndex); err != nil {
			return nil, err
		}
		if err := updateTransferSlotIndex(ctx, tx, targetRow.RecordID, fromSlotIndex); err != nil {
			return nil, err
		}
		if err := updateTransferSlotIndex(ctx, tx, sourceRow.RecordID, toSlotIndex); err != nil {
			return nil, err
		}
		if err := insertMoveOutInLogs(ctx, tx, playerID, containerType, sourceRow.ItemID, sourceRow.ItemUID, fromSlotIndex, toSlotIndex, sourceRow.Quantity, "container_swap"); err != nil {
			return nil, err
		}
		if err := insertMoveOutInLogs(ctx, tx, playerID, containerType, targetRow.ItemID, targetRow.ItemUID, toSlotIndex, fromSlotIndex, targetRow.Quantity, "container_swap"); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &bag.RuntimeMoveResult{
		ContainerType: containerType,
		FromSlotIndex: fromSlotIndex,
		ToSlotIndex:   toSlotIndex,
		Moved:         true,
	}, nil
}

func (r *BagRepository) GrantRuntimeItem(ctx context.Context, playerID uint64, containerType string, itemID uint64, quantity uint64, reasonType string, reasonRefID uint64, operatorType string, operatorID uint64) (*bag.RuntimeGrantResult, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	capacity, err := loadTransferContainerCapacity(ctx, tx, playerID, containerType)
	if err != nil {
		return nil, err
	}
	if capacity == 0 {
		return nil, bag.ErrContainerNotFound
	}
	itemDef, err := loadGrantItemDefinition(ctx, tx, itemID)
	if err != nil {
		return nil, err
	}
	if itemDef == nil {
		return nil, bag.ErrBagItemNotFound
	}
	targetRows, err := loadTransferTargetRows(ctx, tx, playerID, containerType)
	if err != nil {
		return nil, err
	}

	grantedSlotIndex := uint32(0)
	remainingQuantity := quantity
	isBound := itemDef.BindType != "" && !strings.EqualFold(itemDef.BindType, "none")
	if itemDef.MaxStack > 1 {
		for _, targetRow := range targetRows {
			if targetRow.ItemID != itemID || targetRow.ItemUID != "" || targetRow.IsBound != isBound {
				continue
			}
			if targetRow.Quantity >= itemDef.MaxStack {
				continue
			}
			available := itemDef.MaxStack - targetRow.Quantity
			moveQuantity := minUint64(available, remainingQuantity)
			if err := updateTransferItemQuantity(ctx, tx, targetRow.RecordID, targetRow.Quantity+moveQuantity); err != nil {
				return nil, err
			}
			if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
				PlayerID:      playerID,
				ContainerType: containerType,
				SlotIndex:     targetRow.SlotIndex,
				ChangeType:    "reward_merge_in",
				ItemID:        itemID,
				ItemUID:       "",
				BeforeQty:     targetRow.Quantity,
				ChangeQty:     int64(moveQuantity),
				AfterQty:      targetRow.Quantity + moveQuantity,
				ReasonType:    reasonType,
				ReasonRefID:   reasonRefID,
				OperatorType:  operatorType,
				OperatorID:    operatorID,
			}); err != nil {
				return nil, err
			}
			grantedSlotIndex = targetRow.SlotIndex
			remainingQuantity -= moveQuantity
			if remainingQuantity == 0 {
				break
			}
		}
	}

	for remainingQuantity > 0 {
		slotIndex := firstEmptySlotIndex(targetRows, capacity)
		if slotIndex == 0 {
			return nil, bag.ErrContainerCapacityFull
		}
		stackQuantity := remainingQuantity
		if itemDef.MaxStack > 1 {
			stackQuantity = minUint64(itemDef.MaxStack, remainingQuantity)
		}
		if _, err := tx.ExecContext(ctx, runtimeTransferInsertItemQuery,
			playerID,
			containerType,
			slotIndex,
			itemID,
			"",
			stackQuantity,
			isBound,
			nil,
		); err != nil {
			if isPlayerContainerItemUniqueViolation(err) {
				return nil, bag.ErrContainerCapacityFull
			}
			return nil, err
		}
		if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
			PlayerID:      playerID,
			ContainerType: containerType,
			SlotIndex:     slotIndex,
			ChangeType:    "reward_add",
			ItemID:        itemID,
			ItemUID:       "",
			BeforeQty:     0,
			ChangeQty:     int64(stackQuantity),
			AfterQty:      stackQuantity,
			ReasonType:    reasonType,
			ReasonRefID:   reasonRefID,
			OperatorType:  operatorType,
			OperatorID:    operatorID,
		}); err != nil {
			return nil, err
		}
		targetRows = append(targetRows, transferTargetRow{
			SlotIndex: slotIndex,
			ItemID:    itemID,
			ItemUID:   "",
			Quantity:  stackQuantity,
			IsBound:   isBound,
		})
		grantedSlotIndex = slotIndex
		remainingQuantity -= stackQuantity
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &bag.RuntimeGrantResult{
		ContainerType: containerType,
		ItemID:        itemID,
		ItemName:      itemDef.ItemName,
		ItemUID:       "",
		GrantedQty:    quantity,
		SlotIndex:     grantedSlotIndex,
	}, nil
}

// UseRuntimeItem 处理玩家主动使用格子物品。
// 这里统一在同一个数据库事务里完成扣道具、结算效果与日志写入，避免背包和目标状态分叉。
func (r *BagRepository) UseRuntimeItem(ctx context.Context, playerID uint64, containerType string, slotIndex uint32, quantity uint64, targetPetUID uint64, targetPlayerID uint64) (*bag.RuntimeUseResult, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	sourceRow, err := loadUseItemSourceRow(ctx, tx, playerID, containerType, slotIndex)
	if err != nil {
		return nil, err
	}
	if sourceRow == nil {
		return nil, bag.ErrContainerItemNotFound
	}
	if quantity == 0 || quantity > sourceRow.Quantity {
		return nil, bag.ErrInvalidTransferQuantity
	}
	if sourceRow.ItemUID != "" && quantity != sourceRow.Quantity {
		return nil, bag.ErrInvalidTransferQuantity
	}
	if !sourceRow.Usable {
		return nil, bag.ErrItemNotUsable
	}

	useResult, err := applyRuntimeItemEffect(ctx, tx, playerID, sourceRow, quantity, targetPetUID, targetPlayerID)
	if err != nil {
		return nil, err
	}

	if quantity == sourceRow.Quantity {
		if _, err := tx.ExecContext(ctx, runtimeTransferDeleteItemQuery, sourceRow.RecordID); err != nil {
			return nil, err
		}
		if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
			PlayerID:      playerID,
			ContainerType: containerType,
			SlotIndex:     slotIndex,
			ChangeType:    "use_remove",
			ItemID:        sourceRow.ItemID,
			ItemUID:       sourceRow.ItemUID,
			BeforeQty:     sourceRow.Quantity,
			ChangeQty:     -int64(quantity),
			AfterQty:      0,
			ReasonType:    "item_use",
			OperatorType:  "player",
			OperatorID:    playerID,
		}); err != nil {
			return nil, err
		}
	} else {
		if err := updateTransferItemQuantity(ctx, tx, sourceRow.RecordID, sourceRow.Quantity-quantity); err != nil {
			return nil, err
		}
		if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
			PlayerID:      playerID,
			ContainerType: containerType,
			SlotIndex:     slotIndex,
			ChangeType:    "use_reduce",
			ItemID:        sourceRow.ItemID,
			ItemUID:       sourceRow.ItemUID,
			BeforeQty:     sourceRow.Quantity,
			ChangeQty:     -int64(quantity),
			AfterQty:      sourceRow.Quantity - quantity,
			ReasonType:    "item_use",
			OperatorType:  "player",
			OperatorID:    playerID,
		}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &bag.RuntimeUseResult{
		ContainerType: containerType,
		SlotIndex:     slotIndex,
		ItemID:        sourceRow.ItemID,
		UsedQuantity:  quantity,
		Result:        useResult,
	}, nil
}

// ConsumeRuntimeItemStack 只扣减背包数量并写变更日志，不触发道具使用效果。
func (r *BagRepository) ConsumeRuntimeItemStack(ctx context.Context, playerID uint64, containerType string, slotIndex uint32, quantity uint64, reasonType string, reasonRefID uint64) (*bag.RuntimeContainerSnapshot, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	sourceRow, err := loadUseItemSourceRow(ctx, tx, playerID, containerType, slotIndex)
	if err != nil {
		return nil, err
	}
	if sourceRow == nil {
		return nil, bag.ErrContainerItemNotFound
	}
	if quantity == 0 || quantity > sourceRow.Quantity {
		return nil, bag.ErrInvalidTransferQuantity
	}
	if sourceRow.ItemUID != "" && quantity != sourceRow.Quantity {
		return nil, bag.ErrInvalidTransferQuantity
	}

	if quantity == sourceRow.Quantity {
		if _, err := tx.ExecContext(ctx, runtimeTransferDeleteItemQuery, sourceRow.RecordID); err != nil {
			return nil, err
		}
		if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
			PlayerID:      playerID,
			ContainerType: containerType,
			SlotIndex:     slotIndex,
			ChangeType:    "use_remove",
			ItemID:        sourceRow.ItemID,
			ItemUID:       sourceRow.ItemUID,
			BeforeQty:     sourceRow.Quantity,
			ChangeQty:     -int64(quantity),
			AfterQty:      0,
			ReasonType:    reasonType,
			ReasonRefID:   reasonRefID,
			OperatorType:  "player",
			OperatorID:    playerID,
		}); err != nil {
			return nil, err
		}
	} else {
		if err := updateTransferItemQuantity(ctx, tx, sourceRow.RecordID, sourceRow.Quantity-quantity); err != nil {
			return nil, err
		}
		if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
			PlayerID:      playerID,
			ContainerType: containerType,
			SlotIndex:     slotIndex,
			ChangeType:    "use_reduce",
			ItemID:        sourceRow.ItemID,
			ItemUID:       sourceRow.ItemUID,
			BeforeQty:     sourceRow.Quantity,
			ChangeQty:     -int64(quantity),
			AfterQty:      sourceRow.Quantity - quantity,
			ReasonType:    reasonType,
			ReasonRefID:   reasonRefID,
			OperatorType:  "player",
			OperatorID:    playerID,
		}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.ListRuntimeContainer(ctx, playerID, containerType)
}

func (r *BagRepository) playerExists(ctx context.Context, playerID uint64) (bool, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, adminBagPlayerExistsQuery, playerID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *BagRepository) itemExists(ctx context.Context, itemID uint64) (bool, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, adminBagItemDefinitionExistsQuery, itemID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func scanAdminBagSummary(rows *sql.Rows) (bag.AdminItemSummary, error) {
	var value bag.AdminItemSummary
	if err := rows.Scan(
		&value.RecordID,
		&value.PlayerID,
		&value.PlayerName,
		&value.ContainerType,
		&value.SlotIndex,
		&value.ItemID,
		&value.ItemUID,
		&value.ItemName,
		&value.ItemType,
		&value.Quantity,
		&value.IsBound,
		&value.CreatedAt,
		&value.UpdatedAt,
	); err != nil {
		return bag.AdminItemSummary{}, err
	}
	return value, nil
}

func scanAdminBagDetailRow(row *sql.Row) (*bag.AdminItemDetail, error) {
	var value bag.AdminItemDetail
	if err := row.Scan(
		&value.RecordID,
		&value.PlayerID,
		&value.PlayerName,
		&value.ContainerType,
		&value.SlotIndex,
		&value.ItemID,
		&value.ItemUID,
		&value.ItemName,
		&value.ItemType,
		&value.Quantity,
		&value.IsBound,
		&value.ExpireAt,
		&value.CreatedAt,
		&value.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &value, nil
}

func isPlayerContainerItemUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type transferSourceRow struct {
	RecordID  int64
	SlotIndex uint32
	ItemID    uint64
	ItemUID   string
	Quantity  uint64
	IsBound   bool
	ExpireAt  sql.NullTime
	MaxStack  uint64
	CanStore  bool
}

type transferTargetRow struct {
	RecordID  int64
	SlotIndex uint32
	ItemID    uint64
	ItemUID   string
	Quantity  uint64
	IsBound   bool
}

type sortRow struct {
	RecordID   int64
	SlotIndex  uint32
	ItemID     uint64
	ItemUID    string
	Quantity   uint64
	SortWeight int64
	Quality    int64
}

type grantItemDefinitionRow struct {
	ItemName string
	MaxStack uint64
	BindType string
}

type useItemSourceRow struct {
	RecordID     int64
	SlotIndex    uint32
	ItemID       uint64
	ItemUID      string
	Quantity     uint64
	IsBound      bool
	ExpireAt     sql.NullTime
	ItemType     string
	Usable       bool
	TargetType   string
	EffectType   string
	EffectValue  int64
	EffectParams []byte
	ExpandTarget string
	ExpandSlots  uint32
}

type runtimeUseTargetPetRow struct {
	PetUID   uint64
	PetID    uint32
	Level    uint32
	Exp      uint64
	Quality  uint32
	HP       uint32
	HPMax    uint32
	ATK      uint32
	DEF      uint32
	SPD      uint32
	SkillIDs []uint32
	InLineup bool
}

type containerCapacityRow struct {
	Capacity    uint32
	MaxCapacity uint32
}

type itemChangeLogEntry struct {
	PlayerID      uint64
	ContainerType string
	SlotIndex     uint32
	ChangeType    string
	ItemID        uint64
	ItemUID       string
	BeforeQty     uint64
	ChangeQty     int64
	AfterQty      uint64
	ReasonType    string
	ReasonRefID   uint64
	OperatorType  string
	OperatorID    uint64
}

func loadTransferSourceRow(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string, slotIndex uint32) (*transferSourceRow, error) {
	var value transferSourceRow
	if err := tx.QueryRowContext(ctx, runtimeTransferSourceItemQuery, playerID, containerType, slotIndex).Scan(
		&value.RecordID,
		&value.SlotIndex,
		&value.ItemID,
		&value.ItemUID,
		&value.Quantity,
		&value.IsBound,
		&value.ExpireAt,
		&value.MaxStack,
		&value.CanStore,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &value, nil
}

func loadUseItemSourceRow(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string, slotIndex uint32) (*useItemSourceRow, error) {
	var value useItemSourceRow
	if err := tx.QueryRowContext(ctx, runtimeUseItemQuery, playerID, containerType, slotIndex).Scan(
		&value.RecordID,
		&value.SlotIndex,
		&value.ItemID,
		&value.ItemUID,
		&value.Quantity,
		&value.IsBound,
		&value.ExpireAt,
		&value.ItemType,
		&value.Usable,
		&value.TargetType,
		&value.EffectType,
		&value.EffectValue,
		&value.EffectParams,
		&value.ExpandTarget,
		&value.ExpandSlots,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &value, nil
}

func loadRuntimeUseTargetPetRow(ctx context.Context, tx *sql.Tx, playerID uint64, petUID uint64) (*runtimeUseTargetPetRow, error) {
	var (
		value        runtimeUseTargetPetRow
		skillIDsJSON []byte
	)
	if err := tx.QueryRowContext(ctx, runtimePetUseTargetQuery, playerID, petUID).Scan(
		&value.PetUID,
		&value.PetID,
		&value.Level,
		&value.Exp,
		&value.Quality,
		&value.HP,
		&value.HPMax,
		&value.ATK,
		&value.DEF,
		&value.SPD,
		&skillIDsJSON,
		&value.InLineup,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if len(skillIDsJSON) > 0 {
		if err := json.Unmarshal(skillIDsJSON, &value.SkillIDs); err != nil {
			return nil, fmt.Errorf("unmarshal runtime target pet skills: %w", err)
		}
	}
	return &value, nil
}

func loadTransferContainerCapacity(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string) (uint32, error) {
	var (
		capacity    uint32
		maxCapacity uint32
	)
	if err := tx.QueryRowContext(ctx, runtimeTransferContainerMetaQuery, playerID, containerType).Scan(&capacity, &maxCapacity); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return capacity, nil
}

func loadContainerCapacityDetail(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string) (*containerCapacityRow, error) {
	var value containerCapacityRow
	if err := tx.QueryRowContext(ctx, runtimeContainerCapacityDetailQuery, playerID, containerType).Scan(&value.Capacity, &value.MaxCapacity); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &value, nil
}

func loadTransferTargetRows(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string) ([]transferTargetRow, error) {
	rows, err := tx.QueryContext(ctx, runtimeTransferTargetRowsQuery, playerID, containerType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]transferTargetRow, 0)
	for rows.Next() {
		var value transferTargetRow
		if err := rows.Scan(&value.RecordID, &value.SlotIndex, &value.ItemID, &value.ItemUID, &value.Quantity, &value.IsBound); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func updateTransferItemQuantity(ctx context.Context, tx *sql.Tx, recordID int64, quantity uint64) error {
	_, err := tx.ExecContext(ctx, runtimeTransferUpdateQuantityQuery, recordID, quantity)
	return err
}

func updateTransferSlotIndex(ctx context.Context, tx *sql.Tx, recordID int64, slotIndex uint32) error {
	_, err := tx.ExecContext(ctx, runtimeUpdateSlotIndexQuery, recordID, slotIndex)
	return err
}

func loadSortRows(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string) ([]sortRow, error) {
	rows, err := tx.QueryContext(ctx, runtimeSortRowsQuery, playerID, containerType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]sortRow, 0)
	for rows.Next() {
		var value sortRow
		if err := rows.Scan(&value.RecordID, &value.SlotIndex, &value.ItemID, &value.ItemUID, &value.Quantity, &value.SortWeight, &value.Quality); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func loadGrantItemDefinition(ctx context.Context, tx *sql.Tx, itemID uint64) (*grantItemDefinitionRow, error) {
	var value grantItemDefinitionRow
	if err := tx.QueryRowContext(ctx, runtimeGrantItemDefinitionQuery, itemID).Scan(&value.ItemName, &value.MaxStack, &value.BindType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if value.MaxStack == 0 {
		value.MaxStack = 1
	}
	return &value, nil
}

func insertItemChangeLog(ctx context.Context, tx *sql.Tx, entry itemChangeLogEntry) error {
	_, err := tx.ExecContext(ctx, insertItemChangeLogQuery,
		entry.PlayerID,
		entry.ContainerType,
		entry.SlotIndex,
		entry.ChangeType,
		entry.ItemID,
		entry.ItemUID,
		entry.BeforeQty,
		entry.ChangeQty,
		entry.AfterQty,
		entry.ReasonType,
		entry.ReasonRefID,
		entry.OperatorType,
		entry.OperatorID,
	)
	return err
}

func insertContainerExpandLog(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string, beforeCapacity uint32, expandSlots uint32, afterCapacity uint32, itemID uint64, reasonType string) error {
	_, err := tx.ExecContext(ctx, insertContainerExpandLogQuery,
		playerID,
		containerType,
		beforeCapacity,
		expandSlots,
		afterCapacity,
		itemID,
		reasonType,
		0,
	)
	return err
}

func applyRuntimeItemEffect(ctx context.Context, tx *sql.Tx, playerID uint64, sourceRow *useItemSourceRow, quantity uint64, targetPetUID uint64, targetPlayerID uint64) (bag.RuntimeUseEffect, error) {
	switch normalizedEffectType := strings.TrimSpace(sourceRow.EffectType); normalizedEffectType {
	case "bag_expand", "warehouse_expand", "expand":
		return applyRuntimeExpandEffect(ctx, tx, playerID, sourceRow, quantity)
	case "pet_hp_restore":
		return applyRuntimePetHPRestoreEffect(ctx, tx, playerID, sourceRow, quantity, targetPetUID, targetPlayerID)
	case "reward_box", "gift_box", "box_open":
		return applyRuntimeRewardBoxEffect(sourceRow, quantity)
	default:
		return bag.RuntimeUseEffect{}, bag.ErrUnsupportedItemEffect
	}
}

type runtimeRewardBoxParams struct {
	Rewards []struct {
		Type     string `json:"type"`
		Value    uint64 `json:"value"`
		ItemID   uint64 `json:"item_id"`
		ItemName string `json:"item_name"`
		Count    uint64 `json:"count"`
		PetID    uint64 `json:"pet_id"`
	} `json:"rewards"`
}

func applyRuntimeRewardBoxEffect(sourceRow *useItemSourceRow, quantity uint64) (bag.RuntimeUseEffect, error) {
	if sourceRow == nil || quantity == 0 {
		return bag.RuntimeUseEffect{}, bag.ErrUnsupportedItemEffect
	}
	if len(sourceRow.EffectParams) == 0 {
		return bag.RuntimeUseEffect{}, bag.ErrUnsupportedItemEffect
	}
	var params runtimeRewardBoxParams
	if err := json.Unmarshal(sourceRow.EffectParams, &params); err != nil {
		return bag.RuntimeUseEffect{}, bag.ErrUnsupportedItemEffect
	}
	if len(params.Rewards) == 0 {
		return bag.RuntimeUseEffect{}, bag.ErrUnsupportedItemEffect
	}
	rewards := make([]bag.RuntimeRewardItem, 0, len(params.Rewards))
	for _, configuredReward := range params.Rewards {
		rewardType := strings.TrimSpace(configuredReward.Type)
		if rewardType == "" {
			continue
		}
		rewardItem := bag.RuntimeRewardItem{
			Type:     rewardType,
			Value:    configuredReward.Value,
			ItemID:   configuredReward.ItemID,
			ItemName: configuredReward.ItemName,
			Count:    configuredReward.Count,
			PetID:    configuredReward.PetID,
		}
		if rewardItem.Value > 0 {
			rewardItem.Value = rewardItem.Value * quantity
		}
		if rewardItem.Count > 0 {
			rewardItem.Count = rewardItem.Count * quantity
		}
		rewards = append(rewards, rewardItem)
	}
	if len(rewards) == 0 {
		return bag.RuntimeUseEffect{}, bag.ErrUnsupportedItemEffect
	}
	return bag.RuntimeUseEffect{
		EffectType: "reward_box",
		Rewards:    rewards,
	}, nil
}

func applyRuntimeExpandEffect(ctx context.Context, tx *sql.Tx, playerID uint64, sourceRow *useItemSourceRow, quantity uint64) (bag.RuntimeUseEffect, error) {
	normalizedExpandTarget, normalizedEffectType := normalizeRuntimeUseEffect(sourceRow.EffectType, sourceRow.ExpandTarget)
	if normalizedEffectType == "" || normalizedExpandTarget == "" || sourceRow.ExpandSlots == 0 {
		return bag.RuntimeUseEffect{}, bag.ErrUnsupportedItemEffect
	}

	containerDetail, err := loadContainerCapacityDetail(ctx, tx, playerID, normalizedExpandTarget)
	if err != nil {
		return bag.RuntimeUseEffect{}, err
	}
	if containerDetail == nil {
		return bag.RuntimeUseEffect{}, bag.ErrContainerNotFound
	}

	totalExpandSlots := sourceRow.ExpandSlots * uint32(quantity)
	nextCapacity := containerDetail.Capacity + totalExpandSlots
	if nextCapacity > containerDetail.MaxCapacity {
		return bag.RuntimeUseEffect{}, bag.ErrContainerCapacityLimit
	}
	if _, err := tx.ExecContext(ctx, runtimeUpdateContainerCapacityQuery, playerID, normalizedExpandTarget, nextCapacity); err != nil {
		return bag.RuntimeUseEffect{}, err
	}
	if err := insertContainerExpandLog(ctx, tx, playerID, normalizedExpandTarget, containerDetail.Capacity, totalExpandSlots, nextCapacity, sourceRow.ItemID, "item_use"); err != nil {
		return bag.RuntimeUseEffect{}, err
	}

	return bag.RuntimeUseEffect{
		EffectType:   normalizedEffectType,
		ExpandTarget: normalizedExpandTarget,
		ExpandSlots:  totalExpandSlots,
		NewCapacity:  nextCapacity,
	}, nil
}

func applyRuntimePetHPRestoreEffect(ctx context.Context, tx *sql.Tx, playerID uint64, sourceRow *useItemSourceRow, quantity uint64, targetPetUID uint64, targetPlayerID uint64) (bag.RuntimeUseEffect, error) {
	// 当前第一版仅支持玩家给自己拥有的宠物恢复生命，先明确拒绝其他跨角色目标。
	if targetPlayerID != 0 && targetPlayerID != playerID {
		return bag.RuntimeUseEffect{}, bag.ErrUseTargetNotFound
	}
	if targetPetUID == 0 {
		return bag.RuntimeUseEffect{}, bag.ErrUseTargetRequired
	}
	if sourceRow.EffectValue <= 0 {
		return bag.RuntimeUseEffect{}, bag.ErrUnsupportedItemEffect
	}

	targetPet, err := loadRuntimeUseTargetPetRow(ctx, tx, playerID, targetPetUID)
	if err != nil {
		return bag.RuntimeUseEffect{}, err
	}
	if targetPet == nil {
		return bag.RuntimeUseEffect{}, bag.ErrUseTargetNotFound
	}
	if targetPet.HP >= targetPet.HPMax {
		return bag.RuntimeUseEffect{}, bag.ErrItemUseNoEffect
	}

	restorePerUse := uint64(sourceRow.EffectValue)
	totalRestore := restorePerUse * quantity
	nextHP := uint64(targetPet.HP) + totalRestore
	if nextHP > uint64(targetPet.HPMax) {
		nextHP = uint64(targetPet.HPMax)
	}
	if _, err := tx.ExecContext(ctx, runtimeUpdatePetHPByUIDQuery, playerID, targetPetUID, nextHP); err != nil {
		return bag.RuntimeUseEffect{}, err
	}

	updatedPet := *targetPet
	updatedPet.HP = uint32(nextHP)
	return bag.RuntimeUseEffect{
		EffectType:   "pet_hp_restore",
		TargetPetUID: updatedPet.PetUID,
		RestoredHP:   updatedPet.HP - targetPet.HP,
		NewPetHP:     updatedPet.HP,
		UpdatedPet: &bag.RuntimePetSnapshot{
			PetUID:   updatedPet.PetUID,
			PetID:    updatedPet.PetID,
			Level:    updatedPet.Level,
			Exp:      updatedPet.Exp,
			Quality:  updatedPet.Quality,
			HP:       updatedPet.HP,
			HPMax:    updatedPet.HPMax,
			ATK:      updatedPet.ATK,
			DEF:      updatedPet.DEF,
			SPD:      updatedPet.SPD,
			SkillIDs: append([]uint32{}, updatedPet.SkillIDs...),
			InLineup: updatedPet.InLineup,
		},
	}, nil
}

func normalizeRuntimeUseEffect(effectType string, expandTarget string) (string, string) {
	normalizedEffectType := strings.TrimSpace(effectType)
	normalizedExpandTarget := strings.TrimSpace(expandTarget)
	switch normalizedEffectType {
	case "bag_expand":
		if normalizedExpandTarget == "" {
			normalizedExpandTarget = bag.ContainerTypeBag
		}
	case "warehouse_expand":
		if normalizedExpandTarget == "" {
			normalizedExpandTarget = bag.ContainerTypeWarehouse
		}
	}
	if normalizedExpandTarget != bag.ContainerTypeBag && normalizedExpandTarget != bag.ContainerTypeWarehouse {
		return "", ""
	}
	switch normalizedEffectType {
	case "bag_expand", "warehouse_expand":
		return normalizedExpandTarget, normalizedEffectType
	case "expand":
		return normalizedExpandTarget, normalizedEffectType
	default:
		return "", ""
	}
}

func insertMoveOutInLogs(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string, itemID uint64, itemUID string, fromSlotIndex uint32, toSlotIndex uint32, quantity uint64, reasonType string) error {
	if err := insertItemChangeLog(ctx, tx, itemChangeLogEntry{
		PlayerID:      playerID,
		ContainerType: containerType,
		SlotIndex:     fromSlotIndex,
		ChangeType:    "slot_move_out",
		ItemID:        itemID,
		ItemUID:       itemUID,
		BeforeQty:     quantity,
		ChangeQty:     -int64(quantity),
		AfterQty:      0,
		ReasonType:    reasonType,
		OperatorType:  "player",
		OperatorID:    playerID,
	}); err != nil {
		return err
	}
	return insertItemChangeLog(ctx, tx, itemChangeLogEntry{
		PlayerID:      playerID,
		ContainerType: containerType,
		SlotIndex:     toSlotIndex,
		ChangeType:    "slot_move_in",
		ItemID:        itemID,
		ItemUID:       itemUID,
		BeforeQty:     0,
		ChangeQty:     int64(quantity),
		AfterQty:      quantity,
		ReasonType:    reasonType,
		OperatorType:  "player",
		OperatorID:    playerID,
	})
}

func buildTransferReasonType(fromContainerType string, toContainerType string) string {
	if fromContainerType == bag.ContainerTypeBag && toContainerType == bag.ContainerTypeWarehouse {
		return "bag_to_warehouse"
	}
	if fromContainerType == bag.ContainerTypeWarehouse && toContainerType == bag.ContainerTypeBag {
		return "warehouse_to_bag"
	}
	return "container_transfer"
}

func firstEmptySlotIndex(rows []transferTargetRow, capacity uint32) uint32 {
	occupied := make(map[uint32]struct{}, len(rows))
	for _, row := range rows {
		occupied[row.SlotIndex] = struct{}{}
	}
	for slotIndex := uint32(1); slotIndex <= capacity; slotIndex++ {
		if _, exists := occupied[slotIndex]; !exists {
			return slotIndex
		}
	}
	return 0
}

func minUint64(left uint64, right uint64) uint64 {
	if left < right {
		return left
	}
	return right
}

const playerHasEverOwnedItemQuery = `
SELECT EXISTS (
  SELECT 1
  FROM player_unique_item_obtained
  WHERE player_id = $1 AND item_id = $2
)
OR EXISTS (
  SELECT 1
  FROM player_container_item
  WHERE player_id = $1 AND item_id = $2
)
`

const insertPlayerUniqueItemObtainedQuery = `
INSERT INTO player_unique_item_obtained (
  player_id,
  item_id,
  first_reason_type,
  first_reason_ref_id
) VALUES ($1, $2, $3, $4)
ON CONFLICT (player_id, item_id) DO NOTHING
`

// PlayerHasEverOwnedItem 判断玩家是否已获得过指定道具。
func (r *BagRepository) PlayerHasEverOwnedItem(ctx context.Context, playerID uint64, itemID uint64) (bool, error) {
	if playerID == 0 || itemID == 0 {
		return false, nil
	}
	var owned bool
	if err := r.db.QueryRowContext(ctx, playerHasEverOwnedItemQuery, playerID, itemID).Scan(&owned); err != nil {
		return false, err
	}
	return owned, nil
}

// RecordUniqueItemObtained 记录玩家首次获得唯一道具。
func (r *BagRepository) RecordUniqueItemObtained(ctx context.Context, playerID uint64, itemID uint64, reasonType string, reasonRefID uint64) error {
	if playerID == 0 || itemID == 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, insertPlayerUniqueItemObtainedQuery, playerID, itemID, reasonType, reasonRefID)
	return err
}
