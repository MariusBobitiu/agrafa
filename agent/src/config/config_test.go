package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAllowsMissingNodeID(t *testing.T) {
	t.Setenv(apiBaseURLEnvKey, "https://backend.example.com/v1")
	t.Setenv(agentTokenEnvKey, "test-agent-token")
	t.Setenv(nodeIDEnvKey, "")

	withWorkingDirectory(t, t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.NodeID != 0 {
		t.Fatalf("NodeID = %d, want 0", cfg.NodeID)
	}
}

func TestLoadRejectsInvalidNodeID(t *testing.T) {
	t.Setenv(apiBaseURLEnvKey, "https://backend.example.com/v1")
	t.Setenv(agentTokenEnvKey, "test-agent-token")
	t.Setenv(nodeIDEnvKey, "abc")

	withWorkingDirectory(t, t.TempDir())

	_, err := Load()
	if err == nil || err.Error() != "AGRAFA_NODE_ID must be a positive integer" {
		t.Fatalf("Load() error = %v, want AGRAFA_NODE_ID must be a positive integer", err)
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
