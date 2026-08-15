package world

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type movementStateRepositoryStub struct {
	state        MovementState
	loadErr      error
	compareErr   error
	loadCount    int
	compareCount int
	compareFrom  uint32
	compareNext  MovementState
}

func (r *movementStateRepositoryStub) Load(_ context.Context, _ uint64) (*MovementState, error) {
	r.loadCount++
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	state := r.state
	return &state, nil
}

func (r *movementStateRepositoryStub) Initialize(_ context.Context, state MovementState) error {
	r.state = state
	return nil
}

func (r *movementStateRepositoryStub) CompareAndSet(_ context.Context, expectedMoveSeq uint32, state MovementState) error {
	r.compareCount++
	r.compareFrom = expectedMoveSeq
	r.compareNext = state
	return r.compareErr
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

// TestMovePlayerAdvancesAuthoritativeState 验证普通移动用例会在领域层完成加载、旧字段归一化、移动计算和单次 CAS 推进。
func TestMovePlayerAdvancesAuthoritativeState(t *testing.T) {
	current := movementStateForTest()
	current.LastServerTickMS = 1000
	repo := &movementStateRepositoryStub{state: current}
	service := movementServiceForEvaluationTest()
	service.SetMovementStateRepository(repo)
	right := Vec2i{X: 1}
	moving := true

	result, err := service.MovePlayer(context.Background(), MovePlayerInput{
		PlayerID:  current.PlayerID,
		SessionID: current.SessionID,
		SceneID:   current.SceneID,
		MoveSeq:   current.LastMoveSeq + 1,
		CandidatePos: Vec2i{
			X: 12300,
			Y: 8000,
		},
		CandidateCell: Vec2i{X: 12, Y: 8},
		HasCandidate:  true,
		Input:         &right,
		Facing:        &right,
		Moving:        &moving,
		ServerTickMS:  1100,
	})
	if err != nil {
		t.Fatalf("MovePlayer() error = %v", err)
	}
	if result.PreviousState != current {
		t.Fatalf("PreviousState = %+v, want %+v", result.PreviousState, current)
	}
	if result.State.PrecisePos != (Vec2i{X: 12300, Y: 8000}) || result.State.LastMoveSeq != 8 {
		t.Fatalf("State = %+v, want precise position (12300,8000) and sequence 8", result.State)
	}
	if result.State.Facing != right || !result.State.Moving {
		t.Fatalf("facing/moving = %+v/%t, want explicitly reported right/true state", result.State.Facing, result.State.Moving)
	}
	if repo.loadCount != 1 || repo.compareCount != 1 || repo.compareFrom != current.LastMoveSeq {
		t.Fatalf("repository calls load=%d compare=%d expected_seq=%d", repo.loadCount, repo.compareCount, repo.compareFrom)
	}
	if repo.compareNext.PositionVersion != current.PositionVersion+1 {
		t.Fatalf("position version = %d, want %d", repo.compareNext.PositionVersion, current.PositionVersion+1)
	}
}

// TestMovePlayerReturnsMovementStateLoadError 验证 Redis 状态读取失败时不会进入移动计算或 CAS 写入。
func TestMovePlayerReturnsMovementStateLoadError(t *testing.T) {
	loadErr := errors.New("movement repository unavailable")
	repo := &movementStateRepositoryStub{loadErr: loadErr}
	service := movementServiceForEvaluationTest()
	service.SetMovementStateRepository(repo)

	result, err := service.MovePlayer(context.Background(), MovePlayerInput{PlayerID: 10001, SessionID: "session-1", SceneID: 3})
	if !errors.Is(err, loadErr) {
		t.Fatalf("MovePlayer() error = %v, want %v", err, loadErr)
	}
	if result != (MovePlayerResult{}) || repo.loadCount != 1 || repo.compareCount != 0 {
		t.Fatalf("result=%+v load_count=%d compare_count=%d, want one failed load without CAS", result, repo.loadCount, repo.compareCount)
	}
}

// TestMovePlayerRejectsMismatchedAuthority 验证普通移动入口在计算前拒绝旧会话和错误场景，且不会写入 Redis。
func TestMovePlayerRejectsMismatchedAuthority(t *testing.T) {
	current := movementStateForTest()
	testCases := []struct {
		name      string
		sessionID string
		sceneID   uint32
		wantErr   error
	}{
		{name: "old session", sessionID: "old-session", sceneID: current.SceneID, wantErr: ErrMovementSessionMismatch},
		{name: "wrong scene", sessionID: current.SessionID, sceneID: current.SceneID + 1, wantErr: ErrMovementSceneMismatch},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repo := &movementStateRepositoryStub{state: current}
			service := movementServiceForEvaluationTest()
			service.SetMovementStateRepository(repo)
			result, err := service.MovePlayer(context.Background(), MovePlayerInput{
				PlayerID: current.PlayerID, SessionID: testCase.sessionID, SceneID: testCase.sceneID,
				MoveSeq: current.LastMoveSeq + 1, HasCandidate: true,
			})
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("MovePlayer() error = %v, want %v", err, testCase.wantErr)
			}
			if result.PreviousState != current {
				t.Fatalf("PreviousState = %+v, want current authoritative state", result.PreviousState)
			}
			if repo.compareCount != 0 {
				t.Fatalf("CompareAndSet() called for rejected authority")
			}
		})
	}
}

// TestMovePlayerRejectsInvalidOrStaleMovementWithoutWrite 验证序号倒退和非法输入不会推进 Redis 状态。
func TestMovePlayerRejectsInvalidOrStaleMovementWithoutWrite(t *testing.T) {
	current := movementStateForTest()
	current.LastServerTickMS = 1000
	testCases := []struct {
		name    string
		moveSeq uint32
		input   Vec2i
		wantErr error
	}{
		{name: "stale sequence", moveSeq: current.LastMoveSeq, input: Vec2i{X: 1}, wantErr: ErrMovementSequenceStale},
		{name: "diagonal input", moveSeq: current.LastMoveSeq + 1, input: Vec2i{X: 1, Y: 1}, wantErr: ErrMovementInputInvalid},
	}

	moving := true
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repo := &movementStateRepositoryStub{state: current}
			service := movementServiceForEvaluationTest()
			service.SetMovementStateRepository(repo)
			_, err := service.MovePlayer(context.Background(), MovePlayerInput{
				PlayerID: current.PlayerID, SessionID: current.SessionID, SceneID: current.SceneID,
				MoveSeq: testCase.moveSeq, CandidatePos: Vec2i{X: 12100, Y: 8000}, CandidateCell: Vec2i{X: 12, Y: 8},
				HasCandidate: true, Input: &testCase.input, Moving: &moving, ServerTickMS: 1100,
			})
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("MovePlayer() error = %v, want %v", err, testCase.wantErr)
			}
			if repo.compareCount != 0 {
				t.Fatalf("CompareAndSet() called for rejected movement")
			}
		})
	}
}

// TestMovePlayerPreservesLegacyHeartbeat 验证无 target_pos 的旧客户端心跳不会消耗序号或触发 Redis 写入。
func TestMovePlayerPreservesLegacyHeartbeat(t *testing.T) {
	current := movementStateForTest()
	repo := &movementStateRepositoryStub{state: current}
	service := movementServiceForEvaluationTest()
	service.SetMovementStateRepository(repo)

	result, err := service.MovePlayer(context.Background(), MovePlayerInput{
		PlayerID: current.PlayerID, SessionID: current.SessionID, SceneID: current.SceneID, MoveSeq: current.LastMoveSeq + 10,
	})
	if err != nil {
		t.Fatalf("MovePlayer() error = %v", err)
	}
	if result.State != current || repo.compareCount != 0 {
		t.Fatalf("result=%+v compare_count=%d, want unchanged state without CAS", result, repo.compareCount)
	}
}

// TestMovePlayerReturnsCompareAndSetConflict 验证并发 CAS 冲突会原样返回给传输层，便于客户端使用旧权威状态纠偏。
func TestMovePlayerReturnsCompareAndSetConflict(t *testing.T) {
	current := movementStateForTest()
	current.LastServerTickMS = 1000
	repo := &movementStateRepositoryStub{state: current, compareErr: ErrMovementSequenceStale}
	service := movementServiceForEvaluationTest()
	service.SetMovementStateRepository(repo)
	right := Vec2i{X: 1}
	moving := true

	result, err := service.MovePlayer(context.Background(), MovePlayerInput{
		PlayerID: current.PlayerID, SessionID: current.SessionID, SceneID: current.SceneID,
		MoveSeq: current.LastMoveSeq + 1, CandidatePos: Vec2i{X: 12100, Y: 8000}, CandidateCell: Vec2i{X: 12, Y: 8},
		HasCandidate: true, Input: &right, Moving: &moving, ServerTickMS: 1100,
	})
	if !errors.Is(err, ErrMovementSequenceStale) {
		t.Fatalf("MovePlayer() error = %v, want %v", err, ErrMovementSequenceStale)
	}
	if result.PreviousState != current || repo.compareCount != 1 {
		t.Fatalf("result=%+v compare_count=%d, want current state and one CAS attempt", result, repo.compareCount)
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
