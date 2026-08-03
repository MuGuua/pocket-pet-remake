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

// ApplyClientPortalSpawn 在普通门拓扑、目标场景与等级均验证通过后，采用客户端目标场景脚本选定的入口落点。
// portalID 必须是本次已验证的普通门编号；clientSpawn 为空或包含负坐标时保留仓储返回的兼容落点。
func (s *Service) ApplyClientPortalSpawn(decision *MoveDecision, portalID uint32, clientSpawn *Vec2i) *MoveDecision {
	if decision == nil || !decision.Accepted || portalID == 0 || clientSpawn == nil {
		return decision
	}
	if clientSpawn.X < 0 || clientSpawn.Y < 0 {
		return decision
	}
	decision.SpawnPos = *clientSpawn
	return decision
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
