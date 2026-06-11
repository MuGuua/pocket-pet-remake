package player

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetProfile(ctx context.Context, playerID uint64) (*Profile, error) {
	profile, err := s.repo.FindByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrPlayerNotFound
	}
	return profile, nil
}

// ListAdminPlayers 返回后台玩家列表。
// 这里统一做分页兜底，避免多个后台入口各自复制 page/page_size 默认值。
func (s *Service) ListAdminPlayers(ctx context.Context, query AdminListQuery) (*AdminPlayerList, error) {
	result, err := s.repo.ListForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminPlayerList{Items: []AdminPlayerSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

// GetAdminPlayerDetail 返回后台详情面板需要的人物全量快照。
func (s *Service) GetAdminPlayerDetail(ctx context.Context, playerID uint64) (*AdminPlayerDetail, error) {
	detail, err := s.repo.FindAdminDetailByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, ErrPlayerNotFound
	}
	return detail, nil
}

// CreateAdminPlayer 创建完整玩家账号与人物档案，确保后台新增后能直接走正式登录链路。
func (s *Service) CreateAdminPlayer(ctx context.Context, input AdminCreatePlayerInput) (*AdminPlayerDetail, error) {
	input = input.Normalize()
	if input.AccountName == "" || input.Password == "" || input.Name == "" {
		return nil, ErrInvalidAdminInput
	}
	return s.repo.CreateForAdmin(ctx, input)
}

// UpdateAdminPlayer 会覆写后台允许调整的玩家持久化字段。
func (s *Service) UpdateAdminPlayer(ctx context.Context, playerID uint64, input AdminUpdatePlayerInput) (*AdminPlayerDetail, error) {
	input = input.Normalize()
	if input.Name == "" {
		return nil, ErrInvalidAdminInput
	}
	detail, err := s.repo.UpdateForAdmin(ctx, playerID, input)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, ErrPlayerNotFound
	}
	return detail, nil
}

// DeleteAdminPlayer 使用软删除禁用 account/player，避免直接物理删除破坏联机关联数据。
func (s *Service) DeleteAdminPlayer(ctx context.Context, playerID uint64) error {
	return s.repo.DeleteForAdmin(ctx, playerID)
}

func (s *Service) UpdatePosition(ctx context.Context, playerID uint64, sceneID uint32, posX, posY int32) error {
	return s.repo.UpdatePosition(ctx, playerID, sceneID, posX, posY)
}

func (s *Service) AddGoldAndExp(ctx context.Context, playerID uint64, gold uint32, exp uint64) (*Profile, error) {
	return s.repo.AddGoldAndExp(ctx, playerID, gold, exp)
}
