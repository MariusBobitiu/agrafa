package runner

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/MariusBobitiu/agrafa-agent/src/client"
)

type authFailureTracker struct {
	mu            sync.Mutex
	logf          func(format string, args ...any)
	lastFailureID string
}

func newAuthFailureTracker(logf func(format string, args ...any)) *authFailureTracker {
	return &authFailureTracker{logf: logf}
}

func (t *authFailureTracker) HandleResult(operation string, err error) bool {
	if err == nil {
		t.mu.Lock()
		t.lastFailureID = ""
		t.mu.Unlock()
		return false
	}

	apiErr, ok := client.AsAPIError(err)
	if !ok || apiErr.StatusCode != http.StatusUnauthorized {
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	reason := unauthorizedReason(apiErr)
	failureID := apiErr.Path + "|" + reason
	if t.lastFailureID != failureID {
		t.lastFailureID = failureID
		t.logf(
			"agent authentication failed\n  operation: %s\n  path: %s\n  reason: %q\n  attempts: %d\n  action: no retry; repeated identical auth failures will be suppressed",
			operation,
			apiErr.Path,
			reason,
			apiErr.Attempts,
		)
	}

	return true
}

func unauthorizedReason(apiErr *client.APIError) string {
	if apiErr == nil {
		return "unauthorized"
	}

	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(apiErr.Body), &payload); err == nil && strings.TrimSpace(payload.Error) != "" {
		return payload.Error
	}

	if strings.TrimSpace(apiErr.Body) != "" {
		return strings.TrimSpace(apiErr.Body)
	}

	return "unauthorized"
}
