package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateOpaqueID(prefix string, size int) (string, error) {
	if size <= 0 {
		size = 16
	}

	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	encoded := hex.EncodeToString(buffer)
	if prefix == "" {
		return encoded, nil
	}

	return prefix + "_" + encoded, nil
}
