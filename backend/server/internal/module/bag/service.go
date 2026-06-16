package bag

import "context"

// Service 负责把后台背包管理请求收口到统一领域入口。
// HTTP handler 只解析参数，默认值、空结果与存在性判断都在这里处理。
type Service struct {
	repo Repository
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
	if input.PlayerID == 0 || input.ItemID == 0 || input.SlotIndex == 0 || input.Quantity == 0 || input.ContainerType == "" {
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
	result, err := s.repo.ListRuntimeContainer(ctx, playerID, normalizedContainerType)
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
func (s *Service) UseRuntimeItem(ctx context.Context, playerID uint64, containerType string, slotIndex uint32, quantity uint64, targetPetUID uint64, targetPlayerID uint64) (*RuntimeUseResult, error) {
	if playerID == 0 || slotIndex == 0 || quantity == 0 {
		return nil, ErrInvalidTransferQuantity
	}
	normalizedContainerType, err := NormalizeRuntimeContainerType(containerType)
	if err != nil {
		return nil, err
	}
	return s.repo.UseRuntimeItem(ctx, playerID, normalizedContainerType, slotIndex, quantity, targetPetUID, targetPlayerID)
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
