package npcdialogue

import (
	"strings"
	"time"
)

// AdminDialogueListQuery 描述后台 NPC 剧情列表的筛选条件。
type AdminDialogueListQuery struct {
	EntityID uint64
	EntryID  string
	Status   *uint32
	Page     uint32
	PageSize uint32
}

// Normalize 统一整理后台剧情列表筛选条件，避免分页参数失控。
func (q AdminDialogueListQuery) Normalize() AdminDialogueListQuery {
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

// AdminDialogueOptionInput 描述后台提交的单条剧情选项配置。
type AdminDialogueOptionInput struct {
	OptionID     string                  `json:"option_id"`
	OptionText   string                  `json:"option_text"`
	OptionFormat string                  `json:"option_format"`
	NextNodeID   string                  `json:"next_node_id"`
	SortOrder    uint32                  `json:"sort_order"`
	Conditions   AdminDialogueConditions `json:"conditions"`
}

// Normalize 统一去掉前后空格并补齐默认展示格式。
func (input AdminDialogueOptionInput) Normalize() AdminDialogueOptionInput {
	input.OptionID = strings.TrimSpace(input.OptionID)
	input.OptionText = strings.TrimSpace(input.OptionText)
	input.OptionFormat = strings.TrimSpace(input.OptionFormat)
	input.NextNodeID = strings.TrimSpace(input.NextNodeID)
	if input.OptionFormat == "" {
		input.OptionFormat = "plain"
	}
	input.Conditions = input.Conditions.Normalize()
	return input
}

// AdminDialogueNodeInput 描述后台提交的单个剧情节点配置；options 只在 choice 节点使用。
type AdminDialogueNodeInput struct {
	NodeID               string                     `json:"node_id"`
	NodeType             string                     `json:"node_type"`
	Speaker              string                     `json:"speaker"`
	Content              string                     `json:"content"`
	ContentFormat        string                     `json:"content_format"`
	PortraitKey          string                     `json:"portrait_key"`
	NextNodeID           string                     `json:"next_node_id"`
	ClientAnimationKey   string                     `json:"client_animation_key"`
	ClientAnimationBlock bool                       `json:"client_animation_block"`
	SortOrder            uint32                     `json:"sort_order"`
	Conditions           AdminDialogueConditions    `json:"conditions"`
	Effects              AdminDialogueEffects       `json:"effects"`
	Options              []AdminDialogueOptionInput `json:"options"`
}

// Normalize 统一整理节点字段和选项数组，保证后台保存前格式稳定。
func (input AdminDialogueNodeInput) Normalize() AdminDialogueNodeInput {
	input.NodeID = strings.TrimSpace(input.NodeID)
	input.NodeType = strings.TrimSpace(input.NodeType)
	input.Speaker = strings.TrimSpace(input.Speaker)
	input.Content = strings.TrimSpace(input.Content)
	input.ContentFormat = strings.TrimSpace(input.ContentFormat)
	input.PortraitKey = strings.TrimSpace(input.PortraitKey)
	input.NextNodeID = strings.TrimSpace(input.NextNodeID)
	input.ClientAnimationKey = strings.TrimSpace(input.ClientAnimationKey)
	if input.ContentFormat == "" {
		input.ContentFormat = "plain"
	}
	input.Conditions = input.Conditions.Normalize()
	input.Effects = input.Effects.Normalize()
	result := make([]AdminDialogueOptionInput, 0, len(input.Options))
	for _, option := range input.Options {
		result = append(result, option.Normalize())
	}
	input.Options = result
	return input
}

// AdminCreateDialogueInput 描述后台新增 NPC 剧情聚合配置时允许维护的字段。
type AdminCreateDialogueInput struct {
	EntityID     uint64                   `json:"entity_id"`
	EntryID      string                   `json:"entry_id"`
	DialogueCode string                   `json:"dialogue_code"`
	Title        string                   `json:"title"`
	StartNodeID  string                   `json:"start_node_id"`
	Version      int32                    `json:"version"`
	Status       uint32                   `json:"status"`
	Nodes        []AdminDialogueNodeInput `json:"nodes"`
}

// Normalize 统一整理后台新增剧情聚合配置。
func (input AdminCreateDialogueInput) Normalize() AdminCreateDialogueInput {
	input.EntryID = strings.TrimSpace(input.EntryID)
	input.DialogueCode = strings.TrimSpace(input.DialogueCode)
	input.Title = strings.TrimSpace(input.Title)
	input.StartNodeID = strings.TrimSpace(input.StartNodeID)
	if input.Version == 0 {
		input.Version = 1
	}
	if input.Status == 0 {
		input.Status = 1
	}
	result := make([]AdminDialogueNodeInput, 0, len(input.Nodes))
	for _, node := range input.Nodes {
		result = append(result, node.Normalize())
	}
	input.Nodes = result
	return input
}

// AdminUpdateDialogueInput 描述后台编辑 NPC 剧情聚合配置时允许维护的字段。
type AdminUpdateDialogueInput struct {
	EntityID     uint64                   `json:"entity_id"`
	DialogueCode string                   `json:"dialogue_code"`
	Title        string                   `json:"title"`
	StartNodeID  string                   `json:"start_node_id"`
	Version      int32                    `json:"version"`
	Status       uint32                   `json:"status"`
	Nodes        []AdminDialogueNodeInput `json:"nodes"`
}

// Normalize 统一整理后台编辑剧情聚合配置。
func (input AdminUpdateDialogueInput) Normalize() AdminUpdateDialogueInput {
	input.DialogueCode = strings.TrimSpace(input.DialogueCode)
	input.Title = strings.TrimSpace(input.Title)
	input.StartNodeID = strings.TrimSpace(input.StartNodeID)
	if input.Version == 0 {
		input.Version = 1
	}
	if input.Status == 0 {
		input.Status = 1
	}
	result := make([]AdminDialogueNodeInput, 0, len(input.Nodes))
	for _, node := range input.Nodes {
		result = append(result, node.Normalize())
	}
	input.Nodes = result
	return input
}

// AdminDialogueSummary 描述后台剧情列表中的单条汇总记录。
type AdminDialogueSummary struct {
	EntityID     uint64    `json:"entity_id"`
	EntryID      string    `json:"entry_id"`
	DialogueCode string    `json:"dialogue_code"`
	Title        string    `json:"title"`
	StartNodeID  string    `json:"start_node_id"`
	Version      int32     `json:"version"`
	Status       uint32    `json:"status"`
	UpdatedAt    time.Time `json:"updated_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// AdminDialogueList 描述后台剧情列表分页结果。
type AdminDialogueList struct {
	Items    []AdminDialogueSummary `json:"items"`
	Total    uint64                 `json:"total"`
	Page     uint32                 `json:"page"`
	PageSize uint32                 `json:"page_size"`
}

// AdminDialogueOptionDetail 描述后台详情页中的单条剧情选项。
type AdminDialogueOptionDetail struct {
	OptionID     string `json:"option_id"`
	OptionText   string `json:"option_text"`
	OptionFormat string `json:"option_format"`
	NextNodeID   string                  `json:"next_node_id"`
	SortOrder    uint32                  `json:"sort_order"`
	Conditions   AdminDialogueConditions `json:"conditions"`
}

// AdminDialogueNodeDetail 描述后台详情页中的单个剧情节点和其选项列表。
type AdminDialogueNodeDetail struct {
	NodeID               string                      `json:"node_id"`
	NodeType             string                      `json:"node_type"`
	Speaker              string                      `json:"speaker"`
	Content              string                      `json:"content"`
	ContentFormat        string                      `json:"content_format"`
	PortraitKey          string                      `json:"portrait_key"`
	NextNodeID           string                      `json:"next_node_id"`
	ClientAnimationKey   string                      `json:"client_animation_key"`
	ClientAnimationBlock bool                        `json:"client_animation_block"`
	SortOrder            uint32                      `json:"sort_order"`
	Conditions           AdminDialogueConditions     `json:"conditions"`
	Effects              AdminDialogueEffects        `json:"effects"`
	Options              []AdminDialogueOptionDetail `json:"options"`
}

// AdminDialogueDetail 描述后台剧情详情页中的整段剧情聚合配置。
type AdminDialogueDetail struct {
	EntityID     uint64                    `json:"entity_id"`
	EntryID      string                    `json:"entry_id"`
	DialogueCode string                    `json:"dialogue_code"`
	Title        string                    `json:"title"`
	StartNodeID  string                    `json:"start_node_id"`
	Version      int32                     `json:"version"`
	Status       uint32                    `json:"status"`
	UpdatedAt    time.Time                 `json:"updated_at"`
	CreatedAt    time.Time                 `json:"created_at"`
	Nodes        []AdminDialogueNodeDetail `json:"nodes"`
}
