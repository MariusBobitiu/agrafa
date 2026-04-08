package services

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

func TestMapEventNormalizesNullableFieldsAndDetails(t *testing.T) {
	timestamp := "2026-04-05T13:48:41.960112+01:00"

	event := generated.Event{
		ID:         7,
		ProjectID:  3,
		NodeID:     sql.NullInt64{Int64: 11, Valid: true},
		ServiceID:  sql.NullInt64{},
		EventType:  "alert_resolved",
		Severity:   "info",
		Title:      "Alert resolved",
		Details:    json.RawMessage(`{"node_id":{"Int64":11,"Valid":true},"service_id":{"Int64":0,"Valid":false},"threshold_value":{"Float64":0,"Valid":false},"resolved_at":{"Time":"` + timestamp + `","Valid":true},"nested":{"value":{"Float64":80,"Valid":true}}}`),
		OccurredAt: time.Date(2026, 4, 5, 12, 48, 41, 0, time.UTC),
		CreatedAt:  time.Date(2026, 4, 5, 12, 49, 41, 0, time.UTC),
	}

	mapped := mapEvent(event)
	if mapped.NodeID == nil || *mapped.NodeID != 11 {
		t.Fatalf("expected node_id to be 11, got %#v", mapped.NodeID)
	}
	if mapped.ServiceID != nil {
		t.Fatalf("expected service_id to be nil, got %#v", mapped.ServiceID)
	}

	details, ok := mapped.Details.(map[string]any)
	if !ok {
		t.Fatalf("expected details to be a map, got %T", mapped.Details)
	}
	if details["service_id"] != nil {
		t.Fatalf("expected nested service_id to be nil, got %#v", details["service_id"])
	}
	if details["threshold_value"] != nil {
		t.Fatalf("expected threshold_value to be nil, got %#v", details["threshold_value"])
	}
	if details["resolved_at"] != timestamp {
		t.Fatalf("expected resolved_at to be %q, got %#v", timestamp, details["resolved_at"])
	}

	nested, ok := details["nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested details to be a map, got %T", details["nested"])
	}
	if value, ok := nested["value"].(json.Number); !ok || value.String() != "80" {
		t.Fatalf("expected nested value to be 80, got %#v", nested["value"])
	}
}

func TestMapAlertNormalizesNullableFields(t *testing.T) {
	resolvedAt := time.Date(2026, 4, 5, 12, 48, 41, 0, time.UTC)
	alert := generated.AlertInstance{
		ID:          9,
		AlertRuleID: 2,
		ProjectID:   1,
		NodeID:      sql.NullInt64{},
		ServiceID:   sql.NullInt64{Int64: 21, Valid: true},
		Status:      "resolved",
		TriggeredAt: time.Date(2026, 4, 5, 12, 40, 0, 0, time.UTC),
		ResolvedAt:  sql.NullTime{Time: resolvedAt, Valid: true},
		Title:       "Service unhealthy",
		Message:     "Recovered",
		CreatedAt:   time.Date(2026, 4, 5, 12, 40, 1, 0, time.UTC),
	}

	mapped := mapAlert(alert)
	if mapped.NodeID != nil {
		t.Fatalf("expected node_id to be nil, got %#v", mapped.NodeID)
	}
	if mapped.ServiceID == nil || *mapped.ServiceID != 21 {
		t.Fatalf("expected service_id to be 21, got %#v", mapped.ServiceID)
	}
	if mapped.ResolvedAt == nil || !mapped.ResolvedAt.Equal(resolvedAt) {
		t.Fatalf("expected resolved_at to be %v, got %#v", resolvedAt, mapped.ResolvedAt)
	}
}

func TestMapAlertRuleNormalizesNullableFields(t *testing.T) {
	rule := generated.AlertRule{
		ID:             4,
		ProjectID:      1,
		NodeID:         sql.NullInt64{},
		ServiceID:      sql.NullInt64{Int64: 9, Valid: true},
		RuleType:       "service_unhealthy",
		Severity:       "critical",
		MetricName:     sql.NullString{},
		ThresholdValue: sql.NullFloat64{},
		IsEnabled:      true,
		CreatedAt:      time.Date(2026, 4, 5, 12, 40, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 4, 5, 12, 41, 0, 0, time.UTC),
	}

	mapped := mapAlertRule(rule)
	if mapped.NodeID != nil {
		t.Fatalf("expected node_id to be nil, got %#v", mapped.NodeID)
	}
	if mapped.ServiceID == nil || *mapped.ServiceID != 9 {
		t.Fatalf("expected service_id to be 9, got %#v", mapped.ServiceID)
	}
	if mapped.MetricName != nil {
		t.Fatalf("expected metric_name to be nil, got %#v", mapped.MetricName)
	}
	if mapped.ThresholdValue != nil {
		t.Fatalf("expected threshold_value to be nil, got %#v", mapped.ThresholdValue)
	}
	if mapped.Severity != "critical" {
		t.Fatalf("expected severity to be critical, got %q", mapped.Severity)
	}
}

func TestMapNotificationRecipientIncludesMinSeverity(t *testing.T) {
	recipient := generated.NotificationRecipient{
		ID:          3,
		ProjectID:   1,
		ChannelType: "email",
		Target:      "ops@example.com",
		MinSeverity: "warning",
		IsEnabled:   true,
		CreatedAt:   time.Date(2026, 4, 5, 12, 40, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 5, 12, 41, 0, 0, time.UTC),
	}

	mapped := mapNotificationRecipient(recipient)
	if mapped.MinSeverity != "warning" {
		t.Fatalf("expected min_severity to be warning, got %q", mapped.MinSeverity)
	}
}

func TestMapNodeResponseNormalizesNullableFields(t *testing.T) {
	lastHeartbeatAt := time.Date(2026, 4, 5, 12, 48, 41, 0, time.UTC)
	node := generated.Node{
		ID:              1,
		ProjectID:       2,
		Name:            "node",
		Identifier:      "node-1",
		CurrentState:    "online",
		LastHeartbeatAt: sql.NullTime{Time: lastHeartbeatAt, Valid: true},
		Metadata:        json.RawMessage(`{"region":"eu-west"}`),
		CreatedAt:       time.Date(2026, 4, 5, 12, 40, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 4, 5, 12, 41, 0, 0, time.UTC),
	}

	mapped := MapNodeResponse(node)
	if mapped.LastHeartbeatAt == nil || !mapped.LastHeartbeatAt.Equal(lastHeartbeatAt) {
		t.Fatalf("expected last_heartbeat_at to be %v, got %#v", lastHeartbeatAt, mapped.LastHeartbeatAt)
	}
}

func TestMapServiceResponseNormalizesNullableFields(t *testing.T) {
	lastCheckAt := time.Date(2026, 4, 5, 12, 48, 41, 0, time.UTC)
	service := generated.Service{
		ID:                   1,
		ProjectID:            2,
		NodeID:               3,
		Name:                 "service",
		CheckType:            "http",
		CheckTarget:          "https://example.com/health",
		CurrentState:         "healthy",
		ConsecutiveFailures:  0,
		ConsecutiveSuccesses: 5,
		LastCheckAt:          sql.NullTime{Time: lastCheckAt, Valid: true},
		CreatedAt:            time.Date(2026, 4, 5, 12, 40, 0, 0, time.UTC),
		UpdatedAt:            time.Date(2026, 4, 5, 12, 41, 0, 0, time.UTC),
	}

	mapped := MapServiceResponse(service)
	if mapped.LastCheckAt == nil || !mapped.LastCheckAt.Equal(lastCheckAt) {
		t.Fatalf("expected last_check_at to be %v, got %#v", lastCheckAt, mapped.LastCheckAt)
	}
}
