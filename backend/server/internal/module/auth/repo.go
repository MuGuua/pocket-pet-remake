package auth

import "context"

type AccountRepository interface {
	FindByAccountName(ctx context.Context, accountName string) (*Account, error)
}

type WSTokenRepository interface {
	Store(ctx context.Context, record WSTokenRecord) error
	Consume(ctx context.Context, token string) (*WSTokenRecord, error)
}
