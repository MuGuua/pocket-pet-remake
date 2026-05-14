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
}
