---
description: Delete one or more opengoal roadmaps (multi-select with confirmation)
agent: general
subtask: false
argument-hint: [roadmap-name...]
---

Arguments: `$ARGUMENTS`

All roadmaps:
!`og list-all`

You are running `/ogd` — the opengoal roadmap deleter.

Steps:

1. If `$ARGUMENTS` is a non-empty list of roadmap names or IDs, skip to step 3 and treat them as the selection.

2. If `$ARGUMENTS` is empty:
   - Parse the roadmaps shown above.
   - If there are zero roadmaps, reply "No roadmaps to delete." and stop.
   - Use the `question` tool with:
     - `header`: "Select roadmaps to delete"
     - `question`: "Which roadmaps do you want to delete? Each removal includes ALL its goals and tasks."
     - `multiple`: true
     - `options`: one entry per roadmap. Label: `{name}` (suffix " (active)" for the active one). Description: include ID and goal/task counts.
   - If the user picks nothing, reply "Cancelled." and stop.

3. Build a confirmation summary listing each selected roadmap and its counts, then use the `question` tool:
   - `header`: "Confirm delete"
   - `question`: "Delete N roadmap(s) and ALL their contents? This cannot be undone."
   - options: "Yes, delete them" (Recommended), "No, cancel"
   - If user picks cancel, reply "Cancelled." and stop.

4. Run `og list-delete -y <name1> <name2> …` in a single invocation and show the output as-is.

Rules:
- Never delete without explicit confirmation.
- Always pass `-y` once the user confirms (so the CLI doesn't re-prompt).
- If a selected roadmap name contains spaces, prefer its ID instead.
- If anything is ambiguous, reply "Cancelled."
