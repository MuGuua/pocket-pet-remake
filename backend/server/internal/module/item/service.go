package item

import "context"

// Service 负责把后台物品模板请求收口到统一领域入口。
// HTTP handler 只负责解析参数，字段校验、默认值和空结果语义都在这里统一处理。
type Service struct {
	repo Repository
}

// NewService 构造后台物品模板服务。
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListAdminItems 返回后台物品模板分页结果。
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

// GetAdminItemDetail 返回指定模板的完整详情。
func (s *Service) GetAdminItemDetail(ctx context.Context, itemID uint64) (*AdminItemDetail, error) {
	result, err := s.repo.FindAdminDetailByItemID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrItemDefinitionNotFound
	}
	return result, nil
}

// GetRuntimeItemDetail returns the formal item template used by runtime systems
// such as bag, shop and reward flows.
func (s *Service) GetRuntimeItemDetail(ctx context.Context, itemID uint64) (*AdminItemDetail, error) {
	if itemID == 0 {
		return nil, ErrItemDefinitionNotFound
	}
	result, err := s.repo.FindAdminDetailByItemID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrItemDefinitionNotFound
	}
	return result, nil
}

// CreateAdminItem 创建新的正式物品模板。
func (s *Service) CreateAdminItem(ctx context.Context, input AdminUpsertItemInput) (*AdminItemDetail, error) {
	input = input.Normalize()
	if input.ItemID == 0 || input.ItemCode == "" || input.ItemName == "" || input.ItemType == "" {
		return nil, ErrInvalidAdminItemInput
	}
	if input.EnhanceMaterialConfig != nil {
		if err := input.EnhanceMaterialConfig.Validate(); err != nil {
			return nil, ErrInvalidAdminItemInput
		}
	}
	return s.repo.CreateForAdmin(ctx, input)
}

// UpdateAdminItem 更新既有物品模板。
func (s *Service) UpdateAdminItem(ctx context.Context, itemID uint64, input AdminUpsertItemInput) (*AdminItemDetail, error) {
	input = input.Normalize()
	if itemID == 0 || input.ItemCode == "" || input.ItemName == "" || input.ItemType == "" {
		return nil, ErrInvalidAdminItemInput
	}
	if input.EnhanceMaterialConfig != nil {
		if err := input.EnhanceMaterialConfig.Validate(); err != nil {
			return nil, ErrInvalidAdminItemInput
		}
	}
	result, err := s.repo.UpdateForAdmin(ctx, itemID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrItemDefinitionNotFound
	}
	return result, nil
}

// DeleteAdminItem 删除指定模板。
func (s *Service) DeleteAdminItem(ctx context.Context, itemID uint64) error {
	return s.repo.DeleteForAdmin(ctx, itemID)
}
