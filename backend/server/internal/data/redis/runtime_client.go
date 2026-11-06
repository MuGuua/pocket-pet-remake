package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/platform/telemetry"
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

func (c *RuntimeClient) SetEX(ctx context.Context, key string, value []byte, ttl time.Duration) (err error) {
	started := time.Now()
	defer func() { telemetry.Duration("redis.SET", time.Since(started), err != nil) }()
	return c.inner.Set(ctx, key, value, ttl).Err()
}

func (c *RuntimeClient) GetDel(ctx context.Context, key string) (value []byte, err error) {
	started := time.Now()
	defer func() {
		telemetry.Duration("redis.GETDEL", time.Since(started), err != nil && !errors.Is(err, ErrCacheMiss))
	}()
	value, err = c.inner.GetDel(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (c *RuntimeClient) Get(ctx context.Context, key string) (value []byte, err error) {
	started := time.Now()
	defer func() {
		telemetry.Duration("redis.GET", time.Since(started), err != nil && !errors.Is(err, ErrCacheMiss))
	}()
	value, err = c.inner.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, ErrCacheMiss
	}
	return value, err
}

func (c *RuntimeClient) Eval(ctx context.Context, script string, keys []string, args ...any) (result any, err error) {
	started := time.Now()
	defer func() { telemetry.Duration("redis.EVAL", time.Since(started), err != nil) }()
	return c.inner.Eval(ctx, script, keys, args...).Result()
}

func (c *RuntimeClient) Del(ctx context.Context, key string) (err error) {
	started := time.Now()
	defer func() { telemetry.Duration("redis.DEL", time.Since(started), err != nil) }()
	return c.inner.Del(ctx, key).Err()
}

func (c *RuntimeClient) Close() error {
	return c.inner.Close()
}
