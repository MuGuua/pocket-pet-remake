package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"pocket-pet-remake/server/internal/module/equipment"
)

const runtimeEnhanceGoldCostByItemQuery = `
SELECT
  iee.enhance_gold_cost_enabled,
  iee.enhance_gold_base_copper,
  iee.enhance_gold_increment_mode,
  iee.enhance_gold_increment_fixed,
  iee.enhance_gold_increment_percent
FROM item_equipment_extra iee
WHERE iee.item_id = $1
LIMIT 1
`

const runtimeEnhanceGoldCostConfigQuery = `
SELECT
  is_enabled,
  base_copper,
  increment_mode,
  increment_fixed,
  increment_percent,
  COALESCE(description, ''),
  updated_at
FROM equipment_enhance_gold_cost_config
WHERE config_id = 1
LIMIT 1
`

const upsertEnhanceGoldCostConfigQuery = `
INSERT INTO equipment_enhance_gold_cost_config (
  config_id,
  is_enabled,
  base_copper,
  increment_mode,
  increment_fixed,
  increment_percent,
  description
) VALUES (
  1,
  $1,
  $2,
  $3,
  $4,
  $5,
  $6
)
ON CONFLICT (config_id) DO UPDATE SET
  is_enabled = EXCLUDED.is_enabled,
  base_copper = EXCLUDED.base_copper,
  increment_mode = EXCLUDED.increment_mode,
  increment_fixed = EXCLUDED.increment_fixed,
  increment_percent = EXCLUDED.increment_percent,
  description = EXCLUDED.description,
  updated_at = CURRENT_TIMESTAMP
RETURNING updated_at
`

// GetEnhanceGoldCostConfigForAdmin 读取后台可编辑的强化铜币公式配置。
func (r *EquipmentRepository) GetEnhanceGoldCostConfigForAdmin(ctx context.Context) (*equipment.AdminEnhanceGoldCostConfigDetail, error) {
	config, err := loadRuntimeEnhanceGoldCostConfig(ctx, r.db)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, equipment.ErrEnhanceGoldCostConfigNotFound
	}
	return &equipment.AdminEnhanceGoldCostConfigDetail{
		Config:  *config,
		Preview: equipment.BuildEnhanceGoldCostPreview(*config, 15),
	}, nil
}

// UpsertEnhanceGoldCostConfigForAdmin 保存强化铜币公式并返回最新预览。
func (r *EquipmentRepository) UpsertEnhanceGoldCostConfigForAdmin(
	ctx context.Context,
	input equipment.AdminUpsertEnhanceGoldCostConfigInput,
) (*equipment.AdminEnhanceGoldCostConfigDetail, error) {
	input = input.Normalize()
	var updatedAt time.Time
	err := r.db.QueryRowContext(
		ctx,
		upsertEnhanceGoldCostConfigQuery,
		input.IsEnabled,
		input.BaseCopper,
		input.IncrementMode,
		input.IncrementFixed,
		input.IncrementPercent,
		input.Description,
	).Scan(&updatedAt)
	if err != nil {
		return nil, err
	}
	config := input.ToConfig(updatedAt)
	return &equipment.AdminEnhanceGoldCostConfigDetail{
		Config:  config,
		Preview: equipment.BuildEnhanceGoldCostPreview(config, 15),
	}, nil
}

// loadRuntimeEnhanceGoldCostConfigByItemID 读取单件装备模板的强化铜币公式。
func loadRuntimeEnhanceGoldCostConfigByItemID(ctx context.Context, db DBTX, itemID uint64) (*equipment.EnhanceGoldCostConfig, error) {
	if itemID == 0 {
		return nil, nil
	}
	var (
		config           equipment.EnhanceGoldCostConfig
		baseCopper       int64
		incrementFixed   int64
		incrementPercent int64
		incrementMode    string
	)
	err := db.QueryRowContext(ctx, runtimeEnhanceGoldCostByItemQuery, itemID).Scan(
		&config.IsEnabled,
		&baseCopper,
		&incrementMode,
		&incrementFixed,
		&incrementPercent,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if baseCopper < 0 {
		baseCopper = 0
	}
	if incrementFixed < 0 {
		incrementFixed = 0
	}
	if incrementPercent < 0 {
		incrementPercent = 0
	}
	config.BaseCopper = uint64(baseCopper)
	config.IncrementMode = incrementMode
	config.IncrementFixed = uint64(incrementFixed)
	config.IncrementPercent = uint32(incrementPercent)
	return &config, nil
}

// loadRuntimeEnhanceGoldCostConfig 读取运行时强化铜币公式；表未初始化时返回 nil。
func loadRuntimeEnhanceGoldCostConfig(ctx context.Context, db DBTX) (*equipment.EnhanceGoldCostConfig, error) {
	var (
		config            equipment.EnhanceGoldCostConfig
		baseCopper        int64
		incrementFixed    int64
		incrementPercent  int64
		incrementMode     string
		description       string
	)
	err := db.QueryRowContext(ctx, runtimeEnhanceGoldCostConfigQuery).Scan(
		&config.IsEnabled,
		&baseCopper,
		&incrementMode,
		&incrementFixed,
		&incrementPercent,
		&description,
		&config.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if baseCopper < 0 {
		baseCopper = 0
	}
	if incrementFixed < 0 {
		incrementFixed = 0
	}
	if incrementPercent < 0 {
		incrementPercent = 0
	}
	config.BaseCopper = uint64(baseCopper)
	config.IncrementMode = incrementMode
	config.IncrementFixed = uint64(incrementFixed)
	config.IncrementPercent = uint32(incrementPercent)
	config.Description = description
	return &config, nil
}
