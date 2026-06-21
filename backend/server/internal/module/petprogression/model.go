package petprogression

// MaxPetLevel 是宠物可达到的最高等级。
const MaxPetLevel uint32 = 100

// AptitudeProfile 描述不同宠物模板在有效资质公式中使用的倍率。
const (
	AptitudeProfileNormal  = "normal"
	AptitudeProfileSpecial = "special"
	AptitudeProfileArctic  = "arctic"
)

// AttrType 是五项可直接分配的战斗属性类型。
type AttrType string

const (
	AttrHPMax AttrType = "hp_max"
	AttrATK   AttrType = "atk"
	AttrSPD   AttrType = "spd"
	AttrMANA  AttrType = "mana"
	AttrDEF   AttrType = "def"
)

// GrowthAptitudes 保存基础资质与红色资质；与 pet.GrowthAptitudes 字段语义对齐但拆分存储。
type GrowthAptitudes struct {
	BaseHPApt   uint32
	BaseATKApt  uint32
	BaseDEFApt  uint32
	BaseSPDApt  uint32
	BaseMANAApt uint32
	ExtraHPApt  uint32
	ExtraATKApt uint32
	ExtraDEFApt uint32
	ExtraSPDApt uint32
	ExtraMANAApt uint32
}

// ManualAllocatedPoints 保存玩家手动分配的自由属性点累计值。
type ManualAllocatedPoints struct {
	HP   uint32
	ATK  uint32
	SPD  uint32
	MANA uint32
	DEF  uint32
}

// Total 返回手动分配点总数。
func (points ManualAllocatedPoints) Total() uint32 {
	return points.HP + points.ATK + points.SPD + points.MANA + points.DEF
}

// AutoAllocatedPoints 是等级/进化/转生产生的系统自动分配点。
type AutoAllocatedPoints struct {
	HP   uint32
	ATK  uint32
	SPD  uint32
	MANA uint32
	DEF  uint32
}

// CombatStats 是公式重算后的五项战斗属性。
type CombatStats struct {
	HPMax uint32
	ATK   uint32
	SPD   uint32
	MANA  uint32
	DEF   uint32
}

// ConvertRates 保存后台可配的转化率常数。
type ConvertRates struct {
	HPMax float64
	ATK   float64
	SPD   float64
	MANA  float64
	DEF   float64
}

// DefaultConvertRates 对应《资质数据》表9 种子值。
func DefaultConvertRates() ConvertRates {
	return ConvertRates{
		HPMax: 27.77,
		ATK:   277.77,
		SPD:   2081.51,
		MANA:  1388.73,
		DEF:   277.77,
	}
}

// ProgressionInput 是计算最终战斗属性所需的完整快照。
type ProgressionInput struct {
	Level           uint32
	EvolutionLevel  uint32
	RebirthLevel    uint32
	AptitudeProfile string
	Aptitudes       GrowthAptitudes
	ManualPoints    ManualAllocatedPoints
	ConvertRates    ConvertRates
	// BoostPct 首期固定为 0；二期接入贴纸/天赋/技能/法宝后改为每项独立系数。
	BoostPct float64
}
