//go:build unix

package tui

import (
	"os"

	"golang.org/x/sys/unix"
)

// ensureWindowSize gives a terminal that reports a 0x0 window (some ptys,
// e.g. expect's) a usable 80x24 size. Bubble Tea always uses the size the
// terminal reports and renders nothing at 0x0.
func ensureWindowSize() {
	fd := int(os.Stderr.Fd())
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil || (ws.Row > 0 && ws.Col > 0) {
		return
	}
	_ = unix.IoctlSetWinsize(fd, unix.TIOCSWINSZ, &unix.Winsize{Row: 24, Col: 80})
}
