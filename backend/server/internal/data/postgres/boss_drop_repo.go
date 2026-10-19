package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"pocket-pet-remake/server/internal/module/monster"
)

const listBossDropsQuery = `
SELECT pool.id, pool.scene_id, pool.item_id, pool.quantity, pool.weight,
       pool.draw_group, pool.grant_once, pool.sort_order, pool.status,
       COALESCE(item.display_name, '')
FROM monster_boss_drop_pool AS pool
LEFT JOIN item_definition AS item ON item.item_id = pool.item_id
WHERE pool.status = 1
ORDER BY pool.scene_id ASC, pool.draw_group ASC, pool.sort_order ASC, pool.id ASC
`

const listBossDropsByEncounterQuery = `
SELECT pool.id, pool.scene_id, pool.item_id, pool.quantity, pool.weight,
       pool.draw_group, pool.grant_once, pool.sort_order, pool.status,
       COALESCE(item.display_name, '')
FROM monster_boss_drop_pool AS pool
LEFT JOIN item_definition AS item ON item.item_id = pool.item_id
WHERE pool.scene_id = $1
ORDER BY pool.draw_group ASC, pool.sort_order ASC, pool.id ASC
`

const deleteBossDropsByEncounterQuery = `
DELETE FROM monster_boss_drop_pool WHERE scene_id = $1
`

const insertBossDropQuery = `
INSERT INTO monster_boss_drop_pool (
  scene_id, item_id, quantity, weight, draw_group, grant_once, sort_order, status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id
`

// ListBossDrops 读取全部启用中的 Boss 掉落池配置，供启动时装配只读缓存。
func (r *MonsterRepository) ListBossDrops(ctx context.Context) ([]monster.BossDropEntry, error) {
	rows, err := r.db.QueryContext(ctx, listBossDropsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBossDropRows(rows)
}

// ListBossDropsByScene 读取指定遭遇战的全部掉落配置，供后台编辑使用。
func (r *MonsterRepository) ListBossDropsByScene(ctx context.Context, sceneID uint32) ([]monster.BossDropEntry, error) {
	rows, err := r.db.QueryContext(ctx, listBossDropsByEncounterQuery, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBossDropRows(rows)
}

// ReplaceBossDropsForScene 覆盖保存某遭遇战的 Boss 掉落池。
// 先删后插放在同一事务内，避免保存中途失败留下半套配置。
func (r *MonsterRepository) ReplaceBossDropsForScene(ctx context.Context, sceneID uint32, drops []monster.AdminBossDropInput) ([]monster.BossDropEntry, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("monster repository transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err := tx.ExecContext(ctx, deleteBossDropsByEncounterQuery, sceneID); err != nil {
		return nil, err
	}
	for _, entry := range monster.SanitizeBossDropInputs(drops) {
		if _, err := tx.ExecContext(
			ctx,
			insertBossDropQuery,
			sceneID,
			entry.ItemID,
			entry.Quantity,
			entry.Weight,
			entry.DrawGroup,
			entry.GrantOnce,
			entry.SortOrder,
			entry.Status,
		); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.ListBossDropsByScene(ctx, sceneID)
}

func scanBossDropRows(rows *sql.Rows) ([]monster.BossDropEntry, error) {
	result := make([]monster.BossDropEntry, 0)
	for rows.Next() {
		var entry monster.BossDropEntry
		if err := rows.Scan(
			&entry.ID,
			&entry.SceneID,
			&entry.ItemID,
			&entry.Quantity,
			&entry.Weight,
			&entry.DrawGroup,
			&entry.GrantOnce,
			&entry.SortOrder,
			&entry.Status,
			&entry.ItemName,
		); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}
