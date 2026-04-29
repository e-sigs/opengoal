---
name: goal-tracker
description: Track daily main goals, sub-goals and tasks across multiple named lists with persistent memory and progress tracking
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: productivity
---

## What I Do

I am your personal goal tracking assistant with persistent memory. I help you:

- **Organize work into named lists** (e.g. work, personal, side-project) and switch between them
- **Track main goals and sub-goals** across multiple days until completion
- **Manage tasks** with optional priorities (high / medium / low)
- **Calculate progress** automatically based on completed sub-goals
- **Generate daily summaries** of what you accomplished
- **Show a unified `today` dashboard** combining goals, tasks, focus and stats

## Persistent Memory

All data is stored locally in `~/.local/share/opencode/goals.json` and persists across:

- Multiple OpenCode sessions
- Different days
- Different projects

Goals remain active until you explicitly mark them complete. Writes are atomic (write to a temp file then rename), so the data file is never left in a partial state.

## How I Work

### Memory Structure

I maintain four collections in a single JSON file:

1. **Lists** — top-level containers. Each list has a unique id and a human-readable name. Exactly one list is "active" at a time; all goal/task commands operate on the active list.
2. **Main Goals** — high-level objectives that belong to a list. Auto-calculated progress based on sub-goals. Status: `pending`, `in_progress`, `completed`.
3. **Sub-Goals** — actionable steps under a main goal. Completing all sub-goals auto-completes the parent.
4. **Tasks** — quick standalone todos with optional priority (`high` / `medium` / `low`).

### Progress Tracking

```
Progress = (Completed Sub-Goals / Total Sub-Goals) × 100%
```

When all sub-goals are complete, the main goal is auto-completed.

## Commands Available

### Lists
- `/og` — interactive list browser (pick / view / switch / rename / delete)
- `/ogl` — show all lists
- `/ogc <name>` — create a new list and switch to it
- `/ogs <name>` — switch the active list
- `/ogd [name]` — delete a list (with confirmation; prompts to pick if no name)

### Goals
- `/goals-main <title>` — add a main goal in the active list
- `/goals-sub <parent-id> <title>` — add a sub-goal under a main goal
- `/goals-list` — show goals in the active list
- `/goals-done <id>` — mark a goal (main or sub) as complete
- `/goals-summary` — daily summary for the active list
- `/goals-remind` — show focus reminder for the active list

### Tasks
- `/task-add <title> [priority]` — add a task (priority optional: high/medium/low)
- `/task-list` — show tasks in the active list
- `/task-done <id>` — mark a task complete
- `/task-delete <id>` — delete a task
- `/task-clear` — remove all completed tasks

### Dashboard
- `/today` — full dashboard for the active list (goals + tasks + focus + stats)

## Underlying CLI

All slash commands shell out to the same Go binary at
`~/.config/opencode/skills/goal-tracker/goals`. Run `goals help` for the
full subcommand list. Notable subcommands map 1:1 with the slash commands:

```
goals list                            # /goals-list
goals add-main <title>                # /goals-main
goals add-sub <parent-id> <title>     # /goals-sub
goals done <id>                       # /goals-done
goals summary | remind | today
goals task-list | task-add | task-done | task-delete | task-clear
goals list-ls | list-create <name> | list-use <id|name>
goals list-rename <id|name> <new-name> | list-delete <id|name> | list-show <id|name>
```

## Example Workflow

**First-time setup:**
```
You: /ogc work
Me:  ✅ Created list: "work" (now active)
You: /goals-main Ship the v1.2 release
You: /task-add Review PR high
```

**Morning check-in:**
```
You: /today
Me:  shows active goals, prioritized tasks, what was completed yesterday,
     and the top 3 things to focus on now
```

**Switching contexts:**
```
You: /ogc personal      # creates and switches to "personal"
You: /goals-main Plan vacation
You: /ogs work          # back to work list
```

**End of day:**
```
You: /goals-summary
Me:  completed today, in progress, added today, next focus
```

## Error Handling

- Writes are atomic: temp file + rename. If anything fails, `goals.json` is unchanged.
- Pre-existing data without a `list_id` is migrated to the active list once and persisted.
- Invalid IDs and missing lists fail with clear error messages, never silent corruption.

## Privacy

All goal data is stored locally on your machine. Nothing is sent to external servers.
