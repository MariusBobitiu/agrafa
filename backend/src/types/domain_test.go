package types

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNodeReadDataMarshalsNullableFieldsAsNull(t *testing.T) {
	payload, err := json.Marshal(NodeReadData{
		ID:               1,
		ProjectID:        2,
		Name:             "node",
		Identifier:       "node-1",
		CurrentState:     NodeStateOffline,
		Metadata:         json.RawMessage(`{"region":"eu-west"}`),
		ActiveAlertCount: 0,
		ServiceCount:     0,
		CreatedAt:        time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 4, 5, 12, 1, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("marshal node read data: %v", err)
	}

	jsonString := string(payload)
	if !strings.Contains(jsonString, `"last_seen_at":null`) {
		t.Fatalf("expected last_seen_at null in payload: %s", jsonString)
	}
}

func TestServiceReadDataMarshalsNestedNullableFieldsAsNull(t *testing.T) {
	payload, err := json.Marshal(ServiceReadData{
		ID:                  1,
		ProjectID:           2,
		NodeID:              3,
		ExecutionMode:       ExecutionModeAgent,
		Name:                "service",
		CheckType:           "http",
		CheckTarget:         "https://example.com/health",
		Status:              ServiceStateHealthy,
		ConsecutiveFailures: 0,
		LatestHealthCheck: &HealthCheckSummaryData{
			ObservedAt:     time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC),
			IsSuccess:      true,
			StatusCode:     nil,
			ResponseTimeMs: nil,
			Message:        "ok",
		},
		ActiveAlertCount: 0,
	})
	if err != nil {
		t.Fatalf("marshal service read data: %v", err)
	}

	jsonString := string(payload)
	if !strings.Contains(jsonString, `"last_checked_at":null`) {
		t.Fatalf("expected last_checked_at null in payload: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"execution_mode":"agent"`) {
		t.Fatalf("expected execution_mode in payload: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"status_code":null`) {
		t.Fatalf("expected status_code null in payload: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"response_time_ms":null`) {
		t.Fatalf("expected response_time_ms null in payload: %s", jsonString)
	}
}

func TestServiceReadDataMarshalsMissingLatestHealthCheckAsNull(t *testing.T) {
	payload, err := json.Marshal(ServiceReadData{
		ID:                  1,
		ProjectID:           2,
		NodeID:              3,
		ExecutionMode:       ExecutionModeManaged,
		Name:                "service",
		CheckType:           "http",
		CheckTarget:         "https://example.com/health",
		Status:              ServiceStateHealthy,
		ConsecutiveFailures: 0,
		ActiveAlertCount:    0,
	})
	if err != nil {
		t.Fatalf("marshal service read data without health check: %v", err)
	}

	jsonString := string(payload)
	if !strings.Contains(jsonString, `"latest_health_check":null`) {
		t.Fatalf("expected latest_health_check null in payload: %s", jsonString)
	}
}
