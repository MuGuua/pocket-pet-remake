package world

import (
	"bytes"
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
	service.sceneBoundaries = map[uint32]SceneBoundary{
		3: {SceneID: 3, MinX: 0, MinY: 0, MaxX: 14000, MaxY: 14000},
	}
	service.sceneNavigations = map[uint32]SceneNavigation{
		3: {
			SceneID: 3, Version: 1, OriginX: 0, OriginY: 0,
			GridWidth: 15, GridHeight: 15, CellSizeMilli: 1000,
			NavigationData: bytes.Repeat([]byte{0xff}, 29), Status: SceneNavigationStatusPublished,
		},
	}
	return service
}

// TestEvaluateMovementClampsPositionToSceneBoundary 验证速度裁剪后仍会应用数据库场景矩形边界。
func TestEvaluateMovementClampsPositionToSceneBoundary(t *testing.T) {
	service := movementServiceForEvaluationTest()
	service.sceneBoundaries[3] = SceneBoundary{SceneID: 3, MinX: 0, MinY: 0, MaxX: 1000, MaxY: 1000}
	current := movementStateForTest()
	current.PrecisePos = Vec2i{X: 900, Y: 500}
	current.PersistedPos = Vec2i{X: 1, Y: 1}
	current.Facing = Vec2i{X: 1}
	current.Moving = true
	current.LastServerTickMS = 1000
	right := Vec2i{X: 1}

	result, err := service.EvaluateMovement(current, MovementIntent{
		Input: &right, CandidatePos: Vec2i{X: 1400, Y: 500}, HasCandidate: true,
		Facing: right, Moving: true, MoveSeq: 8, ServerTickMS: 1200,
	})
	if err != nil {
		t.Fatalf("EvaluateMovement() error = %v", err)
	}
	if result.State.PrecisePos != (Vec2i{X: 1000, Y: 500}) || result.State.PersistedPos != (Vec2i{X: 1, Y: 1}) || !result.Corrected {
		t.Fatalf("result = %+v, want scene boundary correction at (1000,500)", result)
	}
}

// TestEvaluateMovementRejectsMissingSceneBoundary 验证运行时缓存缺少当前场景时不能绕过边界校验。
func TestEvaluateMovementRejectsMissingSceneBoundary(t *testing.T) {
	service := movementServiceForEvaluationTest()
	delete(service.sceneBoundaries, 3)
	current := movementStateForTest()
	current.LastServerTickMS = 1000
	right := Vec2i{X: 1}

	_, err := service.EvaluateMovement(current, MovementIntent{
		Input: &right, CandidatePos: Vec2i{X: 12100, Y: 8000}, HasCandidate: true,
		Facing: right, Moving: true, MoveSeq: 8, ServerTickMS: 1100,
	})
	if !errors.Is(err, ErrSceneBoundaryUnavailable) {
		t.Fatalf("EvaluateMovement() error = %v, want %v", err, ErrSceneBoundaryUnavailable)
	}
}

type sceneBoundaryRepositoryStub struct {
	boundaries map[uint32]SceneBoundary
}

func (r *sceneBoundaryRepositoryStub) ListSceneBoundaries(_ context.Context) ([]SceneBoundary, error) {
	items := make([]SceneBoundary, 0, len(r.boundaries))
	for _, boundary := range r.boundaries {
		items = append(items, boundary)
	}
	return items, nil
}

func (r *sceneBoundaryRepositoryStub) UpdateSceneBoundary(_ context.Context, sceneID uint32, input AdminUpdateSceneBoundaryInput) (SceneBoundary, error) {
	boundary, ok := r.boundaries[sceneID]
	if !ok {
		return SceneBoundary{}, ErrSceneBoundaryUnavailable
	}
	boundary.MinX = input.MinX
	boundary.MinY = input.MinY
	boundary.MaxX = input.MaxX
	boundary.MaxY = input.MaxY
	boundary.LastUpdateReason = input.Reason
	boundary.UpdatedByAdminUserID = input.AdminUserID
	r.boundaries[sceneID] = boundary
	return boundary, nil
}

// TestUpdateAdminSceneBoundaryRefreshesSnapshot 验证后台写库成功后当前进程立即切换到新边界。
func TestUpdateAdminSceneBoundaryRefreshesSnapshot(t *testing.T) {
	repo := &sceneBoundaryRepositoryStub{boundaries: map[uint32]SceneBoundary{
		3: {SceneID: 3, MinX: 0, MinY: 0, MaxX: 13000, MaxY: 13000},
	}}
	service := &Service{sceneBoundaryRepo: repo}
	if err := service.RefreshSceneBoundaryCache(context.Background()); err != nil {
		t.Fatalf("RefreshSceneBoundaryCache() error = %v", err)
	}

	updated, err := service.UpdateAdminSceneBoundary(context.Background(), 3, AdminUpdateSceneBoundaryInput{
		MinX: 1000, MinY: 2000, MaxX: 12000, MaxY: 11000,
		Reason: " 缩小市场测试边界 ", AdminUserID: 7,
	})
	if err != nil {
		t.Fatalf("UpdateAdminSceneBoundary() error = %v", err)
	}
	cached, err := service.SceneBoundarySnapshot(3)
	if err != nil {
		t.Fatalf("SceneBoundarySnapshot() error = %v", err)
	}
	if updated.MinX != 1000 || cached != updated || updated.LastUpdateReason != "缩小市场测试边界" || updated.UpdatedByAdminUserID != 7 {
		t.Fatalf("updated=%+v cached=%+v", updated, cached)
	}
}
