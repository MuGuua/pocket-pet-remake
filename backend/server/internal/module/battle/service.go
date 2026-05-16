package battle

import (
	"context"
	"sync"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/world"
)

type Service struct {
	mu             sync.Mutex
	nextBattleID   uint64
	activeByPlayer map[uint64]*activeBattle
}

type activeBattle struct {
	battleID      uint64
	battleType    uint32
	battleVersion uint32
	round         uint32
	playerID      uint64
	returnSceneID uint32
	returnPos     world.Vec2i
	ally          actorRuntime
	enemy         actorRuntime
}

type actorRuntime struct {
	actorID     uint64
	actorType   uint32
	petUID      uint64
	petID       uint32
	lineupIndex uint32
	name        string
	level       uint32
	hp          uint32
	hpMax       uint32
	skillIDs    []uint32
}

func NewService() *Service {
	return &Service{
		nextBattleID:   70000,
		activeByPlayer: make(map[uint64]*activeBattle),
	}
}

func (s *Service) StartPVE(_ context.Context, profile *player.Profile, lineup []pet.LineupPet, enemy world.Entity) (*StartSnapshot, error) {
	if profile == nil {
		return nil, ErrTargetUnavailable
	}
	if len(lineup) == 0 {
		return nil, ErrNoLineupAvailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.activeByPlayer[profile.PlayerID]; exists {
		return nil, ErrBattleAlreadyActive
	}

	leadPet := lineup[0]
	s.nextBattleID++
	battleID := s.nextBattleID

	battle := &activeBattle{
		battleID:      battleID,
		battleType:    BattleTypePVE,
		battleVersion: 1,
		round:         1,
		playerID:      profile.PlayerID,
		returnSceneID: profile.SceneID,
		returnPos:     world.Vec2i{X: profile.PosX, Y: profile.PosY},
		ally: actorRuntime{
			actorID:     leadPet.PetUID,
			actorType:   PlayerActorType,
			petUID:      leadPet.PetUID,
			petID:       leadPet.PetID,
			lineupIndex: 0,
			name:        profile.Name + " 的主战宠",
			level:       leadPet.Level,
			hp:          leadPet.HP,
			hpMax:       leadPet.HPMax,
			skillIDs:    []uint32{DefaultAttackSkillID, 1002},
		},
		enemy: actorRuntime{
			actorID:   enemy.EntityID + 100000,
			actorType: EnemyActorType,
			petUID:    0,
			petID:     DefaultEnemyPetID,
			name:      enemy.Name,
			level:     1 + profile.Level,
			hp:        18 + uint32(profile.Level)*4,
			hpMax:     18 + uint32(profile.Level)*4,
			skillIDs:  []uint32{DefaultEnemySkillID, 90002},
		},
	}
	s.activeByPlayer[profile.PlayerID] = battle
	return battle.toStartSnapshot(), nil
}

func (s *Service) SubmitAction(_ context.Context, playerID uint64, request ActionRequest) (*ActionOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	battle, ok := s.activeByPlayer[playerID]
	if !ok {
		return nil, ErrBattleNotFound
	}
	if battle.battleID != request.BattleID || battle.round != request.Round {
		return nil, ErrInvalidAction
	}
	if request.ActorID != 0 && request.ActorID != battle.ally.actorID {
		return nil, ErrInvalidAction
	}

	switch request.ActionType {
	case ActionTypeSkill:
		return s.resolveSkillActionLocked(playerID, battle, request)
	case ActionTypeEscape:
		result := &ResultSnapshot{
			BattleID:      battle.battleID,
			ActivePetUID:  battle.ally.petUID,
			ActivePetHP:   battle.ally.hp,
			Win:           false,
			ReturnSceneID: battle.returnSceneID,
			ReturnPos:     battle.returnPos,
			Reason:        "player escaped battle",
		}
		delete(s.activeByPlayer, playerID)
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: true, Reason: "escape accepted"},
			Result:   result,
		}, nil
	default:
		return nil, ErrInvalidAction
	}
}

func (s *Service) resolveSkillActionLocked(playerID uint64, battle *activeBattle, request ActionRequest) (*ActionOutcome, error) {
	targetID := request.TargetID
	if targetID == 0 {
		targetID = battle.enemy.actorID
	}
	if targetID != battle.enemy.actorID {
		return nil, ErrInvalidAction
	}

	skillID := request.SkillID
	if skillID == 0 {
		skillID = DefaultAttackSkillID
	}
	playerSkill, ok := getSkillDef(skillID)
	if !ok || !battle.ally.hasSkill(skillID) {
		return nil, ErrInvalidAction
	}

	events := []Event{
		{
			EventType: EventTypeUseSkill,
			SourceID:  battle.ally.actorID,
			TargetID:  battle.enemy.actorID,
			SkillID:   skillID,
		},
	}

	playerDamage := playerSkill.damageForLevel(battle.ally.level)
	actualPlayerDamage := clampDamage(playerDamage, battle.enemy.hp)
	battle.enemy.hp -= uint32(actualPlayerDamage)
	events = append(events, Event{
		EventType: EventTypeDamage,
		SourceID:  battle.ally.actorID,
		TargetID:  battle.enemy.actorID,
		SkillID:   skillID,
		Value:     actualPlayerDamage,
	})

	if battle.enemy.hp == 0 {
		battle.battleVersion++
		state := battle.toStateSnapshot(events)
		result := &ResultSnapshot{
			BattleID:      battle.battleID,
			ActivePetUID:  battle.ally.petUID,
			ActivePetHP:   battle.ally.hp,
			Win:           true,
			ReturnSceneID: battle.returnSceneID,
			ReturnPos:     battle.returnPos,
			Reason:        "enemy defeated",
		}
		delete(s.activeByPlayer, playerID)
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: true, Reason: "action accepted"},
			State:    state,
			Result:   result,
		}, nil
	}

	enemySkillID := battle.enemy.nextSkillIDForRound(battle.round)
	enemySkill, ok := getSkillDef(enemySkillID)
	if !ok {
		enemySkillID = DefaultEnemySkillID
		enemySkill, _ = getSkillDef(enemySkillID)
	}
	enemyDamage := enemySkill.damageForLevel(battle.enemy.level)
	actualEnemyDamage := clampDamage(enemyDamage, battle.ally.hp)
	events = append(events, Event{
		EventType: EventTypeUseSkill,
		SourceID:  battle.enemy.actorID,
		TargetID:  battle.ally.actorID,
		SkillID:   enemySkillID,
	})
	battle.ally.hp -= uint32(actualEnemyDamage)
	events = append(events, Event{
		EventType: EventTypeDamage,
		SourceID:  battle.enemy.actorID,
		TargetID:  battle.ally.actorID,
		SkillID:   enemySkillID,
		Value:     actualEnemyDamage,
	})

	battle.round++
	battle.battleVersion++
	state := battle.toStateSnapshot(events)
	if battle.ally.hp == 0 {
		result := &ResultSnapshot{
			BattleID:      battle.battleID,
			ActivePetUID:  battle.ally.petUID,
			ActivePetHP:   battle.ally.hp,
			Win:           false,
			ReturnSceneID: battle.returnSceneID,
			ReturnPos:     battle.returnPos,
			Reason:        "player defeated",
		}
		delete(s.activeByPlayer, playerID)
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: true, Reason: "action accepted"},
			State:    state,
			Result:   result,
		}, nil
	}

	return &ActionOutcome{
		Response: BattleActionResponse{Accepted: true, Reason: "action accepted"},
		State:    state,
	}, nil
}

func (b *activeBattle) toStartSnapshot() *StartSnapshot {
	return &StartSnapshot{
		BattleID:      b.battleID,
		BattleType:    b.battleType,
		BattleVersion: b.battleVersion,
		Allies:        []ActorSnapshot{b.ally.toSnapshot()},
		Enemies:       []ActorSnapshot{b.enemy.toSnapshot()},
		Round:         b.round,
		ActiveActorID: b.ally.actorID,
		ActivePetUID:  b.ally.petUID,
	}
}

func (b *activeBattle) toStateSnapshot(events []Event) *StateSnapshot {
	copiedEvents := make([]Event, 0, len(events))
	copiedEvents = append(copiedEvents, events...)
	return &StateSnapshot{
		BattleID:      b.battleID,
		BattleVersion: b.battleVersion,
		Round:         b.round,
		Events:        copiedEvents,
		Actors: []ActorState{
			{
				ActorID: b.ally.actorID,
				HP:      b.ally.hp,
				HPMax:   b.ally.hpMax,
				Dead:    b.ally.hp == 0,
			},
			{
				ActorID: b.enemy.actorID,
				HP:      b.enemy.hp,
				HPMax:   b.enemy.hpMax,
				Dead:    b.enemy.hp == 0,
			},
		},
		ActiveActorID: b.ally.actorID,
		ActivePetUID:  b.ally.petUID,
	}
}

func (a actorRuntime) toSnapshot() ActorSnapshot {
	skills := make([]uint32, 0, len(a.skillIDs))
	skills = append(skills, a.skillIDs...)
	return ActorSnapshot{
		ActorID:     a.actorID,
		ActorType:   a.actorType,
		PetUID:      a.petUID,
		PetID:       a.petID,
		Name:        a.name,
		HP:          a.hp,
		HPMax:       a.hpMax,
		SkillIDs:    skills,
		LineupIndex: a.lineupIndex,
	}
}

func (a actorRuntime) hasSkill(skillID uint32) bool {
	for _, candidate := range a.skillIDs {
		if candidate == skillID {
			return true
		}
	}
	return false
}

func (a actorRuntime) nextSkillIDForRound(round uint32) uint32 {
	if len(a.skillIDs) == 0 {
		return DefaultEnemySkillID
	}
	index := int((round - 1) % uint32(len(a.skillIDs)))
	return a.skillIDs[index]
}

func (s skillDef) damageForLevel(level uint32) int32 {
	if s.FixedDamage > 0 {
		return s.FixedDamage
	}
	levelFactor := maxInt(int(level), 1)
	return s.BaseDamage + int32(levelFactor)*s.LevelBonus
}

func clampDamage(damage int32, currentHP uint32) int32 {
	if damage <= 0 {
		return 0
	}
	if uint32(damage) > currentHP {
		return int32(currentHP)
	}
	return damage
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
