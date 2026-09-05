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

	// Both icons are U+1F5xx, which every terminal draws two cells wide. The
	// charging icon used to be ⚡ (U+26A1), which go-runewidth calls Wide but
	// many terminals draw one cell, so lipgloss.Width disagreed with reality
	// and everything after the battery chip sat one column off.
	icon := "🔋"
	if bat.Charging {
		icon = "🔌"
	}

	color := BarColor(100 - bat.Percent) // invert: low battery = red
	// %3.0f keeps the width constant across 9% -> 100%.
	label := fmt.Sprintf("%s %s",
		icon,
		lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%3.0f%%", bat.Percent)),
	)

	if bat.Status != "" {
		label += SubtleStyle.Render(" " + strings.ToLower(bat.Status))
	}

	return label
}
