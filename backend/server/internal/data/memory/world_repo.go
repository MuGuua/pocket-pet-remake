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
			// 这里对齐主世界 demo 里的 roxus_house 入门落点。
			2: {X: 8, Y: 12},
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
				// 这里对齐主世界 demo 里的东路上方入口。
				targetPos: world.Vec2i{X: 4, Y: 1},
			},
		},
	},
	2: {
		spawnPos: world.Vec2i{X: 4, Y: 1},
		entries: map[uint32]world.Vec2i{
			// 这里对齐主世界 demo 里的上方入口。
			1: {X: 4, Y: 1},
			// 这里对齐主世界 demo 里的左侧入口。
			3: {X: 0, Y: 4},
		},
		nearby: []world.Entity{
			{
				EntityID:   90002,
				EntityType: 2,
				Pos:        world.Vec2i{X: 2, Y: 3},
				Dir:        1,
				Speed:      0,
				Name:       "StationKeeper",
			},
		},
		exits: map[uint32]struct{}{1: {}, 3: {}},
		portals: map[uint32]portalData{
			2001: {
				targetSceneID: 1,
				// 这里对齐主世界 demo 里的 roxus_house 入门落点。
				targetPos: world.Vec2i{X: 8, Y: 12},
			},
			2002: {
				targetSceneID: 3,
				targetPos:     world.Vec2i{X: 12, Y: 10},
			},
		},
	},
	3: {
		spawnPos: world.Vec2i{X: 12, Y: 10},
		entries: map[uint32]world.Vec2i{
			2: {X: 12, Y: 10},
			4: {X: 5, Y: 2},
			5: {X: 4, Y: 13},
		},
		nearby: []world.Entity{
			{
				EntityID:   93001,
				EntityType: 2,
				Pos:        world.Vec2i{X: 13, Y: 8},
				Dir:        2,
				Speed:      0,
				Name:       "市场理萌",
			},
			{
				EntityID:   93002,
				EntityType: 2,
				Pos:        world.Vec2i{X: 14, Y: 6},
				Dir:        2,
				Speed:      0,
				Name:       "市场罗格",
			},
		},
		exits: map[uint32]struct{}{2: {}, 4: {}, 5: {}},
		portals: map[uint32]portalData{
			3001: {
				targetSceneID: 2,
				targetPos:     world.Vec2i{X: 0, Y: 4},
			},
			3002: {
				targetSceneID: 4,
				targetPos:     world.Vec2i{X: 2, Y: 8},
			},
			3003: {
				targetSceneID: 5,
				targetPos:     world.Vec2i{X: 11, Y: 2},
			},
		},
	},
	4: {
		spawnPos: world.Vec2i{X: 2, Y: 8},
		entries: map[uint32]world.Vec2i{
			3: {X: 2, Y: 8},
		},
		nearby: []world.Entity{
			{
				EntityID:   90004,
				EntityType: 2,
				Pos:        world.Vec2i{X: 4, Y: 7},
				Dir:        2,
				Speed:      0,
				Name:       "NorthFieldScout",
			},
		},
		exits: map[uint32]struct{}{3: {}},
		portals: map[uint32]portalData{
			4001: {
				targetSceneID: 3,
				targetPos:     world.Vec2i{X: 5, Y: 2},
			},
		},
	},
	5: {
		spawnPos: world.Vec2i{X: 11, Y: 2},
		entries: map[uint32]world.Vec2i{
			3: {X: 11, Y: 2},
			6: {X: 6, Y: 10},
		},
		nearby: []world.Entity{
			{
				EntityID:   90005,
				EntityType: 2,
				Pos:        world.Vec2i{X: 9, Y: 4},
				Dir:        1,
				Speed:      0,
				Name:       "SchoolCaretaker",
			},
		},
		exits: map[uint32]struct{}{3: {}, 6: {}},
		portals: map[uint32]portalData{
			5001: {
				targetSceneID: 3,
				targetPos:     world.Vec2i{X: 4, Y: 13},
			},
			5002: {
				targetSceneID: 6,
				targetPos:     world.Vec2i{X: 6, Y: 10},
			},
		},
	},
	6: {
		spawnPos: world.Vec2i{X: 6, Y: 10},
		entries: map[uint32]world.Vec2i{
			5: {X: 6, Y: 10},
		},
		nearby: []world.Entity{
			{
				EntityID:   90006,
				EntityType: 2,
				Pos:        world.Vec2i{X: 7, Y: 8},
				Dir:        0,
				Speed:      0,
				Name:       "BattleGuide",
			},
		},
		exits: map[uint32]struct{}{5: {}},
		portals: map[uint32]portalData{
			6001: {
				targetSceneID: 5,
				targetPos:     world.Vec2i{X: 6, Y: 10},
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
