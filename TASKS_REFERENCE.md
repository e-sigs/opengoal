# Task List Quick Reference

## What's the Difference?

**Goals** (Main + Sub-goals):
- For bigger objectives that take multiple days
- Track progress and completion percentages
- Organized hierarchically
- Example: "Implement authentication system"

**Tasks** (Simple todo list):
- Quick, standalone items
- No hierarchy or sub-items
- Simple check-off list
- Example: "Buy groceries", "Review PRs"

## Task Commands

| Command | Description | Example |
|---------|-------------|---------|
| `/task-add <title> [priority]` | Add new task | `/task-add Review code high` |
| `/task-list` | Show all tasks | `/task-list` |
| `/task-done <id>` | Mark complete | `/task-done task-xxx` |
| `/task-delete <id>` | Delete task | `/task-delete task-xxx` |
| `/task-clear` | Clear completed | `/task-clear` |

## Priority Levels

- `high` - Urgent, do first
- `medium` - Important, do soon
- `low` - Can wait
- (none) - No priority set

## Usage Examples

```bash
# Add a simple task
/task-add Buy groceries

# Add with priority
/task-add Review pull requests high
/task-add Schedule team meeting medium

# View all tasks
/task-list

# Mark done
/task-done task-xxx

# Clean up completed tasks
/task-clear
```

## When to Use Tasks vs Goals

**Use Tasks for:**
- Quick reminders
- Shopping lists
- Email/message responses
- Small code reviews
- Meeting scheduling
- Bug fixes (single file)

**Use Goals for:**
- Feature implementations
- Refactoring projects
- Learning new technologies
- Multi-day projects
- Anything with sub-steps

## Example Workflow

```bash
# Morning
/task-list                              # Check your tasks
/task-add Respond to client email high  # Add urgent items
/task-add Update dependencies           # Add routine tasks

# During day
/task-done task-xxx                     # Check off as you go

# End of day
/task-clear                             # Clean up completed tasks
```

## Tips

✅ Keep tasks short and actionable  
✅ Use priority for urgent items  
✅ Clear completed tasks regularly  
✅ Use goals for multi-step work  
✅ Tasks are independent - no sub-items  

## Storage

Tasks are stored alongside goals in:
```
~/.local/share/opencode/goals.json
```

Under the `tasks` array.
