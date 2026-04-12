package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

const managedBootstrapRunnerSource = "managed"

type managedBootstrapHeartbeatIngester interface {
	Ingest(ctx context.Context, input types.HeartbeatInput) (generated.Node, error)
}

type managedBootstrapHealthIngester interface {
	Ingest(ctx context.Context, input types.HealthCheckInput) (generated.Service, error)
}

type ManagedServiceBootstrapChecker struct {
	heartbeats managedBootstrapHeartbeatIngester
	health     managedBootstrapHealthIngester
	timeout    time.Duration
	httpClient *http.Client
	tcpDialer  *net.Dialer
}

func NewManagedServiceBootstrapChecker(
	heartbeatService *HeartbeatService,
	healthService *HealthIngestionService,
	timeout time.Duration,
) *ManagedServiceBootstrapChecker {
	return &ManagedServiceBootstrapChecker{
		heartbeats: heartbeatService,
		health:     healthService,
		timeout:    timeout,
		httpClient: &http.Client{},
		tcpDialer:  &net.Dialer{Timeout: timeout},
	}
}

func (c *ManagedServiceBootstrapChecker) CheckNow(ctx context.Context, service generated.Service) error {
	if c == nil {
		return nil
	}

	observedAt := time.Now().UTC()
	if _, err := c.heartbeats.Ingest(ctx, types.HeartbeatInput{
		AuthenticatedNodeID: service.NodeID,
		ObservedAt:          observedAt,
		Source:              managedBootstrapRunnerSource,
		Payload:             managedBootstrapCheckPayload("heartbeat"),
	}); err != nil {
		return fmt.Errorf("ingest heartbeat: %w", err)
	}

	result := c.executeCheck(ctx, service.CheckType, service.CheckTarget)
	if _, err := c.health.Ingest(ctx, types.HealthCheckInput{
		AuthenticatedNodeID: service.NodeID,
		ServiceID:           service.ID,
		ObservedAt:          result.observedAt,
		IsSuccess:           result.isSuccess,
		StatusCode:          result.statusCode,
		ResponseTimeMs:      result.responseTimeMs,
		Message:             result.message,
		Payload:             result.payload,
	}); err != nil {
		return fmt.Errorf("ingest health check: %w", err)
	}

	return nil
}

type managedBootstrapCheckResult struct {
	observedAt     time.Time
	isSuccess      bool
	statusCode     *int32
	responseTimeMs *int32
	message        string
	payload        json.RawMessage
}

func (c *ManagedServiceBootstrapChecker) executeCheck(ctx context.Context, checkType string, checkTarget string) managedBootstrapCheckResult {
	switch strings.ToLower(strings.TrimSpace(checkType)) {
	case "http":
		return c.executeHTTPCheck(ctx, checkTarget)
	case "tcp":
		return c.executeTCPCheck(ctx, checkTarget)
	default:
		observedAt := time.Now().UTC()
		return managedBootstrapCheckResult{
			observedAt: observedAt,
			isSuccess:  false,
			message:    "unsupported check type",
			payload:    managedBootstrapCheckPayload(strings.ToLower(strings.TrimSpace(checkType))),
		}
	}
}

func (c *ManagedServiceBootstrapChecker) executeHTTPCheck(ctx context.Context, checkTarget string) managedBootstrapCheckResult {
	observedAt := time.Now().UTC()
	checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(checkCtx, http.MethodGet, checkTarget, nil)
	if err != nil {
		return managedBootstrapCheckResult{
			observedAt: observedAt,
			isSuccess:  false,
			message:    err.Error(),
			payload:    managedBootstrapCheckPayload("http"),
		}
	}

	startedAt := time.Now()
	response, err := c.httpClient.Do(request)
	elapsed := time.Since(startedAt)
	if err != nil {
		return managedBootstrapCheckResult{
			observedAt:     observedAt,
			isSuccess:      false,
			responseTimeMs: durationToMillisecondsPtr(elapsed),
			message:        err.Error(),
			payload:        managedBootstrapCheckPayload("http"),
		}
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)

	statusCode := int32(response.StatusCode)
	return managedBootstrapCheckResult{
		observedAt:     observedAt,
		isSuccess:      response.StatusCode >= 200 && response.StatusCode < 400,
		statusCode:     &statusCode,
		responseTimeMs: durationToMillisecondsPtr(elapsed),
		message:        response.Status,
		payload:        managedBootstrapCheckPayload("http"),
	}
}

func (c *ManagedServiceBootstrapChecker) executeTCPCheck(ctx context.Context, checkTarget string) managedBootstrapCheckResult {
	observedAt := time.Now().UTC()
	address, err := normalizeManagedBootstrapTCPAddress(checkTarget)
	if err != nil {
		return managedBootstrapCheckResult{
			observedAt: observedAt,
			isSuccess:  false,
			message:    err.Error(),
			payload:    managedBootstrapCheckPayload("tcp"),
		}
	}

	checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	startedAt := time.Now()
	conn, err := c.tcpDialer.DialContext(checkCtx, "tcp", address)
	elapsed := time.Since(startedAt)
	if err != nil {
		return managedBootstrapCheckResult{
			observedAt:     observedAt,
			isSuccess:      false,
			responseTimeMs: durationToMillisecondsPtr(elapsed),
			message:        err.Error(),
			payload:        managedBootstrapCheckPayload("tcp"),
		}
	}
	_ = conn.Close()

	return managedBootstrapCheckResult{
		observedAt:     observedAt,
		isSuccess:      true,
		responseTimeMs: durationToMillisecondsPtr(elapsed),
		message:        "tcp connection succeeded",
		payload:        managedBootstrapCheckPayload("tcp"),
	}
}

func normalizeManagedBootstrapTCPAddress(target string) (string, error) {
	trimmedTarget := strings.TrimSpace(target)
	if trimmedTarget == "" {
		return "", types.ErrInvalidCheckTarget
	}

	if !strings.Contains(trimmedTarget, "://") {
		return trimmedTarget, nil
	}

	parsedURL, err := url.Parse(trimmedTarget)
	if err != nil {
		return "", err
	}

	if parsedURL.Host != "" {
		return parsedURL.Host, nil
	}

	return "", types.ErrInvalidCheckTarget
}

func durationToMillisecondsPtr(value time.Duration) *int32 {
	milliseconds := int32(value.Milliseconds())
	return &milliseconds
}

func managedBootstrapCheckPayload(checkType string) json.RawMessage {
	return json.RawMessage(`{"runner":"managed","check_type":"` + checkType + `"}`)
}
