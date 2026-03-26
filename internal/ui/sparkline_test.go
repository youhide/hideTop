package ui

import (
	"strings"
	"testing"

	"github.com/youhide/hideTop/internal/metrics"
)

func TestRenderSparkline_EmptyValues(t *testing.T) {
	result := RenderSparkline(nil, 20, "#FFFFFF")
	if result != "" {
		t.Errorf("expected empty string for nil values, got %q", result)
	}
}

func TestRenderSparkline_SingleValue(t *testing.T) {
	result := RenderSparkline([]float64{50}, 20, "#00FF00")
	if result == "" {
		t.Error("expected non-empty sparkline for single value")
	}
}

func TestRenderSparkline_Clamping(t *testing.T) {
	// Values outside 0-100 should be clamped
	result := RenderSparkline([]float64{-10, 150}, 20, "#FFFFFF")
	if result == "" {
		t.Error("expected non-empty sparkline for clamped values")
	}
}

func TestRenderSparkline_MaxWidth(t *testing.T) {
	values := make([]float64, 100)
	for i := range values {
		values[i] = float64(i)
	}
	result := RenderSparkline(values, 10, "#FFFFFF")
	// Should only show last 10 values worth of characters
	// (can't check exact rune count due to ANSI escapes, but string shouldn't be empty)
	if result == "" {
		t.Error("expected non-empty sparkline")
	}
}

func TestRenderSparklineCompact(t *testing.T) {
	values := []float64{10, 20, 30, 40, 50}
	result := RenderSparklineCompact("cpu", values, 40)
	if !strings.HasPrefix(result, "cpu ") {
		t.Errorf("expected result to start with 'cpu ', got %q", result)
	}
}

func TestBuildTreeDisplay_NoProcesses(t *testing.T) {
	result := buildTreeDisplay(nil)
	if len(result) != 0 {
		t.Errorf("expected empty tree for nil processes, got %d", len(result))
	}
}

func TestBuildTreeDisplay_FlatList(t *testing.T) {
	procs := []metrics.ProcessInfo{
		{PID: 1, PPID: 0, Name: "init"},
		{PID: 2, PPID: 0, Name: "kthread"},
	}
	result := buildTreeDisplay(procs)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result[0].prefix != "" || result[1].prefix != "" {
		t.Error("root processes should have no prefix")
	}
}

func TestBuildTreeDisplay_ParentChild(t *testing.T) {
	procs := []metrics.ProcessInfo{
		{PID: 1, PPID: 0, Name: "init"},
		{PID: 10, PPID: 1, Name: "bash"},
		{PID: 20, PPID: 1, Name: "sshd"},
	}
	result := buildTreeDisplay(procs)
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}
	// First should be root with no prefix
	if result[0].prefix != "" {
		t.Errorf("root should have empty prefix, got %q", result[0].prefix)
	}
	// bash should have tree prefix
	if result[1].prefix != "├─" {
		t.Errorf("first child should have ├─ prefix, got %q", result[1].prefix)
	}
	// sshd should have last-child prefix
	if result[2].prefix != "└─" {
		t.Errorf("last child should have └─ prefix, got %q", result[2].prefix)
	}
}

func TestRenderProcesses_TreeViewHeader(t *testing.T) {
	state := ProcessViewState{
		SortBy:      metrics.SortByCPU,
		SelectedIdx: -1,
		TreeView:    true,
		HideSystem:  true,
	}
	result := RenderProcesses(nil, state, 80, 10)
	if !strings.Contains(result, "[tree]") {
		t.Error("expected [tree] indicator in output")
	}
	if !strings.Contains(result, "[user]") {
		t.Error("expected [user] indicator in output")
	}
}
