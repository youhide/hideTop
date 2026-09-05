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

// TestWideCharProcessNamesDoNotOverflow pins the fix for %-Ns padding by runes
// while fitPlain trims by display cells: a CJK process name survived the trim
// and then had a full rune-count of spaces appended, overflowing the column,
// wrapping the row and growing the panel by a line.
func TestWideCharProcessNamesDoNotOverflow(t *testing.T) {
	procs := []metrics.ProcessInfo{
		{PID: 1, Name: "日本語プロセス名前テスト", User: "ユーザー", State: "S", CPUPercent: 1},
		{PID: 2, Name: "ascii-process", User: "youri", State: "S", CPUPercent: 2},
		{PID: 3, Name: "混合 mixed 名前", User: "u", State: "R", CPUPercent: 3},
	}
	// The symptom is height, not width: an over-wide row is wrapped by
	// lipgloss, which adds a line to the panel and pushes the layout down.
	for _, w := range []int{30, 50, 80, 120} {
		out := RenderProcesses(procs, ProcessViewState{TotalProcs: len(procs)}, w, len(procs))
		want := ProcessPanelChrome + len(procs)
		if got := strings.Count(out, "\n") + 1; got != want {
			t.Errorf("width %d: panel is %d lines, want %d (a row wrapped)\n%s", w, got, want, out)
		}
		for i, line := range strings.Split(out, "\n") {
			if got := lipgloss.Width(line); got > w {
				t.Errorf("width %d: line %d is %d cells wide:\n%q", w, i, got, line)
			}
		}
	}
}

// TestWideCharSensorLabelsDoNotOverflow is the same guard for the temperature
// panel, which pads sensor labels the same way.
func TestWideCharSensorLabelsDoNotOverflow(t *testing.T) {
	temp := metrics.TemperatureStats{Available: true, Sensors: []metrics.SensorReading{
		{Label: "温度センサー一番", Temperature: 40},
		{Label: "PMU tdie2", Temperature: 41},
	}}
	for _, w := range []int{30, 50, 80} {
		out, _ := RenderTemperatureScrollRows(temp, w, 0, 4)
		plain, _ := RenderTemperatureScrollRows(metrics.TemperatureStats{Available: true,
			Sensors: []metrics.SensorReading{{Label: "aaaaaaaa", Temperature: 40},
				{Label: "PMU tdie2", Temperature: 41}}}, w, 0, 4)
		if got, want := strings.Count(out, "\n"), strings.Count(plain, "\n"); got != want {
			t.Errorf("width %d: wide-char labels made the panel %d lines instead of %d\n%s",
				w, got+1, want+1, out)
		}
		for i, line := range strings.Split(out, "\n") {
			if got := lipgloss.Width(line); got > w {
				t.Errorf("width %d: line %d is %d cells wide:\n%q", w, i, got, line)
			}
		}
	}
}

// TestGPUHeaderShowsNameAndCores pins that both identities appear: the header
// used to be an either/or, so a named GPU hid its core count.
func TestGPUHeaderShowsNameAndCores(t *testing.T) {
	stats := &gpu.Stats{Available: true, Name: "Apple M5", CoreCount: 10, Utilization: 20}
	out := RenderGPU(stats, 80, nil)
	for _, want := range []string{"Apple M5", "10 cores"} {
		if !strings.Contains(out, want) {
			t.Errorf("GPU header missing %q:\n%s", want, out)
		}
	}
	// A GPU with no name must not render a stray separator.
	if got := RenderGPU(&gpu.Stats{Available: true, CoreCount: 10}, 80, nil); !strings.Contains(got, "GPU  10 cores") {
		t.Errorf("unnamed GPU header malformed:\n%s", got)
	}
}
