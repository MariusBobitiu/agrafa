package types

import "time"

type HeartbeatRequest struct {
	NodeID     int64          `json:"node_id"`
	ObservedAt time.Time      `json:"observed_at"`
	Source     string         `json:"source,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type HealthRequest struct {
	ServiceID      int64          `json:"service_id"`
	ObservedAt     time.Time      `json:"observed_at"`
	IsSuccess      bool           `json:"is_success"`
	StatusCode     *int32         `json:"status_code,omitempty"`
	ResponseTimeMs *int32         `json:"response_time_ms,omitempty"`
	Message        string         `json:"message,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
}

type MetricSampleRequest struct {
	MetricName  string         `json:"metric_name"`
	MetricValue float64        `json:"metric_value"`
	MetricUnit  string         `json:"metric_unit,omitempty"`
	ObservedAt  time.Time      `json:"observed_at"`
	Payload     map[string]any `json:"payload,omitempty"`
}

type MetricsRequest struct {
	NodeID    int64                 `json:"node_id"`
	ServiceID *int64                `json:"service_id,omitempty"`
	Samples   []MetricSampleRequest `json:"samples"`
}
