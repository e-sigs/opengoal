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

// Roadmap is a top-level container for goals and tasks. The on-disk JSON
// keys (`lists`, `active_list_id`, `list_id`) are kept from the v1
// "lists" naming so existing data files keep working.
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

// GoalsData is the top-level shape persisted to goals.json. Legacy JSON
// keys (`lists`, `active_list_id`) are intentional — see Roadmap.
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
// Misc helpers
// ──────────────────────────────────────────────────────────────────

func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d-%05d", prefix, time.Now().UnixMilli(), rand.Intn(99999))
}

func printSeparator() {
	fmt.Println(strings.Repeat("━", separatorWidth))
}

func today() string {
	return time.Now().Format(dateFmtISO)
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
  list-all                          Show all roadmaps with their goals (condensed)
  list-create <name>                Create a roadmap and switch to it
  list-use <id|name>                Switch active roadmap
  list-rename <id|name> <new-name>  Rename a roadmap
  list-delete <id|name>... [-y]     Delete one or more roadmaps and their contents
  list-show <id|name>               Show one roadmap's full tree

Inside OpenCode, prefer the slash commands:
  /today                            Dashboard
  /og                               Show active roadmap
  /ogl                              List all roadmaps + goals
  /ogc <name>                       Create roadmap
  /ogs <name>   /ogd [name]         Roadmaps: switch, delete
  /og-main   /og-sub          Add main / sub-goals
  /og-list   /og-done         Roadmap goals, mark complete
  /og-summary   /og-remind    Daily summary, reminder
  /task-add   /task-list            Tasks: add, list
  /task-done   /task-delete         Tasks: complete, delete
  /og-commands                      Condensed reference of all commands
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

	case "list-all":
		listAll()

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
