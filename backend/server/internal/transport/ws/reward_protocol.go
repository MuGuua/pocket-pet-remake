package wstransport

import (
	"context"
	"strings"

	"pocket-pet-remake/server/internal/module/item"
	"pocket-pet-remake/server/internal/module/progression"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/reward"
	"pocket-pet-remake/server/internal/protocol"
)

// toProtocolPopupRewards 把统一发奖结果转成客户端奖励弹窗可展示的结构。
// 当前弹窗仅展示经验与物品奖励，其它类型仍会继续发奖但不进入弹窗列表。
func toProtocolPopupRewards(values []reward.Entry) []protocol.QuestReward {
	result := make([]protocol.QuestReward, 0, len(values))
	for _, value := range values {
		rewardType := strings.ToLower(strings.TrimSpace(value.Type))
		if rewardType != "exp" && rewardType != "item" && rewardType != "gold" {
			continue
		}
		result = append(result, protocol.QuestReward{
			Type:     rewardType,
			Value:    value.Value,
			ItemID:   value.ItemID,
			ItemName: strings.TrimSpace(value.ItemName),
			Count:    value.Count,
			PetID:    value.PetID,
		})
	}
	return result
}

// enrichProtocolPopupRewardItemNames 为弹窗奖励里缺失展示名的物品条目补全服务端权威名称。
// 战斗快照回退、旧任务奖励模板等路径可能只携带 item_id，客户端无法自行推导正式名称。
func enrichProtocolPopupRewardItemNames(ctx context.Context, itemService *item.Service, rewards []protocol.QuestReward) []protocol.QuestReward {
	if len(rewards) == 0 || itemService == nil {
		return rewards
	}
	for index := range rewards {
		rewardType := strings.ToLower(strings.TrimSpace(rewards[index].Type))
		if rewardType != "item" {
			continue
		}
		if strings.TrimSpace(rewards[index].ItemName) != "" {
			continue
		}
		if rewards[index].ItemID == 0 {
			continue
		}
		detail, err := itemService.GetRuntimeItemDetail(ctx, rewards[index].ItemID)
		if err != nil || detail == nil {
			continue
		}
		rewards[index].ItemName = strings.TrimSpace(detail.ItemName)
	}
	return rewards
}

// toProtocolQuestRewards 保留任务协议兼容转换，供旧逻辑复用。
func toProtocolQuestRewardsFromQuest(values []quest.Reward) []protocol.QuestReward {
	result := make([]protocol.QuestReward, 0, len(values))
	for _, value := range values {
		result = append(result, protocol.QuestReward{
			Type:   value.Type,
			Value:  value.Value,
			ItemID: value.ItemID,
			Count:  value.Count,
			PetID:  value.PetID,
		})
	}
	return result
}

// toProtocolLevelUpBonus 把服务端权威升级加成摘要转成客户端升级弹窗字段。
func toProtocolLevelUpBonus(levelUpCount uint32, bonus progression.LevelUpCombatBonus) *protocol.LevelUpBonus {
	if levelUpCount == 0 {
		return nil
	}
	return &protocol.LevelUpBonus{
		HPMax: bonus.HPMax,
		ATK:   bonus.ATK,
		SPD:   bonus.SPD,
		MANA:  bonus.MANA,
	}
}
