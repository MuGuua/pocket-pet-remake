package memory

import (
	"context"

	"pocket-pet-remake/server/internal/module/world"
)

type WorldRepository struct{}

type portalData struct {
	targetSceneID uint32
	targetPos     world.Vec2i
}

type sceneData struct {
	spawnPos world.Vec2i
	entries  map[uint32]world.Vec2i
	nearby   []world.Entity
	exits    map[uint32]struct{}
	portals  map[uint32]portalData
}

var scenes = map[uint32]sceneData{
	1: {
		spawnPos: world.Vec2i{X: 8, Y: 6},
		entries: map[uint32]world.Vec2i{
			2: {X: 16, Y: 6},
		},
		nearby: []world.Entity{
			{
				EntityID:   90001,
				EntityType: 2,
				Pos:        world.Vec2i{X: 10, Y: 6},
				Dir:        2,
				Speed:      0,
				Name:       "GuideNPC",
			},
		},
		exits: map[uint32]struct{}{2: {}},
		portals: map[uint32]portalData{
			1001: {
				targetSceneID: 2,
				targetPos:     world.Vec2i{X: -6, Y: 4},
			},
		},
	},
	2: {
		spawnPos: world.Vec2i{X: 2, Y: 4},
		entries: map[uint32]world.Vec2i{
			1: {X: -6, Y: 4},
			3: {X: 10, Y: 4},
		},
		nearby: []world.Entity{
			{
				EntityID:   90002,
				EntityType: 2,
				Pos:        world.Vec2i{X: 5, Y: 4},
				Dir:        1,
				Speed:      0,
				Name:       "StationKeeper",
			},
		},
		exits: map[uint32]struct{}{1: {}, 3: {}},
		portals: map[uint32]portalData{
			2001: {
				targetSceneID: 1,
				targetPos:     world.Vec2i{X: 16, Y: 6},
			},
			2002: {
				targetSceneID: 3,
				targetPos:     world.Vec2i{X: -5, Y: 9},
			},
		},
	},
	3: {
		spawnPos: world.Vec2i{X: 3, Y: 9},
		entries: map[uint32]world.Vec2i{
			2: {X: -5, Y: 9},
		},
		nearby: []world.Entity{
			{
				EntityID:   90003,
				EntityType: 2,
				Pos:        world.Vec2i{X: 6, Y: 9},
				Dir:        3,
				Speed:      0,
				Name:       "ForestGuard",
			},
		},
		exits: map[uint32]struct{}{2: {}},
		portals: map[uint32]portalData{
			3001: {
				targetSceneID: 2,
				targetPos:     world.Vec2i{X: 10, Y: 4},
			},
		},
	},
}

func NewWorldRepository() *WorldRepository {
	return &WorldRepository{}
}

func (r *WorldRepository) GetSceneSnapshot(_ context.Context, _ uint64, sceneID uint32, selfPos world.Vec2i) (*world.SceneSnapshot, error) {
	scene, ok := scenes[sceneID]
	if !ok {
		return nil, world.ErrSnapshotUnavailable
	}

	return &world.SceneSnapshot{
		SceneID:        sceneID,
		SelfPos:        selfPos,
		SceneVersion:   1,
		NearbyEntities: scene.nearby,
	}, nil
}

func (r *WorldRepository) EvaluateTransfer(_ context.Context, _ uint64, sceneID uint32, currentPos world.Vec2i, targetSceneID uint32, portalID uint32) (*world.MoveDecision, error) {
	decision := &world.MoveDecision{
		SceneVersion: 1,
		ToSceneID:    sceneID,
		SpawnPos:     currentPos,
	}

	currentScene, ok := scenes[sceneID]
	if !ok {
		decision.Accepted = false
		decision.Reason = "current scene unavailable"
		return decision, nil
	}

	targetScene, ok := scenes[targetSceneID]
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
