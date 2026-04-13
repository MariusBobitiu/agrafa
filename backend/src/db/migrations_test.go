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

func TestAlertSeverityMigrationBackfillsExistingRows(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	migrationPath := filepath.Join(filepath.Dir(currentFile), "migrations", "app", "000011_alert_rule_recipient_severity.up.sql")
	contents, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}

	sql := string(contents)
	if !strings.Contains(sql, "WHEN rule_type IN ('node_offline', 'service_unhealthy') THEN 'critical'") {
		t.Fatalf("expected alert rule severity backfill in migration:\n%s", sql)
	}
	if !strings.Contains(sql, "SET min_severity = 'info'") {
		t.Fatalf("expected notification recipient min_severity backfill in migration:\n%s", sql)
	}
}

func TestInstanceSettingsMigrationCreatesMetaTable(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	migrationPath := filepath.Join(filepath.Dir(currentFile), "migrations", "app", "000012_instance_settings.up.sql")
	contents, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}

	sql := string(contents)
	if !strings.Contains(sql, "CREATE TABLE agrafa_meta.instance_settings") {
		t.Fatalf("expected instance_settings table creation in migration:\n%s", sql)
	}
	if !strings.Contains(sql, "key TEXT NOT NULL UNIQUE") {
		t.Fatalf("expected unique key constraint in migration:\n%s", sql)
	}
	if !strings.Contains(sql, "is_encrypted BOOLEAN NOT NULL DEFAULT FALSE") {
		t.Fatalf("expected is_encrypted column in migration:\n%s", sql)
	}
}

func TestProjectScopeRLSMigrationVerifiesMembershipInDatabase(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	migrationPath := filepath.Join(filepath.Dir(currentFile), "migrations", "app", "000013_project_scope_rls.up.sql")
	contents, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}

	sql := string(contents)
	if !strings.Contains(sql, "FROM app.project_members AS pm") {
		t.Fatalf("expected helper functions to verify membership from app.project_members:\n%s", sql)
	}
	if strings.Contains(sql, "app.current_project_id() = target_project_id") {
		t.Fatalf("helpers must not trust current_project_id for authorization:\n%s", sql)
	}
	if strings.Contains(sql, "pm.role = app.current_project_role()") {
		t.Fatalf("helpers must not trust current_project_role for authorization:\n%s", sql)
	}
}
