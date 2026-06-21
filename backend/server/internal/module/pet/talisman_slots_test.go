package pet

import "testing"

func TestResolveTalismanSlotColumns(t *testing.T) {
	columns, err := ResolveTalismanSlotColumns(TalismanSlotKeyActive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if columns.EnabledColumn != "active_talisman_enabled" {
		t.Fatalf("enabled column=%s", columns.EnabledColumn)
	}
}

func TestApplyTalismanSlotUnlock(t *testing.T) {
	loadout := SkillLoadout{}
	if err := ApplyTalismanSlotUnlock(&loadout, TalismanSlotKey1, 9001); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}
	if !loadout.TalismanSlot1Enabled || loadout.TalismanSlot1SkillID != 9001 {
		t.Fatalf("loadout=%+v", loadout)
	}
}
