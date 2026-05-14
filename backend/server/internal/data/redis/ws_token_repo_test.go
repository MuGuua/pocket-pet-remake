package redis

import (
	"context"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/auth"
)

type fakeClient struct {
	store map[string][]byte
}

func (f *fakeClient) SetEX(_ context.Context, key string, value []byte, _ time.Duration) error {
	if f.store == nil {
		f.store = make(map[string][]byte)
	}
	copied := make([]byte, len(value))
	copy(copied, value)
	f.store[key] = copied
	return nil
}

func (f *fakeClient) GetDel(_ context.Context, key string) ([]byte, error) {
	value, ok := f.store[key]
	if !ok {
		return nil, ErrCacheMiss
	}
	delete(f.store, key)
	copied := make([]byte, len(value))
	copy(copied, value)
	return copied, nil
}

func TestWSTokenRepositoryStoreAndConsume(t *testing.T) {
	repo := NewWSTokenRepository(&fakeClient{}, "test")
	repo.now = func() time.Time {
		return time.Unix(1_700_000_000, 0)
	}

	record := auth.WSTokenRecord{
		Token:     "token-1",
		PlayerID:  10001,
		DeviceID:  "ios-demo",
		ExpiresAt: repo.now().Add(time.Minute),
	}

	if err := repo.Store(context.Background(), record); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	consumed, err := repo.Consume(context.Background(), record.Token)
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if consumed == nil {
		t.Fatal("Consume() returned nil record")
	}
	if consumed.PlayerID != record.PlayerID {
		t.Fatalf("Consume().PlayerID = %d, want %d", consumed.PlayerID, record.PlayerID)
	}

	consumedAgain, err := repo.Consume(context.Background(), record.Token)
	if err != nil {
		t.Fatalf("second Consume() error = %v", err)
	}
	if consumedAgain != nil {
		t.Fatalf("second Consume() = %#v, want nil", consumedAgain)
	}
}
