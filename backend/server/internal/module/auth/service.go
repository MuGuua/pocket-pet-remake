package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"

	"pocket-pet-remake/server/internal/platform/idgen"
)

type Service struct {
	accounts   AccountRepository
	wsTokens   WSTokenRepository
	signer     AccessTokenSigner
	wsTokenTTL time.Duration
	now        func() time.Time
}

func NewService(accounts AccountRepository, wsTokens WSTokenRepository, signer AccessTokenSigner, wsTokenTTL time.Duration) *Service {
	return &Service{
		accounts:   accounts,
		wsTokens:   wsTokens,
		signer:     signer,
		wsTokenTTL: wsTokenTTL,
		now:        time.Now,
	}
}

func (s *Service) Login(ctx context.Context, accountName, password, deviceID string) (*LoginResult, error) {
	accountName = strings.TrimSpace(accountName)
	password = strings.TrimSpace(password)
	if accountName == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	account, err := s.accounts.FindByAccountName(ctx, accountName)
	if err != nil || account == nil {
		return nil, ErrInvalidCredentials
	}
	if !secureEqual(account.PasswordHash, HashPassword(password)) {
		return nil, ErrInvalidCredentials
	}

	accessJWT, err := s.signer.Sign(account.AccountID, account.PlayerID)
	if err != nil {
		return nil, err
	}

	wsToken, err := idgen.RandomHex(24)
	if err != nil {
		return nil, err
	}

	expireAt := s.now().Add(s.wsTokenTTL)
	if err := s.wsTokens.Store(ctx, WSTokenRecord{
		Token:     wsToken,
		PlayerID:  account.PlayerID,
		DeviceID:  deviceID,
		ExpiresAt: expireAt,
	}); err != nil {
		return nil, err
	}

	return &LoginResult{
		PlayerID:   account.PlayerID,
		PlayerName: account.PlayerName,
		AccessJWT:  accessJWT,
		WSToken:    wsToken,
		WSExpireAt: expireAt.Unix(),
	}, nil
}

func (s *Service) ConsumeWSToken(ctx context.Context, token, deviceID string) (*SessionPrincipal, error) {
	record, err := s.wsTokens.Consume(ctx, strings.TrimSpace(token))
	if err != nil || record == nil {
		return nil, ErrInvalidWSToken
	}
	if record.DeviceID != "" && deviceID != "" && record.DeviceID != deviceID {
		return nil, ErrInvalidWSToken
	}
	if record.ExpiresAt.Before(s.now()) {
		return nil, ErrInvalidWSToken
	}
	return &SessionPrincipal{PlayerID: record.PlayerID}, nil
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
