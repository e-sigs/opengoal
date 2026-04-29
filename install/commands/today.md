---
description: Show today's dashboard with goals, tasks, and focus
agent: general
subtask: false
---

!`og today`

Current context:
- Working in: !`pwd`
- Current branch: !`git branch --show-current 2>/dev/null || echo "N/A"`

Quick commands:
- /og-main <title> - Add a new main goal
- /task-add <title> [priority] - Add a quick task
- /og-done <id> - Mark goal complete
- /task-done <id> - Mark task complete
- /og-summary - See end-of-day summary
