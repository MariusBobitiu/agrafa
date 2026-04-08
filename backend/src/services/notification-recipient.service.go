package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type notificationRecipientRepository interface {
	Create(ctx context.Context, params generated.CreateNotificationRecipientParams) (generated.NotificationRecipient, error)
	List(ctx context.Context, projectID *int64) ([]generated.NotificationRecipient, error)
	GetByID(ctx context.Context, id int64) (generated.NotificationRecipient, error)
	UpdateEnabled(ctx context.Context, id int64, isEnabled bool) (generated.NotificationRecipient, error)
	Delete(ctx context.Context, id int64) (int64, error)
}

type notificationRecipientProjectRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Project, error)
}

type NotificationRecipientService struct {
	notificationRecipientRepo notificationRecipientRepository
	projectRepo               notificationRecipientProjectRepository
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
