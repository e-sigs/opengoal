---
description: Interactive opengoal list browser — pick a list and view/manage it
agent: general
subtask: false
---

All lists:
!`~/.config/opencode/skills/goal-tracker/goals list-ls`

You are running `/og` — the opengoal list picker.

Steps:

1. Parse the lists shown above. For each list, capture its name, ID, and counts (goals / sub-goals / tasks pending).

2. **If there are zero lists**: do not call `question` for picking. Instead, call the `question` tool once with header "Create first list", question "You have no lists yet. Create one?", options:
   - "Yes — name it" (Recommended) — then call `question` again with header "List name" and a single option labeled "Type a name" (the user will use the custom-answer field). Then run `~/.config/opencode/skills/goal-tracker/goals list-create <name>` and stop.
   - "No, cancel" — reply "Cancelled." and stop.

3. **If at least one list exists**: use the `question` tool once with:
   - `header`: "Pick a list"
   - `question`: "Which opengoal list do you want to view?"
   - `multiple`: false
   - `options`: one entry per list. Label format: `{name} — {goals}g/{subs}s/{pending}t pending`. Mark the currently active list (the one with `▶`) by appending " (active)" to its label. Description: include the list ID. Do not add any catch-all option.

4. After the user picks, run:
   `~/.config/opencode/skills/goal-tracker/goals list-show <name>`
   using the chosen list's name, and show the output as-is.

5. Then ask one follow-up `question` with `header` "Action", options:
   - "Switch active list to this one" — runs `goals list-use <name>` (Recommended if the picked list is not already active)
   - "Rename this list" — ask for the new name with another `question` (rely on the default custom answer field), then run `goals list-rename <name> <new-name>`
   - "Delete this list" — runs `goals list-delete <name>` (warn that all goals/tasks in it will be removed)
   - "Done" — exit silently

6. Execute the chosen action and print its output. Keep your final reply to one short line confirming what happened.

Rules:
- Do NOT skip the `question` calls — that's the whole point of `/og`.
- If the user types something that doesn't map to a listed option, reply with one line: "Cancelled."
- Never modify lists without an explicit user pick.
- There is no auto-created default list. Users must always name their first one.
