package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"pocket-pet-remake/server/internal/module/world"
)

const sceneNavigationSelectColumns = `
SELECT navigation.navigation_id,
       navigation.scene_id,
       scene.scene_code,
       scene.scene_name,
       navigation.version,
       navigation.origin_x_milli,
       navigation.origin_y_milli,
       navigation.grid_width,
       navigation.grid_height,
       navigation.cell_size_milli,
       navigation.navigation_data,
       navigation.data_hash,
       navigation.walkable_cell_count,
       navigation.source_scene_path,
       navigation.status,
       navigation.change_reason,
       navigation.publish_reason,
       COALESCE(navigation.created_by_admin_user_id, 0),
       COALESCE(navigation.published_by_admin_user_id, 0),
       navigation.created_at,
       navigation.published_at,
       navigation.updated_at
FROM world_scene_navigation AS navigation
JOIN world_scene_definition AS scene ON scene.scene_id = navigation.scene_id
`

// ListPublishedSceneNavigations 读取全部当前发布版本，供服务启动时构建只读位图缓存。
func (r *WorldRepository) ListPublishedSceneNavigations(ctx context.Context) ([]world.SceneNavigation, error) {
	rows, err := r.db.QueryContext(ctx, sceneNavigationSelectColumns+`
WHERE navigation.status = 1
ORDER BY navigation.scene_id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSceneNavigations(rows)
}

// ListAdminSceneNavigations 按版本倒序返回指定场景的草稿、发布和历史记录。
func (r *WorldRepository) ListAdminSceneNavigations(ctx context.Context, sceneID uint32) ([]world.SceneNavigation, error) {
	rows, err := r.db.QueryContext(ctx, sceneNavigationSelectColumns+`
WHERE navigation.scene_id = $1
ORDER BY navigation.version DESC
`, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSceneNavigations(rows)
}

// CreateSceneNavigationDraft 锁定场景定义行并分配递增版本，避免并发上传生成重复版本号。
func (r *WorldRepository) CreateSceneNavigationDraft(ctx context.Context, input world.CreateSceneNavigationDraftInput) (world.SceneNavigation, error) {
	tx, err := r.beginWorldNavigationTx(ctx)
	if err != nil {
		return world.SceneNavigation{}, err
	}
	defer tx.Rollback()
	if err := lockWorldScene(ctx, tx, input.SceneID); err != nil {
		return world.SceneNavigation{}, err
	}
	var nextVersion uint32
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(MAX(version), 0) + 1
FROM world_scene_navigation
WHERE scene_id = $1
`, input.SceneID).Scan(&nextVersion); err != nil {
		return world.SceneNavigation{}, err
	}
	var navigationID uint64
	if err := tx.QueryRowContext(ctx, `
INSERT INTO world_scene_navigation (
    scene_id, version, origin_x_milli, origin_y_milli,
    grid_width, grid_height, cell_size_milli, navigation_data,
    data_hash, walkable_cell_count, source_scene_path, status,
    change_reason, publish_reason, created_by_admin_user_id,
    published_by_admin_user_id, published_at, updated_at
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8,
    $9, $10, $11, 2,
    $12, '', $13,
    NULL, NULL, CURRENT_TIMESTAMP
)
RETURNING navigation_id
`, input.SceneID, nextVersion, input.OriginX, input.OriginY,
		input.GridWidth, input.GridHeight, input.CellSizeMilli, input.NavigationData,
		input.DataHash, input.WalkableCellCount, input.SourceScenePath, input.Reason, input.AdminUserID,
	).Scan(&navigationID); err != nil {
		return world.SceneNavigation{}, err
	}
	navigation, err := getSceneNavigationByID(ctx, tx, navigationID)
	if err != nil {
		return world.SceneNavigation{}, err
	}
	if err := tx.Commit(); err != nil {
		return world.SceneNavigation{}, err
	}
	return navigation, nil
}

// PublishSceneNavigation 在单个事务中归档当前版本并发布指定草稿，唯一索引提供最终并发保护。
func (r *WorldRepository) PublishSceneNavigation(ctx context.Context, navigationID uint64, input world.AdminPublishSceneNavigationInput) (world.SceneNavigation, error) {
	tx, err := r.beginWorldNavigationTx(ctx)
	if err != nil {
		return world.SceneNavigation{}, err
	}
	defer tx.Rollback()
	var sceneID uint32
	var status int16
	if err := tx.QueryRowContext(ctx, `
SELECT scene_id, status
FROM world_scene_navigation
WHERE navigation_id = $1
FOR UPDATE
`, navigationID).Scan(&sceneID, &status); errors.Is(err, sql.ErrNoRows) {
		return world.SceneNavigation{}, world.ErrSceneNavigationNotFound
	} else if err != nil {
		return world.SceneNavigation{}, err
	}
	if status != world.SceneNavigationStatusDraft {
		return world.SceneNavigation{}, world.ErrSceneNavigationStateInvalid
	}
	if err := lockWorldScene(ctx, tx, sceneID); err != nil {
		return world.SceneNavigation{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE world_scene_navigation
SET status = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE scene_id = $1 AND status = 1
`, sceneID); err != nil {
		return world.SceneNavigation{}, err
	}
	result, err := tx.ExecContext(ctx, `
UPDATE world_scene_navigation
SET status = 1,
    publish_reason = $2,
    published_by_admin_user_id = $3,
    published_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE navigation_id = $1 AND status = 2
`, navigationID, input.Reason, input.AdminUserID)
	if err != nil {
		return world.SceneNavigation{}, err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return world.SceneNavigation{}, err
		}
		return world.SceneNavigation{}, world.ErrSceneNavigationStateInvalid
	}
	navigation, err := getSceneNavigationByID(ctx, tx, navigationID)
	if err != nil {
		return world.SceneNavigation{}, err
	}
	if err := tx.Commit(); err != nil {
		return world.SceneNavigation{}, err
	}
	return navigation, nil
}

// RollbackSceneNavigation 复制历史位图为新版本并发布，原历史记录始终保持不可变。
func (r *WorldRepository) RollbackSceneNavigation(ctx context.Context, sceneID uint32, input world.AdminRollbackSceneNavigationInput) (world.SceneNavigation, error) {
	tx, err := r.beginWorldNavigationTx(ctx)
	if err != nil {
		return world.SceneNavigation{}, err
	}
	defer tx.Rollback()
	if err := lockWorldScene(ctx, tx, sceneID); err != nil {
		return world.SceneNavigation{}, err
	}
	source, err := getSceneNavigationBySceneVersionForUpdate(ctx, tx, sceneID, input.SourceVersion)
	if err != nil {
		return world.SceneNavigation{}, err
	}
	if source.Status == world.SceneNavigationStatusPublished {
		return world.SceneNavigation{}, world.ErrSceneNavigationStateInvalid
	}
	var nextVersion uint32
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(MAX(version), 0) + 1
FROM world_scene_navigation
WHERE scene_id = $1
`, sceneID).Scan(&nextVersion); err != nil {
		return world.SceneNavigation{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE world_scene_navigation
SET status = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE scene_id = $1 AND status = 1
`, sceneID); err != nil {
		return world.SceneNavigation{}, err
	}
	var navigationID uint64
	changeReason := fmt.Sprintf("回滚自版本 %d：%s", input.SourceVersion, input.Reason)
	if len([]rune(changeReason)) > 500 {
		changeReason = input.Reason
	}
	if err := tx.QueryRowContext(ctx, `
INSERT INTO world_scene_navigation (
    scene_id, version, origin_x_milli, origin_y_milli,
    grid_width, grid_height, cell_size_milli, navigation_data,
    data_hash, walkable_cell_count, source_scene_path, status,
    change_reason, publish_reason, created_by_admin_user_id,
    published_by_admin_user_id, published_at, updated_at
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8,
    $9, $10, $11, 1,
    $12, $13, $14,
    $14, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
RETURNING navigation_id
`, sceneID, nextVersion, source.OriginX, source.OriginY,
		source.GridWidth, source.GridHeight, source.CellSizeMilli, source.NavigationData,
		source.DataHash, source.WalkableCellCount, source.SourceScenePath,
		changeReason, input.Reason, input.AdminUserID,
	).Scan(&navigationID); err != nil {
		return world.SceneNavigation{}, err
	}
	navigation, err := getSceneNavigationByID(ctx, tx, navigationID)
	if err != nil {
		return world.SceneNavigation{}, err
	}
	if err := tx.Commit(); err != nil {
		return world.SceneNavigation{}, err
	}
	return navigation, nil
}

// beginWorldNavigationTx 要求正式 PostgreSQL 连接支持事务，避免发布过程出现短暂无发布版本。
func (r *WorldRepository) beginWorldNavigationTx(ctx context.Context) (*sql.Tx, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, errors.New("world navigation repository requires transaction support")
	}
	return beginner.BeginTx(ctx, nil)
}

// lockWorldScene 锁定场景定义作为同场景导航版本写操作的串行化互斥点。
func lockWorldScene(ctx context.Context, tx *sql.Tx, sceneID uint32) error {
	var lockedSceneID uint32
	err := tx.QueryRowContext(ctx, `
SELECT scene_id
FROM world_scene_definition
WHERE scene_id = $1 AND status = 1
FOR UPDATE
`, sceneID).Scan(&lockedSceneID)
	if errors.Is(err, sql.ErrNoRows) {
		return world.ErrSceneNavigationNotFound
	}
	return err
}

// getSceneNavigationByID 在当前事务中读取完整版本，供提交前构造领域返回值。
func getSceneNavigationByID(ctx context.Context, db DBTX, navigationID uint64) (world.SceneNavigation, error) {
	row := db.QueryRowContext(ctx, sceneNavigationSelectColumns+`
WHERE navigation.navigation_id = $1
`, navigationID)
	navigation, err := scanSceneNavigation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return world.SceneNavigation{}, world.ErrSceneNavigationNotFound
	}
	return navigation, err
}

// getSceneNavigationBySceneVersionForUpdate 锁定回滚来源，防止并发状态更新影响事务判断。
func getSceneNavigationBySceneVersionForUpdate(ctx context.Context, tx *sql.Tx, sceneID uint32, version uint32) (world.SceneNavigation, error) {
	row := tx.QueryRowContext(ctx, sceneNavigationSelectColumns+`
WHERE navigation.scene_id = $1 AND navigation.version = $2
FOR UPDATE OF navigation
`, sceneID, version)
	navigation, err := scanSceneNavigation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return world.SceneNavigation{}, world.ErrSceneNavigationNotFound
	}
	return navigation, err
}

// rowScanner 抽象 sql.Row 和 sql.Rows 的公共 Scan 能力，保持字段映射只有一份。
type sceneNavigationRowScanner interface {
	Scan(dest ...any) error
}

// scanSceneNavigation 将数据库可空发布时间转换为领域结构的零值时间。
func scanSceneNavigation(scanner sceneNavigationRowScanner) (world.SceneNavigation, error) {
	var navigation world.SceneNavigation
	var publishedAt sql.NullTime
	err := scanner.Scan(
		&navigation.NavigationID, &navigation.SceneID, &navigation.SceneCode, &navigation.SceneName,
		&navigation.Version, &navigation.OriginX, &navigation.OriginY,
		&navigation.GridWidth, &navigation.GridHeight, &navigation.CellSizeMilli,
		&navigation.NavigationData, &navigation.DataHash, &navigation.WalkableCellCount,
		&navigation.SourceScenePath, &navigation.Status, &navigation.ChangeReason, &navigation.PublishReason,
		&navigation.CreatedByAdminUserID, &navigation.PublishedByAdminUserID,
		&navigation.CreatedAt, &publishedAt, &navigation.UpdatedAt,
	)
	if err != nil {
		return world.SceneNavigation{}, err
	}
	if publishedAt.Valid {
		navigation.PublishedAt = publishedAt.Time
	}
	navigation.DataHash = strings.ToLower(navigation.DataHash)
	return navigation, nil
}

// scanSceneNavigations 扫描列表并保留空切片，保证后台统一响应 data 字段不为缺失状态。
func scanSceneNavigations(rows *sql.Rows) ([]world.SceneNavigation, error) {
	navigations := make([]world.SceneNavigation, 0)
	for rows.Next() {
		navigation, err := scanSceneNavigation(rows)
		if err != nil {
			return nil, err
		}
		navigations = append(navigations, navigation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return navigations, nil
}
