package main

// Lightweight ANSI color helpers. Auto-disables when stdout is not a TTY
// (so piped/redirected output stays clean) or when NO_COLOR is set
// (https://no-color.org).
//
// Semantic roles map to a small palette so the look stays consistent:
//   Title    — section/board titles (bold cyan)
//   Heading  — section headers like "ACTIVE GOALS" (bold magenta)
//   Subtitle — item titles (default white, no escape — readable as-is)
//   Caption  — IDs, dates, progress %, "ago" suffixes (dim)
//   Comment  — help text and hints (dim italic)
//   Success / Warn / Danger / Info for status accents.
//
// Priority colors map: high=danger, medium=warn, low=info.

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

const (
	ansiReset     = "\033[0m"
	ansiBold      = "\033[1m"
	ansiDim       = "\033[2m"
	ansiItalic    = "\033[3m"
	ansiRed       = "\033[31m"
	ansiGreen     = "\033[32m"
	ansiYellow    = "\033[33m"
	ansiBlue      = "\033[34m"
	ansiMagenta   = "\033[35m"
	ansiCyan      = "\033[36m"
	ansiWhite     = "\033[37m"
	ansiBoldCyan  = "\033[1;36m"
	ansiBoldMag   = "\033[1;35m"
	ansiBoldGreen = "\033[1;32m"
)

var (
	colorOnce    sync.Once
	colorEnabled bool
)

// colorOn reports whether ANSI color output is enabled. The decision is
// cached on first call.
//
// Resolution order:
//  1. NO_COLOR (any non-empty value) → off. https://no-color.org
//  2. OG_COLOR=never|none|off|0       → off
//     OG_COLOR=always|force|on|1     → on (used by OpenCode slash commands,
//                                     which capture stdout through a pipe)
//     OG_COLOR=auto|"" or unset      → auto (TTY detection)
//  3. Auto: stdout is a real terminal.
func colorOn() bool {
	colorOnce.Do(func() {
		if v, ok := os.LookupEnv("NO_COLOR"); ok && v != "" {
			colorEnabled = false
			return
		}
		switch strings.ToLower(os.Getenv("OG_COLOR")) {
		case "never", "none", "off", "0", "false":
			colorEnabled = false
			return
		case "always", "force", "on", "1", "true":
			colorEnabled = true
			return
		}
		colorEnabled = term.IsTerminal(int(os.Stdout.Fd()))
	})
	return colorEnabled
}

// wrap surrounds s with the given ANSI escape and a reset. Returns s
// unchanged when color is disabled.
func wrap(escape, s string) string {
	if !colorOn() || escape == "" {
		return s
	}
	return escape + s + ansiReset
}

// Semantic helpers — keep callsites readable.
func cTitle(s string) string    { return wrap(ansiBoldCyan, s) }
func cHeading(s string) string  { return wrap(ansiBoldMag, s) }
func cSubtitle(s string) string { return s } // default fg; reserved for future tweaks
func cCaption(s string) string  { return wrap(ansiDim, s) }
func cComment(s string) string  { return wrap(ansiDim+ansiItalic, s) }
func cSuccess(s string) string  { return wrap(ansiGreen, s) }
func cWarn(s string) string     { return wrap(ansiYellow, s) }
func cDanger(s string) string   { return wrap(ansiRed, s) }
func cInfo(s string) string     { return wrap(ansiBlue, s) }
func cBold(s string) string     { return wrap(ansiBold, s) }
func cDim(s string) string      { return wrap(ansiDim, s) }

// cPriority returns the task title styled by its priority bucket.
func cPriority(priority, s string) string {
	switch priority {
	case PriorityHigh:
		return cDanger(s)
	case PriorityMedium:
		return cWarn(s)
	case PriorityLow:
		return cInfo(s)
	}
	return s
}

// cf is sprintf + color wrap convenience. Not used heavily; kept for
// readability in a few formatted-then-styled spots.
func cf(escape, format string, args ...any) string {
	return wrap(escape, fmt.Sprintf(format, args...))
}
