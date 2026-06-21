package npcdialogue

import (
	"encoding/json"
	"strings"
)

// AdminDialogueConditions 描述后台可编辑的任务可见性条件；用于剧情节点/选项与 NPC 菜单项。
type AdminDialogueConditions struct {
	QuestID              uint64 `json:"quest_id"`
	QuestState           string `json:"quest_state"`
	ObjectiveID          uint64 `json:"objective_id"`
	ObjectiveCompleted   *bool  `json:"objective_completed"`
}

// Normalize 去掉前后空格，避免运营误填空白条件。
func (c AdminDialogueConditions) Normalize() AdminDialogueConditions {
	c.QuestState = strings.TrimSpace(c.QuestState)
	return c
}

// IsEmpty 判断当前是否未配置任何可见性条件。
func (c AdminDialogueConditions) IsEmpty() bool {
	c = c.Normalize()
	return c.QuestID == 0 && c.QuestState == "" && c.ObjectiveID == 0 && c.ObjectiveCompleted == nil
}

// EncodeAdminConditionsJSON 把后台结构化条件编码成数据库 conditions_json。
func EncodeAdminConditionsJSON(conditions AdminDialogueConditions) json.RawMessage {
	conditions = conditions.Normalize()
	if conditions.IsEmpty() {
		return json.RawMessage("{}")
	}
	payload := map[string]any{}
	if conditions.QuestID > 0 {
		payload["quest_id"] = conditions.QuestID
	}
	if conditions.QuestState != "" {
		payload["quest_state"] = conditions.QuestState
	}
	if conditions.ObjectiveID > 0 {
		payload["objective_id"] = conditions.ObjectiveID
	}
	if conditions.ObjectiveCompleted != nil {
		payload["objective_completed"] = *conditions.ObjectiveCompleted
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return json.RawMessage("{}")
	}
	return encoded
}

// DecodeAdminConditionsJSON 把数据库 conditions_json 还原成后台表单可直接编辑的结构。
func DecodeAdminConditionsJSON(raw json.RawMessage) AdminDialogueConditions {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "{}" || text == "null" {
		return AdminDialogueConditions{}
	}
	var payload struct {
		QuestID            uint64 `json:"quest_id"`
		QuestState         string `json:"quest_state"`
		ObjectiveID        uint64 `json:"objective_id"`
		ObjectiveCompleted *bool  `json:"objective_completed"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return AdminDialogueConditions{}
	}
	return AdminDialogueConditions{
		QuestID:            payload.QuestID,
		QuestState:         strings.TrimSpace(payload.QuestState),
		ObjectiveID:        payload.ObjectiveID,
		ObjectiveCompleted: payload.ObjectiveCompleted,
	}
}
