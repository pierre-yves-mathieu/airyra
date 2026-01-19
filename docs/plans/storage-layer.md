# Airyra Storage Layer - Architecture Plan

## Overview

Design for the Storage Layer component of Airyra with:
- Per-project SQLite databases in `~/.airyra/projects/{project}.db`
- WAL mode for concurrent reads
- Lazy database creation on first use

---

## 1. Package Structure

```
internal/storage/
├── storage.go      # Store interface and factory
├── sqlite.go       # SQLite implementation
├── pool.go         # ProjectDBManager (connection management)
├── schema.go       # Migrations (embedded SQL)
├── tx.go           # Transaction wrapper
└── models.go       # Data structs (Task, Dependency, AuditEntry)
```

---

## 2. Core Interfaces

### Store
```
Store interface:
  Tasks         TaskRepository
  Dependencies  DependencyRepository
  AuditLogs     AuditRepository
  WithTx(ctx, fn) error    // Execute fn in transaction
  Close() error
```

### TaskRepository
```
Create, Get, List, ListReady, Update, Delete
Claim(id, agentID)    // Atomic: open → in_progress
Release(id, agentID)  // in_progress → open
MarkDone(id, agentID) // in_progress → done
```

### DependencyRepository
```
Add, Remove, ListForTask, CheckCycle
```

### AuditRepository
```
Log, ListForTask, Query
```

---

## 3. Connection Management (ProjectDBManager)

```
ProjectDBManager:
  basePath   string              // ~/.airyra/projects/
  databases  map[string]*sql.DB  // project → connection
  mutex      sync.RWMutex

  GetStore(project) → Store     // Lazy creation
  CloseAll() error
```

**Lazy Initialization Flow:**
1. Check map with RLock (fast path)
2. If missing: Lock, double-check, create DB file
3. Configure WAL + PRAGMAs, run migrations
4. Cache in map, return

**SQLite PRAGMAs:**
```sql
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
PRAGMA foreign_keys=ON;
PRAGMA synchronous=NORMAL;
```

---

## 4. Schema

### Tables

**tasks**
| Column | Type | Notes |
|--------|------|-------|
| id | TEXT PK | ar-xxxx format |
| parent_id | TEXT FK | Hierarchy |
| title | TEXT NOT NULL | |
| description | TEXT | |
| status | TEXT | open/in_progress/blocked/done |
| priority | INTEGER | 0-4, default 2 |
| claimed_by | TEXT | Agent ID |
| claimed_at | TEXT | Timestamp |
| created_at | TEXT | |
| updated_at | TEXT | |

**dependencies**
| Column | Type |
|--------|------|
| child_id | TEXT FK (PK) |
| parent_id | TEXT FK (PK) |

**audit_log**
| Column | Type |
|--------|------|
| id | INTEGER PK AUTOINCREMENT |
| task_id | TEXT |
| action | TEXT |
| field | TEXT |
| old_value | TEXT (JSON) |
| new_value | TEXT (JSON) |
| changed_at | TEXT |
| changed_by | TEXT |

### Indexes
- `idx_tasks_status`, `idx_tasks_parent`, `idx_tasks_priority`
- `idx_deps_parent`
- `idx_audit_task`, `idx_audit_time`

---

## 5. Migration Strategy

**Embedded migrations** using `//go:embed migrations/*.sql`

```
internal/storage/migrations/
├── 001_initial_schema.sql
└── ...
```

**_migrations table** tracks applied versions:
```sql
CREATE TABLE _migrations (version INTEGER PK, applied_at TEXT);
```

Runner: iterate files > current version, execute in transaction.

---

## 6. Transaction Pattern

**WithTx wrapper:**
```go
store.WithTx(ctx, func(tx TxStore) error {
    // All operations use tx
    // Return nil → commit
    // Return error → rollback
})
```

**Atomic Claim:**
```sql
UPDATE tasks
SET status='in_progress', claimed_by=?, claimed_at=datetime('now')
WHERE id=? AND status='open';
-- RowsAffected()==0 means task wasn't open
```

---

## 7. Architecture Diagram

```
┌─────────────────────────────────────────────┐
│              HTTP Handlers                   │
└──────────────────────┬──────────────────────┘
                       ▼
┌─────────────────────────────────────────────┐
│           ProjectDBManager                   │
│   map[project]*sql.DB (RWMutex protected)   │
│   GetStore(project) → Store                  │
└──────────────────────┬──────────────────────┘
                       ▼
┌─────────────────────────────────────────────┐
│                 Store                        │
│  Tasks | Dependencies | AuditLogs | WithTx  │
└──────────────────────┬──────────────────────┘
                       ▼
┌─────────────────────────────────────────────┐
│          ~/.airyra/projects/                 │
│   proj-a.db  │  proj-b.db  │  proj-c.db     │
│     (WAL)    │    (WAL)    │    (WAL)       │
└─────────────────────────────────────────────┘
```

---

## 8. Files to Create

| File | Purpose |
|------|---------|
| `internal/storage/models.go` | Task, Dependency, AuditEntry structs |
| `internal/storage/storage.go` | Store interface, NewStore factory |
| `internal/storage/pool.go` | ProjectDBManager with lazy init |
| `internal/storage/schema.go` | Migration runner |
| `internal/storage/sqlite.go` | SQLite repository implementations |
| `internal/storage/tx.go` | Transaction wrapper |
| `internal/storage/migrations/001_initial_schema.sql` | Initial DDL |

---

## 9. Verification

1. **Unit tests**: Mock `*sql.DB` to test repository logic
2. **Integration tests**: Real SQLite in temp directory
3. **Verify lazy creation**: Request non-existent project, check file created
4. **Verify WAL**: `PRAGMA journal_mode` returns `wal`
5. **Verify atomic claim**: Concurrent goroutines racing to claim same task

---

## 10. Test Plan

### 10.1 Unit Tests

#### TaskRepository Tests (`internal/storage/sqlite_test.go`)
- `TestTaskCreate` - creates a task with valid data
- `TestTaskCreate_GeneratesID` - auto-generates ar-xxxx format ID
- `TestTaskCreate_WithParent` - creates task with parent_id
- `TestTaskCreate_InvalidParent` - returns error for non-existent parent
- `TestTaskGet` - retrieves existing task by ID
- `TestTaskGet_NotFound` - returns appropriate error for missing task
- `TestTaskList` - lists all tasks
- `TestTaskList_FilterByStatus` - filters by status (open/in_progress/blocked/done)
- `TestTaskList_FilterByParent` - lists children of a parent task
- `TestTaskListReady` - returns only tasks with no unmet dependencies
- `TestTaskUpdate` - updates task fields
- `TestTaskUpdate_NotFound` - returns error for missing task
- `TestTaskDelete` - removes task
- `TestTaskDelete_WithChildren` - handles cascade or error for tasks with children
- `TestTaskClaim` - atomically sets status=in_progress, claimed_by, claimed_at
- `TestTaskClaim_AlreadyClaimed` - returns error when task not open
- `TestTaskClaim_NotFound` - returns error for missing task
- `TestTaskRelease` - sets status=open, clears claimed_by/claimed_at
- `TestTaskRelease_WrongAgent` - only claiming agent can release
- `TestTaskMarkDone` - sets status=done
- `TestTaskMarkDone_WrongAgent` - only claiming agent can mark done

#### DependencyRepository Tests
- `TestDependencyAdd` - creates dependency link
- `TestDependencyAdd_Duplicate` - handles duplicate gracefully
- `TestDependencyAdd_InvalidTask` - returns error for non-existent task IDs
- `TestDependencyRemove` - removes dependency link
- `TestDependencyListForTask` - lists all dependencies for a task
- `TestDependencyCheckCycle_NoCycle` - returns false when no cycle
- `TestDependencyCheckCycle_DirectCycle` - detects A→B→A
- `TestDependencyCheckCycle_IndirectCycle` - detects A→B→C→A
- `TestDependencyCheckCycle_SelfReference` - detects A→A

#### AuditRepository Tests
- `TestAuditLog` - creates audit entry
- `TestAuditListForTask` - lists all audit entries for a task
- `TestAuditQuery_ByAction` - filters by action type
- `TestAuditQuery_ByTimeRange` - filters by changed_at range
- `TestAuditQuery_ByChangedBy` - filters by actor

#### Transaction Tests (`internal/storage/tx_test.go`)
- `TestWithTx_Commit` - commits on nil return
- `TestWithTx_Rollback` - rolls back on error return
- `TestWithTx_PanicRollback` - rolls back on panic
- `TestWithTx_NestedOperations` - multiple operations in single tx

### 10.2 Integration Tests

#### ProjectDBManager Tests (`internal/storage/pool_test.go`)
- `TestGetStore_CreatesDatabase` - creates .db file on first access
- `TestGetStore_ReusesConnection` - returns same Store for same project
- `TestGetStore_IsolatesProjects` - different projects get different DBs
- `TestGetStore_ConcurrentAccess` - thread-safe lazy initialization
- `TestCloseAll` - closes all open connections

#### SQLite Configuration Tests
- `TestPragma_WALMode` - `PRAGMA journal_mode` returns `wal`
- `TestPragma_ForeignKeys` - foreign key constraints enforced
- `TestPragma_BusyTimeout` - busy_timeout is set correctly

#### Migration Tests (`internal/storage/schema_test.go`)
- `TestMigration_InitialSchema` - creates all tables and indexes
- `TestMigration_Idempotent` - running twice doesn't error
- `TestMigration_TracksVersion` - updates _migrations table
- `TestMigration_PartialFailure` - rolls back failed migration

### 10.3 Concurrency Tests

#### Atomic Claim Tests
- `TestClaim_ConcurrentRace` - only one goroutine succeeds when many try to claim same task
- `TestClaim_DifferentTasks` - multiple goroutines can claim different tasks simultaneously

#### WAL Concurrent Access
- `TestWAL_ConcurrentReads` - multiple readers don't block
- `TestWAL_ReadDuringWrite` - reads succeed during long write

### 10.4 Edge Cases

#### Data Integrity
- `TestTask_StatusTransitions` - only valid transitions allowed (open→in_progress→done)
- `TestTask_PriorityBounds` - priority must be 0-4
- `TestTask_TimestampFormat` - created_at/updated_at in correct format

#### Error Handling
- `TestStore_ClosedConnection` - operations fail gracefully after Close()
- `TestStore_InvalidProject` - rejects invalid project names (path traversal, etc.)

### 10.5 Model Tests (`internal/storage/models_test.go`)
- `TestTask_Validation` - title required, priority in range
- `TestAuditEntry_JSONValues` - old_value/new_value serialize correctly

### 10.6 Test Files Summary

| Test File | Purpose |
|-----------|---------|
| `internal/storage/models_test.go` | Model validation |
| `internal/storage/sqlite_test.go` | Repository unit tests |
| `internal/storage/pool_test.go` | ProjectDBManager tests |
| `internal/storage/schema_test.go` | Migration tests |
| `internal/storage/tx_test.go` | Transaction tests |
| `internal/storage/integration_test.go` | Full integration tests with real SQLite |
