package metrics

import (
	"cmp"
	"context"
	"slices"

	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

type SortField int

const (
	SortByCPU SortField = iota
	SortByMem
	SortByPID
)

type processSample struct {
	process *process.Process
	pid     int32
	cpu     float64
	mem     float32
}

// memoryPercent computes a process's share of physical memory from its RSS
// against a total the caller already fetched, falling back to gopsutil's own
// (much more expensive) helper when the total is unknown.
func memoryPercent(ctx context.Context, p *process.Process, totalMem uint64) (float32, error) {
	if totalMem == 0 {
		return p.MemoryPercentWithContext(ctx)
	}
	info, err := p.MemoryInfoWithContext(ctx)
	if err != nil {
		return 0, err
	}
	return float32(info.RSS) / float32(totalMem) * 100, nil
}

func CollectProcesses(ctx context.Context, sortBy SortField, limit int) ([]ProcessInfo, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	// Total physical memory, fetched once. p.MemoryPercent() re-queries it for
	// every PID, so on a machine with ~600 processes that was ~600 redundant
	// full VirtualMemory() calls per sample.
	var totalMem uint64
	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		totalMem = vm.Total
	}

	samples := make([]processSample, 0, len(procs))
	for _, p := range procs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		cpuPct, cpuErr := p.CPUPercentWithContext(ctx)
		memPct, memErr := memoryPercent(ctx, p, totalMem)
		if cpuErr != nil || memErr != nil {
			// Skip the process if either metric is unavailable to avoid
			// reporting a misleading zero for the failed measurement.
			continue
		}

		samples = append(samples, processSample{
			process: p,
			pid:     p.Pid,
			cpu:     cpuPct,
			mem:     memPct,
		})
	}

	slices.SortFunc(samples, func(a, b processSample) int {
		switch sortBy {
		case SortByMem:
			return cmp.Compare(b.mem, a.mem)
		case SortByPID:
			return cmp.Compare(a.pid, b.pid)
		default:
			return cmp.Compare(b.cpu, a.cpu)
		}
	})

	if limit > 0 && limit < len(samples) {
		samples = samples[:limit]
	}

	infos := make([]ProcessInfo, 0, len(samples))
	for _, sample := range samples {
		name, _ := sample.process.NameWithContext(ctx)
		if name == "" {
			name = "?"
		}
		user, _ := sample.process.UsernameWithContext(ctx)
		ppid, _ := sample.process.PpidWithContext(ctx)

		var state string
		if ss, err := sample.process.StatusWithContext(ctx); err == nil && len(ss) > 0 {
			state = ss[0]
		}

		var threads int32
		if t, err := sample.process.NumThreadsWithContext(ctx); err == nil {
			threads = t
		}

		infos = append(infos, ProcessInfo{
			PID:        sample.pid,
			PPID:       ppid,
			Name:       name,
			User:       user,
			CPUPercent: sample.cpu,
			MemPercent: sample.mem,
			State:      state,
			NumThreads: threads,
		})
	}

	return infos, nil
}
