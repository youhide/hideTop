package metrics

import (
	"context"

	psnet "github.com/shirou/gopsutil/v4/net"
)

// InterfaceStats holds network throughput for a single interface.
type InterfaceStats struct {
	Name     string
	BytesIn  uint64 // total bytes received (cumulative)
	BytesOut uint64 // total bytes sent (cumulative)
}

// NetworkStats holds network I/O counters.
type NetworkStats struct {
	Available  bool
	Interfaces []InterfaceStats
	TotalIn    uint64 // total bytes received across all interfaces
	TotalOut   uint64 // total bytes sent across all interfaces
}

// NetworkDelta holds computed throughput (bytes/sec) between two snapshots.
type NetworkDelta struct {
	Available   bool
	Interfaces  []InterfaceDelta
	TotalInSec  float64 // bytes/sec in
	TotalOutSec float64 // bytes/sec out
}

// InterfaceDelta holds per-interface bytes/sec.
type InterfaceDelta struct {
	Name   string
	InSec  float64 // bytes/sec received
	OutSec float64 // bytes/sec sent
}

// CollectNetwork gathers network I/O counters.
func CollectNetwork(ctx context.Context) (NetworkStats, error) {
	counters, err := psnet.IOCountersWithContext(ctx, true)
	if err != nil {
		return NetworkStats{}, err
	}

	if len(counters) == 0 {
		return NetworkStats{}, nil
	}

	stats := NetworkStats{Available: true}
	for _, c := range counters {
		// Skip loopback
		if c.Name == "lo" || c.Name == "lo0" {
			continue
		}
		// Skip inactive interfaces (no traffic ever)
		if c.BytesRecv == 0 && c.BytesSent == 0 {
			continue
		}
		stats.Interfaces = append(stats.Interfaces, InterfaceStats{
			Name:     c.Name,
			BytesIn:  c.BytesRecv,
			BytesOut: c.BytesSent,
		})
		stats.TotalIn += c.BytesRecv
		stats.TotalOut += c.BytesSent
	}

	if len(stats.Interfaces) == 0 {
		stats.Available = false
	}
	return stats, nil
}

// ComputeNetworkDelta calculates throughput between two network snapshots.
func ComputeNetworkDelta(current, previous NetworkStats, intervalSecs float64) NetworkDelta {
	if !current.Available || !previous.Available || intervalSecs <= 0 {
		return NetworkDelta{}
	}

	delta := NetworkDelta{
		Available:   true,
		TotalInSec:  safeDeltaRate(current.TotalIn, previous.TotalIn, intervalSecs),
		TotalOutSec: safeDeltaRate(current.TotalOut, previous.TotalOut, intervalSecs),
	}

	// Build a map of previous interface stats for lookup
	prevMap := make(map[string]InterfaceStats, len(previous.Interfaces))
	for _, iface := range previous.Interfaces {
		prevMap[iface.Name] = iface
	}

	for _, cur := range current.Interfaces {
		prev, ok := prevMap[cur.Name]
		if !ok {
			continue
		}
		delta.Interfaces = append(delta.Interfaces, InterfaceDelta{
			Name:   cur.Name,
			InSec:  safeDeltaRate(cur.BytesIn, prev.BytesIn, intervalSecs),
			OutSec: safeDeltaRate(cur.BytesOut, prev.BytesOut, intervalSecs),
		})
	}

	return delta
}

// safeDeltaRate computes (current - previous) / interval, returning 0
// when counters have wrapped (e.g. after a reboot).
func safeDeltaRate(current, previous uint64, intervalSecs float64) float64 {
	if current < previous {
		return 0
	}
	return float64(current-previous) / intervalSecs
}
