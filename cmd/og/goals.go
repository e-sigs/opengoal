package main

// Goal operations: main goals, sub-goals, summary, reminder, and the
// progress calculation that ties them together. Pure UI/business logic;
// all persistence goes through readGoals/writeGoals in main.go.

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ──────────────────────────────────────────────────────────────────
// Per-roadmap filters (goals/sub-goals)
// ──────────────────────────────────────────────────────────────────

func goalsForRoadmap(data GoalsData, listID string) []MainGoal {
	return filterByRoadmap(data.MainGoals, func(m MainGoal) string { return m.RoadmapID }, listID)
}

func subGoalsForRoadmap(data GoalsData, listID string) []SubGoal {
	return filterByRoadmap(data.SubGoals, func(s SubGoal) string { return s.RoadmapID }, listID)
}

// calculateProgress returns the integer percentage of completed sub-goals
// under the given main goal. Returns 0 if the main goal has no sub-goals.
func calculateProgress(mainGoalID string, data GoalsData) int {
	var total, completed int
	for _, sg := range data.SubGoals {
		if sg.ParentID == mainGoalID {
			total++
			if sg.Status == StatusCompleted {
				completed++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return (completed * 100) / total
}

// ──────────────────────────────────────────────────────────────────
// Goal operations
// ──────────────────────────────────────────────────────────────────

func listGoals() {
	data := readGoals()
	if len(data.Roadmaps) == 0 {
		fmt.Println("\n📋 Current Goals")
		printSeparator()
		fmt.Printf("\n  No roadmaps yet. Create one with:  og list-create <name>\n\n")
		return
	}
	listID := data.ActiveRoadmapID
	mainGoals := goalsForRoadmap(data, listID)

	fmt.Printf("\n📋 Current Goals  [roadmap: %s]\n", activeRoadmapName(data))
	printSeparator()

	if len(mainGoals) == 0 {
		fmt.Printf("\nNo goals yet! Use /og-main to add your first goal.\n\n")
		return
	}

	for _, mg := range mainGoals {
		progress := calculateProgress(mg.ID, data)

		statusIcon := "⏸️"
		switch mg.Status {
		case StatusCompleted:
			statusIcon = "✅"
		case StatusInProgress:
			statusIcon = "🔄"
		}

		fmt.Printf("\n%s %s\n", statusIcon, mg.Title)
		fmt.Printf("   ID: %s | Progress: %d%% | Status: %s\n", mg.ID, progress, mg.Status)

		if len(mg.Context) > 0 {
			fmt.Printf("   Context: %s\n", strings.Join(mg.Context, ", "))
		}

		subs := subGoalsForRoadmap(data, listID)
		var children []SubGoal
		for _, sg := range subs {
			if sg.ParentID == mg.ID {
				children = append(children, sg)
			}
		}

		if len(children) > 0 {
			fmt.Println("   Sub-goals:")
			for _, sg := range children {
				icon := "○"
				if sg.Status == StatusCompleted {
					icon = "✓"
				}
				fmt.Printf("     %s %s (%s)\n", icon, sg.Title, sg.Status)
			}
		}
	}

	fmt.Println()
	printSeparator()
	fmt.Println()
}

func addMainGoal(title string, context []string) {
	data := readGoals()
	requireActiveRoadmap(data)

	newGoal := MainGoal{
		ID:        generateID("mg"),
		RoadmapID: data.ActiveRoadmapID,
		Title:     title,
		Created:   time.Now(),
		Status:    StatusInProgress,
		Progress:  0,
		SubGoals:  []string{},
		Context:   context,
	}

	data.MainGoals = append(data.MainGoals, newGoal)
	writeGoals(data)

	fmt.Printf("\n✅ Added main goal: %q  [roadmap: %s]\n", title, activeRoadmapName(data))
	fmt.Printf("   ID: %s\n\n", newGoal.ID)
}

func addSubGoal(title, parentID string) {
	data := readGoals()
	requireActiveRoadmap(data)

	parentIdx := -1
	for i, mg := range data.MainGoals {
		if mg.ID == parentID {
			parentIdx = i
			break
		}
	}
	if parentIdx == -1 {
		die("Error: Main goal with ID %q not found.", parentID)
	}

	newSubGoal := SubGoal{
		ID:        generateID("sg"),
		RoadmapID: data.MainGoals[parentIdx].RoadmapID,
		Title:     title,
		ParentID:  parentID,
		Created:   time.Now(),
		Status:    StatusPending,
	}

	data.SubGoals = append(data.SubGoals, newSubGoal)
	data.MainGoals[parentIdx].SubGoals = append(data.MainGoals[parentIdx].SubGoals, newSubGoal.ID)

	writeGoals(data)

	fmt.Printf("\n✅ Added sub-goal: %q\n", title)
	fmt.Printf("   Parent: %s\n", data.MainGoals[parentIdx].Title)
	fmt.Printf("   ID: %s\n\n", newSubGoal.ID)
}

func markDone(goalID string) {
	data := readGoals()
	now := time.Now()

	// Main goals
	for i := range data.MainGoals {
		if data.MainGoals[i].ID == goalID {
			data.MainGoals[i].Status = StatusCompleted
			data.MainGoals[i].CompletedAt = &now
			writeGoals(data)
			fmt.Printf("\n✅ Marked as completed: %q\n\n", data.MainGoals[i].Title)
			return
		}
	}

	// Sub-goals
	for i := range data.SubGoals {
		if data.SubGoals[i].ID != goalID {
			continue
		}
		data.SubGoals[i].Status = StatusCompleted
		data.SubGoals[i].CompletedAt = &now

		parentID := data.SubGoals[i].ParentID
		for j := range data.MainGoals {
			if data.MainGoals[j].ID != parentID {
				continue
			}
			data.MainGoals[j].Progress = calculateProgress(parentID, data)

			allComplete := true
			for _, sg := range data.SubGoals {
				if sg.ParentID == parentID && sg.Status != StatusCompleted {
					allComplete = false
					break
				}
			}
			if allComplete {
				data.MainGoals[j].Status = StatusCompleted
				data.MainGoals[j].CompletedAt = &now
			}
			break
		}

		writeGoals(data)
		fmt.Printf("\n✅ Marked as completed: %q\n\n", data.SubGoals[i].Title)
		return
	}

	die("Error: Goal with ID %q not found.", goalID)
}

// markDoneBulk marks multiple goals (main or sub) as completed in one pass.
// Reports each found goal; missing IDs are warned about but don't abort.
func markDoneBulk(ids []string, yes bool) {
	if len(ids) == 0 {
		die("Error: at least one goal ID is required.")
	}
	if len(ids) == 1 {
		// Preserve the single-ID behavior (no confirmation prompt for one).
		markDone(ids[0])
		return
	}

	data := readGoals()

	type pending struct {
		id, title, kind string // kind: "main" | "sub"
	}
	var found []pending
	var missing []string

	for _, id := range ids {
		matched := false
		for _, mg := range data.MainGoals {
			if mg.ID == id {
				found = append(found, pending{id: id, title: mg.Title, kind: "main"})
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		for _, sg := range data.SubGoals {
			if sg.ID == id {
				found = append(found, pending{id: id, title: sg.Title, kind: "sub"})
				matched = true
				break
			}
		}
		if !matched {
			missing = append(missing, id)
		}
	}

	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  %d ID(s) not found: %s\n", len(missing), strings.Join(missing, ", "))
	}

	if len(found) == 0 {
		fmt.Fprintln(os.Stderr, "ℹ️  No goals matched.")
		os.Exit(1)
	}

	titles := make([]string, len(found))
	for i, p := range found {
		titles[i] = fmt.Sprintf("%s: %s  (id: %s)", p.kind, p.title, p.id)
	}
	fmt.Printf("\nAbout to mark %d goal(s) as complete:\n", len(found))
	printSummaryList(titles, 12)

	if !confirmPrompt(fmt.Sprintf("\nMark %d goal(s) complete?", len(found)), yes) {
		fmt.Fprintln(os.Stderr, "Aborted.")
		return
	}

	// Apply each via the existing single-goal markDone, which handles
	// progress propagation and parent auto-completion.
	for _, p := range found {
		markDone(p.id)
	}
}

func generateSummary() {
	data := readGoals()
	if len(data.Roadmaps) == 0 {
		fmt.Println("\n📊 Daily Summary")
		printSeparator()
		fmt.Printf("\n  No roadmaps yet. Create one with:  og list-create <name>\n\n")
		return
	}
	listID := data.ActiveRoadmapID
	todayStr := today()

	roadmapMains := goalsForRoadmap(data, listID)
	listSubs := subGoalsForRoadmap(data, listID)

	var completedToday []SubGoal
	for _, sg := range listSubs {
		if sg.CompletedAt != nil && sg.CompletedAt.Format(dateFmtISO) == todayStr {
			completedToday = append(completedToday, sg)
		}
	}

	var addedToday []string
	for _, mg := range roadmapMains {
		if mg.Created.Format(dateFmtISO) == todayStr {
			addedToday = append(addedToday, mg.Title)
		}
	}
	for _, sg := range listSubs {
		if sg.Created.Format(dateFmtISO) == todayStr {
			addedToday = append(addedToday, sg.Title)
		}
	}

	var inProgress []MainGoal
	for _, mg := range roadmapMains {
		if mg.Status == StatusInProgress {
			inProgress = append(inProgress, mg)
		}
	}

	fmt.Printf("\n📊 Daily Summary - %s  [roadmap: %s]\n",
		time.Now().Format(dateFmtSummary), activeRoadmapName(data))
	printSeparator()

	fmt.Printf("\n✅ Completed (%d goals):\n", len(completedToday))
	if len(completedToday) > 0 {
		for _, sg := range completedToday {
			fmt.Printf("  - %s\n", sg.Title)
		}
	} else {
		fmt.Println("  (none)")
	}

	fmt.Printf("\n🔄 In Progress (%d goals):\n", len(inProgress))
	if len(inProgress) > 0 {
		for _, mg := range inProgress {
			fmt.Printf("  - %s (%d%%)\n", mg.Title, calculateProgress(mg.ID, data))
		}
	} else {
		fmt.Println("  (none)")
	}

	fmt.Printf("\n📝 Added Today (%d goals):\n", len(addedToday))
	if len(addedToday) > 0 {
		for _, title := range addedToday {
			fmt.Printf("  - %s\n", title)
		}
	} else {
		fmt.Println("  (none)")
	}

	fmt.Println("\n🎯 Next Focus:")
	if len(inProgress) > 0 {
		limit := focusListLimit
		if len(inProgress) < limit {
			limit = len(inProgress)
		}
		for i := 0; i < limit; i++ {
			mg := inProgress[i]
			var nextSubGoal *SubGoal
			for j := range listSubs {
				if listSubs[j].ParentID == mg.ID && listSubs[j].Status == StatusPending {
					nextSubGoal = &listSubs[j]
					break
				}
			}
			if nextSubGoal != nil {
				fmt.Printf("  %d. %s → %s\n", i+1, mg.Title, nextSubGoal.Title)
			} else {
				fmt.Printf("  %d. %s\n", i+1, mg.Title)
			}
		}
	} else {
		fmt.Println("  Add some goals to get started!")
	}

	fmt.Println()
	printSeparator()
	fmt.Println()
}

func remindMe() {
	data := readGoals()
	if len(data.Roadmaps) == 0 {
		fmt.Println("\n🎯 Goal Reminder")
		printSeparator()
		fmt.Printf("\n  No roadmaps yet. Create one with:  og list-create <name>\n\n")
		return
	}
	listID := data.ActiveRoadmapID

	var inProgress []MainGoal
	for _, mg := range goalsForRoadmap(data, listID) {
		if mg.Status == StatusInProgress {
			inProgress = append(inProgress, mg)
		}
	}

	var pendingSubGoals []SubGoal
	subs := subGoalsForRoadmap(data, listID)
	for _, sg := range subs {
		if sg.Status == StatusPending {
			pendingSubGoals = append(pendingSubGoals, sg)
		}
	}

	fmt.Printf("\n🎯 Goal Reminder  [roadmap: %s]\n", activeRoadmapName(data))
	printSeparator()

	if len(inProgress) == 0 {
		fmt.Printf("\n📭 No active goals. Use /og-main to add a goal!\n\n")
		return
	}

	fmt.Printf("\n📌 %d main goal(s) in progress\n", len(inProgress))
	fmt.Printf("📌 %d sub-goal(s) pending\n\n", len(pendingSubGoals))

	topGoal := inProgress[0]
	fmt.Printf("🔥 Focus Now: %s (%d%%)\n", topGoal.Title, calculateProgress(topGoal.ID, data))

	for _, sg := range subs {
		if sg.ParentID == topGoal.ID && sg.Status == StatusPending {
			fmt.Printf("   Next Step: %s\n", sg.Title)
			break
		}
	}

	fmt.Println()
	printSeparator()
	fmt.Println()
}
