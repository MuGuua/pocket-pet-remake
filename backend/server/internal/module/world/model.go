package world

import "errors"

var ErrSnapshotUnavailable = errors.New("scene snapshot unavailable")

const (
	// FallbackSceneID 表示玩家当前持久化 scene_id 无法解析时，服务端权威回退到的默认市场场景。
	// 这里选择市场作为兜底出生点，避免玩家因为脏数据或旧档错误 scene_id 卡死在无法进入世界的状态。
	FallbackSceneID uint32 = 3
)

type Vec2i struct {
	X int32
	Y int32
}

// FallbackSpawnPos 返回默认回退场景中的权威出生点。
// 当前市场场景的安全出生点固定为 (12, 10)，与现有世界配置和测试桩保持一致。
func FallbackSpawnPos() Vec2i {
	return Vec2i{X: 12, Y: 10}
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
