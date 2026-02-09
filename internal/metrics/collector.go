package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/youhide/hideTop/internal/metrics/gpu"
)

func Collect(ctx context.Context, cpuInterval time.Duration, sortBy SortField, procLimit int) Snapshot {
	var (
		wg   sync.WaitGroup
		snap Snapshot
		mu   sync.Mutex
	)

	wg.Add(4)

	go func() {
		defer wg.Done()
		c, err := CollectCPU(ctx, cpuInterval)
		if err == nil {
			mu.Lock()
			snap.CPU = c
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		m, err := CollectMemory(ctx)
		if err == nil {
			mu.Lock()
			snap.Memory = m
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		l, err := CollectLoad(ctx)
		if err == nil {
			mu.Lock()
			snap.Load = l
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		p, err := CollectProcesses(ctx, sortBy, procLimit)
		if err == nil {
			mu.Lock()
			snap.Processes = p
			mu.Unlock()
		}
	}()

	wg.Wait()

	// GPU collection runs after CPU so it can use cpuTotal for energy impact.
	// gpu.Collect runs its own sub-collectors concurrently and is fast.
	g := gpu.Collect(ctx, snap.CPU.Total)
	snap.GPU = &g

	return snap
}
