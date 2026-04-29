package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ──────────────────────────────────────────────────────────────────
// Argument parsing for bulk-delete commands
// ──────────────────────────────────────────────────────────────────

// bulkArgs holds parsed flags for delete-style commands.
type bulkArgs struct {
	yes      bool   // -y, --yes
	all      bool   // --all
	priority string // --priority high|medium|low
	filter   string // --completed, --blocked, etc. (single name)
	ids      []string
}

// parseBulkArgs strips known flags from args and returns positional IDs.
// Unknown flags result in a hard error.
func parseBulkArgs(args []string, allowedFilters map[string]bool) bulkArgs {
	var b bulkArgs
	i := 0
	for i < len(args) {
		a := args[i]
		switch {
		case a == "-y" || a == "--yes":
			b.yes = true
		case a == "--all":
			b.all = true
		case a == "--priority" || a == "-p":
			if i+1 >= len(args) {
				die("Error: --priority requires a value (high|medium|low).")
			}
			p := strings.ToLower(args[i+1])
			if p != PriorityHigh && p != PriorityMedium && p != PriorityLow {
				die("Error: --priority must be one of high|medium|low; got %q.", args[i+1])
			}
			b.priority = p
			i++
		case strings.HasPrefix(a, "--"):
			name := strings.TrimPrefix(a, "--")
			if allowedFilters[name] {
				if b.filter != "" {
					die("Error: only one filter flag (--%s) allowed at a time.", b.filter)
				}
				b.filter = name
			} else {
				die("Error: unknown flag: %s", a)
			}
		default:
			b.ids = append(b.ids, a)
		}
		i++
	}
	return b
}

// ──────────────────────────────────────────────────────────────────
// Confirmation prompt
// ──────────────────────────────────────────────────────────────────

// confirmPrompt prints a message and waits for y/N. Returns true on yes.
// If skip is true (e.g. -y was passed), returns true without prompting.
// If stdin is not a TTY and skip is false, returns false with a stderr hint
// to use --yes for non-interactive invocations.
func confirmPrompt(message string, skip bool) bool {
	if skip {
		return true
	}
	if !isStdinTerminal() {
		fmt.Fprintln(os.Stderr, "\n⚠️  Refusing to perform a destructive action without confirmation.")
		fmt.Fprintln(os.Stderr, "   stdin is not a terminal; pass --yes (-y) to confirm non-interactively.")
		return false
	}
	fmt.Print(message + " [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}

// printSummaryList prints a numbered, bulleted preview of items to be acted on.
// Caps verbose output to first/last few items if the list is very long.
func printSummaryList(items []string, max int) {
	if max <= 0 {
		max = 10
	}
	if len(items) <= max {
		for _, it := range items {
			fmt.Printf("  • %s\n", it)
		}
		return
	}
	half := max / 2
	for _, it := range items[:half] {
		fmt.Printf("  • %s\n", it)
	}
	fmt.Printf("  … (%d more) …\n", len(items)-max)
	for _, it := range items[len(items)-half:] {
		fmt.Printf("  • %s\n", it)
	}
}
