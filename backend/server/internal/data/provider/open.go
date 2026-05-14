package provider

import (
	"io"

	"pocket-pet-remake/server/internal/config"
	pgrepo "pocket-pet-remake/server/internal/data/postgres"
	redisrepo "pocket-pet-remake/server/internal/data/redis"
)

func OpenDependencies(cfg config.Config) (Dependencies, []io.Closer, error) {
	if cfg.EffectiveRepositoryMode() == config.RepositoryModeMemory {
		return Dependencies{}, nil, nil
	}

	postgresDB, err := pgrepo.Open(cfg.Postgres)
	if err != nil {
		return Dependencies{}, nil, err
	}

	redisClient, err := redisrepo.Open(cfg.Redis)
	if err != nil {
		_ = postgresDB.Close()
		return Dependencies{}, nil, err
	}

	closers := []io.Closer{redisClient, postgresDB}
	deps := Dependencies{
		Postgres: postgresDB,
		Redis:    redisClient,
	}
	return deps, closers, nil
}
