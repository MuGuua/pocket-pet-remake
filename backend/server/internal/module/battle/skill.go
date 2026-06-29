package battle

import "strings"

type skillTargetRule uint32

const (
	targetEnemySingle skillTargetRule = 1
	targetAllySingle  skillTargetRule = 2
	targetEnemyAll    skillTargetRule = 3
	targetEnemyMulti  skillTargetRule = 4
	targetAllyAll     skillTargetRule = 5
	targetSelf        skillTargetRule = 6
)

type skillDef struct {
	ID                     uint32
	Name                   string
	SkillType              string
	TargetRule             skillTargetRule
	AnimationKey           string
	SkillVisualID          string
	CastColor              string
	ImpactColor            string
	Projectile             bool
	IsSkillAttack          bool
	IsBasicAttack          bool
	EnergyCost             uint32
	AttackPct              int32
	ManaPct                int32
	DefensePct             int32
	SpeedPct               int32
	TargetCurrentHPPct     int32
	FixedDamage            int32
	HealPct                int32
	FixedHeal              int32
	// SkillMult 为新表「技能倍数」；缺省时回退 AttackPct/100。
	SkillMult              uint32
	SkillCritAdd           uint32
	AllowCrit              bool
	IgnoreDefense          bool
	ArmorBreakPct          uint32
	VulnerabilityPct       uint32
	BleedChancePct         uint32
	BleedRounds            uint32
	BleedDamage            int32
	SealChancePct          uint32
	SealPower              uint32
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
	ControlPower           uint32
	ControlRounds          uint32
	ControlStatusID        uint32
	PreferredTargetHP      string
	TargetCount            uint32
}

var skillCatalog = map[uint32]skillDef{
	DefaultCharacterSkillID: {
		ID:                  DefaultCharacterSkillID,
		Name:                "裂空斩",
		TargetRule:          targetEnemySingle,
		AnimationKey:        "character_slash",
		SkillVisualID:       "character_slash",
		CastColor:           "#8FD6FF",
		ImpactColor:         "#BDE9FF",
		Projectile:          true,
		IsSkillAttack:       true,
		EnergyCost:          16,
		AttackPct:           135,
		ManaPct:             55,
		SpeedPct:            35,
		AllowCrit:           true,
		ArmorBreakChancePct: 100,
		ArmorBreakRounds:    2,
		BleedChancePct:      45,
		BleedRounds:         2,
		BleedDamage:         3,
	},
	DefaultAttackSkillID: {
		ID:            DefaultAttackSkillID,
		Name:          "普通攻击",
		TargetRule:    targetEnemySingle,
		AnimationKey:  "slash",
		SkillVisualID: "slash",
		CastColor:     "#EBEBF5",
		ImpactColor:   "#FFF2F2",
		Projectile:    false,
		IsBasicAttack: true,
		EnergyCost:    0,
		AttackPct:     100,
		ManaPct:       35,
		SpeedPct:      35,
		AllowCrit:     true,
	},
	1002: {
		ID:                     1002,
		Name:                   "火花冲击",
		TargetRule:             targetEnemyAll,
		AnimationKey:           "burst",
		CastColor:              "#FFAA5C",
		ImpactColor:            "#FFD46B",
		Projectile:             true,
		IsSkillAttack:          true,
		EnergyCost:             18,
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
		AnimationKey:      "heal",
		CastColor:         "#73F5A3",
		ImpactColor:       "#B7FFD0",
		Projectile:        false,
		IsSkillAttack:     true,
		EnergyCost:        14,
		HealPct:           22,
		CritBoostRounds:   2,
		CritBoostPct:      20,
		PreferredTargetHP: "lowest",
	},
	1004: {
		ID:               1004,
		Name:             "弧光连射",
		TargetRule:       targetEnemyMulti,
		AnimationKey:     "volley",
		CastColor:        "#C6D1FF",
		ImpactColor:      "#ECECFF",
		Projectile:       true,
		IsSkillAttack:    true,
		EnergyCost:       16,
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
		AnimationKey:   "slash",
		CastColor:      "#FFB88F",
		ImpactColor:    "#FFDDD1",
		Projectile:     false,
		EnergyCost:     0,
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
		AnimationKey:      "volley",
		CastColor:         "#FF9E85",
		ImpactColor:       "#FFC7BA",
		Projectile:        true,
		EnergyCost:        12,
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
		AnimationKey:      "heal",
		CastColor:         "#84F8B3",
		ImpactColor:       "#C8FFE0",
		Projectile:        false,
		IsSkillAttack:     true,
		EnergyCost:        10,
		HealPct:           18,
		PreferredTargetHP: "lowest",
	},
}

func (r skillTargetRule) protocolName() string {
	switch r {
	case targetAllySingle:
		return "ally_single"
	case targetAllyAll:
		return "ally_all"
	case targetSelf:
		return "self"
	case targetEnemyAll:
		return "enemy_all"
	case targetEnemyMulti:
		return "enemy_multi"
	default:
		return "enemy_single"
	}
}

// isPassiveSkill 判断是否为常驻被动（support 且零消耗），不应出现在可选技能列表。
func (d skillDef) isPassiveSkill() bool {
	return d.SkillType == "support" && d.EnergyCost == 0
}

// isHealSkill 判断是否为治疗类主动技能。
func (d skillDef) isHealSkill() bool {
	if d.SkillType == "heal" {
		return true
	}
	if d.isPassiveSkill() {
		return false
	}
	return d.FixedHeal > 0 || (d.HealPct > 0 && d.SkillType != "attack")
}

// usesManaPanel 是否以法力面板参与口袋伤害公式。
// 仅当 attack_pct 未配置且非法力复合面板技时，才用法力替代攻击面板。
func (d skillDef) usesManaPanel() bool {
	if d.AttackPct > 0 || d.usesCompositePanelDamage() {
		return false
	}
	return d.ManaPct > 0
}

// isJudgmentSkill 审判系：control_chance 表示失手概率。
func (d skillDef) isJudgmentSkill() bool {
	return strings.HasPrefix(d.Name, "审判")
}

// isRampageSkill 暴走系：随机攻击一名敌人。
func (d skillDef) isRampageSkill() bool {
	return strings.HasPrefix(d.Name, "暴走")
}

// isGuaranteedHit 圣技/魂技系列在资料中标注为必中。
func (d skillDef) isGuaranteedHit() bool {
	return strings.HasPrefix(d.Name, "圣技") || strings.HasPrefix(d.Name, "魂技")
}

// usesCompositePanelDamage 多面板加算伤害（攻击/法力/速度百分比叠加）。
func (d skillDef) usesCompositePanelDamage() bool {
	if d.SkillType != "attack" || d.SkillMult > 0 {
		return false
	}
	panels := 0
	if d.AttackPct > 0 {
		panels++
	}
	if d.ManaPct > 0 {
		panels++
	}
	if d.SpeedPct > 0 {
		panels++
	}
	return panels >= 2
}

// prefersRandomMultiTarget 双体技能随机选取目标（不依赖玩家点选主目标）。
func (d skillDef) prefersRandomMultiTarget() bool {
	return d.TargetRule == targetEnemyMulti && d.PreferredTargetHP == "random"
}

// isSoulDevourSkill 噬魂系：control_power 表示扣除目标精力。
func (d skillDef) isSoulDevourSkill() bool {
	return strings.HasPrefix(d.Name, "噬魂")
}
