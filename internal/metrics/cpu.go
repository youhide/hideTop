package metrics

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

func CollectCPU(ctx context.Context, interval time.Duration) (CPUStats, error) {
	perCore, err := cpu.PercentWithContext(ctx, interval, true)
	if err != nil {
		return CPUStats{}, err
	}

	var sum float64
	for _, v := range perCore {
		sum += v
	}
	total := 0.0
	if len(perCore) > 0 {
		total = sum / float64(len(perCore))
	}

	return CPUStats{
		PerCore: perCore,
		Total:   total,
	}, nil
}
