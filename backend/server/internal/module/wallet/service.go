package wallet

import "context"

// Service 负责后台钱包查询与调整。
// 货币增减、非负校验和空结果语义都在这里统一收口。
type Service struct {
	repo Repository
}

// NewService 构造后台钱包服务。
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListAdminWallets 返回后台钱包分页列表。
func (s *Service) ListAdminWallets(ctx context.Context, query AdminListQuery) (*AdminWalletList, error) {
	result, err := s.repo.ListForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminWalletList{Items: []AdminWalletSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

// GetAdminWalletDetail 返回指定玩家的钱包详情。
func (s *Service) GetAdminWalletDetail(ctx context.Context, playerID uint64) (*AdminWalletDetail, error) {
	result, err := s.repo.FindAdminDetailByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrWalletNotFound
	}
	return result, nil
}

// AdjustAdminWallet 对指定玩家的钱包做增量调整。
func (s *Service) AdjustAdminWallet(ctx context.Context, playerID uint64, input AdminAdjustInput) (*AdminWalletDetail, error) {
	input = input.Normalize()
	if playerID == 0 || input.ChangeTotalCopper == 0 || input.Reason == "" {
		return nil, ErrInvalidAdminWalletInput
	}
	result, err := s.repo.AdjustForAdmin(ctx, playerID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrWalletNotFound
	}
	return result, nil
}

// AdjustRuntimeWallet 供正式玩法链路做服务端权威的钱包增减。
// 只有带完整归因信息的变更才允许落库，这样 battle、quest、shop 都能共用同一条流水链路。
func (s *Service) AdjustRuntimeWallet(ctx context.Context, playerID uint64, input RuntimeAdjustInput) (*RuntimeAdjustResult, error) {
	input = input.Normalize()
	if playerID == 0 || input.ChangeTotalCopper == 0 || input.ReasonType == "" || input.OperatorType == "" {
		return nil, ErrInvalidRuntimeAdjustInput
	}
	result, err := s.repo.AdjustRuntime(ctx, playerID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrWalletNotFound
	}
	return result, nil
}

// GetRuntimeWallet 返回玩家端运行时直接消费的钱包快照。
// 钱包不存在时统一返回领域层错误，方便 WebSocket handler 输出稳定错误码。
func (s *Service) GetRuntimeWallet(ctx context.Context, playerID uint64) (*Snapshot, error) {
	if playerID == 0 {
		return nil, ErrWalletNotFound
	}
	result, err := s.repo.GetRuntimeSnapshot(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrWalletNotFound
	}
	return result, nil
}
