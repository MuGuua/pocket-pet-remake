package world

import (
	"context"
	"strings"
	"sync"
)

const (
	MinMovementElapsedMS          uint32 = 50
	MaxMovementElapsedMS          uint32 = 2000
	MaxMovementAxisTolerance      uint32 = 1000
	MaxSceneBoundaryCoordinateAbs int32  = 10000000
)

type Service struct {
	repo                Repository
	movementRepo        MovementStateRepository
	movementConfigRepo  MovementConfigRepository
	movementConfigMu    sync.RWMutex
	movementConfig      MovementConfig
	sceneBoundaryRepo   SceneBoundaryRepository
	sceneBoundaryMu     sync.RWMutex
	sceneBoundaries     map[uint32]SceneBoundary
	sceneNavigationRepo SceneNavigationRepository
	sceneNavigationMu   sync.RWMutex
	sceneNavigations    map[uint32]SceneNavigation
}

func NewService(repo Repository) *Service {
	service := &Service{repo: repo}
	if configRepo, ok := repo.(MovementConfigRepository); ok {
		service.movementConfigRepo = configRepo
	}
	if boundaryRepo, ok := repo.(SceneBoundaryRepository); ok {
		service.sceneBoundaryRepo = boundaryRepo
	}
	if navigationRepo, ok := repo.(SceneNavigationRepository); ok {
		service.sceneNavigationRepo = navigationRepo
	}
	return service
}

// RefreshMovementConfig 从 PostgreSQL刷新服务端权威移动配置。
func (s *Service) RefreshMovementConfig(ctx context.Context) error {
	if s == nil || s.movementConfigRepo == nil {
		return ErrMovementConfigUnavailable
	}
	config, err := s.movementConfigRepo.GetMovementConfig(ctx)
	if err != nil {
		return err
	}
	if config.SpeedMilliCellsPerSecond == 0 || config.MaxElapsedMS == 0 {
		return ErrMovementConfigUnavailable
	}
	s.movementConfigMu.Lock()
	s.movementConfig = config
	s.movementConfigMu.Unlock()
	return nil
}

// MovementConfigSnapshot 返回当前已加载的只读移动配置副本。
func (s *Service) MovementConfigSnapshot() (MovementConfig, error) {
	if s == nil {
		return MovementConfig{}, ErrMovementConfigUnavailable
	}
	s.movementConfigMu.RLock()
	config := s.movementConfig
	s.movementConfigMu.RUnlock()
	if config.SpeedMilliCellsPerSecond == 0 || config.MaxElapsedMS == 0 {
		return MovementConfig{}, ErrMovementConfigUnavailable
	}
	return config, nil
}

// GetAdminMovementConfig 返回数据库中的配置，供后台明确展示持久化事实来源。
func (s *Service) GetAdminMovementConfig(ctx context.Context) (MovementConfig, error) {
	if s == nil || s.movementConfigRepo == nil {
		return MovementConfig{}, ErrMovementConfigUnavailable
	}
	return s.movementConfigRepo.GetMovementConfig(ctx)
}

// UpdateAdminMovementConfig 校验后台输入，持久化成功后立即刷新当前进程的运行时快照。
func (s *Service) UpdateAdminMovementConfig(ctx context.Context, input AdminUpdateMovementConfigInput) (MovementConfig, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if input.SpeedMilliCellsPerSecond == 0 || input.MaxElapsedMS < MinMovementElapsedMS || input.MaxElapsedMS > MaxMovementElapsedMS || input.AxisToleranceMilli > MaxMovementAxisTolerance || input.AdminUserID == 0 || input.Reason == "" || len([]rune(input.Reason)) > 500 {
		return MovementConfig{}, ErrMovementConfigInvalid
	}
	updated, err := s.movementConfigRepo.UpdateMovementConfig(ctx, input)
	if err != nil {
		return MovementConfig{}, err
	}
	s.movementConfigMu.Lock()
	s.movementConfig = updated
	s.movementConfigMu.Unlock()
	return updated, nil
}

// RefreshSceneBoundaryCache 从 PostgreSQL加载全部启用场景边界，并一次性替换运行时只读快照。
func (s *Service) RefreshSceneBoundaryCache(ctx context.Context) error {
	if s == nil || s.sceneBoundaryRepo == nil {
		return ErrSceneBoundaryUnavailable
	}
	boundaries, err := s.sceneBoundaryRepo.ListSceneBoundaries(ctx)
	if err != nil {
		return err
	}
	if len(boundaries) == 0 {
		return ErrSceneBoundaryUnavailable
	}
	next := make(map[uint32]SceneBoundary, len(boundaries))
	for _, boundary := range boundaries {
		if !isValidSceneBoundary(boundary.SceneID, boundary.MinX, boundary.MinY, boundary.MaxX, boundary.MaxY) {
			return ErrSceneBoundaryInvalid
		}
		if _, duplicated := next[boundary.SceneID]; duplicated {
			return ErrSceneBoundaryInvalid
		}
		next[boundary.SceneID] = boundary
	}
	s.sceneBoundaryMu.Lock()
	s.sceneBoundaries = next
	s.sceneBoundaryMu.Unlock()
	return nil
}

// SceneBoundarySnapshot 返回指定场景当前生效的边界副本，调用方不能修改内部缓存。
func (s *Service) SceneBoundarySnapshot(sceneID uint32) (SceneBoundary, error) {
	if s == nil || sceneID == 0 {
		return SceneBoundary{}, ErrSceneBoundaryUnavailable
	}
	s.sceneBoundaryMu.RLock()
	boundary, ok := s.sceneBoundaries[sceneID]
	s.sceneBoundaryMu.RUnlock()
	if !ok {
		return SceneBoundary{}, ErrSceneBoundaryUnavailable
	}
	return boundary, nil
}

// GetAdminSceneBoundaries 返回数据库中的启用场景边界，供后台展示持久化事实来源。
func (s *Service) GetAdminSceneBoundaries(ctx context.Context) ([]SceneBoundary, error) {
	if s == nil || s.sceneBoundaryRepo == nil {
		return nil, ErrSceneBoundaryUnavailable
	}
	return s.sceneBoundaryRepo.ListSceneBoundaries(ctx)
}

// UpdateAdminSceneBoundary 校验并持久化场景边界，成功后原子替换对应运行时快照。
func (s *Service) UpdateAdminSceneBoundary(ctx context.Context, sceneID uint32, input AdminUpdateSceneBoundaryInput) (SceneBoundary, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if !isValidSceneBoundary(sceneID, input.MinX, input.MinY, input.MaxX, input.MaxY) || input.AdminUserID == 0 || input.Reason == "" || len([]rune(input.Reason)) > 500 {
		return SceneBoundary{}, ErrSceneBoundaryInvalid
	}
	if s == nil || s.sceneBoundaryRepo == nil {
		return SceneBoundary{}, ErrSceneBoundaryUnavailable
	}
	updated, err := s.sceneBoundaryRepo.UpdateSceneBoundary(ctx, sceneID, input)
	if err != nil {
		return SceneBoundary{}, err
	}
	s.sceneBoundaryMu.Lock()
	next := make(map[uint32]SceneBoundary, len(s.sceneBoundaries)+1)
	for cachedSceneID, cachedBoundary := range s.sceneBoundaries {
		next[cachedSceneID] = cachedBoundary
	}
	next[sceneID] = updated
	s.sceneBoundaries = next
	s.sceneBoundaryMu.Unlock()
	return updated, nil
}

// isValidSceneBoundary 统一校验后台输入和启动缓存，避免非法矩形进入移动热路径。
func isValidSceneBoundary(sceneID uint32, minX int32, minY int32, maxX int32, maxY int32) bool {
	if sceneID == 0 || maxX <= minX || maxY <= minY {
		return false
	}
	return minX >= -MaxSceneBoundaryCoordinateAbs && minY >= -MaxSceneBoundaryCoordinateAbs && maxX <= MaxSceneBoundaryCoordinateAbs && maxY <= MaxSceneBoundaryCoordinateAbs
}

// EvaluateMovement 按服务端时间、数据库速度和四方向输入裁剪客户端候选位置。
func (s *Service) EvaluateMovement(current MovementState, intent MovementIntent) (MovementResult, error) {
	config, err := s.MovementConfigSnapshot()
	if err != nil {
		return MovementResult{}, err
	}
	direction, err := resolveMovementDirection(current, intent)
	if err != nil {
		return MovementResult{}, err
	}
	next := current
	next.LastMoveSeq = intent.MoveSeq
	next.LastServerTickMS = intent.ServerTickMS
	next.Speed = config.SpeedMilliCellsPerSecond
	next.Moving = intent.Moving
	if isCardinalMovementVector(intent.Facing) {
		next.Facing = intent.Facing
	} else if direction != (Vec2i{}) {
		next.Facing = direction
	}
	if !intent.HasCandidate || direction == (Vec2i{}) {
		return MovementResult{State: next, Corrected: intent.HasCandidate && intent.CandidatePos != current.PrecisePos}, nil
	}

	elapsedMS := intent.ServerTickMS - current.LastServerTickMS
	if elapsedMS < 0 {
		elapsedMS = 0
	}
	if elapsedMS > int64(config.MaxElapsedMS) {
		elapsedMS = int64(config.MaxElapsedMS)
	}
	maxDistance := int32(int64(config.SpeedMilliCellsPerSecond) * elapsedMS / 1000)
	delta := Vec2i{X: intent.CandidatePos.X - current.PrecisePos.X, Y: intent.CandidatePos.Y - current.PrecisePos.Y}
	if direction.X != 0 && absInt32(delta.Y) > int32(config.AxisToleranceMilli) {
		return MovementResult{}, ErrMovementAxisInvalid
	}
	if direction.Y != 0 && absInt32(delta.X) > int32(config.AxisToleranceMilli) {
		return MovementResult{}, ErrMovementAxisInvalid
	}
	progress := delta.X*direction.X + delta.Y*direction.Y
	if progress < 0 {
		progress = 0
	}
	if progress > maxDistance {
		progress = maxDistance
	}
	next.PrecisePos = Vec2i{
		X: current.PrecisePos.X + direction.X*progress,
		Y: current.PrecisePos.Y + direction.Y*progress,
	}
	boundary, err := s.SceneBoundarySnapshot(current.SceneID)
	if err != nil {
		return MovementResult{}, err
	}
	next.PrecisePos.X = clampInt32(next.PrecisePos.X, boundary.MinX, boundary.MaxX)
	next.PrecisePos.Y = clampInt32(next.PrecisePos.Y, boundary.MinY, boundary.MaxY)
	next.PrecisePos, err = s.clampMovementToNavigation(current.SceneID, current.PrecisePos, next.PrecisePos, direction)
	if err != nil {
		return MovementResult{}, err
	}
	next.PersistedPos = Vec2i{X: roundFixedCoordinate(next.PrecisePos.X), Y: roundFixedCoordinate(next.PrecisePos.Y)}
	return MovementResult{State: next, Corrected: next.PrecisePos != intent.CandidatePos}, nil
}

func resolveMovementDirection(current MovementState, intent MovementIntent) (Vec2i, error) {
	if intent.Input != nil {
		if intent.Moving {
			if !isCardinalMovementVector(*intent.Input) {
				return Vec2i{}, ErrMovementInputInvalid
			}
			return *intent.Input, nil
		}
		if *intent.Input != (Vec2i{}) {
			return Vec2i{}, ErrMovementInputInvalid
		}
	}
	if intent.Moving {
		if isCardinalMovementVector(intent.Facing) {
			return intent.Facing, nil
		}
		return Vec2i{}, ErrMovementInputInvalid
	}
	if current.Moving && isCardinalMovementVector(current.Facing) {
		return current.Facing, nil
	}
	return Vec2i{}, nil
}

func isCardinalMovementVector(value Vec2i) bool {
	return ((value.X == -1 || value.X == 1) && value.Y == 0) || ((value.Y == -1 || value.Y == 1) && value.X == 0)
}

func absInt32(value int32) int32 {
	if value < 0 {
		return -value
	}
	return value
}

// clampInt32 把坐标限制在数据库定义的闭区间内。
func clampInt32(value int32, minimum int32, maximum int32) int32 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func roundFixedCoordinate(value int32) int32 {
	if value >= 0 {
		return (value + MovementPositionFixedScale/2) / MovementPositionFixedScale
	}
	return (value - MovementPositionFixedScale/2) / MovementPositionFixedScale
}

// SetMovementStateRepository 注入在线玩家权威移动状态仓储。
func (s *Service) SetMovementStateRepository(repo MovementStateRepository) {
	if s == nil {
		return
	}
	s.movementRepo = repo
}

// MovementStateEnabled 表示服务端已经装配在线权威移动状态仓储。
func (s *Service) MovementStateEnabled() bool {
	return s != nil && s.movementRepo != nil
}

// LoadMovementState 读取玩家当前在线权威移动状态。
func (s *Service) LoadMovementState(ctx context.Context, playerID uint64) (*MovementState, error) {
	if s == nil || s.movementRepo == nil {
		return nil, ErrMovementStateNotFound
	}
	return s.movementRepo.Load(ctx, playerID)
}

// MovePlayer 执行普通同场景移动的服务端权威用例。
// 该入口负责加载 Redis 状态、验证会话与场景、归一化旧客户端字段、计算合法位置，并通过 CAS 原子推进状态。
func (s *Service) MovePlayer(ctx context.Context, input MovePlayerInput) (MovePlayerResult, error) {
	if s == nil || s.movementRepo == nil {
		return MovePlayerResult{}, ErrMovementStateNotFound
	}
	current, err := s.movementRepo.Load(ctx, input.PlayerID)
	if err != nil {
		return MovePlayerResult{}, err
	}
	result := MovePlayerResult{PreviousState: *current, State: *current}
	if current.PlayerID != input.PlayerID {
		// 仓储若返回了其他玩家状态，按读取失败处理，避免把不属于当前连接的位置暴露给传输层。
		return MovePlayerResult{}, ErrMovementStateNotFound
	}
	if current.SessionID != input.SessionID {
		return result, ErrMovementSessionMismatch
	}
	if current.SceneID != input.SceneID {
		return result, ErrMovementSceneMismatch
	}
	// 旧客户端未携带候选坐标时保持原心跳语义：只确认当前权威状态，不占用移动序号，也不写 Redis。
	if !input.HasCandidate {
		return result, nil
	}

	movementResult, err := s.EvaluateMovement(*current, MovementIntent{
		Input:        input.Input,
		CandidatePos: input.CandidatePos,
		HasCandidate: true,
		Facing:       normalizeReportedMovementFacing(current.PersistedPos, input.CandidateCell, input.Facing),
		Moving:       normalizeReportedMovementState(current.PersistedPos, input.CandidateCell, input.Moving),
		MoveSeq:      input.MoveSeq,
		ServerTickMS: input.ServerTickMS,
	})
	if err != nil {
		return result, err
	}
	if err := validateMovementStateAdvance(*current, &movementResult.State); err != nil {
		return result, err
	}
	if err := s.movementRepo.CompareAndSet(ctx, current.LastMoveSeq, movementResult.State); err != nil {
		return result, err
	}
	result.State = movementResult.State
	result.Corrected = movementResult.Corrected
	return result, nil
}

// normalizeReportedMovementFacing 只接受四方向单位向量；旧客户端缺失或上报非法朝向时按整数格位移推导。
func normalizeReportedMovementFacing(fromPos Vec2i, toPos Vec2i, facing *Vec2i) Vec2i {
	if facing != nil && isCardinalMovementVector(*facing) {
		return *facing
	}
	offsetX := toPos.X - fromPos.X
	offsetY := toPos.Y - fromPos.Y
	if offsetX != 0 {
		if offsetX < 0 {
			return Vec2i{X: -1}
		}
		return Vec2i{X: 1}
	}
	if offsetY < 0 {
		return Vec2i{Y: -1}
	}
	return Vec2i{Y: 1}
}

// normalizeReportedMovementState 优先使用客户端明确上报的起停状态；旧客户端则按整数格是否变化兼容推导。
func normalizeReportedMovementState(fromPos Vec2i, toPos Vec2i, moving *bool) bool {
	if moving != nil {
		return *moving
	}
	return fromPos != toPos
}

// InitializeMovementState 使用进入世界或切图后的权威位置初始化当前会话状态。
func (s *Service) InitializeMovementState(ctx context.Context, state MovementState) error {
	if s == nil || s.movementRepo == nil {
		return ErrMovementStateNotFound
	}
	return s.movementRepo.Initialize(ctx, state)
}

// AdvanceMovementState 校验会话、场景代次和移动序号后，通过仓储 CAS 原子推进权威状态。
// 场景切换等非普通移动流程继续复用该入口；普通移动统一通过 MovePlayer 完成加载、计算和推进。
func (s *Service) AdvanceMovementState(ctx context.Context, next MovementState) error {
	if s == nil || s.movementRepo == nil {
		return ErrMovementStateNotFound
	}
	current, err := s.movementRepo.Load(ctx, next.PlayerID)
	if err != nil {
		return err
	}
	if err := validateMovementStateAdvance(*current, &next); err != nil {
		return err
	}
	return s.movementRepo.CompareAndSet(ctx, current.LastMoveSeq, next)
}

// validateMovementStateAdvance 统一验证一次权威状态推进，并确保位置版本严格递增。
func validateMovementStateAdvance(current MovementState, next *MovementState) error {
	if next == nil {
		return ErrMovementStateNotFound
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
	return nil
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
