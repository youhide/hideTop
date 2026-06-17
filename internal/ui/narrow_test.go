package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/youhide/hideTop/internal/metrics"
	"github.com/youhide/hideTop/internal/metrics/gpu"
)

func assertLinesWithinWidth(t *testing.T, rendered string, width int) {
	t.Helper()
	for _, line := range strings.Split(strings.TrimRight(rendered, "\n"), "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("line width = %d, want <= %d\nline: %q\nrendered:\n%s", got, width, line, rendered)
		}
	}
}

func TestRenderBarFitsNarrowWidths(t *testing.T) {
	for width := 1; width <= 30; width++ {
		got := renderBar(73.4, "TOTAL  73.4%  verbose label", width)
		if gotW := lipgloss.Width(got); gotW > width {
			t.Fatalf("bar width = %d, want <= %d: %q", gotW, width, got)
		}
	}
}

func TestMetricPanelsFitNarrowWidth(t *testing.T) {
	for _, width := range []int{20, 28, 40} {
		assertLinesWithinWidth(t, RenderCPU(metrics.CPUStats{
			PerCore: []float64{10, 40, 90, 12},
			Total:   38,
		}, width, []float64{10, 20, 30, 40}), width)

		assertLinesWithinWidth(t, RenderMemory(metrics.MemoryStats{
			TotalGB:     32,
			UsedGB:      12.4,
			Percent:     38.8,
			SwapTotalGB: 8,
			SwapUsedGB:  2,
			SwapPercent: 25,
		}, metrics.LoadAvg{Load1: 1.23, Load5: 2.34, Load15: 3.45}, width, []float64{10, 50}), width)

		assertLinesWithinWidth(t, RenderDisk(metrics.DiskDelta{
			Available: true,
			ReadSec:   1234567,
			WriteSec:  9876543,
		}, metrics.DiskStats{
			Available:   true,
			RootUsedGB:  111,
			RootTotalGB: 512,
			RootPercent: 21.7,
		}, width), width)

		assertLinesWithinWidth(t, RenderNetwork(metrics.NetworkDelta{
			Available:   true,
			TotalInSec:  1234567,
			TotalOutSec: 9876543,
			Interfaces: []metrics.InterfaceDelta{
				{Name: "en-super-long-name", InSec: 1234567, OutSec: 9876543},
			},
		}, width), width)

		assertLinesWithinWidth(t, RenderGPU(&gpu.Stats{
			Available:   true,
			Name:        "Very Long Integrated Graphics Device",
			Utilization: 64,
			Engines: []gpu.EngineStats{
				{Name: "RendererLong", Utilization: 77},
			},
			MemoryUsedMB:  2048,
			MemoryTotalMB: 8192,
		}, width, []float64{20, 60}), width)

		assertLinesWithinWidth(t, RenderTemperature(metrics.TemperatureStats{
			Available: true,
			CPUTemp:   64,
			Sensors: []metrics.SensorReading{
				{Label: "temperature sensor package", Temperature: 64},
				{Label: "gpu proximity", Temperature: 71},
			},
		}, width), width)
	}
}

func TestProcessesFitNarrowWidth(t *testing.T) {
	const width = 32
	rendered := RenderProcesses([]metrics.ProcessInfo{
		{PID: 12345, Name: "very-long-process-name-that-needs-truncation", User: "user", CPUPercent: 91.2, MemPercent: 12.3},
	}, ProcessViewState{SortBy: metrics.SortByCPU, SelectedPID: 12345}, width, 5)

	assertLinesWithinWidth(t, rendered, width)
}

func TestOverlaysFitNarrowWidth(t *testing.T) {
	const width = 30

	assertLinesWithinWidth(t, RenderHelp(width), width)
	assertLinesWithinWidth(t, RenderHelpOverlay(width, 20, "dev"), width)
	assertLinesWithinWidth(t, RenderProcessDetail(ProcessDetail{
		ProcessInfo: metrics.ProcessInfo{
			PID:        12345,
			PPID:       1,
			Name:       "very-long-process-name-that-needs-truncation",
			User:       "a-very-long-user-name",
			State:      "sleeping",
			CPUPercent: 12.3,
			MemPercent: 4.5,
		},
		Cmdline: "this-command-line-is-deliberately-long-and-needs-to-wrap",
	}, width, 20), width)
}
