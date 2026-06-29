package postgres

import (
	"context"
	"database/sql"
	"errors"

	"pocket-pet-remake/server/internal/module/equipment"
)

const adminListEnhanceSuccessConfigsBaseQuery = `
SELECT target_level, required_level_min, success_rate_pct, COALESCE(description, ''), status, updated_at
FROM equipment_enhance_success_config
`

const adminUpsertEnhanceSuccessConfigQuery = `
INSERT INTO equipment_enhance_success_config (
  target_level, required_level_min, success_rate_pct, description, status
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (target_level, required_level_min) DO UPDATE SET
  success_rate_pct = EXCLUDED.success_rate_pct,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP
RETURNING target_level, required_level_min, success_rate_pct, COALESCE(description, ''), status, updated_at
`

const runtimeEnhanceSuccessRateByBandQuery = `
SELECT success_rate_pct
FROM equipment_enhance_success_config
WHERE target_level = $1
  AND required_level_min = $2
  AND status = 1
LIMIT 1
`

// ListEnhanceSuccessConfigsForAdmin 返回全局强化成功率配置，可按穿戴等级段筛选。
func (r *EquipmentRepository) ListEnhanceSuccessConfigsForAdmin(
	ctx context.Context,
	requiredLevelMin *uint32,
) (*equipment.AdminEnhanceSuccessConfigList, error) {
	query := adminListEnhanceSuccessConfigsBaseQuery
	args := make([]any, 0, 1)
	if requiredLevelMin != nil && *requiredLevelMin > 0 {
		query += " WHERE required_level_min = $1"
		args = append(args, *requiredLevelMin)
	}
	query += " ORDER BY required_level_min ASC, target_level ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]equipment.AdminEnhanceSuccessConfig, 0, equipment.MaxEnhanceTargetLevel*4)
	for rows.Next() {
		item, scanErr := scanAdminEnhanceSuccessConfigFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		item.FillRequiredLevelBandDisplay()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &equipment.AdminEnhanceSuccessConfigList{Items: items}, nil
}

// UpsertEnhanceSuccessConfigForAdmin 更新指定穿戴等级段与目标强化等级的成功率。
func (r *EquipmentRepository) UpsertEnhanceSuccessConfigForAdmin(
	ctx context.Context,
	targetLevel uint32,
	requiredLevelMin uint32,
	input equipment.AdminUpsertEnhanceSuccessConfigInput,
) (*equipment.AdminEnhanceSuccessConfig, error) {
	normalized := input.Normalize()
	if err := normalized.Validate(targetLevel, requiredLevelMin); err != nil {
		return nil, err
	}
	item, err := scanAdminEnhanceSuccessConfigRow(
		r.db.QueryRowContext(
			ctx,
			adminUpsertEnhanceSuccessConfigQuery,
			targetLevel,
			requiredLevelMin,
			normalized.SuccessRatePct,
			normalized.Description,
			normalized.Status,
		),
	)
	if err != nil {
		return nil, err
	}
	item.FillRequiredLevelBandDisplay()
	return &item, nil
}

// loadRuntimeEnhanceSuccessRate 按目标强化等级与装备穿戴等级段读取全局基础成功率。
func loadRuntimeEnhanceSuccessRate(ctx context.Context, db DBTX, targetLevel uint32, requiredLevel uint32) (uint32, error) {
	bandMin := equipment.ResolveRequiredLevelBandMin(requiredLevel)
	rate, found, err := queryRuntimeEnhanceSuccessRateByBand(ctx, db, targetLevel, bandMin)
	if err != nil {
		return 0, err
	}
	if found {
		return rate, nil
	}
	if bandMin != 1 {
		rate, found, err = queryRuntimeEnhanceSuccessRateByBand(ctx, db, targetLevel, 1)
		if err != nil {
			return 0, err
		}
		if found {
			return rate, nil
		}
	}
	return equipment.DefaultEnhanceSuccessRate(targetLevel), nil
}

func queryRuntimeEnhanceSuccessRateByBand(
	ctx context.Context,
	db DBTX,
	targetLevel uint32,
	requiredLevelMin uint32,
) (uint32, bool, error) {
	var successRatePct int64
	err := db.QueryRowContext(ctx, runtimeEnhanceSuccessRateByBandQuery, targetLevel, requiredLevelMin).Scan(&successRatePct)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return uint32(successRatePct), true, nil
}

func scanAdminEnhanceSuccessConfigFromRows(rows *sql.Rows) (equipment.AdminEnhanceSuccessConfig, error) {
	var item equipment.AdminEnhanceSuccessConfig
	var targetLevel int64
	var requiredLevelMin int64
	var successRatePct int64
	var status int64
	if err := rows.Scan(
		&targetLevel,
		&requiredLevelMin,
		&successRatePct,
		&item.Description,
		&status,
		&item.UpdatedAt,
	); err != nil {
		return equipment.AdminEnhanceSuccessConfig{}, err
	}
	item.TargetLevel = uint32(targetLevel)
	item.RequiredLevelMin = uint32(requiredLevelMin)
	item.SuccessRatePct = uint32(successRatePct)
	item.Status = uint8(status)
	return item, nil
}

func scanAdminEnhanceSuccessConfigRow(row *sql.Row) (equipment.AdminEnhanceSuccessConfig, error) {
	var item equipment.AdminEnhanceSuccessConfig
	var targetLevel int64
	var requiredLevelMin int64
	var successRatePct int64
	var status int64
	if err := row.Scan(
		&targetLevel,
		&requiredLevelMin,
		&successRatePct,
		&item.Description,
		&status,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return equipment.AdminEnhanceSuccessConfig{}, equipment.ErrEnhanceSuccessConfigNotFound
		}
		return equipment.AdminEnhanceSuccessConfig{}, err
	}
	item.TargetLevel = uint32(targetLevel)
	item.RequiredLevelMin = uint32(requiredLevelMin)
	item.SuccessRatePct = uint32(successRatePct)
	item.Status = uint8(status)
	return item, nil
}
