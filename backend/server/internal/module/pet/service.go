package pet

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListLineup(ctx context.Context, playerID uint64) ([]LineupPet, error) {
	lineup, err := s.repo.ListLineupByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if lineup == nil {
		return []LineupPet{}, nil
	}
	return lineup, nil
}
