---
description: Mark one or more goals as complete (multi-select with confirmation)
agent: general
subtask: false
argument-hint: [goal-id...]
---

Arguments: `$ARGUMENTS`

Current goals:
!`og list`

You are running `/og-done` — mark opengoal goals as complete.

Steps:

1. If `$ARGUMENTS` is a non-empty list of goal IDs, skip to step 3 and treat them as the selection.

2. If `$ARGUMENTS` is empty:
   - Parse the goals shown above. IDs look like `mg-…` (main) or `sg-…` (sub).
   - Skip any goals already marked completed.
   - If there are zero incomplete goals, reply "Nothing to mark — all goals already complete." and stop.
   - Use the `question` tool with:
     - `header`: "Select goals to complete"
     - `question`: "Which goals do you want to mark as complete?"
     - `multiple`: true
     - `options`: one entry per incomplete goal. Label: a short title. Description: include kind (main/sub), progress %, and the ID.
   - If the user picks nothing, reply "Cancelled." and stop.

3. Build a confirmation summary listing the chosen goals, then use the `question` tool:
   - `header`: "Confirm complete"
   - `question`: "Mark N goal(s) as complete?"
   - options: "Yes, mark complete" (Recommended), "No, cancel"
   - If user picks cancel, reply "Cancelled." and stop.

4. Run `og done -y <id1> <id2> …` in a single invocation and show the output as-is.

5. Show updated status:
   !`og list`

Rules:
- Never mutate without explicit confirmation when called with no arguments.
- Always pass `-y` once the user confirms (so the CLI doesn't re-prompt).
- If anything is ambiguous, reply "Cancelled."
