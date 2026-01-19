# Airyra Global HTTP Server - Architecture Plan

## Overview

Design for a single-process HTTP server managing multiple project databases via REST API on `localhost:7432`.

## Go Package Structure

```
airyra/
├── go.mod
├── cmd/
│   └── airyra/
│       └── main.go             # Entry point: server start/stop/status
├── internal/
│   ├── server/
│   │   ├── server.go           # HTTP server lifecycle
│   │   ├── router.go           # Route registration (Go 1.22+ patterns)
│   │   └── middleware.go       # Logging, recovery, content-type
│   ├── api/
│   │   ├── handler.go          # Base handler, project/agent extraction
│   │   ├── tasks.go            # Task CRUD handlers
│   │   ├── status.go           # Claim/done/release/block/unblock
│   │   ├── deps.go             # Dependency handlers
│   │   ├── audit.go            # History handlers
│   │   ├── system.go           # Health, projects list
│   │   └── response.go         # JSON response + error mapping
│   ├── store/
│   │   ├── manager.go          # Project DB connection manager (lazy creation)
│   │   ├── db.go               # ProjectDB wrapper with transaction helper
│   │   ├── task.go             # Task repository
│   │   ├── dependency.go       # Dependency repository + cycle detection
│   │   ├── audit.go            # Audit log repository
│   │   └── schema.sql          # Embedded SQLite schema
│   ├── model/
│   │   ├── task.go             # Task struct
│   │   ├── dependency.go       # Dependency struct
│   │   ├── audit.go            # AuditEntry struct
│   │   └── errors.go           # Domain error types
│   ├── pid/
│   │   └── pid.go              # PID file acquire/release/check
│   └── logging/
│       └── logger.go           # Rotating file logger (10MB, 5 backups)
```

## Key Components

### 1. Server Lifecycle (`internal/server/server.go`)

```go
type Server struct {
    httpServer *http.Server
    pidMgr     *pid.Manager
    dbMgr      *store.Manager
    logger     *logging.Logger
}

func (s *Server) Start() error    // Acquire PID, start HTTP server
func (s *Server) Shutdown(ctx)    // Graceful: finish requests, close DBs, release PID
```

- Uses `http.Server.Shutdown()` for graceful request completion
- Signal handling: SIGINT/SIGTERM trigger 30s graceful shutdown
- PID file locking prevents duplicate instances

### 2. Database Manager (`internal/store/manager.go`)

```go
type Manager struct {
    baseDir string                // ~/.airyra/projects
    dbs     map[string]*ProjectDB // Cached connections
    mu      sync.RWMutex
}

func (m *Manager) Get(project string) (*ProjectDB, error)  // Lazy creation
func (m *Manager) ListProjects() ([]string, error)         // Scan .db files
func (m *Manager) CloseAll()                               // Shutdown cleanup
```

- **Lazy DB creation**: Database created on first API access
- **SQLite WAL mode**: Concurrent reads, single writer
- **Connection**: `SetMaxOpenConns(1)` for SQLite best practice

### 3. Atomic Task Claiming (`internal/store/task.go`)

```sql
-- Only succeeds if task is currently 'open'
UPDATE tasks
SET status = 'in_progress',
    claimed_by = :agent_id,
    claimed_at = datetime('now')
WHERE id = :task_id AND status = 'open';
-- Check rows affected: 0 = task was not open
```

- Returns `AlreadyClaimedError` if task in_progress
- Returns `InvalidTransitionError` for other non-open states

### 4. HTTP Router (`internal/server/router.go`)

Using Go 1.22+ enhanced routing:

```go
mux.HandleFunc("GET /v1/health", h.Health)
mux.HandleFunc("GET /v1/projects", h.ListProjects)
mux.HandleFunc("GET /v1/projects/{project}/tasks", h.ListTasks)
mux.HandleFunc("POST /v1/projects/{project}/tasks/{id}/claim", h.ClaimTask)
// ... etc
```

### 5. Error Response Pattern (`internal/api/response.go`)

```go
func mapError(w http.ResponseWriter, err error) {
    switch e := err.(type) {
    case *model.TaskNotFoundError:
        writeError(w, 404, "TASK_NOT_FOUND", "Task not found", {"id": e.ID})
    case *model.AlreadyClaimedError:
        writeError(w, 409, "ALREADY_CLAIMED", "Task already claimed",
            {"claimed_by": e.ClaimedBy, "claimed_at": e.ClaimedAt})
    // ... etc
    }
}
```

### 6. PID Manager (`internal/pid/pid.go`)

```go
func (m *Manager) Acquire() error  // Create PID file, fail if already running
func (m *Manager) Release() error  // Remove PID file
func (m *Manager) ReadPID() (int, error)  // For status/stop commands
```

- Checks if existing PID process is alive via `syscall.Signal(0)`
- Removes stale PID files automatically

### 7. Rotating Logger (`internal/logging/logger.go`)

```go
type Logger struct {
    file     *os.File
    size     int64
    maxSize  int64  // 10MB
    maxFiles int    // 5
}
```

- Rotates when current file exceeds 10MB
- Keeps 5 backup files: `airyra.log.1`, `airyra.log.2`, etc.

## SQLite Schema (`internal/store/schema.sql`)

```sql
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    parent_id TEXT REFERENCES tasks(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'open'
        CHECK(status IN ('open', 'in_progress', 'blocked', 'done')),
    priority INTEGER NOT NULL DEFAULT 2 CHECK(priority BETWEEN 0 AND 4),
    claimed_by TEXT,
    claimed_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE dependencies (
    child_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    parent_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (child_id, parent_id),
    CHECK (child_id != parent_id)
);

CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT NOT NULL,
    action TEXT NOT NULL,
    field TEXT,
    old_value TEXT,
    new_value TEXT,
    changed_at TEXT NOT NULL DEFAULT (datetime('now')),
    changed_by TEXT NOT NULL
);

-- Indexes
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_priority ON tasks(priority);
CREATE INDEX idx_audit_task ON audit_log(task_id);
```

## External Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/mattn/go-sqlite3` | SQLite driver (CGO-based) |

Everything else uses Go standard library.

## Implementation Phases

### Phase 1: Foundation
- [ ] Set up Go module (`go mod init`)
- [ ] Implement `internal/pid/pid.go`
- [ ] Implement `internal/logging/logger.go`
- [ ] Implement `cmd/airyra/main.go` with signal handling

### Phase 2: Database Layer
- [ ] Create `internal/store/schema.sql`
- [ ] Implement `internal/store/manager.go` (lazy DB creation)
- [ ] Implement `internal/store/db.go` (transaction helper)
- [ ] Implement `internal/store/task.go` (CRUD + atomic claim)
- [ ] Implement `internal/store/dependency.go` (with cycle detection)
- [ ] Implement `internal/store/audit.go`

### Phase 3: HTTP Layer
- [ ] Implement `internal/model/*.go` (structs and errors)
- [ ] Implement `internal/api/response.go` (error mapping)
- [ ] Implement `internal/api/handler.go` (base handler)
- [ ] Implement `internal/server/middleware.go`
- [ ] Implement `internal/server/router.go`
- [ ] Implement `internal/server/server.go`

### Phase 4: API Handlers
- [ ] Implement `internal/api/system.go` (health, projects)
- [ ] Implement `internal/api/tasks.go` (CRUD)
- [ ] Implement `internal/api/status.go` (claim/done/release/block/unblock)
- [ ] Implement `internal/api/deps.go`
- [ ] Implement `internal/api/audit.go`

## Verification

```bash
# Build
go build -o airyra ./cmd/airyra

# Start server
./airyra server start

# Check status
./airyra server status

# Test health endpoint
curl http://localhost:7432/v1/health

# Test task creation (creates project DB lazily)
curl -X POST http://localhost:7432/v1/projects/test/tasks \
  -H "Content-Type: application/json" \
  -H "X-Airyra-Agent: user@host:/path" \
  -d '{"title": "Test task", "priority": 2}'

# Test task claiming
curl -X POST http://localhost:7432/v1/projects/test/tasks/{id}/claim \
  -H "X-Airyra-Agent: user@host:/path"

# Stop server
./airyra server stop

# Verify PID file removed
ls ~/.airyra/airyra.pid  # Should not exist

# Verify log rotation
ls -la ~/.airyra/airyra.log*
```

## Critical Files

- `internal/server/server.go` - Server lifecycle coordination
- `internal/store/manager.go` - Lazy DB creation and connection pooling
- `internal/store/task.go` - Atomic claim implementation (core business logic)
- `internal/api/response.go` - Standardized error format
- `docs/spec/airyra-spec-v2.md` - Authoritative API specification
