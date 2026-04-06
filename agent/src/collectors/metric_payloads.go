package collectors

import "github.com/MariusBobitiu/agrafa-agent/src/types"

const (
	metricNameCPUUsage    = "cpu_usage"
	metricNameMemoryUsage = "memory_usage"
	metricNameDiskUsage   = "disk_usage"
	metricUnitPercent     = "percent"
)

func BuildMetricsRequest(nodeID int64, metrics types.SystemMetrics) types.MetricsRequest {
	return types.MetricsRequest{
		NodeID: nodeID,
		Samples: []types.MetricSampleRequest{
			{
				MetricName:  metricNameCPUUsage,
				MetricValue: metrics.CPUUsage,
				MetricUnit:  metricUnitPercent,
				ObservedAt:  metrics.ObservedAt,
				Payload: map[string]any{
					"scope": "node",
				},
			},
			{
				MetricName:  metricNameMemoryUsage,
				MetricValue: metrics.MemoryUsage,
				MetricUnit:  metricUnitPercent,
				ObservedAt:  metrics.ObservedAt,
				Payload: map[string]any{
					"scope": "node",
				},
			},
			{
				MetricName:  metricNameDiskUsage,
				MetricValue: metrics.DiskUsage,
				MetricUnit:  metricUnitPercent,
				ObservedAt:  metrics.ObservedAt,
				Payload: map[string]any{
					"path":  metrics.DiskPath,
					"scope": "node",
				},
			},
		},
	}
}
