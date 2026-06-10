package world

import "errors"

var ErrSnapshotUnavailable = errors.New("scene snapshot unavailable")

type Vec2i struct {
	X int32
	Y int32
}

type Entity struct {
	EntityID   uint64
	// PlayerID keeps the authoritative player identifier when this scene entity
	// represents an online player avatar. Non-player entities keep this as 0.
	PlayerID   uint64
	EntityType uint32
	Pos        Vec2i
	Dir        uint32
	Speed      uint32
	Name       string
}

type SceneSnapshot struct {
	SceneID        uint32
	SelfPos        Vec2i
	SceneVersion   uint32
	NearbyEntities []Entity
}

type MoveDecision struct {
	Accepted     bool
	SceneVersion uint32
	ToSceneID    uint32
	SpawnPos     Vec2i
	Reason       string
}
