package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type AccessTokenSigner interface {
	Sign(accountID, playerID uint64) (string, error)
}

type HMACSigner struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type jwtClaims struct {
	Sub string `json:"sub"`
	AID uint64 `json:"aid"`
	PID uint64 `json:"pid"`
	IAT int64  `json:"iat"`
	EXP int64  `json:"exp"`
	Typ string `json:"typ"`
}

func NewHMACSigner(secret string, ttl time.Duration) *HMACSigner {
	return &HMACSigner{
		secret: []byte(secret),
		ttl:    ttl,
		now:    time.Now,
	}
}

func (s *HMACSigner) Sign(accountID, playerID uint64) (string, error) {
	headerBytes, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", err
	}

	now := s.now()
	claimsBytes, err := json.Marshal(jwtClaims{
		Sub: strconv.FormatUint(playerID, 10),
		AID: accountID,
		PID: playerID,
		IAT: now.Unix(),
		EXP: now.Add(s.ttl).Unix(),
		Typ: "access",
	})
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signingInput := encodedHeader + "." + encodedClaims

	h := hmac.New(sha256.New, s.secret)
	if _, err := h.Write([]byte(signingInput)); err != nil {
		return "", err
	}
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s.%s", signingInput, signature), nil
}
