# Airyra - High-Level Specification (v2)

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
2. **Local by Default** - Runs on localhost or local network, no cloud required
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
          │  reads airyra.toml    │  project in URL path
          │  to get project name  │  /v1/projects/{project}/...
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
├── airyra.log          # Server logs (rotated: 10MB max, keep 5 files)
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

### Base URL & Headers

All API endpoints are prefixed with `/v1/projects/{project}/` where `{project}` is the project name.

Requests should include:
- Header: `X-Airyra-Agent: agent-id` (for claiming/audit)

CLI auto-generates agent ID as: `{user}@{hostname}:{cwd}`

### Task Operations
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v1/projects/{project}/tasks` | List tasks (filterable, paginated) |
| GET | `/v1/projects/{project}/tasks/ready` | Get actionable tasks (paginated) |
| GET | `/v1/projects/{project}/tasks/:id` | Get single task with deps |
| POST | `/v1/projects/{project}/tasks` | Create task |
| PATCH | `/v1/projects/{project}/tasks/:id` | Update task |
| DELETE | `/v1/projects/{project}/tasks/:id` | Delete task (cascades) |

### Status Transitions
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/v1/projects/{project}/tasks/:id/claim` | Claim task (open → in_progress) |
| POST | `/v1/projects/{project}/tasks/:id/done` | Complete task (in_progress → done) |
| POST | `/v1/projects/{project}/tasks/:id/release` | Release task (in_progress → open) |
| POST | `/v1/projects/{project}/tasks/:id/block` | Block task (any → blocked) |
| POST | `/v1/projects/{project}/tasks/:id/unblock` | Unblock task (blocked → open) |

### Dependency Operations
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v1/projects/{project}/tasks/:id/deps` | List task's dependencies |
| POST | `/v1/projects/{project}/tasks/:id/deps` | Add dependency |
| DELETE | `/v1/projects/{project}/tasks/:id/deps/:dep_id` | Remove dependency |

### Audit Operations
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v1/projects/{project}/tasks/:id/history` | Get task's change history |
| GET | `/v1/projects/{project}/audit` | Query audit log (filterable) |

### System
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v1/health` | Server health check |
| GET | `/v1/projects` | List known projects |

### Pagination

List endpoints support pagination via query parameters:
- `?page=1` - Page number (default: 1)
- `?per_page=50` - Items per page (default: 50, max: 100)

Response includes pagination metadata:
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 50,
    "total": 142,
    "total_pages": 3
  }
}
```

### Optimistic Locking

Update operations include `updated_at` in response. CLI tracks this value and warns if task changed since last read:
```
Warning: Task ar-a1b2 was modified since you last read it.
Current title: "New title" (changed by user@host:/path)
Proceed anyway? [y/N]
```

## 8. CLI Commands

### Help System
```bash
ar --help             # Show all commands with descriptions
ar <cmd> --help       # Show detailed usage for command
ar                    # (no args) Show short usage summary
```

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

**Recovery**: If server crashes, delete `~/.airyra/airyra.pid` and restart.

### Project Setup
```bash
ar init <name>        # Create airyra.toml in current directory
```

### Task Management
```bash
ar create "title" [-p priority] [-d "description"] [--parent=<id>]
ar list [--status=open] [--priority=0] [--page=1] [--per-page=50]
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
- **Optimistic locking**: CLI warns when task modified since last read

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

All errors return a standardized schema:
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message",
    "context": { ... }
  }
}
```

| Error | HTTP Code | Code | Context |
|-------|-----------|------|---------|
| Server not running | N/A | N/A | CLI error with start instructions |
| Project not found | 404 | `PROJECT_NOT_FOUND` | `{"project": "name"}` |
| Task not found | 404 | `TASK_NOT_FOUND` | `{"id": "ar-xxxx"}` |
| Already claimed | 409 | `ALREADY_CLAIMED` | `{"claimed_by": "agent-x", "claimed_at": "..."}` |
| Not claimed by you | 403 | `NOT_OWNER` | `{"claimed_by": "agent-x"}` |
| Invalid transition | 400 | `INVALID_TRANSITION` | `{"from": "done", "to": "in_progress"}` |
| Validation failed | 400 | `VALIDATION_FAILED` | `{"details": [...]}` |
| Cycle detected | 400 | `CYCLE_DETECTED` | `{"path": ["ar-1", "ar-2", "ar-1"]}` |
| Conflict (stale data) | 409 | `CONFLICT` | `{"updated_at": "...", "updated_by": "..."}` |
| Server error | 500 | `INTERNAL_ERROR` | `{}` |

## 11. Server Behavior

- **Explicit start**: Server must be manually started
- **Graceful shutdown**: Finish pending requests on stop
- **Lazy DB creation**: Project database created on first use
- **No auth**: Local network, trusted environment
- **PID file**: `~/.airyra/airyra.pid` for process management
- **Log rotation**: 10MB per file, keep 5 files
- **Recovery**: If server won't start, delete stale PID file and retry

---

## Summary of Decisions

| Decision | Choice |
|----------|--------|
| Language | Go |
| Database | SQLite (one per project) |
| Auth | None (local network, trusted) |
| Server model | Single global server, explicit start |
| Project identity | Explicit via `airyra.toml` |
| Storage location | `~/.airyra/projects/` |
| Concurrency | Atomic claiming via status transition |
| History tracking | Yes - audit log |
| API versioning | `/v1/` prefix |
| Project in URL | `/v1/projects/{project}/...` |
| Pagination | `?page=&per_page=` with metadata |
| Error format | `{error: {code, message, context}}` |
| Agent ID (CLI) | Auto: `user@hostname:cwd` |
| Log rotation | 10MB, keep 5 files |

---

## Changelog (v1 → v2)

| Area | v1 | v2 |
|------|----|----|
| Project context | `X-Airyra-Project` header | URL path `/v1/projects/{project}/` |
| API versioning | None | `/v1/` prefix |
| Agent ID (CLI) | Unspecified | Auto: `user@hostname:cwd` |
| Pagination | None | `?page=&per_page=` with metadata |
| Error format | Inconsistent | Standardized `{error: {code, message, context}}` |
| CLI help | Not mentioned | `ar --help`, `ar <cmd> --help` |
| Log rotation | Not mentioned | 10MB, keep 5 files |
| Crash recovery | Not mentioned | Delete PID file, restart |
| Optimistic locking | None | CLI warns on conflict |
