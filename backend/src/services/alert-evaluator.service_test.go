package services

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeAlertRuleRepo struct {
	rules []generated.AlertRule
}

func (r *fakeAlertRuleRepo) ListEnabled(_ context.Context, ruleType string, nodeID *int64, serviceID *int64, metricName *string) ([]generated.AlertRule, error) {
	result := make([]generated.AlertRule, 0, len(r.rules))

	for _, rule := range r.rules {
		if rule.RuleType != ruleType || !rule.IsEnabled {
			continue
		}

		if nodeID != nil && (!rule.NodeID.Valid || rule.NodeID.Int64 != *nodeID) {
			continue
		}

		if serviceID != nil && (!rule.ServiceID.Valid || rule.ServiceID.Int64 != *serviceID) {
			continue
		}

		if metricName != nil && (!rule.MetricName.Valid || rule.MetricName.String != *metricName) {
			continue
		}

		result = append(result, rule)
	}

	return result, nil
}

type fakeAlertInstanceRepo struct {
	nextID       int64
	instances    []generated.AlertInstance
	createCalls  int
	resolveCalls int
}

func (r *fakeAlertInstanceRepo) FindActiveByRuleID(_ context.Context, ruleID int64) (generated.AlertInstance, error) {
	for _, instance := range r.instances {
		if instance.AlertRuleID == ruleID && instance.Status == types.AlertStatusActive {
			return instance, nil
		}
	}

	return generated.AlertInstance{}, sql.ErrNoRows
}

func (r *fakeAlertInstanceRepo) Create(_ context.Context, params generated.CreateAlertInstanceParams) (generated.AlertInstance, error) {
	r.createCalls++
	r.nextID++

	instance := generated.AlertInstance{
		ID:          r.nextID,
		AlertRuleID: params.AlertRuleID,
		ProjectID:   params.ProjectID,
		NodeID:      params.NodeID,
		ServiceID:   params.ServiceID,
		Status:      params.Status,
		TriggeredAt: params.TriggeredAt,
		ResolvedAt:  params.ResolvedAt,
		Title:       params.Title,
		Message:     params.Message,
		CreatedAt:   params.TriggeredAt,
	}

	r.instances = append(r.instances, instance)
	return instance, nil
}

func (r *fakeAlertInstanceRepo) Resolve(_ context.Context, id int64, resolvedAt time.Time) (generated.AlertInstance, error) {
	for index := range r.instances {
		if r.instances[index].ID == id && r.instances[index].Status == types.AlertStatusActive {
			r.resolveCalls++
			r.instances[index].Status = types.AlertStatusResolved
			r.instances[index].ResolvedAt = sql.NullTime{Time: resolvedAt, Valid: true}
			return r.instances[index], nil
		}
	}

	return generated.AlertInstance{}, sql.ErrNoRows
}

type fakeAlertMetricRepo struct {
	samples map[string]generated.MetricSample
}

func (r *fakeAlertMetricRepo) GetLatestNodeMetricByName(_ context.Context, nodeID int64, metricName string) (generated.MetricSample, error) {
	sample, ok := r.samples[metricKey(nodeID, metricName)]
	if !ok {
		return generated.MetricSample{}, sql.ErrNoRows
	}

	return sample, nil
}

type fakeAlertEventRecorder struct {
	triggered []generated.AlertInstance
	resolved  []generated.AlertInstance
}

func (r *fakeAlertEventRecorder) CreateAlertTriggered(_ context.Context, _ generated.AlertRule, alert generated.AlertInstance, _ time.Time) error {
	r.triggered = append(r.triggered, alert)
	return nil
}

func (r *fakeAlertEventRecorder) CreateAlertResolved(_ context.Context, _ generated.AlertRule, alert generated.AlertInstance, _ time.Time) error {
	r.resolved = append(r.resolved, alert)
	return nil
}

type fakeAlertNotificationService struct {
	triggeredErr   error
	resolvedErr    error
	triggeredCalls []generated.AlertInstance
	resolvedCalls  []generated.AlertInstance
}

func (s *fakeAlertNotificationService) NotifyAlertTriggered(_ context.Context, _ generated.AlertRule, alert generated.AlertInstance) error {
	s.triggeredCalls = append(s.triggeredCalls, alert)
	return s.triggeredErr
}

func (s *fakeAlertNotificationService) NotifyAlertResolved(_ context.Context, _ generated.AlertRule, alert generated.AlertInstance) error {
	s.resolvedCalls = append(s.resolvedCalls, alert)
	return s.resolvedErr
}

func TestEvaluateNodeRulesActivatesOnceAndDoesNotDuplicateActiveAlert(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	rule := generated.AlertRule{
		ID:        1,
		ProjectID: 1,
		NodeID:    sql.NullInt64{Int64: 10, Valid: true},
		RuleType:  types.AlertRuleTypeNodeOffline,
		IsEnabled: true,
	}

	instanceRepo := &fakeAlertInstanceRepo{}
	events := &fakeAlertEventRecorder{}
	service := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        &fakeAlertMetricRepo{},
		eventService:      events,
	}

	node := generated.Node{ID: 10, CurrentState: types.NodeStateOffline}

	if err := service.EvaluateNodeRules(context.Background(), node, occurredAt); err != nil {
		t.Fatalf("first EvaluateNodeRules returned error: %v", err)
	}

	if err := service.EvaluateNodeRules(context.Background(), node, occurredAt.Add(time.Minute)); err != nil {
		t.Fatalf("second EvaluateNodeRules returned error: %v", err)
	}

	if instanceRepo.createCalls != 1 {
		t.Fatalf("expected 1 alert creation, got %d", instanceRepo.createCalls)
	}

	if len(events.triggered) != 1 {
		t.Fatalf("expected 1 triggered event, got %d", len(events.triggered))
	}

	active, err := instanceRepo.FindActiveByRuleID(context.Background(), rule.ID)
	if err != nil {
		t.Fatalf("expected active alert, got error: %v", err)
	}

	if active.Status != types.AlertStatusActive {
		t.Fatalf("expected active status, got %q", active.Status)
	}
}

func TestEvaluateNodeRulesResolvesOnRecovery(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	rule := generated.AlertRule{
		ID:        2,
		ProjectID: 1,
		NodeID:    sql.NullInt64{Int64: 11, Valid: true},
		RuleType:  types.AlertRuleTypeNodeOffline,
		IsEnabled: true,
	}

	instanceRepo := &fakeAlertInstanceRepo{}
	events := &fakeAlertEventRecorder{}
	service := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        &fakeAlertMetricRepo{},
		eventService:      events,
	}

	if err := service.EvaluateNodeRules(context.Background(), generated.Node{ID: 11, CurrentState: types.NodeStateOffline}, occurredAt); err != nil {
		t.Fatalf("offline EvaluateNodeRules returned error: %v", err)
	}

	if err := service.EvaluateNodeRules(context.Background(), generated.Node{ID: 11, CurrentState: types.NodeStateOnline}, occurredAt.Add(2*time.Minute)); err != nil {
		t.Fatalf("online EvaluateNodeRules returned error: %v", err)
	}

	if instanceRepo.resolveCalls != 1 {
		t.Fatalf("expected 1 alert resolution, got %d", instanceRepo.resolveCalls)
	}

	if len(events.resolved) != 1 {
		t.Fatalf("expected 1 resolved event, got %d", len(events.resolved))
	}

	if _, err := instanceRepo.FindActiveByRuleID(context.Background(), rule.ID); err == nil {
		t.Fatal("expected no active alert after recovery")
	}
}

func TestEvaluateServiceRulesActivatesAndResolvesOnRecovery(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	rule := generated.AlertRule{
		ID:        3,
		ProjectID: 1,
		ServiceID: sql.NullInt64{Int64: 21, Valid: true},
		RuleType:  types.AlertRuleTypeServiceUnhealthy,
		IsEnabled: true,
	}

	instanceRepo := &fakeAlertInstanceRepo{}
	service := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        &fakeAlertMetricRepo{},
		eventService:      &fakeAlertEventRecorder{},
	}

	if err := service.EvaluateServiceRules(context.Background(), generated.Service{ID: 21, CurrentState: types.ServiceStateUnhealthy}, occurredAt); err != nil {
		t.Fatalf("unhealthy EvaluateServiceRules returned error: %v", err)
	}

	if err := service.EvaluateServiceRules(context.Background(), generated.Service{ID: 21, CurrentState: types.ServiceStateHealthy}, occurredAt.Add(time.Minute)); err != nil {
		t.Fatalf("healthy EvaluateServiceRules returned error: %v", err)
	}

	if instanceRepo.createCalls != 1 {
		t.Fatalf("expected 1 alert creation, got %d", instanceRepo.createCalls)
	}

	if instanceRepo.resolveCalls != 1 {
		t.Fatalf("expected 1 alert resolution, got %d", instanceRepo.resolveCalls)
	}
}

func TestEvaluateNodeRulesIgnoresNotificationFailures(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	rule := generated.AlertRule{
		ID:        30,
		ProjectID: 1,
		NodeID:    sql.NullInt64{Int64: 10, Valid: true},
		RuleType:  types.AlertRuleTypeNodeOffline,
		IsEnabled: true,
	}

	instanceRepo := &fakeAlertInstanceRepo{}
	service := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        &fakeAlertMetricRepo{},
		eventService:      &fakeAlertEventRecorder{},
		notificationService: &fakeAlertNotificationService{
			triggeredErr: errors.New("notification delivery failed"),
		},
	}

	err := service.EvaluateNodeRules(context.Background(), generated.Node{
		ID:           10,
		CurrentState: types.NodeStateOffline,
	}, occurredAt)
	if err != nil {
		t.Fatalf("EvaluateNodeRules returned error: %v", err)
	}

	if instanceRepo.createCalls != 1 {
		t.Fatalf("expected alert creation to still happen, got %d creates", instanceRepo.createCalls)
	}
}

func TestEvaluateServiceRulesOnlyNotifiesOnLifecycleTransitions(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	rule := generated.AlertRule{
		ID:        31,
		ProjectID: 1,
		ServiceID: sql.NullInt64{Int64: 21, Valid: true},
		RuleType:  types.AlertRuleTypeServiceUnhealthy,
		IsEnabled: true,
	}

	instanceRepo := &fakeAlertInstanceRepo{}
	notifications := &fakeAlertNotificationService{}
	service := &AlertEvaluatorService{
		alertRuleRepo:       &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo:   instanceRepo,
		metricRepo:          &fakeAlertMetricRepo{},
		eventService:        &fakeAlertEventRecorder{},
		notificationService: notifications,
	}

	if err := service.EvaluateServiceRules(context.Background(), generated.Service{ID: 21, CurrentState: types.ServiceStateUnhealthy}, occurredAt); err != nil {
		t.Fatalf("first unhealthy EvaluateServiceRules returned error: %v", err)
	}

	if err := service.EvaluateServiceRules(context.Background(), generated.Service{ID: 21, CurrentState: types.ServiceStateUnhealthy}, occurredAt.Add(time.Minute)); err != nil {
		t.Fatalf("second unhealthy EvaluateServiceRules returned error: %v", err)
	}

	if len(notifications.triggeredCalls) != 1 {
		t.Fatalf("expected 1 triggered notification while alert stays active, got %d", len(notifications.triggeredCalls))
	}

	if len(notifications.resolvedCalls) != 0 {
		t.Fatalf("expected 0 resolved notifications before recovery, got %d", len(notifications.resolvedCalls))
	}

	if err := service.EvaluateServiceRules(context.Background(), generated.Service{ID: 21, CurrentState: types.ServiceStateHealthy}, occurredAt.Add(2*time.Minute)); err != nil {
		t.Fatalf("healthy EvaluateServiceRules returned error: %v", err)
	}

	if len(notifications.resolvedCalls) != 1 {
		t.Fatalf("expected 1 resolved notification on recovery, got %d", len(notifications.resolvedCalls))
	}

	if err := service.EvaluateServiceRules(context.Background(), generated.Service{ID: 21, CurrentState: types.ServiceStateUnhealthy}, occurredAt.Add(3*time.Minute)); err != nil {
		t.Fatalf("re-trigger unhealthy EvaluateServiceRules returned error: %v", err)
	}

	if len(notifications.triggeredCalls) != 2 {
		t.Fatalf("expected 2 triggered notifications after re-trigger, got %d", len(notifications.triggeredCalls))
	}
}

func TestEvaluateMetricRulesActivatesAndResolvesThresholdAlert(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	rule := generated.AlertRule{
		ID:             4,
		ProjectID:      1,
		NodeID:         sql.NullInt64{Int64: 31, Valid: true},
		RuleType:       types.AlertRuleTypeCPUAboveThreshold,
		MetricName:     sql.NullString{String: types.MetricNameCPUUsage, Valid: true},
		ThresholdValue: sql.NullFloat64{Float64: 80, Valid: true},
		IsEnabled:      true,
	}

	metricRepo := &fakeAlertMetricRepo{
		samples: map[string]generated.MetricSample{
			metricKey(31, types.MetricNameCPUUsage): {
				NodeID:      31,
				MetricName:  types.MetricNameCPUUsage,
				MetricValue: 91,
			},
		},
	}
	instanceRepo := &fakeAlertInstanceRepo{}
	service := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        metricRepo,
		eventService:      &fakeAlertEventRecorder{},
	}

	if err := service.EvaluateMetricRules(context.Background(), 31, types.MetricNameCPUUsage, occurredAt); err != nil {
		t.Fatalf("high metric EvaluateMetricRules returned error: %v", err)
	}

	active, err := instanceRepo.FindActiveByRuleID(context.Background(), rule.ID)
	if err != nil {
		t.Fatalf("expected active threshold alert, got error: %v", err)
	}

	if active.Title != "Node 31 CPU usage is above 80" {
		t.Fatalf("unexpected alert title %q", active.Title)
	}

	metricRepo.samples[metricKey(31, types.MetricNameCPUUsage)] = generated.MetricSample{
		NodeID:      31,
		MetricName:  types.MetricNameCPUUsage,
		MetricValue: 75,
	}

	if err := service.EvaluateMetricRules(context.Background(), 31, types.MetricNameCPUUsage, occurredAt.Add(time.Minute)); err != nil {
		t.Fatalf("recovered metric EvaluateMetricRules returned error: %v", err)
	}

	if instanceRepo.resolveCalls != 1 {
		t.Fatalf("expected 1 threshold alert resolution, got %d", instanceRepo.resolveCalls)
	}
}

func TestEvaluateMetricRulesDoesNotActivateWithoutMetricSample(t *testing.T) {
	t.Parallel()

	rule := generated.AlertRule{
		ID:             5,
		ProjectID:      1,
		NodeID:         sql.NullInt64{Int64: 41, Valid: true},
		RuleType:       types.AlertRuleTypeMemoryAboveThreshold,
		MetricName:     sql.NullString{String: types.MetricNameMemoryUsage, Valid: true},
		ThresholdValue: sql.NullFloat64{Float64: 85, Valid: true},
		IsEnabled:      true,
	}

	instanceRepo := &fakeAlertInstanceRepo{}
	service := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        &fakeAlertMetricRepo{samples: map[string]generated.MetricSample{}},
		eventService:      &fakeAlertEventRecorder{},
	}

	if err := service.EvaluateMetricRules(context.Background(), 41, types.MetricNameMemoryUsage, time.Now().UTC()); err != nil {
		t.Fatalf("EvaluateMetricRules returned error without sample: %v", err)
	}

	if instanceRepo.createCalls != 0 {
		t.Fatalf("expected no alert creation without metric sample, got %d", instanceRepo.createCalls)
	}
}

func TestEvaluateNodeRulesIgnoresDisabledRule(t *testing.T) {
	t.Parallel()

	rule := generated.AlertRule{
		ID:        6,
		ProjectID: 1,
		NodeID:    sql.NullInt64{Int64: 51, Valid: true},
		RuleType:  types.AlertRuleTypeNodeOffline,
		IsEnabled: false,
	}

	instanceRepo := &fakeAlertInstanceRepo{}
	service := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        &fakeAlertMetricRepo{},
		eventService:      &fakeAlertEventRecorder{},
	}

	if err := service.EvaluateNodeRules(context.Background(), generated.Node{ID: 51, CurrentState: types.NodeStateOffline}, time.Now().UTC()); err != nil {
		t.Fatalf("EvaluateNodeRules returned error for disabled rule: %v", err)
	}

	if instanceRepo.createCalls != 0 {
		t.Fatalf("expected disabled rule to be ignored, got %d creations", instanceRepo.createCalls)
	}
}

func metricKey(nodeID int64, metricName string) string {
	return strconv.FormatInt(nodeID, 10) + ":" + metricName
}
