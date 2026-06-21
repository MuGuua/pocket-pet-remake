package pet

import "testing"

// TestClampPetCombatStats 验证宠物战斗字段会按默认封顶表截断。
func TestClampPetCombatStats(t *testing.T) {
	item := &Pet{
		HP:                2_000_000,
		HPMax:             2_000_000,
		ATK:               300_000,
		Spirit:            1_500,
		SpiritMax:         1_500,
		HitPct:            999,
		CritDmgPct:        5_000,
		ConfusionResistPct: 900,
	}
	caps := DefaultCombatStatCaps()
	ClampPetCombatStats(item, caps)
	if item.HPMax != caps.Cap(CombatStatCapHPMax) {
		t.Fatalf("hp_max = %d, want %d", item.HPMax, caps.Cap(CombatStatCapHPMax))
	}
	if item.HP != item.HPMax {
		t.Fatalf("hp = %d, want clamped to hp_max %d", item.HP, item.HPMax)
	}
	if item.ATK != caps.Cap(CombatStatCapATK) {
		t.Fatalf("atk = %d, want %d", item.ATK, caps.Cap(CombatStatCapATK))
	}
	if item.Spirit != caps.Cap(CombatStatCapSpirit) {
		t.Fatalf("spirit = %d, want %d", item.Spirit, caps.Cap(CombatStatCapSpirit))
	}
	if item.HitPct != caps.Cap(CombatStatCapHitPct) {
		t.Fatalf("hit_pct = %d, want %d", item.HitPct, caps.Cap(CombatStatCapHitPct))
	}
	if item.CritDmgPct != caps.Cap(CombatStatCapCritDmgPct) {
		t.Fatalf("crit_dmg_pct = %d, want %d", item.CritDmgPct, caps.Cap(CombatStatCapCritDmgPct))
	}
	if item.ConfusionResistPct != caps.Cap(CombatStatCapConfusionResistPct) {
		t.Fatalf("confusion_resist_pct = %d, want %d", item.ConfusionResistPct, caps.Cap(CombatStatCapConfusionResistPct))
	}
}
