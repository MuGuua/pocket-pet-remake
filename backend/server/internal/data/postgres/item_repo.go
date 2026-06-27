package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"pocket-pet-remake/server/internal/module/item"
)

// ItemRepository 把后台物品模板管理映射到 item_definition 主表。
// 当前先覆盖统一模板主表 CRUD，等装备扩展表进入后台后再继续补子表读写。
type ItemRepository struct {
	db DBTX
}

// NewItemRepository 构造物品模板仓储。
func NewItemRepository(db DBTX) *ItemRepository {
	return &ItemRepository{db: db}
}

const adminItemListBaseQuery = `
SELECT
  item_id,
  item_code,
  item_name,
  item_type,
  item_sub_type,
  quality,
  icon,
  "desc",
  max_stack,
  buy_price_copper,
  sell_price_copper,
  usable,
  can_sell,
  can_store,
  is_enabled,
  updated_at,
  created_at
FROM item_definition
`

const adminItemDetailQuery = `
SELECT
  item_id,
  item_code,
  item_name,
  item_type,
  item_sub_type,
  quality,
  rarity,
  icon,
  "desc",
  max_stack,
  occupy_slots,
  auto_merge,
  sort_weight,
  usable,
  use_scope,
  target_type,
  required_level,
  required_scene_id,
  bind_type,
  can_sell,
  can_drop,
  can_store,
  can_trade,
  expire_at_rule,
  effect_type,
  effect_value,
  effect_params_json,
  buy_price_copper,
  sell_price_copper,
  recycle_price_copper,
  price_type,
  is_enabled,
  created_at,
  updated_at
FROM item_definition
WHERE item_id = $1
LIMIT 1
`

const insertAdminItemQuery = `
INSERT INTO item_definition (
  item_id,
  item_code,
  item_name,
  item_type,
  item_sub_type,
  quality,
  rarity,
  icon,
  "desc",
  max_stack,
  occupy_slots,
  auto_merge,
  sort_weight,
  usable,
  use_scope,
  target_type,
  required_level,
  required_scene_id,
  bind_type,
  can_sell,
  can_drop,
  can_store,
  can_trade,
  expire_at_rule,
  effect_type,
  effect_value,
  effect_params_json,
  buy_price_copper,
  sell_price_copper,
  recycle_price_copper,
  price_type,
  is_enabled
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27::jsonb,$28,$29,$30,$31,$32
)
`

const updateAdminItemQuery = `
UPDATE item_definition
SET item_code = $2,
    item_name = $3,
    item_type = $4,
    item_sub_type = $5,
    quality = $6,
    rarity = $7,
    icon = $8,
    "desc" = $9,
    max_stack = $10,
    occupy_slots = $11,
    auto_merge = $12,
    sort_weight = $13,
    usable = $14,
    use_scope = $15,
    target_type = $16,
    required_level = $17,
    required_scene_id = $18,
    bind_type = $19,
    can_sell = $20,
    can_drop = $21,
    can_store = $22,
    can_trade = $23,
    expire_at_rule = $24,
    effect_type = $25,
    effect_value = $26,
    effect_params_json = $27::jsonb,
    buy_price_copper = $28,
    sell_price_copper = $29,
    recycle_price_copper = $30,
    price_type = $31,
    is_enabled = $32
WHERE item_id = $1
`

const deleteAdminItemQuery = `
DELETE FROM item_definition
WHERE item_id = $1
`

func (r *ItemRepository) ListForAdmin(ctx context.Context, query item.AdminListQuery) (*item.AdminItemList, error) {
	query = query.Normalize()
	conditions := make([]string, 0, 4)
	args := make([]any, 0, 6)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.ItemID > 0 {
		conditions = append(conditions, "item_id = "+nextArg(query.ItemID))
	}
	if query.ItemType != "" {
		conditions = append(conditions, "item_type = "+nextArg(query.ItemType))
	}
	if query.ExcludeItemType != "" {
		conditions = append(conditions, "item_type <> "+nextArg(query.ExcludeItemType))
	}
	if query.Keyword != "" {
		placeholder := nextArg("%" + query.Keyword + "%")
		conditions = append(conditions, "(item_code ILIKE "+placeholder+" OR item_name ILIKE "+placeholder+")")
	}
	if query.Enabled != nil {
		conditions = append(conditions, "is_enabled = "+nextArg(*query.Enabled))
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}
	countQuery := "SELECT COUNT(1) FROM item_definition " + whereClause
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	listQuery := adminItemListBaseQuery + whereClause + fmt.Sprintf("\nORDER BY item_id DESC\nLIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]item.AdminItemSummary, 0)
	for rows.Next() {
		summary, err := scanAdminItemSummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &item.AdminItemList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *ItemRepository) FindAdminDetailByItemID(ctx context.Context, itemID uint64) (*item.AdminItemDetail, error) {
	detail, err := scanAdminItemDetailRow(r.db.QueryRowContext(ctx, adminItemDetailQuery, itemID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (r *ItemRepository) CreateForAdmin(ctx context.Context, input item.AdminUpsertItemInput) (*item.AdminItemDetail, error) {
	if _, err := r.db.ExecContext(ctx, insertAdminItemQuery,
		input.ItemID, input.ItemCode, input.ItemName, input.ItemType, input.ItemSubType,
		input.Quality, input.Rarity, input.Icon, input.Desc, input.MaxStack, input.OccupySlots,
		input.AutoMerge, input.SortWeight, input.Usable, input.UseScope, input.TargetType,
		input.RequiredLevel, input.RequiredSceneID, input.BindType, input.CanSell, input.CanDrop,
		input.CanStore, input.CanTrade, input.ExpireAtRule, input.EffectType, input.EffectValue,
		input.EffectParamsJSON, input.BuyPriceCopper, input.SellPriceCopper, input.RecyclePriceCopper,
		input.PriceType, input.IsEnabled,
	); err != nil {
		if isItemDefinitionUniqueViolation(err) {
			return nil, item.ErrItemDefinitionConflict
		}
		return nil, err
	}
	return r.FindAdminDetailByItemID(ctx, input.ItemID)
}

func (r *ItemRepository) UpdateForAdmin(ctx context.Context, itemID uint64, input item.AdminUpsertItemInput) (*item.AdminItemDetail, error) {
	result, err := r.db.ExecContext(ctx, updateAdminItemQuery,
		itemID, input.ItemCode, input.ItemName, input.ItemType, input.ItemSubType,
		input.Quality, input.Rarity, input.Icon, input.Desc, input.MaxStack, input.OccupySlots,
		input.AutoMerge, input.SortWeight, input.Usable, input.UseScope, input.TargetType,
		input.RequiredLevel, input.RequiredSceneID, input.BindType, input.CanSell, input.CanDrop,
		input.CanStore, input.CanTrade, input.ExpireAtRule, input.EffectType, input.EffectValue,
		input.EffectParamsJSON, input.BuyPriceCopper, input.SellPriceCopper, input.RecyclePriceCopper,
		input.PriceType, input.IsEnabled,
	)
	if err != nil {
		if isItemDefinitionUniqueViolation(err) {
			return nil, item.ErrItemDefinitionConflict
		}
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, item.ErrItemDefinitionNotFound
	}
	return r.FindAdminDetailByItemID(ctx, itemID)
}

func (r *ItemRepository) DeleteForAdmin(ctx context.Context, itemID uint64) error {
	result, err := r.db.ExecContext(ctx, deleteAdminItemQuery, itemID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return item.ErrItemDefinitionNotFound
	}
	return nil
}

func scanAdminItemSummary(rows *sql.Rows) (item.AdminItemSummary, error) {
	var value item.AdminItemSummary
	if err := rows.Scan(
		&value.ItemID,
		&value.ItemCode,
		&value.ItemName,
		&value.ItemType,
		&value.ItemSubType,
		&value.Quality,
		&value.Icon,
		&value.Desc,
		&value.MaxStack,
		&value.BuyPriceCopper,
		&value.SellPriceCopper,
		&value.Usable,
		&value.CanSell,
		&value.CanStore,
		&value.IsEnabled,
		&value.UpdatedAt,
		&value.CreatedAt,
	); err != nil {
		return item.AdminItemSummary{}, err
	}
	return value, nil
}

func scanAdminItemDetailRow(row *sql.Row) (*item.AdminItemDetail, error) {
	var (
		value      item.AdminItemDetail
		paramsJSON []byte
	)
	if err := row.Scan(
		&value.ItemID,
		&value.ItemCode,
		&value.ItemName,
		&value.ItemType,
		&value.ItemSubType,
		&value.Quality,
		&value.Rarity,
		&value.Icon,
		&value.Desc,
		&value.MaxStack,
		&value.OccupySlots,
		&value.AutoMerge,
		&value.SortWeight,
		&value.Usable,
		&value.UseScope,
		&value.TargetType,
		&value.RequiredLevel,
		&value.RequiredSceneID,
		&value.BindType,
		&value.CanSell,
		&value.CanDrop,
		&value.CanStore,
		&value.CanTrade,
		&value.ExpireAtRule,
		&value.EffectType,
		&value.EffectValue,
		&paramsJSON,
		&value.BuyPriceCopper,
		&value.SellPriceCopper,
		&value.RecyclePriceCopper,
		&value.PriceType,
		&value.IsEnabled,
		&value.CreatedAt,
		&value.UpdatedAt,
	); err != nil {
		return nil, err
	}
	value.EffectParamsJSON = string(paramsJSON)
	if value.EffectParamsJSON == "" {
		value.EffectParamsJSON = "{}"
	}
	var compacted json.RawMessage
	if json.Valid(paramsJSON) {
		compacted = append(compacted[:0], paramsJSON...)
		value.EffectParamsJSON = string(compacted)
	}
	return &value, nil
}

func isItemDefinitionUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

const runtimeItemNameByIDQuery = `
SELECT COALESCE(item_name, '')
FROM item_definition
WHERE item_id = $1
LIMIT 1
`

// loadItemNamesByIDs 批量读取物品模板名称，供介绍文案中的 {item:ID} 占位符解析。
func loadItemNamesByIDs(ctx context.Context, db DBTX, itemIDs []uint64) (map[uint64]string, error) {
	names := make(map[uint64]string, len(itemIDs))
	for _, itemID := range itemIDs {
		if itemID == 0 {
		 continue
		}
		var itemName string
		err := db.QueryRowContext(ctx, runtimeItemNameByIDQuery, itemID).Scan(&itemName)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		names[itemID] = itemName
	}
	return names, nil
}

// buildRuntimeDescriptionMentions 解析介绍文案中的 {item:ID} 并补齐服务端权威名称。
func buildRuntimeDescriptionMentions(ctx context.Context, db DBTX, description string) ([]item.DescriptionMention, error) {
	itemIDs := item.ExtractMentionItemIDs(description)
	if len(itemIDs) == 0 {
		return nil, nil
	}
	names, err := loadItemNamesByIDs(ctx, db, itemIDs)
	if err != nil {
		return nil, err
	}
	return item.BuildDescriptionMentions(description, names), nil
}
