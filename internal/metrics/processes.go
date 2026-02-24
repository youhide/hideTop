package metrics

import (
	"context"
	"sort"

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

func CollectProcesses(ctx context.Context, sortBy SortField, limit int) ([]ProcessInfo, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	samples := make([]processSample, 0, len(procs))
	for _, p := range procs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		cpuPct, cpuErr := p.CPUPercentWithContext(ctx)
		memPct, memErr := p.MemoryPercentWithContext(ctx)
		if cpuErr != nil && memErr != nil {
			continue
		}

		samples = append(samples, processSample{
			process: p,
			pid:     p.Pid,
			cpu:     cpuPct,
			mem:     memPct,
		})
	}

	sort.Slice(samples, func(i, j int) bool {
		switch sortBy {
		case SortByMem:
			return samples[i].mem > samples[j].mem
		case SortByPID:
			return samples[i].pid < samples[j].pid
		default:
			return samples[i].cpu > samples[j].cpu
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

		infos = append(infos, ProcessInfo{
			PID:        sample.pid,
			Name:       name,
			CPUPercent: sample.cpu,
			MemPercent: sample.mem,
		})
	}

	return infos, nil
}
