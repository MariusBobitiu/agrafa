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

func TestAlertTriggerSnapshotMigrationDatabaseSemantics(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("AGRAFA_RLS_TEST_DSN"))
	if dsn == "" {
		t.Skip("AGRAFA_RLS_TEST_DSN is not set")
	}

	ctx := context.Background()
	db, err := appdb.OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("close postgres: %v", err)
		}
	}()

	withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
		setRLSContext(t, ctx, tx, "", nil, nil, true)

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO app.alert_instances (
				id, alert_rule_id, project_id, node_id, status, triggered_at, resolved_at,
				title, message, trigger_snapshot
			) VALUES (
				-6201, -5001, -1001, -2001, 'resolved', NOW() - INTERVAL '1 minute', NOW(),
				'CPU usage is high', 'CPU usage exceeded its threshold',
				'{"metric_name":"cpu_usage","metric_value":91,"threshold_value":80}'::jsonb
			)
		`); err != nil {
			t.Fatalf("insert alert trigger snapshot (is migration 000018 applied?): %v", err)
		}

		var metricName string
		var metricValue, thresholdValue float64
		if err := tx.QueryRowContext(ctx, `
			SELECT
				trigger_snapshot ->> 'metric_name',
				(trigger_snapshot ->> 'metric_value')::double precision,
				(trigger_snapshot ->> 'threshold_value')::double precision
			FROM app.alert_instances
			WHERE id = -6201
		`).Scan(&metricName, &metricValue, &thresholdValue); err != nil {
			t.Fatalf("read alert trigger snapshot: %v", err)
		}

		if metricName != "cpu_usage" || metricValue != 91 || thresholdValue != 80 {
			t.Fatalf("unexpected persisted snapshot: metric=%q value=%g threshold=%g", metricName, metricValue, thresholdValue)
		}
	})
}
