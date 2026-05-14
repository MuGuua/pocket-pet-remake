package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func RandomHex(bytesLen int) (string, error) {
	if bytesLen <= 0 {
		return "", fmt.Errorf("bytesLen must be positive")
	}
	buffer := make([]byte, bytesLen)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
