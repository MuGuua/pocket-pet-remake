package battle

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

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
	phase         string
	playerID      uint64
	returnSceneID uint32
	returnPos     world.Vec2i
	allies        []*actorRuntime
	enemies       []*actorRuntime
	pendingActors []uint64
	plannedActs   map[uint64]ActionRequest
	autoBattleEnabled bool
	commandDeadline   time.Time
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
	atk         uint32
	def         uint32
	spd         uint32
	mana        uint32
	skillIDs    []uint32
	critRatePct uint32
	critDmgPct  uint32
	statuses    map[uint32]*statusRuntime

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
	genericBlockPct          uint32
	petBlockPct              uint32
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

type turnDecision struct {
	actor  *actorRuntime
	action ActionRequest
	tie    int64
}

const commandTimeout = 15 * time.Second

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

	s.nextBattleID++
	battleID := s.nextBattleID
	battle := &activeBattle{
		battleID:      battleID,
		battleType:    BattleTypePVE,
		battleVersion: 1,
		round:         1,
		phase:         PhaseCommand,
		playerID:      profile.PlayerID,
		returnSceneID: profile.SceneID,
		returnPos:     world.Vec2i{X: profile.PosX, Y: profile.PosY},
		allies:        buildAllyTeam(profile, lineup),
		enemies:       buildEnemyTeam(profile, enemy),
		plannedActs:   make(map[uint64]ActionRequest),
	}
	battle.pendingActors = battle.collectPendingAllies()
	battle.resetCommandDeadline()
	if len(battle.pendingActors) == 0 {
		return nil, ErrNoLineupAvailable
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
	if battle.battleID != request.BattleID || battle.round != request.Round || battle.phase != PhaseCommand {
		return nil, ErrInvalidAction
	}

	switch request.ActionType {
	case ActionTypeSkill:
		return s.queueActionLocked(playerID, battle, request)
	case ActionTypeEscape:
		result := battle.buildResult(false, "player escaped battle")
		delete(s.activeByPlayer, playerID)
		return &ActionOutcome{
			Response: BattleActionResponse{Accepted: true, Reason: "escape accepted"},
			Result:   result,
		}, nil
	case ActionTypeSetAuto:
		return s.setAutoBattleLocked(playerID, battle, request)
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

func (s *Service) queueActionLocked(playerID uint64, battle *activeBattle, request ActionRequest) (*ActionOutcome, error) {
	actor := battle.findActor(request.ActorID)
	if actor == nil || actor.isDead() || actor.actorType != PlayerActorType {
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
			State:    battle.toStateSnapshot(nil),
		}, nil
	}

	state, result := battle.resolveRound()
	if result != nil {
		delete(s.activeByPlayer, playerID)
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
		State:    battle.toStateSnapshot(nil),
	}, nil
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
		delete(s.activeByPlayer, playerID)
	}
	return state, result
}

func buildAllyTeam(profile *player.Profile, lineup []pet.LineupPet) []*actorRuntime {
	allies := make([]*actorRuntime, 0, len(lineup))
	for index, item := range lineup {
		skillIDs := append([]uint32{}, item.SkillIDs...)
		if len(skillIDs) == 0 {
			skillIDs = []uint32{DefaultAttackSkillID}
		}
		allies = append(allies, &actorRuntime{
			actorID:              item.PetUID,
			actorType:            PlayerActorType,
			petUID:               item.PetUID,
			petID:                item.PetID,
			lineupIndex:          uint32(index),
			name:                 fmt.Sprintf("%s 的%d号宠物", profile.Name, index+1),
			level:                item.Level,
			hp:                   item.HP,
			hpMax:                item.HPMax,
			atk:                  item.ATK,
			def:                  item.DEF,
			spd:                  item.SPD,
			mana:                 item.MANA,
			skillIDs:             skillIDs,
			critRatePct:          8,
			critDmgPct:           150,
			statuses:             map[uint32]*statusRuntime{},
			globalMultiplierPct:  100,
			attackMultiplierPct:  100,
			defenseMultiplierPct: 100,
			speedMultiplierPct:   100,
			manaMultiplierPct:    100,
		})
		configurePassiveProfile(allies[len(allies)-1])
	}
	return allies
}

func buildEnemyTeam(profile *player.Profile, enemy world.Entity) []*actorRuntime {
	count := 1
	skillSet := []uint32{DefaultEnemySkillID, 90002}
	baseHP := uint32(18 + profile.Level*4)
	baseATK := uint32(10 + profile.Level*2)
	baseDEF := uint32(8 + profile.Level)
	baseSPD := uint32(7 + profile.Level)
	baseMANA := uint32(9 + profile.Level*2)
	if enemy.EntityID == 90004 || enemy.EntityID == 90005 {
		count = 2
	}
	if enemy.EntityID == 90006 {
		count = 2
		skillSet = []uint32{90002, 90003}
		baseHP += 8
		baseATK += 3
		baseMANA += 4
	}
	enemies := make([]*actorRuntime, 0, count)
	for index := 0; index < count; index++ {
		enemyName := enemy.Name
		if count > 1 {
			enemyName = fmt.Sprintf("%s 随从%d", enemy.Name, index+1)
		}
		enemies = append(enemies, &actorRuntime{
			actorID:              enemy.EntityID*10 + uint64(index+1),
			actorType:            EnemyActorType,
			petUID:               0,
			petID:                DefaultEnemyPetID + uint32(index),
			lineupIndex:          uint32(index),
			name:                 enemyName,
			level:                profile.Level + uint32(index) + 1,
			hp:                   baseHP + uint32(index*4),
			hpMax:                baseHP + uint32(index*4),
			atk:                  baseATK + uint32(index*2),
			def:                  baseDEF + uint32(index),
			spd:                  baseSPD + uint32(index),
			mana:                 baseMANA + uint32(index*2),
			skillIDs:             append([]uint32{}, skillSet...),
			critRatePct:          5,
			critDmgPct:           140,
			statuses:             map[uint32]*statusRuntime{},
			globalMultiplierPct:  100,
			attackMultiplierPct:  100,
			defenseMultiplierPct: 100,
			speedMultiplierPct:   100,
			manaMultiplierPct:    100,
		})
		configurePassiveProfile(enemies[len(enemies)-1])
	}
	return enemies
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
		b.pendingActors = b.collectPendingAllies()
		b.resetCommandDeadline()
	}
	b.battleVersion++
	state := b.toStateSnapshot(events)
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
		decision := b.defaultActionFor(actor)
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
	target := b.resolveDecisionTarget(actor, action.TargetID, skill)
	if target == nil {
		return nil
	}

	events := []Event{{
		EventType: EventTypeUseSkill,
		SourceID:  actor.actorID,
		TargetID:  target.actorID,
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
	if target.dodgePct > 0 && b.rollChance(target.dodgePct, target.actorID+83, actor.actorID+89) {
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
	if skill.BleedChancePct > 0 && b.rollChance(skill.BleedChancePct, actor.actorID, target.actorID) {
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
	if skill.SealChancePct > 0 && b.rollChance(skill.SealChancePct, actor.actorID+9, target.actorID+17) {
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
	if skill.CurseChancePct > 0 && b.rollChance(skill.CurseChancePct, actor.actorID+61, target.actorID+67) {
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
	if skill.ControlChancePct > 0 && skill.ControlStatusID != 0 && b.rollChance(skill.ControlChancePct, actor.actorID+71, target.actorID+73) {
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
	for _, actor := range b.allies {
		petResults = append(petResults, PetResult{PetUID: actor.petUID, HP: actor.hp})
	}
	return &ResultSnapshot{
		BattleID:      b.battleID,
		PetResults:    petResults,
		Win:           win,
		ReturnSceneID: b.returnSceneID,
		ReturnPos:     b.returnPos,
		Reason:        reason,
	}
}

func (b *activeBattle) toStartSnapshot() *StartSnapshot {
	return &StartSnapshot{
		BattleID:             b.battleID,
		BattleType:           b.battleType,
		BattleVersion:        b.battleVersion,
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
	if actor.actorType == EnemyActorType && len(actor.skillIDs) > 0 {
		skillID = actor.skillIDs[int((b.round-1)%uint32(len(actor.skillIDs)))]
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

func (b *activeBattle) collectPendingAllies() []uint64 {
	result := make([]uint64, 0, len(b.allies))
	for _, actor := range b.allies {
		if !actor.isDead() {
			result = append(result, actor.actorID)
		}
	}
	return result
}

func (b *activeBattle) controllableActorIDs() []uint64 {
	result := make([]uint64, 0, len(b.allies))
	for _, actor := range b.allies {
		result = append(result, actor.actorID)
	}
	return result
}

func (b *activeBattle) currentActiveActorID() uint64 {
	if len(b.pendingActors) > 0 {
		return b.pendingActors[0]
	}
	for _, actor := range b.allies {
		if !actor.isDead() {
			return actor.actorID
		}
	}
	return 0
}

func (b *activeBattle) currentActivePetUID() uint64 {
	actorID := b.currentActiveActorID()
	for _, actor := range b.allies {
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
				SkillID:    skillID,
				Name:       def.Name,
				TargetType: def.TargetRule.protocolName(),
			})
			continue
		}
		skillSnapshots = append(skillSnapshots, SkillSnapshot{
			SkillID:    skillID,
			Name:       fmt.Sprintf("技能%d", skillID),
			TargetType: targetEnemySingle.protocolName(),
		})
	}
	return ActorSnapshot{
		ActorID:     a.actorID,
		ActorType:   a.actorType,
		PetUID:      a.petUID,
		PetID:       a.petID,
		Name:        a.name,
		HP:          a.hp,
		HPMax:       a.hpMax,
		ATK:         a.atk,
		DEF:         a.def,
		SPD:         a.spd,
		Skills:      skillSnapshots,
		SkillIDs:    append([]uint32{}, a.skillIDs...),
		StatusIDs:   a.statusIDs(),
		LineupIndex: a.lineupIndex,
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
	blockReduction := calculateBlockReduction(attackerStats, targetStats)
	damage := calculateFinalDamage(baseDamage.Total, defenseReduction, blockReduction)
	crit := false
	critRatePct := clampCritRatePct(attackerStats.CritRatePct)
	critDmgPct := clampCritDmgPct(attackerStats.CritDmgPct)
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
