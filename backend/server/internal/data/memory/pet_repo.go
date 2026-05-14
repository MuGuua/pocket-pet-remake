package memory

import (
	"context"
	"sync"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/module/pet"
)

type PetRepository struct {
	mu     sync.RWMutex
	lineup map[uint64][]pet.LineupPet
}

func NewPetRepository(cfg config.Config) *PetRepository {
	return &PetRepository{
		lineup: map[uint64][]pet.LineupPet{
			cfg.DemoPlayerID: {
				{PetUID: 20001, PetID: 101, Level: 5, HP: 32, HPMax: 32},
				{PetUID: 20002, PetID: 102, Level: 4, HP: 28, HPMax: 30},
			},
		},
	}
}

func (r *PetRepository) ListLineupByPlayerID(_ context.Context, playerID uint64) ([]pet.LineupPet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lineup, ok := r.lineup[playerID]
	if !ok {
		return []pet.LineupPet{}, nil
	}
	copied := make([]pet.LineupPet, len(lineup))
	copy(copied, lineup)
	return copied, nil
}
