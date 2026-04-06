package runner

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MariusBobitiu/agrafa-agent/src/client"
)

func TestAuthFailureTrackerLogsOncePerIdentical401(t *testing.T) {
	t.Parallel()

	var logs []string
	tracker := newAuthFailureTracker(func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	})

	authErr := &client.APIError{
		Path:       "/agent/heartbeat",
		StatusCode: 401,
		Body:       `{"error":"invalid agent token"}`,
	}

	handled := tracker.HandleResult("heartbeat", authErr)
	if !handled {
		t.Fatal("expected unauthorized error to be handled by auth tracker")
	}

	if len(logs) != 1 {
		t.Fatalf("expected a single auth failure log, got %d: %#v", len(logs), logs)
	}

	if !strings.Contains(logs[0], "agent authentication failed\n  operation: heartbeat") {
		t.Fatalf("expected clear auth summary in log, got %q", logs[0])
	}

	if !strings.Contains(logs[0], "\n  action: no retry; repeated identical auth failures will be suppressed") {
		t.Fatalf("expected suppression wording in log, got %q", logs[0])
	}
}

func TestAuthFailureTrackerSuppressesAdditional401Logs(t *testing.T) {
	t.Parallel()

	var logs []string
	tracker := newAuthFailureTracker(func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	})

	authErr := &client.APIError{
		Path:       "/agent/metrics",
		StatusCode: 401,
		Body:       `{"error":"invalid agent token"}`,
	}

	firstHandled := tracker.HandleResult("metrics", authErr)
	secondHandled := tracker.HandleResult("heartbeat", authErr)

	if !firstHandled || !secondHandled {
		t.Fatal("expected repeated unauthorized errors to be handled")
	}

	if len(logs) != 1 {
		t.Fatalf("expected one log across repeated 401s, got %d: %#v", len(logs), logs)
	}
}

func TestAuthFailureTrackerLogsAgainAfterSuccess(t *testing.T) {
	t.Parallel()

	var logs []string
	tracker := newAuthFailureTracker(func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	})

	authErr := &client.APIError{
		Path:       "/agent/metrics",
		StatusCode: 401,
		Body:       `{"error":"invalid agent token"}`,
	}

	tracker.HandleResult("metrics", authErr)
	tracker.HandleResult("metrics", nil)
	tracker.HandleResult("metrics", authErr)

	if len(logs) != 2 {
		t.Fatalf("expected auth log to reset after success, got %d: %#v", len(logs), logs)
	}
}

func TestAuthFailureTrackerIgnoresNon401Errors(t *testing.T) {
	t.Parallel()

	tracker := newAuthFailureTracker(func(string, ...any) {})

	if handled := tracker.HandleResult("heartbeat", fmt.Errorf("network timeout")); handled {
		t.Fatal("expected non-401 error to fall through to normal logging")
	}
}
