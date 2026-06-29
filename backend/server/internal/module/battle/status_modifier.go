package battle

// statusModifierProfile 描述一个战斗状态附带的数值修饰；由 statusRuntime 持有并在 refresh 阶段汇总。
type statusModifierProfile struct {
	HitPenalty                   uint32
	DodgePenalty                 uint32
	CritDmgPenalty               uint32
	PhysicalResistPenalty        uint32
	SkillResistPenalty           uint32
	ReversePhysicalResistPenalty uint32
	ReverseSkillResistPenalty    uint32
	SpeedMultiplierPct           uint32
	BlocksAction                 bool
	ForcesConfusion              bool
	NonStackable                 bool
}

// statusDerivedModifiers 从当前 status 列表汇总后的运行时修饰快照。
type statusDerivedModifiers struct {
	HitPenalty                   uint32
	DodgePenalty                 uint32
	CritDmgPenalty               uint32
	PhysicalResistPenalty        uint32
	SkillResistPenalty           uint32
	ReversePhysicalResistPenalty uint32
	ReverseSkillResistPenalty    uint32
	SpeedMultiplierPct           uint32
	ResistBlessingBonus          uint32
	VulnerabilityPct             uint32
	ArmorBroken                  bool
	CritRateBonusPct             uint32
	BlocksAction                 bool
	ForcesConfusion              bool
}

func defaultStatusProfile(statusID uint32) statusModifierProfile {
	switch statusID {
	case StatusHolyRepentance:
		return statusModifierProfile{
			HitPenalty:                   50,
			DodgePenalty:                 50,
			CritDmgPenalty:               200,
			PhysicalResistPenalty:        25,
			SkillResistPenalty:           25,
			ReversePhysicalResistPenalty: 25,
			ReverseSkillResistPenalty:    25,
			NonStackable:                 true,
		}
	case StatusElectrified:
		return statusModifierProfile{
			SpeedMultiplierPct: 60,
			HitPenalty:         40,
			DodgePenalty:       40,
			NonStackable:       true,
		}
	case StatusBloodConfusion:
		return statusModifierProfile{
			ForcesConfusion: true,
			NonStackable:    true,
		}
	case StatusPhantomFlash:
		return statusModifierProfile{
			SpeedMultiplierPct: 65,
			HitPenalty:         35,
			DodgePenalty:       35,
			CritDmgPenalty:     100,
			NonStackable:       true,
		}
	case StatusCharmWind:
		return statusModifierProfile{
			HitPenalty:                40,
			DodgePenalty:              40,
			SkillResistPenalty:        25,
			ReverseSkillResistPenalty: 25,
			NonStackable:              true,
		}
	case StatusDemonPower:
		return statusModifierProfile{
			CritDmgPenalty:      150,
			PhysicalResistPenalty: 30,
			SkillResistPenalty:    30,
			NonStackable:          true,
		}
	case StatusConfusion:
		return statusModifierProfile{ForcesConfusion: true, NonStackable: true}
	case StatusParalysis:
		return statusModifierProfile{BlocksAction: true, NonStackable: true}
	case StatusResistBlessing:
		return statusModifierProfile{NonStackable: true}
	default:
		return statusModifierProfile{}
	}
}

func statusApplyResistPct(actor *actorRuntime, statusID uint32) uint32 {
	if actor == nil {
		return 0
	}
	switch statusID {
	case StatusHolyRepentance, StatusElectrified, StatusPhantomFlash:
		return actor.paralysisResistPct
	case StatusCharmWind:
		return actor.sealResistPct
	case StatusDemonPower:
		return actor.curseResistPct
	case StatusConfusion, StatusBloodConfusion:
		return actor.confusionResistPct
	case StatusSeal:
		return actor.sealResistPct
	case StatusSleep:
		return actor.sleepResistPct
	case StatusParalysis:
		return actor.paralysisResistPct
	case StatusCurse:
		return actor.curseResistPct
	default:
		return 0
	}
}

func statusApplyLabel(statusID uint32) string {
	switch statusID {
	case StatusHolyRepentance:
		return " 陷入圣光忏悔。"
	case StatusElectrified:
		return " 陷入感电。"
	case StatusBloodConfusion:
		return " 陷入混乱之血。"
	case StatusPhantomFlash:
		return " 陷入幻影闪击。"
	case StatusCharmWind:
		return " 陷入魅惑之风。"
	case StatusDemonPower:
		return " 陷入恶魔之力。"
	case StatusResistBlessing:
		return " 进入光之洗礼。"
	default:
		return controlStatusApplyLabel(statusID)
	}
}

// resolveSignatureStatusApplyChance 资料中「X% 命中率 + Y 点威力」：在概率系与威力系中取较高者，上限 100。
func resolveSignatureStatusApplyChance(chancePct uint32, power uint32, targetResist uint32) uint32 {
	powerChance := resolveControlApplyChance(0, power, targetResist)
	if chancePct == 0 {
		return powerChance
	}
	if power == 0 {
		if chancePct > 100 {
			return 100
		}
		return chancePct
	}
	if powerChance > chancePct {
		return powerChance
	}
	return chancePct
}

func mergeStatusDerivedModifiers(into *statusDerivedModifiers, profile statusModifierProfile) {
	if into == nil {
		return
	}
	into.HitPenalty += profile.HitPenalty
	into.DodgePenalty += profile.DodgePenalty
	into.CritDmgPenalty += profile.CritDmgPenalty
	into.PhysicalResistPenalty += profile.PhysicalResistPenalty
	into.SkillResistPenalty += profile.SkillResistPenalty
	into.ReversePhysicalResistPenalty += profile.ReversePhysicalResistPenalty
	into.ReverseSkillResistPenalty += profile.ReverseSkillResistPenalty
	if profile.SpeedMultiplierPct > 0 {
		if into.SpeedMultiplierPct == 0 || profile.SpeedMultiplierPct < into.SpeedMultiplierPct {
			into.SpeedMultiplierPct = profile.SpeedMultiplierPct
		}
	}
	if profile.BlocksAction {
		into.BlocksAction = true
	}
	if profile.ForcesConfusion {
		into.ForcesConfusion = true
	}
}

func (a *actorRuntime) collectStatusDerivedModifiers() statusDerivedModifiers {
	result := statusDerivedModifiers{SpeedMultiplierPct: 100}
	if a == nil {
		return result
	}
	for _, status := range a.statuses {
		if status == nil || status.remainingRound == 0 {
			continue
		}
		mergeStatusDerivedModifiers(&result, status.modifiers)
		switch status.statusID {
		case StatusVulnerability:
			result.VulnerabilityPct = maxUint32(result.VulnerabilityPct, uint32(maxInt32(status.potency, 0)))
		case StatusArmorBreak:
			result.ArmorBroken = true
		case StatusSlow:
			slowPct := uint32(maxInt32(status.potency, 0))
			if slowPct == 0 {
				slowPct = 100
			}
			if slowPct < result.SpeedMultiplierPct {
				result.SpeedMultiplierPct = slowPct
			}
		case StatusCritBoost:
			result.CritRateBonusPct = maxUint32(result.CritRateBonusPct, uint32(maxInt32(status.potency, 0)))
		case StatusResistBlessing:
			result.ResistBlessingBonus = maxUint32(result.ResistBlessingBonus, uint32(maxInt32(status.potency, 0)))
		}
	}
	return result
}
