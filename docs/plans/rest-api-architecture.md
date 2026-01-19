# REST API Architecture Plan for Airyra

## Overview

Design and implement the REST API layer for airyra based on v2 specification. This is a greenfield Go project - no code exists yet.

**Server**: `localhost:7432`
**API Prefix**: `/v1/projects/{project}/...`
**Database**: SQLite per project in `~/.airyra/projects/`

---

## Project Structure

```
airyra/
├── cmd/
│   └── airyra/main.go           # HTTP server entry point
├── internal/
│   ├── api/
│   │   ├── router.go            # chi router setup, middleware chain
│   │   ├── middleware/
│   │   │   ├── recovery.go      # Panic recovery
│   │   │   ├── logging.go       # Request logging
│   │   │   ├── agent.go         # X-Airyra-Agent extraction
│   │   │   └── project.go       # Project validation, DB injection
│   │   ├── handler/
│   │   │   ├── task.go          # Task CRUD
│   │   │   ├── transition.go    # Status transitions
│   │   │   ├── dependency.go    # Dependency management
│   │   │   ├── audit.go         # History/audit queries
│   │   │   └── system.go        # Health, projects list
│   │   ├── request/             # Request DTOs + validation
│   │   └── response/            # Response DTOs + error handling
│   ├── domain/                  # Core entities + domain errors
│   ├── service/                 # Business logic layer
│   ├── store/
│   │   ├── manager.go           # Multi-DB connection manager
│   │   └── sqlite/              # SQLite repositories
│   └── server/                  # Server lifecycle, PID management
├── pkg/idgen/                   # ar-xxxx ID generation
├── migrations/001_initial.sql   # SQLite schema
└── go.mod
```

---

## Key Decisions

| Aspect | Choice | Rationale |
|--------|--------|-----------|
| Router | `github.com/go-chi/chi/v5` | Lightweight, idiomatic, excellent middleware |
| SQLite Driver | `github.com/mattn/go-sqlite3` | Mature, well-tested CGO driver |
| Concurrency | Atomic UPDATE with WHERE clause | Prevents race conditions on claim |
| Error Handling | Domain errors → API errors | Clean separation of concerns |

---

## API Endpoints

### Task CRUD
| Method | Endpoint | Handler |
|--------|----------|---------|
| GET | `/v1/projects/{project}/tasks` | `ListTasks` |
| POST | `/v1/projects/{project}/tasks` | `CreateTask` |
| GET | `/v1/projects/{project}/tasks/ready` | `ListReadyTasks` |
| GET | `/v1/projects/{project}/tasks/{id}` | `GetTask` |
| PATCH | `/v1/projects/{project}/tasks/{id}` | `UpdateTask` |
| DELETE | `/v1/projects/{project}/tasks/{id}` | `DeleteTask` |

### Status Transitions
| Method | Endpoint | Handler |
|--------|----------|---------|
| POST | `/tasks/{id}/claim` | `ClaimTask` (open → in_progress) |
| POST | `/tasks/{id}/done` | `CompleteTask` (in_progress → done) |
| POST | `/tasks/{id}/release` | `ReleaseTask` (in_progress → open) |
| POST | `/tasks/{id}/block` | `BlockTask` (any → blocked) |
| POST | `/tasks/{id}/unblock` | `UnblockTask` (blocked → open) |

### Dependencies
| Method | Endpoint | Handler |
|--------|----------|---------|
| GET | `/tasks/{id}/deps` | `ListDependencies` |
| POST | `/tasks/{id}/deps` | `AddDependency` |
| DELETE | `/tasks/{id}/deps/{depID}` | `RemoveDependency` |

### Audit & System
| Method | Endpoint | Handler |
|--------|----------|---------|
| GET | `/tasks/{id}/history` | `GetTaskHistory` |
| GET | `/audit` | `QueryAuditLog` |
| GET | `/v1/health` | `Health` |
| GET | `/v1/projects` | `ListProjects` |

---

## Database Schema

```sql
-- Tasks
CREATE TABLE tasks (
    id          TEXT PRIMARY KEY,
    parent_id   TEXT REFERENCES tasks(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    description TEXT,
    status      TEXT NOT NULL DEFAULT 'open'
                CHECK (status IN ('open', 'in_progress', 'blocked', 'done')),
    priority    INTEGER NOT NULL DEFAULT 2 CHECK (priority BETWEEN 0 AND 4),
    claimed_by  TEXT,
    claimed_at  TEXT,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

-- Dependencies (DAG edges)
CREATE TABLE dependencies (
    child_id  TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    parent_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (child_id, parent_id),
    CHECK (child_id != parent_id)
);

-- Audit log
CREATE TABLE audit_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id    TEXT NOT NULL,
    action     TEXT NOT NULL,
    field      TEXT,
    old_value  TEXT,
    new_value  TEXT,
    changed_at TEXT NOT NULL,
    changed_by TEXT NOT NULL
);
```

---

## Middleware Chain

```
Request → Recovery → Logger → RealIP → AgentID → [ProjectContext] → Handler
```

- **Recovery**: Catch panics, return 500
- **Logger**: Log method, path, status, duration
- **AgentID**: Extract `X-Airyra-Agent` header (default: "anonymous")
- **ProjectContext**: Validate project name, inject DB connection into context

---

## Atomic Claim Pattern

```sql
UPDATE tasks
SET status = 'in_progress',
    claimed_by = :agent_id,
    claimed_at = :now,
    updated_at = :now
WHERE id = :task_id AND status = 'open';
-- If 0 rows affected → task not open → return error
```

---

## Error Response Format

```json
{
  "error": {
    "code": "ALREADY_CLAIMED",
    "message": "Task already claimed by another agent",
    "context": {
      "claimed_by": "user@host:/path",
      "claimed_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

Error codes: `TASK_NOT_FOUND` (404), `ALREADY_CLAIMED` (409), `NOT_OWNER` (403), `INVALID_TRANSITION` (400), `VALIDATION_FAILED` (400), `CYCLE_DETECTED` (400), `INTERNAL_ERROR` (500)

---

## Pagination

**Query params**: `?page=1&per_page=50` (default: page=1, per_page=50, max=100)

**Response**:
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

---

## Implementation Order

1. **Foundation**: `go.mod`, project structure, domain types
2. **Database**: Schema, manager, task repository
3. **HTTP Core**: Router, middleware, error handling
4. **Task CRUD**: Create, Read, Update, Delete, List
5. **Transitions**: Claim, Done, Release, Block, Unblock
6. **Dependencies**: Add, Remove, List, Cycle detection
7. **Audit**: History, Query
8. **System**: Health, Projects list

---

## Critical Files

- `internal/api/router.go` - Route definitions, middleware chain
- `internal/store/sqlite/task.go` - Task repo with atomic claim
- `internal/api/response/error.go` - Standardized error handling
- `internal/store/manager.go` - Multi-DB connection manager
- `migrations/001_initial.sql` - SQLite schema

---

## Verification

1. Start server: `go run cmd/airyra/main.go`
2. Health check: `curl localhost:7432/v1/health`
3. Create task: `curl -X POST localhost:7432/v1/projects/test/tasks -d '{"title":"Test"}'`
4. List tasks: `curl localhost:7432/v1/projects/test/tasks`
5. Claim task: `curl -X POST localhost:7432/v1/projects/test/tasks/ar-xxxx/claim -H 'X-Airyra-Agent: test-agent'`
6. Verify claim conflict: Second claim attempt returns 409 ALREADY_CLAIMED
7. Run tests: `go test ./...`

---

## Test Plan

### 1. Unit Tests

#### 1.1 Domain Layer (`internal/domain/`)
- [ ] Task entity validation (valid/invalid status values)
- [ ] Task entity validation (priority range 0-4)
- [ ] Domain error types and messages

#### 1.2 ID Generation (`pkg/idgen/`)
- [ ] Generates IDs in `ar-xxxx` format
- [ ] IDs are unique across multiple calls
- [ ] IDs contain valid characters

#### 1.3 Request Validation (`internal/api/request/`)
- [ ] CreateTask: title required, description optional
- [ ] CreateTask: priority defaults to 2, validates 0-4 range
- [ ] UpdateTask: partial updates allowed
- [ ] Pagination: page defaults to 1, per_page defaults to 50
- [ ] Pagination: per_page capped at 100

#### 1.4 Response Formatting (`internal/api/response/`)
- [ ] Error response structure matches spec
- [ ] Pagination response includes all required fields
- [ ] Domain errors map to correct HTTP status codes

---

### 2. Repository Tests (`internal/store/sqlite/`)

#### 2.1 Task Repository
- [ ] Create task with all fields
- [ ] Create task with minimal fields (defaults applied)
- [ ] Create subtask (with parent_id)
- [ ] Get task by ID (found)
- [ ] Get task by ID (not found)
- [ ] List tasks with pagination
- [ ] List tasks filtered by status
- [ ] Update task fields
- [ ] Delete task
- [ ] Delete task cascades to subtasks

#### 2.2 Atomic Claim Pattern
- [ ] Claim open task succeeds
- [ ] Claim already claimed task fails
- [ ] Claim done task fails
- [ ] Claim blocked task fails
- [ ] Concurrent claims - only one succeeds

#### 2.3 Dependency Repository
- [ ] Add dependency
- [ ] Add duplicate dependency (idempotent or error)
- [ ] Remove dependency
- [ ] List dependencies for task
- [ ] Self-dependency rejected (child_id != parent_id)
- [ ] Cycle detection (A→B→C→A)

#### 2.4 Audit Repository
- [ ] Log action creates entry
- [ ] Query by task_id
- [ ] Query by action type
- [ ] Query by date range
- [ ] Query by agent (changed_by)

---

### 3. Service Layer Tests (`internal/service/`)

#### 3.1 Task Service
- [ ] Create task assigns generated ID
- [ ] Create task sets timestamps
- [ ] Update task updates `updated_at`
- [ ] Delete task removes task and logs action

#### 3.2 Transition Service
- [ ] Claim: open → in_progress
- [ ] Done: in_progress → done (by owner)
- [ ] Done: in_progress → done (by non-owner) - fails with NOT_OWNER
- [ ] Release: in_progress → open (by owner)
- [ ] Release: in_progress → open (by non-owner) - fails
- [ ] Block: any status → blocked
- [ ] Unblock: blocked → open
- [ ] Invalid transitions return INVALID_TRANSITION

#### 3.3 Dependency Service
- [ ] Add dependency logs action
- [ ] Remove dependency logs action
- [ ] Cycle detection prevents circular dependencies

---

### 4. Middleware Tests (`internal/api/middleware/`)

#### 4.1 Recovery Middleware
- [ ] Panic returns 500 Internal Server Error
- [ ] Panic response follows error format
- [ ] Request continues after panic in handler

#### 4.2 Logging Middleware
- [ ] Logs request method and path
- [ ] Logs response status code
- [ ] Logs request duration

#### 4.3 Agent Middleware
- [ ] Extracts X-Airyra-Agent header
- [ ] Defaults to "anonymous" when header missing
- [ ] Agent ID available in context

#### 4.4 Project Middleware
- [ ] Valid project name passes
- [ ] Invalid project name returns 400
- [ ] DB connection injected into context
- [ ] Creates project directory if not exists

---

### 5. Handler/Integration Tests (`internal/api/handler/`)

#### 5.1 System Endpoints
| Test | Method | Endpoint | Expected |
|------|--------|----------|----------|
| Health check returns OK | GET | `/v1/health` | 200, `{"status":"ok"}` |
| List projects (empty) | GET | `/v1/projects` | 200, `[]` |
| List projects (with data) | GET | `/v1/projects` | 200, `["proj1","proj2"]` |

#### 5.2 Task CRUD
| Test | Method | Endpoint | Expected |
|------|--------|----------|----------|
| Create task | POST | `/v1/projects/{p}/tasks` | 201, task with ID |
| Create task - missing title | POST | `/v1/projects/{p}/tasks` | 400, VALIDATION_FAILED |
| Get task | GET | `/v1/projects/{p}/tasks/{id}` | 200, task object |
| Get task - not found | GET | `/v1/projects/{p}/tasks/{id}` | 404, TASK_NOT_FOUND |
| List tasks (empty) | GET | `/v1/projects/{p}/tasks` | 200, empty array |
| List tasks (with data) | GET | `/v1/projects/{p}/tasks` | 200, array of tasks |
| List tasks (pagination) | GET | `/v1/projects/{p}/tasks?page=2&per_page=10` | 200, correct page |
| List ready tasks | GET | `/v1/projects/{p}/tasks/ready` | 200, only ready tasks |
| Update task | PATCH | `/v1/projects/{p}/tasks/{id}` | 200, updated task |
| Update task - not found | PATCH | `/v1/projects/{p}/tasks/{id}` | 404 |
| Delete task | DELETE | `/v1/projects/{p}/tasks/{id}` | 204 |
| Delete task - not found | DELETE | `/v1/projects/{p}/tasks/{id}` | 404 |

#### 5.3 Status Transitions
| Test | Method | Endpoint | Expected |
|------|--------|----------|----------|
| Claim open task | POST | `/tasks/{id}/claim` | 200, claimed task |
| Claim already claimed | POST | `/tasks/{id}/claim` | 409, ALREADY_CLAIMED |
| Claim done task | POST | `/tasks/{id}/claim` | 400, INVALID_TRANSITION |
| Complete task (owner) | POST | `/tasks/{id}/done` | 200 |
| Complete task (non-owner) | POST | `/tasks/{id}/done` | 403, NOT_OWNER |
| Release task (owner) | POST | `/tasks/{id}/release` | 200, status=open |
| Release task (non-owner) | POST | `/tasks/{id}/release` | 403 |
| Block task | POST | `/tasks/{id}/block` | 200, status=blocked |
| Unblock task | POST | `/tasks/{id}/unblock` | 200, status=open |
| Unblock non-blocked task | POST | `/tasks/{id}/unblock` | 400 |

#### 5.4 Dependencies
| Test | Method | Endpoint | Expected |
|------|--------|----------|----------|
| Add dependency | POST | `/tasks/{id}/deps` | 201 |
| Add self-dependency | POST | `/tasks/{id}/deps` | 400 |
| Add cycle | POST | `/tasks/{id}/deps` | 400, CYCLE_DETECTED |
| List dependencies | GET | `/tasks/{id}/deps` | 200, array |
| Remove dependency | DELETE | `/tasks/{id}/deps/{depId}` | 204 |

#### 5.5 Audit
| Test | Method | Endpoint | Expected |
|------|--------|----------|----------|
| Get task history | GET | `/tasks/{id}/history` | 200, array of events |
| Query audit log | GET | `/audit` | 200, paginated results |
| Query audit - filter by action | GET | `/audit?action=claim` | 200, filtered |

---

### 6. Concurrency Tests

- [ ] Concurrent claims on same task - exactly one succeeds
- [ ] Concurrent updates on same task - no data corruption
- [ ] Multiple agents claiming different tasks - all succeed
- [ ] DB connection pool handles concurrent requests

---

### 7. Error Handling Tests

| Error Code | HTTP Status | Trigger |
|------------|-------------|---------|
| TASK_NOT_FOUND | 404 | Get/update/delete non-existent task |
| ALREADY_CLAIMED | 409 | Claim task that's in_progress |
| NOT_OWNER | 403 | Complete/release task claimed by other |
| INVALID_TRANSITION | 400 | Invalid status change |
| VALIDATION_FAILED | 400 | Missing required field, invalid value |
| CYCLE_DETECTED | 400 | Add dependency that creates cycle |
| INTERNAL_ERROR | 500 | Unexpected server error |

---

### 8. End-to-End Workflow Tests

#### 8.1 Happy Path: Complete Task Lifecycle
1. Create task → verify 201, ID assigned
2. List tasks → verify task appears
3. Claim task → verify status=in_progress, claimed_by set
4. Complete task → verify status=done
5. Get task history → verify all actions logged

#### 8.2 Dependency Workflow
1. Create task A
2. Create task B
3. Add dependency: B depends on A
4. List ready tasks → only A is ready
5. Complete A
6. List ready tasks → B is now ready

#### 8.3 Multi-Agent Conflict
1. Create task
2. Agent1 claims task → success
3. Agent2 claims task → 409 ALREADY_CLAIMED
4. Agent1 releases task
5. Agent2 claims task → success

---

### 9. Test Infrastructure

#### Test Helpers
- `setupTestDB()` - Create in-memory SQLite for testing
- `createTestTask()` - Factory for test tasks
- `makeRequest()` - HTTP test client helper
- `assertJSON()` - JSON response assertions

#### Test Database
- Use `:memory:` SQLite for unit/integration tests
- Use temp directory for multi-DB manager tests
- Clean up after each test

---

### 10. Test Commands

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/api/handler/...

# Run with race detection
go test -race ./...

# Verbose output
go test -v ./...
```

---

### 11. Test Files to Create

| File | Purpose |
|------|---------|
| `internal/domain/task_test.go` | Domain entity tests |
| `internal/store/sqlite/task_test.go` | Repository tests |
| `internal/store/sqlite/dependency_test.go` | Dependency repo tests |
| `internal/store/sqlite/audit_test.go` | Audit repo tests |
| `internal/service/task_test.go` | Service layer tests |
| `internal/service/transition_test.go` | Transition logic tests |
| `internal/api/handler/task_test.go` | Handler integration tests |
| `internal/api/handler/transition_test.go` | Transition handler tests |
| `internal/api/handler/dependency_test.go` | Dependency handler tests |
| `internal/api/handler/system_test.go` | Health/projects tests |
| `internal/api/middleware/middleware_test.go` | Middleware tests |
| `pkg/idgen/idgen_test.go` | ID generation tests |
| `tests/e2e/workflow_test.go` | End-to-end tests |
