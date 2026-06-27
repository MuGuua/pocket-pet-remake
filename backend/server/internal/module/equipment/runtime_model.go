package equipment

import (
	"errors"

	"pocket-pet-remake/server/internal/module/item"
)

var (
	ErrEquipmentNotFound       = errors.New("equipment instance not found")
	ErrEquipmentSlotOccupied   = errors.New("equipment slot already occupied")
	ErrEquipmentSlotEmpty      = errors.New("equipment slot empty")
	ErrEquipmentLevelTooLow      = errors.New("player level too low for equipment")
	ErrEquipmentSlotMismatch   = errors.New("equipment slot mismatch")
	ErrEquipmentBagItemInvalid = errors.New("bag item is not equippable equipment")
	ErrEquipmentNotOwned       = errors.New("equipment not owned by player")
	ErrEquipmentBagFull        = errors.New("bag has no empty slot for unequipped item")
)

// RuntimeEquippedItem 描述玩家当前佩戴的单件装备实例与模板摘要。
type RuntimeEquippedItem struct {
	EquipSlot     string            `json:"equip_slot"`
	EquipSlotLabel string           `json:"equip_slot_label"`
	ItemUID       string            `json:"item_uid"`
	ItemID        uint64            `json:"item_id"`
	ItemName      string            `json:"item_name"`
	Icon          string            `json:"icon"`
	RequiredLevel    uint32            `json:"required_level"`
	EnhanceLevel  uint32            `json:"enhance_level"`
	AppearanceSkinID string         `json:"appearance_skin_id,omitempty"`
	AppearanceOnly   bool           `json:"appearance_only"`
	// Description 来自 item_definition.desc，供客户端详情面板展示后台编辑的介绍文案。
	Description string `json:"description,omitempty"`
	// DescriptionMentions 为介绍文案中 {item:ID} 占位符解析出的关联物品，供客户端内联展示 icon。
	DescriptionMentions []item.DescriptionMention `json:"description_mentions,omitempty"`
	Bonus         BonusAggregate    `json:"bonus"`
}

// BonusAggregate 汇总单件或全身装备的固定数值加成。
type BonusAggregate struct {
	HPMax                    uint32 `json:"hp_max"`
	MANA                     uint32 `json:"mana"`
	ATK                      uint32 `json:"atk"`
	DEF                      uint32 `json:"def"`
	SPD                      uint32 `json:"spd"`
	Spirit                   uint32 `json:"spirit"`
	SpiritMax                uint32 `json:"spirit_max"`
	HitPct                   uint32 `json:"hit_pct"`
	DodgePct                 uint32 `json:"dodge_pct"`
	CritRatePct              uint32 `json:"crit_rate_pct"`
	CritDmgPct               uint32 `json:"crit_dmg_pct"`
	PhysicalResistPct        uint32 `json:"physical_resist_pct"`
	ReversePhysicalResistPct uint32 `json:"reverse_physical_resist_pct"`
	SkillResistPct           uint32 `json:"skill_resist_pct"`
	ReverseSkillResistPct    uint32 `json:"reverse_skill_resist_pct"`
	ConfusionResistPct       uint32 `json:"confusion_resist_pct"`
	SleepResistPct           uint32 `json:"sleep_resist_pct"`
	ParalysisResistPct       uint32 `json:"paralysis_resist_pct"`
	SealResistPct            uint32 `json:"seal_resist_pct"`
	CurseResistPct           uint32 `json:"curse_resist_pct"`
	CritDmgResistPct         uint32 `json:"crit_dmg_resist_pct"`
	CritResistPct            uint32 `json:"crit_resist_pct"`
	CharacterResistPct       uint32 `json:"character_resist_pct"`
	PetResistPct             uint32 `json:"pet_resist_pct"`
}

// Add 将另一份加成累加到当前聚合结果。
func (bonus *BonusAggregate) Add(other BonusAggregate) {
	bonus.HPMax += other.HPMax
	bonus.MANA += other.MANA
	bonus.ATK += other.ATK
	bonus.DEF += other.DEF
	bonus.SPD += other.SPD
	bonus.Spirit += other.Spirit
	bonus.SpiritMax += other.SpiritMax
	bonus.HitPct += other.HitPct
	bonus.DodgePct += other.DodgePct
	bonus.CritRatePct += other.CritRatePct
	bonus.CritDmgPct += other.CritDmgPct
	bonus.PhysicalResistPct += other.PhysicalResistPct
	bonus.ReversePhysicalResistPct += other.ReversePhysicalResistPct
	bonus.SkillResistPct += other.SkillResistPct
	bonus.ReverseSkillResistPct += other.ReverseSkillResistPct
	bonus.ConfusionResistPct += other.ConfusionResistPct
	bonus.SleepResistPct += other.SleepResistPct
	bonus.ParalysisResistPct += other.ParalysisResistPct
	bonus.SealResistPct += other.SealResistPct
	bonus.CurseResistPct += other.CurseResistPct
	bonus.CritDmgResistPct += other.CritDmgResistPct
	bonus.CritResistPct += other.CritResistPct
	bonus.CharacterResistPct += other.CharacterResistPct
	bonus.PetResistPct += other.PetResistPct
}

// EquipFromBagResult 是佩戴成功后返回给协议层的结果。
type EquipFromBagResult struct {
	EquippedSlot RuntimeEquippedItem   `json:"equipped"`
	Unequipped   *RuntimeEquippedItem  `json:"unequipped,omitempty"`
	AllEquipped  []RuntimeEquippedItem `json:"all_equipped"`
}

// UnequipSlotResult 是卸下成功后返回给协议层的结果。
type UnequipSlotResult struct {
	Unequipped  RuntimeEquippedItem   `json:"unequipped"`
	AllEquipped []RuntimeEquippedItem `json:"all_equipped"`
}

// EquippedTemplateRefreshEntry 描述某玩家当前佩戴的指定模板实例，供模板热更新后走卸装/再穿戴刷新。
type EquippedTemplateRefreshEntry struct {
	PlayerID  uint64
	EquipSlot string
	ItemUID   string
}
