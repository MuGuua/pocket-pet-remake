package world

import (
	"errors"
	"time"
)

var ErrSnapshotUnavailable = errors.New("scene snapshot unavailable")
var ErrMovementStateNotFound = errors.New("movement state not found")
var ErrMovementSequenceStale = errors.New("movement sequence is stale")
var ErrMovementSessionMismatch = errors.New("movement session mismatch")
var ErrMovementSceneMismatch = errors.New("movement scene mismatch")
var ErrMovementConfigUnavailable = errors.New("movement config unavailable")
var ErrMovementInputInvalid = errors.New("movement input invalid")
var ErrMovementAxisInvalid = errors.New("movement axis invalid")
var ErrMovementConfigInvalid = errors.New("movement config invalid")
var ErrSceneBoundaryUnavailable = errors.New("scene boundary unavailable")
var ErrSceneBoundaryInvalid = errors.New("scene boundary invalid")
var ErrSceneNavigationUnavailable = errors.New("scene navigation unavailable")
var ErrSceneNavigationInvalid = errors.New("scene navigation invalid")
var ErrSceneNavigationNotFound = errors.New("scene navigation not found")
var ErrSceneNavigationStateInvalid = errors.New("scene navigation state invalid")
var ErrSceneNavigationBlocked = errors.New("scene navigation blocks current position")

const (
	// MovementPositionFixedScale 表示世界移动定点坐标每个场景格包含的单位数。
	MovementPositionFixedScale int32 = 1000

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
// 当前市场场景的安全出生点固定为 (11, 10)，并与已发布静态通行位图保持一致。
func FallbackSpawnPos() Vec2i {
	return Vec2i{X: 11, Y: 10}
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

// MovementConfig 定义服务端权威世界移动速度和单包弱网容差，所有值均来自数据库配置。
type MovementConfig struct {
	SpeedMilliCellsPerSecond uint32    `json:"speed_milli_cells_per_second"`
	MaxElapsedMS             uint32    `json:"max_elapsed_ms"`
	AxisToleranceMilli       uint32    `json:"axis_tolerance_milli"`
	UpdatedAt                time.Time `json:"updated_at"`
	LastUpdateReason         string    `json:"last_update_reason"`
	UpdatedByAdminUserID     uint64    `json:"updated_by_admin_user_id"`
}

// AdminUpdateMovementConfigInput 是后台修改权威移动参数时使用的完整审计输入。
type AdminUpdateMovementConfigInput struct {
	SpeedMilliCellsPerSecond uint32 `json:"speed_milli_cells_per_second"`
	MaxElapsedMS             uint32 `json:"max_elapsed_ms"`
	AxisToleranceMilli       uint32 `json:"axis_tolerance_milli"`
	Reason                   string `json:"reason"`
	AdminUserID              uint64 `json:"-"`
}

// SceneBoundary 定义启用场景可移动中心点的闭区间边界，坐标统一使用千分之一场景格。
type SceneBoundary struct {
	SceneID              uint32    `json:"scene_id"`
	SceneCode            string    `json:"scene_code"`
	SceneName            string    `json:"scene_name"`
	MinX                 int32     `json:"min_x_milli"`
	MinY                 int32     `json:"min_y_milli"`
	MaxX                 int32     `json:"max_x_milli"`
	MaxY                 int32     `json:"max_y_milli"`
	UpdatedAt            time.Time `json:"updated_at"`
	LastUpdateReason     string    `json:"last_update_reason"`
	UpdatedByAdminUserID uint64    `json:"updated_by_admin_user_id"`
}

// AdminUpdateSceneBoundaryInput 是后台修改场景矩形边界时使用的完整审计输入。
type AdminUpdateSceneBoundaryInput struct {
	MinX        int32  `json:"min_x_milli"`
	MinY        int32  `json:"min_y_milli"`
	MaxX        int32  `json:"max_x_milli"`
	MaxY        int32  `json:"max_y_milli"`
	Reason      string `json:"reason"`
	AdminUserID uint64 `json:"-"`
}

// MovementIntent 是传输层转换后的移动输入意图和客户端候选表现坐标。
type MovementIntent struct {
	Input        *Vec2i
	CandidatePos Vec2i
	HasCandidate bool
	Facing       Vec2i
	Moving       bool
	MoveSeq      uint32
	ServerTickMS int64
}

// MovementResult 返回服务端按时间、速度和四方向输入裁剪后的权威状态。
type MovementResult struct {
	State     MovementState
	Corrected bool
}
