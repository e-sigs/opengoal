package main

// The "today" dashboard — combined view of goals, tasks, and focus
// suggestions for the active roadmap.

import (
	"fmt"
	"strings"
	"time"
)

func showToday() {
	data := readGoals()

	if len(data.Roadmaps) == 0 {
		fmt.Println("\n╔════════════════════════════════════════════════╗")
		fmt.Printf("║  %s\n", cTitle(fmt.Sprintf("📅 TODAY - %s", time.Now().Format(dateFmtHeader))))
		fmt.Printf("║  %s\n", cCaption("📍 No roadmaps yet"))
		fmt.Println("╚════════════════════════════════════════════════╝")
		fmt.Println("\nYou have no roadmaps yet.")
		fmt.Println("Create your first one with:")
		fmt.Println(cComment("  og list-create <name>"))
		fmt.Printf("\n%s\n\n", cComment("Or use /og to manage roadmaps interactively."))
		return
	}

	listID := data.ActiveRoadmapID
	todayStr := today()

	roadmapMains := goalsForRoadmap(data, listID)
	listSubs := subGoalsForRoadmap(data, listID)
	listTasksAll := tasksForRoadmap(data, listID)

	fmt.Println("\n╔════════════════════════════════════════════════╗")
	fmt.Printf("║  %s\n", cTitle(fmt.Sprintf("📅 TODAY - %s", time.Now().Format(dateFmtHeader))))
	fmt.Printf("║  %s\n", cCaption(fmt.Sprintf("📍 Roadmap: %s", activeRoadmapName(data))))
	fmt.Println("╚════════════════════════════════════════════════╝")

	// Active Goals
	var inProgress []MainGoal
	for _, mg := range roadmapMains {
		if mg.Status == StatusInProgress {
			inProgress = append(inProgress, mg)
		}
	}

	fmt.Printf("\n%s\n", cHeading("🎯 ACTIVE GOALS"))
	printSeparator()

	if len(inProgress) > 0 {
		for _, mg := range inProgress {
			progress := calculateProgress(mg.ID, data)
			var nextSubGoal *SubGoal
			for i := range listSubs {
				if listSubs[i].ParentID == mg.ID && listSubs[i].Status == StatusPending {
					nextSubGoal = &listSubs[i]
					break
				}
			}
			fmt.Printf("\n  • %s %s\n", cBold(mg.Title), cCaption(fmt.Sprintf("[%d%%]", progress)))
			if nextSubGoal != nil {
				fmt.Printf("      %s %s\n", cCaption("→ Next:"), cSubtitle(nextSubGoal.Title))
			}
		}
	} else {
		fmt.Printf("\n  %s\n", cComment("No active goals. Use /og-main to add one!"))
	}

	// Pending Tasks (bucketed by priority)
	var pending, highPriority, mediumPriority, otherTasks []Task
	for _, t := range listTasksAll {
		if t.Completed {
			continue
		}
		pending = append(pending, t)
		switch t.Priority {
		case PriorityHigh:
			highPriority = append(highPriority, t)
		case PriorityMedium:
			mediumPriority = append(mediumPriority, t)
		default:
			otherTasks = append(otherTasks, t)
		}
	}

	fmt.Printf("\n\n%s\n", cHeading("📝 TASKS"))
	printSeparator()

	if len(pending) > 0 {
		printPriorityBucket("🔴 HIGH PRIORITY", PriorityHigh, highPriority)
		printPriorityBucket("🟡 MEDIUM PRIORITY", PriorityMedium, mediumPriority)
		printPriorityBucket("⚪ OTHER", "", otherTasks)
	} else {
		fmt.Printf("\n  %s\n", cComment("No pending tasks. Use /task-add to add one!"))
	}

	// Completed Today
	var completedToday []SubGoal
	for _, sg := range listSubs {
		if sg.CompletedAt != nil && sg.CompletedAt.Format(dateFmtISO) == todayStr {
			completedToday = append(completedToday, sg)
		}
	}
	var tasksCompletedToday []Task
	for _, t := range listTasksAll {
		if t.CompletedAt != nil && t.CompletedAt.Format(dateFmtISO) == todayStr {
			tasksCompletedToday = append(tasksCompletedToday, t)
		}
	}

	fmt.Printf("\n\n%s\n", cHeading("✅ COMPLETED TODAY"))
	printSeparator()

	if len(completedToday) > 0 || len(tasksCompletedToday) > 0 {
		if len(completedToday) > 0 {
			fmt.Printf("\n%s\n", cBold("Goals:"))
			for _, sg := range completedToday {
				fmt.Printf("  %s %s\n", cSuccess("✓"), cDim(sg.Title))
			}
		}
		if len(tasksCompletedToday) > 0 {
			fmt.Printf("\n%s\n", cBold("Tasks:"))
			for _, t := range tasksCompletedToday {
				fmt.Printf("  %s %s\n", cSuccess("✓"), cDim(t.Title))
			}
		}
	} else {
		fmt.Printf("\n  %s\n", cComment("Nothing completed yet today. Let's get started!"))
	}

	// Focus
	fmt.Printf("\n\n%s\n", cHeading("🔥 FOCUS NOW"))
	printSeparator()

	focusCount := 0
	if len(highPriority) > 0 {
		fmt.Printf("\n  • %s %s\n", cDanger(highPriority[0].Title), cCaption("(high priority task)"))
		focusCount++
	}
	if len(inProgress) > 0 {
		topGoal := inProgress[0]
		for _, sg := range listSubs {
			if sg.ParentID == topGoal.ID && sg.Status == StatusPending {
				fmt.Printf("  • %s %s\n", cSubtitle(sg.Title), cCaption(fmt.Sprintf("(%s)", topGoal.Title)))
				focusCount++
				break
			}
		}
	}
	if len(highPriority) > 1 && focusCount < focusListLimit {
		fmt.Printf("  • %s %s\n", cDanger(highPriority[1].Title), cCaption("(high priority task)"))
		focusCount++
	}
	if len(mediumPriority) > 0 && focusCount < focusListLimit {
		fmt.Printf("  • %s %s\n", cWarn(mediumPriority[0].Title), cCaption("(medium priority task)"))
		focusCount++
	}
	if focusCount == 0 {
		fmt.Printf("\n  %s\n", cComment("Add some goals or tasks to get started!"))
		fmt.Printf("  %s\n", cComment("• /og-main <title> - Add a main goal"))
		fmt.Printf("  %s\n", cComment("• /task-add <title> - Add a quick task"))
	}

	// Stats
	fmt.Printf("\n\n%s\n", cHeading("📊 STATS"))
	printSeparator()
	fmt.Printf("  %s %s\n", cCaption("Active Goals:"), cBold(fmt.Sprintf("%d", len(inProgress))))
	fmt.Printf("  %s %s\n", cCaption("Pending Tasks:"), cBold(fmt.Sprintf("%d", len(pending))))
	fmt.Printf("  %s %s\n\n", cCaption("Completed Today:"), cBold(fmt.Sprintf("%d", len(completedToday)+len(tasksCompletedToday))))

	fmt.Printf("%s\n\n", strings.Repeat("═", separatorWidth))
}

func printPriorityBucket(label, priority string, tasks []Task) {
	if len(tasks) == 0 {
		return
	}
	fmt.Printf("\n%s:\n", cBold(label))
	for _, t := range tasks {
		fmt.Printf("  • %s\n", cPriority(priority, t.Title))
	}
}
