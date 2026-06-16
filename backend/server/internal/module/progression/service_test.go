package progression

import (
	"context"
	"testing"
)

func TestApplyExpAppliesLevelCombatBonus(t *testing.T) {
	repo := newMemoryRepo()
	service := NewService(repo)
	if err := service.RefreshRuntimeCache(t.Context()); err != nil {
		t.Fatalf("RefreshRuntimeCache() error = %v", err)
	}

	repo.states[1] = &ProgressionState{
		Level:          1,
		Exp:            0,
		FreeAttrPoints: 0,
		BaseCombat: BaseCombatStats{
			HPMax: 100, ATK: 24, DEF: 12, SPD: 18, MANA: 20, HitPct: 10, DodgePct: 6,
		},
	}

	result, err := service.ApplyExp(t.Context(), 1, 100)
	if err != nil {
		t.Fatalf("ApplyExp() error = %v", err)
	}
	if result.LevelUpCount != 1 {
		t.Fatalf("LevelUpCount = %d, want 1", result.LevelUpCount)
	}
	if result.CombatBonusGain.ATK != 7 {
		t.Fatalf("CombatBonusGain.ATK = %d, want 7", result.CombatBonusGain.ATK)
	}
	if result.CombatBonusGain.HPMax != 38 {
		t.Fatalf("CombatBonusGain.HPMax = %d, want 38", result.CombatBonusGain.HPMax)
	}
	if result.CombatBonusGain.SPD != 2 {
		t.Fatalf("CombatBonusGain.SPD = %d, want 2", result.CombatBonusGain.SPD)
	}
	if result.CombatBonusGain.MANA != 1 {
		t.Fatalf("CombatBonusGain.MANA = %d, want 1", result.CombatBonusGain.MANA)
	}
	if result.Level != 2 {
		t.Fatalf("Level = %d, want 2", result.Level)
	}
	state := repo.states[1]
	if state.BaseCombat.ATK != 31 {
		t.Fatalf("BaseCombat.ATK = %d, want 31 after one level-up bonus", state.BaseCombat.ATK)
	}
	if state.BaseCombat.HPMax != 138 {
		t.Fatalf("BaseCombat.HPMax = %d, want 138 after one level-up bonus", state.BaseCombat.HPMax)
	}
	if state.BaseCombat.SPD != 20 {
		t.Fatalf("BaseCombat.SPD = %d, want 20 after one level-up bonus", state.BaseCombat.SPD)
	}
	if state.BaseCombat.MANA != 21 {
		t.Fatalf("BaseCombat.MANA = %d, want 21 after one level-up bonus", state.BaseCombat.MANA)
	}
}

func TestApplyExpLevelUpWithOverflow(t *testing.T) {
	repo := newMemoryRepo()
	service := NewService(repo)
	if err := service.RefreshRuntimeCache(t.Context()); err != nil {
		t.Fatalf("RefreshRuntimeCache() error = %v", err)
	}

	repo.states[1] = &ProgressionState{
		Level:          1,
		Exp:            0,
		FreeAttrPoints: 0,
		BaseCombat: BaseCombatStats{
			HPMax: 100, ATK: 24, DEF: 12, SPD: 18, MANA: 20, HitPct: 10, DodgePct: 6,
		},
	}

	result, err := service.ApplyExp(t.Context(), 1, 251)
	if err != nil {
		t.Fatalf("ApplyExp() error = %v", err)
	}
	if result.LevelUpCount == 0 {
		t.Fatal("LevelUpCount = 0, want > 0")
	}
	if result.Level <= 1 {
		t.Fatalf("Level = %d, want > 1", result.Level)
	}
	if result.Exp == 0 {
		t.Fatal("expected overflow exp to remain after multi level-up")
	}
}

func TestAllocateAttrPointsRecalculatesCombatBonus(t *testing.T) {
	repo := newMemoryRepo()
	service := NewService(repo)
	if err := service.RefreshRuntimeCache(t.Context()); err != nil {
		t.Fatalf("RefreshRuntimeCache() error = %v", err)
	}

	repo.states[1] = &ProgressionState{
		Level:          5,
		Exp:            0,
		FreeAttrPoints: 3,
		BaseCombat: BaseCombatStats{
			HPMax: 100, ATK: 24, DEF: 12, SPD: 18, MANA: 20, HitPct: 10, DodgePct: 6,
		},
	}

	err := service.AllocateAttrPoints(t.Context(), 1, AttrAllocationDelta{Strength: 2, Mind: 1})
	if err != nil {
		t.Fatalf("AllocateAttrPoints() error = %v", err)
	}
	state := repo.states[1]
	if state.FreeAttrPoints != 0 {
		t.Fatalf("FreeAttrPoints = %d, want 0", state.FreeAttrPoints)
	}
	if state.Allocated.Strength != 2 || state.Allocated.Mind != 1 {
		t.Fatalf("allocated attrs = %+v, want strength=2 mind=1", state.Allocated)
	}
}

func TestAllocateAttrPointsRejectsInsufficientPoints(t *testing.T) {
	repo := newMemoryRepo()
	service := NewService(repo)
	if err := service.RefreshRuntimeCache(t.Context()); err != nil {
		t.Fatalf("RefreshRuntimeCache() error = %v", err)
	}
	repo.states[1] = &ProgressionState{Level: 1, FreeAttrPoints: 1}

	err := service.AllocateAttrPoints(t.Context(), 1, AttrAllocationDelta{Strength: 2})
	if err != ErrInsufficientAttrPoints {
		t.Fatalf("AllocateAttrPoints() error = %v, want %v", err, ErrInsufficientAttrPoints)
	}
}

type memoryRepo struct {
	levelConfigs []LevelConfig
	convertRules []AttrConvertConfig
	states       map[uint64]*ProgressionState
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		levelConfigs: []LevelConfig{
			{Level: 1, ExpRequired: 100, AttrPoints: 5, BonusATK: 7, BonusHPMax: 38, BonusSPD: 2, BonusMANA: 1, Status: 1},
			{Level: 2, ExpRequired: 150, AttrPoints: 5, BonusATK: 7, BonusHPMax: 38, BonusSPD: 2, BonusMANA: 1, Status: 1},
			{Level: 3, ExpRequired: 200, AttrPoints: 5, BonusATK: 7, BonusHPMax: 38, BonusSPD: 2, BonusMANA: 1, Status: 1},
		},
		convertRules: []AttrConvertConfig{
			{ID: 1, SourceAttr: SourceAttrStrength, TargetAttr: "atk", ConvertRate: 3, Status: 1},
			{ID: 2, SourceAttr: SourceAttrMind, TargetAttr: "mana", ConvertRate: 4, Status: 1},
		},
		states: map[uint64]*ProgressionState{},
	}
}

func (r *memoryRepo) ListLevelConfigs(_ context.Context) ([]LevelConfig, error) {
	return append([]LevelConfig(nil), r.levelConfigs...), nil
}

func (r *memoryRepo) ListAttrConvertConfigs(_ context.Context) ([]AttrConvertConfig, error) {
	return append([]AttrConvertConfig(nil), r.convertRules...), nil
}

func (r *memoryRepo) GetLevelConfig(_ context.Context, level uint32) (*LevelConfig, error) {
	for _, item := range r.levelConfigs {
		if item.Level == level {
			copied := item
			return &copied, nil
		}
	}
	return nil, ErrLevelConfigNotFound
}

func (r *memoryRepo) UpsertLevelConfig(_ context.Context, level uint32, input AdminUpsertLevelConfigInput) (*LevelConfig, error) {
	for index, item := range r.levelConfigs {
		if item.Level == level {
			r.levelConfigs[index].ExpRequired = input.ExpRequired
			r.levelConfigs[index].AttrPoints = input.AttrPoints
			r.levelConfigs[index].BonusATK = input.BonusATK
			r.levelConfigs[index].BonusHPMax = input.BonusHPMax
			r.levelConfigs[index].BonusSPD = input.BonusSPD
			r.levelConfigs[index].BonusMANA = input.BonusMANA
			r.levelConfigs[index].Status = input.Status
			copied := r.levelConfigs[index]
			return &copied, nil
		}
	}
	return nil, ErrLevelConfigNotFound
}

func (r *memoryRepo) UpsertAttrConvertConfig(_ context.Context, id uint64, input AdminUpsertAttrConvertInput) (*AttrConvertConfig, error) {
	for index, item := range r.convertRules {
		if item.ID == id {
			r.convertRules[index].SourceAttr = input.SourceAttr
			r.convertRules[index].TargetAttr = input.TargetAttr
			r.convertRules[index].ConvertRate = input.ConvertRate
			r.convertRules[index].Status = input.Status
			copied := r.convertRules[index]
			return &copied, nil
		}
	}
	return nil, ErrConvertConfigNotFound
}

func (r *memoryRepo) LoadProgressionState(_ context.Context, playerID uint64) (*ProgressionState, error) {
	state, ok := r.states[playerID]
	if !ok {
		return nil, ErrLevelConfigNotFound
	}
	copied := *state
	return &copied, nil
}

func (r *memoryRepo) SaveExpProgression(_ context.Context, playerID uint64, result ExpApplyResult, baseCombat BaseCombatStats, _ CombatBonus) error {
	state := r.states[playerID]
	state.Level = result.Level
	state.Exp = result.Exp
	state.FreeAttrPoints = result.FreeAttrPoints
	state.BaseCombat = baseCombat
	return nil
}

func (r *memoryRepo) SaveAttrAllocation(_ context.Context, playerID uint64, delta AttrAllocationDelta, _ uint32, freeAfter uint32, _ CombatBonus) error {
	state := r.states[playerID]
	state.FreeAttrPoints = freeAfter
	state.Allocated.Strength += delta.Strength
	state.Allocated.Vitality += delta.Vitality
	state.Allocated.Agility += delta.Agility
	state.Allocated.Mind += delta.Mind
	return nil
}
