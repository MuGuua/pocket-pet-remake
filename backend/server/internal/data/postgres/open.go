package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"pocket-pet-remake/server/internal/config"
)

const pingTimeout = 5 * time.Second

func Open(cfg config.PostgresConfig) (*sql.DB, error) {
	connectionConfig, err := pgx.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, err
	}
	connectionConfig.Tracer = queryMetricsTracer{}
	db := stdlib.OpenDB(*connectionConfig)

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
