package app

import (
	"context"
	"errors"
	"io"
	"log"
	"reflect"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/world"
)

// movementPersistenceSourceStub 记录 worker 对 Redis 权威移动状态领域入口的调用。
type movementPersistenceSourceStub struct {
	claimLimit        uint32
	claimedPlayerIDs  []uint64
	claimErr          error
	loadedPlayerIDs   []uint64
	states            map[uint64]*world.MovementState
	loadErrors        map[uint64]error
	requeuedPlayerIDs []uint64
	requeueErr        error
	requeueContextErr error
}

// ClaimDirtyMovementPlayerIDs 返回测试预设的 dirty 玩家，并记录 worker 使用的批次上限。
func (s *movementPersistenceSourceStub) ClaimDirtyMovementPlayerIDs(_ context.Context, limit uint32) ([]uint64, error) {
	s.claimLimit = limit
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	return append([]uint64(nil), s.claimedPlayerIDs...), nil
}

// LoadMovementState 返回指定玩家的测试状态或预设错误。
func (s *movementPersistenceSourceStub) LoadMovementState(_ context.Context, playerID uint64) (*world.MovementState, error) {
	s.loadedPlayerIDs = append(s.loadedPlayerIDs, playerID)
	if err := s.loadErrors[playerID]; err != nil {
		return nil, err
	}
	return s.states[playerID], nil
}

// RequeueDirtyMovementPlayerIDs 记录失败编号以及重入队时上下文是否仍可用于清理操作。
func (s *movementPersistenceSourceStub) RequeueDirtyMovementPlayerIDs(ctx context.Context, playerIDs []uint64) error {
	s.requeuedPlayerIDs = append([]uint64(nil), playerIDs...)
	s.requeueContextErr = ctx.Err()
	return s.requeueErr
}

// movementPositionWrite 保存一次 PostgreSQL 玩家位置写入的完整参数。
type movementPositionWrite struct {
	playerID uint64
	sceneID  uint32
	posX     int32
	posY     int32
}

// movementPositionWriterStub 记录永久位置写入，并允许按玩家模拟 PostgreSQL 失败。
type movementPositionWriterStub struct {
	writes      []movementPositionWrite
	writeErrors map[uint64]error
}

// UpdatePosition 记录写入参数，并返回指定玩家的预设写入错误。
func (s *movementPositionWriterStub) UpdatePosition(
	_ context.Context,
	playerID uint64,
	sceneID uint32,
	posX int32,
	posY int32,
) error {
	s.writes = append(s.writes, movementPositionWrite{
		playerID: playerID,
		sceneID:  sceneID,
		posX:     posX,
		posY:     posY,
	})
	return s.writeErrors[playerID]
}

// newMovementPersistenceWorkerForTest 创建使用静默日志的合法测试 worker。
func newMovementPersistenceWorkerForTest(
	t *testing.T,
	source movementPersistenceSource,
	writer movementPositionWriter,
) *movementPersistenceWorker {
	t.Helper()
	worker, err := newMovementPersistenceWorker(
		source,
		writer,
		log.New(io.Discard, "", 0),
		time.Second,
		100,
	)
	if err != nil {
		t.Fatalf("newMovementPersistenceWorker() error = %v", err)
	}
	return worker
}

// TestMovementPersistenceWorkerFlushOncePersistsClaimedPlayers 验证成功批次准确写入 Redis 中的最新整数权威位置。
func TestMovementPersistenceWorkerFlushOncePersistsClaimedPlayers(t *testing.T) {
	source := &movementPersistenceSourceStub{
		claimedPlayerIDs: []uint64{10001, 10002},
		states: map[uint64]*world.MovementState{
			10001: {
				PlayerID:     10001,
				SceneID:      9,
				PersistedPos: world.Vec2i{X: 12, Y: 18},
			},
			10002: {
				PlayerID:     10002,
				SceneID:      10,
				PersistedPos: world.Vec2i{X: 25, Y: 31},
			},
		},
	}
	writer := &movementPositionWriterStub{}
	worker := newMovementPersistenceWorkerForTest(t, source, writer)

	result, err := worker.flushOnce(context.Background())
	if err != nil {
		t.Fatalf("flushOnce() error = %v", err)
	}
	if result != (movementPersistenceBatchResult{Claimed: 2, Persisted: 2}) {
		t.Fatalf("flushOnce() result = %+v, want claimed=2 persisted=2", result)
	}
	if source.claimLimit != 100 {
		t.Fatalf("claim limit = %d, want 100", source.claimLimit)
	}
	if !reflect.DeepEqual(source.loadedPlayerIDs, []uint64{10001, 10002}) {
		t.Fatalf("loaded player ids = %v, want [10001 10002]", source.loadedPlayerIDs)
	}
	wantWrites := []movementPositionWrite{
		{playerID: 10001, sceneID: 9, posX: 12, posY: 18},
		{playerID: 10002, sceneID: 10, posX: 25, posY: 31},
	}
	if !reflect.DeepEqual(writer.writes, wantWrites) {
		t.Fatalf("writes = %+v, want %+v", writer.writes, wantWrites)
	}
	if source.requeuedPlayerIDs != nil {
		t.Fatalf("requeued player ids = %v, want nil", source.requeuedPlayerIDs)
	}
}

// TestMovementPersistenceWorkerFlushOnceHandlesEmptyBatch 验证空 dirty 集合不会触发多余状态读取、数据库写入或重入队。
func TestMovementPersistenceWorkerFlushOnceHandlesEmptyBatch(t *testing.T) {
	source := &movementPersistenceSourceStub{}
	writer := &movementPositionWriterStub{}
	worker := newMovementPersistenceWorkerForTest(t, source, writer)

	result, err := worker.flushOnce(context.Background())
	if err != nil {
		t.Fatalf("flushOnce() error = %v", err)
	}
	if result != (movementPersistenceBatchResult{}) {
		t.Fatalf("flushOnce() result = %+v, want zero result", result)
	}
	if len(source.loadedPlayerIDs) != 0 || len(writer.writes) != 0 || len(source.requeuedPlayerIDs) != 0 {
		t.Fatalf(
			"empty batch caused side effects: loaded=%v writes=%v requeued=%v",
			source.loadedPlayerIDs,
			writer.writes,
			source.requeuedPlayerIDs,
		)
	}
}

// TestMovementPersistenceWorkerFlushOnceReturnsClaimError 验证 Redis 领取失败时不会继续处理不存在的批次。
func TestMovementPersistenceWorkerFlushOnceReturnsClaimError(t *testing.T) {
	claimErr := errors.New("redis unavailable")
	source := &movementPersistenceSourceStub{claimErr: claimErr}
	writer := &movementPositionWriterStub{}
	worker := newMovementPersistenceWorkerForTest(t, source, writer)

	result, err := worker.flushOnce(context.Background())
	if !errors.Is(err, claimErr) {
		t.Fatalf("flushOnce() error = %v, want %v", err, claimErr)
	}
	if result != (movementPersistenceBatchResult{}) {
		t.Fatalf("flushOnce() result = %+v, want zero result", result)
	}
	if len(source.loadedPlayerIDs) != 0 || len(writer.writes) != 0 || len(source.requeuedPlayerIDs) != 0 {
		t.Fatalf(
			"claim failure caused side effects: loaded=%v writes=%v requeued=%v",
			source.loadedPlayerIDs,
			writer.writes,
			source.requeuedPlayerIDs,
		)
	}
}

// TestMovementPersistenceWorkerFlushOnceRequeuesIndividualFailures 验证单玩家读取或写入失败不会阻断同批其他玩家。
func TestMovementPersistenceWorkerFlushOnceRequeuesIndividualFailures(t *testing.T) {
	loadErr := errors.New("redis read failed")
	writeErr := errors.New("postgres write failed")
	source := &movementPersistenceSourceStub{
		claimedPlayerIDs: []uint64{10001, 10002, 10003},
		states: map[uint64]*world.MovementState{
			10001: {
				PlayerID:     10001,
				SceneID:      9,
				PersistedPos: world.Vec2i{X: 10, Y: 11},
			},
			10003: {
				PlayerID:     10003,
				SceneID:      11,
				PersistedPos: world.Vec2i{X: 30, Y: 31},
			},
		},
		loadErrors: map[uint64]error{10002: loadErr},
	}
	writer := &movementPositionWriterStub{
		writeErrors: map[uint64]error{10003: writeErr},
	}
	worker := newMovementPersistenceWorkerForTest(t, source, writer)

	result, err := worker.flushOnce(context.Background())
	if !errors.Is(err, loadErr) || !errors.Is(err, writeErr) {
		t.Fatalf("flushOnce() error = %v, want joined load and write errors", err)
	}
	if result != (movementPersistenceBatchResult{Claimed: 3, Persisted: 1, Requeued: 2}) {
		t.Fatalf("flushOnce() result = %+v, want claimed=3 persisted=1 requeued=2", result)
	}
	wantWrites := []movementPositionWrite{
		{playerID: 10001, sceneID: 9, posX: 10, posY: 11},
		{playerID: 10003, sceneID: 11, posX: 30, posY: 31},
	}
	if !reflect.DeepEqual(writer.writes, wantWrites) {
		t.Fatalf("writes = %+v, want %+v", writer.writes, wantWrites)
	}
	if !reflect.DeepEqual(source.requeuedPlayerIDs, []uint64{10002, 10003}) {
		t.Fatalf("requeued player ids = %v, want [10002 10003]", source.requeuedPlayerIDs)
	}
}

// TestMovementPersistenceWorkerFlushOnceRequeuesInvalidStates 验证空状态和玩家编号错配不会写入错误档案。
func TestMovementPersistenceWorkerFlushOnceRequeuesInvalidStates(t *testing.T) {
	source := &movementPersistenceSourceStub{
		claimedPlayerIDs: []uint64{10001, 10002},
		states: map[uint64]*world.MovementState{
			10002: {
				PlayerID:     99999,
				SceneID:      9,
				PersistedPos: world.Vec2i{X: 12, Y: 13},
			},
		},
	}
	writer := &movementPositionWriterStub{}
	worker := newMovementPersistenceWorkerForTest(t, source, writer)

	result, err := worker.flushOnce(context.Background())
	if err == nil {
		t.Fatal("flushOnce() error = nil, want invalid state error")
	}
	if result != (movementPersistenceBatchResult{Claimed: 2, Requeued: 2}) {
		t.Fatalf("flushOnce() result = %+v, want claimed=2 requeued=2", result)
	}
	if len(writer.writes) != 0 {
		t.Fatalf("writes = %+v, want none", writer.writes)
	}
	if !reflect.DeepEqual(source.requeuedPlayerIDs, []uint64{10001, 10002}) {
		t.Fatalf("requeued player ids = %v, want [10001 10002]", source.requeuedPlayerIDs)
	}
}

// TestMovementPersistenceWorkerFlushOnceReturnsRequeueError 验证重入队失败会保留原始玩家错误并报告 dirty 标记恢复失败。
func TestMovementPersistenceWorkerFlushOnceReturnsRequeueError(t *testing.T) {
	loadErr := errors.New("redis read failed")
	requeueErr := errors.New("redis requeue failed")
	source := &movementPersistenceSourceStub{
		claimedPlayerIDs: []uint64{10001},
		loadErrors:       map[uint64]error{10001: loadErr},
		requeueErr:       requeueErr,
	}
	writer := &movementPositionWriterStub{}
	worker := newMovementPersistenceWorkerForTest(t, source, writer)

	result, err := worker.flushOnce(context.Background())
	if !errors.Is(err, loadErr) || !errors.Is(err, requeueErr) {
		t.Fatalf("flushOnce() error = %v, want joined load and requeue errors", err)
	}
	if result != (movementPersistenceBatchResult{Claimed: 1}) {
		t.Fatalf("flushOnce() result = %+v, want claimed=1 with no confirmed requeue", result)
	}
	if !reflect.DeepEqual(source.requeuedPlayerIDs, []uint64{10001}) {
		t.Fatalf("requeued player ids = %v, want [10001]", source.requeuedPlayerIDs)
	}
}

// TestMovementPersistenceWorkerFlushOnceUsesCleanupContextAfterCancellation 验证停机取消后仍为已领取玩家提供短暂可用的重入队上下文。
func TestMovementPersistenceWorkerFlushOnceUsesCleanupContextAfterCancellation(t *testing.T) {
	loadErr := errors.New("request canceled")
	source := &movementPersistenceSourceStub{
		claimedPlayerIDs: []uint64{10001},
		loadErrors:       map[uint64]error{10001: loadErr},
	}
	writer := &movementPositionWriterStub{}
	worker := newMovementPersistenceWorkerForTest(t, source, writer)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := worker.flushOnce(ctx)
	if !errors.Is(err, loadErr) {
		t.Fatalf("flushOnce() error = %v, want %v", err, loadErr)
	}
	if result != (movementPersistenceBatchResult{Claimed: 1, Requeued: 1}) {
		t.Fatalf("flushOnce() result = %+v, want claimed=1 requeued=1", result)
	}
	if source.requeueContextErr != nil {
		t.Fatalf("requeue context error = %v, want nil cleanup context", source.requeueContextErr)
	}
}

// TestNewMovementPersistenceWorkerRejectsInvalidArguments 验证 worker 在进入应用生命周期前拒绝缺失依赖和非法运行参数。
func TestNewMovementPersistenceWorkerRejectsInvalidArguments(t *testing.T) {
	validSource := &movementPersistenceSourceStub{}
	validWriter := &movementPositionWriterStub{}
	tests := []struct {
		name      string
		source    movementPersistenceSource
		writer    movementPositionWriter
		interval  time.Duration
		batchSize uint32
	}{
		{name: "missing source", writer: validWriter, interval: time.Second, batchSize: 1},
		{name: "missing writer", source: validSource, interval: time.Second, batchSize: 1},
		{name: "zero interval", source: validSource, writer: validWriter, batchSize: 1},
		{name: "negative interval", source: validSource, writer: validWriter, interval: -time.Second, batchSize: 1},
		{name: "zero batch size", source: validSource, writer: validWriter, interval: time.Second},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			worker, err := newMovementPersistenceWorker(
				test.source,
				test.writer,
				nil,
				test.interval,
				test.batchSize,
			)
			if err == nil {
				t.Fatalf("newMovementPersistenceWorker() = %+v, nil; want error", worker)
			}
		})
	}
}
