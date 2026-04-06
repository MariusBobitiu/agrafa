package runner

import (
	"context"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-agent/src/client"
	"github.com/MariusBobitiu/agrafa-agent/src/types"
)

type fakeRunnerAPIClient struct {
	fetchResults      []fakeFetchResult
	fetchCalls        int
	healthRequests    []types.HealthRequest
	heartbeatRequests []types.HeartbeatRequest
	metricsRequests   []types.MetricsRequest
}

type fakeFetchResult struct {
	config types.AgentConfigResponse
	err    error
}

func (c *fakeRunnerAPIClient) SendHeartbeat(_ context.Context, request types.HeartbeatRequest) error {
	c.heartbeatRequests = append(c.heartbeatRequests, request)
	return nil
}

func (c *fakeRunnerAPIClient) SendMetrics(_ context.Context, request types.MetricsRequest) error {
	c.metricsRequests = append(c.metricsRequests, request)
	return nil
}

func (c *fakeRunnerAPIClient) SendHealth(_ context.Context, request types.HealthRequest) error {
	c.healthRequests = append(c.healthRequests, request)
	return nil
}

func (c *fakeRunnerAPIClient) FetchConfig(_ context.Context) (types.AgentConfigResponse, error) {
	if c.fetchCalls >= len(c.fetchResults) {
		return types.AgentConfigResponse{}, nil
	}

	result := c.fetchResults[c.fetchCalls]
	c.fetchCalls++
	return result.config, result.err
}

type fakeRunnerMetricsCollector struct{}

func (c *fakeRunnerMetricsCollector) Collect() (types.SystemMetrics, error) {
	return types.SystemMetrics{ObservedAt: time.Now().UTC()}, nil
}

type fakeRunnerHealthChecker struct {
	runChecks []types.HealthCheck
}

func (c *fakeRunnerHealthChecker) Run(_ context.Context, check types.HealthCheck) types.HTTPCheckResult {
	c.runChecks = append(c.runChecks, check)
	return types.HTTPCheckResult{
		ServiceID:  check.ServiceID,
		Name:       check.Name,
		Type:       check.Type,
		Target:     check.Target,
		ObservedAt: time.Now().UTC(),
		IsSuccess:  true,
		Message:    "ok",
	}
}

func TestRunnerConfigRefreshPopulatesInMemoryChecks(t *testing.T) {
	t.Parallel()

	runner := newTestRunner(t, types.Config{}, &fakeRunnerAPIClient{
		fetchResults: []fakeFetchResult{
			{
				config: types.AgentConfigResponse{
					Node: types.AgentConfigNode{ID: 12, Identifier: "web-01"},
					HealthChecks: []types.AgentConfigCheck{
						{
							ServiceID:       101,
							Name:            "internal-api",
							CheckType:       "http",
							CheckTarget:     "http://internal-api.local/health",
							IntervalSeconds: 30,
							TimeoutSeconds:  5,
						},
					},
				},
			},
		},
	})

	runner.runConfigRefresh(context.Background())

	checks := runner.healthChecksSnapshot()
	if len(checks) != 1 {
		t.Fatalf("len(checks) = %d, want 1", len(checks))
	}
	if checks[0].ServiceID != 101 || checks[0].Target != "http://internal-api.local/health" {
		t.Fatalf("unexpected checks: %#v", checks)
	}
	if runner.currentNodeID() != 12 {
		t.Fatalf("currentNodeID = %d, want 12", runner.currentNodeID())
	}
}

func TestRunnerConfigRefreshReplacesInMemoryChecks(t *testing.T) {
	t.Parallel()

	initialConfig := types.Config{
		NodeID: 1,
		HealthChecks: []types.HealthCheck{
			{ServiceID: 1, Name: "old", Type: "http", Target: "http://old.local/health"},
		},
	}

	runner := newTestRunner(t, initialConfig, &fakeRunnerAPIClient{
		fetchResults: []fakeFetchResult{
			{
				config: types.AgentConfigResponse{
					Node: types.AgentConfigNode{ID: 9, Identifier: "web-09"},
					HealthChecks: []types.AgentConfigCheck{
						{
							ServiceID:   2,
							Name:        "new",
							CheckType:   "http",
							CheckTarget: "http://new.local/health",
						},
					},
				},
			},
		},
	})

	runner.runConfigRefresh(context.Background())

	checks := runner.healthChecksSnapshot()
	if len(checks) != 1 || checks[0].ServiceID != 2 {
		t.Fatalf("unexpected checks after refresh: %#v", checks)
	}
}

func TestRunnerConfigRefreshFailureKeepsPriorChecks(t *testing.T) {
	t.Parallel()

	previousCheck := types.HealthCheck{ServiceID: 1, Name: "fallback", Type: "http", Target: "http://fallback.local/health"}
	runner := newTestRunner(t, types.Config{
		NodeID:       1,
		HealthChecks: []types.HealthCheck{previousCheck},
	}, &fakeRunnerAPIClient{
		fetchResults: []fakeFetchResult{
			{
				err: &client.TransportError{Path: "/agent/config", Attempts: 1, Err: fakeRunnerTimeoutError{message: "timeout"}},
			},
		},
	})

	runner.runConfigRefresh(context.Background())

	checks := runner.healthChecksSnapshot()
	if len(checks) != 1 || checks[0] != previousCheck {
		t.Fatalf("expected prior checks to remain, got %#v", checks)
	}
}

func TestRunnerHealthLoopUsesFetchedChecks(t *testing.T) {
	t.Parallel()

	apiClient := &fakeRunnerAPIClient{
		fetchResults: []fakeFetchResult{
			{
				config: types.AgentConfigResponse{
					Node: types.AgentConfigNode{ID: 12, Identifier: "web-01"},
					HealthChecks: []types.AgentConfigCheck{
						{
							ServiceID:   101,
							Name:        "internal-api",
							CheckType:   "http",
							CheckTarget: "http://internal-api.local/health",
						},
					},
				},
			},
		},
	}
	healthChecker := &fakeRunnerHealthChecker{}
	runner := newTestRunnerWithDeps(t, types.Config{}, apiClient, &fakeRunnerMetricsCollector{}, healthChecker)

	runner.runConfigRefresh(context.Background())
	runner.runHealthChecks(context.Background())

	if len(healthChecker.runChecks) != 1 {
		t.Fatalf("len(runChecks) = %d, want 1", len(healthChecker.runChecks))
	}
	if len(apiClient.healthRequests) != 1 {
		t.Fatalf("len(healthRequests) = %d, want 1", len(apiClient.healthRequests))
	}
	if apiClient.healthRequests[0].ServiceID != 101 {
		t.Fatalf("unexpected health request: %#v", apiClient.healthRequests[0])
	}
}

func TestRunnerHealthLoopHandlesNoChecks(t *testing.T) {
	t.Parallel()

	apiClient := &fakeRunnerAPIClient{}
	healthChecker := &fakeRunnerHealthChecker{}
	runner := newTestRunnerWithDeps(t, types.Config{}, apiClient, &fakeRunnerMetricsCollector{}, healthChecker)

	runner.runHealthChecks(context.Background())

	if len(healthChecker.runChecks) != 0 {
		t.Fatalf("len(runChecks) = %d, want 0", len(healthChecker.runChecks))
	}
	if len(apiClient.healthRequests) != 0 {
		t.Fatalf("len(healthRequests) = %d, want 0", len(apiClient.healthRequests))
	}
}

func newTestRunner(t *testing.T, config types.Config, apiClient *fakeRunnerAPIClient) *Runner {
	t.Helper()
	return newTestRunnerWithDeps(t, config, apiClient, &fakeRunnerMetricsCollector{}, &fakeRunnerHealthChecker{})
}

func newTestRunnerWithDeps(t *testing.T, config types.Config, apiClient *fakeRunnerAPIClient, metricsCollector runnerMetricsCollector, healthChecker runnerHealthChecker) *Runner {
	t.Helper()

	config.NodeID = firstPositive(config.NodeID, 1)
	config.Source = "agent"
	config.HeartbeatInterval = firstDuration(config.HeartbeatInterval, time.Second)
	config.MetricsInterval = firstDuration(config.MetricsInterval, time.Second)
	config.HealthInterval = firstDuration(config.HealthInterval, time.Second)
	config.ConfigRefreshInterval = firstDuration(config.ConfigRefreshInterval, time.Second)

	runner, err := New(config, apiClient, metricsCollector, healthChecker)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return runner
}

func firstPositive(value int64, fallback int64) int64 {
	if value > 0 {
		return value
	}

	return fallback
}

func firstDuration(value time.Duration, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}

	return fallback
}

type fakeRunnerTimeoutError struct {
	message string
}

func (e fakeRunnerTimeoutError) Error() string {
	return e.message
}
