package battle

import (
	"errors"

	"pocket-pet-remake/server/internal/module/world"
)

const (
	BattleTypePVE        uint32 = 1
	ActionTypeSkill      uint32 = 1
	ActionTypeEscape     uint32 = 4
	EventTypeUseSkill    uint32 = 1
	EventTypeDamage      uint32 = 2
	DefaultAttackSkillID uint32 = 1001
	PlayerActorType      uint32 = 1
	EnemyActorType       uint32 = 2
	DefaultEnemyPetID    uint32 = 9001
	DefaultEnemySkillID  uint32 = 90001
)

var (
	ErrBattleAlreadyActive = errors.New("battle already active")
	ErrBattleNotFound      = errors.New("battle not found")
	ErrInvalidAction       = errors.New("invalid battle action")
	ErrNoLineupAvailable   = errors.New("no lineup available")
	ErrTargetUnavailable   = errors.New("target unavailable")
)

type ActorSnapshot struct {
	ActorID     uint64
	ActorType   uint32
	PetUID      uint64
	PetID       uint32
	Name        string
	HP          uint32
	HPMax       uint32
	SkillIDs    []uint32
	LineupIndex uint32
}

type StartSnapshot struct {
	BattleID      uint64
	BattleType    uint32
	BattleVersion uint32
	Allies        []ActorSnapshot
	Enemies       []ActorSnapshot
	Round         uint32
	ActiveActorID uint64
	ActivePetUID  uint64
}

type Event struct {
	EventType uint32
	SourceID  uint64
	TargetID  uint64
	SkillID   uint32
	Value     int32
	StateID   uint32
}

type ActorState struct {
	ActorID uint64
	HP      uint32
	HPMax   uint32
	Dead    bool
}

type StateSnapshot struct {
	BattleID      uint64
	BattleVersion uint32
	Round         uint32
	Events        []Event
	Actors        []ActorState
	ActiveActorID uint64
	ActivePetUID  uint64
}

type ResultSnapshot struct {
	BattleID      uint64
	ActivePetUID  uint64
	ActivePetHP   uint32
	Win           bool
	ReturnSceneID uint32
	ReturnPos     world.Vec2i
	Reason        string
}

type ActionRequest struct {
	BattleID   uint64
	Round      uint32
	ActionType uint32
	ActorID    uint64
	SkillID    uint32
	TargetID   uint64
}

type ActionOutcome struct {
	Response BattleActionResponse
	State    *StateSnapshot
	Result   *ResultSnapshot
}

type BattleActionResponse struct {
	Accepted bool
	Reason   string
}
