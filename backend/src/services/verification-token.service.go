package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

const (
	verificationTokenBytes       = 32
	emailVerificationTokenTTL    = 24 * time.Hour
	passwordResetVerificationTTL = 1 * time.Hour
)

type VerificationTokenService struct {
	now func() time.Time
}

func NewVerificationTokenService() *VerificationTokenService {
	return &VerificationTokenService{
		now: time.Now,
	}
}

func (s *VerificationTokenService) GenerateToken() (string, error) {
	buffer := make([]byte, verificationTokenBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("read verification token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func (s *VerificationTokenService) HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *VerificationTokenService) ExpiresAt(tokenType string) time.Time {
	return s.now().UTC().Add(s.Duration(tokenType))
}

func (s *VerificationTokenService) Duration(tokenType string) time.Duration {
	switch tokenType {
	case types.VerificationTokenTypePasswordReset:
		return passwordResetVerificationTTL
	default:
		return emailVerificationTokenTTL
	}
}
