package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/youhide/hideTop/internal/metrics"
)

// tempVisibleSensors is how many sensors are shown at once in the panel.
const tempVisibleSensors = 6

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

// TemperatureScrollMax returns the maximum scroll offset for the temperature
// panel (0 when all sensors fit on screen).
func TemperatureScrollMax(temp metrics.TemperatureStats) int {
	if !temp.Available {
		return 0
	}
	m := len(temp.Sensors) - tempVisibleSensors
	if m < 0 {
		return 0
	}
	return m
}

// RenderTemperature renders the temperature panel.
// Returns an empty string when no sensors are available.
func RenderTemperature(temp metrics.TemperatureStats, width int) string {
	s, _ := RenderTemperatureScroll(temp, width, 0)
	return s
}

// RenderTemperatureScroll renders the temperature panel starting at the given
// sensor scroll offset, returning the rendered panel and the clamped maximum
// scroll offset. Use scroll = 0 for the default (top) view.
func RenderTemperatureScroll(temp metrics.TemperatureStats, width, scroll int) (string, int) {
	if !temp.Available || len(temp.Sensors) == 0 {
		return "", 0
	}

	total := len(temp.Sensors)
	maxScroll := TemperatureScrollMax(temp)
	if scroll < 0 {
		scroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}

	end := scroll + tempVisibleSensors
	if end > total {
		end = total
	}
	window := temp.Sensors[scroll:end]

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

	if innerW < 44 {
		for _, s := range window {
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
		for i := 0; i < len(window); i += 2 {
			s := window[i]
			c := TempColor(s.Temperature)
			left := fmt.Sprintf("  %-10s %s",
				truncateSensorLabel(s.Label, 10),
				lipgloss.NewStyle().Foreground(c).Render(fmt.Sprintf("%5.1f°C", s.Temperature)))

			if i+1 < len(window) {
				s2 := window[i+1]
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

	// Constant-height scroll status: rendered whenever the panel is scrollable,
	// so scrolling never changes the panel's height and shifts its neighbours.
	if maxScroll > 0 {
		status := fmt.Sprintf("  %d-%d of %d sensors  (scroll)", scroll+1, end, total)
		b.WriteString(SubtleStyle.Render(fitPlain(status, innerW)))
		b.WriteByte('\n')
	}

	return PanelStyle.Width(panelWidth(width)).Render(b.String()), maxScroll
}

// truncateSensorLabel truncates a sensor label for compact display.
func truncateSensorLabel(label string, maxLen int) string {
	return truncateRunes(label, maxLen)
}
