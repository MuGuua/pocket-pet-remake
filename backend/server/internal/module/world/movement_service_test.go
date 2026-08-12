package world

import (
	"context"
	"errors"
	"testing"
)

type movementStateRepositoryStub struct {
	state       MovementState
	compareFrom uint32
	compareNext MovementState
}

func (r *movementStateRepositoryStub) Load(_ context.Context, _ uint64) (*MovementState, error) {
	state := r.state
	return &state, nil
}

func (r *movementStateRepositoryStub) Initialize(_ context.Context, state MovementState) error {
	r.state = state
	return nil
}

func (r *movementStateRepositoryStub) CompareAndSet(_ context.Context, expectedMoveSeq uint32, state MovementState) error {
	r.compareFrom = expectedMoveSeq
	r.compareNext = state
	return nil
}

func (r *movementStateRepositoryStub) Delete(_ context.Context, _ uint64) error {
	return nil
}

func TestAdvanceMovementStateRejectsMismatchedAuthority(t *testing.T) {
	current := movementStateForTest()
	testCases := []struct {
		name    string
		mutate  func(*MovementState)
		wantErr error
	}{
		{name: "old session", mutate: func(state *MovementState) { state.SessionID = "old" }, wantErr: ErrMovementSessionMismatch},
		{name: "old scene", mutate: func(state *MovementState) { state.SceneID++ }, wantErr: ErrMovementSceneMismatch},
		{name: "old scene version", mutate: func(state *MovementState) { state.SceneVersion++ }, wantErr: ErrMovementSceneMismatch},
		{name: "duplicate sequence", mutate: func(state *MovementState) { state.LastMoveSeq = current.LastMoveSeq }, wantErr: ErrMovementSequenceStale},
		{name: "older sequence", mutate: func(state *MovementState) { state.LastMoveSeq-- }, wantErr: ErrMovementSequenceStale},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repo := &movementStateRepositoryStub{state: current}
			service := NewService(nil)
			service.SetMovementStateRepository(repo)
			next := current
			testCase.mutate(&next)
			err := service.AdvanceMovementState(context.Background(), next)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("AdvanceMovementState() error = %v, want %v", err, testCase.wantErr)
			}
			if repo.compareFrom != 0 {
				t.Fatalf("CompareAndSet() called for rejected state")
			}
		})
	}
}

func TestAdvanceMovementStateUsesCurrentSequenceAndAdvancesPositionVersion(t *testing.T) {
	current := movementStateForTest()
	repo := &movementStateRepositoryStub{state: current}
	service := NewService(nil)
	service.SetMovementStateRepository(repo)
	next := current
	next.LastMoveSeq++

	if err := service.AdvanceMovementState(context.Background(), next); err != nil {
		t.Fatalf("AdvanceMovementState() error = %v", err)
	}
	if repo.compareFrom != current.LastMoveSeq {
		t.Fatalf("CompareAndSet() expected sequence = %d, want %d", repo.compareFrom, current.LastMoveSeq)
	}
	if repo.compareNext.PositionVersion != current.PositionVersion+1 {
		t.Fatalf("next position version = %d, want %d", repo.compareNext.PositionVersion, current.PositionVersion+1)
	}
}

func movementStateForTest() MovementState {
	return MovementState{
		PlayerID: 10001, SessionID: "session-1", SceneID: 3, SceneVersion: 2,
		PrecisePos: Vec2i{X: 12000, Y: 8000}, PersistedPos: Vec2i{X: 12, Y: 8},
		LastMoveSeq: 7, PositionVersion: 15,
	}
}

func TestEvaluateMovementClampsCandidateByServerElapsedTime(t *testing.T) {
	service := movementServiceForEvaluationTest()
	current := movementStateForTest()
	current.PrecisePos = Vec2i{}
	current.PersistedPos = Vec2i{}
	current.Facing = Vec2i{X: 1}
	current.Moving = true
	current.LastServerTickMS = 1000
	input := Vec2i{X: 1}

	result, err := service.EvaluateMovement(current, MovementIntent{
		Input: &input, CandidatePos: Vec2i{X: 1000}, HasCandidate: true,
		Facing: input, Moving: true, MoveSeq: 8, ServerTickMS: 1100,
	})
	if err != nil {
		t.Fatalf("EvaluateMovement() error = %v", err)
	}
	if result.State.PrecisePos != (Vec2i{X: 375}) || !result.Corrected {
		t.Fatalf("result = %+v, want corrected position (375,0)", result)
	}
	if result.State.Speed != 3750 {
		t.Fatalf("speed = %d, want 3750", result.State.Speed)
	}
}

func TestEvaluateMovementCapsLongServerElapsedTime(t *testing.T) {
	service := movementServiceForEvaluationTest()
	current := movementStateForTest()
	current.PrecisePos = Vec2i{}
	current.Facing = Vec2i{X: 1}
	current.Moving = true
	current.LastServerTickMS = 1000
	input := Vec2i{X: 1}

	result, err := service.EvaluateMovement(current, MovementIntent{
		Input: &input, CandidatePos: Vec2i{X: 5000}, HasCandidate: true,
		Facing: input, Moving: true, MoveSeq: 8, ServerTickMS: 5000,
	})
	if err != nil {
		t.Fatalf("EvaluateMovement() error = %v", err)
	}
	if result.State.PrecisePos != (Vec2i{X: 1125}) {
		t.Fatalf("position = %+v, want max 300ms movement (1125,0)", result.State.PrecisePos)
	}
}

func TestEvaluateMovementRejectsDiagonalInputAndAxisDrift(t *testing.T) {
	service := movementServiceForEvaluationTest()
	current := movementStateForTest()
	current.PrecisePos = Vec2i{}
	current.LastServerTickMS = 1000

	diagonal := Vec2i{X: 1, Y: 1}
	_, err := service.EvaluateMovement(current, MovementIntent{
		Input: &diagonal, CandidatePos: Vec2i{X: 100, Y: 100}, HasCandidate: true,
		Facing: diagonal, Moving: true, MoveSeq: 8, ServerTickMS: 1100,
	})
	if !errors.Is(err, ErrMovementInputInvalid) {
		t.Fatalf("diagonal error = %v, want %v", err, ErrMovementInputInvalid)
	}

	right := Vec2i{X: 1}
	_, err = service.EvaluateMovement(current, MovementIntent{
		Input: &right, CandidatePos: Vec2i{X: 100, Y: 126}, HasCandidate: true,
		Facing: right, Moving: true, MoveSeq: 8, ServerTickMS: 1100,
	})
	if !errors.Is(err, ErrMovementAxisInvalid) {
		t.Fatalf("axis error = %v, want %v", err, ErrMovementAxisInvalid)
	}
}

func movementServiceForEvaluationTest() *Service {
	service := NewService(nil)
	service.movementConfig = MovementConfig{
		SpeedMilliCellsPerSecond: 3750,
		MaxElapsedMS:             300,
		AxisToleranceMilli:       125,
	}
	return service
}
