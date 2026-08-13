package world

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

const (
	// SceneNavigationStatusArchived 表示曾经发布、当前已被其他版本替代的历史版本。
	SceneNavigationStatusArchived int16 = 0
	// SceneNavigationStatusPublished 表示当前服务端移动判定正在使用的发布版本。
	SceneNavigationStatusPublished int16 = 1
	// SceneNavigationStatusDraft 表示已上传但尚未影响在线玩家的草稿版本。
	SceneNavigationStatusDraft int16 = 2

	// MaxSceneNavigationDimension 限制单边网格尺寸，避免错误上传占用过多内存。
	MaxSceneNavigationDimension uint32 = 4096
	// MaxSceneNavigationCellCount 限制单张地图的总格数，和数据库约束保持一致。
	MaxSceneNavigationCellCount uint64 = 4194304
	// MaxSceneNavigationCellSizeMilli 限制导航单元尺寸，单位为千分之一场景格。
	MaxSceneNavigationCellSizeMilli uint32 = 100000
)

// SceneNavigation 是服务端运行时使用的已发布静态通行位图和后台版本元数据。
type SceneNavigation struct {
	NavigationID           uint64    `json:"navigation_id"`
	SceneID                uint32    `json:"scene_id"`
	SceneCode              string    `json:"scene_code"`
	SceneName              string    `json:"scene_name"`
	Version                uint32    `json:"version"`
	OriginX                int32     `json:"origin_x_milli"`
	OriginY                int32     `json:"origin_y_milli"`
	GridWidth              uint32    `json:"grid_width"`
	GridHeight             uint32    `json:"grid_height"`
	CellSizeMilli          uint32    `json:"cell_size_milli"`
	NavigationData         []byte    `json:"-"`
	DataHash               string    `json:"data_hash"`
	WalkableCellCount      uint32    `json:"walkable_cell_count"`
	SourceScenePath        string    `json:"source_scene_path"`
	Status                 int16     `json:"status"`
	ChangeReason           string    `json:"change_reason"`
	PublishReason          string    `json:"publish_reason"`
	CreatedByAdminUserID   uint64    `json:"created_by_admin_user_id"`
	PublishedByAdminUserID uint64    `json:"published_by_admin_user_id"`
	CreatedAt              time.Time `json:"created_at"`
	PublishedAt            time.Time `json:"published_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// AdminCreateSceneNavigationInput 是后台上传 Godot 导出位图时使用的输入。
type AdminCreateSceneNavigationInput struct {
	SceneID           uint32 `json:"scene_id"`
	OriginX           int32  `json:"origin_x_milli"`
	OriginY           int32  `json:"origin_y_milli"`
	GridWidth         uint32 `json:"grid_width"`
	GridHeight        uint32 `json:"grid_height"`
	CellSizeMilli     uint32 `json:"cell_size_milli"`
	NavigationDataHex string `json:"navigation_data"`
	SourceScenePath   string `json:"source_scene_path"`
	Reason            string `json:"reason"`
	AdminUserID       uint64 `json:"-"`
}

// CreateSceneNavigationDraftInput 是领域服务校验和解码后交给数据库仓储的草稿数据。
type CreateSceneNavigationDraftInput struct {
	SceneID           uint32
	OriginX           int32
	OriginY           int32
	GridWidth         uint32
	GridHeight        uint32
	CellSizeMilli     uint32
	NavigationData    []byte
	DataHash          string
	WalkableCellCount uint32
	SourceScenePath   string
	Reason            string
	AdminUserID       uint64
}

// AdminPublishSceneNavigationInput 是发布草稿时使用的管理员审计输入。
type AdminPublishSceneNavigationInput struct {
	Reason      string `json:"reason"`
	AdminUserID uint64 `json:"-"`
}

// AdminRollbackSceneNavigationInput 是复制历史版本并立即发布时使用的管理员审计输入。
type AdminRollbackSceneNavigationInput struct {
	SourceVersion uint32 `json:"source_version"`
	Reason        string `json:"reason"`
	AdminUserID   uint64 `json:"-"`
}

// RefreshSceneNavigationCache 从数据库加载全部已发布位图，并原子替换运行时只读快照。
func (s *Service) RefreshSceneNavigationCache(ctx context.Context) error {
	if s == nil || s.sceneNavigationRepo == nil {
		return ErrSceneNavigationUnavailable
	}
	navigations, err := s.sceneNavigationRepo.ListPublishedSceneNavigations(ctx)
	if err != nil {
		return err
	}
	next := make(map[uint32]SceneNavigation, len(navigations))
	for _, navigation := range navigations {
		if !isValidSceneNavigation(navigation) || navigation.Status != SceneNavigationStatusPublished {
			return ErrSceneNavigationInvalid
		}
		if _, duplicated := next[navigation.SceneID]; duplicated {
			return ErrSceneNavigationInvalid
		}
		navigation.NavigationData = append([]byte(nil), navigation.NavigationData...)
		next[navigation.SceneID] = navigation
	}
	s.sceneNavigationMu.Lock()
	s.sceneNavigations = next
	s.sceneNavigationMu.Unlock()
	return nil
}

// SceneNavigationSnapshot 返回指定场景当前发布位图的深拷贝，调用方不能修改内部缓存字节。
func (s *Service) SceneNavigationSnapshot(sceneID uint32) (SceneNavigation, error) {
	if s == nil || sceneID == 0 {
		return SceneNavigation{}, ErrSceneNavigationUnavailable
	}
	s.sceneNavigationMu.RLock()
	navigation, ok := s.sceneNavigations[sceneID]
	s.sceneNavigationMu.RUnlock()
	if !ok {
		return SceneNavigation{}, ErrSceneNavigationUnavailable
	}
	navigation.NavigationData = append([]byte(nil), navigation.NavigationData...)
	return navigation, nil
}

// GetAdminSceneNavigations 返回指定场景的全部导航版本，供后台查看发布历史。
func (s *Service) GetAdminSceneNavigations(ctx context.Context, sceneID uint32) ([]SceneNavigation, error) {
	if s == nil || s.sceneNavigationRepo == nil || sceneID == 0 {
		return nil, ErrSceneNavigationInvalid
	}
	return s.sceneNavigationRepo.ListAdminSceneNavigations(ctx, sceneID)
}

// CreateAdminSceneNavigationDraft 校验并解码 Godot 导出位图，数据库自行分配下一版本号。
func (s *Service) CreateAdminSceneNavigationDraft(ctx context.Context, input AdminCreateSceneNavigationInput) (SceneNavigation, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceScenePath = strings.TrimSpace(input.SourceScenePath)
	input.NavigationDataHex = strings.TrimSpace(strings.ToLower(input.NavigationDataHex))
	if s == nil || s.sceneNavigationRepo == nil || input.AdminUserID == 0 || input.SceneID == 0 || input.Reason == "" || len([]rune(input.Reason)) > 500 || len([]rune(input.SourceScenePath)) > 512 {
		return SceneNavigation{}, ErrSceneNavigationInvalid
	}
	navigationData, err := hex.DecodeString(input.NavigationDataHex)
	if err != nil {
		return SceneNavigation{}, ErrSceneNavigationInvalid
	}
	draft := SceneNavigation{
		SceneID: input.SceneID, OriginX: input.OriginX, OriginY: input.OriginY,
		GridWidth: input.GridWidth, GridHeight: input.GridHeight, CellSizeMilli: input.CellSizeMilli,
		NavigationData: navigationData,
	}
	if !isValidSceneNavigation(draft) {
		return SceneNavigation{}, ErrSceneNavigationInvalid
	}
	digest := sha256.Sum256(navigationData)
	return s.sceneNavigationRepo.CreateSceneNavigationDraft(ctx, CreateSceneNavigationDraftInput{
		SceneID: input.SceneID, OriginX: input.OriginX, OriginY: input.OriginY,
		GridWidth: input.GridWidth, GridHeight: input.GridHeight, CellSizeMilli: input.CellSizeMilli,
		NavigationData: append([]byte(nil), navigationData...), DataHash: hex.EncodeToString(digest[:]),
		WalkableCellCount: countWalkableCells(navigationData, uint64(input.GridWidth)*uint64(input.GridHeight)),
		SourceScenePath:   input.SourceScenePath, Reason: input.Reason, AdminUserID: input.AdminUserID,
	})
}

// PublishAdminSceneNavigation 发布草稿，并在事务完成后立即刷新当前场景运行时缓存。
func (s *Service) PublishAdminSceneNavigation(ctx context.Context, navigationID uint64, input AdminPublishSceneNavigationInput) (SceneNavigation, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if s == nil || s.sceneNavigationRepo == nil || navigationID == 0 || input.AdminUserID == 0 || input.Reason == "" || len([]rune(input.Reason)) > 500 {
		return SceneNavigation{}, ErrSceneNavigationInvalid
	}
	published, err := s.sceneNavigationRepo.PublishSceneNavigation(ctx, navigationID, input)
	if err != nil {
		return SceneNavigation{}, err
	}
	if !isValidSceneNavigation(published) || published.Status != SceneNavigationStatusPublished {
		return SceneNavigation{}, ErrSceneNavigationInvalid
	}
	s.replaceSceneNavigationSnapshot(published)
	return published, nil
}

// RollbackAdminSceneNavigation 复制指定历史版本为新版本并发布，避免篡改既有审计记录。
func (s *Service) RollbackAdminSceneNavigation(ctx context.Context, sceneID uint32, input AdminRollbackSceneNavigationInput) (SceneNavigation, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if s == nil || s.sceneNavigationRepo == nil || sceneID == 0 || input.SourceVersion == 0 || input.AdminUserID == 0 || input.Reason == "" || len([]rune(input.Reason)) > 500 {
		return SceneNavigation{}, ErrSceneNavigationInvalid
	}
	published, err := s.sceneNavigationRepo.RollbackSceneNavigation(ctx, sceneID, input)
	if err != nil {
		return SceneNavigation{}, err
	}
	if !isValidSceneNavigation(published) || published.Status != SceneNavigationStatusPublished {
		return SceneNavigation{}, ErrSceneNavigationInvalid
	}
	s.replaceSceneNavigationSnapshot(published)
	return published, nil
}

// replaceSceneNavigationSnapshot 使用复制写方式更新单场景位图，移动热路径只读取不可变 map。
func (s *Service) replaceSceneNavigationSnapshot(navigation SceneNavigation) {
	navigation.NavigationData = append([]byte(nil), navigation.NavigationData...)
	s.sceneNavigationMu.Lock()
	next := make(map[uint32]SceneNavigation, len(s.sceneNavigations)+1)
	for sceneID, cached := range s.sceneNavigations {
		next[sceneID] = cached
	}
	next[navigation.SceneID] = navigation
	s.sceneNavigations = next
	s.sceneNavigationMu.Unlock()
}

// clampMovementToNavigation 沿四方向逐格检查位图，阻止单个移动包跨过中间静态墙体。
func (s *Service) clampMovementToNavigation(sceneID uint32, from Vec2i, candidate Vec2i, direction Vec2i) (Vec2i, error) {
	navigation, err := s.SceneNavigationSnapshot(sceneID)
	if err != nil {
		return Vec2i{}, err
	}
	if !navigation.isWalkablePosition(from) {
		return Vec2i{}, ErrSceneNavigationBlocked
	}
	result := candidate
	if direction.X > 0 {
		result.X = navigation.clampPositiveAxis(from.X, candidate.X, from.Y, true)
	} else if direction.X < 0 {
		result.X = navigation.clampNegativeAxis(from.X, candidate.X, from.Y, true)
	} else if direction.Y > 0 {
		result.Y = navigation.clampPositiveAxis(from.Y, candidate.Y, from.X, false)
	} else if direction.Y < 0 {
		result.Y = navigation.clampNegativeAxis(from.Y, candidate.Y, from.X, false)
	}
	return result, nil
}

// clampPositiveAxis 返回正方向上进入首个阻挡格之前的最后一个定点坐标。
func (navigation SceneNavigation) clampPositiveAxis(from int32, target int32, fixed int32, horizontal bool) int32 {
	currentIndex := navigation.axisIndex(from, horizontal)
	targetIndex := navigation.axisIndex(target, horizontal)
	for index := currentIndex + 1; index <= targetIndex; index++ {
		if !navigation.isWalkableAxisCell(index, fixed, horizontal) {
			return navigation.axisCellStart(index, horizontal) - 1
		}
	}
	if !navigation.isWalkablePositionOnAxis(target, fixed, horizontal) {
		return navigation.axisCellStart(targetIndex, horizontal) - 1
	}
	return target
}

// clampNegativeAxis 返回负方向上进入首个阻挡格之前的单元边界坐标。
func (navigation SceneNavigation) clampNegativeAxis(from int32, target int32, fixed int32, horizontal bool) int32 {
	currentIndex := navigation.axisIndex(from, horizontal)
	targetIndex := navigation.axisIndex(target, horizontal)
	for index := currentIndex - 1; index >= targetIndex; index-- {
		if !navigation.isWalkableAxisCell(index, fixed, horizontal) {
			return navigation.axisCellStart(index, horizontal) + int32(navigation.CellSizeMilli)
		}
	}
	if !navigation.isWalkablePositionOnAxis(target, fixed, horizontal) {
		return navigation.axisCellStart(targetIndex, horizontal) + int32(navigation.CellSizeMilli)
	}
	return target
}

// isWalkablePosition 判断定点坐标所在位图单元是否允许人物中心站立。
func (navigation SceneNavigation) isWalkablePosition(position Vec2i) bool {
	cellX := navigation.axisIndex(position.X, true)
	cellY := navigation.axisIndex(position.Y, false)
	return navigation.isWalkableCell(cellX, cellY)
}

// isWalkablePositionOnAxis 组合移动轴和固定轴坐标，复用统一格子判定。
func (navigation SceneNavigation) isWalkablePositionOnAxis(axis int32, fixed int32, horizontal bool) bool {
	if horizontal {
		return navigation.isWalkablePosition(Vec2i{X: axis, Y: fixed})
	}
	return navigation.isWalkablePosition(Vec2i{X: fixed, Y: axis})
}

// isWalkableAxisCell 把移动轴索引与固定轴坐标转换为二维位图索引。
func (navigation SceneNavigation) isWalkableAxisCell(axisIndex int32, fixed int32, horizontal bool) bool {
	if horizontal {
		return navigation.isWalkableCell(axisIndex, navigation.axisIndex(fixed, false))
	}
	return navigation.isWalkableCell(navigation.axisIndex(fixed, true), axisIndex)
}

// axisIndex 把定点坐标转换为位图索引；负数使用向下取整，不能使用 Go 的向零截断。
func (navigation SceneNavigation) axisIndex(value int32, horizontal bool) int32 {
	origin := navigation.OriginY
	if horizontal {
		origin = navigation.OriginX
	}
	delta := int64(value) - int64(origin)
	cellSize := int64(navigation.CellSizeMilli)
	if delta >= 0 {
		return int32(delta / cellSize)
	}
	return int32((delta - cellSize + 1) / cellSize)
}

// axisCellStart 返回指定位图索引对应单元格的起始定点坐标。
func (navigation SceneNavigation) axisCellStart(index int32, horizontal bool) int32 {
	origin := navigation.OriginY
	if horizontal {
		origin = navigation.OriginX
	}
	return origin + index*int32(navigation.CellSizeMilli)
}

// isWalkableCell 根据行优先、高位优先规则读取单个通行 bit，越界一律视为阻挡。
func (navigation SceneNavigation) isWalkableCell(cellX int32, cellY int32) bool {
	if cellX < 0 || cellY < 0 || uint32(cellX) >= navigation.GridWidth || uint32(cellY) >= navigation.GridHeight {
		return false
	}
	bitIndex := uint64(uint32(cellY))*uint64(navigation.GridWidth) + uint64(uint32(cellX))
	byteIndex := bitIndex / 8
	bitOffset := uint(7 - bitIndex%8)
	return navigation.NavigationData[byteIndex]&(byte(1)<<bitOffset) != 0
}

// isValidSceneNavigation 校验尺寸、坐标和字节长度，避免损坏数据进入运行时热路径。
func isValidSceneNavigation(navigation SceneNavigation) bool {
	if navigation.SceneID == 0 || navigation.GridWidth == 0 || navigation.GridHeight == 0 || navigation.GridWidth > MaxSceneNavigationDimension || navigation.GridHeight > MaxSceneNavigationDimension || navigation.CellSizeMilli == 0 || navigation.CellSizeMilli > MaxSceneNavigationCellSizeMilli {
		return false
	}
	if navigation.OriginX < -MaxSceneBoundaryCoordinateAbs || navigation.OriginY < -MaxSceneBoundaryCoordinateAbs || navigation.OriginX > MaxSceneBoundaryCoordinateAbs || navigation.OriginY > MaxSceneBoundaryCoordinateAbs {
		return false
	}
	cellCount := uint64(navigation.GridWidth) * uint64(navigation.GridHeight)
	if cellCount > MaxSceneNavigationCellCount {
		return false
	}
	expectedLength := (cellCount + 7) / 8
	return uint64(len(navigation.NavigationData)) == expectedLength
}

// countWalkableCells 统计有效格位中的 1 bit；最后一个字节的填充位不会计入数量。
func countWalkableCells(data []byte, cellCount uint64) uint32 {
	var count uint32
	for bitIndex := uint64(0); bitIndex < cellCount; bitIndex++ {
		byteIndex := bitIndex / 8
		bitOffset := uint(7 - bitIndex%8)
		if data[byteIndex]&(byte(1)<<bitOffset) != 0 {
			count++
		}
	}
	return count
}
