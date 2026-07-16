package npcdialogue

import (
	"encoding/json"
	"errors"
)

var (
	// ErrDialogueNotFound 表示当前 NPC 菜单项没有配置结构化剧情。
	ErrDialogueNotFound = errors.New("npc dialogue not found")
	// ErrDialogueNodeNotFound 表示剧情节点缺失或 next_node_id 指向了无效节点。
	ErrDialogueNodeNotFound = errors.New("npc dialogue node not found")
	// ErrDialogueSessionNotFound 表示客户端继续剧情时，服务端没有找到对应会话。
	ErrDialogueSessionNotFound = errors.New("npc dialogue session not found")
	// ErrDialogueSessionMismatch 表示客户端上传的 dialogue_id / node_id 与当前服务端会话不一致。
	ErrDialogueSessionMismatch = errors.New("npc dialogue session mismatch")
	// ErrDialogueOptionInvalid 表示客户端提交了当前节点不存在的选项。
	ErrDialogueOptionInvalid = errors.New("npc dialogue option invalid")
	// ErrInvalidAdminDialogueInput 表示后台提交的剧情聚合配置字段不完整。
	ErrInvalidAdminDialogueInput = errors.New("invalid admin dialogue input")
	// ErrAdminDialogueConflict 表示后台创建剧情配置时命中了唯一键。
	ErrAdminDialogueConflict = errors.New("admin dialogue conflict")
	// ErrDialogueMenuEntryNotFound 表示后台要绑定的 NPC 菜单项不存在，不能生成孤儿剧情配置。
	ErrDialogueMenuEntryNotFound = errors.New("npc dialogue menu entry not found")
)

const (
	// SessionStatusActive 表示玩家当前仍在进行这段剧情。
	SessionStatusActive int16 = 1
	// SessionStatusEnded 表示剧情已自然结束，服务端会尽快清理会话。
	SessionStatusEnded int16 = 2
)

const (
	// NodeTypeLine 表示可直接展示在对话框中的普通台词节点。
	NodeTypeLine string = "line"
	// NodeTypeChoice 表示需要客户端展示选项按钮的分支节点。
	NodeTypeChoice string = "choice"
	// NodeTypeAction 表示客户端需要播放本地内置剧情动画的节点。
	NodeTypeAction string = "action"
	// NodeTypeEnd 表示剧情结束节点，客户端收到后关闭面板即可。
	NodeTypeEnd string = "end"
)

// Dialogue 描述某个 NPC 菜单项绑定的一整段结构化剧情配置。
type Dialogue struct {
	DialogueID   int64
	EntityID     uint64
	EntryID      string
	DialogueCode string
	Title        string
	StartNodeID  string
	Version      int32
	Status       int16
}

// DialogueNode 表示剧情中的单个节点；第一期只支持台词、选项、动画和结束四种类型。
type DialogueNode struct {
	DialogueID           int64
	NodeID               string
	NodeType             string
	Speaker              string
	Content              string
	ContentFormat        string
	PortraitKey          string
	NextNodeID           string
	ClientAnimationKey   string
	ClientAnimationBlock bool
	EffectsJSON          json.RawMessage
	ConditionsJSON       json.RawMessage
}

// DialogueOption 表示 choice 节点下的一条分支选项。
type DialogueOption struct {
	DialogueID     int64
	NodeID         string
	OptionID       string
	OptionText     string
	OptionFormat   string
	NextNodeID     string
	ConditionsJSON json.RawMessage
}

// DialogueSession 记录玩家当前在服务端权威剧情中的推进位置。
type DialogueSession struct {
	PlayerID      uint64
	EntityID      uint64
	DialogueID    int64
	CurrentNodeID string
	Status        int16
}

// RuntimeNode 是发给客户端的运行态节点结构，已经把选项聚合到节点里。
type RuntimeNode struct {
	DialogueID           int64
	NodeID               string
	NodeType             string
	Speaker              string
	Content              string
	ContentFormat        string
	PortraitKey          string
	ClientAnimationKey   string
	ClientAnimationBlock bool
	Options              []DialogueOption
	IsEnd                bool
	EffectNotice         string
	EffectQuestEvent     string
	EffectAcceptQuestID  uint64
	EffectSubmitQuestID  uint64
	EffectGrantItems     []EffectGrantItem
}
