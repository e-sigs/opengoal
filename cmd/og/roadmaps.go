package main

// Roadmap operations: CRUD on top-level Roadmap entries plus the helpers
// that maintain the active-roadmap invariant and migrate legacy data.

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ──────────────────────────────────────────────────────────────────
// Active roadmap state machine + migration
// ──────────────────────────────────────────────────────────────────

// ensureActiveRoadmap syncs ActiveRoadmapID with the current Roadmaps slice.
// It does NOT auto-create lists. Returns true if data was mutated.
func ensureActiveRoadmap(data *GoalsData) bool {
	if len(data.Roadmaps) == 0 {
		if data.ActiveRoadmapID != "" {
			data.ActiveRoadmapID = ""
			return true
		}
		return false
	}
	for _, l := range data.Roadmaps {
		if l.ID == data.ActiveRoadmapID {
			return false
		}
	}
	// Active list missing or unset → fall back to first list.
	data.ActiveRoadmapID = data.Roadmaps[0].ID
	return true
}

// migrateOrphanRoadmapIDs assigns the active list's ID to any goal/sub-goal/task
// that predates multi-list support and has an empty RoadmapID. This is a
// one-shot, idempotent migration; once persisted, future reads are no-ops.
// If there is no active list, nothing is migrated.
func migrateOrphanRoadmapIDs(data *GoalsData) bool {
	if data.ActiveRoadmapID == "" {
		return false
	}
	mutated := false
	for i := range data.MainGoals {
		if data.MainGoals[i].RoadmapID == "" {
			data.MainGoals[i].RoadmapID = data.ActiveRoadmapID
			mutated = true
		}
	}
	for i := range data.SubGoals {
		if data.SubGoals[i].RoadmapID == "" {
			data.SubGoals[i].RoadmapID = data.ActiveRoadmapID
			mutated = true
		}
	}
	for i := range data.Tasks {
		if data.Tasks[i].RoadmapID == "" {
			data.Tasks[i].RoadmapID = data.ActiveRoadmapID
			mutated = true
		}
	}
	return mutated
}

// requireActiveRoadmap errors out if no list exists. Use before any goal/task mutation.
func requireActiveRoadmap(data GoalsData) {
	if len(data.Roadmaps) == 0 || data.ActiveRoadmapID == "" {
		fmt.Fprintf(os.Stderr, "\n❌ No roadmaps exist yet.\n")
		fmt.Fprintf(os.Stderr, "   Create one first:  og list-create <name>\n\n")
		os.Exit(1)
	}
}

// findRoadmap returns index of list matching id or name (case-insensitive), or -1.
func findRoadmap(data GoalsData, idOrName string) int {
	for i, l := range data.Roadmaps {
		if l.ID == idOrName {
			return i
		}
	}
	for i, l := range data.Roadmaps {
		if strings.EqualFold(l.Name, idOrName) {
			return i
		}
	}
	return -1
}

func activeRoadmapName(data GoalsData) string {
	for _, l := range data.Roadmaps {
		if l.ID == data.ActiveRoadmapID {
			return l.Name
		}
	}
	return "(none)"
}

// ──────────────────────────────────────────────────────────────────
// Generic per-list filter
// ──────────────────────────────────────────────────────────────────

// filterByRoadmap returns all items whose list_id equals listID.
// After migration, every item has a list_id, so no fallback is needed.
func filterByRoadmap[T any](items []T, getListID func(T) string, listID string) []T {
	out := make([]T, 0, len(items))
	for _, it := range items {
		if getListID(it) == listID {
			out = append(out, it)
		}
	}
	return out
}

// ──────────────────────────────────────────────────────────────────
// Roadmap commands
// ──────────────────────────────────────────────────────────────────

func listRoadmaps() {
	data := readGoals()

	fmt.Println("\n📍 Roadmaps")
	printSeparator()

	if len(data.Roadmaps) == 0 {
		fmt.Println("\n  No roadmaps yet.")
		fmt.Printf("  Create your first roadmap:  og list-create <name>\n\n")
		return
	}

	for _, l := range data.Roadmaps {
		marker := "  "
		if l.ID == data.ActiveRoadmapID {
			marker = "▶ "
		}
		mainCount := len(goalsForRoadmap(data, l.ID))
		subCount := len(subGoalsForRoadmap(data, l.ID))
		listTasksLocal := tasksForRoadmap(data, l.ID)
		taskCount := len(listTasksLocal)
		pendingTasks := 0
		for _, t := range listTasksLocal {
			if !t.Completed {
				pendingTasks++
			}
		}
		fmt.Printf("\n%s%s\n", marker, l.Name)
		fmt.Printf("    ID: %s\n", l.ID)
		fmt.Printf("    Goals: %d | Sub-goals: %d | Tasks: %d (%d pending)\n",
			mainCount, subCount, taskCount, pendingTasks)
	}

	fmt.Println()
	printSeparator()
	fmt.Printf("\nActive: %s\n\n", activeRoadmapName(data))
}

func listCreate(name string) {
	data := readGoals()

	for _, l := range data.Roadmaps {
		if strings.EqualFold(l.Name, name) {
			die("Error: A roadmap named %q already exists.", name)
		}
	}

	newList := Roadmap{
		ID:      generateID("list"),
		Name:    name,
		Created: time.Now(),
	}
	data.Roadmaps = append(data.Roadmaps, newList)
	data.ActiveRoadmapID = newList.ID
	writeGoals(data)

	fmt.Printf("\n✅ Created roadmap: %q (now active)\n", name)
	fmt.Printf("   ID: %s\n\n", newList.ID)
}

func listUse(idOrName string) {
	data := readGoals()
	idx := findRoadmap(data, idOrName)
	if idx == -1 {
		die("Error: Roadmap %q not found.", idOrName)
	}

	data.ActiveRoadmapID = data.Roadmaps[idx].ID
	writeGoals(data)
	fmt.Printf("\n✅ Active roadmap: %s\n\n", data.Roadmaps[idx].Name)
}

func listRename(idOrName, newName string) {
	data := readGoals()
	idx := findRoadmap(data, idOrName)
	if idx == -1 {
		die("Error: Roadmap %q not found.", idOrName)
	}
	for i, l := range data.Roadmaps {
		if i != idx && strings.EqualFold(l.Name, newName) {
			die("Error: A roadmap named %q already exists.", newName)
		}
	}
	old := data.Roadmaps[idx].Name
	data.Roadmaps[idx].Name = newName
	writeGoals(data)
	fmt.Printf("\n✅ Renamed roadmap: %q → %q\n\n", old, newName)
}

func listDelete(idOrName string) {
	data := readGoals()
	idx := findRoadmap(data, idOrName)
	if idx == -1 {
		die("Error: Roadmap %q not found.", idOrName)
	}
	target := data.Roadmaps[idx]

	// Drop everything belonging to this list. After migration, every item
	// has a non-empty RoadmapID, so equality is sufficient.
	keepMains := data.MainGoals[:0]
	for _, mg := range data.MainGoals {
		if mg.RoadmapID != target.ID {
			keepMains = append(keepMains, mg)
		}
	}
	data.MainGoals = append([]MainGoal{}, keepMains...)

	keepSubs := data.SubGoals[:0]
	for _, sg := range data.SubGoals {
		if sg.RoadmapID != target.ID {
			keepSubs = append(keepSubs, sg)
		}
	}
	data.SubGoals = append([]SubGoal{}, keepSubs...)

	keepTasks := data.Tasks[:0]
	for _, t := range data.Tasks {
		if t.RoadmapID != target.ID {
			keepTasks = append(keepTasks, t)
		}
	}
	data.Tasks = append([]Task{}, keepTasks...)

	data.Roadmaps = append(data.Roadmaps[:idx], data.Roadmaps[idx+1:]...)

	if data.ActiveRoadmapID == target.ID {
		if len(data.Roadmaps) > 0 {
			data.ActiveRoadmapID = data.Roadmaps[0].ID
		} else {
			data.ActiveRoadmapID = ""
		}
	}

	writeGoals(data)
	fmt.Printf("\n🗑️  Deleted roadmap: %q (and its goals/tasks)\n", target.Name)
	if len(data.Roadmaps) == 0 {
		fmt.Printf("   No roadmaps remaining. Create one with:  og list-create <name>\n\n")
	} else {
		fmt.Printf("   Active roadmap is now: %s\n\n", activeRoadmapName(data))
	}
}

// listDeleteBulk deletes multiple roadmaps by ID or name in one pass.
// Confirms by default; pass yes=true to skip the prompt.
func listDeleteBulk(idsOrNames []string, yes bool) {
	if len(idsOrNames) == 0 {
		die("Error: at least one roadmap id or name is required.")
	}
	if len(idsOrNames) == 1 {
		// Preserve the single-item behavior (no extra confirmation prompt).
		listDelete(idsOrNames[0])
		return
	}

	data := readGoals()
	type pending struct {
		idx int
		ref Roadmap
	}
	seen := map[string]bool{} // dedupe by roadmap ID
	var found []pending
	var missing []string

	for _, q := range idsOrNames {
		idx := findRoadmap(data, q)
		if idx == -1 {
			missing = append(missing, q)
			continue
		}
		rm := data.Roadmaps[idx]
		if seen[rm.ID] {
			continue
		}
		seen[rm.ID] = true
		found = append(found, pending{idx: idx, ref: rm})
	}

	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  %d not found: %s\n", len(missing), strings.Join(missing, ", "))
	}
	if len(found) == 0 {
		fmt.Fprintln(os.Stderr, "ℹ️  No roadmaps matched.")
		os.Exit(1)
	}

	// Build summary including counts of nested goals/tasks.
	titles := make([]string, len(found))
	for i, p := range found {
		mains, subs, tasks := 0, 0, 0
		for _, mg := range data.MainGoals {
			if mg.RoadmapID == p.ref.ID {
				mains++
			}
		}
		for _, sg := range data.SubGoals {
			if sg.RoadmapID == p.ref.ID {
				subs++
			}
		}
		for _, t := range data.Tasks {
			if t.RoadmapID == p.ref.ID {
				tasks++
			}
		}
		titles[i] = fmt.Sprintf("%s  (%d main, %d sub, %d tasks)  id=%s",
			p.ref.Name, mains, subs, tasks, p.ref.ID)
	}

	fmt.Printf("\nAbout to delete %d roadmap(s) and ALL their contents:\n", len(found))
	printSummaryList(titles, 12)

	if !confirmPrompt(fmt.Sprintf("\nDelete %d roadmap(s)?", len(found)), yes) {
		fmt.Fprintln(os.Stderr, "Aborted.")
		return
	}

	// Apply each via the existing single-delete (which handles the active-id
	// fallback after each removal correctly).
	for _, p := range found {
		listDelete(p.ref.ID)
	}
}

func listShow(idOrName string) {
	data := readGoals()
	idx := findRoadmap(data, idOrName)
	if idx == -1 {
		die("Error: Roadmap %q not found.", idOrName)
	}
	target := data.Roadmaps[idx]

	fmt.Printf("\n📍 Roadmap: %s", target.Name)
	if target.ID == data.ActiveRoadmapID {
		fmt.Print("  (active)")
	}
	fmt.Println()
	printSeparator()

	mains := goalsForRoadmap(data, target.ID)
	subs := subGoalsForRoadmap(data, target.ID)
	tasks := tasksForRoadmap(data, target.ID)

	fmt.Println("\n🎯 Goals:")
	if len(mains) == 0 {
		fmt.Println("  (none)")
	}
	for _, mg := range mains {
		statusIcon := "⏸️"
		switch mg.Status {
		case StatusCompleted:
			statusIcon = "✅"
		case StatusInProgress:
			statusIcon = "🔄"
		}
		fmt.Printf("  %s %s [%d%%]\n", statusIcon, mg.Title, calculateProgress(mg.ID, data))
		fmt.Printf("     ID: %s\n", mg.ID)
		for _, sg := range subs {
			if sg.ParentID == mg.ID {
				icon := "○"
				if sg.Status == StatusCompleted {
					icon = "✓"
				}
				fmt.Printf("       %s %s\n", icon, sg.Title)
			}
		}
	}

	fmt.Println("\n📝 Tasks:")
	if len(tasks) == 0 {
		fmt.Println("  (none)")
	}
	for _, t := range tasks {
		icon := "○"
		if t.Completed {
			icon = "✓"
		}
		priority := ""
		if t.Priority != "" {
			priority = fmt.Sprintf(" [%s]", t.Priority)
		}
		fmt.Printf("  %s %s%s\n", icon, t.Title, priority)
		fmt.Printf("     ID: %s\n", t.ID)
	}

	fmt.Println()
	printSeparator()
	fmt.Println()
}
