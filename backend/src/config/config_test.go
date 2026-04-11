package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadRequiresPostgresURI(t *testing.T) {
	t.Setenv("POSTGRES_URI", "")
	t.Setenv("APP_SECRET", "test-secret")
	t.Setenv("PORT", "")
	t.Setenv("NODE_HEARTBEAT_TTL_SECONDS", "")
	t.Setenv("NODE_EXPIRY_CHECK_INTERVAL_SECONDS", "")
	t.Setenv("SESSION_TTL_DAYS", "")
	t.Setenv("SESSION_REMEMBER_TTL_DAYS", "")

	withWorkingDirectory(t, t.TempDir())

	_, err := Load()
	if err == nil || err.Error() != "POSTGRES_URI is required" {
		t.Fatalf("Load() error = %v, want POSTGRES_URI is required", err)
	}
}

func TestLoadRequiresAppSecret(t *testing.T) {
	t.Setenv("POSTGRES_URI", "postgres://user:pass@localhost:5432/agrafa?sslmode=disable")
	t.Setenv("APP_SECRET", "")
	t.Setenv("PORT", "")
	t.Setenv("NODE_HEARTBEAT_TTL_SECONDS", "")
	t.Setenv("NODE_EXPIRY_CHECK_INTERVAL_SECONDS", "")
	t.Setenv("SESSION_TTL_DAYS", "")
	t.Setenv("SESSION_REMEMBER_TTL_DAYS", "")

	withWorkingDirectory(t, t.TempDir())

	_, err := Load()
	if err == nil || err.Error() != "APP_SECRET is required" {
		t.Fatalf("Load() error = %v, want APP_SECRET is required", err)
	}
}

func TestLoadUsesSinglePostgresURIAndDefaults(t *testing.T) {
	t.Setenv("POSTGRES_URI", "postgres://user:pass@localhost:5432/agrafa?sslmode=disable")
	t.Setenv("APP_SECRET", "test-secret")
	t.Setenv("PORT", "")
	t.Setenv("NODE_HEARTBEAT_TTL_SECONDS", "")
	t.Setenv("NODE_EXPIRY_CHECK_INTERVAL_SECONDS", "")
	t.Setenv("SESSION_TTL_DAYS", "")
	t.Setenv("SESSION_REMEMBER_TTL_DAYS", "")
	t.Setenv("APP_ENV", "")

	withWorkingDirectory(t, t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.PostgresURI != "postgres://user:pass@localhost:5432/agrafa?sslmode=disable" {
		t.Fatalf("PostgresURI = %q", cfg.PostgresURI)
	}

	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want 8080", cfg.Port)
	}

	if cfg.NodeHeartbeatTTL != 60*time.Second {
		t.Fatalf("NodeHeartbeatTTL = %s, want 60s", cfg.NodeHeartbeatTTL)
	}

	if cfg.NodeExpiryCheckInterval != 15*time.Second {
		t.Fatalf("NodeExpiryCheckInterval = %s, want 15s", cfg.NodeExpiryCheckInterval)
	}

	if cfg.Environment != "development" {
		t.Fatalf("Environment = %q, want development", cfg.Environment)
	}

	if cfg.AppBaseURL != "http://localhost:3000" {
		t.Fatalf("AppBaseURL = %q, want http://localhost:3000", cfg.AppBaseURL)
	}

	if cfg.AppSecret != "test-secret" {
		t.Fatalf("AppSecret = %q", cfg.AppSecret)
	}

	if cfg.SessionTTL != 7*24*time.Hour {
		t.Fatalf("SessionTTL = %s, want 168h", cfg.SessionTTL)
	}

	if cfg.SessionRememberTTL != 30*24*time.Hour {
		t.Fatalf("SessionRememberTTL = %s, want 720h", cfg.SessionRememberTTL)
	}

	if cfg.SessionCookieSecure {
		t.Fatal("SessionCookieSecure = true, want false by default")
	}
}

func withWorkingDirectory(t *testing.T, dir string) {
	t.Helper()

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	if err := os.Chdir(filepath.Clean(dir)); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
}
