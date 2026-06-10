package battle

type skillTargetRule uint32

const (
	targetEnemySingle skillTargetRule = 1
	targetAllySingle  skillTargetRule = 2
	targetEnemyAll    skillTargetRule = 3
	targetEnemyMulti  skillTargetRule = 4
)

type skillDef struct {
	ID                     uint32
	Name                   string
	TargetRule             skillTargetRule
	AttackPct              int32
	ManaPct                int32
	DefensePct             int32
	SpeedPct               int32
	TargetCurrentHPPct     int32
	FixedDamage            int32
	HealPct                int32
	FixedHeal              int32
	AllowCrit              bool
	IgnoreDefense          bool
	ArmorBreakPct          uint32
	VulnerabilityPct       uint32
	BleedChancePct         uint32
	BleedRounds            uint32
	BleedDamage            int32
	SealChancePct          uint32
	SealRounds             uint32
	VulnerabilityChancePct uint32
	VulnerabilityRounds    uint32
	VulnerabilityApplyPct  uint32
	ArmorBreakChancePct    uint32
	ArmorBreakRounds       uint32
	SlowChancePct          uint32
	SlowRounds             uint32
	SlowMultiplierPct      uint32
	CritBoostRounds        uint32
	CritBoostPct           uint32
	CurseChancePct         uint32
	CurseRounds            uint32
	CurseDamage            int32
	CurseManaPct           int32
	ControlChancePct       uint32
	ControlRounds          uint32
	ControlStatusID        uint32
	PreferredTargetHP      string
	TargetCount            uint32
}

var skillCatalog = map[uint32]skillDef{
	DefaultAttackSkillID: {
		ID:         DefaultAttackSkillID,
		Name:       "普通攻击",
		TargetRule: targetEnemySingle,
		AttackPct:  100,
		ManaPct:    35,
		SpeedPct:   35,
		AllowCrit:  true,
	},
	1002: {
		ID:                     1002,
		Name:                   "火花冲击",
		TargetRule:             targetEnemyAll,
		AttackPct:              120,
		ManaPct:                85,
		SpeedPct:               55,
		AllowCrit:              true,
		BleedChancePct:         70,
		BleedRounds:            2,
		BleedDamage:            4,
		VulnerabilityChancePct: 100,
		VulnerabilityRounds:    2,
		VulnerabilityApplyPct:  12,
		ControlChancePct:       35,
		ControlRounds:          1,
		ControlStatusID:        StatusParalysis,
	},
	1003: {
		ID:                1003,
		Name:              "活力治愈",
		TargetRule:        targetAllySingle,
		HealPct:           22,
		CritBoostRounds:   2,
		CritBoostPct:      20,
		PreferredTargetHP: "lowest",
	},
	1004: {
		ID:               1004,
		Name:             "弧光连射",
		TargetRule:       targetEnemyMulti,
		TargetCount:      2,
		AttackPct:        105,
		ManaPct:          40,
		SpeedPct:         25,
		AllowCrit:        true,
		BleedChancePct:   40,
		BleedRounds:      2,
		BleedDamage:      2,
		ControlChancePct: 20,
		ControlRounds:    1,
		ControlStatusID:  StatusConfusion,
	},
	DefaultEnemySkillID: {
		ID:             DefaultEnemySkillID,
		Name:           "野性撞击",
		TargetRule:     targetEnemySingle,
		AttackPct:      95,
		ManaPct:        20,
		FixedDamage:    2,
		AllowCrit:      true,
		CurseChancePct: 40,
		CurseRounds:    2,
		CurseDamage:    3,
	},
	90002: {
		ID:                90002,
		Name:              "利爪突袭",
		TargetRule:        targetEnemySingle,
		AttackPct:         110,
		ManaPct:           30,
		SpeedPct:          20,
		AllowCrit:         true,
		BleedChancePct:    50,
		BleedRounds:       2,
		BleedDamage:       3,
		SlowChancePct:     100,
		SlowRounds:        2,
		SlowMultiplierPct: 70,
		ControlChancePct:  30,
		ControlRounds:     1,
		ControlStatusID:   StatusConfusion,
	},
	90003: {
		ID:                90003,
		Name:              "野性回春",
		TargetRule:        targetAllySingle,
		HealPct:           18,
		PreferredTargetHP: "lowest",
	},
}

func getSkillDef(skillID uint32) (skillDef, bool) {
	def, ok := skillCatalog[skillID]
	return def, ok
}

func (r skillTargetRule) protocolName() string {
	switch r {
	case targetAllySingle:
		return "ally_single"
	case targetEnemyAll:
		return "enemy_all"
	case targetEnemyMulti:
		return "enemy_multi"
	default:
		return "enemy_single"
	}
}
