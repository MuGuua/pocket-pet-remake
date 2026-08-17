package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"pocket-pet-remake/server/internal/module/world"
)

const movementPersistenceRequeueTimeout = 3 * time.Second

// movementPersistenceSource 暴露批量写回所需的最小 Redis 权威移动状态能力。
// 应用层只依赖领域入口，不直接执行 Redis 命令或拼接运行态键。
type movementPersistenceSource interface {
	ClaimDirtyMovementPlayerIDs(ctx context.Context, limit uint32) ([]uint64, error)
	LoadMovementState(ctx context.Context, playerID uint64) (*world.MovementState, error)
	RequeueDirtyMovementPlayerIDs(ctx context.Context, playerIDs []uint64) error
}

// movementPositionWriter 暴露 PostgreSQL 永久位置写入所需的最小能力。
// 当前由 player.Service 实现；worker 只传递 Redis 权威版本，不在应用层复制 SQL 条件判断。
type movementPositionWriter interface {
	UpdatePositionIfNewer(ctx context.Context, playerID uint64, sceneID uint32, posX int32, posY int32, positionVersion uint64) (bool, error)
}

// movementPersistenceBatchResult 记录单次周期批次的处理数量，供日志与测试确认失败玩家没有静默丢失。
type movementPersistenceBatchResult struct {
	Claimed   int
	Persisted int
	Stale     int
	Requeued  int
}

// movementPersistenceWorker 周期领取 Redis dirty 玩家，把最新整数权威位置写回 PostgreSQL。
// 单个玩家失败不会阻止同批其他玩家写回；所有失败编号会在批次末统一重新入队。
type movementPersistenceWorker struct {
	source    movementPersistenceSource
	writer    movementPositionWriter
	logger    *log.Logger
	interval  time.Duration
	batchSize uint32
}

// newMovementPersistenceWorker 创建写回 worker，并在启动前拒绝缺失依赖或非法周期参数。
func newMovementPersistenceWorker(
	source movementPersistenceSource,
	writer movementPositionWriter,
	logger *log.Logger,
	interval time.Duration,
	batchSize uint32,
) (*movementPersistenceWorker, error) {
	if source == nil {
		return nil, fmt.Errorf("movement persistence source is required")
	}
	if writer == nil {
		return nil, fmt.Errorf("movement position writer is required")
	}
	if interval <= 0 {
		return nil, fmt.Errorf("movement persistence interval must be greater than zero")
	}
	if batchSize == 0 {
		return nil, fmt.Errorf("movement persistence batch size must be greater than zero")
	}
	if logger == nil {
		logger = log.Default()
	}
	return &movementPersistenceWorker{
		source: source, writer: writer, logger: logger,
		interval: interval, batchSize: batchSize,
	}, nil
}

// Run 按配置周期串行执行写回；上一批未完成前不会启动重叠批次。
func (w *movementPersistenceWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := w.flushOnce(ctx)
			if err != nil {
				w.logger.Printf(
					"movement persistence batch failed: claimed=%d persisted=%d stale=%d requeued=%d: %v",
					result.Claimed,
					result.Persisted,
					result.Stale,
					result.Requeued,
					err,
				)
			}
		}
	}
}

// PersistPlayerMovement 读取指定玩家当前 Redis 权威移动状态，并按位置版本写入 PostgreSQL。
// 关键生命周期节点不会领取 dirty 标记，因此写回失败时原标记仍保留给周期 worker 或停服排空重试。
func (w *movementPersistenceWorker) PersistPlayerMovement(ctx context.Context, playerID uint64) error {
	if playerID == 0 {
		return fmt.Errorf("movement persistence player id is required")
	}
	state, err := w.source.LoadMovementState(ctx, playerID)
	if errors.Is(err, world.ErrMovementStateNotFound) {
		// 玩家尚未进入世界时不存在 Redis 移动状态，永久档案本身已经是最新可用位置。
		return nil
	}
	if err != nil {
		return fmt.Errorf("load movement state for player %d: %w", playerID, err)
	}
	if state == nil || state.PlayerID != playerID {
		return fmt.Errorf("load movement state for player %d: invalid state", playerID)
	}
	_, err = w.PersistMovementState(ctx, *state)
	return err
}

// PersistMovementState 把调用方已经确认的权威状态按版本写入 PostgreSQL。
// 返回 false 表示数据库已经持有相同或更高版本，调用方可按自身流程决定将其视为安全 stale 或并发冲突。
func (w *movementPersistenceWorker) PersistMovementState(ctx context.Context, state world.MovementState) (bool, error) {
	if state.PlayerID == 0 {
		return false, fmt.Errorf("movement persistence player id is required")
	}
	applied, err := w.writer.UpdatePositionIfNewer(
		ctx,
		state.PlayerID,
		state.SceneID,
		state.PersistedPos.X,
		state.PersistedPos.Y,
		state.PositionVersion,
	)
	if err != nil {
		return false, fmt.Errorf("persist movement state for player %d: %w", state.PlayerID, err)
	}
	return applied, nil
}

// DrainDirtyMovement 在停服阶段有限排空当前 dirty 集合。
// 失败玩家暂存到排空结束后再统一重入队，避免数据库持续故障时立即领取同一玩家形成无限循环。
func (w *movementPersistenceWorker) DrainDirtyMovement(ctx context.Context) (movementPersistenceBatchResult, error) {
	result := movementPersistenceBatchResult{}
	failedPlayerIDs := make([]uint64, 0)
	failedPlayerSet := make(map[uint64]struct{})
	var drainErr error

	for ctx.Err() == nil {
		playerIDs, err := w.source.ClaimDirtyMovementPlayerIDs(ctx, w.batchSize)
		if err != nil {
			drainErr = errors.Join(drainErr, fmt.Errorf("claim dirty movement players: %w", err))
			break
		}
		if len(playerIDs) == 0 {
			break
		}
		result.Claimed += len(playerIDs)
		batchResult, batchFailedPlayerIDs, batchErr := w.persistPlayerIDs(ctx, playerIDs)
		result.Persisted += batchResult.Persisted
		result.Stale += batchResult.Stale
		drainErr = errors.Join(drainErr, batchErr)
		for _, playerID := range batchFailedPlayerIDs {
			if _, exists := failedPlayerSet[playerID]; exists {
				continue
			}
			failedPlayerSet[playerID] = struct{}{}
			failedPlayerIDs = append(failedPlayerIDs, playerID)
		}
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		drainErr = errors.Join(drainErr, fmt.Errorf("drain dirty movement players: %w", ctxErr))
	}
	if len(failedPlayerIDs) == 0 {
		return result, drainErr
	}
	if requeueErr := w.requeueFailedPlayerIDs(ctx, failedPlayerIDs); requeueErr != nil {
		return result, errors.Join(drainErr, fmt.Errorf("requeue dirty movement players: %w", requeueErr))
	}
	result.Requeued = len(failedPlayerIDs)
	return result, drainErr
}

// flushOnce 原子领取一个有界批次，并逐个读取 Redis 最新状态后按版本写入 PostgreSQL。
// 数据库拒绝相同或更旧版本属于已安全处理的 stale 状态，不应重新入队形成无效重试循环。
func (w *movementPersistenceWorker) flushOnce(ctx context.Context) (movementPersistenceBatchResult, error) {
	playerIDs, err := w.source.ClaimDirtyMovementPlayerIDs(ctx, w.batchSize)
	if err != nil {
		return movementPersistenceBatchResult{}, fmt.Errorf("claim dirty movement players: %w", err)
	}
	result := movementPersistenceBatchResult{Claimed: len(playerIDs)}
	if len(playerIDs) == 0 {
		return result, nil
	}

	persistResult, failedPlayerIDs, batchErr := w.persistPlayerIDs(ctx, playerIDs)
	result.Persisted = persistResult.Persisted
	result.Stale = persistResult.Stale
	if len(failedPlayerIDs) == 0 {
		return result, batchErr
	}
	if requeueErr := w.requeueFailedPlayerIDs(ctx, failedPlayerIDs); requeueErr != nil {
		return result, errors.Join(batchErr, fmt.Errorf("requeue dirty movement players: %w", requeueErr))
	}
	result.Requeued = len(failedPlayerIDs)
	return result, batchErr
}

// persistPlayerIDs 处理已经领取的玩家编号，但不负责 dirty 重入队。
// 周期批次可以立即重入失败编号；停服排空则延迟重入，二者共享完全相同的版本写回判定。
func (w *movementPersistenceWorker) persistPlayerIDs(ctx context.Context, playerIDs []uint64) (movementPersistenceBatchResult, []uint64, error) {
	result := movementPersistenceBatchResult{}
	failedPlayerIDs := make([]uint64, 0)
	var batchErr error
	for _, playerID := range playerIDs {
		state, loadErr := w.source.LoadMovementState(ctx, playerID)
		if loadErr != nil {
			failedPlayerIDs = append(failedPlayerIDs, playerID)
			batchErr = errors.Join(batchErr, fmt.Errorf("load movement state for player %d: %w", playerID, loadErr))
			continue
		}
		if state == nil || state.PlayerID != playerID {
			failedPlayerIDs = append(failedPlayerIDs, playerID)
			batchErr = errors.Join(batchErr, fmt.Errorf("load movement state for player %d: invalid state", playerID))
			continue
		}
		applied, persistErr := w.PersistMovementState(ctx, *state)
		if persistErr != nil {
			failedPlayerIDs = append(failedPlayerIDs, playerID)
			batchErr = errors.Join(batchErr, persistErr)
			continue
		}
		if !applied {
			result.Stale++
			continue
		}
		result.Persisted++
	}
	return result, failedPlayerIDs, batchErr
}

// requeueFailedPlayerIDs 在正常运行时复用批次上下文；若应用正在取消，则给 Redis 重入队保留短暂清理窗口。
// 这样已经被 SPOP 领取的玩家不会仅因优雅停机取消了主上下文而立即丢失 dirty 标记。
func (w *movementPersistenceWorker) requeueFailedPlayerIDs(ctx context.Context, playerIDs []uint64) error {
	if ctx.Err() == nil {
		return w.source.RequeueDirtyMovementPlayerIDs(ctx, playerIDs)
	}
	requeueCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), movementPersistenceRequeueTimeout)
	defer cancel()
	return w.source.RequeueDirtyMovementPlayerIDs(requeueCtx, playerIDs)
}
