package services

import (
	"testing"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

func TestCalculateServiceState(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		currentState     string
		failures         int
		successes        int
		isSuccess        bool
		expectedState    string
		expectedFailures int
		expectedSuccess  int
		transitioned     bool
	}{
		{
			name:             "healthy plus failed check becomes degraded",
			currentState:     types.ServiceStateHealthy,
			failures:         0,
			successes:        0,
			isSuccess:        false,
			expectedState:    types.ServiceStateDegraded,
			expectedFailures: 1,
			expectedSuccess:  0,
			transitioned:     true,
		},
		{
			name:             "degraded plus second failure stays degraded",
			currentState:     types.ServiceStateDegraded,
			failures:         1,
			successes:        0,
			isSuccess:        false,
			expectedState:    types.ServiceStateDegraded,
			expectedFailures: 2,
			expectedSuccess:  0,
			transitioned:     false,
		},
		{
			name:             "third consecutive failure becomes unhealthy",
			currentState:     types.ServiceStateDegraded,
			failures:         2,
			successes:        0,
			isSuccess:        false,
			expectedState:    types.ServiceStateUnhealthy,
			expectedFailures: 3,
			expectedSuccess:  0,
			transitioned:     true,
		},
		{
			name:             "unhealthy plus failed check stays unhealthy",
			currentState:     types.ServiceStateUnhealthy,
			failures:         3,
			successes:        0,
			isSuccess:        false,
			expectedState:    types.ServiceStateUnhealthy,
			expectedFailures: 4,
			expectedSuccess:  0,
			transitioned:     false,
		},
		{
			name:             "unhealthy plus one success becomes degraded",
			currentState:     types.ServiceStateUnhealthy,
			failures:         3,
			successes:        0,
			isSuccess:        true,
			expectedState:    types.ServiceStateDegraded,
			expectedFailures: 0,
			expectedSuccess:  1,
			transitioned:     true,
		},
		{
			name:             "degraded plus second consecutive success becomes healthy",
			currentState:     types.ServiceStateDegraded,
			failures:         0,
			successes:        1,
			isSuccess:        true,
			expectedState:    types.ServiceStateHealthy,
			expectedFailures: 0,
			expectedSuccess:  0,
			transitioned:     true,
		},
		{
			name:             "success resets failure count",
			currentState:     types.ServiceStateDegraded,
			failures:         2,
			successes:        0,
			isSuccess:        true,
			expectedState:    types.ServiceStateDegraded,
			expectedFailures: 0,
			expectedSuccess:  1,
			transitioned:     false,
		},
		{
			name:             "failure resets success count",
			currentState:     types.ServiceStateDegraded,
			failures:         0,
			successes:        1,
			isSuccess:        false,
			expectedState:    types.ServiceStateDegraded,
			expectedFailures: 1,
			expectedSuccess:  0,
			transitioned:     false,
		},
		{
			name:             "healthy plus success stays healthy with no transition",
			currentState:     types.ServiceStateHealthy,
			failures:         0,
			successes:        1,
			isSuccess:        true,
			expectedState:    types.ServiceStateHealthy,
			expectedFailures: 0,
			expectedSuccess:  0,
			transitioned:     false,
		},
		{
			name:             "degraded plus first success stays degraded",
			currentState:     types.ServiceStateDegraded,
			failures:         1,
			successes:        0,
			isSuccess:        true,
			expectedState:    types.ServiceStateDegraded,
			expectedFailures: 0,
			expectedSuccess:  1,
			transitioned:     false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			nextState, failures, successes := calculateServiceState(testCase.currentState, testCase.failures, testCase.successes, testCase.isSuccess)

			if nextState != testCase.expectedState {
				t.Fatalf("expected state %q, got %q", testCase.expectedState, nextState)
			}

			if failures != testCase.expectedFailures {
				t.Fatalf("expected failures %d, got %d", testCase.expectedFailures, failures)
			}

			if successes != testCase.expectedSuccess {
				t.Fatalf("expected successes %d, got %d", testCase.expectedSuccess, successes)
			}

			transitioned := nextState != testCase.currentState
			if transitioned != testCase.transitioned {
				t.Fatalf("expected transitioned=%t, got %t", testCase.transitioned, transitioned)
			}
		})
	}
}
