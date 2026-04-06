package services

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeServiceReadRepository struct {
	services    []generated.Service
	serviceByID generated.Service
	lastFilters types.ServiceListFilters
	returnedErr error
}

func (r *fakeServiceReadRepository) GetByID(_ context.Context, _ int64) (generated.Service, error) {
	return r.serviceByID, r.returnedErr
}

func (r *fakeServiceReadRepository) ListForRead(_ context.Context, filters types.ServiceListFilters) ([]generated.Service, error) {
	r.lastFilters = filters
	return r.services, r.returnedErr
}

type fakeServiceReadNodeRepository struct {
	node      generated.Node
	nodes     []generated.Node
	returnErr error
}

func (r *fakeServiceReadNodeRepository) GetByID(_ context.Context, _ int64) (generated.Node, error) {
	return r.node, r.returnErr
}

func (r *fakeServiceReadNodeRepository) List(_ context.Context) ([]generated.Node, error) {
	return r.nodes, r.returnErr
}

func (r *fakeServiceReadNodeRepository) ListByProject(_ context.Context, _ int64) ([]generated.Node, error) {
	return r.nodes, r.returnErr
}

type fakeServiceReadHealthCheckRepository struct {
	rows        []generated.HealthCheckResult
	latest      generated.HealthCheckResult
	lastFilters types.ServiceListFilters
	returnedErr error
}

func (r *fakeServiceReadHealthCheckRepository) GetLatestByServiceID(_ context.Context, _ int64) (generated.HealthCheckResult, error) {
	return r.latest, r.returnedErr
}

func (r *fakeServiceReadHealthCheckRepository) ListLatestForRead(_ context.Context, filters types.ServiceListFilters) ([]generated.HealthCheckResult, error) {
	r.lastFilters = filters
	return r.rows, r.returnedErr
}

type fakeServiceReadAlertInstanceRepository struct {
	rows        []generated.ListActiveAlertCountsByServiceRow
	count       int64
	lastFilters types.ServiceListFilters
	returnedErr error
}

func (r *fakeServiceReadAlertInstanceRepository) CountActiveByServiceID(_ context.Context, _ int64) (int64, error) {
	return r.count, r.returnedErr
}

func (r *fakeServiceReadAlertInstanceRepository) ListActiveCountsByServiceForRead(_ context.Context, filters types.ServiceListFilters) ([]generated.ListActiveAlertCountsByServiceRow, error) {
	r.lastFilters = filters
	return r.rows, r.returnedErr
}

func TestServiceReadServiceListPassesFiltersThrough(t *testing.T) {
	projectID := int64(11)
	nodeID := int64(22)
	status := types.ServiceStateDegraded
	limit := int32(5)

	testCases := []struct {
		name    string
		filters types.ServiceListFilters
	}{
		{
			name: "project_id",
			filters: types.ServiceListFilters{
				ProjectID: &projectID,
			},
		},
		{
			name: "node_id",
			filters: types.ServiceListFilters{
				NodeID: &nodeID,
			},
		},
		{
			name: "status",
			filters: types.ServiceListFilters{
				Status: &status,
			},
		},
		{
			name: "limit",
			filters: types.ServiceListFilters{
				Limit: &limit,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			serviceRepo := &fakeServiceReadRepository{
				services: []generated.Service{{ID: 7, NodeID: nodeID}},
			}
			healthCheckRepo := &fakeServiceReadHealthCheckRepository{}
			alertRepo := &fakeServiceReadAlertInstanceRepository{}

			service := &ServiceReadService{
				serviceRepo:       serviceRepo,
				nodeRepo:          &fakeServiceReadNodeRepository{nodes: []generated.Node{{ID: nodeID, NodeType: types.NodeTypeAgent}}},
				healthCheckRepo:   healthCheckRepo,
				alertInstanceRepo: alertRepo,
			}

			if _, err := service.List(context.Background(), tc.filters); err != nil {
				t.Fatalf("list services: %v", err)
			}

			if !reflect.DeepEqual(serviceRepo.lastFilters, tc.filters) {
				t.Fatalf("expected filters %#v, got %#v", tc.filters, serviceRepo.lastFilters)
			}
			if !reflect.DeepEqual(healthCheckRepo.lastFilters, tc.filters) {
				t.Fatalf("expected health check filters %#v, got %#v", tc.filters, healthCheckRepo.lastFilters)
			}
			if !reflect.DeepEqual(alertRepo.lastFilters, tc.filters) {
				t.Fatalf("expected alert filters %#v, got %#v", tc.filters, alertRepo.lastFilters)
			}
		})
	}
}

func TestServiceReadServiceListMapsFrontendResponseShape(t *testing.T) {
	lastCheckedAt := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	observedAt := time.Date(2026, 4, 5, 12, 1, 0, 0, time.UTC)

	service := &ServiceReadService{
		serviceRepo: &fakeServiceReadRepository{
			services: []generated.Service{
				{
					ID:                  5,
					ProjectID:           2,
					NodeID:              3,
					Name:                "api",
					CheckType:           "http",
					CheckTarget:         "https://example.com/health",
					CurrentState:        types.ServiceStateHealthy,
					ConsecutiveFailures: 2,
					LastCheckAt:         sql.NullTime{Time: lastCheckedAt, Valid: true},
				},
			},
		},
		nodeRepo: &fakeServiceReadNodeRepository{
			nodes: []generated.Node{
				{ID: 3, NodeType: types.NodeTypeManaged},
			},
		},
		healthCheckRepo: &fakeServiceReadHealthCheckRepository{
			rows: []generated.HealthCheckResult{
				{
					ServiceID:      5,
					ObservedAt:     observedAt,
					IsSuccess:      true,
					StatusCode:     sql.NullInt32{},
					ResponseTimeMs: sql.NullInt32{Int32: 87, Valid: true},
					Message:        "ok",
				},
			},
		},
		alertInstanceRepo: &fakeServiceReadAlertInstanceRepository{
			rows: []generated.ListActiveAlertCountsByServiceRow{
				{
					ServiceID:        sql.NullInt64{Int64: 5, Valid: true},
					ActiveAlertCount: 4,
				},
			},
		},
	}

	items, err := service.List(context.Background(), types.ServiceListFilters{})
	if err != nil {
		t.Fatalf("list services: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 service, got %d", len(items))
	}

	item := items[0]
	if item.ID != 5 || item.ProjectID != 2 || item.NodeID != 3 {
		t.Fatalf("unexpected identity fields: %#v", item)
	}
	if item.ExecutionMode != types.ExecutionModeManaged {
		t.Fatalf("expected execution_mode %q, got %q", types.ExecutionModeManaged, item.ExecutionMode)
	}
	if item.Status != types.ServiceStateHealthy {
		t.Fatalf("expected status %q, got %q", types.ServiceStateHealthy, item.Status)
	}
	if item.LastCheckedAt == nil || !item.LastCheckedAt.Equal(lastCheckedAt) {
		t.Fatalf("expected last_checked_at %v, got %#v", lastCheckedAt, item.LastCheckedAt)
	}
	if item.ConsecutiveFailures != 2 {
		t.Fatalf("expected consecutive_failures 2, got %d", item.ConsecutiveFailures)
	}
	if item.ActiveAlertCount != 4 {
		t.Fatalf("expected active_alert_count 4, got %d", item.ActiveAlertCount)
	}
	if item.LatestHealthCheck == nil {
		t.Fatal("expected latest_health_check to be present")
	}
	if item.LatestHealthCheck.StatusCode != nil {
		t.Fatalf("expected status_code nil, got %#v", item.LatestHealthCheck.StatusCode)
	}
	if item.LatestHealthCheck.ResponseTimeMs == nil || *item.LatestHealthCheck.ResponseTimeMs != 87 {
		t.Fatalf("expected response_time_ms 87, got %#v", item.LatestHealthCheck.ResponseTimeMs)
	}
	if item.LatestHealthCheck.Message != "ok" {
		t.Fatalf("expected latest health message %q, got %q", "ok", item.LatestHealthCheck.Message)
	}
	if !item.LatestHealthCheck.ObservedAt.Equal(observedAt) {
		t.Fatalf("expected observed_at %v, got %v", observedAt, item.LatestHealthCheck.ObservedAt)
	}
}

func TestServiceReadServiceGetByIDReturnsDetails(t *testing.T) {
	t.Parallel()

	lastCheckedAt := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	observedAt := time.Date(2026, 4, 5, 12, 1, 0, 0, time.UTC)

	service := &ServiceReadService{
		serviceRepo: &fakeServiceReadRepository{
			serviceByID: generated.Service{
				ID:                  5,
				ProjectID:           2,
				NodeID:              3,
				Name:                "api",
				CheckType:           "http",
				CheckTarget:         "https://example.com/health",
				CurrentState:        types.ServiceStateUnhealthy,
				ConsecutiveFailures: 4,
				LastCheckAt:         sql.NullTime{Time: lastCheckedAt, Valid: true},
			},
		},
		nodeRepo: &fakeServiceReadNodeRepository{
			node: generated.Node{ID: 3, NodeType: types.NodeTypeAgent},
		},
		healthCheckRepo: &fakeServiceReadHealthCheckRepository{
			latest: generated.HealthCheckResult{
				ServiceID:      5,
				ObservedAt:     observedAt,
				IsSuccess:      false,
				StatusCode:     sql.NullInt32{Int32: 503, Valid: true},
				ResponseTimeMs: sql.NullInt32{Int32: 250, Valid: true},
				Message:        "down",
			},
		},
		alertInstanceRepo: &fakeServiceReadAlertInstanceRepository{
			count: 2,
		},
	}

	item, err := service.GetByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if item.ID != 5 || item.ProjectID != 2 || item.NodeID != 3 {
		t.Fatalf("unexpected identity fields: %#v", item)
	}
	if item.ExecutionMode != types.ExecutionModeAgent {
		t.Fatalf("expected execution_mode %q, got %q", types.ExecutionModeAgent, item.ExecutionMode)
	}
	if item.ActiveAlertCount != 2 {
		t.Fatalf("ActiveAlertCount = %d, want 2", item.ActiveAlertCount)
	}
	if item.LatestHealthCheck == nil || item.LatestHealthCheck.StatusCode == nil || *item.LatestHealthCheck.StatusCode != 503 {
		t.Fatalf("unexpected latest health check: %#v", item.LatestHealthCheck)
	}
}

func TestServiceReadServiceGetByIDMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := &ServiceReadService{
		serviceRepo:       &fakeServiceReadRepository{returnedErr: sql.ErrNoRows},
		nodeRepo:          &fakeServiceReadNodeRepository{},
		healthCheckRepo:   &fakeServiceReadHealthCheckRepository{},
		alertInstanceRepo: &fakeServiceReadAlertInstanceRepository{},
	}

	_, err := service.GetByID(context.Background(), 42)
	if !errors.Is(err, types.ErrServiceNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrServiceNotFound", err)
	}
}
