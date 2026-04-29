# OpenGoal Internals

Implementation notes for the `og` Go binary. For the higher-level system
topology (agents, slash commands, file flow), see
[`docs/ARCHITECTURE.md`](ARCHITECTURE.md). For day-to-day usage, see the
top-level [`README.md`](../README.md).

This document is for contributors who want to modify `cmd/og/`.

---

## Source layout

The binary is a single Go package at `cmd/og/`. Files are split by
concern; everything is `package main`.

| File              | LOC  | Responsibility |
|-------------------|------|----------------|
| `main.go`         | ~400 | Types, constants, file I/O, top-level helpers, the `func main()` dispatcher |
| `goals.go`        | ~370 | Main goals, sub-goals, summary, reminder, progress calculation |
| `tasks.go`        | ~330 | Task CRUD, bulk delete, ordering helpers |
| `roadmaps.go`     | ~300 | Roadmap CRUD, the active-roadmap state machine, per-roadmap filters |
| `display.go`      | ~170 | The `today` dashboard renderer |
| `claims.go`       | ~420 | Multi-agent coordination: claim/release, TTL, dependency checks |
| `events.go`       | ~240 | Append-only event log: writer, reader, follow-mode tail |
| `bulk.go`         | ~110 | Shared flag parser for bulk-delete commands, confirmation prompt |
| `lock_unix.go`    |  32  | `withLock` via `syscall.Flock` (darwin/linux) |
| `lock_windows.go` |  41  | `withLock` via `LockFileEx` (windows) |
| `tty.go`          |  14  | `isStdinTerminal()` for interactive prompts |

Total: ~2.7 kLOC.

External dependencies (see `go.mod`):
- `golang.org/x/sys` — Windows file locking only
- `golang.org/x/term` — TTY detection

Everything else is the standard library.

## Data model

A single JSON file at `~/.local/share/opencode/goals.json` holds:

```go
type GoalsData struct {
    Roadmaps        []Roadmap   `json:"lists"`
    ActiveRoadmapID string      `json:"active_list_id"`
    MainGoals       []MainGoal  `json:"main_goals"`
    SubGoals        []SubGoal   `json:"sub_goals"`
    Tasks           []Task      `json:"tasks"`
}
```

The JSON keys (`lists`, `active_list_id`, `list_id`) are intentionally
preserved from the v1 "lists" naming so existing data files keep working.
Internally everything uses the `Roadmap` name.

Every goal/task carries a `RoadmapID` linking it to its parent roadmap.
A one-shot migration (`migrateOrphanRoadmapIDs`) backfills empty IDs on
first read of pre-multi-roadmap data files.

## Persistence guarantees

1. **Atomic snapshot writes.** `writeGoals` (in `main.go`) writes to a
   temp file in the same directory, then `os.Rename`s over the target.
   Readers never see a partial file even if `og` crashes mid-write.
2. **Cross-process serialization.** `withLock(fn)` acquires an exclusive
   advisory `flock` on `goals.json.lock` for the entire read-modify-write
   sequence. Two `og` processes can run concurrently without losing
   updates. The lock file is created lazily and never deleted (deletion
   would race with other holders).
3. **Best-effort event log.** `appendEvent` opens the events file with
   `O_APPEND` and writes one JSON line. Failures are logged to stderr
   but never abort the calling command — observability must not block
   real state changes. Cross-process visibility relies on the POSIX
   guarantee that small `O_APPEND` writes are atomic; each event line
   is well under PIPE_BUF.

## Multi-agent coordination

See `claims.go` for full code. Three concepts:

- **Agent identity** — `$OPENGOAL_AGENT` (fallback: `hostname-pid`).
- **Claim TTL** — `$OPENGOAL_CLAIM_TTL` seconds, default 1800.
  `claimActive(t, ttl)` is the single source of truth for "is this
  claim live?". Stale (past-TTL) claims are treated as unclaimed.
- **Dependencies** — `Task.DependsOn []string` lists task IDs that must
  be `Completed` before the task can be claimed. `blockedDeps()`
  returns the unsatisfied set; `nextTask` and `claimTask` consult it.

The race that the lock prevents: without `withLock`, two agents calling
`task-next --claim` could both observe an unclaimed task and both write
their own `Assignee`. The lock ensures the read-decision-write happens
atomically.

## Event log schema

`~/.local/share/opencode/goals.events.jsonl`. One JSON object per line:

```go
type Event struct {
    TS        time.Time      `json:"ts"`
    Actor     string         `json:"actor"`
    Event     string         `json:"event"`
    TaskID    string         `json:"task_id,omitempty"`
    RoadmapID string         `json:"list_id,omitempty"`
    Title     string         `json:"title,omitempty"`
    Data      map[string]any `json:"data,omitempty"`
}
```

Event names are constants in `events.go`: `task.added`, `task.claimed`,
`task.released`, `task.completed`, `task.unblocked`, `task.deleted`,
`claim.refused`, `task.cleared`. Events are written *inside* the same
`withLock()` critical section that performs the state mutation, so log
order matches the authoritative order of state changes.

## Adding a new subcommand

1. Add the handler function to the appropriate file
   (`goals.go` / `tasks.go` / `roadmaps.go` / `claims.go`).
2. Wrap any read-modify-write block in `withLock(func() { ... })`.
3. Emit an `appendEvent(...)` call inside the locked section if the
   command mutates task state.
4. Add a `case` in the `func main()` switch in `main.go`.
5. Add a line to `printUsage` (also in `main.go`).
6. If user-facing, add a matching `.md` slash-command file under
   `install/commands/` and document it in `README.md` and
   `docs/SKILL.md`.

## Testing

Tests live alongside the source as `*_test.go` files. Run them with:

```bash
go test ./cmd/og/...
```

Tests use a per-test temp `HOME` (via `t.TempDir()` and `t.Setenv("HOME", ...)`)
so they don't touch your real `goals.json`. Coverage focuses on the
high-risk pieces: locking, atomic writes, claim TTL, dependency-blocking.

## Building

```bash
make build      # → ./og
make install    # → ~/.local/bin/og + agents/commands
```

Or directly:

```bash
go build -o og ./cmd/og
```

## Runtime files

| Path | Purpose |
|------|---------|
| `~/.local/share/opencode/goals.json` | State snapshot |
| `~/.local/share/opencode/goals.json.lock` | flock target (never deleted) |
| `~/.local/share/opencode/goals.events.jsonl` | Append-only event log |
| `~/.local/bin/og` | Installed binary (per `install.sh`) |
| `~/.config/opencode/agents/*.md` | Installed OpenCode agents |
| `~/.config/opencode/commands/*.md` | Installed slash commands |

The data files are intentionally NOT renamed away from `goals.*` — that
is the on-disk format and renaming would break existing installs.
