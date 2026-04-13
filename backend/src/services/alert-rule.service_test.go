package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeAlertRuleServiceAlertRuleRepo struct {
	rule            generated.AlertRule
	getErr          error
	deleteRows      int64
	deleteErr       error
	lastCreateInput generated.CreateAlertRuleParams
	lastUpdateInput generated.UpdateAlertRuleParams
}

func (r *fakeAlertRuleServiceAlertRuleRepo) Create(_ context.Context, params generated.CreateAlertRuleParams) (generated.AlertRule, error) {
	r.lastCreateInput = params
	r.rule = generated.AlertRule{
		ID:             11,
		ProjectID:      params.ProjectID,
		NodeID:         params.NodeID,
		ServiceID:      params.ServiceID,
		RuleType:       params.RuleType,
		Severity:       params.Severity,
		MetricName:     params.MetricName,
		ThresholdValue: params.ThresholdValue,
		IsEnabled:      params.IsEnabled,
		CreatedAt:      time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
	}
	return r.rule, nil
}

func (r *fakeAlertRuleServiceAlertRuleRepo) GetByID(_ context.Context, _ int64) (generated.AlertRule, error) {
	return r.rule, r.getErr
}

func (r *fakeAlertRuleServiceAlertRuleRepo) Update(_ context.Context, params generated.UpdateAlertRuleParams) (generated.AlertRule, error) {
	r.lastUpdateInput = params
	if params.Column2 {
		r.rule.NodeID = params.NodeID
	}
	if params.Column4 {
		r.rule.ServiceID = params.ServiceID
	}
	if params.Column6 {
		r.rule.Severity = params.Severity
	}
	if params.Column8 {
		r.rule.ThresholdValue = params.ThresholdValue
	}
	if params.Column10 {
		r.rule.IsEnabled = params.IsEnabled
	}
	r.rule.UpdatedAt = time.Date(2026, time.April, 5, 12, 5, 0, 0, time.UTC)
	return r.rule, nil
}

func (r *fakeAlertRuleServiceAlertRuleRepo) List(_ context.Context, _ *int64) ([]generated.AlertRule, error) {
	return nil, nil
}

func (r *fakeAlertRuleServiceAlertRuleRepo) ListEnabled(_ context.Context, ruleType string, nodeID *int64, serviceID *int64, metricName *string) ([]generated.AlertRule, error) {
	if r.rule.ID == 0 || !r.rule.IsEnabled || r.rule.RuleType != ruleType {
		return nil, nil
	}

	if nodeID != nil && (!r.rule.NodeID.Valid || r.rule.NodeID.Int64 != *nodeID) {
		return nil, nil
	}

	if serviceID != nil && (!r.rule.ServiceID.Valid || r.rule.ServiceID.Int64 != *serviceID) {
		return nil, nil
	}

	if metricName != nil && (!r.rule.MetricName.Valid || r.rule.MetricName.String != *metricName) {
		return nil, nil
	}

	return []generated.AlertRule{r.rule}, nil
}

func (r *fakeAlertRuleServiceAlertRuleRepo) Delete(_ context.Context, _ int64) (int64, error) {
	return r.deleteRows, r.deleteErr
}

type fakeAlertRuleServiceProjectRepo struct {
	projects map[int64]generated.Project
}

func (r *fakeAlertRuleServiceProjectRepo) GetByID(_ context.Context, id int64) (generated.Project, error) {
	project, ok := r.projects[id]
	if !ok {
		return generated.Project{}, sql.ErrNoRows
	}

	return project, nil
}

type fakeAlertRuleServiceNodeRepo struct {
	nodes map[int64]generated.Node
}

func (r *fakeAlertRuleServiceNodeRepo) GetByID(_ context.Context, id int64) (generated.Node, error) {
	node, ok := r.nodes[id]
	if !ok {
		return generated.Node{}, sql.ErrNoRows
	}

	return node, nil
}

type fakeAlertRuleServiceServiceRepo struct {
	services map[int64]generated.Service
}

func (r *fakeAlertRuleServiceServiceRepo) GetByID(_ context.Context, id int64) (generated.Service, error) {
	service, ok := r.services[id]
	if !ok {
		return generated.Service{}, sql.ErrNoRows
	}

	return service, nil
}

func TestAlertRuleServiceCreatePersistsSeverity(t *testing.T) {
	t.Parallel()

	repo := &fakeAlertRuleServiceAlertRuleRepo{}
	service := &AlertRuleService{
		alertRuleRepo: repo,
		projectRepo: &fakeAlertRuleServiceProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1},
			},
		},
		nodeRepo: &fakeAlertRuleServiceNodeRepo{
			nodes: map[int64]generated.Node{
				2: {ID: 2, ProjectID: 1},
			},
		},
		serviceRepo: &fakeAlertRuleServiceServiceRepo{},
	}

	threshold := 80.0
	nodeID := int64(2)
	rule, err := service.Create(context.Background(), types.CreateAlertRuleInput{
		ProjectID:      1,
		NodeID:         &nodeID,
		RuleType:       types.AlertRuleTypeCPUAboveThreshold,
		Severity:       types.AlertSeverityCritical,
		ThresholdValue: &threshold,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if repo.lastCreateInput.Severity != types.AlertSeverityCritical {
		t.Fatalf("stored severity = %q, want %q", repo.lastCreateInput.Severity, types.AlertSeverityCritical)
	}
	if rule.Severity != types.AlertSeverityCritical {
		t.Fatalf("response severity = %q, want %q", rule.Severity, types.AlertSeverityCritical)
	}
}

func TestAlertRuleServiceCreateRejectsInvalidSeverity(t *testing.T) {
	t.Parallel()

	service := &AlertRuleService{
		alertRuleRepo: &fakeAlertRuleServiceAlertRuleRepo{},
		projectRepo: &fakeAlertRuleServiceProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1},
			},
		},
		nodeRepo: &fakeAlertRuleServiceNodeRepo{
			nodes: map[int64]generated.Node{
				2: {ID: 2, ProjectID: 1},
			},
		},
		serviceRepo: &fakeAlertRuleServiceServiceRepo{},
	}

	threshold := 80.0
	nodeID := int64(2)
	_, err := service.Create(context.Background(), types.CreateAlertRuleInput{
		ProjectID:      1,
		NodeID:         &nodeID,
		RuleType:       types.AlertRuleTypeCPUAboveThreshold,
		Severity:       "urgent",
		ThresholdValue: &threshold,
	})
	if !errors.Is(err, types.ErrInvalidAlertSeverity) {
		t.Fatalf("Create() error = %v, want ErrInvalidAlertSeverity", err)
	}
}

func TestAlertRuleServiceUpdatePersistsNodeThresholdSeverityAndEnabled(t *testing.T) {
	t.Parallel()

	repo := &fakeAlertRuleServiceAlertRuleRepo{
		rule: generated.AlertRule{
			ID:             11,
			ProjectID:      1,
			NodeID:         sql.NullInt64{Int64: 2, Valid: true},
			RuleType:       types.AlertRuleTypeCPUAboveThreshold,
			Severity:       types.AlertSeverityWarning,
			MetricName:     sql.NullString{String: types.MetricNameCPUUsage, Valid: true},
			ThresholdValue: sql.NullFloat64{Float64: 80, Valid: true},
			IsEnabled:      true,
			CreatedAt:      time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
			UpdatedAt:      time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
		},
	}
	metricRepo := &fakeAlertMetricRepo{
		samples: map[string]generated.MetricSample{
			metricKey(3, types.MetricNameCPUUsage): {
				NodeID:      3,
				MetricName:  types.MetricNameCPUUsage,
				MetricValue: 95,
				ObservedAt:  time.Date(2026, time.April, 5, 12, 4, 0, 0, time.UTC),
			},
		},
	}
	instanceRepo := &fakeAlertInstanceRepo{}
	evaluator := NewAlertEvaluatorService(repo, instanceRepo, metricRepo, &fakeAlertEventRecorder{}, nil)
	service := &AlertRuleService{
		alertRuleRepo: repo,
		projectRepo:   &fakeAlertRuleServiceProjectRepo{},
		nodeRepo: &fakeAlertRuleServiceNodeRepo{
			nodes: map[int64]generated.Node{
				2: {ID: 2, ProjectID: 1, CurrentState: types.NodeStateOnline},
				3: {ID: 3, ProjectID: 1, CurrentState: types.NodeStateOnline},
			},
		},
		serviceRepo: &fakeAlertRuleServiceServiceRepo{},
		evaluator:   evaluator,
	}

	nodeID := int64(3)
	threshold := 90.0
	severity := types.AlertSeverityCritical
	enabled := false

	rule, err := service.Update(context.Background(), types.UpdateAlertRuleInput{
		ID:             11,
		NodeID:         &nodeID,
		ThresholdValue: &threshold,
		Severity:       &severity,
		IsEnabled:      &enabled,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if repo.lastUpdateInput.NodeID.Int64 != 3 {
		t.Fatalf("updated node_id = %d, want 3", repo.lastUpdateInput.NodeID.Int64)
	}
	if repo.lastUpdateInput.ThresholdValue.Float64 != 90 {
		t.Fatalf("updated threshold = %v, want 90", repo.lastUpdateInput.ThresholdValue.Float64)
	}
	if repo.lastUpdateInput.Severity != types.AlertSeverityCritical {
		t.Fatalf("updated severity = %q, want %q", repo.lastUpdateInput.Severity, types.AlertSeverityCritical)
	}
	if repo.lastUpdateInput.IsEnabled {
		t.Fatal("expected update to disable rule")
	}
	if rule.NodeID == nil || *rule.NodeID != 3 {
		t.Fatalf("rule.NodeID = %#v, want 3", rule.NodeID)
	}
	if rule.ThresholdValue == nil || *rule.ThresholdValue != 90 {
		t.Fatalf("rule.ThresholdValue = %#v, want 90", rule.ThresholdValue)
	}
	if rule.Severity != types.AlertSeverityCritical {
		t.Fatalf("rule.Severity = %q, want %q", rule.Severity, types.AlertSeverityCritical)
	}
	if rule.IsEnabled {
		t.Fatal("expected returned rule to be disabled")
	}
	if instanceRepo.createCalls != 0 {
		t.Fatalf("expected disabled update not to evaluate current state, got %d alert creations", instanceRepo.createCalls)
	}
}

func TestAlertRuleServiceUpdateEvaluatesCurrentStateForUpdatedNode(t *testing.T) {
	t.Parallel()

	repo := &fakeAlertRuleServiceAlertRuleRepo{
		rule: generated.AlertRule{
			ID:             11,
			ProjectID:      1,
			NodeID:         sql.NullInt64{Int64: 2, Valid: true},
			RuleType:       types.AlertRuleTypeCPUAboveThreshold,
			Severity:       types.AlertSeverityWarning,
			MetricName:     sql.NullString{String: types.MetricNameCPUUsage, Valid: true},
			ThresholdValue: sql.NullFloat64{Float64: 80, Valid: true},
			IsEnabled:      true,
			CreatedAt:      time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
		},
	}
	metricRepo := &fakeAlertMetricRepo{
		samples: map[string]generated.MetricSample{
			metricKey(3, types.MetricNameCPUUsage): {
				NodeID:      3,
				MetricName:  types.MetricNameCPUUsage,
				MetricValue: 91,
				ObservedAt:  time.Date(2026, time.April, 5, 12, 4, 0, 0, time.UTC),
			},
		},
	}
	instanceRepo := &fakeAlertInstanceRepo{}
	evaluator := NewAlertEvaluatorService(repo, instanceRepo, metricRepo, &fakeAlertEventRecorder{}, nil)
	service := &AlertRuleService{
		alertRuleRepo: repo,
		projectRepo:   &fakeAlertRuleServiceProjectRepo{},
		nodeRepo: &fakeAlertRuleServiceNodeRepo{
			nodes: map[int64]generated.Node{
				2: {ID: 2, ProjectID: 1, CurrentState: types.NodeStateOnline},
				3: {ID: 3, ProjectID: 1, CurrentState: types.NodeStateOnline},
			},
		},
		serviceRepo: &fakeAlertRuleServiceServiceRepo{},
		evaluator:   evaluator,
	}

	nodeID := int64(3)

	_, err := service.Update(context.Background(), types.UpdateAlertRuleInput{
		ID:     11,
		NodeID: &nodeID,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if instanceRepo.createCalls != 1 {
		t.Fatalf("expected updated rule to be evaluated immediately, got %d alert creations", instanceRepo.createCalls)
	}
}

func TestAlertRuleServiceDeleteSucceeds(t *testing.T) {
	t.Parallel()

	service := &AlertRuleService{
		alertRuleRepo: &fakeAlertRuleServiceAlertRuleRepo{deleteRows: 1},
		projectRepo:   &fakeAlertRuleServiceProjectRepo{},
		nodeRepo:      &fakeAlertRuleServiceNodeRepo{},
		serviceRepo:   &fakeAlertRuleServiceServiceRepo{},
	}

	if err := service.Delete(context.Background(), 8); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestAlertRuleServiceDeleteMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := &AlertRuleService{
		alertRuleRepo: &fakeAlertRuleServiceAlertRuleRepo{deleteRows: 0},
		projectRepo:   &fakeAlertRuleServiceProjectRepo{},
		nodeRepo:      &fakeAlertRuleServiceNodeRepo{},
		serviceRepo:   &fakeAlertRuleServiceServiceRepo{},
	}

	err := service.Delete(context.Background(), 8)
	if !errors.Is(err, types.ErrAlertRuleNotFound) {
		t.Fatalf("Delete() error = %v, want ErrAlertRuleNotFound", err)
	}
}

func TestAlertRuleServiceGetByIDMapsNullableFields(t *testing.T) {
	t.Parallel()

	service := &AlertRuleService{
		alertRuleRepo: &fakeAlertRuleServiceAlertRuleRepo{
			rule: generated.AlertRule{
				ID:             4,
				ProjectID:      1,
				NodeID:         sql.NullInt64{},
				ServiceID:      sql.NullInt64{Int64: 9, Valid: true},
				RuleType:       types.AlertRuleTypeServiceUnhealthy,
				Severity:       types.AlertSeverityCritical,
				ThresholdValue: sql.NullFloat64{},
				IsEnabled:      true,
				CreatedAt:      time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
				UpdatedAt:      time.Date(2026, time.April, 5, 12, 1, 0, 0, time.UTC),
			},
		},
		projectRepo: &fakeAlertRuleServiceProjectRepo{},
		nodeRepo:    &fakeAlertRuleServiceNodeRepo{},
		serviceRepo: &fakeAlertRuleServiceServiceRepo{},
	}

	rule, err := service.GetByID(context.Background(), 4)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if rule.NodeID != nil {
		t.Fatalf("rule.NodeID = %#v, want nil", rule.NodeID)
	}
	if rule.ServiceID == nil || *rule.ServiceID != 9 {
		t.Fatalf("rule.ServiceID = %#v, want 9", rule.ServiceID)
	}
	if rule.ThresholdValue != nil {
		t.Fatalf("rule.ThresholdValue = %#v, want nil", rule.ThresholdValue)
	}
	if rule.Severity != types.AlertSeverityCritical {
		t.Fatalf("rule.Severity = %q, want %q", rule.Severity, types.AlertSeverityCritical)
	}
}

func TestAlertRuleServiceCreateEvaluatesCurrentServiceStateImmediately(t *testing.T) {
	t.Parallel()

	ruleRepo := &fakeAlertRuleServiceAlertRuleRepo{}
	instanceRepo := &fakeAlertInstanceRepo{}
	evaluator := NewAlertEvaluatorService(ruleRepo, instanceRepo, &fakeAlertMetricRepo{}, &fakeAlertEventRecorder{}, nil)
	serviceID := int64(7)

	service := &AlertRuleService{
		alertRuleRepo: ruleRepo,
		projectRepo: &fakeAlertRuleServiceProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1},
			},
		},
		nodeRepo: &fakeAlertRuleServiceNodeRepo{},
		serviceRepo: &fakeAlertRuleServiceServiceRepo{
			services: map[int64]generated.Service{
				7: {ID: 7, ProjectID: 1, CurrentState: types.ServiceStateUnhealthy},
			},
		},
		evaluator: evaluator,
	}

	_, err := service.Create(context.Background(), types.CreateAlertRuleInput{
		ProjectID: 1,
		ServiceID: &serviceID,
		RuleType:  types.AlertRuleTypeServiceUnhealthy,
		Severity:  types.AlertSeverityCritical,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	active, err := instanceRepo.FindActiveByRuleID(context.Background(), ruleRepo.rule.ID)
	if err != nil {
		t.Fatalf("expected active alert after create, got error: %v", err)
	}

	if active.ServiceID.Int64 != serviceID {
		t.Fatalf("active.ServiceID = %d, want %d", active.ServiceID.Int64, serviceID)
	}
}

func TestAlertRuleServiceCreateEvaluatesCurrentNodeStateImmediately(t *testing.T) {
	t.Parallel()

	ruleRepo := &fakeAlertRuleServiceAlertRuleRepo{}
	instanceRepo := &fakeAlertInstanceRepo{}
	evaluator := NewAlertEvaluatorService(ruleRepo, instanceRepo, &fakeAlertMetricRepo{}, &fakeAlertEventRecorder{}, nil)
	nodeID := int64(8)

	service := &AlertRuleService{
		alertRuleRepo: ruleRepo,
		projectRepo: &fakeAlertRuleServiceProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1},
			},
		},
		nodeRepo: &fakeAlertRuleServiceNodeRepo{
			nodes: map[int64]generated.Node{
				8: {ID: 8, ProjectID: 1, CurrentState: types.NodeStateOffline},
			},
		},
		serviceRepo: &fakeAlertRuleServiceServiceRepo{},
		evaluator:   evaluator,
	}

	_, err := service.Create(context.Background(), types.CreateAlertRuleInput{
		ProjectID: 1,
		NodeID:    &nodeID,
		RuleType:  types.AlertRuleTypeNodeOffline,
		Severity:  types.AlertSeverityCritical,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	active, err := instanceRepo.FindActiveByRuleID(context.Background(), ruleRepo.rule.ID)
	if err != nil {
		t.Fatalf("expected active alert after create, got error: %v", err)
	}

	if active.NodeID.Int64 != nodeID {
		t.Fatalf("active.NodeID = %d, want %d", active.NodeID.Int64, nodeID)
	}
}

func TestAlertRuleServiceCreateEvaluatesCurrentThresholdStateImmediately(t *testing.T) {
	t.Parallel()

	ruleRepo := &fakeAlertRuleServiceAlertRuleRepo{}
	instanceRepo := &fakeAlertInstanceRepo{}
	metricRepo := &fakeAlertMetricRepo{
		samples: map[string]generated.MetricSample{
			metricKey(9, types.MetricNameCPUUsage): {
				NodeID:      9,
				MetricName:  types.MetricNameCPUUsage,
				MetricValue: 95,
				ObservedAt:  time.Date(2026, time.April, 5, 11, 59, 0, 0, time.UTC),
			},
		},
	}
	evaluator := NewAlertEvaluatorService(ruleRepo, instanceRepo, metricRepo, &fakeAlertEventRecorder{}, nil)
	nodeID := int64(9)
	threshold := 80.0

	service := &AlertRuleService{
		alertRuleRepo: ruleRepo,
		projectRepo: &fakeAlertRuleServiceProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1},
			},
		},
		nodeRepo: &fakeAlertRuleServiceNodeRepo{
			nodes: map[int64]generated.Node{
				9: {ID: 9, ProjectID: 1, CurrentState: types.NodeStateOnline},
			},
		},
		serviceRepo: &fakeAlertRuleServiceServiceRepo{},
		evaluator:   evaluator,
	}

	_, err := service.Create(context.Background(), types.CreateAlertRuleInput{
		ProjectID:      1,
		NodeID:         &nodeID,
		RuleType:       types.AlertRuleTypeCPUAboveThreshold,
		Severity:       types.AlertSeverityWarning,
		ThresholdValue: &threshold,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	active, err := instanceRepo.FindActiveByRuleID(context.Background(), ruleRepo.rule.ID)
	if err != nil {
		t.Fatalf("expected active threshold alert after create, got error: %v", err)
	}

	if active.Title != "Node 9 CPU usage is above 80" {
		t.Fatalf("unexpected threshold alert title %q", active.Title)
	}
}

func TestAlertRuleServiceCreateDoesNotDuplicateExistingActiveAlert(t *testing.T) {
	t.Parallel()

	ruleRepo := &fakeAlertRuleServiceAlertRuleRepo{}
	instanceRepo := &fakeAlertInstanceRepo{
		instances: []generated.AlertInstance{
			{
				ID:          44,
				AlertRuleID: 11,
				ProjectID:   1,
				NodeID:      sql.NullInt64{Int64: 10, Valid: true},
				Status:      types.AlertStatusActive,
			},
		},
		nextID: 44,
	}
	evaluator := NewAlertEvaluatorService(ruleRepo, instanceRepo, &fakeAlertMetricRepo{}, &fakeAlertEventRecorder{}, nil)
	nodeID := int64(10)

	service := &AlertRuleService{
		alertRuleRepo: ruleRepo,
		projectRepo: &fakeAlertRuleServiceProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1},
			},
		},
		nodeRepo: &fakeAlertRuleServiceNodeRepo{
			nodes: map[int64]generated.Node{
				10: {ID: 10, ProjectID: 1, CurrentState: types.NodeStateOffline},
			},
		},
		serviceRepo: &fakeAlertRuleServiceServiceRepo{},
		evaluator:   evaluator,
	}

	_, err := service.Create(context.Background(), types.CreateAlertRuleInput{
		ProjectID: 1,
		NodeID:    &nodeID,
		RuleType:  types.AlertRuleTypeNodeOffline,
		Severity:  types.AlertSeverityCritical,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if instanceRepo.createCalls != 0 {
		t.Fatalf("expected no new active alert when one already exists, got %d creates", instanceRepo.createCalls)
	}

	if len(instanceRepo.instances) != 1 {
		t.Fatalf("expected existing alert instance to be reused, got %d instances", len(instanceRepo.instances))
	}
}
