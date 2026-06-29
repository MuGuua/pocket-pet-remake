package skill

import "strings"

// 系统技能分类常量，供后台录入、装备引用校验与运营展示复用。
const (
	CategoryCommon    = "common"
	CategoryCharacter = "character"
	CategoryWeapon    = "weapon"
	CategoryPet       = "pet"
	CategoryMonster   = "monster"
)

// PlayerFacingCategories 返回面向玩家/运营侧展示的核心技能分类（不含 common 与 monster）。
func PlayerFacingCategories() []string {
	return []string{CategoryCharacter, CategoryWeapon, CategoryPet}
}

// IsValidCategory 判断技能分类是否为系统支持的枚举值。
func IsValidCategory(value string) bool {
	switch strings.TrimSpace(value) {
	case CategoryCommon, CategoryCharacter, CategoryWeapon, CategoryPet, CategoryMonster:
		return true
	default:
		return false
	}
}
