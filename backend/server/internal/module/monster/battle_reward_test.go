package monster

import "testing"

func TestBattleRewardCacheReplaceKeepsProbabilisticRewards(t *testing.T) {
	cache := newBattleRewardCache()
	cache.replace([]BattleRewardEntry{
		{
			MonsterID:  9001,
			RewardType: RewardTypeItem,
			ItemID:     3202,
			Quantity:   10,
			DropRate:   1,
			Status:     1,
		},
	})

	cache.mu.RLock()
	entries := append([]BattleRewardEntry(nil), cache.entries[9001]...)
	cache.mu.RUnlock()

	if len(entries) != 1 {
		t.Fatalf("cached reward entries = %d, want 1; probability must be rolled at battle settlement, not cache refresh", len(entries))
	}
	if entries[0].ItemID != 3202 || entries[0].Quantity != 10 || entries[0].DropRate != 1 {
		t.Fatalf("cached reward entry = %#v, want repair gem reward kept unchanged", entries[0])
	}
}

func TestBuildPVERewardBundleAlwaysGrantsMaxRateItem(t *testing.T) {
	bundle := BuildPVERewardBundle([]BattleRewardEntry{
		{
			MonsterID:  9001,
			RewardType: RewardTypeItem,
			ItemID:     3202,
			Quantity:   10,
			DropRate:   RewardRateMax,
			Status:     1,
		},
	})

	if len(bundle.Items) != 1 {
		t.Fatalf("len(bundle.Items) = %d, want 1", len(bundle.Items))
	}
	if bundle.Items[0].ItemID != 3202 || bundle.Items[0].Quantity != 10 {
		t.Fatalf("bundle.Items[0] = %#v, want repair gem x10", bundle.Items[0])
	}
}
