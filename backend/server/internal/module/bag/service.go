package bag

import (
	"context"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"time"
)

const runtimeContainerSnapshotMaxAttempts = 2

// Service 负责把后台背包管理请求收口到统一领域入口。
// HTTP handler 只解析参数，默认值、空结果与存在性判断都在这里处理。
type Service struct {
	repo Repository
	// useItemMu 与 useItemLast 共同限制同一玩家的主动使用物品频率，防止脚本高频刷用。
	useItemMu   sync.Mutex
	useItemLast map[uint64]time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListAdminItems(ctx context.Context, query AdminListQuery) (*AdminItemList, error) {
	result, err := s.repo.ListForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminItemList{Items: []AdminItemSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

func (s *Service) GetAdminItemDetail(ctx context.Context, recordID uint64) (*AdminItemDetail, error) {
	result, err := s.repo.FindAdminDetailByRecordID(ctx, recordID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrBagItemNotFound
	}
	return result, nil
}

func (s *Service) CreateAdminItem(ctx context.Context, input AdminCreateItemInput) (*AdminItemDetail, error) {
	input = input.Normalize()
	if input.PlayerID == 0 || input.ItemID == 0 || input.Quantity == 0 || input.ContainerType == "" {
		return nil, ErrInvalidAdminBagInput
	}
	return s.repo.CreateForAdmin(ctx, input)
}

func (s *Service) UpdateAdminItem(ctx context.Context, recordID uint64, input AdminUpdateItemInput) (*AdminItemDetail, error) {
	input = input.Normalize()
	if input.PlayerID == 0 || input.ItemID == 0 || input.SlotIndex == 0 || input.Quantity == 0 || input.ContainerType == "" {
		return nil, ErrInvalidAdminBagInput
	}
	result, err := s.repo.UpdateForAdmin(ctx, recordID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrBagItemNotFound
	}
	return result, nil
}

func (s *Service) DeleteAdminItem(ctx context.Context, recordID uint64) error {
	return s.repo.DeleteForAdmin(ctx, recordID)
}

// ListRuntimeContainer 返回玩家指定容器的完整权威快照。
// 运行时统一在这里做容器类型标准化，避免传输层重复维护默认值逻辑。
func (s *Service) ListRuntimeContainer(ctx context.Context, playerID uint64, containerType string) (*RuntimeContainerSnapshot, error) {
	if playerID == 0 {
		return nil, ErrContainerNotFound
	}
	normalizedContainerType, err := NormalizeRuntimeContainerType(containerType)
	if err != nil {
		return nil, err
	}
	result, err := s.listRuntimeContainerWithRetry(ctx, playerID, normalizedContainerType)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrContainerNotFound
	}
	result.ContainerType = normalizedContainerType
	if result.Items == nil {
		result.Items = []RuntimeItemSnapshot{}
	}
	result.UsedSlots = uint32(len(result.Items))
	return result, nil
}

// listRuntimeContainerWithRetry 只为背包快照的只读查询做一次瞬断重试。
// 移动端客户端打开背包、战斗结算后刷新背包等流程都依赖这份服务端权威快照；
// 远端 PostgreSQL 偶发关闭空闲连接时，第一次查询可能返回 unexpected EOF 或网络超时，
// 这里重试一次可以让 database/sql 丢弃坏连接并重新取一条可用连接，避免把临时断流直接暴露给玩家。
func (s *Service) listRuntimeContainerWithRetry(ctx context.Context, playerID uint64, containerType string) (*RuntimeContainerSnapshot, error) {
	var lastErr error
	for attempt := 1; attempt <= runtimeContainerSnapshotMaxAttempts; attempt++ {
		result, err := s.repo.ListRuntimeContainer(ctx, playerID, containerType)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if attempt == runtimeContainerSnapshotMaxAttempts || !isRetryableRuntimeContainerSnapshotError(err) {
			return nil, err
		}
	}
	return nil, lastErr
}

// isRetryableRuntimeContainerSnapshotError 判断一次背包快照失败是否更像数据库连接瞬断。
// 业务校验类错误不重试，避免隐藏真实的数据问题；只对驱动坏连接、EOF、读超时等网络层错误重试。
func isRetryableRuntimeContainerSnapshotError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unexpected eof") ||
		strings.Contains(message, "operation timed out") ||
		strings.Contains(message, "connection reset by peer") ||
		strings.Contains(message, "broken pipe")
}

// MoveRuntimeItemBetweenContainers 执行背包与仓库之间的权威移动逻辑。
// 当前仅允许 bag 与 warehouse 双向移动，避免把装备栏或临时奖励箱误当成普通容器处理。
func (s *Service) MoveRuntimeItemBetweenContainers(ctx context.Context, playerID uint64, fromContainerType string, toContainerType string, fromSlotIndex uint32, quantity uint64) (*RuntimeTransferResult, error) {
	if playerID == 0 || fromSlotIndex == 0 || quantity == 0 {
		return nil, ErrInvalidTransferQuantity
	}
	normalizedFromContainerType, err := NormalizeRuntimeContainerType(fromContainerType)
	if err != nil {
		return nil, err
	}
	normalizedToContainerType, err := NormalizeRuntimeContainerType(toContainerType)
	if err != nil {
		return nil, err
	}
	if normalizedFromContainerType == normalizedToContainerType {
		return nil, ErrInvalidContainerType
	}
	return s.repo.TransferRuntimeItem(ctx, playerID, normalizedFromContainerType, normalizedToContainerType, fromSlotIndex, quantity)
}

// SortRuntimeContainer 按服务端规则整理指定容器内的格子顺序。
func (s *Service) SortRuntimeContainer(ctx context.Context, playerID uint64, containerType string) (*RuntimeSortResult, error) {
	if playerID == 0 {
		return nil, ErrContainerNotFound
	}
	normalizedContainerType, err := NormalizeRuntimeContainerType(containerType)
	if err != nil {
		return nil, err
	}
	return s.repo.SortRuntimeContainer(ctx, playerID, normalizedContainerType)
}

// MoveRuntimeItem 执行同容器换位或拆分移动。
func (s *Service) MoveRuntimeItem(ctx context.Context, playerID uint64, containerType string, fromSlotIndex uint32, toSlotIndex uint32, quantity uint64) (*RuntimeMoveResult, error) {
	if playerID == 0 || fromSlotIndex == 0 || toSlotIndex == 0 || quantity == 0 {
		return nil, ErrInvalidContainerMove
	}
	normalizedContainerType, err := NormalizeRuntimeContainerType(containerType)
	if err != nil {
		return nil, err
	}
	return s.repo.MoveRuntimeItem(ctx, playerID, normalizedContainerType, fromSlotIndex, toSlotIndex, quantity)
}

// GrantRuntimeItem 让系统把奖励道具直接发放到玩家容器。
// 当前默认只允许发进随身背包，避免把正式奖励误写到未打开的其他容器。
func (s *Service) GrantRuntimeItem(ctx context.Context, playerID uint64, containerType string, itemID uint64, quantity uint64, reasonType string, reasonRefID uint64, operatorType string, operatorID uint64) (*RuntimeGrantResult, error) {
	if playerID == 0 || itemID == 0 || quantity == 0 {
		return nil, ErrInvalidTransferQuantity
	}
	normalizedContainerType, err := NormalizeRuntimeContainerType(containerType)
	if err != nil {
		return nil, err
	}
	return s.repo.GrantRuntimeItem(ctx, playerID, normalizedContainerType, itemID, quantity, reasonType, reasonRefID, operatorType, operatorID)
}

// UseRuntimeItem 执行玩家主动使用背包物品的正式链路。
// 目标宠物、目标玩家等动态目标也统一从这里往下传，确保真正生效的对象由服务端权威判定。
func (s *Service) UseRuntimeItem(ctx context.Context, playerID uint64, containerType string, slotIndex uint32, quantity uint64, targetPetUID uint64, targetPlayerID uint64, targetItemUID string) (*RuntimeUseResult, error) {
	if playerID == 0 || slotIndex == 0 || quantity == 0 {
		return nil, ErrInvalidTransferQuantity
	}
	normalizedContainerType, err := NormalizeRuntimeContainerType(containerType)
	if err != nil {
		return nil, err
	}
	if err := s.acquireRuntimeUseItemSlot(playerID); err != nil {
		return nil, err
	}
	return s.repo.UseRuntimeItem(ctx, playerID, normalizedContainerType, slotIndex, quantity, targetPetUID, targetPlayerID, strings.TrimSpace(targetItemUID))
}

// DropRuntimeItem 执行玩家主动丢弃背包/仓库格子物品的正式链路。
// itemUID 非空时按实例唯一标识定位格子；否则按 slotIndex 定位，可堆叠物支持部分 quantity 丢弃。
func (s *Service) DropRuntimeItem(ctx context.Context, playerID uint64, containerType string, slotIndex uint32, itemUID string, quantity uint64) (*RuntimeDropResult, error) {
	itemUID = strings.TrimSpace(itemUID)
	if playerID == 0 || quantity == 0 {
		return nil, ErrInvalidTransferQuantity
	}
	if itemUID == "" && slotIndex == 0 {
		return nil, ErrInvalidTransferQuantity
	}
	normalizedContainerType, err := NormalizeRuntimeContainerType(containerType)
	if err != nil {
		return nil, err
	}
	return s.repo.DropRuntimeItem(ctx, playerID, normalizedContainerType, slotIndex, itemUID, quantity)
}

// acquireRuntimeUseItemSlot 占用一次主动使用物品的时间片；同一玩家在冷却期内再次使用会返回 ErrUseItemTooFast。
func (s *Service) acquireRuntimeUseItemSlot(playerID uint64) error {
	if s == nil || playerID == 0 {
		return ErrInvalidTransferQuantity
	}
	now := time.Now()
	s.useItemMu.Lock()
	defer s.useItemMu.Unlock()
	if s.useItemLast == nil {
		s.useItemLast = make(map[uint64]time.Time)
	}
	if lastUsed, exists := s.useItemLast[playerID]; exists && now.Sub(lastUsed) < RuntimeUseItemCooldown {
		return ErrUseItemTooFast
	}
	s.useItemLast[playerID] = now
	return nil
}

// ConsumeRuntimeItemStack 仅扣减背包格子数量，不触发道具使用效果；战斗捕捉等场景复用。
func (s *Service) ConsumeRuntimeItemStack(ctx context.Context, playerID uint64, containerType string, slotIndex uint32, quantity uint64, reasonType string, reasonRefID uint64) (*RuntimeContainerSnapshot, error) {
	if playerID == 0 || slotIndex == 0 || quantity == 0 {
		return nil, ErrInvalidTransferQuantity
	}
	normalizedContainerType, err := NormalizeRuntimeContainerType(containerType)
	if err != nil {
		return nil, err
	}
	return s.repo.ConsumeRuntimeItemStack(ctx, playerID, normalizedContainerType, slotIndex, quantity, reasonType, reasonRefID)
}

// PlayerHasEverOwnedItem 判断玩家是否已获得过指定道具。
// 唯一战斗掉落会结合历史获得记录与当前背包/仓库持有量做去重。
func (s *Service) PlayerHasEverOwnedItem(ctx context.Context, playerID uint64, itemID uint64) (bool, error) {
	if s == nil || s.repo == nil || playerID == 0 || itemID == 0 {
		return false, nil
	}
	return s.repo.PlayerHasEverOwnedItem(ctx, playerID, itemID)
}

// RecordUniqueItemObtained 记录玩家首次获得唯一道具。
func (s *Service) RecordUniqueItemObtained(ctx context.Context, playerID uint64, itemID uint64, reasonType string, reasonRefID uint64) error {
	if s == nil || s.repo == nil || playerID == 0 || itemID == 0 {
		return nil
	}
	return s.repo.RecordUniqueItemObtained(ctx, playerID, itemID, reasonType, reasonRefID)
}
