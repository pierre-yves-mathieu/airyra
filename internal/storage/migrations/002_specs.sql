-- Airyra Specs Feature Migration
-- Version: 002
-- Description: Adds specs (epics) for grouping related tasks with computed status

-- Enable foreign key support
PRAGMA foreign_keys = ON;

-- ============================================================================
-- Specs Table
-- ============================================================================
-- Epic-like entity for grouping related tasks
CREATE TABLE IF NOT EXISTS specs (
    id TEXT PRIMARY KEY,                                    -- Format: sp-xxxx
    title TEXT NOT NULL,
    description TEXT,
    manual_status TEXT                                      -- Only 'cancelled' or NULL
        CHECK (manual_status IS NULL OR manual_status = 'cancelled'),
    created_at TEXT NOT NULL,                               -- ISO 8601 timestamp
    updated_at TEXT NOT NULL                                -- ISO 8601 timestamp
);

-- ============================================================================
-- Spec Dependencies Table
-- ============================================================================
-- Tracks spec dependencies (child depends on parent)
CREATE TABLE IF NOT EXISTS spec_dependencies (
    child_id TEXT NOT NULL,
    parent_id TEXT NOT NULL,
    PRIMARY KEY (child_id, parent_id),
    CHECK (child_id != parent_id),                          -- Prevent self-dependency
    FOREIGN KEY (child_id) REFERENCES specs(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES specs(id) ON DELETE CASCADE
);

-- ============================================================================
-- Add spec_id to tasks
-- ============================================================================
-- Link tasks to specs (tasks can optionally belong to a spec)
ALTER TABLE tasks ADD COLUMN spec_id TEXT REFERENCES specs(id) ON DELETE SET NULL;

-- ============================================================================
-- Indexes
-- ============================================================================
-- Spec indexes
CREATE INDEX IF NOT EXISTS idx_specs_manual_status ON specs(manual_status);

-- Spec dependencies index for reverse lookups
CREATE INDEX IF NOT EXISTS idx_spec_deps_parent ON spec_dependencies(parent_id);

-- Tasks by spec
CREATE INDEX IF NOT EXISTS idx_tasks_spec ON tasks(spec_id);

-- ============================================================================
-- Record Migration
-- ============================================================================
INSERT INTO _migrations (version, applied_at) VALUES (2, datetime('now'));
