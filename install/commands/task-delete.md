---
description: Delete one or more tasks (multi-select with confirmation)
agent: general
subtask: false
argument-hint: [task-id...]
---

Arguments: `$ARGUMENTS`

Current tasks:
!`og task-list`

You are running `/task-delete` — the opengoal task deleter.

Steps:

1. If `$ARGUMENTS` is a non-empty list of task IDs (space-separated), skip to step 3 and treat them as the selection.

2. If `$ARGUMENTS` is empty:
   - Parse the tasks shown above. Each task line has an `ID: task-…` reference.
   - If there are zero tasks, reply "No tasks to delete." and stop.
   - Use the `question` tool with:
     - `header`: "Select tasks to delete"
     - `question`: "Which tasks do you want to delete? (check the ones to remove)"
     - `multiple`: true
     - `options`: one entry per task. Label: a short title (truncate to ~50 chars). Description: include priority, completed marker, and the task ID.
   - If the user picks nothing, reply "Cancelled." and stop.
   - Map each selected label back to its task ID.

3. Build a confirmation summary listing the chosen tasks (title + id), then use the `question` tool:
   - `header`: "Confirm delete"
   - `question`: "Delete N task(s)? This cannot be undone."
   - options: "Yes, delete them" (Recommended), "No, cancel"
   - If user picks cancel, reply "Cancelled." and stop.

4. Run `og task-delete -y <id1> <id2> …` in a single invocation and show the output as-is.

Rules:
- Never delete without explicit confirmation.
- Always pass `-y` once the user confirms (so the CLI doesn't re-prompt).
- If anything is ambiguous, reply "Cancelled."
