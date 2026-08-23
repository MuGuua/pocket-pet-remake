package world

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type movementStateRepositoryStub struct {
	state             MovementState
	loadErr           error
	compareErr        error
	claimErr          error
	requeueErr        error
	loadCount         int
	compareCount      int
	compareFrom       uint32
	compareNext       MovementState
	claimLimit        uint32
	claimedPlayerIDs  []uint64
	requeuedPlayerIDs []uint64
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

func (r *movementStateRepositoryStub) ClaimDirtyPlayerIDs(_ context.Context, limit uint32) ([]uint64, error) {
	r.claimLimit = limit
	if r.claimErr != nil {
		return nil, r.claimErr
	}
	return append([]uint64(nil), r.claimedPlayerIDs...), nil
}

func (r *movementStateRepositoryStub) RequeueDirtyPlayerIDs(_ context.Context, playerIDs []uint64) error {
	r.requeuedPlayerIDs = append([]uint64(nil), playerIDs...)
	return r.requeueErr
}

func (r *movementStateRepositoryStub) Delete(_ context.Context, _ uint64) error {
	return nil
}

// TestDirtyMovementPlayerBatchDelegatesToRepository 验证领域服务只编排领取与失败重入队，不泄漏 Redis 命令细节。
func TestDirtyMovementPlayerBatchDelegatesToRepository(t *testing.T) {
	repo := &movementStateRepositoryStub{claimedPlayerIDs: []uint64{10001, 10002}}
	service := NewService(nil)
	service.SetMovementStateRepository(repo)

	claimed, err := service.ClaimDirtyMovementPlayerIDs(context.Background(), 20)
	if err != nil {
		t.Fatalf("ClaimDirtyMovementPlayerIDs() error = %v", err)
	}
	if repo.claimLimit != 20 || len(claimed) != 2 || claimed[0] != 10001 || claimed[1] != 10002 {
		t.Fatalf("claimed players = %v limit=%d, want [10001 10002] with limit 20", claimed, repo.claimLimit)
	}
	if err := service.RequeueDirtyMovementPlayerIDs(context.Background(), claimed); err != nil {
		t.Fatalf("RequeueDirtyMovementPlayerIDs() error = %v", err)
	}
	if len(repo.requeuedPlayerIDs) != 2 || repo.requeuedPlayerIDs[0] != 10001 || repo.requeuedPlayerIDs[1] != 10002 {
		t.Fatalf("requeued players = %v, want [10001 10002]", repo.requeuedPlayerIDs)
	}
}

// TestDirtyMovementPlayerBatchHandlesNoopAndMissingRepository 验证空批次不访问仓储，非空批次缺少 Redis 仓储时明确失败。
func TestDirtyMovementPlayerBatchHandlesNoopAndMissingRepository(t *testing.T) {
	service := NewService(nil)
	claimed, err := service.ClaimDirtyMovementPlayerIDs(context.Background(), 0)
	if err != nil || len(claimed) != 0 {
		t.Fatalf("ClaimDirtyMovementPlayerIDs(0) = %v, %v, want empty success", claimed, err)
	}
	if err := service.RequeueDirtyMovementPlayerIDs(context.Background(), nil); err != nil {
		t.Fatalf("RequeueDirtyMovementPlayerIDs(nil) error = %v", err)
	}
	if _, err := service.ClaimDirtyMovementPlayerIDs(context.Background(), 1); !errors.Is(err, ErrMovementStateNotFound) {
		t.Fatalf("ClaimDirtyMovementPlayerIDs(1) error = %v, want %v", err, ErrMovementStateNotFound)
	}
	if err := service.RequeueDirtyMovementPlayerIDs(context.Background(), []uint64{10001}); !errors.Is(err, ErrMovementStateNotFound) {
		t.Fatalf("RequeueDirtyMovementPlayerIDs() error = %v, want %v", err, ErrMovementStateNotFound)
	}
}

// TestDirtyMovementPlayerBatchPropagatesRepositoryErrors 验证领域入口不会吞掉 Redis 领取或重入队失败。
func TestDirtyMovementPlayerBatchPropagatesRepositoryErrors(t *testing.T) {
	claimErr := errors.New("claim dirty players")
	requeueErr := errors.New("requeue dirty players")
	repo := &movementStateRepositoryStub{claimErr: claimErr, requeueErr: requeueErr}
	service := NewService(nil)
	service.SetMovementStateRepository(repo)

	if _, err := service.ClaimDirtyMovementPlayerIDs(context.Background(), 20); !errors.Is(err, claimErr) {
		t.Fatalf("ClaimDirtyMovementPlayerIDs() error = %v, want %v", err, claimErr)
	}
	if err := service.RequeueDirtyMovementPlayerIDs(context.Background(), []uint64{10001}); !errors.Is(err, requeueErr) {
		t.Fatalf("RequeueDirtyMovementPlayerIDs() error = %v, want %v", err, requeueErr)
	}
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

// TestAdvanceLoadedMovementStateReusesCallerState 验证切图入口复用调用方已加载状态，只执行一次 CAS 且不会再次读取 Redis。
func TestAdvanceLoadedMovementStateReusesCallerState(t *testing.T) {
	current := movementStateForTest()
	repo := &movementStateRepositoryStub{state: current}
	service := NewService(nil)
	service.SetMovementStateRepository(repo)
	next := current
	next.LastMoveSeq++

	if err := service.AdvanceLoadedMovementState(context.Background(), current, &next); err != nil {
		t.Fatalf("AdvanceLoadedMovementState() error = %v", err)
	}
	if repo.loadCount != 0 {
		t.Fatalf("repository load count = %d, want 0", repo.loadCount)
	}
	if repo.compareCount != 1 || repo.compareFrom != current.LastMoveSeq {
		t.Fatalf("repository CAS count=%d expected_seq=%d, want one CAS from %d", repo.compareCount, repo.compareFrom, current.LastMoveSeq)
	}
	if next.PositionVersion != current.PositionVersion+1 {
		t.Fatalf("next position version = %d, want %d", next.PositionVersion, current.PositionVersion+1)
	}
	if repo.compareNext != next {
		t.Fatalf("repository next state = %+v, want %+v", repo.compareNext, next)
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

// TestMovePlayerRejectsStaleMovementWithoutWrite 验证重复或倒退序号不会覆盖 Redis 中较新的客户端坐标。
func TestMovePlayerRejectsStaleMovementWithoutWrite(t *testing.T) {
	current := movementStateForTest()
	current.LastServerTickMS = 1000
	moving := true
	right := Vec2i{X: 1}
	repo := &movementStateRepositoryStub{state: current}
	service := movementServiceForEvaluationTest()
	service.SetMovementStateRepository(repo)

	_, err := service.MovePlayer(context.Background(), MovePlayerInput{
		PlayerID: current.PlayerID, SessionID: current.SessionID, SceneID: current.SceneID,
		MoveSeq: current.LastMoveSeq, CandidatePos: Vec2i{X: 12100, Y: 8000}, CandidateCell: Vec2i{X: 12, Y: 8},
		HasCandidate: true, Input: &right, Moving: &moving, ServerTickMS: 1100,
	})
	if !errors.Is(err, ErrMovementSequenceStale) {
		t.Fatalf("MovePlayer() error = %v, want %v", err, ErrMovementSequenceStale)
	}
	if repo.compareCount != 0 {
		t.Fatalf("CompareAndSet() called for stale movement")
	}
}

// TestMovePlayerAcceptsReportedPositionWithoutInputValidation 验证普通移动不再根据输入向量计算或拒绝客户端坐标。
func TestMovePlayerAcceptsReportedPositionWithoutInputValidation(t *testing.T) {
	current := movementStateForTest()
	current.LastServerTickMS = 1000
	diagonalInput := Vec2i{X: 1, Y: 1}
	diagonalFacing := Vec2i{X: 1, Y: 1}
	moving := true
	repo := &movementStateRepositoryStub{state: current}
	service := movementServiceForEvaluationTest()
	service.SetMovementStateRepository(repo)

	result, err := service.MovePlayer(context.Background(), MovePlayerInput{
		PlayerID: current.PlayerID, SessionID: current.SessionID, SceneID: current.SceneID,
		MoveSeq: current.LastMoveSeq + 1, CandidatePos: Vec2i{X: 18999, Y: 4321}, CandidateCell: Vec2i{X: 19, Y: 4},
		HasCandidate: true, Input: &diagonalInput, Facing: &diagonalFacing, Moving: &moving, ServerTickMS: 1100,
	})
	if err != nil {
		t.Fatalf("MovePlayer() error = %v", err)
	}
	if result.State.PrecisePos != (Vec2i{X: 18999, Y: 4321}) || result.State.PersistedPos != (Vec2i{X: 19, Y: 4}) {
		t.Fatalf("State position = precise %+v cell %+v, want exact client report", result.State.PrecisePos, result.State.PersistedPos)
	}
	if result.State.Facing != (Vec2i{X: 1}) || !result.State.Moving {
		t.Fatalf("State facing/moving = %+v/%t, want fallback cardinal facing and reported moving state", result.State.Facing, result.State.Moving)
	}
	if repo.compareCount != 1 || repo.compareNext != result.State {
		t.Fatalf("CompareAndSet() count=%d state=%+v, want one exact state write", repo.compareCount, repo.compareNext)
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

// TestMovementCorrectionPolicyUsesDatabaseConfig 验证客户端纠偏阈值只从当前数据库移动配置派生。
func TestMovementCorrectionPolicyUsesDatabaseConfig(t *testing.T) {
	policy := movementCorrectionPolicy(MovementConfig{
		SpeedMilliCellsPerSecond: 3750,
		MaxElapsedMS:             300,
		AxisToleranceMilli:       125,
	})
	if policy.IgnoreDistanceMilli != 125 || policy.SnapDistanceMilli != 1125 {
		t.Fatalf("movementCorrectionPolicy() = %+v, want ignore=125 snap=1125", policy)
	}
}

// TestMovementCorrectionPolicyKeepsThresholdsOrdered 验证低速配置仍会为平滑区间保留严格有序的上下界。
func TestMovementCorrectionPolicyKeepsThresholdsOrdered(t *testing.T) {
	policy := movementCorrectionPolicy(MovementConfig{
		SpeedMilliCellsPerSecond: 1,
		MaxElapsedMS:             1,
		AxisToleranceMilli:       125,
	})
	if policy.IgnoreDistanceMilli != 125 || policy.SnapDistanceMilli != 126 {
		t.Fatalf("movementCorrectionPolicy() = %+v, want ignore=125 snap=126", policy)
	}
}

// TestMovementCorrectionPolicyClampsUint32Overflow 验证异常极大配置不会让阈值回绕或失去严格顺序。
func TestMovementCorrectionPolicyClampsUint32Overflow(t *testing.T) {
	const maximumUint32 uint32 = ^uint32(0)
	policy := movementCorrectionPolicy(MovementConfig{
		SpeedMilliCellsPerSecond: maximumUint32,
		MaxElapsedMS:             maximumUint32,
		AxisToleranceMilli:       maximumUint32,
	})
	if policy.IgnoreDistanceMilli != maximumUint32-1 || policy.SnapDistanceMilli != maximumUint32 {
		t.Fatalf("movementCorrectionPolicy() = %+v, want ignore=%d snap=%d", policy, maximumUint32-1, maximumUint32)
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
