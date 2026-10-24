//go:build !windows

package main

import "errors"

// errNoTerminal is returned when the debugger can't draw on standard output.
var errNoTerminal = errors.New("the debugger needs a terminal, but standard output is not a terminal")

// enableVirtualTerminal does nothing outside Windows, where terminals already
// process escape sequences.
func enableVirtualTerminal() func() {
	return func() {}
}
