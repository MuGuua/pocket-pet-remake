package auth

import (
	"context"
	"time"
)

type AccountRepository interface {
	FindByAccountName(ctx context.Context, accountName string) (*Account, error)
	TouchLastLoginAt(ctx context.Context, accountID uint64) error
	GetDashboardAccountMetrics(ctx context.Context, dayStart, dayEnd time.Time) (*AccountDashboardMetrics, error)
}

type WSTokenRepository interface {
	Store(ctx context.Context, record WSTokenRecord) error
	Consume(ctx context.Context, token string) (*WSTokenRecord, error)
}
