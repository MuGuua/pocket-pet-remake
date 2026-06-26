package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/player"
)

const runtimeEquippedListQuery = `
SELECT
  pes.equip_slot,
  ei.item_uid,
  ei.item_id,
  ei.enhance_level,
  idf.item_name,
  COALESCE(idf.icon, ''),
  iee.appearance_skin_id,
  iee.appearance_only,
  iee.base_hp,
  iee.base_mana,
  iee.base_atk,
  iee.base_def,
  iee.base_spd,
  COALESCE(iee.base_stats_json, '{}'::jsonb),
  COALESCE(iee.enhance_per_level_stats_json, '{}'::jsonb)
FROM player_equipment_slot pes
JOIN equipment_instance ei ON ei.item_uid = pes.item_uid
JOIN item_definition idf ON idf.item_id = ei.item_id
JOIN item_equipment_extra iee ON iee.item_id = ei.item_id
WHERE pes.player_id = $1
ORDER BY pes.equip_slot ASC
`

const listEquippedEntriesForItemIDQuery = `
SELECT
  pes.player_id,
  pes.equip_slot,
  ei.item_uid
FROM player_equipment_slot pes
JOIN equipment_instance ei ON ei.item_uid = pes.item_uid
WHERE ei.item_id = $1
ORDER BY pes.player_id ASC, pes.equip_slot ASC
`

const findBagSlotIndexByItemUIDQuery = `
SELECT pci.slot_index
FROM player_container_item pci
WHERE pci.player_id = $1
  AND pci.container_type = $2
  AND pci.item_uid = $3
LIMIT 1
`

const runtimeBagEquipmentSourceQuery = `
SELECT
  pci.id,
  pci.slot_index,
  pci.item_id,
  COALESCE(pci.item_uid, ''),
  pci.quantity,
  pci.is_bound,
  idf.item_type,
  idf.required_level,
  idf.bind_type,
  idf.item_name,
  COALESCE(idf.icon, ''),
  iee.equip_slot,
  iee.appearance_skin_id,
  iee.appearance_only,
  iee.base_hp,
  iee.base_mana,
  iee.base_atk,
  iee.base_def,
  iee.base_spd,
  COALESCE(iee.base_stats_json, '{}'::jsonb),
  COALESCE(iee.enhance_per_level_stats_json, '{}'::jsonb),
  COALESCE(ei.enhance_level, 0)
FROM player_container_item pci
JOIN item_definition idf ON idf.item_id = pci.item_id
JOIN item_equipment_extra iee ON iee.item_id = pci.item_id
LEFT JOIN equipment_instance ei ON ei.item_uid = NULLIF(pci.item_uid, '')
WHERE pci.player_id = $1
  AND pci.container_type = $2
  AND pci.slot_index = $3
LIMIT 1
`

const runtimeEquippedSlotItemQuery = `
SELECT
  pes.equip_slot,
  ei.item_uid,
  ei.item_id,
  ei.enhance_level,
  idf.item_name,
  COALESCE(idf.icon, ''),
  iee.appearance_skin_id,
  iee.appearance_only,
  iee.base_hp,
  iee.base_mana,
  iee.base_atk,
  iee.base_def,
  iee.base_spd,
  COALESCE(iee.base_stats_json, '{}'::jsonb),
  COALESCE(iee.enhance_per_level_stats_json, '{}'::jsonb)
FROM player_equipment_slot pes
JOIN equipment_instance ei ON ei.item_uid = pes.item_uid
JOIN item_definition idf ON idf.item_id = ei.item_id
JOIN item_equipment_extra iee ON iee.item_id = ei.item_id
WHERE pes.player_id = $1
  AND pes.equip_slot = $2
LIMIT 1
`

const insertEquipmentInstanceQuery = `
INSERT INTO equipment_instance (
  item_uid, player_id, item_id, enhance_level, star_level, durability, max_durability, bind_type, state
) VALUES ($1, $2, $3, 0, 0, 0, 0, $4, $5)
`

const upsertPlayerEquipmentSlotQuery = `
INSERT INTO player_equipment_slot (player_id, equip_slot, item_uid)
VALUES ($1, $2, $3)
ON CONFLICT (player_id, equip_slot)
DO UPDATE SET item_uid = EXCLUDED.item_uid, equipped_at = CURRENT_TIMESTAMP
`

const deletePlayerEquipmentSlotQuery = `
DELETE FROM player_equipment_slot
WHERE player_id = $1 AND equip_slot = $2
`

const savePlayerEquipmentRecalcQuery = `
UPDATE player
SET hp_max = $2,
    atk = $3,
    def = $4,
    spd = $5,
    mana = $6,
    hit_pct = $7,
    dodge_pct = $8,
    spirit = LEAST($24::INTEGER, $9::INTEGER),
    spirit_max = $9::INTEGER,
    crit_rate_pct = $10,
    crit_dmg_pct = $11,
    physical_resist_pct = $12,
    skill_resist_pct = $13,
    confusion_resist_pct = $14,
    sleep_resist_pct = $15,
    paralysis_resist_pct = $16,
    seal_resist_pct = $17,
    curse_resist_pct = $18,
    crit_resist_pct = $19,
    crit_dmg_resist_pct = $20,
    character_resist_pct = $21,
    pet_resist_pct = $22,
    skin_id = CASE WHEN $23 <> '' THEN $23 ELSE skin_id END,
    hp = CASE WHEN $25 THEN $2 ELSE LEAST(hp, $2) END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND status = 1
`

type runtimeEquippedRow struct {
	EquipSlot                string
	ItemUID                  string
	ItemID                   uint64
	EnhanceLevel             uint32
	ItemName                 string
	Icon                     string
	AppearanceSkinID         string
	AppearanceOnly           bool
	BaseHP                   uint32
	BaseMana                 uint32
	BaseATK                  uint32
	BaseDEF                  uint32
	BaseSPD                  uint32
	BaseStatsJSON            []byte
	EnhancePerLevelStatsJSON []byte
}

type runtimeBagEquipmentRow struct {
	RecordID                 int64
	SlotIndex                uint32
	ItemID                   uint64
	ItemUID                  string
	Quantity                 uint64
	IsBound                  bool
	ItemType                 string
	RequiredLevel            uint32
	BindType                 string
	ItemName                 string
	Icon                     string
	EquipSlot                string
	AppearanceSkinID         string
	AppearanceOnly           bool
	BaseHP                   uint32
	BaseMana                 uint32
	BaseATK                  uint32
	BaseDEF                  uint32
	BaseSPD                  uint32
	BaseStatsJSON            []byte
	EnhancePerLevelStatsJSON []byte
	EnhanceLevel             uint32
}

// ListEquipped 返回玩家当前全身已佩戴装备。
func (r *EquipmentRepository) ListEquipped(ctx context.Context, playerID uint64) ([]equipment.RuntimeEquippedItem, error) {
	rows, err := r.db.QueryContext(ctx, runtimeEquippedListQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]equipment.RuntimeEquippedItem, 0, 13)
	for rows.Next() {
		row, err := scanRuntimeEquippedRow(rows)
		if err != nil {
			return nil, err
		}
		template, err := row.toPieceTemplate()
		if err != nil {
			return nil, err
		}
		items = append(items, equipment.ToRuntimeEquippedItem(template, row.ItemUID, row.ItemID, row.ItemName, row.Icon))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// EquipFromBagSlot 从背包佩戴装备并在同一事务内重算玩家战斗属性。
func (r *EquipmentRepository) EquipFromBagSlot(
	ctx context.Context,
	playerID uint64,
	containerType string,
	bagSlotIndex uint32,
	recalc equipment.RecalcContext,
	currentProfile *player.Profile,
) (*equipment.EquipFromBagResult, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	sourceRow, err := loadRuntimeBagEquipmentRow(ctx, tx, playerID, containerType, bagSlotIndex)
	if err != nil {
		return nil, err
	}
	if sourceRow == nil {
		return nil, bag.ErrContainerItemNotFound
	}
	if !strings.EqualFold(sourceRow.ItemType, "equipment") {
		return nil, equipment.ErrEquipmentBagItemInvalid
	}
	if currentProfile != nil && currentProfile.Level < sourceRow.RequiredLevel {
		return nil, equipment.ErrEquipmentLevelTooLow
	}
	if !equipment.IsValidEquipSlot(sourceRow.EquipSlot) {
		return nil, equipment.ErrEquipmentSlotMismatch
	}

	itemUID := strings.TrimSpace(sourceRow.ItemUID)
	if itemUID == "" {
		itemUID = generateEquipmentItemUID(playerID)
		bindType := strings.TrimSpace(sourceRow.BindType)
		if bindType == "" {
			bindType = "none"
		}
		if _, err := tx.ExecContext(ctx, insertEquipmentInstanceQuery, itemUID, playerID, sourceRow.ItemID, bindType, "equipped"); err != nil {
			return nil, err
		}
	} else {
		if _, err := tx.ExecContext(ctx, runtimeTransferUpdateEquipmentStateQuery, itemUID, "equipped"); err != nil {
			return nil, err
		}
	}

	var unequippedItem *equipment.RuntimeEquippedItem
	existingRow, err := loadRuntimeEquippedSlotRow(ctx, tx, playerID, sourceRow.EquipSlot)
	if err != nil {
		return nil, err
	}
	if existingRow != nil {
		if err := moveEquippedItemToBag(ctx, tx, playerID, containerType, existingRow); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, deletePlayerEquipmentSlotQuery, playerID, sourceRow.EquipSlot); err != nil {
			return nil, err
		}
		template, err := existingRow.toPieceTemplate()
		if err != nil {
			return nil, err
		}
		item := equipment.ToRuntimeEquippedItem(template, existingRow.ItemUID, existingRow.ItemID, existingRow.ItemName, existingRow.Icon)
		unequippedItem = &item
	}

	if err := consumeBagEquipmentSlot(ctx, tx, playerID, containerType, sourceRow); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, upsertPlayerEquipmentSlotQuery, playerID, sourceRow.EquipSlot, itemUID); err != nil {
		return nil, err
	}

	pieceTemplates, err := loadEquippedPieceTemplatesInTx(ctx, tx, playerID)
	if err != nil {
		return nil, err
	}
	recalcResult := equipment.BuildRecalcResult(recalc, pieceTemplates, currentProfile)
	if err := savePlayerEquipmentRecalcInTx(ctx, tx, playerID, recalcResult, false); err != nil {
		return nil, err
	}

	allEquipped, err := loadEquippedRuntimeItemsInTx(ctx, tx, playerID)
	if err != nil {
		return nil, err
	}
	equippedTemplate, err := sourceRow.toPieceTemplate()
	if err != nil {
		return nil, err
	}
	equippedTemplate.EnhanceLevel = sourceRow.EnhanceLevel

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &equipment.EquipFromBagResult{
		EquippedSlot: equipment.ToRuntimeEquippedItem(equippedTemplate, itemUID, sourceRow.ItemID, sourceRow.ItemName, sourceRow.Icon),
		Unequipped:   unequippedItem,
		AllEquipped:  allEquipped,
	}, nil
}

// UnequipSlot 卸下指定槽位装备并在同一事务内重算玩家战斗属性。
func (r *EquipmentRepository) UnequipSlot(
	ctx context.Context,
	playerID uint64,
	equipSlot string,
	containerType string,
	recalc equipment.RecalcContext,
	currentProfile *player.Profile,
) (*equipment.UnequipSlotResult, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	existingRow, err := loadRuntimeEquippedSlotRow(ctx, tx, playerID, equipSlot)
	if err != nil {
		return nil, err
	}
	if existingRow == nil {
		return nil, equipment.ErrEquipmentSlotEmpty
	}
	if err := moveEquippedItemToBag(ctx, tx, playerID, containerType, existingRow); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, deletePlayerEquipmentSlotQuery, playerID, equipSlot); err != nil {
		return nil, err
	}

	pieceTemplates, err := loadEquippedPieceTemplatesInTx(ctx, tx, playerID)
	if err != nil {
		return nil, err
	}
	recalcResult := equipment.BuildRecalcResult(recalc, pieceTemplates, currentProfile)
	if err := savePlayerEquipmentRecalcInTx(ctx, tx, playerID, recalcResult, false); err != nil {
		return nil, err
	}

	allEquipped, err := loadEquippedRuntimeItemsInTx(ctx, tx, playerID)
	if err != nil {
		return nil, err
	}
	template, err := existingRow.toPieceTemplate()
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &equipment.UnequipSlotResult{
		Unequipped:  equipment.ToRuntimeEquippedItem(template, existingRow.ItemUID, existingRow.ItemID, existingRow.ItemName, existingRow.Icon),
		AllEquipped: allEquipped,
	}, nil
}

// RecalcEquippedCombatStats 仅根据当前成长层与已佩戴装备重算战斗属性，不改动佩戴关系。
func (r *EquipmentRepository) RecalcEquippedCombatStats(
	ctx context.Context,
	playerID uint64,
	recalc equipment.RecalcContext,
	currentProfile *player.Profile,
	refillHP bool,
) error {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	pieceTemplates, err := loadEquippedPieceTemplatesInTx(ctx, tx, playerID)
	if err != nil {
		return err
	}
	recalcResult := equipment.BuildRecalcResult(recalc, pieceTemplates, currentProfile)
	if err := savePlayerEquipmentRecalcInTx(ctx, tx, playerID, recalcResult, refillHP); err != nil {
		return err
	}
	return tx.Commit()
}

// ListEquippedEntriesForItemID 返回当前正佩戴指定装备模板的所有玩家佩戴实例。
func (r *EquipmentRepository) ListEquippedEntriesForItemID(ctx context.Context, itemID uint64) ([]equipment.EquippedTemplateRefreshEntry, error) {
	if itemID == 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, listEquippedEntriesForItemIDQuery, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]equipment.EquippedTemplateRefreshEntry, 0, 8)
	for rows.Next() {
		var (
			playerID  int64
			equipSlot string
			itemUID   string
		)
		if err := rows.Scan(&playerID, &equipSlot, &itemUID); err != nil {
			return nil, err
		}
		if playerID <= 0 {
			continue
		}
		entries = append(entries, equipment.EquippedTemplateRefreshEntry{
			PlayerID:  uint64(playerID),
			EquipSlot: strings.TrimSpace(equipSlot),
			ItemUID:   strings.TrimSpace(itemUID),
		})
	}
	return entries, rows.Err()
}

// FindBagSlotIndexByItemUID 根据装备实例 UID 定位其在背包中的格子序号。
func (r *EquipmentRepository) FindBagSlotIndexByItemUID(ctx context.Context, playerID uint64, containerType string, itemUID string) (uint32, error) {
	itemUID = strings.TrimSpace(itemUID)
	if playerID == 0 || itemUID == "" {
		return 0, nil
	}
	normalizedContainer := strings.TrimSpace(containerType)
	if normalizedContainer == "" {
		normalizedContainer = bag.ContainerTypeBag
	}
	var slotIndex int64
	err := r.db.QueryRowContext(ctx, findBagSlotIndexByItemUIDQuery, playerID, normalizedContainer, itemUID).Scan(&slotIndex)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if slotIndex <= 0 {
		return 0, nil
	}
	return uint32(slotIndex), nil
}

// RefreshEquippedTemplateEntry 在同一事务内完成「必要时临时扩背包 -> 卸下 -> 再穿戴 -> 恢复容量」，
// 用于装备模板数值变更后重算玩家属性，避免背包满格导致卸装失败。
func (r *EquipmentRepository) RefreshEquippedTemplateEntry(
	ctx context.Context,
	playerID uint64,
	equipSlot string,
	itemUID string,
	containerType string,
	recalc equipment.RecalcContext,
	currentProfile *player.Profile,
) error {
	itemUID = strings.TrimSpace(itemUID)
	equipSlot = strings.TrimSpace(equipSlot)
	normalizedContainer := strings.TrimSpace(containerType)
	if normalizedContainer == "" {
		normalizedContainer = bag.ContainerTypeBag
	}
	if playerID == 0 || itemUID == "" || !equipment.IsValidEquipSlot(equipSlot) {
		return equipment.ErrEquipmentNotFound
	}

	beginner, ok := r.db.(txBeginner)
	if !ok {
		return fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	restoreCapacity, err := ensureTemporaryBagCapacityForUnequip(ctx, tx, playerID, normalizedContainer)
	if err != nil {
		return err
	}

	existingRow, err := loadRuntimeEquippedSlotRow(ctx, tx, playerID, equipSlot)
	if err != nil {
		return err
	}
	if existingRow == nil {
		return equipment.ErrEquipmentSlotEmpty
	}
	if strings.TrimSpace(existingRow.ItemUID) != itemUID {
		return equipment.ErrEquipmentNotFound
	}

	if err := moveEquippedItemToBag(ctx, tx, playerID, normalizedContainer, existingRow); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, deletePlayerEquipmentSlotQuery, playerID, equipSlot); err != nil {
		return err
	}

	bagSlotIndex, err := findBagSlotIndexInTx(ctx, tx, playerID, normalizedContainer, itemUID)
	if err != nil {
		return err
	}
	if bagSlotIndex == 0 {
		return equipment.ErrEquipmentBagItemInvalid
	}
	if err := equipFromBagSlotInTx(ctx, tx, playerID, normalizedContainer, bagSlotIndex, recalc, currentProfile); err != nil {
		return err
	}

	if restoreCapacity != nil {
		if err := restoreContainerCapacityInTx(ctx, tx, playerID, normalizedContainer, *restoreCapacity); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func loadRuntimeBagEquipmentRow(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string, slotIndex uint32) (*runtimeBagEquipmentRow, error) {
	var (
		row           runtimeBagEquipmentRow
		requiredLevel int64
		baseHP        int64
		baseMana      int64
		baseATK       int64
		baseDEF       int64
		baseSPD       int64
	)
	err := tx.QueryRowContext(ctx, runtimeBagEquipmentSourceQuery, playerID, containerType, slotIndex).Scan(
		&row.RecordID,
		&row.SlotIndex,
		&row.ItemID,
		&row.ItemUID,
		&row.Quantity,
		&row.IsBound,
		&row.ItemType,
		&requiredLevel,
		&row.BindType,
		&row.ItemName,
		&row.Icon,
		&row.EquipSlot,
		&row.AppearanceSkinID,
		&row.AppearanceOnly,
		&baseHP,
		&baseMana,
		&baseATK,
		&baseDEF,
		&baseSPD,
		&row.BaseStatsJSON,
		&row.EnhancePerLevelStatsJSON,
		&row.EnhanceLevel,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.RequiredLevel = uint32(requiredLevel)
	row.BaseHP = uint32(baseHP)
	row.BaseMana = uint32(baseMana)
	row.BaseATK = uint32(baseATK)
	row.BaseDEF = uint32(baseDEF)
	row.BaseSPD = uint32(baseSPD)
	return &row, nil
}

func loadRuntimeEquippedSlotRow(ctx context.Context, tx *sql.Tx, playerID uint64, equipSlot string) (*runtimeEquippedRow, error) {
	row := runtimeEquippedRow{}
	var enhanceLevel int64
	var baseHP, baseMana, baseATK, baseDEF, baseSPD int64
	err := tx.QueryRowContext(ctx, runtimeEquippedSlotItemQuery, playerID, equipSlot).Scan(
		&row.EquipSlot,
		&row.ItemUID,
		&row.ItemID,
		&enhanceLevel,
		&row.ItemName,
		&row.Icon,
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
	row.BaseHP = uint32(baseHP)
	row.BaseMana = uint32(baseMana)
	row.BaseATK = uint32(baseATK)
	row.BaseDEF = uint32(baseDEF)
	row.BaseSPD = uint32(baseSPD)
	return &row, nil
}

func scanRuntimeEquippedRow(scanner interface {
	Scan(dest ...any) error
}) (runtimeEquippedRow, error) {
	var row runtimeEquippedRow
	var enhanceLevel int64
	var baseHP, baseMana, baseATK, baseDEF, baseSPD int64
	err := scanner.Scan(
		&row.EquipSlot,
		&row.ItemUID,
		&row.ItemID,
		&enhanceLevel,
		&row.ItemName,
		&row.Icon,
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
	if err != nil {
		return runtimeEquippedRow{}, err
	}
	row.EnhanceLevel = uint32(enhanceLevel)
	row.BaseHP = uint32(baseHP)
	row.BaseMana = uint32(baseMana)
	row.BaseATK = uint32(baseATK)
	row.BaseDEF = uint32(baseDEF)
	row.BaseSPD = uint32(baseSPD)
	return row, nil
}

func (row *runtimeEquippedRow) toPieceTemplate() (equipment.EquippedPieceTemplate, error) {
	combatStats, err := unmarshalEquipmentCombatStatsJSON(row.BaseStatsJSON)
	if err != nil {
		return equipment.EquippedPieceTemplate{}, err
	}
	enhancePerLevel, err := unmarshalEnhancePerLevelJSON(row.EnhancePerLevelStatsJSON)
	if err != nil {
		return equipment.EquippedPieceTemplate{}, err
	}
	return equipment.EquippedPieceTemplate{
		EquipSlot:            equipment.EquipSlot(row.EquipSlot),
		AppearanceOnly:       row.AppearanceOnly,
		AppearanceSkinID:     row.AppearanceSkinID,
		BaseHP:               row.BaseHP,
		BaseMana:             row.BaseMana,
		BaseATK:              row.BaseATK,
		BaseDEF:              row.BaseDEF,
		BaseSPD:              row.BaseSPD,
		CombatStats:          combatStats,
		EnhancePerLevelStats: enhancePerLevel,
		EnhanceLevel:         row.EnhanceLevel,
	}, nil
}

func (row *runtimeBagEquipmentRow) toPieceTemplate() (equipment.EquippedPieceTemplate, error) {
	combatStats, err := unmarshalEquipmentCombatStatsJSON(row.BaseStatsJSON)
	if err != nil {
		return equipment.EquippedPieceTemplate{}, err
	}
	enhancePerLevel, err := unmarshalEnhancePerLevelJSON(row.EnhancePerLevelStatsJSON)
	if err != nil {
		return equipment.EquippedPieceTemplate{}, err
	}
	return equipment.EquippedPieceTemplate{
		EquipSlot:            equipment.EquipSlot(row.EquipSlot),
		AppearanceOnly:       row.AppearanceOnly,
		AppearanceSkinID:     row.AppearanceSkinID,
		BaseHP:               row.BaseHP,
		BaseMana:             row.BaseMana,
		BaseATK:              row.BaseATK,
		BaseDEF:              row.BaseDEF,
		BaseSPD:              row.BaseSPD,
		CombatStats:          combatStats,
		EnhancePerLevelStats: enhancePerLevel,
		EnhanceLevel:         row.EnhanceLevel,
	}, nil
}

func loadEquippedPieceTemplatesInTx(ctx context.Context, tx *sql.Tx, playerID uint64) ([]equipment.EquippedPieceTemplate, error) {
	rows, err := tx.QueryContext(ctx, runtimeEquippedListQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := make([]equipment.EquippedPieceTemplate, 0, 13)
	for rows.Next() {
		row, err := scanRuntimeEquippedRow(rows)
		if err != nil {
			return nil, err
		}
		template, err := row.toPieceTemplate()
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}
	return templates, rows.Err()
}

func loadEquippedRuntimeItemsInTx(ctx context.Context, tx *sql.Tx, playerID uint64) ([]equipment.RuntimeEquippedItem, error) {
	rows, err := tx.QueryContext(ctx, runtimeEquippedListQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]equipment.RuntimeEquippedItem, 0, 13)
	for rows.Next() {
		row, err := scanRuntimeEquippedRow(rows)
		if err != nil {
			return nil, err
		}
		template, err := row.toPieceTemplate()
		if err != nil {
			return nil, err
		}
		items = append(items, equipment.ToRuntimeEquippedItem(template, row.ItemUID, row.ItemID, row.ItemName, row.Icon))
	}
	return items, rows.Err()
}

func consumeBagEquipmentSlot(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string, sourceRow *runtimeBagEquipmentRow) error {
	if sourceRow == nil {
		return equipment.ErrEquipmentBagItemInvalid
	}
	if sourceRow.Quantity <= 1 {
		if _, err := tx.ExecContext(ctx, runtimeTransferDeleteItemQuery, sourceRow.RecordID); err != nil {
			return err
		}
		return insertItemChangeLog(ctx, tx, itemChangeLogEntry{
			PlayerID:      playerID,
			ContainerType: containerType,
			SlotIndex:     sourceRow.SlotIndex,
			ChangeType:    "equipment_equip_remove",
			ItemID:        sourceRow.ItemID,
			ItemUID:       sourceRow.ItemUID,
			BeforeQty:     sourceRow.Quantity,
			ChangeQty:     -1,
			AfterQty:      0,
			ReasonType:    "player_equipment_equip",
			OperatorType:  "player",
			OperatorID:    playerID,
		})
	}
	if err := updateTransferItemQuantity(ctx, tx, sourceRow.RecordID, sourceRow.Quantity-1); err != nil {
		return err
	}
	return insertItemChangeLog(ctx, tx, itemChangeLogEntry{
		PlayerID:      playerID,
		ContainerType: containerType,
		SlotIndex:     sourceRow.SlotIndex,
		ChangeType:    "equipment_equip_reduce",
		ItemID:        sourceRow.ItemID,
		ItemUID:       sourceRow.ItemUID,
		BeforeQty:     sourceRow.Quantity,
		ChangeQty:     -1,
		AfterQty:      sourceRow.Quantity - 1,
		ReasonType:    "player_equipment_equip",
		OperatorType:  "player",
		OperatorID:    playerID,
	})
}

func moveEquippedItemToBag(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string, row *runtimeEquippedRow) error {
	if row == nil {
		return equipment.ErrEquipmentNotFound
	}
	capacity, err := loadTransferContainerCapacity(ctx, tx, playerID, containerType)
	if err != nil {
		return err
	}
	if capacity == 0 {
		return bag.ErrContainerNotFound
	}
	targetRows, err := loadTransferTargetRows(ctx, tx, playerID, containerType)
	if err != nil {
		return err
	}
	slotIndex := firstEmptySlotIndex(targetRows, capacity)
	if slotIndex == 0 {
		return equipment.ErrEquipmentBagFull
	}
	if _, err := tx.ExecContext(ctx, runtimeTransferUpdateEquipmentStateQuery, row.ItemUID, "bag"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, runtimeTransferInsertItemQuery,
		playerID,
		containerType,
		slotIndex,
		row.ItemID,
		row.ItemUID,
		uint64(1),
		false,
		nil,
	); err != nil {
		return err
	}
	return insertItemChangeLog(ctx, tx, itemChangeLogEntry{
		PlayerID:      playerID,
		ContainerType: containerType,
		SlotIndex:     slotIndex,
		ChangeType:    "equipment_unequip_add",
		ItemID:        row.ItemID,
		ItemUID:       row.ItemUID,
		BeforeQty:     0,
		ChangeQty:     1,
		AfterQty:      1,
		ReasonType:    "player_equipment_unequip",
		OperatorType:  "player",
		OperatorID:    playerID,
	})
}

// ensureTemporaryBagCapacityForUnequip 在背包无空位时临时 +1 格，返回需恢复的原容量；有空位则返回 nil。
func ensureTemporaryBagCapacityForUnequip(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string) (*uint32, error) {
	capacity, err := loadTransferContainerCapacity(ctx, tx, playerID, containerType)
	if err != nil {
		return nil, err
	}
	if capacity == 0 {
		return nil, bag.ErrContainerNotFound
	}
	targetRows, err := loadTransferTargetRows(ctx, tx, playerID, containerType)
	if err != nil {
		return nil, err
	}
	if firstEmptySlotIndex(targetRows, capacity) > 0 {
		return nil, nil
	}
	original := capacity
	nextCapacity := capacity + 1
	if _, err := tx.ExecContext(ctx, runtimeUpdateContainerCapacityQuery, playerID, containerType, nextCapacity); err != nil {
		return nil, err
	}
	originalCopy := original
	return &originalCopy, nil
}

func restoreContainerCapacityInTx(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string, capacity uint32) error {
	_, err := tx.ExecContext(ctx, runtimeUpdateContainerCapacityQuery, playerID, containerType, capacity)
	return err
}

func findBagSlotIndexInTx(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string, itemUID string) (uint32, error) {
	itemUID = strings.TrimSpace(itemUID)
	if playerID == 0 || itemUID == "" {
		return 0, nil
	}
	normalizedContainer := strings.TrimSpace(containerType)
	if normalizedContainer == "" {
		normalizedContainer = bag.ContainerTypeBag
	}
	var slotIndex int64
	err := tx.QueryRowContext(ctx, findBagSlotIndexByItemUIDQuery, playerID, normalizedContainer, itemUID).Scan(&slotIndex)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if slotIndex <= 0 {
		return 0, nil
	}
	return uint32(slotIndex), nil
}

// equipFromBagSlotInTx 在已有事务内从背包格佩戴装备并重算战斗属性（不含事务提交）。
func equipFromBagSlotInTx(
	ctx context.Context,
	tx *sql.Tx,
	playerID uint64,
	containerType string,
	bagSlotIndex uint32,
	recalc equipment.RecalcContext,
	currentProfile *player.Profile,
) error {
	sourceRow, err := loadRuntimeBagEquipmentRow(ctx, tx, playerID, containerType, bagSlotIndex)
	if err != nil {
		return err
	}
	if sourceRow == nil {
		return bag.ErrContainerItemNotFound
	}
	if !strings.EqualFold(sourceRow.ItemType, "equipment") {
		return equipment.ErrEquipmentBagItemInvalid
	}
	if currentProfile != nil && currentProfile.Level < sourceRow.RequiredLevel {
		return equipment.ErrEquipmentLevelTooLow
	}
	if !equipment.IsValidEquipSlot(sourceRow.EquipSlot) {
		return equipment.ErrEquipmentSlotMismatch
	}

	itemUID := strings.TrimSpace(sourceRow.ItemUID)
	if itemUID == "" {
		itemUID = generateEquipmentItemUID(playerID)
		bindType := strings.TrimSpace(sourceRow.BindType)
		if bindType == "" {
			bindType = "none"
		}
		if _, err := tx.ExecContext(ctx, insertEquipmentInstanceQuery, itemUID, playerID, sourceRow.ItemID, bindType, "equipped"); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx, runtimeTransferUpdateEquipmentStateQuery, itemUID, "equipped"); err != nil {
			return err
		}
	}

	existingRow, err := loadRuntimeEquippedSlotRow(ctx, tx, playerID, sourceRow.EquipSlot)
	if err != nil {
		return err
	}
	if existingRow != nil {
		if err := moveEquippedItemToBag(ctx, tx, playerID, containerType, existingRow); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, deletePlayerEquipmentSlotQuery, playerID, sourceRow.EquipSlot); err != nil {
			return err
		}
	}

	if err := consumeBagEquipmentSlot(ctx, tx, playerID, containerType, sourceRow); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, upsertPlayerEquipmentSlotQuery, playerID, sourceRow.EquipSlot, itemUID); err != nil {
		return err
	}

	pieceTemplates, err := loadEquippedPieceTemplatesInTx(ctx, tx, playerID)
	if err != nil {
		return err
	}
	recalcResult := equipment.BuildRecalcResult(recalc, pieceTemplates, currentProfile)
	return savePlayerEquipmentRecalcInTx(ctx, tx, playerID, recalcResult, false)
}

func savePlayerEquipmentRecalcInTx(ctx context.Context, tx *sql.Tx, playerID uint64, result equipment.RecalcResult, refillHP bool) error {
	execResult, err := tx.ExecContext(
		ctx,
		savePlayerEquipmentRecalcQuery,
		playerID,
		result.HPMax,
		result.ATK,
		result.DEF,
		result.SPD,
		result.MANA,
		result.HitPct,
		result.DodgePct,
		result.SpiritMax,
		result.CritRatePct,
		result.CritDmgPct,
		result.PhysicalResistPct,
		result.SkillResistPct,
		result.ConfusionResistPct,
		result.SleepResistPct,
		result.ParalysisResistPct,
		result.SealResistPct,
		result.CurseResistPct,
		result.CritResistPct,
		result.CritDmgResistPct,
		result.CharacterResistPct,
		result.PetResistPct,
		result.SkinID,
		result.Spirit,
		refillHP,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := execResult.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return player.ErrPlayerNotFound
	}
	return nil
}

func generateEquipmentItemUID(playerID uint64) string {
	return fmt.Sprintf("eq-%d-%d", playerID, time.Now().UnixNano())
}

func unmarshalEquipmentCombatStatsJSON(raw []byte) (equipment.AdminCombatStats, error) {
	var payload map[string]uint32
	if len(raw) == 0 {
		return equipment.AdminCombatStats{}, nil
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return equipment.AdminCombatStats{}, err
	}
	return equipment.AdminCombatStats{
		Spirit:                   payload["spirit"],
		SpiritMax:                payload["spirit_max"],
		HitPct:                   payload["hit_pct"],
		DodgePct:                 payload["dodge_pct"],
		CritRatePct:              payload["crit_rate_pct"],
		CritDmgPct:               payload["crit_dmg_pct"],
		PhysicalResistPct:        payload["physical_resist_pct"],
		ReversePhysicalResistPct: payload["reverse_physical_resist_pct"],
		SkillResistPct:           payload["skill_resist_pct"],
		ReverseSkillResistPct:    payload["reverse_skill_resist_pct"],
		ConfusionResistPct:       payload["confusion_resist_pct"],
		SleepResistPct:           payload["sleep_resist_pct"],
		ParalysisResistPct:       payload["paralysis_resist_pct"],
		SealResistPct:            payload["seal_resist_pct"],
		CurseResistPct:           payload["curse_resist_pct"],
		CritDmgResistPct:         payload["crit_dmg_resist_pct"],
		CritResistPct:            payload["crit_resist_pct"],
		CharacterResistPct:       payload["character_resist_pct"],
		PetResistPct:             payload["pet_resist_pct"],
	}, nil
}

func unmarshalEnhancePerLevelJSON(raw []byte) (map[string]uint32, error) {
	if len(raw) == 0 {
		return map[string]uint32{}, nil
	}
	payload := map[string]uint32{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}
