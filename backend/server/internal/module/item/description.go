package item

import (
	"fmt"
	"regexp"
	"strconv"
)

var mentionTokenPattern = regexp.MustCompile(`\{item:(\d+)\}`)
var mentionPetTokenPattern = regexp.MustCompile(`\{pet:(\d+)\}`)

// DescriptionMention 描述文案中通过 {item:ID} / {pet:ID} 引入的其他模板。
type DescriptionMention struct {
	ItemID   uint64 `json:"item_id,omitempty"`
	PetID    uint64 `json:"pet_id,omitempty"`
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

// ExtractMentionPetIDs 从介绍文案中提取去重后的宠物模板 ID 列表。
func ExtractMentionPetIDs(description string) []uint64 {
	matches := mentionPetTokenPattern.FindAllStringSubmatch(description, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[uint64]struct{}, len(matches))
	petIDs := make([]uint64, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		petID, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil || petID == 0 {
			continue
		}
		if _, ok := seen[petID]; ok {
			continue
		}
		seen[petID] = struct{}{}
		petIDs = append(petIDs, petID)
	}
	return petIDs
}

// BuildDescriptionMentions 根据介绍文案与名称表生成客户端渲染用的提及列表。
func BuildDescriptionMentions(description string, itemNames map[uint64]string, petNames map[uint64]string) []DescriptionMention {
	itemIDs := ExtractMentionItemIDs(description)
	petIDs := ExtractMentionPetIDs(description)
	if len(itemIDs) == 0 && len(petIDs) == 0 {
		return nil
	}
	mentions := make([]DescriptionMention, 0, len(itemIDs)+len(petIDs))
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
	for _, petID := range petIDs {
		petName := petNames[petID]
		if petName == "" {
			petName = fmt.Sprintf("宠物%d", petID)
		}
		mentions = append(mentions, DescriptionMention{
			PetID:    petID,
			ItemName: petName,
		})
	}
	return mentions
}
