# Airyra - High-Level Specification

## 1. Problem Statement

Current git-backed task trackers (like Beads) suffer from:
- **Merge conflicts** when multiple agents modify tasks simultaneously
- **Sync delays** requiring background daemons and eventual consistency
- **State corruption** from concurrent JSONL file edits
- **Complex recovery** when things go wrong

## 2. Solution

A lightweight global HTTP server that provides:
- **Single source of truth** for task state
- **Atomic operations** via database transactions
- **Real-time consistency** - no sync needed
- **Atomic task claiming** - prevents two agents working on same task
- **Project isolation** - separate SQLite database per project

## 3. Design Principles

1. **Simplicity First** - Minimal dependencies, easy to understand
2. **Local by Default** - Runs on localhost, no cloud required
3. **AI-Native** - JSON-first API, designed for agent consumption
4. **Fail-Safe** - SQLite for durability, transactions for consistency
5. **Explicit over Magic** - No auto-start, no hidden behavior
6. **Project Isolation** - Each project's data is fully separate

## 4. Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    AI Agents / Users                     │
└─────────────────────┬───────────────────────────────────┘
                      │
          ┌───────────┴───────────┐
          ▼                       ▼
   ┌─────────────┐         ┌─────────────┐
   │   CLI (ar)  │         │  Direct API │
   │             │         │   (HTTP)    │
   └──────┬──────┘         └──────┬──────┘
          │                       │
          │  reads airyra.toml    │  project header
          │  to get project name  │  X-Airyra-Project
          │                       │
          └───────────┬───────────┘
                      ▼
              ┌─────────────────┐
              │  Global Server  │
              │    (airyra)     │
              │ localhost:7432  │
              └────────┬────────┘
                       │
         ┌─────────────┼─────────────┐
         ▼             ▼             ▼
   ┌──────────┐  ┌──────────┐  ┌──────────┐
   │ proj-a.db│  │ proj-b.db│  │ proj-c.db│
   └──────────┘  └──────────┘  └──────────┘
```

### Global Storage Layout
```
~/.airyra/
├── config.toml         # Global server config (optional)
├── airyra.pid          # Server PID file
├── airyra.log          # Server logs
└── projects/
    ├── my-app.db       # SQLite for "my-app" project
    ├── backend.db      # SQLite for "backend" project
    └── ...
```

### Project Config File
Each project has an `airyra.toml` in its root:
```toml
# airyra.toml
project = "my-app"      # Project name (required)
```

## 5. Core Concepts

### 5.1 Tasks
A task represents a unit of work with:
- **ID**: Short hash-based identifier (e.g., `ar-a1b2`)
- **Title**: Brief description
- **Description**: Optional detailed context
- **Status**: `open` → `in_progress` → `done` (or `blocked`)
- **Priority**: 0 (critical) to 4 (low), default 2

### 5.2 Hierarchy
Tasks can be nested for organization:
```
ar-a1b2           (Epic: "Build auth system")
├── ar-a1b2.1     (Task: "Design schema")
├── ar-a1b2.2     (Task: "Implement login")
│   ├── ar-a1b2.2.1  (Subtask: "Create endpoint")
│   └── ar-a1b2.2.2  (Subtask: "Add validation")
└── ar-a1b2.3     (Task: "Write tests")
```

### 5.3 Dependencies
Tasks can depend on other tasks:
- A task is **blocked** if any dependency is incomplete
- A task is **ready** if it has no incomplete dependencies
- Dependencies form a DAG (directed acyclic graph)

```
[ar-a1b2.2] ──depends on──► [ar-a1b2.1]
    │
    └── ar-a1b2.2 is blocked until ar-a1b2.1 is done
```

### 5.4 Ready Queue
The system automatically computes which tasks are actionable:
- Status is `open` (not in_progress, done, or manually blocked)
- All dependencies are `done`
- Sorted by priority (0 first), then creation time

### 5.5 Atomic Task Claiming
When an agent starts working on a task, the status transition is atomic:

```
Agent A: start-task ar-a1b2   → Success (open → in_progress, claimed by A)
Agent B: start-task ar-a1b2   → FAIL: "Task already in progress by agent-a"
```

**How it works:**
- Transitioning to `in_progress` is an atomic database operation
- The operation only succeeds if the task is currently `open`
- The `claimed_by` field records which agent owns the task
- This prevents race conditions where two agents claim the same task

**Releasing a task:**
- `ar done <id>` - Mark complete (in_progress → done)
- `ar release <id>` - Give up without completing (in_progress → open)
- `ar release <id> --force` - Admin releases task claimed by another agent

### 5.6 Audit Log
All changes are tracked for history:
- What changed (field, old value, new value)
- When it changed (timestamp)
- Who/what made the change (agent ID, user)

## 6. Data Model

### Task
| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier (ar-xxxx) |
| parent_id | string? | Parent task ID for hierarchy |
| title | string | Short description |
| description | string? | Detailed context |
| status | enum | open, in_progress, blocked, done |
| priority | int | 0-4, lower = higher priority |
| claimed_by | string? | Agent working on task (set when in_progress) |
| claimed_at | timestamp? | When task was claimed |
| created_at | timestamp | When created |
| updated_at | timestamp | Last modification |

### Dependency
| Field | Type | Description |
|-------|------|-------------|
| child_id | string | The blocked task |
| parent_id | string | The blocking task |

### AuditLog
| Field | Type | Description |
|-------|------|-------------|
| id | int | Auto-increment |
| task_id | string | Which task changed |
| action | string | create, update, delete, claim, release |
| field | string? | Which field changed (for updates) |
| old_value | string? | Previous value (JSON) |
| new_value | string? | New value (JSON) |
| changed_at | timestamp | When |
| changed_by | string | Agent/user identifier |

## 7. API Design

All requests must include:
- Header: `X-Airyra-Project: my-app` (project context)
- Header: `X-Airyra-Agent: agent-id` (for claiming/audit)

### Task Operations
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/tasks` | List tasks (filterable by status, priority) |
| GET | `/tasks/ready` | Get actionable tasks |
| GET | `/tasks/:id` | Get single task with deps |
| POST | `/tasks` | Create task |
| PATCH | `/tasks/:id` | Update task |
| DELETE | `/tasks/:id` | Delete task (cascades) |

### Status Transitions
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/tasks/:id/claim` | Claim task (open → in_progress) |
| POST | `/tasks/:id/done` | Complete task (in_progress → done) |
| POST | `/tasks/:id/release` | Release task (in_progress → open) |
| POST | `/tasks/:id/block` | Block task (any → blocked) |
| POST | `/tasks/:id/unblock` | Unblock task (blocked → open) |

### Dependency Operations
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/tasks/:id/deps` | List task's dependencies |
| POST | `/tasks/:id/deps` | Add dependency |
| DELETE | `/tasks/:id/deps/:dep_id` | Remove dependency |

### Audit Operations
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/tasks/:id/history` | Get task's change history |
| GET | `/audit` | Query audit log (filterable) |

### System
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Server health check |
| GET | `/projects` | List known projects |

## 8. CLI Commands

### Server Management
```bash
ar server start       # Start global server
ar server stop        # Stop global server
ar server status      # Check server status
```

**Note**: CLI does NOT auto-start server. If server is not running, CLI returns error:
```
Error: airyra server not running
Start with: ar server start
```

### Project Setup
```bash
ar init <name>        # Create airyra.toml in current directory
```

### Task Management
```bash
ar create "title" [-p priority] [-d "description"]
ar list [--status=open] [--priority=0]
ar show <id>
ar edit <id> [-t "title"] [-d "desc"] [-p priority]
ar delete <id>
```

### Task Status (Atomic Operations)
```bash
ar claim <id>         # Claim task (open → in_progress)
ar done <id>          # Complete task (in_progress → done)
ar release <id>       # Release without completing (in_progress → open)
ar release <id> --force  # Force release task claimed by another agent
ar block <id>         # Manually block task
ar unblock <id>       # Unblock task
```

### Dependency Management
```bash
ar dep add <child> <parent>   # child depends on parent
ar dep rm <child> <parent>
ar dep list <id>              # Show task's dependencies
```

### Ready Queue
```bash
ar ready              # List all ready tasks
ar next               # Get single highest-priority ready task
```

### History
```bash
ar history <id>       # Show task's change history
ar log                # Show recent activity
```

### Output Control
```bash
ar list --json        # JSON output for AI agents
ar ready --json
```

## 9. Concurrency Model

- **Atomic claiming**: Status transitions enforced atomically in SQLite
- **No separate locks**: The `in_progress` status IS the lock
- **Claim enforcement**: Only the claiming agent can complete/release (unless --force)
- **WAL mode**: SQLite configured for concurrent reads

### Atomic Claim Implementation
```sql
-- Claim: only succeeds if task is open
UPDATE tasks
SET status = 'in_progress',
    claimed_by = :agent_id,
    claimed_at = CURRENT_TIMESTAMP
WHERE id = :task_id AND status = 'open';
-- If 0 rows affected → task was not open → return error
```

### Status Transition Rules
| From | To | Who can do it |
|------|-----|---------------|
| open | in_progress | Any agent (atomic claim) |
| in_progress | done | Only claiming agent |
| in_progress | open | Only claiming agent (or --force) |
| any | blocked | Any agent |
| blocked | open | Any agent |

## 10. Error Handling

| Error | HTTP Code | Response |
|-------|-----------|----------|
| Server not running | N/A | CLI error with start instructions |
| Project not found | 404 | `{"error": "project not found"}` |
| Task not found | 404 | `{"error": "task not found", "id": "ar-xxxx"}` |
| Already claimed | 409 | `{"error": "task already in progress", "claimed_by": "agent-x"}` |
| Not claimed by you | 403 | `{"error": "task claimed by another agent", "claimed_by": "agent-x"}` |
| Invalid transition | 400 | `{"error": "invalid status transition", "from": "done", "to": "in_progress"}` |
| Invalid input | 400 | `{"error": "validation failed", "details": [...]}` |
| Cycle detected | 400 | `{"error": "dependency would create cycle"}` |
| Server error | 500 | `{"error": "internal error"}` |

## 11. Server Behavior

- **Explicit start**: Server must be manually started
- **Graceful shutdown**: Finish pending requests on stop
- **Lazy DB creation**: Project database created on first use
- **No auth**: Localhost only, trusted environment
- **PID file**: `~/.airyra/airyra.pid` for process management

---

## Summary of Decisions

| Decision | Choice |
|----------|--------|
| Language | Go |
| Database | SQLite (one per project) |
| Auth | None (localhost only) |
| Server model | Single global server, explicit start |
| Project identity | Explicit via `airyra.toml` |
| Storage location | `~/.airyra/projects/` |
| Concurrency | Atomic claiming via status transition |
| History tracking | Yes - audit log |
