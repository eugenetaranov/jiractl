// Package textutil handles terminal text width: truncation never splits a
// character, and padding is measured in display columns, not bytes.
package textutil

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// Truncate shortens s to at most cols display columns, ending with "…" when
// anything was cut.
func Truncate(s string, cols int) string {
	if runewidth.StringWidth(s) <= cols {
		return s
	}
	return runewidth.Truncate(s, cols, "…")
}

// PadRight truncates s to cols columns and pads it with spaces to exactly
// cols columns.
func PadRight(s string, cols int) string {
	return runewidth.FillRight(Truncate(s, cols), cols)
}

// Wrap breaks s into lines of at most cols display columns, keeping existing
// line breaks.
func Wrap(s string, cols int) []string {
	if cols < 1 {
		cols = 1
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		out = append(out, strings.Split(runewidth.Wrap(line, cols), "\n")...)
	}
	return out
}
