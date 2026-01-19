-- Airyra Initial Schema
-- This migration creates the core tables for task management.

-- Enable WAL mode for better concurrent read performance
PRAGMA journal_mode=WAL;

-- Tasks table
CREATE TABLE IF NOT EXISTS tasks (
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

-- Index for listing tasks by status
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

-- Index for listing tasks by priority
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);

-- Index for finding subtasks
CREATE INDEX IF NOT EXISTS idx_tasks_parent_id ON tasks(parent_id);

-- Dependencies table (DAG edges)
CREATE TABLE IF NOT EXISTS dependencies (
    child_id  TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    parent_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (child_id, parent_id),
    CHECK (child_id != parent_id)
);

-- Index for finding what a task depends on
CREATE INDEX IF NOT EXISTS idx_dependencies_child ON dependencies(child_id);

-- Index for finding what depends on a task
CREATE INDEX IF NOT EXISTS idx_dependencies_parent ON dependencies(parent_id);

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id    TEXT NOT NULL,
    action     TEXT NOT NULL,
    field      TEXT,
    old_value  TEXT,
    new_value  TEXT,
    changed_at TEXT NOT NULL,
    changed_by TEXT NOT NULL
);

-- Index for querying audit log by task
CREATE INDEX IF NOT EXISTS idx_audit_log_task_id ON audit_log(task_id);

-- Index for querying audit log by action
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);

-- Index for querying audit log by agent
CREATE INDEX IF NOT EXISTS idx_audit_log_changed_by ON audit_log(changed_by);

-- Index for querying audit log by time
CREATE INDEX IF NOT EXISTS idx_audit_log_changed_at ON audit_log(changed_at);
