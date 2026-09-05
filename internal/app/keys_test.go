package app

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/youhide/hideTop/internal/config"
)

func keyModel() Model {
	return New(config.Config{RefreshInterval: time.Second, ProcLimit: 50})
}

func press(m Model, k tea.KeyMsg) Model {
	next, _ := m.handleKey(k)
	return next.(Model)
}

func runes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// TestSearchAcceptsSpace pins the fix for Bubble Tea reporting a lone space as
// KeySpace rather than KeyRunes, which silently dropped it from the query.
func TestSearchAcceptsSpace(t *testing.T) {
	m := keyModel()
	m.searching = true
	for _, k := range []tea.KeyMsg{runes("Go"), {Type: tea.KeySpace}, runes("Chrome")} {
		next, _ := m.handleSearchKey(k)
		m = next.(Model)
	}
	if m.searchQuery != "Go Chrome" {
		t.Errorf("searchQuery = %q, want %q", m.searchQuery, "Go Chrome")
	}
}

// TestCtrlCQuitsFromOverlays pins that Ctrl+C is not swallowed by the overlay
// and kill-confirmation handlers — the help overlay advertises it as quit.
func TestCtrlCQuitsFromOverlays(t *testing.T) {
	ctrlC := tea.KeyMsg{Type: tea.KeyCtrlC}
	cases := map[string]func(*Model){
		"main":         func(m *Model) {},
		"help overlay": func(m *Model) { m.showHelp = true },
		"network view": func(m *Model) { m.showNetwork = true },
		"kill confirm": func(m *Model) { m.confirmKill = signalTerm; m.pidToKill = 1 },
		"search":       func(m *Model) { m.searching = true },
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			m := keyModel()
			setup(&m)
			if got := press(m, ctrlC); !got.quitting {
				t.Error("Ctrl+C did not quit")
			}
		})
	}
}

// TestIntervalStaysClamped pins both ends. The interval is persisted on quit,
// so an out-of-range value used to survive restarts.
func TestIntervalStaysClamped(t *testing.T) {
	t.Run("floor is not undershot from a non-multiple start", func(t *testing.T) {
		m := keyModel()
		m.cfg.RefreshInterval = 260 * time.Millisecond
		m = press(m, runes("-"))
		if m.cfg.RefreshInterval != minRefreshInterval {
			t.Errorf("interval = %v, want %v", m.cfg.RefreshInterval, minRefreshInterval)
		}
	})
	t.Run("ceiling is enforced", func(t *testing.T) {
		m := keyModel()
		for range 100 {
			m = press(m, runes("+"))
		}
		if m.cfg.RefreshInterval != maxRefreshInterval {
			t.Errorf("interval = %v, want %v", m.cfg.RefreshInterval, maxRefreshInterval)
		}
	})
	t.Run("a no-op does not dirty the config", func(t *testing.T) {
		m := keyModel()
		m.cfg.RefreshInterval = minRefreshInterval
		if m = press(m, runes("-")); m.intervalChanged {
			t.Error("intervalChanged set even though the interval did not move")
		}
	})
}

// TestPanelToggleKeys checks that 1-6 map to the panels in order.
func TestPanelToggleKeys(t *testing.T) {
	m := keyModel()
	m = press(m, runes("4")) // temp
	if m.panelVisible(panelTemp) {
		t.Error("key 4 did not hide the temperature panel")
	}
	if m = press(m, runes("4")); !m.panelVisible(panelTemp) {
		t.Error("key 4 did not restore the temperature panel")
	}
}
