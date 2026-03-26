package metrics

import (
	"testing"
)

func TestComputeNetworkDelta(t *testing.T) {
	prev := NetworkStats{
		Available: true,
		TotalIn:   2000,
		TotalOut:  1000,
		Interfaces: []InterfaceStats{
			{Name: "en0", BytesOut: 1000, BytesIn: 2000},
		},
	}
	curr := NetworkStats{
		Available: true,
		TotalIn:   6000,
		TotalOut:  3000,
		Interfaces: []InterfaceStats{
			{Name: "en0", BytesOut: 3000, BytesIn: 6000},
		},
	}

	delta := ComputeNetworkDelta(curr, prev, 2.0)

	if delta.TotalOutSec != 1000 {
		t.Errorf("expected TotalOutSec=1000, got %f", delta.TotalOutSec)
	}
	if delta.TotalInSec != 2000 {
		t.Errorf("expected TotalInSec=2000, got %f", delta.TotalInSec)
	}
	if len(delta.Interfaces) != 1 {
		t.Fatalf("expected 1 interface delta, got %d", len(delta.Interfaces))
	}
	if delta.Interfaces[0].Name != "en0" {
		t.Errorf("expected interface en0, got %s", delta.Interfaces[0].Name)
	}
}

func TestComputeNetworkDelta_MissingInterface(t *testing.T) {
	prev := NetworkStats{
		Available: true,
		Interfaces: []InterfaceStats{
			{Name: "en0", BytesOut: 1000, BytesIn: 2000},
		},
	}
	curr := NetworkStats{
		Available: true,
		Interfaces: []InterfaceStats{
			{Name: "en1", BytesOut: 500, BytesIn: 800},
		},
	}

	delta := ComputeNetworkDelta(curr, prev, 1.0)
	// en1 has no previous data, so should have no interface deltas
	if len(delta.Interfaces) != 0 {
		t.Errorf("expected 0 interface deltas for new interface, got %d", len(delta.Interfaces))
	}
}

func TestComputeNetworkDelta_ZeroInterval(t *testing.T) {
	prev := NetworkStats{Available: true}
	curr := NetworkStats{Available: true}

	delta := ComputeNetworkDelta(curr, prev, 0)
	if delta.Available {
		t.Errorf("expected not available for zero interval")
	}
}

func TestComputeDiskDelta(t *testing.T) {
	prev := DiskStats{
		Available:  true,
		TotalRead:  1000,
		TotalWrite: 2000,
		Devices: []DiskIOStats{
			{Name: "sda", ReadBytes: 1000, WriteBytes: 2000},
		},
	}
	curr := DiskStats{
		Available:  true,
		TotalRead:  5000,
		TotalWrite: 10000,
		Devices: []DiskIOStats{
			{Name: "sda", ReadBytes: 5000, WriteBytes: 10000},
		},
	}

	delta := ComputeDiskDelta(curr, prev, 2.0)

	if delta.ReadSec != 2000 {
		t.Errorf("expected ReadSec=2000, got %f", delta.ReadSec)
	}
	if delta.WriteSec != 4000 {
		t.Errorf("expected WriteSec=4000, got %f", delta.WriteSec)
	}
	if len(delta.Devices) != 1 {
		t.Fatalf("expected 1 device delta, got %d", len(delta.Devices))
	}
}

func TestComputeDiskDelta_ZeroInterval(t *testing.T) {
	delta := ComputeDiskDelta(
		DiskStats{Available: true},
		DiskStats{Available: true},
		0,
	)
	if delta.Available {
		t.Errorf("expected not available for zero interval")
	}
}
