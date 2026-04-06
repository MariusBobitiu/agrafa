package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type EventService struct {
	eventRepo *repositories.EventRepository
}

func NewEventService(eventRepo *repositories.EventRepository) *EventService {
	return &EventService{eventRepo: eventRepo}
}

func (s *EventService) ListEvents(ctx context.Context, limit int32, projectID *int64) ([]types.EventReadData, error) {
	if limit <= 0 {
		limit = 50
	}

	var (
		events []generated.Event
		err    error
	)

	if projectID != nil {
		events, err = s.eventRepo.ListByProject(ctx, *projectID, limit)
	} else {
		events, err = s.eventRepo.List(ctx, limit)
	}
	if err != nil {
		return nil, err
	}

	return mapEvents(events), nil
}

func (s *EventService) ListAlertEvents(ctx context.Context, limit int32, projectID *int64) ([]types.EventReadData, error) {
	if limit <= 0 {
		limit = 50
	}

	events, err := s.eventRepo.ListRecentAlertEvents(ctx, limit, projectID)
	if err != nil {
		return nil, err
	}

	return mapEvents(events), nil
}

func (s *EventService) CreateNodeStateChange(ctx context.Context, node generated.Node, newState string, occurredAt time.Time) error {
	eventType := types.EventTypeNodeOffline
	severity := "warning"
	title := fmt.Sprintf("Node %s went offline", node.Name)

	if newState == types.NodeStateOnline {
		eventType = types.EventTypeNodeOnline
		severity = "info"
		title = fmt.Sprintf("Node %s is online", node.Name)
	}

	details, err := json.Marshal(map[string]any{
		"node_id":       node.ID,
		"identifier":    node.Identifier,
		"current_state": newState,
	})
	if err != nil {
		return fmt.Errorf("marshal node event details: %w", err)
	}

	_, err = s.eventRepo.Create(ctx, generated.CreateEventParams{
		ProjectID:  node.ProjectID,
		NodeID:     sql.NullInt64{Int64: node.ID, Valid: true},
		ServiceID:  sql.NullInt64{},
		EventType:  eventType,
		Severity:   severity,
		Title:      title,
		Details:    details,
		OccurredAt: occurredAt,
	})
	if err != nil {
		return fmt.Errorf("create node event: %w", err)
	}

	return nil
}

func (s *EventService) CreateServiceStateChange(ctx context.Context, service generated.Service, newState string, occurredAt time.Time) error {
	eventType := types.EventTypeServiceDegraded
	severity := "warning"
	title := fmt.Sprintf("Service %s is degraded", service.Name)

	switch newState {
	case types.ServiceStateUnhealthy:
		eventType = types.EventTypeServiceUnhealthy
		severity = "critical"
		title = fmt.Sprintf("Service %s is unhealthy", service.Name)
	case types.ServiceStateHealthy:
		eventType = types.EventTypeServiceRecovered
		severity = "info"
		title = fmt.Sprintf("Service %s recovered", service.Name)
	}

	details, err := json.Marshal(map[string]any{
		"service_id":            service.ID,
		"node_id":               service.NodeID,
		"current_state":         newState,
		"consecutive_failures":  service.ConsecutiveFailures,
		"consecutive_successes": service.ConsecutiveSuccesses,
	})
	if err != nil {
		return fmt.Errorf("marshal service event details: %w", err)
	}

	_, err = s.eventRepo.Create(ctx, generated.CreateEventParams{
		ProjectID:  service.ProjectID,
		NodeID:     sql.NullInt64{Int64: service.NodeID, Valid: true},
		ServiceID:  sql.NullInt64{Int64: service.ID, Valid: true},
		EventType:  eventType,
		Severity:   severity,
		Title:      title,
		Details:    details,
		OccurredAt: occurredAt,
	})
	if err != nil {
		return fmt.Errorf("create service event: %w", err)
	}

	return nil
}

func (s *EventService) CreateAlertTriggered(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance, occurredAt time.Time) error {
	details, err := json.Marshal(map[string]any{
		"alert_rule_id":     rule.ID,
		"alert_instance_id": alert.ID,
		"rule_type":         rule.RuleType,
		"status":            alert.Status,
		"threshold_value":   nullFloat64Ptr(rule.ThresholdValue),
	})
	if err != nil {
		return fmt.Errorf("marshal alert trigger details: %w", err)
	}

	_, err = s.eventRepo.Create(ctx, generated.CreateEventParams{
		ProjectID:  alert.ProjectID,
		NodeID:     alert.NodeID,
		ServiceID:  alert.ServiceID,
		EventType:  types.EventTypeAlertTriggered,
		Severity:   alertTriggerSeverity(rule.RuleType),
		Title:      "Alert triggered: " + alert.Title,
		Details:    details,
		OccurredAt: occurredAt,
	})
	if err != nil {
		return fmt.Errorf("create alert trigger event: %w", err)
	}

	return nil
}

func (s *EventService) CreateAlertResolved(ctx context.Context, rule generated.AlertRule, alert generated.AlertInstance, occurredAt time.Time) error {
	details, err := json.Marshal(map[string]any{
		"alert_rule_id":     rule.ID,
		"alert_instance_id": alert.ID,
		"rule_type":         rule.RuleType,
		"status":            alert.Status,
		"resolved_at":       nullTimePtr(alert.ResolvedAt),
	})
	if err != nil {
		return fmt.Errorf("marshal alert resolve details: %w", err)
	}

	_, err = s.eventRepo.Create(ctx, generated.CreateEventParams{
		ProjectID:  alert.ProjectID,
		NodeID:     alert.NodeID,
		ServiceID:  alert.ServiceID,
		EventType:  types.EventTypeAlertResolved,
		Severity:   "info",
		Title:      "Alert resolved: " + alert.Title,
		Details:    details,
		OccurredAt: occurredAt,
	})
	if err != nil {
		return fmt.Errorf("create alert resolve event: %w", err)
	}

	return nil
}

func alertTriggerSeverity(ruleType string) string {
	switch ruleType {
	case types.AlertRuleTypeNodeOffline, types.AlertRuleTypeServiceUnhealthy:
		return "critical"
	default:
		return "warning"
	}
}
