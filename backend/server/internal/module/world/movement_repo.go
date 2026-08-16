package world

import "context"

// MovementStateRepository 管理在线玩家的短时权威移动状态；永久位置仍由玩家仓储写入 PostgreSQL。
type MovementStateRepository interface {
	Load(ctx context.Context, playerID uint64) (*MovementState, error)
	Initialize(ctx context.Context, state MovementState) error
	CompareAndSet(ctx context.Context, expectedMoveSeq uint32, state MovementState) error
	ClaimDirtyPlayerIDs(ctx context.Context, limit uint32) ([]uint64, error)
	RequeueDirtyPlayerIDs(ctx context.Context, playerIDs []uint64) error
	Delete(ctx context.Context, playerID uint64) error
}

// MovementConfigRepository 从 PostgreSQL读取已启用的世界移动配置。
type MovementConfigRepository interface {
	GetMovementConfig(ctx context.Context) (MovementConfig, error)
	UpdateMovementConfig(ctx context.Context, input AdminUpdateMovementConfigInput) (MovementConfig, error)
}

// SceneBoundaryRepository 从 PostgreSQL读取和维护启用场景的服务端权威矩形边界。
type SceneBoundaryRepository interface {
	ListSceneBoundaries(ctx context.Context) ([]SceneBoundary, error)
	UpdateSceneBoundary(ctx context.Context, sceneID uint32, input AdminUpdateSceneBoundaryInput) (SceneBoundary, error)
}

// SceneNavigationRepository 从 PostgreSQL读取已发布静态通行位图，并维护后台版本发布与回滚。
type SceneNavigationRepository interface {
	ListPublishedSceneNavigations(ctx context.Context) ([]SceneNavigation, error)
	ListAdminSceneNavigations(ctx context.Context, sceneID uint32) ([]SceneNavigation, error)
	CreateSceneNavigationDraft(ctx context.Context, input CreateSceneNavigationDraftInput) (SceneNavigation, error)
	PublishSceneNavigation(ctx context.Context, navigationID uint64, input AdminPublishSceneNavigationInput) (SceneNavigation, error)
	RollbackSceneNavigation(ctx context.Context, sceneID uint32, input AdminRollbackSceneNavigationInput) (SceneNavigation, error)
}
