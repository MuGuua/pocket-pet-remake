package monster

import "testing"

// TestBuildPVERewardBundleKeepsIndependentEntries 验证 draw_group=0 的条目仍是各自独立判定。
func TestBuildPVERewardBundleKeepsIndependentEntries(t *testing.T) {
	entries := []BattleRewardEntry{
		{RewardType: RewardTypeItem, ItemID: 1, Quantity: 1, DropRate: RewardRateMax, Status: 1},
		{RewardType: RewardTypeItem, ItemID: 2, Quantity: 1, DropRate: RewardRateMax, Status: 1},
	}
	bundle := BuildPVERewardBundle(entries)
	if len(bundle.Items) != 2 {
		t.Fatalf("len(items) = %d, want 2 (independent entries both granted)", len(bundle.Items))
	}
}

// TestBuildPVERewardBundleDrawsOnePerGroup 验证同组只抽 1 个、不同组各抽 1 个。
func TestBuildPVERewardBundleDrawsOnePerGroup(t *testing.T) {
	entries := []BattleRewardEntry{
		{RewardType: RewardTypeItem, ItemID: 1, Quantity: 1, DropRate: RewardRateMax, DrawGroup: 1, Weight: 100, Status: 1},
		{RewardType: RewardTypeItem, ItemID: 2, Quantity: 1, DropRate: RewardRateMax, DrawGroup: 1, Weight: 100, Status: 1},
		{RewardType: RewardTypeItem, ItemID: 3, Quantity: 1, DropRate: RewardRateMax, DrawGroup: 2, Weight: 100, Status: 1},
	}
	bundle := BuildPVERewardBundle(entries)
	if len(bundle.Items) != 2 {
		t.Fatalf("len(items) = %d, want 2 (one per draw group)", len(bundle.Items))
	}
	groupOneHits := 0
	for _, item := range bundle.Items {
		switch item.ItemID {
		case 1, 2:
			groupOneHits++
		case 3:
		default:
			t.Fatalf("unexpected item id %d", item.ItemID)
		}
	}
	if groupOneHits != 1 {
		t.Fatalf("group 1 hit count = %d, want 1", groupOneHits)
	}
}

// TestBuildPVERewardBundleGroupRespectsWeight 验证组内权重影响命中分布。
func TestBuildPVERewardBundleGroupRespectsWeight(t *testing.T) {
	entries := []BattleRewardEntry{
		{RewardType: RewardTypeItem, ItemID: 1, Quantity: 1, DrawGroup: 1, Weight: 900, Status: 1},
		{RewardType: RewardTypeItem, ItemID: 2, Quantity: 1, DrawGroup: 1, Weight: 100, Status: 1},
	}
	hits := map[uint64]int{}
	for i := 0; i < 2000; i++ {
		bundle := BuildPVERewardBundle(entries)
		if len(bundle.Items) != 1 {
			t.Fatalf("len(items) = %d, want 1", len(bundle.Items))
		}
		hits[bundle.Items[0].ItemID]++
	}
	if hits[1] < 1400 {
		t.Fatalf("high weight hits = %d of 2000, want at least 1400", hits[1])
	}
}

// TestBuildPVERewardBundleGroupMixedWithIndependent 验证抽取组与独立条目可以并存，互不影响。
func TestBuildPVERewardBundleGroupMixedWithIndependent(t *testing.T) {
	entries := []BattleRewardEntry{
		{RewardType: RewardTypeItem, ItemID: 100, Quantity: 1, DropRate: RewardRateMax, DrawGroup: 0, Status: 1},
		{RewardType: RewardTypeItem, ItemID: 201, Quantity: 1, DrawGroup: 1, Weight: 100, Status: 1},
		{RewardType: RewardTypeItem, ItemID: 202, Quantity: 1, DrawGroup: 1, Weight: 100, Status: 1},
	}
	for i := 0; i < 100; i++ {
		bundle := BuildPVERewardBundle(entries)
		if len(bundle.Items) != 2 {
			t.Fatalf("len(items) = %d, want 2 (1 independent + 1 from group)", len(bundle.Items))
		}
		if bundle.Items[0].ItemID != 100 {
			t.Fatalf("items[0] = %d, want independent item 100", bundle.Items[0].ItemID)
		}
	}
}

// TestBuildPVERewardBundleGroupSupportsCurrency 验证抽取组内可以抽中非物品类奖励。
func TestBuildPVERewardBundleGroupSupportsCurrency(t *testing.T) {
	entries := []BattleRewardEntry{
		{RewardType: RewardTypeCopper, ExpValue: 500, DrawGroup: 1, Weight: 100, Status: 1},
		{RewardType: RewardTypeCopper, ExpValue: 900, DrawGroup: 1, Weight: 100, Status: 1},
	}
	for i := 0; i < 50; i++ {
		bundle := BuildPVERewardBundle(entries)
		if bundle.Gold != 500 && bundle.Gold != 900 {
			t.Fatalf("gold = %d, want one of the two group candidates", bundle.Gold)
		}
	}
}

// TestBuildPVERewardBundleSkipsDisabledGroupEntries 验证停用条目不会进入组内抽取。
func TestBuildPVERewardBundleSkipsDisabledGroupEntries(t *testing.T) {
	entries := []BattleRewardEntry{
		{RewardType: RewardTypeItem, ItemID: 1, Quantity: 1, DrawGroup: 1, Weight: 1, Status: 0},
		{RewardType: RewardTypeItem, ItemID: 2, Quantity: 1, DrawGroup: 1, Weight: 1, Status: 1},
	}
	for i := 0; i < 50; i++ {
		bundle := BuildPVERewardBundle(entries)
		if len(bundle.Items) != 1 || bundle.Items[0].ItemID != 2 {
			t.Fatalf("items = %+v, want only enabled item 2", bundle.Items)
		}
	}
}

// TestAdminBattleRewardInputRoundTripPreservesDrawGroup 验证后台输入经 Normalize 后保留抽取组与权重。
// 该用例防止后续新增字段时只改结构体、忘记在转换链路里透传。
func TestAdminBattleRewardInputRoundTripPreservesDrawGroup(t *testing.T) {
	input := AdminBattleRewardInput{
		RewardType: RewardTypeItem,
		ExpTarget:  ExpTargetPlayer,
		ItemID:     3101,
		Quantity:   1,
		DropRate:   RewardRateMax,
		DrawGroup:  2,
		Weight:     350,
	}
	normalized := input.Normalize()
	if normalized.DrawGroup != 2 {
		t.Fatalf("DrawGroup = %d, want 2", normalized.DrawGroup)
	}
	if normalized.Weight != 350 {
		t.Fatalf("Weight = %d, want 350", normalized.Weight)
	}
}

// TestAdminBattleRewardInputNormalizesZeroWeight 验证 weight=0 会兜底成 100，避免组内权重求和为 0。
func TestAdminBattleRewardInputNormalizesZeroWeight(t *testing.T) {
	input := AdminBattleRewardInput{
		RewardType: RewardTypeItem,
		ItemID:     3101,
		Quantity:   1,
		DropRate:   RewardRateMax,
		DrawGroup:  1,
		Weight:     0,
	}
	if got := input.Normalize().Weight; got != 100 {
		t.Fatalf("Weight = %d, want 100", got)
	}
}
