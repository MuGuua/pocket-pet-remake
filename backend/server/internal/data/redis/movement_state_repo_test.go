package redis

import (
	"context"
	"errors"
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
