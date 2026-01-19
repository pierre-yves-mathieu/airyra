-- Airyra Initial Schema Migration
-- Version: 001
-- Description: Creates core tables for task management system

-- Enable foreign key support (must be run at connection time in SQLite)
PRAGMA foreign_keys = ON;

-- ============================================================================
-- Tasks Table
-- ============================================================================
-- Core task storage with hierarchical support via parent_id
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,                                    -- Format: ar-xxxx
    parent_id TEXT,                                         -- FK for task hierarchy
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'in_progress', 'blocked', 'done')),
    priority INTEGER NOT NULL DEFAULT 2
        CHECK (priority >= 0 AND priority <= 4),
    claimed_by TEXT,
    claimed_at TEXT,                                        -- ISO 8601 timestamp
    created_at TEXT NOT NULL,                               -- ISO 8601 timestamp
    updated_at TEXT NOT NULL,                               -- ISO 8601 timestamp
    FOREIGN KEY (parent_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- ============================================================================
-- Dependencies Table
-- ============================================================================
-- Tracks task dependencies (child depends on parent)
CREATE TABLE IF NOT EXISTS dependencies (
    child_id TEXT NOT NULL,
    parent_id TEXT NOT NULL,
    PRIMARY KEY (child_id, parent_id),
    FOREIGN KEY (child_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- ============================================================================
-- Audit Log Table
-- ============================================================================
-- Tracks all changes to tasks for history and debugging
CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT,
    action TEXT NOT NULL,                                   -- e.g., 'create', 'update', 'delete'
    field TEXT,                                             -- Field that was changed
    old_value TEXT,                                         -- JSON encoded previous value
    new_value TEXT,                                         -- JSON encoded new value
    changed_at TEXT NOT NULL,                               -- ISO 8601 timestamp
    changed_by TEXT NOT NULL,                               -- Identifier of who made the change
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- ============================================================================
-- Migrations Tracking Table
-- ============================================================================
-- Tracks which migrations have been applied to this database
CREATE TABLE IF NOT EXISTS _migrations (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL                                -- ISO 8601 timestamp
);

-- ============================================================================
-- Indexes
-- ============================================================================
-- Task indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_id);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);

-- Dependencies index for reverse lookups
CREATE INDEX IF NOT EXISTS idx_deps_parent ON dependencies(parent_id);

-- Audit log indexes for querying history
CREATE INDEX IF NOT EXISTS idx_audit_task ON audit_log(task_id);
CREATE INDEX IF NOT EXISTS idx_audit_time ON audit_log(changed_at);

-- ============================================================================
-- Record Migration
-- ============================================================================
INSERT INTO _migrations (version, applied_at) VALUES (1, datetime('now'));
