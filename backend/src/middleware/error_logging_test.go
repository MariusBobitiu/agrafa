package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func TestErrorLoggingLogsServerErrors(t *testing.T) {
	var logs bytes.Buffer
	originalWriter := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(originalWriter)

	handler := chimiddleware.RequestID(ErrorLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"create project: boom"}`))
	})))

	request := httptest.NewRequest(http.MethodPost, "/v1/projects", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	output := logs.String()
	if !strings.Contains(output, "request_id:") {
		t.Fatalf("expected request id in log output, got %q", output)
	}
	if !strings.Contains(output, "path: /v1/projects") {
		t.Fatalf("expected request path in log output, got %q", output)
	}
	if !strings.Contains(output, `error: {"error":"create project: boom"}`) {
		t.Fatalf("expected response body in log output, got %q", output)
	}
}

func TestErrorLoggingSkipsNonServerErrors(t *testing.T) {
	var logs bytes.Buffer
	originalWriter := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(originalWriter)

	handler := ErrorLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))

	request := httptest.NewRequest(http.MethodPost, "/v1/projects", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if logs.Len() != 0 {
		t.Fatalf("expected no logs for non-5xx response, got %q", logs.String())
	}
}

func TestErrorLoggingPreservesSSEStreaming(t *testing.T) {
	handler := ErrorLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"status\":\"healthy\"}\n\n"))
		flusher.Flush()
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/services/12/stream", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %q", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content-type = %q, want %q", got, "text/event-stream")
	}
	if !recorder.Flushed {
		t.Fatal("expected flush to reach the underlying response writer")
	}
	if got := recorder.Body.String(); !strings.Contains(got, `data: {"status":"healthy"}`) {
		t.Fatalf("expected SSE event in body, got %q", got)
	}
}
