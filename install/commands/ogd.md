---
description: Delete an opengoal roadmap (prompts for confirmation)
agent: general
subtask: false
argument-hint: [roadmap-name]
---

Arguments: `$ARGUMENTS`

All roadmaps:
!`og list-ls`

You are running `/ogd` — the opengoal roadmap deleter.

Steps:

1. If `$ARGUMENTS` is non-empty, treat it as the roadmap name to delete. Skip to step 3.

2. If `$ARGUMENTS` is empty:
   - Parse the roadmaps shown above.
   - If there are zero roadmaps, reply "No roadmaps to delete." and stop.
   - Use the `question` tool with:
     - `header`: "Pick a roadmap to delete"
     - `question`: "Which roadmap do you want to delete?"
     - `multiple`: false
     - `options`: one entry per roadmap. Label: `{name}`. Description: include ID and counts. Mark active with " (active)".
   - If the user picks nothing valid, reply "Cancelled." and stop.

3. Confirm with one `question` tool call:
   - `header`: "Confirm delete"
   - `question`: "Delete roadmap '<name>'? All goals/sub-goals/tasks in it will be permanently removed."
   - options: "Yes, delete it", "No, cancel" (Recommended)
   - If user picks cancel or anything else, reply "Cancelled." and stop.

4. Run `og list-delete <name>` and show the output as-is.

Rules:
- Never delete without explicit confirmation.
- If the user types something that doesn't map, reply "Cancelled."
