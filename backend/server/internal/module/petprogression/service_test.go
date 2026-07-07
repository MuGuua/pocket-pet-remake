package petprogression

import (
	"context"
	"testing"
)

// TestServiceApplyExpLevelsUp 验证经验满额时会升级并发放自由点。
func TestServiceApplyExpLevelsUp(t *testing.T) {
	repo := &memoryRepo{
		levelConfigs: []LevelConfig{
			{Level: 1, ExpRequired: 100, AttrPoints: 1, Status: 1},
			{Level: 2, ExpRequired: 100, AttrPoints: 1, Status: 1},
		},
		state: &ProgressionState{
			PlayerID:        1,
			PetUID:          10,
			Level:           1,
			AptitudeProfile: AptitudeProfileNormal,
			Aptitudes: GrowthAptitudes{
				BaseHPApt:  100,
				BaseATKApt: 100,
			},
		},
	}
	service := NewService(repo)
	if err := service.RefreshRuntimeCache(context.Background()); err != nil {
		t.Fatalf("refresh cache: %v", err)
	}

	result, err := service.ApplyExp(context.Background(), 1, 10, 150, 0)
	if err != nil {
		t.Fatalf("apply exp: %v", err)
	}
	if result.Level != 2 {
		t.Fatalf("level = %d, want 2", result.Level)
	}
	if result.LevelUpCount != 1 {
		t.Fatalf("level_up_count = %d, want 1", result.LevelUpCount)
	}
	if result.FreeAttrPoints != 1 {
		t.Fatalf("free_attr_points = %d, want 1", result.FreeAttrPoints)
	}
	if result.Exp != 50 {
		t.Fatalf("exp = %d, want 50", result.Exp)
	}
}

// TestServiceAllocateAttrPointsRecalculatesCombatStats 验证宠物加点后会立即按资质公式重算战斗属性，
// 客户端和战斗入口后续只需要读取服务端返回的权威快照。
func TestServiceAllocateAttrPointsRecalculatesCombatStats(t *testing.T) {
	repo := &memoryRepo{
		levelConfigs: []LevelConfig{
			{Level: 1, ExpRequired: 100, AttrPoints: 1, Status: 1},
		},
		state: &ProgressionState{
			PlayerID:        1,
			PetUID:          10,
			Level:           1,
			FreeAttrPoints:  1,
			AptitudeProfile: AptitudeProfileNormal,
			Aptitudes: GrowthAptitudes{
				BaseATKApt: 10000,
			},
		},
	}
	service := NewService(repo)
	if err := service.RefreshRuntimeCache(context.Background()); err != nil {
		t.Fatalf("refresh cache: %v", err)
	}

	result, err := service.AllocateAttrPoints(context.Background(), 1, 10, ManualAllocatedPoints{ATK: 1})
	if err != nil {
		t.Fatalf("allocate attr points: %v", err)
	}
	if result.FreeAttrPoints != 0 {
		t.Fatalf("free_attr_points = %d, want 0", result.FreeAttrPoints)
	}
	if result.ManualPoints.ATK != 1 {
		t.Fatalf("manual atk points = %d, want 1", result.ManualPoints.ATK)
	}
	if result.Combat.ATK == 0 {
		t.Fatal("combat ATK = 0, want recalculated value after manual allocation")
	}
	if repo.state.Combat.ATK != result.Combat.ATK {
		t.Fatalf("persisted combat ATK = %d, want %d", repo.state.Combat.ATK, result.Combat.ATK)
	}
}

type memoryRepo struct {
	levelConfigs []LevelConfig
	state        *ProgressionState
}

func (r *memoryRepo) ListLevelConfigs(_ context.Context) ([]LevelConfig, error) {
	return append([]LevelConfig(nil), r.levelConfigs...), nil
}

func (r *memoryRepo) ListConvertConfigs(_ context.Context) ([]ConvertConfig, error) {
	rates := DefaultConvertRates()
	return []ConvertConfig{
		{AttrType: string(AttrHPMax), ConvertRate: rates.HPMax, Status: 1},
		{AttrType: string(AttrATK), ConvertRate: rates.ATK, Status: 1},
		{AttrType: string(AttrSPD), ConvertRate: rates.SPD, Status: 1},
		{AttrType: string(AttrMANA), ConvertRate: rates.MANA, Status: 1},
		{AttrType: string(AttrDEF), ConvertRate: rates.DEF, Status: 1},
	}, nil
}

func (r *memoryRepo) LoadProgressionState(_ context.Context, playerID uint64, petUID uint64) (*ProgressionState, error) {
	if r.state == nil || r.state.PlayerID != playerID || r.state.PetUID != petUID {
		return nil, nil
	}
	copied := *r.state
	return &copied, nil
}

func (r *memoryRepo) SaveExpProgression(_ context.Context, _ uint64, _ uint64, result ExpApplyResult, combat CombatStats, hp uint32) error {
	r.state.Level = result.Level
	r.state.Exp = result.Exp
	r.state.FreeAttrPoints = result.FreeAttrPoints
	r.state.Combat = combat
	r.state.HP = hp
	return nil
}

func (r *memoryRepo) SaveAttrAllocation(_ context.Context, _ uint64, _ uint64, delta ManualAllocatedPoints, _ uint32, freeAfter uint32, combat CombatStats) error {
	r.state.ManualPoints.HP += delta.HP
	r.state.ManualPoints.ATK += delta.ATK
	r.state.ManualPoints.SPD += delta.SPD
	r.state.ManualPoints.MANA += delta.MANA
	r.state.ManualPoints.DEF += delta.DEF
	r.state.FreeAttrPoints = freeAfter
	r.state.Combat = combat
	return nil
}

func (r *memoryRepo) UpsertLevelConfig(_ context.Context, level uint32, input AdminUpsertLevelConfigInput) (*LevelConfig, error) {
	for index := range r.levelConfigs {
		if r.levelConfigs[index].Level != level {
			continue
		}
		r.levelConfigs[index].ExpRequired = input.ExpRequired
		r.levelConfigs[index].AttrPoints = input.AttrPoints
		r.levelConfigs[index].Status = input.Status
		copied := r.levelConfigs[index]
		return &copied, nil
	}
	item := LevelConfig{
		Level:       level,
		ExpRequired: input.ExpRequired,
		AttrPoints:  input.AttrPoints,
		Status:      input.Status,
	}
	r.levelConfigs = append(r.levelConfigs, item)
	return &item, nil
}

func (r *memoryRepo) UpsertConvertConfig(_ context.Context, attrType string, input AdminUpsertConvertConfigInput) (*ConvertConfig, error) {
	rates := DefaultConvertRates()
	switch attrType {
	case string(AttrHPMax):
		rates.HPMax = input.ConvertRate
	case string(AttrATK):
		rates.ATK = input.ConvertRate
	case string(AttrSPD):
		rates.SPD = input.ConvertRate
	case string(AttrMANA):
		rates.MANA = input.ConvertRate
	case string(AttrDEF):
		rates.DEF = input.ConvertRate
	default:
		return nil, ErrConvertConfigNotFound
	}
	return &ConvertConfig{
		AttrType:    attrType,
		ConvertRate: input.ConvertRate,
		Status:      input.Status,
	}, nil
}

func (r *memoryRepo) ListProgressionTargets(_ context.Context, playerID uint64, petUID uint64) ([]PetProgressionTarget, error) {
	if r.state == nil {
		return nil, nil
	}
	if playerID != 0 && r.state.PlayerID != playerID {
		return nil, nil
	}
	if petUID != 0 && r.state.PetUID != petUID {
		return nil, nil
	}
	return []PetProgressionTarget{{
		PlayerID: r.state.PlayerID,
		PetUID:   r.state.PetUID,
	}}, nil
}

func (r *memoryRepo) SaveRecalculatedCombatStats(_ context.Context, _ uint64, _ uint64, combat CombatStats, hp uint32) error {
	if r.state == nil {
		return ErrPetProgressionNotFound
	}
	r.state.Combat = combat
	r.state.HP = hp
	return nil
}
