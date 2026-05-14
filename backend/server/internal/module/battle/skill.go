package battle

type skillDef struct {
	ID             uint32
	Name           string
	BaseDamage     int32
	LevelBonus     int32
	FixedDamage    int32
}

var skillCatalog = map[uint32]skillDef{
	DefaultAttackSkillID: {
		ID:         DefaultAttackSkillID,
		Name:       "普通攻击",
		BaseDamage: 5,
		LevelBonus: 1,
	},
	1002: {
		ID:         1002,
		Name:       "火花冲击",
		BaseDamage: 8,
		LevelBonus: 2,
	},
	DefaultEnemySkillID: {
		ID:          DefaultEnemySkillID,
		Name:        "野性撞击",
		FixedDamage: 4,
	},
	90002: {
		ID:          90002,
		Name:        "利爪突袭",
		FixedDamage: 6,
	},
}

func getSkillDef(skillID uint32) (skillDef, bool) {
	def, ok := skillCatalog[skillID]
	return def, ok
}
