package ui

import (
	"fmt"
	"strings"

	"github.com/youhide/hideTop/internal/metrics/gpu"
)

// RenderGPU renders the GPU panel. Returns an empty string when GPU
// metrics are unavailable, causing no visual output.
func RenderGPU(raw interface{}, width int) string {
	if raw == nil {
		return ""
	}
	stats, ok := raw.(*gpu.Stats)
	if !ok || !stats.Available {
		return ""
	}

	var b strings.Builder

	b.WriteString(HeaderStyle.Render("GPU"))
	b.WriteByte('\n')

	// Utilization bar (always shown when available)
	utilLabel := fmt.Sprintf("util  %5.1f%%", stats.Utilization)
	b.WriteString(renderBar(stats.Utilization, utilLabel, width-4))
	b.WriteByte('\n')

	// Frequency (shown only if collected)
	if stats.FrequencyMHz > 0 {
		b.WriteString(SubtleStyle.Render(
			fmt.Sprintf("  freq: %d MHz", stats.FrequencyMHz),
		))
		b.WriteByte('\n')
	}

	return PanelStyle.Width(width).Render(b.String())
}
