package progression

import (
	"context"
	"sort"
	"sync"
)

// Service 负责玩家经验升级、属性加点与战斗属性重算。
type Service struct {
	repo Repository

	mu            sync.RWMutex
	levelConfigs  map[uint32]LevelConfig
	convertRules  []AttrConvertConfig
}

// NewService 构造玩家成长服务。
func NewService(repo Repository) *Service {
	return &Service{
		repo:         repo,
		levelConfigs: map[uint32]LevelConfig{},
	}
}

// RefreshRuntimeCache 从数据库加载等级与转化率配置到内存。
func (s *Service) RefreshRuntimeCache(ctx context.Context) error {
	if s.repo == nil {
		return nil
	}
	levelConfigs, err := s.repo.ListLevelConfigs(ctx)
	if err != nil {
		return err
	}
	convertRules, err := s.repo.ListAttrConvertConfigs(ctx)
	if err != nil {
		return err
	}

	levelMap := make(map[uint32]LevelConfig, len(levelConfigs))
	for _, item := range levelConfigs {
		if item.Status != 1 {
			continue
		}
		levelMap[item.Level] = item
	}
	activeRules := make([]AttrConvertConfig, 0, len(convertRules))
	for _, item := range convertRules {
		if item.Status != 1 || item.ConvertRate == 0 {
			continue
		}
		activeRules = append(activeRules, item)
	}
	sort.Slice(activeRules, func(i, j int) bool {
		if activeRules[i].SourceAttr == activeRules[j].SourceAttr {
			return activeRules[i].TargetAttr < activeRules[j].TargetAttr
		}
		return activeRules[i].SourceAttr < activeRules[j].SourceAttr
	})

	s.mu.Lock()
	s.levelConfigs = levelMap
	s.convertRules = activeRules
	s.mu.Unlock()
	return nil
}

// ApplyExp 在服务端结算经验并处理连升与溢出结转。
func (s *Service) ApplyExp(ctx context.Context, playerID uint64, expGain uint64) (*ExpApplyResult, error) {
	if s.repo == nil || playerID == 0 || expGain == 0 {
		return nil, ErrInvalidAllocateInput
	}
	state, err := s.repo.LoadProgressionState(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, ErrLevelConfigNotFound
	}

	result := ExpApplyResult{
		Level:          state.Level,
		Exp:            state.Exp + expGain,
		FreeAttrPoints: state.FreeAttrPoints,
	}
	startingLevel := state.Level
	for result.Level < MaxPlayerLevel {
		levelConfig := s.levelConfig(result.Level)
		if levelConfig == nil || levelConfig.ExpRequired == 0 {
			break
		}
		if result.Exp < levelConfig.ExpRequired {
			break
		}
		result.Exp -= levelConfig.ExpRequired
		result.Level++
		result.LevelUpCount++
		result.AttrPointsGained += levelConfig.AttrPoints
		result.FreeAttrPoints += levelConfig.AttrPoints
	}
	if result.Level >= MaxPlayerLevel {
		result.Level = MaxPlayerLevel
	}
	// 从旧等级升到新等级时，按已完成等级行的战斗加成累加裸装基础值。
	for level := startingLevel; level < result.Level; level++ {
		config := s.levelConfig(level)
		if config != nil {
			result.CombatBonusGain = result.CombatBonusGain.Add(LevelUpCombatBonus{
				ATK:   config.BonusATK,
				HPMax: config.BonusHPMax,
				SPD:   config.BonusSPD,
				MANA:  config.BonusMANA,
			})
		}
		s.applyLevelCombatBonus(&state.BaseCombat, level)
	}
	result.ExpToNext = s.expToNext(result.Level, result.Exp)

	combatBonus := s.calcCombatBonus(state.Allocated)
	if err := s.repo.SaveExpProgression(ctx, playerID, result, state.BaseCombat, combatBonus); err != nil {
		return nil, err
	}
	return &result, nil
}

// AllocateAttrPoints 校验并持久化玩家主动加点。
func (s *Service) AllocateAttrPoints(ctx context.Context, playerID uint64, delta AttrAllocationDelta) error {
	if s.repo == nil || playerID == 0 || delta.Total() == 0 {
		return ErrInvalidAllocateInput
	}
	state, err := s.repo.LoadProgressionState(ctx, playerID)
	if err != nil {
		return err
	}
	if state == nil {
		return ErrLevelConfigNotFound
	}
	if delta.Total() > state.FreeAttrPoints {
		return ErrInsufficientAttrPoints
	}

	freeBefore := state.FreeAttrPoints
	freeAfter := freeBefore - delta.Total()
	state.Allocated.Strength += delta.Strength
	state.Allocated.Vitality += delta.Vitality
	state.Allocated.Agility += delta.Agility
	state.Allocated.Mind += delta.Mind
	combatBonus := s.calcCombatBonus(state.Allocated)
	return s.repo.SaveAttrAllocation(ctx, playerID, delta, freeBefore, freeAfter, combatBonus)
}

// ExpToNext 根据缓存配置计算当前等级距离下一级还需要的经验。
func (s *Service) ExpToNext(level uint32, exp uint64) uint64 {
	return s.expToNext(level, exp)
}

// ListAdminLevelConfigs 返回后台可编辑的等级配置列表。
func (s *Service) ListAdminLevelConfigs(ctx context.Context) ([]LevelConfig, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.ListLevelConfigs(ctx)
}

// ListAdminAttrConvertConfigs 返回后台可编辑的转化率配置列表。
func (s *Service) ListAdminAttrConvertConfigs(ctx context.Context) ([]AttrConvertConfig, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.ListAttrConvertConfigs(ctx)
}

// UpsertAdminLevelConfig 更新单条等级配置并刷新运行时缓存。
func (s *Service) UpsertAdminLevelConfig(ctx context.Context, level uint32, input AdminUpsertLevelConfigInput) (*LevelConfig, error) {
	if s.repo == nil || level == 0 || level > MaxPlayerLevel {
		return nil, ErrInvalidAllocateInput
	}
	updated, err := s.repo.UpsertLevelConfig(ctx, level, input)
	if err != nil {
		return nil, err
	}
	if err := s.RefreshRuntimeCache(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

// UpsertAdminAttrConvertConfig 更新单条转化率配置并刷新运行时缓存。
func (s *Service) UpsertAdminAttrConvertConfig(ctx context.Context, id uint64, input AdminUpsertAttrConvertInput) (*AttrConvertConfig, error) {
	if s.repo == nil || id == 0 {
		return nil, ErrInvalidAllocateInput
	}
	updated, err := s.repo.UpsertAttrConvertConfig(ctx, id, input)
	if err != nil {
		return nil, err
	}
	if err := s.RefreshRuntimeCache(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) levelConfig(level uint32) *LevelConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if level == 0 || level > MaxPlayerLevel {
		return nil
	}
	config, ok := s.levelConfigs[level]
	if !ok {
		return nil
	}
	copied := config
	return &copied
}

// applyLevelCombatBonus 把指定等级行的战斗加成累加到裸装基础属性。
func (s *Service) applyLevelCombatBonus(base *BaseCombatStats, completedLevel uint32) {
	if base == nil || completedLevel == 0 {
		return
	}
	config := s.levelConfig(completedLevel)
	if config == nil {
		return
	}
	base.ATK += config.BonusATK
	base.HPMax += config.BonusHPMax
	base.SPD += config.BonusSPD
	base.MANA += config.BonusMANA
}

func (s *Service) expToNext(level uint32, exp uint64) uint64 {
	if level >= MaxPlayerLevel {
		return 0
	}
	levelConfig := s.levelConfig(level)
	if levelConfig == nil || levelConfig.ExpRequired == 0 {
		return 0
	}
	if exp >= levelConfig.ExpRequired {
		return 0
	}
	return levelConfig.ExpRequired - exp
}

func (s *Service) calcCombatBonus(allocated AllocatedAttrs) CombatBonus {
	s.mu.RLock()
	rules := append([]AttrConvertConfig(nil), s.convertRules...)
	s.mu.RUnlock()

	bonus := CombatBonus{}
	for _, rule := range rules {
		points := sourceAttrPoints(allocated, rule.SourceAttr)
		if points == 0 {
			continue
		}
		delta := points * rule.ConvertRate
		switch rule.TargetAttr {
		case "hp_max":
			bonus.HPMax += delta
		case "atk":
			bonus.ATK += delta
		case "def":
			bonus.DEF += delta
		case "spd":
			bonus.SPD += delta
		case "mana":
			bonus.MANA += delta
		case "hit_pct":
			bonus.HitPct += delta
		case "dodge_pct":
			bonus.DodgePct += delta
		}
	}
	return bonus
}

func sourceAttrPoints(allocated AllocatedAttrs, sourceAttr string) uint32 {
	switch sourceAttr {
	case SourceAttrStrength:
		return allocated.Strength
	case SourceAttrVitality:
		return allocated.Vitality
	case SourceAttrAgility:
		return allocated.Agility
	case SourceAttrMind:
		return allocated.Mind
	default:
		return 0
	}
}

// FinalCombatValue 根据裸装基础值与加点加成计算最终战斗属性。
func FinalCombatValue(base uint32, bonus uint32) uint32 {
	return base + bonus
}
