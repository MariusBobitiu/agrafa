package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/jackc/pgx/v5/pgconn"
)

type alertRuleEvaluatorRepository interface {
	ListEnabled(ctx context.Context, ruleType string, nodeID *int64, serviceID *int64, metricName *string) ([]generated.AlertRule, error)
}

type alertInstanceLifecycleRepository interface {
	FindActiveByRuleID(ctx context.Context, ruleID int64) (generated.AlertInstance, error)
	Create(ctx context.Context, params generated.CreateAlertInstanceParams) (generated.AlertInstance, error)
	Resolve(ctx context.Context, id int64, resolvedAt time.Time) (generated.AlertInstance, error)
}

type alertMetricRepository interface {
	GetLatestNodeMetricByName(ctx context.Context, nodeID int64, metricName string) (generated.MetricSample, error)
}

type alertEventService interface {
	CreateAlertTriggered(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance, occurredAt time.Time) error
	CreateAlertResolved(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance, occurredAt time.Time) error
}

type alertNotificationService interface {
	NotifyAlertTriggered(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance) error
	NotifyAlertResolved(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance) error
}

type AlertEvaluatorService struct {
	alertRuleRepo       alertRuleEvaluatorRepository
	alertInstanceRepo   alertInstanceLifecycleRepository
	metricRepo          alertMetricRepository
	eventService        alertEventService
	notificationService alertNotificationService
}

func NewAlertEvaluatorService(
	alertRuleRepo alertRuleEvaluatorRepository,
	alertInstanceRepo alertInstanceLifecycleRepository,
	metricRepo alertMetricRepository,
	eventService alertEventService,
	notificationService alertNotificationService,
) *AlertEvaluatorService {
	return &AlertEvaluatorService{
		alertRuleRepo:       alertRuleRepo,
		alertInstanceRepo:   alertInstanceRepo,
		metricRepo:          metricRepo,
		eventService:        eventService,
		notificationService: notificationService,
	}
}

func (s *AlertEvaluatorService) EvaluateNodeRules(ctx context.Context, node generated.Node, occurredAt time.Time) error {
	nodeID := node.ID

	rules, err := s.alertRuleRepo.ListEnabled(ctx, types.AlertRuleTypeNodeOffline, &nodeID, nil, nil)
	if err != nil {
		return fmt.Errorf("list node alert rules: %w", err)
	}

	for _, rule := range rules {
		if err := s.applyRuleCondition(ctx, rule, node.CurrentState == types.NodeStateOffline, occurredAt, nil); err != nil {
			return err
		}
	}

	return nil
}

func (s *AlertEvaluatorService) EvaluateServiceRules(ctx context.Context, service generated.Service, occurredAt time.Time) error {
	serviceID := service.ID

	rules, err := s.alertRuleRepo.ListEnabled(ctx, types.AlertRuleTypeServiceUnhealthy, nil, &serviceID, nil)
	if err != nil {
		return fmt.Errorf("list service alert rules: %w", err)
	}

	for _, rule := range rules {
		if err := s.applyRuleCondition(ctx, rule, service.CurrentState == types.ServiceStateUnhealthy, occurredAt, nil); err != nil {
			return err
		}
	}

	return nil
}

func (s *AlertEvaluatorService) EvaluateMetricRules(ctx context.Context, nodeID int64, metricName string, _ time.Time) error {
	ruleType, ok := ruleTypeForMetricName(metricName)
	if !ok {
		return nil
	}

	latestMetric, err := s.metricRepo.GetLatestNodeMetricByName(ctx, nodeID, metricName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("get latest node metric: %w", err)
	}

	rules, err := s.alertRuleRepo.ListEnabled(ctx, ruleType, &nodeID, nil, &metricName)
	if err != nil {
		return fmt.Errorf("list metric alert rules: %w", err)
	}

	for _, rule := range rules {
		metricValue := latestMetric.MetricValue
		condition := rule.ThresholdValue.Valid && metricValue > rule.ThresholdValue.Float64
		if err := s.applyRuleCondition(ctx, rule, condition, latestMetric.ObservedAt, &metricValue); err != nil {
			return err
		}
	}

	return nil
}

func (s *AlertEvaluatorService) applyRuleCondition(
	ctx context.Context,
	rule generated.AlertRule,
	conditionMet bool,
	occurredAt time.Time,
	metricValue *float64,
) error {
	activeAlert, err := s.alertInstanceRepo.FindActiveByRuleID(ctx, rule.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("find active alert instance: %w", err)
	}

	hasActiveAlert := err == nil

	switch {
	case conditionMet && hasActiveAlert:
		return nil
	case conditionMet:
		title, message := buildAlertCopy(rule, metricValue)
		alert, createErr := s.alertInstanceRepo.Create(ctx, generated.CreateAlertInstanceParams{
			AlertRuleID: rule.ID,
			ProjectID:   rule.ProjectID,
			NodeID:      rule.NodeID,
			ServiceID:   rule.ServiceID,
			Status:      types.AlertStatusActive,
			TriggeredAt: occurredAt,
			ResolvedAt:  sql.NullTime{},
			Title:       title,
			Message:     message,
		})
		if createErr != nil {
			if isUniqueViolation(createErr) {
				return nil
			}

			return fmt.Errorf("create alert instance: %w", createErr)
		}

		if s.eventService != nil {
			if err := s.eventService.CreateAlertTriggered(ctx, rule, alert, occurredAt); err != nil {
				return err
			}
		}

		if s.notificationService != nil {
			if err := s.notificationService.NotifyAlertTriggered(ctx, rule, alert); err != nil {
				log.Printf("notify alert triggered failed\n  alert_rule_id: %d\n  alert_instance_id: %d\n  error: %v", rule.ID, alert.ID, err)
			}
		}

		return nil
	case hasActiveAlert:
		resolvedAlert, resolveErr := s.alertInstanceRepo.Resolve(ctx, activeAlert.ID, occurredAt)
		if resolveErr != nil {
			if errors.Is(resolveErr, sql.ErrNoRows) {
				return nil
			}

			return fmt.Errorf("resolve alert instance: %w", resolveErr)
		}

		if s.eventService != nil {
			if err := s.eventService.CreateAlertResolved(ctx, rule, resolvedAlert, occurredAt); err != nil {
				return err
			}
		}

		if s.notificationService != nil {
			if err := s.notificationService.NotifyAlertResolved(ctx, rule, resolvedAlert); err != nil {
				log.Printf("notify alert resolved failed\n  alert_rule_id: %d\n  alert_instance_id: %d\n  error: %v", rule.ID, resolvedAlert.ID, err)
			}
		}
	}

	return nil
}

func buildAlertCopy(rule generated.AlertRule, metricValue *float64) (string, string) {
	switch rule.RuleType {
	case types.AlertRuleTypeNodeOffline:
		return "Node " + strconv.FormatInt(rule.NodeID.Int64, 10) + " is offline",
			"Node " + strconv.FormatInt(rule.NodeID.Int64, 10) + " is currently offline."
	case types.AlertRuleTypeServiceUnhealthy:
		return "Service " + strconv.FormatInt(rule.ServiceID.Int64, 10) + " is unhealthy",
			"Service " + strconv.FormatInt(rule.ServiceID.Int64, 10) + " is currently unhealthy."
	default:
		entityID := strconv.FormatInt(rule.NodeID.Int64, 10)
		threshold := formatAlertNumber(rule.ThresholdValue.Float64)
		current := ""
		if metricValue != nil {
			current = formatAlertNumber(*metricValue)
		}

		label := metricLabel(rule.RuleType)
		title := "Node " + entityID + " " + label + " is above " + threshold
		message := "Latest " + label + " for node " + entityID + " is " + current + ", above the configured threshold of " + threshold + "."
		return title, message
	}
}

func ruleTypeForMetricName(metricName string) (string, bool) {
	switch metricName {
	case types.MetricNameCPUUsage:
		return types.AlertRuleTypeCPUAboveThreshold, true
	case types.MetricNameMemoryUsage:
		return types.AlertRuleTypeMemoryAboveThreshold, true
	case types.MetricNameDiskUsage:
		return types.AlertRuleTypeDiskAboveThreshold, true
	default:
		return "", false
	}
}

func metricLabel(ruleType string) string {
	switch ruleType {
	case types.AlertRuleTypeCPUAboveThreshold:
		return "CPU usage"
	case types.AlertRuleTypeMemoryAboveThreshold:
		return "memory usage"
	case types.AlertRuleTypeDiskAboveThreshold:
		return "disk usage"
	default:
		return "metric"
	}
}

func formatAlertNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
