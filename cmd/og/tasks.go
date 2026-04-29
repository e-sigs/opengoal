package main

// Task operations: list/add/complete/delete/clear, plus the bulk-delete
// flow and the priority sorter used by display code.

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// tasksForRoadmap returns tasks belonging to the given roadmap.
func tasksForRoadmap(data GoalsData, listID string) []Task {
	return filterByRoadmap(data.Tasks, func(t Task) string { return t.RoadmapID }, listID)
}

func listTasks() {
	data := readGoals()
	if len(data.Roadmaps) == 0 {
		fmt.Println("\n📝 Task Roadmap")
		printSeparator()
		fmt.Printf("\n  No roadmaps yet. Create one with:  og list-create <name>\n\n")
		return
	}
	listID := data.ActiveRoadmapID

	var pending, completed []Task
	for _, t := range tasksForRoadmap(data, listID) {
		if t.Completed {
			completed = append(completed, t)
		} else {
			pending = append(pending, t)
		}
	}

	fmt.Printf("\n📝 Task Roadmap  [roadmap: %s]\n", activeRoadmapName(data))
	printSeparator()

	if len(pending) == 0 && len(completed) == 0 {
		fmt.Printf("\nNo tasks yet! Use /task-add to add your first task.\n\n")
		return
	}

	if len(pending) > 0 {
		ttl := claimTTL()
		fmt.Printf("\n⏳ Pending (%d):\n", len(pending))
		for i, task := range pending {
			priority := ""
			if task.Priority != "" {
				priority = fmt.Sprintf(" [%s]", task.Priority)
			}

			// Status marker: blocked > claimed > stale > ready.
			marker := "○"
			suffix := ""
			if blocked := blockedDeps(task, data.Tasks); len(blocked) > 0 {
				marker = "⏸"
				suffix = fmt.Sprintf(" — blocked by %d dep%s", len(blocked), plural(len(blocked)))
			} else if claimActive(task, ttl) {
				marker = "🔒"
				suffix = fmt.Sprintf(" — %s (%s ago)", task.Assignee, formatAge(time.Since(*task.ClaimedAt)))
			} else if task.Assignee != "" && task.ClaimedAt != nil {
				marker = "⚠"
				suffix = fmt.Sprintf(" — stale claim by %s", task.Assignee)
			}

			fmt.Printf("  %d. %s %s%s%s\n", i+1, marker, task.Title, priority, suffix)
			fmt.Printf("     ID: %s | Created: %s\n", task.ID, task.Created.Format(dateFmtDisplay))
		}
	}

	if len(completed) > 0 {
		fmt.Printf("\n✅ Completed (%d):\n", len(completed))
		limit := completedShowMax
		if len(completed) < limit {
			limit = len(completed)
		}
		for i := 0; i < limit; i++ {
			task := completed[i]
			fmt.Printf("  ✓ %s\n", task.Title)
			if task.CompletedAt != nil {
				fmt.Printf("     Completed: %s\n", task.CompletedAt.Format(dateFmtDisplay))
			}
		}
		if len(completed) > limit {
			fmt.Printf("  ... and %d more\n", len(completed)-limit)
		}
	}

	fmt.Println()
	printSeparator()
	fmt.Println()
}

func addTask(title, priority string, dependsOn []string) {
	withLock(func() {
		data := readGoals()
		requireActiveRoadmap(data)

		// Validate deps exist and aren't self-references.
		for _, depID := range dependsOn {
			if findTaskByID(data.Tasks, depID) == nil {
				die("Error: dependency %q not found.", depID)
			}
		}

		newTask := Task{
			ID:        generateID("task"),
			RoadmapID: data.ActiveRoadmapID,
			Title:     title,
			Created:   time.Now(),
			Priority:  priority,
			DependsOn: dependsOn,
		}

		data.Tasks = append(data.Tasks, newTask)
		writeGoals(data)
		appendEvent(Event{
			Event:  EvTaskAdded,
			TaskID: newTask.ID, RoadmapID: newTask.RoadmapID, Title: newTask.Title,
			Data: map[string]any{
				"priority":   priority,
				"depends_on": dependsOn,
			},
		})

		fmt.Printf("\n✅ Added task: %q\n", title)
		if priority != "" {
			fmt.Printf("   Priority: %s\n", priority)
		}
		if len(dependsOn) > 0 {
			fmt.Printf("   Depends on: %s\n", strings.Join(dependsOn, ", "))
		}
		fmt.Printf("   ID: %s\n\n", newTask.ID)
	})
}

func markTaskDone(taskID string) {
	withLock(func() {
		data := readGoals()
		now := time.Now()

		for i := range data.Tasks {
			if data.Tasks[i].ID == taskID {
				data.Tasks[i].Completed = true
				data.Tasks[i].CompletedAt = &now
				// Completion releases any claim — the work is done.
				prevAssignee := data.Tasks[i].Assignee
				data.Tasks[i].Assignee = ""
				data.Tasks[i].ClaimedAt = nil
				completedTitle := data.Tasks[i].Title
				completedListID := data.Tasks[i].RoadmapID
				writeGoals(data)
				appendEvent(Event{
					Event:  EvTaskCompleted,
					TaskID: taskID, RoadmapID: completedListID,
					Title: completedTitle,
					Data:  map[string]any{"prev_assignee": prevAssignee},
				})
				// Detect newly-unblocked tasks: any pending task that
				// depended on this one and now has all deps satisfied.
				for _, t := range data.Tasks {
					if t.Completed || t.ID == taskID {
						continue
					}
					depended := false
					for _, d := range t.DependsOn {
						if d == taskID {
							depended = true
							break
						}
					}
					if !depended {
						continue
					}
					if len(blockedDeps(t, data.Tasks)) == 0 {
						appendEvent(Event{
							Event:  EvTaskUnblocked,
							TaskID: t.ID, RoadmapID: t.RoadmapID, Title: t.Title,
							Data: map[string]any{"unblocked_by": taskID},
						})
					}
				}
				fmt.Printf("\n✅ Marked task as completed: %q\n\n", completedTitle)
				return
			}
		}
		die("Error: Task with ID %q not found.", taskID)
	})
}

func deleteTask(taskID string) {
	withLock(func() {
		data := readGoals()
		for i := range data.Tasks {
			if data.Tasks[i].ID == taskID {
				title := data.Tasks[i].Title
				listID := data.Tasks[i].RoadmapID
				data.Tasks = append(data.Tasks[:i], data.Tasks[i+1:]...)
				writeGoals(data)
				appendEvent(Event{
					Event:  EvTaskDeleted,
					TaskID: taskID, RoadmapID: listID, Title: title,
				})
				fmt.Printf("\n🗑️  Deleted task: %q\n\n", title)
				return
			}
		}
		die("Error: Task with ID %q not found.", taskID)
	})
}

// deleteTasksBulk deletes multiple tasks identified by IDs and/or filter
// criteria. Confirms by default; pass yes=true to skip the prompt.
//
// Selection rules:
//   - If b.all is set, every task in the active roadmap is targeted.
//   - If b.priority is set, every task with that priority is targeted.
//   - If b.filter == "completed", every completed task is targeted.
//   - Any explicit IDs in b.ids are added to the target set.
//   - Selections are unioned. Duplicates are deduplicated.
//
// Tasks not found are reported but do not abort the operation.
func deleteTasksBulk(b bulkArgs) {
	data := readGoals()
	activeID := data.ActiveRoadmapID

	targets := map[string]Task{} // id → task
	missing := []string{}

	// Explicit IDs first (so we can report missing ones cleanly).
	for _, id := range b.ids {
		found := false
		for _, t := range data.Tasks {
			if t.ID == id {
				targets[t.ID] = t
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, id)
		}
	}

	// Filters operate on tasks in the active roadmap only.
	if b.all || b.priority != "" || b.filter != "" {
		for _, t := range data.Tasks {
			if t.RoadmapID != activeID {
				continue
			}
			switch {
			case b.all:
				targets[t.ID] = t
			case b.priority != "" && t.Priority == b.priority:
				targets[t.ID] = t
			case b.filter == "completed" && t.Completed:
				targets[t.ID] = t
			}
		}
	}

	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  %d ID(s) not found: %s\n", len(missing), strings.Join(missing, ", "))
	}

	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "ℹ️  No tasks matched the selection.")
		if len(missing) > 0 {
			os.Exit(1)
		}
		return
	}

	// Build deterministic preview order: by priority, then created.
	preview := make([]Task, 0, len(targets))
	for _, t := range targets {
		preview = append(preview, t)
	}
	sortTasksForDisplay(preview)

	titles := make([]string, len(preview))
	for i, t := range preview {
		marker := "○"
		if t.Completed {
			marker = "✓"
		}
		titles[i] = fmt.Sprintf("%s [%s] %s  (id: %s)", marker, t.Priority, t.Title, t.ID)
	}

	fmt.Printf("\nAbout to delete %d task(s):\n", len(preview))
	printSummaryList(titles, 12)

	if !confirmPrompt(fmt.Sprintf("\nDelete %d task(s)?", len(preview)), b.yes) {
		fmt.Fprintln(os.Stderr, "Aborted.")
		return
	}

	// Apply.
	withLock(func() {
		fresh := readGoals()
		removed := 0
		kept := fresh.Tasks[:0]
		for _, t := range fresh.Tasks {
			if _, hit := targets[t.ID]; hit {
				removed++
				appendEvent(Event{
					Event:  EvTaskDeleted,
					TaskID: t.ID, RoadmapID: t.RoadmapID, Title: t.Title,
				})
				continue
			}
			kept = append(kept, t)
		}
		fresh.Tasks = append([]Task{}, kept...)
		writeGoals(fresh)
		fmt.Printf("\n🗑️  Deleted %d task(s).\n\n", removed)
	})
}

// sortTasksForDisplay sorts in place: high → medium → low → unset, then by Created ascending.
func sortTasksForDisplay(ts []Task) {
	prio := func(p string) int {
		switch p {
		case PriorityHigh:
			return 0
		case PriorityMedium:
			return 1
		case PriorityLow:
			return 2
		}
		return 3
	}
	for i := 1; i < len(ts); i++ {
		for j := i; j > 0; j-- {
			a, b := ts[j-1], ts[j]
			if prio(a.Priority) > prio(b.Priority) ||
				(prio(a.Priority) == prio(b.Priority) && a.Created.After(b.Created)) {
				ts[j-1], ts[j] = b, a
			} else {
				break
			}
		}
	}
}

func clearCompletedTasks() {
	withLock(func() {
		data := readGoals()
		completedCount := 0
		kept := data.Tasks[:0]
		for _, t := range data.Tasks {
			if t.Completed {
				completedCount++
				continue
			}
			kept = append(kept, t)
		}
		// Make a copy so the underlying array is freed for GC.
		data.Tasks = append([]Task{}, kept...)
		writeGoals(data)
		appendEvent(Event{
			Event: EvTaskCleared,
			Data:  map[string]any{"count": completedCount},
		})
		fmt.Printf("\n🗑️  Cleared %d completed task(s)\n\n", completedCount)
	})
}
