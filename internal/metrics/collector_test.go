package metrics

import (
	"testing"
	"time"
)

func TestShouldCollectProcesses(t *testing.T) {
	now := time.Now()
	previous := Snapshot{
		Processes:       []ProcessInfo{{PID: 1}},
		ProcessSampleAt: now.Add(-time.Second),
		ProcessSortBy:   SortByCPU,
	}

	if shouldCollectProcesses(now, 2*time.Second, SortByCPU, previous) {
		t.Fatalf("expected cached processes to be reused before sampling window")
	}
	if !shouldCollectProcesses(now, 2*time.Second, SortByMem, previous) {
		t.Fatalf("expected recollection when sort field changes")
	}
	if !shouldCollectProcesses(now, 2*time.Second, SortByCPU, Snapshot{}) {
		t.Fatalf("expected initial collection when no previous sample exists")
	}
}

func TestCollectionStatusStaleMetrics(t *testing.T) {
	status := CollectionStatus{
		CPU:       MetricStatus{Stale: true},
		Processes: MetricStatus{Stale: true},
	}

	stale := status.StaleMetrics()
	if len(stale) != 2 || stale[0] != "cpu" || stale[1] != "proc" {
		t.Fatalf("unexpected stale list: %#v", stale)
	}
	if !status.HasStale() {
		t.Fatalf("expected HasStale to return true")
	}
}
