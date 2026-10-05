package textutil

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

func TestTruncateKeepsRunesWhole(t *testing.T) {
	s := strings.Repeat("Привет мир ", 7) // 77 runes, 2 bytes each
	got := Truncate(s, 60)
	if !utf8.ValidString(got) {
		t.Fatalf("invalid UTF-8: %q", got)
	}
	if runewidth.StringWidth(got) > 60 || !strings.HasSuffix(got, "…") {
		t.Fatalf("got %q (width %d)", got, runewidth.StringWidth(got))
	}
}

func TestTruncateShortUnchanged(t *testing.T) {
	if got := Truncate("short", 10); got != "short" {
		t.Fatalf("got %q", got)
	}
}

func TestPadRightWideChars(t *testing.T) {
	got := PadRight("日本語", 10) // 6 columns
	if runewidth.StringWidth(got) != 10 {
		t.Fatalf("width %d", runewidth.StringWidth(got))
	}
	if got := PadRight("😀😀😀😀😀😀", 5); runewidth.StringWidth(got) > 5 {
		t.Fatalf("width %d for %q", runewidth.StringWidth(got), got)
	}
}
