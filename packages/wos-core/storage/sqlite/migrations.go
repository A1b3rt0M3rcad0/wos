package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type migration struct {
	version  int64
	name     string
	sql      string
	checksum string
}

func (s *Store) Migrate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "sqlite store is not open")
	}
	if _, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT PRIMARY KEY,
    name TEXT NOT NULL,
    checksum TEXT NOT NULL,
    applied_at BIGINT NOT NULL
)`); err != nil {
		return mapSQLError("create schema_migrations", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	for _, item := range migrations {
		if err := s.applyMigration(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SchemaVersion(ctx context.Context) (int64, error) {
	var version int64
	err := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, mapSQLError("read schema version", err)
	}
	return version, nil
}

func (s *Store) applyMigration(ctx context.Context, item migration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return mapSQLError("begin migration transaction", err)
	}
	defer tx.Rollback()

	// Check under the same migration transaction (and PostgreSQL advisory
	// lock) before inspecting or applying any compiled migration.
	latest, err := LatestSchemaVersion()
	if err != nil {
		return err
	}
	var current int64
	if err = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&current); err != nil {
		return mapSQLError("read schema compatibility", err)
	}
	if current > latest {
		return domain.NewError(domain.ErrorCodeInvalidConfig, fmt.Sprintf("database schema %d is newer than this binary supports (%d); use a compatible binary", current, latest))
	}
	var existing string
	err = tx.QueryRowContext(ctx,
		"SELECT checksum FROM schema_migrations WHERE version = ?",
		item.version,
	).Scan(&existing)
	switch {
	case err == nil:
		if existing != item.checksum {
			return domain.NewError(
				domain.ErrorCodeInvalidConfig,
				fmt.Sprintf("sqlite migration %d checksum mismatch", item.version),
			)
		}
		return nil
	case !errors.Is(err, sql.ErrNoRows):
		return mapSQLError("read migration checksum", err)
	}

	var migrationTable string
	if item.version == 35 {
		var orphanCount int64
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM work_items WHERE objective_id IS NULL").Scan(&orphanCount); err != nil {
			return mapSQLError("check Task Objective ownership before migration 35", err)
		}
		if orphanCount != 0 {
			return fmt.Errorf("migration 35 requires zero Tasks without an Objective; found %d. Apply reviewed ownership mappings with tools/acceptance/issue_ownership_backfill.py, then retry", orphanCount)
		}
		migrationTable = "work_items"
	}
	if item.version == 36 {
		var orphanCount int64
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM issues WHERE objective_id IS NULL").Scan(&orphanCount); err != nil {
			return mapSQLError("check Issue Objective ownership before migration 36", err)
		}
		if orphanCount != 0 {
			return fmt.Errorf("migration 36 requires zero Issues without an Objective; found %d. Apply reviewed ownership mappings with tools/acceptance/issue_ownership_backfill.py, then retry", orphanCount)
		}
		migrationTable = "issues"
	}
	var migrationTableObjects []string
	if migrationTable != "" {
		migrationTableObjects, err = snapshotTableSchemaObjects(ctx, tx, migrationTable)
		if err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, item.sql); err != nil {
		return mapSQLError(fmt.Sprintf("apply sqlite migration %d", item.version), err)
	}
	if migrationTable != "" {
		for _, statement := range migrationTableObjects {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return mapSQLError(fmt.Sprintf("restore explicit %s index or trigger after migration %d", migrationTable, item.version), err)
			}
		}
		if err := verifyMigrationForeignKeys(ctx, tx, item.version); err != nil {
			return err
		}
		if err := releaseMigrationForeignKeyDeferral(ctx, tx, item.version); err != nil {
			return err
		}
	}
	if item.version == 29 {
		if err := backfillExternalContinuityKeys(ctx, tx); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO schema_migrations(version, name, checksum, applied_at) VALUES (?, ?, ?, ?)",
		item.version,
		item.name,
		item.checksum,
		time.Now().UTC().UnixMicro(),
	); err != nil {
		return mapSQLError("record sqlite migration", err)
	}
	if err := tx.Commit(); err != nil {
		return mapSQLError("commit sqlite migration", err)
	}
	return nil
}

// SQLITE_TABLE_SCHEMA_OBJECTS_BEGIN
func snapshotTableSchemaObjects(ctx context.Context, tx *sql.Tx, table string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT sql
FROM sqlite_master
WHERE tbl_name = ?
  AND type IN ('index', 'trigger')
  AND sql IS NOT NULL
ORDER BY type, name`, table)
	if err != nil {
		return nil, mapSQLError("snapshot migration table indexes and triggers", err)
	}
	defer rows.Close()
	var statements []string
	for rows.Next() {
		var statement string
		if err := rows.Scan(&statement); err != nil {
			return nil, mapSQLError("read migration table index or trigger definition", err)
		}
		statements = append(statements, statement)
	}
	if err := rows.Err(); err != nil {
		return nil, mapSQLError("iterate migration table index or trigger definitions", err)
	}
	return statements, nil
}

func verifyMigrationForeignKeys(ctx context.Context, tx *sql.Tx, version int64) error {
	rows, err := tx.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return mapSQLError(fmt.Sprintf("verify foreign keys after migration %d", version), err)
	}
	defer rows.Close()
	if rows.Next() {
		var table, parent string
		var rowID sql.NullInt64
		var foreignKey int64
		if err := rows.Scan(&table, &rowID, &parent, &foreignKey); err != nil {
			return mapSQLError(fmt.Sprintf("read foreign-key violation after migration %d", version), err)
		}
		return fmt.Errorf("sqlite migration %d left a foreign-key violation in %s (row %v references %s, constraint %d)", version, table, rowID, parent, foreignKey)
	}
	if err := rows.Err(); err != nil {
		return mapSQLError(fmt.Sprintf("iterate foreign-key check after migration %d", version), err)
	}
	return nil
}

func releaseMigrationForeignKeyDeferral(ctx context.Context, tx *sql.Tx, version int64) error {
	// SQLite keeps a deferred-FK violation counter for DROP TABLE even when the
	// replacement has restored every referenced key. foreign_key_check above
	// verifies the complete transactional database state; turning deferral off
	// now clears that stale counter while foreign_keys itself remains enabled.
	if _, err := tx.ExecContext(ctx, "PRAGMA defer_foreign_keys = OFF"); err != nil {
		return mapSQLError(fmt.Sprintf("finish deferred foreign-key checks after migration %d", version), err)
	}
	return nil
}

// SQLITE_TABLE_SCHEMA_OBJECTS_END

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, err
	}
	result := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), ".sql")
		parts := strings.SplitN(base, "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid migration filename %q", entry.Name())
		}
		version, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("invalid migration version in %q", entry.Name())
		}
		content, err := migrationFiles.ReadFile(path.Join("migrations", entry.Name()))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(content)
		result = append(result, migration{
			version:  version,
			name:     entry.Name(),
			sql:      string(content),
			checksum: hex.EncodeToString(sum[:]),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].version < result[j].version })
	for i := 1; i < len(result); i++ {
		if result[i].version == result[i-1].version {
			return nil, fmt.Errorf("duplicate migration version %d", result[i].version)
		}
	}
	return result, nil
}

func LatestSchemaVersion() (int64, error) {
	m, err := loadMigrations()
	if err != nil {
		return 0, err
	}
	return m[len(m)-1].version, nil
}
