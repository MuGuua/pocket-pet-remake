package player

import "context"

type Repository interface {
	FindByPlayerID(ctx context.Context, playerID uint64) (*Profile, error)
	UpdatePosition(ctx context.Context, playerID uint64, sceneID uint32, posX, posY int32) error
	AddGoldAndExp(ctx context.Context, playerID uint64, gold uint32, exp uint64) (*Profile, error)
}
