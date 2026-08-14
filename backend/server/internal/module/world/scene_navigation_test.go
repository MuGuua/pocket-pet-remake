package world

import (
	"context"
	"errors"
	"testing"
)

// sceneNavigationRepositoryStub 为导航领域测试提供最小仓储实现，并允许测试精确控制发布与回滚返回值。
type sceneNavigationRepositoryStub struct {
	publishedItems []SceneNavigation
	publishResult  SceneNavigation
	rollbackResult SceneNavigation
}

// ListPublishedSceneNavigations 返回启动缓存测试所需的已发布导航副本。
func (r *sceneNavigationRepositoryStub) ListPublishedSceneNavigations(_ context.Context) ([]SceneNavigation, error) {
	result := make([]SceneNavigation, len(r.publishedItems))
	copy(result, r.publishedItems)
	for index := range result {
		result[index].NavigationData = append([]byte(nil), result[index].NavigationData...)
	}
	return result, nil
}

// ListAdminSceneNavigations 在当前测试中不负责验证版本列表，因此返回空集合。
func (r *sceneNavigationRepositoryStub) ListAdminSceneNavigations(_ context.Context, _ uint32) ([]SceneNavigation, error) {
	return []SceneNavigation{}, nil
}

// CreateSceneNavigationDraft 在当前测试中不负责验证草稿创建，因此原样返回一个有效草稿占位值。
func (r *sceneNavigationRepositoryStub) CreateSceneNavigationDraft(_ context.Context, input CreateSceneNavigationDraftInput) (SceneNavigation, error) {
	return SceneNavigation{
		SceneID: input.SceneID, Version: 1, OriginX: input.OriginX, OriginY: input.OriginY,
		GridWidth: input.GridWidth, GridHeight: input.GridHeight, CellSizeMilli: input.CellSizeMilli,
		NavigationData: append([]byte(nil), input.NavigationData...), Status: SceneNavigationStatusDraft,
	}, nil
}

// PublishSceneNavigation 返回测试预置的发布版本，用于验证服务层是否即时替换运行时缓存。
func (r *sceneNavigationRepositoryStub) PublishSceneNavigation(_ context.Context, _ uint64, _ AdminPublishSceneNavigationInput) (SceneNavigation, error) {
	result := r.publishResult
	result.NavigationData = append([]byte(nil), result.NavigationData...)
	return result, nil
}

// RollbackSceneNavigation 返回测试预置的回滚版本，用于验证服务层是否即时替换运行时缓存。
func (r *sceneNavigationRepositoryStub) RollbackSceneNavigation(_ context.Context, _ uint32, _ AdminRollbackSceneNavigationInput) (SceneNavigation, error) {
	result := r.rollbackResult
	result.NavigationData = append([]byte(nil), result.NavigationData...)
	return result, nil
}

// TestEvaluateMovementClampsPathBeforeBlockedCell 验证单个移动包不能跨越路径中间的静态阻挡格。
func TestEvaluateMovementClampsPathBeforeBlockedCell(t *testing.T) {
	service := movementServiceForEvaluationTest()
	service.movementConfig.SpeedMilliCellsPerSecond = 10000
	service.sceneBoundaries[3] = SceneBoundary{SceneID: 3, MinX: 0, MinY: 0, MaxX: 2999, MaxY: 999}
	service.sceneNavigations[3] = SceneNavigation{
		SceneID: 3, Version: 1, OriginX: 0, OriginY: 0,
		GridWidth: 3, GridHeight: 1, CellSizeMilli: 1000,
		NavigationData: []byte{0xa0}, Status: SceneNavigationStatusPublished,
	}
	current := movementStateForTest()
	current.PrecisePos = Vec2i{X: 500, Y: 500}
	current.PersistedPos = Vec2i{X: 1, Y: 1}
	current.LastServerTickMS = 1000
	right := Vec2i{X: 1}

	result, err := service.EvaluateMovement(current, MovementIntent{
		Input: &right, CandidatePos: Vec2i{X: 2500, Y: 500}, HasCandidate: true,
		Facing: right, Moving: true, MoveSeq: 8, ServerTickMS: 1300,
	})
	if err != nil {
		t.Fatalf("EvaluateMovement() error = %v", err)
	}
	if result.State.PrecisePos != (Vec2i{X: 999, Y: 500}) || !result.Corrected {
		t.Fatalf("result = %+v, want correction before blocked cell at (999,500)", result)
	}
}

// TestEvaluateMovementRejectsBlockedCurrentPosition 验证异常落在阻挡格的运行态玩家不能继续移动扩大错误状态。
func TestEvaluateMovementRejectsBlockedCurrentPosition(t *testing.T) {
	service := movementServiceForEvaluationTest()
	service.sceneNavigations[3] = SceneNavigation{
		SceneID: 3, Version: 1, OriginX: 0, OriginY: 0,
		GridWidth: 3, GridHeight: 1, CellSizeMilli: 1000,
		NavigationData: []byte{0x60}, Status: SceneNavigationStatusPublished,
	}
	current := movementStateForTest()
	current.PrecisePos = Vec2i{X: 500, Y: 500}
	current.LastServerTickMS = 1000
	right := Vec2i{X: 1}

	_, err := service.EvaluateMovement(current, MovementIntent{
		Input: &right, CandidatePos: Vec2i{X: 700, Y: 500}, HasCandidate: true,
		Facing: right, Moving: true, MoveSeq: 8, ServerTickMS: 1100,
	})
	if !errors.Is(err, ErrSceneNavigationBlocked) {
		t.Fatalf("EvaluateMovement() error = %v, want %v", err, ErrSceneNavigationBlocked)
	}
}

// TestEvaluateMovementRejectsMissingSceneNavigation 验证缺少发布位图时普通移动必须失败关闭，不能退化成无碰撞移动。
func TestEvaluateMovementRejectsMissingSceneNavigation(t *testing.T) {
	service := movementServiceForEvaluationTest()
	delete(service.sceneNavigations, 3)
	current := movementStateForTest()
	current.LastServerTickMS = 1000
	right := Vec2i{X: 1}

	_, err := service.EvaluateMovement(current, MovementIntent{
		Input: &right, CandidatePos: Vec2i{X: 12100, Y: 8000}, HasCandidate: true,
		Facing: right, Moving: true, MoveSeq: 8, ServerTickMS: 1100,
	})
	if !errors.Is(err, ErrSceneNavigationUnavailable) {
		t.Fatalf("EvaluateMovement() error = %v, want %v", err, ErrSceneNavigationUnavailable)
	}
}

// TestSceneNavigationSnapshotReturnsDeepCopy 验证调用方修改返回字节不会污染服务端运行时只读缓存。
func TestSceneNavigationSnapshotReturnsDeepCopy(t *testing.T) {
	service := movementServiceForEvaluationTest()

	first, err := service.SceneNavigationSnapshot(3)
	if err != nil {
		t.Fatalf("SceneNavigationSnapshot() error = %v", err)
	}
	first.NavigationData[0] = 0
	second, err := service.SceneNavigationSnapshot(3)
	if err != nil {
		t.Fatalf("SceneNavigationSnapshot() second error = %v", err)
	}
	if second.NavigationData[0] != 0xff {
		t.Fatalf("cached navigation byte = %#x, want %#x", second.NavigationData[0], byte(0xff))
	}
}

// TestPublishAdminSceneNavigationRefreshesSnapshot 验证后台发布事务成功后当前进程立即使用新版本位图。
func TestPublishAdminSceneNavigationRefreshesSnapshot(t *testing.T) {
	published := SceneNavigation{
		NavigationID: 27, SceneID: 3, Version: 2, OriginX: 0, OriginY: 0,
		GridWidth: 2, GridHeight: 1, CellSizeMilli: 1000,
		NavigationData: []byte{0x80}, Status: SceneNavigationStatusPublished,
	}
	repo := &sceneNavigationRepositoryStub{publishResult: published}
	service := &Service{
		sceneNavigationRepo: repo,
		sceneNavigations: map[uint32]SceneNavigation{
			3: {NavigationID: 3, SceneID: 3, Version: 1, OriginX: 0, OriginY: 0, GridWidth: 1, GridHeight: 1, CellSizeMilli: 1000, NavigationData: []byte{0x80}, Status: SceneNavigationStatusPublished},
		},
	}

	result, err := service.PublishAdminSceneNavigation(context.Background(), 27, AdminPublishSceneNavigationInput{Reason: "发布测试版本", AdminUserID: 7})
	if err != nil {
		t.Fatalf("PublishAdminSceneNavigation() error = %v", err)
	}
	result.NavigationData[0] = 0
	cached, err := service.SceneNavigationSnapshot(3)
	if err != nil {
		t.Fatalf("SceneNavigationSnapshot() error = %v", err)
	}
	if cached.Version != 2 || cached.NavigationID != 27 || cached.NavigationData[0] != 0x80 {
		t.Fatalf("cached navigation = %+v, want published version 2", cached)
	}
}

// TestRollbackAdminSceneNavigationRefreshesSnapshot 验证后台回滚会生成并即时启用新的发布版本，而不是修改历史版本。
func TestRollbackAdminSceneNavigationRefreshesSnapshot(t *testing.T) {
	rolledBack := SceneNavigation{
		NavigationID: 28, SceneID: 3, Version: 3, OriginX: 0, OriginY: 0,
		GridWidth: 1, GridHeight: 1, CellSizeMilli: 1000,
		NavigationData: []byte{0x80}, Status: SceneNavigationStatusPublished,
	}
	repo := &sceneNavigationRepositoryStub{rollbackResult: rolledBack}
	service := &Service{
		sceneNavigationRepo: repo,
		sceneNavigations: map[uint32]SceneNavigation{
			3: {NavigationID: 27, SceneID: 3, Version: 2, OriginX: 0, OriginY: 0, GridWidth: 2, GridHeight: 1, CellSizeMilli: 1000, NavigationData: []byte{0xc0}, Status: SceneNavigationStatusPublished},
		},
	}

	result, err := service.RollbackAdminSceneNavigation(context.Background(), 3, AdminRollbackSceneNavigationInput{SourceVersion: 1, Reason: "回滚测试版本", AdminUserID: 7})
	if err != nil {
		t.Fatalf("RollbackAdminSceneNavigation() error = %v", err)
	}
	result.NavigationData[0] = 0
	cached, err := service.SceneNavigationSnapshot(3)
	if err != nil {
		t.Fatalf("SceneNavigationSnapshot() error = %v", err)
	}
	if cached.Version != 3 || cached.NavigationID != 28 || cached.NavigationData[0] != 0x80 {
		t.Fatalf("cached navigation = %+v, want rollback version 3", cached)
	}
}
