package teststub

import (
	"context"
	"fmt"

	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/progression"
)

// NewProgressionRepository 返回与内存玩家仓储联动的成长配置仓储。
func NewProgressionRepository(players *PlayerRepository) *ProgressionRepository {
	return &ProgressionRepository{
		players: players,
		levelConfigs: []progression.LevelConfig{
			{Level: 1, ExpRequired: 100, AttrPoints: 5, BonusATK: 7, BonusHPMax: 38, BonusSPD: 2, BonusMANA: 1, Status: 1},
			{Level: 2, ExpRequired: 173, AttrPoints: 5, BonusATK: 7, BonusHPMax: 38, BonusSPD: 2, BonusMANA: 1, Status: 1},
			{Level: 3, ExpRequired: 254, AttrPoints: 5, BonusATK: 7, BonusHPMax: 38, BonusSPD: 2, BonusMANA: 1, Status: 1},
			{Level: 4, ExpRequired: 341, AttrPoints: 5, BonusATK: 7, BonusHPMax: 38, BonusSPD: 2, BonusMANA: 1, Status: 1},
			{Level: 5, ExpRequired: 435, AttrPoints: 5, BonusATK: 7, BonusHPMax: 38, BonusSPD: 2, BonusMANA: 1, Status: 1},
		},
		convertRules: []progression.AttrConvertConfig{
			{ID: 1, SourceAttr: progression.SourceAttrStrength, TargetAttr: "atk", ConvertRate: 3, Status: 1},
			{ID: 2, SourceAttr: progression.SourceAttrVitality, TargetAttr: "hp_max", ConvertRate: 50, Status: 1},
			{ID: 3, SourceAttr: progression.SourceAttrVitality, TargetAttr: "def", ConvertRate: 2, Status: 1},
			{ID: 4, SourceAttr: progression.SourceAttrAgility, TargetAttr: "spd", ConvertRate: 2, Status: 1},
			{ID: 5, SourceAttr: progression.SourceAttrAgility, TargetAttr: "dodge_pct", ConvertRate: 1, Status: 1},
			{ID: 6, SourceAttr: progression.SourceAttrMind, TargetAttr: "mana", ConvertRate: 4, Status: 1},
		},
	}
}

// ProgressionRepository 为测试环境提供可运行的玩家成长配置与状态写回能力。
type ProgressionRepository struct {
	players      *PlayerRepository
	levelConfigs []progression.LevelConfig
	convertRules []progression.AttrConvertConfig
}

func (r *ProgressionRepository) ListLevelConfigs(_ context.Context) ([]progression.LevelConfig, error) {
	return append([]progression.LevelConfig(nil), r.levelConfigs...), nil
}

func (r *ProgressionRepository) ListAttrConvertConfigs(_ context.Context) ([]progression.AttrConvertConfig, error) {
	return append([]progression.AttrConvertConfig(nil), r.convertRules...), nil
}

func (r *ProgressionRepository) GetLevelConfig(_ context.Context, level uint32) (*progression.LevelConfig, error) {
	for _, item := range r.levelConfigs {
		if item.Level == level {
			copied := item
			return &copied, nil
		}
	}
	return nil, progression.ErrLevelConfigNotFound
}

func (r *ProgressionRepository) UpsertLevelConfig(_ context.Context, level uint32, input progression.AdminUpsertLevelConfigInput) (*progression.LevelConfig, error) {
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
	return nil, progression.ErrLevelConfigNotFound
}

func (r *ProgressionRepository) UpsertAttrConvertConfig(_ context.Context, id uint64, input progression.AdminUpsertAttrConvertInput) (*progression.AttrConvertConfig, error) {
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
	return nil, progression.ErrConvertConfigNotFound
}

func (r *ProgressionRepository) LoadProgressionState(_ context.Context, playerID uint64) (*progression.ProgressionState, error) {
	if r.players == nil {
		return nil, fmt.Errorf("player progression state not found")
	}
	r.players.mu.RLock()
	defer r.players.mu.RUnlock()
	current, ok := r.players.players[playerID]
	if !ok {
		return nil, fmt.Errorf("player progression state not found")
	}
	return &progression.ProgressionState{
		Level:          current.Level,
		Exp:            current.Exp,
		FreeAttrPoints: current.FreeAttrPoints,
		Allocated: progression.AllocatedAttrs{
			Strength: current.Strength,
			Vitality: current.Vitality,
			Agility:  current.Agility,
			Mind:     current.Mind,
		},
		BaseCombat: progression.BaseCombatStats{
			HPMax:    current.BaseHPMax,
			ATK:      current.BaseATK,
			DEF:      current.BaseDEF,
			SPD:      current.BaseSPD,
			MANA:     current.BaseMANA,
			HitPct:   current.BaseHitPct,
			DodgePct: current.BaseDodgePct,
		},
	}, nil
}

func (r *ProgressionRepository) SaveExpProgression(_ context.Context, playerID uint64, result progression.ExpApplyResult, baseCombat progression.BaseCombatStats, combatBonus progression.CombatBonus) error {
	return r.savePlayerProgression(playerID, result.Level, result.Exp, result.FreeAttrPoints, nil, baseCombat, combatBonus)
}

func (r *ProgressionRepository) SaveAttrAllocation(_ context.Context, playerID uint64, delta progression.AttrAllocationDelta, _ uint32, freeAfter uint32, combatBonus progression.CombatBonus) error {
	baseCombat := progression.BaseCombatStats{}
	if r.players != nil {
		r.players.mu.RLock()
		if current, ok := r.players.players[playerID]; ok {
			baseCombat = progression.BaseCombatStats{
				HPMax: current.BaseHPMax, ATK: current.BaseATK, DEF: current.BaseDEF,
				SPD: current.BaseSPD, MANA: current.BaseMANA, HitPct: current.BaseHitPct, DodgePct: current.BaseDodgePct,
			}
		}
		r.players.mu.RUnlock()
	}
	return r.savePlayerProgression(playerID, 0, 0, freeAfter, &delta, baseCombat, combatBonus)
}

func (r *ProgressionRepository) savePlayerProgression(
	playerID uint64,
	level uint32,
	exp uint64,
	freeAttrPoints uint32,
	delta *progression.AttrAllocationDelta,
	baseCombat progression.BaseCombatStats,
	combatBonus progression.CombatBonus,
) error {
	if r.players == nil {
		return fmt.Errorf("player progression state not found")
	}
	r.players.mu.Lock()
	defer r.players.mu.Unlock()
	current, ok := r.players.players[playerID]
	if !ok {
		return fmt.Errorf("player progression state not found")
	}
	if level > 0 {
		current.Level = level
		current.Exp = exp
		current.BaseHPMax = baseCombat.HPMax
		current.BaseATK = baseCombat.ATK
		current.BaseDEF = baseCombat.DEF
		current.BaseSPD = baseCombat.SPD
		current.BaseMANA = baseCombat.MANA
		current.BaseHitPct = baseCombat.HitPct
		current.BaseDodgePct = baseCombat.DodgePct
	}
	current.FreeAttrPoints = freeAttrPoints
	if delta != nil {
		current.Strength += delta.Strength
		current.Vitality += delta.Vitality
		current.Agility += delta.Agility
		current.Mind += delta.Mind
	}
	current.HPMax = progression.FinalCombatValue(current.BaseHPMax, combatBonus.HPMax)
	current.ATK = progression.FinalCombatValue(current.BaseATK, combatBonus.ATK)
	current.DEF = progression.FinalCombatValue(current.BaseDEF, combatBonus.DEF)
	current.SPD = progression.FinalCombatValue(current.BaseSPD, combatBonus.SPD)
	current.MANA = progression.FinalCombatValue(current.BaseMANA, combatBonus.MANA)
	current.HitPct = progression.FinalCombatValue(current.BaseHitPct, combatBonus.HitPct)
	current.DodgePct = progression.FinalCombatValue(current.BaseDodgePct, combatBonus.DodgePct)
	r.players.players[playerID] = current
	return nil
}

// NewTestPlayerService 为测试组装带成长能力的玩家服务。
func NewTestPlayerService() *player.Service {
	playerRepo := NewPlayerRepository()
	progressionRepo := NewProgressionRepository(playerRepo)
	progressionService := progression.NewService(progressionRepo)
	_ = progressionService.RefreshRuntimeCache(context.Background())
	return player.NewService(playerRepo, nil, progressionService)
}
