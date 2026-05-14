package world

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetSceneSnapshot(ctx context.Context, playerID uint64, sceneID uint32, selfPos Vec2i) (*SceneSnapshot, error) {
	snapshot, err := s.repo.GetSceneSnapshot(ctx, playerID, sceneID, selfPos)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, ErrSnapshotUnavailable
	}
	if snapshot.NearbyEntities == nil {
		snapshot.NearbyEntities = []Entity{}
	}
	return snapshot, nil
}

func (s *Service) EvaluateMove(ctx context.Context, playerID uint64, sceneID uint32, currentPos Vec2i, targetPos Vec2i) (*MoveDecision, error) {
	decision, err := s.repo.EvaluateMove(ctx, playerID, sceneID, currentPos, targetPos)
	if err != nil {
		return nil, err
	}
	if decision == nil {
		return nil, ErrSnapshotUnavailable
	}
	return decision, nil
}
