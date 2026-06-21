package equipment

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEquipmentDefinitionNotFound = errors.New("equipment definition not found")
	ErrInvalidAdminEquipmentInput  = errors.New("invalid admin equipment input")
	ErrEquipmentDefinitionConflict = errors.New("equipment definition conflict")
)

// AdminCombatStats 描述装备模板上的次要战斗属性，键集合与 pet_combat_stat_cap 一致。
type AdminCombatStats struct {
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

// AdminMedicinePouchExtra 描述药囊模板的战后恢复范围。
type AdminMedicinePouchExtra struct {
	RestorePlayerHP      bool `json:"restore_player_hp"`
	RestorePlayerSpirit  bool `json:"restore_player_spirit"`
	RestorePlayerVigor   bool `json:"restore_player_vigor"`
	RestorePetHP         bool `json:"restore_pet_hp"`
	RestorePetSpirit     bool `json:"restore_pet_spirit"`
	RestoreLineupPets    bool `json:"restore_lineup_pets"`
}

// AdminListQuery 定义后台装备模板列表筛选。
type AdminListQuery struct {
	ItemID    uint64
	EquipSlot string
	Keyword   string
	SetID     uint64
	Enabled   *bool
	Page      uint32
	PageSize  uint32
}

// Normalize 补齐默认分页并裁剪字符串。
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
	q.EquipSlot = strings.TrimSpace(q.EquipSlot)
	q.Keyword = strings.TrimSpace(q.Keyword)
	return q
}

// AdminEquipmentSummary 是装备模板列表行。
type AdminEquipmentSummary struct {
	ItemID         uint64    `json:"item_id"`
	ItemCode       string    `json:"item_code"`
	ItemName       string    `json:"item_name"`
	EquipSlot      string    `json:"equip_slot"`
	EquipSlotLabel string    `json:"equip_slot_label"`
	RequiredLevel  uint32    `json:"required_level"`
	Quality        uint32    `json:"quality"`
	CanEnhance     bool      `json:"can_enhance"`
	MaxEnhanceLevel uint32   `json:"max_enhance_level"`
	SetID          uint64    `json:"set_id"`
	IsEnabled      bool      `json:"is_enabled"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// AdminEquipmentList 是列表分页响应。
type AdminEquipmentList struct {
	Items    []AdminEquipmentSummary `json:"items"`
	Total    uint64                  `json:"total"`
	Page     uint32                  `json:"page"`
	PageSize uint32                  `json:"page_size"`
}

// AdminEquipmentDetail 是装备模板详情，合并 item_definition 与 item_equipment_extra。
type AdminEquipmentDetail struct {
	ItemID                    uint64                 `json:"item_id"`
	ItemCode                  string                 `json:"item_code"`
	ItemName                  string                 `json:"item_name"`
	Desc                      string                 `json:"desc"`
	Icon                      string                 `json:"icon"`
	Quality                   uint32                 `json:"quality"`
	Rarity                    uint32                 `json:"rarity"`
	RequiredLevel             uint32                 `json:"required_level"`
	BindType                  string                 `json:"bind_type"`
	CanSell                   bool                   `json:"can_sell"`
	CanStore                  bool                   `json:"can_store"`
	IsEnabled                 bool                   `json:"is_enabled"`
	EquipSlot                 string                 `json:"equip_slot"`
	EquipSlotLabel            string                 `json:"equip_slot_label"`
	CareerLimit               string                 `json:"career_limit"`
	CanEnhance                bool                   `json:"can_enhance"`
	MaxEnhanceLevel           uint32                 `json:"max_enhance_level"`
	SetID                     uint64                 `json:"set_id"`
	AppearanceSkinID          string                 `json:"appearance_skin_id"`
	AppearanceOnly            bool                   `json:"appearance_only"`
	BaseHP                    uint32                 `json:"base_hp"`
	BaseMana                  uint32                 `json:"base_mana"`
	BaseATK                   uint32                 `json:"base_atk"`
	BaseDEF                   uint32                 `json:"base_def"`
	BaseSPD                   uint32                 `json:"base_spd"`
	CombatStats               AdminCombatStats       `json:"combat_stats"`
	EnhancePerLevelStats      map[string]uint32      `json:"enhance_per_level_stats"`
	SocketCount               uint32                 `json:"socket_count"`
	AllowedGemTypes           []string               `json:"allowed_gem_types"`
	MedicinePouch             *AdminMedicinePouchExtra `json:"medicine_pouch,omitempty"`
	CreatedAt                 time.Time              `json:"created_at"`
	UpdatedAt                 time.Time              `json:"updated_at"`
}

// AdminUpsertEquipmentInput 描述后台创建/更新装备模板的请求体。
type AdminUpsertEquipmentInput struct {
	ItemID               uint64                 `json:"item_id"`
	ItemCode             string                 `json:"item_code"`
	ItemName             string                 `json:"item_name"`
	Desc                 string                 `json:"desc"`
	Icon                 string                 `json:"icon"`
	Quality              uint32                 `json:"quality"`
	Rarity               uint32                 `json:"rarity"`
	RequiredLevel        uint32                 `json:"required_level"`
	BindType             string                 `json:"bind_type"`
	CanSell              bool                   `json:"can_sell"`
	CanStore             bool                   `json:"can_store"`
	IsEnabled            bool                   `json:"is_enabled"`
	EquipSlot            string                 `json:"equip_slot"`
	CareerLimit          string                 `json:"career_limit"`
	CanEnhance           bool                   `json:"can_enhance"`
	MaxEnhanceLevel      uint32                 `json:"max_enhance_level"`
	SetID                uint64                 `json:"set_id"`
	AppearanceSkinID     string                 `json:"appearance_skin_id"`
	AppearanceOnly       bool                   `json:"appearance_only"`
	BaseHP               uint32                 `json:"base_hp"`
	BaseMana             uint32                 `json:"base_mana"`
	BaseATK              uint32                 `json:"base_atk"`
	BaseDEF              uint32                 `json:"base_def"`
	BaseSPD              uint32                 `json:"base_spd"`
	CombatStats          AdminCombatStats       `json:"combat_stats"`
	EnhancePerLevelStats map[string]uint32      `json:"enhance_per_level_stats"`
	SocketCount          uint32                 `json:"socket_count"`
	AllowedGemTypes      []string               `json:"allowed_gem_types"`
	MedicinePouch        *AdminMedicinePouchExtra `json:"medicine_pouch,omitempty"`
}

// Normalize 收口默认值，并按槽位自动修正 appearance_only / can_enhance。
func (input AdminUpsertEquipmentInput) Normalize() AdminUpsertEquipmentInput {
	input.ItemCode = strings.TrimSpace(input.ItemCode)
	input.ItemName = strings.TrimSpace(input.ItemName)
	input.Desc = strings.TrimSpace(input.Desc)
	input.Icon = strings.TrimSpace(input.Icon)
	input.BindType = strings.TrimSpace(input.BindType)
	input.EquipSlot = strings.TrimSpace(input.EquipSlot)
	input.CareerLimit = strings.TrimSpace(input.CareerLimit)
	input.AppearanceSkinID = strings.TrimSpace(input.AppearanceSkinID)
	if input.Quality == 0 {
		input.Quality = 1
	}
	if input.Rarity == 0 {
		input.Rarity = 1
	}
	if input.BindType == "" {
		input.BindType = "none"
	}
	if input.EnhancePerLevelStats == nil {
		input.EnhancePerLevelStats = map[string]uint32{}
	}
	if input.AllowedGemTypes == nil {
		input.AllowedGemTypes = []string{}
	}
	slot := EquipSlot(input.EquipSlot)
	if slot == EquipSlotCostume {
		input.AppearanceOnly = true
		input.CanEnhance = false
		input.MaxEnhanceLevel = 0
	}
	if slot == EquipSlotMedicinePouch {
		input.CanEnhance = false
		input.MaxEnhanceLevel = 0
		if input.MedicinePouch == nil {
			input.MedicinePouch = &AdminMedicinePouchExtra{
				RestorePlayerHP:     true,
				RestorePlayerSpirit: true,
				RestorePlayerVigor:  true,
				RestorePetHP:        true,
				RestorePetSpirit:    true,
			}
		}
	}
	if !input.CanEnhance {
		input.MaxEnhanceLevel = 0
		input.EnhancePerLevelStats = map[string]uint32{}
	} else if input.MaxEnhanceLevel == 0 {
		input.MaxEnhanceLevel = 15
	}
	return input
}

// Validate 校验后台提交的关键字段。
func (input AdminUpsertEquipmentInput) Validate() error {
	input = input.Normalize()
	if input.ItemID == 0 || input.ItemCode == "" || input.ItemName == "" {
		return ErrInvalidAdminEquipmentInput
	}
	if !IsValidEquipSlot(input.EquipSlot) {
		return ErrInvalidAdminEquipmentInput
	}
	return nil
}
