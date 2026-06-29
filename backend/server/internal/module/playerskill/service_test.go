package playerskill

import (
	"context"
	"testing"
)

type stubRepo struct {
	items   []Progress
	updates []BattleUseUpdate
}

func (s *stubRepo) ListByPlayerID(_ context.Context, playerID uint64) ([]Progress, error) {
	result := make([]Progress, 0, len(s.items))
	for _, item := range s.items {
		if item.PlayerID == playerID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *stubRepo) UpsertBattleUpdates(_ context.Context, _ uint64, updates []BattleUseUpdate) error {
	s.updates = append(s.updates, updates...)
	return nil
}

func TestApplyBattleUpdatesRejectsZeroExpGain(t *testing.T) {
	repo := &stubRepo{}
	service := NewService(repo)
	err := service.ApplyBattleUpdates(context.Background(), 10001, []BattleUseUpdate{
		{SkillID: 1201, ExpGained: 0, FinalExp: 1},
	})
	if err == nil {
		t.Fatal("ApplyBattleUpdates() error = nil, want invalid input")
	}
}

func TestApplyBattleUpdatesPersistsUpdates(t *testing.T) {
	repo := &stubRepo{}
	service := NewService(repo)
	updates := []BattleUseUpdate{
		{SkillID: 1201, ExpGained: 1, FinalExp: 3, FinalLevel: 2, LearnExpRequired: 10},
	}
	if err := service.ApplyBattleUpdates(context.Background(), 10001, updates); err != nil {
		t.Fatalf("ApplyBattleUpdates() error = %v", err)
	}
	if len(repo.updates) != 1 {
		t.Fatalf("len(repo.updates) = %d, want 1", len(repo.updates))
	}
}

func TestProgressMapIndexesBySkillID(t *testing.T) {
	items := []Progress{
		{SkillID: 1201, SkillExp: 2, IsLearned: false},
		{SkillID: 1202, SkillExp: 10, IsLearned: true, SkillLevel: 3},
	}
	result := ProgressMap(items)
	if len(result) != 2 || result[1202].SkillLevel != 3 {
		t.Fatalf("ProgressMap() = %#v, want indexed progress", result)
	}
}
