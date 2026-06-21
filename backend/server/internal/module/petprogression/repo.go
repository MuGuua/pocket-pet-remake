package petprogression

import "context"

// Repository 负责宠物成长配置读取与实例成长状态持久化。
type Repository interface {
	ListLevelConfigs(ctx context.Context) ([]LevelConfig, error)
	ListConvertConfigs(ctx context.Context) ([]ConvertConfig, error)
	UpsertLevelConfig(ctx context.Context, level uint32, input AdminUpsertLevelConfigInput) (*LevelConfig, error)
	UpsertConvertConfig(ctx context.Context, attrType string, input AdminUpsertConvertConfigInput) (*ConvertConfig, error)
	LoadProgressionState(ctx context.Context, playerID uint64, petUID uint64) (*ProgressionState, error)
	SaveExpProgression(ctx context.Context, playerID uint64, petUID uint64, result ExpApplyResult, combat CombatStats, hp uint32) error
	SaveAttrAllocation(ctx context.Context, playerID uint64, petUID uint64, delta ManualAllocatedPoints, freeBefore uint32, freeAfter uint32, combat CombatStats) error
	ListProgressionTargets(ctx context.Context, playerID uint64, petUID uint64) ([]PetProgressionTarget, error)
	SaveRecalculatedCombatStats(ctx context.Context, playerID uint64, petUID uint64, combat CombatStats, hp uint32) error
}
