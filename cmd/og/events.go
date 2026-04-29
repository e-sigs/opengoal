package main

// Append-only event log for the goal tracker.
//
// Every state mutation (add, claim, release, done, delete) and every
// refused claim writes one JSON line to ~/.local/share/opencode/goals.events.jsonl.
// The log is intentionally separate from goals.json so:
//
//   1. Readers tailing the log don't compete for the state lock.
//   2. The log can grow indefinitely without bloating the snapshot.
//   3. A corrupted state file can be partly reconstructed from the log.
//
// Events are written from inside the same withLock() critical section
// that performs the state mutation, so on-disk log order matches the
// authoritative order of state changes. Cross-process visibility relies
// on O_APPEND being atomic for line-sized writes on darwin/linux — true
// for writes under the platform's atomic-write limit (PIPE_BUF, ≥512B).
// Each event line is well under that, so concurrent appends from
// different processes interleave cleanly without a separate lock.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ──────────────────────────────────────────────────────────────────
// Event schema
// ──────────────────────────────────────────────────────────────────

type Event struct {
	TS     time.Time      `json:"ts"`
	Actor  string         `json:"actor"`
	Event  string         `json:"event"`
	TaskID string         `json:"task_id,omitempty"`
	ListID string         `json:"list_id,omitempty"`
	Title  string         `json:"title,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
}

const (
	EvTaskAdded      = "task.added"
	EvTaskClaimed    = "task.claimed"
	EvTaskReleased   = "task.released"
	EvTaskCompleted  = "task.completed"
	EvTaskUnblocked  = "task.unblocked"
	EvTaskDeleted    = "task.deleted"
	EvClaimRefused   = "claim.refused"
	EvTaskCleared    = "task.cleared"
)

// ──────────────────────────────────────────────────────────────────
// File path
// ──────────────────────────────────────────────────────────────────

func getEventsFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fatalf("Error getting home directory: %v", err)
	}
	return filepath.Join(home, ".local", "share", "opencode", "goals.events.jsonl")
}

// ──────────────────────────────────────────────────────────────────
// Appending
// ──────────────────────────────────────────────────────────────────

// appendEvent writes one event to the log. Failure to write is reported
// to stderr but never aborts the calling command — the log is best-effort
// observability and must not block real state changes.
func appendEvent(ev Event) {
	if ev.TS.IsZero() {
		ev.TS = time.Now()
	}
	if ev.Actor == "" {
		ev.Actor = agentID()
	}

	path := getEventsFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: events dir: %v\n", err)
		return
	}

	line, err := json.Marshal(ev)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: events marshal: %v\n", err)
		return
	}
	line = append(line, '\n')

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: events open: %v\n", err)
		return
	}
	defer f.Close()
	if _, err := f.Write(line); err != nil {
		fmt.Fprintf(os.Stderr, "warning: events write: %v\n", err)
	}
}

// ──────────────────────────────────────────────────────────────────
// Reading / tailing
// ──────────────────────────────────────────────────────────────────

// showEvents prints events from the log. With follow=true, it keeps
// reading new events as they're appended (poll-based, 250ms cadence).
// since filters out events older than the given time. filter, if
// non-empty, keeps only events whose Event name contains the substring.
func showEvents(follow bool, since time.Time, filter string) {
	path := getEventsFilePath()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			if !follow {
				fmt.Fprintln(os.Stderr, "(no events recorded yet)")
				return
			}
			// Wait for the file to appear.
			for {
				time.Sleep(250 * time.Millisecond)
				f, err = os.Open(path)
				if err == nil {
					break
				}
				if !os.IsNotExist(err) {
					fatalf("Error opening events: %v", err)
				}
			}
		} else {
			fatalf("Error opening events: %v", err)
		}
	}
	defer f.Close()

	r := bufio.NewReader(f)

	emit := func(line string) {
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			// Skip malformed lines silently; log is best-effort.
			return
		}
		if !since.IsZero() && ev.TS.Before(since) {
			return
		}
		if filter != "" && !strings.Contains(ev.Event, filter) {
			return
		}
		printEvent(ev)
	}

	// Read backlog.
	for {
		line, err := r.ReadString('\n')
		if line != "" {
			emit(strings.TrimRight(line, "\n"))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fatalf("Error reading events: %v", err)
		}
	}

	if !follow {
		return
	}

	// Tail mode: poll for new lines. We're already at EOF on f; keep
	// reading. On rotation/truncation we'd need to reopen — skipped for
	// now since the log isn't rotated by this tool.
	for {
		line, err := r.ReadString('\n')
		if line != "" {
			emit(strings.TrimRight(line, "\n"))
			continue
		}
		if err == io.EOF {
			time.Sleep(250 * time.Millisecond)
			continue
		}
		if err != nil {
			fatalf("Error tailing events: %v", err)
		}
	}
}

// printEvent writes a single human-readable line for an event.
func printEvent(ev Event) {
	icon := eventIcon(ev.Event)
	ts := ev.TS.Local().Format("15:04:05")
	title := ev.Title
	if title == "" && ev.TaskID != "" {
		title = ev.TaskID
	}
	extra := ""
	if len(ev.Data) > 0 {
		// Render small data inline; skip noisy keys.
		var parts []string
		for k, v := range ev.Data {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		extra = " [" + strings.Join(parts, " ") + "]"
	}
	fmt.Printf("%s %s %-16s %-12s %s%s\n",
		ts, icon, ev.Event, ev.Actor, title, extra)
}

func eventIcon(name string) string {
	switch name {
	case EvTaskAdded:
		return "➕"
	case EvTaskClaimed:
		return "🔒"
	case EvTaskReleased:
		return "🔓"
	case EvTaskCompleted:
		return "✅"
	case EvTaskUnblocked:
		return "▶"
	case EvTaskDeleted:
		return "🗑️"
	case EvClaimRefused:
		return "❌"
	case EvTaskCleared:
		return "🧹"
	default:
		return "•"
	}
}
