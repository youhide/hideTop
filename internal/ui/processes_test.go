package ui

import (
	"testing"
	"unicode/utf8"

	"github.com/youhide/hideTop/internal/metrics"
)

func TestTruncateRunes_PreservesUTF8(t *testing.T) {
	got := truncateRunes("日本語プロセス名", 6)
	if !utf8.ValidString(got) {
		t.Fatalf("result is not valid UTF-8: %q", got)
	}
	if got != "日本語..." {
		t.Fatalf("unexpected truncation: got %q", got)
	}
}

func TestTruncateRunes_ShortOrTinyLimit(t *testing.T) {
	if got := truncateRunes("abc", 5); got != "abc" {
		t.Fatalf("short string should stay unchanged: got %q", got)
	}
	if got := truncateRunes("abcdef", 3); got != "abc" {
		t.Fatalf("tiny limit should trim without ellipsis: got %q", got)
	}
}

func TestTreeDisplayOrderHelpers(t *testing.T) {
	procs := []metrics.ProcessInfo{
		{PID: 20, PPID: 10, Name: "child"},
		{PID: 10, Name: "parent"},
		{PID: 30, Name: "other"},
	}

	if got := DisplayIndexForPID(procs, true, 20, 0); got != 1 {
		t.Fatalf("tree display index for child = %d, want 1", got)
	}
	if got := DisplayIndexForPID(procs, false, 20, 0); got != 0 {
		t.Fatalf("flat display index for child = %d, want 0", got)
	}

	pid, ok := PIDAtDisplayIndex(procs, true, 0)
	if !ok {
		t.Fatalf("expected PID at tree display index 0")
	}
	if pid != 10 {
		t.Fatalf("PID at tree display index 0 = %d, want 10", pid)
	}
}
