package battle

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"pocket-pet-remake/server/internal/module/monster"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/world"
)

type Service struct {
	mu                sync.Mutex
	nextBattleID      uint64
	nextChallengeID   uint64
	activeByPlayer    map[uint64]*activeBattle
	pendingChallenges map[uint64]PVPChallenge
	monsterService    *monster.Service
}

// AutoProgressOutcome carries one server-side auto progression result together
// with the owning player, so higher layers can decide whether to push packets,
// persist rewards, or both.
type AutoProgressOutcome struct {
	PlayerID uint64
	Outcome  *ActionOutcome
}

type PVPChallenge struct {
	ChallengeID        uint64
	ChallengerPlayerID uint64
	DefenderPlayerID   uint64
	CreatedAt          time.Time
	ExpiresAt          time.Time
}

type activeBattle struct {
	battleID             uint64
	battleType           uint32
	battleVersion        uint32
	round                uint32
	phase                string
	playerID             uint64
	participantPlayerIDs []uint64
	returnSceneID        uint32
	returnPos            world.Vec2i
	allies               []*actorRuntime
	enemies              []*actorRuntime
	pendingActors        []uint64
	plannedActs          map[uint64]ActionRequest
	autoBattleEnabled    bool
	commandDeadline      time.Time
	stateHistory         []StateSnapshot
}

type actorRuntime struct {
	actorID       uint64
	actorType     uint32
	unitClass     uint32
	ownerPlayerID uint64
	petUID        uint64
	petID         uint32
	lineupIndex   uint32
	name          string
	level         uint32
	hp            uint32
	hpMax         uint32
	energy        uint32
	energyMax     uint32
	atk           uint32
	def           uint32
	spd           uint32
	mana          uint32
	hitPct        uint32
	dodgeRatePct  uint32
	skillIDs      []uint32
	critRatePct   uint32
	critDmgPct    uint32
	statuses      map[uint32]*statusRuntime

	// Resistance fields live directly on the authoritative runtime so damage,
	// status, and crit branches can all consume one shared source-of-truth
	// instead of rehydrating separate config objects during every action.
	physicalResistPct  uint32
	skillResistPct     uint32
	confusionResistPct uint32
	sleepResistPct     uint32
	paralysisResistPct uint32
	sealResistPct      uint32
	curseResistPct     uint32
	critResistPct      uint32
	critDmgResistPct   uint32
	characterResistPct uint32
	petResistPct       uint32
	mercenaryResistPct uint32
	genericShieldPct   uint32

	// These runtime modifier fields are kept on the actor so future status and
	// passive systems can change battle math without mutating base pet data.
	globalMultiplierPct      uint32
	attackMultiplierPct      uint32
	defenseMultiplierPct     uint32
	speedMultiplierPct       uint32
	manaMultiplierPct        uint32
	attackFlatBonus          int32
	defenseFlatBonus         int32
	speedFlatBonus           int32
	manaFlatBonus            int32
	statusVulnerabilityPct   uint32
	statusArmorBroken        bool
	statusSpeedMultiplierPct uint32
	statusCritRateBonusPct   uint32

	// Passive fields are runtime-owned so later pet templates, buffs, or gear
	// can modify them without changing the combat flow entry points again.
	dodgePct      uint32
	lifestealPct  uint32
	counterPct    uint32
	comboPct      uint32
	revivePct     uint32
	reviveHPPct   uint32
	controlImmune bool
	reviveUsed    bool
}

type statusRuntime struct {
	statusID       uint32
	remainingRound uint32
	potency        int32
}

// combatAttributeTemplate centralizes the non-persistent runtime-facing battle
// numbers for one actor archetype. We keep it separate from actorRuntime so
// future DB/config loading can fill the same shape without rewriting builders.
type combatAttributeTemplate struct {
	HP          uint32
	Energy      uint32
	Attack      uint32
	Defense     uint32
	Speed       uint32
	Mana        uint32
	HitPct      uint32
	DodgePct    uint32
	CritRatePct uint32
	CritDmgPct  uint32
}

// resistanceTemplate collects all incoming-damage and incoming-status
// mitigation knobs that the user requested for character and enemy runtimes.
type resistanceTemplate struct {
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
}

type turnDecision struct {
	actor  *actorRuntime
	action ActionRequest
	tie    int64
}

const commandTimeout = 15 * time.Second
const battleReplayHistoryLimit = 12

func NewService(monsterService *monster.Service) *Service {
	return &Service{
		nextBattleID:      70000,
		nextChallengeID:   90000,
		activeByPlayer:    make(map[uint64]*activeBattle),
		pendingChallenges: make(map[uint64]PVPChallenge),
		monsterService:    monsterService,
	}
}

const pvpChallengeTimeout = 30 * time.Second

func (s *Service) StartPVE(ctx context.Context, profile *player.Profile, lineup []pet.LineupPet, enemy world.Entity) (*StartSnapshot, error) {
	if profile == nil {
		return nil, ErrTargetUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.activeByPlayer[profile.PlayerID]; exists {
		return nil, ErrBattleAlreadyActive
	}

	s.nextBattleID++
	battleID := s.nextBattleID
	battle := &activeBattle{
		battleID:             battleID,
		battleType:           BattleTypePVE,
		battleVersion:        1,
		round:                1,
		phase:                PhaseCommand,
		playerID:             profile.PlayerID,
		participantPlayerIDs: []uint64{profile.PlayerID},
		returnSceneID:        profile.SceneID,
		returnPos:            world.Vec2i{X: profile.PosX, Y: profile.PosY},
		// 单人 PVE 现在始终由人物作为我方权威入口开战，宠物编队再按原顺序追加，
		// 这样客户端和服务端都能围绕同一个人物 actor 组织后续动作与表现。
		allies:      buildSoloPVEAllies(profile, lineup),
		enemies:     s.buildEnemyTeam(ctx, profile, enemy),
		plannedActs: make(map[uint64]ActionRequest),
	}
	battle.pendingActors = battle.collectPendingControllableActors()
	battle.resetCommandDeadline()
	if len(battle.pendingActors) == 0 {
		return nil, ErrNoLineupAvailable
	}

	s.activeByPlayer[profile.PlayerID] = battle
	return battle.toStartSnapshot(), nil
}

// StartPVEWildEncounter 在客户端暗雷上报通过后，按 scene_id 解析刷怪并开启 PVE。
func (s *Service) StartPVEWildEncounter(ctx context.Context, profile *player.Profile, lineup []pet.LineupPet, encounter *monster.RuntimeWildEncounter) (*StartSnapshot, error) {
	if profile == nil || encounter == nil || len(encounter.Slots) == 0 {
		return nil, ErrWildEncounterUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.activeByPlayer[profile.PlayerID]; exists {
		return nil, ErrBattleAlreadyActive
	}

	virtualEnemy := world.Entity{
		EntityID:   monster.WildEncounterEntityID(encounter.SceneID),
		EntityType: ActorUnitClassMonster,
		Name:       encounter.EncounterName,
	}

	s.nextBattleID++
	battleID := s.nextBattleID
	battle := &activeBattle{
		battleID:             battleID,
		battleType:           BattleTypePVE,
		battleVersion:        1,
		round:                1,
		phase:                PhaseCommand,
		playerID:             profile.PlayerID,
		participantPlayerIDs: []uint64{profile.PlayerID},
		returnSceneID:        profile.SceneID,
		returnPos:            world.Vec2i{X: profile.PosX, Y: profile.PosY},
		allies:               buildSoloPVEAllies(profile, lineup),
		enemies:              s.buildEnemyTeamFromSlots(ctx, profile, virtualEnemy, encounter.Slots),
		plannedActs:          make(map[uint64]ActionRequest),
	}
	battle.pendingActors = battle.collectPendingControllableActors()
	battle.resetCommandDeadline()
	if len(battle.pendingActors) == 0 {
		return nil, ErrNoLineupAvailable
	}

	s.activeByPlayer[profile.PlayerID] = battle
	return battle.toStartSnapshot(), nil
}

// StartPVEWildEncounterByScene 校验 scene_id 对应暗雷配置后开战。
func (s *Service) StartPVEWildEncounterByScene(ctx context.Context, profile *player.Profile, lineup []pet.LineupPet, sceneID uint32) (*StartSnapshot, error) {
	if s.monsterService == nil || sceneID == 0 {
		return nil, ErrWildEncounterUnavailable
	}
	encounter, err := s.monsterService.ResolveWildEncounterForScene(ctx, sceneID)
	if err != nil {
		return nil, err
	}
	return s.StartPVEWildEncounter(ctx, profile, lineup, encounter)
}

// StartPVP creates one shared authoritative battle for two online players.
// The challenger controls the ally side and the defender controls the enemy
// side, but both sides are still fully player-authored rather than AI.
func (s *Service) StartPVP(_ context.Context, challenger *player.Profile, challengerLineup []pet.LineupPet, defender *player.Profile, defenderLineup []pet.LineupPet) (*StartSnapshot, error) {
	if challenger == nil || defender == nil {
		return nil, ErrTargetUnavailable
	}
	if len(challengerLineup) == 0 || len(defenderLineup) == 0 {
		return nil, ErrNoLineupAvailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.activeByPlayer[challenger.PlayerID]; exists {
		return nil, ErrBattleAlreadyActive
	}
	if _, exists := s.activeByPlayer[defender.PlayerID]; exists {
		return nil, ErrBattleAlreadyActive
	}

	s.nextBattleID++
	battleID := s.nextBattleID
	battle := &activeBattle{
		battleID:             battleID,
		battleType:           BattleTypePVP,
		battleVersion:        1,
		round:                1,
		phase:                PhaseCommand,
		playerID:             challenger.PlayerID,
		participantPlayerIDs: []uint64{challenger.PlayerID, defender.PlayerID},
		returnSceneID:        challenger.SceneID,
		returnPos:            world.Vec2i{X: challenger.PosX, Y: challenger.PosY},
		allies:               buildPlayerTeam(challenger, challengerLineup, PlayerActorType),
		enemies:              buildPlayerTeam(defender, defenderLineup, EnemyActorType),
		plannedActs:          make(map[uint64]ActionRequest),
	}
	battle.pendingActors = battle.collectPendingControllableActors()
	battle.resetCommandDeadline()
	if len(battle.pendingActors) == 0 {
		return nil, ErrNoLineupAvailable
	}

	s.activeByPlayer[challenger.PlayerID] = battle
	s.activeByPlayer[defender.PlayerID] = battle
	return battle.toStartSnapshot(), nil
}

func (s *Service) CreatePVPChallenge(_ context.Context, challengerPlayerID uint64, defenderPlayerID uint64) (*PVPChallenge, error) {
	if challengerPlayerID == 0 || defenderPlayerID == 0 || challengerPlayerID == defenderPlayerID {
		return nil, ErrChallengeInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.activeByPlayer[challengerPlayerID]; exists {
		return nil, ErrBattleAlreadyActive
	}
	if _, exists := s.activeByPlayer[defenderPlayerID]; exists {
		return nil, ErrBattleAlreadyActive
	}
	now := time.Now()
	for challengeID, challenge := range s.pendingChallenges {
		if now.After(challenge.ExpiresAt) {
			delete(s.pendingChallenges, challengeID)
		}
		if challenge.ChallengerPlayerID == challengerPlayerID && challenge.DefenderPlayerID == defenderPlayerID {
			copy := challenge
			return &copy, nil
		}
	}

	s.nextChallengeID++
	challenge := PVPChallenge{
		ChallengeID:        s.nextChallengeID,
		ChallengerPlayerID: challengerPlayerID,
		DefenderPlayerID:   defenderPlayerID,
		CreatedAt:          now,
		ExpiresAt:          now.Add(pvpChallengeTimeout),
	}
	s.pendingChallenges[challenge.ChallengeID] = challenge
	return &challenge, nil
}

func (s *Service) ResolvePVPChallenge(_ context.Context, challengeID uint64, defenderPlayerID uint64, accept bool) (*PVPChallenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	challenge, ok := s.pendingChallenges[challengeID]
	if !ok {
		return nil, ErrChallengeNotFound
	}
	delete(s.pendingChallenges, challengeID)
	if challenge.DefenderPlayerID != defenderPlayerID {
		return nil, ErrChallengeInvalid
	}
	if time.Now().After(challenge.ExpiresAt) {
		return nil, ErrChallengeExpired
	}
	if !accept {
		copy := challenge
		return &copy, nil
	}
	if _, exists := s.activeByPlayer[challenge.ChallengerPlayerID]; exists {
		return nil, ErrBattleAlreadyActive
	}
	if _, exists := s.activeByPlayer[challenge.DefenderPlayerID]; exists {
		return nil, ErrBattleAlreadyActive
	}
	copy := challenge
	return &copy, nil
}

func (s *Service) SubmitAction(_ context.Context, playerID uint64, request ActionRequest) (*ActionOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	battle, ok := s.activeByPlayer[playerID]
	if !ok {
		return nil, ErrBattleNotFound
	}
	if battle.battleID != request.BattleID || battle.round != request.Round || battle.phase != PhaseCommand {
		return nil, ErrInvalidAction
	}

	switch request.ActionType {
	case ActionTypeSkill:
		return s.queueActionLocked(playerID, battle, request)
	case ActionTypeEscape:
		result := battle.buildResult(false, "player escaped battle")
		s.removeBattleLocked(battle)
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: true, Reason: "escape accepted"},
			Result:   result,
		}, nil
	case ActionTypeSetAuto:
		return s.setAutoBattleLocked(playerID, battle, request)
	case ActionTypeCapture:
		return s.submitCaptureLocked(playerID, battle, request)
	default:
		return nil, ErrInvalidAction
	}
}

func (s *Service) ProgressAuto(_ context.Context, playerID uint64) (*ActionOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	battle, ok := s.activeByPlayer[playerID]
	if !ok || battle.phase != PhaseCommand {
		return nil, nil
	}
	if !battle.shouldAutoResolve(time.Now()) {
		return nil, nil
	}
	state, result := s.autoResolvePendingLocked(playerID, battle)
	return &ActionOutcome{
		Response: BattleActionResponse{Accepted: true, Reason: "server auto resolved pending actions"},
		State:    state,
		Result:   result,
	}, nil
}

// GetActiveSnapshot returns a full reconnect-friendly snapshot of the current
// authoritative battle, including both the actor roster and the latest command
// phase state. Callers should treat a nil result as "player is not in battle".
func (s *Service) GetActiveSnapshot(_ context.Context, playerID uint64) (*StartSnapshot, *StateSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.activeByPlayer[playerID]
	if !ok {
		return nil, nil, false
	}
	return current.toStartSnapshot(), current.toStateSnapshot(nil), true
}

// GetReplaySnapshots returns recent authoritative state snapshots newer than
// the client-reported frame for the same active battle.
func (s *Service) GetReplaySnapshots(_ context.Context, playerID uint64, battleID uint64, lastFrame uint32) []StateSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.activeByPlayer[playerID]
	if !ok || battleID == 0 || current.battleID != battleID {
		return nil
	}
	result := make([]StateSnapshot, 0, len(current.stateHistory))
	for _, item := range current.stateHistory {
		if item.Frame <= lastFrame {
			continue
		}
		result = append(result, cloneStateSnapshot(item))
	}
	return result
}

// EnableAutoForPlayer flips the active battle into server custody mode. The
// next heartbeat or background sweep will pick up any remaining pending actors.
func (s *Service) EnableAutoForPlayer(_ context.Context, playerID uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	battle, ok := s.activeByPlayer[playerID]
	if !ok || battle.phase != PhaseCommand {
		return false
	}
	if battle.autoBattleEnabled {
		return true
	}
	battle.autoBattleEnabled = true
	battle.commandDeadline = time.Time{}
	battle.battleVersion++
	return true
}

// ResolveDisconnect keeps PVE and PVP disconnect handling separated: PVE can
// still move into AI custody, while the current minimal PVP skeleton ends
// immediately and treats the disconnected side as the loser.
func (s *Service) ResolveDisconnect(_ context.Context, playerID uint64) *ResultSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.activeByPlayer[playerID]
	if !ok || current.battleType != BattleTypePVP {
		return nil
	}
	win := !current.isPlayerOnAllySide(playerID)
	result := current.buildResult(win, "player disconnected")
	s.removeBattleLocked(current)
	return result
}

// ProgressAutoAll scans every active battle once so disconnected players can
// still be progressed by a background server loop without requiring heartbeats.
func (s *Service) ProgressAutoAll(_ context.Context) []AutoProgressOutcome {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	outcomes := make([]AutoProgressOutcome, 0)
	for playerID, battle := range s.activeByPlayer {
		if battle.phase != PhaseCommand || !battle.shouldAutoResolve(now) {
			continue
		}
		state, result := s.autoResolvePendingLocked(playerID, battle)
		outcomes = append(outcomes, AutoProgressOutcome{
			PlayerID: playerID,
			Outcome: &ActionOutcome{
				Response: BattleActionResponse{Accepted: true, Reason: "server auto resolved pending actions"},
				State:    state,
				Result:   result,
			},
		})
	}
	return outcomes
}

func (s *Service) queueActionLocked(playerID uint64, battle *activeBattle, request ActionRequest) (*ActionOutcome, error) {
	actor := battle.findActor(request.ActorID)
	if actor == nil || actor.isDead() || actor.ownerPlayerID != playerID {
		return nil, ErrInvalidAction
	}
	if !battle.isPendingActor(actor.actorID) {
		return nil, ErrInvalidAction
	}

	skillID := request.SkillID
	if skillID == 0 {
		skillID = DefaultAttackSkillID
	}
	if !actor.hasSkill(skillID) {
		skillID = DefaultAttackSkillID
		if !actor.hasSkill(skillID) {
			return nil, ErrInvalidAction
		}
	}
	request.SkillID = skillID
	request.TargetID = battle.normalizeRequestedTarget(actor, request.TargetID, skillID)
	battle.plannedActs[actor.actorID] = request
	battle.pendingActors = removeActorID(battle.pendingActors, actor.actorID)

	if len(battle.pendingActors) > 0 {
		battle.resetCommandDeadline()
		if battle.autoBattleEnabled {
			state, result := s.autoResolvePendingLocked(playerID, battle)
			return &ActionOutcome{
				Response: BattleActionResponse{Accepted: true, Reason: "action accepted"},
				State:    state,
				Result:   result,
			}, nil
		}
		battle.battleVersion++
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: true, Reason: "action queued"},
			State:    battle.recordStateSnapshot(nil),
		}, nil
	}

	state, result := battle.resolveRound()
	if result != nil {
		s.removeBattleLocked(battle)
	}
	return &ActionOutcome{
		Response: BattleActionResponse{Accepted: true, Reason: "action accepted"},
		State:    state,
		Result:   result,
	}, nil
}

func (s *Service) setAutoBattleLocked(playerID uint64, battle *activeBattle, request ActionRequest) (*ActionOutcome, error) {
	battle.autoBattleEnabled = request.AutoBattleEnabled
	if battle.autoBattleEnabled && battle.phase == PhaseCommand && len(battle.pendingActors) > 0 {
		state, result := s.autoResolvePendingLocked(playerID, battle)
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: true, Reason: "auto battle enabled"},
			State:    state,
			Result:   result,
		}, nil
	}
	battle.resetCommandDeadline()
	battle.battleVersion++
	return &ActionOutcome{
		Response: BattleActionResponse{Accepted: true, Reason: "auto battle updated"},
		State:    battle.recordStateSnapshot(nil),
	}, nil
}

// submitCaptureLocked 处理 PVE 战斗中的捕捉尝试：服务端判定道具、目标 HP 与成功率，不继承战斗数值。
func (s *Service) submitCaptureLocked(playerID uint64, battle *activeBattle, request ActionRequest) (*ActionOutcome, error) {
	if battle.battleType != BattleTypePVE {
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: false, Reason: "capture only allowed in pve"},
		}, nil
	}
	actor := battle.findActor(request.ActorID)
	if actor == nil || actor.isDead() || actor.ownerPlayerID != playerID || actor.actorType != PlayerActorType {
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: false, Reason: "invalid capture actor"},
		}, nil
	}
	if !battle.isPendingActor(actor.actorID) {
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: false, Reason: "actor cannot act now"},
		}, nil
	}
	target := battle.findActor(request.TargetID)
	if target == nil || target.isDead() || target.actorType != EnemyActorType {
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: false, Reason: "invalid capture target"},
		}, nil
	}
	if request.ItemID == 0 || request.BagSlotIndex == 0 {
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: false, Reason: "capture item required"},
		}, nil
	}
	if s.monsterService == nil {
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: false, Reason: "capture unavailable"},
		}, nil
	}
	config, err := s.monsterService.GetCaptureConfig(context.Background(), target.petID)
	if err != nil || config == nil || !config.IsCapturable || config.CapturePetID == 0 {
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: false, Reason: "target not capturable"},
		}, nil
	}
	if !captureItemAllowed(config.CaptureItemIDs, request.ItemID) {
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: false, Reason: "capture item not allowed"},
		}, nil
	}
	if target.hpMax == 0 {
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: false, Reason: "invalid target hp"},
		}, nil
	}
	hpPct := target.hp * 100 / target.hpMax
	if hpPct > config.CaptureMinHPPct {
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: false, Reason: "target hp too high"},
		}, nil
	}

	rateBase := config.CaptureRateBase
	if rateBase == 0 {
		rateBase = 5000
	}
	if rateBase > 10000 {
		rateBase = 10000
	}
	roll := rand.Intn(10000)
	if roll >= int(rateBase) {
		battle.pendingActors = removeActorID(battle.pendingActors, actor.actorID)
		events := []Event{{
			EventType: EventTypeCapture,
			SourceID:  actor.actorID,
			TargetID:  target.actorID,
			Value:     0,
			Label:     fmt.Sprintf("%s 使用捕捉道具失败。", actor.name),
		}}
		if len(battle.pendingActors) > 0 {
			battle.resetCommandDeadline()
			battle.battleVersion++
			return &ActionOutcome{
				Response: BattleActionResponse{Accepted: true, Reason: "capture failed"},
				State:    battle.recordStateSnapshot(events),
			}, nil
		}
		state, result := battle.resolveRound()
		if len(events) > 0 && state != nil {
			state.Events = append(events, state.Events...)
		}
		if result != nil {
			s.removeBattleLocked(battle)
		}
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: true, Reason: "capture failed"},
			State:    state,
			Result:   result,
		}, nil
	}

	result := battle.buildCaptureSuccessResult(target.petID, config.CapturePetID)
	s.removeBattleLocked(battle)
	return &ActionOutcome{
		Response: BattleActionResponse{
			Accepted:       true,
			Reason:         "capture success",
			CaptureSuccess: true,
		},
		Result: result,
	}, nil
}

func captureItemAllowed(allowed []uint32, itemID uint32) bool {
	for _, current := range allowed {
		if current == itemID {
			return true
		}
	}
	return false
}

func (b *activeBattle) buildCaptureSuccessResult(monsterID uint32, petID uint32) *ResultSnapshot {
	result := b.buildResult(true, "capture success")
	result.RewardGold = 0
	result.RewardPlayerExp = 0
	result.DropItems = nil
	result.DropTexts = nil
	for index := range result.PetResults {
		result.PetResults[index].ExpGained = 0
	}
	result.CaptureSuccess = true
	result.CaptureMonsterID = monsterID
	result.CapturedPetID = petID
	return result
}

func (s *Service) autoResolvePendingLocked(playerID uint64, battle *activeBattle) (*StateSnapshot, *ResultSnapshot) {
	for _, actorID := range append([]uint64{}, battle.pendingActors...) {
		actor := battle.findActor(actorID)
		if actor == nil || actor.isDead() {
			continue
		}
		battle.plannedActs[actorID] = battle.defaultActionFor(actor)
	}
	battle.pendingActors = nil
	state, result := battle.resolveRound()
	if result != nil {
		s.removeBattleLocked(battle)
	}
	return state, result
}

func (s *Service) removeBattleLocked(current *activeBattle) {
	if current == nil {
		return
	}
	for _, participantPlayerID := range current.participantPlayerIDs {
		delete(s.activeByPlayer, participantPlayerID)
	}
}

// defaultPlayerCharacterTemplate prepares the first server-side solo-character
// combat profile. It deliberately ignores level growth for now, matching the
// user's request to land the base stat system before the level curve.
func defaultPlayerCharacterTemplate() (combatAttributeTemplate, resistanceTemplate) {
	return combatAttributeTemplate{
			HP:          120,
			Energy:      100,
			Attack:      24,
			Defense:     12,
			Speed:       18,
			Mana:        20,
			HitPct:      10,
			DodgePct:    6,
			CritRatePct: 10,
			CritDmgPct:  155,
		}, resistanceTemplate{
			PhysicalResistPct:  6,
			SkillResistPct:     4,
			ConfusionResistPct: 8,
			SleepResistPct:     8,
			ParalysisResistPct: 6,
			SealResistPct:      6,
			CurseResistPct:     5,
			CritResistPct:      4,
			CritDmgResistPct:   10,
			PetResistPct:       4,
			GenericShieldPct:   2,
		}
}

// combatTemplateFromProfile converts persistent player attributes into the
// battle-layer template. Any zero-valued field still falls back to the current
// starter template so older fixtures or partially migrated rows do not break
// combat assembly.
func combatTemplateFromProfile(profile *player.Profile) (combatAttributeTemplate, resistanceTemplate) {
	defaultAttributes, defaultResistances := defaultPlayerCharacterTemplate()
	if profile == nil {
		return defaultAttributes, defaultResistances
	}
	attributes := combatAttributeTemplate{
		HP:          profile.HP,
		Energy:      profile.Energy,
		Attack:      profile.ATK,
		Defense:     profile.DEF,
		Speed:       profile.SPD,
		Mana:        profile.MANA,
		HitPct:      profile.HitPct,
		DodgePct:    profile.DodgePct,
		CritRatePct: profile.CritRatePct,
		CritDmgPct:  profile.CritDmgPct,
	}
	resistances := resistanceTemplate{
		PhysicalResistPct:  profile.PhysicalResistPct,
		SkillResistPct:     profile.SkillResistPct,
		ConfusionResistPct: profile.ConfusionResistPct,
		SleepResistPct:     profile.SleepResistPct,
		ParalysisResistPct: profile.ParalysisResistPct,
		SealResistPct:      profile.SealResistPct,
		CurseResistPct:     profile.CurseResistPct,
		CritResistPct:      profile.CritResistPct,
		CritDmgResistPct:   profile.CritDmgResistPct,
		CharacterResistPct: profile.CharacterResistPct,
		PetResistPct:       profile.PetResistPct,
		MercenaryResistPct: profile.MercenaryResistPct,
		GenericShieldPct:   profile.GenericShieldPct,
	}
	if attributes.HP == 0 {
		attributes.HP = defaultAttributes.HP
	}
	if profile.HPMax > 0 && profile.HPMax > attributes.HP {
		attributes.HP = profile.HPMax
	}
	if attributes.Energy == 0 {
		attributes.Energy = maxUint32(profile.EnergyMax, defaultAttributes.Energy)
	}
	if attributes.Attack == 0 {
		attributes.Attack = defaultAttributes.Attack
	}
	if attributes.Defense == 0 {
		attributes.Defense = defaultAttributes.Defense
	}
	if attributes.Speed == 0 {
		attributes.Speed = defaultAttributes.Speed
	}
	if attributes.Mana == 0 {
		attributes.Mana = defaultAttributes.Mana
	}
	if attributes.HitPct == 0 {
		attributes.HitPct = defaultAttributes.HitPct
	}
	if attributes.DodgePct == 0 {
		attributes.DodgePct = defaultAttributes.DodgePct
	}
	if attributes.CritRatePct == 0 {
		attributes.CritRatePct = defaultAttributes.CritRatePct
	}
	if attributes.CritDmgPct == 0 {
		attributes.CritDmgPct = defaultAttributes.CritDmgPct
	}
	if resistances.PhysicalResistPct == 0 {
		resistances.PhysicalResistPct = defaultResistances.PhysicalResistPct
	}
	if resistances.SkillResistPct == 0 {
		resistances.SkillResistPct = defaultResistances.SkillResistPct
	}
	if resistances.ConfusionResistPct == 0 {
		resistances.ConfusionResistPct = defaultResistances.ConfusionResistPct
	}
	if resistances.SleepResistPct == 0 {
		resistances.SleepResistPct = defaultResistances.SleepResistPct
	}
	if resistances.ParalysisResistPct == 0 {
		resistances.ParalysisResistPct = defaultResistances.ParalysisResistPct
	}
	if resistances.SealResistPct == 0 {
		resistances.SealResistPct = defaultResistances.SealResistPct
	}
	if resistances.CurseResistPct == 0 {
		resistances.CurseResistPct = defaultResistances.CurseResistPct
	}
	if resistances.CritResistPct == 0 {
		resistances.CritResistPct = defaultResistances.CritResistPct
	}
	if resistances.CritDmgResistPct == 0 {
		resistances.CritDmgResistPct = defaultResistances.CritDmgResistPct
	}
	if resistances.PetResistPct == 0 {
		resistances.PetResistPct = defaultResistances.PetResistPct
	}
	if resistances.GenericShieldPct == 0 {
		resistances.GenericShieldPct = defaultResistances.GenericShieldPct
	}
	return attributes, resistances
}

// defaultPetTemplate keeps the current pet battle flow working while migrating
// it onto the richer character/enemy attribute model introduced in this task.
func defaultPetTemplate(item pet.LineupPet) (combatAttributeTemplate, resistanceTemplate) {
	return combatAttributeTemplate{
			HP:          item.HPMax,
			Energy:      100,
			Attack:      item.ATK,
			Defense:     item.DEF,
			Speed:       item.SPD,
			Mana:        item.MANA,
			HitPct:      8,
			DodgePct:    5,
			CritRatePct: 8,
			CritDmgPct:  150,
		}, resistanceTemplate{
			PhysicalResistPct:  3,
			SkillResistPct:     2,
			ConfusionResistPct: 4,
			SleepResistPct:     4,
			ParalysisResistPct: 4,
			SealResistPct:      4,
			CurseResistPct:     3,
			CritResistPct:      2,
			CritDmgResistPct:   8,
			PetResistPct:       3,
		}
}

// defaultEnemyTemplate gives monsters the same stat dimensions as characters
// and pets, but still keeps their numbers deterministic and easy to tune.
func defaultEnemyTemplate(profile *player.Profile, enemy world.Entity, index int) (combatAttributeTemplate, resistanceTemplate, uint32) {
	baseHP := uint32(18)
	baseAttack := uint32(10)
	baseDefense := uint32(8)
	baseSpeed := uint32(7)
	baseMana := uint32(9)
	baseLevel := uint32(1)
	if profile != nil {
		baseHP += profile.Level * 4
		baseAttack += profile.Level * 2
		baseDefense += profile.Level
		baseSpeed += profile.Level
		baseMana += profile.Level * 2
		baseLevel = profile.Level + 1
	}
	base := combatAttributeTemplate{
		HP:          baseHP + uint32(index*4),
		Energy:      100,
		Attack:      baseAttack + uint32(index*2),
		Defense:     baseDefense + uint32(index),
		Speed:       baseSpeed + uint32(index),
		Mana:        baseMana + uint32(index*2),
		HitPct:      8,
		DodgePct:    4,
		CritRatePct: 6,
		CritDmgPct:  145,
	}
	resist := resistanceTemplate{
		PhysicalResistPct:  4,
		SkillResistPct:     4,
		ConfusionResistPct: 5,
		SleepResistPct:     5,
		ParalysisResistPct: 5,
		SealResistPct:      5,
		CurseResistPct:     6,
		CritResistPct:      3,
		CritDmgResistPct:   10,
		PetResistPct:       2,
		GenericShieldPct:   1,
	}
	level := baseLevel + uint32(index)
	switch enemy.EntityID {
	case 90006:
		base.HP += 8
		base.Attack += 3
		base.Mana += 4
		resist.SkillResistPct += 3
		resist.CurseResistPct += 4
	case 90004, 90005:
		base.Speed += 1
	}
	return base, resist, level
}

func (a *actorRuntime) applyCombatTemplate(template combatAttributeTemplate) {
	if a == nil {
		return
	}
	a.hp = template.HP
	a.hpMax = template.HP
	a.energy = template.Energy
	a.energyMax = template.Energy
	a.atk = template.Attack
	a.def = template.Defense
	a.spd = template.Speed
	a.mana = template.Mana
	a.hitPct = template.HitPct
	a.dodgeRatePct = template.DodgePct
	a.critRatePct = template.CritRatePct
	a.critDmgPct = template.CritDmgPct
}

func (a *actorRuntime) applyResistanceTemplate(template resistanceTemplate) {
	if a == nil {
		return
	}
	a.physicalResistPct = template.PhysicalResistPct
	a.skillResistPct = template.SkillResistPct
	a.confusionResistPct = template.ConfusionResistPct
	a.sleepResistPct = template.SleepResistPct
	a.paralysisResistPct = template.ParalysisResistPct
	a.sealResistPct = template.SealResistPct
	a.curseResistPct = template.CurseResistPct
	a.critResistPct = template.CritResistPct
	a.critDmgResistPct = template.CritDmgResistPct
	a.characterResistPct = template.CharacterResistPct
	a.petResistPct = template.PetResistPct
	a.mercenaryResistPct = template.MercenaryResistPct
	a.genericShieldPct = template.GenericShieldPct
}

func (a *actorRuntime) initRuntimeDefaults() {
	if a == nil {
		return
	}
	a.statuses = map[uint32]*statusRuntime{}
	a.globalMultiplierPct = 100
	a.attackMultiplierPct = 100
	a.defenseMultiplierPct = 100
	a.speedMultiplierPct = 100
	a.manaMultiplierPct = 100
}

// buildPlayerCharacterActor is not wired into the current StartPVE flow yet,
// but landing it now gives the upcoming solo-character PVE work one stable
// service-side entry point instead of mixing character stats into pet builders.
func buildPlayerCharacterActor(profile *player.Profile, actorType uint32) *actorRuntime {
	if profile == nil {
		return nil
	}
	attributes, resistances := combatTemplateFromProfile(profile)
	actor := &actorRuntime{
		actorID:       profile.PlayerID,
		actorType:     actorType,
		unitClass:     ActorUnitClassCharacter,
		ownerPlayerID: profile.PlayerID,
		name:          profile.Name,
		level:         maxUint32(profile.Level, 1),
		// 人物技能优先从持久化玩家档案读取；只有旧数据尚未迁移时才回退到默认模板。
		skillIDs: playerCharacterSkillIDs(profile),
	}
	actor.initRuntimeDefaults()
	actor.applyCombatTemplate(attributes)
	if profile.HP > 0 {
		actor.hp = profile.HP
	}
	if profile.HPMax > 0 {
		actor.hpMax = profile.HPMax
	}
	if profile.Energy > 0 {
		actor.energy = profile.Energy
	}
	if profile.EnergyMax > 0 {
		actor.energyMax = profile.EnergyMax
	}
	actor.applyResistanceTemplate(resistances)
	return actor
}

func playerCharacterSkillIDs(profile *player.Profile) []uint32 {
	if profile == nil || len(profile.SkillIDs) == 0 {
		return []uint32{DefaultCharacterSkillID, DefaultAttackSkillID}
	}
	return append([]uint32{}, profile.SkillIDs...)
}

func buildSoloPVEAllies(profile *player.Profile, lineup []pet.LineupPet) []*actorRuntime {
	allies := make([]*actorRuntime, 0, len(lineup)+1)
	if characterActor := buildPlayerCharacterActor(profile, PlayerActorType); characterActor != nil {
		allies = append(allies, characterActor)
	}
	allies = append(allies, buildPlayerTeam(profile, lineup, PlayerActorType)...)
	return allies
}

func buildPlayerTeam(profile *player.Profile, lineup []pet.LineupPet, actorType uint32) []*actorRuntime {
	allies := make([]*actorRuntime, 0, len(lineup))
	for index, item := range lineup {
		skillIDs := append([]uint32{}, item.SkillIDs...)
		if len(skillIDs) == 0 {
			skillIDs = []uint32{DefaultAttackSkillID}
		}
		attributes, resistances := defaultPetTemplate(item)
		actor := &actorRuntime{
			actorID:       item.PetUID,
			actorType:     actorType,
			unitClass:     ActorUnitClassPet,
			ownerPlayerID: profile.PlayerID,
			petUID:        item.PetUID,
			petID:         item.PetID,
			lineupIndex:   uint32(index),
			name:          fmt.Sprintf("%s 的%d号宠物", profile.Name, index+1),
			level:         item.Level,
			skillIDs:      skillIDs,
		}
		actor.initRuntimeDefaults()
		actor.applyCombatTemplate(attributes)
		actor.applyResistanceTemplate(resistances)
		actor.hp = item.HP
		if item.HPMax > 0 {
			actor.hpMax = item.HPMax
		}
		allies = append(allies, actor)
		configurePassiveProfile(actor)
	}
	return allies
}

func (s *Service) buildEnemyTeam(ctx context.Context, profile *player.Profile, enemy world.Entity) []*actorRuntime {
	slots := s.resolveEncounterSlots(ctx, enemy)
	return s.buildEnemyTeamFromSlots(ctx, profile, enemy, slots)
}

func (s *Service) buildEnemyTeamFromSlots(ctx context.Context, profile *player.Profile, enemy world.Entity, slots []monster.RuntimeEncounterSlot) []*actorRuntime {
	enemies := make([]*actorRuntime, 0, len(slots))
	for index, slot := range slots {
		enemyName := enemy.Name
		if len(slots) > 1 {
			enemyName = fmt.Sprintf("%s 随从%d", enemy.Name, index+1)
		}
		if slot.MonsterName != "" && len(slots) == 1 {
			enemyName = slot.MonsterName
		} else if slot.MonsterName != "" && len(slots) > 1 {
			enemyName = fmt.Sprintf("%s·%s", enemy.Name, slot.MonsterName)
		}
		attributes, resistances, enemyLevel := combatTemplateFromMonsterSlot(profile, enemy, index, slot)
		skillIDs := append([]uint32{}, slot.SkillIDs...)
		if len(skillIDs) == 0 {
			skillIDs = []uint32{DefaultEnemySkillID, 90002}
		}
		actor := &actorRuntime{
			actorID:       enemy.EntityID*10 + uint64(index+1),
			actorType:     EnemyActorType,
			unitClass:     ActorUnitClassMonster,
			ownerPlayerID: 0,
			petUID:        0,
			petID:         slot.MonsterID,
			lineupIndex:   uint32(index),
			name:          enemyName,
			level:         enemyLevel,
			skillIDs:      skillIDs,
		}
		actor.initRuntimeDefaults()
		actor.applyCombatTemplate(attributes)
		actor.applyResistanceTemplate(resistances)
		enemies = append(enemies, actor)
		configurePassiveProfile(actor)
	}
	return enemies
}

func (s *Service) resolveEncounterSlots(ctx context.Context, enemy world.Entity) []monster.RuntimeEncounterSlot {
	if s.monsterService != nil {
		encounter, err := s.monsterService.ResolveEncounterForEntity(ctx, enemy.EntityID)
		if err == nil && encounter != nil && len(encounter.Slots) > 0 {
			return encounter.Slots
		}
	}
	return fallbackEncounterSlots(enemy)
}

func fallbackEncounterSlots(enemy world.Entity) []monster.RuntimeEncounterSlot {
	count := 1
	skillSet := []uint32{DefaultEnemySkillID, 90002}
	if enemy.EntityID == 90004 || enemy.EntityID == 90005 {
		count = 2
	}
	if enemy.EntityID == 90006 {
		count = 2
		skillSet = []uint32{90002, 90003}
	}
	slots := make([]monster.RuntimeEncounterSlot, 0, count)
	for index := 0; index < count; index++ {
		slots = append(slots, monster.RuntimeEncounterSlot{
			MonsterID: DefaultEnemyPetID + uint32(index),
			SkillIDs:  append([]uint32{}, skillSet...),
		})
	}
	return slots
}

func combatTemplateFromMonsterSlot(profile *player.Profile, enemy world.Entity, index int, slot monster.RuntimeEncounterSlot) (combatAttributeTemplate, resistanceTemplate, uint32) {
	base, resist, level := defaultEnemyTemplate(profile, enemy, index)
	if slot.HPMax > 0 {
		base.HP = slot.HPMax + uint32(index*4)
		base.Attack = slot.ATK + uint32(index*2)
		base.Defense = slot.DEF + uint32(index)
		base.Speed = slot.SPD + uint32(index)
		base.Mana = slot.MANA + uint32(index*2)
	}
	if slot.Level > 0 {
		level = slot.Level + uint32(index)
	}
	return base, resist, level
}

func configurePassiveProfile(actor *actorRuntime) {
	if actor == nil {
		return
	}
	// The current project does not have a full passive repository yet, so we use
	// small deterministic demo profiles to validate the server-authoritative
	// passive pipeline before data-driven passive loading lands.
	switch actor.petID {
	case 101:
		actor.lifestealPct = 12
		actor.comboPct = 30
	case 102:
		actor.dodgePct = 18
		actor.revivePct = 100
		actor.reviveHPPct = 35
		actor.controlImmune = true
	case DefaultEnemyPetID:
		actor.counterPct = 25
	default:
		if actor.actorType == EnemyActorType {
			actor.counterPct = 12
		}
	}
}

func (b *activeBattle) resetCommandDeadline() {
	if b == nil || b.phase != PhaseCommand {
		return
	}
	b.commandDeadline = time.Now().Add(commandTimeout)
}

func (b *activeBattle) shouldAutoResolve(now time.Time) bool {
	if b == nil || b.phase != PhaseCommand || len(b.pendingActors) == 0 {
		return false
	}
	if b.autoBattleEnabled {
		return true
	}
	if b.commandDeadline.IsZero() {
		return false
	}
	return !now.Before(b.commandDeadline)
}

func (b *activeBattle) resolveRound() (*StateSnapshot, *ResultSnapshot) {
	events := make([]Event, 0, 16)
	decisions := b.collectTurnDecisions()
	for _, decision := range decisions {
		if b.hasWinner() {
			break
		}
		if decision.actor.isDead() {
			continue
		}
		if blockedStatusID, blockedLabel, blocked := decision.actor.actionBlockedStatus(); blocked {
			events = append(events, Event{
				EventType: EventTypeSkipTurn,
				SourceID:  decision.actor.actorID,
				TargetID:  decision.actor.actorID,
				StateID:   blockedStatusID,
				Label:     decision.actor.name + blockedLabel,
			})
			continue
		}
		events = append(events, b.executeDecision(decision)...)
	}

	events = append(events, b.resolveStatusTicks()...)
	b.expireRoundStatuses()
	result := b.buildRoundResult()
	b.plannedActs = make(map[uint64]ActionRequest)
	if result != nil {
		b.phase = PhaseFinished
		b.commandDeadline = time.Time{}
	} else {
		b.round++
		b.phase = PhaseCommand
		b.pendingActors = b.collectPendingControllableActors()
		b.resetCommandDeadline()
	}
	b.battleVersion++
	state := b.recordStateSnapshot(events)
	return state, result
}

func (b *activeBattle) collectTurnDecisions() []turnDecision {
	decisions := make([]turnDecision, 0, len(b.allies)+len(b.enemies))
	rng := rand.New(rand.NewSource(int64(b.battleID) + int64(b.round)*97))
	for _, actor := range b.livingActors(b.allies) {
		decision := b.plannedActs[actor.actorID]
		if decision.SkillID == 0 {
			decision = b.defaultActionFor(actor)
		}
		decisions = append(decisions, turnDecision{actor: actor, action: decision, tie: rng.Int63()})
	}
	for _, actor := range b.livingActors(b.enemies) {
		decision := b.plannedActs[actor.actorID]
		if decision.SkillID == 0 {
			decision = b.defaultActionFor(actor)
		}
		decisions = append(decisions, turnDecision{actor: actor, action: decision, tie: rng.Int63()})
	}
	sort.SliceStable(decisions, func(left, right int) bool {
		leftSpeed := decisions[left].actor.effectiveSpeed()
		rightSpeed := decisions[right].actor.effectiveSpeed()
		if leftSpeed == rightSpeed {
			return decisions[left].tie < decisions[right].tie
		}
		return leftSpeed > rightSpeed
	})
	return decisions
}

func (b *activeBattle) executeDecision(decision turnDecision) []Event {
	actor := decision.actor
	action := decision.action
	skillID := action.SkillID
	if skillID == 0 || !actor.hasSkill(skillID) || actor.hasStatus(StatusSeal) {
		skillID = DefaultAttackSkillID
	}
	skill, ok := getSkillDef(skillID)
	if !ok {
		skillID = DefaultAttackSkillID
		skill, _ = getSkillDef(skillID)
	}
	if skill.EnergyCost > 0 && !actor.canSpendEnergy(skill.EnergyCost) {
		skillID = DefaultAttackSkillID
		skill, _ = getSkillDef(skillID)
	}
	target := b.resolveDecisionTarget(actor, action.TargetID, skill)
	if target == nil && skill.TargetRule != targetEnemyAll {
		return nil
	}
	if skill.EnergyCost > 0 {
		actor.spendEnergy(skill.EnergyCost)
	}

	primaryTargetID := uint64(0)
	if target != nil {
		primaryTargetID = target.actorID
	}
	events := []Event{{
		EventType: EventTypeUseSkill,
		SourceID:  actor.actorID,
		TargetID:  primaryTargetID,
		SkillID:   skillID,
		Label:     fmt.Sprintf("%s 使用了 %s。", actor.name, skill.Name),
	}}

	if skill.HealPct > 0 {
		healValue := calculateHealAmount(actor.effectiveStats(), skill)
		restored := target.restoreHP(healValue)
		events = append(events, Event{
			EventType: EventTypeHeal,
			SourceID:  actor.actorID,
			TargetID:  target.actorID,
			SkillID:   skillID,
			Value:     restored,
			Label:     fmt.Sprintf("%s 恢复了 %d 点生命。", target.name, restored),
		})
		if !target.isDead() && skill.CritBoostRounds > 0 && skill.CritBoostPct > 0 {
			if target.applyStatus(StatusCritBoost, skill.CritBoostRounds, int32(skill.CritBoostPct)) {
				events = append(events, Event{
					EventType: EventTypeApplyStatus,
					SourceID:  actor.actorID,
					TargetID:  target.actorID,
					SkillID:   skillID,
					StateID:   StatusCritBoost,
					Value:     int32(skill.CritBoostPct),
					Label:     fmt.Sprintf("%s 的暴击率提升了 %d%%。", target.name, skill.CritBoostPct),
				})
			}
		}
		return events
	}
	if skill.TargetRule == targetEnemyAll && target == nil {
		for _, multiTarget := range b.resolveAllEnemyTargets(actor) {
			events = append(events, b.resolveDamageSkill(actor, multiTarget, skillID, skill, true, true)...)
		}
		return events
	}
	if skill.TargetRule == targetEnemyMulti && target != nil {
		for _, multiTarget := range b.resolveMultiEnemyTargets(actor, target, skill.TargetCount) {
			events = append(events, b.resolveDamageSkill(actor, multiTarget, skillID, skill, true, true)...)
		}
		return events
	}
	events = append(events, b.resolveDamageSkill(actor, target, skillID, skill, true, true)...)
	return events
}

func (b *activeBattle) resolveDamageSkill(actor *actorRuntime, target *actorRuntime, skillID uint32, skill skillDef, allowCounter bool, allowCombo bool) []Event {
	if actor == nil || target == nil || actor.isDead() || target.isDead() {
		return nil
	}

	// Evasion is checked on the authoritative server before any damage, on-hit
	// status, lifesteal, or counter logic so later passive layers share one
	// consistent "attack connected or not" branch.
	if dodgeChancePct := b.calculateDodgeChancePct(actor, target); dodgeChancePct > 0 && b.rollChance(dodgeChancePct, target.actorID+83, actor.actorID+89) {
		return []Event{{
			EventType: EventTypeDodge,
			SourceID:  target.actorID,
			TargetID:  actor.actorID,
			SkillID:   skillID,
			Label:     target.name + " 闪避了这次攻击。",
		}}
	}

	damage, crit := actor.damageAgainst(target, skill, b.battleID, b.round)
	actualDamage := target.applyDamage(damage)
	events := make([]Event, 0, 8)
	damageLabel := fmt.Sprintf("%s 受到 %d 点伤害。", target.name, actualDamage)
	if crit {
		damageLabel = fmt.Sprintf("%s 被暴击，受到 %d 点伤害。", target.name, actualDamage)
	}
	events = append(events, Event{
		EventType: EventTypeDamage,
		SourceID:  actor.actorID,
		TargetID:  target.actorID,
		SkillID:   skillID,
		Value:     actualDamage,
		Label:     damageLabel,
	})
	if target.isDead() {
		if b.tryRevive(target, &events, actor.actorID, skillID) {
			// Revive restores life immediately, so later on-hit status and reactive
			// branches still see the revived target as a living battle unit.
		} else {
			events = append(events, Event{
				EventType: EventTypeDefeat,
				SourceID:  actor.actorID,
				TargetID:  target.actorID,
				SkillID:   skillID,
				Label:     target.name + " 倒下了。",
			})
		}
	}

	if actualDamage > 0 && actor.lifestealPct > 0 && !actor.isDead() {
		healAmount := maxInt32(int32(actualDamage)*int32(actor.lifestealPct)/100, 1)
		restored := actor.restoreHP(healAmount)
		if restored > 0 {
			events = append(events, Event{
				EventType: EventTypeHeal,
				SourceID:  actor.actorID,
				TargetID:  actor.actorID,
				SkillID:   skillID,
				Value:     restored,
				Label:     fmt.Sprintf("%s 通过吸血恢复了 %d 点生命。", actor.name, restored),
			})
		}
	}

	if !target.isDead() {
		events = append(events, b.applyOnHitStatuses(actor, target, skillID, skill)...)
	}

	if allowCombo && actualDamage > 0 && !target.isDead() && b.canCombo(actor) {
		comboSkill, _ := getSkillDef(DefaultAttackSkillID)
		events = append(events, Event{
			EventType: EventTypeCombo,
			SourceID:  actor.actorID,
			TargetID:  target.actorID,
			SkillID:   DefaultAttackSkillID,
			Label:     actor.name + " 触发了连击。",
		})
		events = append(events, b.resolveDamageSkill(actor, target, DefaultAttackSkillID, comboSkill, false, false)...)
	}

	if allowCounter && actualDamage > 0 && b.canCounter(target) {
		counterSkill, _ := getSkillDef(DefaultAttackSkillID)
		events = append(events, Event{
			EventType: EventTypeCounter,
			SourceID:  target.actorID,
			TargetID:  actor.actorID,
			SkillID:   DefaultAttackSkillID,
			Label:     target.name + " 发动了反击。",
		})
		events = append(events, b.resolveDamageSkill(target, actor, DefaultAttackSkillID, counterSkill, false, false)...)
	}

	return events
}

func (b *activeBattle) applyOnHitStatuses(actor *actorRuntime, target *actorRuntime, skillID uint32, skill skillDef) []Event {
	if actor == nil || target == nil || target.isDead() {
		return nil
	}

	events := make([]Event, 0, 6)
	if chancePct := b.adjustStatusChancePct(skill.BleedChancePct, target, StatusBleed); chancePct > 0 && b.rollChance(chancePct, actor.actorID, target.actorID) {
		if target.applyStatus(StatusBleed, skill.BleedRounds, skill.BleedDamage) {
			events = append(events, Event{
				EventType: EventTypeApplyStatus,
				SourceID:  actor.actorID,
				TargetID:  target.actorID,
				SkillID:   skillID,
				StateID:   StatusBleed,
				Value:     skill.BleedDamage,
				Label:     target.name + " 进入流血状态。",
			})
		}
	}
	if chancePct := b.adjustStatusChancePct(skill.SealChancePct, target, StatusSeal); chancePct > 0 && b.rollChance(chancePct, actor.actorID+9, target.actorID+17) {
		if target.applyStatus(StatusSeal, skill.SealRounds, 0) {
			events = append(events, Event{
				EventType: EventTypeApplyStatus,
				SourceID:  actor.actorID,
				TargetID:  target.actorID,
				SkillID:   skillID,
				StateID:   StatusSeal,
				Label:     target.name + " 被封印了。",
			})
		}
	}
	if skill.VulnerabilityChancePct > 0 && b.rollChance(skill.VulnerabilityChancePct, actor.actorID+23, target.actorID+31) {
		if target.applyStatus(StatusVulnerability, skill.VulnerabilityRounds, int32(skill.VulnerabilityApplyPct)) {
			events = append(events, Event{
				EventType: EventTypeApplyStatus,
				SourceID:  actor.actorID,
				TargetID:  target.actorID,
				SkillID:   skillID,
				StateID:   StatusVulnerability,
				Value:     int32(skill.VulnerabilityApplyPct),
				Label:     fmt.Sprintf("%s 进入易伤状态，防御减伤降低 %d%%。", target.name, skill.VulnerabilityApplyPct),
			})
		}
	}
	if skill.ArmorBreakChancePct > 0 && b.rollChance(skill.ArmorBreakChancePct, actor.actorID+43, target.actorID+47) {
		if target.applyStatus(StatusArmorBreak, skill.ArmorBreakRounds, 0) {
			events = append(events, Event{
				EventType: EventTypeApplyStatus,
				SourceID:  actor.actorID,
				TargetID:  target.actorID,
				SkillID:   skillID,
				StateID:   StatusArmorBreak,
				Label:     target.name + " 被破甲了。",
			})
		}
	}
	if skill.SlowChancePct > 0 && b.rollChance(skill.SlowChancePct, actor.actorID+53, target.actorID+59) {
		if target.applyStatus(StatusSlow, skill.SlowRounds, int32(skill.SlowMultiplierPct)) {
			events = append(events, Event{
				EventType: EventTypeApplyStatus,
				SourceID:  actor.actorID,
				TargetID:  target.actorID,
				SkillID:   skillID,
				StateID:   StatusSlow,
				Value:     int32(skill.SlowMultiplierPct),
				Label:     fmt.Sprintf("%s 被减速，本回合速度倍率变为 %d%%。", target.name, skill.SlowMultiplierPct),
			})
		}
	}
	if chancePct := b.adjustStatusChancePct(skill.CurseChancePct, target, StatusCurse); chancePct > 0 && b.rollChance(chancePct, actor.actorID+61, target.actorID+67) {
		curseDamage := skill.CurseDamage + actor.effectiveStats().Mana*skill.CurseManaPct/100
		if curseDamage < 1 {
			curseDamage = 1
		}
		if target.applyStatus(StatusCurse, skill.CurseRounds, curseDamage) {
			events = append(events, Event{
				EventType: EventTypeApplyStatus,
				SourceID:  actor.actorID,
				TargetID:  target.actorID,
				SkillID:   skillID,
				StateID:   StatusCurse,
				Value:     curseDamage,
				Label:     fmt.Sprintf("%s 进入诅咒状态，每回合损失 %d 点生命。", target.name, curseDamage),
			})
		}
	}
	if chancePct := b.adjustStatusChancePct(skill.ControlChancePct, target, skill.ControlStatusID); chancePct > 0 && skill.ControlStatusID != 0 && b.rollChance(chancePct, actor.actorID+71, target.actorID+73) {
		if target.applyStatus(skill.ControlStatusID, skill.ControlRounds, 0) {
			events = append(events, Event{
				EventType: EventTypeApplyStatus,
				SourceID:  actor.actorID,
				TargetID:  target.actorID,
				SkillID:   skillID,
				StateID:   skill.ControlStatusID,
				Label:     target.name + controlStatusApplyLabel(skill.ControlStatusID),
			})
		}
	}
	return events
}

func (b *activeBattle) resolveStatusTicks() []Event {
	events := make([]Event, 0, 8)
	for _, actor := range b.allActors() {
		if actor.isDead() {
			continue
		}
		if bleed := actor.statuses[StatusBleed]; bleed != nil && bleed.remainingRound > 0 {
			actualDamage := actor.applyDamage(bleed.potency)
			events = append(events, Event{
				EventType: EventTypeStatusTick,
				SourceID:  actor.actorID,
				TargetID:  actor.actorID,
				StateID:   StatusBleed,
				Value:     actualDamage,
				Label:     fmt.Sprintf("%s 因流血损失 %d 点生命。", actor.name, actualDamage),
			})
			if actor.isDead() {
				if !b.tryRevive(actor, &events, actor.actorID, 0) {
					events = append(events, Event{
						EventType: EventTypeDefeat,
						SourceID:  actor.actorID,
						TargetID:  actor.actorID,
						StateID:   StatusBleed,
						Label:     actor.name + " 因持续伤害倒下了。",
					})
				}
				continue
			}
		}
		if curse := actor.statuses[StatusCurse]; curse != nil && curse.remainingRound > 0 {
			actualDamage := actor.applyDamage(curse.potency)
			events = append(events, Event{
				EventType: EventTypeStatusTick,
				SourceID:  actor.actorID,
				TargetID:  actor.actorID,
				StateID:   StatusCurse,
				Value:     actualDamage,
				Label:     fmt.Sprintf("%s 因诅咒损失 %d 点生命。", actor.name, actualDamage),
			})
			if actor.isDead() {
				if !b.tryRevive(actor, &events, actor.actorID, 0) {
					events = append(events, Event{
						EventType: EventTypeDefeat,
						SourceID:  actor.actorID,
						TargetID:  actor.actorID,
						StateID:   StatusCurse,
						Label:     actor.name + " 因诅咒倒下了。",
					})
				}
			}
		}
	}
	return events
}

func (b *activeBattle) expireRoundStatuses() {
	for _, actor := range b.allActors() {
		for statusID, status := range actor.statuses {
			if status.remainingRound == 0 {
				delete(actor.statuses, statusID)
				continue
			}
			status.remainingRound--
			if status.remainingRound == 0 {
				delete(actor.statuses, statusID)
			}
		}
		actor.refreshStatusDerivedModifiers()
	}
}

func (b *activeBattle) buildRoundResult() *ResultSnapshot {
	allyAlive := len(b.livingActors(b.allies))
	enemyAlive := len(b.livingActors(b.enemies))
	switch {
	case enemyAlive == 0 && allyAlive > 0:
		return b.buildResult(true, "enemy defeated")
	case allyAlive == 0:
		return b.buildResult(false, "player defeated")
	default:
		return nil
	}
}

func (b *activeBattle) buildResult(win bool, reason string) *ResultSnapshot {
	petResults := make([]PetResult, 0, len(b.allies))
	rewardGold := uint32(0)
	rewardPlayerExp := uint64(0)
	rewardPetExp := uint64(0)
	dropItems := []DropReward{}
	dropTexts := []string{}
	if win && b.battleType == BattleTypePVE {
		rewardGold, rewardPlayerExp, rewardPetExp, dropItems, dropTexts = b.buildPVERewards()
	}
	for _, actor := range b.allies {
		// 结算持久化仍然只写回真实宠物，人物 actor 的生命与奖励由玩家档案单独承担。
		if actor == nil || actor.petUID == 0 {
			continue
		}
		petResults = append(petResults, PetResult{
			PetUID:    actor.petUID,
			HP:        actor.hp,
			ExpGained: rewardPetExp,
		})
	}
	return &ResultSnapshot{
		BattleID:             b.battleID,
		BattleType:           b.battleType,
		ParticipantPlayerIDs: append([]uint64{}, b.participantPlayerIDs...),
		PetResults:           petResults,
		Win:                  win,
		ReturnSceneID:        b.returnSceneID,
		ReturnPos:            b.returnPos,
		Reason:               reason,
		RewardGold:           rewardGold,
		RewardPlayerExp:      rewardPlayerExp,
		DropItems:            append([]DropReward{}, dropItems...),
		DropTexts:            append([]string{}, dropTexts...),
	}
}

// buildPVERewards keeps the first reward formula deliberately simple and fully
// server-authored: enemy lineup size and level determine stable gold / exp
// payouts, while each participating ally receives the same pet exp packet.
func (b *activeBattle) buildPVERewards() (uint32, uint64, uint64, []DropReward, []string) {
	totalGold := uint32(0)
	totalPlayerExp := uint64(0)
	totalPetExp := uint64(0)
	dropItems := make([]DropReward, 0, len(b.enemies))
	dropTexts := make([]string, 0, len(b.enemies))
	for _, enemy := range b.enemies {
		baseGold := enemy.level*6 + 12
		baseExp := uint64(enemy.level)*10 + 18
		if enemy.petID >= 9002 {
			baseGold += 6
			baseExp += 8
		}
		totalGold += baseGold
		totalPlayerExp += baseExp
		totalPetExp += baseExp
		if dropReward, ok := buildEnemyDropReward(enemy); ok {
			dropItems = append(dropItems, dropReward)
			dropTexts = append(dropTexts, buildEnemyDropText(enemy))
		}
	}
	return totalGold, totalPlayerExp, totalPetExp, dropItems, dropTexts
}

// buildEnemyDropReward 先为当前 MVP 提供稳定的掉落物品主键与数量。
// 当前掉落表还没有完全数据化，因此这里暂时按敌方模板做最小映射，
// 后续应继续迁移到数据库配置或共享掉落表。
func buildEnemyDropReward(enemy *actorRuntime) (DropReward, bool) {
	if enemy == nil {
		return DropReward{}, false
	}
	switch enemy.petID {
	case 9001:
		return DropReward{ItemID: 3101, Quantity: 1}, true
	case 9002:
		return DropReward{ItemID: 3102, Quantity: 1}, true
	default:
		if enemy.level >= 4 {
			return DropReward{ItemID: 3103, Quantity: 1}, true
		}
		return DropReward{ItemID: 3104, Quantity: 1}, true
	}
}

// buildEnemyDropText gives the current MVP a deterministic text-only loot
// preview. BattleHandler 在真正发奖成功后会再用数据库模板名称覆盖这里的文本。
func buildEnemyDropText(enemy *actorRuntime) string {
	if enemy == nil {
		return ""
	}
	switch enemy.petID {
	case 9001:
		return "掉落: 野性毛皮 x1"
	case 9002:
		return "掉落: 锋爪碎片 x1"
	default:
		if enemy.level >= 4 {
			return "掉落: 训练徽记 x1"
		}
		return "掉落: 治愈草叶 x1"
	}
}

func (b *activeBattle) toStartSnapshot() *StartSnapshot {
	return &StartSnapshot{
		BattleID:             b.battleID,
		BattleType:           b.battleType,
		BattleVersion:        b.battleVersion,
		Frame:                b.battleVersion,
		ParticipantPlayerIDs: append([]uint64{}, b.participantPlayerIDs...),
		Allies:               b.snapshotActors(b.allies),
		Enemies:              b.snapshotActors(b.enemies),
		Round:                b.round,
		Phase:                b.phase,
		ActiveActorID:        b.currentActiveActorID(),
		ActivePetUID:         b.currentActivePetUID(),
		CommandDeadlineMS:    b.commandDeadline.UnixMilli(),
		AutoBattleEnabled:    b.autoBattleEnabled,
		PendingActorIDs:      append([]uint64{}, b.pendingActors...),
		ControllableActorIDs: b.controllableActorIDs(),
	}
}

func (b *activeBattle) toStateSnapshot(events []Event) *StateSnapshot {
	copiedEvents := make([]Event, 0, len(events))
	copiedEvents = append(copiedEvents, events...)
	return &StateSnapshot{
		BattleID:             b.battleID,
		BattleVersion:        b.battleVersion,
		Frame:                b.battleVersion,
		ParticipantPlayerIDs: append([]uint64{}, b.participantPlayerIDs...),
		Round:                b.round,
		Phase:                b.phase,
		Events:               copiedEvents,
		Actors:               b.snapshotActorStates(),
		ActiveActorID:        b.currentActiveActorID(),
		ActivePetUID:         b.currentActivePetUID(),
		CommandDeadlineMS:    b.commandDeadline.UnixMilli(),
		AutoBattleEnabled:    b.autoBattleEnabled,
		PendingActorIDs:      append([]uint64{}, b.pendingActors...),
		ControllableActorIDs: b.controllableActorIDs(),
	}
}

func (b *activeBattle) recordStateSnapshot(events []Event) *StateSnapshot {
	snapshot := b.toStateSnapshot(events)
	b.appendStateHistory(*snapshot)
	return snapshot
}

func (b *activeBattle) appendStateHistory(snapshot StateSnapshot) {
	b.stateHistory = append(b.stateHistory, cloneStateSnapshot(snapshot))
	if len(b.stateHistory) > battleReplayHistoryLimit {
		b.stateHistory = append([]StateSnapshot{}, b.stateHistory[len(b.stateHistory)-battleReplayHistoryLimit:]...)
	}
}

func cloneStateSnapshot(input StateSnapshot) StateSnapshot {
	clone := input
	clone.Events = append([]Event{}, input.Events...)
	clone.Actors = append([]ActorState{}, input.Actors...)
	clone.PendingActorIDs = append([]uint64{}, input.PendingActorIDs...)
	clone.ControllableActorIDs = append([]uint64{}, input.ControllableActorIDs...)
	return clone
}

func (b *activeBattle) snapshotActors(actors []*actorRuntime) []ActorSnapshot {
	result := make([]ActorSnapshot, 0, len(actors))
	for _, actor := range actors {
		result = append(result, actor.toSnapshot())
	}
	return result
}

func (b *activeBattle) snapshotActorStates() []ActorState {
	actors := b.allActors()
	result := make([]ActorState, 0, len(actors))
	for _, actor := range actors {
		result = append(result, ActorState{
			ActorID:    actor.actorID,
			HP:         actor.hp,
			HPMax:      actor.hpMax,
			Energy:     actor.energy,
			EnergyMax:  actor.energyMax,
			Dead:       actor.isDead(),
			CanAct:     !actor.isDead() && !actor.isActionBlocked(),
			StatusIDs:  actor.statusIDs(),
			ChargeDone: !b.isPendingActor(actor.actorID),
		})
	}
	return result
}

func (b *activeBattle) collectibleDefaultTarget(candidates []*actorRuntime) *actorRuntime {
	living := b.livingActors(candidates)
	if len(living) == 0 {
		return nil
	}
	return living[0]
}

func (b *activeBattle) defaultActionFor(actor *actorRuntime) ActionRequest {
	skillID := DefaultAttackSkillID
	if actor != nil {
		skillID = b.defaultSkillIDFor(actor)
	}
	targetID := uint64(0)
	skill, ok := getSkillDef(skillID)
	if ok && skill.TargetRule == targetAllySingle {
		if actor.actorType == EnemyActorType {
			if target := b.lowestHPActor(b.enemies); target != nil {
				targetID = target.actorID
			}
		} else if target := b.lowestHPActor(b.allies); target != nil {
			targetID = target.actorID
		}
	} else if actor.actorType == EnemyActorType {
		if target := b.collectibleDefaultTarget(b.allies); target != nil {
			targetID = target.actorID
		}
	} else if target := b.collectibleDefaultTarget(b.enemies); target != nil {
		targetID = target.actorID
	}
	return ActionRequest{
		BattleID:   b.battleID,
		Round:      b.round,
		ActionType: ActionTypeSkill,
		ActorID:    actor.actorID,
		SkillID:    skillID,
		TargetID:   targetID,
	}
}

func (b *activeBattle) defaultSkillIDFor(actor *actorRuntime) uint32 {
	if actor == nil {
		return DefaultAttackSkillID
	}
	if actor.actorType == EnemyActorType && len(actor.skillIDs) > 0 {
		return actor.skillIDs[int((b.round-1)%uint32(len(actor.skillIDs)))]
	}
	if actor.unitClass == ActorUnitClassCharacter {
		for _, skillID := range actor.skillIDs {
			if skillID == DefaultAttackSkillID {
				continue
			}
			skill, ok := getSkillDef(skillID)
			if ok && actor.canSpendEnergy(skill.EnergyCost) {
				return skillID
			}
		}
	}
	if len(actor.skillIDs) > 0 {
		return actor.skillIDs[0]
	}
	return DefaultAttackSkillID
}

func (b *activeBattle) normalizeRequestedTarget(actor *actorRuntime, targetID uint64, skillID uint32) uint64 {
	skill, ok := getSkillDef(skillID)
	if !ok {
		skill, _ = getSkillDef(DefaultAttackSkillID)
	}
	if skill.TargetRule == targetAllySingle {
		var target *actorRuntime
		if actor.actorType == PlayerActorType {
			target = b.findLivingActorFromList(b.allies, targetID)
			if target == nil {
				target = b.lowestHPActor(b.allies)
			}
		} else {
			target = b.findLivingActorFromList(b.enemies, targetID)
			if target == nil {
				target = b.lowestHPActor(b.enemies)
			}
		}
		if target != nil {
			return target.actorID
		}
		return 0
	}
	if skill.TargetRule == targetEnemyAll {
		return 0
	}
	var target *actorRuntime
	if actor.actorType == PlayerActorType {
		target = b.findLivingActorFromList(b.enemies, targetID)
		if target == nil {
			target = b.collectibleDefaultTarget(b.enemies)
		}
	} else {
		target = b.findLivingActorFromList(b.allies, targetID)
		if target == nil {
			target = b.collectibleDefaultTarget(b.allies)
		}
	}
	if target != nil {
		return target.actorID
	}
	return 0
}

func (b *activeBattle) resolveSkillTarget(actor *actorRuntime, targetID uint64, skill skillDef) *actorRuntime {
	if skill.TargetRule == targetAllySingle {
		if actor.actorType == PlayerActorType {
			return b.findLivingActorFromList(b.allies, targetID)
		}
		return b.findLivingActorFromList(b.enemies, targetID)
	}
	if skill.TargetRule == targetEnemyAll {
		return nil
	}
	if actor.actorType == PlayerActorType {
		return b.findLivingActorFromList(b.enemies, targetID)
	}
	return b.findLivingActorFromList(b.allies, targetID)
}

func (b *activeBattle) resolveDecisionTarget(actor *actorRuntime, targetID uint64, skill skillDef) *actorRuntime {
	// Confusion is resolved server-side right before execution so the client
	// cannot predict or override the forced random target.
	if actor.hasStatus(StatusConfusion) {
		return b.randomConfusionTarget(actor)
	}
	return b.resolveSkillTarget(actor, targetID, skill)
}

func (b *activeBattle) resolveAllEnemyTargets(actor *actorRuntime) []*actorRuntime {
	if actor == nil {
		return nil
	}
	if actor.actorType == PlayerActorType {
		return b.livingActors(b.enemies)
	}
	return b.livingActors(b.allies)
}

func (b *activeBattle) resolveMultiEnemyTargets(actor *actorRuntime, primary *actorRuntime, count uint32) []*actorRuntime {
	if actor == nil || primary == nil {
		return nil
	}
	if count == 0 {
		count = 1
	}
	candidates := b.resolveAllEnemyTargets(actor)
	if len(candidates) == 0 {
		return nil
	}

	result := make([]*actorRuntime, 0, count)
	seen := map[uint64]bool{}
	result = append(result, primary)
	seen[primary.actorID] = true
	for _, candidate := range candidates {
		if uint32(len(result)) >= count {
			break
		}
		if candidate == nil || seen[candidate.actorID] {
			continue
		}
		result = append(result, candidate)
		seen[candidate.actorID] = true
	}
	return result
}

func (b *activeBattle) findActor(actorID uint64) *actorRuntime {
	for _, actor := range b.allActors() {
		if actor.actorID == actorID {
			return actor
		}
	}
	return nil
}

func (b *activeBattle) findLivingActorFromList(actors []*actorRuntime, actorID uint64) *actorRuntime {
	for _, actor := range actors {
		if actor.actorID == actorID && !actor.isDead() {
			return actor
		}
	}
	return nil
}

func (b *activeBattle) allActors() []*actorRuntime {
	result := make([]*actorRuntime, 0, len(b.allies)+len(b.enemies))
	result = append(result, b.allies...)
	result = append(result, b.enemies...)
	return result
}

func (b *activeBattle) livingActors(actors []*actorRuntime) []*actorRuntime {
	result := make([]*actorRuntime, 0, len(actors))
	for _, actor := range actors {
		if !actor.isDead() {
			result = append(result, actor)
		}
	}
	return result
}

func (b *activeBattle) collectPendingControllableActors() []uint64 {
	result := make([]uint64, 0, len(b.allies)+len(b.enemies))
	for _, actor := range b.allActors() {
		if actor == nil || actor.isDead() || actor.ownerPlayerID == 0 {
			continue
		}
		result = append(result, actor.actorID)
	}
	return result
}

func (b *activeBattle) controllableActorIDs() []uint64 {
	result := make([]uint64, 0, len(b.allies)+len(b.enemies))
	for _, actor := range b.allActors() {
		if actor == nil || actor.ownerPlayerID == 0 {
			continue
		}
		result = append(result, actor.actorID)
	}
	return result
}

func (b *activeBattle) currentActiveActorID() uint64 {
	if len(b.pendingActors) > 0 {
		return b.pendingActors[0]
	}
	for _, actor := range b.allActors() {
		if !actor.isDead() && actor.ownerPlayerID != 0 {
			return actor.actorID
		}
	}
	return 0
}

func (b *activeBattle) currentActivePetUID() uint64 {
	actorID := b.currentActiveActorID()
	for _, actor := range b.allActors() {
		if actor.actorID == actorID {
			return actor.petUID
		}
	}
	if len(b.allies) > 0 {
		return b.allies[0].petUID
	}
	return 0
}

func (b *activeBattle) isPendingActor(actorID uint64) bool {
	for _, pendingActorID := range b.pendingActors {
		if pendingActorID == actorID {
			return true
		}
	}
	return false
}

func (b *activeBattle) lowestHPActor(actors []*actorRuntime) *actorRuntime {
	var target *actorRuntime
	for _, actor := range actors {
		if actor.isDead() {
			continue
		}
		if target == nil || actor.hp*target.hpMax < target.hp*actor.hpMax {
			target = actor
		}
	}
	return target
}

func (b *activeBattle) rollChance(chancePct uint32, sourceID, targetID uint64) bool {
	if chancePct == 0 {
		return false
	}
	if chancePct >= 100 {
		return true
	}
	rng := rand.New(rand.NewSource(int64(b.battleID) + int64(b.round)*131 + int64(sourceID) + int64(targetID)))
	return uint32(rng.Intn(100)) < chancePct
}

// calculateDodgeChancePct resolves the final dodge branch from the newly added
// hit and dodge attributes plus any passive dodge bonus already present on the
// runtime. Hit directly subtracts from the target's dodge budget.
func (b *activeBattle) calculateDodgeChancePct(attacker *actorRuntime, target *actorRuntime) uint32 {
	if attacker == nil || target == nil {
		return 0
	}
	attackerStats := attacker.effectiveStats()
	targetStats := target.effectiveStats()
	finalChance := int32(targetStats.DodgePct) + int32(target.dodgePct) - int32(attackerStats.HitPct)
	if finalChance < 0 {
		return 0
	}
	if finalChance > 80 {
		return 80
	}
	return uint32(finalChance)
}

// adjustStatusChancePct lets each target consume its own status-specific
// resistance without changing the rest of the battle pipeline.
func (b *activeBattle) adjustStatusChancePct(baseChancePct uint32, target *actorRuntime, statusID uint32) uint32 {
	if target == nil || baseChancePct == 0 {
		return 0
	}
	resistPct := target.statusResistPct(statusID)
	if resistPct >= baseChancePct {
		return 0
	}
	return baseChancePct - resistPct
}

func (b *activeBattle) isPlayerOnAllySide(playerID uint64) bool {
	for _, actor := range b.allies {
		if actor.ownerPlayerID == playerID {
			return true
		}
	}
	return false
}

func (b *activeBattle) hasWinner() bool {
	return len(b.livingActors(b.allies)) == 0 || len(b.livingActors(b.enemies)) == 0
}

func (b *activeBattle) canCounter(actor *actorRuntime) bool {
	if actor == nil || actor.isDead() || actor.counterPct == 0 {
		return false
	}
	// Counter is a passive reaction, but we still block it under hard crowd
	// control so control effects meaningfully suppress both active and reactive
	// offense in the current MVP rule set.
	if _, _, blocked := actor.actionBlockedStatus(); blocked {
		return false
	}
	return b.rollChance(actor.counterPct, actor.actorID+97, actor.actorID+101)
}

func (b *activeBattle) canCombo(actor *actorRuntime) bool {
	if actor == nil || actor.isDead() || actor.comboPct == 0 {
		return false
	}
	if _, _, blocked := actor.actionBlockedStatus(); blocked {
		return false
	}
	return b.rollChance(actor.comboPct, actor.actorID+103, actor.actorID+107)
}

func (b *activeBattle) tryRevive(actor *actorRuntime, events *[]Event, sourceID uint64, skillID uint32) bool {
	if actor == nil || actor.hp != 0 || actor.reviveUsed || actor.revivePct == 0 || actor.reviveHPPct == 0 {
		return false
	}
	if !b.rollChance(actor.revivePct, actor.actorID+109, actor.actorID+113) {
		return false
	}
	reviveHP := maxInt32(int32(actor.hpMax)*int32(actor.reviveHPPct)/100, 1)
	restored := actor.restoreFromRevive(reviveHP)
	if restored <= 0 {
		return false
	}
	actor.reviveUsed = true
	if events != nil {
		*events = append(*events, Event{
			EventType: EventTypeRevive,
			SourceID:  sourceID,
			TargetID:  actor.actorID,
			SkillID:   skillID,
			Value:     restored,
			Label:     fmt.Sprintf("%s 触发复活，恢复了 %d 点生命。", actor.name, restored),
		})
	}
	return true
}

func (b *activeBattle) randomConfusionTarget(actor *actorRuntime) *actorRuntime {
	candidates := make([]*actorRuntime, 0, len(b.allies)+len(b.enemies))
	for _, candidate := range b.allActors() {
		if candidate.actorID == actor.actorID || candidate.isDead() {
			continue
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return nil
	}
	rng := rand.New(rand.NewSource(int64(b.battleID) + int64(b.round)*181 + int64(actor.actorID)))
	return candidates[rng.Intn(len(candidates))]
}

func (a *actorRuntime) toSnapshot() ActorSnapshot {
	skillSnapshots := make([]SkillSnapshot, 0, len(a.skillIDs))
	for _, skillID := range a.skillIDs {
		if def, ok := getSkillDef(skillID); ok {
			skillSnapshots = append(skillSnapshots, SkillSnapshot{
				SkillID:      skillID,
				Name:         def.Name,
				TargetType:   def.TargetRule.protocolName(),
				TargetCount:  def.TargetCount,
				AnimationKey: def.AnimationKey,
				CastColor:    def.CastColor,
				ImpactColor:  def.ImpactColor,
				Projectile:   def.Projectile,
			})
			continue
		}
		skillSnapshots = append(skillSnapshots, SkillSnapshot{
			SkillID:      skillID,
			Name:         fmt.Sprintf("技能%d", skillID),
			TargetType:   targetEnemySingle.protocolName(),
			TargetCount:  1,
			AnimationKey: "slash",
			CastColor:    "#EBEBF5",
			ImpactColor:  "#FFF2F2",
			Projectile:   false,
		})
	}
	return ActorSnapshot{
		ActorID:            a.actorID,
		ActorType:          a.actorType,
		UnitClass:          a.unitClass,
		OwnerPlayerID:      a.ownerPlayerID,
		PetUID:             a.petUID,
		PetID:              a.petID,
		Name:               a.name,
		HP:                 a.hp,
		HPMax:              a.hpMax,
		Energy:             a.energy,
		EnergyMax:          a.energyMax,
		ATK:                a.atk,
		DEF:                a.def,
		SPD:                a.spd,
		MANA:               a.mana,
		HitPct:             a.hitPct,
		DodgePct:           a.dodgeRatePct,
		CritRatePct:        a.critRatePct,
		CritDmgPct:         a.critDmgPct,
		PhysicalResistPct:  a.physicalResistPct,
		SkillResistPct:     a.skillResistPct,
		ConfusionResistPct: a.confusionResistPct,
		SleepResistPct:     a.sleepResistPct,
		ParalysisResistPct: a.paralysisResistPct,
		SealResistPct:      a.sealResistPct,
		CurseResistPct:     a.curseResistPct,
		CritResistPct:      a.critResistPct,
		CritDmgResistPct:   a.critDmgResistPct,
		CharacterResistPct: a.characterResistPct,
		PetResistPct:       a.petResistPct,
		MercenaryResistPct: a.mercenaryResistPct,
		GenericShieldPct:   a.genericShieldPct,
		Skills:             skillSnapshots,
		SkillIDs:           append([]uint32{}, a.skillIDs...),
		StatusIDs:          a.statusIDs(),
		LineupIndex:        a.lineupIndex,
	}
}

func (a *actorRuntime) hasSkill(skillID uint32) bool {
	for _, candidate := range a.skillIDs {
		if candidate == skillID {
			return true
		}
	}
	return false
}

func (a *actorRuntime) isDead() bool {
	return a.hp == 0
}

func (a *actorRuntime) effectiveSpeed() uint32 {
	return uint32(maxInt32(a.effectiveStats().Speed, 0))
}

func (a *actorRuntime) damageAgainst(target *actorRuntime, skill skillDef, battleID uint64, round uint32) (int32, bool) {
	attackerStats := a.effectiveStats()
	targetStats := target.effectiveStats()
	baseDamage := calculateBaseDamage(attackerStats, targetStats, skill)
	defenseReduction := calculateDefenseReduction(targetStats, skill)
	blockReduction := calculateBlockReduction(attackerStats, targetStats, skill)
	damage := calculateFinalDamage(baseDamage.Total, defenseReduction, blockReduction)
	crit := false
	critRatePct := clampCritRatePct(saturatingSubUint32(attackerStats.CritRatePct, targetStats.CritResistPct))
	critDmgPct := clampCritDmgPct(saturatingSubUint32(attackerStats.CritDmgPct, targetStats.CritDmgResistPct))
	if skill.AllowCrit && baseDamage.allowsCriticalHit() && critRatePct > 0 {
		rng := rand.New(rand.NewSource(int64(battleID) + int64(round)*151 + int64(a.actorID) + int64(target.actorID)))
		crit = uint32(rng.Intn(100)) < critRatePct
		if crit {
			damage = damage * int32(critDmgPct) / 100
		}
	}
	return damage, crit
}

func (a *actorRuntime) applyDamage(amount int32) int32 {
	if amount <= 0 || a.hp == 0 {
		return 0
	}
	if uint32(amount) >= a.hp {
		actual := int32(a.hp)
		a.hp = 0
		return actual
	}
	a.hp -= uint32(amount)
	return amount
}

func (a *actorRuntime) restoreHP(amount int32) int32 {
	if amount <= 0 {
		return 0
	}
	missing := int32(a.hpMax - a.hp)
	if missing <= 0 {
		return 0
	}
	if amount > missing {
		amount = missing
	}
	a.hp += uint32(amount)
	return amount
}

func (a *actorRuntime) restoreFromRevive(amount int32) int32 {
	if amount <= 0 || a.hpMax == 0 {
		return 0
	}
	if amount > int32(a.hpMax) {
		amount = int32(a.hpMax)
	}
	a.hp = uint32(amount)
	return amount
}

func (a *actorRuntime) canSpendEnergy(cost uint32) bool {
	if a == nil {
		return false
	}
	return cost == 0 || a.energy >= cost
}

func (a *actorRuntime) spendEnergy(cost uint32) uint32 {
	if a == nil || cost == 0 {
		return 0
	}
	if cost > a.energy {
		cost = a.energy
	}
	a.energy -= cost
	return cost
}

func (a *actorRuntime) statusResistPct(statusID uint32) uint32 {
	if a == nil {
		return 0
	}
	switch statusID {
	case StatusConfusion:
		return a.confusionResistPct
	case StatusSleep:
		return a.sleepResistPct
	case StatusParalysis:
		return a.paralysisResistPct
	case StatusSeal:
		return a.sealResistPct
	case StatusCurse:
		return a.curseResistPct
	default:
		return 0
	}
}

func (a *actorRuntime) applyStatus(statusID uint32, rounds uint32, potency int32) bool {
	if rounds == 0 || a.isDead() {
		return false
	}
	if a.controlImmune && isControlStatus(statusID) {
		return false
	}
	current, exists := a.statuses[statusID]
	if !exists || current.remainingRound <= rounds {
		a.statuses[statusID] = &statusRuntime{statusID: statusID, remainingRound: rounds, potency: potency}
		a.refreshStatusDerivedModifiers()
		return true
	}
	return false
}

func (a *actorRuntime) hasStatus(statusID uint32) bool {
	status, ok := a.statuses[statusID]
	return ok && status.remainingRound > 0
}

func (a *actorRuntime) actionBlockedStatus() (uint32, string, bool) {
	switch {
	case a.hasStatus(StatusStun):
		return StatusStun, " 被眩晕，跳过行动。", true
	case a.hasStatus(StatusBind):
		return StatusBind, " 被束缚，无法行动。", true
	case a.hasStatus(StatusSleep):
		return StatusSleep, " 陷入沉睡，无法行动。", true
	case a.hasStatus(StatusParalysis):
		return StatusParalysis, " 被麻痹，无法行动。", true
	default:
		return 0, "", false
	}
}

func (a *actorRuntime) isActionBlocked() bool {
	_, _, blocked := a.actionBlockedStatus()
	return blocked
}

func (a *actorRuntime) statusIDs() []uint32 {
	result := make([]uint32, 0, len(a.statuses))
	for statusID, status := range a.statuses {
		if status.remainingRound > 0 {
			result = append(result, statusID)
		}
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left] < result[right]
	})
	return result
}

func removeActorID(actorIDs []uint64, targetID uint64) []uint64 {
	result := make([]uint64, 0, len(actorIDs))
	for _, actorID := range actorIDs {
		if actorID != targetID {
			result = append(result, actorID)
		}
	}
	return result
}

func controlStatusApplyLabel(statusID uint32) string {
	switch statusID {
	case StatusBind:
		return " 被束缚了。"
	case StatusSleep:
		return " 陷入沉睡。"
	case StatusParalysis:
		return " 被麻痹了。"
	case StatusConfusion:
		return " 陷入混乱。"
	default:
		return " 进入控制状态。"
	}
}

func isControlStatus(statusID uint32) bool {
	switch statusID {
	case StatusStun, StatusBind, StatusSleep, StatusParalysis, StatusConfusion, StatusSeal:
		return true
	default:
		return false
	}
}

func (a *actorRuntime) refreshStatusDerivedModifiers() {
	// We rebuild all status-derived fields from the authoritative status map so
	// status expiry and overwrite rules cannot leave stale battle modifiers on
	// the actor runtime.
	a.statusVulnerabilityPct = 0
	a.statusArmorBroken = false
	a.statusSpeedMultiplierPct = 100
	a.statusCritRateBonusPct = 0
	for _, status := range a.statuses {
		if status == nil || status.remainingRound == 0 {
			continue
		}
		switch status.statusID {
		case StatusVulnerability:
			a.statusVulnerabilityPct = maxUint32(a.statusVulnerabilityPct, uint32(maxInt32(status.potency, 0)))
		case StatusArmorBreak:
			a.statusArmorBroken = true
		case StatusSlow:
			slowMultiplierPct := uint32(maxInt32(status.potency, 0))
			if slowMultiplierPct == 0 {
				slowMultiplierPct = 100
			}
			if slowMultiplierPct < a.statusSpeedMultiplierPct {
				a.statusSpeedMultiplierPct = slowMultiplierPct
			}
		case StatusCritBoost:
			a.statusCritRateBonusPct = maxUint32(a.statusCritRateBonusPct, uint32(maxInt32(status.potency, 0)))
		}
	}
}

func maxInt32(left int32, right int32) int32 {
	if left > right {
		return left
	}
	return right
}
