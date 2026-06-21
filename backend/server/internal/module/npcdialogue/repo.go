package npcdialogue

import "context"

// Repository 负责从数据库读取 NPC 剧情配置，并维护玩家当前剧情会话位置。
type Repository interface {
	FindDialogueByEntityEntry(ctx context.Context, entityID uint64, entryID string) (*Dialogue, error)
	MenuEntryExists(ctx context.Context, entityID uint64, entryID string) (bool, error)
	FindNode(ctx context.Context, dialogueID int64, nodeID string) (*DialogueNode, error)
	ListOptions(ctx context.Context, dialogueID int64, nodeID string) ([]DialogueOption, error)
	UpsertSession(ctx context.Context, session DialogueSession) error
	FindSessionByPlayerID(ctx context.Context, playerID uint64) (*DialogueSession, error)
	DeleteSession(ctx context.Context, playerID uint64) error
	ListDialoguesForAdmin(ctx context.Context, query AdminDialogueListQuery) (*AdminDialogueList, error)
	FindDialogueDetailForAdmin(ctx context.Context, entityID uint64, entryID string) (*AdminDialogueDetail, error)
	CreateDialogueForAdmin(ctx context.Context, input AdminCreateDialogueInput) (*AdminDialogueDetail, error)
	UpdateDialogueForAdmin(ctx context.Context, entityID uint64, entryID string, input AdminUpdateDialogueInput) (*AdminDialogueDetail, error)
	DeleteDialogueForAdmin(ctx context.Context, entityID uint64, entryID string) error
}
