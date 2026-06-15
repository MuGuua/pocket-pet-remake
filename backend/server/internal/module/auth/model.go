package auth

import (
	"errors"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidWSToken     = errors.New("invalid ws token")
)

type Account struct {
	AccountID    uint64
	AccountName  string
	PasswordHash string
	PlayerID     uint64
	PlayerName   string
	PlayerLevel  uint32
}

type WSTokenRecord struct {
	Token     string
	PlayerID  uint64
	DeviceID  string
	ExpiresAt time.Time
}

type LoginResult struct {
	PlayerID   uint64
	PlayerName string
	AccessJWT  string
	WSToken    string
	WSExpireAt int64
}

type SessionPrincipal struct {
	PlayerID uint64
}

// AccountDashboardMetrics 汇总账号侧控制台指标，日活按 account.last_login_at 统计。
type AccountDashboardMetrics struct {
	TotalAccounts       uint64
	DailyActiveAccounts uint64
	NewAccountsToday    uint64
}
