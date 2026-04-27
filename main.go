package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Data structures
type MainGoal struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Created     time.Time `json:"created"`
	Status      string    `json:"status"`
	Progress    int       `json:"progress"`
	SubGoals    []string  `json:"sub_goals"`
	Context     []string  `json:"context,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type SubGoal struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	ParentID    string     `json:"parent_id"`
	Created     time.Time  `json:"created"`
	Status      string     `json:"status"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Created     time.Time  `json:"created"`
	Completed   bool       `json:"completed"`
	Priority    string     `json:"priority,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type DailySummary struct {
	Date      string `json:"date"`
	Completed int    `json:"completed"`
	Added     int    `json:"added"`
	Notes     string `json:"notes,omitempty"`
}

type GoalsData struct {
	MainGoals      []MainGoal     `json:"main_goals"`
	SubGoals       []SubGoal      `json:"sub_goals"`
	DailySummaries []DailySummary `json:"daily_summaries"`
	LastReminder   *time.Time     `json:"last_reminder"`
	Tasks          []Task         `json:"tasks"`
}

// File operations
func getGoalsFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".local", "share", "opencode", "goals.json")
}

func ensureGoalsFile() {
	path := getGoalsFilePath()
	dir := filepath.Dir(path)
	
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}
	
	if _, err := os.Stat(path); os.IsNotExist(err) {
		data := GoalsData{
			MainGoals:      []MainGoal{},
			SubGoals:       []SubGoal{},
			DailySummaries: []DailySummary{},
			Tasks:          []Task{},
		}
		writeGoals(data)
	}
}

func readGoals() GoalsData {
	ensureGoalsFile()
	path := getGoalsFilePath()
	
	file, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading goals file: %v\n", err)
		os.Exit(1)
	}
	
	var data GoalsData
	if err := json.Unmarshal(file, &data); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing goals file: %v\n", err)
		os.Exit(1)
	}
	
	// Ensure tasks array exists
	if data.Tasks == nil {
		data.Tasks = []Task{}
	}
	
	return data
}

func writeGoals(data GoalsData) {
	path := getGoalsFilePath()
	
	// Backup existing file
	if _, err := os.Stat(path); err == nil {
		backup := path + ".backup"
		if err := os.Rename(path, backup); err != nil {
			// If rename fails, try copy
			input, _ := os.ReadFile(path)
			os.WriteFile(backup, input, 0644)
		}
		// Restore original file name for writing
		defer func() {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				os.Rename(backup, path)
			}
		}()
	}
	
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing goals file: %v\n", err)
		os.Exit(1)
	}
}

// Helper functions
func generateID(prefix string) string {
	timestamp := time.Now().UnixMilli()
	random := rand.Intn(99999)
	return fmt.Sprintf("%s-%d-%05d", prefix, timestamp, random)
}

func calculateProgress(mainGoalID string, data GoalsData) int {
	var total, completed int
	for _, sg := range data.SubGoals {
		if sg.ParentID == mainGoalID {
			total++
			if sg.Status == "completed" {
				completed++
			}
		}
	}
	
	if total == 0 {
		return 0
	}
	return (completed * 100) / total
}

func printSeparator() {
	fmt.Println(strings.Repeat("━", 50))
}

// Goal operations
func listGoals() {
	data := readGoals()
	
	fmt.Println("\n📋 Current Goals")
	printSeparator()
	
	if len(data.MainGoals) == 0 {
		fmt.Println("\nNo goals yet! Use /goals-main to add your first goal.\n")
		return
	}
	
	for _, mg := range data.MainGoals {
		progress := calculateProgress(mg.ID, data)
		
		statusIcon := "⏸️"
		if mg.Status == "completed" {
			statusIcon = "✅"
		} else if mg.Status == "in_progress" {
			statusIcon = "🔄"
		}
		
		fmt.Printf("\n%s %s\n", statusIcon, mg.Title)
		fmt.Printf("   ID: %s | Progress: %d%% | Status: %s\n", mg.ID, progress, mg.Status)
		
		if len(mg.Context) > 0 {
			fmt.Printf("   Context: %s\n", strings.Join(mg.Context, ", "))
		}
		
		// Print sub-goals
		subGoals := []SubGoal{}
		for _, sg := range data.SubGoals {
			if sg.ParentID == mg.ID {
				subGoals = append(subGoals, sg)
			}
		}
		
		if len(subGoals) > 0 {
			fmt.Println("   Sub-goals:")
			for _, sg := range subGoals {
				icon := "○"
				if sg.Status == "completed" {
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
	
	newGoal := MainGoal{
		ID:       generateID("mg"),
		Title:    title,
		Created:  time.Now(),
		Status:   "in_progress",
		Progress: 0,
		SubGoals: []string{},
		Context:  context,
	}
	
	data.MainGoals = append(data.MainGoals, newGoal)
	writeGoals(data)
	
	fmt.Printf("\n✅ Added main goal: \"%s\"\n", title)
	fmt.Printf("   ID: %s\n\n", newGoal.ID)
}

func addSubGoal(title, parentID string) {
	data := readGoals()
	
	// Find parent
	parentIdx := -1
	for i, mg := range data.MainGoals {
		if mg.ID == parentID {
			parentIdx = i
			break
		}
	}
	
	if parentIdx == -1 {
		fmt.Fprintf(os.Stderr, "\n❌ Error: Main goal with ID \"%s\" not found.\n\n", parentID)
		os.Exit(1)
	}
	
	newSubGoal := SubGoal{
		ID:       generateID("sg"),
		Title:    title,
		ParentID: parentID,
		Created:  time.Now(),
		Status:   "pending",
	}
	
	data.SubGoals = append(data.SubGoals, newSubGoal)
	data.MainGoals[parentIdx].SubGoals = append(data.MainGoals[parentIdx].SubGoals, newSubGoal.ID)
	
	writeGoals(data)
	
	fmt.Printf("\n✅ Added sub-goal: \"%s\"\n", title)
	fmt.Printf("   Parent: %s\n", data.MainGoals[parentIdx].Title)
	fmt.Printf("   ID: %s\n\n", newSubGoal.ID)
}

func markDone(goalID string) {
	data := readGoals()
	now := time.Now()
	
	// Check main goals
	for i := range data.MainGoals {
		if data.MainGoals[i].ID == goalID {
			data.MainGoals[i].Status = "completed"
			data.MainGoals[i].CompletedAt = &now
			writeGoals(data)
			fmt.Printf("\n✅ Marked as completed: \"%s\"\n\n", data.MainGoals[i].Title)
			return
		}
	}
	
	// Check sub goals
	for i := range data.SubGoals {
		if data.SubGoals[i].ID == goalID {
			data.SubGoals[i].Status = "completed"
			data.SubGoals[i].CompletedAt = &now
			
			// Update parent progress
			parentID := data.SubGoals[i].ParentID
			for j := range data.MainGoals {
				if data.MainGoals[j].ID == parentID {
					data.MainGoals[j].Progress = calculateProgress(parentID, data)
					
					// Auto-complete main goal if all sub-goals done
					allComplete := true
					for _, sg := range data.SubGoals {
						if sg.ParentID == parentID && sg.Status != "completed" {
							allComplete = false
							break
						}
					}
					
					if allComplete {
						data.MainGoals[j].Status = "completed"
						data.MainGoals[j].CompletedAt = &now
					}
					break
				}
			}
			
			writeGoals(data)
			fmt.Printf("\n✅ Marked as completed: \"%s\"\n\n", data.SubGoals[i].Title)
			return
		}
	}
	
	fmt.Fprintf(os.Stderr, "\n❌ Error: Goal with ID \"%s\" not found.\n\n", goalID)
	os.Exit(1)
}

func generateSummary() {
	data := readGoals()
	today := time.Now().Format("2006-01-02")
	
	// Filter by today
	completedToday := []SubGoal{}
	for _, sg := range data.SubGoals {
		if sg.CompletedAt != nil && sg.CompletedAt.Format("2006-01-02") == today {
			completedToday = append(completedToday, sg)
		}
	}
	
	addedToday := []string{}
	for _, mg := range data.MainGoals {
		if mg.Created.Format("2006-01-02") == today {
			addedToday = append(addedToday, mg.Title)
		}
	}
	for _, sg := range data.SubGoals {
		if sg.Created.Format("2006-01-02") == today {
			addedToday = append(addedToday, sg.Title)
		}
	}
	
	inProgress := []MainGoal{}
	for _, mg := range data.MainGoals {
		if mg.Status == "in_progress" {
			inProgress = append(inProgress, mg)
		}
	}
	
	fmt.Printf("\n📊 Daily Summary - %s\n", time.Now().Format("January 2, 2006"))
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
			progress := calculateProgress(mg.ID, data)
			fmt.Printf("  - %s (%d%%)\n", mg.Title, progress)
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
		limit := 3
		if len(inProgress) < limit {
			limit = len(inProgress)
		}
		
		for i := 0; i < limit; i++ {
			mg := inProgress[i]
			var nextSubGoal *SubGoal
			for j := range data.SubGoals {
				if data.SubGoals[j].ParentID == mg.ID && data.SubGoals[j].Status == "pending" {
					nextSubGoal = &data.SubGoals[j]
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
	
	inProgress := []MainGoal{}
	for _, mg := range data.MainGoals {
		if mg.Status == "in_progress" {
			inProgress = append(inProgress, mg)
		}
	}
	
	pendingSubGoals := []SubGoal{}
	for _, sg := range data.SubGoals {
		if sg.Status == "pending" {
			pendingSubGoals = append(pendingSubGoals, sg)
		}
	}
	
	fmt.Println("\n🎯 Goal Reminder")
	printSeparator()
	
	if len(inProgress) == 0 {
		fmt.Println("\n📭 No active goals. Use /goals-main to add a goal!\n")
		return
	}
	
	fmt.Printf("\n📌 %d main goal(s) in progress\n", len(inProgress))
	fmt.Printf("📌 %d sub-goal(s) pending\n\n", len(pendingSubGoals))
	
	// Show top priority
	topGoal := inProgress[0]
	progress := calculateProgress(topGoal.ID, data)
	
	fmt.Printf("🔥 Focus Now: %s (%d%%)\n", topGoal.Title, progress)
	
	for _, sg := range data.SubGoals {
		if sg.ParentID == topGoal.ID && sg.Status == "pending" {
			fmt.Printf("   Next Step: %s\n", sg.Title)
			break
		}
	}
	
	fmt.Println()
	printSeparator()
	fmt.Println()
}

// Task operations
func listTasks() {
	data := readGoals()
	
	pending := []Task{}
	completed := []Task{}
	
	for _, t := range data.Tasks {
		if t.Completed {
			completed = append(completed, t)
		} else {
			pending = append(pending, t)
		}
	}
	
	fmt.Println("\n📝 Task List")
	printSeparator()
	
	if len(pending) == 0 && len(completed) == 0 {
		fmt.Println("\nNo tasks yet! Use /task-add to add your first task.\n")
		return
	}
	
	if len(pending) > 0 {
		fmt.Printf("\n⏳ Pending (%d):\n", len(pending))
		for i, task := range pending {
			priority := ""
			if task.Priority != "" {
				priority = fmt.Sprintf(" [%s]", task.Priority)
			}
			fmt.Printf("  %d. ○ %s%s\n", i+1, task.Title, priority)
			fmt.Printf("     ID: %s | Created: %s\n", task.ID, task.Created.Format("1/2/2006"))
		}
	}
	
	if len(completed) > 0 {
		fmt.Printf("\n✅ Completed (%d):\n", len(completed))
		limit := 10
		if len(completed) < limit {
			limit = len(completed)
		}
		for i := 0; i < limit; i++ {
			task := completed[i]
			fmt.Printf("  ✓ %s\n", task.Title)
			if task.CompletedAt != nil {
				fmt.Printf("     Completed: %s\n", task.CompletedAt.Format("1/2/2006"))
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

func addTask(title, priority string) {
	data := readGoals()
	
	newTask := Task{
		ID:       generateID("task"),
		Title:    title,
		Created:  time.Now(),
		Completed: false,
		Priority: priority,
	}
	
	data.Tasks = append(data.Tasks, newTask)
	writeGoals(data)
	
	fmt.Printf("\n✅ Added task: \"%s\"\n", title)
	if priority != "" {
		fmt.Printf("   Priority: %s\n", priority)
	}
	fmt.Printf("   ID: %s\n\n", newTask.ID)
}

func markTaskDone(taskID string) {
	data := readGoals()
	now := time.Now()
	
	for i := range data.Tasks {
		if data.Tasks[i].ID == taskID {
			data.Tasks[i].Completed = true
			data.Tasks[i].CompletedAt = &now
			writeGoals(data)
			fmt.Printf("\n✅ Marked task as completed: \"%s\"\n\n", data.Tasks[i].Title)
			return
		}
	}
	
	fmt.Fprintf(os.Stderr, "\n❌ Error: Task with ID \"%s\" not found.\n\n", taskID)
	os.Exit(1)
}

func deleteTask(taskID string) {
	data := readGoals()
	
	for i := range data.Tasks {
		if data.Tasks[i].ID == taskID {
			title := data.Tasks[i].Title
			data.Tasks = append(data.Tasks[:i], data.Tasks[i+1:]...)
			writeGoals(data)
			fmt.Printf("\n🗑️  Deleted task: \"%s\"\n\n", title)
			return
		}
	}
	
	fmt.Fprintf(os.Stderr, "\n❌ Error: Task with ID \"%s\" not found.\n\n", taskID)
	os.Exit(1)
}

func clearCompletedTasks() {
	data := readGoals()
	
	completedCount := 0
	newTasks := []Task{}
	
	for _, t := range data.Tasks {
		if t.Completed {
			completedCount++
		} else {
			newTasks = append(newTasks, t)
		}
	}
	
	data.Tasks = newTasks
	writeGoals(data)
	
	fmt.Printf("\n🗑️  Cleared %d completed task(s)\n\n", completedCount)
}

// Today dashboard
func showToday() {
	data := readGoals()
	today := time.Now().Format("2006-01-02")
	
	fmt.Println("\n╔════════════════════════════════════════════════╗")
	fmt.Printf("║  📅 TODAY - %s\n", time.Now().Format("Monday, January 2, 2006"))
	fmt.Println("╚════════════════════════════════════════════════╝")
	
	// Active Goals Section
	inProgress := []MainGoal{}
	for _, mg := range data.MainGoals {
		if mg.Status == "in_progress" {
			inProgress = append(inProgress, mg)
		}
	}
	
	fmt.Println("\n🎯 ACTIVE GOALS")
	printSeparator()
	
	if len(inProgress) > 0 {
		for _, mg := range inProgress {
			progress := calculateProgress(mg.ID, data)
			var nextSubGoal *SubGoal
			for i := range data.SubGoals {
				if data.SubGoals[i].ParentID == mg.ID && data.SubGoals[i].Status == "pending" {
					nextSubGoal = &data.SubGoals[i]
					break
				}
			}
			
			fmt.Printf("\n%s [%d%%]\n", mg.Title, progress)
			if nextSubGoal != nil {
				fmt.Printf("  → Next: %s\n", nextSubGoal.Title)
			}
		}
	} else {
		fmt.Println("\n  No active goals. Use /goals-main to add one!")
	}
	
	// Pending Tasks Section
	pending := []Task{}
	highPriority := []Task{}
	mediumPriority := []Task{}
	otherTasks := []Task{}
	
	for _, t := range data.Tasks {
		if !t.Completed {
			pending = append(pending, t)
			switch t.Priority {
			case "high":
				highPriority = append(highPriority, t)
			case "medium":
				mediumPriority = append(mediumPriority, t)
			default:
				otherTasks = append(otherTasks, t)
			}
		}
	}
	
	fmt.Println("\n\n📝 TASKS")
	printSeparator()
	
	if len(pending) > 0 {
		if len(highPriority) > 0 {
			fmt.Println("\n🔴 HIGH PRIORITY:")
			for i, task := range highPriority {
				fmt.Printf("  %d. %s\n", i+1, task.Title)
			}
		}
		
		if len(mediumPriority) > 0 {
			fmt.Println("\n🟡 MEDIUM PRIORITY:")
			for i, task := range mediumPriority {
				fmt.Printf("  %d. %s\n", i+1, task.Title)
			}
		}
		
		if len(otherTasks) > 0 {
			fmt.Println("\n⚪ OTHER:")
			for i, task := range otherTasks {
				fmt.Printf("  %d. %s\n", i+1, task.Title)
			}
		}
	} else {
		fmt.Println("\n  No pending tasks. Use /task-add to add one!")
	}
	
	// Today's Progress Section
	completedToday := []SubGoal{}
	for _, sg := range data.SubGoals {
		if sg.CompletedAt != nil && sg.CompletedAt.Format("2006-01-02") == today {
			completedToday = append(completedToday, sg)
		}
	}
	
	tasksCompletedToday := []Task{}
	for _, t := range data.Tasks {
		if t.CompletedAt != nil && t.CompletedAt.Format("2006-01-02") == today {
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
	
	// Focus Section
	fmt.Println("\n\n🔥 FOCUS NOW")
	printSeparator()
	
	focusCount := 0
	
	// Suggest high priority tasks first
	if len(highPriority) > 0 {
		fmt.Printf("\n  1. %s (high priority task)\n", highPriority[0].Title)
		focusCount++
	}
	
	// Then suggest next sub-goal
	if len(inProgress) > 0 {
		topGoal := inProgress[0]
		for _, sg := range data.SubGoals {
			if sg.ParentID == topGoal.ID && sg.Status == "pending" {
				fmt.Printf("\n  %d. %s (%s)\n", focusCount+1, sg.Title, topGoal.Title)
				focusCount++
				break
			}
		}
	}
	
	// Then other tasks
	if len(highPriority) > 1 && focusCount < 3 {
		fmt.Printf("\n  %d. %s (high priority task)\n", focusCount+1, highPriority[1].Title)
		focusCount++
	}
	
	if len(mediumPriority) > 0 && focusCount < 3 {
		fmt.Printf("\n  %d. %s (medium priority task)\n", focusCount+1, mediumPriority[0].Title)
		focusCount++
	}
	
	if focusCount == 0 {
		fmt.Println("\n  Add some goals or tasks to get started!")
		fmt.Println("  • /goals-main <title> - Add a main goal")
		fmt.Println("  • /task-add <title> - Add a quick task")
	}
	
	// Quick Stats
	fmt.Println("\n\n📊 STATS")
	printSeparator()
	fmt.Printf("  Active Goals: %d\n", len(inProgress))
	fmt.Printf("  Pending Tasks: %d\n", len(pending))
	fmt.Printf("  Completed Today: %d\n", len(completedToday)+len(tasksCompletedToday))
	
	fmt.Println("\n" + strings.Repeat("═", 50) + "\n")
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "\n❌ Error: Unknown command\n")
	fmt.Fprintf(os.Stderr, "\nAvailable commands:\n")
	fmt.Fprintf(os.Stderr, "  Goals:\n")
	fmt.Fprintf(os.Stderr, "    list              - List all goals\n")
	fmt.Fprintf(os.Stderr, "    add-main <title>  - Add a main goal\n")
	fmt.Fprintf(os.Stderr, "    add-sub <id> <title> - Add a sub-goal\n")
	fmt.Fprintf(os.Stderr, "    done <id>         - Mark goal as complete\n")
	fmt.Fprintf(os.Stderr, "    summary           - Generate daily summary\n")
	fmt.Fprintf(os.Stderr, "    remind            - Show reminder\n")
	fmt.Fprintf(os.Stderr, "  Tasks:\n")
	fmt.Fprintf(os.Stderr, "    task-list         - List all tasks\n")
	fmt.Fprintf(os.Stderr, "    task-add <title> [priority] - Add a task (priority: high/medium/low)\n")
	fmt.Fprintf(os.Stderr, "    task-done <id>    - Mark task as complete\n")
	fmt.Fprintf(os.Stderr, "    task-delete <id>  - Delete a task\n")
	fmt.Fprintf(os.Stderr, "    task-clear        - Clear all completed tasks\n")
	fmt.Fprintf(os.Stderr, "  Dashboard:\n")
	fmt.Fprintf(os.Stderr, "    today             - Show today's dashboard\n\n")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	
	command := os.Args[1]
	args := os.Args[2:]
	
	switch command {
	case "list":
		listGoals()
		
	case "add-main":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "\n❌ Error: Please provide a goal title.\n\n")
			os.Exit(1)
		}
		addMainGoal(strings.Join(args, " "), []string{})
		
	case "add-sub":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "\n❌ Error: Please provide parent ID and goal title.\n")
			fmt.Fprintf(os.Stderr, "Usage: goals add-sub <parent-id> <title>\n\n")
			os.Exit(1)
		}
		addSubGoal(strings.Join(args[1:], " "), args[0])
		
	case "done":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "\n❌ Error: Please provide a goal ID.\n\n")
			os.Exit(1)
		}
		markDone(args[0])
		
	case "summary":
		generateSummary()
		
	case "remind":
		remindMe()
		
	case "task-list":
		listTasks()
		
	case "task-add":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "\n❌ Error: Please provide a task title.\n\n")
			os.Exit(1)
		}
		
		// Check if last arg is a priority
		priorities := map[string]bool{"high": true, "medium": true, "low": true}
		lastArg := strings.ToLower(args[len(args)-1])
		
		var title, priority string
		if priorities[lastArg] {
			priority = lastArg
			title = strings.Join(args[:len(args)-1], " ")
		} else {
			title = strings.Join(args, " ")
		}
		
		addTask(title, priority)
		
	case "task-done":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "\n❌ Error: Please provide a task ID.\n\n")
			os.Exit(1)
		}
		markTaskDone(args[0])
		
	case "task-delete":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "\n❌ Error: Please provide a task ID.\n\n")
			os.Exit(1)
		}
		deleteTask(args[0])
		
	case "task-clear":
		clearCompletedTasks()
		
	case "today":
		showToday()
		
	default:
		printUsage()
		os.Exit(1)
	}
}
