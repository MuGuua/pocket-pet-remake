package redis

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"pocket-pet-remake/server/internal/module/world"
)

func TestMovementStateRepositoryInitializeAndLoad(t *testing.T) {
	client := &fakeClient{}
	repo := NewMovementStateRepository(client, "test")
	want := movementStateFixture()

	if err := repo.Initialize(context.Background(), want); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	got, err := repo.Load(context.Background(), want.PlayerID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if *got != want {
		t.Fatalf("Load() = %+v, want %+v", *got, want)
	}
}

func TestMovementStateRepositoryCompareAndSetMapsAtomicResult(t *testing.T) {
	testCases := []struct {
		name    string
		result  string
		wantErr error
	}{
		{name: "accepted", result: "ok"},
		{name: "missing", result: "not_found", wantErr: world.ErrMovementStateNotFound},
		{name: "old session", result: "session_mismatch", wantErr: world.ErrMovementSessionMismatch},
		{name: "old scene", result: "scene_mismatch", wantErr: world.ErrMovementSceneMismatch},
		{name: "old sequence", result: "stale_sequence", wantErr: world.ErrMovementSequenceStale},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client := &fakeClient{evalResult: testCase.result}
			repo := NewMovementStateRepository(client, "test")
			err := repo.CompareAndSet(context.Background(), 6, movementStateFixture())
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("CompareAndSet() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

// TestMovementStateRepositoryCompareAndSetMarksPlayerDirty 验证移动状态 CAS 成功时会同时写入 dirty 集合。
// 后续批量持久化只消费这个集合，因此 dirty 键和玩家编号都属于移动热路径的重要契约。
func TestMovementStateRepositoryCompareAndSetMarksPlayerDirty(t *testing.T) {
	client := &fakeClient{evalResult: "ok"}
	repo := NewMovementStateRepository(client, "test")
	state := movementStateFixture()

	if err := repo.CompareAndSet(context.Background(), 6, state); err != nil {
		t.Fatalf("CompareAndSet() error = %v", err)
	}
	wantKeys := []string{"test:world:movement:player:10001", "test:world:movement:dirty"}
	if !reflect.DeepEqual(client.evalKeys, wantKeys) {
		t.Fatalf("CompareAndSet() keys = %v, want %v", client.evalKeys, wantKeys)
	}
	if len(client.evalArgs) != 8 || client.evalArgs[7] != state.PlayerID {
		t.Fatalf("CompareAndSet() args = %v, want final player id %d", client.evalArgs, state.PlayerID)
	}
}

// TestMovementStateRepositoryClaimDirtyPlayerIDs 验证仓储通过单次 Lua 调用原子领取 dirty 玩家并解析 Redis 成员。
func TestMovementStateRepositoryClaimDirtyPlayerIDs(t *testing.T) {
	client := &fakeClient{evalResult: []any{"10001", []byte("10002")}}
	repo := NewMovementStateRepository(client, "test")

	got, err := repo.ClaimDirtyPlayerIDs(context.Background(), 20)
	if err != nil {
		t.Fatalf("ClaimDirtyPlayerIDs() error = %v", err)
	}
	want := []uint64{10001, 10002}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ClaimDirtyPlayerIDs() = %v, want %v", got, want)
	}
	if client.evalScript != claimDirtyMovementPlayersScript {
		t.Fatal("ClaimDirtyPlayerIDs() did not use the atomic claim script")
	}
	if !reflect.DeepEqual(client.evalKeys, []string{"test:world:movement:dirty"}) {
		t.Fatalf("ClaimDirtyPlayerIDs() keys = %v, want dirty key", client.evalKeys)
	}
	if !reflect.DeepEqual(client.evalArgs, []any{uint32(20)}) {
		t.Fatalf("ClaimDirtyPlayerIDs() args = %v, want [20]", client.evalArgs)
	}
}

// TestMovementStateRepositoryClaimDirtyPlayerIDsHandlesEmptyAndInvalidResults 验证空集合返回可直接迭代的空切片，损坏成员会明确报错。
func TestMovementStateRepositoryClaimDirtyPlayerIDsHandlesEmptyAndInvalidResults(t *testing.T) {
	testCases := []struct {
		name       string
		result     any
		wantErr    bool
		wantNonNil bool
	}{
		{name: "empty", result: []any{}, wantNonNil: true},
		{name: "invalid member", result: []any{"invalid"}, wantErr: true},
		{name: "zero member", result: []any{"0"}, wantErr: true},
		{name: "invalid member type", result: []any{uint64(10001)}, wantErr: true},
		{name: "invalid result type", result: "10001", wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repo := NewMovementStateRepository(&fakeClient{evalResult: testCase.result}, "test")
			got, err := repo.ClaimDirtyPlayerIDs(context.Background(), 20)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("ClaimDirtyPlayerIDs() = %v, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ClaimDirtyPlayerIDs() error = %v", err)
			}
			if testCase.wantNonNil && got == nil {
				t.Fatal("ClaimDirtyPlayerIDs() returned nil, want non-nil empty slice")
			}
			if len(got) != 0 {
				t.Fatalf("ClaimDirtyPlayerIDs() = %v, want empty slice", got)
			}
		})
	}
}

// TestMovementStateRepositoryClaimDirtyPlayerIDsWithZeroLimit 验证零上限是本地空操作，不产生无意义的 Redis 往返。
func TestMovementStateRepositoryClaimDirtyPlayerIDsWithZeroLimit(t *testing.T) {
	client := &fakeClient{}
	repo := NewMovementStateRepository(client, "test")

	got, err := repo.ClaimDirtyPlayerIDs(context.Background(), 0)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("ClaimDirtyPlayerIDs(0) = %v, %v, want non-nil empty success", got, err)
	}
	if client.evalCount != 0 {
		t.Fatalf("ClaimDirtyPlayerIDs(0) eval count = %d, want 0", client.evalCount)
	}
}

// TestMovementStateRepositoryRequeueDirtyPlayerIDs 验证写回失败的玩家会通过单次 Lua 调用整体放回 dirty 集合。
func TestMovementStateRepositoryRequeueDirtyPlayerIDs(t *testing.T) {
	client := &fakeClient{evalResult: int64(2)}
	repo := NewMovementStateRepository(client, "test")

	if err := repo.RequeueDirtyPlayerIDs(context.Background(), []uint64{10001, 10002}); err != nil {
		t.Fatalf("RequeueDirtyPlayerIDs() error = %v", err)
	}
	if client.evalScript != requeueDirtyMovementPlayersScript {
		t.Fatal("RequeueDirtyPlayerIDs() did not use the atomic requeue script")
	}
	if !reflect.DeepEqual(client.evalKeys, []string{"test:world:movement:dirty"}) {
		t.Fatalf("RequeueDirtyPlayerIDs() keys = %v, want dirty key", client.evalKeys)
	}
	if !reflect.DeepEqual(client.evalArgs, []any{uint64(10001), uint64(10002)}) {
		t.Fatalf("RequeueDirtyPlayerIDs() args = %v, want [10001 10002]", client.evalArgs)
	}
}

// TestMovementStateRepositoryRequeueDirtyPlayerIDsValidatesBeforeWrite 验证空批次不访问 Redis，非法编号也不会造成部分玩家提前入队。
func TestMovementStateRepositoryRequeueDirtyPlayerIDsValidatesBeforeWrite(t *testing.T) {
	testCases := []struct {
		name      string
		playerIDs []uint64
		wantErr   bool
	}{
		{name: "empty", playerIDs: nil},
		{name: "contains zero", playerIDs: []uint64{10001, 0, 10002}, wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client := &fakeClient{}
			repo := NewMovementStateRepository(client, "test")
			err := repo.RequeueDirtyPlayerIDs(context.Background(), testCase.playerIDs)
			if testCase.wantErr && err == nil {
				t.Fatal("RequeueDirtyPlayerIDs() error = nil, want validation error")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("RequeueDirtyPlayerIDs() error = %v", err)
			}
			if client.evalCount != 0 {
				t.Fatalf("RequeueDirtyPlayerIDs() eval count = %d, want 0", client.evalCount)
			}
		})
	}
}

func TestMovementStateRepositoryLoadMissingState(t *testing.T) {
	repo := NewMovementStateRepository(&fakeClient{}, "test")
	_, err := repo.Load(context.Background(), 10001)
	if !errors.Is(err, world.ErrMovementStateNotFound) {
		t.Fatalf("Load() error = %v, want %v", err, world.ErrMovementStateNotFound)
	}
}

func movementStateFixture() world.MovementState {
	return world.MovementState{
		PlayerID: 10001, SessionID: "session-1", SceneID: 3, SceneVersion: 2,
		PrecisePos: world.Vec2i{X: 12310, Y: 8000}, PersistedPos: world.Vec2i{X: 12, Y: 8},
		Facing: world.Vec2i{X: 1, Y: 0}, Moving: true, Speed: 90,
		LastMoveSeq: 7, LastServerTickMS: 92001, PositionVersion: 15,
	}
}
