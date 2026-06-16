package progression

import "context"

// Repository 负责成长配置读取与玩家成长状态持久化。
type Repository interface {
	ListLevelConfigs(ctx context.Context) ([]LevelConfig, error)
	ListAttrConvertConfigs(ctx context.Context) ([]AttrConvertConfig, error)
	GetLevelConfig(ctx context.Context, level uint32) (*LevelConfig, error)
	UpsertLevelConfig(ctx context.Context, level uint32, input AdminUpsertLevelConfigInput) (*LevelConfig, error)
	UpsertAttrConvertConfig(ctx context.Context, id uint64, input AdminUpsertAttrConvertInput) (*AttrConvertConfig, error)
	LoadProgressionState(ctx context.Context, playerID uint64) (*ProgressionState, error)
	SaveExpProgression(ctx context.Context, playerID uint64, result ExpApplyResult, baseCombat BaseCombatStats, combatBonus CombatBonus) error
	SaveAttrAllocation(ctx context.Context, playerID uint64, delta AttrAllocationDelta, freeBefore uint32, freeAfter uint32, combatBonus CombatBonus) error
}
