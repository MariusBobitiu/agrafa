package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/MariusBobitiu/agrafa-agent/src/types"
)

type APIClient struct {
	baseURL    string
	agentToken string
	retryCount int
	httpClient *http.Client
}

type APIError struct {
	Path       string
	StatusCode int
	Body       string
	Attempts   int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request to %s failed with status %d: %s", e.Path, e.StatusCode, e.Body)
}

func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return nil, false
	}

	return apiErr, true
}

type TransportError struct {
	Path     string
	Attempts int
	Err      error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("request to %s failed after %d attempt(s): %v", e.Path, e.Attempts, e.Err)
}

func (e *TransportError) Unwrap() error {
	return e.Err
}

func AsTransportError(err error) (*TransportError, bool) {
	var transportErr *TransportError
	if !errors.As(err, &transportErr) {
		return nil, false
	}

	return transportErr, true
}

func NewAPIClient(baseURL string, agentToken string, timeout time.Duration, retryCount int) *APIClient {
	return &APIClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		agentToken: agentToken,
		retryCount: retryCount,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (client *APIClient) SendHeartbeat(ctx context.Context, request types.HeartbeatRequest) error {
	return client.doJSON(ctx, http.MethodPost, "/agent/heartbeat", request, nil)
}

func (client *APIClient) SendShutdown(ctx context.Context, request types.ShutdownRequest) error {
	return client.doJSON(ctx, http.MethodPost, "/agent/shutdown", request, nil)
}

func (client *APIClient) SendMetrics(ctx context.Context, request types.MetricsRequest) error {
	return client.doJSON(ctx, http.MethodPost, "/agent/metrics", request, nil)
}

func (client *APIClient) SendHealth(ctx context.Context, request types.HealthRequest) error {
	return client.doJSON(ctx, http.MethodPost, "/agent/health", request, nil)
}

func (client *APIClient) FetchConfig(ctx context.Context) (types.AgentConfigResponse, error) {
	var response types.AgentConfigResponse
	if err := client.doJSON(ctx, http.MethodGet, "/agent/config", nil, &response); err != nil {
		return types.AgentConfigResponse{}, err
	}

	return response, nil
}

func (client *APIClient) doJSON(ctx context.Context, method string, path string, payload any, destination any) error {
	var body []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal request for %s: %w", path, err)
		}
		body = encoded
	}

	attempts := client.retryCount + 1

	for attempt := 1; attempt <= attempts; attempt++ {
		request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("build request for %s: %w", path, err)
		}

		if payload != nil {
			request.Header.Set("Content-Type", "application/json")
		}
		request.Header.Set("Accept", "application/json")
		request.Header.Set("X-Agent-Token", client.agentToken)

		response, err := client.httpClient.Do(request)
		if err != nil {
			if attempt < attempts && isRetryableTransportError(err) {
				if sleepErr := waitForRetry(ctx); sleepErr != nil {
					return &TransportError{Path: path, Attempts: attempt, Err: err}
				}

				continue
			}

			return &TransportError{Path: path, Attempts: attempt, Err: err}
		}

		responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
		response.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read response body for %s: %w", path, readErr)
		}

		if response.StatusCode >= http.StatusInternalServerError {
			if attempt < attempts {
				if sleepErr := waitForRetry(ctx); sleepErr != nil {
					return &APIError{
						Path:       path,
						StatusCode: response.StatusCode,
						Body:       strings.TrimSpace(string(responseBody)),
						Attempts:   attempt,
					}
				}

				continue
			}

			return &APIError{
				Path:       path,
				StatusCode: response.StatusCode,
				Body:       strings.TrimSpace(string(responseBody)),
				Attempts:   attempt,
			}
		}

		if response.StatusCode >= http.StatusMultipleChoices {
			return &APIError{
				Path:       path,
				StatusCode: response.StatusCode,
				Body:       strings.TrimSpace(string(responseBody)),
				Attempts:   attempt,
			}
		}

		if destination != nil && len(responseBody) > 0 {
			if err := json.Unmarshal(responseBody, destination); err != nil {
				return fmt.Errorf("decode response body for %s: %w", path, err)
			}
		}

		return nil
	}

	return nil
}

func isRetryableTransportError(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr)
}

func waitForRetry(ctx context.Context) error {
	timer := time.NewTimer(200 * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
