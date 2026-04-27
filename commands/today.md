---
description: Show today's dashboard with goals, tasks, and focus
agent: general
subtask: false
---

!`~/.config/opencode/skills/goal-tracker/goals today`

Current context:
- Working in: !`pwd`
- Current branch: !`git branch --show-current 2>/dev/null || echo "N/A"`

Quick commands:
- /goals-main <title> - Add a new main goal
- /task-add <title> [priority] - Add a quick task
- /goals-done <id> - Mark goal complete
- /task-done <id> - Mark task complete
- /goals-summary - See end-of-day summary
