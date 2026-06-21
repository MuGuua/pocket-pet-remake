package npc

import "encoding/json"

type MenuEntry struct {
	EntityID                uint64
	EntryID                 string
	EntryType               string
	Title                   string
	Subtitle                string
	State                   string
	Priority                uint32
	ActionResultType        string
	ActionNotice            string
	BattleEncounterEntityID uint64
	ConditionsJSON          json.RawMessage
	LinkedQuestID           uint64
}

type ActionResult struct {
	EntityID                uint64
	EntryID                 string
	ResultType              string
	Notice                  string
	BattleEncounterEntityID uint64
	LinkedQuestID           uint64
}

// ShopGood 描述某个商店 NPC 当前可售的一件商品；价格来自物品模板并由服务端权威返回。
type ShopGood struct {
	ItemID         uint64
	ItemName       string
	BuyPriceCopper uint64
	SortOrder      int
}
