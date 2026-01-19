// Package storage provides database schema migration functionality for Airyra.
package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migration represents a single database migration.
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// RunMigrations executes all pending migrations on the database.
// It reads migration files from the embedded filesystem, determines
// which migrations need to be applied based on the current version,
// and executes each pending migration in a transaction.
func RunMigrations(db *sql.DB) error {
	currentVersion, err := GetCurrentVersion(db)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	// Apply pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		if err := applyMigration(db, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w",
				migration.Version, migration.Name, err)
		}
	}

	return nil
}

// GetCurrentVersion returns the current schema version from the _migrations table.
// If the table doesn't exist, it returns 0 (fresh database).
func GetCurrentVersion(db *sql.DB) (int, error) {
	// Check if _migrations table exists
	var tableName string
	err := db.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name='_migrations'
	`).Scan(&tableName)

	if err == sql.ErrNoRows {
		// Table doesn't exist, return version 0
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to check for _migrations table: %w", err)
	}

	// Get the maximum version
	var version sql.NullInt64
	err = db.QueryRow("SELECT MAX(version) FROM _migrations").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to query version: %w", err)
	}

	if !version.Valid {
		// Table exists but is empty
		return 0, nil
	}

	return int(version.Int64), nil
}

// loadMigrations reads all migration files from the embedded filesystem.
func loadMigrations() ([]Migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		version, err := extractVersion(name)
		if err != nil {
			return nil, fmt.Errorf("failed to extract version from %s: %w", name, err)
		}

		content, err := migrationsFS.ReadFile(path.Join("migrations", name))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", name, err)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		})
	}

	return migrations, nil
}

// extractVersion parses the version number from a migration filename.
// Expected format: NNN_description.sql (e.g., 001_initial_schema.sql -> 1)
func extractVersion(filename string) (int, error) {
	// Remove .sql extension
	name := strings.TrimSuffix(filename, ".sql")

	// Split by underscore to get version prefix
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid migration filename format: %s", filename)
	}

	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid version number in filename %s: %w", filename, err)
	}

	return version, nil
}

// applyMigration executes a single migration within a transaction.
func applyMigration(db *sql.DB, migration Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute the migration SQL
	// The migration file already handles inserting into _migrations table
	if _, err := tx.Exec(migration.SQL); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("migration failed: %v (rollback also failed: %w)", err, rbErr)
		}
		return fmt.Errorf("migration SQL execution failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}
