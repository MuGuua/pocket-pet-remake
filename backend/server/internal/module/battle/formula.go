package battle

import "math"

// The current pet demo data still uses two-digit stats, so we keep the
// original "def/(def+K)" formula shape from the design doc but temporarily use
// a smaller balance scale. This preserves the intended curve while keeping
// visible damage in the current MVP environment.
const defenseReductionScale = 1000.0

// The design doc suggests explicit upper bounds for crit-related values so one
// malformed data row cannot create runaway numbers in battle resolution.
const (
	maxCritRatePct = 100
	maxCritDmgPct  = 2000
)

// effectiveStats is the normalized combat stat snapshot used by the formula
// layer. Keeping it separate from actorRuntime makes future temporary buffs,
// passives, and state-derived overrides easier to plug in without rewriting the
// whole battle actor model.
type effectiveStats struct {
	Attack      int32
	Defense     int32
	Speed       int32
	Mana        int32
	CurrentHP   int32
	MaxHP       int32
	CritRatePct uint32
	CritDmgPct  uint32
	ArmorBroken bool

	// The current MVP only needs a small subset of the future buff/debuff model,
	// but keeping these formula-facing fields explicit lets status and passive
	// systems plug into damage math without changing every call site again.
	VulnerabilityPct uint32
	GenericBlockPct  uint32
	PetBlockPct      uint32
}

// baseDamageBreakdown records each formula contribution separately so tests and
// future battle logs can inspect where the final value came from.
type baseDamageBreakdown struct {
	AttackPart    int32
	ManaPart      int32
	DefensePart   int32
	SpeedPart     int32
	CurrentHPPart int32
	FixedPart     int32
	Total         int32
}

// allowsCriticalHit reports whether the skill's damage payload should enter the
// crit branch. Pure fixed-damage skills stay non-crit by default, matching the
// current design document and keeping future "true damage" style skills stable.
func (b baseDamageBreakdown) allowsCriticalHit() bool {
	return b.AttackPart > 0 || b.ManaPart > 0 || b.DefensePart > 0 || b.SpeedPart > 0 || b.CurrentHPPart > 0
}

func (a *actorRuntime) effectiveStats() effectiveStats {
	// We normalize runtime values here so later status/passive work only needs to
	// modify actorRuntime modifiers instead of every individual battle formula.
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
	return effectiveStats{
		Attack:      attack,
		Defense:     defense,
		Speed:       speed,
		Mana:        mana,
		CurrentHP:   int32(a.hp),
		MaxHP:       int32(a.hpMax),
		CritRatePct: a.critRatePct + a.statusCritRateBonusPct,
		CritDmgPct:  a.critDmgPct,
		ArmorBroken: a.statusArmorBroken,

		VulnerabilityPct: a.statusVulnerabilityPct,
		GenericBlockPct:  a.genericBlockPct,
		PetBlockPct:      a.petBlockPct,
	}
}

// calculateBaseDamage mirrors the design doc's "multiple stat coefficients are
// summed first" rule. Even if some coefficients are unused in the current
// skills, keeping them in one place avoids another refactor when richer skills
// land later.
func calculateBaseDamage(attacker effectiveStats, target effectiveStats, skill skillDef) baseDamageBreakdown {
	breakdown := baseDamageBreakdown{
		AttackPart:    attacker.Attack * skill.AttackPct / 100,
		ManaPart:      attacker.Mana * skill.ManaPct / 100,
		DefensePart:   attacker.Defense * skill.DefensePct / 100,
		SpeedPart:     attacker.Speed * skill.SpeedPct / 100,
		CurrentHPPart: target.CurrentHP * skill.TargetCurrentHPPct / 100,
		FixedPart:     skill.FixedDamage,
	}
	breakdown.Total = breakdown.AttackPart + breakdown.ManaPart + breakdown.DefensePart + breakdown.SpeedPart + breakdown.CurrentHPPart + breakdown.FixedPart
	if breakdown.Total < 1 {
		breakdown.Total = 1
	}
	return breakdown
}

// calculateDefenseReduction keeps the doc's recommended curve and 90% cap.
// Future armor break / vulnerability effects can be wired into the same entry
// point without touching the damage call sites again.
func calculateDefenseReduction(target effectiveStats, skill skillDef) float64 {
	if skill.IgnoreDefense {
		return 0.0
	}
	effectiveDefense := float64(target.Defense)
	if target.ArmorBroken {
		effectiveDefense = 0
	}
	if effectiveDefense < 0 {
		effectiveDefense = 0
	}
	if skill.ArmorBreakPct > 0 {
		effectiveDefense *= 1.0 - float64(skill.ArmorBreakPct)/100.0
	}
	reduction := effectiveDefense / (effectiveDefense + defenseReductionScale)
	reduction -= float64(skill.VulnerabilityPct+target.VulnerabilityPct) / 100.0
	reduction = math.Max(0.0, reduction)
	return math.Min(reduction, 0.90)
}

func calculateBlockReduction(attacker effectiveStats, target effectiveStats) float64 {
	blockPct := target.GenericBlockPct
	// The current battle MVP only has pet-vs-monster combat, so both sides are
	// treated as pet-sourced damage until humanoid/mercenary actors arrive.
	if attacker.Attack > 0 || attacker.Mana > 0 {
		blockPct = maxUint32(blockPct, target.PetBlockPct)
	}
	if blockPct > 100 {
		blockPct = 100
	}
	return float64(blockPct) / 100.0
}

func calculateFinalDamage(baseDamage int32, defenseReduction float64, blockReduction float64) int32 {
	damage := int32(math.Round(float64(baseDamage) * (1.0 - defenseReduction) * (1.0 - blockReduction)))
	if damage < 1 {
		return 1
	}
	return damage
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
	if critRatePct > maxCritRatePct {
		return maxCritRatePct
	}
	return critRatePct
}

func clampCritDmgPct(critDmgPct uint32) uint32 {
	if critDmgPct < 100 {
		return 100
	}
	if critDmgPct > maxCritDmgPct {
		return maxCritDmgPct
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
