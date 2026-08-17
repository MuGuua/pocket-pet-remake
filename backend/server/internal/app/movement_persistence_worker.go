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
// 当前由 player.Service 实现；P1-07 会在该边界后增加位置版本条件更新，worker 无需了解 SQL 细节。
type movementPositionWriter interface {
	UpdatePosition(ctx context.Context, playerID uint64, sceneID uint32, posX int32, posY int32) error
}

// movementPersistenceBatchResult 记录单次周期批次的处理数量，供日志与测试确认失败玩家没有静默丢失。
type movementPersistenceBatchResult struct {
	Claimed   int
	Persisted int
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
					"movement persistence batch failed: claimed=%d persisted=%d requeued=%d: %v",
					result.Claimed,
					result.Persisted,
					result.Requeued,
					err,
				)
			}
		}
	}
}

// flushOnce 原子领取一个有界批次，并逐个读取 Redis 最新状态后写入 PostgreSQL。
// P1-07 完成前写入仍使用现有无版本位置接口，因此本方法保留状态版本但不在应用层自行判断覆盖顺序。
func (w *movementPersistenceWorker) flushOnce(ctx context.Context) (movementPersistenceBatchResult, error) {
	playerIDs, err := w.source.ClaimDirtyMovementPlayerIDs(ctx, w.batchSize)
	if err != nil {
		return movementPersistenceBatchResult{}, fmt.Errorf("claim dirty movement players: %w", err)
	}
	result := movementPersistenceBatchResult{Claimed: len(playerIDs)}
	if len(playerIDs) == 0 {
		return result, nil
	}

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
		if persistErr := w.writer.UpdatePosition(
			ctx,
			playerID,
			state.SceneID,
			state.PersistedPos.X,
			state.PersistedPos.Y,
		); persistErr != nil {
			failedPlayerIDs = append(failedPlayerIDs, playerID)
			batchErr = errors.Join(batchErr, fmt.Errorf("persist movement state for player %d: %w", playerID, persistErr))
			continue
		}
		result.Persisted++
	}

	if len(failedPlayerIDs) == 0 {
		return result, batchErr
	}
	if requeueErr := w.requeueFailedPlayerIDs(ctx, failedPlayerIDs); requeueErr != nil {
		return result, errors.Join(batchErr, fmt.Errorf("requeue dirty movement players: %w", requeueErr))
	}
	result.Requeued = len(failedPlayerIDs)
	return result, batchErr
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
