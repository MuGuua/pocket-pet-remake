package skill

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrSkillDefinitionNotFound 表示后台请求的系统技能模板不存在。
	ErrSkillDefinitionNotFound = errors.New("skill definition not found")
	// ErrInvalidAdminSkillDefinitionInput 表示后台提交的系统技能模板字段非法。
	ErrInvalidAdminSkillDefinitionInput = errors.New("invalid admin skill definition input")
	// ErrSkillDefinitionConflict 表示 skill_id 或 skill_code 已存在。
	ErrSkillDefinitionConflict = errors.New("skill definition conflict")
	// ErrInvalidSkillReference 表示引用了不存在或已停用的 skill_id。
	ErrInvalidSkillReference = errors.New("invalid skill reference")
)

const (
	ActivationModeActive  = "active"
	ActivationModePassive = "passive"
	QualityNormal         = "normal"
	QualityDivine         = "divine"
	QualitySoul           = "soul"
	QualitySacred         = "sacred"
	QualityPeerless       = "peerless"

	PassiveAttrModeFlat    = "flat"
	PassiveAttrModePercent = "percent"

	PassiveAttrKeyHPMax              = "hp_max"
	PassiveAttrKeyATK                = "atk"
	PassiveAttrKeySPD                = "spd"
	PassiveAttrKeyMana               = "mana"
	PassiveAttrKeyCritRatePct        = "crit_rate_pct"
	PassiveAttrKeyCritDmgPct         = "crit_dmg_pct"
	PassiveAttrKeyPhysicalResistPct  = "physical_resist_pct"
	PassiveAttrKeySkillResistPct     = "skill_resist_pct"
	PassiveAttrKeyAllStatusResistPct = "all_status_resist_pct"
)

// AdminListQuery 定义系统技能模板列表筛选参数。
type AdminListQuery struct {
	SkillID        uint32
	Name           string
	Category       string
	Type           string
	ActivationMode string
	Quality        string
	Enabled        *bool
	OrderBy        string
	Page           uint32
	PageSize       uint32
}

// Normalize 收口分页与筛选默认值。
func (q AdminListQuery) Normalize() AdminListQuery {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	q.Name = strings.TrimSpace(q.Name)
	q.Category = strings.TrimSpace(q.Category)
	q.Type = strings.TrimSpace(q.Type)
	q.ActivationMode = strings.TrimSpace(q.ActivationMode)
	q.Quality = strings.TrimSpace(q.Quality)
	q.OrderBy = strings.TrimSpace(q.OrderBy)
	return q
}

// AdminSummary 是列表页展示字段；公式与效果字段仅在详情页展示。
type AdminSummary struct {
	SkillID        uint32    `json:"skill_id"`
	SkillCode      string    `json:"skill_code"`
	SkillName      string    `json:"skill_name"`
	SkillCategory  string    `json:"skill_category"`
	SkillType      string    `json:"skill_type"`
	ActivationMode string    `json:"activation_mode"`
	SkillQuality   string    `json:"skill_quality"`
	TargetType     string    `json:"target_type"`
	EnergyCost     uint32    `json:"energy_cost"`
	IsBasicAttack  bool      `json:"is_basic_attack"`
	IsEnabled      bool      `json:"is_enabled"`
	StatusText     string    `json:"status_text"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// AdminList 是系统技能模板分页响应。
type AdminList struct {
	Items    []AdminSummary `json:"items"`
	Total    uint64         `json:"total"`
	Page     uint32         `json:"page"`
	PageSize uint32         `json:"page_size"`
}

// AdminTargetRule 描述技能目标选择规则。
type AdminTargetRule struct {
	TargetType        string `json:"target_type"`
	TargetCount       uint32 `json:"target_count"`
	PreferredTargetHP string `json:"preferred_target_hp"`
}

// AdminFormula 描述战斗结算使用的伤害/治疗公式系数。
type AdminFormula struct {
	AttackPct          int32  `json:"attack_pct"`
	ManaPct            int32  `json:"mana_pct"`
	DefensePct         int32  `json:"defense_pct"`
	SpeedPct           int32  `json:"speed_pct"`
	TargetCurrentHPPct int32  `json:"target_current_hp_pct"`
	FixedDamage        int32  `json:"fixed_damage"`
	HealPct            int32  `json:"heal_pct"`
	FixedHeal          int32  `json:"fixed_heal"`
	EnergyCost         uint32 `json:"energy_cost"`
	IsSkillAttack      bool   `json:"is_skill_attack"`
	AllowCrit          bool   `json:"allow_crit"`
	IgnoreDefense      bool   `json:"ignore_defense"`
	SkillMult          uint32 `json:"skill_mult"`
	SkillCritAdd       uint32 `json:"skill_crit_add"`
	SkillHitBonus      uint32 `json:"skill_hit_bonus"`
}

// AdminStatusEffects 描述技能附带的状态效果概率与数值。
type AdminStatusEffects struct {
	ArmorBreakPct          uint32 `json:"armor_break_pct"`
	VulnerabilityPct       uint32 `json:"vulnerability_pct"`
	BleedChancePct         uint32 `json:"bleed_chance_pct"`
	BleedRounds            uint32 `json:"bleed_rounds"`
	BleedDamage            int32  `json:"bleed_damage"`
	SealChancePct          uint32 `json:"seal_chance_pct"`
	SealPower              uint32 `json:"seal_power"`
	SealRounds             uint32 `json:"seal_rounds"`
	VulnerabilityChancePct uint32 `json:"vulnerability_chance_pct"`
	VulnerabilityRounds    uint32 `json:"vulnerability_rounds"`
	VulnerabilityApplyPct  uint32 `json:"vulnerability_apply_pct"`
	ArmorBreakChancePct    uint32 `json:"armor_break_chance_pct"`
	ArmorBreakRounds       uint32 `json:"armor_break_rounds"`
	SlowChancePct          uint32 `json:"slow_chance_pct"`
	SlowRounds             uint32 `json:"slow_rounds"`
	SlowMultiplierPct      uint32 `json:"slow_multiplier_pct"`
	CritBoostRounds        uint32 `json:"crit_boost_rounds"`
	CritBoostPct           uint32 `json:"crit_boost_pct"`
	CurseChancePct         uint32 `json:"curse_chance_pct"`
	CurseRounds            uint32 `json:"curse_rounds"`
	CurseDamage            int32  `json:"curse_damage"`
	CurseManaPct           int32  `json:"curse_mana_pct"`
	ControlChancePct       uint32 `json:"control_chance_pct"`
	ControlPower           uint32 `json:"control_power"`
	ControlRounds          uint32 `json:"control_rounds"`
	ControlStatusID        uint32 `json:"control_status_id"`
}

// AdminPresentation 描述客户端战斗表现层使用的展示参数。
type AdminPresentation struct {
	AnimationKey  string `json:"animation_key"`
	SkillVisualID string `json:"skill_visual_id"`
	CastColor     string `json:"cast_color"`
	ImpactColor   string `json:"impact_color"`
	Projectile    bool   `json:"projectile"`
}

// AdminDetail 是详情抽屉所需的完整模板信息。
type AdminDetail struct {
	SkillID          uint32             `json:"skill_id"`
	SkillCode        string             `json:"skill_code"`
	SkillName        string             `json:"skill_name"`
	SkillCategory    string             `json:"skill_category"`
	WeaponDiscipline string             `json:"weapon_discipline"`
	LearnExpRequired uint32             `json:"learn_exp_required"`
	LearnExpPerUse   uint32             `json:"learn_exp_per_use"`
	SkillType        string             `json:"skill_type"`
	ActivationMode   string             `json:"activation_mode"`
	SkillQuality     string             `json:"skill_quality"`
	Description      string             `json:"description"`
	AcquireMethod    string             `json:"acquire_method"`
	IsBasicAttack    bool               `json:"is_basic_attack"`
	IsEnabled        bool               `json:"is_enabled"`
	StatusText       string             `json:"status_text"`
	SortWeight       uint32             `json:"sort_weight"`
	TargetRule       AdminTargetRule    `json:"target_rule"`
	Formula          AdminFormula       `json:"formula"`
	StatusEffects    AdminStatusEffects `json:"status_effects"`
	Presentation     AdminPresentation  `json:"presentation"`
	PassiveAttrKey   string             `json:"passive_attr_key"`
	PassiveAttrMode  string             `json:"passive_attr_mode"`
	PassiveAttrValue int32              `json:"passive_attr_value"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// AdminUpsertInput 描述后台新增或编辑系统技能模板时提交的字段。
type AdminUpsertInput struct {
	SkillID                uint32 `json:"skill_id"`
	SkillCode              string `json:"skill_code"`
	SkillName              string `json:"skill_name"`
	SkillCategory          string `json:"skill_category"`
	WeaponDiscipline       string `json:"weapon_discipline"`
	LearnExpRequired       uint32 `json:"learn_exp_required"`
	LearnExpPerUse         uint32 `json:"learn_exp_per_use"`
	SkillType              string `json:"skill_type"`
	ActivationMode         string `json:"activation_mode"`
	SkillQuality           string `json:"skill_quality"`
	Description            string `json:"description"`
	AcquireMethod          string `json:"acquire_method"`
	IsBasicAttack          bool   `json:"is_basic_attack"`
	IsEnabled              bool   `json:"is_enabled"`
	SortWeight             uint32 `json:"sort_weight"`
	TargetType             string `json:"target_type"`
	TargetCount            uint32 `json:"target_count"`
	PreferredTargetHP      string `json:"preferred_target_hp"`
	AnimationKey           string `json:"animation_key"`
	SkillVisualID          string `json:"skill_visual_id"`
	CastColor              string `json:"cast_color"`
	ImpactColor            string `json:"impact_color"`
	Projectile             bool   `json:"projectile"`
	IsSkillAttack          bool   `json:"is_skill_attack"`
	EnergyCost             uint32 `json:"energy_cost"`
	AllowCrit              bool   `json:"allow_crit"`
	IgnoreDefense          bool   `json:"ignore_defense"`
	SkillMult              uint32 `json:"skill_mult"`
	SkillCritAdd           uint32 `json:"skill_crit_add"`
	SkillHitBonus          uint32 `json:"skill_hit_bonus"`
	AttackPct              int32  `json:"attack_pct"`
	ManaPct                int32  `json:"mana_pct"`
	DefensePct             int32  `json:"defense_pct"`
	SpeedPct               int32  `json:"speed_pct"`
	TargetCurrentHPPct     int32  `json:"target_current_hp_pct"`
	FixedDamage            int32  `json:"fixed_damage"`
	HealPct                int32  `json:"heal_pct"`
	FixedHeal              int32  `json:"fixed_heal"`
	ArmorBreakPct          uint32 `json:"armor_break_pct"`
	VulnerabilityPct       uint32 `json:"vulnerability_pct"`
	BleedChancePct         uint32 `json:"bleed_chance_pct"`
	BleedRounds            uint32 `json:"bleed_rounds"`
	BleedDamage            int32  `json:"bleed_damage"`
	SealChancePct          uint32 `json:"seal_chance_pct"`
	SealPower              uint32 `json:"seal_power"`
	SealRounds             uint32 `json:"seal_rounds"`
	VulnerabilityChancePct uint32 `json:"vulnerability_chance_pct"`
	VulnerabilityRounds    uint32 `json:"vulnerability_rounds"`
	VulnerabilityApplyPct  uint32 `json:"vulnerability_apply_pct"`
	ArmorBreakChancePct    uint32 `json:"armor_break_chance_pct"`
	ArmorBreakRounds       uint32 `json:"armor_break_rounds"`
	SlowChancePct          uint32 `json:"slow_chance_pct"`
	SlowRounds             uint32 `json:"slow_rounds"`
	SlowMultiplierPct      uint32 `json:"slow_multiplier_pct"`
	CritBoostRounds        uint32 `json:"crit_boost_rounds"`
	CritBoostPct           uint32 `json:"crit_boost_pct"`
	CurseChancePct         uint32 `json:"curse_chance_pct"`
	CurseRounds            uint32 `json:"curse_rounds"`
	CurseDamage            int32  `json:"curse_damage"`
	CurseManaPct           int32  `json:"curse_mana_pct"`
	ControlChancePct       uint32 `json:"control_chance_pct"`
	ControlPower           uint32 `json:"control_power"`
	ControlRounds          uint32 `json:"control_rounds"`
	ControlStatusID        uint32 `json:"control_status_id"`
	PassiveAttrKey         string `json:"passive_attr_key"`
	PassiveAttrMode        string `json:"passive_attr_mode"`
	PassiveAttrValue       int32  `json:"passive_attr_value"`
}

// Normalize 补齐模板字段默认值，避免运营漏填导致战斗链路异常。
func (input AdminUpsertInput) Normalize() AdminUpsertInput {
	input.SkillCode = strings.TrimSpace(input.SkillCode)
	input.SkillName = strings.TrimSpace(input.SkillName)
	input.SkillCategory = strings.TrimSpace(input.SkillCategory)
	input.WeaponDiscipline = strings.TrimSpace(input.WeaponDiscipline)
	input.SkillType = strings.TrimSpace(input.SkillType)
	input.ActivationMode = strings.TrimSpace(input.ActivationMode)
	input.SkillQuality = strings.TrimSpace(input.SkillQuality)
	input.Description = strings.TrimSpace(input.Description)
	input.AcquireMethod = strings.TrimSpace(input.AcquireMethod)
	input.TargetType = strings.TrimSpace(input.TargetType)
	input.PreferredTargetHP = strings.TrimSpace(input.PreferredTargetHP)
	input.AnimationKey = strings.TrimSpace(input.AnimationKey)
	input.SkillVisualID = strings.TrimSpace(input.SkillVisualID)
	input.CastColor = strings.TrimSpace(input.CastColor)
	input.ImpactColor = strings.TrimSpace(input.ImpactColor)
	input.PassiveAttrKey = strings.TrimSpace(input.PassiveAttrKey)
	input.PassiveAttrMode = strings.TrimSpace(input.PassiveAttrMode)
	if input.SkillCategory == "" {
		input.SkillCategory = CategoryCommon
	}
	if !IsValidCategory(input.SkillCategory) {
		input.SkillCategory = CategoryCommon
	}
	if input.SkillCategory == CategoryWeapon {
		if input.LearnExpPerUse == 0 {
			input.LearnExpPerUse = 1
		}
	}
	if input.SkillType == "" {
		input.SkillType = "attack"
	}
	if input.ActivationMode == "" {
		input.ActivationMode = ActivationModeActive
	}
	if input.ActivationMode != ActivationModePassive {
		input.ActivationMode = ActivationModeActive
	}
	if input.SkillQuality == "" {
		input.SkillQuality = QualityNormal
	}
	if input.TargetType == "" {
		input.TargetType = "enemy_single"
	}
	if input.ActivationMode == ActivationModePassive {
		input.TargetType = "self"
		input.TargetCount = 0
		input.PreferredTargetHP = ""
		input.EnergyCost = 0
	} else {
		if input.AnimationKey == "" {
			input.AnimationKey = "slash"
		}
		if input.CastColor == "" {
			input.CastColor = "#EBEBF5"
		}
		if input.ImpactColor == "" {
			input.ImpactColor = "#FFF2F2"
		}
		if input.AttackPct == 0 && input.HealPct == 0 && input.FixedDamage == 0 && input.FixedHeal == 0 {
			input.AttackPct = 100
		}
	}
	if input.PassiveAttrKey == "" {
		input.PassiveAttrMode = ""
		input.PassiveAttrValue = 0
	}
	return input
}

// RuntimeDefinition 是战斗运行时读取的技能模板快照。
type RuntimeDefinition struct {
	SkillID                uint32
	SkillCode              string
	SkillType              string
	SkillCategory          string
	WeaponDiscipline       string
	LearnExpRequired       uint32
	LearnExpPerUse         uint32
	SkillName              string
	Description            string
	ActivationMode         string
	SkillQuality           string
	TargetType             string
	TargetCount            uint32
	PreferredTargetHP      string
	AnimationKey           string
	SkillVisualID          string
	CastColor              string
	ImpactColor            string
	Projectile             bool
	IsSkillAttack          bool
	EnergyCost             uint32
	SkillMult              uint32
	SkillCritAdd           uint32
	SkillHitBonus          uint32
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
	PassiveAttrKey         string
	PassiveAttrMode        string
	PassiveAttrValue       int32
	IsBasicAttack          bool
}

// 返回客户端技能表现资源标识；历史数据未配置专用标识时使用数据库技能编码兼容。
func (d RuntimeDefinition) ResolvedSkillVisualID() string {
	if visualID := strings.TrimSpace(d.SkillVisualID); visualID != "" {
		return visualID
	}
	return strings.TrimSpace(d.SkillCode)
}

// 判断技能品质是否属于客户端已支持的五档边框范围。
func IsValidQuality(value string) bool {
	switch strings.TrimSpace(value) {
	case QualityNormal, QualityDivine, QualitySoul, QualitySacred, QualityPeerless:
		return true
	default:
		return false
	}
}

// IsValidPassiveAttrKey 判断后台选择的永久被动属性字段是否在当前受支持范围内。
func IsValidPassiveAttrKey(value string) bool {
	switch value {
	case "",
		PassiveAttrKeyHPMax,
		PassiveAttrKeyATK,
		PassiveAttrKeySPD,
		PassiveAttrKeyMana,
		PassiveAttrKeyCritRatePct,
		PassiveAttrKeyCritDmgPct,
		PassiveAttrKeyPhysicalResistPct,
		PassiveAttrKeySkillResistPct,
		PassiveAttrKeyAllStatusResistPct:
		return true
	default:
		return false
	}
}

// IsValidPassiveAttrMode 判断后台提交的永久被动加成方式是否合法。
func IsValidPassiveAttrMode(value string) bool {
	switch value {
	case "", PassiveAttrModeFlat, PassiveAttrModePercent:
		return true
	default:
		return false
	}
}

// SupportsPassiveAttrPercent 判断某个永久属性字段是否允许按百分比提升基础属性。
func SupportsPassiveAttrPercent(value string) bool {
	switch value {
	case PassiveAttrKeyHPMax, PassiveAttrKeyATK, PassiveAttrKeySPD, PassiveAttrKeyMana:
		return true
	default:
		return false
	}
}
