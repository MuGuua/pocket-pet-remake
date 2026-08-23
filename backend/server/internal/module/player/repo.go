package player

import "context"

type Repository interface {
	FindByPlayerID(ctx context.Context, playerID uint64) (*Profile, error)
	ListForAdmin(ctx context.Context, query AdminListQuery) (*AdminPlayerList, error)
	FindAdminDetailByPlayerID(ctx context.Context, playerID uint64) (*AdminPlayerDetail, error)
	CreateForAdmin(ctx context.Context, input AdminCreatePlayerInput) (*AdminPlayerDetail, error)
	UpdateForAdmin(ctx context.Context, playerID uint64, input AdminUpdatePlayerInput) (*AdminPlayerDetail, error)
	DeleteForAdmin(ctx context.Context, playerID uint64) error
	PurgeDisabledAccountForAdmin(ctx context.Context, playerID uint64) error
	UpdatePosition(ctx context.Context, playerID uint64, sceneID uint32, posX, posY int32) error
	// UpdatePositionIfNewer 仅在传入版本高于数据库现有版本时写入，并返回本次位置是否实际生效。
	UpdatePositionIfNewer(ctx context.Context, playerID uint64, sceneID uint32, posX, posY int32, positionVersion uint64) (bool, error)
	AddRewardAttribute(ctx context.Context, playerID uint64, attrKey string, value uint32) error
	CountActivePlayers(ctx context.Context) (uint64, error)
}

// WorldSummaryRepository 批量读取场景同屏展示所需的轻量玩家摘要。
type WorldSummaryRepository interface {
	ListWorldSummariesByPlayerIDs(ctx context.Context, playerIDs []uint64) ([]WorldSummary, error)
}

// WorldTransferProfileRepository 提供场景切换专用轻量档案读取，避免复用完整战斗档案查询。
type WorldTransferProfileRepository interface {
	FindWorldTransferProfile(ctx context.Context, playerID uint64) (*WorldTransferProfile, error)
}
