package playerskill

import "context"

// Repository 定义玩家武器技能学习进度的数据访问边界。
type Repository interface {
	ListByPlayerID(ctx context.Context, playerID uint64) ([]Progress, error)
	UpsertBattleUpdates(ctx context.Context, playerID uint64, updates []BattleUseUpdate) error
}
