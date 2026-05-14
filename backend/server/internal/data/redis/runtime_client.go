package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"pocket-pet-remake/server/internal/config"
)

const pingTimeout = 5 * time.Second

type RuntimeClient struct {
	inner *goredis.Client
}

func Open(cfg config.RedisConfig) (*RuntimeClient, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &RuntimeClient{inner: client}, nil
}

func (c *RuntimeClient) SetEX(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.inner.Set(ctx, key, value, ttl).Err()
}

func (c *RuntimeClient) GetDel(ctx context.Context, key string) ([]byte, error) {
	value, err := c.inner.GetDel(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (c *RuntimeClient) Close() error {
	return c.inner.Close()
}
