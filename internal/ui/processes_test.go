package ui

import (
	"testing"
	"unicode/utf8"
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
