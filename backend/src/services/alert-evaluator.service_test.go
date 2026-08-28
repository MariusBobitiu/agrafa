package services

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeAlertRuleRepo struct {
	rules []generated.AlertRule
}

func (r *fakeAlertRuleRepo) ListEnabled(_ context.Context, projectID int64, ruleType string, nodeID *int64, serviceID *int64, metricName *string) ([]generated.AlertRule, error) {
	result := make([]generated.AlertRule, 0, len(r.rules))

	for _, rule := range r.rules {
		if rule.ProjectID != projectID || rule.RuleType != ruleType || !rule.IsEnabled {
			continue
		}

		if nodeID != nil && rule.NodeID.Valid && rule.NodeID.Int64 != *nodeID {
			continue
		}

		if serviceID != nil && rule.ServiceID.Valid && rule.ServiceID.Int64 != *serviceID {
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

func (r *fakeAlertInstanceRepo) FindActiveByRuleAndTarget(_ context.Context, ruleID int64, nodeID sql.NullInt64, serviceID sql.NullInt64) (generated.AlertInstance, error) {
	for _, instance := range r.instances {
		if instance.AlertRuleID == ruleID && instance.Status == types.AlertStatusActive &&
			nullInt64Equal(instance.NodeID, nodeID) && nullInt64Equal(instance.ServiceID, serviceID) {
			return instance, nil
		}
	}

	return generated.AlertInstance{}, sql.ErrNoRows
}

func (r *fakeAlertInstanceRepo) FindActiveByRuleID(_ context.Context, ruleID int64) (generated.AlertInstance, error) {
	for _, instance := range r.instances {
		if instance.AlertRuleID == ruleID && instance.Status == types.AlertStatusActive {
			return instance, nil
		}
	}
	return generated.AlertInstance{}, sql.ErrNoRows
}

func (r *fakeAlertInstanceRepo) ListActiveByRuleID(_ context.Context, ruleID int64) ([]generated.AlertInstance, error) {
	result := make([]generated.AlertInstance, 0)
	for _, instance := range r.instances {
		if instance.AlertRuleID == ruleID && instance.Status == types.AlertStatusActive {
			result = append(result, instance)
		}
	}
	return result, nil
}

func nullInt64Equal(left, right sql.NullInt64) bool {
	return left.Valid == right.Valid && (!left.Valid || left.Int64 == right.Int64)
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

type fakeAlertEvaluatorNodeRepo struct {
	nodes map[int64]generated.Node
}

func (r *fakeAlertEvaluatorNodeRepo) GetByID(_ context.Context, id int64) (generated.Node, error) {
	node, ok := r.nodes[id]
	if !ok {
		return generated.Node{}, sql.ErrNoRows
	}
	return node, nil
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

	node := generated.Node{ID: 10, ProjectID: 1, CurrentState: types.NodeStateOffline}

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

	if err := service.EvaluateNodeRules(context.Background(), generated.Node{ID: 11, ProjectID: 1, CurrentState: types.NodeStateOffline}, occurredAt); err != nil {
		t.Fatalf("offline EvaluateNodeRules returned error: %v", err)
	}

	if err := service.EvaluateNodeRules(context.Background(), generated.Node{ID: 11, ProjectID: 1, CurrentState: types.NodeStateOnline}, occurredAt.Add(2*time.Minute)); err != nil {
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

	if err := service.EvaluateServiceRules(context.Background(), generated.Service{ID: 21, ProjectID: 1, CurrentState: types.ServiceStateUnhealthy}, occurredAt); err != nil {
		t.Fatalf("unhealthy EvaluateServiceRules returned error: %v", err)
	}

	if err := service.EvaluateServiceRules(context.Background(), generated.Service{ID: 21, ProjectID: 1, CurrentState: types.ServiceStateHealthy}, occurredAt.Add(time.Minute)); err != nil {
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
		ProjectID:    1,
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

	if err := service.EvaluateServiceRules(context.Background(), generated.Service{ID: 21, ProjectID: 1, CurrentState: types.ServiceStateUnhealthy}, occurredAt); err != nil {
		t.Fatalf("first unhealthy EvaluateServiceRules returned error: %v", err)
	}

	if err := service.EvaluateServiceRules(context.Background(), generated.Service{ID: 21, ProjectID: 1, CurrentState: types.ServiceStateUnhealthy}, occurredAt.Add(time.Minute)); err != nil {
		t.Fatalf("second unhealthy EvaluateServiceRules returned error: %v", err)
	}

	if len(notifications.triggeredCalls) != 1 {
		t.Fatalf("expected 1 triggered notification while alert stays active, got %d", len(notifications.triggeredCalls))
	}

	if len(notifications.resolvedCalls) != 0 {
		t.Fatalf("expected 0 resolved notifications before recovery, got %d", len(notifications.resolvedCalls))
	}

	if err := service.EvaluateServiceRules(context.Background(), generated.Service{ID: 21, ProjectID: 1, CurrentState: types.ServiceStateHealthy}, occurredAt.Add(2*time.Minute)); err != nil {
		t.Fatalf("healthy EvaluateServiceRules returned error: %v", err)
	}

	if len(notifications.resolvedCalls) != 1 {
		t.Fatalf("expected 1 resolved notification on recovery, got %d", len(notifications.resolvedCalls))
	}

	if err := service.EvaluateServiceRules(context.Background(), generated.Service{ID: 21, ProjectID: 1, CurrentState: types.ServiceStateUnhealthy}, occurredAt.Add(3*time.Minute)); err != nil {
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
		nodeRepo:          &fakeAlertEvaluatorNodeRepo{nodes: map[int64]generated.Node{31: {ID: 31, ProjectID: 1}}},
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
		nodeRepo:          &fakeAlertEvaluatorNodeRepo{nodes: map[int64]generated.Node{41: {ID: 41, ProjectID: 1}}},
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

	if err := service.EvaluateNodeRules(context.Background(), generated.Node{ID: 51, ProjectID: 1, CurrentState: types.NodeStateOffline}, time.Now().UTC()); err != nil {
		t.Fatalf("EvaluateNodeRules returned error for disabled rule: %v", err)
	}

	if instanceRepo.createCalls != 0 {
		t.Fatalf("expected disabled rule to be ignored, got %d creations", instanceRepo.createCalls)
	}
}

func TestGlobalServiceRuleMaintainsIndependentTargetLifecycles(t *testing.T) {
	t.Parallel()

	rule := generated.AlertRule{
		ID: 70, ProjectID: 1, RuleType: types.AlertRuleTypeServiceUnhealthy, IsEnabled: true,
	}
	instanceRepo := &fakeAlertInstanceRepo{}
	evaluator := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        &fakeAlertMetricRepo{},
		eventService:      &fakeAlertEventRecorder{},
	}
	startedAt := time.Date(2026, time.August, 27, 10, 0, 0, 0, time.UTC)

	for _, serviceID := range []int64{21, 22} {
		err := evaluator.EvaluateServiceRules(context.Background(), generated.Service{
			ID: serviceID, ProjectID: 1, CurrentState: types.ServiceStateUnhealthy,
		}, startedAt)
		if err != nil {
			t.Fatalf("evaluate service %d: %v", serviceID, err)
		}
	}

	if len(instanceRepo.instances) != 2 {
		t.Fatalf("created instances = %d, want 2", len(instanceRepo.instances))
	}
	for _, instance := range instanceRepo.instances {
		if !instance.ServiceID.Valid || instance.NodeID.Valid {
			t.Fatalf("instance target = node %#v service %#v, want concrete service", instance.NodeID, instance.ServiceID)
		}
		wantTitle := "Service " + strconv.FormatInt(instance.ServiceID.Int64, 10) + " is unhealthy"
		if instance.Title != wantTitle {
			t.Fatalf("instance title = %q, want %q", instance.Title, wantTitle)
		}
	}

	if err := evaluator.EvaluateServiceRules(context.Background(), generated.Service{
		ID: 21, ProjectID: 1, CurrentState: types.ServiceStateHealthy,
	}, startedAt.Add(time.Minute)); err != nil {
		t.Fatalf("recover service 21: %v", err)
	}

	if _, err := instanceRepo.FindActiveByRuleAndTarget(context.Background(), rule.ID, sql.NullInt64{}, sql.NullInt64{Int64: 21, Valid: true}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("service 21 active lookup error = %v, want sql.ErrNoRows", err)
	}
	if _, err := instanceRepo.FindActiveByRuleAndTarget(context.Background(), rule.ID, sql.NullInt64{}, sql.NullInt64{Int64: 22, Valid: true}); err != nil {
		t.Fatalf("service 22 should remain active: %v", err)
	}
	if instanceRepo.resolveCalls != 1 {
		t.Fatalf("resolve calls = %d, want 1", instanceRepo.resolveCalls)
	}
}

func TestGlobalServiceRuleMatchesServiceCreatedAfterRule(t *testing.T) {
	t.Parallel()

	rule := generated.AlertRule{
		ID: 75, ProjectID: 1, RuleType: types.AlertRuleTypeServiceUnhealthy,
		IsEnabled: true, CreatedAt: time.Now().UTC().Add(-time.Hour),
	}
	instanceRepo := &fakeAlertInstanceRepo{}
	evaluator := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        &fakeAlertMetricRepo{},
		eventService:      &fakeAlertEventRecorder{},
	}
	newService := generated.Service{
		ID: 29, ProjectID: 1, CurrentState: types.ServiceStateUnhealthy,
		CreatedAt: time.Now().UTC(),
	}

	if err := evaluator.EvaluateServiceRules(context.Background(), newService, time.Now().UTC()); err != nil {
		t.Fatalf("EvaluateServiceRules() error = %v", err)
	}
	if _, err := instanceRepo.FindActiveByRuleAndTarget(
		context.Background(), rule.ID, sql.NullInt64{}, sql.NullInt64{Int64: newService.ID, Valid: true},
	); err != nil {
		t.Fatalf("new service was not covered by global rule: %v", err)
	}
}

func TestGlobalAndSpecificServiceRulesCoexistAndStayProjectScoped(t *testing.T) {
	t.Parallel()

	rules := []generated.AlertRule{
		{ID: 80, ProjectID: 1, RuleType: types.AlertRuleTypeServiceUnhealthy, IsEnabled: true},
		{ID: 81, ProjectID: 1, ServiceID: sql.NullInt64{Int64: 31, Valid: true}, RuleType: types.AlertRuleTypeServiceUnhealthy, IsEnabled: true},
		{ID: 82, ProjectID: 2, RuleType: types.AlertRuleTypeServiceUnhealthy, IsEnabled: true},
	}
	instanceRepo := &fakeAlertInstanceRepo{}
	evaluator := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: rules},
		alertInstanceRepo: instanceRepo,
		metricRepo:        &fakeAlertMetricRepo{},
		eventService:      &fakeAlertEventRecorder{},
	}

	for _, serviceID := range []int64{31, 32} {
		if err := evaluator.EvaluateServiceRules(context.Background(), generated.Service{
			ID: serviceID, ProjectID: 1, CurrentState: types.ServiceStateUnhealthy,
		}, time.Now().UTC()); err != nil {
			t.Fatalf("evaluate service %d: %v", serviceID, err)
		}
	}

	if len(instanceRepo.instances) != 3 {
		t.Fatalf("created instances = %d, want global+specific for 31 and global for 32", len(instanceRepo.instances))
	}
	if _, err := instanceRepo.FindActiveByRuleAndTarget(context.Background(), 81, sql.NullInt64{}, sql.NullInt64{Int64: 32, Valid: true}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("specific rule matched service 32: %v", err)
	}
	for _, instance := range instanceRepo.instances {
		if instance.AlertRuleID == 82 {
			t.Fatal("rule from project 2 matched a project 1 service")
		}
	}
}

func TestGlobalNodeOfflineRuleCoversMultipleNodes(t *testing.T) {
	t.Parallel()

	rule := generated.AlertRule{ID: 90, ProjectID: 1, RuleType: types.AlertRuleTypeNodeOffline, IsEnabled: true}
	instanceRepo := &fakeAlertInstanceRepo{}
	evaluator := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        &fakeAlertMetricRepo{},
		eventService:      &fakeAlertEventRecorder{},
	}

	for _, nodeID := range []int64{41, 42} {
		if err := evaluator.EvaluateNodeRules(context.Background(), generated.Node{
			ID: nodeID, ProjectID: 1, IsVisible: true, CurrentState: types.NodeStateOffline,
		}, time.Now().UTC()); err != nil {
			t.Fatalf("evaluate node %d: %v", nodeID, err)
		}
	}

	if len(instanceRepo.instances) != 2 {
		t.Fatalf("created instances = %d, want 2", len(instanceRepo.instances))
	}
	for _, instance := range instanceRepo.instances {
		if !instance.NodeID.Valid || instance.ServiceID.Valid {
			t.Fatalf("instance target = node %#v service %#v, want concrete node", instance.NodeID, instance.ServiceID)
		}
	}
}

func TestGlobalMetricRuleCoversMultipleNodes(t *testing.T) {
	t.Parallel()

	rule := generated.AlertRule{
		ID: 100, ProjectID: 1, RuleType: types.AlertRuleTypeCPUAboveThreshold,
		MetricName:     sql.NullString{String: types.MetricNameCPUUsage, Valid: true},
		ThresholdValue: sql.NullFloat64{Float64: 80, Valid: true}, IsEnabled: true,
	}
	metricRepo := &fakeAlertMetricRepo{samples: map[string]generated.MetricSample{
		metricKey(51, types.MetricNameCPUUsage): {NodeID: 51, MetricName: types.MetricNameCPUUsage, MetricValue: 91, ObservedAt: time.Now().UTC()},
		metricKey(52, types.MetricNameCPUUsage): {NodeID: 52, MetricName: types.MetricNameCPUUsage, MetricValue: 92, ObservedAt: time.Now().UTC()},
	}}
	instanceRepo := &fakeAlertInstanceRepo{}
	evaluator := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        metricRepo,
		nodeRepo: &fakeAlertEvaluatorNodeRepo{nodes: map[int64]generated.Node{
			51: {ID: 51, ProjectID: 1, IsVisible: true}, 52: {ID: 52, ProjectID: 1, IsVisible: true},
		}},
		eventService: &fakeAlertEventRecorder{},
	}

	for _, nodeID := range []int64{51, 52} {
		if err := evaluator.EvaluateMetricRules(context.Background(), nodeID, types.MetricNameCPUUsage, time.Now().UTC()); err != nil {
			t.Fatalf("evaluate metric for node %d: %v", nodeID, err)
		}
	}

	if len(instanceRepo.instances) != 2 {
		t.Fatalf("created instances = %d, want 2", len(instanceRepo.instances))
	}
	for _, instance := range instanceRepo.instances {
		if !instance.NodeID.Valid || instance.NodeID.Int64 == 0 {
			t.Fatalf("metric alert missing concrete node: %#v", instance)
		}
		if !strings.Contains(instance.Title, strconv.FormatInt(instance.NodeID.Int64, 10)) {
			t.Fatalf("metric alert title %q does not use evaluated node %d", instance.Title, instance.NodeID.Int64)
		}
	}
}

func TestHiddenNodeResolvesActiveGlobalNodeAlert(t *testing.T) {
	t.Parallel()

	rule := generated.AlertRule{
		ID: 110, ProjectID: 1, RuleType: types.AlertRuleTypeNodeOffline, IsEnabled: true,
	}
	instanceRepo := &fakeAlertInstanceRepo{}
	evaluator := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        &fakeAlertMetricRepo{},
		eventService:      &fakeAlertEventRecorder{},
	}
	triggeredAt := time.Date(2026, time.August, 28, 10, 0, 0, 0, time.UTC)
	resolvedAt := triggeredAt.Add(time.Minute)
	node := generated.Node{
		ID: 61, ProjectID: 1, IsVisible: true, CurrentState: types.NodeStateOffline,
	}

	if err := evaluator.EvaluateNodeRules(context.Background(), node, triggeredAt); err != nil {
		t.Fatalf("evaluate visible offline node: %v", err)
	}
	node.IsVisible = false
	if err := evaluator.EvaluateNodeRules(context.Background(), node, resolvedAt); err != nil {
		t.Fatalf("evaluate hidden node: %v", err)
	}

	if instanceRepo.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", instanceRepo.createCalls)
	}
	if instanceRepo.resolveCalls != 1 {
		t.Fatalf("resolve calls = %d, want 1", instanceRepo.resolveCalls)
	}
	if instanceRepo.instances[0].Status != types.AlertStatusResolved ||
		!instanceRepo.instances[0].ResolvedAt.Valid ||
		!instanceRepo.instances[0].ResolvedAt.Time.Equal(resolvedAt) {
		t.Fatalf("resolved instance = %#v, want resolution at %v", instanceRepo.instances[0], resolvedAt)
	}
}

func TestHiddenNodeResolvesActiveGlobalMetricAlert(t *testing.T) {
	t.Parallel()

	rule := generated.AlertRule{
		ID: 120, ProjectID: 1, RuleType: types.AlertRuleTypeCPUAboveThreshold,
		MetricName:     sql.NullString{String: types.MetricNameCPUUsage, Valid: true},
		ThresholdValue: sql.NullFloat64{Float64: 80, Valid: true}, IsEnabled: true,
	}
	observedAt := time.Date(2026, time.August, 28, 10, 0, 0, 0, time.UTC)
	resolvedAt := observedAt.Add(time.Minute)
	metricRepo := &fakeAlertMetricRepo{samples: map[string]generated.MetricSample{
		metricKey(71, types.MetricNameCPUUsage): {
			NodeID: 71, MetricName: types.MetricNameCPUUsage, MetricValue: 95, ObservedAt: observedAt,
		},
	}}
	nodes := map[int64]generated.Node{
		71: {ID: 71, ProjectID: 1, IsVisible: true},
	}
	instanceRepo := &fakeAlertInstanceRepo{}
	evaluator := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        metricRepo,
		nodeRepo:          &fakeAlertEvaluatorNodeRepo{nodes: nodes},
		eventService:      &fakeAlertEventRecorder{},
	}

	if err := evaluator.EvaluateMetricRules(context.Background(), 71, types.MetricNameCPUUsage, observedAt); err != nil {
		t.Fatalf("evaluate visible high CPU: %v", err)
	}
	node := nodes[71]
	node.IsVisible = false
	nodes[71] = node
	if err := evaluator.EvaluateMetricRules(context.Background(), 71, types.MetricNameCPUUsage, resolvedAt); err != nil {
		t.Fatalf("evaluate hidden node metric: %v", err)
	}

	if instanceRepo.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", instanceRepo.createCalls)
	}
	if instanceRepo.resolveCalls != 1 {
		t.Fatalf("resolve calls = %d, want 1", instanceRepo.resolveCalls)
	}
	if instanceRepo.instances[0].Status != types.AlertStatusResolved ||
		!instanceRepo.instances[0].ResolvedAt.Valid ||
		!instanceRepo.instances[0].ResolvedAt.Time.Equal(resolvedAt) {
		t.Fatalf("resolved instance = %#v, want resolution at %v", instanceRepo.instances[0], resolvedAt)
	}
}

func TestHiddenGlobalTargetWithoutActiveAlertIsNoOp(t *testing.T) {
	t.Parallel()

	rule := generated.AlertRule{
		ID: 130, ProjectID: 1, RuleType: types.AlertRuleTypeNodeOffline, IsEnabled: true,
	}
	instanceRepo := &fakeAlertInstanceRepo{}
	evaluator := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        &fakeAlertMetricRepo{},
		eventService:      &fakeAlertEventRecorder{},
	}

	err := evaluator.EvaluateNodeRules(context.Background(), generated.Node{
		ID: 81, ProjectID: 1, IsVisible: false, CurrentState: types.NodeStateOffline,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("EvaluateNodeRules() error = %v", err)
	}
	if instanceRepo.createCalls != 0 || instanceRepo.resolveCalls != 0 {
		t.Fatalf("lifecycle calls = create %d resolve %d, want no-op", instanceRepo.createCalls, instanceRepo.resolveCalls)
	}
}

func TestHiddenNodeResolutionIsTargetSpecific(t *testing.T) {
	t.Parallel()

	rule := generated.AlertRule{
		ID: 140, ProjectID: 1, RuleType: types.AlertRuleTypeNodeOffline, IsEnabled: true,
	}
	instanceRepo := &fakeAlertInstanceRepo{}
	evaluator := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: []generated.AlertRule{rule}},
		alertInstanceRepo: instanceRepo,
		metricRepo:        &fakeAlertMetricRepo{},
		eventService:      &fakeAlertEventRecorder{},
	}
	startedAt := time.Date(2026, time.August, 28, 10, 0, 0, 0, time.UTC)

	for _, nodeID := range []int64{91, 92} {
		err := evaluator.EvaluateNodeRules(context.Background(), generated.Node{
			ID: nodeID, ProjectID: 1, IsVisible: true, CurrentState: types.NodeStateOffline,
		}, startedAt)
		if err != nil {
			t.Fatalf("evaluate visible node %d: %v", nodeID, err)
		}
	}
	if err := evaluator.EvaluateNodeRules(context.Background(), generated.Node{
		ID: 91, ProjectID: 1, IsVisible: false, CurrentState: types.NodeStateOffline,
	}, startedAt.Add(time.Minute)); err != nil {
		t.Fatalf("evaluate hidden node 91: %v", err)
	}

	if _, err := instanceRepo.FindActiveByRuleAndTarget(
		context.Background(), rule.ID, sql.NullInt64{Int64: 91, Valid: true}, sql.NullInt64{},
	); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("node 91 active lookup error = %v, want sql.ErrNoRows", err)
	}
	if _, err := instanceRepo.FindActiveByRuleAndTarget(
		context.Background(), rule.ID, sql.NullInt64{Int64: 92, Valid: true}, sql.NullInt64{},
	); err != nil {
		t.Fatalf("node 92 should remain active: %v", err)
	}
	if instanceRepo.resolveCalls != 1 {
		t.Fatalf("resolve calls = %d, want 1", instanceRepo.resolveCalls)
	}
}

func TestHiddenNodeEventsSkipGlobalRulesButKeepSpecificRules(t *testing.T) {
	t.Parallel()

	nodeRules := []generated.AlertRule{
		{ID: 110, ProjectID: 1, RuleType: types.AlertRuleTypeNodeOffline, IsEnabled: true},
		{ID: 111, ProjectID: 1, NodeID: sql.NullInt64{Int64: 61, Valid: true}, RuleType: types.AlertRuleTypeNodeOffline, IsEnabled: true},
	}
	nodeInstances := &fakeAlertInstanceRepo{}
	nodeEvaluator := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: nodeRules},
		alertInstanceRepo: nodeInstances,
		metricRepo:        &fakeAlertMetricRepo{},
		eventService:      &fakeAlertEventRecorder{},
	}
	hiddenNode := generated.Node{
		ID: 61, ProjectID: 1, IsVisible: false, CurrentState: types.NodeStateOffline,
	}

	if err := nodeEvaluator.EvaluateNodeRules(context.Background(), hiddenNode, time.Now().UTC()); err != nil {
		t.Fatalf("EvaluateNodeRules() error = %v", err)
	}
	if len(nodeInstances.instances) != 1 || nodeInstances.instances[0].AlertRuleID != 111 {
		t.Fatalf("node instances = %#v, want only specific rule 111", nodeInstances.instances)
	}
	if nodeInstances.resolveCalls != 0 {
		t.Fatalf("node resolve calls = %d, want 0", nodeInstances.resolveCalls)
	}

	metricRules := []generated.AlertRule{
		{
			ID: 120, ProjectID: 1, RuleType: types.AlertRuleTypeCPUAboveThreshold,
			MetricName:     sql.NullString{String: types.MetricNameCPUUsage, Valid: true},
			ThresholdValue: sql.NullFloat64{Float64: 80, Valid: true}, IsEnabled: true,
		},
		{
			ID: 121, ProjectID: 1, NodeID: sql.NullInt64{Int64: 61, Valid: true},
			RuleType:       types.AlertRuleTypeCPUAboveThreshold,
			MetricName:     sql.NullString{String: types.MetricNameCPUUsage, Valid: true},
			ThresholdValue: sql.NullFloat64{Float64: 80, Valid: true}, IsEnabled: true,
		},
	}
	metricInstances := &fakeAlertInstanceRepo{}
	metricEvaluator := &AlertEvaluatorService{
		alertRuleRepo:     &fakeAlertRuleRepo{rules: metricRules},
		alertInstanceRepo: metricInstances,
		metricRepo: &fakeAlertMetricRepo{samples: map[string]generated.MetricSample{
			metricKey(61, types.MetricNameCPUUsage): {
				NodeID: 61, MetricName: types.MetricNameCPUUsage, MetricValue: 95, ObservedAt: time.Now().UTC(),
			},
		}},
		nodeRepo:     &fakeAlertEvaluatorNodeRepo{nodes: map[int64]generated.Node{61: hiddenNode}},
		eventService: &fakeAlertEventRecorder{},
	}

	if err := metricEvaluator.EvaluateMetricRules(context.Background(), 61, types.MetricNameCPUUsage, time.Now().UTC()); err != nil {
		t.Fatalf("EvaluateMetricRules() error = %v", err)
	}
	if len(metricInstances.instances) != 1 || metricInstances.instances[0].AlertRuleID != 121 {
		t.Fatalf("metric instances = %#v, want only specific rule 121", metricInstances.instances)
	}
	if metricInstances.resolveCalls != 0 {
		t.Fatalf("metric resolve calls = %d, want 0", metricInstances.resolveCalls)
	}
}

func metricKey(nodeID int64, metricName string) string {
	return strconv.FormatInt(nodeID, 10) + ":" + metricName
}
