package metrics

import (
	"context"

	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
)

func CollectMemory(ctx context.Context) (MemoryStats, error) {
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return MemoryStats{}, err
	}

	const gb = 1 << 30
	ms := MemoryStats{
		TotalGB:     float64(vm.Total) / gb,
		UsedGB:      float64(vm.Used) / gb,
		AvailableGB: float64(vm.Available) / gb,
		Percent:     vm.UsedPercent,
	}

	sw, err := mem.SwapMemoryWithContext(ctx)
	if err == nil && sw.Total > 0 {
		ms.SwapTotalGB = float64(sw.Total) / gb
		ms.SwapUsedGB = float64(sw.Used) / gb
		ms.SwapPercent = sw.UsedPercent
	}

	return ms, nil
}

func CollectLoad(ctx context.Context) (LoadAvg, error) {
	avg, err := load.AvgWithContext(ctx)
	if err != nil {
		return LoadAvg{}, err
	}
	return LoadAvg{
		Load1:  avg.Load1,
		Load5:  avg.Load5,
		Load15: avg.Load15,
	}, nil
}
