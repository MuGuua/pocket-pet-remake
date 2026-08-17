package wstransport

import (
	"context"
	"fmt"
	"time"

	"pocket-pet-remake/server/internal/module/world"
)

const movementFinalPersistenceTimeout = 3 * time.Second

// movementFinalPersister 暴露关键玩法节点所需的最小位置写回能力。
// 传输层只提交服务端权威移动状态，不读取数据库版本，也不复制 PostgreSQL 条件更新逻辑。
type movementFinalPersister interface {
	PersistPlayerMovement(ctx context.Context, playerID uint64) error
	PersistMovementState(ctx context.Context, state world.MovementState) (bool, error)
}

// persistPlayerMovementWithTimeout 使用独立短超时上下文写回玩家当前 Redis 权威位置。
// 请求上下文可能在响应结束或连接关闭后立即取消，因此生命周期写回不能继续复用它。
func persistPlayerMovementWithTimeout(persister movementFinalPersister, playerID uint64) error {
	if persister == nil || playerID == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), movementFinalPersistenceTimeout)
	defer cancel()
	if err := persister.PersistPlayerMovement(ctx, playerID); err != nil {
		return fmt.Errorf("persist final movement for player %d: %w", playerID, err)
	}
	return nil
}
