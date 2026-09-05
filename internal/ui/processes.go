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
	SelectedPID int32
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

// ProcessSortAtX maps an x-coordinate clicked on the process column-header row
// to the sort field of that column. It is only meaningful in the wide
// (non-compact) layout, where the column positions are fixed; ok is false when
// the click did not land on a sortable column (PID, CPU% or MEM%).
//
// The relative column ranges below mirror the header built in RenderProcesses:
//
//	"  " PID(7) " " S(2) " " USER(10) " " NAME(20) " " THR(4) " " CPU%(8) " " MEM%(8)
func ProcessSortAtX(width, x int) (metrics.SortField, bool) {
	if contentWidth(width) < 68 {
		return 0, false
	}
	rel := x - (PanelStyle.GetBorderLeftSize() + PanelStyle.GetPaddingLeft())
	switch {
	case rel >= 2 && rel < 9:
		return metrics.SortByPID, true
	case rel >= 50 && rel < 58:
		return metrics.SortByCPU, true
	case rel >= 59 && rel < 67:
		return metrics.SortByMem, true
	}
	return 0, false
}

// Process panel chrome. RenderProcesses always emits these lines regardless of
// state and width: top border, title, column header, separator, a trailing
// blank content line, and the bottom border. Pinned by
// TestProcessPanelChromeMatchesRender.
const (
	// ProcessPanelChrome is every line the panel uses that is not a process row.
	ProcessPanelChrome = 5
	// ProcessPanelHeaderRows is how many of those sit above the first row.
	ProcessPanelHeaderRows = 4
)

func RenderProcesses(procs []metrics.ProcessInfo, state ProcessViewState, width, maxRows int) string {
	var b strings.Builder
	innerW := contentWidth(width)
	compact := innerW < 68

	// Header with optional search indicator
	head := HeaderStyle.Render("Processes")
	if len(procs) > 0 || state.TotalProcs > 0 {
		shown := len(procs)
		total := state.TotalProcs
		if total > 0 && total != shown {
			head += SubtleStyle.Render(fmt.Sprintf("  %d/%d", shown, total))
		} else {
			head += SubtleStyle.Render(fmt.Sprintf("  %d", shown))
		}
	}
	if state.TreeView {
		head += SubtleStyle.Render("  [tree]")
	}
	if state.HideSystem {
		head += SubtleStyle.Render("  [user]")
	}
	if state.SearchQuery != "" || state.Searching {
		cursor := ""
		if state.Searching {
			cursor = "█"
		}
		head += SubtleStyle.Render("  /" + state.SearchQuery + cursor)
	}

	if compact {
		b.WriteString(renderCompactProcessHeader(innerW))
	} else {
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
	}
	b.WriteByte('\n')

	sep := SubtleStyle.Render(strings.Repeat("─", innerW))
	b.WriteString(sep)
	b.WriteByte('\n')

	displayList := processDisplayList(procs, state.TreeView)
	selectedIdx := displayIndexForPID(displayList, state.SelectedPID, state.SelectedIdx)

	// Compute visible window that keeps selection on screen
	n := len(displayList)
	start := 0
	if maxRows > 0 && selectedIdx >= maxRows {
		start = selectedIdx - maxRows + 1
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

	for i := start; i < end; i++ {
		dp := displayList[i]
		line := renderProcessLine(dp, compact, innerW)

		if selectedIdx >= 0 && i == selectedIdx {
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

	return renderPanel(head, "", b.String(), width)
}

func renderCompactProcessHeader(width int) string {
	switch {
	case width >= 42:
		nameW := width - 22
		return fmt.Sprintf("  %-5s %-*s %5s %5s", "PID", nameW, "NAME", "CPU%", "MEM%")
	case width >= 26:
		nameW := width - 15
		return fmt.Sprintf("  %-5s %-*s %5s", "PID", nameW, "NAME", "CPU%")
	default:
		return fitPlain("  PID NAME", width)
	}
}

func renderProcessLine(dp displayProc, compact bool, width int) string {
	p := dp.proc
	cpuColor := BarColor(p.CPUPercent)
	memColor := BarColor(float64(p.MemPercent))

	if compact {
		name := dp.prefix + p.Name
		switch {
		case width >= 42:
			nameW := width - 22
			return fmt.Sprintf("  %-5d %-*s %s %s",
				p.PID,
				nameW,
				fitPlain(name, nameW),
				lipgloss.NewStyle().Foreground(cpuColor).Width(5).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f", p.CPUPercent)),
				lipgloss.NewStyle().Foreground(memColor).Width(5).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f", p.MemPercent)),
			)
		case width >= 26:
			nameW := width - 15
			return fmt.Sprintf("  %-5d %-*s %s",
				p.PID,
				nameW,
				fitPlain(name, nameW),
				lipgloss.NewStyle().Foreground(cpuColor).Width(5).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f", p.CPUPercent)),
			)
		default:
			return fitPlain(fmt.Sprintf("  %d %s", p.PID, name), width)
		}
	}

	user := truncateRunes(p.User, 10)
	name := fitPlain(dp.prefix+p.Name, 20)
	stateChar := stateLabel(p.State)

	thrStr := ""
	if p.NumThreads > 0 {
		thrStr = fmt.Sprintf("%d", p.NumThreads)
	}

	return fmt.Sprintf("  %-7d %s %-10s %-20s %s %s %s",
		p.PID,
		lipgloss.NewStyle().Foreground(stateColor(p.State)).Width(2).Render(stateChar),
		user,
		name,
		lipgloss.NewStyle().Foreground(ColorSubtle).Width(4).Align(lipgloss.Right).Render(thrStr),
		lipgloss.NewStyle().Foreground(cpuColor).Width(8).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f", p.CPUPercent)),
		lipgloss.NewStyle().Foreground(memColor).Width(8).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f", p.MemPercent)),
	)
}

// ProcessDisplayCount returns the number of processes in the rendered order.
func ProcessDisplayCount(procs []metrics.ProcessInfo, treeView bool) int {
	return len(processDisplayList(procs, treeView))
}

// DisplayIndexForPID returns a PID's index in the rendered order.
func DisplayIndexForPID(procs []metrics.ProcessInfo, treeView bool, pid int32, fallbackIdx int) int {
	return displayIndexForPID(processDisplayList(procs, treeView), pid, fallbackIdx)
}

// PIDAtDisplayIndex returns the PID at an index in the rendered order.
func PIDAtDisplayIndex(procs []metrics.ProcessInfo, treeView bool, idx int) (int32, bool) {
	displayList := processDisplayList(procs, treeView)
	if idx < 0 || idx >= len(displayList) {
		return 0, false
	}
	return displayList[idx].proc.PID, true
}

// displayProc wraps a process with a tree-indent prefix.
type displayProc struct {
	proc   metrics.ProcessInfo
	prefix string
}

func processDisplayList(procs []metrics.ProcessInfo, treeView bool) []displayProc {
	if treeView && len(procs) > 0 {
		return buildTreeDisplay(procs)
	}

	displayList := make([]displayProc, 0, len(procs))
	for _, p := range procs {
		displayList = append(displayList, displayProc{proc: p})
	}
	return displayList
}

func displayIndexForPID(displayList []displayProc, pid int32, fallbackIdx int) int {
	if pid > 0 {
		for i, dp := range displayList {
			if dp.proc.PID == pid {
				return i
			}
		}
	}

	if fallbackIdx < 0 || len(displayList) == 0 {
		return -1
	}
	if fallbackIdx >= len(displayList) {
		return len(displayList) - 1
	}
	return fallbackIdx
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
