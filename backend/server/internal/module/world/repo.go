package world

import "context"

type Repository interface {
	GetSceneSnapshot(ctx context.Context, playerID uint64, sceneID uint32, selfPos Vec2i) (*SceneSnapshot, error)
	EvaluateMove(ctx context.Context, playerID uint64, sceneID uint32, currentPos Vec2i, targetPos Vec2i) (*MoveDecision, error)
}
