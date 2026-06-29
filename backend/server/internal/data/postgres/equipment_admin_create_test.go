package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/module/equipment"
)

// TestCreateForAdminWithEnhanceGoldPayload 复现后台新增装备请求体，定位 500 根因。
func TestCreateForAdminWithEnhanceGoldPayload(t *testing.T) {
	raw := `{"item_id":4002,"item_code":"equipment_4002","item_name":"新手长剑(10)","desc":"测试武器","icon":"","quality":1,"rarity":1,"required_level":10,"bind_type":"none","can_sell":true,"can_drop":true,"can_store":true,"is_enabled":true,"equip_slot":"weapon","career_limit":"","can_enhance":true,"max_enhance_level":15,"set_id":0,"appearance_skin_id":"","appearance_only":false,"base_hp":0,"base_mana":0,"base_atk":120,"base_def":0,"base_spd":0,"combat_stats":{"spirit":0,"spirit_max":0,"hit_pct":0,"dodge_pct":0,"crit_rate_pct":0,"crit_dmg_pct":0,"physical_resist_pct":0,"reverse_physical_resist_pct":0,"skill_resist_pct":0,"reverse_skill_resist_pct":0,"confusion_resist_pct":0,"sleep_resist_pct":0,"paralysis_resist_pct":0,"seal_resist_pct":0,"curse_resist_pct":0,"crit_dmg_resist_pct":0,"crit_resist_pct":0,"character_resist_pct":0,"pet_resist_pct":0},"enhance_per_level_stats":{"atk":50},"enhance_gold_cost":{"is_enabled":true,"base_copper":100,"increment_mode":"fixed","increment_fixed":200,"increment_percent":0},"socket_count":0,"allowed_gem_types":[]}`

	var input equipment.AdminUpsertEquipmentInput
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if err := input.Validate(); err != nil {
		t.Fatalf("validate input: %v", err)
	}

	db, err := Open(config.PostgresConfig{DSN: "postgres://postgres:postgres@127.0.0.1:5432/pocket_pet?sslmode=disable"})
	if err != nil {
		t.Skipf("skip integration test, open db failed: %v", err)
	}
	defer db.Close()

	repo := NewEquipmentRepository(db)
	_, _ = db.ExecContext(context.Background(), `DELETE FROM item_equipment_extra WHERE item_id = 4002`)
	_, _ = db.ExecContext(context.Background(), `DELETE FROM item_definition WHERE item_id = 4002`)

	created, err := repo.CreateForAdmin(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateForAdmin() error = %v", err)
	}
	if created.ItemID != 4002 {
		t.Fatalf("ItemID = %d, want 4002", created.ItemID)
	}
}
