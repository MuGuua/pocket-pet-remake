package player

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrPlayerNotFound        = errors.New("player not found")
	ErrPlayerNameDuplicated  = errors.New("player name duplicated")
	ErrAccountNameDuplicated = errors.New("account name duplicated")
	ErrInvalidAdminInput     = errors.New("invalid admin player input")
)

// DefaultPlayerSkinID 是玩家尚未配置形象时的服务端默认资源 ID。
const DefaultPlayerSkinID = "初始形象男_001"

type Profile struct {
	PlayerID           uint64
	Name               string
	Level              uint32
	Exp                uint64
	Gold               uint32
	SceneID            uint32
	PosX               int32
	PosY               int32
	HP                 uint32
	HPMax              uint32
	Energy             uint32
	EnergyMax          uint32
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
	SkillIDs           []uint32
	// SkinID 是当前玩家形象资源 ID，世界与战斗表现层共用。
	SkinID string
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
	Energy      uint32   `json:"energy"`
	EnergyMax   uint32   `json:"energy_max"`
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
	input.AccountName = strings.TrimSpace(input.AccountName)
	input.Password = strings.TrimSpace(input.Password)
	input.Name = strings.TrimSpace(input.Name)
	if input.Level == 0 {
		input.Level = 1
	}
	if input.SceneID == 0 {
		input.SceneID = 1
	}
	if input.HP == 0 {
		input.HP = 100
	}
	if input.HPMax == 0 {
		input.HPMax = input.HP
	}
	if input.Energy == 0 {
		input.Energy = 100
	}
	if input.EnergyMax == 0 {
		input.EnergyMax = input.Energy
	}
	input.SkinID = strings.TrimSpace(input.SkinID)
	return input
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
	Energy    uint32   `json:"energy"`
	EnergyMax uint32   `json:"energy_max"`
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
	if input.EnergyMax == 0 {
		input.EnergyMax = input.Energy
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
	Energy      uint32     `json:"energy"`
	EnergyMax   uint32     `json:"energy_max"`
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
	PlayerID           uint64     `json:"player_id"`
	AccountID          uint64     `json:"account_id"`
	AccountName        string     `json:"account_name"`
	Name               string     `json:"name"`
	Level              uint32     `json:"level"`
	Exp                uint64     `json:"exp"`
	Gold               uint64     `json:"gold"`
	Status             uint32     `json:"status"`
	StatusText         string     `json:"status_text"`
	SceneID            uint32     `json:"scene_id"`
	PosX               int32      `json:"pos_x"`
	PosY               int32      `json:"pos_y"`
	HP                 uint32     `json:"hp"`
	HPMax              uint32     `json:"hp_max"`
	Energy             uint32     `json:"energy"`
	EnergyMax          uint32     `json:"energy_max"`
	ATK                uint32     `json:"atk"`
	DEF                uint32     `json:"def"`
	SPD                uint32     `json:"spd"`
	MANA               uint32     `json:"mana"`
	HitPct             uint32     `json:"hit_pct"`
	DodgePct           uint32     `json:"dodge_pct"`
	CritRatePct        uint32     `json:"crit_rate_pct"`
	CritDmgPct         uint32     `json:"crit_dmg_pct"`
	PhysicalResistPct  uint32     `json:"physical_resist_pct"`
	SkillResistPct     uint32     `json:"skill_resist_pct"`
	ConfusionResistPct uint32     `json:"confusion_resist_pct"`
	SleepResistPct     uint32     `json:"sleep_resist_pct"`
	ParalysisResistPct uint32     `json:"paralysis_resist_pct"`
	SealResistPct      uint32     `json:"seal_resist_pct"`
	CurseResistPct     uint32     `json:"curse_resist_pct"`
	CritResistPct      uint32     `json:"crit_resist_pct"`
	CritDmgResistPct   uint32     `json:"crit_dmg_resist_pct"`
	CharacterResistPct uint32     `json:"character_resist_pct"`
	PetResistPct       uint32     `json:"pet_resist_pct"`
	MercenaryResistPct uint32     `json:"mercenary_resist_pct"`
	GenericShieldPct   uint32     `json:"generic_shield_pct"`
	SkillIDs           []uint32   `json:"skill_ids"`
	SkinID             string     `json:"skin_id"`
	LastLoginAt        *time.Time `json:"last_login_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
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
