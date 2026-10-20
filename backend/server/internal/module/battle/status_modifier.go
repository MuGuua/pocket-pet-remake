package battle

// paralysisScaledResistMask 标识麻痹会削弱的抗性项。具体强弱由施加技能配置的
// control_status_power 决定，这里只描述「规则允许影响哪些属性」。
// 用位掩码是为了让同一套逻辑既能表达「全部抗性」，也能只削弱其中若干项。
const (
	paralysisScalePhysicalResist uint32 = 1 << iota
	paralysisScaleSkillResist
	paralysisScaleCritDmgResist
	paralysisScaleHit
	paralysisScaleDodge
	// paralysisScaleAllResist 是规则《战斗系统规则》§6 中麻痹的默认作用范围。
	paralysisScaleAllResist = paralysisScalePhysicalResist | paralysisScaleSkillResist |
		paralysisScaleCritDmgResist | paralysisScaleHit | paralysisScaleDodge
)

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
	// PropertyMultiplierPct 按百分比整体缩放若干战斗属性，用于「属性降至 50%」这类效果。
	// 与上面的 Penalty 字段不同，Penalty 是固定点数扣减，这里是对有效值的乘法。
	// 0 或 100 表示不缩放；多个来源取最小值（即削弱最强的那一个）。
	PropertyMultiplierPct uint32
	// PropertyScaleMask 指定 PropertyMultiplierPct 作用于哪些属性，取值见 paralysisScale* 常量。
	// 具体削弱哪几项由施加该状态的技能决定，而不是在状态里写死。
	PropertyScaleMask uint32
	// PropertyMultiplierFromPotency 为真时，使用状态实例上的 potency 作为缩放比例，
	// 让同一种状态可以被不同技能配置成不同的削弱强度。
	PropertyMultiplierFromPotency bool
	BlocksAction                  bool
	ForcesConfusion               bool
	NonStackable                  bool
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
	PropertyMultiplierPct        uint32
	PropertyScaleMask            uint32
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
			CritDmgPenalty:        150,
			PhysicalResistPenalty: 30,
			SkillResistPenalty:    30,
			NonStackable:          true,
		}
	case StatusConfusion:
		return statusModifierProfile{ForcesConfusion: true, NonStackable: true}
	case StatusParalysis:
		// 《战斗系统规则》§6：麻痹使物理抗性、技能抗性、抗爆伤、命中、闪避降低，并且无法行动。
		// 具体降低幅度由施加技能配置的 control_status_power 决定（缺省 50%），
		// 作用范围在此给出规则默认值，技能可用更小的掩码覆盖。
		return statusModifierProfile{
			PropertyMultiplierPct:         50,
			PropertyMultiplierFromPotency: true,
			PropertyScaleMask:             paralysisScaleAllResist,
			BlocksAction:                  true,
			NonStackable:                  true,
		}
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

// mergeStatusDerivedModifiers 合并单条状态的修饰。potency 是该状态实例上的强度参数，
// 用于「同一状态被不同技能配置成不同强度」的场景（例如麻痹由技能决定削弱到百分之多少）。
func mergeStatusDerivedModifiers(into *statusDerivedModifiers, profile statusModifierProfile, potency int32) {
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
	multiplierPct := profile.PropertyMultiplierPct
	if profile.PropertyMultiplierFromPotency {
		// 技能可覆盖削弱幅度；未配置（potency 为 0）时回退到状态默认值。
		if fromPotency := uint32(maxInt32(potency, 0)); fromPotency > 0 && fromPotency < 100 {
			multiplierPct = fromPotency
		}
	}
	if multiplierPct > 0 && multiplierPct < 100 {
		if into.PropertyMultiplierPct == 0 || multiplierPct < into.PropertyMultiplierPct {
			into.PropertyMultiplierPct = multiplierPct
			into.PropertyScaleMask = profile.PropertyScaleMask
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
		mergeStatusDerivedModifiers(&result, status.modifiers, status.potency)
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
