package types

import (
	"encoding/json"
	"time"
)

type Config struct {
	APIBaseURL            string
	AgentToken            string
	NodeID                int64
	Source                string
	APITimeout            time.Duration
	APIRetryCount         int
	HeartbeatInterval     time.Duration
	MetricsInterval       time.Duration
	HealthInterval        time.Duration
	ConfigRefreshInterval time.Duration
	HTTPTimeout           time.Duration
	DiskPath              string
	HealthChecks          []HealthCheck
}

type HealthCheck struct {
	ServiceID       int64  `json:"service_id"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	Target          string `json:"target"`
	IntervalSeconds int    `json:"interval_seconds,omitempty"`
	TimeoutSeconds  int    `json:"timeout_seconds,omitempty"`
}

func (h *HealthCheck) UnmarshalJSON(data []byte) error {
	type rawHealthCheck struct {
		ServiceID       int64  `json:"service_id"`
		Name            string `json:"name"`
		Type            string `json:"type"`
		CheckType       string `json:"check_type"`
		Target          string `json:"target"`
		CheckTarget     string `json:"check_target"`
		IntervalSeconds int    `json:"interval_seconds"`
		TimeoutSeconds  int    `json:"timeout_seconds"`
	}

	var raw rawHealthCheck
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	h.ServiceID = raw.ServiceID
	h.Name = raw.Name
	h.Type = firstNonEmptyString(raw.Type, raw.CheckType)
	h.Target = firstNonEmptyString(raw.Target, raw.CheckTarget)
	h.IntervalSeconds = raw.IntervalSeconds
	h.TimeoutSeconds = raw.TimeoutSeconds

	return nil
}

type HealthChecksFile struct {
	HealthChecks []HealthCheck `json:"health_checks"`
}

type AgentConfigNode struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
}

type AgentConfigCheck struct {
	ServiceID       int64  `json:"service_id"`
	Name            string `json:"name"`
	CheckType       string `json:"check_type"`
	CheckTarget     string `json:"check_target"`
	IntervalSeconds int    `json:"interval_seconds"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
}

type AgentConfigResponse struct {
	Node         AgentConfigNode    `json:"node"`
	HealthChecks []AgentConfigCheck `json:"health_checks"`
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
