---
description: Interactive opengoal roadmap browser — pick a roadmap and view/manage it
agent: general
subtask: false
---

All roadmaps:
!`og list-ls`

You are running `/og` — the opengoal roadmap picker.

Steps:

1. Parse the roadmaps shown above. For each roadmap, capture its name, ID, and counts (goals / sub-goals / tasks pending).

2. **If there are zero roadmaps**: do not call `question` for picking. Instead, call the `question` tool once with header "Create first roadmap", question "You have no roadmaps yet. Create one?", options:
   - "Yes — name it" (Recommended) — then call `question` again with header "Roadmap name" and a single option labeled "Type a name" (the user will use the custom-answer field). Then run `og list-create <name>` and stop.
   - "No, cancel" — reply "Cancelled." and stop.

3. **If at least one roadmap exists**: use the `question` tool once with:
   - `header`: "Pick a roadmap"
   - `question`: "Which opengoal roadmap do you want to view?"
   - `multiple`: false
   - `options`: one entry per roadmap. Label format: `{name} — {goals}g/{subs}s/{pending}t pending`. Mark the currently active roadmap (the one with `▶`) by appending " (active)" to its label. Description: include the roadmap ID. Do not add any catch-all option.

4. After the user picks, run:
   `og list-show <name>`
   using the chosen roadmap's name, and show the output as-is.

5. Then ask one follow-up `question` with `header` "Action", options:
   - "Switch active roadmap to this one" — runs `og list-use <name>` (Recommended if the picked roadmap is not already active)
   - "Rename this roadmap" — ask for the new name with another `question` (rely on the default custom answer field), then run `og list-rename <name> <new-name>`
   - "Delete this roadmap" — runs `og list-delete <name>` (warn that all goals/tasks in it will be removed)
   - "Done" — exit silently

6. Execute the chosen action and print its output. Keep your final reply to one short line confirming what happened.

Rules:
- Do NOT skip the `question` calls — that's the whole point of `/og`.
- If the user types something that doesn't map to a listed option, reply with one line: "Cancelled."
- Never modify roadmaps without an explicit user pick.
- There is no auto-created default roadmap. Users must always name their first one.
