package pet

import "context"

type Repository interface {
	ListPetsByPlayerID(ctx context.Context, playerID uint64) ([]Pet, error)
	ListLineupByPlayerID(ctx context.Context, playerID uint64) ([]LineupPet, error)
	SetLineupByPlayerID(ctx context.Context, playerID uint64, petUIDs []uint64) error
	UpdatePetHPByUID(ctx context.Context, playerID uint64, petUID uint64, hp uint32) (Pet, error)
}
