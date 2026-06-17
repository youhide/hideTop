package ui

import (
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
