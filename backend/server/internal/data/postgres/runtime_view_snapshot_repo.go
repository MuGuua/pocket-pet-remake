package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/playerskill"
)

const deletePlayerEquipmentSnapshotQuery = `
DELETE FROM player_equipment_snapshot
WHERE player_id = $1
`

const upsertPlayerEquipmentSnapshotQuery = `
INSERT INTO player_equipment_snapshot (
  player_id,
  equip_slot,
  item_uid,
  item_id,
  item_name,
  icon,
  required_level,
  enhance_level,
  is_damaged,
  appearance_skin_id,
  appearance_only,
  description,
  bonus_json,
  weapon_skills_json,
  weapon_type,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
  $11, $12, $13, $14, $15, CURRENT_TIMESTAMP
)
ON CONFLICT (player_id, equip_slot) DO UPDATE SET
  item_uid = EXCLUDED.item_uid,
  item_id = EXCLUDED.item_id,
  item_name = EXCLUDED.item_name,
  icon = EXCLUDED.icon,
  required_level = EXCLUDED.required_level,
  enhance_level = EXCLUDED.enhance_level,
  is_damaged = EXCLUDED.is_damaged,
  appearance_skin_id = EXCLUDED.appearance_skin_id,
  appearance_only = EXCLUDED.appearance_only,
  description = EXCLUDED.description,
  bonus_json = EXCLUDED.bonus_json,
  weapon_skills_json = EXCLUDED.weapon_skills_json,
  weapon_type = EXCLUDED.weapon_type,
  updated_at = CURRENT_TIMESTAMP
`

const listPlayerEquipmentSnapshotQuery = `
SELECT
  equip_slot,
  item_uid,
  item_id,
  item_name,
  icon,
  required_level,
  enhance_level,
  is_damaged,
  appearance_skin_id,
  appearance_only,
  description,
  bonus_json,
  weapon_skills_json,
  weapon_type
FROM player_equipment_snapshot
WHERE player_id = $1
ORDER BY equip_slot ASC
`

const deletePlayerSkillProgressSnapshotQuery = `
DELETE FROM player_skill_progress_snapshot
WHERE player_id = $1
`

const upsertPlayerSkillProgressSnapshotQuery = `
INSERT INTO player_skill_progress_snapshot (
  player_id,
  skill_id,
  skill_exp,
  skill_level,
  is_learned,
  learned_at,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP
)
ON CONFLICT (player_id, skill_id) DO UPDATE SET
  skill_exp = EXCLUDED.skill_exp,
  skill_level = EXCLUDED.skill_level,
  is_learned = EXCLUDED.is_learned,
  learned_at = EXCLUDED.learned_at,
  updated_at = CURRENT_TIMESTAMP
`

const listPlayerSkillProgressSnapshotQuery = `
SELECT player_id, skill_id, skill_exp, skill_level, is_learned, learned_at
FROM player_skill_progress_snapshot
WHERE player_id = $1
ORDER BY skill_id ASC
`

// RefreshPlayerEquipmentSnapshot 把当前已佩戴装备写入独立视图快照表。
func (r *EquipmentRepository) RefreshPlayerEquipmentSnapshot(ctx context.Context, playerID uint64) error {
	items, err := r.listEquippedRaw(ctx, playerID)
	if err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, deletePlayerEquipmentSnapshotQuery, playerID); err != nil {
		return err
	}
	for _, item := range items {
		bonusJSON, err := json.Marshal(item.Bonus)
		if err != nil {
			return fmt.Errorf("marshal equipment snapshot bonus: %w", err)
		}
		weaponSkillsJSON, err := json.Marshal(item.WeaponSkills)
		if err != nil {
			return fmt.Errorf("marshal equipment snapshot weapon skills: %w", err)
		}
		if _, err := r.db.ExecContext(
			ctx,
			upsertPlayerEquipmentSnapshotQuery,
			playerID,
			item.EquipSlot,
			item.ItemUID,
			item.ItemID,
			item.ItemName,
			item.Icon,
			item.RequiredLevel,
			item.EnhanceLevel,
			item.IsDamaged,
			item.AppearanceSkinID,
			item.AppearanceOnly,
			item.Description,
			bonusJSON,
			weaponSkillsJSON,
			item.WeaponType,
		); err != nil {
			return err
		}
	}
	return nil
}

// ListEquipped 从装备视图快照读取玩家当前佩戴装备；若快照缺失会先刷新。
func (r *EquipmentRepository) ListEquipped(ctx context.Context, playerID uint64) ([]equipment.RuntimeEquippedItem, error) {
	if playerID == 0 {
		return []equipment.RuntimeEquippedItem{}, nil
	}
	if err := r.RefreshPlayerEquipmentSnapshot(ctx, playerID); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, listPlayerEquipmentSnapshotQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]equipment.RuntimeEquippedItem, 0, 13)
	for rows.Next() {
		item, err := scanPlayerEquipmentSnapshotRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for index := range items {
		mentions, err := buildRuntimeDescriptionMentions(ctx, r.db, items[index].Description)
		if err != nil {
			return nil, err
		}
		items[index].DescriptionMentions = mentions
	}
	if items == nil {
		return []equipment.RuntimeEquippedItem{}, nil
	}
	return items, nil
}

func (r *EquipmentRepository) listEquippedRaw(ctx context.Context, playerID uint64) ([]equipment.RuntimeEquippedItem, error) {
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
		items = append(items, equipment.ToRuntimeEquippedItem(
			template,
			row.ItemUID,
			row.ItemID,
			row.ItemName,
			row.Icon,
			row.Description,
		))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for index := range items {
		mentions, err := buildRuntimeDescriptionMentions(ctx, r.db, items[index].Description)
		if err != nil {
			return nil, err
		}
		items[index].DescriptionMentions = mentions
	}
	return items, nil
}

// RefreshPlayerSkillProgressSnapshot 把人物技能学习进度同步到快照表。
func (r *PlayerSkillProgressRepository) RefreshPlayerSkillProgressSnapshot(ctx context.Context, playerID uint64) error {
	if playerID == 0 {
		return nil
	}
	rows, err := r.db.QueryContext(ctx, listPlayerSkillProgressQuery, playerID)
	if err != nil {
		return err
	}
	defer rows.Close()

	items := make([]playerskill.Progress, 0, 16)
	for rows.Next() {
		item, scanErr := scanPlayerSkillProgressRow(rows)
		if scanErr != nil {
			return scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, deletePlayerSkillProgressSnapshotQuery, playerID); err != nil {
		return err
	}
	for _, item := range items {
		if _, err := r.db.ExecContext(
			ctx,
			upsertPlayerSkillProgressSnapshotQuery,
			item.PlayerID,
			item.SkillID,
			item.SkillExp,
			item.SkillLevel,
			item.IsLearned,
			item.LearnedAt,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *PlayerSkillProgressRepository) ListByPlayerID(ctx context.Context, playerID uint64) ([]playerskill.Progress, error) {
	if playerID == 0 {
		return []playerskill.Progress{}, nil
	}
	if err := r.RefreshPlayerSkillProgressSnapshot(ctx, playerID); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, listPlayerSkillProgressSnapshotQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]playerskill.Progress, 0, 8)
	for rows.Next() {
		item, scanErr := scanPlayerSkillProgressRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if items == nil {
		return []playerskill.Progress{}, nil
	}
	return items, rows.Err()
}

func scanPlayerEquipmentSnapshotRow(scanner interface{ Scan(dest ...any) error }) (equipment.RuntimeEquippedItem, error) {
	var (
		item             equipment.RuntimeEquippedItem
		itemID           int64
		requiredLevel    int64
		enhanceLevel     int64
		bonusJSON        []byte
		weaponSkillsJSON []byte
	)
	if err := scanner.Scan(
		&item.EquipSlot,
		&item.ItemUID,
		&itemID,
		&item.ItemName,
		&item.Icon,
		&requiredLevel,
		&enhanceLevel,
		&item.IsDamaged,
		&item.AppearanceSkinID,
		&item.AppearanceOnly,
		&item.Description,
		&bonusJSON,
		&weaponSkillsJSON,
		&item.WeaponType,
	); err != nil {
		return equipment.RuntimeEquippedItem{}, err
	}
	item.ItemID = uint64(itemID)
	item.RequiredLevel = uint32(requiredLevel)
	item.EnhanceLevel = uint32(enhanceLevel)
	item.EquipSlotLabel = equipment.EquipSlotLabel(equipment.EquipSlot(item.EquipSlot))
	if len(bonusJSON) > 0 {
		if err := json.Unmarshal(bonusJSON, &item.Bonus); err != nil {
			return equipment.RuntimeEquippedItem{}, fmt.Errorf("unmarshal equipment snapshot bonus: %w", err)
		}
	}
	if len(weaponSkillsJSON) > 0 {
		if err := json.Unmarshal(weaponSkillsJSON, &item.WeaponSkills); err != nil {
			return equipment.RuntimeEquippedItem{}, fmt.Errorf("unmarshal equipment snapshot weapon skills: %w", err)
		}
	}
	return item, nil
}

func loadPlayerSkillProgressRawByPlayerID(ctx context.Context, db DBTX, playerID uint64) ([]playerskill.Progress, error) {
	if playerID == 0 {
		return []playerskill.Progress{}, nil
	}
	rows, err := db.QueryContext(ctx, listPlayerSkillProgressQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]playerskill.Progress, 0, 8)
	for rows.Next() {
		item, scanErr := scanPlayerSkillProgressRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func loadPlayerSkillProgressSnapshotByPlayerID(ctx context.Context, db DBTX, playerID uint64) ([]playerskill.Progress, error) {
	if playerID == 0 {
		return []playerskill.Progress{}, nil
	}
	rows, err := db.QueryContext(ctx, listPlayerSkillProgressSnapshotQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]playerskill.Progress, 0, 8)
	for rows.Next() {
		item, scanErr := scanPlayerSkillProgressRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func maybeNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
