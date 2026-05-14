package memory

import (
	"context"
	"sync"
	"time"

	"pocket-pet-remake/server/internal/module/auth"
)

type WSTokenRepository struct {
	mu     sync.Mutex
	tokens map[string]auth.WSTokenRecord
	now    func() time.Time
}

func NewWSTokenRepository() *WSTokenRepository {
	return &WSTokenRepository{
		tokens: make(map[string]auth.WSTokenRecord),
		now:    time.Now,
	}
}

func (r *WSTokenRepository) Store(_ context.Context, record auth.WSTokenRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[record.Token] = record
	return nil
}

func (r *WSTokenRepository) Consume(_ context.Context, token string) (*auth.WSTokenRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.tokens[token]
	if !ok {
		return nil, nil
	}
	delete(r.tokens, token)
	if record.ExpiresAt.Before(r.now()) {
		return nil, nil
	}
	copy := record
	return &copy, nil
}
