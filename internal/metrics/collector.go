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

	wg.Add(5)

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

	go func() {
		defer wg.Done()
		g := gpu.Collect(ctx)
		mu.Lock()
		snap.GPU = &g
		mu.Unlock()
	}()

	wg.Wait()
	return snap
}
