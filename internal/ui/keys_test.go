package ui

import (
	"os"
	"strings"
	"testing"
)

// TestBindingsHaveNoDuplicateKeyPerScope guards the table against two rows
// claiming the same key in the same scope, which would make dispatch ambiguous.
func TestBindingsHaveNoDuplicateKeyPerScope(t *testing.T) {
	type sk struct {
		scope Scope
		key   string
	}
	seen := map[sk]string{}
	for _, b := range Bindings {
		for _, k := range b.Keys {
			for _, scope := range []Scope{ScopeMain, ScopeOverlay, ScopeSearch} {
				if b.Scopes&scope == 0 {
					continue
				}
				id := sk{scope, k}
				if prev, dup := seen[id]; dup {
					t.Errorf("key %q in scope %d claimed by both %q and %q", k, scope, prev, b.Display)
				}
				seen[id] = b.Display
			}
		}
	}
}

// TestBindingsAreComplete checks the table is usable as documentation.
func TestBindingsAreComplete(t *testing.T) {
	for _, b := range Bindings {
		if b.Display == "" || b.Desc == "" || b.Section == "" {
			t.Errorf("incomplete binding: %+v", b)
		}
	}
}

// TestREADMEDocumentsAllBindings pins the README against the keymap. The
// shortcut table used to drift: Ctrl+F/Ctrl+B and click-to-select were
// implemented but undocumented.
func TestREADMEDocumentsAllBindings(t *testing.T) {
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Skipf("README not readable: %v", err)
	}
	readme := string(data)
	for _, b := range Bindings {
		if !strings.Contains(readme, b.Display) {
			t.Errorf("README does not document %q (%s)", b.Display, b.Desc)
		}
	}
}

// TestHelpOverlayCoversEveryBinding pins that the overlay is generated from
// the table rather than a hand-maintained literal that drifts.
func TestHelpOverlayCoversEveryBinding(t *testing.T) {
	out := strings.Join(helpOverlayLines(80), "\n")
	for _, b := range Bindings {
		if !strings.Contains(out, b.Display) {
			t.Errorf("help overlay omits %q", b.Display)
		}
	}
}
