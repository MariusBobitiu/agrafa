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

type staticNodeReader struct {
	node  types.NodeReadData
	err   error
	calls int
}

func (r *staticNodeReader) GetByID(_ context.Context, _ int64) (types.NodeReadData, error) {
	r.calls++
	return r.node, r.err
}

type cancelOnFlushRecorder struct {
	*httptest.ResponseRecorder
	cancel   context.CancelFunc
	flushes  int
	cancelAt int
}

func (r *cancelOnFlushRecorder) Flush() {
	r.flushes++
	if r.cancel != nil && r.flushes >= r.cancelAt {
		r.cancel()
		r.cancel = nil
	}
}

func TestNodeControllerStreamSendsInitialSnapshot(t *testing.T) {
	t.Parallel()

	controller := NewNodeController(nil, &staticNodeReader{
		node: types.NodeReadData{
			ID:               31,
			ProjectID:        12,
			Name:             "node-31",
			Identifier:       "agent-node-31",
			CurrentState:     types.NodeStateOnline,
			Metadata:         json.RawMessage(`{"region":"eu-west"}`),
			ActiveAlertCount: 2,
			ServiceCount:     4,
			CreatedAt:        time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC),
			UpdatedAt:        time.Date(2026, 4, 19, 10, 1, 0, 0, time.UTC),
		},
	})
	controller.streamInterval = 10 * time.Millisecond
	controller.streamMaxDuration = 20 * time.Millisecond

	request := httptest.NewRequest(http.MethodGet, "/v1/nodes/31/stream", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "31")
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
	if !strings.Contains(body, `data: {"node":{"id":31`) {
		t.Fatalf("expected initial snapshot in stream body: %s", body)
	}
}

func TestNodeControllerStreamStopsCleanlyOnDisconnect(t *testing.T) {
	t.Parallel()

	reader := &staticNodeReader{
		node: types.NodeReadData{
			ID:               31,
			ProjectID:        12,
			Name:             "node-31",
			Identifier:       "agent-node-31",
			CurrentState:     types.NodeStateOnline,
			Metadata:         json.RawMessage(`{}`),
			ActiveAlertCount: 0,
			ServiceCount:     1,
			CreatedAt:        time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC),
			UpdatedAt:        time.Date(2026, 4, 19, 10, 1, 0, 0, time.UTC),
		},
	}
	controller := NewNodeController(nil, reader)
	controller.streamInterval = time.Millisecond
	controller.streamMaxDuration = 50 * time.Millisecond

	baseRequest := httptest.NewRequest(http.MethodGet, "/v1/nodes/31/stream", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "31")
	ctx, cancel := context.WithCancel(baseRequest.Context())
	request := baseRequest.WithContext(context.WithValue(ctx, chi.RouteCtxKey, routeContext))
	recorder := &cancelOnFlushRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		cancel:           cancel,
		cancelAt:         1,
	}

	controller.Stream(recorder, request)

	if reader.calls != 1 {
		t.Fatalf("reader calls = %d, want 1", reader.calls)
	}
}
