package main

// Bordered-card rendering primitives shared by /today, /og, /ogl.

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const cardInnerWidth = 60 // chars between the left and right border

var ansiStripRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// isWideRune reports whether r is rendered as ~2 terminal cells but counts
// as a single rune. Covers the emoji blocks we actually use (status icons,
// markers, faces). Not exhaustive — good enough for our card titles.
func isWideRune(r rune) bool {
	switch {
	case r >= 0x1F300 && r <= 0x1FAFF: // misc symbols & pictographs, emoji
		return true
	case r >= 0x2600 && r <= 0x27BF: // misc symbols, dingbats (✅, ✓ is narrow but ✔ wide)
		return r == 0x2705 || r == 0x274C || r == 0x2728 || r == 0x274E ||
			(r >= 0x2614 && r <= 0x2615) || (r >= 0x26A0 && r <= 0x26FF)
	case r == 0x25B6: // ▶
		return false
	}
	return false
}

// visibleWidth returns the rendered cell width of s after stripping ANSI
// escapes. Wide runes (most emoji) count as 2.
func visibleWidth(s string) int {
	stripped := ansiStripRE.ReplaceAllString(s, "")
	w := 0
	for _, r := range stripped {
		if isWideRune(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// padRight right-pads s with spaces so its visible cell width equals width.
func padRight(s string, width int) string {
	w := visibleWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func boxTop(title string) string {
	t := " " + title + " "
	tw := visibleWidth(t)
	dashes := cardInnerWidth + 1 - tw
	if dashes < 0 {
		dashes = 0
	}
	return "╭─" + t + strings.Repeat("─", dashes) + "╮"
}

func boxBottom() string {
	return "╰" + strings.Repeat("─", cardInnerWidth+2) + "╯"
}

// boxLine renders "│ <content padded to inner width> │".
func boxLine(content string, _ int) string {
	return "│ " + padRight(truncateVisible(content, cardInnerWidth), cardInnerWidth) + " │"
}

// truncateVisible shortens s so its visible cell width fits within width,
// preserving ANSI escapes. If s is too long, append "…".
func truncateVisible(s string, width int) string {
	if visibleWidth(s) <= width {
		return s
	}
	// Walk runes, tracking visible width while keeping ANSI escapes inline.
	var b strings.Builder
	w := 0
	in := []byte(s)
	i := 0
	for i < len(in) {
		// Pass through ANSI escape unchanged.
		if in[i] == 0x1b && i+1 < len(in) && in[i+1] == '[' {
			j := i + 2
			for j < len(in) && in[j] != 'm' {
				j++
			}
			if j < len(in) {
				b.WriteString(string(in[i : j+1]))
				i = j + 1
				continue
			}
		}
		r, size := utf8.DecodeRune(in[i:])
		rw := 1
		if isWideRune(r) {
			rw = 2
		}
		if w+rw > width-1 { // leave room for ellipsis
			b.WriteRune('…')
			if colorOn() {
				b.WriteString(ansiReset)
			}
			return b.String()
		}
		b.WriteRune(r)
		w += rw
		i += size
	}
	return b.String()
}

func boxBlank() string {
	return "│" + strings.Repeat(" ", cardInnerWidth+2) + "│"
}

