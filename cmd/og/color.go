package main

// Lightweight ANSI color helpers. Auto-disables when stdout is not a TTY
// (so piped/redirected output stays clean), when NO_COLOR is set
// (https://no-color.org), or when running under OpenCode (which renders
// shell-command output verbatim and would display escape codes as raw
// characters).
//
// Semantic roles map to a small palette so the look stays consistent:
//   Title    — section/board titles (bold cyan)
//   Heading  — section headers like "ACTIVE GOALS" (bold magenta)
//   Subtitle — item titles (default fg)
//   Caption  — IDs, dates, progress %, "ago" suffixes (dim)
//   Comment  — help text and hints (dim italic)
//   Success / Warn / Danger / Info for status accents.
//
// Priority colors map: high=danger, medium=warn, low=info.

import (
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
	colorOnce       sync.Once
	colorEnabled    bool
	plainOnce       sync.Once
	plainEnabled    bool
)

// plainTextOn reports whether output should be unstyled plain text
// (no ANSI escapes, suitable for OpenCode's chat UI which renders
// shell-command output verbatim). When plain mode is on, color is forced
// off.
//
// Resolution order:
//  1. OG_FORMAT=plain|text       → on
//     OG_FORMAT=ansi             → off (explicit)
//  2. OPENCODE=1 (running under OpenCode) → on
//  3. Otherwise off (let colorOn decide based on TTY/env).
func plainTextOn() bool {
	plainOnce.Do(func() {
		switch strings.ToLower(os.Getenv("OG_FORMAT")) {
		case "plain", "text":
			plainEnabled = true
			return
		case "ansi":
			plainEnabled = false
			return
		}
		if v := os.Getenv("OPENCODE"); v != "" && v != "0" {
			plainEnabled = true
			return
		}
		plainEnabled = false
	})
	return plainEnabled
}

// colorOn reports whether ANSI color output is enabled. The decision is
// cached on first call.
//
// Resolution order:
//  1. Plain-text mode forces ANSI off.
//  2. NO_COLOR (any non-empty value) → off. https://no-color.org
//  3. OG_COLOR=never|none|off|0       → off
//     OG_COLOR=always|force|on|1     → on (manual override)
//     OG_COLOR=auto|"" or unset      → auto
//  4. Auto: stdout is a real terminal.
func colorOn() bool {
	colorOnce.Do(func() {
		if plainTextOn() {
			colorEnabled = false
			return
		}
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

// Semantic helpers — keep callsites readable. When color is disabled
// (plain-text mode, non-TTY, NO_COLOR, etc.) wrap() returns s unchanged,
// so each helper safely degrades to plain text.
func cTitle(s string) string    { return wrap(ansiBoldCyan, s) }
func cHeading(s string) string  { return wrap(ansiBoldMag, s) }
func cSubtitle(s string) string { return s }
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
