package db_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNodeExecutionModeMigrationBackfillsExistingNodes(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	migrationPath := filepath.Join(filepath.Dir(currentFile), "migrations", "app", "000010_node_execution_modes.up.sql")
	contents, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}

	sql := string(contents)
	if !strings.Contains(sql, "SET node_type = 'agent'") {
		t.Fatalf("expected managed-node migration to backfill node_type to agent:\n%s", sql)
	}
	if !strings.Contains(sql, "is_visible = TRUE") {
		t.Fatalf("expected managed-node migration to backfill is_visible to TRUE:\n%s", sql)
	}
}
