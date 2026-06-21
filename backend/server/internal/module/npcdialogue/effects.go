package npcdialogue

import (
	"encoding/json"
	"strings"
)

// EffectGrantItem 描述剧情节点进入时要发放给玩家的一件物品。
type EffectGrantItem struct {
	ItemID   uint64 `json:"item_id"`
	Quantity uint64 `json:"quantity"`
}

// NodeEffects 描述剧情节点进入时可触发的服务端权威副作用。
type NodeEffects struct {
	Notice        string
	QuestEvent    string
	AcceptQuestID uint64
	GrantItems    []EffectGrantItem
}

// ParseNodeEffects 解析节点配置中的 effects_json，供运行时推进剧情时执行。
func ParseNodeEffects(raw json.RawMessage) NodeEffects {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "{}" || text == "null" {
		return NodeEffects{}
	}
	var payload struct {
		Notice        string `json:"notice"`
		QuestEvent    string `json:"quest_event"`
		AcceptQuestID uint64 `json:"accept_quest_id"`
		GrantItems    []struct {
			ItemID   uint64 `json:"item_id"`
			Quantity uint64 `json:"quantity"`
		} `json:"grant_items"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return NodeEffects{}
	}
	grantItems := make([]EffectGrantItem, 0, len(payload.GrantItems))
	for _, item := range payload.GrantItems {
		if item.ItemID == 0 || item.Quantity == 0 {
			continue
		}
		grantItems = append(grantItems, EffectGrantItem{ItemID: item.ItemID, Quantity: item.Quantity})
	}
	return NodeEffects{
		Notice:        strings.TrimSpace(payload.Notice),
		QuestEvent:    strings.TrimSpace(payload.QuestEvent),
		AcceptQuestID: payload.AcceptQuestID,
		GrantItems:    grantItems,
	}
}
