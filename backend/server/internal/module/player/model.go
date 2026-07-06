package player

import (
	"errors"
	"strings"
	"time"

	"pocket-pet-remake/server/internal/module/progression"
)

var (
	ErrPlayerNotFound        = errors.New("player not found")
	ErrPlayerNameDuplicated  = errors.New("player name duplicated")
	ErrAccountNameDuplicated = errors.New("account name duplicated")
	ErrInvalidAdminInput     = errors.New("invalid admin player input")
	ErrInvalidRegisterInput  = errors.New("invalid register input")
	ErrInvalidRewardAttrKey  = errors.New("invalid reward attr key")
)

// DefaultPlayerSkinID 是玩家尚未配置形象时的服务端默认资源 ID。
const DefaultPlayerSkinID = "初始形象男_001"

// DefaultFemalePlayerSkinID 是女性新玩家默认使用的初始形象资源 ID。
const DefaultFemalePlayerSkinID = "初始形象女_002"

// RegisterGenderMale 表示男性注册选项。
const RegisterGenderMale = "male"

// RegisterGenderFemale 表示女性注册选项。
const RegisterGenderFemale = "female"

// StarterProfile 描述新注册玩家的服务端权威初始战斗属性与背包容量。
// 所有创建玩家入口都应复用这份配置，避免后台、仓储与数据库默认值各自维护一套口径。
type StarterProfile struct {
	ATK            uint32
	HPMax          uint32
	SPD            uint32
	MANA           uint32
	DEF            uint32
	Vigor          uint32
	VigorMax       uint32
	Spirit         uint32
	SpiritMax      uint32
	HitPct         uint32
	DodgePct       uint32
	CritRatePct    uint32
	CritDmgPct     uint32
	BagCapacity    uint32
	BagMaxCapacity uint32
	SkillIDs       []uint32
}

// DefaultStarterProfile 返回当前版本的新玩家初始属性。
func DefaultStarterProfile() StarterProfile {
	return StarterProfile{
		ATK:            42,
		HPMax:          148,
		SPD:            11,
		MANA:           30,
		DEF:            15,
		Vigor:          100,
		VigorMax:       100,
		Spirit:         40,
		SpiritMax:      40,
		HitPct:         95,
		DodgePct:       3,
		CritRatePct:    7,
		CritDmgPct:     150,
		BagCapacity:    50,
		BagMaxCapacity: 50,
		SkillIDs:       []uint32{1101, 1001},
	}
}

type Profile struct {
	PlayerID uint64
	Name     string
	Level    uint32
	Exp      uint64
	// ExpToNext 表示当前等级距离下一级还需要的经验，满级时为 0。
	ExpToNext          uint64
	FreeAttrPoints     uint32
	Strength           uint32
	Vitality           uint32
	Agility            uint32
	Mind               uint32
	Gold               uint32
	SceneID            uint32
	PosX               int32
	PosY               int32
	HP                 uint32
	HPMax              uint32
	Vigor              uint32
	VigorMax           uint32
	Spirit             uint32
	SpiritMax          uint32
	ATK                uint32
	DEF                uint32
	SPD                uint32
	MANA               uint32
	HitPct             uint32
	DodgePct           uint32
	CritRatePct        uint32
	CritDmgPct         uint32
	PhysicalResistPct  uint32
	SkillResistPct     uint32
	ConfusionResistPct uint32
	SleepResistPct     uint32
	ParalysisResistPct uint32
	SealResistPct      uint32
	CurseResistPct     uint32
	CritResistPct      uint32
	CritDmgResistPct   uint32
	CharacterResistPct uint32
	PetResistPct       uint32
	MercenaryResistPct uint32
	GenericShieldPct   uint32
	Guard              uint32
	TalentDmgPct       uint32
	TalentReducePct    uint32
	ElementAdvPct      uint32
	ElementPenaltyPct  uint32
	SkillIDs           []uint32
	// SkinID 是当前玩家形象资源 ID，世界与战斗表现层共用。
	SkinID string
	// BaseHPMax 等字段保存裸装基础战斗属性，加点加成会叠加在其上。
	BaseHPMax    uint32
	BaseATK      uint32
	BaseDEF      uint32
	BaseSPD      uint32
	BaseMANA     uint32
	BaseHitPct   uint32
	BaseDodgePct uint32
}

// RegisterInput 描述公开注册接口所需的最小字段。
// 当前版本直接把账号名复用为玩家名，避免额外增加昵称流程。
type RegisterInput struct {
	AccountName string `json:"account"`
	Password    string `json:"password"`
	Gender      string `json:"gender"`
}

// Normalize 统一裁剪注册输入，并把性别规范成服务端内部枚举。
func (input RegisterInput) Normalize() RegisterInput {
	input.AccountName = strings.TrimSpace(input.AccountName)
	input.Password = strings.TrimSpace(input.Password)
	input.Gender = normalizeRegisterGender(input.Gender)
	return input
}

// RegisterResult 描述公开注册成功后的最小角色摘要。
type RegisterResult struct {
	PlayerID    uint64 `json:"player_id"`
	AccountName string `json:"account"`
	PlayerName  string `json:"player_name"`
	SkinID      string `json:"skin_id"`
}

// ExpGrantResult 描述一次经验发放后的玩家档案与升级摘要。
type ExpGrantResult struct {
	Profile          *Profile
	LevelUpCount     uint32
	AttrPointsGained uint32
	CombatBonusGain  progression.LevelUpCombatBonus
}

// AdminListQuery 描述后台玩家列表检索条件。
// 后台管理页会把筛选和分页参数显式传给服务端，避免客户端假造分页结果。
type AdminListQuery struct {
	PlayerID uint64
	Name     string
	Status   *uint32
	Page     uint32
	PageSize uint32
}

// Normalize 为后台玩家列表补齐默认分页并裁剪危险入参。
// 这样 HTTP handler 只负责解析字符串，具体默认值保持在领域层统一收口。
func (q AdminListQuery) Normalize() AdminListQuery {
	q.Name = strings.TrimSpace(q.Name)
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	return q
}

// AdminCreatePlayerInput 描述后台创建玩家所需的最小持久化信息。
// 新玩家会同时创建 account 和 player 记录，保证后续登录与世界入口链路完整可用。
type AdminCreatePlayerInput struct {
	AccountName string   `json:"account_name"`
	Password    string   `json:"password"`
	Name        string   `json:"name"`
	Level       uint32   `json:"level"`
	Gold        uint64   `json:"gold"`
	SceneID     uint32   `json:"scene_id"`
	PosX        int32    `json:"pos_x"`
	PosY        int32    `json:"pos_y"`
	HP          uint32   `json:"hp"`
	HPMax       uint32   `json:"hp_max"`
	Vigor       uint32   `json:"vigor"`
	VigorMax    uint32   `json:"vigor_max"`
	Spirit      uint32   `json:"spirit"`
	SpiritMax   uint32   `json:"spirit_max"`
	ATK         uint32   `json:"atk"`
	DEF         uint32   `json:"def"`
	SPD         uint32   `json:"spd"`
	MANA        uint32   `json:"mana"`
	Status      uint32   `json:"status"`
	SkillIDs    []uint32 `json:"skill_ids"`
	SkinID      string   `json:"skin_id"`
}

// Normalize 会为后台创建玩家补齐服务端默认值，避免前端遗漏字段导致落库残缺。
func (input AdminCreatePlayerInput) Normalize() AdminCreatePlayerInput {
	starter := DefaultStarterProfile()
	input.AccountName = strings.TrimSpace(input.AccountName)
	input.Password = strings.TrimSpace(input.Password)
	input.Name = strings.TrimSpace(input.Name)
	if input.Level == 0 {
		input.Level = 1
	}
	if input.SceneID == 0 {
		input.SceneID = 1
	}
	if input.HPMax == 0 {
		input.HPMax = starter.HPMax
	}
	if input.HP == 0 {
		input.HP = input.HPMax
	}
	if input.VigorMax == 0 {
		input.VigorMax = starter.VigorMax
	}
	if input.Vigor == 0 {
		input.Vigor = input.VigorMax
	}
	if input.SpiritMax == 0 {
		input.SpiritMax = starter.SpiritMax
	}
	if input.Spirit == 0 {
		input.Spirit = input.SpiritMax
	}
	if input.ATK == 0 {
		input.ATK = starter.ATK
	}
	if input.DEF == 0 {
		input.DEF = starter.DEF
	}
	if input.SPD == 0 {
		input.SPD = starter.SPD
	}
	if input.MANA == 0 {
		input.MANA = starter.MANA
	}
	if len(input.SkillIDs) == 0 {
		input.SkillIDs = append([]uint32{}, starter.SkillIDs...)
	}
	if input.Status == 0 {
		input.Status = 1
	}
	input.SkinID = strings.TrimSpace(input.SkinID)
	return input
}

// resolveStarterSkinIDByGender 把注册性别映射成服务端权威初始形象。
func resolveStarterSkinIDByGender(gender string) string {
	if normalizeRegisterGender(gender) == RegisterGenderFemale {
		return DefaultFemalePlayerSkinID
	}
	return DefaultPlayerSkinID
}

func normalizeRegisterGender(gender string) string {
	switch strings.ToLower(strings.TrimSpace(gender)) {
	case "female", "woman", "girl", "f", "女":
		return RegisterGenderFemale
	default:
		return RegisterGenderMale
	}
}

// ResolvedCreateStats 是创建玩家时最终落库的战斗属性快照。
type ResolvedCreateStats struct {
	HP           uint32
	HPMax        uint32
	Vigor        uint32
	VigorMax     uint32
	Spirit       uint32
	SpiritMax    uint32
	ATK          uint32
	DEF          uint32
	SPD          uint32
	MANA         uint32
	HitPct       uint32
	DodgePct     uint32
	CritRatePct  uint32
	CritDmgPct   uint32
	BaseHPMax    uint32
	BaseATK      uint32
	BaseDEF      uint32
	BaseSPD      uint32
	BaseMANA     uint32
	BaseHitPct   uint32
	BaseDodgePct uint32
}

// ResolveCreateStats 把后台输入与新手初始模板合并成创建玩家时的权威属性。
func (input AdminCreatePlayerInput) ResolveCreateStats() ResolvedCreateStats {
	normalized := input.Normalize()
	starter := DefaultStarterProfile()
	return ResolvedCreateStats{
		HP:           normalized.HP,
		HPMax:        normalized.HPMax,
		Vigor:        normalized.Vigor,
		VigorMax:     normalized.VigorMax,
		Spirit:       normalized.Spirit,
		SpiritMax:    normalized.SpiritMax,
		ATK:          normalized.ATK,
		DEF:          normalized.DEF,
		SPD:          normalized.SPD,
		MANA:         normalized.MANA,
		HitPct:       starter.HitPct,
		DodgePct:     starter.DodgePct,
		CritRatePct:  starter.CritRatePct,
		CritDmgPct:   starter.CritDmgPct,
		BaseHPMax:    normalized.HPMax,
		BaseATK:      normalized.ATK,
		BaseDEF:      normalized.DEF,
		BaseSPD:      normalized.SPD,
		BaseMANA:     normalized.MANA,
		BaseHitPct:   starter.HitPct,
		BaseDodgePct: starter.DodgePct,
	}
}

// AdminUpdatePlayerInput 描述后台编辑玩家时允许修改的持久化字段。
// 编辑接口要求传完整快照，减少部分字段漏传后出现新旧值混杂的问题。
type AdminUpdatePlayerInput struct {
	Name      string   `json:"name"`
	Level     uint32   `json:"level"`
	Exp       uint64   `json:"exp"`
	Gold      uint64   `json:"gold"`
	SceneID   uint32   `json:"scene_id"`
	PosX      int32    `json:"pos_x"`
	PosY      int32    `json:"pos_y"`
	HP        uint32   `json:"hp"`
	HPMax     uint32   `json:"hp_max"`
	Vigor     uint32   `json:"vigor"`
	VigorMax  uint32   `json:"vigor_max"`
	Spirit    uint32   `json:"spirit"`
	SpiritMax uint32   `json:"spirit_max"`
	ATK       uint32   `json:"atk"`
	DEF       uint32   `json:"def"`
	SPD       uint32   `json:"spd"`
	MANA      uint32   `json:"mana"`
	Status    uint32   `json:"status"`
	SkillIDs  []uint32 `json:"skill_ids"`
	SkinID    string   `json:"skin_id"`
}

func (input AdminUpdatePlayerInput) Normalize() AdminUpdatePlayerInput {
	input.Name = strings.TrimSpace(input.Name)
	if input.Level == 0 {
		input.Level = 1
	}
	if input.SceneID == 0 {
		input.SceneID = 1
	}
	if input.HPMax == 0 {
		input.HPMax = input.HP
	}
	if input.VigorMax == 0 {
		input.VigorMax = input.Vigor
	}
	if input.SpiritMax == 0 {
		input.SpiritMax = input.Spirit
	}
	input.SkinID = strings.TrimSpace(input.SkinID)
	return input
}

// AdminPlayerSummary 是后台列表页消费的玩家摘要 DTO。
// 这里额外包含最近登录时间和状态文案，避免前端自己猜状态枚举含义。
type AdminPlayerSummary struct {
	PlayerID    uint64     `json:"player_id"`
	AccountName string     `json:"account_name"`
	Name        string     `json:"name"`
	Level       uint32     `json:"level"`
	Gold        uint64     `json:"gold"`
	Status      uint32     `json:"status"`
	StatusText  string     `json:"status_text"`
	SceneID     uint32     `json:"scene_id"`
	HP          uint32     `json:"hp"`
	HPMax       uint32     `json:"hp_max"`
	Vigor       uint32     `json:"vigor"`
	VigorMax    uint32     `json:"vigor_max"`
	Spirit      uint32     `json:"spirit"`
	SpiritMax   uint32     `json:"spirit_max"`
	LastLoginAt *time.Time `json:"last_login_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// AdminPlayerList 封装后台列表响应，保证前端分页器使用服务端总数。
type AdminPlayerList struct {
	Items    []AdminPlayerSummary `json:"items"`
	Total    uint64               `json:"total"`
	Page     uint32               `json:"page"`
	PageSize uint32               `json:"page_size"`
}

// AdminPlayerDetail 是后台详情页使用的玩家完整快照。
// 它仍然来自数据库权威数据，不复用客户端 EnterWorld 协议，以便后台字段更清晰稳定。
type AdminPlayerDetail struct {
	PlayerID           uint64                    `json:"player_id"`
	AccountID          uint64                    `json:"account_id"`
	AccountName        string                    `json:"account_name"`
	Name               string                    `json:"name"`
	Level              uint32                    `json:"level"`
	Exp                uint64                    `json:"exp"`
	FreeAttrPoints     uint32                    `json:"free_attr_points"`
	Strength           uint32                    `json:"strength"`
	Vitality           uint32                    `json:"vitality"`
	Agility            uint32                    `json:"agility"`
	Mind               uint32                    `json:"mind"`
	Gold               uint64                    `json:"gold"`
	Status             uint32                    `json:"status"`
	StatusText         string                    `json:"status_text"`
	SceneID            uint32                    `json:"scene_id"`
	PosX               int32                     `json:"pos_x"`
	PosY               int32                     `json:"pos_y"`
	HP                 uint32                    `json:"hp"`
	HPMax              uint32                    `json:"hp_max"`
	Vigor              uint32                    `json:"vigor"`
	VigorMax           uint32                    `json:"vigor_max"`
	Spirit             uint32                    `json:"spirit"`
	SpiritMax          uint32                    `json:"spirit_max"`
	ATK                uint32                    `json:"atk"`
	DEF                uint32                    `json:"def"`
	SPD                uint32                    `json:"spd"`
	MANA               uint32                    `json:"mana"`
	HitPct             uint32                    `json:"hit_pct"`
	DodgePct           uint32                    `json:"dodge_pct"`
	CritRatePct        uint32                    `json:"crit_rate_pct"`
	CritDmgPct         uint32                    `json:"crit_dmg_pct"`
	PhysicalResistPct  uint32                    `json:"physical_resist_pct"`
	SkillResistPct     uint32                    `json:"skill_resist_pct"`
	ConfusionResistPct uint32                    `json:"confusion_resist_pct"`
	SleepResistPct     uint32                    `json:"sleep_resist_pct"`
	ParalysisResistPct uint32                    `json:"paralysis_resist_pct"`
	SealResistPct      uint32                    `json:"seal_resist_pct"`
	CurseResistPct     uint32                    `json:"curse_resist_pct"`
	CritResistPct      uint32                    `json:"crit_resist_pct"`
	CritDmgResistPct   uint32                    `json:"crit_dmg_resist_pct"`
	CharacterResistPct uint32                    `json:"character_resist_pct"`
	PetResistPct       uint32                    `json:"pet_resist_pct"`
	MercenaryResistPct uint32                    `json:"mercenary_resist_pct"`
	GenericShieldPct   uint32                    `json:"generic_shield_pct"`
	Guard              uint32                    `json:"guard"`
	TalentDmgPct       uint32                    `json:"talent_dmg_pct"`
	TalentReducePct    uint32                    `json:"talent_reduce_pct"`
	ElementAdvPct      uint32                    `json:"element_adv_pct"`
	ElementPenaltyPct  uint32                    `json:"element_penalty_pct"`
	SkillIDs           []uint32                  `json:"skill_ids"`
	SkinID             string                    `json:"skin_id"`
	EquippedItems      []AdminPlayerEquippedItem `json:"equipped_items"`
	LastLoginAt        *time.Time                `json:"last_login_at"`
	CreatedAt          time.Time                 `json:"created_at"`
	UpdatedAt          time.Time                 `json:"updated_at"`
}

// AdminPlayerEquippedItem 描述后台玩家详情页里的单个装备槽状态。
// 无论槽位是否已佩戴，都返回一条记录，避免前端自己补齐“空槽位”。
type AdminPlayerEquippedItem struct {
	EquipSlot      string `json:"equip_slot"`
	EquipSlotLabel string `json:"equip_slot_label"`
	ItemUID        string `json:"item_uid"`
	ItemID         uint64 `json:"item_id"`
	ItemName       string `json:"item_name"`
	EnhanceLevel   uint32 `json:"enhance_level"`
	IsEmpty        bool   `json:"is_empty"`
}

// DefaultAdminPlayerEquippedItems 返回玩家详情页默认展示的全槽位空列表。
func DefaultAdminPlayerEquippedItems() []AdminPlayerEquippedItem {
	items := make([]AdminPlayerEquippedItem, 0, len(adminPlayerEquipSlots))
	for _, slot := range adminPlayerEquipSlots {
		items = append(items, AdminPlayerEquippedItem{
			EquipSlot:      slot.key,
			EquipSlotLabel: slot.label,
			ItemUID:        "",
			ItemID:         0,
			ItemName:       "",
			EnhanceLevel:   0,
			IsEmpty:        true,
		})
	}
	return items
}

var adminPlayerEquipSlots = []struct {
	key   string
	label string
}{
	{key: "weapon", label: "武器"},
	{key: "class_weapon", label: "职业武器"},
	{key: "hat", label: "帽子"},
	{key: "clothes", label: "衣服"},
	{key: "pants", label: "裤子"},
	{key: "shoes", label: "鞋子"},
	{key: "necklace", label: "项链"},
	{key: "ring", label: "戒指"},
	{key: "hero_ring", label: "英雄之戒"},
	{key: "badge", label: "徽章"},
	{key: "charm", label: "护符"},
	{key: "medicine_pouch", label: "药囊"},
	{key: "guardian_ring", label: "守护之戒"},
	{key: "class_badge", label: "职业徽章"},
	{key: "costume", label: "时装"},
	{key: "element_bracelet", label: "元素手镯"},
	{key: "rebirth_stone", label: "转生之石"},
}

func AdminPlayerStatusText(status uint32) string {
	switch status {
	case 1:
		return "NORMAL"
	case 2:
		return "BANNED"
	case 0:
		return "DELETED"
	default:
		return "UNKNOWN"
	}
}
