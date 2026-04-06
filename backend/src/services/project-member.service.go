package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type projectMemberRepository interface {
	Create(ctx context.Context, params generated.CreateProjectMemberParams) (generated.ProjectMember, error)
	GetByID(ctx context.Context, id string) (generated.ProjectMember, error)
	GetByProjectAndUser(ctx context.Context, projectID int64, userID string) (generated.ProjectMember, error)
	GetForReadByID(ctx context.Context, id string) (generated.GetProjectMemberForReadByIDRow, error)
	ListForRead(ctx context.Context, projectID int64) ([]generated.ListProjectMembersForReadRow, error)
	UpdateRole(ctx context.Context, id string, role string) (generated.ProjectMember, error)
	Delete(ctx context.Context, id string) (int64, error)
	CountOwners(ctx context.Context, projectID int64) (int64, error)
}

type projectMemberProjectRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Project, error)
}

type projectMemberUserRepository interface {
	GetByID(ctx context.Context, id string) (generated.User, error)
}

type ProjectMemberService struct {
	projectMemberRepo projectMemberRepository
	projectRepo       projectMemberProjectRepository
	userRepo          projectMemberUserRepository
}

func NewProjectMemberService(
	projectMemberRepo *repositories.ProjectMemberRepository,
	projectRepo *repositories.ProjectRepository,
	userRepo *repositories.UserRepository,
) *ProjectMemberService {
	return &ProjectMemberService{
		projectMemberRepo: projectMemberRepo,
		projectRepo:       projectRepo,
		userRepo:          userRepo,
	}
}

func (s *ProjectMemberService) List(ctx context.Context, projectID int64) ([]types.ProjectMemberReadData, error) {
	if projectID <= 0 {
		return nil, types.ErrInvalidProjectID
	}

	rows, err := s.projectMemberRepo.ListForRead(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project members: %w", err)
	}

	return mapProjectMembers(rows), nil
}

func (s *ProjectMemberService) GetByID(ctx context.Context, id string) (types.ProjectMemberReadData, error) {
	return s.getForReadByID(ctx, id)
}

func (s *ProjectMemberService) Create(ctx context.Context, input types.CreateProjectMemberInput) (types.ProjectMemberReadData, error) {
	if input.ProjectID <= 0 {
		return types.ProjectMemberReadData{}, types.ErrInvalidProjectID
	}

	userID := utils.NormalizeRequiredString(input.UserID)
	if userID == "" {
		return types.ProjectMemberReadData{}, types.ErrInvalidUserID
	}

	role, err := normalizeProjectRole(input.Role)
	if err != nil {
		return types.ProjectMemberReadData{}, err
	}

	if _, err := s.projectRepo.GetByID(ctx, input.ProjectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ProjectMemberReadData{}, types.ErrProjectNotFound
		}

		return types.ProjectMemberReadData{}, fmt.Errorf("get project: %w", err)
	}

	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ProjectMemberReadData{}, types.ErrUserNotFound
		}

		return types.ProjectMemberReadData{}, fmt.Errorf("get user: %w", err)
	}

	if _, err := s.projectMemberRepo.GetByProjectAndUser(ctx, input.ProjectID, userID); err == nil {
		return types.ProjectMemberReadData{}, types.ErrProjectMemberAlreadyExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return types.ProjectMemberReadData{}, fmt.Errorf("get existing project membership: %w", err)
	}

	projectMemberID, err := utils.GenerateOpaqueID("pm", 16)
	if err != nil {
		return types.ProjectMemberReadData{}, fmt.Errorf("generate project member id: %w", err)
	}

	member, err := s.projectMemberRepo.Create(ctx, generated.CreateProjectMemberParams{
		ID:        projectMemberID,
		ProjectID: input.ProjectID,
		UserID:    userID,
		Role:      role,
	})
	if err != nil {
		return types.ProjectMemberReadData{}, fmt.Errorf("create project member: %w", err)
	}

	return s.getForReadByID(ctx, member.ID)
}

func (s *ProjectMemberService) UpdateRole(ctx context.Context, input types.UpdateProjectMemberInput) (types.ProjectMemberReadData, error) {
	member, err := s.projectMemberRepo.GetByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ProjectMemberReadData{}, types.ErrProjectMemberNotFound
		}

		return types.ProjectMemberReadData{}, fmt.Errorf("get project member: %w", err)
	}

	role, err := normalizeProjectRole(input.Role)
	if err != nil {
		return types.ProjectMemberReadData{}, err
	}

	if member.Role == role {
		return s.getForReadByID(ctx, member.ID)
	}

	if member.Role == ProjectRoleOwner && role != ProjectRoleOwner {
		ownerCount, err := s.projectMemberRepo.CountOwners(ctx, member.ProjectID)
		if err != nil {
			return types.ProjectMemberReadData{}, fmt.Errorf("count project owners: %w", err)
		}

		if ownerCount <= 1 {
			return types.ProjectMemberReadData{}, types.ErrCannotRemoveLastProjectOwner
		}
	}

	if _, err := s.projectMemberRepo.UpdateRole(ctx, member.ID, role); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ProjectMemberReadData{}, types.ErrProjectMemberNotFound
		}

		return types.ProjectMemberReadData{}, fmt.Errorf("update project member role: %w", err)
	}

	return s.getForReadByID(ctx, member.ID)
}

func (s *ProjectMemberService) Delete(ctx context.Context, id string) error {
	member, err := s.projectMemberRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrProjectMemberNotFound
		}

		return fmt.Errorf("get project member: %w", err)
	}

	if member.Role == ProjectRoleOwner {
		ownerCount, err := s.projectMemberRepo.CountOwners(ctx, member.ProjectID)
		if err != nil {
			return fmt.Errorf("count project owners: %w", err)
		}

		if ownerCount <= 1 {
			return types.ErrCannotRemoveLastProjectOwner
		}
	}

	rowsDeleted, err := s.projectMemberRepo.Delete(ctx, member.ID)
	if err != nil {
		return fmt.Errorf("delete project member: %w", err)
	}
	if rowsDeleted == 0 {
		return types.ErrProjectMemberNotFound
	}

	return nil
}

func (s *ProjectMemberService) getForReadByID(ctx context.Context, id string) (types.ProjectMemberReadData, error) {
	row, err := s.projectMemberRepo.GetForReadByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ProjectMemberReadData{}, types.ErrProjectMemberNotFound
		}

		return types.ProjectMemberReadData{}, fmt.Errorf("get project member for read: %w", err)
	}

	return mapProjectMember(row), nil
}

func normalizeProjectRole(role string) (string, error) {
	role = strings.ToLower(utils.NormalizeRequiredString(role))
	if !IsValidProjectRole(role) {
		return "", types.ErrInvalidProjectMemberRole
	}

	return role, nil
}
