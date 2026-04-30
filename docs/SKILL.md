---
name: goal-tracker
description: Track daily main goals, sub-goals and tasks across multiple named roadmaps with persistent memory and progress tracking
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: productivity
---

## What I Do

I am your personal goal tracking assistant with persistent memory. I help you:

- **Organize work into named roadmaps** (e.g. work, personal, side-project) and switch between them
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

1. **Roadmaps** — top-level containers. Each roadmap has a unique id and a human-readable name. Exactly one roadmap is "active" at a time; all goal/task commands operate on the active roadmap.
2. **Main Goals** — high-level objectives that belong to a roadmap. Auto-calculated progress based on sub-goals. Status: `pending`, `in_progress`, `completed`.
3. **Sub-Goals** — actionable steps under a main goal. Completing all sub-goals auto-completes the parent.
4. **Tasks** — quick standalone todos with optional priority (`high` / `medium` / `low`).

### Progress Tracking

```
Progress = (Completed Sub-Goals / Total Sub-Goals) × 100%
```

When all sub-goals are complete, the main goal is auto-completed.

## Commands Available

### Roadmaps
- `/og` — interactive roadmap browser (pick / view / switch / rename / delete)
- `/ogl` — show all roadmaps
- `/ogc <name>` — create a new roadmap and switch to it
- `/ogs <name>` — switch the active roadmap
- `/ogd [name]` — delete a roadmap (with confirmation; prompts to pick if no name)

### Goals
- `/og-main <title>` — add a main goal in the active roadmap
- `/og-sub <parent-id> <title>` — add a sub-goal under a main goal
- `/og-list` — show goals in the active roadmap
- `/og-done <id>` — mark a goal (main or sub) as complete
- `/og-summary` — daily summary for the active roadmap
- `/og-remind` — show focus reminder for the active roadmap

### Tasks
- `/task-add <title> [priority]` — add a task (priority optional: high/medium/low)
- `/task-list` — show tasks in the active roadmap
- `/task-done <id>` — mark a task complete
- `/task-delete <id>` — delete a task

### Dashboard
- `/today` — full dashboard for the active roadmap (goals + tasks + focus + stats)

## Underlying CLI

All slash commands shell out to the same Go binary at
`~/.config/opencode/skills/goal-tracker/og`. Run `og help` for the
full subcommand list. Notable subcommands map 1:1 with the slash commands:

```
og list                            # /og-list
og add-main <title>                # /og-main
og add-sub <parent-id> <title>     # /og-sub
og done <id>                       # /og-done
og summary | remind | today
og task-list | task-add | task-done | task-delete
og list-all | list-create <name> | list-use <id|name>
og list-rename <id|name> <new-name> | list-delete <id|name> | list-show <id|name>
```

## Example Workflow

**First-time setup:**
```
You: /ogc work
Me:  ✅ Created roadmap: "work" (now active)
You: /og-main Ship the v1.2 release
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
You: /og-main Plan vacation
You: /ogs work          # back to work roadmap
```

**End of day:**
```
You: /og-summary
Me:  completed today, in progress, added today, next focus
```

## Error Handling

- Writes are atomic: temp file + rename. If anything fails, `goals.json` is unchanged.
- Pre-existing data without a `list_id` is migrated to the active roadmap once and persisted.
- Invalid IDs and missing roadmaps fail with clear error messages, never silent corruption.

## Privacy

All goal data is stored locally on your machine. Nothing is sent to external servers.
