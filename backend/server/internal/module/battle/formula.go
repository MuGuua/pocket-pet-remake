package battle

import "math"

const (
	// minCritMultiplierPct 为暴击后的最低爆伤倍率。《战斗系统规则》规定抗爆伤最多把实际爆伤压到 150%。
	minCritMultiplierPct int64 = 150
	// maxCritDmgPct 为爆伤属性与暴击倍率的共同上限，对应规则中的 2000%。
	maxCritDmgPct int64 = 2000
)

// pocketDamageInput 对应《口袋伤害计算新表》中的进攻/防守参数。
type pocketDamageInput struct {
	AttackerPanel        int64
	DefenderPanel        int64
	SkillMult            uint32
	AttackScalePct       int32
	CritDmg              uint32
	SkillCritAdd         uint32
	AntiCrit             uint32
	RevSkill             uint32
	SkillRes             uint32
	Guard                uint32
	TalentDmgPct         uint32
	TalentReducePct      uint32
	ElementAdvPct        uint32
	ElementPenaltyPct    uint32
	AntiClassPct         uint32
	IsAOE                bool
	ElementAdvantaged    bool
	ElementDisadvantaged bool
	// IsCrit 表示本次结算是否通过暴击判定；未暴击时倍率固定为 100，抗爆伤不参与计算。
	IsCrit bool
}

// effectiveStats 是公式层与部分战斗分支共用的快照；伤害面板取 Attack 字段。
type effectiveStats struct {
	UnitClass                uint32
	Attack                   int32
	Defense                  int32
	Speed                    int32
	Mana                     int32
	CurrentHP                int32
	MaxHP                    int32
	HitPct                   uint32
	DodgePct                 uint32
	CritRatePct              uint32
	CritDmgPct               uint32
	CritResistPct            uint32
	CritDmgResistPct         uint32
	ArmorBroken              bool
	VulnerabilityPct         uint32
	PhysicalResistPct        uint32
	SkillResistPct           uint32
	ReverseSkillResistPct    uint32
	ReversePhysicalResistPct uint32
	GenericShieldPct         uint32
	CharacterResistPct       uint32
	PetResistPct             uint32
	MercenaryResistPct       uint32
	Guard                    uint32
	TalentDmgPct             uint32
	TalentReducePct          uint32
	ElementAdvPct            uint32
	ElementPenaltyPct        uint32
}

func (a *actorRuntime) effectiveStats() effectiveStats {
	attack := applyPercentModifier(int32(a.atk)+a.attackFlatBonus, a.attackMultiplierPct, a.globalMultiplierPct)
	defense := applyPercentModifier(int32(a.def)+a.defenseFlatBonus, a.defenseMultiplierPct, a.globalMultiplierPct)
	speedMultiplierPct := a.speedMultiplierPct
	if speedMultiplierPct == 0 {
		speedMultiplierPct = 100
	}
	derived := a.statusDerived
	if derived.SpeedMultiplierPct > 0 && derived.SpeedMultiplierPct < 100 {
		speedMultiplierPct = speedMultiplierPct * derived.SpeedMultiplierPct / 100
	}
	speed := applyPercentModifier(int32(a.spd)+a.speedFlatBonus, speedMultiplierPct, a.globalMultiplierPct)
	mana := applyPercentModifier(int32(a.mana)+a.manaFlatBonus, a.manaMultiplierPct, a.globalMultiplierPct)
	guard := a.guard
	if guard == 0 {
		guard = uint32(maxInt32(defense, 0))
	}

	hitPct := a.hitPct
	if derived.HitPenalty >= hitPct {
		hitPct = 0
	} else {
		hitPct -= derived.HitPenalty
	}
	dodgePct := a.dodgeRatePct + a.dodgePct
	if derived.DodgePenalty >= dodgePct {
		dodgePct = 0
	} else {
		dodgePct -= derived.DodgePenalty
	}
	critDmgPct := a.critDmgPct
	if derived.CritDmgPenalty >= critDmgPct {
		critDmgPct = 100
	} else {
		critDmgPct -= derived.CritDmgPenalty
	}
	critDmgPct = clampCritDmgPct(critDmgPct)

	physicalResist := a.physicalResistPct
	if derived.PhysicalResistPenalty >= physicalResist {
		physicalResist = 0
	} else {
		physicalResist -= derived.PhysicalResistPenalty
	}
	skillResist := a.skillResistPct
	if derived.SkillResistPenalty >= skillResist {
		skillResist = 0
	} else {
		skillResist -= derived.SkillResistPenalty
	}
	reversePhysical := a.reversePhysicalResistPct
	if derived.ReversePhysicalResistPenalty >= reversePhysical {
		reversePhysical = 0
	} else {
		reversePhysical -= derived.ReversePhysicalResistPenalty
	}
	reverseSkill := a.reverseSkillResistPct
	if derived.ReverseSkillResistPenalty >= reverseSkill {
		reverseSkill = 0
	} else {
		reverseSkill -= derived.ReverseSkillResistPenalty
	}

	critDmgResistPct := a.critDmgResistPct

	// 麻痹等「属性降至 X%」类的整体缩放。按规则作用于物理抗性、技能抗性、抗爆伤、命中、闪避，
	// 必须在固定点数扣减之后应用，才是「原有效属性 × 50%」的口径。
	// 具体削弱哪几项由状态掩码决定：规则给出默认范围，技能可用更小的掩码只削减其中几项。
	if multiplier := derived.PropertyMultiplierPct; multiplier > 0 && multiplier < 100 {
		mask := derived.PropertyScaleMask
		if mask == 0 {
			// 未显式配置掩码时按规则默认范围处理，避免遗漏削弱。
			mask = paralysisScaleAllResist
		}
		if mask&paralysisScaleHit != 0 {
			hitPct = scalePercentValue(hitPct, multiplier)
		}
		if mask&paralysisScaleDodge != 0 {
			dodgePct = scalePercentValue(dodgePct, multiplier)
		}
		if mask&paralysisScaleCritDmgResist != 0 {
			critDmgResistPct = scalePercentValue(critDmgResistPct, multiplier)
		}
		if mask&paralysisScalePhysicalResist != 0 {
			physicalResist = scalePercentValue(physicalResist, multiplier)
		}
		if mask&paralysisScaleSkillResist != 0 {
			skillResist = scalePercentValue(skillResist, multiplier)
		}
		// 攻击者自身的爆伤同样属于「抗性」体系里的对应项，麻痹时一并削弱。
		if mask&paralysisScaleCritDmgResist != 0 {
			critDmgPct = scalePercentValue(critDmgPct, multiplier)
		}
	}

	return effectiveStats{
		UnitClass:                a.unitClass,
		Attack:                   attack,
		Defense:                  defense,
		Speed:                    speed,
		Mana:                     mana,
		CurrentHP:                int32(a.hp),
		MaxHP:                    int32(a.hpMax),
		HitPct:                   hitPct,
		DodgePct:                 dodgePct,
		CritRatePct:              a.critRatePct + derived.CritRateBonusPct,
		CritDmgPct:               critDmgPct,
		CritResistPct:            a.critResistPct,
		CritDmgResistPct:         critDmgResistPct,
		ArmorBroken:              derived.ArmorBroken,
		VulnerabilityPct:         derived.VulnerabilityPct,
		PhysicalResistPct:        physicalResist,
		SkillResistPct:           skillResist,
		ReversePhysicalResistPct: reversePhysical,
		ReverseSkillResistPct:    reverseSkill,
		GenericShieldPct:         a.genericShieldPct,
		CharacterResistPct:       a.characterResistPct,
		PetResistPct:             a.petResistPct,
		MercenaryResistPct:       a.mercenaryResistPct,
		Guard:                    guard,
		TalentDmgPct:             a.talentDmgPct,
		TalentReducePct:          a.talentReducePct,
		ElementAdvPct:            a.elementAdvPct,
		ElementPenaltyPct:        a.elementPenaltyPct,
	}
}

// pocketAttackerPanel 为口袋伤害公式选择进攻面板：攻击 > 法力 > 速度。
func pocketAttackerPanel(attackerStats effectiveStats, skill skillDef) int64 {
	if skill.AttackPct > 0 || (skill.SkillMult > 0 && skill.ManaPct == 0 && skill.SpeedPct == 0) {
		return int64(attackerStats.Attack)
	}
	if skill.ManaPct > 0 {
		return int64(attackerStats.Mana)
	}
	if skill.SpeedPct > 0 {
		return int64(attackerStats.Speed)
	}
	return int64(attackerStats.Attack)
}

func buildPocketDamageInput(attacker *actorRuntime, target *actorRuntime, skill skillDef) pocketDamageInput {
	attackerStats := attacker.effectiveStats()
	targetStats := target.effectiveStats()
	antiClassPct := target.petResistPct
	switch attacker.unitClass {
	case ActorUnitClassCharacter:
		antiClassPct = target.characterResistPct
	case ActorUnitClassMercenary:
		antiClassPct = target.mercenaryResistPct
	case ActorUnitClassPet:
		antiClassPct = target.petResistPct
	}
	talentReducePct := target.talentReducePct
	if talentReducePct == 0 {
		talentReducePct = target.genericShieldPct
	}
	elementAdvantaged := attackerStats.ElementAdvPct > 0
	elementDisadvantaged := !elementAdvantaged && targetStats.ElementPenaltyPct > 0
	isAOE := skill.TargetRule == targetEnemyAll || skill.TargetRule == targetEnemyMulti
	attackerPanel := pocketAttackerPanel(attackerStats, skill)
	return pocketDamageInput{
		AttackerPanel:  attackerPanel,
		DefenderPanel:  int64(targetStats.Defense),
		SkillMult:      skill.SkillMult,
		AttackScalePct: skill.AttackPct,
		// 这三项必须取 effectiveStats 的值，而不是 actorRuntime 的原始字段。
		// effectiveStats 已经把状态带来的抗性/爆伤削减算进去了（麻痹、圣光忏悔、魅惑之风、
		// 恶魔之力、幻影闪击等），直接读原始字段会让这些状态在伤害公式里完全失效。
		CritDmg:              attackerStats.CritDmgPct,
		SkillCritAdd:         skill.SkillCritAdd,
		AntiCrit:             targetStats.CritDmgResistPct,
		RevSkill:             attackerStats.ReverseSkillResistPct,
		SkillRes:             targetStats.SkillResistPct,
		Guard:                targetStats.Guard,
		TalentDmgPct:         attackerStats.TalentDmgPct,
		TalentReducePct:      talentReducePct,
		ElementAdvPct:        attackerStats.ElementAdvPct,
		ElementPenaltyPct:    targetStats.ElementPenaltyPct,
		AntiClassPct:         antiClassPct,
		IsAOE:                isAOE,
		ElementAdvantaged:    elementAdvantaged,
		ElementDisadvantaged: elementDisadvantaged,
	}
}

func (skill skillDef) usesPocketPanelScaling() bool {
	return skill.SkillMult > 0 || skill.AttackPct > 0
}

// isPureFixedDamage 判断技能是否只结算固定伤害、不参与面板倍率。
// 《战斗系统规则》要求固定伤害默认不暴击，这条判定在伤害入口与暴击入口共用。
func (skill skillDef) isPureFixedDamage() bool {
	return !skill.usesPocketPanelScaling() && skill.FixedDamage > 0
}

func scaledPanelBase(input pocketDamageInput) int64 {
	if input.SkillMult > 0 {
		return input.AttackerPanel * int64(input.SkillMult)
	}
	if input.AttackScalePct > 0 {
		return input.AttackerPanel * int64(input.AttackScalePct) / 100
	}
	return 0
}

// critMultiplierPct 计算本次伤害实际使用的暴击倍率（百分比）。
// 规则来自《战斗系统规则》：未暴击时倍率固定为 100；暴击时为「攻击者爆伤 + 技能附加爆伤 - 目标抗爆伤」，
// 且最低不得低于 150%，避免抗爆伤把暴击收益压到比普攻还低。
func critMultiplierPct(input pocketDamageInput) int64 {
	if !input.IsCrit {
		return 100
	}
	critChain := int64(input.CritDmg) + int64(input.SkillCritAdd) - int64(input.AntiCrit)
	if critChain < minCritMultiplierPct {
		return minCritMultiplierPct
	}
	if critChain > maxCritDmgPct {
		return maxCritDmgPct
	}
	return critChain
}

func calculatePocketDamageNumerator(input pocketDamageInput) int64 {
	panelBase := scaledPanelBase(input)
	if panelBase <= 0 {
		return 0
	}
	critChain := critMultiplierPct(input)
	skillResFactor := int64(100) - (int64(input.SkillRes) - int64(input.RevSkill))
	return panelBase*critChain/100*skillResFactor/100 - input.DefenderPanel
}

func calculatePocketDamageDenominator(guard uint32, isAOE bool) float64 {
	if isAOE {
		return float64(guard)*0.01 + 1.0
	}
	return float64(guard)*0.001 + 1.0
}

func calculatePocketDamageMultiplier(input pocketDamageInput) float64 {
	talentReduce := float64(input.TalentReducePct) / 100.0
	if talentReduce > 1 {
		talentReduce = 1
	}
	multiplier := (1.0 + float64(input.TalentDmgPct)/100.0) * (1.0 - talentReduce)
	switch {
	case input.ElementAdvantaged:
		multiplier *= 1.0 + float64(input.ElementAdvPct)/100.0
	case input.ElementDisadvantaged:
		penalty := float64(input.ElementPenaltyPct) / 100.0
		if penalty > 1 {
			penalty = 1
		}
		multiplier *= 1.0 - penalty
	}
	antiClass := float64(input.AntiClassPct) / 100.0
	if antiClass > 1 {
		antiClass = 1
	}
	multiplier *= 1.0 - antiClass
	return multiplier * 0.5
}

func calculatePocketDamage(input pocketDamageInput) int32 {
	numerator := calculatePocketDamageNumerator(input)
	if numerator <= 0 {
		return 1
	}
	denominator := calculatePocketDamageDenominator(input.Guard, input.IsAOE)
	multiplier := calculatePocketDamageMultiplier(input)
	damage := int64(math.Round(float64(numerator) / denominator * multiplier))
	if damage < 1 {
		return 1
	}
	return int32(damage)
}

func applyPercentModifier(base int32, specificMultiplierPct uint32, globalMultiplierPct uint32) int32 {
	specific := specificMultiplierPct
	if specific == 0 {
		specific = 100
	}
	global := globalMultiplierPct
	if global == 0 {
		global = 100
	}
	value := int32(math.Round(float64(base) * float64(specific) / 100.0 * float64(global) / 100.0))
	if value < 0 {
		return 0
	}
	return value
}

// scalePercentValue 按百分比缩放一个百分比类属性（命中、闪避、抗性、爆伤等）。
// 用于「属性降至 50%」这类乘法效果；结果向下取整，避免削弱效果被四舍五入放大。
func scalePercentValue(value uint32, multiplierPct uint32) uint32 {
	if multiplierPct == 0 || multiplierPct >= 100 {
		return value
	}
	return value * multiplierPct / 100
}

func clampCritRatePct(critRatePct uint32) uint32 {
	if critRatePct > 100 {
		return 100
	}
	return critRatePct
}

// clampCritDmgPct 限制爆伤属性本身的取值范围；100 表示没有爆伤加成，2000 为上限。
func clampCritDmgPct(critDmgPct uint32) uint32 {
	if critDmgPct < 100 {
		return 100
	}
	if critDmgPct > uint32(maxCritDmgPct) {
		return uint32(maxCritDmgPct)
	}
	return critDmgPct
}

func maxUint32(left uint32, right uint32) uint32 {
	if left > right {
		return left
	}
	return right
}

func calculateHealAmount(caster effectiveStats, skill skillDef) int32 {
	var heal int32
	switch {
	case skill.SkillType == "heal" && skill.HealPct > 0:
		// 治疗技 heal_pct 存法力倍率×10（如 30 表示法力×300%）。
		heal = caster.Mana * skill.HealPct / 10
	case skill.HealPct > 0:
		heal = caster.MaxHP * skill.HealPct / 100
	}
	heal += skill.FixedHeal
	if heal < 1 {
		return 1
	}
	return heal
}

// calculateCompositePanelRaw 按攻击/法力/速度面板百分比加算技能基础伤害。
func calculateCompositePanelRaw(attacker effectiveStats, skill skillDef) int64 {
	var raw int64
	if skill.AttackPct > 0 {
		raw += int64(attacker.Attack) * int64(skill.AttackPct) / 100
	}
	if skill.ManaPct > 0 {
		raw += int64(attacker.Mana) * int64(skill.ManaPct) / 100
	}
	if skill.SpeedPct > 0 {
		raw += int64(attacker.Speed) * int64(skill.SpeedPct) / 100
	}
	return raw
}

// calculateCompositeSkillDamage 多面板技能伤害；ignore_defense 为真时视为真伤。
func calculateCompositeSkillDamage(attacker effectiveStats, target effectiveStats, skill skillDef, isAOE bool) int32 {
	raw := calculateCompositePanelRaw(attacker, skill)
	if raw < 1 {
		return 1
	}
	if skill.IgnoreDefense {
		return int32(raw)
	}
	denom := calculatePocketDamageDenominator(target.Guard, isAOE)
	damage := int64(math.Round(float64(raw) / denom * 0.5))
	if damage < 1 {
		return 1
	}
	return int32(damage)
}
