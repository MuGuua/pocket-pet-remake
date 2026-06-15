package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"pocket-pet-remake/server/internal/module/auth"
)

type AccountRepository struct {
	db DBTX
}

func NewAccountRepository(db DBTX) *AccountRepository {
	return &AccountRepository{db: db}
}

const findByAccountNameQuery = `
SELECT
  a.id,
  a.account_name,
  a.password_hash,
  p.id,
  p.name,
  p.level
FROM account a
JOIN player p ON p.account_id = a.id
WHERE a.account_name = $1 AND a.status = 1
ORDER BY p.id ASC
LIMIT 1
`

func (r *AccountRepository) FindByAccountName(ctx context.Context, accountName string) (*auth.Account, error) {
	var (
		account   auth.Account
		accountID int64
		playerID  int64
		level     int64
	)

	err := r.db.QueryRowContext(ctx, findByAccountNameQuery, accountName).Scan(
		&accountID,
		&account.AccountName,
		&account.PasswordHash,
		&playerID,
		&account.PlayerName,
		&level,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	account.AccountID = uint64(accountID)
	account.PlayerID = uint64(playerID)
	account.PlayerLevel = uint32(level)
	return &account, nil
}

const touchAccountLastLoginQuery = `
UPDATE account
SET last_login_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND status = 1
`

const dashboardAccountMetricsQuery = `
SELECT
  COUNT(*) FILTER (WHERE status = 1) AS total_accounts,
  COUNT(*) FILTER (WHERE status = 1 AND last_login_at >= $1 AND last_login_at < $2) AS daily_active_accounts,
  COUNT(*) FILTER (WHERE status = 1 AND created_at >= $1 AND created_at < $2) AS new_accounts_today
FROM account
`

func (r *AccountRepository) TouchLastLoginAt(ctx context.Context, accountID uint64) error {
	_, err := r.db.ExecContext(ctx, touchAccountLastLoginQuery, accountID)
	return err
}

func (r *AccountRepository) GetDashboardAccountMetrics(ctx context.Context, dayStart, dayEnd time.Time) (*auth.AccountDashboardMetrics, error) {
	var metrics auth.AccountDashboardMetrics
	err := r.db.QueryRowContext(ctx, dashboardAccountMetricsQuery, dayStart, dayEnd).Scan(
		&metrics.TotalAccounts,
		&metrics.DailyActiveAccounts,
		&metrics.NewAccountsToday,
	)
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}
