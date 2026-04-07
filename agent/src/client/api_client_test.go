package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-agent/src/types"
)

func TestSendHeartbeatRetriesOnceOnServerError(t *testing.T) {
	t.Parallel()

	client := NewAPIClient("http://example.com", "token", time.Second, 1)
	var requests int32
	client.httpClient.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		attempt := atomic.AddInt32(&requests, 1)
		if attempt == 1 {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader("temporary outage\n")),
				Header:     make(http.Header),
			}, nil
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})

	err := client.SendHeartbeat(context.Background(), types.HeartbeatRequest{
		NodeID:     1,
		ObservedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}

	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Fatalf("expected 2 requests, got %d", got)
	}
}

func TestSendHeartbeatDoesNotRetryUnauthorized(t *testing.T) {
	t.Parallel()

	client := NewAPIClient("http://example.com", "token", time.Second, 2)
	var requests int32
	client.httpClient.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		atomic.AddInt32(&requests, 1)
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader("invalid agent token\n")),
			Header:     make(http.Header),
		}, nil
	})

	err := client.SendHeartbeat(context.Background(), types.HeartbeatRequest{
		NodeID:     1,
		ObservedAt: time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}

	apiErr, ok := AsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", apiErr.StatusCode)
	}

	if apiErr.Attempts != 1 {
		t.Fatalf("expected a single attempt, got %d", apiErr.Attempts)
	}

	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Fatalf("expected 1 request, got %d", got)
	}
}

func TestFetchConfigDecodesResponseAndSendsAgentToken(t *testing.T) {
	t.Parallel()

	client := NewAPIClient("http://example.com", "token", time.Second, 1)
	client.httpClient.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/agent/config" {
			t.Fatalf("path = %s, want /agent/config", r.URL.Path)
		}
		if got := r.Header.Get("X-Agent-Token"); got != "token" {
			t.Fatalf("X-Agent-Token = %q, want token", got)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{"node":{"id":12,"name":"web-01","identifier":"web-01"},"health_checks":[{"service_id":101,"name":"internal-api","check_type":"http","check_target":"http://internal-api.local/health","interval_seconds":30,"timeout_seconds":10}]}`,
			)),
			Header: make(http.Header),
		}, nil
	})

	config, err := client.FetchConfig(context.Background())
	if err != nil {
		t.Fatalf("FetchConfig() error = %v", err)
	}

	if config.Node.ID != 12 || config.Node.Identifier != "web-01" {
		t.Fatalf("unexpected node: %#v", config.Node)
	}
	if len(config.HealthChecks) != 1 {
		t.Fatalf("len(config.HealthChecks) = %d, want 1", len(config.HealthChecks))
	}
	if config.HealthChecks[0].CheckTarget != "http://internal-api.local/health" {
		t.Fatalf("unexpected health check: %#v", config.HealthChecks[0])
	}
}

func TestSendShutdownPostsToShutdownEndpoint(t *testing.T) {
	t.Parallel()

	client := NewAPIClient("http://example.com", "token", time.Second, 1)
	client.httpClient.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/agent/shutdown" {
			t.Fatalf("path = %s, want /agent/shutdown", r.URL.Path)
		}
		if got := r.Header.Get("X-Agent-Token"); got != "token" {
			t.Fatalf("X-Agent-Token = %q, want token", got)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})

	err := client.SendShutdown(context.Background(), types.ShutdownRequest{
		NodeID:     1,
		ObservedAt: time.Now().UTC(),
		Reason:     "user_closed",
		Payload: map[string]any{
			"signal": "SIGINT",
		},
	})
	if err != nil {
		t.Fatalf("SendShutdown() error = %v", err)
	}
}

func TestNewAPIClientUsesConfiguredTimeout(t *testing.T) {
	t.Parallel()

	client := NewAPIClient("http://example.com", "token", 3*time.Second, 1)
	if client.httpClient.Timeout != 3*time.Second {
		t.Fatalf("expected timeout 3s, got %s", client.httpClient.Timeout)
	}
}

func TestSendHeartbeatRetriesOnTimeout(t *testing.T) {
	t.Parallel()

	client := NewAPIClient("http://example.com", "token", time.Second, 1)

	var attempts int32
	client.httpClient.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt == 1 {
			return nil, timeoutError{message: "request timed out"}
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})

	err := client.SendHeartbeat(context.Background(), types.HeartbeatRequest{
		NodeID:     1,
		ObservedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("expected timeout retry to succeed, got %v", err)
	}

	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type timeoutError struct {
	message string
}

func (e timeoutError) Error() string {
	return e.message
}

func (e timeoutError) Timeout() bool {
	return true
}

func (e timeoutError) Temporary() bool {
	return true
}

func TestAsTransportError(t *testing.T) {
	t.Parallel()

	err := &TransportError{Path: "/agent/heartbeat", Attempts: 2, Err: timeoutError{message: "timeout"}}
	transportErr, ok := AsTransportError(err)
	if !ok {
		t.Fatal("expected transport error to unwrap")
	}

	if !errors.Is(transportErr.Err, timeoutError{message: "timeout"}) && transportErr.Path != "/agent/heartbeat" {
		t.Fatal("unexpected transport error contents")
	}
}
