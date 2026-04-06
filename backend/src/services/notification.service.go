package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	emailpkg "github.com/MariusBobitiu/agrafa-backend/src/email"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type notificationDispatchRecipientRepository interface {
	ListByProjectAndChannel(ctx context.Context, projectID int64, channelType string) ([]generated.NotificationRecipient, error)
}

type notificationDispatchProjectRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Project, error)
}

type notificationDeliveryRecorder interface {
	Record(ctx context.Context, input types.CreateNotificationDeliveryInput) error
}

type alertEmailService interface {
	SendAlertTriggeredEmail(ctx context.Context, to string, data emailpkg.AlertTemplateData) error
	SendAlertResolvedEmail(ctx context.Context, to string, data emailpkg.AlertTemplateData) error
}

type NotificationService struct {
	notificationRecipientRepo notificationDispatchRecipientRepository
	projectRepo               notificationDispatchProjectRepository
	notificationDeliverySvc   notificationDeliveryRecorder
	emailService              alertEmailService
}

func NewNotificationService(
	notificationRecipientRepo *repositories.NotificationRecipientRepository,
	projectRepo *repositories.ProjectRepository,
	notificationDeliverySvc notificationDeliveryRecorder,
	emailService alertEmailService,
) *NotificationService {
	return &NotificationService{
		notificationRecipientRepo: notificationRecipientRepo,
		projectRepo:               projectRepo,
		notificationDeliverySvc:   notificationDeliverySvc,
		emailService:              emailService,
	}
}

func (s *NotificationService) NotifyAlertTriggered(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance) error {
	return s.notifyAlert(ctx, types.EventTypeAlertTriggered, rule, alert)
}

func (s *NotificationService) NotifyAlertResolved(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance) error {
	return s.notifyAlert(ctx, types.EventTypeAlertResolved, rule, alert)
}

func (s *NotificationService) notifyAlert(ctx context.Context, eventType string, rule generated.AlertRule, alert generated.AlertInstance) error {
	if s == nil || s.emailService == nil {
		return nil
	}

	recipients, err := s.notificationRecipientRepo.ListByProjectAndChannel(ctx, alert.ProjectID, types.NotificationChannelTypeEmail)
	if err != nil {
		return fmt.Errorf("list notification recipients: %w", err)
	}

	data := s.buildAlertTemplateData(ctx, rule, alert)

	for _, recipient := range recipients {
		if !recipient.IsEnabled {
			continue
		}

		attemptedAt := time.Now().UTC()
		var sendErr error
		switch eventType {
		case types.EventTypeAlertResolved:
			sendErr = s.emailService.SendAlertResolvedEmail(ctx, recipient.Target, data)
		default:
			sendErr = s.emailService.SendAlertTriggeredEmail(ctx, recipient.Target, data)
		}

		deliveryStatus := types.NotificationDeliveryStatusSent
		var errorMessage *string
		if sendErr != nil {
			deliveryStatus = types.NotificationDeliveryStatusFailed
			sendErrText := sendErr.Error()
			errorMessage = &sendErrText
			log.Printf(
				"send alert notification email failed\n  event_type: %s\n  notification_recipient_id: %d\n  project_id: %d\n  target: %s\n  error: %v",
				eventType,
				recipient.ID,
				recipient.ProjectID,
				recipient.Target,
				sendErr,
			)
		}

		if s.notificationDeliverySvc != nil {
			recipientID := recipient.ID
			alertInstanceID := alert.ID
			recordErr := s.notificationDeliverySvc.Record(ctx, types.CreateNotificationDeliveryInput{
				ProjectID:               alert.ProjectID,
				NotificationRecipientID: &recipientID,
				AlertInstanceID:         &alertInstanceID,
				ChannelType:             recipient.ChannelType,
				Target:                  recipient.Target,
				EventType:               eventType,
				Status:                  deliveryStatus,
				ErrorMessage:            errorMessage,
				SentAt:                  attemptedAt,
			})
			if recordErr != nil {
				log.Printf(
					"record notification delivery failed\n  event_type: %s\n  notification_recipient_id: %d\n  project_id: %d\n  target: %s\n  error: %v",
					eventType,
					recipient.ID,
					recipient.ProjectID,
					recipient.Target,
					recordErr,
				)
			}
		}
	}

	return nil
}

func (s *NotificationService) buildAlertTemplateData(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance) emailpkg.AlertTemplateData {
	data := emailpkg.AlertTemplateData{
		ProjectID:    alert.ProjectID,
		AlertTitle:   alert.Title,
		AlertMessage: alert.Message,
		RuleType:     rule.RuleType,
		Status:       alert.Status,
		NodeID:       nullInt64Ptr(alert.NodeID),
		ServiceID:    nullInt64Ptr(alert.ServiceID),
		TriggeredAt:  alert.TriggeredAt,
		ResolvedAt:   nullTimePtr(alert.ResolvedAt),
	}

	project, err := s.projectRepo.GetByID(ctx, alert.ProjectID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("lookup project for alert email failed\n  project_id: %d\n  error: %v", alert.ProjectID, err)
		}

		return data
	}

	data.ProjectName = project.Name
	return data
}
