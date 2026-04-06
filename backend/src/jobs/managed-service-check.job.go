package jobs

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

const managedRunnerSource = "managed"

type managedServiceLister interface {
	ListServices(ctx context.Context, projectID *int64) ([]generated.Service, error)
}

type managedNodeLister interface {
	ListNodes(ctx context.Context, projectID *int64) ([]generated.Node, error)
}

type managedHeartbeatIngester interface {
	Ingest(ctx context.Context, input types.HeartbeatInput) (generated.Node, error)
}

type managedHealthIngester interface {
	Ingest(ctx context.Context, input types.HealthCheckInput) (generated.Service, error)
}

type managedCheckRunner func(ctx context.Context, checkType string, checkTarget string) managedCheckResult

type ManagedServiceCheckJob struct {
	services    managedServiceLister
	nodes       managedNodeLister
	heartbeats  managedHeartbeatIngester
	health      managedHealthIngester
	interval    time.Duration
	timeout     time.Duration
	httpClient  *http.Client
	tcpDialer   *net.Dialer
	checkRunner managedCheckRunner
}

type managedCheckResult struct {
	observedAt     time.Time
	isSuccess      bool
	statusCode     *int32
	responseTimeMs *int32
	message        string
	payload        json.RawMessage
}

func NewManagedServiceCheckJob(
	serviceState *services.ServiceStateService,
	nodeState *services.NodeStateService,
	heartbeatService *services.HeartbeatService,
	healthService *services.HealthIngestionService,
	interval time.Duration,
	timeout time.Duration,
) *ManagedServiceCheckJob {
	job := &ManagedServiceCheckJob{
		services:   serviceState,
		nodes:      nodeState,
		heartbeats: heartbeatService,
		health:     healthService,
		interval:   interval,
		timeout:    timeout,
		httpClient: &http.Client{},
		tcpDialer:  &net.Dialer{Timeout: timeout},
	}
	job.checkRunner = job.executeCheck

	return job
}

func (j *ManagedServiceCheckJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	j.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.runOnce(ctx)
		}
	}
}

func (j *ManagedServiceCheckJob) runOnce(ctx context.Context) {
	nodes, err := j.nodes.ListNodes(ctx, nil)
	if err != nil {
		log.Printf("managed service check node query failed: %v", err)
		return
	}

	allServices, err := j.services.ListServices(ctx, nil)
	if err != nil {
		log.Printf("managed service check service query failed: %v", err)
		return
	}

	nodeByID := make(map[int64]generated.Node, len(nodes))
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}

	managedServicesByNode := make(map[int64][]generated.Service)
	for _, service := range allServices {
		node, ok := nodeByID[service.NodeID]
		if !ok || node.NodeType != types.NodeTypeManaged {
			continue
		}

		managedServicesByNode[service.NodeID] = append(managedServicesByNode[service.NodeID], service)
	}

	for nodeID, managedServices := range managedServicesByNode {
		observedAt := timeNow().UTC()
		if _, err := j.heartbeats.Ingest(ctx, types.HeartbeatInput{
			AuthenticatedNodeID: nodeID,
			ObservedAt:          observedAt,
			Source:              managedRunnerSource,
			Payload:             managedCheckPayload("heartbeat"),
		}); err != nil {
			log.Printf("managed service check heartbeat failed for node %d: %v", nodeID, err)
			continue
		}

		for _, service := range managedServices {
			result := j.checkRunner(ctx, service.CheckType, service.CheckTarget)
			if _, err := j.health.Ingest(ctx, types.HealthCheckInput{
				AuthenticatedNodeID: service.NodeID,
				ServiceID:           service.ID,
				ObservedAt:          result.observedAt,
				IsSuccess:           result.isSuccess,
				StatusCode:          result.statusCode,
				ResponseTimeMs:      result.responseTimeMs,
				Message:             result.message,
				Payload:             result.payload,
			}); err != nil {
				log.Printf("managed service check ingestion failed for service %d: %v", service.ID, err)
			}
		}
	}
}

func (j *ManagedServiceCheckJob) executeCheck(ctx context.Context, checkType string, checkTarget string) managedCheckResult {
	switch strings.ToLower(strings.TrimSpace(checkType)) {
	case "http":
		return j.executeHTTPCheck(ctx, checkTarget)
	case "tcp":
		return j.executeTCPCheck(ctx, checkTarget)
	default:
		observedAt := timeNow().UTC()
		return managedCheckResult{
			observedAt: observedAt,
			isSuccess:  false,
			message:    "unsupported check type",
			payload:    managedCheckPayload(strings.ToLower(strings.TrimSpace(checkType))),
		}
	}
}

func (j *ManagedServiceCheckJob) executeHTTPCheck(ctx context.Context, checkTarget string) managedCheckResult {
	observedAt := timeNow().UTC()
	checkCtx, cancel := context.WithTimeout(ctx, j.timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(checkCtx, http.MethodGet, checkTarget, nil)
	if err != nil {
		return managedCheckResult{
			observedAt: observedAt,
			isSuccess:  false,
			message:    err.Error(),
			payload:    managedCheckPayload("http"),
		}
	}

	startedAt := time.Now()
	response, err := j.httpClient.Do(request)
	elapsed := time.Since(startedAt)
	if err != nil {
		return managedCheckResult{
			observedAt:     observedAt,
			isSuccess:      false,
			responseTimeMs: durationToMillisecondsPtr(elapsed),
			message:        err.Error(),
			payload:        managedCheckPayload("http"),
		}
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)

	statusCode := int32(response.StatusCode)
	return managedCheckResult{
		observedAt:     observedAt,
		isSuccess:      response.StatusCode >= 200 && response.StatusCode < 400,
		statusCode:     &statusCode,
		responseTimeMs: durationToMillisecondsPtr(elapsed),
		message:        response.Status,
		payload:        managedCheckPayload("http"),
	}
}

func (j *ManagedServiceCheckJob) executeTCPCheck(ctx context.Context, checkTarget string) managedCheckResult {
	observedAt := timeNow().UTC()
	address, err := normalizeTCPAddress(checkTarget)
	if err != nil {
		return managedCheckResult{
			observedAt: observedAt,
			isSuccess:  false,
			message:    err.Error(),
			payload:    managedCheckPayload("tcp"),
		}
	}

	checkCtx, cancel := context.WithTimeout(ctx, j.timeout)
	defer cancel()

	startedAt := time.Now()
	conn, err := j.tcpDialer.DialContext(checkCtx, "tcp", address)
	elapsed := time.Since(startedAt)
	if err != nil {
		return managedCheckResult{
			observedAt:     observedAt,
			isSuccess:      false,
			responseTimeMs: durationToMillisecondsPtr(elapsed),
			message:        err.Error(),
			payload:        managedCheckPayload("tcp"),
		}
	}
	_ = conn.Close()

	return managedCheckResult{
		observedAt:     observedAt,
		isSuccess:      true,
		responseTimeMs: durationToMillisecondsPtr(elapsed),
		message:        "tcp connection succeeded",
		payload:        managedCheckPayload("tcp"),
	}
}

func normalizeTCPAddress(target string) (string, error) {
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

func managedCheckPayload(checkType string) json.RawMessage {
	return json.RawMessage(`{"runner":"managed","check_type":"` + checkType + `"}`)
}
