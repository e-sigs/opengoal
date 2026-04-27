# OpenCode Goal Tracker 🎯

A lightning-fast goal and task tracking system for [OpenCode](https://opencode.ai) with persistent memory, progress tracking, and a comprehensive dashboard.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?logo=go)](https://go.dev)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)]()

## ⚡ Performance

- **12ms** average response time
- **14x faster** than JavaScript alternatives
- Single compiled binary, no runtime needed
- Only **~2.5MB** compressed

## ✨ Features

- 🎯 **Goals System** - Main goals with sub-goals and automatic progress tracking
- ✅ **Task List** - Quick todos with priority levels (high/medium/low)
- 📊 **Today Dashboard** - See everything at a glance with `/today`
- 💾 **Persistent Memory** - All data survives across sessions
- 🔄 **Auto-completion** - Main goals auto-complete when all sub-goals are done
- 📈 **Progress Tracking** - Automatic percentage calculations
- 📅 **Daily Summaries** - Review your accomplishments
- 🎨 **Beautiful Output** - Color-coded, emoji-rich interface

## 📸 Screenshot

```
╔════════════════════════════════════════════════╗
║  📅 TODAY - Thursday, April 23, 2026
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

✅ COMPLETED TODAY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Goals:
  ✓ Research JWT implementation

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

## 🚀 Quick Install

### One-Line Install

```bash
curl -fsSL https://gitlab.com/sig/opengoal/-/raw/main/install.sh | bash
```

### Manual Install

1. Download the binary for your platform from [Releases](https://gitlab.com/sig/opengoal/-/releases)
2. Extract and run the install script:

```bash
chmod +x install.sh
./install.sh
```

### Build from Source

```bash
git clone https://gitlab.com/sig/opengoal.git
cd opengoal
go build -o goals main.go
./install.sh
```

## 📚 Usage

### Commands

| Command | Description |
|---------|-------------|
| `/today` | Show complete dashboard |
| `/goals-main <title>` | Add a main goal |
| `/goals-sub <parent-id> <title>` | Add a sub-goal |
| `/goals-list` | List all goals with progress |
| `/goals-done <id>` | Mark goal complete |
| `/goals-summary` | Generate daily summary |
| `/goals-remind` | Show reminders |
| `/task-add <title> [priority]` | Add task (priority: high/medium/low) |
| `/task-list` | List all tasks |
| `/task-done <id>` | Mark task complete |
| `/task-delete <id>` | Delete a task |
| `/task-clear` | Clear completed tasks |

### Example Workflow

```bash
# Morning - Check your day
/today

# Add a main goal
/goals-main Implement API authentication system

# Agent will suggest sub-goals, add them:
/goals-sub mg-xxx Research JWT best practices
/goals-sub mg-xxx Create middleware
/goals-sub mg-xxx Write tests

# Add quick tasks
/task-add Review pull requests high
/task-add Update documentation medium

# During the day - mark things complete
/task-done task-xxx
/goals-done sg-xxx

# Evening - review your progress
/goals-summary
```

## 🏗️ Architecture

### File Structure

```
~/.config/opencode/
├── skills/goal-tracker/
│   ├── goals              # Compiled binary
│   ├── main.go            # Source code
│   ├── go.mod             # Go module
│   └── SKILL.md           # OpenCode skill definition
│
└── commands/
    ├── goals-*.md         # Goal commands
    ├── task-*.md          # Task commands
    └── today.md           # Dashboard command

~/.local/share/opencode/
└── goals.json             # Persistent data store
```

### Data Format

Goals are stored in `~/.local/share/opencode/goals.json`:

```json
{
  "main_goals": [{
    "id": "mg-xxx",
    "title": "Implement authentication",
    "status": "in_progress",
    "progress": 60,
    "sub_goals": ["sg-xxx"]
  }],
  "sub_goals": [{
    "id": "sg-xxx",
    "title": "Research JWT",
    "parent_id": "mg-xxx",
    "status": "completed"
  }],
  "tasks": [{
    "id": "task-xxx",
    "title": "Review PR",
    "priority": "high",
    "completed": false
  }]
}
```

## 🔧 Development

### Prerequisites

- Go 1.20 or higher
- OpenCode installed

### Building

```bash
# Build for your platform
go build -o goals main.go

# Build for all platforms
./build.sh
```

### Testing

```bash
# Test the binary
./goals today
./goals list
./goals task-list
```

### Project Structure

```
.
├── main.go           # Main application
├── go.mod            # Go module file
├── install.sh        # Installation script
├── build.sh          # Cross-platform build script
├── commands/         # OpenCode command definitions
│   ├── goals-*.md
│   ├── task-*.md
│   └── today.md
├── SKILL.md          # OpenCode skill definition
└── docs/             # Documentation
    ├── README_GO.md
    ├── GETTING_STARTED.md
    └── QUICK_REFERENCE.md
```

## 📦 Release Process

1. Update version in scripts
2. Run build script: `./build.sh`
3. Test binaries on each platform
4. Create GitHub release
5. Upload binaries from `dist/` folder
6. Update install script URL

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a merge request

## 📝 License

MIT License - see [LICENSE](LICENSE) file for details

## 🙏 Acknowledgments

- Built for [OpenCode](https://opencode.ai)
- Inspired by the need for fast, persistent goal tracking
- Written in Go for maximum performance

## 🐛 Issues & Support

- **Issues**: [GitLab Issues](https://gitlab.com/sig/opengoal/-/issues)
- **Merge Requests**: [GitLab MRs](https://gitlab.com/sig/opengoal/-/merge_requests)

## 🗺️ Roadmap

- [ ] Weekly/monthly reports
- [ ] Goal dependencies graph
- [ ] Time tracking per goal
- [ ] Export to Markdown/CSV
- [ ] Goal templates
- [ ] Tag system for filtering
- [ ] Built-in encryption for sensitive goals
- [ ] Sync across devices (optional cloud backend)

## ⭐ Show Your Support

If you find this tool useful, please consider:
- Giving it a star on GitLab ⭐
- Sharing it with others
- Contributing to the project
- Reporting bugs or suggesting features

---

**Made with ❤️ for the OpenCode community**
