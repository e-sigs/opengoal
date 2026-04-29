# opengoal

A local-first goal & task tracker designed to coordinate **multiple AI agents** working on the same plan. Built as a tiny single-binary CLI (`og`) plus a set of [OpenCode](https://opencode.ai) agents and slash commands.

> Roadmaps hold goals, sub-goals, and tasks. Multiple agents can claim, complete, and review tasks concurrently — with file locks, dependency tracking, and an append-only event log.

---

## Features

- **Roadmaps** — independent named plans, each with their own goals/tasks
- **Goals & sub-goals** — hierarchical, with progress rolled up
- **Tasks** — priorities, dependencies (`--depends id1,id2`), claim/release for multi-agent coordination
- **Atomic data** — flock-protected writes, append-only event log
- **OpenCode integration** — `orchestrator` / `worker` / `reviewer` agents, plus `/today`, `/og*`, `/task-*` slash commands
- **No daemon, no server** — single Go binary, JSON file under `~/.local/share/opencode/`

---

## Install

Requires **Go 1.21+**.

```bash
git clone https://github.com/e-sigs/opengoal.git
cd opengoal
make install         # or: ./install.sh
```

This will:
1. Build `og` and install it to `~/.local/bin/og`
2. Copy the OpenCode agents into `~/.config/opencode/agents/`
3. Copy the OpenCode slash commands into `~/.config/opencode/commands/`

Make sure `~/.local/bin` is on your `PATH`. Then:

```bash
og help
og list-create my-first-roadmap
og add-main "Ship v1"
og task-add "Write the README" high
og today
```

### Just the binary

If you don't use OpenCode, only install the binary:

```bash
make install-bin
```

### Custom install location

```bash
PREFIX=/usr/local sudo -E make install
```

### Uninstall

```bash
make uninstall
# Your data at ~/.local/share/opencode/goals* is preserved.
```

---

## Quick reference

```text
Goals:
  og list                              List goals in active roadmap
  og add-main <title>                  Add a main goal
  og add-sub <parent-id> <title>       Add a sub-goal
  og done <id>                         Mark goal complete
  og summary                           Daily summary
  og remind                            Reminder

Tasks:
  og task-list                         List tasks
  og task-add <title> [priority] [--depends id1,id2]
  og task-show <id>                    Show task with deps + claim
  og task-done <id>                    Mark complete
  og task-delete <id>                  Delete
  og task-clear                        Remove all completed

Multi-agent:
  og task-next [--claim]               Next actionable task
  og task-claim <id>                   Claim for current agent
  og task-release <id>                 Release a claim

  Set $OPENGOAL_AGENT to identify the agent (default: hostname-pid).
  Stale claims auto-expire after $OPENGOAL_CLAIM_TTL seconds (default 1800).

Event log:
  og events [--follow] [--since 5m|RFC3339] [--filter substr]

Roadmaps:
  og list-ls                           Show all roadmaps
  og list-create <name>                Create + switch
  og list-use <id|name>                Switch active roadmap
  og list-rename <id|name> <new>       Rename
  og list-delete <id|name>             Delete + contents
  og list-show <id|name>               Show full tree

Dashboard:
  og today                             Today's view
```

---

## OpenCode integration

After install, these slash commands are available inside OpenCode:

| Slash command | What it does |
|---|---|
| `/today` | Dashboard for the active roadmap |
| `/og` | Interactive roadmap browser |
| `/ogl`, `/ogc <name>`, `/ogs <name>`, `/ogd [name]` | List, create, switch, delete roadmaps |
| `/og-main`, `/og-sub` | Add main / sub-goals |
| `/og-list`, `/og-done`, `/og-summary`, `/og-remind` | Goal management |
| `/task-add`, `/task-list`, `/task-done`, `/task-delete`, `/task-clear` | Task management |

### Agents

| Agent | Role |
|---|---|
| `@orchestrator` | Primary agent: inspects roadmap state, dispatches `worker` and `reviewer` subagents, loops up to 10 iterations per turn. |
| `@worker` | Subagent: claims an unblocked work task, completes it, releases. |
| `@reviewer` | Subagent: claims a `Review:` task, validates the work, marks done or files a `Fix:` task. |

The reviewer pattern uses task dependencies: a `Review:` task is added with `--depends <work-task-id>` so it stays blocked until the worker finishes.

---

## Data files

| Path | Purpose |
|---|---|
| `~/.local/share/opencode/goals.json` | All roadmaps, goals, sub-goals, tasks |
| `~/.local/share/opencode/goals.json.lock` | flock target (never deleted) |
| `~/.local/share/opencode/goals.events.jsonl` | Append-only event log |

The data files are intentionally NOT renamed from `goals.*` — that's the on-disk format.

---

## More docs

- [`CHANGELOG.md`](CHANGELOG.md) — release notes
- [`docs/SKILL.md`](docs/SKILL.md) — the OpenCode skill that ties it together
- [`docs/GETTING_STARTED.md`](docs/GETTING_STARTED.md) — a longer walk-through
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — system topology, agents, concurrency model
- [`docs/internals.md`](docs/internals.md) — Go implementation notes

---

## License

MIT. See [LICENSE](LICENSE).
