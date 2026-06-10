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
