package battle

import "strings"

// applySkillPassives 根据宠物已学会的技能模板，叠加战斗被动效果。
// 返回 true 表示至少应用了一个数据库驱动的被动技能。
func applySkillPassives(actor *actorRuntime) bool {
	if actor == nil || len(actor.skillIDs) == 0 {
		return false
	}
	applied := false
	for _, skillID := range actor.skillIDs {
		def, ok := getSkillDef(skillID)
		if !ok || !def.isPassiveSkill() {
			continue
		}
		applyPassiveSkillDef(actor, def)
		applied = true
	}
	return applied
}

// applyPassiveSkillDef 将单条被动技能模板映射到 actorRuntime 修饰字段。
func applyPassiveSkillDef(actor *actorRuntime, def skillDef) {
	name := def.Name
	switch {
	case strings.HasPrefix(name, "强壮"):
		applyMaxHPBonus(actor, def.HealPct)
	case strings.HasPrefix(name, "嗜血"):
		actor.lifestealPct += def.CritBoostPct
	case strings.HasPrefix(name, "连击"):
		actor.comboPct += def.BleedChancePct
	case strings.HasPrefix(name, "涅槃"):
		actor.revivePct += def.CurseChancePct
		if def.HealPct > 0 {
			actor.reviveHPPct += uint32(def.HealPct)
		}
	case strings.HasPrefix(name, "反噬"):
		actor.counterPct += def.ControlChancePct
	case strings.HasPrefix(name, "致命"):
		actor.critRatePct += def.CritBoostPct
	case strings.HasPrefix(name, "暴伤"):
		actor.critDmgPct += def.SkillCritAdd
	case strings.HasPrefix(name, "强力"):
		if actor.attackMultiplierPct < 100 {
			actor.attackMultiplierPct = 100
		}
		actor.attackMultiplierPct += uint32(def.AttackPct)
	case strings.HasPrefix(name, "迅捷"):
		if actor.speedMultiplierPct < 100 {
			actor.speedMultiplierPct = 100
		}
		actor.speedMultiplierPct += uint32(def.SpeedPct)
	case strings.HasPrefix(name, "魔心"):
		if actor.manaMultiplierPct < 100 {
			actor.manaMultiplierPct = 100
		}
		actor.manaMultiplierPct += uint32(def.ManaPct)
	case strings.HasPrefix(name, "厚甲"):
		actor.physicalResistPct += uint32(def.DefensePct)
	case strings.HasPrefix(name, "坚韧"):
		addAllStatusResist(actor, def.SealPower)
	case strings.HasPrefix(name, "结界"):
		actor.skillResistPct += def.VulnerabilityApplyPct
	case strings.HasPrefix(name, "刺甲"):
		actor.reflectPct += def.ArmorBreakPct
	}
}

// applyMaxHPBonus 按百分比提升最大生命，并按比例同步当前生命。
func applyMaxHPBonus(actor *actorRuntime, bonusPct int32) {
	if actor == nil || bonusPct <= 0 || actor.hpMax == 0 {
		return
	}
	oldMax := actor.hpMax
	newMax := uint32(int64(oldMax) * int64(100+bonusPct) / 100)
	if newMax <= oldMax {
		return
	}
	if actor.hp > 0 {
		actor.hp = uint32(int64(actor.hp) * int64(newMax) / int64(oldMax))
	}
	actor.hpMax = newMax
}

// addAllStatusResist 为全部状态抗性增加固定点数（坚韧系被动）。
func addAllStatusResist(actor *actorRuntime, points uint32) {
	if actor == nil || points == 0 {
		return
	}
	actor.confusionResistPct += points
	actor.sleepResistPct += points
	actor.paralysisResistPct += points
	actor.sealResistPct += points
	actor.curseResistPct += points
}
