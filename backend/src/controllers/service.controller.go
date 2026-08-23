package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
	"github.com/go-chi/chi/v5"
)

type serviceWriter interface {
	Create(ctx context.Context, input types.CreateServiceInput) (generated.Service, error)
	Update(ctx context.Context, serviceID int64, input types.UpdateServiceInput) (generated.Service, error)
	Delete(ctx context.Context, serviceID int64) error
}

type serviceReader interface {
	GetByID(ctx context.Context, serviceID int64) (types.ServiceDetailData, error)
	GetStreamSnapshot(ctx context.Context, serviceID int64) (types.ServiceStreamData, error)
	ListStreamObservations(ctx context.Context, serviceID int64, afterID int64) ([]types.ServiceHistoryEntryData, error)
	ListHistory(ctx context.Context, serviceID int64, filters types.ServiceHistoryFilters) (types.ServiceHistoryPageData, error)
	SummarizeHistory(ctx context.Context, serviceID int64, historyRange types.ServiceHistoryRange) (types.ServiceHistorySummaryData, error)
}

const maxServiceHistoryRange = 31 * 24 * time.Hour

type ServiceController struct {
	serviceService     serviceWriter
	serviceReadService serviceReader
	streamInterval     time.Duration
	streamMaxDuration  time.Duration
}

func NewServiceController(serviceService serviceWriter, serviceReadService serviceReader) *ServiceController {
	return &ServiceController{
		serviceService:     serviceService,
		serviceReadService: serviceReadService,
		streamInterval:     5 * time.Second,
		streamMaxDuration:  25 * time.Second,
	}
}

// Create creates a service.
//
// @Summary      Create service
// @Description  Creates a service under a project and resolves the execution node from execution_mode.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        request  body      types.ServiceCreateRequest  true  "Service payload"
// @Success      201      {object}  types.ServiceDetailResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      409      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /services [post]
func (c *ServiceController) Create(w http.ResponseWriter, r *http.Request) {
	var request types.ServiceCreateRequest

	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service payload")
		return
	}

	service, err := c.serviceService.Create(r.Context(), types.CreateServiceInput{
		ProjectID:     request.ProjectID,
		NodeID:        request.NodeID,
		ExecutionMode: request.ExecutionMode,
		Name:          request.Name,
		CheckType:     request.CheckType,
		CheckTarget:   request.CheckTarget,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	item, err := c.serviceReadService.GetByID(r.Context(), service.ID)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]any{"service": item})
}

// Get returns a service by id.
//
// @Summary      Get service
// @Description  Returns service details for a service the current user can read.
// @Tags         inventory
// @Produce      json
// @Param        id   path      int  true  "Service ID"
// @Success      200  {object}  types.ServiceDetailResponse
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /services/{id} [get]
func (c *ServiceController) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	service, err := c.serviceReadService.GetByID(r.Context(), id)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"service": service})
}

// History returns historical observations for a service.
//
// @Summary      Get service check history
// @Description  Returns append-only HTTP or TCP check observations newest-first. Use next_cursor as before for keyset pagination.
// @Tags         inventory
// @Produce      json
// @Param        id      path   int     true   "Service ID"
// @Param        limit   query  int     false  "Number of observations (default 100, maximum 500)"
// @Param        before  query  string  false  "Opaque cursor returned as next_cursor"
// @Param        from    query  string  false  "Inclusive range start (RFC3339; requires to)"
// @Param        to      query  string  false  "Inclusive range end (RFC3339; requires from)"
// @Success      200  {object}  types.ServiceHistoryResponse
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /services/{id}/history [get]
func (c *ServiceController) History(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	limit := services.DefaultServiceHistoryLimit
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, parseErr := strconv.ParseInt(rawLimit, 10, 32)
		if parseErr != nil || parsedLimit <= 0 || parsedLimit > int64(services.MaxServiceHistoryLimit) {
			utils.WriteError(w, http.StatusBadRequest, "limit must be between 1 and 500")
			return
		}
		limit = int32(parsedLimit)
	}

	var before *types.ServiceHistoryCursor
	if rawBefore := r.URL.Query().Get("before"); rawBefore != "" {
		before, err = decodeServiceHistoryCursor(rawBefore)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "before must be a valid history cursor")
			return
		}
	}

	historyRange, hasRange, err := parseServiceHistoryRange(r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	page, err := c.serviceReadService.ListHistory(r.Context(), id, types.ServiceHistoryFilters{
		Limit:  limit,
		Before: before,
		From:   timePointerIf(hasRange, historyRange.From),
		To:     timePointerIf(hasRange, historyRange.To),
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
		encoded, encodeErr := encodeServiceHistoryCursor(*page.NextCursor)
		if encodeErr != nil {
			utils.WriteError(w, http.StatusInternalServerError, "encode service history cursor")
			return
		}
		nextCursor = &encoded
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"history": page.Entries,
		"pagination": map[string]any{
			"limit":       limit,
			"has_more":    nextCursor != nil,
			"next_cursor": nextCursor,
		},
	})
}

// HistorySummary returns authoritative aggregates for a bounded service-history range.
//
// @Summary      Summarize service check history
// @Description  Computes full-range check counts, uptime, average successful-check latency, and latest observation directly in PostgreSQL.
// @Tags         inventory
// @Produce      json
// @Param        id    path   int     true  "Service ID"
// @Param        from  query  string  true  "Inclusive range start (RFC3339)"
// @Param        to    query  string  true  "Inclusive range end (RFC3339, not in the future)"
// @Success      200  {object}  types.ServiceHistorySummaryData
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /services/{id}/history/summary [get]
func (c *ServiceController) HistorySummary(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	historyRange, hasRange, err := parseServiceHistoryRange(r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !hasRange {
		utils.WriteError(w, http.StatusBadRequest, "from and to are required")
		return
	}

	summary, err := c.serviceReadService.SummarizeHistory(r.Context(), id, historyRange)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, summary)
}

func parseServiceHistoryRange(r *http.Request) (types.ServiceHistoryRange, bool, error) {
	rawFrom := r.URL.Query().Get("from")
	rawTo := r.URL.Query().Get("to")
	if rawFrom == "" && rawTo == "" {
		return types.ServiceHistoryRange{}, false, nil
	}
	if rawFrom == "" || rawTo == "" {
		return types.ServiceHistoryRange{}, false, fmt.Errorf("from and to must be provided together")
	}

	from, err := time.Parse(time.RFC3339Nano, rawFrom)
	if err != nil {
		return types.ServiceHistoryRange{}, false, fmt.Errorf("from must be an RFC3339 timestamp")
	}
	to, err := time.Parse(time.RFC3339Nano, rawTo)
	if err != nil {
		return types.ServiceHistoryRange{}, false, fmt.Errorf("to must be an RFC3339 timestamp")
	}
	from, to = from.UTC(), to.UTC()
	if from.After(to) {
		return types.ServiceHistoryRange{}, false, fmt.Errorf("from must be before or equal to to")
	}
	if to.After(time.Now().UTC()) {
		return types.ServiceHistoryRange{}, false, fmt.Errorf("to must not be in the future")
	}
	if to.Sub(from) > maxServiceHistoryRange {
		return types.ServiceHistoryRange{}, false, fmt.Errorf("history range must not exceed 31 days")
	}

	return types.ServiceHistoryRange{From: from, To: to}, true, nil
}

func timePointerIf(ok bool, value time.Time) *time.Time {
	if !ok {
		return nil
	}
	return &value
}

type serviceHistoryCursorPayload struct {
	ObservedAt time.Time `json:"observed_at"`
	ID         int64     `json:"id"`
}

func encodeServiceHistoryCursor(cursor types.ServiceHistoryCursor) (string, error) {
	payload, err := json.Marshal(serviceHistoryCursorPayload(cursor))
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeServiceHistoryCursor(encoded string) (*types.ServiceHistoryCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var cursor serviceHistoryCursorPayload
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, err
	}
	if cursor.ID <= 0 || cursor.ObservedAt.IsZero() {
		return nil, strconv.ErrSyntax
	}

	return &types.ServiceHistoryCursor{ObservedAt: cursor.ObservedAt.UTC(), ID: cursor.ID}, nil
}

// Stream streams service detail snapshots over SSE.
//
// @Summary      Stream service detail
// @Description  Streams the current service detail payload over Server-Sent Events for a service the current user can read.
// @Tags         inventory
// @Produce      text/event-stream
// @Param        id   path      int  true  "Service ID"
// @Success      200
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /services/{id}/stream [get]
func (c *ServiceController) Stream(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	streamServiceSSESnapshots(w, r, c.streamMaxDuration, c.streamInterval, func(ctx context.Context) (types.ServiceStreamData, error) {
		snapshot, err := c.serviceReadService.GetStreamSnapshot(ctx, id)
		if err != nil {
			return types.ServiceStreamData{}, err
		}

		return snapshot, nil
	}, func(ctx context.Context, afterID int64) (types.ServiceDetailData, []types.ServiceHistoryEntryData, error) {
		service, err := c.serviceReadService.GetByID(ctx, id)
		if err != nil {
			return types.ServiceDetailData{}, nil, err
		}
		observations, err := c.serviceReadService.ListStreamObservations(ctx, id, afterID)
		return service, observations, err
	})
}

// Update updates a service.
//
// @Summary      Update service
// @Description  Updates the editable service definition fields only. Execution mode and attached node are unchanged.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        id       path      int                         true  "Service ID"
// @Param        request  body      types.ServiceUpdateRequest  true  "Service update payload"
// @Success      200      {object}  types.ServiceDetailResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      401      {object}  types.ErrorResponse
// @Failure      403      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /services/{id} [patch]
func (c *ServiceController) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	var request types.ServiceUpdateRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service update payload")
		return
	}

	service, err := c.serviceService.Update(r.Context(), id, types.UpdateServiceInput{
		Name:        request.Name,
		CheckType:   request.CheckType,
		CheckTarget: request.CheckTarget,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	item, err := c.serviceReadService.GetByID(r.Context(), service.ID)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"service": item})
}

// Delete deletes a service.
//
// @Summary      Delete service
// @Description  Deletes a service and relies on existing schema cleanup for dependent operational records.
// @Tags         inventory
// @Produce      json
// @Param        id  path      int  true  "Service ID"
// @Success      204
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /services/{id} [delete]
func (c *ServiceController) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	if err := c.serviceService.Delete(r.Context(), id); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
