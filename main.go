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

// ──────────────────────────────────────────────────────────────────
// Constants
// ──────────────────────────────────────────────────────────────────

const (
	// Goal/sub-goal statuses.
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"

	// Task priorities.
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"

	// Date formats.
	dateFmtISO     = "2006-01-02"
	dateFmtDisplay = "1/2/2006"
	dateFmtHeader  = "Monday, January 2, 2006"
	dateFmtSummary = "January 2, 2006"

	// UI.
	separatorWidth   = 50
	focusListLimit   = 3
	completedShowMax = 10
)

// ──────────────────────────────────────────────────────────────────
// Data structures
// ──────────────────────────────────────────────────────────────────

type List struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Created time.Time `json:"created"`
}

type MainGoal struct {
	ID          string     `json:"id"`
	ListID      string     `json:"list_id"`
	Title       string     `json:"title"`
	Created     time.Time  `json:"created"`
	Status      string     `json:"status"`
	Progress    int        `json:"progress"`
	SubGoals    []string   `json:"sub_goals"`
	Context     []string   `json:"context,omitempty"`
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

type GoalsData struct {
	Lists        []List     `json:"lists"`
	ActiveListID string     `json:"active_list_id"`
	MainGoals    []MainGoal `json:"main_goals"`
	SubGoals     []SubGoal  `json:"sub_goals"`
	Tasks        []Task     `json:"tasks"`
}

// ──────────────────────────────────────────────────────────────────
// Error helpers
// ──────────────────────────────────────────────────────────────────

// die prints an error message to stderr and exits with code 1.
func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\n❌ "+format+"\n\n", args...)
	os.Exit(1)
}

// fatalf is like die but for internal/IO errors (no leading emoji).
func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// ──────────────────────────────────────────────────────────────────
// File I/O
// ──────────────────────────────────────────────────────────────────

func getGoalsFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fatalf("Error getting home directory: %v", err)
	}
	return filepath.Join(home, ".local", "share", "opencode", "goals.json")
}

func ensureGoalsFile() {
	path := getGoalsFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		fatalf("Error creating directory: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		writeGoals(GoalsData{
			Lists:     []List{},
			MainGoals: []MainGoal{},
			SubGoals:  []SubGoal{},
			Tasks:     []Task{},
		})
	}
}

func readGoals() GoalsData {
	ensureGoalsFile()
	path := getGoalsFilePath()

	raw, err := os.ReadFile(path)
	if err != nil {
		fatalf("Error reading goals file: %v", err)
	}

	var data GoalsData
	if err := json.Unmarshal(raw, &data); err != nil {
		fatalf("Error parsing goals file: %v", err)
	}

	// Defensive nil-init for forward compatibility with older files.
	if data.Tasks == nil {
		data.Tasks = []Task{}
	}
	if data.Lists == nil {
		data.Lists = []List{}
	}

	mutated := ensureActiveList(&data)
	mutated = migrateOrphanListIDs(&data) || mutated

	if mutated {
		writeGoals(data)
	}
	return data
}

// writeGoals atomically replaces goals.json by writing to a temp file
// in the same directory and renaming over the target. This guarantees
// the original is intact if marshalling or writing fails.
func writeGoals(data GoalsData) {
	path := getGoalsFilePath()
	dir := filepath.Dir(path)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fatalf("Error marshaling JSON: %v", err)
	}

	tmp, err := os.CreateTemp(dir, "goals-*.json.tmp")
	if err != nil {
		fatalf("Error creating temp file: %v", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(jsonData); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		fatalf("Error writing temp file: %v", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		fatalf("Error closing temp file: %v", err)
	}
	if err := os.Chmod(tmpName, 0644); err != nil {
		os.Remove(tmpName)
		fatalf("Error chmod temp file: %v", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		fatalf("Error replacing goals file: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────
// List management & migration
// ──────────────────────────────────────────────────────────────────

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
	for _, l := range data.Lists {
		if l.ID == data.ActiveListID {
			return false
		}
	}
	// Active list missing or unset → fall back to first list.
	data.ActiveListID = data.Lists[0].ID
	return true
}

// migrateOrphanListIDs assigns the active list's ID to any goal/sub-goal/task
// that predates multi-list support and has an empty ListID. This is a
// one-shot, idempotent migration; once persisted, future reads are no-ops.
// If there is no active list, nothing is migrated.
func migrateOrphanListIDs(data *GoalsData) bool {
	if data.ActiveListID == "" {
		return false
	}
	mutated := false
	for i := range data.MainGoals {
		if data.MainGoals[i].ListID == "" {
			data.MainGoals[i].ListID = data.ActiveListID
			mutated = true
		}
	}
	for i := range data.SubGoals {
		if data.SubGoals[i].ListID == "" {
			data.SubGoals[i].ListID = data.ActiveListID
			mutated = true
		}
	}
	for i := range data.Tasks {
		if data.Tasks[i].ListID == "" {
			data.Tasks[i].ListID = data.ActiveListID
			mutated = true
		}
	}
	return mutated
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

func activeListName(data GoalsData) string {
	for _, l := range data.Lists {
		if l.ID == data.ActiveListID {
			return l.Name
		}
	}
	return "(none)"
}

// ──────────────────────────────────────────────────────────────────
// Generic per-list filter
// ──────────────────────────────────────────────────────────────────

// filterByList returns all items whose list_id equals listID.
// After migration, every item has a list_id, so no fallback is needed.
func filterByList[T any](items []T, getListID func(T) string, listID string) []T {
	out := make([]T, 0, len(items))
	for _, it := range items {
		if getListID(it) == listID {
			out = append(out, it)
		}
	}
	return out
}

func goalsForList(data GoalsData, listID string) []MainGoal {
	return filterByList(data.MainGoals, func(m MainGoal) string { return m.ListID }, listID)
}

func subGoalsForList(data GoalsData, listID string) []SubGoal {
	return filterByList(data.SubGoals, func(s SubGoal) string { return s.ListID }, listID)
}

func tasksForList(data GoalsData, listID string) []Task {
	return filterByList(data.Tasks, func(t Task) string { return t.ListID }, listID)
}

// ──────────────────────────────────────────────────────────────────
// Misc helpers
// ──────────────────────────────────────────────────────────────────

func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d-%05d", prefix, time.Now().UnixMilli(), rand.Intn(99999))
}

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

func printSeparator() {
	fmt.Println(strings.Repeat("━", separatorWidth))
}

func today() string {
	return time.Now().Format(dateFmtISO)
}

// ──────────────────────────────────────────────────────────────────
// Goal operations
// ──────────────────────────────────────────────────────────────────

func listGoals() {
	data := readGoals()
	if len(data.Lists) == 0 {
		fmt.Println("\n📋 Current Goals")
		printSeparator()
		fmt.Printf("\n  No lists yet. Create one with:  goals list-create <name>\n\n")
		return
	}
	listID := data.ActiveListID
	mainGoals := goalsForList(data, listID)

	fmt.Printf("\n📋 Current Goals  [list: %s]\n", activeListName(data))
	printSeparator()

	if len(mainGoals) == 0 {
		fmt.Printf("\nNo goals yet! Use /goals-main to add your first goal.\n\n")
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

		subs := subGoalsForList(data, listID)
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
	requireActiveList(data)

	newGoal := MainGoal{
		ID:       generateID("mg"),
		ListID:   data.ActiveListID,
		Title:    title,
		Created:  time.Now(),
		Status:   StatusInProgress,
		Progress: 0,
		SubGoals: []string{},
		Context:  context,
	}

	data.MainGoals = append(data.MainGoals, newGoal)
	writeGoals(data)

	fmt.Printf("\n✅ Added main goal: %q  [list: %s]\n", title, activeListName(data))
	fmt.Printf("   ID: %s\n\n", newGoal.ID)
}

func addSubGoal(title, parentID string) {
	data := readGoals()
	requireActiveList(data)

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
		ID:       generateID("sg"),
		ListID:   data.MainGoals[parentIdx].ListID,
		Title:    title,
		ParentID: parentID,
		Created:  time.Now(),
		Status:   StatusPending,
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

func generateSummary() {
	data := readGoals()
	if len(data.Lists) == 0 {
		fmt.Println("\n📊 Daily Summary")
		printSeparator()
		fmt.Printf("\n  No lists yet. Create one with:  goals list-create <name>\n\n")
		return
	}
	listID := data.ActiveListID
	todayStr := today()

	listMains := goalsForList(data, listID)
	listSubs := subGoalsForList(data, listID)

	var completedToday []SubGoal
	for _, sg := range listSubs {
		if sg.CompletedAt != nil && sg.CompletedAt.Format(dateFmtISO) == todayStr {
			completedToday = append(completedToday, sg)
		}
	}

	var addedToday []string
	for _, mg := range listMains {
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
	for _, mg := range listMains {
		if mg.Status == StatusInProgress {
			inProgress = append(inProgress, mg)
		}
	}

	fmt.Printf("\n📊 Daily Summary - %s  [list: %s]\n",
		time.Now().Format(dateFmtSummary), activeListName(data))
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
	if len(data.Lists) == 0 {
		fmt.Println("\n🎯 Goal Reminder")
		printSeparator()
		fmt.Printf("\n  No lists yet. Create one with:  goals list-create <name>\n\n")
		return
	}
	listID := data.ActiveListID

	var inProgress []MainGoal
	for _, mg := range goalsForList(data, listID) {
		if mg.Status == StatusInProgress {
			inProgress = append(inProgress, mg)
		}
	}

	var pendingSubGoals []SubGoal
	subs := subGoalsForList(data, listID)
	for _, sg := range subs {
		if sg.Status == StatusPending {
			pendingSubGoals = append(pendingSubGoals, sg)
		}
	}

	fmt.Printf("\n🎯 Goal Reminder  [list: %s]\n", activeListName(data))
	printSeparator()

	if len(inProgress) == 0 {
		fmt.Printf("\n📭 No active goals. Use /goals-main to add a goal!\n\n")
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

// ──────────────────────────────────────────────────────────────────
// Task operations
// ──────────────────────────────────────────────────────────────────

func listTasks() {
	data := readGoals()
	if len(data.Lists) == 0 {
		fmt.Println("\n📝 Task List")
		printSeparator()
		fmt.Printf("\n  No lists yet. Create one with:  goals list-create <name>\n\n")
		return
	}
	listID := data.ActiveListID

	var pending, completed []Task
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
		fmt.Printf("\nNo tasks yet! Use /task-add to add your first task.\n\n")
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

func addTask(title, priority string) {
	data := readGoals()
	requireActiveList(data)

	newTask := Task{
		ID:       generateID("task"),
		ListID:   data.ActiveListID,
		Title:    title,
		Created:  time.Now(),
		Priority: priority,
	}

	data.Tasks = append(data.Tasks, newTask)
	writeGoals(data)

	fmt.Printf("\n✅ Added task: %q\n", title)
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
			fmt.Printf("\n✅ Marked task as completed: %q\n\n", data.Tasks[i].Title)
			return
		}
	}
	die("Error: Task with ID %q not found.", taskID)
}

func deleteTask(taskID string) {
	data := readGoals()
	for i := range data.Tasks {
		if data.Tasks[i].ID == taskID {
			title := data.Tasks[i].Title
			data.Tasks = append(data.Tasks[:i], data.Tasks[i+1:]...)
			writeGoals(data)
			fmt.Printf("\n🗑️  Deleted task: %q\n\n", title)
			return
		}
	}
	die("Error: Task with ID %q not found.", taskID)
}

func clearCompletedTasks() {
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
	fmt.Printf("\n🗑️  Cleared %d completed task(s)\n\n", completedCount)
}

// ──────────────────────────────────────────────────────────────────
// Today dashboard
// ──────────────────────────────────────────────────────────────────

func showToday() {
	data := readGoals()

	if len(data.Lists) == 0 {
		fmt.Println("\n╔════════════════════════════════════════════════╗")
		fmt.Printf("║  📅 TODAY - %s\n", time.Now().Format(dateFmtHeader))
		fmt.Println("║  📂 No lists yet")
		fmt.Println("╚════════════════════════════════════════════════╝")
		fmt.Println("\nYou have no goal lists yet.")
		fmt.Println("Create your first one with:")
		fmt.Println("  goals list-create <name>")
		fmt.Printf("\nOr use /og to manage lists interactively.\n\n")
		return
	}

	listID := data.ActiveListID
	todayStr := today()

	listMains := goalsForList(data, listID)
	listSubs := subGoalsForList(data, listID)
	listTasksAll := tasksForList(data, listID)

	fmt.Println("\n╔════════════════════════════════════════════════╗")
	fmt.Printf("║  📅 TODAY - %s\n", time.Now().Format(dateFmtHeader))
	fmt.Printf("║  📂 List: %s\n", activeListName(data))
	fmt.Println("╚════════════════════════════════════════════════╝")

	// Active Goals
	var inProgress []MainGoal
	for _, mg := range listMains {
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
		fmt.Println("\n  No active goals. Use /goals-main to add one!")
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
		fmt.Println("  • /goals-main <title> - Add a main goal")
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

// ──────────────────────────────────────────────────────────────────
// List operations
// ──────────────────────────────────────────────────────────────────

func listLists() {
	data := readGoals()

	fmt.Println("\n📂 Lists")
	printSeparator()

	if len(data.Lists) == 0 {
		fmt.Println("\n  No lists yet.")
		fmt.Printf("  Create your first list:  goals list-create <name>\n\n")
		return
	}

	for _, l := range data.Lists {
		marker := "  "
		if l.ID == data.ActiveListID {
			marker = "▶ "
		}
		mainCount := len(goalsForList(data, l.ID))
		subCount := len(subGoalsForList(data, l.ID))
		listTasksLocal := tasksForList(data, l.ID)
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
	fmt.Printf("\nActive: %s\n\n", activeListName(data))
}

func listCreate(name string) {
	data := readGoals()

	for _, l := range data.Lists {
		if strings.EqualFold(l.Name, name) {
			die("Error: A list named %q already exists.", name)
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

	fmt.Printf("\n✅ Created list: %q (now active)\n", name)
	fmt.Printf("   ID: %s\n\n", newList.ID)
}

func listUse(idOrName string) {
	data := readGoals()
	idx := findList(data, idOrName)
	if idx == -1 {
		die("Error: List %q not found.", idOrName)
	}
	data.ActiveListID = data.Lists[idx].ID
	writeGoals(data)
	fmt.Printf("\n✅ Active list: %s\n\n", data.Lists[idx].Name)
}

func listRename(idOrName, newName string) {
	data := readGoals()
	idx := findList(data, idOrName)
	if idx == -1 {
		die("Error: List %q not found.", idOrName)
	}
	for i, l := range data.Lists {
		if i != idx && strings.EqualFold(l.Name, newName) {
			die("Error: A list named %q already exists.", newName)
		}
	}
	old := data.Lists[idx].Name
	data.Lists[idx].Name = newName
	writeGoals(data)
	fmt.Printf("\n✅ Renamed list: %q → %q\n\n", old, newName)
}

func listDelete(idOrName string) {
	data := readGoals()
	idx := findList(data, idOrName)
	if idx == -1 {
		die("Error: List %q not found.", idOrName)
	}
	target := data.Lists[idx]

	// Drop everything belonging to this list. After migration, every item
	// has a non-empty ListID, so equality is sufficient.
	keepMains := data.MainGoals[:0]
	for _, mg := range data.MainGoals {
		if mg.ListID != target.ID {
			keepMains = append(keepMains, mg)
		}
	}
	data.MainGoals = append([]MainGoal{}, keepMains...)

	keepSubs := data.SubGoals[:0]
	for _, sg := range data.SubGoals {
		if sg.ListID != target.ID {
			keepSubs = append(keepSubs, sg)
		}
	}
	data.SubGoals = append([]SubGoal{}, keepSubs...)

	keepTasks := data.Tasks[:0]
	for _, t := range data.Tasks {
		if t.ListID != target.ID {
			keepTasks = append(keepTasks, t)
		}
	}
	data.Tasks = append([]Task{}, keepTasks...)

	data.Lists = append(data.Lists[:idx], data.Lists[idx+1:]...)

	if data.ActiveListID == target.ID {
		if len(data.Lists) > 0 {
			data.ActiveListID = data.Lists[0].ID
		} else {
			data.ActiveListID = ""
		}
	}

	writeGoals(data)
	fmt.Printf("\n🗑️  Deleted list: %q (and its goals/tasks)\n", target.Name)
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
		die("Error: List %q not found.", idOrName)
	}
	target := data.Lists[idx]

	fmt.Printf("\n📂 List: %s", target.Name)
	if target.ID == data.ActiveListID {
		fmt.Print("  (active)")
	}
	fmt.Println()
	printSeparator()

	mains := goalsForList(data, target.ID)
	subs := subGoalsForList(data, target.ID)
	tasks := tasksForList(data, target.ID)

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

// ──────────────────────────────────────────────────────────────────
// CLI dispatch
// ──────────────────────────────────────────────────────────────────

func printUsage(toStderr bool) {
	w := os.Stdout
	if toStderr {
		w = os.Stderr
	}
	fmt.Fprint(w, `
Usage: goals <command> [args]

Goals:
  list                              List goals in active list
  add-main <title>                  Add a main goal
  add-sub <parent-id> <title>       Add a sub-goal under <parent-id>
  done <id>                         Mark goal as complete
  summary                           Generate daily summary
  remind                            Show reminder

Tasks:
  task-list                         List tasks in active list
  task-add <title> [priority]       Add a task (priority: high|medium|low)
  task-done <id>                    Mark task as complete
  task-delete <id>                  Delete a task
  task-clear                        Remove all completed tasks

Dashboard:
  today                             Show today's dashboard

Lists:
  list-ls                           Show all lists
  list-create <name>                Create a list and switch to it
  list-use <id|name>                Switch active list
  list-rename <id|name> <new-name>  Rename a list
  list-delete <id|name>             Delete a list and its contents
  list-show <id|name>               Show one list's full tree

Inside OpenCode, prefer the slash commands:
  /today                            Dashboard
  /og   /ogl   /ogc <name>          Lists: browse, list, create
  /ogs <name>   /ogd [name]         Lists: switch, delete
  /goals-main   /goals-sub          Add main / sub-goals
  /goals-list   /goals-done         List goals, mark complete
  /goals-summary   /goals-remind    Daily summary, reminder
  /task-add   /task-list            Tasks: add, list
  /task-done   /task-delete         Tasks: complete, delete
  /task-clear                       Remove all completed tasks
`)
}

func requireArg(args []string, label string) {
	if len(args) == 0 {
		die("Error: Please provide %s.", label)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "❌ No command given.")
		printUsage(true)
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "list":
		listGoals()

	case "add-main":
		requireArg(args, "a goal title")
		addMainGoal(strings.Join(args, " "), []string{})

	case "add-sub":
		if len(args) < 2 {
			die("Error: Usage: goals add-sub <parent-id> <title>")
		}
		addSubGoal(strings.Join(args[1:], " "), args[0])

	case "done":
		requireArg(args, "a goal ID")
		markDone(args[0])

	case "summary":
		generateSummary()

	case "remind":
		remindMe()

	case "task-list":
		listTasks()

	case "task-add":
		requireArg(args, "a task title")
		// Optional last arg is a priority.
		priorities := map[string]bool{
			PriorityHigh: true, PriorityMedium: true, PriorityLow: true,
		}
		lastArg := strings.ToLower(args[len(args)-1])
		var title, priority string
		if priorities[lastArg] && len(args) > 1 {
			priority = lastArg
			title = strings.Join(args[:len(args)-1], " ")
		} else {
			title = strings.Join(args, " ")
		}
		addTask(title, priority)

	case "task-done":
		requireArg(args, "a task ID")
		markTaskDone(args[0])

	case "task-delete":
		requireArg(args, "a task ID")
		deleteTask(args[0])

	case "task-clear":
		clearCompletedTasks()

	case "today":
		showToday()

	case "list-ls":
		listLists()

	case "list-create":
		requireArg(args, "a list name")
		listCreate(strings.Join(args, " "))

	case "list-use":
		requireArg(args, "a list id or name")
		listUse(strings.Join(args, " "))

	case "list-rename":
		if len(args) < 2 {
			die("Error: Usage: goals list-rename <id|name> <new-name>")
		}
		listRename(args[0], strings.Join(args[1:], " "))

	case "list-delete":
		requireArg(args, "a list id or name")
		listDelete(strings.Join(args, " "))

	case "list-show":
		requireArg(args, "a list id or name")
		listShow(strings.Join(args, " "))

	case "help", "-h", "--help":
		printUsage(false)

	default:
		fmt.Fprintf(os.Stderr, "❌ Unknown command: %s\n", command)
		printUsage(true)
		os.Exit(1)
	}
}
