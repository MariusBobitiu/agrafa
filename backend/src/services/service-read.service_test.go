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
	rows         []generated.HealthCheckResult
	historyRows  []generated.HealthCheckResult
	streamRows   []generated.HealthCheckResult
	latest       generated.HealthCheckResult
	lastFilters  types.ServiceListFilters
	lastHistory  types.ServiceHistoryFilters
	serviceID    int64
	returnedErr  error
	summary      generated.SummarizeHealthCheckHistoryByServiceIDRow
	summaryRange types.ServiceHistoryRange
}

func (r *fakeServiceReadHealthCheckRepository) SummarizeHistoryByServiceID(_ context.Context, _ int64, historyRange types.ServiceHistoryRange) (generated.SummarizeHealthCheckHistoryByServiceIDRow, error) {
	r.summaryRange = historyRange
	return r.summary, r.returnedErr
}

func (r *fakeServiceReadHealthCheckRepository) GetLatestByServiceID(_ context.Context, _ int64) (generated.HealthCheckResult, error) {
	return r.latest, r.returnedErr
}

func (r *fakeServiceReadHealthCheckRepository) ListByServiceIDAfterID(_ context.Context, serviceID int64, afterID int64) ([]generated.HealthCheckResult, error) {
	r.serviceID = serviceID
	rows := make([]generated.HealthCheckResult, 0, len(r.streamRows))
	for _, row := range r.streamRows {
		if row.ID > afterID {
			rows = append(rows, row)
		}
	}
	return rows, r.returnedErr
}

func (r *fakeServiceReadHealthCheckRepository) ListLatestForRead(_ context.Context, filters types.ServiceListFilters) ([]generated.HealthCheckResult, error) {
	r.lastFilters = filters
	return r.rows, r.returnedErr
}

func (r *fakeServiceReadHealthCheckRepository) ListHistoryByServiceID(_ context.Context, serviceID int64, filters types.ServiceHistoryFilters) ([]generated.HealthCheckResult, error) {
	r.serviceID = serviceID
	r.lastHistory = filters
	return r.historyRows, r.returnedErr
}

func TestServiceReadServiceListHistoryOrdersMapsAndPaginates(t *testing.T) {
	t.Parallel()

	oldest := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)
	newest := oldest.Add(2 * time.Minute)
	middle := oldest.Add(time.Minute)
	before := &types.ServiceHistoryCursor{ObservedAt: newest.Add(time.Minute), ID: 4}
	repo := &fakeServiceReadHealthCheckRepository{historyRows: []generated.HealthCheckResult{
		{ID: 1, ServiceID: 7, NodeID: 8, CheckType: "tcp", Source: "managed", ObservedAt: oldest, IsSuccess: false, ResponseTimeMs: sql.NullInt32{Int32: 45, Valid: true}, Message: "refused", Payload: []byte(`{"runner":"managed"}`)},
		{ID: 3, ServiceID: 7, NodeID: 8, CheckType: "tcp", Source: "managed", ObservedAt: newest, IsSuccess: true, ResponseTimeMs: sql.NullInt32{Int32: 12, Valid: true}, Message: "ok", Payload: []byte(`{"runner":"managed"}`)},
		{ID: 2, ServiceID: 7, NodeID: 8, CheckType: "tcp", Source: "managed", ObservedAt: middle, IsSuccess: true, ResponseTimeMs: sql.NullInt32{Int32: 20, Valid: true}, Message: "ok", Payload: []byte(`{"runner":"managed"}`)},
	}}
	service := &ServiceReadService{healthCheckRepo: repo}

	page, err := service.ListHistory(context.Background(), 7, types.ServiceHistoryFilters{Limit: 2, Before: before})
	if err != nil {
		t.Fatalf("ListHistory() error = %v", err)
	}
	if repo.serviceID != 7 || repo.lastHistory.Limit != 3 {
		t.Fatalf("repository request = service %d limit %d, want service 7 limit 3", repo.serviceID, repo.lastHistory.Limit)
	}
	if repo.lastHistory.Before == nil || repo.lastHistory.Before.ID != 4 || !repo.lastHistory.Before.ObservedAt.Equal(before.ObservedAt) {
		t.Fatalf("repository before cursor = %#v, want %#v", repo.lastHistory.Before, before)
	}
	if len(page.Entries) != 2 || page.Entries[0].ID != 3 || page.Entries[1].ID != 2 {
		t.Fatalf("history order = %#v, want ids 3,2", page.Entries)
	}
	if page.Entries[0].StatusCode != nil {
		t.Fatalf("TCP status code = %#v, want nil", page.Entries[0].StatusCode)
	}
	if page.Entries[0].ResponseTimeMs == nil || *page.Entries[0].ResponseTimeMs != 12 {
		t.Fatalf("response time = %#v, want 12", page.Entries[0].ResponseTimeMs)
	}
	if page.NextCursor == nil || page.NextCursor.ID != 2 || !page.NextCursor.ObservedAt.Equal(middle) {
		t.Fatalf("next cursor = %#v, want middle row", page.NextCursor)
	}
}

func TestServiceReadServiceSummarizesFullRangeAndPreservesZeroLatency(t *testing.T) {
	t.Parallel()

	from := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	repo := &fakeServiceReadHealthCheckRepository{summary: generated.SummarizeHealthCheckHistoryByServiceIDRow{
		TotalChecks: 2505, SuccessfulChecks: 2004, MeasuredLatencyChecks: 2,
		TotalLatencyMs: 10, LastCheckedUnixNano: to.UnixNano(),
	}}
	service := &ServiceReadService{healthCheckRepo: repo}

	summary, err := service.SummarizeHistory(context.Background(), 7, types.ServiceHistoryRange{From: from, To: to})
	if err != nil {
		t.Fatalf("SummarizeHistory() error = %v", err)
	}
	if summary.TotalChecks != 2505 || summary.SuccessfulChecks != 2004 {
		t.Fatalf("unexpected counts: %#v", summary)
	}
	if summary.UptimePercent == nil || *summary.UptimePercent != 80 {
		t.Fatalf("uptime = %#v, want 80", summary.UptimePercent)
	}
	if summary.AverageLatencyMs == nil || *summary.AverageLatencyMs != 5 {
		t.Fatalf("average latency = %#v, want 5", summary.AverageLatencyMs)
	}
	if summary.LastCheckedAt == nil || !summary.LastCheckedAt.Equal(to) {
		t.Fatalf("last checked = %#v, want %v", summary.LastCheckedAt, to)
	}
	if repo.summaryRange.From != from || repo.summaryRange.To != to {
		t.Fatalf("summary range = %#v", repo.summaryRange)
	}

	repo.summary = generated.SummarizeHealthCheckHistoryByServiceIDRow{
		TotalChecks: 1, SuccessfulChecks: 1, MeasuredLatencyChecks: 1, TotalLatencyMs: 0,
		LastCheckedUnixNano: to.UnixNano(),
	}
	summary, err = service.SummarizeHistory(context.Background(), 7, types.ServiceHistoryRange{From: from, To: to})
	if err != nil || summary.AverageLatencyMs == nil || *summary.AverageLatencyMs != 0 {
		t.Fatalf("exact zero latency was not retained: summary=%#v err=%v", summary, err)
	}
}

func TestServiceReadServiceListStreamObservationsUsesPersistedIDOrder(t *testing.T) {
	t.Parallel()

	repo := &fakeServiceReadHealthCheckRepository{streamRows: []generated.HealthCheckResult{
		{ID: 13, ServiceID: 7, CheckType: "http", ObservedAt: time.Date(2026, 8, 23, 9, 0, 0, 0, time.UTC)},
		{ID: 12, ServiceID: 7, CheckType: "http", ObservedAt: time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)},
	}}
	service := &ServiceReadService{healthCheckRepo: repo}

	observations, err := service.ListStreamObservations(context.Background(), 7, 11)
	if err != nil {
		t.Fatalf("ListStreamObservations() error = %v", err)
	}
	if len(observations) != 2 || observations[0].ID != 12 || observations[1].ID != 13 {
		t.Fatalf("observation IDs = %#v, want 12,13", observations)
	}
	if !observations[1].ObservedAt.Before(observations[0].ObservedAt) {
		t.Fatalf("delayed observed_at row was not retained: %#v", observations)
	}
}

func TestServiceReadServiceStreamSnapshotUsesCanonicalPersistedHistoryEntry(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.August, 23, 12, 30, 0, 0, time.UTC)
	statusOK := sql.NullInt32{Int32: 200, Valid: true}
	latency := sql.NullInt32{Int32: 14, Valid: true}

	testCases := []struct {
		name         string
		checkType    string
		statusCode   sql.NullInt32
		responseTime sql.NullInt32
	}{
		{name: "HTTP with nullable latency", checkType: "http", statusCode: statusOK},
		{name: "TCP without HTTP status", checkType: "tcp", responseTime: latency},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			persisted := generated.HealthCheckResult{
				ID:             91,
				ServiceID:      7,
				NodeID:         8,
				CheckType:      testCase.checkType,
				Source:         types.HealthCheckSourceManaged,
				ObservedAt:     observedAt,
				IsSuccess:      true,
				StatusCode:     testCase.statusCode,
				ResponseTimeMs: testCase.responseTime,
				Message:        "ok",
				Payload:        []byte(`{"runner":"managed"}`),
			}
			healthRepo := &fakeServiceReadHealthCheckRepository{
				latest:      persisted,
				historyRows: []generated.HealthCheckResult{persisted},
			}
			service := &ServiceReadService{
				serviceRepo: &fakeServiceReadRepository{serviceByID: generated.Service{
					ID: 7, ProjectID: 2, NodeID: 8, CheckType: testCase.checkType,
				}},
				nodeRepo:          &fakeServiceReadNodeRepository{node: generated.Node{ID: 8, NodeType: types.NodeTypeManaged}},
				healthCheckRepo:   healthRepo,
				alertInstanceRepo: &fakeServiceReadAlertInstanceRepository{},
			}

			snapshot, err := service.GetStreamSnapshot(context.Background(), 7)
			if err != nil {
				t.Fatalf("GetStreamSnapshot() error = %v", err)
			}
			page, err := service.ListHistory(context.Background(), 7, types.ServiceHistoryFilters{Limit: 20})
			if err != nil {
				t.Fatalf("ListHistory() error = %v", err)
			}
			if snapshot.Observation == nil {
				t.Fatal("expected stream observation")
			}
			if len(page.Entries) != 1 || !reflect.DeepEqual(*snapshot.Observation, page.Entries[0]) {
				t.Fatalf("stream observation = %#v, history observation = %#v", snapshot.Observation, page.Entries)
			}
			if snapshot.Observation.CheckType != testCase.checkType {
				t.Fatalf("check_type = %q, want %q", snapshot.Observation.CheckType, testCase.checkType)
			}
			if testCase.checkType == "http" && snapshot.Observation.ResponseTimeMs != nil {
				t.Fatalf("response_time_ms = %#v, want nil", snapshot.Observation.ResponseTimeMs)
			}
			if testCase.checkType == "tcp" && snapshot.Observation.StatusCode != nil {
				t.Fatalf("status_code = %#v, want nil", snapshot.Observation.StatusCode)
			}
		})
	}
}

func TestServiceReadServiceListHistoryCapsRepositoryLimit(t *testing.T) {
	t.Parallel()

	repo := &fakeServiceReadHealthCheckRepository{}
	service := &ServiceReadService{healthCheckRepo: repo}
	if _, err := service.ListHistory(context.Background(), 7, types.ServiceHistoryFilters{Limit: 999}); err != nil {
		t.Fatalf("ListHistory() error = %v", err)
	}
	if repo.lastHistory.Limit != MaxServiceHistoryLimit+1 {
		t.Fatalf("repository limit = %d, want %d", repo.lastHistory.Limit, MaxServiceHistoryLimit+1)
	}
}

type fakeServiceReadAlertInstanceRepository struct {
	rows          []generated.ListActiveAlertCountsByServiceRow
	count         int64
	activeDetails []generated.ListActiveAlertDetailsByServiceIDRow
	lastFilters   types.ServiceListFilters
	returnedErr   error
}

func (r *fakeServiceReadAlertInstanceRepository) CountActiveByServiceID(_ context.Context, _ int64) (int64, error) {
	return r.count, r.returnedErr
}

func (r *fakeServiceReadAlertInstanceRepository) ListActiveDetailsByServiceID(_ context.Context, _ int64) ([]generated.ListActiveAlertDetailsByServiceIDRow, error) {
	return r.activeDetails, r.returnedErr
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
			activeDetails: []generated.ListActiveAlertDetailsByServiceIDRow{
				{
					ID:          101,
					RuleID:      201,
					RuleType:    types.AlertRuleTypeServiceUnhealthy,
					Severity:    types.AlertSeverityWarning,
					Title:       "Service 5 is unhealthy",
					Status:      types.AlertStatusActive,
					TriggeredAt: observedAt,
				},
			},
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
	if len(item.ActiveAlerts) != 1 {
		t.Fatalf("len(ActiveAlerts) = %d, want 1", len(item.ActiveAlerts))
	}
	if item.ActiveAlerts[0].RuleType != types.AlertRuleTypeServiceUnhealthy || item.ActiveAlerts[0].Severity != types.AlertSeverityWarning {
		t.Fatalf("unexpected active alert: %#v", item.ActiveAlerts[0])
	}
	if item.LatestHealthCheck == nil || item.LatestHealthCheck.StatusCode == nil || *item.LatestHealthCheck.StatusCode != 503 {
		t.Fatalf("unexpected latest health check: %#v", item.LatestHealthCheck)
	}
}

func TestServiceReadServiceGetByIDReturnsEmptyActiveAlertsWhenNoneExist(t *testing.T) {
	t.Parallel()

	service := &ServiceReadService{
		serviceRepo: &fakeServiceReadRepository{
			serviceByID: generated.Service{
				ID:        10,
				ProjectID: 1,
				NodeID:    26,
				Name:      "api",
			},
		},
		nodeRepo: &fakeServiceReadNodeRepository{
			node: generated.Node{ID: 26, NodeType: types.NodeTypeAgent},
		},
		healthCheckRepo:   &fakeServiceReadHealthCheckRepository{returnedErr: sql.ErrNoRows},
		alertInstanceRepo: &fakeServiceReadAlertInstanceRepository{},
	}

	item, err := service.GetByID(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if item.ActiveAlerts == nil {
		t.Fatal("expected active_alerts to be an empty slice, got nil")
	}
	if len(item.ActiveAlerts) != 0 {
		t.Fatalf("len(ActiveAlerts) = %d, want 0", len(item.ActiveAlerts))
	}
}

func TestServiceReadServiceGetByIDExcludesResolvedAlertsFromActiveAlerts(t *testing.T) {
	t.Parallel()

	service := &ServiceReadService{
		serviceRepo: &fakeServiceReadRepository{
			serviceByID: generated.Service{
				ID:        10,
				ProjectID: 1,
				NodeID:    26,
				Name:      "api",
			},
		},
		nodeRepo: &fakeServiceReadNodeRepository{
			node: generated.Node{ID: 26, NodeType: types.NodeTypeAgent},
		},
		healthCheckRepo: &fakeServiceReadHealthCheckRepository{returnedErr: sql.ErrNoRows},
		alertInstanceRepo: &fakeServiceReadAlertInstanceRepository{
			count:         0,
			activeDetails: []generated.ListActiveAlertDetailsByServiceIDRow{},
		},
	}

	item, err := service.GetByID(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if len(item.ActiveAlerts) != 0 {
		t.Fatalf("len(ActiveAlerts) = %d, want 0", len(item.ActiveAlerts))
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
