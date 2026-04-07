package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	agentmiddleware "github.com/MariusBobitiu/agrafa-backend/src/middleware"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type heartbeatIngester interface {
	Ingest(ctx context.Context, input types.HeartbeatInput) (generated.Node, error)
}

type nodeShutdowner interface {
	MarkOfflineFromShutdown(ctx context.Context, nodeID int64, occurredAt time.Time, reason string, payload json.RawMessage) (generated.Node, bool, error)
}

type healthIngester interface {
	Ingest(ctx context.Context, input types.HealthCheckInput) (generated.Service, error)
}

type metricIngester interface {
	Ingest(ctx context.Context, input types.MetricIngestionInput) error
}

type agentConfigGetter interface {
	GetForNode(ctx context.Context, node generated.Node) (types.AgentConfigData, error)
}

type AgentController struct {
	heartbeatService       heartbeatIngester
	nodeStateService       nodeShutdowner
	healthIngestionService healthIngester
	metricIngestionService metricIngester
	agentConfigService     agentConfigGetter
}

func NewAgentController(
	heartbeatService heartbeatIngester,
	nodeStateService nodeShutdowner,
	healthIngestionService healthIngester,
	metricIngestionService metricIngester,
	agentConfigService agentConfigGetter,
) *AgentController {
	return &AgentController{
		heartbeatService:       heartbeatService,
		nodeStateService:       nodeStateService,
		healthIngestionService: healthIngestionService,
		metricIngestionService: metricIngestionService,
		agentConfigService:     agentConfigService,
	}
}

// GetConfig returns backend-driven health check config for the authenticated node.
//
// @Summary      Get agent config
// @Description  Returns the authenticated node summary and the node's assigned agent-executed health checks.
// @Tags         agent
// @Produce      json
// @Param        X-Agent-Token  header    string  true  "Per-node agent token"
// @Success      200            {object}  types.AgentConfigResponseDocument
// @Failure      401            {object}  types.ErrorResponse
// @Failure      500            {object}  types.ErrorResponse
// @Router       /agent/config [get]
func (c *AgentController) GetConfig(w http.ResponseWriter, r *http.Request) {
	authenticatedNode, ok := agentmiddleware.AuthenticatedNode(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusInternalServerError, "authenticated agent node missing from context")
		return
	}

	config, err := c.agentConfigService.GetForNode(r.Context(), authenticatedNode)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, config)
}

// IngestHeartbeat ingests a node heartbeat.
//
// @Summary      Ingest heartbeat
// @Description  Records a heartbeat for a node and updates node state when needed.
// @Tags         agent
// @Accept       json
// @Produce      json
// @Param        X-Agent-Token  header    string                   true  "Per-node agent token"
// @Param        request  body      types.HeartbeatRequest   true  "Heartbeat payload"
// @Success      200      {object}  types.HeartbeatResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      401      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /agent/heartbeat [post]
func (c *AgentController) IngestHeartbeat(w http.ResponseWriter, r *http.Request) {
	var request types.HeartbeatRequest

	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid heartbeat payload")
		return
	}

	authenticatedNode, ok := agentmiddleware.AuthenticatedNode(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusInternalServerError, "authenticated agent node missing from context")
		return
	}

	observedAt := time.Now().UTC()
	if request.ObservedAt != nil {
		observedAt = request.ObservedAt.UTC()
	}

	payload, err := utils.MarshalPayloadMap(request.Payload)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid heartbeat payload")
		return
	}

	var reportedNodeID *int64
	if request.NodeID != nil && *request.NodeID > 0 {
		reportedNodeID = request.NodeID
	}

	node, err := c.heartbeatService.Ingest(r.Context(), types.HeartbeatInput{
		AuthenticatedNodeID: authenticatedNode.ID,
		ReportedNodeID:      reportedNodeID,
		ObservedAt:          observedAt,
		Source:              request.Source,
		Payload:             payload,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"node":   services.MapNodeResponse(node),
	})
}

// IngestShutdown records an agent shutdown signal and marks the node offline immediately.
//
// @Summary      Ingest shutdown
// @Description  Records an agent shutdown signal so the authenticated node can be marked offline without waiting for heartbeat expiry.
// @Tags         agent
// @Accept       json
// @Produce      json
// @Param        X-Agent-Token  header    string                     true  "Per-node agent token"
// @Param        request        body      types.AgentShutdownRequest true  "Shutdown payload"
// @Success      200            {object}  types.HeartbeatResponse
// @Failure      400            {object}  types.ErrorResponse
// @Failure      401            {object}  types.ErrorResponse
// @Failure      500            {object}  types.ErrorResponse
// @Router       /agent/shutdown [post]
func (c *AgentController) IngestShutdown(w http.ResponseWriter, r *http.Request) {
	var request types.AgentShutdownRequest

	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid shutdown payload")
		return
	}

	authenticatedNode, ok := agentmiddleware.AuthenticatedNode(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusInternalServerError, "authenticated agent node missing from context")
		return
	}

	observedAt := time.Now().UTC()
	if request.ObservedAt != nil {
		observedAt = request.ObservedAt.UTC()
	}

	payload, err := utils.MarshalPayloadMap(request.Payload)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid shutdown payload")
		return
	}

	var reportedNodeID *int64
	if request.NodeID != nil && *request.NodeID > 0 {
		reportedNodeID = request.NodeID
	}

	if reportedNodeID != nil && *reportedNodeID != authenticatedNode.ID {
		utils.WriteError(w, http.StatusBadRequest, types.ErrAgentNodeMismatch.Error())
		return
	}

	node, _, err := c.nodeStateService.MarkOfflineFromShutdown(
		r.Context(),
		authenticatedNode.ID,
		observedAt,
		request.Reason,
		payload,
	)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"node":   services.MapNodeResponse(node),
	})
}

// IngestHealth ingests a service health check result.
//
// @Summary      Ingest health result
// @Description  Records a health check result for a service and updates service state when needed.
// @Tags         agent
// @Accept       json
// @Produce      json
// @Param        X-Agent-Token  header    string                true  "Per-node agent token"
// @Param        request  body      types.HealthRequest   true  "Health payload"
// @Success      200      {object}  types.HealthResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      401      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /agent/health [post]
func (c *AgentController) IngestHealth(w http.ResponseWriter, r *http.Request) {
	var request types.HealthRequest

	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid health payload")
		return
	}

	if request.ServiceID <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "service_id is required")
		return
	}

	if request.IsSuccess == nil {
		utils.WriteError(w, http.StatusBadRequest, "is_success is required")
		return
	}

	authenticatedNode, ok := agentmiddleware.AuthenticatedNode(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusInternalServerError, "authenticated agent node missing from context")
		return
	}

	observedAt := time.Now().UTC()
	if request.ObservedAt != nil {
		observedAt = request.ObservedAt.UTC()
	}

	payload, err := utils.MarshalPayloadMap(request.Payload)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid health payload")
		return
	}

	service, err := c.healthIngestionService.Ingest(r.Context(), types.HealthCheckInput{
		AuthenticatedNodeID: authenticatedNode.ID,
		ServiceID:           request.ServiceID,
		ObservedAt:          observedAt,
		IsSuccess:           *request.IsSuccess,
		StatusCode:          request.StatusCode,
		ResponseTimeMs:      request.ResponseTimeMs,
		Message:             request.Message,
		Payload:             payload,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": services.MapServiceResponse(service),
	})
}

// IngestMetrics ingests metric samples for a node or service.
//
// @Summary      Ingest metrics
// @Description  Records one or more metric samples associated with a node and optionally a service.
// @Tags         agent
// @Accept       json
// @Produce      json
// @Param        X-Agent-Token  header    string                 true  "Per-node agent token"
// @Param        request  body      types.MetricsRequest   true  "Metrics payload"
// @Success      200      {object}  types.MetricsResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      401      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /agent/metrics [post]
func (c *AgentController) IngestMetrics(w http.ResponseWriter, r *http.Request) {
	var request types.MetricsRequest

	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid metrics payload")
		return
	}

	if len(request.Samples) == 0 {
		utils.WriteError(w, http.StatusBadRequest, "at least one metric sample is required")
		return
	}

	authenticatedNode, ok := agentmiddleware.AuthenticatedNode(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusInternalServerError, "authenticated agent node missing from context")
		return
	}

	samples := make([]types.MetricSampleInput, 0, len(request.Samples))

	for index := range request.Samples {
		if request.Samples[index].MetricName == "" {
			utils.WriteError(w, http.StatusBadRequest, "metric_name is required for every sample")
			return
		}

		metricValue := request.Samples[index].MetricValue
		if metricValue == nil {
			metricValue = request.Samples[index].Value
		}

		if metricValue == nil {
			utils.WriteError(w, http.StatusBadRequest, "value or metric_value is required for every sample")
			return
		}

		observedAt := request.Samples[index].ObservedAt
		if observedAt.IsZero() {
			observedAt = time.Now().UTC()
		} else {
			observedAt = observedAt.UTC()
		}

		payload, err := utils.MarshalPayloadMap(request.Samples[index].Payload)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid metrics payload")
			return
		}

		samples = append(samples, types.MetricSampleInput{
			MetricName:  request.Samples[index].MetricName,
			MetricValue: *metricValue,
			MetricUnit:  request.Samples[index].MetricUnit,
			ObservedAt:  observedAt,
			Payload:     payload,
		})
	}

	var reportedNodeID *int64
	if request.NodeID != nil && *request.NodeID > 0 {
		reportedNodeID = request.NodeID
	}

	if err := c.metricIngestionService.Ingest(r.Context(), types.MetricIngestionInput{
		AuthenticatedNodeID: authenticatedNode.ID,
		ReportedNodeID:      reportedNodeID,
		ServiceID:           request.ServiceID,
		Samples:             samples,
	}); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
