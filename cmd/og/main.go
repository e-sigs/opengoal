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

type Roadmap struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Created time.Time `json:"created"`
}

type MainGoal struct {
	ID          string     `json:"id"`
	RoadmapID   string     `json:"list_id"`
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
	RoadmapID   string     `json:"list_id"`
	Title       string     `json:"title"`
	ParentID    string     `json:"parent_id"`
	Created     time.Time  `json:"created"`
	Status      string     `json:"status"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type Task struct {
	ID          string     `json:"id"`
	RoadmapID   string     `json:"list_id"`
	Title       string     `json:"title"`
	Created     time.Time  `json:"created"`
	Completed   bool       `json:"completed"`
	Priority    string     `json:"priority,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Multi-agent coordination (see claims.go).
	// Assignee is the agent ID currently holding the claim, or "".
	// ClaimedAt is when the claim was taken; combined with the TTL
	// (claimTTL) it determines whether the claim is live or stale.
	Assignee  string     `json:"assignee,omitempty"`
	ClaimedAt *time.Time `json:"claimed_at,omitempty"`

	// DependsOn lists task IDs that must be completed before this task
	// can be claimed. Stale or live claims on dependencies do not block
	// — only the Completed flag matters. Empty/nil means no deps.
	DependsOn []string `json:"depends_on,omitempty"`
}

type GoalsData struct {
	Roadmaps        []Roadmap  `json:"lists"`
	ActiveRoadmapID string     `json:"active_list_id"`
	MainGoals       []MainGoal `json:"main_goals"`
	SubGoals        []SubGoal  `json:"sub_goals"`
	Tasks           []Task     `json:"tasks"`
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
			Roadmaps:  []Roadmap{},
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
	if data.Roadmaps == nil {
		data.Roadmaps = []Roadmap{}
	}

	mutated := ensureActiveRoadmap(&data)
	mutated = migrateOrphanRoadmapIDs(&data) || mutated

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
// Roadmap management & migration
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

func goalsForRoadmap(data GoalsData, listID string) []MainGoal {
	return filterByRoadmap(data.MainGoals, func(m MainGoal) string { return m.RoadmapID }, listID)
}

func subGoalsForRoadmap(data GoalsData, listID string) []SubGoal {
	return filterByRoadmap(data.SubGoals, func(s SubGoal) string { return s.RoadmapID }, listID)
}

func tasksForRoadmap(data GoalsData, listID string) []Task {
	return filterByRoadmap(data.Tasks, func(t Task) string { return t.RoadmapID }, listID)
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

// ──────────────────────────────────────────────────────────────────
// Task operations
// ──────────────────────────────────────────────────────────────────

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

// ──────────────────────────────────────────────────────────────────
// Today dashboard
// ──────────────────────────────────────────────────────────────────

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

// ──────────────────────────────────────────────────────────────────
// Roadmap operations
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

// ──────────────────────────────────────────────────────────────────
// CLI dispatch
// ──────────────────────────────────────────────────────────────────

func printUsage(toStderr bool) {
	w := os.Stdout
	if toStderr {
		w = os.Stderr
	}
	fmt.Fprint(w, `
Usage: og <command> [args]

Goals:
  list                              Roadmap goals in active roadmap
  add-main <title>                  Add a main goal
  add-sub <parent-id> <title>       Add a sub-goal under <parent-id>
  done <id...> [-y]                 Mark one or more goals as complete
  summary                           Generate daily summary
  remind                            Show reminder

Tasks:
  task-list                         Roadmap tasks in active roadmap
  task-add <title> [priority] [--depends id1,id2]
                                    Add a task (priority: high|medium|low)
  task-show <id>                    Show one task with deps + claim status
  task-done <id>                    Mark task as complete
  task-delete <id...> [-y]          Delete one or more tasks
  task-delete --all [-y]            Delete every task in active roadmap
  task-delete --priority <h|m|l>    Delete every task at given priority
  task-delete --completed           Delete every completed task
  task-clear                        Remove all completed tasks

Multi-agent coordination:
  task-next [--claim]               Show next actionable task; --claim takes it
                                    (skips completed, claimed, and blocked tasks)
  task-claim <id>                   Claim a task for the current agent
                                    (refused if deps unfinished or claim live)
  task-release <id>                 Release a claim you hold (or a stale one)

  Set $OPENGOAL_AGENT to identify the agent (default: hostname-pid).
  Stale claims auto-expire after $OPENGOAL_CLAIM_TTL seconds (default 1800).

Event log:
  events [--follow] [--since 5m|RFC3339] [--filter substr]
                                    Stream the append-only event log.
                                    Recorded at ~/.local/share/opencode/goals.events.jsonl

Dashboard:
  today                             Show today's dashboard

Roadmaps:
  list-ls                           Show all roadmaps
  list-create <name>                Create a roadmap and switch to it
  list-use <id|name>                Switch active roadmap
  list-rename <id|name> <new-name>  Rename a roadmap
  list-delete <id|name>... [-y]     Delete one or more roadmaps and their contents
  list-show <id|name>               Show one roadmap's full tree

Inside OpenCode, prefer the slash commands:
  /today                            Dashboard
  /og   /ogl   /ogc <name>          Roadmaps: browse, list, create
  /ogs <name>   /ogd [name]         Roadmaps: switch, delete
  /og-main   /og-sub          Add main / sub-goals
  /og-list   /og-done         Roadmap goals, mark complete
  /og-summary   /og-remind    Daily summary, reminder
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
			die("Error: Usage: og add-sub <parent-id> <title>")
		}
		addSubGoal(strings.Join(args[1:], " "), args[0])

	case "done":
		requireArg(args, "a goal ID")
		b := parseBulkArgs(args, nil)
		if len(b.ids) == 0 {
			die("Error: done requires a goal ID.")
		}
		markDoneBulk(b.ids, b.yes)

	case "summary":
		generateSummary()

	case "remind":
		remindMe()

	case "task-list":
		listTasks()

	case "task-add":
		requireArg(args, "a task title")
		// Parse --depends <id,id,...> anywhere in args; remove from args.
		var deps []string
		var rest []string
		for i := 0; i < len(args); i++ {
			if args[i] == "--depends" || args[i] == "-d" {
				if i+1 >= len(args) {
					die("Error: --depends requires a comma-separated list of task IDs.")
				}
				for _, id := range strings.Split(args[i+1], ",") {
					id = strings.TrimSpace(id)
					if id != "" {
						deps = append(deps, id)
					}
				}
				i++ // skip the value
				continue
			}
			rest = append(rest, args[i])
		}
		if len(rest) == 0 {
			die("Error: Please provide a task title.")
		}
		// Optional last arg is a priority.
		priorities := map[string]bool{
			PriorityHigh: true, PriorityMedium: true, PriorityLow: true,
		}
		lastArg := strings.ToLower(rest[len(rest)-1])
		var title, priority string
		if priorities[lastArg] && len(rest) > 1 {
			priority = lastArg
			title = strings.Join(rest[:len(rest)-1], " ")
		} else {
			title = strings.Join(rest, " ")
		}
		addTask(title, priority, deps)

	case "task-done":
		requireArg(args, "a task ID")
		markTaskDone(args[0])

	case "task-delete":
		b := parseBulkArgs(args, map[string]bool{"completed": true})
		if len(b.ids) == 0 && !b.all && b.priority == "" && b.filter == "" {
			die("Error: task-delete requires task ID(s) or one of --all, --priority, --completed.")
		}
		deleteTasksBulk(b)

	case "task-clear":
		clearCompletedTasks()

	case "task-claim":
		requireArg(args, "a task ID")
		claimTask(args[0])

	case "task-release":
		requireArg(args, "a task ID")
		releaseTask(args[0])

	case "task-next":
		autoClaim := false
		for _, a := range args {
			if a == "--claim" || a == "-c" {
				autoClaim = true
			}
		}
		nextTask(autoClaim)

	case "task-show":
		requireArg(args, "a task ID")
		showTask(args[0])

	case "events":
		follow := false
		var since time.Time
		filter := ""
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--follow", "-f":
				follow = true
			case "--since":
				if i+1 >= len(args) {
					die("Error: --since requires a timestamp (RFC3339 or duration like 5m, 1h).")
				}
				v := args[i+1]
				if d, derr := time.ParseDuration(v); derr == nil {
					since = time.Now().Add(-d)
				} else if ts, terr := time.Parse(time.RFC3339, v); terr == nil {
					since = ts
				} else {
					die("Error: --since must be RFC3339 timestamp or duration (e.g. 5m, 1h). Got %q.", v)
				}
				i++
			case "--filter":
				if i+1 >= len(args) {
					die("Error: --filter requires a substring.")
				}
				filter = args[i+1]
				i++
			default:
				die("Error: unknown flag for events: %s", args[i])
			}
		}
		showEvents(follow, since, filter)

	case "today":
		showToday()

	case "list-ls":
		listRoadmaps()

	case "list-create":
		requireArg(args, "a list name")
		listCreate(strings.Join(args, " "))

	case "list-use":
		requireArg(args, "a list id or name")
		listUse(strings.Join(args, " "))

	case "list-rename":
		if len(args) < 2 {
			die("Error: Usage: og list-rename <id|name> <new-name>")
		}
		listRename(args[0], strings.Join(args[1:], " "))

	case "list-delete":
		requireArg(args, "a list id or name")
		// If any flag is present, parse in bulk mode (positional = ids/names).
		// Otherwise preserve legacy behavior where a multi-word name is
		// joined back together (e.g. `og list-delete My Roadmap`).
		hasFlag := false
		for _, a := range args {
			if strings.HasPrefix(a, "-") {
				hasFlag = true
				break
			}
		}
		if hasFlag {
			b := parseBulkArgs(args, nil)
			if len(b.ids) == 0 {
				die("Error: list-delete requires at least one roadmap id or name.")
			}
			listDeleteBulk(b.ids, b.yes)
		} else {
			listDelete(strings.Join(args, " "))
		}

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
