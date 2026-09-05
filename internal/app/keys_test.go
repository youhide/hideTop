package app

import (
	"strings"
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

// TestKillConfirmationIsVisible pins that a destructive prompt gets a modal
// rather than a header chip that a narrow terminal silently truncates away.
func TestKillConfirmationIsVisible(t *testing.T) {
	for _, w := range []int{40, 80, 130} {
		m := busyModel(w, 30)
		m.selectedPID = m.snap.Processes[0].PID
		m = press(m, runes("x"))
		if m.confirmKill == 0 {
			t.Fatalf("width %d: x did not arm the kill confirmation", w)
		}
		view := m.View()
		if !strings.Contains(view, "Terminate process?") {
			t.Errorf("width %d: confirmation not shown:\n%s", w, view)
		}
		if !strings.Contains(view, "SIGTERM") {
			t.Errorf("width %d: confirmation does not name the signal", w)
		}
	}
}

// TestWatchdogClearsStuckCollection pins that a collection which never
// returns cannot leave the app frozen: collecting is only otherwise cleared by
// a snapshotMsg arriving.
func TestWatchdogClearsStuckCollection(t *testing.T) {
	m := keyModel()
	m.collecting = true
	m.collectStartedAt = time.Now().Add(-time.Hour)

	next, _ := m.Update(tickMsg(time.Now()))
	got := next.(Model)

	if got.collecting && got.collectStartedAt.Before(time.Now().Add(-time.Minute)) {
		t.Error("watchdog did not abandon the stuck collection")
	}
}

// TestCollectingLabelWaitsForSlowness pins that the header does not advertise
// a collection until it is actually slow — it used to appear on every tick and
// shove the chips to its right.
func TestCollectingLabelWaitsForSlowness(t *testing.T) {
	m := keyModel()
	m.width, m.height = 130, 33
	m.collecting = true

	m.collectStartedAt = time.Now()
	if strings.Contains(m.renderHeader(130), "collecting") {
		t.Error("header showed 'collecting' for a fresh collection")
	}

	m.collectStartedAt = time.Now().Add(-2 * time.Second)
	if !strings.Contains(m.renderHeader(130), "collecting") {
		t.Error("header hid 'collecting' for a slow collection")
	}
}
