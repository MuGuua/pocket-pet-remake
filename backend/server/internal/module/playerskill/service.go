package playerskill

import (
	"context"
	"fmt"
)

// Service 负责玩家武器技能学习进度的读取与战斗结算写回。
type Service struct {
	repo Repository
}

// NewService 构造玩家技能进度服务。
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListProgress 返回玩家全部武器技能学习进度。
func (s *Service) ListProgress(ctx context.Context, playerID uint64) ([]Progress, error) {
	if s == nil || s.repo == nil || playerID == 0 {
		return []Progress{}, nil
	}
	items, err := s.repo.ListByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []Progress{}, nil
	}
	return items, nil
}

// ApplyBattleUpdates 在战斗结束后批量落库技能经验与学会状态。
func (s *Service) ApplyBattleUpdates(ctx context.Context, playerID uint64, updates []BattleUseUpdate) error {
	if s == nil || s.repo == nil || playerID == 0 || len(updates) == 0 {
		return nil
	}
	for _, update := range updates {
		if update.SkillID == 0 || update.ExpGained == 0 {
			return fmt.Errorf("%w: skill_id=%d exp_gained=%d", ErrInvalidSkillProgressInput, update.SkillID, update.ExpGained)
		}
	}
	return s.repo.UpsertBattleUpdates(ctx, playerID, updates)
}

// ProgressMap 将进度列表转为 skill_id 索引 map，供战斗开战前组装输入。
func ProgressMap(items []Progress) map[uint32]Progress {
	result := make(map[uint32]Progress, len(items))
	for _, item := range items {
		if item.SkillID == 0 {
			continue
		}
		result[item.SkillID] = item
	}
	return result
}
