package redis

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"pocket-pet-remake/server/internal/module/auth"
)

type WSTokenRepository struct {
	client    Client
	keyPrefix string
	now       func() time.Time
}

func NewWSTokenRepository(client Client, keyPrefix string) *WSTokenRepository {
	return &WSTokenRepository{
		client:    client,
		keyPrefix: normalizeKeyPrefix(keyPrefix),
		now:       time.Now,
	}
}

func (r *WSTokenRepository) Store(ctx context.Context, record auth.WSTokenRecord) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return r.client.SetEX(ctx, r.buildKey(record.Token), payload, r.ttl(record.ExpiresAt))
}

func (r *WSTokenRepository) Consume(ctx context.Context, token string) (*auth.WSTokenRecord, error) {
	payload, err := r.client.GetDel(ctx, r.buildKey(token))
	if errors.Is(err, ErrCacheMiss) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var record auth.WSTokenRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *WSTokenRepository) buildKey(token string) string {
	return r.keyPrefix + ":ws_token:" + token
}

func (r *WSTokenRepository) ttl(expiresAt time.Time) time.Duration {
	ttl := expiresAt.Sub(r.now())
	if ttl <= 0 {
		return time.Second
	}
	return ttl
}

func normalizeKeyPrefix(prefix string) string {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		return "pocket_pet"
	}
	return strings.TrimRight(trimmed, ":")
}
