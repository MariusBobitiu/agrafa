package controllers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type ReadController struct {
	nodeReadService    *services.NodeReadService
	serviceReadService *services.ServiceReadService
	eventService       *services.EventService
	alertRuleService   *services.AlertRuleService
	alertService       *services.AlertService
	overviewService    *services.OverviewService
}

func NewReadController(
	nodeReadService *services.NodeReadService,
	serviceReadService *services.ServiceReadService,
	eventService *services.EventService,
	alertRuleService *services.AlertRuleService,
	alertService *services.AlertService,
	overviewService *services.OverviewService,
) *ReadController {
	return &ReadController{
		nodeReadService:    nodeReadService,
		serviceReadService: serviceReadService,
		eventService:       eventService,
		alertRuleService:   alertRuleService,
		alertService:       alertService,
		overviewService:    overviewService,
	}
}

// ListNodes lists nodes, optionally filtered by project.
//
// @Summary      List nodes
// @Description  Returns visible nodes ordered by id. Hidden managed nodes are excluded.
// @Tags         inventory
// @Produce      json
// @Param        project_id  query     int  false  "Project ID"
// @Success      200         {object}  types.NodesResponse
// @Failure      400         {object}  types.ErrorResponse
// @Failure      500         {object}  types.ErrorResponse
// @Router       /nodes [get]
func (c *ReadController) ListNodes(w http.ResponseWriter, r *http.Request) {
	projectID, err := utils.ParseOptionalPositiveInt64Query(r, "project_id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	nodes, err := c.nodeReadService.List(r.Context(), projectID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"nodes": nodes})
}

// ListServices lists services.
//
// @Summary      List services
// @Description  Returns newest services first, with execution_mode derived from the attached node and optional project_id, node_id, status, and limit filters.
// @Tags         inventory
// @Produce      json
// @Param        project_id  query     int  false  "Project ID"
// @Param        node_id     query     int  false  "Node ID"
// @Param        status      query     string  false  "Service status"  Enums(healthy, degraded, unhealthy)
// @Param        limit       query     int  false  "Maximum number of services"  minimum(1)
// @Success      200         {object}  types.ServicesResponse
// @Failure      400         {object}  types.ErrorResponse
// @Failure      500         {object}  types.ErrorResponse
// @Router       /services [get]
func (c *ReadController) ListServices(w http.ResponseWriter, r *http.Request) {
	projectID, err := utils.ParseOptionalPositiveInt64Query(r, "project_id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	nodeID, err := utils.ParseOptionalPositiveInt64Query(r, "node_id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	var status *string
	if rawStatus := r.URL.Query().Get("status"); rawStatus != "" {
		switch rawStatus {
		case types.ServiceStateHealthy, types.ServiceStateDegraded, types.ServiceStateUnhealthy:
			status = &rawStatus
		default:
			utils.WriteError(w, http.StatusBadRequest, "status must be healthy, degraded, or unhealthy")
			return
		}
	}

	var limit *int32
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.ParseInt(rawLimit, 10, 32)
		if err != nil || parsed <= 0 {
			utils.WriteError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}

		parsedLimit := int32(parsed)
		limit = &parsedLimit
	}

	services, err := c.serviceReadService.List(r.Context(), types.ServiceListFilters{
		ProjectID: projectID,
		NodeID:    nodeID,
		Status:    status,
		Limit:     limit,
	})
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"services": services})
}

// ListEvents lists recent events.
//
// @Summary      List events
// @Description  Returns newest events first, with optional project filter and result limit.
// @Tags         events
// @Produce      json
// @Param        project_id  query     int  false  "Project ID"
// @Param        limit       query     int  false  "Maximum number of events"  minimum(1)  default(50)
// @Success      200         {object}  types.EventsResponse
// @Failure      400         {object}  types.ErrorResponse
// @Failure      500         {object}  types.ErrorResponse
// @Router       /events [get]
func (c *ReadController) ListEvents(w http.ResponseWriter, r *http.Request) {
	projectID, err := utils.ParseOptionalPositiveInt64Query(r, "project_id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	limit := int32(50)
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.ParseInt(rawLimit, 10, 32)
		if err != nil || parsed <= 0 {
			utils.WriteError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}

		limit = int32(parsed)
	}

	events, err := c.eventService.ListEvents(r.Context(), limit, projectID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"events": events})
}

// ListAlertRules lists alert rules.
//
// @Summary      List alert rules
// @Description  Returns alert rules ordered by newest first. Optionally filters by project_id.
// @Tags         alerts
// @Produce      json
// @Param        project_id  query     int  false  "Project ID"
// @Success      200         {object}  types.AlertRulesResponse
// @Failure      400         {object}  types.ErrorResponse
// @Failure      500         {object}  types.ErrorResponse
// @Router       /alert-rules [get]
func (c *ReadController) ListAlertRules(w http.ResponseWriter, r *http.Request) {
	projectID, err := utils.ParseOptionalPositiveInt64Query(r, "project_id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	rules, err := c.alertRuleService.List(r.Context(), projectID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"alert_rules": rules})
}

// ListAlerts lists alert instances.
//
// @Summary      List alerts
// @Description  Returns authoritative alert-rule and affected-resource presentation data. status=active is an independent read with no history limit; status=resolved uses keyset pagination ordered by triggered_at and id newest-first.
// @Tags         alerts
// @Produce      json
// @Param        project_id  query     int     false  "Project ID"
// @Param        status      query     string  false  "Alert status"  Enums(active, resolved)
// @Param        service_id  query     int     false  "Affected service ID"
// @Param        node_id     query     int     false  "Affected node ID"
// @Param        rule_type   query     string  false  "Alert rule type"  Enums(node_offline, service_unhealthy, cpu_above_threshold, memory_above_threshold, disk_above_threshold)
// @Param        severity    query     string  false  "Alert severity"  Enums(info, warning, critical)
// @Param        category    query     string  false  "Alert category"  Enums(node, service, metric)
// @Param        limit       query     int     false  "Resolved/combined page size (default 50, maximum 100)"  minimum(1)  maximum(100)  default(50)
// @Param        before      query     string  false  "Opaque resolved-history cursor returned as next_cursor; requires status=resolved"
// @Success      200         {object}  types.AlertsResponse
// @Failure      400         {object}  types.ErrorResponse
// @Failure      500         {object}  types.ErrorResponse
// @Router       /alerts [get]
func (c *ReadController) ListAlerts(w http.ResponseWriter, r *http.Request) {
	projectID, err := utils.ParseOptionalPositiveInt64Query(r, "project_id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	serviceID, err := utils.ParseOptionalPositiveInt64Query(r, "service_id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	nodeID, err := utils.ParseOptionalPositiveInt64Query(r, "node_id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	var status *string
	if rawStatus := r.URL.Query().Get("status"); rawStatus != "" {
		if rawStatus != types.AlertStatusActive && rawStatus != types.AlertStatusResolved {
			utils.WriteError(w, http.StatusBadRequest, "status must be active or resolved")
			return
		}

		status = &rawStatus
	}

	var ruleType *string
	if rawRuleType := r.URL.Query().Get("rule_type"); rawRuleType != "" {
		switch rawRuleType {
		case types.AlertRuleTypeNodeOffline,
			types.AlertRuleTypeServiceUnhealthy,
			types.AlertRuleTypeCPUAboveThreshold,
			types.AlertRuleTypeMemoryAboveThreshold,
			types.AlertRuleTypeDiskAboveThreshold:
			ruleType = &rawRuleType
		default:
			utils.WriteError(w, http.StatusBadRequest, "rule_type is invalid")
			return
		}
	}

	var severity *string
	if rawSeverity := r.URL.Query().Get("severity"); rawSeverity != "" {
		switch rawSeverity {
		case types.AlertSeverityInfo, types.AlertSeverityWarning, types.AlertSeverityCritical:
			severity = &rawSeverity
		default:
			utils.WriteError(w, http.StatusBadRequest, "severity must be info, warning, or critical")
			return
		}
	}

	var category *string
	if rawCategory := r.URL.Query().Get("category"); rawCategory != "" {
		switch rawCategory {
		case types.AlertCategoryNode, types.AlertCategoryService, types.AlertCategoryMetric:
			category = &rawCategory
		default:
			utils.WriteError(w, http.StatusBadRequest, "category must be node, service, or metric")
			return
		}
	}

	limit := services.DefaultAlertHistoryLimit
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.ParseInt(rawLimit, 10, 32)
		if err != nil || parsed <= 0 || parsed > int64(services.MaxAlertHistoryLimit) {
			utils.WriteError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}

		limit = int32(parsed)
	}

	var before *types.AlertCursor
	if rawBefore := r.URL.Query().Get("before"); rawBefore != "" {
		if status == nil || *status != types.AlertStatusResolved {
			utils.WriteError(w, http.StatusBadRequest, "before requires status=resolved")
			return
		}
		before, err = decodeAlertCursor(rawBefore)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "before must be a valid alert cursor")
			return
		}
	}

	page, err := c.alertService.List(r.Context(), types.AlertListFilters{
		ProjectID: projectID,
		Status:    status,
		ServiceID: serviceID,
		NodeID:    nodeID,
		RuleType:  ruleType,
		Severity:  severity,
		Category:  category,
		Limit:     limit,
		Before:    before,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var nextCursor *string
	if page.NextCursor != nil {
		encoded, encodeErr := encodeAlertCursor(*page.NextCursor)
		if encodeErr != nil {
			utils.WriteError(w, http.StatusInternalServerError, "encode alert cursor")
			return
		}
		nextCursor = &encoded
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"alerts": page.Alerts,
		"pagination": map[string]any{
			"limit":       limit,
			"has_more":    nextCursor != nil,
			"next_cursor": nextCursor,
		},
	})
}

type alertCursorPayload struct {
	TriggeredAt time.Time `json:"triggered_at"`
	ID          int64     `json:"id"`
}

func encodeAlertCursor(cursor types.AlertCursor) (string, error) {
	payload, err := json.Marshal(alertCursorPayload(cursor))
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeAlertCursor(encoded string) (*types.AlertCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var cursor alertCursorPayload
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, err
	}
	if cursor.ID <= 0 || cursor.TriggeredAt.IsZero() {
		return nil, strconv.ErrSyntax
	}
	return &types.AlertCursor{TriggeredAt: cursor.TriggeredAt.UTC(), ID: cursor.ID}, nil
}

// GetOverview returns system overview statistics.
//
// @Summary      Get overview
// @Description  Returns aggregate node and service counts plus recent events, optionally scoped to a project.
// @Tags         overview
// @Produce      json
// @Param        project_id  query     int  false  "Project ID"
// @Success      200         {object}  types.OverviewResponse
// @Failure      400         {object}  types.ErrorResponse
// @Failure      500         {object}  types.ErrorResponse
// @Router       /overview [get]
func (c *ReadController) GetOverview(w http.ResponseWriter, r *http.Request) {
	projectID, err := utils.ParseOptionalPositiveInt64Query(r, "project_id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	overview, err := c.overviewService.Get(r.Context(), projectID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, overview)
}
