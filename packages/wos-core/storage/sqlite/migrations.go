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

	if _, err := tx.ExecContext(ctx, item.sql); err != nil {
		return mapSQLError(fmt.Sprintf("apply sqlite migration %d", item.version), err)
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
