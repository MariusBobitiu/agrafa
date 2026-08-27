package db_test

import (
	"context"
	"database/sql"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	appdb "github.com/MariusBobitiu/agrafa-backend/src/db"
	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestProjectScopeRLSMembershipBackedAuthorization(t *testing.T) {
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

	assertMigrationApplied(t, ctx, db)

	t.Run("member can read project resources in own project", func(t *testing.T) {
		withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
			setRLSContext(t, ctx, tx, "usr_viewer", int64Ptr(-1001), stringPtr("viewer"), false)

			var count int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM app.nodes WHERE project_id = -1001`).Scan(&count); err != nil {
				t.Fatalf("count nodes: %v", err)
			}
			if count != 1 {
				t.Fatalf("count = %d, want 1", count)
			}
		})
	})

	t.Run("non-member cannot read another project resources", func(t *testing.T) {
		withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
			setRLSContext(t, ctx, tx, "usr_outsider", int64Ptr(-1001), stringPtr("owner"), false)

			var count int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM app.nodes WHERE project_id = -1001`).Scan(&count); err != nil {
				t.Fatalf("count nodes: %v", err)
			}
			if count != 0 {
				t.Fatalf("count = %d, want 0", count)
			}
		})
	})

	t.Run("member can read service history in own project", func(t *testing.T) {
		withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
			setRLSContext(t, ctx, tx, "usr_viewer", int64Ptr(-1001), stringPtr("viewer"), false)

			var count int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM app.health_check_results WHERE service_id = -3001`).Scan(&count); err != nil {
				t.Fatalf("count service history: %v", err)
			}
			if count != 1 {
				t.Fatalf("count = %d, want 1", count)
			}
		})
	})

	t.Run("member cannot read service history from another project", func(t *testing.T) {
		withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
			setRLSContext(t, ctx, tx, "usr_viewer", int64Ptr(-1002), stringPtr("owner"), false)

			var count int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM app.health_check_results WHERE service_id = -3002`).Scan(&count); err != nil {
				t.Fatalf("count service history: %v", err)
			}
			if count != 0 {
				t.Fatalf("count = %d, want 0", count)
			}
		})
	})

	t.Run("member alert reads keep project isolation through presentation joins", func(t *testing.T) {
		withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
			setRLSContext(t, ctx, tx, "usr_viewer", int64Ptr(-1001), stringPtr("viewer"), false)

			rows, err := generated.New(tx).ListActiveAlertInstancesForRead(ctx, generated.ListActiveAlertInstancesForReadParams{})
			if err != nil {
				t.Fatalf("list active alert presentation rows: %v", err)
			}
			if len(rows) != 1 || rows[0].ProjectID != -1001 || rows[0].NodeName.String != "Project One Node" {
				t.Fatalf("visible alert rows = %#v, want only project one", rows)
			}
			if rows[0].RuleType != "node_offline" || rows[0].Severity != "critical" {
				t.Fatalf("authoritative rule fields = %q/%q", rows[0].RuleType, rows[0].Severity)
			}
		})
	})

	t.Run("full-range aggregate handles more than 2000 rows boundaries null latency and future timestamps", func(t *testing.T) {
		withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
			from := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
			to := from.Add(2504 * time.Second)
			setRLSContext(t, ctx, tx, "", nil, nil, true)
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO app.health_check_results (
					id, service_id, node_id, check_type, source, observed_at,
					is_success, status_code, response_time_ms, message, payload
				)
				SELECT
					-10000 - g, -3001, -2001, 'http', 'agent', $1::timestamptz + (g - 1) * interval '1 second',
					g <= 2004, CASE WHEN g <= 2004 THEN 200 ELSE 503 END,
					CASE WHEN g = 2 THEN NULL WHEN g <= 2004 THEN CASE WHEN g = 1 THEN 0 ELSE 10 END ELSE NULL END,
					'ok', '{}'::jsonb
				FROM generate_series(1, 2505) AS g
			`, from); err != nil {
				t.Fatalf("insert aggregate history: %v", err)
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO app.health_check_results (
					id, service_id, node_id, check_type, source, observed_at,
					is_success, status_code, response_time_ms, message, payload
				) VALUES (-20000, -3001, -2001, 'http', 'agent', $1, FALSE, 503, 9999, 'future', '{}'::jsonb)
			`, to.Add(time.Hour)); err != nil {
				t.Fatalf("insert future history: %v", err)
			}

			setRLSContext(t, ctx, tx, "usr_viewer", int64Ptr(-1001), stringPtr("viewer"), false)
			row, err := generated.New(tx).SummarizeHealthCheckHistoryByServiceID(ctx, generated.SummarizeHealthCheckHistoryByServiceIDParams{
				ServiceID: -3001, FromObservedAt: from, ToObservedAt: to,
			})
			if err != nil {
				t.Fatalf("summarize history: %v", err)
			}
			if row.TotalChecks != 2505 || row.SuccessfulChecks != 2004 {
				t.Fatalf("aggregate counts = %d/%d, want 2004/2505", row.SuccessfulChecks, row.TotalChecks)
			}
			if row.MeasuredLatencyChecks != 2003 || row.TotalLatencyMs != 20020 {
				t.Fatalf("latency aggregate = count %d total %d, want 2003/20020", row.MeasuredLatencyChecks, row.TotalLatencyMs)
			}
			if row.LastCheckedUnixNano != to.UnixNano() {
				t.Fatalf("last checked unix nano = %d, want %d", row.LastCheckedUnixNano, to.UnixNano())
			}
		})
	})

	t.Run("corrective history migration updates only trustworthy type metadata", func(t *testing.T) {
		withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
			setRLSContext(t, ctx, tx, "", nil, nil, true)
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO app.health_check_results (
					id, service_id, node_id, check_type, source, observed_at,
					is_success, message, payload
				) VALUES
					(-5001, -3001, -2001, 'http', 'managed', NOW(), TRUE, 'ok', '{"check_type":"\tTCP\t"}'::jsonb),
					(-5002, -3001, -2001, 'tcp', 'agent', NOW(), TRUE, 'ok', '{"check_type":"\nHTTP\n"}'::jsonb),
					(-5003, -3001, -2001, 'tcp', 'agent', NOW(), TRUE, 'ok', '{"type":"\t\nHTTP\n\t"}'::jsonb),
					(-5004, -3001, -2001, 'tcp', 'agent', NOW(), TRUE, 'ok', '{"check_type":"\tSMTP\n"}'::jsonb),
					(-5005, -3001, -2001, 'tcp', 'agent', NOW(), TRUE, 'ok', '{"check_type":"smtp","type":"http"}'::jsonb)
			`); err != nil {
				t.Fatalf("insert corrective migration fixtures: %v", err)
			}
			migration, err := os.ReadFile("migrations/app/000015_correct_service_history_check_types.up.sql")
			if err != nil {
				t.Fatalf("read corrective migration: %v", err)
			}
			if _, err := tx.ExecContext(ctx, string(migration)); err != nil {
				t.Fatalf("execute corrective migration: %v", err)
			}

			setRLSContext(t, ctx, tx, "usr_viewer", int64Ptr(-1001), stringPtr("viewer"), false)
			rows, err := tx.QueryContext(ctx, `
				SELECT id, check_type
				FROM app.health_check_results
				WHERE id BETWEEN -5005 AND -5001
				ORDER BY id
			`)
			if err != nil {
				t.Fatalf("read corrected history: %v", err)
			}
			defer rows.Close()
			got := map[int64]string{}
			for rows.Next() {
				var id int64
				var checkType string
				if err := rows.Scan(&id, &checkType); err != nil {
					t.Fatalf("scan corrected history: %v", err)
				}
				got[id] = checkType
			}
			if err := rows.Err(); err != nil {
				t.Fatalf("iterate corrected history: %v", err)
			}
			want := map[int64]string{-5001: "tcp", -5002: "http", -5003: "http", -5004: "tcp", -5005: "tcp"}
			for id, checkType := range want {
				if got[id] != checkType {
					t.Fatalf("row %d check_type = %q, want %q", id, got[id], checkType)
				}
			}
		})
	})

	t.Run("viewer cannot write project resources", func(t *testing.T) {
		withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
			setRLSContext(t, ctx, tx, "usr_viewer", int64Ptr(-1001), stringPtr("viewer"), false)

			_, err := tx.ExecContext(ctx, `
				INSERT INTO app.nodes (project_id, name, identifier, current_state, metadata)
				VALUES (-1001, 'Viewer Node', 'viewer-node', 'offline', '{}'::jsonb)
			`)
			if err == nil {
				t.Fatal("expected viewer insert to be denied")
			}
		})
	})

	t.Run("admin can write project resources", func(t *testing.T) {
		withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
			setRLSContext(t, ctx, tx, "usr_admin", int64Ptr(-1001), stringPtr("admin"), false)

			if _, err := tx.ExecContext(ctx, `
				INSERT INTO app.nodes (project_id, name, identifier, current_state, metadata)
				VALUES (-1001, 'Admin Node', 'admin-node', 'offline', '{}'::jsonb)
			`); err != nil {
				t.Fatalf("admin insert denied: %v", err)
			}
		})
	})

	t.Run("admin cannot perform owner only actions", func(t *testing.T) {
		withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
			setRLSContext(t, ctx, tx, "usr_admin", int64Ptr(-1001), stringPtr("admin"), false)
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO app.project_members (id, project_id, user_id, role)
				VALUES ('pm_fail_admin', -1001, 'usr_extra', 'viewer')
			`); err == nil {
				t.Fatal("expected admin membership insert to be denied")
			}
		})
	})

	t.Run("owner can perform owner only actions", func(t *testing.T) {
		withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
			setRLSContext(t, ctx, tx, "usr_owner", int64Ptr(-1001), stringPtr("owner"), false)
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO app.project_members (id, project_id, user_id, role)
				VALUES ('pm_owner_add', -1001, 'usr_extra', 'viewer')
			`); err != nil {
				t.Fatalf("owner membership insert denied: %v", err)
			}
		})
	})

	t.Run("incorrect current project role or project id alone does not grant access", func(t *testing.T) {
		withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
			setRLSContext(t, ctx, tx, "usr_outsider", int64Ptr(-1001), stringPtr("owner"), false)

			var count int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM app.nodes WHERE project_id = -1001`).Scan(&count); err != nil {
				t.Fatalf("count nodes: %v", err)
			}
			if count != 0 {
				t.Fatalf("count = %d, want 0", count)
			}

			setRLSContext(t, ctx, tx, "usr_outsider", int64Ptr(-9999), stringPtr("owner"), false)
			_, err := tx.ExecContext(ctx, `
				INSERT INTO app.nodes (project_id, name, identifier, current_state, metadata)
				VALUES (-1001, 'Forged Node', 'forged-node', 'offline', '{}'::jsonb)
			`)
			if err == nil {
				t.Fatal("expected forged context insert to be denied")
			}
		})
	})
}

func assertMigrationApplied(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_proc AS p
			JOIN pg_namespace AS n ON n.oid = p.pronamespace
			WHERE n.nspname = 'app'
			  AND p.proname = 'project_read_context_matches'
		)
	`).Scan(&exists)
	if err != nil {
		t.Fatalf("check rls migration: %v", err)
	}
	if !exists {
		t.Fatal("RLS migration is not applied; run migration 000013 before executing this test")
	}

	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'app'
			  AND table_name = 'health_check_results'
			  AND column_name = 'node_id'
		)
	`).Scan(&exists)
	if err != nil {
		t.Fatalf("check service history migration: %v", err)
	}
	if !exists {
		t.Fatal("service history migration is not applied; run migration 000014 before executing this test")
	}
}

func withSeededRLSTx(t *testing.T, ctx context.Context, db *sql.DB, fn func(*sql.Tx)) {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	setRLSContext(t, ctx, tx, "", nil, nil, true)
	seedRLSFixture(t, ctx, tx)
	setRLSContext(t, ctx, tx, "", nil, nil, false)

	fn(tx)
}

func seedRLSFixture(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()

	statements := []string{
		`INSERT INTO auth.users (id, name, email, email_verified) VALUES
			('usr_owner', 'Owner', 'owner-rls@example.com', TRUE),
			('usr_admin', 'Admin', 'admin-rls@example.com', TRUE),
			('usr_viewer', 'Viewer', 'viewer-rls@example.com', TRUE),
			('usr_outsider', 'Outsider', 'outsider-rls@example.com', TRUE),
			('usr_extra', 'Extra', 'extra-rls@example.com', TRUE)`,
		`INSERT INTO app.projects (id, slug, name) VALUES
			(-1001, 'rls-project-one', 'RLS Project One'),
			(-1002, 'rls-project-two', 'RLS Project Two')`,
		`INSERT INTO app.project_members (id, project_id, user_id, role) VALUES
			('pm_owner', -1001, 'usr_owner', 'owner'),
			('pm_admin', -1001, 'usr_admin', 'admin'),
			('pm_viewer', -1001, 'usr_viewer', 'viewer'),
			('pm_other_owner', -1002, 'usr_extra', 'owner')`,
		`INSERT INTO app.nodes (id, project_id, name, identifier, current_state, metadata) VALUES
			(-2001, -1001, 'Project One Node', 'project-one-node', 'offline', '{}'::jsonb),
			(-2002, -1002, 'Project Two Node', 'project-two-node', 'offline', '{}'::jsonb)`,
		`INSERT INTO app.services (id, project_id, node_id, name, check_type, check_target) VALUES
			(-3001, -1001, -2001, 'Project One API', 'http', 'https://one.example.com/health'),
			(-3002, -1002, -2002, 'Project Two TCP', 'tcp', 'two.example.com:443')`,
		`INSERT INTO app.health_check_results (id, service_id, node_id, check_type, source, observed_at, is_success, status_code, response_time_ms, message, payload) VALUES
			(-4001, -3001, -2001, 'http', 'agent', NOW(), TRUE, 200, 25, 'ok', '{}'::jsonb),
			(-4002, -3002, -2002, 'tcp', 'managed', NOW(), FALSE, NULL, 40, 'connection refused', '{}'::jsonb)`,
		`INSERT INTO app.alert_rules (id, project_id, node_id, service_id, rule_type, severity, is_enabled) VALUES
			(-5001, -1001, -2001, NULL, 'node_offline', 'critical', TRUE),
			(-5002, -1002, NULL, -3002, 'service_unhealthy', 'warning', TRUE)`,
		`INSERT INTO app.alert_instances (id, alert_rule_id, project_id, node_id, service_id, status, triggered_at, title, message) VALUES
			(-6001, -5001, -1001, -2001, NULL, 'active', NOW(), 'Project one node offline', 'offline'),
			(-6002, -5002, -1002, NULL, -3002, 'active', NOW(), 'Project two service unhealthy', 'unhealthy')`,
	}

	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			t.Fatalf("seed fixture: %v", err)
		}
	}
}

func setRLSContext(t *testing.T, ctx context.Context, tx *sql.Tx, userID string, projectID *int64, projectRole *string, internalBypass bool) {
	t.Helper()

	projectIDValue := ""
	if projectID != nil {
		projectIDValue = strconv.FormatInt(*projectID, 10)
	}

	roleValue := ""
	if projectRole != nil {
		roleValue = *projectRole
	}

	bypassValue := "off"
	if internalBypass {
		bypassValue = "on"
	}

	if _, err := tx.ExecContext(ctx, `
		SELECT
			set_config('app.current_user_id', $1, true),
			set_config('app.current_project_id', $2, true),
			set_config('app.current_project_role', $3, true),
			set_config('app.internal_bypass_rls', $4, true)
	`, userID, projectIDValue, roleValue, bypassValue); err != nil {
		t.Fatalf("set rls context: %v", err)
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
