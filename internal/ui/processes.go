package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/youhide/hideTop/internal/metrics"
)

// ProcessViewState holds pure rendering state for the process panel.
type ProcessViewState struct {
	SortBy      metrics.SortField
	SelectedIdx int // -1 = no selection
	SearchQuery string
	Searching   bool
	TreeView    bool
	HideSystem  bool
	TotalProcs  int // total process count before filtering
}

func columnHeader(label string, width int, align lipgloss.Position, sortBy, target metrics.SortField) string {
	indicator := ""
	if sortBy == target {
		switch target {
		case metrics.SortByPID:
			indicator = " ▲"
		default:
			indicator = " ▼"
		}
	}
	text := label + indicator
	style := lipgloss.NewStyle().Bold(true).Foreground(ColorHeader).Width(width).Align(align)
	if sortBy == target {
		style = style.Underline(true)
	}
	return style.Render(text)
}

func RenderProcesses(procs []metrics.ProcessInfo, state ProcessViewState, width, maxRows int) string {
	var b strings.Builder

	// Header with optional search indicator
	b.WriteString(HeaderStyle.Render("Processes"))
	if len(procs) > 0 || state.TotalProcs > 0 {
		shown := len(procs)
		total := state.TotalProcs
		if total > 0 && total != shown {
			b.WriteString(SubtleStyle.Render(fmt.Sprintf("  %d/%d", shown, total)))
		} else {
			b.WriteString(SubtleStyle.Render(fmt.Sprintf("  %d", shown)))
		}
	}
	if state.TreeView {
		b.WriteString(SubtleStyle.Render("  [tree]"))
	}
	if state.HideSystem {
		b.WriteString(SubtleStyle.Render("  [user]"))
	}
	if state.SearchQuery != "" || state.Searching {
		cursor := ""
		if state.Searching {
			cursor = "█"
		}
		b.WriteString(SubtleStyle.Render("  /" + state.SearchQuery + cursor))
	}
	b.WriteByte('\n')

	// Column headers with sort direction + underline on active column
	hdr := "  " +
		columnHeader("PID", 7, lipgloss.Left, state.SortBy, metrics.SortByPID) + " " +
		columnHeader("S", 2, lipgloss.Left, state.SortBy, metrics.SortField(-1)) + " " +
		columnHeader("USER", 10, lipgloss.Left, state.SortBy, metrics.SortField(-1)) + " " +
		columnHeader("NAME", 20, lipgloss.Left, state.SortBy, metrics.SortField(-1)) + " " +
		columnHeader("THR", 4, lipgloss.Right, state.SortBy, metrics.SortField(-1)) + " " +
		columnHeader("CPU%", 8, lipgloss.Right, state.SortBy, metrics.SortByCPU) + " " +
		columnHeader("MEM%", 8, lipgloss.Right, state.SortBy, metrics.SortByMem)
	b.WriteString(hdr)
	b.WriteByte('\n')

	sepWidth := width - 4
	if sepWidth < 1 {
		sepWidth = 1
	}
	sep := SubtleStyle.Render(strings.Repeat("─", sepWidth))
	b.WriteString(sep)
	b.WriteByte('\n')

	// Build display list: flat or tree
	var displayList []displayProc

	if state.TreeView && len(procs) > 0 {
		displayList = buildTreeDisplay(procs)
	} else {
		for _, p := range procs {
			displayList = append(displayList, displayProc{proc: p})
		}
	}

	// Compute visible window that keeps selection on screen
	n := len(displayList)
	start := 0
	if maxRows > 0 && state.SelectedIdx >= maxRows {
		start = state.SelectedIdx - maxRows + 1
	}
	end := n
	if maxRows > 0 {
		end = start + maxRows
	}
	if end > n {
		end = n
		if maxRows > 0 {
			start = end - maxRows
			if start < 0 {
				start = 0
			}
		}
	}

	innerW := width - 4
	for i := start; i < end; i++ {
		dp := displayList[i]
		p := dp.proc
		user := truncateRunes(p.User, 10)
		name := dp.prefix + truncateRunes(p.Name, 20-len(dp.prefix))

		cpuColor := BarColor(p.CPUPercent)
		memColor := BarColor(float64(p.MemPercent))

		stateChar := stateLabel(p.State)

		thrStr := ""
		if p.NumThreads > 0 {
			thrStr = fmt.Sprintf("%d", p.NumThreads)
		}

		line := fmt.Sprintf("  %-7d %s %-10s %-20s %s %s %s",
			p.PID,
			lipgloss.NewStyle().Foreground(stateColor(p.State)).Width(2).Render(stateChar),
			user,
			name,
			lipgloss.NewStyle().Foreground(ColorSubtle).Width(4).Align(lipgloss.Right).Render(thrStr),
			lipgloss.NewStyle().Foreground(cpuColor).Width(8).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f", p.CPUPercent)),
			lipgloss.NewStyle().Foreground(memColor).Width(8).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f", p.MemPercent)),
		)

		if state.SelectedIdx >= 0 && i == state.SelectedIdx {
			visible := lipgloss.Width(line)
			if visible < innerW {
				line += strings.Repeat(" ", innerW-visible)
			}
			line = strings.TrimPrefix(line, " ")
			line = lipgloss.NewStyle().
				Background(ColorSelectedBg).
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Render("▎" + line)
		}

		b.WriteString(line)
		b.WriteByte('\n')
	}

	return PanelStyle.Width(width - 2).Render(b.String())
}

// displayProc wraps a process with a tree-indent prefix.
type displayProc struct {
	proc   metrics.ProcessInfo
	prefix string
}

// buildTreeDisplay builds a tree-ordered display list from a flat process list.
func buildTreeDisplay(procs []metrics.ProcessInfo) []displayProc {
	// Build parent → children map
	pidSet := make(map[int32]bool)
	children := make(map[int32][]metrics.ProcessInfo)
	for _, p := range procs {
		pidSet[p.PID] = true
	}
	var roots []metrics.ProcessInfo
	for _, p := range procs {
		if !pidSet[p.PPID] || p.PPID == 0 {
			roots = append(roots, p)
		} else {
			children[p.PPID] = append(children[p.PPID], p)
		}
	}

	var result []displayProc
	var walk func(p metrics.ProcessInfo, indent string)
	walk = func(p metrics.ProcessInfo, indent string) {
		result = append(result, displayProc{proc: p, prefix: indent})
		kids := children[p.PID]
		for i, child := range kids {
			childIndent := "├─"
			if i == len(kids)-1 {
				childIndent = "└─"
			}
			walk(child, indent+childIndent)
		}
	}
	for _, root := range roots {
		walk(root, "")
	}
	return result
}

func truncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}

	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	if maxRunes <= 3 {
		return string(r[:maxRunes])
	}
	return string(r[:maxRunes-3]) + "..."
}

// stateLabel returns a short display character for a process state.
func stateLabel(state string) string {
	switch state {
	case "running":
		return "R"
	case "sleeping", "sleep", "idle":
		return "S"
	case "zombie":
		return "Z"
	case "stopped", "stop":
		return "T"
	case "disk-sleep":
		return "D"
	default:
		if len(state) > 0 {
			return string([]rune(state)[0:1])
		}
		return "?"
	}
}

// stateColor returns a color for a process state badge.
func stateColor(state string) lipgloss.Color {
	switch state {
	case "running":
		return ColorGreen
	case "zombie":
		return ColorRed
	case "stopped", "stop":
		return ColorYellow
	default:
		return ColorSubtle
	}
}
