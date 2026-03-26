package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Sparkline characters from lowest to highest, 8 levels.
var sparkChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// RenderSparkline renders a sparkline from a slice of values (0-100).
// maxWidth limits the number of characters rendered (uses last N values).
func RenderSparkline(values []float64, maxWidth int, color lipgloss.Color) string {
	if len(values) == 0 || maxWidth <= 0 {
		return ""
	}

	// Use only the most recent values that fit in maxWidth
	start := 0
	if len(values) > maxWidth {
		start = len(values) - maxWidth
	}
	vals := values[start:]

	var b strings.Builder
	for _, v := range vals {
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		idx := int(v / 100 * float64(len(sparkChars)-1))
		if idx >= len(sparkChars) {
			idx = len(sparkChars) - 1
		}
		b.WriteRune(sparkChars[idx])
	}

	return lipgloss.NewStyle().Foreground(color).Render(b.String())
}

// RenderSparklineCompact renders a sparkline with a label prefix.
func RenderSparklineCompact(label string, values []float64, maxWidth int) string {
	if len(values) == 0 {
		return ""
	}

	labelLen := len(label) + 1
	sparkWidth := maxWidth - labelLen
	if sparkWidth < 4 {
		sparkWidth = 4
	}

	// Determine color from latest value
	latest := values[len(values)-1]
	color := BarColor(latest)

	return label + " " + RenderSparkline(values, sparkWidth, color)
}
