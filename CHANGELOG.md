# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[1.0.0]: https://gitlab.com/sig/opengoal/-/tags/v1.0.0
