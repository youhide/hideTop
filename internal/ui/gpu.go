package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/youhide/hideTop/internal/metrics/gpu"
)

// RenderGPU renders the GPU panel. Returns an empty string when GPU
// metrics are unavailable, causing no visual output.
func RenderGPU(stats *gpu.Stats, width int, history []float64) string {
	return RenderGPUCompact(stats, width, history, false)
}

// RenderGPUCompact renders the GPU panel, optionally dropping the per-engine
// bars and the frequency/temp/vram detail rows to save vertical space.
func RenderGPUCompact(stats *gpu.Stats, width int, history []float64, compact bool) string {
	if stats == nil {
		return ""
	}
	if !stats.Available {
		return ""
	}

	var b strings.Builder
	innerW := contentWidth(width)

	header := "GPU"
	if stats.Name != "" {
		header += "  " + stats.Name
	} else if stats.CoreCount > 0 {
		header += fmt.Sprintf("  %d cores", stats.CoreCount)
	}
	header = fitPlain(header, innerW)
	head := HeaderStyle.Render(header)
	headerWidth := lipgloss.Width(header)

	// Thermal indicator (inline, after header) — only on elevated states
	if innerW >= 32 && stats.ThermalOK && stats.Thermal > gpu.ThermalNominal {
		badge := thermalBadge(stats.Thermal)
		if headerWidth+2+lipgloss.Width(badge) <= innerW {
			head += "  " + badge
			headerWidth += 2 + lipgloss.Width(badge)
		}
	}

	// Energy impact (inline, after thermal)
	if innerW >= 32 && stats.Energy.Available {
		label := energyLabel(stats.Energy.Score)
		if headerWidth+2+lipgloss.Width(label) <= innerW {
			head += "  " + label
		}
	}

	// Total utilization bar (always shown, bold like CPU TOTAL)
	totalLabel := fmt.Sprintf("%-8s %5.1f%%", "TOTAL", stats.Utilization)
	b.WriteString(boldStyle.Render(renderBar(stats.Utilization, totalLabel, innerW)))
	b.WriteByte('\n')

	// Per-engine bars when available (Renderer, Tiler, etc.)
	if !compact && len(stats.Engines) > 0 {
		for _, eng := range stats.Engines {
			label := fmt.Sprintf("%-8s %5.1f%%", eng.Name, eng.Utilization)
			b.WriteString(renderBar(eng.Utilization, label, innerW))
			b.WriteByte('\n')
		}
	}

	// Frequency (shown only if collected)
	if !compact && stats.FrequencyMHz > 0 {
		b.WriteString(SubtleStyle.Render(
			fitPlain(fmt.Sprintf("  freq: %d MHz", stats.FrequencyMHz), innerW),
		))
		b.WriteByte('\n')
	}

	// GPU temperature (shown only if collected)
	if !compact && stats.Temperature > 0 {
		fmt.Fprintf(&b, "  temp: %s",
			levelText[tempLevel(stats.Temperature)].Render(fmt.Sprintf("%.0f°C", stats.Temperature)),
		)
		b.WriteByte('\n')
	}

	// VRAM usage (shown only for discrete GPUs)
	if !compact && stats.MemoryTotalMB > 0 {
		vram := gpu.FormatVRAM(stats.MemoryUsedMB, stats.MemoryTotalMB)
		b.WriteString(SubtleStyle.Render(fitPlain("  vram: "+vram, innerW)))
		b.WriteByte('\n')
	}

	// Sparkline history
	if len(history) > 1 {
		b.WriteString(RenderSparklineCompact("gpu", history, innerW))
		b.WriteByte('\n')
	}

	return renderPanel(head, "", b.String(), width)
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
