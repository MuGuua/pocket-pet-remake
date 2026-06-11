package admin

import (
	"errors"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid admin credentials")
	ErrTokenInvalid       = errors.New("invalid admin token")
)

type User struct {
	AdminUserID  uint64
	AccountName  string
	PasswordHash string
	DisplayName  string
	Status       uint32
	RoleKeys     []string
	Permissions  []string
	LastLoginAt  *time.Time
}

type LoginResult struct {
	AdminUserID uint64   `json:"admin_user_id"`
	DisplayName string   `json:"display_name"`
	RoleKeys    []string `json:"role_keys"`
	Token       string   `json:"access_token"`
	ExpireAt    int64    `json:"expire_at"`
}

type SessionProfile struct {
	AdminUserID uint64   `json:"admin_user_id"`
	AccountName string   `json:"account_name"`
	DisplayName string   `json:"display_name"`
	RoleKeys    []string `json:"role_keys"`
	Permissions []string `json:"permissions"`
}
