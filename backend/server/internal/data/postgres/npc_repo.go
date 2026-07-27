package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"pocket-pet-remake/server/internal/module/npc"
)

type NPCRepository struct {
	db DBTX
}

func NewNPCRepository(db DBTX) *NPCRepository {
	return &NPCRepository{db: db}
}

const listNPCMenuEntriesByEntityIDQuery = `
SELECT
  entity_id,
  entry_id,
  entry_type,
  title,
  subtitle,
  state,
  priority,
  action_result_type,
  action_notice,
  battle_encounter_entity_id,
  conditions_json,
  linked_quest_id
FROM npc_menu_entry
WHERE entity_id = $1 AND status = 1
ORDER BY priority DESC, sort_order ASC, entry_id ASC
`

const listNPCMenuEntriesByEntityIDsQuery = `
SELECT
  entity_id,
  entry_id,
  entry_type,
  title,
  subtitle,
  state,
  priority,
  action_result_type,
  action_notice,
  battle_encounter_entity_id,
  conditions_json,
  linked_quest_id
FROM npc_menu_entry
WHERE entity_id = ANY($1) AND status = 1
ORDER BY entity_id ASC, priority DESC, sort_order ASC, entry_id ASC
`

const findNPCActionResultQuery = `
SELECT
  entity_id,
  entry_id,
  action_result_type,
  action_notice,
  battle_encounter_entity_id,
  linked_quest_id
FROM npc_menu_entry
WHERE entity_id = $1 AND entry_id = $2 AND status = 1
LIMIT 1
`

func (r *NPCRepository) ListMenuEntriesByEntityID(ctx context.Context, entityID uint64) ([]npc.MenuEntry, error) {
	rows, err := r.db.QueryContext(ctx, listNPCMenuEntriesByEntityIDQuery, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []npc.MenuEntry{}
	for rows.Next() {
		var value npc.MenuEntry
		var conditionsJSON []byte
		if err := rows.Scan(
			&value.EntityID,
			&value.EntryID,
			&value.EntryType,
			&value.Title,
			&value.Subtitle,
			&value.State,
			&value.Priority,
			&value.ActionResultType,
			&value.ActionNotice,
			&value.BattleEncounterEntityID,
			&conditionsJSON,
			&value.LinkedQuestID,
		); err != nil {
			return nil, err
		}
		value.ConditionsJSON = json.RawMessage(conditionsJSON)
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// ListMenuEntriesByEntityIDs 用单条 SQL 返回整张地图 NPC 的菜单配置，并按实体 ID 分组。
func (r *NPCRepository) ListMenuEntriesByEntityIDs(ctx context.Context, entityIDs []uint64) (map[uint64][]npc.MenuEntry, error) {
	result := make(map[uint64][]npc.MenuEntry, len(entityIDs))
	if len(entityIDs) == 0 {
		return result, nil
	}
	databaseIDs := make([]int64, 0, len(entityIDs))
	for _, entityID := range entityIDs {
		databaseIDs = append(databaseIDs, int64(entityID))
	}
	rows, err := r.db.QueryContext(ctx, listNPCMenuEntriesByEntityIDsQuery, databaseIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var value npc.MenuEntry
		var conditionsJSON []byte
		if err := rows.Scan(
			&value.EntityID,
			&value.EntryID,
			&value.EntryType,
			&value.Title,
			&value.Subtitle,
			&value.State,
			&value.Priority,
			&value.ActionResultType,
			&value.ActionNotice,
			&value.BattleEncounterEntityID,
			&conditionsJSON,
			&value.LinkedQuestID,
		); err != nil {
			return nil, err
		}
		value.ConditionsJSON = json.RawMessage(conditionsJSON)
		result[value.EntityID] = append(result[value.EntityID], value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *NPCRepository) FindActionResult(ctx context.Context, entityID uint64, entryID string) (*npc.ActionResult, error) {
	var value npc.ActionResult
	err := r.db.QueryRowContext(ctx, findNPCActionResultQuery, entityID, entryID).Scan(
		&value.EntityID,
		&value.EntryID,
		&value.ResultType,
		&value.Notice,
		&value.BattleEncounterEntityID,
		&value.LinkedQuestID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &value, nil
}

const listNPCShopGoodsQuery = `
SELECT
  g.item_id,
  i.item_name,
  i.buy_price_copper,
  g.sort_order
FROM npc_shop_goods g
JOIN item_definition i ON i.item_id = g.item_id
WHERE g.entity_id = $1
  AND g.status = 1
  AND i.is_enabled = true
  AND i.buy_price_copper > 0
  AND LOWER(i.price_type) = 'base_coin'
ORDER BY g.sort_order ASC, g.item_id ASC
`

const shopGoodExistsQuery = `
SELECT 1
FROM npc_shop_goods g
JOIN item_definition i ON i.item_id = g.item_id
WHERE g.entity_id = $1
  AND g.item_id = $2
  AND g.status = 1
  AND i.is_enabled = true
  AND i.buy_price_copper > 0
LIMIT 1
`

// ListShopGoodsByEntityID 读取某个商店 NPC 当前可售商品，并把展示价与名称一并返回给客户端。
func (r *NPCRepository) ListShopGoodsByEntityID(ctx context.Context, entityID uint64) ([]npc.ShopGood, error) {
	rows, err := r.db.QueryContext(ctx, listNPCShopGoodsQuery, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]npc.ShopGood, 0)
	for rows.Next() {
		var item npc.ShopGood
		if err := rows.Scan(&item.ItemID, &item.ItemName, &item.BuyPriceCopper, &item.SortOrder); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// ShopGoodExists 判断指定商品是否属于该商店 NPC 的可售清单。
func (r *NPCRepository) ShopGoodExists(ctx context.Context, entityID uint64, itemID uint64) (bool, error) {
	var marker int
	err := r.db.QueryRowContext(ctx, shopGoodExistsQuery, entityID, itemID).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return marker == 1, nil
}
