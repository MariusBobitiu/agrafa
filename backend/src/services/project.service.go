package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type projectRepository interface {
	Create(ctx context.Context, params generated.CreateProjectParams) (generated.Project, error)
	CreateWithOwner(ctx context.Context, projectParams generated.CreateProjectParams, projectMemberParams generated.CreateProjectMemberParams) (generated.Project, error)
	GetByID(ctx context.Context, id int64) (generated.Project, error)
	ListForUser(ctx context.Context, userID string) ([]generated.ListProjectsForUserRow, error)
	UpdateName(ctx context.Context, id int64, name string) (generated.Project, error)
	Delete(ctx context.Context, id int64) (int64, error)
}

type projectMembershipRepository interface {
	GetByProjectAndUser(ctx context.Context, projectID int64, userID string) (generated.ProjectMember, error)
}

type projectOverviewRepository interface {
	GetStatsByProject(ctx context.Context, projectID int64) (generated.GetOverviewStatsByProjectRow, error)
}

type ProjectService struct {
	projectRepo       projectRepository
	projectMemberRepo projectMembershipRepository
	overviewRepo      projectOverviewRepository
}

func NewProjectService(
	projectRepo *repositories.ProjectRepository,
	projectMemberRepo *repositories.ProjectMemberRepository,
	overviewRepo *repositories.OverviewRepository,
) *ProjectService {
	return &ProjectService{
		projectRepo:       projectRepo,
		projectMemberRepo: projectMemberRepo,
		overviewRepo:      overviewRepo,
	}
}

func (s *ProjectService) Create(ctx context.Context, userID string, name string) (generated.Project, error) {
	userID = utils.NormalizeRequiredString(userID)
	if userID == "" {
		return generated.Project{}, types.ErrUnauthenticated
	}

	name = utils.NormalizeRequiredString(name)
	if name == "" {
		return generated.Project{}, types.ErrInvalidName
	}

	slug := utils.BuildSlug(name)
	if slug == "" {
		return generated.Project{}, types.ErrInvalidName
	}

	projectMemberID, err := utils.GenerateOpaqueID("pm", 16)
	if err != nil {
		return generated.Project{}, fmt.Errorf("generate project member id: %w", err)
	}

	project, err := s.projectRepo.CreateWithOwner(ctx, generated.CreateProjectParams{
		Slug: slug,
		Name: name,
	}, generated.CreateProjectMemberParams{
		ID:     projectMemberID,
		UserID: userID,
		Role:   ProjectRoleOwner,
	})
	if err != nil {
		return generated.Project{}, fmt.Errorf("create project: %w", err)
	}

	return project, nil
}

func (s *ProjectService) ListForUser(ctx context.Context, userID string) ([]types.ProjectSummaryData, error) {
	userID = utils.NormalizeRequiredString(userID)
	if userID == "" {
		return nil, types.ErrUnauthenticated
	}

	projects, err := s.projectRepo.ListForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	return mapProjectSummaries(projects), nil
}

func (s *ProjectService) Get(ctx context.Context, userID string, projectID int64) (types.ProjectDetailData, error) {
	userID = utils.NormalizeRequiredString(userID)
	if userID == "" {
		return types.ProjectDetailData{}, types.ErrUnauthenticated
	}
	if projectID <= 0 {
		return types.ProjectDetailData{}, types.ErrInvalidProjectID
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ProjectDetailData{}, types.ErrProjectNotFound
		}

		return types.ProjectDetailData{}, fmt.Errorf("get project: %w", err)
	}

	member, err := s.projectMemberRepo.GetByProjectAndUser(ctx, projectID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ProjectDetailData{}, types.ErrForbidden
		}

		return types.ProjectDetailData{}, fmt.Errorf("get project membership: %w", err)
	}

	stats, err := s.overviewRepo.GetStatsByProject(ctx, projectID)
	if err != nil {
		return types.ProjectDetailData{}, fmt.Errorf("get project stats: %w", err)
	}

	return types.ProjectDetailData{
		ID:               project.ID,
		Slug:             project.Slug,
		Name:             project.Name,
		CreatedAt:        project.CreatedAt,
		CurrentUserRole:  member.Role,
		NodeCount:        stats.TotalNodes,
		ServiceCount:     stats.TotalServices,
		ActiveAlertCount: stats.ActiveAlerts,
	}, nil
}

func (s *ProjectService) Delete(ctx context.Context, projectID int64) error {
	if projectID <= 0 {
		return types.ErrInvalidProjectID
	}

	rowsDeleted, err := s.projectRepo.Delete(ctx, projectID)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if rowsDeleted == 0 {
		return types.ErrProjectNotFound
	}

	return nil
}

func (s *ProjectService) Update(ctx context.Context, userID string, projectID int64, input types.UpdateProjectInput) (types.ProjectDetailData, error) {
	userID = utils.NormalizeRequiredString(userID)
	if userID == "" {
		return types.ProjectDetailData{}, types.ErrUnauthenticated
	}
	if projectID <= 0 {
		return types.ProjectDetailData{}, types.ErrInvalidProjectID
	}
	if input.Name == nil {
		return types.ProjectDetailData{}, types.ErrNoFieldsToUpdate
	}

	name := utils.NormalizeRequiredString(*input.Name)
	if name == "" {
		return types.ProjectDetailData{}, types.ErrInvalidName
	}

	if _, err := s.projectRepo.UpdateName(ctx, projectID, name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ProjectDetailData{}, types.ErrProjectNotFound
		}

		return types.ProjectDetailData{}, fmt.Errorf("update project name: %w", err)
	}

	return s.Get(ctx, userID, projectID)
}
