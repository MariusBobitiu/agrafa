package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

const (
	SessionCookieName = "agrafa_session"
	sessionTokenBytes = 32
)

type SessionService struct {
	sessionTTL         time.Duration
	sessionRememberTTL time.Duration
	cookieSecure       bool
	now                func() time.Time
}

func NewSessionService(sessionTTL, sessionRememberTTL time.Duration, cookieSecure bool) *SessionService {
	return &SessionService{
		sessionTTL:         sessionTTL,
		sessionRememberTTL: sessionRememberTTL,
		cookieSecure:       cookieSecure,
		now:                time.Now,
	}
}

func (s *SessionService) Duration(rememberMe bool) time.Duration {
	if rememberMe {
		return s.sessionRememberTTL
	}

	return s.sessionTTL
}

func (s *SessionService) ExpiresAt(rememberMe bool) time.Time {
	return s.now().UTC().Add(s.Duration(rememberMe))
}

func (s *SessionService) GenerateToken() (string, error) {
	buffer := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("read session token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func (s *SessionService) HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *SessionService) BuildCookie(token string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt.UTC(),
	}
}

func (s *SessionService) ClearCookie() *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
	}
}
