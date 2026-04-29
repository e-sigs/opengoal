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
		fmt.Printf("║  📅 TODAY - %s\n", time.Now().Format(dateFmtHeader))
		fmt.Println("║  📍 No roadmaps yet")
		fmt.Println("╚════════════════════════════════════════════════╝")
		fmt.Println("\nYou have no roadmaps yet.")
		fmt.Println("Create your first one with:")
		fmt.Println("  og list-create <name>")
		fmt.Printf("\nOr use /og to manage roadmaps interactively.\n\n")
		return
	}

	listID := data.ActiveRoadmapID
	todayStr := today()

	roadmapMains := goalsForRoadmap(data, listID)
	listSubs := subGoalsForRoadmap(data, listID)
	listTasksAll := tasksForRoadmap(data, listID)

	fmt.Println("\n╔════════════════════════════════════════════════╗")
	fmt.Printf("║  📅 TODAY - %s\n", time.Now().Format(dateFmtHeader))
	fmt.Printf("║  📍 Roadmap: %s\n", activeRoadmapName(data))
	fmt.Println("╚════════════════════════════════════════════════╝")

	// Active Goals
	var inProgress []MainGoal
	for _, mg := range roadmapMains {
		if mg.Status == StatusInProgress {
			inProgress = append(inProgress, mg)
		}
	}

	fmt.Println("\n🎯 ACTIVE GOALS")
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
			fmt.Printf("\n%s [%d%%]\n", mg.Title, progress)
			if nextSubGoal != nil {
				fmt.Printf("  → Next: %s\n", nextSubGoal.Title)
			}
		}
	} else {
		fmt.Println("\n  No active goals. Use /og-main to add one!")
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

	fmt.Println("\n\n📝 TASKS")
	printSeparator()

	if len(pending) > 0 {
		printPriorityBucket("🔴 HIGH PRIORITY", highPriority)
		printPriorityBucket("🟡 MEDIUM PRIORITY", mediumPriority)
		printPriorityBucket("⚪ OTHER", otherTasks)
	} else {
		fmt.Println("\n  No pending tasks. Use /task-add to add one!")
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

	fmt.Println("\n\n✅ COMPLETED TODAY")
	printSeparator()

	if len(completedToday) > 0 || len(tasksCompletedToday) > 0 {
		if len(completedToday) > 0 {
			fmt.Println("\nGoals:")
			for _, sg := range completedToday {
				fmt.Printf("  ✓ %s\n", sg.Title)
			}
		}
		if len(tasksCompletedToday) > 0 {
			fmt.Println("\nTasks:")
			for _, t := range tasksCompletedToday {
				fmt.Printf("  ✓ %s\n", t.Title)
			}
		}
	} else {
		fmt.Println("\n  Nothing completed yet today. Let's get started!")
	}

	// Focus
	fmt.Println("\n\n🔥 FOCUS NOW")
	printSeparator()

	focusCount := 0
	if len(highPriority) > 0 {
		fmt.Printf("\n  1. %s (high priority task)\n", highPriority[0].Title)
		focusCount++
	}
	if len(inProgress) > 0 {
		topGoal := inProgress[0]
		for _, sg := range listSubs {
			if sg.ParentID == topGoal.ID && sg.Status == StatusPending {
				fmt.Printf("\n  %d. %s (%s)\n", focusCount+1, sg.Title, topGoal.Title)
				focusCount++
				break
			}
		}
	}
	if len(highPriority) > 1 && focusCount < focusListLimit {
		fmt.Printf("\n  %d. %s (high priority task)\n", focusCount+1, highPriority[1].Title)
		focusCount++
	}
	if len(mediumPriority) > 0 && focusCount < focusListLimit {
		fmt.Printf("\n  %d. %s (medium priority task)\n", focusCount+1, mediumPriority[0].Title)
		focusCount++
	}
	if focusCount == 0 {
		fmt.Println("\n  Add some goals or tasks to get started!")
		fmt.Println("  • /og-main <title> - Add a main goal")
		fmt.Println("  • /task-add <title> - Add a quick task")
	}

	// Stats
	fmt.Println("\n\n📊 STATS")
	printSeparator()
	fmt.Printf("  Active Goals: %d\n", len(inProgress))
	fmt.Printf("  Pending Tasks: %d\n", len(pending))
	fmt.Printf("  Completed Today: %d\n\n", len(completedToday)+len(tasksCompletedToday))

	fmt.Printf("%s\n\n", strings.Repeat("═", separatorWidth))
}

func printPriorityBucket(label string, tasks []Task) {
	if len(tasks) == 0 {
		return
	}
	fmt.Printf("\n%s:\n", label)
	for i, t := range tasks {
		fmt.Printf("  %d. %s\n", i+1, t.Title)
	}
}
