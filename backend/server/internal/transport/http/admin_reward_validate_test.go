package httptransport

import (
	"testing"

	"pocket-pet-remake/server/internal/module/reward"
)

// TestValidateAdminRewardEntries 锁定后台补发入口的白名单与上限规则。
func TestValidateAdminRewardEntries(t *testing.T) {
	valid := []reward.Entry{
		{Type: "item", ItemID: 1001, Count: 5},
		{Type: "gold", Value: 1000},
		{Type: "exp", Value: 200},
		{Type: "pet", PetID: 101},
	}
	if err := validateAdminRewardEntries(valid); err != nil {
		t.Fatalf("valid entries err = %v", err)
	}

	invalid := map[string][]reward.Entry{
		"未知类型":  {{Type: "diamond", Value: 1}},
		"物品缺ID": {{Type: "item", Count: 1}},
		"物品数量0": {{Type: "item", ItemID: 1001}},
		"金币为0":  {{Type: "gold"}},
		"金币超上限": {{Type: "gold", Value: adminRewardMaxAmount + 1}},
		"宠物缺ID": {{Type: "pet"}},
	}
	for name, entries := range invalid {
		if err := validateAdminRewardEntries(entries); err == nil {
			t.Fatalf("%s: err = nil, want rejection", name)
		}
	}

	tooMany := make([]reward.Entry, adminRewardMaxEntries+1)
	for i := range tooMany {
		tooMany[i] = reward.Entry{Type: "gold", Value: 1}
	}
	if err := validateAdminRewardEntries(tooMany); err == nil {
		t.Fatal("too many entries should be rejected")
	}
}
