package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	emailpkg "github.com/MariusBobitiu/agrafa-backend/src/email"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

const projectInvitationTTL = 7 * 24 * time.Hour

type projectInvitationRepository interface {
	CreateReplacingPending(ctx context.Context, params generated.CreateProjectInvitationParams) (generated.ProjectInvitation, error)
	GetByID(ctx context.Context, id string) (generated.ProjectInvitation, error)
	GetActiveByProjectAndEmail(ctx context.Context, projectID int64, email string, now time.Time) (generated.ProjectInvitation, error)
	ListByProjectID(ctx context.Context, projectID int64) ([]generated.ProjectInvitation, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (generated.GetProjectInvitationByTokenHashRow, error)
	MarkAccepted(ctx context.Context, id string, acceptedAt time.Time) (generated.ProjectInvitation, error)
	Delete(ctx context.Context, id string) (int64, error)
}

type projectInvitationProjectRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Project, error)
}

type projectInvitationMemberRepository interface {
	Create(ctx context.Context, params generated.CreateProjectMemberParams) (generated.ProjectMember, error)
	GetByProjectAndUser(ctx context.Context, projectID int64, userID string) (generated.ProjectMember, error)
}

type projectInvitationUserRepository interface {
	MarkEmailVerifiedByID(ctx context.Context, id string) error
}

type projectInviteEmailSender interface {
	SendProjectInvite(ctx context.Context, to string, data emailpkg.ProjectInviteTemplateData) error
}

type projectInviteEmailProvider interface {
	Notifications(ctx context.Context) (*emailpkg.Service, error)
}

type ProjectInvitationService struct {
	invitationRepo projectInvitationRepository
	projectRepo    projectInvitationProjectRepository
	memberRepo     projectInvitationMemberRepository
	userRepo       projectInvitationUserRepository
	tokenService   *VerificationTokenService
	emailService   projectInviteEmailSender
	emailProvider  projectInviteEmailProvider
	appBaseURL     string
	now            func() time.Time
}

func NewProjectInvitationService(
	invitationRepo *repositories.ProjectInvitationRepository,
	projectRepo *repositories.ProjectRepository,
	memberRepo *repositories.ProjectMemberRepository,
	userRepo *repositories.UserRepository,
) *ProjectInvitationService {
	return &ProjectInvitationService{
		invitationRepo: invitationRepo,
		projectRepo:    projectRepo,
		memberRepo:     memberRepo,
		userRepo:       userRepo,
		tokenService:   NewVerificationTokenService(),
		now:            time.Now,
	}
}

func (s *ProjectInvitationService) WithEmail(emailService projectInviteEmailSender, appBaseURL string) *ProjectInvitationService {
	s.emailService = emailService
	s.emailProvider = nil
	s.appBaseURL = strings.TrimRight(strings.TrimSpace(appBaseURL), "/")
	return s
}

func (s *ProjectInvitationService) WithEmailProvider(emailProvider projectInviteEmailProvider, appBaseURL string) *ProjectInvitationService {
	s.emailProvider = emailProvider
	s.emailService = nil
	s.appBaseURL = strings.TrimRight(strings.TrimSpace(appBaseURL), "/")
	return s
}

func (s *ProjectInvitationService) Create(ctx context.Context, input types.CreateProjectInvitationInput) (types.ProjectInvitationReadData, error) {
	project, invitedByUserID, err := s.loadCreateContext(ctx, input.ProjectID, input.InvitedByUserID)
	if err != nil {
		return types.ProjectInvitationReadData{}, err
	}

	email, role, err := normalizeInvitationCreateItem(input.Email, input.Role)
	if err != nil {
		return types.ProjectInvitationReadData{}, err
	}

	return s.createForProject(ctx, project, input.ProjectID, email, role, invitedByUserID, input.InvitedByName)
}

func (s *ProjectInvitationService) CreateMany(ctx context.Context, inputs []types.CreateProjectInvitationInput) (types.ProjectInvitationCreateBatchData, error) {
	if len(inputs) == 0 {
		return types.ProjectInvitationCreateBatchData{}, types.ErrEmptyProjectInvitations
	}

	projectID := inputs[0].ProjectID
	project, invitedByUserID, err := s.loadCreateContext(ctx, projectID, inputs[0].InvitedByUserID)
	if err != nil {
		return types.ProjectInvitationCreateBatchData{}, err
	}

	results := make([]types.ProjectInvitationCreateResultData, 0, len(inputs))
	seenEmails := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		normalizedRole := strings.ToLower(utils.NormalizeRequiredString(input.Role))
		normalizedEmail, err := utils.NormalizeEmail(input.Email)
		if err != nil {
			results = append(results, failedProjectInvitationCreateResult(strings.TrimSpace(input.Email), normalizedRole, "invalid_email", types.ErrInvalidEmail.Error()))
			continue
		}

		if _, exists := seenEmails[normalizedEmail]; exists {
			results = append(results, failedProjectInvitationCreateResult(normalizedEmail, normalizedRole, "duplicate_in_request", "This email appears more than once in the same request."))
			continue
		}
		seenEmails[normalizedEmail] = struct{}{}

		role, err := normalizeInvitationRole(input.Role)
		if err != nil {
			results = append(results, failedProjectInvitationCreateResult(normalizedEmail, normalizedRole, "invalid_role", types.ErrInvalidProjectInvitationRole.Error()))
			continue
		}

		invitation, err := s.createForProject(ctx, project, projectID, normalizedEmail, role, invitedByUserID, input.InvitedByName)
		if err != nil {
			errorCode, errorMessage := classifyProjectInvitationCreateError(err)
			results = append(results, failedProjectInvitationCreateResult(normalizedEmail, role, errorCode, errorMessage))
			continue
		}

		results = append(results, types.ProjectInvitationCreateResultData{
			Email:      normalizedEmail,
			Role:       role,
			Status:     "created",
			Invitation: &invitation,
		})
	}

	return types.ProjectInvitationCreateBatchData{
		ProjectID: projectID,
		Results:   results,
	}, nil
}

func (s *ProjectInvitationService) List(ctx context.Context, projectID int64) ([]types.ProjectInvitationReadData, error) {
	if projectID <= 0 {
		return nil, types.ErrInvalidProjectID
	}

	invitations, err := s.invitationRepo.ListByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project invitations: %w", err)
	}

	return mapProjectInvitations(invitations), nil
}

func (s *ProjectInvitationService) GetByToken(ctx context.Context, rawToken string) (types.ProjectInvitationLookupData, error) {
	invitation, err := s.getValidInvitationByToken(ctx, rawToken)
	if err != nil {
		return types.ProjectInvitationLookupData{}, err
	}

	return types.ProjectInvitationLookupData{
		ID:          invitation.ID,
		ProjectID:   invitation.ProjectID,
		ProjectName: invitation.ProjectName,
		Email:       invitation.Email,
		Role:        invitation.Role,
		ExpiresAt:   invitation.ExpiresAt,
	}, nil
}

func (s *ProjectInvitationService) Accept(ctx context.Context, rawToken string, user generated.User) (bool, error) {
	if strings.TrimSpace(user.ID) == "" {
		return false, types.ErrUnauthenticated
	}

	invitation, err := s.getValidInvitationByToken(ctx, rawToken)
	if err != nil {
		return false, err
	}

	userEmail, err := utils.NormalizeEmail(user.Email)
	if err != nil {
		return false, err
	}
	if userEmail != invitation.Email {
		return false, types.ErrProjectInvitationEmailMismatch
	}

	alreadyMember := false
	if _, err := s.memberRepo.GetByProjectAndUser(ctx, invitation.ProjectID, user.ID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("get project membership: %w", err)
		}

		projectMemberID, err := utils.GenerateOpaqueID("pm", 16)
		if err != nil {
			return false, fmt.Errorf("generate project member id: %w", err)
		}

		if _, err := s.memberRepo.Create(ctx, generated.CreateProjectMemberParams{
			ID:        projectMemberID,
			ProjectID: invitation.ProjectID,
			UserID:    user.ID,
			Role:      invitation.Role,
		}); err != nil {
			return false, fmt.Errorf("create project membership: %w", err)
		}
	} else {
		alreadyMember = true
	}

	if _, err := s.invitationRepo.MarkAccepted(ctx, invitation.ID, s.now().UTC()); err != nil {
		return false, fmt.Errorf("mark project invitation accepted: %w", err)
	}

	if !user.EmailVerified && s.userRepo != nil {
		if err := s.userRepo.MarkEmailVerifiedByID(ctx, user.ID); err != nil {
			return false, fmt.Errorf("mark user email verified: %w", err)
		}
	}

	return alreadyMember, nil
}

func (s *ProjectInvitationService) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return types.ErrProjectInvitationNotFound
	}

	rowsDeleted, err := s.invitationRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete project invitation: %w", err)
	}
	if rowsDeleted == 0 {
		return types.ErrProjectInvitationNotFound
	}

	return nil
}

func (s *ProjectInvitationService) getValidInvitationByToken(ctx context.Context, rawToken string) (generated.GetProjectInvitationByTokenHashRow, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return generated.GetProjectInvitationByTokenHashRow{}, types.ErrInvalidProjectInvitation
	}

	invitation, err := s.invitationRepo.GetByTokenHash(ctx, s.tokenService.HashToken(rawToken))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.GetProjectInvitationByTokenHashRow{}, types.ErrInvalidProjectInvitation
		}

		return generated.GetProjectInvitationByTokenHashRow{}, fmt.Errorf("get project invitation: %w", err)
	}

	if invitation.AcceptedAt.Valid || !invitation.ExpiresAt.After(s.now().UTC()) {
		return generated.GetProjectInvitationByTokenHashRow{}, types.ErrInvalidProjectInvitation
	}

	return invitation, nil
}

func (s *ProjectInvitationService) buildAcceptURL(rawToken string) string {
	baseURL := s.appBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	return baseURL + "/invite?token=" + url.QueryEscape(rawToken)
}

func (s *ProjectInvitationService) loadCreateContext(ctx context.Context, projectID int64, invitedByUserID string) (generated.Project, string, error) {
	if projectID <= 0 {
		return generated.Project{}, "", types.ErrInvalidProjectID
	}

	invitedByUserID = strings.TrimSpace(invitedByUserID)
	if invitedByUserID == "" {
		return generated.Project{}, "", types.ErrUnauthenticated
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.Project{}, "", types.ErrProjectNotFound
		}

		return generated.Project{}, "", fmt.Errorf("get project: %w", err)
	}

	return project, invitedByUserID, nil
}

func (s *ProjectInvitationService) createForProject(ctx context.Context, project generated.Project, projectID int64, email string, role string, invitedByUserID string, invitedByName string) (types.ProjectInvitationReadData, error) {
	if _, err := s.invitationRepo.GetActiveByProjectAndEmail(ctx, projectID, email, s.now().UTC()); err == nil {
		return types.ProjectInvitationReadData{}, types.ErrProjectInvitationAlreadyExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return types.ProjectInvitationReadData{}, fmt.Errorf("get active project invitation: %w", err)
	}

	invitationID, err := utils.GenerateOpaqueID("pinv", 16)
	if err != nil {
		return types.ProjectInvitationReadData{}, fmt.Errorf("generate invitation id: %w", err)
	}

	rawToken, err := s.tokenService.GenerateToken()
	if err != nil {
		return types.ProjectInvitationReadData{}, fmt.Errorf("generate invitation token: %w", err)
	}

	invitation, err := s.invitationRepo.CreateReplacingPending(ctx, generated.CreateProjectInvitationParams{
		ID:              invitationID,
		ProjectID:       projectID,
		Email:           email,
		Role:            role,
		TokenHash:       s.tokenService.HashToken(rawToken),
		InvitedByUserID: invitedByUserID,
		ExpiresAt:       s.now().UTC().Add(projectInvitationTTL),
	})
	if err != nil {
		return types.ProjectInvitationReadData{}, fmt.Errorf("create project invitation: %w", err)
	}

	emailService, err := s.resolveEmailService(ctx)
	if err != nil {
		return types.ProjectInvitationReadData{}, err
	}

	if emailService != nil {
		if err := emailService.SendProjectInvite(ctx, email, emailpkg.ProjectInviteTemplateData{
			ProjectName: project.Name,
			Role:        role,
			InviterName: strings.TrimSpace(invitedByName),
			AcceptURL:   s.buildAcceptURL(rawToken),
		}); err != nil {
			return types.ProjectInvitationReadData{}, fmt.Errorf("send invite email: %w", err)
		}
	}

	return mapProjectInvitation(invitation), nil
}

func (s *ProjectInvitationService) resolveEmailService(ctx context.Context) (projectInviteEmailSender, error) {
	if s.emailProvider != nil {
		emailService, err := s.emailProvider.Notifications(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve invitation email service: %w", err)
		}

		return emailService, nil
	}

	return s.emailService, nil
}

func normalizeInvitationRole(role string) (string, error) {
	role = strings.ToLower(utils.NormalizeRequiredString(role))
	switch role {
	case ProjectRoleAdmin, ProjectRoleViewer:
		return role, nil
	default:
		return "", types.ErrInvalidProjectInvitationRole
	}
}

func normalizeInvitationCreateItem(email string, role string) (string, string, error) {
	normalizedEmail, err := utils.NormalizeEmail(email)
	if err != nil {
		return "", "", err
	}

	normalizedRole, err := normalizeInvitationRole(role)
	if err != nil {
		return "", "", err
	}

	return normalizedEmail, normalizedRole, nil
}

func failedProjectInvitationCreateResult(email string, role string, errorCode string, errorMessage string) types.ProjectInvitationCreateResultData {
	return types.ProjectInvitationCreateResultData{
		Email:        email,
		Role:         role,
		Status:       "failed",
		ErrorCode:    stringPtr(errorCode),
		ErrorMessage: stringPtr(errorMessage),
	}
}

func classifyProjectInvitationCreateError(err error) (string, string) {
	switch {
	case errors.Is(err, types.ErrInvalidEmail):
		return "invalid_email", types.ErrInvalidEmail.Error()
	case errors.Is(err, types.ErrInvalidProjectInvitationRole):
		return "invalid_role", types.ErrInvalidProjectInvitationRole.Error()
	case errors.Is(err, types.ErrProjectInvitationAlreadyExists):
		return "already_invited", "An active invitation already exists for this email."
	default:
		return "invite_creation_failed", "Unable to create invitation right now."
	}
}

func stringPtr(value string) *string {
	return &value
}
