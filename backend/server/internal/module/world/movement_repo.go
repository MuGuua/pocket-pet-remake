package world

import "context"

// MovementStateRepository 管理在线玩家的短时权威移动状态；永久位置仍由玩家仓储写入 PostgreSQL。
type MovementStateRepository interface {
	Load(ctx context.Context, playerID uint64) (*MovementState, error)
	Initialize(ctx context.Context, state MovementState) error
	CompareAndSet(ctx context.Context, expectedMoveSeq uint32, state MovementState) error
	Delete(ctx context.Context, playerID uint64) error
}

// MovementConfigRepository 从 PostgreSQL读取已启用的世界移动配置。
type MovementConfigRepository interface {
	GetMovementConfig(ctx context.Context) (MovementConfig, error)
}
