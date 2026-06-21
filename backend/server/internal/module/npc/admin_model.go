package npc

import (
	"errors"
	"strings"
	"time"

	"pocket-pet-remake/server/internal/module/npcdialogue"
)

var (
	// ErrAdminNPCNotFound 表示后台指定的 NPC 实体不存在。
	ErrAdminNPCNotFound = errors.New("admin npc not found")
	// ErrAdminNPCMenuEntryNotFound 表示后台指定的 NPC 菜单项不存在。
	ErrAdminNPCMenuEntryNotFound = errors.New("admin npc menu entry not found")
	// ErrInvalidAdminNPCInput 表示后台提交的 NPC 配置字段不完整。
	ErrInvalidAdminNPCInput = errors.New("invalid admin npc input")
	// ErrAdminNPCConflict 表示后台新增或编辑 NPC 配置时命中了唯一键。
	ErrAdminNPCConflict = errors.New("admin npc conflict")
)

// AdminEntityListQuery 描述后台 NPC 实体列表的筛选条件。
// 这里把 scene_id 一起暴露出来，方便按地图查看 NPC 分布。
type AdminEntityListQuery struct {
	EntityID   uint64
	SceneID    uint32
	EntityType *uint32
	Status     *uint32
	Name       string
	Page       uint32
	PageSize   uint32
}

func (q AdminEntityListQuery) Normalize() AdminEntityListQuery {
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

// AdminMenuEntryListQuery 描述后台 NPC 菜单配置列表的筛选条件。
type AdminMenuEntryListQuery struct {
	EntityID uint64
	EntryID  string
	Status   *uint32
	Page     uint32
	PageSize uint32
}

func (q AdminMenuEntryListQuery) Normalize() AdminMenuEntryListQuery {
	q.EntryID = strings.TrimSpace(q.EntryID)
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

// AdminWorldSceneSummary 描述后台可选地图场景；坐标由客户端场景资源维护，这里只提供 scene_id 与展示名。
type AdminWorldSceneSummary struct {
	SceneID   uint32 `json:"scene_id"`
	SceneCode string `json:"scene_code"`
	SceneName string `json:"scene_name"`
	Status    uint32 `json:"status"`
}

// AdminCreateEntityInput 描述后台新增 NPC/世界实体时允许维护的字段。
// 坐标与朝向由客户端场景资源维护，服务端只记录 entity 归属 scene_id。
type AdminCreateEntityInput struct {
	EntityID    uint64 `json:"entity_id"`
	EntityCode  string `json:"entity_code"`
	DisplayName string `json:"display_name"`
	EntityType  uint32 `json:"entity_type"`
	SceneID     uint32 `json:"scene_id"`
	Status      uint32 `json:"status"`
}

func (input AdminCreateEntityInput) Normalize() AdminCreateEntityInput {
	input.EntityCode = strings.TrimSpace(input.EntityCode)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.EntityType == 0 {
		input.EntityType = 2
	}
	if input.Status == 0 {
		input.Status = 1
	}
	return input
}

// AdminUpdateEntityInput 描述后台编辑实体分布时允许调整的字段。
type AdminUpdateEntityInput struct {
	EntityCode  string `json:"entity_code"`
	DisplayName string `json:"display_name"`
	EntityType  uint32 `json:"entity_type"`
	SceneID     uint32 `json:"scene_id"`
	Status      uint32 `json:"status"`
}

func (input AdminUpdateEntityInput) Normalize() AdminUpdateEntityInput {
	input.EntityCode = strings.TrimSpace(input.EntityCode)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.EntityType == 0 {
		input.EntityType = 2
	}
	if input.Status == 0 {
		input.Status = 1
	}
	return input
}

// AdminCreateMenuEntryInput 描述后台新增 NPC 菜单项时允许维护的字段。
type AdminCreateMenuEntryInput struct {
	EntityID         uint64 `json:"entity_id"`
	EntryID          string `json:"entry_id"`
	EntryType        string `json:"entry_type"`
	Title            string `json:"title"`
	Subtitle         string `json:"subtitle"`
	State            string `json:"state"`
	Priority         uint32 `json:"priority"`
	SortOrder        uint32 `json:"sort_order"`
	ActionResultType        string `json:"action_result_type"`
	ActionNotice            string `json:"action_notice"`
	BattleEncounterEntityID uint64 `json:"battle_encounter_entity_id"`
	LinkedQuestID           uint64 `json:"linked_quest_id"`
	Conditions              npcdialogue.AdminDialogueConditions `json:"conditions"`
	Status                  uint32 `json:"status"`
}

func (input AdminCreateMenuEntryInput) Normalize() AdminCreateMenuEntryInput {
	input.EntryID = strings.TrimSpace(input.EntryID)
	input.EntryType = strings.TrimSpace(input.EntryType)
	input.Title = strings.TrimSpace(input.Title)
	input.Subtitle = strings.TrimSpace(input.Subtitle)
	input.State = strings.TrimSpace(input.State)
	input.ActionResultType = strings.TrimSpace(input.ActionResultType)
	input.ActionNotice = strings.TrimSpace(input.ActionNotice)
	if input.State == "" {
		input.State = "available"
	}
	normalizeMenuEntryTypeDefaults(&input.EntryType, &input.ActionResultType, &input.Title, input.EntityID, &input.BattleEncounterEntityID)
	input.Conditions = input.Conditions.Normalize()
	if input.ActionResultType == "" {
		input.ActionResultType = "notice"
	}
	if input.Status == 0 {
		input.Status = 1
	}
	return input
}

// AdminUpdateMenuEntryInput 描述后台编辑 NPC 菜单项时允许调整的字段。
type AdminUpdateMenuEntryInput struct {
	EntityID         uint64 `json:"entity_id"`
	EntryType        string `json:"entry_type"`
	Title            string `json:"title"`
	Subtitle         string `json:"subtitle"`
	State            string `json:"state"`
	Priority         uint32 `json:"priority"`
	SortOrder        uint32 `json:"sort_order"`
	ActionResultType        string `json:"action_result_type"`
	ActionNotice            string `json:"action_notice"`
	BattleEncounterEntityID uint64 `json:"battle_encounter_entity_id"`
	LinkedQuestID           uint64 `json:"linked_quest_id"`
	Conditions              npcdialogue.AdminDialogueConditions `json:"conditions"`
	Status                  uint32 `json:"status"`
}

func (input AdminUpdateMenuEntryInput) Normalize() AdminUpdateMenuEntryInput {
	input.EntryType = strings.TrimSpace(input.EntryType)
	input.Title = strings.TrimSpace(input.Title)
	input.Subtitle = strings.TrimSpace(input.Subtitle)
	input.State = strings.TrimSpace(input.State)
	input.ActionResultType = strings.TrimSpace(input.ActionResultType)
	input.ActionNotice = strings.TrimSpace(input.ActionNotice)
	if input.State == "" {
		input.State = "available"
	}
	normalizeMenuEntryTypeDefaults(&input.EntryType, &input.ActionResultType, &input.Title, input.EntityID, &input.BattleEncounterEntityID)
	input.Conditions = input.Conditions.Normalize()
	if input.ActionResultType == "" {
		input.ActionResultType = "notice"
	}
	if input.Status == 0 {
		input.Status = 1
	}
	return input
}

// normalizeMenuEntryTypeDefaults 根据入口类型补齐默认动作类型与战斗绑定。
func normalizeMenuEntryTypeDefaults(entryType *string, actionResultType *string, title *string, entityID uint64, battleEncounterEntityID *uint64) {
	if *entryType == "battle" || *actionResultType == "battle" {
		if *entryType == "" {
			*entryType = "battle"
		}
		if *actionResultType == "" {
			*actionResultType = "battle"
		}
		if *title == "" {
			*title = "挑战"
		}
		if *battleEncounterEntityID == 0 {
			*battleEncounterEntityID = entityID
		}
	}
	if *entryType == "dialog" {
		if *actionResultType == "" || *actionResultType == "dialog" {
			*actionResultType = "dialogue"
		}
	}
	if *entryType == "shop" {
		if *actionResultType == "" {
			*actionResultType = "shop"
		}
	}
	if *entryType == "quest" {
		if *actionResultType == "" {
			*actionResultType = "quest_accept"
		}
	}
}

type AdminEntitySummary struct {
	EntityID    uint64    `json:"entity_id"`
	EntityCode  string    `json:"entity_code"`
	DisplayName string    `json:"display_name"`
	EntityType  uint32    `json:"entity_type"`
	SceneID     uint32    `json:"scene_id"`
	SceneName   string    `json:"scene_name"`
	Status      uint32    `json:"status"`
	StatusText  string    `json:"status_text"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type AdminEntityList struct {
	Items    []AdminEntitySummary `json:"items"`
	Total    uint64               `json:"total"`
	Page     uint32               `json:"page"`
	PageSize uint32               `json:"page_size"`
}

type AdminEntityDetail struct {
	EntityID    uint64    `json:"entity_id"`
	EntityCode  string    `json:"entity_code"`
	DisplayName string    `json:"display_name"`
	EntityType  uint32    `json:"entity_type"`
	SceneID     uint32    `json:"scene_id"`
	SceneName   string    `json:"scene_name"`
	Status      uint32    `json:"status"`
	StatusText  string    `json:"status_text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AdminMenuEntrySummary struct {
	EntityID         uint64    `json:"entity_id"`
	EntryID          string    `json:"entry_id"`
	EntryType        string    `json:"entry_type"`
	Title            string    `json:"title"`
	Subtitle         string    `json:"subtitle"`
	State            string    `json:"state"`
	Priority         uint32    `json:"priority"`
	SortOrder        uint32    `json:"sort_order"`
	ActionResultType        string    `json:"action_result_type"`
	BattleEncounterEntityID uint64    `json:"battle_encounter_entity_id"`
	LinkedQuestID           uint64    `json:"linked_quest_id"`
	Status                  uint32    `json:"status"`
	StatusText       string    `json:"status_text"`
	UpdatedAt        time.Time `json:"updated_at"`
	CreatedAt        time.Time `json:"created_at"`
}

type AdminMenuEntryList struct {
	Items    []AdminMenuEntrySummary `json:"items"`
	Total    uint64                  `json:"total"`
	Page     uint32                  `json:"page"`
	PageSize uint32                  `json:"page_size"`
}

type AdminMenuEntryDetail struct {
	EntityID         uint64    `json:"entity_id"`
	EntryID          string    `json:"entry_id"`
	EntryType        string    `json:"entry_type"`
	Title            string    `json:"title"`
	Subtitle         string    `json:"subtitle"`
	State            string    `json:"state"`
	Priority         uint32    `json:"priority"`
	SortOrder        uint32    `json:"sort_order"`
	ActionResultType        string    `json:"action_result_type"`
	ActionNotice            string    `json:"action_notice"`
	BattleEncounterEntityID uint64    `json:"battle_encounter_entity_id"`
	LinkedQuestID           uint64    `json:"linked_quest_id"`
	Conditions              npcdialogue.AdminDialogueConditions `json:"conditions"`
	Status                  uint32    `json:"status"`
	StatusText       string    `json:"status_text"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func AdminNPCStatusText(status uint32) string {
	switch status {
	case 1:
		return "启用"
	case 0:
		return "停用"
	default:
		return "未知"
	}
}
