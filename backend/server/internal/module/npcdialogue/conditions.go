package npcdialogue

import (
	"context"
	"encoding/json"
	"strings"

	"pocket-pet-remake/server/internal/module/quest"
)

// QuestSummaryReader 供剧情条件校验读取玩家当前任务摘要。
type QuestSummaryReader interface {
	ListSummaries(ctx context.Context, playerID uint64) ([]quest.Summary, error)
}

// NodeConditions 描述节点、选项或菜单项进入前需要满足的任务条件。
type NodeConditions struct {
	QuestID            uint64
	QuestState         string
	ObjectiveID        uint64
	ObjectiveCompleted *bool
}

// ParseNodeConditions 解析 conditions_json，支持 quest_id / quest_state / objective_id / objective_completed。
func ParseNodeConditions(raw json.RawMessage) NodeConditions {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "{}" || text == "null" {
		return NodeConditions{}
	}
	var payload struct {
		QuestID            uint64 `json:"quest_id"`
		QuestState         string `json:"quest_state"`
		ObjectiveID        uint64 `json:"objective_id"`
		ObjectiveCompleted *bool  `json:"objective_completed"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return NodeConditions{}
	}
	return NodeConditions{
		QuestID:            payload.QuestID,
		QuestState:         strings.TrimSpace(payload.QuestState),
		ObjectiveID:        payload.ObjectiveID,
		ObjectiveCompleted: payload.ObjectiveCompleted,
	}
}

// MatchNodeConditions 判断玩家是否满足配置的条件；未配置条件时默认通过。
func MatchNodeConditions(ctx context.Context, reader QuestSummaryReader, playerID uint64, raw json.RawMessage) (bool, error) {
	conditions := ParseNodeConditions(raw)
	if conditions.QuestID == 0 {
		return true, nil
	}
	if reader == nil || playerID == 0 {
		return false, nil
	}
	summaries, err := reader.ListSummaries(ctx, playerID)
	if err != nil {
		return false, err
	}
	for _, summary := range summaries {
		if summary.QuestID != conditions.QuestID {
			continue
		}
		if conditions.QuestState != "" && !strings.EqualFold(summary.State, conditions.QuestState) {
			return false, nil
		}
		if conditions.ObjectiveID == 0 {
			return true, nil
		}
		for _, objective := range summary.Objectives {
			if objective.ObjectiveID != conditions.ObjectiveID {
				continue
			}
			if conditions.ObjectiveCompleted == nil {
				return !objective.Completed, nil
			}
			return objective.Completed == *conditions.ObjectiveCompleted, nil
		}
		return false, nil
	}
	return false, nil
}
