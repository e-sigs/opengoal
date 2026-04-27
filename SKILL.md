---
name: goal-tracker
description: Track daily main goals and sub-goals with persistent memory, progress tracking, and context awareness
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: productivity
---

## What I Do

I am your personal goal tracking assistant with persistent memory. I help you:

- **Track main goals and sub-goals** across multiple days until completion
- **Calculate progress** automatically based on completed sub-goals
- **Provide context-aware suggestions** based on your current work
- **Generate daily summaries** of what you accomplished
- **Remind you proactively** of pending goals and next steps
- **Maintain goal history** for future reference

## Persistent Memory

All goals are stored in `~/.local/share/opencode/goals.json` and persist across:
- Multiple OpenCode sessions
- Different days
- Different projects

Goals remain active until you explicitly mark them complete.

## When to Use Me

Use this skill when you need to:
- Start your workday and review pending goals
- Add a new main goal or break it into sub-goals
- Check progress on current objectives
- Mark goals as complete
- Generate end-of-day summaries
- Get reminded of what to focus on next

## How I Work

### Memory Structure

I maintain three types of data:

1. **Main Goals**: High-level objectives that span multiple days
   - Auto-calculated progress based on sub-goals
   - Can have associated context (files, directories)
   - Status: pending, in_progress, completed

2. **Sub Goals**: Actionable tasks that contribute to main goals
   - Linked to a parent main goal
   - Tracked individually
   - Completion updates parent progress

3. **Daily Summaries**: Historical record of your productivity
   - What you completed each day
   - What you added
   - What's still in progress

### Context Awareness

I analyze your current work environment:
- Current working directory
- Git repository status
- Recently modified files
- Open files in your editor

This helps me:
- Suggest relevant sub-goals
- Remind you of related goals when you're working on related code
- Link goals to specific parts of your codebase

### Progress Tracking

Main goal progress is automatically calculated:
```
Progress = (Completed Sub-Goals / Total Sub-Goals) × 100%
```

When all sub-goals are complete, the main goal is auto-completed.

## Commands Available

The following custom commands work with me:

- `/goals-main <title>` - Add a new main goal
- `/goals-sub <parent-id> <title>` - Add a sub-goal
- `/goals-list` - Show all current goals with progress
- `/goals-done <goal-id>` - Mark a goal as complete
- `/goals-summary` - Generate today's summary
- `/goals-remind` - Show what to focus on now

## Smart Suggestions

When you add a main goal, I can:
- Analyze your codebase to suggest relevant sub-goals
- Identify related files and directories
- Detect dependencies between goals
- Estimate effort based on similar past goals

## Proactive Reminders

I can remind you of your goals:
- When you start a new OpenCode session
- When you're editing files related to a goal
- At regular intervals during your workday
- When you haven't made progress in a while

## Example Workflow

**Morning:**
```
You: /goals-remind
Me: Shows your 3 active main goals and suggests what to focus on first
```

**Adding a goal:**
```
You: /goals-main Implement authentication for API endpoints
Me: Creates main goal, suggests sub-goals like:
    - Research authentication strategies
    - Implement JWT token generation
    - Add middleware for protected routes
    - Write authentication tests
```

**Working on code:**
```
Context: You're editing src/api/auth.ts
Me: [Proactive] You're working on "Implement authentication" (60% complete)
    Next sub-goal: "Add middleware for protected routes"
```

**Marking complete:**
```
You: /goals-done sg-xxx
Me: Marked complete! Main goal "Implement authentication" is now 80% done.
```

**End of day:**
```
You: /goals-summary
Me: Shows comprehensive summary:
    - 4 goals completed today
    - 2 goals in progress
    - Tomorrow's recommended focus
```

## Integration Tips

Add this to your project's `AGENTS.md` for automatic context:

```markdown
## Daily Goals

Check `/goals-remind` at the start of each session to see active goals.
When making progress, use `/goals-done <id>` to track completion.
```

## Error Handling

If something goes wrong:
- Goals file is automatically backed up before changes
- Invalid IDs are caught with helpful error messages
- Corrupted JSON is restored from backup
- All operations are atomic (all-or-nothing)

## Privacy Note

All goal data is stored locally on your machine. Nothing is sent to external servers.
