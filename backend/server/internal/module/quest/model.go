package quest

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrAdminQuestTemplateNotFound 表示后台指定的任务模板不存在。
	ErrAdminQuestTemplateNotFound = errors.New("admin quest template not found")
	// ErrAdminPlayerQuestNotFound 表示后台指定的玩家任务记录不存在。
	ErrAdminPlayerQuestNotFound = errors.New("admin player quest not found")
	// ErrInvalidAdminQuestInput 表示后台任务请求体缺少必要字段或字段非法。
	ErrInvalidAdminQuestInput = errors.New("invalid admin quest input")
	// ErrAdminQuestConflict 表示后台创建或编辑任务时命中了唯一键冲突。
	ErrAdminQuestConflict = errors.New("admin quest conflict")
)

type Template struct {
	QuestID     uint64
	QuestType   string
	Title       string
	Description string
	AcceptMode  string
	SubmitMode  string
	StartNPCID  uint64
	SubmitNPCID uint64
	AutoTrack   bool
	PreQuestIDs []uint64
	Objectives  []ObjectiveTemplate
	Rewards     []Reward
}

type ObjectiveTemplate struct {
	ObjectiveID         uint64
	EventType           string
	Description         string
	TargetValue         uint32
	TargetSelector      map[string]any
	AutoCompleteOnMatch bool
}

type PlayerQuest struct {
	PlayerID uint64
	QuestID  uint64
	State    string
	Tracked  bool
}

type PlayerObjective struct {
	PlayerID     uint64
	QuestID      uint64
	ObjectiveID  uint64
	Description  string
	CurrentValue uint32
	TargetValue  uint32
	Completed    bool
}

type Summary struct {
	QuestID     uint64
	QuestType   string
	State       string
	Tracked     bool
	StartNPCID  uint64
	SubmitNPCID uint64
	Title       string
	Description string
	Objectives  []ObjectiveSummary
}

type ObjectiveSummary struct {
	ObjectiveID uint64
	Description string
	Current     uint32
	Target      uint32
	Completed   bool
}

type Event struct {
	PlayerID  uint64
	EventType string
	SceneID   uint32
	NPCID     uint64
	Count     uint32
	Meta      map[string]any
}

// Reward 描述任务模板里声明的一条奖励配置。
// 当前先覆盖金币、经验、道具、宠物和功能解锁几类基础字段，后续可继续扩展更多经济类型。
type Reward struct {
	Type   string `json:"type"`
	Value  uint64 `json:"value"`
	ItemID uint64 `json:"item_id"`
	Count  uint64 `json:"count"`
	PetID  uint64 `json:"pet_id"`
}

// SubmitResult 统一返回任务提交后的状态与本次已发放奖励。
// WebSocket handler 可以直接基于这里的奖励结果继续推送钱包变化，避免再重复推导模板配置。
type SubmitResult struct {
	Summary Summary  `json:"summary"`
	Rewards []Reward `json:"rewards"`
}

// AdminTemplateListQuery 描述后台任务模板分页与筛选条件。
// 后台列表始终以数据库为准，避免前端本地再拼接假筛选结果。
type AdminTemplateListQuery struct {
	QuestID   uint64
	QuestType string
	Title     string
	Status    *uint32
	Page      uint32
	PageSize  uint32
}

func (q AdminTemplateListQuery) Normalize() AdminTemplateListQuery {
	q.QuestType = strings.TrimSpace(q.QuestType)
	q.Title = strings.TrimSpace(q.Title)
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

// AdminPlayerQuestListQuery 描述后台玩家任务分页与筛选条件。
type AdminPlayerQuestListQuery struct {
	RecordID uint64
	PlayerID uint64
	QuestID  uint64
	State    string
	Tracked  *bool
	Page     uint32
	PageSize uint32
}

func (q AdminPlayerQuestListQuery) Normalize() AdminPlayerQuestListQuery {
	q.State = strings.TrimSpace(q.State)
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

// ObjectiveGuideInput 描述单个任务阶段在客户端/运营侧的引导信息。
// menu_entry_id 与 dialogue_entry_id 对应 npc_menu_entry.entry_id，便于策划把阶段与 NPC 菜单剧情绑定。
type ObjectiveGuideInput struct {
	SceneID         uint32 `json:"scene_id,omitempty"`
	NPCID           uint64 `json:"npc_id,omitempty"`
	Text            string `json:"text,omitempty"`
	MenuEntryID     uint64 `json:"menu_entry_id,omitempty"`
	DialogueEntryID uint64 `json:"dialogue_entry_id,omitempty"`
}

// AdminObjectiveInput 让后台可以直接维护模板目标定义。
// 多阶段任务通过多个 objective_id 顺序排列；guide 字段用于记录 NPC/菜单/剧情绑定提示。
type AdminObjectiveInput struct {
	ObjectiveID    uint64              `json:"objective_id"`
	EventType      string              `json:"event_type"`
	Description    string              `json:"description"`
	TargetValue    uint32              `json:"target_value"`
	TargetSelector map[string]any      `json:"target_selector"`
	Guide          *ObjectiveGuideInput `json:"guide,omitempty"`
}

// AdminRewardInput 描述后台可配置的任务奖励，当前仅支持经验与物品。
type AdminRewardInput struct {
	Type   string `json:"type"`
	Value  uint64 `json:"value"`
	ItemID uint64 `json:"item_id"`
	Count  uint64 `json:"count"`
}

func (input AdminRewardInput) Normalize() AdminRewardInput {
	input.Type = strings.ToLower(strings.TrimSpace(input.Type))
	return input
}

// AdminCreateTemplateInput 描述后台新增任务模板时允许写入的持久化字段。
type AdminCreateTemplateInput struct {
	QuestID        uint64                `json:"quest_id"`
	Name           string                `json:"name"`
	QuestType      string                `json:"quest_type"`
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	Chapter        uint32                `json:"chapter"`
	SortOrder      uint32                `json:"sort_order"`
	AcceptMode     string                `json:"accept_mode"`
	SubmitMode     string                `json:"submit_mode"`
	AutoTrack      bool                  `json:"auto_track"`
	StartNPCID     uint64                `json:"start_npc_id"`
	SubmitNPCID    uint64                `json:"submit_npc_id"`
	MinPlayerLevel uint32                `json:"min_player_level"`
	Status         uint32                `json:"status"`
	PreQuestIDs    []uint64              `json:"pre_quest_ids"`
	Objectives     []AdminObjectiveInput `json:"objectives"`
	Rewards        []AdminRewardInput    `json:"rewards"`
}

func (input AdminCreateTemplateInput) Normalize() AdminCreateTemplateInput {
	input.Name = strings.TrimSpace(input.Name)
	input.QuestType = strings.TrimSpace(input.QuestType)
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.AcceptMode = strings.TrimSpace(input.AcceptMode)
	input.SubmitMode = strings.TrimSpace(input.SubmitMode)
	if input.QuestType == "" {
		input.QuestType = "MAIN"
	}
	if input.AcceptMode == "" {
		input.AcceptMode = "AUTO"
	}
	if input.SubmitMode == "" {
		input.SubmitMode = "AUTO"
	}
	if input.MinPlayerLevel == 0 {
		input.MinPlayerLevel = 1
	}
	if input.Status == 0 {
		input.Status = 1
	}
	return input
}

// AdminUpdateTemplateInput 描述后台编辑任务模板时允许调整的字段。
type AdminUpdateTemplateInput struct {
	Name           string                `json:"name"`
	QuestType      string                `json:"quest_type"`
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	Chapter        uint32                `json:"chapter"`
	SortOrder      uint32                `json:"sort_order"`
	AcceptMode     string                `json:"accept_mode"`
	SubmitMode     string                `json:"submit_mode"`
	AutoTrack      bool                  `json:"auto_track"`
	StartNPCID     uint64                `json:"start_npc_id"`
	SubmitNPCID    uint64                `json:"submit_npc_id"`
	MinPlayerLevel uint32                `json:"min_player_level"`
	Status         uint32                `json:"status"`
	PreQuestIDs    []uint64              `json:"pre_quest_ids"`
	Objectives     []AdminObjectiveInput `json:"objectives"`
	Rewards        []AdminRewardInput    `json:"rewards"`
}

func (input AdminUpdateTemplateInput) Normalize() AdminUpdateTemplateInput {
	input.Name = strings.TrimSpace(input.Name)
	input.QuestType = strings.TrimSpace(input.QuestType)
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.AcceptMode = strings.TrimSpace(input.AcceptMode)
	input.SubmitMode = strings.TrimSpace(input.SubmitMode)
	if input.QuestType == "" {
		input.QuestType = "MAIN"
	}
	if input.AcceptMode == "" {
		input.AcceptMode = "AUTO"
	}
	if input.SubmitMode == "" {
		input.SubmitMode = "AUTO"
	}
	if input.MinPlayerLevel == 0 {
		input.MinPlayerLevel = 1
	}
	if input.Status == 0 {
		input.Status = 1
	}
	return input
}

type AdminTemplateSummary struct {
	QuestID        uint64    `json:"quest_id"`
	Name           string    `json:"name"`
	QuestType      string    `json:"quest_type"`
	Title          string    `json:"title"`
	Chapter        uint32    `json:"chapter"`
	SortOrder      uint32    `json:"sort_order"`
	AcceptMode     string    `json:"accept_mode"`
	SubmitMode     string    `json:"submit_mode"`
	AutoTrack      bool      `json:"auto_track"`
	MinPlayerLevel uint32    `json:"min_player_level"`
	Status         uint32    `json:"status"`
	StatusText     string    `json:"status_text"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type AdminTemplateList struct {
	Items    []AdminTemplateSummary `json:"items"`
	Total    uint64                 `json:"total"`
	Page     uint32                 `json:"page"`
	PageSize uint32                 `json:"page_size"`
}

type AdminTemplateDetail struct {
	QuestID        uint64                `json:"quest_id"`
	Name           string                `json:"name"`
	QuestType      string                `json:"quest_type"`
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	Chapter        uint32                `json:"chapter"`
	SortOrder      uint32                `json:"sort_order"`
	AcceptMode     string                `json:"accept_mode"`
	SubmitMode     string                `json:"submit_mode"`
	AutoTrack      bool                  `json:"auto_track"`
	StartNPCID     uint64                `json:"start_npc_id"`
	SubmitNPCID    uint64                `json:"submit_npc_id"`
	MinPlayerLevel uint32                `json:"min_player_level"`
	Status         uint32                `json:"status"`
	StatusText     string                `json:"status_text"`
	PreQuestIDs    []uint64              `json:"pre_quest_ids"`
	Objectives     []AdminObjectiveInput `json:"objectives"`
	Rewards        []AdminRewardInput    `json:"rewards"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

// AdminPlayerObjectiveInput 让后台可以直接修正玩家任务目标进度。
type AdminPlayerObjectiveInput struct {
	ObjectiveID  uint64 `json:"objective_id"`
	Description  string `json:"description"`
	CurrentValue uint32 `json:"current_value"`
	TargetValue  uint32 `json:"target_value"`
	Completed    bool   `json:"completed"`
}

type AdminCreatePlayerQuestInput struct {
	PlayerID      uint64                      `json:"player_id"`
	QuestID       uint64                      `json:"quest_id"`
	State         string                      `json:"state"`
	Tracked       bool                        `json:"tracked"`
	RewardClaimed bool                        `json:"reward_claimed"`
	Objectives    []AdminPlayerObjectiveInput `json:"objectives"`
}

func (input AdminCreatePlayerQuestInput) Normalize() AdminCreatePlayerQuestInput {
	input.State = strings.TrimSpace(input.State)
	if input.State == "" {
		input.State = StateAccepted
	}
	return input
}

type AdminUpdatePlayerQuestInput struct {
	PlayerID      uint64                      `json:"player_id"`
	QuestID       uint64                      `json:"quest_id"`
	State         string                      `json:"state"`
	Tracked       bool                        `json:"tracked"`
	RewardClaimed bool                        `json:"reward_claimed"`
	Objectives    []AdminPlayerObjectiveInput `json:"objectives"`
}

func (input AdminUpdatePlayerQuestInput) Normalize() AdminUpdatePlayerQuestInput {
	input.State = strings.TrimSpace(input.State)
	if input.State == "" {
		input.State = StateAccepted
	}
	return input
}

type AdminPlayerQuestSummary struct {
	RecordID      uint64     `json:"record_id"`
	PlayerID      uint64     `json:"player_id"`
	PlayerName    string     `json:"player_name"`
	QuestID       uint64     `json:"quest_id"`
	QuestTitle    string     `json:"quest_title"`
	QuestType     string     `json:"quest_type"`
	State         string     `json:"state"`
	Tracked       bool       `json:"tracked"`
	RewardClaimed bool       `json:"reward_claimed"`
	AcceptedAt    *time.Time `json:"accepted_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

type AdminPlayerQuestList struct {
	Items    []AdminPlayerQuestSummary `json:"items"`
	Total    uint64                    `json:"total"`
	Page     uint32                    `json:"page"`
	PageSize uint32                    `json:"page_size"`
}

type AdminPlayerQuestDetail struct {
	RecordID      uint64                      `json:"record_id"`
	PlayerID      uint64                      `json:"player_id"`
	PlayerName    string                      `json:"player_name"`
	QuestID       uint64                      `json:"quest_id"`
	QuestTitle    string                      `json:"quest_title"`
	QuestType     string                      `json:"quest_type"`
	State         string                      `json:"state"`
	Tracked       bool                        `json:"tracked"`
	RewardClaimed bool                        `json:"reward_claimed"`
	AcceptedAt    *time.Time                  `json:"accepted_at"`
	CompletedAt   *time.Time                  `json:"completed_at"`
	SubmittedAt   *time.Time                  `json:"submitted_at"`
	CreatedAt     time.Time                   `json:"created_at"`
	UpdatedAt     time.Time                   `json:"updated_at"`
	Objectives    []AdminPlayerObjectiveInput `json:"objectives"`
}

func AdminQuestTemplateStatusText(status uint32) string {
	switch status {
	case 1:
		return "启用"
	case 0:
		return "停用"
	default:
		return "未知"
	}
}
