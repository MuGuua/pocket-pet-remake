package petprogression

import (
	"context"
	"sync"
	"strings"
)

// Service 负责宠物经验升级、属性加点与战斗属性重算。
type Service struct {
	repo Repository

	mu           sync.RWMutex
	levelConfigs map[uint32]LevelConfig
	convertRates ConvertRates
}

// NewService 构造宠物成长服务。
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
	convertConfigs, err := s.repo.ListConvertConfigs(ctx)
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
	rates := DefaultConvertRates()
	for _, item := range convertConfigs {
		if item.Status != 1 || item.ConvertRate <= 0 {
			continue
		}
		switch item.AttrType {
		case string(AttrHPMax):
			rates.HPMax = item.ConvertRate
		case string(AttrATK):
			rates.ATK = item.ConvertRate
		case string(AttrSPD):
			rates.SPD = item.ConvertRate
		case string(AttrMANA):
			rates.MANA = item.ConvertRate
		case string(AttrDEF):
			rates.DEF = item.ConvertRate
		}
	}

	s.mu.Lock()
	s.levelConfigs = levelMap
	s.convertRates = rates
	s.mu.Unlock()
	return nil
}

// ApplyExp 在服务端结算宠物经验并处理连升、发点与战斗属性重算。
func (s *Service) ApplyExp(ctx context.Context, playerID uint64, petUID uint64, expGain uint64, hp uint32) (*ExpApplyResult, error) {
	if s.repo == nil || playerID == 0 || petUID == 0 || expGain == 0 {
		return nil, ErrInvalidAllocateInput
	}
	state, err := s.repo.LoadProgressionState(ctx, playerID, petUID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, ErrPetProgressionNotFound
	}

	result := ExpApplyResult{
		Level:          state.Level,
		Exp:            state.Exp + expGain,
		FreeAttrPoints: state.FreeAttrPoints,
		HP:             hp,
	}
	for result.Level < MaxPetLevel {
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
	if result.Level >= MaxPetLevel {
		result.Level = MaxPetLevel
	}
	result.ExpToNext = s.ExpToNext(result.Level, result.Exp)

	state.Level = result.Level
	state.Exp = result.Exp
	state.FreeAttrPoints = result.FreeAttrPoints
	combat := ClampFormulaCombatStats(RecalculateCombatStats(*state, s.convertRatesSnapshot()))
	result.Combat = combat
	if hp > 0 && hp > combat.HPMax {
		hp = combat.HPMax
	}
	if hp > 0 {
		result.HP = hp
	} else if state.HP > 0 {
		result.HP = minUint32(state.HP, combat.HPMax)
	} else {
		result.HP = combat.HPMax
	}

	if err := s.repo.SaveExpProgression(ctx, playerID, petUID, result, combat, result.HP); err != nil {
		return nil, err
	}
	return &result, nil
}

// AllocateAttrPoints 校验并持久化宠物主动加点。
func (s *Service) AllocateAttrPoints(ctx context.Context, playerID uint64, petUID uint64, delta ManualAllocatedPoints) (*ProgressionState, error) {
	if s.repo == nil || playerID == 0 || petUID == 0 || delta.Total() == 0 {
		return nil, ErrInvalidAllocateInput
	}
	state, err := s.repo.LoadProgressionState(ctx, playerID, petUID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, ErrPetProgressionNotFound
	}
	if delta.Total() > state.FreeAttrPoints {
		return nil, ErrInsufficientAttrPoints
	}

	freeBefore := state.FreeAttrPoints
	freeAfter := freeBefore - delta.Total()
	state.ManualPoints.HP += delta.HP
	state.ManualPoints.ATK += delta.ATK
	state.ManualPoints.SPD += delta.SPD
	state.ManualPoints.MANA += delta.MANA
	state.ManualPoints.DEF += delta.DEF
	state.FreeAttrPoints = freeAfter
	combat := ClampFormulaCombatStats(RecalculateCombatStats(*state, s.convertRatesSnapshot()))
	hp := minUint32(state.HP, combat.HPMax)
	if hp == 0 {
		hp = combat.HPMax
	}
	state.Combat = combat
	state.HP = hp
	if err := s.repo.SaveAttrAllocation(ctx, playerID, petUID, delta, freeBefore, freeAfter, combat); err != nil {
		return nil, err
	}
	return state, nil
}

// ExpToNext 根据缓存配置计算当前等级距离下一级还需要的经验。
func (s *Service) ExpToNext(level uint32, exp uint64) uint64 {
	return s.expToNext(level, exp)
}

// BuildInitialCombatStats 为新发放宠物按 1 级、零手动点初始化战斗属性。
func (s *Service) BuildInitialCombatStats(profile string, aptitudes GrowthAptitudes) CombatStats {
	state := ProgressionState{
		Level:           1,
		AptitudeProfile: profile,
		Aptitudes:       aptitudes,
	}
	return ClampFormulaCombatStats(RecalculateCombatStats(state, s.convertRatesSnapshot()))
}

func (s *Service) levelConfig(level uint32) *LevelConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if level == 0 || level > MaxPetLevel {
		return nil
	}
	config, ok := s.levelConfigs[level]
	if !ok {
		return nil
	}
	copied := config
	return &copied
}

func (s *Service) convertRatesSnapshot() ConvertRates {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.convertRates == (ConvertRates{}) {
		return DefaultConvertRates()
	}
	return s.convertRates
}

func (s *Service) expToNext(level uint32, exp uint64) uint64 {
	if level >= MaxPetLevel {
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

func minUint32(a uint32, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

// ListAdminLevelConfigs 返回后台可编辑的宠物等级配置列表。
func (s *Service) ListAdminLevelConfigs(ctx context.Context) ([]LevelConfig, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.ListLevelConfigs(ctx)
}

// ListAdminConvertConfigs 返回后台可编辑的宠物资质转化率配置列表。
func (s *Service) ListAdminConvertConfigs(ctx context.Context) ([]ConvertConfig, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.ListConvertConfigs(ctx)
}

// UpsertAdminLevelConfig 更新单条宠物等级配置并刷新运行时缓存。
func (s *Service) UpsertAdminLevelConfig(ctx context.Context, level uint32, input AdminUpsertLevelConfigInput) (*LevelConfig, error) {
	if s.repo == nil || level == 0 || level > MaxPetLevel {
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

// UpsertAdminConvertConfig 更新单项宠物资质转化率并刷新运行时缓存。
func (s *Service) UpsertAdminConvertConfig(ctx context.Context, attrType string, input AdminUpsertConvertConfigInput) (*ConvertConfig, error) {
	if s.repo == nil || strings.TrimSpace(attrType) == "" || input.ConvertRate <= 0 {
		return nil, ErrInvalidAllocateInput
	}
	updated, err := s.repo.UpsertConvertConfig(ctx, attrType, input)
	if err != nil {
		return nil, err
	}
	if err := s.RefreshRuntimeCache(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

// RecalculateAllPetCombatStats 按当前成长公式批量重算宠物战斗属性，供迁移后或改配置后运维使用。
func (s *Service) RecalculateAllPetCombatStats(ctx context.Context, input AdminRecalculateCombatStatsInput) (*AdminRecalculateCombatStatsResult, error) {
	if s.repo == nil {
		return nil, ErrInvalidAllocateInput
	}
	targets, err := s.repo.ListProgressionTargets(ctx, input.PlayerID, input.PetUID)
	if err != nil {
		return nil, err
	}
	rates := s.convertRatesSnapshot()
	var updatedCount uint32
	for _, target := range targets {
		state, err := s.repo.LoadProgressionState(ctx, target.PlayerID, target.PetUID)
		if err != nil {
			return nil, err
		}
		if state == nil {
			continue
		}
		combat := ClampFormulaCombatStats(RecalculateCombatStats(*state, rates))
		hp := state.HP
		if hp == 0 || hp > combat.HPMax {
			hp = combat.HPMax
		}
		if err := s.repo.SaveRecalculatedCombatStats(ctx, target.PlayerID, target.PetUID, combat, hp); err != nil {
			return nil, err
		}
		updatedCount++
	}
	return &AdminRecalculateCombatStatsResult{UpdatedCount: updatedCount}, nil
}
