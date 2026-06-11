package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type AccessTokenSigner interface {
	Sign(adminUserID uint64, accountName string) (string, int64, error)
	Parse(token string) (*TokenClaims, error)
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

type TokenClaims struct {
	Sub string `json:"sub"`
	AID uint64 `json:"aid"`
	Acc string `json:"acc"`
	IAT int64  `json:"iat"`
	EXP int64  `json:"exp"`
	Typ string `json:"typ"`
}

func NewHMACSigner(secret string, ttl time.Duration) *HMACSigner {
	return &HMACSigner{secret: []byte(secret), ttl: ttl, now: time.Now}
}

func (s *HMACSigner) Sign(adminUserID uint64, accountName string) (string, int64, error) {
	headerBytes, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", 0, err
	}
	now := s.now()
	expireAt := now.Add(s.ttl).Unix()
	claimsBytes, err := json.Marshal(TokenClaims{
		Sub: strconv.FormatUint(adminUserID, 10),
		AID: adminUserID,
		Acc: accountName,
		IAT: now.Unix(),
		EXP: expireAt,
		Typ: "admin_access",
	})
	if err != nil {
		return "", 0, err
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signingInput := encodedHeader + "." + encodedClaims
	h := hmac.New(sha256.New, s.secret)
	if _, err := h.Write([]byte(signingInput)); err != nil {
		return "", 0, err
	}
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s.%s", signingInput, signature), expireAt, nil
}

func (s *HMACSigner) Parse(token string) (*TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrTokenInvalid
	}
	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return nil, ErrTokenInvalid
	}

	signingInput := parts[0] + "." + parts[1]
	h := hmac.New(sha256.New, s.secret)
	if _, err := h.Write([]byte(signingInput)); err != nil {
		return nil, ErrTokenInvalid
	}
	expectedSignature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[2])) {
		return nil, ErrTokenInvalid
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrTokenInvalid
	}
	var claims TokenClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, ErrTokenInvalid
	}
	if claims.Typ != "admin_access" || claims.EXP <= s.now().Unix() {
		return nil, ErrTokenInvalid
	}
	return &claims, nil
}
