package combatcalc

import "math"

// MajorStatModifiers 描述单个主战斗属性在最终结算前的所有加成项。
// 当前先统一支持“基础值 + 加算 + 多段百分比乘算”，后续装备、被动、BUFF
// 继续扩展时，只需要往 FlatBonus / PercentBonuses 里继续累积即可。
type MajorStatModifiers struct {
	Base           uint32
	FlatBonus      int32
	PercentBonuses []int32
}

// FinalMajorStat 按统一公式计算五大主属性最终值：
// (基础值 + 所有加算加成) × ∏(1 + 百分比加成)。
// 为了保持数据库与客户端口径一致，统一在服务端做四舍五入并限制最小值为 0。
func FinalMajorStat(modifiers MajorStatModifiers) uint32 {
	value := float64(int64(modifiers.Base) + int64(modifiers.FlatBonus))
	if value < 0 {
		value = 0
	}
	for _, pct := range modifiers.PercentBonuses {
		value *= 1 + (float64(pct) / 100.0)
	}
	if value <= 0 {
		return 0
	}
	return uint32(math.Round(value))
}
