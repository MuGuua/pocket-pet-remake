package battle

import "math"

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
}

// effectiveStats 是公式层与部分战斗分支共用的快照；伤害面板取 Attack 字段。
type effectiveStats struct {
	UnitClass            uint32
	Attack               int32
	Defense              int32
	Speed                int32
	Mana                 int32
	CurrentHP            int32
	MaxHP                int32
	HitPct               uint32
	DodgePct             uint32
	CritRatePct          uint32
	CritDmgPct           uint32
	CritResistPct        uint32
	CritDmgResistPct     uint32
	ArmorBroken          bool
	VulnerabilityPct     uint32
	PhysicalResistPct    uint32
	SkillResistPct       uint32
	ReverseSkillResistPct uint32
	GenericShieldPct     uint32
	CharacterResistPct   uint32
	PetResistPct         uint32
	MercenaryResistPct   uint32
	Guard                uint32
	TalentDmgPct         uint32
	TalentReducePct      uint32
	ElementAdvPct        uint32
	ElementPenaltyPct    uint32
}

func (a *actorRuntime) effectiveStats() effectiveStats {
	attack := applyPercentModifier(int32(a.atk)+a.attackFlatBonus, a.attackMultiplierPct, a.globalMultiplierPct)
	defense := applyPercentModifier(int32(a.def)+a.defenseFlatBonus, a.defenseMultiplierPct, a.globalMultiplierPct)
	speedMultiplierPct := a.speedMultiplierPct
	if speedMultiplierPct == 0 {
		speedMultiplierPct = 100
	}
	if a.statusSpeedMultiplierPct > 0 {
		speedMultiplierPct = speedMultiplierPct * a.statusSpeedMultiplierPct / 100
	}
	speed := applyPercentModifier(int32(a.spd)+a.speedFlatBonus, speedMultiplierPct, a.globalMultiplierPct)
	mana := applyPercentModifier(int32(a.mana)+a.manaFlatBonus, a.manaMultiplierPct, a.globalMultiplierPct)
	guard := a.guard
	if guard == 0 {
		guard = uint32(maxInt32(defense, 0))
	}
	return effectiveStats{
		UnitClass:             a.unitClass,
		Attack:                attack,
		Defense:               defense,
		Speed:                 speed,
		Mana:                  mana,
		CurrentHP:             int32(a.hp),
		MaxHP:                 int32(a.hpMax),
		HitPct:                a.hitPct,
		DodgePct:              a.dodgeRatePct,
		CritRatePct:           a.critRatePct + a.statusCritRateBonusPct,
		CritDmgPct:            a.critDmgPct,
		CritResistPct:         a.critResistPct,
		CritDmgResistPct:      a.critDmgResistPct,
		ArmorBroken:           a.statusArmorBroken,
		VulnerabilityPct:      a.statusVulnerabilityPct,
		PhysicalResistPct:     a.physicalResistPct,
		SkillResistPct:        a.skillResistPct,
		ReverseSkillResistPct: a.reverseSkillResistPct,
		GenericShieldPct:      a.genericShieldPct,
		CharacterResistPct:    a.characterResistPct,
		PetResistPct:          a.petResistPct,
		MercenaryResistPct:    a.mercenaryResistPct,
		Guard:                 guard,
		TalentDmgPct:          a.talentDmgPct,
		TalentReducePct:       a.talentReducePct,
		ElementAdvPct:         a.elementAdvPct,
		ElementPenaltyPct:     a.elementPenaltyPct,
	}
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
	return pocketDamageInput{
		AttackerPanel:        int64(attackerStats.Attack),
		DefenderPanel:        int64(targetStats.Defense),
		SkillMult:            skill.SkillMult,
		AttackScalePct:       skill.AttackPct,
		CritDmg:              attackerStats.CritDmgPct,
		SkillCritAdd:         skill.SkillCritAdd,
		AntiCrit:             target.critDmgResistPct,
		RevSkill:             attackerStats.ReverseSkillResistPct,
		SkillRes:             target.skillResistPct,
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

func scaledPanelBase(input pocketDamageInput) int64 {
	if input.SkillMult > 0 {
		return input.AttackerPanel * int64(input.SkillMult)
	}
	if input.AttackScalePct > 0 {
		return input.AttackerPanel * int64(input.AttackScalePct) / 100
	}
	return 0
}

func calculatePocketDamageNumerator(input pocketDamageInput) int64 {
	panelBase := scaledPanelBase(input)
	if panelBase <= 0 {
		return 0
	}
	critChain := int64(input.CritDmg) + int64(input.SkillCritAdd) - int64(input.AntiCrit)
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

func clampCritRatePct(critRatePct uint32) uint32 {
	if critRatePct > 100 {
		return 100
	}
	return critRatePct
}

func clampCritDmgPct(critDmgPct uint32) uint32 {
	if critDmgPct < 100 {
		return 100
	}
	if critDmgPct > 2000 {
		return 2000
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
	heal := caster.MaxHP * skill.HealPct / 100
	heal += skill.FixedHeal
	if heal < 1 {
		return 1
	}
	return heal
}
