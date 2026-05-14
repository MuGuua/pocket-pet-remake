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
