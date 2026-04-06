package repositories

import (
	"context"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type ProjectMemberRepository struct {
	queries *generated.Queries
}

func NewProjectMemberRepository(queries *generated.Queries) *ProjectMemberRepository {
	return &ProjectMemberRepository{queries: queries}
}

func (r *ProjectMemberRepository) Create(ctx context.Context, params generated.CreateProjectMemberParams) (generated.ProjectMember, error) {
	return r.queries.CreateProjectMember(ctx, params)
}

func (r *ProjectMemberRepository) GetByID(ctx context.Context, id string) (generated.ProjectMember, error) {
	return r.queries.GetProjectMemberByID(ctx, id)
}

func (r *ProjectMemberRepository) GetByProjectAndUser(ctx context.Context, projectID int64, userID string) (generated.ProjectMember, error) {
	return r.queries.GetProjectMemberByProjectAndUser(ctx, generated.GetProjectMemberByProjectAndUserParams{
		ProjectID: projectID,
		UserID:    userID,
	})
}

func (r *ProjectMemberRepository) GetForReadByID(ctx context.Context, id string) (generated.GetProjectMemberForReadByIDRow, error) {
	return r.queries.GetProjectMemberForReadByID(ctx, id)
}

func (r *ProjectMemberRepository) ListForRead(ctx context.Context, projectID int64) ([]generated.ListProjectMembersForReadRow, error) {
	return r.queries.ListProjectMembersForRead(ctx, projectID)
}

func (r *ProjectMemberRepository) UpdateRole(ctx context.Context, id string, role string) (generated.ProjectMember, error) {
	return r.queries.UpdateProjectMemberRole(ctx, generated.UpdateProjectMemberRoleParams{
		ID:   id,
		Role: role,
	})
}

func (r *ProjectMemberRepository) Delete(ctx context.Context, id string) (int64, error) {
	return r.queries.DeleteProjectMemberByID(ctx, id)
}

func (r *ProjectMemberRepository) CountOwners(ctx context.Context, projectID int64) (int64, error) {
	return r.queries.CountProjectOwners(ctx, projectID)
}
