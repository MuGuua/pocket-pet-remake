package provider

import (
	"fmt"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/data/memory"
	pgrepo "pocket-pet-remake/server/internal/data/postgres"
	redisrepo "pocket-pet-remake/server/internal/data/redis"
	"pocket-pet-remake/server/internal/module/auth"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/world"
)

type Bundle struct {
	Accounts auth.AccountRepository
	Players  player.Repository
	Pets     pet.Repository
	World    world.Repository
	WSTokens auth.WSTokenRepository
}

type Dependencies struct {
	Postgres pgrepo.DBTX
	Redis    redisrepo.Client
}

func NewConfiguredBundle(cfg config.Config, deps Dependencies) (Bundle, error) {
	switch cfg.EffectiveRepositoryMode() {
	case config.RepositoryModeMemory:
		return NewMemoryBundle(cfg), nil
	case config.RepositoryModePostgresRedis:
		if deps.Postgres == nil {
			return Bundle{}, fmt.Errorf("repository mode %q requires a postgres query executor", config.RepositoryModePostgresRedis)
		}
		if deps.Redis == nil {
			return Bundle{}, fmt.Errorf("repository mode %q requires a redis client", config.RepositoryModePostgresRedis)
		}
		return Bundle{
			Accounts: pgrepo.NewAccountRepository(deps.Postgres),
			Players:  pgrepo.NewPlayerRepository(deps.Postgres),
			Pets:     pgrepo.NewPetRepository(deps.Postgres),
			World:    memory.NewWorldRepository(),
			WSTokens: redisrepo.NewWSTokenRepository(deps.Redis, cfg.Redis.KeyPrefix),
		}, nil
	default:
		return Bundle{}, fmt.Errorf("unsupported repository mode: %s", cfg.EffectiveRepositoryMode())
	}
}

func NewMemoryBundle(cfg config.Config) Bundle {
	return Bundle{
		Accounts: memory.NewAccountRepository(cfg),
		Players:  memory.NewPlayerRepository(cfg),
		Pets:     memory.NewPetRepository(cfg),
		World:    memory.NewWorldRepository(),
		WSTokens: memory.NewWSTokenRepository(),
	}
}
