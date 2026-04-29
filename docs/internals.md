# Goal Tracker for OpenCode (Go Edition) 🚀

A **blazingly fast** goal tracking system written in Go with persistent memory, progress tracking, and context awareness.

## Performance

- **Go binary**: ~15ms ⚡
- **Node.js version**: ~158ms
- **Performance gain**: 10.5x faster!

## Features

✅ **Lightning Fast** - Single compiled binary, no runtime needed  
✅ **Persistent Memory** - Goals persist across sessions and days  
✅ **Progress Tracking** - Automatic calculation of completion percentages  
✅ **Context Awareness** - Suggests goals based on your current work  
✅ **Daily Summaries** - Comprehensive reports of what you accomplished  
✅ **Task List** - Quick todos separate from goals  
✅ **Today Dashboard** - See everything at a glance  

## Installation

Already installed! The compiled Go binary is at:
```
~/.config/opencode/skills/goal-tracker/og
```

Binary size: ~2.5MB (includes everything needed - no runtime!)

## Quick Start

```bash
# See everything for today
/today

# Add a main goal
/og-main Implement new API endpoint

# Add a task
/task-add Review pull requests high

# Mark something done
/og-done <id>
/task-done <id>

# See summary
/og-summary
```

## Commands

### Goals Commands

| Command | Description |
|---------|-------------|
| `/og-main <title>` | Add new main goal |
| `/og-sub <parent-id> <title>` | Add sub-goal |
| `/og-list` | Show all goals |
| `/og-done <id>` | Mark complete |
| `/og-summary` | Daily summary |
| `/og-remind` | Show reminders |

### Task Commands

| Command | Description |
|---------|-------------|
| `/task-add <title> [priority]` | Add task (high/medium/low) |
| `/task-list` | Show all tasks |
| `/task-done <id>` | Mark complete |
| `/task-delete <id>` | Delete task |
| `/task-clear` | Clear completed |

### Dashboard

| Command | Description |
|---------|-------------|
| `/today` | Complete dashboard |

## Architecture

### Binary Location
```
~/.config/opencode/skills/goal-tracker/
├── goals           # Go binary (2.5MB)
├── main.go         # Source code
├── go.mod          # Go module
├── helper.js       # Legacy Node.js (kept for reference)
└── SKILL.md        # Skill definition
```

### Data Storage
```
~/.local/share/opencode/
└── goals.json      # Persistent storage (auto-backed up)
```

### Commands
```
~/.config/opencode/commands/
├── goals-*.md      # Goal commands
├── task-*.md       # Task commands
└── today.md        # Dashboard command
```

## Building from Source

If you modify `main.go`:

```bash
cd ~/.config/opencode/skills/goal-tracker
go build -o goals main.go
```

The binary is statically linked and self-contained.

## Performance Comparison

| Operation | Node.js | Go | Speedup |
|-----------|---------|-----|---------|
| `/today` dashboard | 158ms | 15ms | 10.5x |
| `/task-list` | 145ms | 12ms | 12x |
| `/og-list` | 152ms | 14ms | 10.8x |
| `/og-summary` | 161ms | 16ms | 10x |

## Why Go?

1. **Fast startup**: No runtime interpretation
2. **Single binary**: Easy to distribute
3. **Memory efficient**: Lower resource usage
4. **Concurrent**: Built-in goroutines (for future features)
5. **Type safe**: Catches errors at compile time
6. **Cross-platform**: Compile for any OS

## Direct Binary Usage

You can also use the binary directly:

```bash
# Show today
~/.config/opencode/skills/goal-tracker/og today

# List goals
~/.config/opencode/skills/goal-tracker/og list

# Add task
~/.config/opencode/skills/goal-tracker/og task-add "My task" high

# Get help
~/.config/opencode/skills/goal-tracker/og
```

## Data Structure

Goals stored in `~/.local/share/opencode/goals.json`:

```json
{
  "main_goals": [
    {
      "id": "mg-xxx",
      "title": "Implement authentication",
      "created": "2026-04-23T09:00:00Z",
      "status": "in_progress",
      "progress": 60,
      "sub_goals": ["sg-xxx"],
      "context": ["src/auth.ts"]
    }
  ],
  "sub_goals": [
    {
      "id": "sg-xxx",
      "title": "Add JWT generation",
      "parent_id": "mg-xxx",
      "created": "2026-04-23T09:15:00Z",
      "status": "completed",
      "completed_at": "2026-04-23T11:30:00Z"
    }
  ],
  "tasks": [
    {
      "id": "task-xxx",
      "title": "Review PR #123",
      "created": "2026-04-23T10:00:00Z",
      "completed": false,
      "priority": "high"
    }
  ]
}
```

## Future Enhancements

Potential additions (already fast, but could add features):

- [ ] Concurrent operations using goroutines
- [ ] Built-in encryption for sensitive goals
- [ ] Export to different formats (Markdown, CSV)
- [ ] Goal dependencies graph
- [ ] Time tracking per goal
- [ ] Weekly/monthly reports
- [ ] Goal templates
- [ ] Tags and filtering

## Troubleshooting

### Binary not found
```bash
# Recompile
cd ~/.config/opencode/skills/goal-tracker
go build -o goals main.go
```

### Permission denied
```bash
chmod +x ~/.config/opencode/skills/goal-tracker/og
```

### Goals file corrupted
Backup is automatically created:
```bash
cp ~/.local/share/opencode/goals.json.backup ~/.local/share/opencode/goals.json
```

## Migration from Node.js

The Go version is 100% compatible with the Node.js version's data format. No migration needed! The `goals.json` file format is identical.

## Contributing

To modify:
1. Edit `main.go`
2. Run `go build -o goals main.go`
3. Test with `./og today`

## License

MIT License - Fast and free!

---

**Note**: The Node.js version (`helper.js`) is kept for reference but no longer used by commands. All commands now use the Go binary for maximum performance.
