package monster

import "testing"

// TestDrawBossDropsPicksOnePerGroup 验证同组只抽 1 个、不同组互不影响。
func TestDrawBossDropsPicksOnePerGroup(t *testing.T) {
	entries := []BossDropEntry{
		{SceneID: 4, ItemID: 3101, Quantity: 1, Weight: 100, DrawGroup: 1, Status: 1},
		{SceneID: 4, ItemID: 3102, Quantity: 1, Weight: 100, DrawGroup: 1, Status: 1},
		{SceneID: 4, ItemID: 3103, Quantity: 2, Weight: 100, DrawGroup: 2, Status: 1},
	}
	drops := DrawBossDrops(entries)
	if len(drops) != 2 {
		t.Fatalf("len(drops) = %d, want 2 (one per draw group)", len(drops))
	}
	firstGroupHit := 0
	for _, drop := range drops {
		switch drop.ItemID {
		case 3101, 3102:
			firstGroupHit++
		case 3103:
			if drop.Quantity != 2 {
				t.Fatalf("item 3103 quantity = %d, want 2", drop.Quantity)
			}
		default:
			t.Fatalf("unexpected drop item = %d", drop.ItemID)
		}
	}
	if firstGroupHit != 1 {
		t.Fatalf("group 1 hit count = %d, want exactly 1", firstGroupHit)
	}
}

// TestDrawBossDropsRespectsWeight 验证权重大的条目在多轮抽取中明显更常命中。
func TestDrawBossDropsRespectsWeight(t *testing.T) {
	entries := []BossDropEntry{
		{SceneID: 4, ItemID: 1, Quantity: 1, Weight: 900, DrawGroup: 1, Status: 1},
		{SceneID: 4, ItemID: 2, Quantity: 1, Weight: 100, DrawGroup: 1, Status: 1},
	}
	hits := map[uint64]int{}
	for i := 0; i < 2000; i++ {
		drops := DrawBossDrops(entries)
		if len(drops) != 1 {
			t.Fatalf("len(drops) = %d, want 1", len(drops))
		}
		hits[drops[0].ItemID]++
	}
	if hits[1] <= hits[2] {
		t.Fatalf("weighted draw hits = %+v, want item 1 (weight 900) more frequent than item 2 (weight 100)", hits)
	}
	// 900:100 的期望比例下，2000 次里高权重条目应占绝大多数；放宽到 70% 避免偶发失败。
	if hits[1] < 1400 {
		t.Fatalf("high weight hit count = %d, want at least 1400 of 2000", hits[1])
	}
}

// TestDrawBossDropsSkipsDisabledEntries 验证停用条目不会进入抽取。
func TestDrawBossDropsSkipsDisabledEntries(t *testing.T) {
	entries := []BossDropEntry{
		{SceneID: 4, ItemID: 1, Quantity: 1, Weight: 1, DrawGroup: 1, Status: 0},
		{SceneID: 4, ItemID: 2, Quantity: 1, Weight: 1, DrawGroup: 1, Status: 1},
	}
	for i := 0; i < 50; i++ {
		drops := DrawBossDrops(entries)
		if len(drops) != 1 || drops[0].ItemID != 2 {
			t.Fatalf("drops = %+v, want only enabled item 2", drops)
		}
	}
}

// TestDrawBossDropsEmptyPool 验证空池返回空结果而不是 panic。
func TestDrawBossDropsEmptyPool(t *testing.T) {
	if drops := DrawBossDrops(nil); len(drops) != 0 {
		t.Fatalf("len(drops) = %d, want 0", len(drops))
	}
	if drops := DrawBossDrops([]BossDropEntry{}); len(drops) != 0 {
		t.Fatalf("len(drops) = %d, want 0", len(drops))
	}
}

// TestBossDropsForSceneFiltersBySceneAndStatus 验证按场景与启用状态过滤。
func TestBossDropsForSceneFiltersBySceneAndStatus(t *testing.T) {
	entries := []BossDropEntry{
		{SceneID: 4, ItemID: 1, Status: 1},
		{SceneID: 4, ItemID: 2, Status: 0},
		{SceneID: 5, ItemID: 3, Status: 1},
	}
	filtered := BossDropsForScene(entries, 4)
	if len(filtered) != 1 || filtered[0].ItemID != 1 {
		t.Fatalf("filtered = %+v, want only scene 4 enabled item 1", filtered)
	}
	if filtered := BossDropsForScene(entries, 0); filtered != nil {
		t.Fatalf("filtered for scene 0 = %+v, want nil", filtered)
	}
}

// TestSanitizeBossDropInputsFillsDefaults 验证后台输入的默认值与非法值兜底。
func TestSanitizeBossDropInputsFillsDefaults(t *testing.T) {
	inputs := []AdminBossDropInput{
		{ItemID: 100},
		{ItemID: 0},
		{ItemID: 200, Quantity: 3, Weight: -5, DrawGroup: 0, GrantOnce: 9, Status: 0},
	}
	result := SanitizeBossDropInputs(inputs)
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2 (item_id=0 dropped)", len(result))
	}
	if result[0].Quantity != 1 || result[0].Weight != 100 || result[0].DrawGroup != 1 || result[0].Status != 1 {
		t.Fatalf("result[0] = %+v, want normalized defaults", result[0])
	}
	if result[0].SortOrder != 1 {
		t.Fatalf("result[0].SortOrder = %d, want 1 (auto-filled)", result[0].SortOrder)
	}
	if result[1].Weight != 100 || result[1].GrantOnce != 1 {
		t.Fatalf("result[1] = %+v, want weight 100 and grant_once capped to 1", result[1])
	}
}
