package types

import "time"

type SystemMetrics struct {
	ObservedAt  time.Time
	CPUUsage    float64
	MemoryUsage float64
	DiskUsage   float64
	DiskPath    string
}

type HTTPCheckResult struct {
	ServiceID      int64
	Name           string
	Type           string
	Target         string
	ObservedAt     time.Time
	IsSuccess      bool
	StatusCode     *int32
	ResponseTimeMs *int32
	Message        string
}
