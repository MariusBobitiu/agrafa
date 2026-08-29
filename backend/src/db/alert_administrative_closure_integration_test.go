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

func TestAlertAdministrativeClosureMigrationDatabaseSemantics(t *testing.T) {
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

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO app.alert_instances (
				id, alert_rule_id, project_id, node_id, status, triggered_at,
				closed_at, closure_reason, title, message
			) VALUES (
				-6101, -5001, -1001, -2001, 'closed', NOW(),
				NOW(), 'rule_disabled', 'Closed administratively', 'Monitoring stopped'
			)
		`); err != nil {
			t.Fatalf("insert administratively closed alert: %v", err)
		}

		expectPostgresStatementRejected(t, ctx, tx, `
			INSERT INTO app.alert_instances (
				id, alert_rule_id, project_id, node_id, status, triggered_at,
				resolved_at, closed_at, closure_reason, title, message
			) VALUES (
				-6102, -5001, -1001, -2001, 'closed', NOW(),
				NOW(), NOW(), 'rule_disabled', 'Invalid closure', 'Cannot also be resolved'
			)
		`)

		expectPostgresStatementRejected(t, ctx, tx, `
			INSERT INTO app.alert_instances (
				id, alert_rule_id, project_id, node_id, status, triggered_at,
				closed_at, closure_reason, title, message
			) VALUES (
				-6103, -5001, -1001, -2001, 'closed', NOW(),
				NOW(), 'condition_cleared', 'Invalid reason', 'Unsupported closure reason'
			)
		`)
	})
}
