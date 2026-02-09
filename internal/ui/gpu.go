package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	if stats.CoreCount > 0 {
		b.WriteString(SubtleStyle.Render(fmt.Sprintf("  %d cores", stats.CoreCount)))
	}

	// Thermal indicator (inline, after header) — only on elevated states
	if stats.ThermalOK && stats.Thermal > gpu.ThermalNominal {
		b.WriteString("  ")
		b.WriteString(thermalBadge(stats.Thermal))
	}

	// Energy impact (inline, after thermal)
	if stats.Energy.Available {
		b.WriteString("  ")
		b.WriteString(energyLabel(stats.Energy.Score))
	}

	b.WriteByte('\n')

	// Total utilization bar (always shown, bold like CPU TOTAL)
	totalLabel := fmt.Sprintf("%-8s %5.1f%%", "TOTAL", stats.Utilization)
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(renderBar(stats.Utilization, totalLabel, width-4)))
	b.WriteByte('\n')

	// Per-engine bars when available (Renderer, Tiler, etc.)
	if len(stats.Engines) > 0 {
		for _, eng := range stats.Engines {
			label := fmt.Sprintf("%-8s %5.1f%%", eng.Name, eng.Utilization)
			b.WriteString(renderBar(eng.Utilization, label, width-4))
			b.WriteByte('\n')
		}
	}

	// Frequency (shown only if collected)
	if stats.FrequencyMHz > 0 {
		b.WriteString(SubtleStyle.Render(
			fmt.Sprintf("  freq: %d MHz", stats.FrequencyMHz),
		))
		b.WriteByte('\n')
	}

	return PanelStyle.Width(width - 2).Render(b.String())
}

// thermalBadge renders a small colored label for elevated thermal states.
func thermalBadge(state gpu.ThermalState) string {
	label := "thermal:" + state.String()
	switch state {
	case gpu.ThermalCritical:
		return lipgloss.NewStyle().Bold(true).Foreground(ColorRed).Render(label)
	case gpu.ThermalSerious:
		return lipgloss.NewStyle().Bold(true).Foreground(ColorYellow).Render(label)
	default:
		return SubtleStyle.Render(label)
	}
}

// energyLabel renders a compact energy impact score with color coding.
func energyLabel(score float64) string {
	text := fmt.Sprintf("energy %.0f", score)
	color := BarColor(score)
	return lipgloss.NewStyle().Foreground(color).Render(text)
}
