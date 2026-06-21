package npcdialogue

import "context"

// AdminRepository 描述后台剧情配置页所需的聚合 CRUD 接口。
type AdminRepository interface {
	ListDialoguesForAdmin(ctx context.Context, query AdminDialogueListQuery) (*AdminDialogueList, error)
	FindDialogueDetailForAdmin(ctx context.Context, entityID uint64, entryID string) (*AdminDialogueDetail, error)
	CreateDialogueForAdmin(ctx context.Context, input AdminCreateDialogueInput) (*AdminDialogueDetail, error)
	UpdateDialogueForAdmin(ctx context.Context, entityID uint64, entryID string, input AdminUpdateDialogueInput) (*AdminDialogueDetail, error)
	DeleteDialogueForAdmin(ctx context.Context, entityID uint64, entryID string) error
}
