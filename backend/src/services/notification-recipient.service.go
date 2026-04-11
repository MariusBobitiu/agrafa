package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	emailpkg "github.com/MariusBobitiu/agrafa-backend/src/email"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type notificationRecipientRepository interface {
	Create(ctx context.Context, params generated.CreateNotificationRecipientParams) (generated.NotificationRecipient, error)
	List(ctx context.Context, projectID *int64) ([]generated.NotificationRecipient, error)
	ListByProjectAndChannel(ctx context.Context, projectID int64, channelType string) ([]generated.NotificationRecipient, error)
	GetByID(ctx context.Context, id int64) (generated.NotificationRecipient, error)
	UpdateEnabled(ctx context.Context, id int64, isEnabled bool) (generated.NotificationRecipient, error)
	Delete(ctx context.Context, id int64) (int64, error)
}

type notificationRecipientProjectRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Project, error)
}

type notificationRecipientEmailService interface {
	SendNotificationRecipientTestEmail(ctx context.Context, to string, data emailpkg.NotificationRecipientTestTemplateData) error
}

type notificationRecipientEmailProvider interface {
	Notifications(ctx context.Context) (*emailpkg.Service, error)
}

type NotificationRecipientService struct {
	notificationRecipientRepo notificationRecipientRepository
	projectRepo               notificationRecipientProjectRepository
	emailService              notificationRecipientEmailService
	emailProvider             notificationRecipientEmailProvider
}

func NewNotificationRecipientService(
	notificationRecipientRepo *repositories.NotificationRecipientRepository,
	projectRepo *repositories.ProjectRepository,
) *NotificationRecipientService {
	return &NotificationRecipientService{
		notificationRecipientRepo: notificationRecipientRepo,
		projectRepo:               projectRepo,
	}
}

func (s *NotificationRecipientService) WithEmail(emailService notificationRecipientEmailService) {
	s.emailService = emailService
	s.emailProvider = nil
}

func (s *NotificationRecipientService) WithEmailProvider(emailProvider notificationRecipientEmailProvider) {
	s.emailProvider = emailProvider
	s.emailService = nil
}

func (s *NotificationRecipientService) Create(ctx context.Context, input types.CreateNotificationRecipientsInput) ([]types.NotificationRecipientReadData, error) {
	if input.ProjectID <= 0 {
		return nil, types.ErrInvalidProjectID
	}

	channelType := utils.NormalizeRequiredString(input.ChannelType)
	if channelType != types.NotificationChannelTypeEmail {
		return nil, types.ErrInvalidNotificationChannelType
	}

	if len(input.Recipients) == 0 {
		return nil, types.ErrEmptyNotificationRecipients
	}

	if _, err := s.projectRepo.GetByID(ctx, input.ProjectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.ErrProjectNotFound
		}

		return nil, fmt.Errorf("get project: %w", err)
	}

	dedupedRecipients := make([]generated.CreateNotificationRecipientParams, 0, len(input.Recipients))
	indexByTarget := make(map[string]int, len(input.Recipients))

	for _, rawRecipient := range input.Recipients {
		target, err := normalizeNotificationEmailTarget(rawRecipient.Target)
		if err != nil {
			return nil, err
		}

		minSeverity := normalizeAlertSeverity(rawRecipient.MinSeverity)
		if minSeverity == "" || !isSupportedAlertSeverity(minSeverity) {
			return nil, types.ErrInvalidNotificationMinSeverity
		}

		params := generated.CreateNotificationRecipientParams{
			ProjectID:   input.ProjectID,
			ChannelType: channelType,
			Target:      target,
			MinSeverity: minSeverity,
			IsEnabled:   true,
		}

		if existingIndex, ok := indexByTarget[target]; ok {
			dedupedRecipients[existingIndex] = params
			continue
		}

		indexByTarget[target] = len(dedupedRecipients)
		dedupedRecipients = append(dedupedRecipients, params)
	}

	recipients := make([]generated.NotificationRecipient, 0, len(dedupedRecipients))
	for _, recipientInput := range dedupedRecipients {
		recipient, err := s.notificationRecipientRepo.Create(ctx, recipientInput)
		if err != nil {
			return nil, fmt.Errorf("create notification recipient: %w", err)
		}

		recipients = append(recipients, recipient)
	}

	return mapNotificationRecipients(recipients), nil
}

func (s *NotificationRecipientService) List(ctx context.Context, projectID *int64) ([]types.NotificationRecipientReadData, error) {
	recipients, err := s.notificationRecipientRepo.List(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list notification recipients: %w", err)
	}

	return mapNotificationRecipients(recipients), nil
}

func (s *NotificationRecipientService) GetByID(ctx context.Context, notificationRecipientID int64) (types.NotificationRecipientReadData, error) {
	if notificationRecipientID <= 0 {
		return types.NotificationRecipientReadData{}, types.ErrNotificationRecipientNotFound
	}

	recipient, err := s.notificationRecipientRepo.GetByID(ctx, notificationRecipientID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.NotificationRecipientReadData{}, types.ErrNotificationRecipientNotFound
		}

		return types.NotificationRecipientReadData{}, fmt.Errorf("get notification recipient: %w", err)
	}

	return mapNotificationRecipient(recipient), nil
}

func (s *NotificationRecipientService) SetEnabled(ctx context.Context, input types.UpdateNotificationRecipientInput) (types.NotificationRecipientReadData, error) {
	recipient, err := s.notificationRecipientRepo.UpdateEnabled(ctx, input.ID, input.IsEnabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.NotificationRecipientReadData{}, types.ErrNotificationRecipientNotFound
		}

		return types.NotificationRecipientReadData{}, fmt.Errorf("update notification recipient enabled state: %w", err)
	}

	return mapNotificationRecipient(recipient), nil
}

func (s *NotificationRecipientService) Delete(ctx context.Context, notificationRecipientID int64) error {
	if notificationRecipientID <= 0 {
		return types.ErrNotificationRecipientNotFound
	}

	rowsDeleted, err := s.notificationRecipientRepo.Delete(ctx, notificationRecipientID)
	if err != nil {
		return fmt.Errorf("delete notification recipient: %w", err)
	}
	if rowsDeleted == 0 {
		return types.ErrNotificationRecipientNotFound
	}

	return nil
}

func (s *NotificationRecipientService) SendTestEmail(ctx context.Context, projectID int64, email string) error {
	if projectID <= 0 {
		return types.ErrInvalidProjectID
	}

	emailService, err := s.resolveEmailService(ctx)
	if err != nil {
		return err
	}
	if s == nil || emailService == nil {
		return types.ErrEmailNotConfigured
	}

	target, err := normalizeNotificationEmailTarget(email)
	if err != nil {
		return err
	}

	recipients, err := s.notificationRecipientRepo.ListByProjectAndChannel(ctx, projectID, types.NotificationChannelTypeEmail)
	if err != nil {
		return fmt.Errorf("list notification recipients: %w", err)
	}

	found := false
	for _, recipient := range recipients {
		if recipient.Target == target {
			found = true
			break
		}
	}
	if !found {
		return types.ErrNotificationRecipientNotFound
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrProjectNotFound
		}

		return fmt.Errorf("get project: %w", err)
	}

	return emailService.SendNotificationRecipientTestEmail(ctx, target, emailpkg.NotificationRecipientTestTemplateData{
		ProjectName: project.Name,
		ProjectID:   projectID,
		Recipient:   target,
		SentAt:      time.Now().UTC(),
	})
}

func (s *NotificationRecipientService) resolveEmailService(ctx context.Context) (notificationRecipientEmailService, error) {
	if s == nil {
		return nil, nil
	}

	if s.emailProvider != nil {
		emailService, err := s.emailProvider.Notifications(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve notification email service: %w", err)
		}

		return emailService, nil
	}

	return s.emailService, nil
}

func normalizeNotificationEmailTarget(value string) (string, error) {
	normalized := strings.ToLower(utils.NormalizeRequiredString(value))
	if normalized == "" {
		return "", types.ErrInvalidNotificationTarget
	}

	address, err := mail.ParseAddress(normalized)
	if err != nil || !strings.EqualFold(address.Address, normalized) {
		return "", types.ErrInvalidNotificationTarget
	}

	return normalized, nil
}
