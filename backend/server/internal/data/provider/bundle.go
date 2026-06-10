package provider

import (
	"fmt"

	"pocket-pet-remake/server/internal/config"
	pgrepo "pocket-pet-remake/server/internal/data/postgres"
	redisrepo "pocket-pet-remake/server/internal/data/redis"
	"pocket-pet-remake/server/internal/module/auth"
	"pocket-pet-remake/server/internal/module/battle"
	"pocket-pet-remake/server/internal/module/npc"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/world"
)

type Bundle struct {
	Accounts auth.AccountRepository
	Battles  battle.Repository
	Players  player.Repository
	Pets     pet.Repository
	Quests   quest.Repository
	NPCs     npc.Repository
	World    world.Repository
	WSTokens auth.WSTokenRepository
}

type Dependencies struct {
	Postgres pgrepo.DBTX
	Redis    redisrepo.Client
}

func NewConfiguredBundle(cfg config.Config, deps Dependencies) (Bundle, error) {
	if deps.Postgres == nil {
		return Bundle{}, fmt.Errorf("postgres query executor is required")
	}
	if deps.Redis == nil {
		return Bundle{}, fmt.Errorf("redis client is required")
	}

	return Bundle{
		Accounts: pgrepo.NewAccountRepository(deps.Postgres),
		Battles:  pgrepo.NewBattleRepository(deps.Postgres),
		Players:  pgrepo.NewPlayerRepository(deps.Postgres),
		Pets:     pgrepo.NewPetRepository(deps.Postgres),
		Quests:   pgrepo.NewQuestRepository(deps.Postgres),
		NPCs:     pgrepo.NewNPCRepository(deps.Postgres),
		World:    pgrepo.NewWorldRepository(deps.Postgres),
		WSTokens: redisrepo.NewWSTokenRepository(deps.Redis, cfg.Redis.KeyPrefix),
	}, nil
}
