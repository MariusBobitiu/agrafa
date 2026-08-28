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

func TestServiceCheckHistoryMigrationAddsMetadataIndexAndRLS(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	migrationPath := filepath.Join(filepath.Dir(currentFile), "migrations", "app", "000014_service_check_history.up.sql")
	contents, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}

	sql := string(contents)
	for _, expected := range []string{
		"ADD COLUMN node_id BIGINT",
		"ADD COLUMN check_type TEXT",
		"ADD COLUMN source TEXT",
		"service_id, observed_at DESC, id DESC",
		"health_check_results_select_member_access",
		"health_check_results_insert_access",
		"Append-only service check observations",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected migration to contain %q:\n%s", expected, sql)
		}
	}
}

func TestServiceCheckHistoryCorrectionUsesOnlyCanonicalPayloadMetadata(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	migrationPath := filepath.Join(filepath.Dir(currentFile), "migrations", "app", "000015_correct_service_history_check_types.up.sql")
	contents, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}

	sql := string(contents)
	for _, expected := range []string{
		"payload ->> 'check_type'",
		"payload ->> 'type'",
		"E' \\t\\n\\r\\f\\013'",
		"IN ('http', 'tcp')",
		"ELSE check_type",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected corrective migration to contain %q:\n%s", expected, sql)
		}
	}
	if strings.Contains(sql, "FROM app.services") {
		t.Fatalf("corrective migration must not guess from current service definitions:\n%s", sql)
	}
	if strings.Contains(sql, "BTRIM(payload ->> 'check_type')") || strings.Contains(sql, "BTRIM(payload ->> 'type')") {
		t.Fatalf("corrective migration must use the explicit supported-whitespace character set:\n%s", sql)
	}
}

func TestAlertRuleTargetingMigrationPreservesSpecificRulesAndAddsPerTargetIdentity(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	migrationPath := filepath.Join(filepath.Dir(currentFile), "migrations", "app", "000016_alert_rule_targeting.up.sql")
	contents, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}

	sql := string(contents)
	for _, expected := range []string{
		"DROP CONSTRAINT IF EXISTS alert_rules_check",
		"alert_rules_target_check",
		"num_nonnulls(node_id, service_id) = 1",
		"(alert_rule_id, node_id)",
		"(alert_rule_id, service_id)",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected targeting migration to contain %q:\n%s", expected, sql)
		}
	}
	if strings.Contains(strings.ToUpper(sql), "UPDATE APP.ALERT_RULES") {
		t.Fatalf("targeting migration must not rewrite existing specific rule targets:\n%s", sql)
	}
}
