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

func CollectProcesses(ctx context.Context, sortBy SortField, limit int) ([]ProcessInfo, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	infos := make([]ProcessInfo, 0, len(procs))
	for _, p := range procs {
		name, _ := p.NameWithContext(ctx)
		cpuPct, _ := p.CPUPercentWithContext(ctx)
		memPct, _ := p.MemoryPercentWithContext(ctx)

		infos = append(infos, ProcessInfo{
			PID:        p.Pid,
			Name:       name,
			CPUPercent: cpuPct,
			MemPercent: memPct,
		})
	}

	sort.Slice(infos, func(i, j int) bool {
		switch sortBy {
		case SortByMem:
			return infos[i].MemPercent > infos[j].MemPercent
		case SortByPID:
			return infos[i].PID < infos[j].PID
		default:
			return infos[i].CPUPercent > infos[j].CPUPercent
		}
	})

	if limit > 0 && limit < len(infos) {
		infos = infos[:limit]
	}
	return infos, nil
}
