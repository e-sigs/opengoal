# OpenCode Goal Tracker 🎯

A lightning-fast goal and task tracking system for [OpenCode](https://opencode.ai) with multiple lists, persistent memory, progress tracking, and a unified dashboard.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?logo=go)](https://go.dev)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)]()

## ✨ Features

- 📂 **Multiple lists** — separate work, personal, and side-project items; switch with one command
- 🎯 **Goals & sub-goals** with automatic progress tracking and auto-completion
- ✅ **Tasks** with optional priorities (high / medium / low)
- 📊 **`/today` dashboard** — goals + tasks + focus + stats in one view
- 💾 **Atomic writes** — `goals.json` is replaced via temp-file rename, never corrupted
- 🚀 **Single static Go binary** (~2 MB), no runtime dependencies
- 🔒 **Local-only** — nothing ever leaves your machine

## 📸 Dashboard

```
╔════════════════════════════════════════════════╗
║  📅 TODAY - Tuesday, April 28, 2026
║  📂 List: work
╚════════════════════════════════════════════════╝

🎯 ACTIVE GOALS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Implement user authentication [60%]
  → Next: Create auth middleware

📝 TASKS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔴 HIGH PRIORITY:
  1. Review PR #123

🟡 MEDIUM PRIORITY:
  1. Update documentation

🔥 FOCUS NOW
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  1. Review PR #123 (high priority task)
  2. Create auth middleware (Implement user authentication)

📊 STATS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Active Goals: 1
  Pending Tasks: 2
  Completed Today: 1
```

## 📋 Requirements

- **[OpenCode](https://opencode.ai)** — opengoal is an OpenCode skill, not a standalone tool
- **OS**: macOS, Linux, or Windows (amd64 or arm64)
- **Building from source only**: Go 1.25+

No runtime dependencies — opengoal is a single static binary.

## 🚀 Install

### One-line install (latest release)

```bash
curl -fsSL https://raw.githubusercontent.com/e-sigs/opengoal/main/install.sh | bash
```

### From source

```bash
git clone https://github.com/e-sigs/opengoal.git
cd opengoal
./install.sh
```

The installer:
- Builds (or downloads) the `goals` binary
- Copies the skill to `~/.config/opencode/skills/goal-tracker/`
- Copies all slash commands to `~/.config/opencode/commands/`
- Initializes `~/.local/share/opencode/goals.json` if missing

After installing, restart OpenCode and run `/today`.

## 📚 Commands

### Lists

| Command | Description |
|---|---|
| `/og` | Interactive list browser (pick / switch / rename / delete) |
| `/ogl` | Show all lists |
| `/ogc <name>` | Create a new list and switch to it |
| `/ogs <name>` | Switch the active list |
| `/ogd [name]` | Delete a list (prompts for confirmation) |

### Goals

| Command | Description |
|---|---|
| `/goals-main <title>` | Add a main goal in the active list |
| `/goals-sub <parent-id> <title>` | Add a sub-goal under a main goal |
| `/goals-list` | List goals in the active list |
| `/goals-done <id>` | Mark a goal complete (main or sub) |
| `/goals-summary` | Daily summary for the active list |
| `/goals-remind` | Show what to focus on now |

### Tasks

| Command | Description |
|---|---|
| `/task-add <title> [priority]` | Add a task (priority: high/medium/low) |
| `/task-list` | List tasks in the active list |
| `/task-done <id>` | Mark a task complete |
| `/task-delete <id>` | Delete a task |
| `/task-clear` | Remove all completed tasks |

### Dashboard

| Command | Description |
|---|---|
| `/today` | Full dashboard for the active list |

## 🧭 Example Workflow

```bash
# First time
/ogc work                                # create + switch to "work" list
/goals-main Implement API authentication
/goals-sub mg-xxx Research JWT best practices
/goals-sub mg-xxx Create middleware
/goals-sub mg-xxx Write tests
/task-add Review pull requests high

# Switch contexts
/ogc personal                            # new list, now active
/goals-main Plan vacation
/ogs work                                # back to work

# During the day
/today
/task-done task-xxx
/goals-done sg-xxx

# End of day
/goals-summary
```

## 🏗️ Architecture

```
~/.config/opencode/
├── skills/goal-tracker/
│   ├── goals          # Compiled Go binary
│   ├── main.go        # Source
│   ├── go.mod
│   └── SKILL.md       # OpenCode skill definition
└── commands/
    ├── og*.md         # List browser commands
    ├── goals-*.md     # Goal commands
    ├── task-*.md      # Task commands
    └── today.md       # Dashboard

~/.local/share/opencode/
└── goals.json         # Persistent data store
```

### Data format (excerpt)

```json
{
  "lists": [
    { "id": "list-xxx", "name": "work", "created": "..." }
  ],
  "active_list_id": "list-xxx",
  "main_goals": [
    { "id": "mg-xxx", "list_id": "list-xxx", "title": "...",
      "status": "in_progress", "progress": 60, "sub_goals": ["sg-xxx"] }
  ],
  "sub_goals": [
    { "id": "sg-xxx", "list_id": "list-xxx", "parent_id": "mg-xxx",
      "title": "...", "status": "completed" }
  ],
  "tasks": [
    { "id": "task-xxx", "list_id": "list-xxx", "title": "...",
      "priority": "high", "completed": false }
  ]
}
```

Pre-existing files from v1.0 (no `lists`, no `list_id`) are migrated automatically the first time you create a list after upgrading.

## 🔧 Development

```bash
# Build for current platform
go build -o goals main.go

# Build all release binaries into ./dist
./build.sh

# Run a quick smoke test against a sandbox HOME
HOME=/tmp/goals-smoke ./goals today
```

## 📝 License

MIT — see [LICENSE](LICENSE).

## 🐛 Issues & Support

- **Issues**: https://github.com/e-sigs/opengoal/issues
- **Pull requests**: https://github.com/e-sigs/opengoal/pulls

## 🗺️ Roadmap

- [ ] Weekly / monthly reports
- [ ] Goal dependencies graph
- [ ] Time tracking per goal
- [ ] Export to Markdown / CSV
- [ ] Tags for filtering across lists

---

**Made with ❤️ for the OpenCode community**
