package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/go-chi/chi/v5"
)

type staticServiceReader struct {
	service            types.ServiceDetailData
	observation        *types.ServiceHistoryEntryData
	streamObservations []types.ServiceHistoryEntryData
	streamWatermarks   []int64
	history            types.ServiceHistoryPageData
	err                error
	calls              int
	filters            types.ServiceHistoryFilters
	summary            types.ServiceHistorySummaryData
	historyRange       types.ServiceHistoryRange
}

func (r *staticServiceReader) SummarizeHistory(_ context.Context, _ int64, historyRange types.ServiceHistoryRange) (types.ServiceHistorySummaryData, error) {
	r.calls++
	r.historyRange = historyRange
	return r.summary, r.err
}

func (r *staticServiceReader) ListStreamObservations(_ context.Context, _ int64, afterID int64) ([]types.ServiceHistoryEntryData, error) {
	r.calls++
	r.streamWatermarks = append(r.streamWatermarks, afterID)
	observations := make([]types.ServiceHistoryEntryData, 0, len(r.streamObservations))
	for _, observation := range r.streamObservations {
		if observation.ID > afterID {
			observations = append(observations, observation)
		}
	}
	return observations, r.err
}

func (r *staticServiceReader) GetStreamSnapshot(_ context.Context, _ int64) (types.ServiceStreamData, error) {
	r.calls++
	return types.ServiceStreamData{Service: r.service, Observation: r.observation}, r.err
}

func (r *staticServiceReader) GetByID(_ context.Context, _ int64) (types.ServiceDetailData, error) {
	r.calls++
	return r.service, r.err
}

func (r *staticServiceReader) ListHistory(_ context.Context, _ int64, filters types.ServiceHistoryFilters) (types.ServiceHistoryPageData, error) {
	r.calls++
	r.filters = filters
	return r.history, r.err
}

func TestServiceControllerHistoryReturnsBoundedPageAndCursor(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	reader := &staticServiceReader{history: types.ServiceHistoryPageData{
		Entries: []types.ServiceHistoryEntryData{{
			ID: 9, ServiceID: 21, NodeID: 4, CheckType: "tcp", Source: "managed",
			ObservedAt: observedAt, IsSuccess: true, ResponseTimeMs: int32Pointer(18), Message: "ok", Metadata: map[string]any{"runner": "managed"},
		}},
		NextCursor: &types.ServiceHistoryCursor{ObservedAt: observedAt, ID: 9},
	}}
	controller := NewServiceController(nil, reader)

	request := httptest.NewRequest(http.MethodGet, "/v1/services/21/history?limit=25", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "21")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	recorder := httptest.NewRecorder()

	controller.History(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if reader.filters.Limit != 25 || reader.filters.Before != nil {
		t.Fatalf("filters = %#v, want limit 25 without cursor", reader.filters)
	}
	var response struct {
		History    []types.ServiceHistoryEntryData `json:"history"`
		Pagination struct {
			Limit      int32   `json:"limit"`
			HasMore    bool    `json:"has_more"`
			NextCursor *string `json:"next_cursor"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.History) != 1 || response.History[0].StatusCode != nil || response.History[0].ResponseTimeMs == nil || *response.History[0].ResponseTimeMs != 18 {
		t.Fatalf("unexpected history response: %#v", response.History)
	}
	if response.Pagination.Limit != 25 || !response.Pagination.HasMore || response.Pagination.NextCursor == nil {
		t.Fatalf("unexpected pagination: %#v", response.Pagination)
	}
	decoded, err := decodeServiceHistoryCursor(*response.Pagination.NextCursor)
	if err != nil || decoded.ID != 9 || !decoded.ObservedAt.Equal(observedAt) {
		t.Fatalf("decoded cursor = %#v, err = %v", decoded, err)
	}
}

func TestServiceControllerHistoryRejectsUnboundedLimit(t *testing.T) {
	t.Parallel()

	reader := &staticServiceReader{}
	controller := NewServiceController(nil, reader)
	request := httptest.NewRequest(http.MethodGet, "/v1/services/21/history?limit=501", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "21")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	recorder := httptest.NewRecorder()

	controller.History(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if reader.calls != 0 {
		t.Fatalf("history reader calls = %d, want 0", reader.calls)
	}
}

func TestServiceControllerHistoryPassesInclusiveRange(t *testing.T) {
	t.Parallel()

	to := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	from := to.Add(-time.Hour)
	reader := &staticServiceReader{}
	controller := NewServiceController(nil, reader)
	request := httptest.NewRequest(http.MethodGet, "/v1/services/21/history?from="+from.Format(time.RFC3339)+"&to="+to.Format(time.RFC3339), nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "21")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	recorder := httptest.NewRecorder()

	controller.History(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if reader.filters.From == nil || reader.filters.To == nil || !reader.filters.From.Equal(from) || !reader.filters.To.Equal(to) {
		t.Fatalf("range filters = %#v", reader.filters)
	}
}

func TestServiceControllerHistorySummaryReturnsAuthoritativeRange(t *testing.T) {
	t.Parallel()

	to := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	from := to.Add(-24 * time.Hour)
	uptime := 80.0
	average := 12.5
	reader := &staticServiceReader{summary: types.ServiceHistorySummaryData{
		From: from, To: to, TotalChecks: 2505, SuccessfulChecks: 2004,
		UptimePercent: &uptime, AverageLatencyMs: &average, LastCheckedAt: &to,
	}}
	controller := NewServiceController(nil, reader)
	request := httptest.NewRequest(http.MethodGet, "/v1/services/21/history/summary?from="+from.Format(time.RFC3339)+"&to="+to.Format(time.RFC3339), nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "21")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	recorder := httptest.NewRecorder()

	controller.HistorySummary(recorder, request)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"total_checks":2505`) {
		t.Fatalf("unexpected response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if reader.historyRange.From != from || reader.historyRange.To != to {
		t.Fatalf("summary range = %#v", reader.historyRange)
	}
}

func TestServiceControllerHistoryRejectsFutureAndUnreasonableRanges(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"from=" + time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano) + "&to=" + time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano),
		"from=" + time.Now().UTC().Add(-32*24*time.Hour).Format(time.RFC3339Nano) + "&to=" + time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano),
		"from=not-a-time&to=also-not-a-time",
	}
	for _, query := range testCases {
		reader := &staticServiceReader{}
		controller := NewServiceController(nil, reader)
		request := httptest.NewRequest(http.MethodGet, "/v1/services/21/history?"+query, nil)
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("id", "21")
		request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
		recorder := httptest.NewRecorder()

		controller.History(recorder, request)
		if recorder.Code != http.StatusBadRequest || reader.calls != 0 {
			t.Fatalf("query %q: status=%d calls=%d", query, recorder.Code, reader.calls)
		}
	}
}

func int32Pointer(value int32) *int32 {
	return &value
}

func TestServiceControllerStreamSendsInitialSnapshot(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, 4, 19, 10, 2, 0, 0, time.UTC)
	controller := NewServiceController(nil, &staticServiceReader{
		service: types.ServiceDetailData{
			ID:                  21,
			ProjectID:           8,
			NodeID:              31,
			ExecutionMode:       types.ExecutionModeAgent,
			Name:                "api",
			CheckType:           "http",
			CheckTarget:         "https://example.com/health",
			Status:              types.ServiceStateHealthy,
			ConsecutiveFailures: 0,
			ActiveAlertCount:    1,
			ActiveAlerts:        []types.ServiceActiveAlertData{},
			LatestHealthCheck: &types.HealthCheckSummaryData{
				ObservedAt:     observedAt,
				IsSuccess:      true,
				StatusCode:     nil,
				ResponseTimeMs: nil,
				Message:        "ok",
			},
		},
		observation: &types.ServiceHistoryEntryData{
			ID: 42, ServiceID: 21, NodeID: 31, CheckType: "http", Source: "agent",
			ObservedAt: observedAt, IsSuccess: true, ResponseTimeMs: nil, Message: "ok", Metadata: map[string]any{"runner": "agent"},
		},
	})
	controller.streamInterval = 10 * time.Millisecond
	controller.streamMaxDuration = 20 * time.Millisecond

	request := httptest.NewRequest(http.MethodGet, "/v1/services/21/stream", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "21")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	recorder := httptest.NewRecorder()

	controller.Stream(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content-type = %q, want %q", got, "text/event-stream")
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "retry: 5000") {
		t.Fatalf("expected retry header in stream body: %s", body)
	}
	if !strings.Contains(body, `data: {"service":{"id":21`) {
		t.Fatalf("expected initial snapshot in stream body: %s", body)
	}
	if !strings.Contains(body, `"observation":{"id":42,"service_id":21,"node_id":31,"check_type":"http"`) {
		t.Fatalf("expected persisted observation in stream body: %s", body)
	}
	if !strings.Contains(body, `"response_time_ms":null`) {
		t.Fatalf("expected nullable latency in stream body: %s", body)
	}
}

func TestServiceControllerStreamDeliversEveryUnseenObservationOnceByPersistedID(t *testing.T) {
	t.Parallel()

	initialObservedAt := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	reader := &staticServiceReader{
		service: types.ServiceDetailData{ID: 21},
		observation: &types.ServiceHistoryEntryData{
			ID: 10, ServiceID: 21, CheckType: "http", ObservedAt: initialObservedAt,
		},
		streamObservations: []types.ServiceHistoryEntryData{
			{ID: 11, ServiceID: 21, CheckType: "http", ObservedAt: initialObservedAt.Add(time.Minute)},
			{ID: 12, ServiceID: 21, CheckType: "http", ObservedAt: initialObservedAt.Add(-time.Hour)},
		},
	}
	controller := NewServiceController(nil, reader)
	controller.streamInterval = 5 * time.Millisecond
	controller.streamMaxDuration = 24 * time.Millisecond

	request := httptest.NewRequest(http.MethodGet, "/v1/services/21/stream", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "21")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	recorder := httptest.NewRecorder()

	controller.Stream(recorder, request)

	body := recorder.Body.String()
	id11 := `"observation":{"id":11`
	id12 := `"observation":{"id":12`
	if strings.Count(body, id11) != 1 || strings.Count(body, id12) != 1 {
		t.Fatalf("unseen observation delivery counts are wrong: %s", body)
	}
	if strings.Index(body, id11) > strings.Index(body, id12) {
		t.Fatalf("observations were not delivered in persisted ID order: %s", body)
	}
	if len(reader.streamWatermarks) < 2 || reader.streamWatermarks[0] != 10 || reader.streamWatermarks[1] != 12 {
		t.Fatalf("stream watermarks = %#v, want first 10 then 12", reader.streamWatermarks)
	}
}
