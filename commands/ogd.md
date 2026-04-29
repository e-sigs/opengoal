---
description: Delete an opengoal list (prompts for confirmation)
agent: general
subtask: false
argument-hint: [list-name]
---

Arguments: `$ARGUMENTS`

All lists:
!`~/.config/opencode/skills/goal-tracker/goals list-ls`

You are running `/ogd` — the opengoal list deleter.

Steps:

1. If `$ARGUMENTS` is non-empty, treat it as the list name to delete. Skip to step 3.

2. If `$ARGUMENTS` is empty:
   - Parse the lists shown above.
   - If there are zero lists, reply "No lists to delete." and stop.
   - Use the `question` tool with:
     - `header`: "Pick a list to delete"
     - `question`: "Which list do you want to delete?"
     - `multiple`: false
     - `options`: one entry per list. Label: `{name}`. Description: include ID and counts. Mark active with " (active)".
   - If the user picks nothing valid, reply "Cancelled." and stop.

3. Confirm with one `question` tool call:
   - `header`: "Confirm delete"
   - `question`: "Delete list '<name>'? All goals/sub-goals/tasks in it will be permanently removed."
   - options: "Yes, delete it", "No, cancel" (Recommended)
   - If user picks cancel or anything else, reply "Cancelled." and stop.

4. Run `~/.config/opencode/skills/goal-tracker/goals list-delete <name>` and show the output as-is.

Rules:
- Never delete without explicit confirmation.
- If the user types something that doesn't map, reply "Cancelled."
