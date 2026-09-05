package metrics

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/youhide/hideTop/internal/metrics/gpu"
)

// CollectOptions controls which metrics to skip.
type CollectOptions struct {
	SkipGPU  bool
	SkipTemp bool
}

func Collect(
	ctx context.Context,
	cpuInterval time.Duration,
	sortBy SortField,
	procLimit int,
	processSampleEvery time.Duration,
	previous Snapshot,
	opts CollectOptions,
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
	if !processesDue {
		snap.Processes = previous.Processes
		snap.ProcessSampleAt = previous.ProcessSampleAt
		snap.ProcessSortBy = sortBy
	}

	if !opts.SkipTemp {
		wg.Go(func() {
			t, err := CollectTemperature(ctx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				snap.Status.Temperature = staleStatus(err)
				if previous.Temperature.Available {
					snap.Temperature = previous.Temperature
				}
				return
			}
			snap.Temperature = t
		})
	}

	wg.Go(func() {
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
	})

	wg.Go(func() {
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
	})

	wg.Go(func() {
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
	})

	wg.Go(func() {
		n, err := CollectNetwork(ctx)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			snap.Status.Network = staleStatus(err)
			if previous.Network.Available {
				snap.Network = previous.Network
			}
			return
		}
		snap.Network = n
	})

	wg.Go(func() {
		d, err := CollectDisk(ctx)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			snap.Status.Disk = staleStatus(err)
			if previous.Disk.Available {
				snap.Disk = previous.Disk
			}
			return
		}
		snap.Disk = d
	})

	wg.Go(func() {
		b, err := CollectBattery(ctx)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			snap.Status.Battery = staleStatus(err)
			if previous.Battery.Available {
				snap.Battery = previous.Battery
			}
			return
		}
		snap.Battery = b
	})

	if processesDue {
		wg.Go(func() {
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
		})
	}

	if !opts.SkipGPU {
		wg.Go(func() {
			g := gpu.Collect(ctx, 0) // cpuTotal not needed for raw GPU metrics
			mu.Lock()
			defer mu.Unlock()
			if g.Available {
				snap.GPU = &g
			} else if previous.GPU != nil && previous.GPU.Available {
				// Copy rather than alias: the energy estimate below writes
				// through this pointer after wg.Wait(), on the collection
				// goroutine, while the UI goroutine is reading the previous
				// snapshot's GPU stats in View.
				stale := *previous.GPU
				snap.GPU = &stale
				snap.Status.GPU = staleStatus(errors.New("collector unavailable"))
			}
		})
	}

	if !waitCtx(ctx, &wg) {
		// A collector is still running past the deadline. Return what we have;
		// the stragglers write into snap under mu and their results are
		// dropped with this snapshot. Without this the app would sit on
		// collecting=true forever and silently stop refreshing.
		mu.Lock()
		defer mu.Unlock()
		snap.Status.Timeout = true
		return snap
	}

	// Compute energy impact after all metrics are collected, since it
	// depends on both CPU and GPU utilization. Skip when CPU data is stale
	// so the estimate is not based on an outdated CPU reading.
	if snap.GPU != nil && snap.GPU.Available && !snap.Status.CPU.Stale {
		snap.GPU.Energy = gpu.ComputeEnergyImpact(snap.CPU.Total, snap.GPU.Utilization, true, snap.GPU.Thermal)
	}

	return snap
}

// waitCtx waits for wg, giving up if ctx is done first. It reports whether the
// wait completed.
func waitCtx(ctx context.Context, wg *sync.WaitGroup) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
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
