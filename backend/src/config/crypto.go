package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"crypto/sha256"
	"golang.org/x/crypto/hkdf"
)

const (
	instanceSettingsKeyLength = 32
	instanceSettingsNonceSize = 12
	instanceSettingsAAD       = "agrafa.instance_settings.v1"
	instanceSettingsInfo      = "agrafa.instance_settings.encryption_key"
)

type Encryptor struct {
	aead cipher.AEAD
}

func NewEncryptor(appSecret string) (*Encryptor, error) {
	secret := strings.TrimSpace(appSecret)
	if secret == "" {
		return nil, fmt.Errorf("APP_SECRET is required")
	}

	key, err := deriveKey(secret)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("build aes cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("build gcm cipher: %w", err)
	}

	return &Encryptor{aead: aead}, nil
}

func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if e == nil || e.aead == nil {
		return "", fmt.Errorf("encryptor is not configured")
	}

	nonce := make([]byte, instanceSettingsNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("read encryption nonce: %w", err)
	}

	ciphertext := e.aead.Seal(nil, nonce, []byte(plaintext), []byte(instanceSettingsAAD))
	payload := append(nonce, ciphertext...)

	return base64.RawStdEncoding.EncodeToString(payload), nil
}

func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if e == nil || e.aead == nil {
		return "", fmt.Errorf("encryptor is not configured")
	}

	payload, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(ciphertext))
	if err != nil {
		return "", fmt.Errorf("decode encrypted setting: %w", err)
	}
	if len(payload) < instanceSettingsNonceSize {
		return "", fmt.Errorf("encrypted setting payload is too short")
	}

	nonce := payload[:instanceSettingsNonceSize]
	encrypted := payload[instanceSettingsNonceSize:]
	plaintext, err := e.aead.Open(nil, nonce, encrypted, []byte(instanceSettingsAAD))
	if err != nil {
		return "", fmt.Errorf("decrypt instance setting: %w", err)
	}

	return string(plaintext), nil
}

func deriveKey(secret string) ([]byte, error) {
	reader := hkdf.New(sha256.New, []byte(secret), []byte(instanceSettingsAAD), []byte(instanceSettingsInfo))
	key := make([]byte, instanceSettingsKeyLength)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, fmt.Errorf("derive encryption key: %w", err)
	}

	return key, nil
}
