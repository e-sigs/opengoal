# OpenGoal Architecture

This document describes how OpenGoal is organized as a system. For Go
implementation specifics, see [`internals.md`](internals.md). For
day-to-day usage, see the top-level [`README.md`](../README.md).

## TL;DR

OpenGoal is a **single Go CLI binary plus a set of OpenCode markdown
files**. There is no daemon, no server, and no socket. All concurrency
between AI agents happens through filesystem primitives: an exclusive
flock for serialization, atomic-rename writes for the state snapshot,
and an append-only event log for audit.

## System topology

```
┌───────────────────────────── User's machine ────────────────────────────┐
│                                                                         │
│   OpenCode session  (interactive AI shell)                              │
│   │                                                                     │
│   ├── Slash commands  (~/.config/opencode/commands/*.md)                │
│   │     /today  /og  /ogl  /ogc  /ogs  /ogd                             │
│   │     /og-main  /og-sub  /og-list  /og-done ...                       │
│   │     /task-add /task-list /task-done /task-delete ...                │
│   │                                                                     │
│   └── Agents  (~/.config/opencode/agents/*.md)                          │
│         @orchestrator  (primary,  edit: deny  — delegates only)         │
│         @worker        (subagent, edit: allow — implements tasks)       │
│         @reviewer      (subagent, edit: deny  — audits worker output)   │
│                                                                         │
│   Each .md command embeds a shell call:  !`og <subcmd> ...`             │
│         │                                                               │
│         ▼                                                               │
│   og  (Go binary at ~/.local/bin/og — built from cmd/og)                │
│         │ takes flock,  reads/writes JSON,  appends event,              │
│         │ releases flock,  exits.                                       │
│         ▼                                                               │
│   Data dir:  ~/.local/share/opencode/                                   │
│         goals.json           (state snapshot — atomic rename)           │
│         goals.json.lock      (flock target, never deleted)              │
│         goals.events.jsonl   (append-only audit log)                    │
└─────────────────────────────────────────────────────────────────────────┘
```

## Components

### 1. The `og` binary

A small Go CLI built from `cmd/og/`. Every invocation:

1. Parses one subcommand (`og task-add`, `og task-claim`, …).
2. For mutations, takes an exclusive `flock` on `goals.json.lock`.
3. Reads `goals.json` into memory.
4. Mutates the in-memory snapshot.
5. Writes a temp file, then atomically renames it over `goals.json`.
6. Appends a JSON event line to `goals.events.jsonl`.
7. Releases the lock and exits.

Startup is ~15 ms, so it's fine to invoke per command. There is no
state held between invocations beyond the on-disk files.

For source layout, see [`internals.md`](internals.md#source-layout).

### 2. Slash commands (`install/commands/*.md`)

Each is a YAML front-matter file plus a single embedded `!`og <cmd>``
directive. OpenCode runs the embedded shell command and shows its
output; the surrounding prose tells the AI agent what (if anything) to
say in addition.

For example, `ogl.md` is essentially:

```markdown
---
description: List all opengoal roadmaps
agent: general
subtask: false
---

!`og list-all`

Do not add any commentary or repeat the output above.
```

This pattern keeps the slash commands trivial — all the real logic
lives in the `og` binary.

### 3. Agents (`install/agents/*.md`)

Three agents make up the multi-agent loop:

| Agent           | Mode      | Permissions                        | Role |
|-----------------|-----------|------------------------------------|------|
| `@orchestrator` | primary   | `og *`, read-only bash, no edit    | Plans work/review pairs, delegates, verifies via the event log |
| `@worker`       | subagent  | full edit, full bash               | Claims a task, implements it, marks done |
| `@reviewer`     | subagent  | read-only (`git diff`, `og *`, …)  | Claims a `Review:` task, audits worker output, approves or files a `Fix:` task |

The orchestrator never claims tasks itself. It runs `og task-next`,
spawns either `@worker` or `@reviewer` (based on whether the task title
starts with `Review:`), and uses `og events --since 30s` as ground
truth — never trusting the subagent's narrative report.

The worker→reviewer handoff is enforced by the dependency system: the
`Review:` task is added with `--depends <work-task-id>`, so it stays
blocked until the worker finishes.

### 4. Persistent state

Three files under `~/.local/share/opencode/`:

| File                  | Purpose | Concurrency |
|-----------------------|---------|-------------|
| `goals.json`          | Authoritative state snapshot | Atomic rename writes |
| `goals.json.lock`     | flock target | One exclusive holder at a time |
| `goals.events.jsonl`  | Append-only audit log | `O_APPEND`, sub-PIPE_BUF writes |

## Concurrency model

This is the most interesting part of the design. There is no central
process, yet many `og` invocations across many AI agents must coordinate
safely on the same data files.

### Serialization: `withLock`

Every read-modify-write sequence is wrapped in `withLock(fn)`, which
acquires an exclusive advisory `flock` on `goals.json.lock`, runs the
function, and releases. This serializes mutations across any number of
`og` processes on the same machine.

The lock file is created lazily on first use and never deleted —
deleting it would race with other holders. It exists as a stable name
to lock against, nothing more.

### Atomic snapshots: temp + rename

`writeGoals` writes the new JSON to a temp file in the same directory
as `goals.json`, then `os.Rename`s it over the target. Since `rename(2)`
is atomic on a single filesystem, readers always see either the old
file or the new file — never a partial write, even if `og` crashes
mid-write.

### Audit log: append-only

The event log is intentionally separate from the state snapshot:

1. Readers tailing the log don't compete for the state lock.
2. The log can grow indefinitely without bloating the snapshot.
3. A corrupted state file can be partly reconstructed from the log.

Events are written *inside* the same `withLock` critical section as the
state mutation, so on-disk log order matches the authoritative order of
state changes. Cross-process visibility of appends relies on POSIX's
guarantee that `O_APPEND` writes under PIPE_BUF (≥ 512 B) are atomic.
Each event line is well under that, so concurrent appends from different
processes interleave cleanly without an extra lock.

### Claim TTL

Multi-agent coordination needs a way to recover from crashed agents
without a supervisor. Each task carries `Assignee` and `ClaimedAt`
fields. A claim is "live" if `time.Since(ClaimedAt) <= ttl`; otherwise
it's "stale" and any other agent may take it. The default TTL is 30
minutes (`$OPENGOAL_CLAIM_TTL` overrides it).

`claimActive(t, ttl)` is the single source of truth for "is this claim
live?" — used by both `task-next` (to skip claimed tasks) and
`task-claim` (to refuse races).

### Dependency blocking

`Task.DependsOn []string` references task IDs that must be `Completed`
before the task is actionable. `blockedDeps()` returns the unsatisfied
set. The reviewer pattern leans entirely on this: a `Review:` task is
created with `--depends <work-task-id>`, and the dep system prevents
any agent from claiming the review until the worker finishes.

Note that having a *live claim* on a dep is not the same as the dep
being completed — only `Completed: true` satisfies a dependency.

## End-to-end flow: `/task-add "Write tests" high`

```
User types  /task-add "Write tests" high
      │
      ▼
OpenCode reads ~/.config/opencode/commands/task-add.md
  → executes embedded   !`og task-add "Write tests" high`
      │
      ▼
og process starts (~15 ms)
  1. main.go  parses args → calls addTask()
  2. withLock(...)
       a. readGoals()                 → goals.json
       b. mutate Tasks slice
       c. writeGoals()                → temp + rename
       d. appendEvent(EvTaskAdded)    → goals.events.jsonl
     unlock
  3. prints confirmation, exits 0
      │
      ▼
OpenCode shows the output, command finishes.
```

## End-to-end flow: orchestrator loop

```
User: "Add password reset to the auth flow."
      │
      ▼
@orchestrator
  1. plans:   og task-add "Implement password reset endpoint" high       → WORK1
              og task-add "Review: password reset endpoint"   high       --depends WORK1
              … (more pairs as needed)
  2. shows plan to user, waits for confirmation
  3. loop (cap 10 iterations):
       og task-next               → picks the next actionable task
       if title starts with "Review:":  spawn @reviewer
       else:                            spawn @worker
       og events --since 30s --filter task-<id>
                                  → verify a `task.completed` event
       repeat
  4. terminates when:
       - "No actionable tasks" → drained
       - cap reached
       - subagent fails unrecoverably
       - user goal satisfied
```

`@worker` and `@reviewer` each invoke `og task-next --claim`, do their
work, and run `og task-done <id>` (worker) or `og task-done`/
`og task-add "Fix: …"` (reviewer). All claim/done writes go through
`withLock` and append to the event log — so when the orchestrator
verifies via `og events`, it's looking at the same source of truth that
serialized the writes.

## What this design buys you

- **No daemon** — nothing to install/supervise/restart. `og` is just a
  CLI.
- **No network** — all data is local; nothing is sent anywhere.
- **Crash-safe** — atomic rename means `goals.json` is never partial.
  flock + claim TTL means a crashed agent can't permanently block work.
- **Auditable** — the event log is the truth; agent self-reports are
  cross-checkable.
- **Multi-agent native** — the same primitives that protect a single
  human from concurrent shells (lock + atomic write) protect dozens of
  AI agents from racing each other.
- **Trivially backed up** — the data dir is three files. `cp` is a
  valid backup strategy.

## What this design does *not* do

- **No multi-machine sync.** flock is local-only. To share a roadmap
  between machines, you'd need to put `~/.local/share/opencode/` on a
  shared filesystem and accept the failure modes of locking there
  (NFS flock, etc.). Not currently tested or supported.
- **No history queries on the snapshot.** The state file holds only the
  current state. Historical analysis comes from the event log.
- **No log rotation.** `goals.events.jsonl` grows forever. For now this
  is fine — events are tiny — but very long-lived installs may want to
  archive periodically.
- **No conflict resolution.** flock prevents races, but the design
  assumes a single source of truth. Two clones of the data dir cannot
  be merged.
