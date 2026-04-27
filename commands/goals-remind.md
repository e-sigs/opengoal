---
description: Show current focus and reminders
agent: general
subtask: false
---

!`~/.config/opencode/skills/goal-tracker/goals remind`

Current context:
- Working in: !`pwd`
- Current branch: !`git branch --show-current 2>/dev/null || echo "N/A"`
- Modified files: !`git status --short 2>/dev/null | head -5 || echo "N/A"`

Based on your current work context, do any of these files relate to your active goals?
If so, let me know and I can help you make progress!
