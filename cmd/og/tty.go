package main

import (
	"os"

	"golang.org/x/term"
)

// isStdinTerminal reports whether stdin is an interactive TTY.
// Uses golang.org/x/term for portability across darwin/linux/windows;
// returns false for pipes, files, and /dev/null.
func isStdinTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
