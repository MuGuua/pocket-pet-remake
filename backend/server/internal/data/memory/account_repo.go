package memory

import (
	"context"
	"sync"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/module/auth"
)

type AccountRepository struct {
	mu       sync.RWMutex
	accounts map[string]auth.Account
}

func NewAccountRepository(cfg config.Config) *AccountRepository {
	return &AccountRepository{
		accounts: map[string]auth.Account{
			cfg.DemoAccount: {
				AccountID:    cfg.DemoAccountID,
				AccountName:  cfg.DemoAccount,
				PasswordHash: auth.HashPassword(cfg.DemoPassword),
				PlayerID:     cfg.DemoPlayerID,
				PlayerName:   cfg.DemoPlayerName,
				PlayerLevel:  1,
			},
		},
	}
}

func (r *AccountRepository) FindByAccountName(_ context.Context, accountName string) (*auth.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	account, ok := r.accounts[accountName]
	if !ok {
		return nil, nil
	}
	copy := account
	return &copy, nil
}
