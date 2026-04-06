package health

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MariusBobitiu/agrafa-agent/src/types"
	"github.com/MariusBobitiu/agrafa-agent/src/utils"
)

type HTTPChecker struct {
	client         *http.Client
	defaultTimeout time.Duration
}

func NewHTTPChecker(defaultTimeout time.Duration) *HTTPChecker {
	return &HTTPChecker{
		client:         &http.Client{},
		defaultTimeout: defaultTimeout,
	}
}

func (checker *HTTPChecker) Run(ctx context.Context, check types.HealthCheck) types.HTTPCheckResult {
	observedAt := utils.NowUTC()
	timeout := checker.defaultTimeout
	if check.TimeoutSeconds > 0 {
		timeout = time.Duration(check.TimeoutSeconds) * time.Second
	}

	requestContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, check.Target, nil)
	if err != nil {
		return types.HTTPCheckResult{
			ServiceID:      check.ServiceID,
			Name:           check.Name,
			Type:           check.Type,
			Target:         check.Target,
			ObservedAt:     observedAt,
			IsSuccess:      false,
			ResponseTimeMs: utils.DurationMillisecondsInt32(time.Since(startedAt)),
			Message:        err.Error(),
		}
	}

	response, err := checker.client.Do(request)
	if err != nil {
		return types.HTTPCheckResult{
			ServiceID:      check.ServiceID,
			Name:           check.Name,
			Type:           check.Type,
			Target:         check.Target,
			ObservedAt:     observedAt,
			IsSuccess:      false,
			ResponseTimeMs: utils.DurationMillisecondsInt32(time.Since(startedAt)),
			Message:        err.Error(),
		}
	}
	defer response.Body.Close()

	io.Copy(io.Discard, io.LimitReader(response.Body, 1024))

	statusCode := int32(response.StatusCode)
	isSuccess := response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusBadRequest
	message := "ok"
	if !isSuccess {
		message = fmt.Sprintf("unexpected status: %d", response.StatusCode)
	}

	return types.HTTPCheckResult{
		ServiceID:      check.ServiceID,
		Name:           check.Name,
		Type:           check.Type,
		Target:         check.Target,
		ObservedAt:     observedAt,
		IsSuccess:      isSuccess,
		StatusCode:     &statusCode,
		ResponseTimeMs: utils.DurationMillisecondsInt32(time.Since(startedAt)),
		Message:        message,
	}
}

func BuildHealthRequest(result types.HTTPCheckResult) types.HealthRequest {
	return types.HealthRequest{
		ServiceID:      result.ServiceID,
		ObservedAt:     result.ObservedAt,
		IsSuccess:      result.IsSuccess,
		StatusCode:     result.StatusCode,
		ResponseTimeMs: result.ResponseTimeMs,
		Message:        result.Message,
		Payload: map[string]any{
			"name":   result.Name,
			"target": result.Target,
			"type":   result.Type,
		},
	}
}
