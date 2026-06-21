package petprogression

// 下列封顶与 pet.DefaultCombatStatCaps / pet_combat_stat_cap 种子保持一致，供公式重算后截断五项战斗属性。
const (
	maxPetHPMax = 1_500_000
	maxPetATK   = 250_000
	maxPetDEF   = 250_000
	maxPetSPD   = 30_000
	maxPetMANA  = 50_000
)

// ClampFormulaCombatStats 把资质公式产出的五项战斗属性限制在玩法封顶内。
func ClampFormulaCombatStats(combat CombatStats) CombatStats {
	return CombatStats{
		HPMax: clampStat(combat.HPMax, maxPetHPMax),
		ATK:   clampStat(combat.ATK, maxPetATK),
		DEF:   clampStat(combat.DEF, maxPetDEF),
		SPD:   clampStat(combat.SPD, maxPetSPD),
		MANA:  clampStat(combat.MANA, maxPetMANA),
	}
}

func clampStat(value uint32, cap uint32) uint32 {
	if cap > 0 && value > cap {
		return cap
	}
	return value
}
