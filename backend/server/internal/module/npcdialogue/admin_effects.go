package npcdialogue

import (
	"encoding/json"
	"strings"
)

// AdminDialogueEffectGrantItem 描述后台表单里配置的一条剧情发物品。
type AdminDialogueEffectGrantItem struct {
	ItemID   uint64 `json:"item_id"`
	Quantity uint64 `json:"quantity"`
}

// AdminDialogueEffects 描述后台可编辑的节点副作用；保存时会序列化到 effects_json。
type AdminDialogueEffects struct {
	Notice        string                         `json:"notice"`
	QuestEvent    string                         `json:"quest_event"`
	AcceptQuestID uint64                         `json:"accept_quest_id"`
	GrantItems    []AdminDialogueEffectGrantItem `json:"grant_items"`
}

// Normalize 去掉前后空格并清理无效发物品条目。
func (e AdminDialogueEffects) Normalize() AdminDialogueEffects {
	e.Notice = strings.TrimSpace(e.Notice)
	e.QuestEvent = strings.TrimSpace(e.QuestEvent)
	grantItems := make([]AdminDialogueEffectGrantItem, 0, len(e.GrantItems))
	for _, item := range e.GrantItems {
		if item.ItemID == 0 || item.Quantity == 0 {
			continue
		}
		grantItems = append(grantItems, item)
	}
	e.GrantItems = grantItems
	return e
}

// IsEmpty 判断当前是否未配置任何副作用。
func (e AdminDialogueEffects) IsEmpty() bool {
	e = e.Normalize()
	return e.Notice == "" && e.QuestEvent == "" && e.AcceptQuestID == 0 && len(e.GrantItems) == 0
}

// EncodeAdminEffectsJSON 把后台结构化副作用编码成数据库 effects_json。
func EncodeAdminEffectsJSON(effects AdminDialogueEffects) json.RawMessage {
	effects = effects.Normalize()
	if effects.IsEmpty() {
		return json.RawMessage("{}")
	}
	payload := map[string]any{}
	if effects.Notice != "" {
		payload["notice"] = effects.Notice
	}
	if effects.QuestEvent != "" {
		payload["quest_event"] = effects.QuestEvent
	}
	if effects.AcceptQuestID > 0 {
		payload["accept_quest_id"] = effects.AcceptQuestID
	}
	if len(effects.GrantItems) > 0 {
		payload["grant_items"] = effects.GrantItems
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return json.RawMessage("{}")
	}
	return encoded
}

// DecodeAdminEffectsJSON 把数据库 effects_json 还原成后台表单可直接编辑的结构。
func DecodeAdminEffectsJSON(raw json.RawMessage) AdminDialogueEffects {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "{}" || text == "null" {
		return AdminDialogueEffects{}
	}
	var payload struct {
		Notice        string                         `json:"notice"`
		QuestEvent    string                         `json:"quest_event"`
		AcceptQuestID uint64                         `json:"accept_quest_id"`
		GrantItems    []AdminDialogueEffectGrantItem `json:"grant_items"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return AdminDialogueEffects{}
	}
	return AdminDialogueEffects{
		Notice:        strings.TrimSpace(payload.Notice),
		QuestEvent:    strings.TrimSpace(payload.QuestEvent),
		AcceptQuestID: payload.AcceptQuestID,
		GrantItems:    payload.GrantItems,
	}
}
