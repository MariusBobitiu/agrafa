package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeNodeReadNodeRepo struct {
	node         generated.Node
	visibleNodes []generated.Node
	err          error
}

func (r *fakeNodeReadNodeRepo) GetByID(_ context.Context, _ int64) (generated.Node, error) {
	return r.node, r.err
}

func (r *fakeNodeReadNodeRepo) ListVisible(_ context.Context) ([]generated.Node, error) {
	if r.visibleNodes != nil {
		return r.visibleNodes, r.err
	}

	return []generated.Node{r.node}, r.err
}

func (r *fakeNodeReadNodeRepo) ListVisibleByProject(_ context.Context, _ int64) ([]generated.Node, error) {
	if r.visibleNodes != nil {
		return r.visibleNodes, r.err
	}

	return []generated.Node{r.node}, r.err
}

type fakeNodeReadMetricRepo struct {
	metrics map[string]generated.MetricSample
	errs    map[string]error
}

func (r *fakeNodeReadMetricRepo) GetLatestNodeMetricByName(_ context.Context, _ int64, metricName string) (generated.MetricSample, error) {
	if err := r.errs[metricName]; err != nil {
		return generated.MetricSample{}, err
	}
	metric, ok := r.metrics[metricName]
	if !ok {
		return generated.MetricSample{}, sql.ErrNoRows
	}

	return metric, nil
}

func (r *fakeNodeReadMetricRepo) ListLatestNodeMetrics(_ context.Context, _ *int64) ([]generated.ListLatestOperationalNodeMetricsRow, error) {
	return nil, nil
}

type fakeNodeReadAlertRepo struct {
	count int64
	err   error
}

func (r *fakeNodeReadAlertRepo) CountActiveByNodeID(_ context.Context, _ int64) (int64, error) {
	return r.count, r.err
}

func (r *fakeNodeReadAlertRepo) ListActiveCountsByNode(_ context.Context, _ *int64) ([]generated.ListActiveAlertCountsByNodeRow, error) {
	return nil, nil
}

type fakeNodeReadServiceRepo struct {
	count int64
	err   error
}

func (r *fakeNodeReadServiceRepo) CountByNodeID(_ context.Context, _ int64) (int64, error) {
	return r.count, r.err
}

func (r *fakeNodeReadServiceRepo) ListCountsByNode(_ context.Context, _ *int64) ([]generated.ListServiceCountsByNodeRow, error) {
	return nil, nil
}

func TestNodeReadServiceGetByIDMapsFrontendShape(t *testing.T) {
	t.Parallel()

	lastSeenAt := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	observedAt := time.Date(2026, time.April, 5, 12, 1, 0, 0, time.UTC)

	service := &NodeReadService{
		nodeRepo: &fakeNodeReadNodeRepo{
			node: generated.Node{
				ID:              5,
				ProjectID:       2,
				Name:            "edge-1",
				Identifier:      "edge-1",
				CurrentState:    types.NodeStateOnline,
				LastHeartbeatAt: sql.NullTime{Time: lastSeenAt, Valid: true},
				Metadata:        json.RawMessage(`{"region":"eu-west"}`),
				CreatedAt:       time.Date(2026, time.April, 5, 11, 0, 0, 0, time.UTC),
				UpdatedAt:       time.Date(2026, time.April, 5, 12, 2, 0, 0, time.UTC),
			},
		},
		metricRepo: &fakeNodeReadMetricRepo{
			metrics: map[string]generated.MetricSample{
				types.MetricNameCPUUsage: {
					MetricValue: 73.4,
					MetricUnit:  "percent",
					ObservedAt:  observedAt,
				},
			},
		},
		alertInstanceRepo: &fakeNodeReadAlertRepo{count: 3},
		serviceRepo:       &fakeNodeReadServiceRepo{count: 4},
	}

	node, err := service.GetByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if node.LastSeenAt == nil || !node.LastSeenAt.Equal(lastSeenAt) {
		t.Fatalf("node.LastSeenAt = %#v, want %v", node.LastSeenAt, lastSeenAt)
	}
	if node.LatestCPU == nil || node.LatestCPU.Value != 73.4 {
		t.Fatalf("node.LatestCPU = %#v, want populated value", node.LatestCPU)
	}
	if node.LatestMemory != nil || node.LatestDisk != nil {
		t.Fatalf("expected missing metrics to map to nil, got memory=%#v disk=%#v", node.LatestMemory, node.LatestDisk)
	}
	if node.ActiveAlertCount != 3 || node.ServiceCount != 4 {
		t.Fatalf("unexpected counts: %#v", node)
	}
}

func TestNodeReadServiceGetByIDMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := &NodeReadService{
		nodeRepo:          &fakeNodeReadNodeRepo{err: sql.ErrNoRows},
		metricRepo:        &fakeNodeReadMetricRepo{},
		alertInstanceRepo: &fakeNodeReadAlertRepo{},
		serviceRepo:       &fakeNodeReadServiceRepo{},
	}

	_, err := service.GetByID(context.Background(), 5)
	if !errors.Is(err, types.ErrNodeNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNodeNotFound", err)
	}
}

func TestNodeReadServiceListExcludesHiddenManagedNodes(t *testing.T) {
	t.Parallel()

	service := &NodeReadService{
		nodeRepo: &fakeNodeReadNodeRepo{
			visibleNodes: []generated.Node{
				{ID: 7, ProjectID: 2, Name: "edge-1", Identifier: "edge-1", NodeType: types.NodeTypeAgent, IsVisible: true},
			},
		},
		metricRepo:        &fakeNodeReadMetricRepo{},
		alertInstanceRepo: &fakeNodeReadAlertRepo{},
		serviceRepo:       &fakeNodeReadServiceRepo{},
	}

	nodes, err := service.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("len(nodes) = %d, want 1", len(nodes))
	}
	if nodes[0].ID != 7 {
		t.Fatalf("nodes[0].ID = %d, want 7", nodes[0].ID)
	}
}
