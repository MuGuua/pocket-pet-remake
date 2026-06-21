package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"pocket-pet-remake/server/internal/module/equipment"
)

// EquipmentRepository 把人物装备模板 Admin CRUD 映射到 item_definition + item_equipment_extra。
type EquipmentRepository struct {
	db DBTX
}

// NewEquipmentRepository 构造装备模板仓储。
func NewEquipmentRepository(db DBTX) *EquipmentRepository {
	return &EquipmentRepository{db: db}
}

const adminEquipmentListBaseQuery = `
SELECT
  idf.item_id,
  idf.item_code,
  idf.item_name,
  iee.equip_slot,
  idf.required_level,
  idf.quality,
  iee.can_enhance,
  iee.max_enhance_level,
  iee.set_id,
  idf.is_enabled,
  idf.updated_at,
  idf.created_at
FROM item_definition idf
JOIN item_equipment_extra iee ON iee.item_id = idf.item_id
WHERE idf.item_type = 'equipment'
`

const adminEquipmentDetailQuery = `
SELECT
  idf.item_id,
  idf.item_code,
  idf.item_name,
  idf."desc",
  idf.icon,
  idf.quality,
  idf.rarity,
  idf.required_level,
  idf.bind_type,
  idf.can_sell,
  idf.can_store,
  idf.is_enabled,
  iee.equip_slot,
  iee.career_limit,
  iee.can_enhance,
  iee.max_enhance_level,
  iee.set_id,
  iee.appearance_skin_id,
  iee.appearance_only,
  iee.base_hp,
  iee.base_mana,
  iee.base_atk,
  iee.base_def,
  iee.base_spd,
  COALESCE(iee.base_stats_json, '{}'::jsonb),
  COALESCE(iee.enhance_per_level_stats_json, '{}'::jsonb),
  iee.socket_count,
  COALESCE(iee.allowed_gem_types_json, '[]'::jsonb),
  idf.created_at,
  idf.updated_at
FROM item_definition idf
JOIN item_equipment_extra iee ON iee.item_id = idf.item_id
WHERE idf.item_id = $1
  AND idf.item_type = 'equipment'
LIMIT 1
`

const insertAdminEquipmentItemQuery = `
INSERT INTO item_definition (
  item_id, item_code, item_name, item_type, item_sub_type, quality, rarity, icon, "desc",
  max_stack, occupy_slots, auto_merge, sort_weight, usable, use_scope, target_type,
  required_level, required_scene_id, bind_type, can_sell, can_drop, can_store, can_trade,
  expire_at_rule, effect_type, effect_value, effect_params_json,
  buy_price_copper, sell_price_copper, recycle_price_copper, price_type, is_enabled
) VALUES (
  $1,$2,$3,'equipment','', $4,$5,$6,$7,
  1,1,FALSE,0,FALSE,'','',
  $8,0,$9,$10,FALSE,TRUE,$11,
  '','',0,'{}'::jsonb,
  0,0,0,'base_coin',$12
)
`

const insertAdminEquipmentExtraQuery = `
INSERT INTO item_equipment_extra (
  item_id, equip_slot, base_hp, base_mana, base_atk, base_def, base_spd,
  career_limit, pet_only, player_only, extra_rule_json,
  can_enhance, max_enhance_level, set_id, appearance_skin_id, appearance_only,
  base_stats_json, enhance_per_level_stats_json, socket_count, allowed_gem_types_json
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,
  $8,FALSE,TRUE,'{}'::jsonb,
  $9,$10,$11,$12,$13,
  $14::jsonb,$15::jsonb,$16,$17::jsonb
)
`

const updateAdminEquipmentItemQuery = `
UPDATE item_definition
SET item_code = $2,
    item_name = $3,
    quality = $4,
    rarity = $5,
    icon = $6,
    "desc" = $7,
    required_level = $8,
    bind_type = $9,
    can_sell = $10,
    can_store = $11,
    is_enabled = $12
WHERE item_id = $1
  AND item_type = 'equipment'
`

const updateAdminEquipmentExtraQuery = `
UPDATE item_equipment_extra
SET equip_slot = $2,
    base_hp = $3,
    base_mana = $4,
    base_atk = $5,
    base_def = $6,
    base_spd = $7,
    career_limit = $8,
    can_enhance = $9,
    max_enhance_level = $10,
    set_id = $11,
    appearance_skin_id = $12,
    appearance_only = $13,
    base_stats_json = $14::jsonb,
    enhance_per_level_stats_json = $15::jsonb,
    socket_count = $16,
    allowed_gem_types_json = $17::jsonb
WHERE item_id = $1
`

const disableAdminEquipmentItemQuery = `
UPDATE item_definition
SET is_enabled = FALSE
WHERE item_id = $1
  AND item_type = 'equipment'
`

const adminMedicinePouchDetailQuery = `
SELECT restore_player_hp, restore_player_spirit, restore_player_vigor,
       restore_pet_hp, restore_pet_spirit, restore_lineup_pets
FROM item_medicine_pouch_extra
WHERE item_id = $1
LIMIT 1
`

const upsertMedicinePouchExtraQuery = `
INSERT INTO item_medicine_pouch_extra (
  item_id, restore_player_hp, restore_player_spirit, restore_player_vigor,
  restore_pet_hp, restore_pet_spirit, restore_lineup_pets
) VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (item_id) DO UPDATE SET
  restore_player_hp = EXCLUDED.restore_player_hp,
  restore_player_spirit = EXCLUDED.restore_player_spirit,
  restore_player_vigor = EXCLUDED.restore_player_vigor,
  restore_pet_hp = EXCLUDED.restore_pet_hp,
  restore_pet_spirit = EXCLUDED.restore_pet_spirit,
  restore_lineup_pets = EXCLUDED.restore_lineup_pets,
  updated_at = CURRENT_TIMESTAMP
`

const deleteMedicinePouchExtraQuery = `
DELETE FROM item_medicine_pouch_extra
WHERE item_id = $1
`

func (r *EquipmentRepository) ListForAdmin(ctx context.Context, query equipment.AdminListQuery) (*equipment.AdminEquipmentList, error) {
	query = query.Normalize()
	conditions := make([]string, 0, 4)
	args := make([]any, 0, 6)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.ItemID > 0 {
		conditions = append(conditions, "idf.item_id = "+nextArg(query.ItemID))
	}
	if query.EquipSlot != "" {
		conditions = append(conditions, "iee.equip_slot = "+nextArg(query.EquipSlot))
	}
	if query.SetID > 0 {
		conditions = append(conditions, "iee.set_id = "+nextArg(query.SetID))
	}
	if query.Keyword != "" {
		placeholder := nextArg("%" + query.Keyword + "%")
		conditions = append(conditions, "(idf.item_code ILIKE "+placeholder+" OR idf.item_name ILIKE "+placeholder+")")
	}
	if query.Enabled != nil {
		conditions = append(conditions, "idf.is_enabled = "+nextArg(*query.Enabled))
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " AND " + strings.Join(conditions, " AND ")
	}
	countQuery := `SELECT COUNT(1) FROM item_definition idf JOIN item_equipment_extra iee ON iee.item_id = idf.item_id WHERE idf.item_type = 'equipment'` + whereClause
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	listQuery := adminEquipmentListBaseQuery + whereClause + fmt.Sprintf("\nORDER BY idf.item_id DESC\nLIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]equipment.AdminEquipmentSummary, 0)
	for rows.Next() {
		item, err := scanAdminEquipmentSummaryRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &equipment.AdminEquipmentList{
		Items:    items,
		Total:    uint64(total),
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (r *EquipmentRepository) FindForAdminByItemID(ctx context.Context, itemID uint64) (*equipment.AdminEquipmentDetail, error) {
	row := r.db.QueryRowContext(ctx, adminEquipmentDetailQuery, itemID)
	detail, err := scanAdminEquipmentDetailRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if detail.EquipSlot == string(equipment.EquipSlotMedicinePouch) {
		pouch, err := r.loadMedicinePouchExtra(ctx, itemID)
		if err != nil {
			return nil, err
		}
		detail.MedicinePouch = pouch
	}
	return &detail, nil
}

func (r *EquipmentRepository) CreateForAdmin(ctx context.Context, input equipment.AdminUpsertEquipmentInput) (*equipment.AdminEquipmentDetail, error) {
	input = input.Normalize()
	baseStatsJSON, err := marshalCombatStatsJSON(input.CombatStats)
	if err != nil {
		return nil, err
	}
	enhanceJSON, err := marshalUint32MapJSON(input.EnhancePerLevelStats)
	if err != nil {
		return nil, err
	}
	gemTypesJSON, err := marshalStringSliceJSON(input.AllowedGemTypes)
	if err != nil {
		return nil, err
	}
	if _, err := r.db.ExecContext(ctx, insertAdminEquipmentItemQuery,
		input.ItemID, input.ItemCode, input.ItemName, input.Quality, input.Rarity, input.Icon, input.Desc,
		input.RequiredLevel, input.BindType, input.CanSell, input.CanStore, input.IsEnabled,
	); err != nil {
		if isUniqueViolation(err) {
			return nil, equipment.ErrEquipmentDefinitionConflict
		}
		return nil, err
	}
	if _, err := r.db.ExecContext(ctx, insertAdminEquipmentExtraQuery,
		input.ItemID, input.EquipSlot,
		input.BaseHP, input.BaseMana, input.BaseATK, input.BaseDEF, input.BaseSPD,
		input.CareerLimit,
		input.CanEnhance, input.MaxEnhanceLevel, input.SetID, input.AppearanceSkinID, input.AppearanceOnly,
		baseStatsJSON, enhanceJSON, input.SocketCount, gemTypesJSON,
	); err != nil {
		return nil, err
	}
	if err := r.syncMedicinePouchExtra(ctx, input.ItemID, input); err != nil {
		return nil, err
	}
	return r.FindForAdminByItemID(ctx, input.ItemID)
}

func (r *EquipmentRepository) UpdateForAdmin(ctx context.Context, itemID uint64, input equipment.AdminUpsertEquipmentInput) (*equipment.AdminEquipmentDetail, error) {
	input = input.Normalize()
	baseStatsJSON, err := marshalCombatStatsJSON(input.CombatStats)
	if err != nil {
		return nil, err
	}
	enhanceJSON, err := marshalUint32MapJSON(input.EnhancePerLevelStats)
	if err != nil {
		return nil, err
	}
	gemTypesJSON, err := marshalStringSliceJSON(input.AllowedGemTypes)
	if err != nil {
		return nil, err
	}
	result, err := r.db.ExecContext(ctx, updateAdminEquipmentItemQuery,
		itemID, input.ItemCode, input.ItemName, input.Quality, input.Rarity, input.Icon, input.Desc,
		input.RequiredLevel, input.BindType, input.CanSell, input.CanStore, input.IsEnabled,
	)
	if err != nil {
		return nil, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, nil
	}
	if _, err := r.db.ExecContext(ctx, updateAdminEquipmentExtraQuery,
		itemID, input.EquipSlot,
		input.BaseHP, input.BaseMana, input.BaseATK, input.BaseDEF, input.BaseSPD,
		input.CareerLimit,
		input.CanEnhance, input.MaxEnhanceLevel, input.SetID, input.AppearanceSkinID, input.AppearanceOnly,
		baseStatsJSON, enhanceJSON, input.SocketCount, gemTypesJSON,
	); err != nil {
		return nil, err
	}
	if err := r.syncMedicinePouchExtra(ctx, itemID, input); err != nil {
		return nil, err
	}
	return r.FindForAdminByItemID(ctx, itemID)
}

func (r *EquipmentRepository) DeleteForAdmin(ctx context.Context, itemID uint64) error {
	result, err := r.db.ExecContext(ctx, disableAdminEquipmentItemQuery, itemID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return equipment.ErrEquipmentDefinitionNotFound
	}
	return nil
}

func (r *EquipmentRepository) syncMedicinePouchExtra(ctx context.Context, itemID uint64, input equipment.AdminUpsertEquipmentInput) error {
	if input.EquipSlot != string(equipment.EquipSlotMedicinePouch) {
		_, err := r.db.ExecContext(ctx, deleteMedicinePouchExtraQuery, itemID)
		return err
	}
	pouch := input.MedicinePouch
	if pouch == nil {
		pouch = &equipment.AdminMedicinePouchExtra{
			RestorePlayerHP: true, RestorePlayerSpirit: true, RestorePlayerVigor: true,
			RestorePetHP: true, RestorePetSpirit: true,
		}
	}
	_, err := r.db.ExecContext(ctx, upsertMedicinePouchExtraQuery,
		itemID,
		pouch.RestorePlayerHP, pouch.RestorePlayerSpirit, pouch.RestorePlayerVigor,
		pouch.RestorePetHP, pouch.RestorePetSpirit, pouch.RestoreLineupPets,
	)
	return err
}

func (r *EquipmentRepository) loadMedicinePouchExtra(ctx context.Context, itemID uint64) (*equipment.AdminMedicinePouchExtra, error) {
	row := r.db.QueryRowContext(ctx, adminMedicinePouchDetailQuery, itemID)
	var extra equipment.AdminMedicinePouchExtra
	if err := row.Scan(
		&extra.RestorePlayerHP, &extra.RestorePlayerSpirit, &extra.RestorePlayerVigor,
		&extra.RestorePetHP, &extra.RestorePetSpirit, &extra.RestoreLineupPets,
	); errors.Is(err, sql.ErrNoRows) {
		return &equipment.AdminMedicinePouchExtra{
			RestorePlayerHP: true, RestorePlayerSpirit: true, RestorePlayerVigor: true,
			RestorePetHP: true, RestorePetSpirit: true,
		}, nil
	} else if err != nil {
		return nil, err
	}
	return &extra, nil
}

type adminEquipmentScanner interface {
	Scan(dest ...any) error
}

func scanAdminEquipmentSummaryRow(scanner adminEquipmentScanner) (equipment.AdminEquipmentSummary, error) {
	var item equipment.AdminEquipmentSummary
	var canEnhance bool
	var maxEnhance int32
	var setID int64
	if err := scanner.Scan(
		&item.ItemID, &item.ItemCode, &item.ItemName, &item.EquipSlot,
		&item.RequiredLevel, &item.Quality,
		&canEnhance, &maxEnhance, &setID, &item.IsEnabled,
		&item.UpdatedAt, &item.CreatedAt,
	); err != nil {
		return equipment.AdminEquipmentSummary{}, err
	}
	item.CanEnhance = canEnhance
	item.MaxEnhanceLevel = uint32(maxEnhance)
	item.SetID = uint64(setID)
	item.EquipSlotLabel = equipment.EquipSlotLabel(equipment.EquipSlot(item.EquipSlot))
	return item, nil
}

func scanAdminEquipmentDetailRow(scanner adminEquipmentScanner) (equipment.AdminEquipmentDetail, error) {
	var detail equipment.AdminEquipmentDetail
	var baseHP, baseMana, baseATK, baseDEF, baseSPD int64
	var canEnhance bool
	var maxEnhance int32
	var setID int64
	var socketCount int32
	var baseStatsRaw, enhanceRaw, gemTypesRaw []byte
	if err := scanner.Scan(
		&detail.ItemID, &detail.ItemCode, &detail.ItemName, &detail.Desc, &detail.Icon,
		&detail.Quality, &detail.Rarity, &detail.RequiredLevel, &detail.BindType,
		&detail.CanSell, &detail.CanStore, &detail.IsEnabled,
		&detail.EquipSlot, &detail.CareerLimit, &canEnhance, &maxEnhance, &setID,
		&detail.AppearanceSkinID, &detail.AppearanceOnly,
		&baseHP, &baseMana, &baseATK, &baseDEF, &baseSPD,
		&baseStatsRaw, &enhanceRaw, &socketCount, &gemTypesRaw,
		&detail.CreatedAt, &detail.UpdatedAt,
	); err != nil {
		return equipment.AdminEquipmentDetail{}, err
	}
	detail.BaseHP = uint32(baseHP)
	detail.BaseMana = uint32(baseMana)
	detail.BaseATK = uint32(baseATK)
	detail.BaseDEF = uint32(baseDEF)
	detail.BaseSPD = uint32(baseSPD)
	detail.CanEnhance = canEnhance
	detail.MaxEnhanceLevel = uint32(maxEnhance)
	detail.SetID = uint64(setID)
	detail.SocketCount = uint32(socketCount)
	detail.EquipSlotLabel = equipment.EquipSlotLabel(equipment.EquipSlot(detail.EquipSlot))
	combatStats, err := unmarshalCombatStatsJSON(baseStatsRaw)
	if err != nil {
		return equipment.AdminEquipmentDetail{}, err
	}
	detail.CombatStats = combatStats
	detail.EnhancePerLevelStats, err = unmarshalUint32MapJSON(enhanceRaw)
	if err != nil {
		return equipment.AdminEquipmentDetail{}, err
	}
	detail.AllowedGemTypes, err = unmarshalStringSliceJSON(gemTypesRaw)
	if err != nil {
		return equipment.AdminEquipmentDetail{}, err
	}
	return detail, nil
}

func marshalCombatStatsJSON(stats equipment.AdminCombatStats) (string, error) {
	payload := map[string]uint32{
		"spirit":                      stats.Spirit,
		"spirit_max":                  stats.SpiritMax,
		"hit_pct":                     stats.HitPct,
		"dodge_pct":                   stats.DodgePct,
		"crit_rate_pct":               stats.CritRatePct,
		"crit_dmg_pct":                stats.CritDmgPct,
		"physical_resist_pct":         stats.PhysicalResistPct,
		"reverse_physical_resist_pct": stats.ReversePhysicalResistPct,
		"skill_resist_pct":            stats.SkillResistPct,
		"reverse_skill_resist_pct":    stats.ReverseSkillResistPct,
		"confusion_resist_pct":        stats.ConfusionResistPct,
		"sleep_resist_pct":            stats.SleepResistPct,
		"paralysis_resist_pct":        stats.ParalysisResistPct,
		"seal_resist_pct":             stats.SealResistPct,
		"curse_resist_pct":            stats.CurseResistPct,
		"crit_dmg_resist_pct":         stats.CritDmgResistPct,
		"crit_resist_pct":             stats.CritResistPct,
		"character_resist_pct":        stats.CharacterResistPct,
		"pet_resist_pct":              stats.PetResistPct,
	}
	raw, err := json.Marshal(payload)
	return string(raw), err
}

func unmarshalCombatStatsJSON(raw []byte) (equipment.AdminCombatStats, error) {
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

func marshalUint32MapJSON(values map[string]uint32) (string, error) {
	if values == nil {
		values = map[string]uint32{}
	}
	raw, err := json.Marshal(values)
	return string(raw), err
}

func unmarshalUint32MapJSON(raw []byte) (map[string]uint32, error) {
	if len(raw) == 0 {
		return map[string]uint32{}, nil
	}
	result := make(map[string]uint32)
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func marshalStringSliceJSON(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	raw, err := json.Marshal(values)
	return string(raw), err
}

func unmarshalStringSliceJSON(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}
	var result []string
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result, nil
}
