package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/youhide/hideTop/internal/metrics"
)

// RenderBattery renders the battery indicator. Returns empty if no battery.
func RenderBattery(bat metrics.BatteryStats) string {
	if !bat.Available {
		return ""
	}

	icon := "🔋"
	if bat.Charging {
		icon = "⚡"
	}

	color := BarColor(100 - bat.Percent) // invert: low battery = red
	label := fmt.Sprintf("%s %s",
		icon,
		lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%.0f%%", bat.Percent)),
	)

	if bat.Status != "" {
		label += SubtleStyle.Render(" " + strings.ToLower(bat.Status))
	}

	return label
}
