package pet

import "context"

type Repository interface {
	ListLineupByPlayerID(ctx context.Context, playerID uint64) ([]LineupPet, error)
}
