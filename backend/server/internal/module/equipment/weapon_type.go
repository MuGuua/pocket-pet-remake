package equipment

import "strings"

// 武器类型常量：与 skill.WeaponDiscipline 取值一致，供战斗判定「已学会技能需装备同类型武器」。
const (
	WeaponTypeSword = "sword"
	WeaponTypeSpear = "spear"
	WeaponTypeStaff = "staff"
)

// PlayerFacingWeaponTypes 返回运营/后台下拉使用的武器类型列表。
func PlayerFacingWeaponTypes() []string {
	return []string{WeaponTypeSword, WeaponTypeSpear, WeaponTypeStaff}
}

// IsValidWeaponType 判断武器类型是否为系统支持的枚举值。
func IsValidWeaponType(value string) bool {
	switch strings.TrimSpace(value) {
	case WeaponTypeSword, WeaponTypeSpear, WeaponTypeStaff:
		return true
	default:
		return false
	}
}

// WeaponTypeLabel 返回武器类型的简体中文展示名。
func WeaponTypeLabel(value string) string {
	switch strings.TrimSpace(value) {
	case WeaponTypeSword:
		return "剑"
	case WeaponTypeSpear:
		return "枪"
	case WeaponTypeStaff:
		return "法杖"
	default:
		return value
	}
}
