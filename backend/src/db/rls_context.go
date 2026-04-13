package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

const (
	rlsCurrentUserIDSetting      = "app.current_user_id"
	rlsCurrentProjectIDSetting   = "app.current_project_id"
	rlsCurrentProjectRoleSetting = "app.current_project_role"
	rlsInternalBypassSetting     = "app.internal_bypass_rls"
)

type rlsContextKey struct{}

type RLSSessionContext struct {
	UserID         string
	ProjectID      *int64
	ProjectRole    string
	InternalBypass bool
}

func WithUserRLSContext(ctx context.Context, userID string) context.Context {
	session := rlsSessionFromContext(ctx)
	session.UserID = strings.TrimSpace(userID)
	return context.WithValue(ctx, rlsContextKey{}, session)
}

func WithProjectRLSContext(ctx context.Context, projectID int64, projectRole string) context.Context {
	session := rlsSessionFromContext(ctx)
	session.ProjectID = &projectID
	session.ProjectRole = strings.TrimSpace(projectRole)
	return context.WithValue(ctx, rlsContextKey{}, session)
}

func WithInternalRLSBypass(ctx context.Context) context.Context {
	session := rlsSessionFromContext(ctx)
	session.InternalBypass = true
	return context.WithValue(ctx, rlsContextKey{}, session)
}

func HasRLSSessionContext(ctx context.Context) bool {
	session, ok := ctx.Value(rlsContextKey{}).(RLSSessionContext)
	if !ok {
		return false
	}

	return session.InternalBypass || session.UserID != "" || session.ProjectID != nil || session.ProjectRole != ""
}

func ApplyRLSSessionContext(ctx context.Context, tx *sql.Tx) error {
	session := rlsSessionFromContext(ctx)

	projectID := ""
	if session.ProjectID != nil {
		projectID = strconv.FormatInt(*session.ProjectID, 10)
	}

	internalBypass := "off"
	if session.InternalBypass {
		internalBypass = "on"
	}

	if _, err := tx.ExecContext(
		ctx,
		`SELECT
			set_config($1, $2, true),
			set_config($3, $4, true),
			set_config($5, $6, true),
			set_config($7, $8, true)`,
		rlsCurrentUserIDSetting,
		session.UserID,
		rlsCurrentProjectIDSetting,
		projectID,
		rlsCurrentProjectRoleSetting,
		session.ProjectRole,
		rlsInternalBypassSetting,
		internalBypass,
	); err != nil {
		return fmt.Errorf("apply rls session context: %w", err)
	}

	return nil
}

func rlsSessionFromContext(ctx context.Context) RLSSessionContext {
	session, _ := ctx.Value(rlsContextKey{}).(RLSSessionContext)
	return session
}
