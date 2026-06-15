package battle

import "pocket-pet-remake/server/internal/module/skill"

var runtimeSkillResolver func(uint32) (skill.RuntimeDefinition, bool)

// SetRuntimeSkillResolver 注册数据库驱动的技能模板解析器。
// 生产环境在 bootstrap 阶段注入；测试未注入时会回退到内置 skillCatalog。
func SetRuntimeSkillResolver(resolver func(uint32) (skill.RuntimeDefinition, bool)) {
	runtimeSkillResolver = resolver
}

func getSkillDef(skillID uint32) (skillDef, bool) {
	if runtimeSkillResolver != nil {
		if runtimeDef, ok := runtimeSkillResolver(skillID); ok {
			return runtimeDefinitionToSkillDef(runtimeDef), true
		}
	}
	def, ok := skillCatalog[skillID]
	return def, ok
}

func runtimeDefinitionToSkillDef(item skill.RuntimeDefinition) skillDef {
	return skillDef{
		ID:                     item.SkillID,
		Name:                   item.SkillName,
		TargetRule:             targetRuleFromProtocolName(item.TargetType),
		AnimationKey:           item.AnimationKey,
		SkillVisualID:          item.SkillVisualID,
		CastColor:              item.CastColor,
		ImpactColor:            item.ImpactColor,
		Projectile:             item.Projectile,
		IsSkillAttack:          item.IsSkillAttack,
		EnergyCost:             item.EnergyCost,
		AttackPct:              item.AttackPct,
		ManaPct:                item.ManaPct,
		DefensePct:             item.DefensePct,
		SpeedPct:               item.SpeedPct,
		TargetCurrentHPPct:     item.TargetCurrentHPPct,
		FixedDamage:            item.FixedDamage,
		HealPct:                item.HealPct,
		FixedHeal:              item.FixedHeal,
		AllowCrit:              item.AllowCrit,
		IgnoreDefense:          item.IgnoreDefense,
		ArmorBreakPct:          item.ArmorBreakPct,
		VulnerabilityPct:       item.VulnerabilityPct,
		BleedChancePct:         item.BleedChancePct,
		BleedRounds:            item.BleedRounds,
		BleedDamage:            item.BleedDamage,
		SealChancePct:          item.SealChancePct,
		SealRounds:             item.SealRounds,
		VulnerabilityChancePct: item.VulnerabilityChancePct,
		VulnerabilityRounds:    item.VulnerabilityRounds,
		VulnerabilityApplyPct:  item.VulnerabilityApplyPct,
		ArmorBreakChancePct:    item.ArmorBreakChancePct,
		ArmorBreakRounds:       item.ArmorBreakRounds,
		SlowChancePct:          item.SlowChancePct,
		SlowRounds:             item.SlowRounds,
		SlowMultiplierPct:      item.SlowMultiplierPct,
		CritBoostRounds:        item.CritBoostRounds,
		CritBoostPct:           item.CritBoostPct,
		CurseChancePct:         item.CurseChancePct,
		CurseRounds:            item.CurseRounds,
		CurseDamage:            item.CurseDamage,
		CurseManaPct:           item.CurseManaPct,
		ControlChancePct:       item.ControlChancePct,
		ControlRounds:          item.ControlRounds,
		ControlStatusID:        item.ControlStatusID,
		PreferredTargetHP:      item.PreferredTargetHP,
		TargetCount:            item.TargetCount,
	}
}

func targetRuleFromProtocolName(name string) skillTargetRule {
	switch name {
	case "ally_single":
		return targetAllySingle
	case "enemy_all":
		return targetEnemyAll
	case "enemy_multi":
		return targetEnemyMulti
	default:
		return targetEnemySingle
	}
}
