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
	return MemoryStats{
		TotalGB: float64(vm.Total) / gb,
		UsedGB:  float64(vm.Used) / gb,
		Percent: vm.UsedPercent,
	}, nil
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
