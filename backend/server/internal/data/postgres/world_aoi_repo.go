package postgres

import (
	"context"
	"database/sql"
	"errors"

	"pocket-pet-remake/server/internal/module/world"
)

// 读取数据库的唯一 AOI 配置，未执行迁移必须明确报错，禁止静默启用硬编码可见范围。
func (r *WorldRepository) GetAOIConfig(ctx context.Context) (world.AOIConfig, error) {
	var value world.AOIConfig
	err := r.db.QueryRowContext(ctx, `SELECT relay_enabled,enabled,cell_size_milli,neighbor_radius,updated_at,change_reason,COALESCE(updated_by_admin_user_id,0) FROM world_aoi_config WHERE config_id=1`).Scan(&value.RelayEnabled, &value.Enabled, &value.CellSizeMilli, &value.Radius, &value.UpdatedAt, &value.Reason, &value.AdminUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return value, world.ErrAOIConfigUnavailable
	}
	return value, err
}

// 只持久化配置及审计，不改变当前进程的运行可见关系。
func (r *WorldRepository) UpdateAOIConfig(ctx context.Context, input world.UpdateAOIConfigInput) (world.AOIConfig, error) {
	var value world.AOIConfig
	err := r.db.QueryRowContext(ctx, `UPDATE world_aoi_config SET relay_enabled=COALESCE($6,relay_enabled),enabled=$1,cell_size_milli=$2,neighbor_radius=$3,change_reason=$4,updated_by_admin_user_id=$5,updated_at=CURRENT_TIMESTAMP WHERE config_id=1 RETURNING relay_enabled,enabled,cell_size_milli,neighbor_radius,updated_at,change_reason,COALESCE(updated_by_admin_user_id,0)`, input.Enabled, input.CellSizeMilli, input.Radius, input.Reason, input.AdminUserID, input.RelayEnabled).Scan(&value.RelayEnabled, &value.Enabled, &value.CellSizeMilli, &value.Radius, &value.UpdatedAt, &value.Reason, &value.AdminUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return value, world.ErrAOIConfigUnavailable
	}
	return value, err
}
