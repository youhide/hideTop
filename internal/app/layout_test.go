package app

import (
	"strings"
	"testing"

	"github.com/youhide/hideTop/internal/config"
	"github.com/youhide/hideTop/internal/metrics"
	"github.com/youhide/hideTop/internal/ui"
)

// busyModel builds a model with every metric panel populated, which is the
// worst case for the height budget.
func busyModel(w, h int) Model {
	sensors := make([]metrics.SensorReading, 40)
	for i := range sensors {
		sensors[i] = metrics.SensorReading{Label: "sensor", Temperature: 40}
	}
	procs := make([]metrics.ProcessInfo, 50)
	for i := range procs {
		procs[i] = metrics.ProcessInfo{PID: int32(i + 1), Name: "proc", User: "u", State: "S"}
	}

	m := New(config.Config{RefreshInterval: 1e9, ProcLimit: 50})
	m.width, m.height = w, h
	m.snap = metrics.Snapshot{
		CPU:         metrics.CPUStats{Total: 25, PerCore: make([]float64, 10)},
		Memory:      metrics.MemoryStats{Percent: 50, UsedGB: 12, TotalGB: 24},
		Disk:        metrics.DiskStats{Available: true, RootTotalGB: 900, RootUsedGB: 450, RootPercent: 50},
		Temperature: metrics.TemperatureStats{Available: true, CPUTemp: 40, Sensors: sensors},
		Processes:   procs,
	}
	m.netDelta = metrics.NetworkDelta{
		Available:  true,
		TotalInSec: 1000, TotalOutSec: 2000,
		Interfaces: []metrics.InterfaceDelta{{Name: "en0", InSec: 1000, OutSec: 2000}},
	}
	m.conns = metrics.NetConnections{Available: true, Listening: []metrics.PortInfo{
		{Port: 3722, Proto: "udp", Process: "rapportd"},
		{Port: 5000, Proto: "tcp", Process: "ControlCenter"},
	}}
	return m
}

// TestProcessListKeepsUsableHeight guards the height budget: the metric panels
// used to take whatever they wanted, leaving three process rows on a typical
// 33-row terminal.
func TestProcessListKeepsUsableHeight(t *testing.T) {
	for _, h := range []int{28, 33, 40, 60} {
		m := busyModel(130, h)
		rows := strings.Count(m.View(), "\n") + 1
		if rows > h {
			t.Errorf("h=%d: view is %d lines, taller than the terminal", h, rows)
		}
		procRows, _ := m.procViewport()
		if procRows < 5 {
			t.Errorf("h=%d: process list got %d rows, want at least 5", h, procRows)
		}
		t.Logf("h=%2d -> %2d process rows", h, procRows)
	}
}

// TestPanelToggleFreesHeight checks that hiding panels gives the space to the
// process list.
func TestPanelToggleFreesHeight(t *testing.T) {
	m := busyModel(130, 33)
	before, _ := m.procViewport()

	m.togglePanel(panelTemp)
	m.togglePanel(panelNet)
	after, _ := m.procViewport()

	if after <= before {
		t.Errorf("hiding temp+net gave %d process rows, was %d; expected more", after, before)
	}
	if got := m.hiddenPanelList(); len(got) != 2 {
		t.Errorf("hiddenPanelList() = %v, want two entries", got)
	}
	if strings.Contains(m.View(), "Temperature") {
		t.Error("hidden temperature panel still rendered")
	}
	t.Logf("hiding temp+net: %d -> %d process rows", before, after)
}

// TestNetworkPanelHeightIsStable pins the fix for interfaces dropping out of
// the list when they go idle, which resized the panel and shifted everything
// below it every second.
func TestNetworkPanelHeightIsStable(t *testing.T) {
	busy := metrics.NetworkDelta{Available: true, Interfaces: []metrics.InterfaceDelta{
		{Name: "en0", InSec: 100, OutSec: 200},
		{Name: "lo0", InSec: 5, OutSec: 5},
	}}
	idle := metrics.NetworkDelta{Available: true, Interfaces: []metrics.InterfaceDelta{
		{Name: "en0", InSec: 100, OutSec: 200},
		{Name: "lo0", InSec: 0, OutSec: 0},
	}}
	a, _ := ui.RenderNetworkScrollRows(busy, nil, 60, 0, ui.NetMinRows)
	b, _ := ui.RenderNetworkScrollRows(idle, nil, 60, 0, ui.NetMinRows)
	if ha, hb := strings.Count(a, "\n"), strings.Count(b, "\n"); ha != hb {
		t.Errorf("panel height changed when an interface went idle: %d vs %d lines", ha+1, hb+1)
	}
}
