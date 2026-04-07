package runner

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"reflect"
	"sync"
	"time"

	"github.com/MariusBobitiu/agrafa-agent/src/client"
	"github.com/MariusBobitiu/agrafa-agent/src/collectors"
	"github.com/MariusBobitiu/agrafa-agent/src/health"
	"github.com/MariusBobitiu/agrafa-agent/src/heartbeat"
	"github.com/MariusBobitiu/agrafa-agent/src/types"
)

type runnerAPIClient interface {
	SendHeartbeat(ctx context.Context, request types.HeartbeatRequest) error
	SendShutdown(ctx context.Context, request types.ShutdownRequest) error
	SendMetrics(ctx context.Context, request types.MetricsRequest) error
	SendHealth(ctx context.Context, request types.HealthRequest) error
	FetchConfig(ctx context.Context) (types.AgentConfigResponse, error)
}

type runnerMetricsCollector interface {
	Collect() (types.SystemMetrics, error)
}

type runnerHealthChecker interface {
	Run(ctx context.Context, check types.HealthCheck) types.HTTPCheckResult
}

type Runner struct {
	config           types.Config
	apiClient        runnerAPIClient
	metricsCollector runnerMetricsCollector
	healthChecker    runnerHealthChecker
	authFailures     *authFailureTracker
	configMu         sync.RWMutex
	nodeID           int64
	healthChecks     []types.HealthCheck
}

func New(
	config types.Config,
	apiClient runnerAPIClient,
	metricsCollector runnerMetricsCollector,
	healthChecker runnerHealthChecker,
) (*Runner, error) {
	return &Runner{
		config:           config,
		apiClient:        apiClient,
		metricsCollector: metricsCollector,
		healthChecker:    healthChecker,
		authFailures:     newAuthFailureTracker(log.Printf),
		nodeID:           config.NodeID,
		healthChecks:     append([]types.HealthCheck(nil), config.HealthChecks...),
	}, nil
}

func (runner *Runner) Start(ctx context.Context) error {
	var waitGroup sync.WaitGroup
	runner.runConfigRefresh(ctx)

	loops := []struct {
		name      string
		interval  time.Duration
		immediate bool
		run       func(context.Context)
	}{
		{name: "heartbeat", interval: runner.config.HeartbeatInterval, immediate: true, run: runner.runHeartbeat},
		{name: "metrics", interval: runner.config.MetricsInterval, immediate: true, run: runner.runMetrics},
		{name: "health", interval: runner.config.HealthInterval, immediate: true, run: runner.runHealthChecks},
		{name: "config", interval: runner.config.ConfigRefreshInterval, immediate: false, run: runner.runConfigRefresh},
	}

	for _, loop := range loops {
		waitGroup.Add(1)

		go func(loop struct {
			name      string
			interval  time.Duration
			immediate bool
			run       func(context.Context)
		}) {
			defer waitGroup.Done()
			runner.runLoop(ctx, loop.interval, loop.immediate, loop.run)
		}(loop)
	}

	<-ctx.Done()
	waitGroup.Wait()

	return ctx.Err()
}

func (runner *Runner) NotifyShutdown(ctx context.Context, reason string, payload map[string]any) error {
	request := types.ShutdownRequest{
		NodeID:     runner.currentNodeID(),
		ObservedAt: time.Now().UTC(),
		Reason:     reason,
		Payload:    payload,
	}

	if err := runner.apiClient.SendShutdown(ctx, request); err != nil {
		logSendFailure("shutdown", "/agent/shutdown", "", err)
		return err
	}

	return nil
}

func (runner *Runner) runLoop(ctx context.Context, interval time.Duration, immediate bool, run func(context.Context)) {
	if immediate {
		run(ctx)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run(ctx)
		}
	}
}

func (runner *Runner) runHeartbeat(ctx context.Context) {
	request := heartbeat.BuildRequest(runner.currentNodeID(), runner.config.Source)
	if err := runner.apiClient.SendHeartbeat(ctx, request); err != nil {
		if runner.authFailures.HandleResult("heartbeat", err) {
			return
		}

		logSendFailure("heartbeat", "/agent/heartbeat", "", err)
		return
	}

	runner.authFailures.HandleResult("heartbeat", nil)
}

func (runner *Runner) runMetrics(ctx context.Context) {
	metrics, err := runner.metricsCollector.Collect()
	if err != nil {
		log.Printf("metrics collection failed\n  error: %v", err)
		return
	}

	request := collectors.BuildMetricsRequest(runner.currentNodeID(), metrics)
	if err := runner.apiClient.SendMetrics(ctx, request); err != nil {
		if runner.authFailures.HandleResult("metrics", err) {
			return
		}

		logSendFailure("metrics", "/agent/metrics", "", err)
		return
	}

	runner.authFailures.HandleResult("metrics", nil)
}

func (runner *Runner) runHealthChecks(ctx context.Context) {
	for _, check := range runner.healthChecksSnapshot() {
		result := runner.healthChecker.Run(ctx, check)
		request := health.BuildHealthRequest(result)

		if err := runner.apiClient.SendHealth(ctx, request); err != nil {
			if runner.authFailures.HandleResult("health", err) {
				continue
			}

			logSendFailure("health", "/agent/health", healthTargetLabel(check), err)
			continue
		}

		runner.authFailures.HandleResult("health", nil)
	}
}

func (runner *Runner) runConfigRefresh(ctx context.Context) {
	configResponse, err := runner.apiClient.FetchConfig(ctx)
	if err != nil {
		if runner.authFailures.HandleResult("config", err) {
			return
		}

		logSendFailure("config refresh", "/agent/config", "", err)
		return
	}

	healthChecks, err := mapFetchedHealthChecks(configResponse)
	if err != nil {
		log.Printf("config refresh failed\n  path: /agent/config\n  kind: invalid config\n  error: %v\n  action: keeping last known config", err)
		return
	}

	runner.authFailures.HandleResult("config", nil)
	if changed := runner.replaceConfig(configResponse.Node.ID, healthChecks); changed {
		log.Printf(
			"agent config updated\n  node_id: %d\n  node_identifier: %s\n  health_checks: %d",
			configResponse.Node.ID,
			formatLogValue(configResponse.Node.Identifier),
			len(healthChecks),
		)
	}
}

func (runner *Runner) currentNodeID() int64 {
	runner.configMu.RLock()
	defer runner.configMu.RUnlock()

	return runner.nodeID
}

func (runner *Runner) healthChecksSnapshot() []types.HealthCheck {
	runner.configMu.RLock()
	defer runner.configMu.RUnlock()

	return append([]types.HealthCheck(nil), runner.healthChecks...)
}

func (runner *Runner) replaceConfig(nodeID int64, healthChecks []types.HealthCheck) bool {
	runner.configMu.Lock()
	defer runner.configMu.Unlock()

	if runner.nodeID == nodeID && reflect.DeepEqual(runner.healthChecks, healthChecks) {
		return false
	}

	runner.nodeID = nodeID
	runner.healthChecks = append([]types.HealthCheck(nil), healthChecks...)
	return true
}

func mapFetchedHealthChecks(configResponse types.AgentConfigResponse) ([]types.HealthCheck, error) {
	if configResponse.Node.ID <= 0 {
		return nil, fmt.Errorf("node.id must be greater than 0")
	}

	healthChecks := make([]types.HealthCheck, 0, len(configResponse.HealthChecks))
	for _, check := range configResponse.HealthChecks {
		mapped := types.HealthCheck{
			ServiceID:       check.ServiceID,
			Name:            check.Name,
			Type:            check.CheckType,
			Target:          check.CheckTarget,
			IntervalSeconds: check.IntervalSeconds,
			TimeoutSeconds:  check.TimeoutSeconds,
		}

		if err := validateFetchedHealthCheck(mapped); err != nil {
			return nil, fmt.Errorf("service_id %d: %w", check.ServiceID, err)
		}

		healthChecks = append(healthChecks, mapped)
	}

	return healthChecks, nil
}

func validateFetchedHealthCheck(check types.HealthCheck) error {
	if check.ServiceID <= 0 {
		return fmt.Errorf("health check service_id must be greater than 0")
	}
	if check.Name == "" {
		return fmt.Errorf("health check name is required")
	}
	if check.Type != "http" {
		return fmt.Errorf("unsupported health check type %q", check.Type)
	}
	if _, err := url.ParseRequestURI(check.Target); err != nil {
		return fmt.Errorf("invalid health check target: %w", err)
	}
	if check.TimeoutSeconds < 0 {
		return fmt.Errorf("health check timeout_seconds must be 0 or greater")
	}
	if check.IntervalSeconds < 0 {
		return fmt.Errorf("health check interval_seconds must be 0 or greater")
	}

	return nil
}

func logSendFailure(operation string, path string, target string, err error) {
	if apiErr, ok := client.AsAPIError(err); ok {
		if apiErr.StatusCode >= 500 {
			log.Printf(
				"%s send failed\n  path: %s\n  kind: backend unavailable\n  status: %d\n  attempts: %d\n  action: will try again on the next loop tick",
				operation,
				apiErr.Path,
				apiErr.StatusCode,
				apiErr.Attempts,
			)
			return
		}

		if target != "" {
			log.Printf(
				"%s send failed\n  target: %s\n  path: %s\n  kind: client error\n  status: %d\n  attempts: %d\n  response: %s",
				operation,
				target,
				apiErr.Path,
				apiErr.StatusCode,
				apiErr.Attempts,
				formatLogValue(apiErr.Body),
			)
			return
		}

		log.Printf(
			"%s send failed\n  path: %s\n  kind: client error\n  status: %d\n  attempts: %d\n  response: %s",
			operation,
			apiErr.Path,
			apiErr.StatusCode,
			apiErr.Attempts,
			formatLogValue(apiErr.Body),
		)
		return
	}

	if transportErr, ok := client.AsTransportError(err); ok {
		kind := "network error"
		if isTimeoutError(transportErr.Err) {
			kind = "timeout"
		}

		if target != "" {
			log.Printf(
				"%s send failed\n  target: %s\n  path: %s\n  kind: %s\n  attempts: %d\n  error: %v\n  action: will try again on the next loop tick",
				operation,
				target,
				transportErr.Path,
				kind,
				transportErr.Attempts,
				transportErr.Err,
			)
			return
		}

		log.Printf(
			"%s send failed\n  path: %s\n  kind: %s\n  attempts: %d\n  error: %v\n  action: will try again on the next loop tick",
			operation,
			transportErr.Path,
			kind,
			transportErr.Attempts,
			transportErr.Err,
		)
		return
	}

	if target != "" {
		log.Printf("%s send failed\n  target: %s\n  error: %v", operation, target, err)
		return
	}

	log.Printf("%s send failed\n  path: %s\n  error: %v", operation, path, err)
}

func isTimeoutError(err error) bool {
	type timeout interface {
		Timeout() bool
	}

	timeoutErr, ok := err.(timeout)
	return ok && timeoutErr.Timeout()
}

func healthTargetLabel(check types.HealthCheck) string {
	return fmt.Sprintf("service_id=%d name=%s", check.ServiceID, check.Name)
}

func formatLogValue(value string) string {
	if value == "" {
		return "n/a"
	}

	return value
}
