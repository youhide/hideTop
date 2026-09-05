package ui

import "github.com/charmbracelet/lipgloss"

// Theme colors, exported for the few callers that need a raw color rather than
// a style. Assigned only by rebuildStyles.
var (
	ColorTitle      lipgloss.Color
	ColorGreen      lipgloss.Color
	ColorYellow     lipgloss.Color
	ColorRed        lipgloss.Color
	ColorSubtle     lipgloss.Color
	ColorBorder     lipgloss.Color
	ColorHeader     lipgloss.Color
	ColorSelectedBg lipgloss.Color
	ColorSelectedFg lipgloss.Color
)

// Shared styles. Never construct these anywhere but rebuildStyles: they used
// to be built twice, once here and once again in ApplyTheme, so adding a style
// in one place left it silently unthemed in the other.
var (
	TitleStyle  lipgloss.Style
	PanelStyle  lipgloss.Style
	HeaderStyle lipgloss.Style
	SubtleStyle lipgloss.Style
	GreenStyle  lipgloss.Style
	YellowStyle lipgloss.Style
	RedStyle    lipgloss.Style
)

// level indexes the small fixed set of colors every per-row style is keyed on.
// BarColor returns one of three and stateColor one of four, so the styles that
// used to be allocated per process row (four per row, ~50 rows a frame) reduce
// to a handful built once per theme.
type level uint8

const (
	levelGreen level = iota
	levelYellow
	levelRed
	levelSubtle
	numLevels
)

// Hot styles, indexed by level. Fixed arrays: no hashing, no allocation.
var (
	levelColor     [numLevels]lipgloss.Color
	levelText      [numLevels]lipgloss.Style // plain Foreground
	pctCellWide    [numLevels]lipgloss.Style // Width(8) right-aligned
	pctCellCompact [numLevels]lipgloss.Style // Width(5) right-aligned
	stateCell      [numLevels]lipgloss.Style // Width(2)
	boldLevel      [numLevels]lipgloss.Style // Bold + Foreground
)

// Level-independent hot styles.
var (
	barEmpty    lipgloss.Style
	boldStyle   lipgloss.Style
	thrCell     lipgloss.Style
	selectedRow lipgloss.Style
)

func init() { rebuildStyles(themes["dark"]) }

// rebuildStyles is the single place any style is constructed. A style added
// here is themed automatically; there is no second copy to keep in sync.
func rebuildStyles(t Theme) {
	ColorTitle, ColorGreen, ColorYellow, ColorRed = t.Title, t.Green, t.Yellow, t.Red
	ColorSubtle, ColorBorder, ColorHeader = t.Subtle, t.Border, t.Header
	ColorSelectedBg, ColorSelectedFg = t.SelectedBg, t.SelectedFg

	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorTitle)
	PanelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorHeader)
	SubtleStyle = lipgloss.NewStyle().Foreground(ColorSubtle)
	GreenStyle = lipgloss.NewStyle().Foreground(ColorGreen)
	YellowStyle = lipgloss.NewStyle().Foreground(ColorYellow)
	RedStyle = lipgloss.NewStyle().Foreground(ColorRed)

	levelColor = [numLevels]lipgloss.Color{ColorGreen, ColorYellow, ColorRed, ColorSubtle}
	for i := range numLevels {
		c := levelColor[i]
		levelText[i] = lipgloss.NewStyle().Foreground(c)
		boldLevel[i] = lipgloss.NewStyle().Bold(true).Foreground(c)
		pctCellWide[i] = lipgloss.NewStyle().Foreground(c).Width(8).Align(lipgloss.Right)
		pctCellCompact[i] = lipgloss.NewStyle().Foreground(c).Width(5).Align(lipgloss.Right)
		stateCell[i] = lipgloss.NewStyle().Foreground(c).Width(2)
	}
	barEmpty = lipgloss.NewStyle().Foreground(ColorBorder)
	boldStyle = lipgloss.NewStyle().Bold(true)
	thrCell = lipgloss.NewStyle().Foreground(ColorSubtle).Width(4).Align(lipgloss.Right)
	selectedRow = lipgloss.NewStyle().
		Background(ColorSelectedBg).
		Bold(true).
		Foreground(ColorSelectedFg)
}

// barLevel maps a percentage to a severity level.
func barLevel(pct float64) level {
	switch {
	case pct > 80:
		return levelRed
	case pct > 50:
		return levelYellow
	default:
		return levelGreen
	}
}

// tempLevel maps a temperature in degrees Celsius to a severity level.
func tempLevel(temp float64) level {
	switch {
	case temp > 80:
		return levelRed
	case temp > 60:
		return levelYellow
	default:
		return levelGreen
	}
}

// BarColor returns the color for a utilization percentage.
func BarColor(pct float64) lipgloss.Color { return levelColor[barLevel(pct)] }

// TempColor returns the color for a temperature reading.
func TempColor(temp float64) lipgloss.Color { return levelColor[tempLevel(temp)] }
