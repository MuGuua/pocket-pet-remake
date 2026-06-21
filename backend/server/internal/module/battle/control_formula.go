package battle

const (
	// ControlPowerGuaranteedGap 威力超出抗性达到该差值时，控制命中率为 100%。
	ControlPowerGuaranteedGap uint32 = 50
	// ControlPowerChanceDecayPerPoint 威力-抗性差值每缩小 1 点，命中率下降 2%。
	ControlPowerChanceDecayPerPoint uint32 = 2
)

// CalculateControlChanceByPower 按「控制威力 vs 控制抗性」计算命中概率（0~100）。
// 抗性 >= 威力时完全免疫；差值 >= 50 稳控；差值在 0~49 之间按每点 2% 衰减。
func CalculateControlChanceByPower(power uint32, resist uint32) uint32 {
	if power == 0 || resist >= power {
		return 0
	}
	diff := power - resist
	if diff >= ControlPowerGuaranteedGap {
		return 100
	}
	penalty := (ControlPowerGuaranteedGap - diff) * ControlPowerChanceDecayPerPoint
	if penalty >= 100 {
		return 0
	}
	return 100 - penalty
}

// resolveControlApplyChance 解析控制命中概率：威力 > 0 走抗性对抗；否则概率字段直接生效且无视抗性。
func resolveControlApplyChance(chancePct uint32, power uint32, targetResist uint32) uint32 {
	if power > 0 {
		return CalculateControlChanceByPower(power, targetResist)
	}
	if chancePct == 0 {
		return 0
	}
	if chancePct > 100 {
		return 100
	}
	return chancePct
}
