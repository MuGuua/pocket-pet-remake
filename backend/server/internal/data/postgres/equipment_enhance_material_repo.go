package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/item"
)

const runtimeEnhanceMaterialConfigQuery = `
SELECT
  item_id,
  success_rate_mode,
  success_rate_bonus_pct,
  success_rate_override_pct,
  guaranteed_success,
  failure_penalty,
  failure_level_delta,
  COALESCE(description, ''),
  status
FROM equipment_enhance_material_config
WHERE item_id = $1
  AND status = 1
LIMIT 1
`

const upsertEnhanceMaterialConfigQuery = `
INSERT INTO equipment_enhance_material_config (
  item_id, success_rate_mode, success_rate_bonus_pct, success_rate_override_pct,
  guaranteed_success, failure_penalty, failure_level_delta, description, status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 1)
ON CONFLICT (item_id) DO UPDATE SET
  success_rate_mode = EXCLUDED.success_rate_mode,
  success_rate_bonus_pct = EXCLUDED.success_rate_bonus_pct,
  success_rate_override_pct = EXCLUDED.success_rate_override_pct,
  guaranteed_success = EXCLUDED.guaranteed_success,
  failure_penalty = EXCLUDED.failure_penalty,
  failure_level_delta = EXCLUDED.failure_level_delta,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP
`

const deleteEnhanceMaterialConfigQuery = `
DELETE FROM equipment_enhance_material_config
WHERE item_id = $1
`

const adminEnhanceMaterialConfigQuery = `
SELECT
  success_rate_mode,
  success_rate_bonus_pct,
  success_rate_override_pct,
  guaranteed_success,
  failure_penalty,
  failure_level_delta,
  COALESCE(description, '')
FROM equipment_enhance_material_config
WHERE item_id = $1
LIMIT 1
`

// loadRuntimeEnhanceMaterialConfig 读取强化材料运行时配置；缺失时返回默认 base+damage。
func loadRuntimeEnhanceMaterialConfig(ctx context.Context, db DBTX, itemID uint64) (equipment.EnhanceMaterialConfig, error) {
	if itemID == 0 {
		return equipment.DefaultRuntimeEnhanceMaterialConfig(0), nil
	}
	var (
		cfg        equipment.EnhanceMaterialConfig
		bonus      int64
		override   int64
		levelDelta int64
		status     int64
	)
	err := db.QueryRowContext(ctx, runtimeEnhanceMaterialConfigQuery, itemID).Scan(
		&cfg.ItemID,
		&cfg.SuccessRateMode,
		&bonus,
		&override,
		&cfg.GuaranteedSuccess,
		&cfg.FailurePenalty,
		&levelDelta,
		&cfg.Description,
		&status,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return equipment.DefaultRuntimeEnhanceMaterialConfig(itemID), nil
	}
	if err != nil {
		return equipment.EnhanceMaterialConfig{}, err
	}
	cfg.SuccessRateBonusPct = uint32(bonus)
	cfg.SuccessRateOverridePct = uint32(override)
	cfg.FailureLevelDelta = uint32(levelDelta)
	cfg.Status = uint8(status)
	return cfg, nil
}

// loadAdminEnhanceMaterialConfig 读取后台编辑弹窗所需的锻造材料配置。
func loadAdminEnhanceMaterialConfig(ctx context.Context, db DBTX, itemID uint64) (*item.AdminEnhanceMaterialConfig, error) {
	var cfg equipment.AdminEnhanceMaterialConfig
	var bonus int64
	var override int64
	var levelDelta int64
	err := db.QueryRowContext(ctx, adminEnhanceMaterialConfigQuery, itemID).Scan(
		&cfg.SuccessRateMode,
		&bonus,
		&override,
		&cfg.GuaranteedSuccess,
		&cfg.FailurePenalty,
		&levelDelta,
		&cfg.Description,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	cfg.SuccessRateBonusPct = uint32(bonus)
	cfg.SuccessRateOverridePct = uint32(override)
	cfg.FailureLevelDelta = uint32(levelDelta)
	converted := toItemEnhanceMaterialConfig(cfg)
	return &converted, nil
}

// upsertAdminEnhanceMaterialConfig 保存后台提交的锻造材料配置。
func upsertAdminEnhanceMaterialConfig(ctx context.Context, db DBTX, itemID uint64, input equipment.AdminEnhanceMaterialConfig) error {
	normalized := input.Normalize()
	if err := normalized.Validate(); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, upsertEnhanceMaterialConfigQuery,
		itemID,
		normalized.SuccessRateMode,
		normalized.SuccessRateBonusPct,
		normalized.SuccessRateOverridePct,
		normalized.GuaranteedSuccess,
		normalized.FailurePenalty,
		normalized.FailureLevelDelta,
		normalized.Description,
	)
	return err
}

// deleteAdminEnhanceMaterialConfig 删除非强化材料切换子分类时的旧配置。
func deleteAdminEnhanceMaterialConfig(ctx context.Context, db DBTX, itemID uint64) error {
	_, err := db.ExecContext(ctx, deleteEnhanceMaterialConfigQuery, itemID)
	return err
}

// syncAdminEnhanceMaterialConfig 在物品保存后同步锻造材料配置。
func syncAdminEnhanceMaterialConfig(ctx context.Context, db DBTX, itemID uint64, itemSubType string, input *item.AdminEnhanceMaterialConfig) error {
	if itemID == 0 {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(itemSubType), "equipment_enhance") {
		return deleteAdminEnhanceMaterialConfig(ctx, db, itemID)
	}
	if input == nil {
		return upsertAdminEnhanceMaterialConfig(ctx, db, itemID, toEquipmentEnhanceMaterialConfig(item.DefaultAdminEnhanceMaterialConfig()))
	}
	return upsertAdminEnhanceMaterialConfig(ctx, db, itemID, toEquipmentEnhanceMaterialConfig(*input))
}

func toEquipmentEnhanceMaterialConfig(input item.AdminEnhanceMaterialConfig) equipment.AdminEnhanceMaterialConfig {
	normalized := input.Normalize()
	return equipment.AdminEnhanceMaterialConfig{
		SuccessRateMode:        normalized.SuccessRateMode,
		SuccessRateBonusPct:    normalized.SuccessRateBonusPct,
		SuccessRateOverridePct: normalized.SuccessRateOverridePct,
		GuaranteedSuccess:      normalized.GuaranteedSuccess,
		FailurePenalty:         normalized.FailurePenalty,
		FailureLevelDelta:      normalized.FailureLevelDelta,
		Description:            normalized.Description,
	}
}

func toItemEnhanceMaterialConfig(input equipment.AdminEnhanceMaterialConfig) item.AdminEnhanceMaterialConfig {
	normalized := input.Normalize()
	return item.AdminEnhanceMaterialConfig{
		SuccessRateMode:        normalized.SuccessRateMode,
		SuccessRateBonusPct:    normalized.SuccessRateBonusPct,
		SuccessRateOverridePct: normalized.SuccessRateOverridePct,
		GuaranteedSuccess:      normalized.GuaranteedSuccess,
		FailurePenalty:         normalized.FailurePenalty,
		FailureLevelDelta:      normalized.FailureLevelDelta,
		Description:            normalized.Description,
	}
}
