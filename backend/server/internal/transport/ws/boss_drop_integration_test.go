package wstransport

import (
	"context"
	"testing"

	"pocket-pet-remake/server/internal/module/monster"
	"pocket-pet-remake/server/internal/teststub"
)

// TestResolveWildEncounterIncludesBossDrops 验证遭遇战解析会把该场景的 Boss 掉落池挂到运行时遭遇上。
func TestResolveWildEncounterIncludesBossDrops(t *testing.T) {
	repo := teststub.NewMonsterRepository()
	service := monster.NewService(repo, nil, nil)
	ctx := context.Background()

	if _, err := repo.ReplaceBossDropsForScene(ctx, 4, []monster.AdminBossDropInput{
		{ItemID: 3101, Quantity: 1, Weight: 100, DrawGroup: 1},
		{ItemID: 3102, Quantity: 1, Weight: 100, DrawGroup: 1},
	}); err != nil {
		t.Fatalf("ReplaceBossDropsForScene() error = %v", err)
	}

	encounter, err := service.ResolveWildEncounterForScene(ctx, 4)
	if err != nil {
		t.Fatalf("ResolveWildEncounterForScene() error = %v", err)
	}
	if encounter == nil {
		t.Fatal("ResolveWildEncounterForScene() = nil, want encounter")
	}
	if len(encounter.BossDrops) != 2 {
		t.Fatalf("len(BossDrops) = %d, want 2", len(encounter.BossDrops))
	}
}

// TestResolveWildEncounterWithoutBossDrops 验证没有配置掉落池时遭遇战仍然可用。
func TestResolveWildEncounterWithoutBossDrops(t *testing.T) {
	repo := teststub.NewMonsterRepository()
	service := monster.NewService(repo, nil, nil)
	encounter, err := service.ResolveWildEncounterForScene(context.Background(), 4)
	if err != nil {
		t.Fatalf("ResolveWildEncounterForScene() error = %v", err)
	}
	if encounter == nil {
		t.Fatal("ResolveWildEncounterForScene() = nil, want encounter")
	}
	if len(encounter.BossDrops) != 0 {
		t.Fatalf("len(BossDrops) = %d, want 0", len(encounter.BossDrops))
	}
}

// TestAdminBossDropsRoundTrip 验证后台保存后会覆盖原配置并原样读回。
func TestAdminBossDropsRoundTrip(t *testing.T) {
	repo := teststub.NewMonsterRepository()
	service := monster.NewService(repo, nil, nil)
	ctx := context.Background()

	if _, err := service.ReplaceAdminBossDrops(ctx, 4, monster.AdminReplaceBossDropsInput{
		Drops: []monster.AdminBossDropInput{
			{ItemID: 3101, Quantity: 2, Weight: 300, DrawGroup: 1},
			{ItemID: 3102, Quantity: 1, Weight: 700, DrawGroup: 2},
		},
	}); err != nil {
		t.Fatalf("ReplaceAdminBossDrops() error = %v", err)
	}
	items, err := service.ListAdminBossDrops(ctx, 4)
	if err != nil {
		t.Fatalf("ListAdminBossDrops() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ItemID != 3101 || items[0].Quantity != 2 || items[0].Weight != 300 {
		t.Fatalf("items[0] = %+v, want item 3101 qty 2 weight 300", items[0])
	}

	// 再次覆盖保存后旧配置应被清空，不残留第一次的条目。
	if _, err := service.ReplaceAdminBossDrops(ctx, 4, monster.AdminReplaceBossDropsInput{
		Drops: []monster.AdminBossDropInput{{ItemID: 9999, Quantity: 1, Weight: 100, DrawGroup: 1}},
	}); err != nil {
		t.Fatalf("ReplaceAdminBossDrops(second) error = %v", err)
	}
	items, err = service.ListAdminBossDrops(ctx, 4)
	if err != nil {
		t.Fatalf("ListAdminBossDrops(second) error = %v", err)
	}
	if len(items) != 1 || items[0].ItemID != 9999 {
		t.Fatalf("items after replace = %+v, want only item 9999", items)
	}
}
