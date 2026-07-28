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

// EvaluateTransfer 使用服务端玩家档案中的权威等级判断目标地图是否允许进入。
// playerLevel 只能由服务端玩家服务提供，客户端请求不会参与该值计算。
func (s *Service) EvaluateTransfer(ctx context.Context, playerID uint64, playerLevel uint32, sceneID uint32, currentPos Vec2i, targetSceneID uint32, portalID uint32) (*MoveDecision, error) {
	decision, err := s.repo.EvaluateTransfer(ctx, playerID, playerLevel, sceneID, currentPos, targetSceneID, portalID)
	if err != nil {
		return nil, err
	}
	if decision == nil {
		return nil, ErrSnapshotUnavailable
	}
	return decision, nil
}

// EvaluateMapTeleport 校验世界地图快速传送，并返回数据库配置的目标地图中心落点。
func (s *Service) EvaluateMapTeleport(ctx context.Context, playerID uint64, playerLevel uint32, sceneID uint32, currentPos Vec2i, targetSceneID uint32) (*MoveDecision, error) {
	decision, err := s.repo.EvaluateMapTeleport(ctx, playerID, playerLevel, sceneID, currentPos, targetSceneID)
	if err != nil {
		return nil, err
	}
	if decision == nil {
		return nil, ErrSnapshotUnavailable
	}
	return decision, nil
}
