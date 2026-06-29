package wstransport

import (
	"context"
	"testing"

	"pocket-pet-remake/server/internal/module/item"
	"pocket-pet-remake/server/internal/protocol"
	"pocket-pet-remake/server/internal/teststub"
)

// TestEnrichProtocolPopupRewardItemNamesFillsMissingItemName 验证弹窗奖励会补全缺失的物品展示名。
func TestEnrichProtocolPopupRewardItemNamesFillsMissingItemName(t *testing.T) {
	t.Parallel()

	itemService := item.NewService(teststub.NewItemRepository())
	rewards := []protocol.QuestReward{
		{Type: "exp", Value: 28},
		{Type: "item", ItemID: 3004, Count: 1},
	}

	enriched := enrichProtocolPopupRewardItemNames(context.Background(), itemService, rewards)
	if len(enriched) != 2 {
		t.Fatalf("len(enriched) = %d, want 2", len(enriched))
	}
	if enriched[1].ItemName != "新手补给礼包" {
		t.Fatalf("item_name = %q, want %q", enriched[1].ItemName, "新手补给礼包")
	}
}

// TestEnrichProtocolPopupRewardItemNamesKeepsExistingName 验证已有 item_name 不会被覆盖。
func TestEnrichProtocolPopupRewardItemNamesKeepsExistingName(t *testing.T) {
	t.Parallel()

	itemService := item.NewService(teststub.NewItemRepository())
	rewards := []protocol.QuestReward{
		{Type: "item", ItemID: 3004, ItemName: "自定义展示名", Count: 1},
	}

	enriched := enrichProtocolPopupRewardItemNames(context.Background(), itemService, rewards)
	if enriched[0].ItemName != "自定义展示名" {
		t.Fatalf("item_name = %q, want preserved custom name", enriched[0].ItemName)
	}
}
