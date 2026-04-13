package db_test

import (
	"context"
	"database/sql"
	"os"
	"strconv"
	"strings"
	"testing"

	appdb "github.com/MariusBobitiu/agrafa-backend/src/db"
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

	t.Run("owner only actions remain owner only", func(t *testing.T) {
		withSeededRLSTx(t, ctx, db, func(tx *sql.Tx) {
			setRLSContext(t, ctx, tx, "usr_admin", int64Ptr(-1001), stringPtr("admin"), false)
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO app.project_members (id, project_id, user_id, role)
				VALUES ('pm_fail_admin', -1001, 'usr_extra', 'viewer')
			`); err == nil {
				t.Fatal("expected admin membership insert to be denied")
			}

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
