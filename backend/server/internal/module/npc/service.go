package npc

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListMenuEntriesByEntityID(ctx context.Context, entityID uint64) ([]MenuEntry, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	return s.repo.ListMenuEntriesByEntityID(ctx, entityID)
}

func (s *Service) FindActionResult(ctx context.Context, entityID uint64, entryID string) (*ActionResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	return s.repo.FindActionResult(ctx, entityID, entryID)
}

// ListAdminEntities 返回后台 NPC / 世界实体配置列表。
// 后台页会直接消费这里的分页结果，避免自己在前端拼接地图分布数据。
func (s *Service) ListAdminEntities(ctx context.Context, query AdminEntityListQuery) (*AdminEntityList, error) {
	result, err := s.repo.ListEntitiesForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminEntityList{Items: []AdminEntitySummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

func (s *Service) GetAdminEntityDetail(ctx context.Context, entityID uint64) (*AdminEntityDetail, error) {
	result, err := s.repo.FindAdminEntityDetailByEntityID(ctx, entityID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrAdminNPCNotFound
	}
	return result, nil
}

func (s *Service) CreateAdminEntity(ctx context.Context, input AdminCreateEntityInput) (*AdminEntityDetail, error) {
	input = input.Normalize()
	if input.EntityID == 0 || input.EntityCode == "" || input.DisplayName == "" || input.SceneID == 0 {
		return nil, ErrInvalidAdminNPCInput
	}
	return s.repo.CreateEntityForAdmin(ctx, input)
}

func (s *Service) UpdateAdminEntity(ctx context.Context, entityID uint64, input AdminUpdateEntityInput) (*AdminEntityDetail, error) {
	input = input.Normalize()
	if entityID == 0 || input.EntityCode == "" || input.DisplayName == "" || input.SceneID == 0 {
		return nil, ErrInvalidAdminNPCInput
	}
	result, err := s.repo.UpdateEntityForAdmin(ctx, entityID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrAdminNPCNotFound
	}
	return result, nil
}

func (s *Service) DeleteAdminEntity(ctx context.Context, entityID uint64) error {
	return s.repo.DeleteEntityForAdmin(ctx, entityID)
}

func (s *Service) ListAdminMenuEntries(ctx context.Context, query AdminMenuEntryListQuery) (*AdminMenuEntryList, error) {
	result, err := s.repo.ListMenuEntriesForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminMenuEntryList{Items: []AdminMenuEntrySummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

func (s *Service) GetAdminMenuEntryDetail(ctx context.Context, entityID uint64, entryID string) (*AdminMenuEntryDetail, error) {
	result, err := s.repo.FindAdminMenuEntryDetail(ctx, entityID, entryID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrAdminNPCMenuEntryNotFound
	}
	return result, nil
}

func (s *Service) CreateAdminMenuEntry(ctx context.Context, input AdminCreateMenuEntryInput) (*AdminMenuEntryDetail, error) {
	input = input.Normalize()
	if input.EntityID == 0 || input.EntryID == "" || input.EntryType == "" || input.Title == "" {
		return nil, ErrInvalidAdminNPCInput
	}
	return s.repo.CreateMenuEntryForAdmin(ctx, input)
}

func (s *Service) UpdateAdminMenuEntry(ctx context.Context, entityID uint64, entryID string, input AdminUpdateMenuEntryInput) (*AdminMenuEntryDetail, error) {
	input = input.Normalize()
	if entityID == 0 || entryID == "" || input.EntityID == 0 || input.EntryType == "" || input.Title == "" {
		return nil, ErrInvalidAdminNPCInput
	}
	result, err := s.repo.UpdateMenuEntryForAdmin(ctx, entityID, entryID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrAdminNPCMenuEntryNotFound
	}
	return result, nil
}

func (s *Service) DeleteAdminMenuEntry(ctx context.Context, entityID uint64, entryID string) error {
	return s.repo.DeleteMenuEntryForAdmin(ctx, entityID, entryID)
}
