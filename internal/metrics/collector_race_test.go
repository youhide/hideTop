package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/youhide/hideTop/internal/metrics/gpu"
)

// TestCollectDoesNotAliasPreviousGPU pins the fix for a data race: when the
// GPU collector failed, Collect used to carry the previous snapshot's *gpu.Stats
// pointer forward and then write the energy estimate through it after
// wg.Wait() — on the collection goroutine, while the UI goroutine was reading
// the very same struct in View.
func TestCollectDoesNotAliasPreviousGPU(t *testing.T) {
	prev := Snapshot{GPU: &gpu.Stats{Available: true, Utilization: 42}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// SkipGPU leaves snap.GPU nil, so force the carry-forward path by asking
	// for GPU on a machine where it may or may not be present; either way the
	// invariant below must hold.
	snap := Collect(ctx, time.Millisecond, SortByCPU, 1, time.Hour, prev, CollectOptions{SkipTemp: true})

	if snap.GPU != nil && snap.GPU == prev.GPU {
		t.Fatal("snapshot aliases the previous snapshot's *gpu.Stats; " +
			"the energy write after wg.Wait() would race with View")
	}
}

// TestCollectHonoursContextDeadline pins that a stuck collector cannot hold
// Collect open indefinitely: m.collecting would stay true and the app would
// silently stop refreshing.
func TestCollectHonoursContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	done := make(chan Snapshot, 1)
	go func() {
		// A long CPU sampling interval keeps a collector busy past the deadline.
		done <- Collect(ctx, 5*time.Second, SortByCPU, 1, time.Hour, Snapshot{}, CollectOptions{})
	}()

	select {
	case snap := <-done:
		if !snap.Status.Timeout {
			t.Error("Collect returned before the deadline without marking Timeout")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Collect ignored the context deadline and blocked on wg.Wait()")
	}
}
