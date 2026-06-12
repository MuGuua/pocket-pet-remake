package battle

import (
	"errors"

	"pocket-pet-remake/server/internal/module/world"
)

const (
	BattleTypePVE uint32 = 1
	BattleTypePVP uint32 = 2

	ActionTypeSkill   uint32 = 1
	ActionTypeEscape  uint32 = 4
	ActionTypeSetAuto uint32 = 5
	ActionTypeCapture uint32 = 6

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
	EventTypeCapture     uint32 = 12

	PlayerActorType uint32 = 1
	EnemyActorType  uint32 = 2

	// Actor unit classes distinguish the source body behind one battle actor.
	// They are intentionally separate from ActorType because "which side am I
	// on" and "am I a character / pet / mercenary / monster" answer different
	// combat questions such as source-specific resistance.
	ActorUnitClassCharacter uint32 = 1
	ActorUnitClassPet       uint32 = 2
	ActorUnitClassMercenary uint32 = 3
	ActorUnitClassMonster   uint32 = 4

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

	DefaultAttackSkillID    uint32 = 1001
	DefaultCharacterSkillID uint32 = 1101
	DefaultEnemyPetID       uint32 = 9001
	DefaultEnemySkillID     uint32 = 90001

	PhaseCommand  string = "command"
	PhaseFinished string = "finished"
)

var (
	ErrBattleAlreadyActive = errors.New("battle already active")
	ErrBattleNotFound      = errors.New("battle not found")
	ErrInvalidAction       = errors.New("invalid battle action")
	ErrNoLineupAvailable   = errors.New("no lineup available")
	ErrTargetUnavailable   = errors.New("target unavailable")
	ErrChallengeNotFound   = errors.New("pvp challenge not found")
	ErrChallengeExpired    = errors.New("pvp challenge expired")
	ErrChallengeInvalid    = errors.New("pvp challenge invalid")
	ErrWildEncounterUnavailable = errors.New("wild encounter unavailable")
)

type ActorSnapshot struct {
	ActorID            uint64
	ActorType          uint32
	UnitClass          uint32
	OwnerPlayerID      uint64
	PetUID             uint64
	PetID              uint32
	Name               string
	HP                 uint32
	HPMax              uint32
	Energy             uint32
	EnergyMax          uint32
	ATK                uint32
	DEF                uint32
	SPD                uint32
	MANA               uint32
	HitPct             uint32
	DodgePct           uint32
	CritRatePct        uint32
	CritDmgPct         uint32
	PhysicalResistPct  uint32
	SkillResistPct     uint32
	ConfusionResistPct uint32
	SleepResistPct     uint32
	ParalysisResistPct uint32
	SealResistPct      uint32
	CurseResistPct     uint32
	CritResistPct      uint32
	CritDmgResistPct   uint32
	CharacterResistPct uint32
	PetResistPct       uint32
	MercenaryResistPct uint32
	GenericShieldPct   uint32
	Skills             []SkillSnapshot
	SkillIDs           []uint32
	StatusIDs          []uint32
	LineupIndex        uint32
}

type SkillSnapshot struct {
	SkillID      uint32
	Name         string
	TargetType   string
	TargetCount  uint32
	AnimationKey string
	CastColor    string
	ImpactColor  string
	Projectile   bool
}

type StartSnapshot struct {
	BattleID             uint64
	BattleType           uint32
	BattleVersion        uint32
	Frame                uint32
	ParticipantPlayerIDs []uint64
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
	Energy     uint32
	EnergyMax  uint32
	Dead       bool
	CanAct     bool
	StatusIDs  []uint32
	ChargeDone bool
}

type StateSnapshot struct {
	BattleID             uint64
	BattleVersion        uint32
	Frame                uint32
	ParticipantPlayerIDs []uint64
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
	PetUID    uint64
	HP        uint32
	ExpGained uint64
}

// DropReward 描述一条服务端权威掉落结果。
// 当前先只保留 item_id 与数量，掉落名称统一以后续数据库模板或发奖结果为准。
type DropReward struct {
	ItemID   uint64
	Quantity uint64
}

type ResultSnapshot struct {
	BattleID             uint64
	BattleType           uint32
	ParticipantPlayerIDs []uint64
	PetResults           []PetResult
	Win                  bool
	ReturnSceneID        uint32
	ReturnPos            world.Vec2i
	Reason               string
	RewardGold           uint32
	RewardPlayerExp      uint64
	DropItems            []DropReward
	DropTexts            []string
	CaptureSuccess       bool
	CaptureMonsterID     uint32
	CapturedPetID        uint32
}

type ActionRequest struct {
	BattleID          uint64
	Round             uint32
	ActionType        uint32
	ActorID           uint64
	SkillID           uint32
	TargetID          uint64
	ItemID            uint32
	BagSlotIndex      uint32
	AutoBattleEnabled bool
}

type ActionOutcome struct {
	Response BattleActionResponse
	State    *StateSnapshot
	Result   *ResultSnapshot
}

type BattleActionResponse struct {
	Accepted       bool
	Reason         string
	CaptureSuccess bool
}
