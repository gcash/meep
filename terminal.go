package main

import (
	"regexp"
	"strings"
	"unicode/utf8"

	tm "github.com/buger/goterm"
)

// ansiEscape matches the escape sequences that goterm writes for colors and
// cursor movement. They take no room on screen.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// fitFrame cuts frame to the rows that fit in a terminal of the given size,
// counting a long line as the rows it wraps onto. It also drops the final
// newline, so drawing the frame never scrolls its top off the screen. A size
// of zero or less means unknown and leaves the frame whole.
func fitFrame(frame string, width, height int) string {
	lines := strings.Split(strings.TrimSuffix(frame, "\n"), "\n")
	if width > 0 && height > 0 {
		rows := 0
		for i, line := range lines {
			n := utf8.RuneCountInString(ansiEscape.ReplaceAllString(line, ""))
			rows += max(1, (n+width-1)/width)
			if rows > height {
				lines = lines[:i]
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

// flush draws the frame in goterm's buffer, cut to fit the window, and empties
// the buffer. It replaces goterm's Flush, which stops without emptying the
// buffer when the frame is taller than the window and so freezes the display.
func flush() {
	_, _ = tm.Output.WriteString(fitFrame(tm.Screen.String(), tm.Width(), tm.Height()))
	_ = tm.Output.Flush()
	tm.Screen.Reset()
}
