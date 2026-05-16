package memory

import (
	"context"
	"sync"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/module/pet"
)

type PetRepository struct {
	mu     sync.RWMutex
	pets   map[uint64][]pet.Pet
	lineup map[uint64][]pet.LineupPet
}

func NewPetRepository(cfg config.Config) *PetRepository {
	return &PetRepository{
		pets: map[uint64][]pet.Pet{
			cfg.DemoPlayerID: {
				{
					PetUID:   20001,
					PetID:    101,
					Level:    5,
					Exp:      120,
					Quality:  1,
					HP:       32,
					HPMax:    32,
					ATK:      14,
					DEF:      10,
					SPD:      12,
					SkillIDs: []uint32{1001, 1002},
				},
				{
					PetUID:   20002,
					PetID:    102,
					Level:    4,
					Exp:      80,
					Quality:  1,
					HP:       28,
					HPMax:    30,
					ATK:      12,
					DEF:      11,
					SPD:      9,
					SkillIDs: []uint32{1001},
				},
				{
					PetUID:   20003,
					PetID:    101,
					Level:    3,
					Exp:      40,
					Quality:  1,
					HP:       24,
					HPMax:    24,
					ATK:      10,
					DEF:      8,
					SPD:      11,
					SkillIDs: []uint32{1001},
				},
			},
		},
		lineup: map[uint64][]pet.LineupPet{
			cfg.DemoPlayerID: {
				{PetUID: 20001, PetID: 101, Level: 5, HP: 32, HPMax: 32},
				{PetUID: 20002, PetID: 102, Level: 4, HP: 28, HPMax: 30},
			},
		},
	}
}

func (r *PetRepository) ListPetsByPlayerID(_ context.Context, playerID uint64) ([]pet.Pet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pets, ok := r.pets[playerID]
	if !ok {
		return []pet.Pet{}, nil
	}

	copied := make([]pet.Pet, 0, len(pets))
	for _, item := range pets {
		next := item
		if len(item.SkillIDs) > 0 {
			next.SkillIDs = append([]uint32{}, item.SkillIDs...)
		}
		copied = append(copied, next)
	}
	return copied, nil
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

func (r *PetRepository) SetLineupByPlayerID(_ context.Context, playerID uint64, petUIDs []uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pets, ok := r.pets[playerID]
	if !ok {
		return nil
	}

	byUID := make(map[uint64]pet.Pet, len(pets))
	for _, item := range pets {
		byUID[item.PetUID] = item
	}

	nextLineup := make([]pet.LineupPet, 0, len(petUIDs))
	for _, petUID := range petUIDs {
		item, exists := byUID[petUID]
		if !exists {
			return pet.ErrPetNotFound
		}
		nextLineup = append(nextLineup, pet.LineupPet{
			PetUID: item.PetUID,
			PetID:  item.PetID,
			Level:  item.Level,
			HP:     item.HP,
			HPMax:  item.HPMax,
		})
	}

	r.lineup[playerID] = nextLineup
	return nil
}

func (r *PetRepository) UpdatePetHPByUID(_ context.Context, playerID uint64, petUID uint64, hp uint32) (pet.Pet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pets, ok := r.pets[playerID]
	if !ok {
		return pet.Pet{}, pet.ErrPetNotFound
	}

	for index := range pets {
		if pets[index].PetUID != petUID {
			continue
		}
		if hp > pets[index].HPMax {
			hp = pets[index].HPMax
		}
		pets[index].HP = hp
		r.pets[playerID] = pets

		lineup := r.lineup[playerID]
		for lineupIndex := range lineup {
			if lineup[lineupIndex].PetUID == petUID {
				lineup[lineupIndex].HP = hp
			}
		}
		r.lineup[playerID] = lineup

		updated := pets[index]
		if len(updated.SkillIDs) > 0 {
			updated.SkillIDs = append([]uint32{}, updated.SkillIDs...)
		}
		return updated, nil
	}

	return pet.Pet{}, pet.ErrPetNotFound
}
