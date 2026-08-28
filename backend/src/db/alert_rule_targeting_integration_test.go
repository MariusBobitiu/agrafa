package db_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	appdb "github.com/MariusBobitiu/agrafa-backend/src/db"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestAlertRuleTargetingMigrationDatabaseSemantics(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("AGRAFA_RLS_TEST_DSN"))
	if dsn == "" {
		t.Skip("AGRAFA_RLS_TEST_DSN is not set")
	}

	ctx := context.Background()
	db, err := appdb.OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
		setRLSContext(t, ctx, tx, "", nil, nil, true)

		var newConstraintExists bool
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conrelid = 'app.alert_rules'::regclass
				  AND contype = 'c'
				  AND conname = 'alert_rules_target_check'
			)
		`).Scan(&newConstraintExists); err != nil {
			t.Fatalf("check explicit target constraint: %v", err)
		}
		if !newConstraintExists {
			t.Fatal("alert_rules_target_check is not applied; run migration 000016")
		}

		var legacyConstraintCount int
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM pg_constraint
			WHERE conrelid = 'app.alert_rules'::regclass
			  AND contype = 'c'
			  AND conname <> 'alert_rules_target_check'
			  AND LOWER(pg_get_constraintdef(oid)) LIKE '%rule_type = ''node_offline''%'
			  AND LOWER(pg_get_constraintdef(oid)) LIKE '%rule_type = ''service_unhealthy''%'
			  AND LOWER(pg_get_constraintdef(oid)) LIKE '%node_id is not null%'
			  AND LOWER(pg_get_constraintdef(oid)) LIKE '%service_id is not null%'
		`).Scan(&legacyConstraintCount); err != nil {
			t.Fatalf("check legacy target constraints: %v", err)
		}
		if legacyConstraintCount != 0 {
			t.Fatalf("legacy target constraint count = %d, want 0", legacyConstraintCount)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO app.alert_rules (
				id, project_id, node_id, service_id, rule_type, severity,
				metric_name, threshold_value, is_enabled
			) VALUES
				(-5101, -1001, NULL, NULL, 'node_offline', 'critical', NULL, NULL, TRUE),
				(-5102, -1001, NULL, NULL, 'service_unhealthy', 'critical', NULL, NULL, TRUE),
				(-5103, -1001, NULL, NULL, 'cpu_above_threshold', 'warning', 'cpu_usage', 80, TRUE),
				(-5104, -1001, NULL, NULL, 'memory_above_threshold', 'warning', 'memory_usage', 80, TRUE),
				(-5105, -1001, NULL, NULL, 'disk_above_threshold', 'warning', 'disk_usage', 80, TRUE)
		`); err != nil {
			t.Fatalf("insert global alert rules: %v", err)
		}

		expectPostgresStatementRejected(t, ctx, tx, `
			INSERT INTO app.alert_rules (
				id, project_id, node_id, service_id, rule_type, severity, is_enabled
			) VALUES (
				-5199, -1001, -2001, -3001, 'node_offline', 'critical', TRUE
			)
		`)

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO app.alert_instances (
				id, alert_rule_id, project_id, node_id, service_id,
				status, triggered_at, title, message
			) VALUES (
				-6101, -5101, -1001, -2001, NULL,
				'active', NOW(), 'offline', 'offline'
			)
		`); err != nil {
			t.Fatalf("insert first active alert for target: %v", err)
		}

		expectPostgresStatementRejected(t, ctx, tx, `
			INSERT INTO app.alert_instances (
				id, alert_rule_id, project_id, node_id, service_id,
				status, triggered_at, title, message
			) VALUES (
				-6102, -5101, -1001, -2001, NULL,
				'active', NOW(), 'duplicate', 'duplicate'
			)
		`)

		if _, err := tx.ExecContext(ctx, `
			UPDATE app.alert_instances
			SET status = 'resolved', resolved_at = NOW()
			WHERE id = -6101
		`); err != nil {
			t.Fatalf("resolve first target alert: %v", err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO app.alert_instances (
				id, alert_rule_id, project_id, node_id, service_id,
				status, triggered_at, title, message
			) VALUES (
				-6103, -5101, -1001, -2001, NULL,
				'active', NOW(), 'offline again', 'offline again'
			)
		`); err != nil {
			t.Fatalf("insert active alert after resolution: %v", err)
		}
	})
}

func expectPostgresStatementRejected(t *testing.T, ctx context.Context, tx *sql.Tx, statement string) {
	t.Helper()

	if _, err := tx.ExecContext(ctx, "SAVEPOINT expected_failure"); err != nil {
		t.Fatalf("create failure savepoint: %v", err)
	}
	if _, err := tx.ExecContext(ctx, statement); err == nil {
		t.Fatal("statement unexpectedly succeeded")
	}
	if _, err := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT expected_failure"); err != nil {
		t.Fatalf("rollback failure savepoint: %v", err)
	}
	if _, err := tx.ExecContext(ctx, "RELEASE SAVEPOINT expected_failure"); err != nil {
		t.Fatalf("release failure savepoint: %v", err)
	}
}
