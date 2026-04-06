package collectors

import (
	"fmt"

	"github.com/MariusBobitiu/agrafa-agent/src/types"
	"github.com/MariusBobitiu/agrafa-agent/src/utils"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

type SystemMetricsCollector struct {
	diskPath string
}

func NewSystemMetricsCollector(diskPath string) *SystemMetricsCollector {
	return &SystemMetricsCollector{
		diskPath: diskPath,
	}
}

func (collector *SystemMetricsCollector) Collect() (types.SystemMetrics, error) {
	cpuPercentages, err := cpu.Percent(0, false)
	if err != nil {
		return types.SystemMetrics{}, fmt.Errorf("read cpu usage: %w", err)
	}

	if len(cpuPercentages) == 0 {
		return types.SystemMetrics{}, fmt.Errorf("read cpu usage: no samples returned")
	}

	memoryStats, err := mem.VirtualMemory()
	if err != nil {
		return types.SystemMetrics{}, fmt.Errorf("read memory usage: %w", err)
	}

	diskStats, err := disk.Usage(collector.diskPath)
	if err != nil {
		return types.SystemMetrics{}, fmt.Errorf("read disk usage for %s: %w", collector.diskPath, err)
	}

	return types.SystemMetrics{
		ObservedAt:  utils.NowUTC(),
		CPUUsage:    cpuPercentages[0],
		MemoryUsage: memoryStats.UsedPercent,
		DiskUsage:   diskStats.UsedPercent,
		DiskPath:    collector.diskPath,
	}, nil
}
