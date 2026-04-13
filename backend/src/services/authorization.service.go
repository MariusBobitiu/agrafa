package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	appdb "github.com/MariusBobitiu/agrafa-backend/src/db"
	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type authorizationProjectMembershipRepository interface {
	GetByProjectAndUser(ctx context.Context, projectID int64, userID string) (generated.ProjectMember, error)
	GetByID(ctx context.Context, id string) (generated.ProjectMember, error)
}

type authorizationNodeRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Node, error)
}

type authorizationProjectRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Project, error)
}

type authorizationServiceRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Service, error)
}

type authorizationAlertRuleRepository interface {
	GetByID(ctx context.Context, id int64) (generated.AlertRule, error)
}

type authorizationNotificationRecipientRepository interface {
	GetByID(ctx context.Context, id int64) (generated.NotificationRecipient, error)
}

type authorizationProjectInvitationRepository interface {
	GetByID(ctx context.Context, id string) (generated.ProjectInvitation, error)
}

type AuthorizationService struct {
	projectMemberRepo         authorizationProjectMembershipRepository
	projectRepo               authorizationProjectRepository
	nodeRepo                  authorizationNodeRepository
	serviceRepo               authorizationServiceRepository
	alertRuleRepo             authorizationAlertRuleRepository
	notificationRecipientRepo authorizationNotificationRecipientRepository
	projectInvitationRepo     authorizationProjectInvitationRepository
}

func NewAuthorizationService(
	projectMemberRepo *repositories.ProjectMemberRepository,
	projectRepo *repositories.ProjectRepository,
	nodeRepo *repositories.NodeRepository,
	serviceRepo *repositories.ServiceRepository,
	alertRuleRepo *repositories.AlertRuleRepository,
	notificationRecipientRepo *repositories.NotificationRecipientRepository,
	projectInvitationRepo *repositories.ProjectInvitationRepository,
) *AuthorizationService {
	return &AuthorizationService{
		projectMemberRepo:         projectMemberRepo,
		projectRepo:               projectRepo,
		nodeRepo:                  nodeRepo,
		serviceRepo:               serviceRepo,
		alertRuleRepo:             alertRuleRepo,
		notificationRecipientRepo: notificationRecipientRepo,
		projectInvitationRepo:     projectInvitationRepo,
	}
}

func (s *AuthorizationService) RequireProjectPermission(ctx context.Context, userID string, projectID int64, permission string) (string, error) {
	if projectID <= 0 {
		return "", types.ErrInvalidProjectID
	}
	if userID == "" {
		return "", types.ErrUnauthenticated
	}

	member, err := s.projectMemberRepo.GetByProjectAndUser(ctx, projectID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", types.ErrForbidden
		}

		return "", fmt.Errorf("get project membership: %w", err)
	}

	if !RoleHasPermission(member.Role, permission) {
		return "", types.ErrForbidden
	}

	return member.Role, nil
}

func (s *AuthorizationService) ProjectIDForNode(ctx context.Context, nodeID int64) (int64, error) {
	if nodeID <= 0 {
		return 0, types.ErrInvalidNodeID
	}

	node, err := s.nodeRepo.GetByID(appdb.WithInternalRLSBypass(ctx), nodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, types.ErrNodeNotFound
		}

		return 0, fmt.Errorf("get node: %w", err)
	}

	return node.ProjectID, nil
}

func (s *AuthorizationService) ProjectIDForProject(ctx context.Context, projectID int64) (int64, error) {
	if projectID <= 0 {
		return 0, types.ErrInvalidProjectID
	}

	if _, err := s.projectRepo.GetByID(appdb.WithInternalRLSBypass(ctx), projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, types.ErrProjectNotFound
		}

		return 0, fmt.Errorf("get project: %w", err)
	}

	return projectID, nil
}

func (s *AuthorizationService) ProjectIDForService(ctx context.Context, serviceID int64) (int64, error) {
	if serviceID <= 0 {
		return 0, types.ErrInvalidServiceID
	}

	service, err := s.serviceRepo.GetByID(appdb.WithInternalRLSBypass(ctx), serviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, types.ErrServiceNotFound
		}

		return 0, fmt.Errorf("get service: %w", err)
	}

	return service.ProjectID, nil
}

func (s *AuthorizationService) ProjectIDForAlertRule(ctx context.Context, alertRuleID int64) (int64, error) {
	if alertRuleID <= 0 {
		return 0, types.ErrAlertRuleNotFound
	}

	rule, err := s.alertRuleRepo.GetByID(appdb.WithInternalRLSBypass(ctx), alertRuleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, types.ErrAlertRuleNotFound
		}

		return 0, fmt.Errorf("get alert rule: %w", err)
	}

	return rule.ProjectID, nil
}

func (s *AuthorizationService) ProjectIDForNotificationRecipient(ctx context.Context, notificationRecipientID int64) (int64, error) {
	if notificationRecipientID <= 0 {
		return 0, types.ErrNotificationRecipientNotFound
	}

	recipient, err := s.notificationRecipientRepo.GetByID(appdb.WithInternalRLSBypass(ctx), notificationRecipientID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, types.ErrNotificationRecipientNotFound
		}

		return 0, fmt.Errorf("get notification recipient: %w", err)
	}

	return recipient.ProjectID, nil
}

func (s *AuthorizationService) ProjectIDForProjectMember(ctx context.Context, projectMemberID string) (int64, error) {
	projectMemberID = strings.TrimSpace(projectMemberID)
	if projectMemberID == "" {
		return 0, types.ErrProjectMemberNotFound
	}

	member, err := s.projectMemberRepo.GetByID(appdb.WithInternalRLSBypass(ctx), projectMemberID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, types.ErrProjectMemberNotFound
		}

		return 0, fmt.Errorf("get project member: %w", err)
	}

	return member.ProjectID, nil
}

func (s *AuthorizationService) ProjectIDForProjectInvitation(ctx context.Context, projectInvitationID string) (int64, error) {
	projectInvitationID = strings.TrimSpace(projectInvitationID)
	if projectInvitationID == "" {
		return 0, types.ErrProjectInvitationNotFound
	}

	invitation, err := s.projectInvitationRepo.GetByID(appdb.WithInternalRLSBypass(ctx), projectInvitationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, types.ErrProjectInvitationNotFound
		}

		return 0, fmt.Errorf("get project invitation: %w", err)
	}

	return invitation.ProjectID, nil
}
