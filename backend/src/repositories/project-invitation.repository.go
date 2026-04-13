package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	appdb "github.com/MariusBobitiu/agrafa-backend/src/db"
	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type ProjectInvitationRepository struct {
	db      *sql.DB
	queries *generated.Queries
}

func NewProjectInvitationRepository(db *sql.DB, queries *generated.Queries) *ProjectInvitationRepository {
	return &ProjectInvitationRepository{
		db:      db,
		queries: queries,
	}
}

func (r *ProjectInvitationRepository) CreateReplacingPending(ctx context.Context, params generated.CreateProjectInvitationParams) (generated.ProjectInvitation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return generated.ProjectInvitation{}, fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if appdb.HasRLSSessionContext(ctx) {
		if err := appdb.ApplyRLSSessionContext(ctx, tx); err != nil {
			return generated.ProjectInvitation{}, err
		}
	}

	queries := r.queries.WithTx(tx)
	if _, err := queries.DeletePendingProjectInvitationsByProjectAndEmail(ctx, generated.DeletePendingProjectInvitationsByProjectAndEmailParams{
		ProjectID: params.ProjectID,
		Email:     params.Email,
	}); err != nil {
		return generated.ProjectInvitation{}, err
	}

	invitation, err := queries.CreateProjectInvitation(ctx, params)
	if err != nil {
		return generated.ProjectInvitation{}, err
	}

	if err := tx.Commit(); err != nil {
		return generated.ProjectInvitation{}, fmt.Errorf("commit tx: %w", err)
	}

	return invitation, nil
}

func (r *ProjectInvitationRepository) GetByID(ctx context.Context, id string) (generated.ProjectInvitation, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.ProjectInvitation, error) {
		return queries.GetProjectInvitationByID(ctx, id)
	})
}

func (r *ProjectInvitationRepository) GetActiveByProjectAndEmail(ctx context.Context, projectID int64, email string, now time.Time) (generated.ProjectInvitation, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.ProjectInvitation, error) {
		return queries.GetActiveProjectInvitationByProjectAndEmail(ctx, generated.GetActiveProjectInvitationByProjectAndEmailParams{
			ProjectID: projectID,
			Email:     email,
			ExpiresAt: now.UTC(),
		})
	})
}

func (r *ProjectInvitationRepository) ListByProjectID(ctx context.Context, projectID int64) ([]generated.ProjectInvitation, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.ProjectInvitation, error) {
		return queries.ListProjectInvitationsByProjectID(ctx, projectID)
	})
}

func (r *ProjectInvitationRepository) GetByTokenHash(ctx context.Context, tokenHash string) (generated.GetProjectInvitationByTokenHashRow, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.GetProjectInvitationByTokenHashRow, error) {
		return queries.GetProjectInvitationByTokenHash(ctx, tokenHash)
	})
}

func (r *ProjectInvitationRepository) MarkAccepted(ctx context.Context, id string, acceptedAt time.Time) (generated.ProjectInvitation, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.ProjectInvitation, error) {
		return queries.MarkProjectInvitationAccepted(ctx, generated.MarkProjectInvitationAcceptedParams{
			ID:         id,
			AcceptedAt: sql.NullTime{Time: acceptedAt.UTC(), Valid: true},
		})
	})
}

func (r *ProjectInvitationRepository) Delete(ctx context.Context, id string) (int64, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (int64, error) {
		return queries.DeleteProjectInvitationByID(ctx, id)
	})
}
