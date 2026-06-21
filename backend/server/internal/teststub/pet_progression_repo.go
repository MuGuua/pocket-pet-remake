package teststub

import (
	"context"
	"fmt"

	"pocket-pet-remake/server/internal/module/petprogression"
)

// NewPetProgressionRepository 提供内存版宠物成长仓储，供 WS/单元测试使用。
func NewPetProgressionRepository() *PetProgressionRepository {
	return &PetProgressionRepository{
		levelConfigs: defaultPetLevelConfigs(),
		convertRates: petprogression.DefaultConvertRates(),
		states:       map[string]*petprogression.ProgressionState{},
	}
}

// PetProgressionRepository 在内存中模拟宠物成长配置与状态持久化。
type PetProgressionRepository struct {
	levelConfigs []petprogression.LevelConfig
	convertRates petprogression.ConvertRates
	states       map[string]*petprogression.ProgressionState
}

func defaultPetLevelConfigs() []petprogression.LevelConfig {
	items := make([]petprogression.LevelConfig, 0, 100)
	for level := uint32(1); level <= petprogression.MaxPetLevel; level++ {
		expRequired := uint64(100)
		attrPoints := uint32(1)
		if level >= petprogression.MaxPetLevel {
			expRequired = 0
			attrPoints = 0
		}
		items = append(items, petprogression.LevelConfig{
			Level:       level,
			ExpRequired: expRequired,
			AttrPoints:  attrPoints,
			Status:      1,
		})
	}
	return items
}

func petProgressionStateKey(playerID uint64, petUID uint64) string {
	return fmt.Sprintf("%d:%d", playerID, petUID)
}

// ListLevelConfigs 返回内存中的等级配置。
func (r *PetProgressionRepository) ListLevelConfigs(_ context.Context) ([]petprogression.LevelConfig, error) {
	return append([]petprogression.LevelConfig(nil), r.levelConfigs...), nil
}

// ListConvertConfigs 返回内存中的转化率配置。
func (r *PetProgressionRepository) ListConvertConfigs(_ context.Context) ([]petprogression.ConvertConfig, error) {
	rates := r.convertRates
	if rates == (petprogression.ConvertRates{}) {
		rates = petprogression.DefaultConvertRates()
	}
	return []petprogression.ConvertConfig{
		{AttrType: string(petprogression.AttrHPMax), ConvertRate: rates.HPMax, Status: 1},
		{AttrType: string(petprogression.AttrATK), ConvertRate: rates.ATK, Status: 1},
		{AttrType: string(petprogression.AttrSPD), ConvertRate: rates.SPD, Status: 1},
		{AttrType: string(petprogression.AttrMANA), ConvertRate: rates.MANA, Status: 1},
		{AttrType: string(petprogression.AttrDEF), ConvertRate: rates.DEF, Status: 1},
	}, nil
}

// LoadProgressionState 从内存读取宠物成长快照；若不存在则按 1 级默认初始化。
func (r *PetProgressionRepository) LoadProgressionState(_ context.Context, playerID uint64, petUID uint64) (*petprogression.ProgressionState, error) {
	key := petProgressionStateKey(playerID, petUID)
	if state, ok := r.states[key]; ok {
		copied := *state
		return &copied, nil
	}
	return &petprogression.ProgressionState{
		PlayerID:        playerID,
		PetUID:          petUID,
		Level:           1,
		AptitudeProfile: petprogression.AptitudeProfileNormal,
		Aptitudes: petprogression.GrowthAptitudes{
			BaseHPApt:   10,
			BaseATKApt:  10,
			BaseDEFApt:  10,
			BaseSPDApt:  10,
			BaseMANAApt: 10,
		},
	}, nil
}

// SaveExpProgression 持久化经验结算结果到内存。
func (r *PetProgressionRepository) SaveExpProgression(_ context.Context, playerID uint64, petUID uint64, result petprogression.ExpApplyResult, combat petprogression.CombatStats, hp uint32) error {
	key := petProgressionStateKey(playerID, petUID)
	state, err := r.LoadProgressionState(context.Background(), playerID, petUID)
	if err != nil {
		return err
	}
	state.Level = result.Level
	state.Exp = result.Exp
	state.FreeAttrPoints = result.FreeAttrPoints
	state.Combat = combat
	state.HP = hp
	r.states[key] = state
	return nil
}

// SaveAttrAllocation 持久化主动加点结果到内存。
func (r *PetProgressionRepository) SaveAttrAllocation(_ context.Context, playerID uint64, petUID uint64, delta petprogression.ManualAllocatedPoints, _ uint32, freeAfter uint32, combat petprogression.CombatStats) error {
	key := petProgressionStateKey(playerID, petUID)
	state, err := r.LoadProgressionState(context.Background(), playerID, petUID)
	if err != nil {
		return err
	}
	state.ManualPoints.HP += delta.HP
	state.ManualPoints.ATK += delta.ATK
	state.ManualPoints.SPD += delta.SPD
	state.ManualPoints.MANA += delta.MANA
	state.ManualPoints.DEF += delta.DEF
	state.FreeAttrPoints = freeAfter
	state.Combat = combat
	if state.HP > combat.HPMax {
		state.HP = combat.HPMax
	}
	r.states[key] = state
	return nil
}

func (r *PetProgressionRepository) UpsertLevelConfig(_ context.Context, level uint32, input petprogression.AdminUpsertLevelConfigInput) (*petprogression.LevelConfig, error) {
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
	item := petprogression.LevelConfig{
		Level:       level,
		ExpRequired: input.ExpRequired,
		AttrPoints:  input.AttrPoints,
		Status:      input.Status,
	}
	r.levelConfigs = append(r.levelConfigs, item)
	return &item, nil
}

func (r *PetProgressionRepository) UpsertConvertConfig(_ context.Context, attrType string, input petprogression.AdminUpsertConvertConfigInput) (*petprogression.ConvertConfig, error) {
	switch attrType {
	case string(petprogression.AttrHPMax):
		r.convertRates.HPMax = input.ConvertRate
	case string(petprogression.AttrATK):
		r.convertRates.ATK = input.ConvertRate
	case string(petprogression.AttrSPD):
		r.convertRates.SPD = input.ConvertRate
	case string(petprogression.AttrMANA):
		r.convertRates.MANA = input.ConvertRate
	case string(petprogression.AttrDEF):
		r.convertRates.DEF = input.ConvertRate
	default:
		return nil, petprogression.ErrConvertConfigNotFound
	}
	return &petprogression.ConvertConfig{
		AttrType:    attrType,
		ConvertRate: input.ConvertRate,
		Status:      input.Status,
	}, nil
}

func (r *PetProgressionRepository) ListProgressionTargets(_ context.Context, playerID uint64, petUID uint64) ([]petprogression.PetProgressionTarget, error) {
	items := make([]petprogression.PetProgressionTarget, 0, len(r.states))
	for key, state := range r.states {
		if state == nil {
			continue
		}
		if playerID != 0 && state.PlayerID != playerID {
			continue
		}
		if petUID != 0 && state.PetUID != petUID {
			continue
		}
		items = append(items, petprogression.PetProgressionTarget{
			PlayerID: state.PlayerID,
			PetUID:   state.PetUID,
		})
		_ = key
	}
	return items, nil
}

func (r *PetProgressionRepository) SaveRecalculatedCombatStats(_ context.Context, playerID uint64, petUID uint64, combat petprogression.CombatStats, hp uint32) error {
	key := petProgressionStateKey(playerID, petUID)
	state, ok := r.states[key]
	if !ok || state == nil {
		return petprogression.ErrPetProgressionNotFound
	}
	state.Combat = combat
	state.HP = hp
	return nil
}

// SeedProgressionState 供测试注入指定成长快照。
func (r *PetProgressionRepository) SeedProgressionState(playerID uint64, petUID uint64, state petprogression.ProgressionState) {
	state.PlayerID = playerID
	state.PetUID = petUID
	r.states[petProgressionStateKey(playerID, petUID)] = &state
}
