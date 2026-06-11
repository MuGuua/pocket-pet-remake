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
	if input.PlayerID == 0 || input.ItemID == 0 || input.Count == 0 {
		return nil, ErrInvalidAdminBagInput
	}
	return s.repo.CreateForAdmin(ctx, input)
}

func (s *Service) UpdateAdminItem(ctx context.Context, recordID uint64, input AdminUpdateItemInput) (*AdminItemDetail, error) {
	input = input.Normalize()
	if input.PlayerID == 0 || input.ItemID == 0 || input.Count == 0 {
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
