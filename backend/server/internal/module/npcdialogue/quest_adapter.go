package npcdialogue

import (
	"context"

	"pocket-pet-remake/server/internal/module/quest"
)

// QuestServiceAdapter 把任务服务适配成剧情条件校验所需的摘要读取接口。
type QuestServiceAdapter struct {
	Service *quest.Service
}

// ListSummaries 返回玩家当前任务摘要，供 conditions_json 做 quest_id / quest_state 校验。
func (a *QuestServiceAdapter) ListSummaries(ctx context.Context, playerID uint64) ([]quest.Summary, error) {
	if a == nil || a.Service == nil {
		return nil, nil
	}
	summaries, _, err := a.Service.List(ctx, playerID)
	return summaries, err
}
