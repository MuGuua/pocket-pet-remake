package battle

import (
	"errors"

	"pocket-pet-remake/server/internal/module/world"
)

const (
	BattleTypePVE uint32 = 1

	ActionTypeSkill   uint32 = 1
	ActionTypeEscape  uint32 = 4
	ActionTypeSetAuto uint32 = 5

	EventTypeUseSkill    uint32 = 1
	EventTypeDamage      uint32 = 2
	EventTypeHeal        uint32 = 3
	EventTypeApplyStatus uint32 = 4
	EventTypeStatusTick  uint32 = 5
	EventTypeSkipTurn    uint32 = 6
	EventTypeDefeat      uint32 = 7
	EventTypeDodge       uint32 = 8
	EventTypeCounter     uint32 = 9
	EventTypeRevive      uint32 = 10
	EventTypeCombo       uint32 = 11

	PlayerActorType uint32 = 1
	EnemyActorType  uint32 = 2

	StatusBleed         uint32 = 1
	StatusSeal          uint32 = 2
	StatusStun          uint32 = 3
	StatusVulnerability uint32 = 4
	StatusArmorBreak    uint32 = 5
	StatusSlow          uint32 = 6
	StatusCritBoost     uint32 = 7
	StatusCurse         uint32 = 8
	StatusBind          uint32 = 9
	StatusSleep         uint32 = 10
	StatusParalysis     uint32 = 11
	StatusConfusion     uint32 = 12

	DefaultAttackSkillID uint32 = 1001
	DefaultEnemyPetID    uint32 = 9001
	DefaultEnemySkillID  uint32 = 90001

	PhaseCommand  string = "command"
	PhaseFinished string = "finished"
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
	ATK         uint32
	DEF         uint32
	SPD         uint32
	Skills      []SkillSnapshot
	SkillIDs    []uint32
	StatusIDs   []uint32
	LineupIndex uint32
}

type SkillSnapshot struct {
	SkillID    uint32
	Name       string
	TargetType string
}

type StartSnapshot struct {
	BattleID             uint64
	BattleType           uint32
	BattleVersion        uint32
	Allies               []ActorSnapshot
	Enemies              []ActorSnapshot
	Round                uint32
	Phase                string
	ActiveActorID        uint64
	ActivePetUID         uint64
	CommandDeadlineMS    int64
	AutoBattleEnabled    bool
	PendingActorIDs      []uint64
	ControllableActorIDs []uint64
}

type Event struct {
	EventType uint32
	SourceID  uint64
	TargetID  uint64
	SkillID   uint32
	Value     int32
	StateID   uint32
	Label     string
}

type ActorState struct {
	ActorID    uint64
	HP         uint32
	HPMax      uint32
	Dead       bool
	CanAct     bool
	StatusIDs  []uint32
	ChargeDone bool
}

type StateSnapshot struct {
	BattleID             uint64
	BattleVersion        uint32
	Round                uint32
	Phase                string
	Events               []Event
	Actors               []ActorState
	ActiveActorID        uint64
	ActivePetUID         uint64
	CommandDeadlineMS    int64
	AutoBattleEnabled    bool
	PendingActorIDs      []uint64
	ControllableActorIDs []uint64
}

type PetResult struct {
	PetUID uint64
	HP     uint32
}

type ResultSnapshot struct {
	BattleID      uint64
	PetResults    []PetResult
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
	AutoBattleEnabled bool
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
