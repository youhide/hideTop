package ui

import "strings"

// Scope is where a binding is active. The same key can mean different things
// in different scopes: q quits on the main screen but closes an overlay.
type Scope uint8

const (
	ScopeMain    Scope = 1 << iota // main screen, not searching, no overlay
	ScopeOverlay                   // help, process detail or network view
	ScopeSearch                    // incremental search input
)

// Binding is one row of the keymap: the single source of truth for the help
// overlay, the hint bar and the README shortcut table. Keys is nil for
// mouse-only actions, which are documented but never matched against a key.
type Binding struct {
	Scopes  Scope
	Keys    []string // tea.KeyMsg.String() values
	Display string   // human label, e.g. "↑ / k"
	Desc    string
	Section string // help-overlay group
	Hint    string // non-empty makes it eligible for the one-line hint bar
	// HintKeys overrides Display in the hint bar, where a compact label that
	// covers a pair of bindings ("↑↓/jk") reads better than one row's own.
	//
	// HintPri orders the bar: the lowest values are placed first and survive
	// longest as it narrows, so help and quit outlast the rest — they are what
	// a user who cannot find their way needs.
	HintKeys string
	HintPri  int // lower survives longer as the bar narrows
}

// Bindings documents every key the app handles. Adding a key here and nowhere
// else leaves it undispatched; adding it to the handler and not here fails
// TestREADMEDocumentsAllBindings. Before this table, Ctrl+F/Ctrl+B, the
// panel toggles and click-to-select were implemented but undocumented, while
// Ctrl+C in an overlay was documented but did not work.
var Bindings = []Binding{
	// Navigation
	{ScopeMain | ScopeOverlay, []string{"up", "k"}, "↑ / k", "Move up", "Navigation", "move", "↑↓/jk", 30},
	{ScopeMain | ScopeOverlay, []string{"down", "j"}, "↓ / j", "Move down", "Navigation", "", "", 0},
	{ScopeMain | ScopeOverlay, []string{"pgup", "ctrl+b"}, "PgUp / Ctrl+B", "Jump one page up", "Navigation", "", "", 0},
	{ScopeMain | ScopeOverlay, []string{"pgdown", "ctrl+f"}, "PgDn / Ctrl+F", "Jump one page down", "Navigation", "", "", 0},
	{ScopeMain | ScopeOverlay, []string{"home", "g"}, "Home / g", "Jump to first", "Navigation", "", "", 0},
	{ScopeMain | ScopeOverlay, []string{"end", "G"}, "End / G", "Jump to last", "Navigation", "", "", 0},
	{ScopeMain, nil, "Wheel", "Scroll the process list; over Temp/Net panels scrolls those", "Navigation", "", "", 0},
	{ScopeMain, nil, "Click", "Select a process row", "Navigation", "", "", 0},
	{ScopeMain, []string{"/"}, "/", "Start incremental search (name, PID or user)", "Navigation", "search", "", 40},
	{ScopeSearch, []string{"esc"}, "Esc", "Cancel search", "Navigation", "", "", 0},
	{ScopeMain, []string{"enter"}, "Enter", "Open process detail", "Navigation", "", "", 0},
	{ScopeOverlay, []string{"esc", "q"}, "Esc / q", "Close the overlay", "Navigation", "", "", 0},

	// Sorting
	{ScopeMain, []string{"c"}, "c", "Sort by CPU% (descending)", "Sorting", "sort", "c/m/p", 50},
	{ScopeMain, []string{"m"}, "m", "Sort by MEM% (descending)", "Sorting", "", "", 0},
	{ScopeMain, []string{"p"}, "p", "Sort by PID (ascending)", "Sorting", "", "", 0},
	{ScopeMain, nil, "Click header", "Click a PID/CPU%/MEM% column header to sort", "Sorting", "", "", 0},

	// Process actions
	{ScopeMain, []string{"t"}, "t", "Toggle tree view", "Process actions", "", "", 0},
	{ScopeMain, []string{"s"}, "s", "Toggle the system process filter", "Process actions", "", "", 0},
	{ScopeMain, []string{"x"}, "x", "Terminate the selected process (SIGTERM)", "Process actions", "", "", 0},
	{ScopeMain, []string{"K"}, "K", "Force kill the selected process (SIGKILL)", "Process actions", "", "", 0},
	{ScopeMain, []string{"e"}, "e", "Export a snapshot to JSON", "Process actions", "", "", 0},

	// Display
	{ScopeMain, []string{"n"}, "n", "Open the network / ports view", "Display", "net", "", 60},
	{ScopeMain, []string{"1", "2", "3", "4", "5", "6"}, "1 – 6", "Show/hide the CPU, GPU, memory, temperature, network and disk panels", "Display", "", "", 0},
	{ScopeMain, []string{" "}, "Space", "Pause / resume auto-refresh", "Display", "", "", 0},
	{ScopeMain, []string{"z"}, "z", "Reset the Temp/Network panel scroll", "Display", "", "", 0},
	{ScopeMain, []string{"+", "="}, "+ / =", "Increase the refresh interval (+250ms)", "Display", "", "", 0},
	{ScopeMain, []string{"-", "_"}, "- / _", "Decrease the refresh interval (-250ms)", "Display", "", "", 0},
	{ScopeMain | ScopeOverlay, []string{"?"}, "?", "Toggle this help overlay", "Display", "more", "", 20},
	{ScopeMain, []string{"q"}, "q", "Quit", "Display", "quit", "", 10},
	{ScopeMain | ScopeOverlay | ScopeSearch, []string{"ctrl+c"}, "Ctrl+C", "Quit from anywhere", "Display", "", "", 0},
}

// sections returns the help-overlay section names in first-appearance order.
func sections() []string {
	var out []string
	seen := map[string]bool{}
	for _, b := range Bindings {
		if !seen[b.Section] {
			seen[b.Section] = true
			out = append(out, b.Section)
		}
	}
	return out
}

// BindingsMarkdown renders the keymap as a Markdown table for the README.
func BindingsMarkdown() string {
	var b strings.Builder
	b.WriteString("| Key | Action |\n| --- | --- |\n")
	for _, bind := range Bindings {
		b.WriteString("| `" + bind.Display + "` | " + bind.Desc + " |\n")
	}
	return b.String()
}
