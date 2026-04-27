# Goal Tracker Quick Reference

## Commands

| Command | Description | Example |
|---------|-------------|---------|
| `/goals-main <title>` | Add new main goal | `/goals-main Implement authentication` |
| `/goals-sub <id> <title>` | Add sub-goal | `/goals-sub mg-xxx Add JWT tokens` |
| `/goals-list` | Show all goals | `/goals-list` |
| `/goals-done <id>` | Mark complete | `/goals-done sg-xxx` |
| `/goals-summary` | Daily report | `/goals-summary` |
| `/goals-remind` | Current focus | `/goals-remind` |

## Daily Workflow

**Morning:**
1. `/goals-remind` - See what's pending
2. `/goals-main` - Add today's main goal
3. Add sub-goals as suggested

**During Work:**
1. `/goals-list` - Check progress
2. `/goals-done <id>` - Mark tasks complete

**Evening:**
1. `/goals-summary` - Review the day
2. Plan tomorrow's focus

## Goal IDs

- Main goals: `mg-xxx`
- Sub-goals: `sg-xxx`

Get IDs from `/goals-list` output.

## Progress Tracking

- Progress auto-calculated from sub-goals
- Main goal auto-completes when all sub-goals done
- Status: `pending` → `in_progress` → `completed`

## Files

- **Config:** `~/.config/opencode/skills/goal-tracker/`
- **Memory:** `~/.local/share/opencode/goals.json`
- **Backup:** `~/.local/share/opencode/goals.json.backup`

## Tips

✅ Use clear, actionable goal titles  
✅ Create 3-5 sub-goals per main goal  
✅ Check `/goals-remind` when starting work  
✅ Run `/goals-summary` at end of day  
✅ Mark sub-goals complete as you finish them  

## Example Session

```bash
# Morning
/goals-remind
/goals-main Refactor authentication module

# Agent suggests sub-goals:
# - Review current auth code
# - Extract JWT logic to separate service
# - Add unit tests
# - Update documentation

/goals-sub mg-xxx Review current auth code
/goals-sub mg-xxx Extract JWT logic to separate service

# During work
/goals-list                    # Check progress
/goals-done sg-xxx             # Mark first sub-goal done

# Evening
/goals-summary                 # See what you accomplished
```

## Need Help?

See full documentation: `~/.config/opencode/skills/goal-tracker/README.md`
