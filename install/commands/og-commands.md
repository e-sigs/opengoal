---
description: Show all opengoal CLI + slash commands (condensed reference)
agent: general
subtask: false
---

```
opengoal — command reference

SLASH (inside OpenCode)              CLI (og <cmd>)
─────────────────────────────        ─────────────────────────────
Dashboard
  /today                             today

Roadmaps
  /og                                list
  /ogl                               list-all
  /ogc <name>                        list-create <name>
  /ogs <name>                        list-use <id|name>
  /ogd [name...]                     list-delete <id|name>...
  —                                  list-rename <id|name> <new>
  —                                  list-show <id|name>

Goals
  /og-list                           list
  /og-main <title>                   add-main <title>
  /og-sub <parent-id> <title>        add-sub <parent-id> <title>
  /og-done <id...>                   done <id...> [-y]
  /og-summary                        summary
  /og-remind                         remind

Tasks
  /task-list                         task-list
  /task-add <title> [pri] [--depends ids]
                                     task-add ...
  /task-done <id>                    task-done <id>
  /task-delete <id...>               task-delete <id...> [-y]
                                     task-delete --all | --priority h|m|l | --completed
  —                                  task-show <id>

Multi-agent
  —                                  task-next [--claim]
  —                                  task-claim <id>
  —                                  task-release <id>
  Env: $OPENGOAL_AGENT, $OPENGOAL_CLAIM_TTL (default 1800s)

Events
  —                                  events [--follow] [--since 5m|RFC3339] [--filter sub]

Help
  /og-commands                       help | -h | --help
```

Do not add commentary; the user has already seen this reference.
