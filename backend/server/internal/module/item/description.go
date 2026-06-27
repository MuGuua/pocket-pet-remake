package item

import (
	"fmt"
	"regexp"
	"strconv"
)

var mentionTokenPattern = regexp.MustCompile(`\{item:(\d+)\}`)

// DescriptionMention 描述物品介绍文案中通过 {item:ID} 引入的其他物品。
type DescriptionMention struct {
	ItemID   uint64 `json:"item_id"`
	ItemName string `json:"item_name"`
}

// ExtractMentionItemIDs 从介绍文案中提取去重后的物品模板 ID 列表。
func ExtractMentionItemIDs(description string) []uint64 {
	matches := mentionTokenPattern.FindAllStringSubmatch(description, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[uint64]struct{}, len(matches))
	itemIDs := make([]uint64, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		itemID, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil || itemID == 0 {
			continue
		}
		if _, ok := seen[itemID]; ok {
			continue
		}
		seen[itemID] = struct{}{}
		itemIDs = append(itemIDs, itemID)
	}
	return itemIDs
}

// BuildDescriptionMentions 根据介绍文案与名称表生成客户端渲染用的提及列表。
func BuildDescriptionMentions(description string, itemNames map[uint64]string) []DescriptionMention {
	itemIDs := ExtractMentionItemIDs(description)
	if len(itemIDs) == 0 {
		return nil
	}
	mentions := make([]DescriptionMention, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		itemName := itemNames[itemID]
		if itemName == "" {
			itemName = fmt.Sprintf("物品%d", itemID)
		}
		mentions = append(mentions, DescriptionMention{
			ItemID:   itemID,
			ItemName: itemName,
		})
	}
	return mentions
}
