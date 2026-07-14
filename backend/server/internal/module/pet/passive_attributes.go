package pet

import (
	"regexp"
	"strings"

	"pocket-pet-remake/server/internal/module/skill"
)

// systemNameBBCodePattern 只剥离后台名称编辑器允许写入的展示标签。
// 技能效果仍由稳定配置决定；此前按名称前缀兼容的旧数据也不能因刷色而失效。
var systemNameBBCodePattern = regexp.MustCompile(`(?i)\[/?(?:b|i|u|color(?:=[^\]]+)?)\]`)

// applyPersistentPassiveBonusesToPets 把会常驻参与面板计算的被动技能效果应用到宠物快照。
// 这里只处理“应该直接反映到属性面板”的加成，吸血/反伤等战斗期效果仍留在 battle 运行时处理。
func applyPersistentPassiveBonusesToPets(items []Pet, resolver func(uint32) (skill.RuntimeDefinition, bool)) {
	for index := range items {
		applyPersistentPassiveBonusesToPet(&items[index], resolver)
	}
}

// applyPersistentPassiveBonusesToLineupPets 把同一套常驻属性加成应用到编队快照，
// 确保进入战斗前服务端编队数据与宠物面板展示保持一致。
func applyPersistentPassiveBonusesToLineupPets(items []LineupPet, resolver func(uint32) (skill.RuntimeDefinition, bool)) {
	for index := range items {
		applyPersistentPassiveBonusesToLineupPet(&items[index], resolver)
	}
}

// applyPersistentPassiveBonusesToPet 只负责把“永久参与属性计算”的被动技能折算进宠物最终快照。
func applyPersistentPassiveBonusesToPet(item *Pet, resolver func(uint32) (skill.RuntimeDefinition, bool)) bool {
	if item == nil {
		return false
	}
	ResolvePetBattleSkills(item)
	if len(item.SkillIDs) == 0 || resolver == nil {
		return false
	}

	profile, applied := collectPersistentPassiveProfile(item.SkillIDs, resolver)
	if !applied {
		return false
	}
	applyPersistentProfileToPet(item, profile)
	return true
}

// applyPersistentPassiveBonusesToLineupPet 把永久属性型被动折算进编队快照，
// 这样 battle 模块拿到的基础属性已经是最终值，不会和属性面板脱节。
func applyPersistentPassiveBonusesToLineupPet(item *LineupPet, resolver func(uint32) (skill.RuntimeDefinition, bool)) bool {
	if item == nil || len(item.SkillIDs) == 0 || resolver == nil {
		return false
	}

	profile, applied := collectPersistentPassiveProfile(item.SkillIDs, resolver)
	if !applied {
		return false
	}
	applyPersistentProfileToLineupPet(item, profile)
	return true
}

type persistentPassiveProfile struct {
	hpMaxPct           int32
	hpMaxFlat          int32
	atkPct             uint32
	atkFlat            int32
	spdPct             uint32
	spdFlat            int32
	manaPct            uint32
	manaFlat           int32
	critRatePct        uint32
	critDmgPct         uint32
	physicalResistPct  uint32
	skillResistPct     uint32
	allStatusResistPct uint32
}

// collectPersistentPassiveProfile 从技能库里抽取会直接进入最终属性面板的被动效果。
func collectPersistentPassiveProfile(
	skillIDs []uint32,
	resolver func(uint32) (skill.RuntimeDefinition, bool),
) (persistentPassiveProfile, bool) {
	var profile persistentPassiveProfile
	applied := false
	for _, skillID := range skillIDs {
		def, ok := resolver(skillID)
		if !ok || !isPersistentPassiveSkill(def) {
			continue
		}
		if applyExplicitPersistentPassiveProfile(def, &profile) {
			applied = true
			continue
		}
		plainSkillName := systemNameBBCodePattern.ReplaceAllString(def.SkillName, "")
		switch {
		case strings.HasPrefix(plainSkillName, "强壮"):
			profile.hpMaxPct += def.HealPct
			applied = true
		case strings.HasPrefix(plainSkillName, "强力"):
			profile.atkPct += uint32(maxInt32(def.AttackPct, 0))
			applied = true
		case strings.HasPrefix(plainSkillName, "迅捷"):
			profile.spdPct += uint32(maxInt32(def.SpeedPct, 0))
			applied = true
		case strings.HasPrefix(plainSkillName, "魔心"):
			profile.manaPct += uint32(maxInt32(def.ManaPct, 0))
			applied = true
		case strings.HasPrefix(plainSkillName, "致命"):
			profile.critRatePct += def.CritBoostPct
			applied = true
		case strings.HasPrefix(plainSkillName, "暴伤"):
			profile.critDmgPct += def.SkillCritAdd
			applied = true
		case strings.HasPrefix(plainSkillName, "厚甲"):
			profile.physicalResistPct += uint32(maxInt32(def.DefensePct, 0))
			applied = true
		case strings.HasPrefix(plainSkillName, "坚韧"):
			profile.allStatusResistPct += def.SealPower
			applied = true
		case strings.HasPrefix(plainSkillName, "结界"):
			profile.skillResistPct += def.VulnerabilityApplyPct
			applied = true
		}
	}
	return profile, applied
}

// applyExplicitPersistentPassiveProfile 优先读取后台显式配置的永久属性加成；
// 只有未配置时，调用方才会回退到旧的技能名前缀兼容规则。
func applyExplicitPersistentPassiveProfile(def skill.RuntimeDefinition, profile *persistentPassiveProfile) bool {
	if profile == nil || def.PassiveAttrKey == "" || def.PassiveAttrMode == "" || def.PassiveAttrValue <= 0 {
		return false
	}

	switch def.PassiveAttrKey {
	case skill.PassiveAttrKeyHPMax:
		if def.PassiveAttrMode == skill.PassiveAttrModePercent {
			profile.hpMaxPct += def.PassiveAttrValue
			return true
		}
		profile.hpMaxFlat += def.PassiveAttrValue
		return true
	case skill.PassiveAttrKeyATK:
		if def.PassiveAttrMode == skill.PassiveAttrModePercent {
			profile.atkPct += uint32(def.PassiveAttrValue)
			return true
		}
		profile.atkFlat += def.PassiveAttrValue
		return true
	case skill.PassiveAttrKeySPD:
		if def.PassiveAttrMode == skill.PassiveAttrModePercent {
			profile.spdPct += uint32(def.PassiveAttrValue)
			return true
		}
		profile.spdFlat += def.PassiveAttrValue
		return true
	case skill.PassiveAttrKeyMana:
		if def.PassiveAttrMode == skill.PassiveAttrModePercent {
			profile.manaPct += uint32(def.PassiveAttrValue)
			return true
		}
		profile.manaFlat += def.PassiveAttrValue
		return true
	case skill.PassiveAttrKeyCritRatePct:
		profile.critRatePct += uint32(def.PassiveAttrValue)
		return true
	case skill.PassiveAttrKeyCritDmgPct:
		profile.critDmgPct += uint32(def.PassiveAttrValue)
		return true
	case skill.PassiveAttrKeyPhysicalResistPct:
		profile.physicalResistPct += uint32(def.PassiveAttrValue)
		return true
	case skill.PassiveAttrKeySkillResistPct:
		profile.skillResistPct += uint32(def.PassiveAttrValue)
		return true
	case skill.PassiveAttrKeyAllStatusResistPct:
		profile.allStatusResistPct += uint32(def.PassiveAttrValue)
		return true
	default:
		return false
	}
}

// isPersistentPassiveSkill 保持与 battle 模块一致：优先读 activation_mode，老数据再回退旧推断规则。
func isPersistentPassiveSkill(def skill.RuntimeDefinition) bool {
	if def.ActivationMode == skill.ActivationModePassive {
		return true
	}
	return def.SkillType == "support" && def.EnergyCost == 0
}

// applyPersistentProfileToPet 把永久属性型被动折算到宠物快照，供列表/详情/推送直接展示最终值。
func applyPersistentProfileToPet(item *Pet, profile persistentPassiveProfile) {
	applyHPBonusToPet(item, profile.hpMaxPct)
	applyFlatBonusToPet(&item.HPMax, &item.HP, profile.hpMaxFlat)
	item.ATK = applyPctBonus(item.ATK, profile.atkPct)
	item.ATK = applyFlatBonus(item.ATK, profile.atkFlat)
	item.SPD = applyPctBonus(item.SPD, profile.spdPct)
	item.SPD = applyFlatBonus(item.SPD, profile.spdFlat)
	item.MANA = applyPctBonus(item.MANA, profile.manaPct)
	item.MANA = applyFlatBonus(item.MANA, profile.manaFlat)
	item.CritRatePct += profile.critRatePct
	item.CritDmgPct += profile.critDmgPct
	item.PhysicalResistPct += profile.physicalResistPct
	item.SkillResistPct += profile.skillResistPct
	addAllStatusResistToPet(item, profile.allStatusResistPct)
}

// applyPersistentProfileToLineupPet 对编队快照应用同样的永久属性被动，保证战斗入口读取到的就是最终面板值。
func applyPersistentProfileToLineupPet(item *LineupPet, profile persistentPassiveProfile) {
	applyHPBonusToLineupPet(item, profile.hpMaxPct)
	applyFlatBonusToLineupPet(&item.HPMax, &item.HP, profile.hpMaxFlat)
	item.ATK = applyPctBonus(item.ATK, profile.atkPct)
	item.ATK = applyFlatBonus(item.ATK, profile.atkFlat)
	item.SPD = applyPctBonus(item.SPD, profile.spdPct)
	item.SPD = applyFlatBonus(item.SPD, profile.spdFlat)
	item.MANA = applyPctBonus(item.MANA, profile.manaPct)
	item.MANA = applyFlatBonus(item.MANA, profile.manaFlat)
	item.CritRatePct += profile.critRatePct
	item.CritDmgPct += profile.critDmgPct
	item.PhysicalResistPct += profile.physicalResistPct
	item.SkillResistPct += profile.skillResistPct
	addAllStatusResistToLineupPet(item, profile.allStatusResistPct)
}

// applyHPBonusToPet 按比例抬升生命上限，并同步当前生命百分比。
func applyHPBonusToPet(item *Pet, bonusPct int32) {
	if item == nil || bonusPct <= 0 || item.HPMax == 0 {
		return
	}
	oldMax := item.HPMax
	newMax := uint32(int64(oldMax) * int64(100+bonusPct) / 100)
	if newMax <= oldMax {
		return
	}
	if item.HP > 0 {
		item.HP = uint32(int64(item.HP) * int64(newMax) / int64(oldMax))
	}
	item.HPMax = newMax
}

// applyHPBonusToLineupPet 对编队快照执行同样的生命同步规则。
func applyHPBonusToLineupPet(item *LineupPet, bonusPct int32) {
	if item == nil || bonusPct <= 0 || item.HPMax == 0 {
		return
	}
	oldMax := item.HPMax
	newMax := uint32(int64(oldMax) * int64(100+bonusPct) / 100)
	if newMax <= oldMax {
		return
	}
	if item.HP > 0 {
		item.HP = uint32(int64(item.HP) * int64(newMax) / int64(oldMax))
	}
	item.HPMax = newMax
}

// addAllStatusResistToPet 为宠物面板上的全部状态抗性叠加固定点数。
func addAllStatusResistToPet(item *Pet, points uint32) {
	if item == nil || points == 0 {
		return
	}
	item.ConfusionResistPct += points
	item.SleepResistPct += points
	item.ParalysisResistPct += points
	item.SealResistPct += points
	item.CurseResistPct += points
}

// addAllStatusResistToLineupPet 为编队快照同步增加状态抗性。
func addAllStatusResistToLineupPet(item *LineupPet, points uint32) {
	if item == nil || points == 0 {
		return
	}
	item.ConfusionResistPct += points
	item.SleepResistPct += points
	item.ParalysisResistPct += points
	item.SealResistPct += points
	item.CurseResistPct += points
}

// applyPctBonus 使用“基础值 * (100 + 百分比) / 100”的口径计算最终属性。
func applyPctBonus(base uint32, bonusPct uint32) uint32 {
	if base == 0 || bonusPct == 0 {
		return base
	}
	return uint32(int64(base) * int64(100+bonusPct) / 100)
}

// applyFlatBonus 对攻击、速度、法力这类基础属性做固定值抬升。
func applyFlatBonus(base uint32, bonus int32) uint32 {
	if bonus <= 0 {
		return base
	}
	return base + uint32(bonus)
}

// applyFlatBonusToPet 为宠物生命做固定值抬升，同时维持当前生命百分比。
func applyFlatBonusToPet(maxValue *uint32, currentValue *uint32, bonus int32) {
	if maxValue == nil || currentValue == nil || bonus <= 0 {
		return
	}
	oldMax := *maxValue
	newMax := oldMax + uint32(bonus)
	if oldMax > 0 && *currentValue > 0 {
		*currentValue = uint32(int64(*currentValue) * int64(newMax) / int64(oldMax))
	}
	*maxValue = newMax
}

// applyFlatBonusToLineupPet 对编队快照执行同样的生命固定值抬升规则。
func applyFlatBonusToLineupPet(maxValue *uint32, currentValue *uint32, bonus int32) {
	applyFlatBonusToPet(maxValue, currentValue, bonus)
}

func maxInt32(value int32, minimum int32) int32 {
	if value < minimum {
		return minimum
	}
	return value
}
