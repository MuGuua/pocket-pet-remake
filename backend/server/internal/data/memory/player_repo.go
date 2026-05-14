package memory

import (
	"context"
	"sync"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/module/player"
)

type PlayerRepository struct {
	mu      sync.RWMutex
	players map[uint64]player.Profile
}

func NewPlayerRepository(cfg config.Config) *PlayerRepository {
	return &PlayerRepository{
		players: map[uint64]player.Profile{
			cfg.DemoPlayerID: {
				PlayerID: cfg.DemoPlayerID,
				Name:     cfg.DemoPlayerName,
				Level:    1,
				Gold:     100,
				SceneID:  1,
				PosX:     8,
				PosY:     6,
			},
		},
	}
}

func (r *PlayerRepository) FindByPlayerID(_ context.Context, playerID uint64) (*player.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profile, ok := r.players[playerID]
	if !ok {
		return nil, nil
	}
	copy := profile
	return &copy, nil
}

func (r *PlayerRepository) UpdatePosition(_ context.Context, playerID uint64, sceneID uint32, posX, posY int32) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.players[playerID]
	if !ok {
		return player.ErrPlayerNotFound
	}
	current.SceneID = sceneID
	current.PosX = posX
	current.PosY = posY
	r.players[playerID] = current
	return nil
}
