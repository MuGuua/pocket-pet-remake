package petprogression

// BuildProgressionInput 把持久化快照转为公式输入。
func BuildProgressionInput(state ProgressionState, rates ConvertRates) ProgressionInput {
	if rates == (ConvertRates{}) {
		rates = DefaultConvertRates()
	}
	return ProgressionInput{
		Level:           state.Level,
		EvolutionLevel:  state.EvolutionLevel,
		RebirthLevel:    state.RebirthLevel,
		AptitudeProfile: state.AptitudeProfile,
		Aptitudes:       state.Aptitudes,
		ManualPoints:    state.ManualPoints,
		ConvertRates:    rates,
		BoostPct:        0,
	}
}

// RecalculateCombatStats 根据成长快照重算战斗属性。
func RecalculateCombatStats(state ProgressionState, rates ConvertRates) CombatStats {
	return ClampFormulaCombatStats(finalCombatStatsUncapped(state, rates))
}

func finalCombatStatsUncapped(state ProgressionState, rates ConvertRates) CombatStats {
	return FinalCombatStats(BuildProgressionInput(state, rates))
}

// AutoPointsForState 返回当前等级下的系统自动分配点。
func AutoPointsForState(state ProgressionState) AutoAllocatedPoints {
	return AutoAllocatedPointsForLevel(state.Level, state.EvolutionLevel, state.RebirthLevel)
}

// TotalAptitudes 合并基础与红色资质合计，供协议 growth_aptitudes 展示。
func TotalAptitudes(aptitudes GrowthAptitudes) GrowthAptitudes {
	return GrowthAptitudes{
		BaseHPApt:   aptitudes.BaseHPApt + aptitudes.ExtraHPApt,
		BaseATKApt:  aptitudes.BaseATKApt + aptitudes.ExtraATKApt,
		BaseDEFApt:  aptitudes.BaseDEFApt + aptitudes.ExtraDEFApt,
		BaseSPDApt:  aptitudes.BaseSPDApt + aptitudes.ExtraSPDApt,
		BaseMANAApt: aptitudes.BaseMANAApt + aptitudes.ExtraMANAApt,
	}
}

// SplitTotalAptitudes 把总资质拆成 base/extra；extra 为超出模板基础值的部分。
func SplitTotalAptitudes(baseTemplate GrowthAptitudes, total GrowthAptitudes) GrowthAptitudes {
	return GrowthAptitudes{
		BaseHPApt:    baseTemplate.BaseHPApt,
		BaseATKApt:   baseTemplate.BaseATKApt,
		BaseDEFApt:   baseTemplate.BaseDEFApt,
		BaseSPDApt:   baseTemplate.BaseSPDApt,
		BaseMANAApt:  baseTemplate.BaseMANAApt,
		ExtraHPApt:   maxInt32(total.BaseHPApt, baseTemplate.BaseHPApt) - baseTemplate.BaseHPApt,
		ExtraATKApt:  maxInt32(total.BaseATKApt, baseTemplate.BaseATKApt) - baseTemplate.BaseATKApt,
		ExtraDEFApt:  maxInt32(total.BaseDEFApt, baseTemplate.BaseDEFApt) - baseTemplate.BaseDEFApt,
		ExtraSPDApt:  maxInt32(total.BaseSPDApt, baseTemplate.BaseSPDApt) - baseTemplate.BaseSPDApt,
		ExtraMANAApt: maxInt32(total.BaseMANAApt, baseTemplate.BaseMANAApt) - baseTemplate.BaseMANAApt,
	}
}

func maxInt32(a uint32, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}

// GrowthAptitudesFromBaseExtra 构造公式使用的 GrowthAptitudes（含 base/extra 字段）。
func GrowthAptitudesFromBaseExtra(base GrowthAptitudes, extra GrowthAptitudes) GrowthAptitudes {
	return GrowthAptitudes{
		BaseHPApt:    base.BaseHPApt,
		BaseATKApt:   base.BaseATKApt,
		BaseDEFApt:   base.BaseDEFApt,
		BaseSPDApt:   base.BaseSPDApt,
		BaseMANAApt:  base.BaseMANAApt,
		ExtraHPApt:   extra.ExtraHPApt,
		ExtraATKApt:  extra.ExtraATKApt,
		ExtraDEFApt:  extra.ExtraDEFApt,
		ExtraSPDApt:  extra.ExtraSPDApt,
		ExtraMANAApt: extra.ExtraMANAApt,
	}
}

// BaseTemplateFromTotals 把模板五项资质写入 GrowthAptitudes 的 Base 字段。
func BaseTemplateFromTotals(hp, atk, def, spd, mana uint32) GrowthAptitudes {
	return GrowthAptitudes{
		BaseHPApt:   hp,
		BaseATKApt:  atk,
		BaseDEFApt:  def,
		BaseSPDApt:  spd,
		BaseMANAApt: mana,
	}
}

// TotalsFromLegacyAptitudes 兼容仅存储合计资质的旧结构。
func TotalsFromLegacyAptitudes(hp, atk, def, spd, mana uint32) GrowthAptitudes {
	return GrowthAptitudes{
		BaseHPApt:   hp,
		BaseATKApt:  atk,
		BaseDEFApt:  def,
		BaseSPDApt:  spd,
		BaseMANAApt: mana,
	}
}
