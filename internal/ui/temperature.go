package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/youhide/hideTop/internal/metrics"
)

// Temperature panel sizing. The visible sensor count adapts to the terminal
// height rather than being fixed, so a tall terminal shows more sensors and a
// short one gives its rows back to the process list.
const (
	tempVisibleSensors = 6 // default when the caller does not constrain height
	TempMinRows        = 4
	TempMaxRows        = 16
)

// TemperatureTwoColumn reports whether the panel renders two sensors per line
// at the given panel width. Callers use it to convert a row budget into a
// sensor budget.
func TemperatureTwoColumn(width int) bool {
	return contentWidth(width) >= 44
}

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
	return TemperatureScrollMaxRows(temp, tempVisibleSensors)
}

// TemperatureScrollMaxRows is TemperatureScrollMax for a given visible row count.
func TemperatureScrollMaxRows(temp metrics.TemperatureStats, rows int) int {
	if !temp.Available {
		return 0
	}
	return max(0, len(temp.Sensors)-rows)
}

// RenderTemperature renders the temperature panel.
// Returns an empty string when no sensors are available.
func RenderTemperature(temp metrics.TemperatureStats, width int) string {
	s, _ := RenderTemperatureScroll(temp, width, 0)
	return s
}

// RenderTemperatureScroll renders with the default visible sensor count.
func RenderTemperatureScroll(temp metrics.TemperatureStats, width, scroll int) (string, int) {
	return RenderTemperatureScrollRows(temp, width, scroll, tempVisibleSensors)
}

// RenderTemperatureScrollRows renders the temperature panel starting at the
// given sensor scroll offset and showing at most rows sensors, returning the
// rendered panel and the clamped maximum scroll offset.
func RenderTemperatureScrollRows(temp metrics.TemperatureStats, width, scroll, rows int) (string, int) {
	if !temp.Available || len(temp.Sensors) == 0 {
		return "", 0
	}
	rows = min(max(rows, TempMinRows), TempMaxRows)

	total := len(temp.Sensors)
	maxScroll := TemperatureScrollMaxRows(temp, rows)
	scroll = min(max(scroll, 0), maxScroll)

	end := min(scroll+rows, total)
	window := temp.Sensors[scroll:end]

	innerW := contentWidth(width)

	// Title line carries an inline CPU/GPU summary.
	head := HeaderStyle.Render(fitPlain("Temperature", innerW))
	if innerW >= 28 && temp.CPUTemp > 0 {
		c := TempColor(temp.CPUTemp)
		head += fmt.Sprintf("  CPU %s",
			lipgloss.NewStyle().Foreground(c).Render(fmt.Sprintf("%.0f°C", temp.CPUTemp)))
	}
	if innerW >= 40 && temp.GPUTemp > 0 {
		c := TempColor(temp.GPUTemp)
		head += fmt.Sprintf("  GPU %s",
			lipgloss.NewStyle().Foreground(c).Render(fmt.Sprintf("%.0f°C", temp.GPUTemp)))
	}

	var b strings.Builder

	if !TemperatureTwoColumn(width) {
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

	return renderPanel(head, scrollStatus(scroll+1, end, total, maxScroll > 0),
		b.String(), width), maxScroll
}

// truncateSensorLabel truncates a sensor label for compact display.
func truncateSensorLabel(label string, maxLen int) string {
	return truncateRunes(label, maxLen)
}
