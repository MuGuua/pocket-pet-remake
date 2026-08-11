package world

import "context"

type Service struct {
	repo         Repository
	movementRepo MovementStateRepository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SetMovementStateRepository 注入在线玩家权威移动状态仓储。
func (s *Service) SetMovementStateRepository(repo MovementStateRepository) {
	if s == nil {
		return
	}
	s.movementRepo = repo
}

// InitializeMovementState 使用进入世界或切图后的权威位置初始化当前会话状态。
func (s *Service) InitializeMovementState(ctx context.Context, state MovementState) error {
	if s == nil || s.movementRepo == nil {
		return ErrMovementStateNotFound
	}
	return s.movementRepo.Initialize(ctx, state)
}

// AdvanceMovementState 校验会话、场景代次和移动序号后，通过仓储 CAS 原子推进权威状态。
func (s *Service) AdvanceMovementState(ctx context.Context, next MovementState) error {
	if s == nil || s.movementRepo == nil {
		return ErrMovementStateNotFound
	}
	current, err := s.movementRepo.Load(ctx, next.PlayerID)
	if err != nil {
		return err
	}
	if current.SessionID != next.SessionID {
		return ErrMovementSessionMismatch
	}
	if current.SceneID != next.SceneID || current.SceneVersion != next.SceneVersion {
		return ErrMovementSceneMismatch
	}
	if next.LastMoveSeq <= current.LastMoveSeq {
		return ErrMovementSequenceStale
	}
	if next.PositionVersion <= current.PositionVersion {
		next.PositionVersion = current.PositionVersion + 1
	}
	return s.movementRepo.CompareAndSet(ctx, current.LastMoveSeq, next)
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
