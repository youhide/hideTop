package metrics

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/youhide/hideTop/internal/metrics/gpu"
)

func Collect(
	ctx context.Context,
	cpuInterval time.Duration,
	sortBy SortField,
	procLimit int,
	processSampleEvery time.Duration,
	previous Snapshot,
) Snapshot {
	now := time.Now()

	var (
		wg   sync.WaitGroup
		snap = Snapshot{
			CollectedAt:     now,
			ProcessSampleAt: previous.ProcessSampleAt,
			ProcessSortBy:   previous.ProcessSortBy,
		}
		mu sync.Mutex
	)

	processesDue := shouldCollectProcesses(now, processSampleEvery, sortBy, previous)
	workers := 3
	if processesDue {
		workers++
	} else {
		snap.Processes = previous.Processes
		snap.ProcessSampleAt = previous.ProcessSampleAt
		snap.ProcessSortBy = sortBy
	}

	wg.Add(workers)

	go func() {
		defer wg.Done()
		c, err := CollectCPU(ctx, cpuInterval)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			snap.Status.CPU = staleStatus(err)
			if !previous.CollectedAt.IsZero() {
				snap.CPU = previous.CPU
			}
			return
		}
		snap.CPU = c
	}()

	go func() {
		defer wg.Done()
		m, err := CollectMemory(ctx)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			snap.Status.Memory = staleStatus(err)
			if !previous.CollectedAt.IsZero() {
				snap.Memory = previous.Memory
			}
			return
		}
		snap.Memory = m
	}()

	go func() {
		defer wg.Done()
		l, err := CollectLoad(ctx)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			snap.Status.Load = staleStatus(err)
			if !previous.CollectedAt.IsZero() {
				snap.Load = previous.Load
			}
			return
		}
		snap.Load = l
	}()

	if processesDue {
		go func() {
			defer wg.Done()
			p, err := CollectProcesses(ctx, sortBy, procLimit)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				snap.Status.Processes = staleStatus(err)
				if len(previous.Processes) > 0 {
					snap.Processes = previous.Processes
					snap.ProcessSampleAt = previous.ProcessSampleAt
					snap.ProcessSortBy = previous.ProcessSortBy
				}
				return
			}
			snap.Processes = p
			snap.ProcessSampleAt = now
			snap.ProcessSortBy = sortBy
		}()
	}

	wg.Wait()

	// GPU collection runs after CPU so it can use cpuTotal for energy impact.
	g := gpu.Collect(ctx, snap.CPU.Total)
	if g.Available {
		snap.GPU = &g
	} else if previous.GPU != nil && previous.GPU.Available {
		snap.GPU = previous.GPU
		snap.Status.GPU = staleStatus(errors.New("collector unavailable"))
	}

	return snap
}

func shouldCollectProcesses(now time.Time, interval time.Duration, sortBy SortField, previous Snapshot) bool {
	if len(previous.Processes) == 0 || previous.ProcessSampleAt.IsZero() {
		return true
	}
	if previous.ProcessSortBy != sortBy {
		return true
	}
	if interval <= 0 {
		return true
	}
	return now.Sub(previous.ProcessSampleAt) >= interval
}

func staleStatus(err error) MetricStatus {
	if err == nil {
		return MetricStatus{}
	}
	return MetricStatus{Stale: true, Error: err.Error()}
}
