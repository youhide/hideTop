package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/youhide/hideTop/internal/metrics"
)

// TempColor returns a color based on temperature thresholds.
func TempColor(temp float64) lipgloss.Color {
	switch {
	case temp > 80:
		return ColorRed
	case temp > 60:
		return ColorYellow
	default:
		return ColorGreen
	}
}

// RenderTemperature renders the temperature panel.
// Returns an empty string when no sensors are available.
func RenderTemperature(temp metrics.TemperatureStats, width int) string {
	if !temp.Available || len(temp.Sensors) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Temperature"))

	// Inline CPU/GPU summary in header line
	if temp.CPUTemp > 0 {
		c := TempColor(temp.CPUTemp)
		b.WriteString(fmt.Sprintf("  CPU %s",
			lipgloss.NewStyle().Foreground(c).Render(fmt.Sprintf("%.0f°C", temp.CPUTemp))))
	}
	if temp.GPUTemp > 0 {
		c := TempColor(temp.GPUTemp)
		b.WriteString(fmt.Sprintf("  GPU %s",
			lipgloss.NewStyle().Foreground(c).Render(fmt.Sprintf("%.0f°C", temp.GPUTemp))))
	}
	b.WriteByte('\n')

	// Compact two-column sensor grid (max 6 sensors, no bars)
	maxSensors := 6
	sensors := temp.Sensors
	if len(sensors) > maxSensors {
		sensors = sensors[:maxSensors]
	}

	colW := (width - 6) / 2
	if colW < 20 {
		colW = 20
	}

	for i := 0; i < len(sensors); i += 2 {
		s := sensors[i]
		c := TempColor(s.Temperature)
		left := fmt.Sprintf("  %-10s %s",
			truncateSensorLabel(s.Label, 10),
			lipgloss.NewStyle().Foreground(c).Render(fmt.Sprintf("%5.1f°C", s.Temperature)))

		if i+1 < len(sensors) {
			s2 := sensors[i+1]
			c2 := TempColor(s2.Temperature)
			right := fmt.Sprintf("  %-10s %s",
				truncateSensorLabel(s2.Label, 10),
				lipgloss.NewStyle().Foreground(c2).Render(fmt.Sprintf("%5.1f°C", s2.Temperature)))
			b.WriteString(left)
			// Pad left to column width, then add right
			pad := colW - lipgloss.Width(left)
			if pad > 0 {
				b.WriteString(strings.Repeat(" ", pad))
			}
			b.WriteString(right)
		} else {
			b.WriteString(left)
		}
		b.WriteByte('\n')
	}

	remaining := len(temp.Sensors) - maxSensors
	if remaining > 0 {
		b.WriteString(SubtleStyle.Render(fmt.Sprintf("  +%d more sensors", remaining)))
		b.WriteByte('\n')
	}

	return PanelStyle.Width(width - 2).Render(b.String())
}

// truncateSensorLabel truncates a sensor label for compact display.
func truncateSensorLabel(label string, maxLen int) string {
	if len(label) <= maxLen {
		return label
	}
	if maxLen <= 3 {
		return label[:maxLen]
	}
	return label[:maxLen-3] + "..."
}
