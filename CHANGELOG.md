# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2026-04-29

### Breaking
- **CLI renamed**: `goals` → `og`. Update all scripts/invocations.
- **Slash commands renamed**: `/goals-*` → `/og-*` (e.g. `/goals-main` → `/og-main`).
- **Repo restructured**: source moved from `main.go` (root) to `cmd/og/`. OpenCode integration files moved to `install/agents/` and `install/commands/`.
- Display vocabulary changed: "list" → "roadmap" in user-facing output (CLI subcommands `list-create`, `list-use`, etc. unchanged for backwards compat).

### Added
- **Multi-agent coordination**: `task-claim`, `task-release`, `task-next [--claim]`, `task-show`, `task-add --depends id1,id2`. File-locked (flock) so multiple agents can safely operate on the same data.
- **Append-only event log** at `~/.local/share/opencode/goals.events.jsonl`. New `og events [--follow] [--since 5m] [--filter substr]` subcommand.
- **Three OpenCode agents**: `orchestrator` (primary, event-driven loop), `worker` (claims/completes work), `reviewer` (validates with `Review:`-task pattern using `--depends`).
- **Stale claim auto-expiry** (default 1800s, configurable via `$OPENGOAL_CLAIM_TTL`).
- **`Makefile`** with `make install`, `make install-bin`, `make install-opencode`, `make uninstall`, `make test`, `make clean`.
- Consolidated entry-point `README.md`; deeper docs moved to `docs/`.

### Changed
- `install.sh` rewritten to install both binary and OpenCode integration in one step.
- Release workflow updates binary names from `goals-*` to `og-*` and uses `./cmd/og`.

### Migration from 1.x
1. `git pull && make install` (binary will be at `~/.local/bin/og`).
2. Update any custom scripts: `goals X` → `og X`.
3. Update slash command muscle memory: `/goals-X` → `/og-X`.
4. Data file at `~/.local/share/opencode/goals.json` is unchanged — no migration needed.

## [1.1.1] - 2026-04-28

### Fixed
- **Atomic writes**: `goals.json` is now replaced via `os.CreateTemp` + `os.Rename`. The old "rename to `.backup` then write" dance left the data file in an inconsistent state on failure. The persistent `.backup` file is no longer created.
- **Orphan migration**: pre-existing items lacking `list_id` are migrated once at read time and persisted, instead of being re-interpreted on every list switch (which previously caused them to follow the active list around and survive list deletion).

### Changed
- Refactored `main.go`: introduced `StatusPending/InProgress/Completed`, `PriorityHigh/Medium/Low`, and date-format constants; consolidated error exits into a `die()` helper; deduplicated the three per-list filter helpers behind a generic `filterByList`.
- Removed unused `DailySummaries` and `LastReminder` fields from the data model.
- `printUsage` now distinguishes "no command" from "unknown command" and supports `goals help`.
- All single-line `Println("\n…\n")` patterns rewritten to use `Printf` so trailing whitespace isn't doubled.

### Removed
- Stale docs: `GITLAB_SETUP.md`, `READY_TO_PUSH.md`, `SHARING_GUIDE.md`, `DISTRIBUTION.md`, `README_GO.md`, `GETTING_STARTED.md`, `QUICK_REFERENCE.md`, `TASKS_REFERENCE.md`, `package.sh`, `.gitlab-ci.yml`. README.md and SKILL.md are now the canonical user docs.

## [1.1.0] - 2026-04-28

### Added
- **Multiple lists** — organize goals/tasks into separate, switchable lists (e.g. work, personal, side-project).
- New `goals` subcommands: `list-ls`, `list-create`, `list-use`, `list-rename`, `list-delete`, `list-show`.
- `/og` — interactive list browser (pick a list, view its tree, switch/rename/delete).
- `/ogl` — show all lists.
- `/ogc <name>` — create a new list and switch to it.
- `/ogs <name>` — switch the active list.
- `/ogd [name]` — delete a list (with confirmation; prompts to pick if no name given).

### Changed
- All goal/task commands now operate on the *active* list. Pre-existing data is migrated into the active list automatically.
- Dashboard header now shows the current list name.

### Notes
- No auto-created default list. After upgrading, run `goals list-create <name>` (or `/ogc <name>`) to create your first list.

## [1.0.0] - 2026-04-23

### Added
- Initial release of OpenCode Goal Tracker
- Main goals system with sub-goals and progress tracking
- Task list with priority levels (high, medium, low)
- `/today` dashboard command showing complete overview
- `/goals-main` - Add main goals
- `/goals-sub` - Add sub-goals to main goals
- `/goals-list` - List all goals with progress
- `/goals-done` - Mark goals complete
- `/goals-summary` - Generate daily summary
- `/goals-remind` - Show reminders
- `/task-add` - Add tasks with optional priority
- `/task-list` - List all tasks
- `/task-done` - Mark tasks complete
- `/task-delete` - Delete tasks
- `/task-clear` - Clear completed tasks
- Persistent memory storage in `~/.local/share/opencode/goals.json`
- Auto-completion of main goals when all sub-goals complete
- Automatic progress percentage calculation
- Auto-backup of data on every write
- Cross-platform support (macOS, Linux, Windows)
- Installation script for easy setup
- Build script for cross-compilation
- GitLab CI/CD pipeline for automated releases
- Comprehensive documentation

### Performance
- Lightning-fast 12ms average response time
- 14x faster than JavaScript alternatives
- Single compiled binary (~2.5MB)
- No runtime dependencies

[1.1.1]: https://github.com/e-sigs/opengoal/releases/tag/v1.1.1
[1.1.0]: https://github.com/e-sigs/opengoal/releases/tag/v1.1.0
[1.0.0]: https://github.com/e-sigs/opengoal/releases/tag/v1.0.0
[2.0.0]: https://github.com/e-sigs/opengoal/releases/tag/v2.0.0
