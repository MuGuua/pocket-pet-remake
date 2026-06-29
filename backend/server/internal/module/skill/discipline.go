package skill

import "strings"

// 武器技能流派常量，与 equipment.WeaponType 对齐。
const (
	DisciplineSword = "sword"
	DisciplineSpear = "spear"
	DisciplineStaff = "staff"
)

// PlayerFacingWeaponDisciplines 返回后台录入武器技能流派时的选项。
func PlayerFacingWeaponDisciplines() []string {
	return []string{DisciplineSword, DisciplineSpear, DisciplineStaff}
}

// IsValidWeaponDiscipline 判断武器技能流派是否为系统支持的枚举值。
func IsValidWeaponDiscipline(value string) bool {
	switch strings.TrimSpace(value) {
	case DisciplineSword, DisciplineSpear, DisciplineStaff:
		return true
	default:
		return false
	}
}

// WeaponDisciplineLabel 返回武器技能流派的简体中文展示名。
func WeaponDisciplineLabel(value string) string {
	switch strings.TrimSpace(value) {
	case DisciplineSword:
		return "剑类"
	case DisciplineSpear:
		return "枪类"
	case DisciplineStaff:
		return "法杖类"
	default:
		return value
	}
}

// MatchesWeaponDiscipline 判断已学会的武器技能是否可在当前装备武器类型下使用。
func MatchesWeaponDiscipline(discipline string, weaponType string) bool {
	discipline = strings.TrimSpace(discipline)
	weaponType = strings.TrimSpace(weaponType)
	if discipline == "" || weaponType == "" {
		return false
	}
	return discipline == weaponType
}
