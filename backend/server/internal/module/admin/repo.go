package admin

import "context"

type UserRepository interface {
	FindByAccountName(ctx context.Context, accountName string) (*User, error)
	FindByID(ctx context.Context, adminUserID uint64) (*User, error)
	TouchLastLoginAt(ctx context.Context, adminUserID uint64) error
}
