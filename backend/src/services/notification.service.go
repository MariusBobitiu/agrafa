package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
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

type notificationDispatchNodeRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Node, error)
}

type notificationDispatchServiceRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Service, error)
}

type notificationDispatchHealthCheckRepository interface {
	GetLatestByServiceID(ctx context.Context, serviceID int64) (generated.HealthCheckResult, error)
}

type notificationDeliveryRecorder interface {
	Record(ctx context.Context, input types.CreateNotificationDeliveryInput) error
}

type alertEmailService interface {
	SendAlertTriggeredEmail(ctx context.Context, to string, data emailpkg.AlertTemplateData) error
	SendAlertResolvedEmail(ctx context.Context, to string, data emailpkg.AlertTemplateData) error
}

type alertEmailProvider interface {
	Alerts(ctx context.Context) (*emailpkg.Service, error)
}

type NotificationService struct {
	notificationRecipientRepo notificationDispatchRecipientRepository
	projectRepo               notificationDispatchProjectRepository
	notificationDeliverySvc   notificationDeliveryRecorder
	emailService              alertEmailService
	emailProvider             alertEmailProvider
	nodeRepo                  notificationDispatchNodeRepository
	serviceRepo               notificationDispatchServiceRepository
	healthCheckRepo           notificationDispatchHealthCheckRepository
	appBaseURL                string
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

func (s *NotificationService) WithEmailProvider(emailProvider alertEmailProvider) {
	s.emailProvider = emailProvider
}

func (s *NotificationService) WithAlertPresentation(
	nodeRepo notificationDispatchNodeRepository,
	serviceRepo notificationDispatchServiceRepository,
	healthCheckRepo notificationDispatchHealthCheckRepository,
	appBaseURL string,
) {
	s.nodeRepo = nodeRepo
	s.serviceRepo = serviceRepo
	s.healthCheckRepo = healthCheckRepo
	s.appBaseURL = strings.TrimRight(strings.TrimSpace(appBaseURL), "/")
}

func (s *NotificationService) NotifyAlertTriggered(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance) error {
	return s.notifyAlert(ctx, types.EventTypeAlertTriggered, rule, alert)
}

func (s *NotificationService) NotifyAlertResolved(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance) error {
	return s.notifyAlert(ctx, types.EventTypeAlertResolved, rule, alert)
}

func (s *NotificationService) notifyAlert(ctx context.Context, eventType string, rule generated.AlertRule, alert generated.AlertInstance) error {
	emailService, err := s.resolveEmailService(ctx)
	if err != nil {
		return err
	}
	if s == nil || emailService == nil {
		return nil
	}

	recipients, err := s.notificationRecipientRepo.ListByProjectAndChannel(ctx, alert.ProjectID, types.NotificationChannelTypeEmail)
	if err != nil {
		return fmt.Errorf("list notification recipients: %w", err)
	}

	data := s.buildAlertTemplateData(ctx, rule, alert)
	alertSeverity := rule.Severity

	for _, recipient := range recipients {
		if !recipient.IsEnabled {
			continue
		}

		if !shouldNotifyForSeverity(recipient.MinSeverity, alertSeverity) {
			continue
		}

		attemptedAt := time.Now().UTC()
		var sendErr error
		switch eventType {
		case types.EventTypeAlertResolved:
			sendErr = emailService.SendAlertResolvedEmail(ctx, recipient.Target, data)
		default:
			sendErr = emailService.SendAlertTriggeredEmail(ctx, recipient.Target, data)
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

func (s *NotificationService) resolveEmailService(ctx context.Context) (alertEmailService, error) {
	if s == nil {
		return nil, nil
	}

	if s.emailProvider != nil {
		emailService, err := s.emailProvider.Alerts(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve alert email service: %w", err)
		}

		return emailService, nil
	}

	return s.emailService, nil
}

func (s *NotificationService) buildAlertTemplateData(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance) emailpkg.AlertTemplateData {
	data := emailpkg.AlertTemplateData{
		ProjectID:        alert.ProjectID,
		RuleType:         rule.RuleType,
		RuleLabel:        alertRuleLabel(rule.RuleType),
		Severity:         rule.Severity,
		Status:           alert.Status,
		TriggeredAt:      alert.TriggeredAt,
		ResolvedAt:       nullTimePtr(alert.ResolvedAt),
		AlertsURL:        joinAppURL(s.appBaseURL, fmt.Sprintf("/alerts?project_id=%d", alert.ProjectID)),
		NotificationsURL: joinAppURL(s.appBaseURL, fmt.Sprintf("/settings?tab=notifications&project_id=%d", alert.ProjectID)),
	}
	if data.ResolvedAt != nil {
		data.Duration = formatAlertDuration(data.ResolvedAt.Sub(data.TriggeredAt))
	}

	project, err := s.projectRepo.GetByID(ctx, alert.ProjectID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("lookup project for alert email failed\n  project_id: %d\n  error: %v", alert.ProjectID, err)
		}

	} else {
		data.ProjectName = project.Name
	}

	s.enrichAlertResource(ctx, rule, alert, &data)
	buildAlertPresentation(&data)
	return data
}

func (s *NotificationService) enrichAlertResource(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance, data *emailpkg.AlertTemplateData) {
	switch rule.RuleType {
	case types.AlertRuleTypeServiceUnhealthy:
		data.ResourceType = types.AlertCategoryService
		data.ResourceID = nullInt64Ptr(alert.ServiceID)
		if alert.ServiceID.Valid {
			data.ResourceURL = joinAppURL(s.appBaseURL, fmt.Sprintf("/services/%d?project_id=%d", alert.ServiceID.Int64, alert.ProjectID))
		}
		if !alert.ServiceID.Valid || s.serviceRepo == nil {
			return
		}
		service, err := s.serviceRepo.GetByID(ctx, alert.ServiceID.Int64)
		if err != nil {
			logAlertEnrichmentError("service", alert.ServiceID.Int64, err)
			return
		}
		data.ServiceName = service.Name
		data.ServiceCheckType = strings.ToUpper(service.CheckType)
		data.ServiceTarget = service.CheckTarget
		data.ServiceState = service.CurrentState
		data.ResourceName = service.Name

		if s.healthCheckRepo == nil {
			return
		}
		check, err := s.healthCheckRepo.GetLatestByServiceID(ctx, service.ID)
		if err != nil {
			logAlertEnrichmentError("health check for service", service.ID, err)
			return
		}
		if check.StatusCode.Valid {
			statusCode := int(check.StatusCode.Int32)
			data.StatusCode = &statusCode
			data.FailureReason = http.StatusText(statusCode)
		}
		if check.ResponseTimeMs.Valid {
			responseTime := int64(check.ResponseTimeMs.Int32)
			data.ResponseTimeMs = &responseTime
		}
		if data.FailureReason == "" {
			data.FailureReason = strings.TrimSpace(check.Message)
		}

	case types.AlertRuleTypeNodeOffline, types.AlertRuleTypeCPUAboveThreshold, types.AlertRuleTypeMemoryAboveThreshold, types.AlertRuleTypeDiskAboveThreshold:
		data.ResourceType = types.AlertCategoryNode
		data.ResourceID = nullInt64Ptr(alert.NodeID)
		if rule.RuleType != types.AlertRuleTypeNodeOffline {
			data.MetricLabel = alertMetricLabel(rule.RuleType)
			if snapshot, ok := parseMetricAlertTriggerSnapshot(alert.TriggerSnapshot); ok {
				data.MetricName = snapshot.MetricName
				data.MetricValue = snapshot.MetricValue
				data.ThresholdValue = snapshot.ThresholdValue
			}
		}
		if alert.NodeID.Valid {
			data.ResourceURL = joinAppURL(s.appBaseURL, fmt.Sprintf("/nodes/%d?project_id=%d", alert.NodeID.Int64, alert.ProjectID))
		}
		if !alert.NodeID.Valid || s.nodeRepo == nil {
			return
		}
		node, err := s.nodeRepo.GetByID(ctx, alert.NodeID.Int64)
		if err != nil {
			logAlertEnrichmentError("node", alert.NodeID.Int64, err)
			return
		}
		data.NodeName = node.Name
		data.NodeIdentifier = node.Identifier
		data.NodeState = node.CurrentState
		data.LastSeenAt = nullTimePtr(node.LastHeartbeatAt)
		data.ResourceName = node.Name
	}
}

func buildAlertPresentation(data *emailpkg.AlertTemplateData) {
	resolved := data.Status == types.AlertStatusResolved
	switch data.RuleType {
	case types.AlertRuleTypeNodeOffline:
		name := fallbackAlertName(data.NodeName, "Node")
		if resolved {
			data.AlertTitle = "✓ " + name + " is back online"
			data.AlertMessage = "Heartbeat and health checks have recovered. The node is responding normally again."
		} else {
			data.AlertTitle = "⚠ " + name + " is offline"
			data.AlertMessage = "Agrafa stopped receiving heartbeats from this node."
		}
	case types.AlertRuleTypeServiceUnhealthy:
		name := fallbackAlertName(data.ServiceName, "Service")
		if resolved {
			data.AlertTitle = "✓ " + name + " has recovered"
			data.AlertMessage = fallbackAlertName(data.ServiceCheckType, "Service") + " health checks are passing again. The service is responding normally."
			return
		}
		data.AlertTitle = "⚠ " + name + " is unhealthy"
		if data.ServiceCheckType == "TCP" {
			data.AlertMessage = "TCP connection to " + fallbackAlertName(data.ServiceTarget, "the service") + " failed."
		} else if data.StatusCode != nil {
			data.AlertMessage = fmt.Sprintf("HTTP check to %s returned %d", fallbackAlertName(data.ServiceTarget, "the service"), *data.StatusCode)
			if data.FailureReason != "" {
				data.AlertMessage += " " + data.FailureReason
			}
			data.AlertMessage += "."
		} else {
			data.AlertMessage = "HTTP check to " + fallbackAlertName(data.ServiceTarget, "the service") + " failed."
		}
	default:
		metric := fallbackAlertName(data.MetricLabel, "Metric")
		node := fallbackAlertName(data.NodeName, "node")
		if resolved {
			data.AlertTitle = "✓ " + metric + " is back within threshold on " + node
			if data.ThresholdValue != nil {
				data.AlertMessage = fmt.Sprintf("%s returned below the configured %g%% threshold.", metric, *data.ThresholdValue)
			} else {
				data.AlertMessage = metric + " returned within the configured threshold."
			}
		} else {
			data.AlertTitle = "⚠ " + metric + " is high on " + node
			if data.MetricValue != nil && data.ThresholdValue != nil {
				data.AlertMessage = fmt.Sprintf("%s reached %g%%, above the configured threshold of %g%%.", metric, *data.MetricValue, *data.ThresholdValue)
			} else {
				data.AlertMessage = metric + " exceeded the configured threshold."
			}
		}
	}
}

func alertRuleLabel(ruleType string) string {
	switch ruleType {
	case types.AlertRuleTypeServiceUnhealthy:
		return "Service unhealthy"
	case types.AlertRuleTypeNodeOffline:
		return "Node offline"
	case types.AlertRuleTypeCPUAboveThreshold:
		return "CPU above threshold"
	case types.AlertRuleTypeMemoryAboveThreshold:
		return "Memory above threshold"
	case types.AlertRuleTypeDiskAboveThreshold:
		return "Disk above threshold"
	default:
		return "Alert rule"
	}
}

func alertMetricLabel(ruleType string) string {
	switch ruleType {
	case types.AlertRuleTypeCPUAboveThreshold:
		return "CPU usage"
	case types.AlertRuleTypeMemoryAboveThreshold:
		return "Memory usage"
	case types.AlertRuleTypeDiskAboveThreshold:
		return "Disk usage"
	default:
		return "Metric"
	}
}

func formatAlertDuration(duration time.Duration) string {
	if duration < 0 {
		return ""
	}
	duration = duration.Truncate(time.Second)
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int64(duration/time.Second))
	}
	if duration < time.Hour {
		minutes := int64(duration / time.Minute)
		seconds := int64(duration%time.Minute) / int64(time.Second)
		if seconds == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	if duration < 24*time.Hour {
		hours := int64(duration / time.Hour)
		minutes := int64(duration%time.Hour) / int64(time.Minute)
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	days := int64(duration / (24 * time.Hour))
	hours := int64(duration%(24*time.Hour)) / int64(time.Hour)
	if hours == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd %dh", days, hours)
}

func joinAppURL(baseURL, path string) string {
	if strings.TrimSpace(baseURL) == "" {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + path
}

func fallbackAlertName(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}

func logAlertEnrichmentError(resource string, id int64, err error) {
	if !errors.Is(err, sql.ErrNoRows) {
		log.Printf("lookup %s for alert email failed\n  resource_id: %d\n  error: %v", resource, id, err)
	}
}
