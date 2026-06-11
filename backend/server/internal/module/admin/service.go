package admin

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

type Service struct {
	users  UserRepository
	signer AccessTokenSigner
}

func NewService(users UserRepository, signer AccessTokenSigner) *Service {
	return &Service{users: users, signer: signer}
}

func (s *Service) Login(ctx context.Context, accountName, password string) (*LoginResult, error) {
	accountName = strings.TrimSpace(accountName)
	password = strings.TrimSpace(password)
	if accountName == "" || password == "" {
		return nil, ErrInvalidCredentials
	}
	user, err := s.users.FindByAccountName(ctx, accountName)
	if err != nil || user == nil || user.Status != 1 {
		return nil, ErrInvalidCredentials
	}
	if !secureEqual(user.PasswordHash, HashPassword(password)) {
		return nil, ErrInvalidCredentials
	}
	if err := s.users.TouchLastLoginAt(ctx, user.AdminUserID); err != nil {
		return nil, err
	}
	token, expireAt, err := s.signer.Sign(user.AdminUserID, user.AccountName)
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		AdminUserID: user.AdminUserID,
		DisplayName: user.DisplayName,
		RoleKeys:    append([]string{}, user.RoleKeys...),
		Token:       token,
		ExpireAt:    expireAt,
	}, nil
}

func (s *Service) ProfileByToken(ctx context.Context, token string) (*SessionProfile, error) {
	claims, err := s.signer.Parse(strings.TrimSpace(token))
	if err != nil {
		return nil, ErrTokenInvalid
	}
	user, err := s.users.FindByID(ctx, claims.AID)
	if err != nil || user == nil || user.Status != 1 {
		return nil, ErrTokenInvalid
	}
	return &SessionProfile{
		AdminUserID: user.AdminUserID,
		AccountName: user.AccountName,
		DisplayName: user.DisplayName,
		RoleKeys:    append([]string{}, user.RoleKeys...),
		Permissions: append([]string{}, user.Permissions...),
	}, nil
}

func HashPassword(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func secureEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}
