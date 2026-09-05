package ui

import (
	"slices"
	"strings"
	"testing"
)

func TestApplyThemeKeepsTitleSingleLine(t *testing.T) {
	ApplyTheme("dark")

	title := TitleStyle.Render("hideTop")
	if strings.Contains(title, "\n") {
		t.Fatalf("title rendered with an unexpected newline: %q", title)
	}
}

// TestApplyThemeRebuildsHotStyles guards the single-constructor invariant: the
// per-row style cache must be rebuilt by ApplyTheme, not just the shared
// styles. Styles used to be constructed twice, so a new one added in styles.go
// stayed on the dark palette under any other theme.
func TestApplyThemeRebuildsHotStyles(t *testing.T) {
	t.Cleanup(func() { ApplyTheme("dark") })

	for _, name := range AvailableThemes() {
		ApplyTheme(name)
		want := themes[name]
		if got := pctCellWide[levelRed].GetForeground(); got != want.Red {
			t.Errorf("%s: pctCellWide[red] foreground = %v, want %v", name, got, want.Red)
		}
		if got := selectedRow.GetForeground(); got != want.SelectedFg {
			t.Errorf("%s: selectedRow foreground = %v, want %v", name, got, want.SelectedFg)
		}
		if got := selectedRow.GetBackground(); got != want.SelectedBg {
			t.Errorf("%s: selectedRow background = %v, want %v", name, got, want.SelectedBg)
		}
		if want.SelectedFg == "" {
			t.Errorf("%s: theme has no SelectedFg; the selected row would be unreadable", name)
		}
	}
}

// TestAvailableThemesIsSorted pins the fix for ranging a map, which made the
// list in the "unknown theme" error message shuffle on every run.
func TestAvailableThemesIsSorted(t *testing.T) {
	got := AvailableThemes()
	if !slices.IsSorted(got) {
		t.Errorf("AvailableThemes() = %v, want sorted", got)
	}
}
