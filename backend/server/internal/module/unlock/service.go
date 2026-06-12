package unlock

import "context"

// Service 负责把正式玩法里的功能解锁请求收口到统一入口。
// 这样 quest、activity、mail 等来源都可以复用同一套持久化语义。
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GrantRuntimeFeature 在服务端权威链路中授予一个功能解锁标记。
func (s *Service) GrantRuntimeFeature(ctx context.Context, playerID uint64, featureID uint64, reasonType string, reasonRefID uint64, operatorType string, operatorID uint64) (*RuntimeGrantResult, error) {
	if playerID == 0 || featureID == 0 || reasonType == "" || operatorType == "" {
		return nil, ErrInvalidRuntimeUnlockInput
	}
	return s.repo.GrantRuntimeFeature(ctx, playerID, featureID, reasonType, reasonRefID, operatorType, operatorID)
}
