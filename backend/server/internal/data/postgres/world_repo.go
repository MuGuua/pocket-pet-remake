package postgres

import (
	"context"

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
	1: {
		spawnPos: world.Vec2i{X: 8, Y: 6},
		entries: map[uint32]world.Vec2i{
			2: {X: 8, Y: 12},
		},
		exits: map[uint32]struct{}{2: {}},
		portals: map[uint32]portalData{
			1001: {targetSceneID: 2, targetPos: world.Vec2i{X: 4, Y: 1}},
		},
	},
	2: {
		spawnPos: world.Vec2i{X: 4, Y: 1},
		entries: map[uint32]world.Vec2i{
			1: {X: 4, Y: 1},
			3: {X: 0, Y: 4},
		},
		exits: map[uint32]struct{}{1: {}, 3: {}},
		portals: map[uint32]portalData{
			2001: {targetSceneID: 1, targetPos: world.Vec2i{X: 8, Y: 12}},
			2002: {targetSceneID: 3, targetPos: world.Vec2i{X: 12, Y: 10}},
		},
	},
	3: {
		spawnPos: world.Vec2i{X: 12, Y: 10},
		entries: map[uint32]world.Vec2i{
			2: {X: 12, Y: 10},
			4: {X: 5, Y: 2},
			5: {X: 4, Y: 13},
		},
		exits: map[uint32]struct{}{2: {}, 4: {}, 5: {}},
		portals: map[uint32]portalData{
			3001: {targetSceneID: 2, targetPos: world.Vec2i{X: 0, Y: 4}},
			3002: {targetSceneID: 4, targetPos: world.Vec2i{X: 2, Y: 8}},
			3003: {targetSceneID: 5, targetPos: world.Vec2i{X: 11, Y: 2}},
		},
	},
	4: {
		spawnPos: world.Vec2i{X: 2, Y: 8},
		entries: map[uint32]world.Vec2i{
			3: {X: 2, Y: 8},
		},
		exits: map[uint32]struct{}{3: {}},
		portals: map[uint32]portalData{
			4001: {targetSceneID: 3, targetPos: world.Vec2i{X: 5, Y: 2}},
		},
	},
	5: {
		spawnPos: world.Vec2i{X: 11, Y: 2},
		entries: map[uint32]world.Vec2i{
			3: {X: 11, Y: 2},
			6: {X: 6, Y: 10},
		},
		exits: map[uint32]struct{}{3: {}, 6: {}},
		portals: map[uint32]portalData{
			5001: {targetSceneID: 3, targetPos: world.Vec2i{X: 4, Y: 13}},
			5002: {targetSceneID: 6, targetPos: world.Vec2i{X: 6, Y: 10}},
		},
	},
	6: {
		spawnPos: world.Vec2i{X: 6, Y: 10},
		entries: map[uint32]world.Vec2i{
			5: {X: 6, Y: 10},
		},
		exits: map[uint32]struct{}{5: {}},
		portals: map[uint32]portalData{
			6001: {targetSceneID: 5, targetPos: world.Vec2i{X: 6, Y: 10}},
		},
	},
}

const listSceneEntitiesQuery = `
SELECT entity_id, entity_type, pos_x, pos_y, dir, speed, display_name
FROM world_entity_definition
WHERE scene_id = $1 AND status = 1
ORDER BY entity_id ASC
`

func NewWorldRepository(db DBTX) *WorldRepository {
	return &WorldRepository{db: db}
}

func (r *WorldRepository) GetSceneSnapshot(ctx context.Context, _ uint64, sceneID uint32, selfPos world.Vec2i) (*world.SceneSnapshot, error) {
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
		if err := rows.Scan(
			&value.EntityID,
			&value.EntityType,
			&value.Pos.X,
			&value.Pos.Y,
			&value.Dir,
			&value.Speed,
			&value.Name,
		); err != nil {
			return nil, err
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

func (r *WorldRepository) EvaluateTransfer(_ context.Context, _ uint64, sceneID uint32, currentPos world.Vec2i, targetSceneID uint32, portalID uint32) (*world.MoveDecision, error) {
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

	if _, ok := currentScene.exits[targetSceneID]; !ok {
		decision.Accepted = false
		decision.Reason = "target scene unreachable"
		return decision, nil
	}

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
		decision.Accepted = true
		decision.ToSceneID = portal.targetSceneID
		decision.SpawnPos = portal.targetPos
		return decision, nil
	}

	decision.Accepted = true
	decision.ToSceneID = targetSceneID
	decision.SpawnPos = targetScene.spawnPos
	if entryPos, ok := targetScene.entries[sceneID]; ok {
		decision.SpawnPos = entryPos
	}
	return decision, nil
}
