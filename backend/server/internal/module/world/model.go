package world

import "errors"

var ErrSnapshotUnavailable = errors.New("scene snapshot unavailable")
var ErrMovementStateNotFound = errors.New("movement state not found")
var ErrMovementSequenceStale = errors.New("movement sequence is stale")
var ErrMovementSessionMismatch = errors.New("movement session mismatch")
var ErrMovementSceneMismatch = errors.New("movement scene mismatch")

const (
	// SceneLevelRestrictedReason 是玩家等级不足时下发给客户端的统一提示。
	// 客户端只负责展示该服务端权威判定结果，不能在本地自行判断或切换地图。
	SceneLevelRestrictedReason = "前面的路以后再来探索吧"

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
	EntityID uint64
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

// MovementState 保存在线玩家的服务端权威移动状态；高精度坐标使用千分之一场景格定点整数。
type MovementState struct {
	PlayerID         uint64
	SessionID        string
	SceneID          uint32
	SceneVersion     uint32
	PrecisePos       Vec2i
	PersistedPos     Vec2i
	Facing           Vec2i
	Moving           bool
	Speed            uint32
	LastMoveSeq      uint32
	LastServerTickMS int64
	PositionVersion  uint64
}
