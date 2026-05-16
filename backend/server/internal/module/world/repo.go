package world

import "context"

type Repository interface {
	GetSceneSnapshot(ctx context.Context, playerID uint64, sceneID uint32, selfPos Vec2i) (*SceneSnapshot, error)
	EvaluateTransfer(ctx context.Context, playerID uint64, sceneID uint32, currentPos Vec2i, targetSceneID uint32, portalID uint32) (*MoveDecision, error)
}
