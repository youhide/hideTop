package metrics

import (
	"context"

	"github.com/shirou/gopsutil/v4/disk"
)

// DiskIOStats holds disk I/O counters for a single device.
type DiskIOStats struct {
	Name       string
	ReadBytes  uint64 // cumulative bytes read
	WriteBytes uint64 // cumulative bytes written
}

// DiskStats holds disk I/O and usage information.
type DiskStats struct {
	Available  bool
	Devices    []DiskIOStats
	TotalRead  uint64
	TotalWrite uint64
	// Root filesystem usage
	RootUsedGB  float64
	RootTotalGB float64
	RootPercent float64
}

// DiskDelta holds computed throughput (bytes/sec).
type DiskDelta struct {
	Available bool
	ReadSec   float64 // bytes/sec
	WriteSec  float64 // bytes/sec
	Devices   []DiskDeviceDelta
}

// DiskDeviceDelta holds per-device bytes/sec.
type DiskDeviceDelta struct {
	Name     string
	ReadSec  float64
	WriteSec float64
}

// CollectDisk gathers disk I/O counters and root filesystem usage.
func CollectDisk(ctx context.Context) (DiskStats, error) {
	stats := DiskStats{}

	// Disk I/O counters
	counters, err := disk.IOCountersWithContext(ctx)
	if err == nil && len(counters) > 0 {
		stats.Available = true
		for name, c := range counters {
			stats.Devices = append(stats.Devices, DiskIOStats{
				Name:       name,
				ReadBytes:  c.ReadBytes,
				WriteBytes: c.WriteBytes,
			})
			stats.TotalRead += c.ReadBytes
			stats.TotalWrite += c.WriteBytes
		}
	}

	// Root filesystem usage
	usage, err := disk.UsageWithContext(ctx, "/")
	if err == nil && usage != nil {
		const gb = 1 << 30
		stats.RootUsedGB = float64(usage.Used) / gb
		stats.RootTotalGB = float64(usage.Total) / gb
		stats.RootPercent = usage.UsedPercent
		stats.Available = true
	}

	return stats, nil
}

// ComputeDiskDelta calculates throughput between two disk snapshots.
func ComputeDiskDelta(current, previous DiskStats, intervalSecs float64) DiskDelta {
	if !current.Available || !previous.Available || intervalSecs <= 0 {
		return DiskDelta{}
	}

	delta := DiskDelta{
		Available: true,
		ReadSec:   safeDiskDeltaRate(current.TotalRead, previous.TotalRead, intervalSecs),
		WriteSec:  safeDiskDeltaRate(current.TotalWrite, previous.TotalWrite, intervalSecs),
	}

	prevMap := make(map[string]DiskIOStats, len(previous.Devices))
	for _, d := range previous.Devices {
		prevMap[d.Name] = d
	}

	for _, cur := range current.Devices {
		prev, ok := prevMap[cur.Name]
		if !ok {
			continue
		}
		delta.Devices = append(delta.Devices, DiskDeviceDelta{
			Name:     cur.Name,
			ReadSec:  safeDiskDeltaRate(cur.ReadBytes, prev.ReadBytes, intervalSecs),
			WriteSec: safeDiskDeltaRate(cur.WriteBytes, prev.WriteBytes, intervalSecs),
		})
	}

	return delta
}

// safeDiskDeltaRate computes (current - previous) / interval, returning 0
// when counters have wrapped (e.g. after a reboot).
func safeDiskDeltaRate(current, previous uint64, intervalSecs float64) float64 {
	if current < previous {
		return 0
	}
	return float64(current-previous) / intervalSecs
}
