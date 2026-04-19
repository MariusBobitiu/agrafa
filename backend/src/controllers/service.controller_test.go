package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/go-chi/chi/v5"
)

type staticServiceReader struct {
	service types.ServiceDetailData
	err     error
	calls   int
}

func (r *staticServiceReader) GetByID(_ context.Context, _ int64) (types.ServiceDetailData, error) {
	r.calls++
	return r.service, r.err
}

func TestServiceControllerStreamSendsInitialSnapshot(t *testing.T) {
	t.Parallel()

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
				ObservedAt:     time.Date(2026, 4, 19, 10, 2, 0, 0, time.UTC),
				IsSuccess:      true,
				StatusCode:     nil,
				ResponseTimeMs: nil,
				Message:        "ok",
			},
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
}
