package textutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateKeepsRunesWhole(t *testing.T) {
	s := strings.Repeat("Привет мир ", 7) // 77 runes, 2 bytes each
	got := Truncate(s, 60)
	if !utf8.ValidString(got) {
		t.Fatalf("invalid UTF-8: %q", got)
	}
	if Width(got) > 60 || !strings.HasSuffix(got, "…") {
		t.Fatalf("got %q (width %d)", got, Width(got))
	}
}

func TestTruncateShortUnchanged(t *testing.T) {
	if got := Truncate("short", 10); got != "short" {
		t.Fatalf("got %q", got)
	}
}

func TestPadRightWideChars(t *testing.T) {
	got := PadRight("日本語", 10) // 6 columns
	if Width(got) != 10 {
		t.Fatalf("width %d", Width(got))
	}
	if got := PadRight("😀😀😀😀😀😀", 5); Width(got) > 5 {
		t.Fatalf("width %d for %q", Width(got), got)
	}
}
