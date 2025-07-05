package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateToken(length int) (string, error) {
	if length <= 0 {
		length = 32
	}

	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}