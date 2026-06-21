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
		SkillMult:              item.SkillMult,
		SkillCritAdd:           item.SkillCritAdd,
		ArmorBreakPct:          item.ArmorBreakPct,
		VulnerabilityPct:       item.VulnerabilityPct,
		BleedChancePct:         item.BleedChancePct,
		BleedRounds:            item.BleedRounds,
		BleedDamage:            item.BleedDamage,
		SealChancePct:          item.SealChancePct,
		SealPower:              item.SealPower,
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
		ControlPower:           item.ControlPower,
		ControlRounds:          item.ControlRounds,
		ControlStatusID:        item.ControlStatusID,
		PreferredTargetHP:      item.PreferredTargetHP,
		TargetCount:            item.TargetCount,
		IsBasicAttack:          item.IsBasicAttack,
	}
}

// isBasicAttackSkill 判断该技能是否属于普攻，普攻应走「攻击」按钮而非技能列表。
func (d skillDef) isBasicAttackSkill() bool {
	if d.ID == DefaultAttackSkillID {
		return true
	}
	return d.IsBasicAttack
}

// skillIDsForClientSnapshot 过滤普攻技能 ID，客户端通过「攻击」按钮隐式使用 DefaultAttackSkillID。
func skillIDsForClientSnapshot(skillIDs []uint32) []uint32 {
	if len(skillIDs) == 0 {
		return []uint32{}
	}
	filtered := make([]uint32, 0, len(skillIDs))
	for _, skillID := range skillIDs {
		if def, ok := getSkillDef(skillID); ok && def.isBasicAttackSkill() {
			continue
		}
		if skillID == DefaultAttackSkillID {
			continue
		}
		filtered = append(filtered, skillID)
	}
	return filtered
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
