package services

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeNotificationNodeRepo struct{ nodes map[int64]generated.Node }

func (r *fakeNotificationNodeRepo) GetByID(_ context.Context, id int64) (generated.Node, error) {
	if node, ok := r.nodes[id]; ok {
		return node, nil
	}
	return generated.Node{}, sql.ErrNoRows
}

type fakeNotificationServiceRepo struct{ services map[int64]generated.Service }

func (r *fakeNotificationServiceRepo) GetByID(_ context.Context, id int64) (generated.Service, error) {
	if service, ok := r.services[id]; ok {
		return service, nil
	}
	return generated.Service{}, sql.ErrNoRows
}

type fakeNotificationHealthRepo struct {
	checks map[int64]generated.HealthCheckResult
}

func (r *fakeNotificationHealthRepo) GetLatestByServiceID(_ context.Context, id int64) (generated.HealthCheckResult, error) {
	if check, ok := r.checks[id]; ok {
		return check, nil
	}
	return generated.HealthCheckResult{}, sql.ErrNoRows
}

func presentationService() *NotificationService {
	service := &NotificationService{
		projectRepo: &fakeNotificationProjectLookupRepo{projects: map[int64]generated.Project{7: {ID: 7, Name: "Test"}}},
	}
	service.WithAlertPresentation(
		&fakeNotificationNodeRepo{nodes: map[int64]generated.Node{
			31: {ID: 31, ProjectID: 7, Name: "web-01", Identifier: "web-01.internal", CurrentState: types.NodeStateOffline, LastHeartbeatAt: sql.NullTime{Time: time.Date(2026, 8, 29, 7, 48, 0, 0, time.UTC), Valid: true}},
		}},
		&fakeNotificationServiceRepo{services: map[int64]generated.Service{
			13: {ID: 13, ProjectID: 7, NodeID: 31, Name: "Landing", CheckType: "http", CheckTarget: "https://landing.example.com/health", CurrentState: types.ServiceStateUnhealthy},
			14: {ID: 14, ProjectID: 7, NodeID: 31, Name: "Database", CheckType: "tcp", CheckTarget: "db.internal:5432", CurrentState: types.ServiceStateUnhealthy},
		}},
		&fakeNotificationHealthRepo{checks: map[int64]generated.HealthCheckResult{
			13: {ServiceID: 13, CheckType: "http", StatusCode: sql.NullInt32{Int32: 503, Valid: true}, ResponseTimeMs: sql.NullInt32{Int32: 412, Valid: true}, Message: "503 Service Unavailable"},
			14: {ServiceID: 14, CheckType: "tcp", Message: "Connection refused"},
		}},
		"https://app.agrafa.test/",
	)
	return service
}

func testMetricTriggerSnapshot(t *testing.T, metricName string, metricValue, threshold float64) generated.AlertInstance {
	t.Helper()
	snapshot, err := buildMetricAlertTriggerSnapshot(generated.AlertRule{
		MetricName:     sql.NullString{String: metricName, Valid: true},
		ThresholdValue: sql.NullFloat64{Float64: threshold, Valid: true},
	}, &metricValue)
	if err != nil {
		t.Fatal(err)
	}
	return generated.AlertInstance{TriggerSnapshot: snapshot}
}

func TestBuildAlertTemplateDataEnrichesHTTPService(t *testing.T) {
	data := presentationService().buildAlertTemplateData(context.Background(), generated.AlertRule{
		RuleType: types.AlertRuleTypeServiceUnhealthy, Severity: types.AlertSeverityCritical,
	}, generated.AlertInstance{
		ProjectID: 7, ServiceID: sql.NullInt64{Int64: 13, Valid: true}, Status: types.AlertStatusActive,
		TriggeredAt: time.Date(2026, 8, 29, 7, 49, 26, 0, time.UTC), Title: "Service 13 is unhealthy",
	})

	if data.AlertTitle != "⚠ Landing is unhealthy" || strings.Contains(data.AlertTitle, "13") {
		t.Fatalf("unexpected enriched title %q", data.AlertTitle)
	}
	if data.AlertMessage != "HTTP check to https://landing.example.com/health returned 503 Service Unavailable." {
		t.Fatalf("unexpected service summary %q", data.AlertMessage)
	}
	if data.StatusCode == nil || *data.StatusCode != 503 || data.ResponseTimeMs == nil || *data.ResponseTimeMs != 412 || data.FailureReason != "Service Unavailable" {
		t.Fatalf("unexpected HTTP context: %#v", data)
	}
	if data.RuleLabel != "Service unhealthy" || data.ResourceURL != "https://app.agrafa.test/services/13?project_id=7" || data.AlertsURL != "https://app.agrafa.test/alerts?project_id=7" || data.NotificationsURL != "https://app.agrafa.test/settings?tab=notifications&project_id=7" {
		t.Fatalf("unexpected labels or URLs: %#v", data)
	}
}

func TestBuildAlertTemplateDataTCPHasNoHTTPFields(t *testing.T) {
	data := presentationService().buildAlertTemplateData(context.Background(), generated.AlertRule{
		RuleType: types.AlertRuleTypeServiceUnhealthy, Severity: types.AlertSeverityCritical,
	}, generated.AlertInstance{ProjectID: 7, ServiceID: sql.NullInt64{Int64: 14, Valid: true}, Status: types.AlertStatusActive, TriggeredAt: time.Now()})

	if data.AlertMessage != "TCP connection to db.internal:5432 failed." || data.FailureReason != "Connection refused" {
		t.Fatalf("unexpected TCP presentation: %#v", data)
	}
	if data.StatusCode != nil || data.ResponseTimeMs != nil {
		t.Fatalf("TCP alert must omit HTTP-only values: %#v", data)
	}
}

func TestBuildAlertTemplateDataNodeTriggeredAndResolvedCopy(t *testing.T) {
	service := presentationService()
	started := time.Date(2026, 8, 28, 22, 35, 57, 0, time.UTC)
	resolved := time.Date(2026, 8, 29, 8, 9, 57, 0, time.UTC)
	rule := generated.AlertRule{RuleType: types.AlertRuleTypeNodeOffline, Severity: types.AlertSeverityCritical}
	triggered := service.buildAlertTemplateData(context.Background(), rule, generated.AlertInstance{ProjectID: 7, NodeID: sql.NullInt64{Int64: 31, Valid: true}, Status: types.AlertStatusActive, TriggeredAt: started})
	if triggered.AlertTitle != "⚠ web-01 is offline" || triggered.AlertMessage != "Agrafa stopped receiving heartbeats from this node." || triggered.NodeIdentifier != "web-01.internal" || triggered.ResourceURL != "https://app.agrafa.test/nodes/31?project_id=7" || triggered.AlertsURL != "https://app.agrafa.test/alerts?project_id=7" {
		t.Fatalf("unexpected node triggered presentation: %#v", triggered)
	}

	recovered := service.buildAlertTemplateData(context.Background(), rule, generated.AlertInstance{ProjectID: 7, NodeID: sql.NullInt64{Int64: 31, Valid: true}, Status: types.AlertStatusResolved, TriggeredAt: started, ResolvedAt: sql.NullTime{Time: resolved, Valid: true}})
	if recovered.AlertTitle != "✓ web-01 is back online" || strings.Contains(recovered.AlertMessage, "offline") || recovered.Duration != "9h 34m" {
		t.Fatalf("unexpected node recovery presentation: %#v", recovered)
	}
}

func TestBuildAlertTemplateDataServiceAndMetricRecoveryCopy(t *testing.T) {
	service := presentationService()
	resolved := sql.NullTime{Time: time.Date(2026, 8, 29, 8, 10, 0, 0, time.UTC), Valid: true}
	serviceData := service.buildAlertTemplateData(context.Background(), generated.AlertRule{RuleType: types.AlertRuleTypeServiceUnhealthy, Severity: types.AlertSeverityCritical}, generated.AlertInstance{ProjectID: 7, ServiceID: sql.NullInt64{Int64: 13, Valid: true}, Status: types.AlertStatusResolved, TriggeredAt: resolved.Time.Add(-time.Minute), ResolvedAt: resolved})
	if serviceData.AlertTitle != "✓ Landing has recovered" || serviceData.AlertMessage != "HTTP health checks are passing again. The service is responding normally." {
		t.Fatalf("unexpected service recovery: %#v", serviceData)
	}

	metricAlert := testMetricTriggerSnapshot(t, types.MetricNameCPUUsage, 91, 80)
	metricAlert.ProjectID = 7
	metricAlert.NodeID = sql.NullInt64{Int64: 31, Valid: true}
	metricAlert.Status = types.AlertStatusResolved
	metricAlert.TriggeredAt = resolved.Time.Add(-time.Minute)
	metricAlert.ResolvedAt = resolved
	metricData := service.buildAlertTemplateData(context.Background(), generated.AlertRule{RuleType: types.AlertRuleTypeCPUAboveThreshold, Severity: types.AlertSeverityWarning, MetricName: sql.NullString{String: types.MetricNameCPUUsage, Valid: true}, ThresholdValue: sql.NullFloat64{Float64: 95, Valid: true}}, metricAlert)
	if metricData.AlertTitle != "✓ CPU usage is back within threshold on web-01" || metricData.AlertMessage != "CPU usage returned below the configured 80% threshold." || metricData.MetricValue == nil || *metricData.MetricValue != 91 {
		t.Fatalf("unexpected metric recovery: %#v", metricData)
	}
}

func TestBuildAlertTemplateDataMetricTriggeredIncludesObservedAndThreshold(t *testing.T) {
	alert := testMetricTriggerSnapshot(t, types.MetricNameCPUUsage, 91, 80)
	alert.ProjectID = 7
	alert.NodeID = sql.NullInt64{Int64: 31, Valid: true}
	alert.Status = types.AlertStatusActive
	alert.TriggeredAt = time.Now()
	data := presentationService().buildAlertTemplateData(context.Background(), generated.AlertRule{RuleType: types.AlertRuleTypeCPUAboveThreshold, Severity: types.AlertSeverityWarning, MetricName: sql.NullString{String: types.MetricNameCPUUsage, Valid: true}, ThresholdValue: sql.NullFloat64{Float64: 95, Valid: true}}, alert)
	if data.AlertTitle != "⚠ CPU usage is high on web-01" || data.AlertMessage != "CPU usage reached 91%, above the configured threshold of 80%." || data.MetricValue == nil || *data.MetricValue != 91 || data.ThresholdValue == nil || *data.ThresholdValue != 80 {
		t.Fatalf("unexpected metric presentation: %#v", data)
	}
	if strings.Contains(data.AlertTitle+data.AlertMessage+data.RuleLabel, types.AlertRuleTypeCPUAboveThreshold) {
		t.Fatalf("raw rule type leaked into presentation: %#v", data)
	}
}

func TestBuildAlertTemplateDataMetricRulesUseTriggerSnapshot(t *testing.T) {
	tests := []struct {
		ruleType, metricName, metricLabel string
	}{
		{types.AlertRuleTypeCPUAboveThreshold, types.MetricNameCPUUsage, "CPU usage"},
		{types.AlertRuleTypeMemoryAboveThreshold, types.MetricNameMemoryUsage, "Memory usage"},
		{types.AlertRuleTypeDiskAboveThreshold, types.MetricNameDiskUsage, "Disk usage"},
	}
	for _, test := range tests {
		t.Run(test.metricName, func(t *testing.T) {
			alert := testMetricTriggerSnapshot(t, test.metricName, 91, 80)
			alert.ProjectID = 7
			alert.NodeID = sql.NullInt64{Int64: 31, Valid: true}
			alert.Status = types.AlertStatusActive
			alert.TriggeredAt = time.Now()
			data := presentationService().buildAlertTemplateData(context.Background(), generated.AlertRule{
				RuleType: test.ruleType, Severity: types.AlertSeverityWarning,
				MetricName:     sql.NullString{String: test.metricName, Valid: true},
				ThresholdValue: sql.NullFloat64{Float64: 95, Valid: true},
			}, alert)
			if data.MetricName != test.metricName || data.MetricLabel != test.metricLabel || data.MetricValue == nil || *data.MetricValue != 91 || data.ThresholdValue == nil || *data.ThresholdValue != 80 {
				t.Fatalf("snapshot was not authoritative: %#v", data)
			}
		})
	}
}

func TestBuildAlertTemplateDataLegacyMetricAlertOmitsExactValues(t *testing.T) {
	data := presentationService().buildAlertTemplateData(context.Background(), generated.AlertRule{
		RuleType: types.AlertRuleTypeCPUAboveThreshold, Severity: types.AlertSeverityWarning,
		MetricName:     sql.NullString{String: types.MetricNameCPUUsage, Valid: true},
		ThresholdValue: sql.NullFloat64{Float64: 80, Valid: true},
	}, generated.AlertInstance{
		ProjectID: 7, NodeID: sql.NullInt64{Int64: 31, Valid: true}, Status: types.AlertStatusActive, TriggeredAt: time.Now(),
	})
	if data.MetricValue != nil || data.ThresholdValue != nil {
		t.Fatalf("legacy alert fabricated exact metric values: %#v", data)
	}
	if data.AlertMessage != "CPU usage exceeded the configured threshold." {
		t.Fatalf("legacy alert did not use neutral fallback copy: %q", data.AlertMessage)
	}
}

func TestBuildAlertTemplateDataKeepsMetricSnapshotWhenNodeEnrichmentFails(t *testing.T) {
	service := presentationService()
	service.nodeRepo = &fakeNotificationNodeRepo{nodes: map[int64]generated.Node{}}
	alert := testMetricTriggerSnapshot(t, types.MetricNameCPUUsage, 91, 80)
	alert.ProjectID = 7
	alert.NodeID = sql.NullInt64{Int64: 404, Valid: true}
	alert.Status = types.AlertStatusActive
	alert.TriggeredAt = time.Now()

	data := service.buildAlertTemplateData(context.Background(), generated.AlertRule{
		RuleType: types.AlertRuleTypeCPUAboveThreshold, Severity: types.AlertSeverityWarning,
	}, alert)
	if data.MetricValue == nil || *data.MetricValue != 91 || data.ThresholdValue == nil || *data.ThresholdValue != 80 {
		t.Fatalf("optional node lookup discarded trigger snapshot: %#v", data)
	}
	if data.ResourceURL != "https://app.agrafa.test/nodes/404?project_id=7" {
		t.Fatalf("resource URL should remain available without enrichment: %q", data.ResourceURL)
	}
}

func TestResolvedMetricEmailDoesNotMutateTriggerSnapshot(t *testing.T) {
	alert := testMetricTriggerSnapshot(t, types.MetricNameCPUUsage, 91, 80)
	original := string(alert.TriggerSnapshot.RawMessage)
	alert.ProjectID = 7
	alert.NodeID = sql.NullInt64{Int64: 31, Valid: true}
	alert.Status = types.AlertStatusResolved
	alert.TriggeredAt = time.Now().Add(-time.Minute)
	alert.ResolvedAt = sql.NullTime{Time: time.Now(), Valid: true}
	_ = presentationService().buildAlertTemplateData(context.Background(), generated.AlertRule{RuleType: types.AlertRuleTypeCPUAboveThreshold, Severity: types.AlertSeverityWarning}, alert)
	if string(alert.TriggerSnapshot.RawMessage) != original {
		t.Fatal("resolved notification mutated the trigger snapshot")
	}
}

func TestFormatAlertDuration(t *testing.T) {
	tests := map[time.Duration]string{
		34 * time.Second:               "34s",
		8*time.Minute + 12*time.Second: "8m 12s",
		time.Hour + 4*time.Minute:      "1h 4m",
		9*time.Hour + 34*time.Minute:   "9h 34m",
		51 * time.Hour:                 "2d 3h",
	}
	for duration, want := range tests {
		if got := formatAlertDuration(duration); got != want {
			t.Errorf("formatAlertDuration(%s) = %q, want %q", duration, got, want)
		}
	}
}
