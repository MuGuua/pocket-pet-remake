package equipment

import "testing"

// TestComputeWeaponSkillLevel 验证武器技能等级随强化等级线性成长。
func TestComputeWeaponSkillLevel(t *testing.T) {
	got := ComputeWeaponSkillLevel(1, 1, 5)
	if got != 6 {
		t.Fatalf("ComputeWeaponSkillLevel() = %d, want 6", got)
	}
}

// TestResolveWeaponSkillsDedupesSkillID 验证同一 skill_id 只保留一条有效等级。
func TestResolveWeaponSkillsDedupesSkillID(t *testing.T) {
	configs := []AdminWeaponSkillConfig{
		{SkillID: 1201, BaseLevel: 1},
		{SkillID: 1201, BaseLevel: 9},
	}
	got := ResolveWeaponSkills(configs, map[string]uint32{"1201": 1}, 2)
	if len(got) != 1 {
		t.Fatalf("ResolveWeaponSkills() len = %d, want 1", len(got))
	}
	if got[0].Level != 3 {
		t.Fatalf("ResolveWeaponSkills() level = %d, want 3", got[0].Level)
	}
}

// TestIsWeaponEquipSlot 验证仅武器槽位允许挂载武器技能。
func TestIsWeaponEquipSlot(t *testing.T) {
	if !IsWeaponEquipSlot(EquipSlotWeapon) {
		t.Fatal("IsWeaponEquipSlot(weapon) = false, want true")
	}
	if IsWeaponEquipSlot(EquipSlotHat) {
		t.Fatal("IsWeaponEquipSlot(hat) = true, want false")
	}
}
