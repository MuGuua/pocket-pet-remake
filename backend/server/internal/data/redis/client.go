package redis

import (
	"context"
	"errors"
	"time"
)

var ErrCacheMiss = errors.New("redis cache miss")

type Client interface {
	SetEX(ctx context.Context, key string, value []byte, ttl time.Duration) error
	GetDel(ctx context.Context, key string) ([]byte, error)
	Get(ctx context.Context, key string) ([]byte, error)
	Eval(ctx context.Context, script string, keys []string, args ...any) (any, error)
	Del(ctx context.Context, key string) error
}
