---
description: Add a new main goal
agent: general
subtask: false
---

Current goals:
!`~/.config/opencode/skills/goal-tracker/goals list`

Add new main goal: $ARGUMENTS

Current work context:
- Working directory: !`pwd`
- Recent files: !`git status --short 2>/dev/null || echo "Not a git repository"`

Based on the goal title "$ARGUMENTS" and the current codebase context, please:

1. Add this as a main goal using the helper script
2. Analyze the codebase and suggest 3-5 relevant sub-goals that would help achieve this main goal
3. Consider the current project structure and files when suggesting sub-goals
4. Format suggestions clearly so I can easily add them with /goals-sub

After analyzing, add the main goal by running:
!`~/.config/opencode/skills/goal-tracker/goals add-main $ARGUMENTS`
