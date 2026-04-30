package main

// The "today" dashboard — combined view of goals, tasks, and focus
// suggestions for the active roadmap, rendered as bordered cards.

import (
	"fmt"
	"time"
)

func showToday() {
	data := readGoals()

	if len(data.Roadmaps) == 0 {
		fmt.Println()
		fmt.Println(boxTop(cTitle("📅 " + time.Now().Format(dateFmtHeader))))
		fmt.Println(boxLine(cCaption("No roadmaps yet."), 0))
		fmt.Println(boxLine(cComment("Create one: og list-create <name>"), 0))
		fmt.Println(boxBottom())
		fmt.Println()
		return
	}

	listID := data.ActiveRoadmapID
	todayStr := today()

	roadmapMains := goalsForRoadmap(data, listID)
	listSubs := subGoalsForRoadmap(data, listID)
	listTasksAll := tasksForRoadmap(data, listID)

	// Header card.
	fmt.Println()
	header := cTitle("📅 "+time.Now().Format(dateFmtHeader)) +
		"  " + cCaption("· "+activeRoadmapName(data))
	fmt.Println(boxTop(header))
	fmt.Println(boxBottom())

	// Goals card.
	var inProgress []MainGoal
	for _, mg := range roadmapMains {
		if mg.Status == StatusInProgress {
			inProgress = append(inProgress, mg)
		}
	}

	fmt.Println()
	fmt.Println(boxTop(cHeading("🎯 Goals")))
	if len(inProgress) > 0 {
		for i, mg := range inProgress {
			progress := calculateProgress(mg.ID, data)
			var nextSubGoal *SubGoal
			for j := range listSubs {
				if listSubs[j].ParentID == mg.ID && listSubs[j].Status == StatusPending {
					nextSubGoal = &listSubs[j]
					break
				}
			}
			// Title line: "  0%  Title"
			title := fmt.Sprintf("%s  %s",
				cCaption(fmt.Sprintf("%3d%%", progress)),
				cBold(mg.Title))
			fmt.Println(boxLine(title, 0))
			// Next line, indented under the title.
			if nextSubGoal != nil {
				next := "         " + cCaption("→ ") + cSubtitle(nextSubGoal.Title)
				fmt.Println(boxLine(next, 0))
			} else {
				fmt.Println(boxLine("         "+cComment("(no pending sub-goals)"), 0))
			}
			if i < len(inProgress)-1 {
				fmt.Println(boxBlank())
			}
		}
	} else {
		fmt.Println(boxLine(cComment("none — /og-main <title>"), 0))
	}
	fmt.Println(boxBottom())

	// Tasks card.
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

	fmt.Println()
	fmt.Println(boxTop(cHeading("📝 Tasks")))
	if len(pending) > 0 {
		printTaskBucket("🔴 high", PriorityHigh, highPriority)
		if len(highPriority) > 0 && (len(mediumPriority) > 0 || len(otherTasks) > 0) {
			fmt.Println(boxBlank())
		}
		printTaskBucket("🟡 medium", PriorityMedium, mediumPriority)
		if len(mediumPriority) > 0 && len(otherTasks) > 0 {
			fmt.Println(boxBlank())
		}
		printTaskBucket("⚪ other", "", otherTasks)
	} else {
		fmt.Println(boxLine(cComment("none — /task-add <title>"), 0))
	}
	fmt.Println(boxBottom())

	// Done-today card (only if any).
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
	if len(completedToday)+len(tasksCompletedToday) > 0 {
		fmt.Println()
		fmt.Println(boxTop(cHeading("✅ Done today")))
		for _, sg := range completedToday {
			fmt.Println(boxLine("  "+cSuccess("✓")+" "+cDim(sg.Title), 0))
		}
		for _, t := range tasksCompletedToday {
			fmt.Println(boxLine("  "+cSuccess("✓")+" "+cDim(t.Title), 0))
		}
		fmt.Println(boxBottom())
	}

	// Focus card.
	type focusItem struct {
		marker string
		title  string
		hint   string
	}
	var focus []focusItem
	if len(highPriority) > 0 {
		focus = append(focus, focusItem{cDanger("●"), cDanger(highPriority[0].Title), "high priority task"})
	}
	if len(inProgress) > 0 {
		topGoal := inProgress[0]
		for _, sg := range listSubs {
			if sg.ParentID == topGoal.ID && sg.Status == StatusPending {
				focus = append(focus, focusItem{cInfo("●"), cSubtitle(sg.Title), topGoal.Title})
				break
			}
		}
	}
	if len(highPriority) > 1 && len(focus) < focusListLimit {
		focus = append(focus, focusItem{cDanger("●"), cDanger(highPriority[1].Title), "high priority task"})
	}
	if len(mediumPriority) > 0 && len(focus) < focusListLimit {
		focus = append(focus, focusItem{cWarn("●"), cWarn(mediumPriority[0].Title), "medium priority task"})
	}

	fmt.Println()
	fmt.Println(boxTop(cHeading("🔥 Focus")))
	if len(focus) > 0 {
		for i, f := range focus {
			fmt.Println(boxLine("  "+f.marker+" "+f.title, 0))
			fmt.Println(boxLine("    "+cCaption("· "+f.hint), 0))
			if i < len(focus)-1 {
				fmt.Println(boxBlank())
			}
		}
	} else {
		fmt.Println(boxLine(cComment("add a goal or task to get started"), 0))
	}
	fmt.Println(boxBottom())

	// Stats footer.
	stats := fmt.Sprintf("%s active goals  ·  %s pending tasks  ·  %s done today",
		cBold(fmt.Sprintf("%d", len(inProgress))),
		cBold(fmt.Sprintf("%d", len(pending))),
		cBold(fmt.Sprintf("%d", len(completedToday)+len(tasksCompletedToday))))
	fmt.Println()
	fmt.Println("  " + stats)
	fmt.Println()
}

// printTaskBucket prints each task on its own bordered line. Empty buckets
// produce no output.
func printTaskBucket(label, priority string, tasks []Task) {
	if len(tasks) == 0 {
		return
	}
	fmt.Println(boxLine(cBold(label), 0))
	for _, t := range tasks {
		fmt.Println(boxLine("  • "+cPriority(priority, t.Title), 0))
	}
}
