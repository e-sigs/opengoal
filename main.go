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
type List struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Created time.Time `json:"created"`
}

type MainGoal struct {
	ID          string    `json:"id"`
	ListID      string    `json:"list_id"`
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
	ListID      string     `json:"list_id"`
	Title       string     `json:"title"`
	ParentID    string     `json:"parent_id"`
	Created     time.Time  `json:"created"`
	Status      string     `json:"status"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type Task struct {
	ID          string     `json:"id"`
	ListID      string     `json:"list_id"`
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
	Lists          []List         `json:"lists"`
	ActiveListID   string         `json:"active_list_id"`
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
			Lists:          []List{},
			ActiveListID:   "",
			MainGoals:      []MainGoal{},
			SubGoals:       []SubGoal{},
			DailySummaries: []DailySummary{},
			Tasks:          []Task{},
		}
		writeGoals(data)
	}
}

// ensureActiveList syncs ActiveListID with the current Lists slice.
// It does NOT auto-create lists. Returns true if data was mutated.
func ensureActiveList(data *GoalsData) bool {
	if len(data.Lists) == 0 {
		if data.ActiveListID != "" {
			data.ActiveListID = ""
			return true
		}
		return false
	}
	// Verify ActiveListID still exists
	for _, l := range data.Lists {
		if l.ID == data.ActiveListID {
			return false
		}
	}
	// Active list missing or unset → fall back to first list
	data.ActiveListID = data.Lists[0].ID
	return true
}

// requireActiveList errors out if no list exists. Use before any goal/task mutation.
func requireActiveList(data GoalsData) {
	if len(data.Lists) == 0 || data.ActiveListID == "" {
		fmt.Fprintf(os.Stderr, "\n❌ No lists exist yet.\n")
		fmt.Fprintf(os.Stderr, "   Create one first:  goals list-create <name>\n\n")
		os.Exit(1)
	}
}

// findList returns index of list matching id or name (case-insensitive), or -1.
func findList(data GoalsData, idOrName string) int {
	for i, l := range data.Lists {
		if l.ID == idOrName {
			return i
		}
	}
	for i, l := range data.Lists {
		if strings.EqualFold(l.Name, idOrName) {
			return i
		}
	}
	return -1
}

func activeListID(data GoalsData) string {
	return data.ActiveListID
}

func activeListName(data GoalsData) string {
	for _, l := range data.Lists {
		if l.ID == data.ActiveListID {
			return l.Name
		}
	}
	return "(none)"
}

// Filtered views by list_id. Pre-existing items lacking list_id are treated as belonging to the active list.
func goalsForList(data GoalsData, listID string) []MainGoal {
	out := []MainGoal{}
	for _, mg := range data.MainGoals {
		lid := mg.ListID
		if lid == "" {
			lid = data.ActiveListID
		}
		if lid == listID {
			out = append(out, mg)
		}
	}
	return out
}

func subGoalsForList(data GoalsData, listID string) []SubGoal {
	out := []SubGoal{}
	for _, sg := range data.SubGoals {
		lid := sg.ListID
		if lid == "" {
			lid = data.ActiveListID
		}
		if lid == listID {
			out = append(out, sg)
		}
	}
	return out
}

func tasksForList(data GoalsData, listID string) []Task {
	out := []Task{}
	for _, t := range data.Tasks {
		lid := t.ListID
		if lid == "" {
			lid = data.ActiveListID
		}
		if lid == listID {
			out = append(out, t)
		}
	}
	return out
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
	if data.Lists == nil {
		data.Lists = []List{}
	}
	if ensureActiveList(&data) {
		writeGoals(data)
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
	if len(data.Lists) == 0 {
		fmt.Println("\n📋 Current Goals")
		printSeparator()
		fmt.Println("\n  No lists yet. Create one with:  goals list-create <name>\n")
		return
	}
	listID := activeListID(data)
	mainGoals := goalsForList(data, listID)

	fmt.Printf("\n📋 Current Goals  [list: %s]\n", activeListName(data))
	printSeparator()

	if len(mainGoals) == 0 {
		fmt.Println("\nNo goals yet! Use /goals-main to add your first goal.\n")
		return
	}

	for _, mg := range mainGoals {
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

		// Print sub-goals (in same list)
		subGoals := []SubGoal{}
		for _, sg := range subGoalsForList(data, listID) {
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
	requireActiveList(data)

	newGoal := MainGoal{
		ID:       generateID("mg"),
		ListID:   data.ActiveListID,
		Title:    title,
		Created:  time.Now(),
		Status:   "in_progress",
		Progress: 0,
		SubGoals: []string{},
		Context:  context,
	}

	data.MainGoals = append(data.MainGoals, newGoal)
	writeGoals(data)

	fmt.Printf("\n✅ Added main goal: \"%s\"  [list: %s]\n", title, activeListName(data))
	fmt.Printf("   ID: %s\n\n", newGoal.ID)
}

func addSubGoal(title, parentID string) {
	data := readGoals()
	requireActiveList(data)

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
		ListID:   data.MainGoals[parentIdx].ListID,
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
	if len(data.Lists) == 0 {
		fmt.Println("\n📊 Daily Summary")
		printSeparator()
		fmt.Println("\n  No lists yet. Create one with:  goals list-create <name>\n")
		return
	}
	listID := activeListID(data)
	today := time.Now().Format("2006-01-02")

	listMains := goalsForList(data, listID)
	listSubs := subGoalsForList(data, listID)

	// Filter by today
	completedToday := []SubGoal{}
	for _, sg := range listSubs {
		if sg.CompletedAt != nil && sg.CompletedAt.Format("2006-01-02") == today {
			completedToday = append(completedToday, sg)
		}
	}

	addedToday := []string{}
	for _, mg := range listMains {
		if mg.Created.Format("2006-01-02") == today {
			addedToday = append(addedToday, mg.Title)
		}
	}
	for _, sg := range listSubs {
		if sg.Created.Format("2006-01-02") == today {
			addedToday = append(addedToday, sg.Title)
		}
	}

	inProgress := []MainGoal{}
	for _, mg := range listMains {
		if mg.Status == "in_progress" {
			inProgress = append(inProgress, mg)
		}
	}

	fmt.Printf("\n📊 Daily Summary - %s  [list: %s]\n", time.Now().Format("January 2, 2006"), activeListName(data))
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
			for j := range listSubs {
				if listSubs[j].ParentID == mg.ID && listSubs[j].Status == "pending" {
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
	if len(data.Lists) == 0 {
		fmt.Println("\n🎯 Goal Reminder")
		printSeparator()
		fmt.Println("\n  No lists yet. Create one with:  goals list-create <name>\n")
		return
	}
	listID := activeListID(data)

	inProgress := []MainGoal{}
	for _, mg := range goalsForList(data, listID) {
		if mg.Status == "in_progress" {
			inProgress = append(inProgress, mg)
		}
	}

	pendingSubGoals := []SubGoal{}
	for _, sg := range subGoalsForList(data, listID) {
		if sg.Status == "pending" {
			pendingSubGoals = append(pendingSubGoals, sg)
		}
	}

	fmt.Printf("\n🎯 Goal Reminder  [list: %s]\n", activeListName(data))
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

	for _, sg := range subGoalsForList(data, listID) {
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
	if len(data.Lists) == 0 {
		fmt.Println("\n📝 Task List")
		printSeparator()
		fmt.Println("\n  No lists yet. Create one with:  goals list-create <name>\n")
		return
	}
	listID := activeListID(data)

	pending := []Task{}
	completed := []Task{}

	for _, t := range tasksForList(data, listID) {
		if t.Completed {
			completed = append(completed, t)
		} else {
			pending = append(pending, t)
		}
	}

	fmt.Printf("\n📝 Task List  [list: %s]\n", activeListName(data))
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
	requireActiveList(data)

	newTask := Task{
		ID:       generateID("task"),
		ListID:   data.ActiveListID,
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

	if len(data.Lists) == 0 {
		fmt.Println("\n╔════════════════════════════════════════════════╗")
		fmt.Printf("║  📅 TODAY - %s\n", time.Now().Format("Monday, January 2, 2006"))
		fmt.Println("║  📂 No lists yet")
		fmt.Println("╚════════════════════════════════════════════════╝")
		fmt.Println("\nYou have no goal lists yet.")
		fmt.Println("Create your first one with:")
		fmt.Println("  goals list-create <name>")
		fmt.Println("\nOr use /ogl to manage lists interactively.\n")
		return
	}

	listID := activeListID(data)
	today := time.Now().Format("2006-01-02")

	listMains := goalsForList(data, listID)
	listSubs := subGoalsForList(data, listID)
	listTasksAll := tasksForList(data, listID)

	fmt.Println("\n╔════════════════════════════════════════════════╗")
	fmt.Printf("║  📅 TODAY - %s\n", time.Now().Format("Monday, January 2, 2006"))
	fmt.Printf("║  📂 List: %s\n", activeListName(data))
	fmt.Println("╚════════════════════════════════════════════════╝")

	// Active Goals Section
	inProgress := []MainGoal{}
	for _, mg := range listMains {
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
			for i := range listSubs {
				if listSubs[i].ParentID == mg.ID && listSubs[i].Status == "pending" {
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
		fmt.Println("\n  No active goals. Use /goals-main to add one!")
	}

	// Pending Tasks Section
	pending := []Task{}
	highPriority := []Task{}
	mediumPriority := []Task{}
	otherTasks := []Task{}

	for _, t := range listTasksAll {
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
	for _, sg := range listSubs {
		if sg.CompletedAt != nil && sg.CompletedAt.Format("2006-01-02") == today {
			completedToday = append(completedToday, sg)
		}
	}

	tasksCompletedToday := []Task{}
	for _, t := range listTasksAll {
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
		for _, sg := range listSubs {
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

// List operations
func listLists() {
	data := readGoals()

	fmt.Println("\n📂 Lists")
	printSeparator()

	if len(data.Lists) == 0 {
		fmt.Println("\n  No lists yet.")
		fmt.Println("  Create your first list:  goals list-create <name>\n")
		return
	}

	for _, l := range data.Lists {
		marker := "  "
		if l.ID == data.ActiveListID {
			marker = "▶ "
		}
		mainCount := len(goalsForList(data, l.ID))
		subCount := len(subGoalsForList(data, l.ID))
		taskCount := len(tasksForList(data, l.ID))
		pendingTasks := 0
		for _, t := range tasksForList(data, l.ID) {
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
	fmt.Printf("\nActive: %s\n\n", activeListName(data))
}

func listCreate(name string) {
	data := readGoals()

	// Prevent dup names
	for _, l := range data.Lists {
		if strings.EqualFold(l.Name, name) {
			fmt.Fprintf(os.Stderr, "\n❌ Error: A list named \"%s\" already exists.\n\n", name)
			os.Exit(1)
		}
	}

	newList := List{
		ID:      generateID("list"),
		Name:    name,
		Created: time.Now(),
	}
	data.Lists = append(data.Lists, newList)
	data.ActiveListID = newList.ID
	writeGoals(data)

	fmt.Printf("\n✅ Created list: \"%s\" (now active)\n", name)
	fmt.Printf("   ID: %s\n\n", newList.ID)
}

func listUse(idOrName string) {
	data := readGoals()
	idx := findList(data, idOrName)
	if idx == -1 {
		fmt.Fprintf(os.Stderr, "\n❌ Error: List \"%s\" not found.\n\n", idOrName)
		os.Exit(1)
	}
	data.ActiveListID = data.Lists[idx].ID
	writeGoals(data)
	fmt.Printf("\n✅ Active list: %s\n\n", data.Lists[idx].Name)
}

func listRename(idOrName, newName string) {
	data := readGoals()
	idx := findList(data, idOrName)
	if idx == -1 {
		fmt.Fprintf(os.Stderr, "\n❌ Error: List \"%s\" not found.\n\n", idOrName)
		os.Exit(1)
	}
	for i, l := range data.Lists {
		if i != idx && strings.EqualFold(l.Name, newName) {
			fmt.Fprintf(os.Stderr, "\n❌ Error: A list named \"%s\" already exists.\n\n", newName)
			os.Exit(1)
		}
	}
	old := data.Lists[idx].Name
	data.Lists[idx].Name = newName
	writeGoals(data)
	fmt.Printf("\n✅ Renamed list: \"%s\" → \"%s\"\n\n", old, newName)
}

func listDelete(idOrName string) {
	data := readGoals()
	idx := findList(data, idOrName)
	if idx == -1 {
		fmt.Fprintf(os.Stderr, "\n❌ Error: List \"%s\" not found.\n\n", idOrName)
		os.Exit(1)
	}
	target := data.Lists[idx]

	// Drop all goals/subs/tasks belonging to this list
	newMains := []MainGoal{}
	for _, mg := range data.MainGoals {
		if mg.ListID != target.ID {
			newMains = append(newMains, mg)
		}
	}
	newSubs := []SubGoal{}
	for _, sg := range data.SubGoals {
		if sg.ListID != target.ID {
			newSubs = append(newSubs, sg)
		}
	}
	newTasks := []Task{}
	for _, t := range data.Tasks {
		if t.ListID != target.ID {
			newTasks = append(newTasks, t)
		}
	}
	data.MainGoals = newMains
	data.SubGoals = newSubs
	data.Tasks = newTasks
	data.Lists = append(data.Lists[:idx], data.Lists[idx+1:]...)

	if data.ActiveListID == target.ID {
		if len(data.Lists) > 0 {
			data.ActiveListID = data.Lists[0].ID
		} else {
			data.ActiveListID = ""
		}
	}

	writeGoals(data)
	fmt.Printf("\n🗑️  Deleted list: \"%s\" (and its goals/tasks)\n", target.Name)
	if len(data.Lists) == 0 {
		fmt.Printf("   No lists remaining. Create one with:  goals list-create <name>\n\n")
	} else {
		fmt.Printf("   Active list is now: %s\n\n", activeListName(data))
	}
}

func listShow(idOrName string) {
	data := readGoals()
	idx := findList(data, idOrName)
	if idx == -1 {
		fmt.Fprintf(os.Stderr, "\n❌ Error: List \"%s\" not found.\n\n", idOrName)
		os.Exit(1)
	}
	target := data.Lists[idx]

	fmt.Printf("\n📂 List: %s", target.Name)
	if target.ID == data.ActiveListID {
		fmt.Printf("  (active)")
	}
	fmt.Println()
	printSeparator()

	mains := goalsForList(data, target.ID)
	subs := subGoalsForList(data, target.ID)
	tasks := tasksForList(data, target.ID)

	// Goals
	fmt.Println("\n🎯 Goals:")
	if len(mains) == 0 {
		fmt.Println("  (none)")
	}
	for _, mg := range mains {
		progress := calculateProgress(mg.ID, data)
		statusIcon := "⏸️"
		if mg.Status == "completed" {
			statusIcon = "✅"
		} else if mg.Status == "in_progress" {
			statusIcon = "🔄"
		}
		fmt.Printf("  %s %s [%d%%]\n", statusIcon, mg.Title, progress)
		fmt.Printf("     ID: %s\n", mg.ID)
		for _, sg := range subs {
			if sg.ParentID == mg.ID {
				icon := "○"
				if sg.Status == "completed" {
					icon = "✓"
				}
				fmt.Printf("       %s %s\n", icon, sg.Title)
			}
		}
	}

	// Tasks
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
	fmt.Fprintf(os.Stderr, "    today             - Show today's dashboard\n")
	fmt.Fprintf(os.Stderr, "  Lists:\n")
	fmt.Fprintf(os.Stderr, "    list-ls           - Show all lists\n")
	fmt.Fprintf(os.Stderr, "    list-create <name> - Create a new list (and switch to it)\n")
	fmt.Fprintf(os.Stderr, "    list-use <id|name> - Switch active list\n")
	fmt.Fprintf(os.Stderr, "    list-rename <id|name> <new-name> - Rename a list\n")
	fmt.Fprintf(os.Stderr, "    list-delete <id|name> - Delete a list and its contents\n")
	fmt.Fprintf(os.Stderr, "    list-show <id|name> - Show one list's full tree\n\n")
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

	case "list-ls":
		listLists()

	case "list-create":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "\n❌ Error: Please provide a list name.\n\n")
			os.Exit(1)
		}
		listCreate(strings.Join(args, " "))

	case "list-use":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "\n❌ Error: Please provide a list id or name.\n\n")
			os.Exit(1)
		}
		listUse(strings.Join(args, " "))

	case "list-rename":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "\n❌ Error: Usage: goals list-rename <id|name> <new-name>\n\n")
			os.Exit(1)
		}
		listRename(args[0], strings.Join(args[1:], " "))

	case "list-delete":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "\n❌ Error: Please provide a list id or name.\n\n")
			os.Exit(1)
		}
		listDelete(strings.Join(args, " "))

	case "list-show":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "\n❌ Error: Please provide a list id or name.\n\n")
			os.Exit(1)
		}
		listShow(strings.Join(args, " "))

	default:
		printUsage()
		os.Exit(1)
	}
}
