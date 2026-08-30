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
	ListEnabled(ctx context.Context, projectID int64, ruleType string, nodeID *int64, serviceID *int64, metricName *string) ([]generated.AlertRule, error)
}

type alertInstanceLifecycleRepository interface {
	FindActiveByRuleAndTarget(ctx context.Context, ruleID int64, nodeID sql.NullInt64, serviceID sql.NullInt64) (generated.AlertInstance, error)
	ListActiveByRuleID(ctx context.Context, ruleID int64) ([]generated.AlertInstance, error)
	Create(ctx context.Context, params generated.CreateAlertInstanceParams) (generated.AlertInstance, error)
	Resolve(ctx context.Context, id int64, resolvedAt time.Time) (generated.AlertInstance, error)
	Close(ctx context.Context, id int64, closedAt time.Time, reason string) (generated.AlertInstance, error)
}

type alertEvaluatorNodeRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Node, error)
}

type alertMetricRepository interface {
	GetLatestNodeMetricByName(ctx context.Context, nodeID int64, metricName string) (generated.MetricSample, error)
}

type alertEventService interface {
	CreateAlertTriggered(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance, occurredAt time.Time) error
	CreateAlertResolved(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance, occurredAt time.Time) error
	CreateAlertClosed(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance, occurredAt time.Time) error
}

type alertNotificationService interface {
	NotifyAlertTriggered(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance) error
	NotifyAlertResolved(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance) error
}

type AlertEvaluatorService struct {
	alertRuleRepo       alertRuleEvaluatorRepository
	alertInstanceRepo   alertInstanceLifecycleRepository
	metricRepo          alertMetricRepository
	nodeRepo            alertEvaluatorNodeRepository
	eventService        alertEventService
	notificationService alertNotificationService
}

func NewAlertEvaluatorService(
	alertRuleRepo alertRuleEvaluatorRepository,
	alertInstanceRepo alertInstanceLifecycleRepository,
	metricRepo alertMetricRepository,
	nodeRepo alertEvaluatorNodeRepository,
	eventService alertEventService,
	notificationService alertNotificationService,
) *AlertEvaluatorService {
	return &AlertEvaluatorService{
		alertRuleRepo:       alertRuleRepo,
		alertInstanceRepo:   alertInstanceRepo,
		metricRepo:          metricRepo,
		nodeRepo:            nodeRepo,
		eventService:        eventService,
		notificationService: notificationService,
	}
}

func (s *AlertEvaluatorService) EvaluateNodeRules(ctx context.Context, node generated.Node, occurredAt time.Time) error {
	nodeID := node.ID

	rules, err := s.alertRuleRepo.ListEnabled(ctx, node.ProjectID, types.AlertRuleTypeNodeOffline, &nodeID, nil, nil)
	if err != nil {
		return fmt.Errorf("list node alert rules: %w", err)
	}

	for _, rule := range rules {
		target := alertTarget{NodeID: node.ID}
		if !node.IsVisible && !rule.NodeID.Valid {
			if err := s.closeActiveAlertForTarget(ctx, rule, target, occurredAt, types.AlertClosureReasonTargetHidden); err != nil {
				return err
			}
			continue
		}
		if err := s.applyRuleCondition(ctx, rule, target, node.CurrentState == types.NodeStateOffline, occurredAt, nil); err != nil {
			return err
		}
	}

	return nil
}

func (s *AlertEvaluatorService) EvaluateServiceRules(ctx context.Context, service generated.Service, occurredAt time.Time) error {
	serviceID := service.ID

	rules, err := s.alertRuleRepo.ListEnabled(ctx, service.ProjectID, types.AlertRuleTypeServiceUnhealthy, nil, &serviceID, nil)
	if err != nil {
		return fmt.Errorf("list service alert rules: %w", err)
	}

	for _, rule := range rules {
		if err := s.applyRuleCondition(ctx, rule, alertTarget{ServiceID: service.ID}, service.CurrentState == types.ServiceStateUnhealthy, occurredAt, nil); err != nil {
			return err
		}
	}

	return nil
}

func (s *AlertEvaluatorService) EvaluateMetricRules(ctx context.Context, nodeID int64, metricName string, occurredAt time.Time) error {
	ruleType, ok := ruleTypeForMetricName(metricName)
	if !ok {
		return nil
	}

	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("get metric node: %w", err)
	}

	latestMetric, err := s.metricRepo.GetLatestNodeMetricByName(ctx, nodeID, metricName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("get latest node metric: %w", err)
	}

	rules, err := s.alertRuleRepo.ListEnabled(ctx, node.ProjectID, ruleType, &nodeID, nil, &metricName)
	if err != nil {
		return fmt.Errorf("list metric alert rules: %w", err)
	}

	for _, rule := range rules {
		target := alertTarget{NodeID: nodeID}
		if !node.IsVisible && !rule.NodeID.Valid {
			if err := s.closeActiveAlertForTarget(ctx, rule, target, occurredAt, types.AlertClosureReasonTargetHidden); err != nil {
				return err
			}
			continue
		}
		metricValue := latestMetric.MetricValue
		condition := rule.ThresholdValue.Valid && metricValue > rule.ThresholdValue.Float64
		if err := s.applyRuleCondition(ctx, rule, target, condition, latestMetric.ObservedAt, &metricValue); err != nil {
			return err
		}
	}

	return nil
}

type alertTarget struct {
	NodeID    int64
	ServiceID int64
}

func (target alertTarget) nullableNodeID() sql.NullInt64 {
	return sql.NullInt64{Int64: target.NodeID, Valid: target.NodeID > 0}
}

func (target alertTarget) nullableServiceID() sql.NullInt64 {
	return sql.NullInt64{Int64: target.ServiceID, Valid: target.ServiceID > 0}
}

func (s *AlertEvaluatorService) applyRuleCondition(
	ctx context.Context,
	rule generated.AlertRule,
	target alertTarget,
	conditionMet bool,
	occurredAt time.Time,
	metricValue *float64,
) error {
	activeAlert, hasActiveAlert, err := s.findActiveAlertForTarget(ctx, rule, target)
	if err != nil {
		return err
	}

	nodeID := target.nullableNodeID()
	serviceID := target.nullableServiceID()

	switch {
	case conditionMet && hasActiveAlert:
		return nil
	case conditionMet:
		title, message := buildAlertCopy(rule, target, metricValue)
		triggerSnapshot, snapshotErr := buildMetricAlertTriggerSnapshot(rule, metricValue)
		if snapshotErr != nil {
			return fmt.Errorf("build alert trigger snapshot: %w", snapshotErr)
		}
		alert, createErr := s.alertInstanceRepo.Create(ctx, generated.CreateAlertInstanceParams{
			AlertRuleID:     rule.ID,
			ProjectID:       rule.ProjectID,
			NodeID:          nodeID,
			ServiceID:       serviceID,
			Status:          types.AlertStatusActive,
			TriggeredAt:     occurredAt,
			ResolvedAt:      sql.NullTime{},
			ClosedAt:        sql.NullTime{},
			ClosureReason:   sql.NullString{},
			Title:           title,
			Message:         message,
			TriggerSnapshot: triggerSnapshot,
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
		return s.resolveActiveAlert(ctx, rule, activeAlert, occurredAt)
	}

	return nil
}

func (s *AlertEvaluatorService) ReconcileRule(ctx context.Context, rule generated.AlertRule, occurredAt time.Time) error {
	activeAlerts, err := s.alertInstanceRepo.ListActiveByRuleID(ctx, rule.ID)
	if err != nil {
		return fmt.Errorf("list active alert instances for reconciliation: %w", err)
	}

	for _, alert := range activeAlerts {
		if rule.IsEnabled && ruleAppliesToAlert(rule, alert) {
			continue
		}
		reason := types.AlertClosureReasonRuleScopeChanged
		if !rule.IsEnabled {
			reason = types.AlertClosureReasonRuleDisabled
		}
		if err := s.closeActiveAlert(ctx, rule, alert, occurredAt, reason); err != nil {
			return err
		}
	}

	return nil
}

func (s *AlertEvaluatorService) closeActiveAlertForTarget(
	ctx context.Context,
	rule generated.AlertRule,
	target alertTarget,
	occurredAt time.Time,
	reason string,
) error {
	activeAlert, hasActiveAlert, err := s.findActiveAlertForTarget(ctx, rule, target)
	if err != nil {
		return err
	}
	if !hasActiveAlert {
		return nil
	}

	return s.closeActiveAlert(ctx, rule, activeAlert, occurredAt, reason)
}

func (s *AlertEvaluatorService) closeActiveAlert(
	ctx context.Context,
	rule generated.AlertRule,
	activeAlert generated.AlertInstance,
	occurredAt time.Time,
	reason string,
) error {
	closedAlert, err := s.alertInstanceRepo.Close(ctx, activeAlert.ID, occurredAt, reason)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("close alert instance: %w", err)
	}

	if s.eventService != nil {
		if err := s.eventService.CreateAlertClosed(ctx, rule, closedAlert, occurredAt); err != nil {
			return err
		}
	}

	return nil
}

func (s *AlertEvaluatorService) findActiveAlertForTarget(
	ctx context.Context,
	rule generated.AlertRule,
	target alertTarget,
) (generated.AlertInstance, bool, error) {
	activeAlert, err := s.alertInstanceRepo.FindActiveByRuleAndTarget(
		ctx,
		rule.ID,
		target.nullableNodeID(),
		target.nullableServiceID(),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return generated.AlertInstance{}, false, nil
	}
	if err != nil {
		return generated.AlertInstance{}, false, fmt.Errorf("find active alert instance: %w", err)
	}

	return activeAlert, true, nil
}

func ruleAppliesToAlert(rule generated.AlertRule, alert generated.AlertInstance) bool {
	if alert.ProjectID != rule.ProjectID {
		return false
	}

	if rule.RuleType == types.AlertRuleTypeServiceUnhealthy {
		return alert.ServiceID.Valid && (!rule.ServiceID.Valid || rule.ServiceID.Int64 == alert.ServiceID.Int64)
	}

	return alert.NodeID.Valid && (!rule.NodeID.Valid || rule.NodeID.Int64 == alert.NodeID.Int64)
}

func (s *AlertEvaluatorService) resolveActiveAlert(
	ctx context.Context,
	rule generated.AlertRule,
	activeAlert generated.AlertInstance,
	occurredAt time.Time,
) error {
	resolvedAlert, err := s.alertInstanceRepo.Resolve(ctx, activeAlert.ID, occurredAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("resolve alert instance: %w", err)
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

	return nil
}

func buildAlertCopy(rule generated.AlertRule, target alertTarget, metricValue *float64) (string, string) {
	switch rule.RuleType {
	case types.AlertRuleTypeNodeOffline:
		return "Node " + strconv.FormatInt(target.NodeID, 10) + " is offline",
			"Node " + strconv.FormatInt(target.NodeID, 10) + " is currently offline."
	case types.AlertRuleTypeServiceUnhealthy:
		return "Service " + strconv.FormatInt(target.ServiceID, 10) + " is unhealthy",
			"Service " + strconv.FormatInt(target.ServiceID, 10) + " is currently unhealthy."
	default:
		entityID := strconv.FormatInt(target.NodeID, 10)
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
