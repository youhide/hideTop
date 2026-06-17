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
	innerW := contentWidth(width)

	b.WriteString(HeaderStyle.Render(fitPlain("Temperature", innerW)))

	// Inline CPU/GPU summary in header line
	if innerW >= 28 && temp.CPUTemp > 0 {
		c := TempColor(temp.CPUTemp)
		b.WriteString(fmt.Sprintf("  CPU %s",
			lipgloss.NewStyle().Foreground(c).Render(fmt.Sprintf("%.0f°C", temp.CPUTemp))))
	}
	if innerW >= 40 && temp.GPUTemp > 0 {
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

	if innerW < 44 {
		for _, s := range sensors {
			c := TempColor(s.Temperature)
			labelW := innerW - 10
			if labelW < 1 {
				labelW = 1
			}
			line := fmt.Sprintf("  %-*s %s",
				labelW,
				fitPlain(s.Label, labelW),
				lipgloss.NewStyle().Foreground(c).Render(fmt.Sprintf("%4.0f°C", s.Temperature)))
			b.WriteString(line)
			b.WriteByte('\n')
		}
	} else {
		colW := (innerW - 2) / 2
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
				// Pad left to column width, then add right.
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
	}

	remaining := len(temp.Sensors) - maxSensors
	if remaining > 0 {
		msg := fmt.Sprintf("  +%d more sensors", remaining)
		b.WriteString(SubtleStyle.Render(fitPlain(msg, innerW)))
		b.WriteByte('\n')
	}

	return PanelStyle.Width(panelWidth(width)).Render(b.String())
}

// truncateSensorLabel truncates a sensor label for compact display.
func truncateSensorLabel(label string, maxLen int) string {
	return truncateRunes(label, maxLen)
}
