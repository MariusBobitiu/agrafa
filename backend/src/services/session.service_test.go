package services

import (
	"net/http"
	"testing"
	"time"
)

func TestSessionServiceDurationHonorsRememberMe(t *testing.T) {
	t.Parallel()

	service := NewSessionService(7*24*time.Hour, 30*24*time.Hour, true)

	if got := service.Duration(false); got != 7*24*time.Hour {
		t.Fatalf("Duration(false) = %s, want 168h", got)
	}

	if got := service.Duration(true); got != 30*24*time.Hour {
		t.Fatalf("Duration(true) = %s, want 720h", got)
	}
}

func TestSessionServiceBuildCookieUsesExpectedSettings(t *testing.T) {
	t.Parallel()

	service := NewSessionService(7*24*time.Hour, 30*24*time.Hour, true)
	expiresAt := time.Date(2026, time.April, 6, 12, 0, 0, 0, time.UTC)

	cookie := service.BuildCookie("token-value", expiresAt)

	if cookie.Name != SessionCookieName {
		t.Fatalf("cookie.Name = %q", cookie.Name)
	}
	if cookie.Value != "token-value" {
		t.Fatalf("cookie.Value = %q", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Fatal("cookie.HttpOnly = false, want true")
	}
	if !cookie.Secure {
		t.Fatal("cookie.Secure = false, want true")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie.SameSite = %v, want Lax", cookie.SameSite)
	}
	if !cookie.Expires.Equal(expiresAt) {
		t.Fatalf("cookie.Expires = %s, want %s", cookie.Expires, expiresAt)
	}
}
