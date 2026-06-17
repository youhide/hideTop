package app

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/youhide/hideTop/internal/config"
	"github.com/youhide/hideTop/internal/metrics"
)

func TestSnapshotDeltaUsesImmediatelyPreviousSnapshot(t *testing.T) {
	t0 := time.Unix(100, 0)
	m := New(config.Config{RefreshInterval: time.Second})

	first := metrics.Snapshot{
		CollectedAt: t0,
		Network: metrics.NetworkStats{
			Available: true,
			TotalIn:   1000,
			TotalOut:  500,
			Interfaces: []metrics.InterfaceStats{
				{Name: "en0", BytesIn: 1000, BytesOut: 500},
			},
		},
		Disk: metrics.DiskStats{
			Available:  true,
			TotalRead:  2000,
			TotalWrite: 1000,
			Devices: []metrics.DiskIOStats{
				{Name: "disk0", ReadBytes: 2000, WriteBytes: 1000},
			},
		},
	}

	updated, _ := m.Update(snapshotMsg(first))
	m = updated.(Model)
	if m.netDelta.Available {
		t.Fatalf("first network delta should not be available")
	}
	if m.diskDelta.Available {
		t.Fatalf("first disk delta should not be available")
	}

	second := metrics.Snapshot{
		CollectedAt: t0.Add(time.Second),
		Network: metrics.NetworkStats{
			Available: true,
			TotalIn:   2000,
			TotalOut:  2500,
			Interfaces: []metrics.InterfaceStats{
				{Name: "en0", BytesIn: 2000, BytesOut: 2500},
			},
		},
		Disk: metrics.DiskStats{
			Available:  true,
			TotalRead:  5000,
			TotalWrite: 5000,
			Devices: []metrics.DiskIOStats{
				{Name: "disk0", ReadBytes: 5000, WriteBytes: 5000},
			},
		},
	}

	updated, _ = m.Update(snapshotMsg(second))
	m = updated.(Model)

	if !m.netDelta.Available {
		t.Fatalf("second network delta should be available")
	}
	if got := m.netDelta.TotalInSec; got != 1000 {
		t.Fatalf("network in/sec = %v, want 1000", got)
	}
	if got := m.netDelta.TotalOutSec; got != 2000 {
		t.Fatalf("network out/sec = %v, want 2000", got)
	}
	if !m.diskDelta.Available {
		t.Fatalf("second disk delta should be available")
	}
	if got := m.diskDelta.ReadSec; got != 3000 {
		t.Fatalf("disk read/sec = %v, want 3000", got)
	}
	if got := m.diskDelta.WriteSec; got != 4000 {
		t.Fatalf("disk write/sec = %v, want 4000", got)
	}
}

func TestMoveSelectionUsesTreeDisplayOrder(t *testing.T) {
	m := New(config.Config{RefreshInterval: time.Second})
	m.treeView = true
	m.selectedPID = 10
	m.snap.Processes = []metrics.ProcessInfo{
		{PID: 20, PPID: 10, Name: "child"},
		{PID: 10, Name: "parent"},
		{PID: 30, Name: "other"},
	}

	m.moveSelection(1)

	if m.selectedPID != 20 {
		t.Fatalf("selected PID after moving down = %d, want 20", m.selectedPID)
	}
}

func TestSearchResolvesSelectionToVisibleProcess(t *testing.T) {
	m := New(config.Config{RefreshInterval: time.Second})
	m.searching = true
	m.selectedPID = 10
	m.snap.Processes = []metrics.ProcessInfo{
		{PID: 10, Name: "bash"},
		{PID: 20, Name: "zsh"},
	}

	updated, _ := m.handleSearchKey(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'z'},
	})
	m = updated.(Model)

	if m.selectedPID != 20 {
		t.Fatalf("selected PID after search filter = %d, want 20", m.selectedPID)
	}
}

func TestHideSystemResolvesSelectionToVisibleProcess(t *testing.T) {
	m := New(config.Config{
		RefreshInterval: time.Second,
		FilterUsers:     []string{"root"},
	})
	m.selectedPID = 10
	m.snap.Processes = []metrics.ProcessInfo{
		{PID: 10, Name: "launchd", User: "root"},
		{PID: 20, Name: "shell", User: "youri"},
	}

	updated, _ := m.handleKey(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'s'},
	})
	m = updated.(Model)

	if m.selectedPID != 20 {
		t.Fatalf("selected PID after hiding system processes = %d, want 20", m.selectedPID)
	}
}
