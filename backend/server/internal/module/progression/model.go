package progression

import "errors"

const (
	// MaxPlayerLevel 是玩家角色可达到的最高等级。
	MaxPlayerLevel uint32 = 100

	// SourceAttrStrength 力量，主要转化为攻击。
	SourceAttrStrength = "strength"
	// SourceAttrVitality 体质，主要转化为生命与防御。
	SourceAttrVitality = "vitality"
	// SourceAttrAgility 敏捷，主要转化为速度与闪避。
	SourceAttrAgility = "agility"
	// SourceAttrMind 灵力，主要转化为法力。
	SourceAttrMind = "mind"
)

var (
	ErrInvalidAllocateInput = errors.New("invalid allocate attr input")
	ErrInsufficientAttrPoints = errors.New("insufficient attr points")
	ErrLevelConfigNotFound    = errors.New("player level config not found")
	ErrConvertConfigNotFound  = errors.New("player attr convert config not found")
)

// LevelConfig 描述某一等级升到下一级所需经验、升级奖励与战斗属性加成。
// 战斗加成在玩家从该等级升到下一级时累加到 base_* 裸装基础值。
type LevelConfig struct {
	Level       uint32 `json:"level"`
	ExpRequired uint64 `json:"exp_required"`
	AttrPoints  uint32 `json:"attr_points"`
	BonusATK    uint32 `json:"bonus_atk"`
	BonusHPMax  uint32 `json:"bonus_hp_max"`
	BonusSPD    uint32 `json:"bonus_spd"`
	BonusMANA   uint32 `json:"bonus_mana"`
	Status      uint32 `json:"status"`
}

// AttrConvertConfig 描述基础属性点向战斗属性的转化率。
type AttrConvertConfig struct {
	ID          uint64 `json:"id"`
	SourceAttr  string `json:"source_attr"`
	TargetAttr  string `json:"target_attr"`
	ConvertRate uint32 `json:"convert_rate"`
	Status      uint32 `json:"status"`
}

// AttrAllocationDelta 是玩家一次加点请求中的四维增量。
type AttrAllocationDelta struct {
	Strength uint32 `json:"strength"`
	Vitality uint32 `json:"vitality"`
	Agility  uint32 `json:"agility"`
	Mind     uint32 `json:"mind"`
}

// Total 返回本次加点消耗的自由属性点总数。
func (delta AttrAllocationDelta) Total() uint32 {
	return delta.Strength + delta.Vitality + delta.Agility + delta.Mind
}

// BaseCombatStats 保存不受加点影响的裸装基础战斗属性。
type BaseCombatStats struct {
	HPMax    uint32
	ATK      uint32
	DEF      uint32
	SPD      uint32
	MANA     uint32
	HitPct   uint32
	DodgePct uint32
}

// AllocatedAttrs 保存玩家已分配的基础属性点累计值。
type AllocatedAttrs struct {
	Strength uint32
	Vitality uint32
	Agility  uint32
	Mind     uint32
}

// CombatBonus 是根据转化率计算出的战斗属性加成。
type CombatBonus struct {
	HPMax    uint32
	ATK      uint32
	DEF      uint32
	SPD      uint32
	MANA     uint32
	HitPct   uint32
	DodgePct uint32
}

// ProgressionState 是升级与加点计算使用的玩家成长快照。
type ProgressionState struct {
	Level           uint32
	Exp             uint64
	FreeAttrPoints  uint32
	Allocated       AllocatedAttrs
	BaseCombat      BaseCombatStats
}

// ExpApplyResult 描述一次经验结算后的玩家成长变化。
type ExpApplyResult struct {
	Level            uint32
	Exp              uint64
	FreeAttrPoints   uint32
	LevelUpCount     uint32
	AttrPointsGained uint32
	ExpToNext        uint64
	CombatBonusGain  LevelUpCombatBonus
}

// LevelUpCombatBonus 描述本次连升过程中累加的裸装战斗属性加成。
type LevelUpCombatBonus struct {
	HPMax uint32 `json:"hp_max"`
	ATK   uint32 `json:"atk"`
	SPD   uint32 `json:"spd"`
	MANA  uint32 `json:"mana"`
}

// Add 把两次升级加成合并为一条展示/推送摘要。
func (bonus LevelUpCombatBonus) Add(other LevelUpCombatBonus) LevelUpCombatBonus {
	return LevelUpCombatBonus{
		HPMax: bonus.HPMax + other.HPMax,
		ATK:   bonus.ATK + other.ATK,
		SPD:   bonus.SPD + other.SPD,
		MANA:  bonus.MANA + other.MANA,
	}
}

// AdminUpsertLevelConfigInput 是后台编辑等级经验配置的入参。
type AdminUpsertLevelConfigInput struct {
	ExpRequired uint64 `json:"exp_required"`
	AttrPoints  uint32 `json:"attr_points"`
	BonusATK    uint32 `json:"bonus_atk"`
	BonusHPMax  uint32 `json:"bonus_hp_max"`
	BonusSPD    uint32 `json:"bonus_spd"`
	BonusMANA   uint32 `json:"bonus_mana"`
	Status      uint32 `json:"status"`
}

// AdminUpsertAttrConvertInput 是后台编辑属性转化率的入参。
type AdminUpsertAttrConvertInput struct {
	SourceAttr  string `json:"source_attr"`
	TargetAttr  string `json:"target_attr"`
	ConvertRate uint32 `json:"convert_rate"`
	Status      uint32 `json:"status"`
}
