package storyprogress

import (
	"context"
	"strings"
)

// Service 负责玩家个人剧情进度判断。
// 世界模块进入地图时调用 EvaluateSceneEntry，客户端播完动画后调用 CompleteSceneTrigger。
type Service struct {
	repo Repository
}

// NewService 创建剧情进度服务。
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// EvaluateSceneEntry 判断玩家进入指定场景后是否有待播放的一次性剧情。
func (s *Service) EvaluateSceneEntry(ctx context.Context, playerID uint64, sceneID uint32) (*SceneTrigger, error) {
	if s == nil || s.repo == nil || playerID == 0 || sceneID == 0 {
		return nil, nil
	}
	return s.repo.FindPendingSceneTrigger(ctx, playerID, sceneID)
}

// CompleteSceneTrigger 标记玩家已完成指定场景剧情，并返回需要执行的服务端副作用配置。
func (s *Service) CompleteSceneTrigger(ctx context.Context, playerID uint64, triggerCode string) (*SceneTrigger, error) {
	if s == nil || s.repo == nil || playerID == 0 || strings.TrimSpace(triggerCode) == "" {
		return nil, nil
	}
	return s.repo.CompleteSceneTrigger(ctx, playerID, strings.TrimSpace(triggerCode))
}
