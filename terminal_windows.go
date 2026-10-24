package main

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// errNoTerminal is returned when the debugger can't draw on standard output.
// That is the case in Git Bash's default MinTTY window, which isn't a Windows
// console.
var errNoTerminal = errors.New("the debugger needs a Windows console, but standard output is not a console. In Git Bash, start meep with winpty (winpty meep debug ...), or use Windows Terminal, PowerShell or cmd.exe")

// enableVirtualTerminal turns on escape sequence processing for standard
// output, which classic Windows consoles leave off, so that colors and screen
// clearing work. The returned function restores the previous mode.
func enableVirtualTerminal() func() {
	out := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if windows.GetConsoleMode(out, &mode) != nil {
		return func() {}
	}
	if windows.SetConsoleMode(out, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) != nil {
		return func() {}
	}
	return func() { _ = windows.SetConsoleMode(out, mode) }
}
