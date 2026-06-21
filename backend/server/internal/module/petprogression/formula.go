package petprogression

import "math"

// AptitudeMultiplier 返回有效资质公式中的宠物倍率。
func AptitudeMultiplier(profile string) float64 {
	switch profile {
	case AptitudeProfileSpecial:
		return 1.5
	case AptitudeProfileArctic:
		return 3.2
	default:
		return 1.0
	}
}

// EffectiveAptitude 计算单项有效资质：基础×1.1 + 红色×1.2，再乘宠物倍率。
func EffectiveAptitude(baseApt uint32, extraApt uint32, multiplier float64) float64 {
	if multiplier <= 0 {
		multiplier = 1
	}
	return (float64(baseApt)*1.1 + float64(extraApt)*1.2) * multiplier
}

// AutoAllocatedPointsForLevel 按参考表《综合计算器》Q29~Q33 计算系统自动分配点。
// rebirthLevel 首期恒为 0；后续由 pet_rebirth_config 扩展 rebirth 自动点。
func AutoAllocatedPointsForLevel(level uint32, evolutionLevel uint32, _ uint32) AutoAllocatedPoints {
	if level == 0 {
		level = 1
	}
	basePoints := level - 1
	evolutionBonus := uint32(math.Floor(float64(evolutionLevel) * 3.0 / 4.0))
	common := basePoints + evolutionBonus
	return AutoAllocatedPoints{
		HP:   common,
		ATK:  common,
		SPD:  common,
		MANA: common,
		DEF:  basePoints,
	}
}

func totalAllocatedPoints(autoPoints AutoAllocatedPoints, manualPoints ManualAllocatedPoints) (hp, atk, spd, mana, def uint32) {
	return autoPoints.HP + manualPoints.HP,
		autoPoints.ATK + manualPoints.ATK,
		autoPoints.SPD + manualPoints.SPD,
		autoPoints.MANA + manualPoints.MANA,
		autoPoints.DEF + manualPoints.DEF
}

func finalStat(effectiveApt float64, convertRate float64, allocatedPoints uint32, boostPct float64) uint32 {
	if convertRate <= 0 || allocatedPoints == 0 || effectiveApt <= 0 {
		return 0
	}
	value := effectiveApt / convertRate * float64(allocatedPoints) * (1 + boostPct)
	if value <= 0 {
		return 0
	}
	return uint32(math.Floor(value))
}

// FinalCombatStats 按设计文档 §2.3 重算五项战斗属性。
func FinalCombatStats(input ProgressionInput) CombatStats {
	rates := input.ConvertRates
	if rates == (ConvertRates{}) {
		rates = DefaultConvertRates()
	}
	multiplier := AptitudeMultiplier(input.AptitudeProfile)
	autoPoints := AutoAllocatedPointsForLevel(input.Level, input.EvolutionLevel, input.RebirthLevel)
	hpPoints, atkPoints, spdPoints, manaPoints, defPoints := totalAllocatedPoints(autoPoints, input.ManualPoints)
	aptitudes := input.Aptitudes

	return CombatStats{
		HPMax: finalStat(EffectiveAptitude(aptitudes.BaseHPApt, aptitudes.ExtraHPApt, multiplier), rates.HPMax, hpPoints, input.BoostPct),
		ATK:   finalStat(EffectiveAptitude(aptitudes.BaseATKApt, aptitudes.ExtraATKApt, multiplier), rates.ATK, atkPoints, input.BoostPct),
		SPD:   finalStat(EffectiveAptitude(aptitudes.BaseSPDApt, aptitudes.ExtraSPDApt, multiplier), rates.SPD, spdPoints, input.BoostPct),
		MANA:  finalStat(EffectiveAptitude(aptitudes.BaseMANAApt, aptitudes.ExtraMANAApt, multiplier), rates.MANA, manaPoints, input.BoostPct),
		DEF:   finalStat(EffectiveAptitude(aptitudes.BaseDEFApt, aptitudes.ExtraDEFApt, multiplier), rates.DEF, defPoints, input.BoostPct),
	}
}

// RequiredEffectiveAptitudeForStatGain 反推：在给定分配点数与增幅下，获得 targetGain 点战斗属性所需的有效资质。
func RequiredEffectiveAptitudeForStatGain(convertRate float64, allocatedPoints uint32, targetGain uint32, boostPct float64) float64 {
	if targetGain == 0 || allocatedPoints == 0 || convertRate <= 0 {
		return 0
	}
	return float64(targetGain) * convertRate / float64(allocatedPoints) / (1 + boostPct)
}
