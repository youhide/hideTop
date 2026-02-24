package metrics

import (
	"time"

	"github.com/youhide/hideTop/internal/metrics/gpu"
)

type CPUStats struct {
	PerCore []float64
	Total   float64
}

type MemoryStats struct {
	TotalGB     float64
	UsedGB      float64
	AvailableGB float64
	Percent     float64
	SwapTotalGB float64
	SwapUsedGB  float64
	SwapPercent float64
}

type LoadAvg struct {
	Load1  float64
	Load5  float64
	Load15 float64
}

type ProcessInfo struct {
	PID        int32
	Name       string
	CPUPercent float64
	MemPercent float32
}

type MetricStatus struct {
	Stale bool
	Error string
}

type CollectionStatus struct {
	CPU       MetricStatus
	Memory    MetricStatus
	Load      MetricStatus
	Processes MetricStatus
	GPU       MetricStatus
}

func (s CollectionStatus) HasStale() bool {
	return s.CPU.Stale || s.Memory.Stale || s.Load.Stale || s.Processes.Stale || s.GPU.Stale
}

func (s CollectionStatus) StaleMetrics() []string {
	stale := make([]string, 0, 5)
	if s.CPU.Stale {
		stale = append(stale, "cpu")
	}
	if s.Memory.Stale {
		stale = append(stale, "mem")
	}
	if s.Load.Stale {
		stale = append(stale, "load")
	}
	if s.Processes.Stale {
		stale = append(stale, "proc")
	}
	if s.GPU.Stale {
		stale = append(stale, "gpu")
	}
	return stale
}

type Snapshot struct {
	CPU       CPUStats
	Memory    MemoryStats
	Load      LoadAvg
	Processes []ProcessInfo
	GPU       *gpu.Stats

	CollectedAt     time.Time
	ProcessSampleAt time.Time
	ProcessSortBy   SortField
	Status          CollectionStatus
}
