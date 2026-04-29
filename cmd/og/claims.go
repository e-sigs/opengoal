package main

// Coordination primitives for multi-agent use of the goal tracker.
//
// This adds two things on top of the existing read/write helpers:
//
//  1. withLock(fn): an exclusive flock on goals.json.lock that serializes
//     read-modify-write sequences across processes. Atomic file replacement
//     (writeGoals) protects readers from torn writes, but it does NOT prevent
//     two concurrent claimers from both observing "unclaimed" and racing.
//     withLock closes that race.
//
//  2. Task claim helpers (Assignee / ClaimedAt) so agents can safely pick up
//     work without stepping on each other. Stale claims expire after a TTL,
//     so a crashed agent never blocks the queue forever.
//
// The Task struct itself has the new optional fields added in main.go.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// ──────────────────────────────────────────────────────────────────
// Configuration
// ──────────────────────────────────────────────────────────────────

const defaultClaimTTLSecs = 1800 // 30 min

// claimTTL returns the active claim TTL in seconds. Override with
// $OPENGOAL_CLAIM_TTL (parsed as a positive integer; invalid values fall
// back to the default).
func claimTTL() time.Duration {
	if v := strings.TrimSpace(os.Getenv("OPENGOAL_CLAIM_TTL")); v != "" {
		var secs int
		if _, err := fmt.Sscanf(v, "%d", &secs); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return time.Duration(defaultClaimTTLSecs) * time.Second
}

// agentID resolves the current agent's identity. Resolution order:
//
//	$OPENGOAL_AGENT  →  hostname-pid (fallback)
//
// The fallback ensures every claim is attributable even when an agent
// forgets to set the env var.
func agentID() string {
	if id := strings.TrimSpace(os.Getenv("OPENGOAL_AGENT")); id != "" {
		return id
	}
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}

// ──────────────────────────────────────────────────────────────────
// Cross-process locking
// ──────────────────────────────────────────────────────────────────

// withLock takes an exclusive advisory lock on goals.json.lock for the
// duration of fn. Implemented per-OS (see lock_unix.go, lock_windows.go).
// Wrap every read-modify-write sequence with this.

// ──────────────────────────────────────────────────────────────────
// Claim state helpers
// ──────────────────────────────────────────────────────────────────

// claimActive reports whether the task currently has a live (non-stale,
// non-completed) claim.
func claimActive(t Task, ttl time.Duration) bool {
	if t.Completed || t.Assignee == "" || t.ClaimedAt == nil {
		return false
	}
	return time.Since(*t.ClaimedAt) <= ttl
}

// claimStatusLabel returns a short human-readable label for display, or
// "" if the task is not claimed in any meaningful way.
func claimStatusLabel(t Task, ttl time.Duration) string {
	if t.Completed || t.Assignee == "" || t.ClaimedAt == nil {
		return ""
	}
	age := time.Since(*t.ClaimedAt)
	if age > ttl {
		return fmt.Sprintf("⚠ stale claim by %s (%s ago)", t.Assignee, formatAge(age))
	}
	return fmt.Sprintf("🔒 claimed by %s (%s ago)", t.Assignee, formatAge(age))
}

func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}

// ──────────────────────────────────────────────────────────────────
// Dependency helpers
// ──────────────────────────────────────────────────────────────────

// findTaskByID returns a pointer into the slice for in-place edits, or
// nil if no task with that ID exists.
func findTaskByID(tasks []Task, id string) *Task {
	for i := range tasks {
		if tasks[i].ID == id {
			return &tasks[i]
		}
	}
	return nil
}

// blockedDeps returns the IDs of unsatisfied dependencies for t. A dep
// is unsatisfied if it doesn't exist (orphan) or is not Completed.
// Returns nil/empty when the task is ready to run.
func blockedDeps(t Task, all []Task) []string {
	if len(t.DependsOn) == 0 {
		return nil
	}
	var blocked []string
	for _, depID := range t.DependsOn {
		dep := findTaskByID(all, depID)
		if dep == nil {
			blocked = append(blocked, depID+" (missing)")
			continue
		}
		if !dep.Completed {
			blocked = append(blocked, depID+" ("+dep.Title+", "+depStatus(*dep)+")")
		}
	}
	return blocked
}

func depStatus(t Task) string {
	if t.Completed {
		return "completed"
	}
	if t.Assignee != "" {
		return "in progress by " + t.Assignee
	}
	return "pending"
}

// ──────────────────────────────────────────────────────────────────
// Commands
// ──────────────────────────────────────────────────────────────────

// claimTask atomically claims a task for the current agent. Refuses if
// the task is already claimed by someone else within the TTL, completed,
// or not found.
func claimTask(taskID string) {
	withLock(func() {
		data := readGoals()
		ttl := claimTTL()
		me := agentID()

		idx := -1
		for i := range data.Tasks {
			if data.Tasks[i].ID == taskID {
				idx = i
				break
			}
		}
		if idx == -1 {
			die("Error: Task with ID %q not found.", taskID)
		}
		t := &data.Tasks[idx]

		if t.Completed {
			die("Task %q is already completed.", t.Title)
		}
		if blocked := blockedDeps(*t, data.Tasks); len(blocked) > 0 {
			appendEvent(Event{
				Actor: me, Event: EvClaimRefused,
				TaskID: t.ID, ListID: t.ListID, Title: t.Title,
				Data: map[string]any{"reason": "blocked", "blocked_by": blocked},
			})
			fmt.Fprintf(os.Stderr, "\n❌ Task is blocked by unfinished dependencies:\n")
			for _, b := range blocked {
				fmt.Fprintf(os.Stderr, "   • %s\n", b)
			}
			fmt.Fprintln(os.Stderr)
			os.Exit(1)
		}
		if claimActive(*t, ttl) && t.Assignee != me {
			age := time.Since(*t.ClaimedAt)
			appendEvent(Event{
				Actor: me, Event: EvClaimRefused,
				TaskID: t.ID, ListID: t.ListID, Title: t.Title,
				Data: map[string]any{"reason": "already_claimed", "holder": t.Assignee, "age": age.String()},
			})
			die("Task is already claimed by %s (%s ago, TTL %s). Wait or override after expiry.",
				t.Assignee, formatAge(age), ttl)
		}

		now := time.Now()
		t.Assignee = me
		t.ClaimedAt = &now
		writeGoals(data)
		appendEvent(Event{
			Actor: me, Event: EvTaskClaimed,
			TaskID: t.ID, ListID: t.ListID, Title: t.Title,
		})

		fmt.Printf("\n🔒 Claimed task: %q\n", t.Title)
		fmt.Printf("   ID: %s\n", t.ID)
		fmt.Printf("   Agent: %s\n", me)
		fmt.Printf("   TTL: %s (auto-released if not completed/refreshed)\n\n", ttl)
	})
}

// releaseTask clears a claim. Only the holding agent can release a live
// claim; anyone may release one whose TTL has expired.
func releaseTask(taskID string) {
	withLock(func() {
		data := readGoals()
		ttl := claimTTL()
		me := agentID()

		idx := -1
		for i := range data.Tasks {
			if data.Tasks[i].ID == taskID {
				idx = i
				break
			}
		}
		if idx == -1 {
			die("Error: Task with ID %q not found.", taskID)
		}
		t := &data.Tasks[idx]

		if t.Assignee == "" {
			fmt.Printf("\nℹ️  Task %q has no claim to release.\n\n", t.Title)
			return
		}
		if claimActive(*t, ttl) && t.Assignee != me {
			die("Task is held by %s (live). Only that agent or a TTL expiry can release it.", t.Assignee)
		}

		prev := t.Assignee
		t.Assignee = ""
		t.ClaimedAt = nil
		writeGoals(data)
		appendEvent(Event{
			Actor: me, Event: EvTaskReleased,
			TaskID: t.ID, ListID: t.ListID, Title: t.Title,
			Data: map[string]any{"prev_holder": prev},
		})

		fmt.Printf("\n🔓 Released claim on %q (was held by %s)\n\n", t.Title, prev)
	})
}

// nextTask prints the next pending, unclaimed task in the active list.
// With autoClaim=true, atomically claims it in the same locked section
// (so two agents calling task-next --claim concurrently are guaranteed
// to receive different tasks).
//
// Selection order: priority (high → medium → low → none), then creation
// time ascending. Stale claims are treated as unclaimed.
func nextTask(autoClaim bool) {
	withLock(func() {
		data := readGoals()
		requireActiveList(data)
		ttl := claimTTL()
		me := agentID()

		candidates := []int{}
		for i, t := range data.Tasks {
			if t.ListID != data.ActiveListID {
				continue
			}
			if t.Completed {
				continue
			}
			if claimActive(t, ttl) {
				continue
			}
			if len(blockedDeps(t, data.Tasks)) > 0 {
				continue
			}
			candidates = append(candidates, i)
		}

		if len(candidates) == 0 {
			fmt.Fprintln(os.Stderr, "\nℹ️  No actionable tasks (all completed, claimed, or list empty).")
			os.Exit(1)
		}

		priorityRank := func(p string) int {
			switch p {
			case PriorityHigh:
				return 0
			case PriorityMedium:
				return 1
			case PriorityLow:
				return 2
			default:
				return 3
			}
		}
		sort.SliceStable(candidates, func(a, b int) bool {
			ta, tb := data.Tasks[candidates[a]], data.Tasks[candidates[b]]
			pa, pb := priorityRank(ta.Priority), priorityRank(tb.Priority)
			if pa != pb {
				return pa < pb
			}
			return ta.Created.Before(tb.Created)
		})

		pick := &data.Tasks[candidates[0]]

		if autoClaim {
			now := time.Now()
			pick.Assignee = me
			pick.ClaimedAt = &now
			writeGoals(data)
			appendEvent(Event{
				Actor: me, Event: EvTaskClaimed,
				TaskID: pick.ID, ListID: pick.ListID, Title: pick.Title,
				Data: map[string]any{"via": "task-next"},
			})
			fmt.Printf("\n🔒 Claimed next task: %q\n", pick.Title)
			fmt.Printf("   ID: %s\n", pick.ID)
			if pick.Priority != "" {
				fmt.Printf("   Priority: %s\n", pick.Priority)
			}
			fmt.Printf("   Agent: %s\n", me)
			fmt.Printf("   TTL: %s\n\n", ttl)
			return
		}

		fmt.Printf("\n▶ Next task: %q\n", pick.Title)
		fmt.Printf("   ID: %s\n", pick.ID)
		if pick.Priority != "" {
			fmt.Printf("   Priority: %s\n", pick.Priority)
		}
		fmt.Printf("\nClaim it with:  og task-claim %s\n", pick.ID)
		fmt.Printf("Or atomically:  og task-next --claim\n\n")
	})
}

// showTask prints a detailed view of a single task: status, claim,
// dependencies (with their current status), and timing.
func showTask(taskID string) {
	data := readGoals()
	ttl := claimTTL()

	t := findTaskByID(data.Tasks, taskID)
	if t == nil {
		die("Error: Task with ID %q not found.", taskID)
	}

	fmt.Printf("\n📋 Task: %s\n", t.Title)
	fmt.Printf("   ID: %s\n", t.ID)
	fmt.Printf("   List: %s\n", t.ListID)
	fmt.Printf("   Created: %s\n", t.Created.Format("2006-01-02 15:04"))

	switch {
	case t.Completed:
		when := ""
		if t.CompletedAt != nil {
			when = " at " + t.CompletedAt.Format("2006-01-02 15:04")
		}
		fmt.Printf("   Status: ✅ completed%s\n", when)
	case claimActive(*t, ttl):
		fmt.Printf("   Status: 🔒 in progress\n")
		fmt.Printf("   Held by: %s (%s ago)\n", t.Assignee, formatAge(time.Since(*t.ClaimedAt)))
	case t.Assignee != "" && t.ClaimedAt != nil:
		fmt.Printf("   Status: ⚠ stale claim\n")
		fmt.Printf("   Was held by: %s (%s ago, TTL %s)\n",
			t.Assignee, formatAge(time.Since(*t.ClaimedAt)), ttl)
	default:
		fmt.Printf("   Status: ⏳ pending\n")
	}

	if t.Priority != "" {
		fmt.Printf("   Priority: %s\n", t.Priority)
	}

	if len(t.DependsOn) > 0 {
		fmt.Printf("   Dependencies:\n")
		for _, depID := range t.DependsOn {
			dep := findTaskByID(data.Tasks, depID)
			if dep == nil {
				fmt.Printf("     • %s — ⚠ missing\n", depID)
				continue
			}
			mark := "⏳"
			if dep.Completed {
				mark = "✅"
			} else if claimActive(*dep, ttl) {
				mark = "🔒"
			}
			fmt.Printf("     %s %s — %s\n", mark, depID, dep.Title)
		}
		blocked := blockedDeps(*t, data.Tasks)
		if len(blocked) > 0 && !t.Completed {
			fmt.Printf("   Blocked: yes (%d unfinished dep%s)\n", len(blocked), plural(len(blocked)))
		} else if !t.Completed {
			fmt.Printf("   Blocked: no — ready to claim\n")
		}
	}

	fmt.Println()
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
