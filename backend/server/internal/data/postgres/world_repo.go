package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"pocket-pet-remake/server/internal/module/world"
)

type WorldRepository struct {
	db DBTX
}

type portalData struct {
	targetSceneID uint32
	targetPos     world.Vec2i
}

type sceneData struct {
	spawnPos world.Vec2i
	entries  map[uint32]world.Vec2i
	exits    map[uint32]struct{}
	portals  map[uint32]portalData
}

var worldScenes = map[uint32]sceneData{
	// 坐标统一采用“场景内坐标 = 服务端世界坐标”的口径：每张地图左上角为 (0,0)，
	// 客户端只负责把该格子坐标按地图像素倍率渲染，不能再维护一套独立本地落点。
	// portals/entries 中的落点是多人同屏的唯一事实来源：进入者本人、同场景旁观者
	// 和数据库持久化坐标都使用这里的值，客户端场景脚本不再覆盖出生点。
	1: {
		spawnPos: world.Vec2i{X: 5, Y: 10},
		entries: map[uint32]world.Vec2i{
			2: {X: 5, Y: 13},
		},
		exits: map[uint32]struct{}{2: {}},
		portals: map[uint32]portalData{
			1001: {targetSceneID: 2, targetPos: world.Vec2i{X: 4, Y: 4}},
		},
	},
	2: {
		spawnPos: world.Vec2i{X: 4, Y: 5},
		entries: map[uint32]world.Vec2i{
			1: {X: 4, Y: 4},
			3: {X: 1, Y: 7},
			7: {X: 9, Y: 5},
			8: {X: 9, Y: 5},
		},
		exits: map[uint32]struct{}{1: {}, 3: {}, 8: {}},
		portals: map[uint32]portalData{
			2001: {targetSceneID: 1, targetPos: world.Vec2i{X: 5, Y: 13}},
			2002: {targetSceneID: 3, targetPos: world.Vec2i{X: 12, Y: 11}},
			2003: {targetSceneID: 8, targetPos: world.Vec2i{X: 1, Y: 13}},
		},
	},
	3: {
		spawnPos: world.Vec2i{X: 12, Y: 10},
		entries: map[uint32]world.Vec2i{
			2: {X: 12, Y: 11},
			4: {X: 5, Y: 1},
			5: {X: 4, Y: 12},
		},
		exits: map[uint32]struct{}{2: {}, 4: {}, 5: {}},
		portals: map[uint32]portalData{
			3001: {targetSceneID: 2, targetPos: world.Vec2i{X: 1, Y: 7}},
			3002: {targetSceneID: 4, targetPos: world.Vec2i{X: 4, Y: 10}},
			3003: {targetSceneID: 5, targetPos: world.Vec2i{X: 12, Y: 3}},
		},
	},
	4: {
		spawnPos: world.Vec2i{X: 4, Y: 7},
		entries: map[uint32]world.Vec2i{
			3: {X: 4, Y: 10},
		},
		exits: map[uint32]struct{}{3: {}},
		portals: map[uint32]portalData{
			4001: {targetSceneID: 3, targetPos: world.Vec2i{X: 5, Y: 1}},
		},
	},
	5: {
		spawnPos: world.Vec2i{X: 11, Y: 5},
		entries: map[uint32]world.Vec2i{
			3: {X: 12, Y: 3},
			6: {X: 6, Y: 11},
		},
		exits: map[uint32]struct{}{3: {}, 6: {}},
		portals: map[uint32]portalData{
			5001: {targetSceneID: 3, targetPos: world.Vec2i{X: 4, Y: 12}},
			5002: {targetSceneID: 6, targetPos: world.Vec2i{X: 6, Y: 2}},
		},
	},
	6: {
		spawnPos: world.Vec2i{X: 6, Y: 2},
		entries: map[uint32]world.Vec2i{
			5: {X: 6, Y: 2},
		},
		exits: map[uint32]struct{}{5: {}},
		portals: map[uint32]portalData{
			6001: {targetSceneID: 5, targetPos: world.Vec2i{X: 6, Y: 11}},
		},
	},
	7: {
		spawnPos: world.Vec2i{X: 6, Y: 6},
		entries:  map[uint32]world.Vec2i{},
		exits:    map[uint32]struct{}{2: {}},
		portals: map[uint32]portalData{
			7001: {targetSceneID: 2, targetPos: world.Vec2i{X: 9, Y: 5}},
		},
	},
	8: {
		spawnPos: world.Vec2i{X: 4, Y: 6},
		entries: map[uint32]world.Vec2i{
			2: {X: 1, Y: 13},
			9: {X: 6, Y: 9},
		},
		exits: map[uint32]struct{}{2: {}, 9: {}},
		portals: map[uint32]portalData{
			8001: {targetSceneID: 2, targetPos: world.Vec2i{X: 9, Y: 5}},
			8002: {targetSceneID: 9, targetPos: world.Vec2i{X: 20, Y: 12}},
		},
	},
	9: {
		spawnPos: world.Vec2i{X: 14, Y: 8},
		entries: map[uint32]world.Vec2i{
			8:  {X: 20, Y: 12},
			10: {X: 16, Y: 6},
			15: {X: 5, Y: 7},
			16: {X: 2, Y: 8},
			17: {X: 23, Y: 8},
			20: {X: 12, Y: 12},
		},
		exits: map[uint32]struct{}{8: {}, 10: {}, 15: {}, 16: {}, 17: {}, 20: {}},
		portals: map[uint32]portalData{
			9001: {targetSceneID: 8, targetPos: world.Vec2i{X: 6, Y: 9}},
			9002: {targetSceneID: 20, targetPos: world.Vec2i{X: 9, Y: 2}},
			9003: {targetSceneID: 16, targetPos: world.Vec2i{X: 13, Y: 9}},
			9004: {targetSceneID: 15, targetPos: world.Vec2i{X: 5, Y: 16}},
			9005: {targetSceneID: 10, targetPos: world.Vec2i{X: 3, Y: 10}},
			9006: {targetSceneID: 17, targetPos: world.Vec2i{X: 1, Y: 7}},
		},
	},
	10: {
		spawnPos: world.Vec2i{X: 5, Y: 8},
		entries: map[uint32]world.Vec2i{
			9:  {X: 3, Y: 10},
			14: {X: 6, Y: 10},
		},
		exits: map[uint32]struct{}{9: {}, 14: {}},
		portals: map[uint32]portalData{
			10001: {targetSceneID: 9, targetPos: world.Vec2i{X: 16, Y: 6}},
			10002: {targetSceneID: 14, targetPos: world.Vec2i{X: 5, Y: 10}},
		},
	},
	11: {
		spawnPos: world.Vec2i{X: 5, Y: 7},
		entries: map[uint32]world.Vec2i{
			15: {X: 9, Y: 10},
		},
		exits: map[uint32]struct{}{15: {}},
		portals: map[uint32]portalData{
			11001: {targetSceneID: 15, targetPos: world.Vec2i{X: 1, Y: 7}},
		},
	},
	12: {
		spawnPos: world.Vec2i{X: 5, Y: 7},
		entries: map[uint32]world.Vec2i{
			15: {X: 4, Y: 10},
		},
		exits: map[uint32]struct{}{15: {}},
		portals: map[uint32]portalData{
			12001: {targetSceneID: 15, targetPos: world.Vec2i{X: 5, Y: 2}},
		},
	},
	13: {
		spawnPos: world.Vec2i{X: 6, Y: 7},
		entries: map[uint32]world.Vec2i{
			15: {X: 1, Y: 9},
		},
		exits: map[uint32]struct{}{15: {}},
		portals: map[uint32]portalData{
			13001: {targetSceneID: 15, targetPos: world.Vec2i{X: 9, Y: 7}},
		},
	},
	14: {
		spawnPos: world.Vec2i{X: 5, Y: 8},
		entries: map[uint32]world.Vec2i{
			10: {X: 5, Y: 10},
		},
		exits: map[uint32]struct{}{10: {}},
		portals: map[uint32]portalData{
			14001: {targetSceneID: 10, targetPos: world.Vec2i{X: 6, Y: 10}},
		},
	},
	15: {
		spawnPos: world.Vec2i{X: 5, Y: 9},
		entries: map[uint32]world.Vec2i{
			9:  {X: 5, Y: 16},
			11: {X: 1, Y: 7},
			12: {X: 5, Y: 2},
			13: {X: 9, Y: 7},
		},
		exits: map[uint32]struct{}{9: {}, 11: {}, 12: {}, 13: {}},
		portals: map[uint32]portalData{
			15001: {targetSceneID: 11, targetPos: world.Vec2i{X: 9, Y: 10}},
			15002: {targetSceneID: 12, targetPos: world.Vec2i{X: 4, Y: 10}},
			15003: {targetSceneID: 13, targetPos: world.Vec2i{X: 1, Y: 9}},
			15004: {targetSceneID: 9, targetPos: world.Vec2i{X: 5, Y: 7}},
		},
	},
	16: {
		spawnPos: world.Vec2i{X: 7, Y: 7},
		entries: map[uint32]world.Vec2i{
			9: {X: 13, Y: 9},
		},
		exits: map[uint32]struct{}{9: {}},
		portals: map[uint32]portalData{
			16001: {targetSceneID: 9, targetPos: world.Vec2i{X: 2, Y: 8}},
		},
	},
	17: {
		spawnPos: world.Vec2i{X: 5, Y: 7},
		entries: map[uint32]world.Vec2i{
			9:  {X: 1, Y: 7},
			18: {X: 9, Y: 6},
		},
		exits: map[uint32]struct{}{9: {}, 18: {}},
		portals: map[uint32]portalData{
			17001: {targetSceneID: 9, targetPos: world.Vec2i{X: 23, Y: 8}},
			17002: {targetSceneID: 18, targetPos: world.Vec2i{X: 2, Y: 5}},
			17003: {targetSceneID: 18, targetPos: world.Vec2i{X: 7, Y: 5}},
		},
	},
	18: {
		spawnPos: world.Vec2i{X: 5, Y: 7},
		entries: map[uint32]world.Vec2i{
			17: {X: 2, Y: 5},
		},
		exits: map[uint32]struct{}{17: {}},
		portals: map[uint32]portalData{
			18001: {targetSceneID: 17, targetPos: world.Vec2i{X: 9, Y: 6}},
		},
	},
	19: {
		spawnPos: world.Vec2i{X: 7, Y: 10},
		entries: map[uint32]world.Vec2i{
			20: {X: 12, Y: 10},
		},
		exits: map[uint32]struct{}{20: {}},
		portals: map[uint32]portalData{
			19001: {targetSceneID: 20, targetPos: world.Vec2i{X: 4, Y: 8}},
		},
	},
	20: {
		spawnPos: world.Vec2i{X: 6, Y: 7},
		entries: map[uint32]world.Vec2i{
			9:  {X: 9, Y: 2},
			19: {X: 4, Y: 8},
			21: {X: 11, Y: 8},
			23: {X: 8, Y: 11},
		},
		exits: map[uint32]struct{}{9: {}, 19: {}, 21: {}, 23: {}},
		portals: map[uint32]portalData{
			20001: {targetSceneID: 9, targetPos: world.Vec2i{X: 12, Y: 12}},
			20002: {targetSceneID: 19, targetPos: world.Vec2i{X: 12, Y: 10}},
			20003: {targetSceneID: 21, targetPos: world.Vec2i{X: 1, Y: 5}},
			20004: {targetSceneID: 23, targetPos: world.Vec2i{X: 7, Y: 2}},
		},
	},
	21: {
		spawnPos: world.Vec2i{X: 9, Y: 7},
		entries: map[uint32]world.Vec2i{
			20: {X: 1, Y: 5},
		},
		exits: map[uint32]struct{}{20: {}},
		portals: map[uint32]portalData{
			21001: {targetSceneID: 20, targetPos: world.Vec2i{X: 11, Y: 8}},
		},
	},
	22: {
		spawnPos: world.Vec2i{X: 5, Y: 7},
		entries: map[uint32]world.Vec2i{
			23: {X: 9, Y: 9},
			24: {X: 4, Y: 10},
		},
		exits: map[uint32]struct{}{23: {}, 24: {}},
		portals: map[uint32]portalData{
			22001: {targetSceneID: 23, targetPos: world.Vec2i{X: 2, Y: 6}},
			22002: {targetSceneID: 24, targetPos: world.Vec2i{X: 5, Y: 2}},
		},
	},
	23: {
		spawnPos: world.Vec2i{X: 6, Y: 7},
		entries: map[uint32]world.Vec2i{
			20: {X: 7, Y: 2},
			22: {X: 2, Y: 6},
			26: {X: 8, Y: 11},
		},
		exits: map[uint32]struct{}{20: {}, 22: {}, 26: {}},
		portals: map[uint32]portalData{
			23001: {targetSceneID: 20, targetPos: world.Vec2i{X: 8, Y: 11}},
			23002: {targetSceneID: 22, targetPos: world.Vec2i{X: 9, Y: 9}},
			23003: {targetSceneID: 26, targetPos: world.Vec2i{X: 6, Y: 2}},
		},
	},
	24: {
		spawnPos: world.Vec2i{X: 5, Y: 7},
		entries: map[uint32]world.Vec2i{
			22: {X: 5, Y: 2},
		},
		exits: map[uint32]struct{}{22: {}},
		portals: map[uint32]portalData{
			24001: {targetSceneID: 22, targetPos: world.Vec2i{X: 4, Y: 10}},
		},
	},
	25: {
		spawnPos: world.Vec2i{X: 7, Y: 7},
		entries: map[uint32]world.Vec2i{
			26: {X: 2, Y: 8},
		},
		exits: map[uint32]struct{}{26: {}},
		portals: map[uint32]portalData{
			25001: {targetSceneID: 26, targetPos: world.Vec2i{X: 10, Y: 8}},
		},
	},
	26: {
		spawnPos: world.Vec2i{X: 6, Y: 7},
		entries: map[uint32]world.Vec2i{
			23: {X: 6, Y: 2},
			25: {X: 10, Y: 8},
		},
		exits: map[uint32]struct{}{23: {}, 25: {}},
		portals: map[uint32]portalData{
			26001: {targetSceneID: 23, targetPos: world.Vec2i{X: 8, Y: 11}},
			26002: {targetSceneID: 25, targetPos: world.Vec2i{X: 2, Y: 8}},
		},
	},
}

const listSceneEntitiesQuery = `
SELECT entity_id, entity_type, display_name, visibility_conditions_json
FROM world_entity_definition
WHERE scene_id = $1 AND status = 1
ORDER BY entity_id ASC
`

const getMapTeleportTargetQuery = `
SELECT teleport.center_x, teleport.center_y, scene.required_level, scene.status
FROM world_map_teleport_node AS teleport
JOIN world_scene_definition AS scene ON scene.scene_id = teleport.scene_id
WHERE teleport.scene_id = $1
  AND teleport.status = 1
  AND scene.status = 1
`

func NewWorldRepository(db DBTX) *WorldRepository {
	return &WorldRepository{db: db}
}

// GetMovementConfig 读取唯一启用的服务端权威世界移动配置。
func (r *WorldRepository) GetMovementConfig(ctx context.Context) (world.MovementConfig, error) {
	var config world.MovementConfig
	err := r.db.QueryRowContext(ctx, `
SELECT speed_milli_cells_per_second, max_elapsed_ms, axis_tolerance_milli,
       updated_at, COALESCE(last_update_reason, ''), COALESCE(updated_by_admin_user_id, 0)
FROM world_movement_config
WHERE config_id = 1 AND status = 1
LIMIT 1
`).Scan(&config.SpeedMilliCellsPerSecond, &config.MaxElapsedMS, &config.AxisToleranceMilli, &config.UpdatedAt, &config.LastUpdateReason, &config.UpdatedByAdminUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return world.MovementConfig{}, world.ErrMovementConfigUnavailable
	}
	return config, err
}

// UpdateMovementConfig 更新单例移动配置，并保存管理员和操作原因用于审计追踪。
func (r *WorldRepository) UpdateMovementConfig(ctx context.Context, input world.AdminUpdateMovementConfigInput) (world.MovementConfig, error) {
	var config world.MovementConfig
	err := r.db.QueryRowContext(ctx, `
UPDATE world_movement_config
SET speed_milli_cells_per_second = $1,
    max_elapsed_ms = $2,
    axis_tolerance_milli = $3,
    last_update_reason = $4,
    updated_by_admin_user_id = $5,
    updated_at = NOW()
WHERE config_id = 1 AND status = 1
RETURNING speed_milli_cells_per_second, max_elapsed_ms, axis_tolerance_milli,
          updated_at, last_update_reason, updated_by_admin_user_id
`, input.SpeedMilliCellsPerSecond, input.MaxElapsedMS, input.AxisToleranceMilli, input.Reason, input.AdminUserID).Scan(
		&config.SpeedMilliCellsPerSecond, &config.MaxElapsedMS, &config.AxisToleranceMilli,
		&config.UpdatedAt, &config.LastUpdateReason, &config.UpdatedByAdminUserID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return world.MovementConfig{}, world.ErrMovementConfigUnavailable
	}
	return config, err
}

// ListSceneBoundaries 按场景编号读取全部启用场景的数据库权威矩形边界。
func (r *WorldRepository) ListSceneBoundaries(ctx context.Context) ([]world.SceneBoundary, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT scene_id, scene_code, scene_name,
       boundary_min_x_milli, boundary_min_y_milli, boundary_max_x_milli, boundary_max_y_milli,
       updated_at, COALESCE(boundary_last_update_reason, ''), COALESCE(boundary_updated_by_admin_user_id, 0)
FROM world_scene_definition
WHERE status = 1
ORDER BY scene_id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	boundaries := make([]world.SceneBoundary, 0)
	for rows.Next() {
		var boundary world.SceneBoundary
		if err := rows.Scan(
			&boundary.SceneID, &boundary.SceneCode, &boundary.SceneName,
			&boundary.MinX, &boundary.MinY, &boundary.MaxX, &boundary.MaxY,
			&boundary.UpdatedAt, &boundary.LastUpdateReason, &boundary.UpdatedByAdminUserID,
		); err != nil {
			return nil, err
		}
		boundaries = append(boundaries, boundary)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(boundaries) == 0 {
		return nil, world.ErrSceneBoundaryUnavailable
	}
	return boundaries, nil
}

// UpdateSceneBoundary 更新指定启用场景的矩形边界和审计字段。
func (r *WorldRepository) UpdateSceneBoundary(ctx context.Context, sceneID uint32, input world.AdminUpdateSceneBoundaryInput) (world.SceneBoundary, error) {
	var boundary world.SceneBoundary
	err := r.db.QueryRowContext(ctx, `
UPDATE world_scene_definition
SET boundary_min_x_milli = $2,
    boundary_min_y_milli = $3,
    boundary_max_x_milli = $4,
    boundary_max_y_milli = $5,
    boundary_last_update_reason = $6,
    boundary_updated_by_admin_user_id = $7,
    updated_at = NOW()
WHERE scene_id = $1 AND status = 1
RETURNING scene_id, scene_code, scene_name,
          boundary_min_x_milli, boundary_min_y_milli, boundary_max_x_milli, boundary_max_y_milli,
          updated_at, boundary_last_update_reason, boundary_updated_by_admin_user_id
`, sceneID, input.MinX, input.MinY, input.MaxX, input.MaxY, input.Reason, input.AdminUserID).Scan(
		&boundary.SceneID, &boundary.SceneCode, &boundary.SceneName,
		&boundary.MinX, &boundary.MinY, &boundary.MaxX, &boundary.MaxY,
		&boundary.UpdatedAt, &boundary.LastUpdateReason, &boundary.UpdatedByAdminUserID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return world.SceneBoundary{}, world.ErrSceneBoundaryUnavailable
	}
	return boundary, err
}

func (r *WorldRepository) GetSceneSnapshot(ctx context.Context, playerID uint64, sceneID uint32, selfPos world.Vec2i) (*world.SceneSnapshot, error) {
	if _, ok := worldScenes[sceneID]; !ok {
		return nil, world.ErrSnapshotUnavailable
	}

	rows, err := r.db.QueryContext(ctx, listSceneEntitiesQuery, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nearby := []world.Entity{}
	for rows.Next() {
		var value world.Entity
		var visibilityConditionsJSON []byte
		if err := rows.Scan(
			&value.EntityID,
			&value.EntityType,
			&value.Name,
			&visibilityConditionsJSON,
		); err != nil {
			return nil, err
		}
		visible, err := r.entityVisibleForPlayer(ctx, playerID, visibilityConditionsJSON)
		if err != nil {
			return nil, err
		}
		if !visible {
			continue
		}
		// NPC 坐标由客户端场景资源维护；服务端快照只保留 entity_id/name 供交互与附近列表使用。
		value.Pos = world.Vec2i{}
		value.Dir = 0
		value.Speed = 0
		// The current table does not yet store a dedicated player_id column for
		// scene avatars, so we mirror entity_id for player entities to keep the
		// client-side PVP targeting contract explicit and stable.
		if value.EntityType == 1 {
			value.PlayerID = value.EntityID
		}
		nearby = append(nearby, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &world.SceneSnapshot{
		SceneID:        sceneID,
		SelfPos:        selfPos,
		SceneVersion:   1,
		NearbyEntities: nearby,
	}, nil
}

// entityVisibleForPlayer 根据 world_entity_definition.visibility_conditions_json 判断 NPC 是否对当前玩家可见。
// 第一版只支持 required_flags，后续任务状态、等级、活动开关都可以继续扩展到这里。
func (r *WorldRepository) entityVisibleForPlayer(ctx context.Context, playerID uint64, raw []byte) (bool, error) {
	conditions := struct {
		RequiredFlags []string `json:"required_flags"`
	}{}
	if len(raw) == 0 || string(raw) == "{}" || string(raw) == "null" {
		return true, nil
	}
	if err := json.Unmarshal(raw, &conditions); err != nil {
		return true, nil
	}
	if len(conditions.RequiredFlags) == 0 {
		return true, nil
	}
	if playerID == 0 {
		return false, nil
	}
	for _, flagKey := range conditions.RequiredFlags {
		if flagKey == "" {
			continue
		}
		var exists bool
		err := r.db.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1 FROM player_story_flag
  WHERE player_id = $1 AND flag_key = $2
)
`, playerID, flagKey).Scan(&exists)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}
	return true, nil
}

func (r *WorldRepository) EvaluateTransfer(ctx context.Context, _ uint64, playerLevel uint32, sceneID uint32, currentPos world.Vec2i, targetSceneID uint32, portalID uint32) (*world.MoveDecision, error) {
	decision := &world.MoveDecision{
		SceneVersion: 1,
		ToSceneID:    sceneID,
		SpawnPos:     currentPos,
	}

	currentScene, ok := worldScenes[sceneID]
	if !ok {
		decision.Accepted = false
		decision.Reason = "current scene unavailable"
		return decision, nil
	}

	targetScene, ok := worldScenes[targetSceneID]
	if !ok {
		decision.Accepted = false
		decision.Reason = "target scene unavailable"
		return decision, nil
	}

	// 先验证地图拓扑和传送门是否合法，再读取目标地图准入配置。
	// 这样非法客户端请求只能得到连通性错误，不能借等级提示探测不可达地图。
	if _, ok := currentScene.exits[targetSceneID]; !ok {
		decision.Accepted = false
		decision.Reason = "target scene unreachable"
		return decision, nil
	}

	targetPos := targetScene.spawnPos
	if portalID != 0 {
		portal, ok := currentScene.portals[portalID]
		if !ok {
			decision.Accepted = false
			decision.Reason = "portal unavailable"
			return decision, nil
		}
		if portal.targetSceneID != targetSceneID {
			decision.Accepted = false
			decision.Reason = "portal target mismatch"
			return decision, nil
		}
		targetPos = portal.targetPos
	} else if entryPos, ok := targetScene.entries[sceneID]; ok {
		targetPos = entryPos
	}

	// 玩家等级来自服务端玩家档案，地图要求来自数据库，客户端请求不参与两者计算。
	if r.db != nil {
		var requiredLevel uint32
		var status uint32
		err := r.db.QueryRowContext(ctx, `
SELECT required_level, status
FROM world_scene_definition
WHERE scene_id = $1
LIMIT 1
`, targetSceneID).Scan(&requiredLevel, &status)
		if errors.Is(err, sql.ErrNoRows) {
			decision.Accepted = false
			decision.Reason = "target scene unavailable"
			return decision, nil
		}
		if err != nil {
			return nil, err
		}
		if reason := sceneAccessRejectionReason(playerLevel, requiredLevel, status); reason != "" {
			decision.Accepted = false
			decision.Reason = reason
			return decision, nil
		}
	}

	decision.Accepted = true
	decision.ToSceneID = targetSceneID
	decision.SpawnPos = targetPos
	return decision, nil
}

// EvaluateMapTeleport 只接受数据库中已启用的地图节点，并使用服务端配置的中心格作为权威出生点。
func (r *WorldRepository) EvaluateMapTeleport(ctx context.Context, _ uint64, playerLevel uint32, sceneID uint32, currentPos world.Vec2i, targetSceneID uint32) (*world.MoveDecision, error) {
	decision := &world.MoveDecision{
		SceneVersion: 1,
		ToSceneID:    sceneID,
		SpawnPos:     currentPos,
	}
	if _, ok := worldScenes[sceneID]; !ok {
		decision.Reason = "current scene unavailable"
		return decision, nil
	}
	if targetSceneID == 0 {
		decision.Reason = "map teleport unavailable"
		return decision, nil
	}

	var center world.Vec2i
	var requiredLevel uint32
	var status uint32
	err := r.db.QueryRowContext(ctx, getMapTeleportTargetQuery, targetSceneID).Scan(&center.X, &center.Y, &requiredLevel, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			decision.Reason = "map teleport unavailable"
			return decision, nil
		}
		return nil, err
	}
	if reason := sceneAccessRejectionReason(playerLevel, requiredLevel, status); reason != "" {
		decision.Reason = reason
		return decision, nil
	}

	decision.Accepted = true
	decision.ToSceneID = targetSceneID
	decision.SpawnPos = center
	decision.Reason = "map teleport accepted"
	return decision, nil
}

// sceneAccessRejectionReason 根据数据库地图配置返回拒绝原因；空字符串表示允许进入。
// 该纯逻辑函数便于在不连接正式数据库的情况下覆盖等级边界测试。
func sceneAccessRejectionReason(playerLevel uint32, requiredLevel uint32, status uint32) string {
	if status != 1 {
		return "target scene unavailable"
	}
	if playerLevel < requiredLevel {
		return world.SceneLevelRestrictedReason
	}
	return ""
}
