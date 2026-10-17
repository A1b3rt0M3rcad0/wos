// Package mobility provides explicit operator-controlled installation moves.
// It never runs in the server, manages private keys, or transfers a Tenant.
package mobility

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	wossqlite "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// Only fixed operator messages may be returned verbatim by the command. Driver
// errors remain private because connection diagnostics can contain credentials.
type operationRefusal string

func (e operationRefusal) Error() string { return string(e) }
func refuse(message string) error        { return operationRefusal(message) }

type column struct{ Name, Kind string }
type table struct {
	Name    string
	Columns []column
}
type digest struct {
	Rows   int64  `json:"rows"`
	SHA256 string `json:"sha256"`
}
type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type identity struct {
	ServerID    string `json:"server_id"`
	Fingerprint string `json:"fingerprint"`
}
type backupManifest struct {
	Schema   int    `json:"manifest_schema"`
	Backend  string `json:"backend"`
	File     string `json:"data_file"`
	DataSHA  string `json:"data_sha256"`
	Snapshot struct {
		SchemaVersion int64  `json:"schema_version"`
		ServerID      string `json:"server_id"`
		Fingerprint   string `json:"issuer_fingerprint"`
	} `json:"snapshot"`
}

func quoted(name string) string { return `"` + name + `"` }
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func regular(path string) error {
	s, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !s.Mode().IsRegular() {
		return refuse("backup files must be regular files, not symlinks")
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	source io.Reader
}

func (r contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.source.Read(buffer)
}
func readBackup(ctx context.Context, directory, approved string) (path string, manifest backupManifest, cleanup func(), err error) {
	cleanup = func() {}
	defer func() {
		if err != nil {
			cleanup()
		}
	}()
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(approved) {
		return "", manifest, cleanup, refuse("an independently approved --manifest-sha256 is required")
	}
	manifestPath := filepath.Join(directory, "manifest.json")
	if err = regular(manifestPath); err != nil {
		return "", manifest, cleanup, err
	}
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return "", manifest, cleanup, err
	}
	defer manifestFile.Close()
	info, err := manifestFile.Stat()
	if err != nil {
		return "", manifest, cleanup, err
	}
	if !info.Mode().IsRegular() {
		return "", manifest, cleanup, refuse("backup manifest must be a regular file")
	}
	// Decode exactly the bytes whose independently approved hash was checked.
	// Never hash a path and then reopen it for parsing.
	raw, err := io.ReadAll(io.LimitReader(contextReader{ctx: ctx, source: manifestFile}, 4*1024*1024+1))
	if err != nil {
		return "", manifest, cleanup, err
	}
	sum := sha256.Sum256(raw)
	if len(raw) > 4*1024*1024 || hex.EncodeToString(sum[:]) != approved {
		return "", manifest, cleanup, refuse("approved backup manifest SHA256 mismatch")
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return "", manifest, cleanup, err
	}
	if manifest.Schema != 1 || manifest.Backend != "sqlite" || manifest.File != "database.sqlite3" {
		return "", manifest, cleanup, refuse("migration requires a version 1 SQLite installation backup")
	}
	original := filepath.Join(directory, manifest.File)
	if err = regular(original); err != nil {
		return "", manifest, cleanup, err
	}
	source, err := os.Open(original)
	if err != nil {
		return "", manifest, cleanup, err
	}
	defer source.Close()
	info, err = source.Stat()
	if err != nil {
		return "", manifest, cleanup, err
	}
	if !info.Mode().IsRegular() {
		return "", manifest, cleanup, refuse("backup database must be a regular file")
	}
	// The approved data stream is copied once into a private directory. A file
	// replacement or adjacent WAL cannot change the snapshot opened afterwards.
	directory, err = os.MkdirTemp("", "wos-approved-sqlite-")
	if err != nil {
		return "", manifest, cleanup, err
	}
	cleanup = func() { _ = os.RemoveAll(directory) }
	path = filepath.Join(directory, "snapshot.db")
	target, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", manifest, cleanup, err
	}
	defer target.Close()
	hash := sha256.New()
	if _, err = io.Copy(io.MultiWriter(target, hash), contextReader{ctx: ctx, source: source}); err != nil {
		return "", manifest, cleanup, err
	}
	if hex.EncodeToString(hash.Sum(nil)) != manifest.DataSHA {
		return "", manifest, cleanup, refuse("backup database SHA256 mismatch")
	}
	if err = target.Sync(); err != nil {
		return "", manifest, cleanup, err
	}
	if err = target.Close(); err != nil {
		return "", manifest, cleanup, err
	}
	return path, manifest, cleanup, nil
}
func openSQLite(path string) (*sql.DB, error) { return openSQLiteMode(path, false) }
func openSQLiteMode(path string, immutable bool) (*sql.DB, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err = regular(abs); err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(abs)
	u := url.URL{Scheme: "file", Path: uriPath}
	// Match the Core driver's native Windows drive-name contract. Host
	// operations remain Linux-only; internal read fixtures can still run there.
	if len(uriPath) >= 3 && uriPath[1] == ':' && uriPath[2] == '/' {
		u.Path = ""
		u.Opaque = (&url.URL{Path: uriPath}).EscapedPath()
	}
	q := u.Query()
	q.Set("mode", "ro")
	if immutable {
		q.Set("immutable", "1")
	}
	q.Set("_txlock", "deferred")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "query_only(1)")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite3", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}
func openPostgres(dsn string) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, refuse("invalid destination PostgreSQL configuration")
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	cfg.RuntimeParams["search_path"] = "public"
	cfg.RuntimeParams["lock_timeout"] = "5000"
	db := stdlib.OpenDB(*cfg)
	db.SetMaxOpenConns(1)
	return db, nil
}
func catalog(ctx context.Context, db queryer, backend string) ([]table, error) {
	query := `SELECT name FROM sqlite_schema WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`
	if backend == "postgres" {
		query = `SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE' ORDER BY table_name`
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	var tables []table
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		if !identifierPattern.MatchString(name) {
			rows.Close()
			return nil, refuse("unsupported table identifier")
		}
		tables = append(tables, table{Name: name})
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	for i := range tables {
		query = `SELECT name,type FROM pragma_table_info('` + tables[i].Name + `') ORDER BY cid`
		if backend == "postgres" {
			query = `SELECT column_name,data_type FROM information_schema.columns WHERE table_schema='public' AND table_name='` + tables[i].Name + `' ORDER BY ordinal_position`
		}
		rows, err = db.QueryContext(ctx, query)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var c column
			if err = rows.Scan(&c.Name, &c.Kind); err != nil {
				rows.Close()
				return nil, err
			}
			if !identifierPattern.MatchString(c.Name) {
				rows.Close()
				return nil, refuse("unsupported column identifier")
			}
			switch strings.ToLower(c.Kind) {
			case "text":
				c.Kind = "text"
			case "integer", "bigint":
				c.Kind = "integer"
			case "numeric":
				c.Kind = "uint64"
			default:
				rows.Close()
				return nil, fmt.Errorf("unsupported storage type in %s.%s", tables[i].Name, c.Name)
			}
			tables[i].Columns = append(tables[i].Columns, c)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err = rows.Close(); err != nil {
			return nil, err
		}
	}
	return tables, nil
}
func sameCatalog(a, b []table) bool {
	normalize := func(tables []table) []table {
		result := make([]table, len(tables))
		for i, t := range tables {
			result[i] = table{Name: t.Name, Columns: append([]column(nil), t.Columns...)}
			for j, c := range result[i].Columns {
				if c.Kind == "uint64" {
					result[i].Columns[j].Kind = "text"
				}
			}
		}
		return result
	}
	x, _ := json.Marshal(normalize(a))
	y, _ := json.Marshal(normalize(b))
	return string(x) == string(y)
}
func selectRows(t table, backend string) string {
	var fields, order []string
	for _, c := range t.Columns {
		part := quoted(c.Name)
		if c.Kind == "uint64" {
			part += "::text"
		}
		fields = append(fields, part)
		if c.Kind == "text" || c.Kind == "uint64" {
			if backend == "postgres" {
				part += ` COLLATE "C"`
			} else {
				part += ` COLLATE BINARY`
			}
		}
		order = append(order, part+" NULLS FIRST")
	}
	return "SELECT " + strings.Join(fields, ",") + " FROM " + quoted(t.Name) + " ORDER BY " + strings.Join(order, ",")
}
func walkRows(ctx context.Context, db queryer, t table, backend string, visit func([]any) error) (digest, error) {
	rows, err := db.QueryContext(ctx, selectRows(t, backend))
	if err != nil {
		return digest{}, err
	}
	defer rows.Close()
	h := sha256.New()
	var count int64
	for rows.Next() {
		values := make([]any, len(t.Columns))
		pointers := make([]any, len(values))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err = rows.Scan(pointers...); err != nil {
			return digest{}, err
		}
		for i, value := range values {
			if value == nil {
				continue
			}
			if t.Columns[i].Kind == "text" || t.Columns[i].Kind == "uint64" {
				text, ok := value.(string)
				if !ok || !utf8.ValidString(text) {
					return digest{}, fmt.Errorf("non-text source value in %s.%s", t.Name, t.Columns[i].Name)
				}
			} else {
				if _, ok := value.(int64); !ok {
					return digest{}, fmt.Errorf("non-integer source value in %s.%s", t.Name, t.Columns[i].Name)
				}
			}
			// SQLite indexes activity sequence as exactly 20 digits; PostgreSQL
			// stores the same unsigned value as NUMERIC. Only this structural
			// encoding changes. The immutable signed payload remains verbatim.
			if backend == "sqlite" && t.Name == "work_activities" && t.Columns[i].Name == "sequence" {
				text, ok := value.(string)
				if !ok || len(text) != 20 {
					return digest{}, refuse("invalid fixed-width activity sequence")
				}
				number, e := strconv.ParseUint(text, 10, 64)
				if e != nil || number == 0 {
					return digest{}, refuse("invalid unsigned activity sequence")
				}
				values[i] = strconv.FormatUint(number, 10)
			}
		}
		encoded, err := json.Marshal(values)
		if err != nil {
			return digest{}, err
		}
		h.Write(encoded)
		h.Write([]byte{'\n'})
		count++
		if visit != nil {
			if err = visit(values); err != nil {
				return digest{}, err
			}
		}
	}
	if err = rows.Err(); err != nil {
		return digest{}, err
	}
	return digest{Rows: count, SHA256: hex.EncodeToString(h.Sum(nil))}, nil
}
func readIdentity(ctx context.Context, db queryer) (identity, error) {
	var raw string
	err := db.QueryRowContext(ctx, "SELECT state_json FROM server_protocol_identity WHERE singleton=1").Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return identity{}, nil
	}
	if err != nil {
		return identity{}, err
	}
	var result identity
	err = json.Unmarshal([]byte(raw), &result)
	return result, err
}
func validateInstallationIdentity(ctx context.Context, db queryer) error {
	var raw string
	err := db.QueryRowContext(ctx, "SELECT state_json FROM server_protocol_identity WHERE singleton=1").Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		var required bool
		err = db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM tenant_work_protocol WHERE phase IN ('signed_contracts_v2','signed_contracts_v3')) OR EXISTS(SELECT 1 FROM signed_work_contracts) OR EXISTS(SELECT 1 FROM signed_protocol_facts) OR EXISTS(SELECT 1 FROM signed_operation_results)`).Scan(&required)
		if err != nil {
			return err
		}
		if required {
			return refuse("signed state has no installation issuer identity")
		}
		return nil
	}
	if err != nil {
		return err
	}
	var id domain.ServerIdentity
	if err = json.Unmarshal([]byte(raw), &id); err != nil {
		return refuse("invalid installation public identity")
	}
	return id.Validate()
}
func validateSource(ctx context.Context, db queryer) ([]table, error) {
	var check string
	if err := db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&check); err != nil {
		return nil, err
	}
	if check != "ok" {
		return nil, refuse("SQLite integrity check failed")
	}
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return nil, err
	}
	invalid := rows.Next()
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if invalid {
		return nil, refuse("SQLite foreign-key closure check failed")
	}
	temporary, err := os.MkdirTemp("", "wos-mobility-schema-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temporary)
	path := filepath.Join(temporary, "schema.db")
	store, err := wossqlite.Open(path, wossqlite.Options{MigrateOnOpen: true})
	if err != nil {
		return nil, err
	}
	if err = store.Close(); err != nil {
		return nil, err
	}
	expected, err := openSQLite(path)
	if err != nil {
		return nil, err
	}
	defer expected.Close()
	want, err := catalog(ctx, expected, "sqlite")
	if err != nil {
		return nil, err
	}
	got, err := catalog(ctx, db, "sqlite")
	if err != nil {
		return nil, err
	}
	if !sameCatalog(got, want) {
		return nil, refuse("source schema differs from the compiled WOS schema; upgrade and back up with this version first")
	}
	readChain := func(q queryer) (string, error) {
		rows, err := q.QueryContext(ctx, "SELECT version,name,checksum FROM schema_migrations ORDER BY version")
		if err != nil {
			return "", err
		}
		defer rows.Close()
		var chain []any
		for rows.Next() {
			var version int64
			var name, sum string
			if err = rows.Scan(&version, &name, &sum); err != nil {
				return "", err
			}
			chain = append(chain, []any{version, name, sum})
		}
		raw, err := json.Marshal(chain)
		if err != nil {
			return "", err
		}
		return string(raw), rows.Err()
	}
	wanted, err := readChain(expected)
	if err != nil {
		return nil, err
	}
	actual, err := readChain(db)
	if err != nil {
		return nil, err
	}
	if wanted != actual {
		return nil, refuse("source migration chain differs from this binary")
	}
	if err = validateInstallationIdentity(ctx, db); err != nil {
		return nil, err
	}
	return got, nil
}

type migrateOptions struct {
	Backup, ManifestSHA256, DSN, ExpectedServerID, IssuerFingerprint string
	WritersStopped                                                   bool
}

func migrate(ctx context.Context, opts migrateOptions) (map[string]any, error) {
	if !opts.WritersStopped {
		return nil, refuse("migration requires --writers-stopped; stop and fence all source writers before taking the backup")
	}
	path, manifest, cleanup, err := readBackup(ctx, opts.Backup, opts.ManifestSHA256)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	if (manifest.Snapshot.ServerID != "" || opts.ExpectedServerID != "" || opts.IssuerFingerprint != "") && (opts.ExpectedServerID != manifest.Snapshot.ServerID || opts.IssuerFingerprint != manifest.Snapshot.Fingerprint) {
		return nil, refuse("signed installation migration requires the independently approved server ID and issuer fingerprint")
	}
	source, err := openSQLiteMode(path, true)
	if err != nil {
		return nil, err
	}
	defer source.Close()
	snapshot, err := source.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer snapshot.Rollback()
	tables, err := validateSource(ctx, snapshot)
	if err != nil {
		return nil, err
	}
	id, err := readIdentity(ctx, snapshot)
	if err != nil {
		return nil, err
	}
	if id.ServerID != manifest.Snapshot.ServerID || id.Fingerprint != manifest.Snapshot.Fingerprint {
		return nil, refuse("backup identity differs from its manifest")
	}
	var version int64
	if err = snapshot.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil {
		return nil, err
	}
	if version != manifest.Snapshot.SchemaVersion {
		return nil, refuse("backup schema version differs from its manifest")
	}
	if opts.DSN == "" {
		return nil, refuse("an explicit destination PostgreSQL DSN is required")
	}
	target, err := openPostgres(opts.DSN)
	if err != nil {
		return nil, err
	}
	defer target.Close()
	var objects int64
	err = target.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_toast%' AND c.relkind IN ('r','p','v','m','S','f')) + (SELECT COUNT(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema'))`).Scan(&objects)
	if err != nil {
		return nil, err
	}
	if objects != 0 {
		return nil, refuse("destination must be a new empty PostgreSQL database")
	}
	// Initialize the destination using its own canonical migration chain. Source
	// SQLite migration checksums must never masquerade as PostgreSQL checksums.
	store, err := postgres.Open(opts.DSN, postgres.Options{MigrateOnOpen: true, Schema: "public", MaxOpenConnections: 1})
	if err != nil {
		return nil, err
	}
	if err = store.Close(); err != nil {
		return nil, err
	}
	tx, err := target.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var names []string
	for _, t := range tables {
		names = append(names, quoted(t.Name))
	}
	if _, err = tx.ExecContext(ctx, "LOCK TABLE "+strings.Join(names, ",")+" IN ACCESS EXCLUSIVE MODE"); err != nil {
		return nil, err
	}
	destinationCatalog, err := catalog(ctx, tx, "postgres")
	if err != nil {
		return nil, err
	}
	if !sameCatalog(tables, destinationCatalog) {
		return nil, refuse("destination/source column schemas differ")
	}
	for _, t := range tables {
		if t.Name == "schema_migrations" {
			continue
		}
		var n int64
		if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+quoted(t.Name)).Scan(&n); err != nil {
			return nil, err
		}
		if n != 0 {
			return nil, refuse("destination was written during initialization; migration refused")
		}
	}
	type constraint struct {
		Table, Name          string
		Deferrable, Deferred bool
	}
	rows, err := tx.QueryContext(ctx, `SELECT c.relname,k.conname,k.condeferrable,k.condeferred FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND k.contype='f' ORDER BY c.relname,k.conname`)
	if err != nil {
		return nil, err
	}
	var constraints []constraint
	for rows.Next() {
		var c constraint
		if err = rows.Scan(&c.Table, &c.Name, &c.Deferrable, &c.Deferred); err != nil {
			rows.Close()
			return nil, err
		}
		constraints = append(constraints, c)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	for _, c := range constraints {
		// Only a new, exclusively locked destination is affected. Validate every
		// deferred reference and restore all original flags before committing.
		if _, err = tx.ExecContext(ctx, "ALTER TABLE "+quoted(c.Table)+" ALTER CONSTRAINT "+quoted(c.Name)+" DEFERRABLE INITIALLY DEFERRED"); err != nil {
			return nil, err
		}
	}
	expected := map[string]digest{}
	for index, t := range tables {
		if t.Name == "schema_migrations" {
			continue
		}
		var fields, args []string
		for i, c := range t.Columns {
			fields = append(fields, quoted(c.Name))
			args = append(args, fmt.Sprintf("$%d", i+1))
		}
		insert := "INSERT INTO " + quoted(t.Name) + "(" + strings.Join(fields, ",") + ") VALUES(" + strings.Join(args, ",") + ")"
		d, err := walkRows(ctx, snapshot, t, "sqlite", func(values []any) error {
			for i, c := range destinationCatalog[index].Columns {
				if c.Kind == "uint64" && values[i] != nil {
					text, ok := values[i].(string)
					if !ok {
						return refuse("non-text uint64 source")
					}
					number, e := strconv.ParseUint(text, 10, 64)
					if e != nil || number == 0 || strconv.FormatUint(number, 10) != text {
						return refuse("noncanonical uint64 source")
					}
				}
			}
			_, err := tx.ExecContext(ctx, insert, values...)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("copy %s: %w", t.Name, err)
		}
		expected[t.Name] = d
	}
	if _, err = tx.ExecContext(ctx, "SET CONSTRAINTS ALL IMMEDIATE"); err != nil {
		return nil, fmt.Errorf("destination reference closure rejected: %w", err)
	}
	for _, c := range constraints {
		flags := "NOT DEFERRABLE"
		if c.Deferrable {
			flags = "DEFERRABLE INITIALLY IMMEDIATE"
			if c.Deferred {
				flags = "DEFERRABLE INITIALLY DEFERRED"
			}
		}
		if _, err = tx.ExecContext(ctx, "ALTER TABLE "+quoted(c.Table)+" ALTER CONSTRAINT "+quoted(c.Name)+" "+flags); err != nil {
			return nil, err
		}
	}
	for _, t := range destinationCatalog {
		if t.Name == "schema_migrations" {
			continue
		}
		actual, err := walkRows(ctx, tx, t, "postgres", nil)
		if err != nil {
			return nil, err
		}
		if actual != expected[t.Name] {
			return nil, fmt.Errorf("logical row verification failed for %s", t.Name)
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]any{"operation": "migrate_sqlite_to_postgres", "completed": true, "verified": true, "scope": "whole_installation", "server_id": id.ServerID, "issuer_fingerprint": id.Fingerprint, "issuer_seed_collected": false, "source_manifest_sha256": opts.ManifestSHA256, "tables": expected, "migration_metadata": "destination canonical PostgreSQL chain retained", "source_modified": false, "routing_changed": false}, nil
}

// preflight intentionally cannot authorize or perform tenant relocation. It
// reports a coherent structural inventory and explicit missing trust contracts.
func preflight(ctx context.Context, db queryer, backend, tenant string) (map[string]any, error) {
	if tenant == "" {
		return nil, refuse("--tenant is required")
	}
	placeholder := "?"
	if backend == "postgres" {
		placeholder = "$1"
	}
	var exists int
	if err := db.QueryRowContext(ctx, "SELECT 1 FROM tenants WHERE id="+placeholder, tenant).Scan(&exists); err != nil {
		return nil, refuse("Tenant is absent from this database snapshot")
	}
	tables, err := catalog(ctx, db, backend)
	if err != nil {
		return nil, err
	}
	id, err := readIdentity(ctx, db)
	if err != nil {
		return nil, err
	}
	counts := map[string]int64{}
	var indirect []string
	for _, t := range tables {
		scoped := false
		for _, c := range t.Columns {
			if c.Name == "tenant_id" {
				scoped = true
				break
			}
		}
		if !scoped {
			indirect = append(indirect, t.Name)
			continue
		}
		var count int64
		if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+quoted(t.Name)+" WHERE tenant_id="+placeholder, tenant).Scan(&count); err != nil {
			return nil, err
		}
		counts[t.Name] = count
	}
	sort.Strings(indirect)
	inspection, err := tenantInspection(ctx, db, tables, backend, tenant)
	if err != nil {
		return nil, err
	}
	return map[string]any{"operation": "tenant_transfer_preflight", "read_only": true, "ready_for_transfer": false, "tenant_id": tenant, "server_id": id.ServerID, "issuer_fingerprint": id.Fingerprint, "scoped_table_rows": counts, "tables_requiring_global_or_indirect_reference_closure": indirect, "inspection": inspection, "issuer_seed_collected": false, "blockers": []string{"tenant_transfer_protocol_not_implemented", "source_write_fence_and_cutover_not_implemented", "historical_issuer_trust_and_destination_enrollment_required", "global_principal_and_indirect_reference_closure_required", "durable_receipt_and_pending_delivery_reconciliation_required"}}, nil
}

func run(ctx context.Context, args []string, out io.Writer) error {
	return runCommand(ctx, args, out, nil)
}

func runCommand(ctx context.Context, args []string, out io.Writer, checkHost func() error) error {
	if len(args) == 0 {
		return refuse("use migrate-sqlite or tenant-preflight")
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		_, err := fmt.Fprintln(out, "Usage: wos storage migrate-sqlite [flags]\n       wos storage tenant-preflight [flags]\nUse COMMAND --help for flags. All database URIs come from named environment variables.")
		return err
	}
	p := flag.NewFlagSet(args[0], flag.ContinueOnError)
	p.SetOutput(io.Discard)
	dsnEnv := p.String("dsn-env", "WOS_MOBILITY_POSTGRES_DSN", "name of a PostgreSQL DSN environment variable")
	var result map[string]any
	var err error
	switch args[0] {
	case "migrate-sqlite", "migrate-sqlite-to-postgres":
		var opts migrateOptions
		p.StringVar(&opts.Backup, "backup", "", "approved SQLite backup directory")
		p.StringVar(&opts.ManifestSHA256, "manifest-sha256", "", "independently approved manifest SHA256")
		p.StringVar(&opts.ExpectedServerID, "expected-server-id", "", "approved installation identity")
		p.StringVar(&opts.IssuerFingerprint, "issuer-fingerprint", "", "approved current public issuer fingerprint")
		p.BoolVar(&opts.WritersStopped, "writers-stopped", false, "operator confirms all source writers stopped and fenced")
		if err = p.Parse(args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				p.SetOutput(out)
				p.Usage()
				return nil
			}
			return refuse("invalid migration arguments")
		}
		if p.NArg() != 0 {
			return refuse("unexpected migration arguments")
		}
		if checkHost != nil {
			if err = checkHost(); err != nil {
				return err
			}
		}
		opts.DSN = os.Getenv(*dsnEnv)
		if opts.DSN == "" {
			return refuse("named destination PostgreSQL DSN environment variable is required")
		}
		result, err = migrate(ctx, opts)
	case "tenant-preflight":
		path := p.String("sqlite", "", "existing SQLite source file; omit for PostgreSQL")
		tenant := p.String("tenant", "", "Tenant UUID")
		if err = p.Parse(args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				p.SetOutput(out)
				p.Usage()
				return nil
			}
			return refuse("invalid preflight arguments")
		}
		if p.NArg() != 0 {
			return refuse("unexpected preflight arguments")
		}
		if checkHost != nil {
			if err = checkHost(); err != nil {
				return err
			}
		}
		backend := "postgres"
		var db *sql.DB
		if *path != "" {
			backend = "sqlite"
			db, err = openSQLite(*path)
		} else {
			dsn := os.Getenv(*dsnEnv)
			if dsn == "" {
				return refuse("source database is required")
			}
			db, err = openPostgres(dsn)
		}
		if err != nil {
			return err
		}
		defer db.Close()
		isolation := sql.LevelRepeatableRead
		if backend == "sqlite" {
			isolation = sql.LevelDefault
		}
		tx, e := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: isolation})
		if e != nil {
			return e
		}
		defer tx.Rollback()
		result, err = preflight(ctx, tx, backend, *tenant)
	default:
		return refuse("unknown operation; use migrate-sqlite or tenant-preflight")
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(result)
}

// Run executes an offline operator command without opening the WOS server,
// bootstrapping authentication, or loading client workspace configuration.
// Actual storage operations are qualified only on Linux server hosts. Help is
// available on every platform. Errors exclude private database diagnostics.
func Run(parent context.Context, args []string, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(parent, 30*time.Minute)
	defer cancel()
	checkHost := func() error {
		if runtime.GOOS != "linux" {
			return refuse("storage operator commands are supported only on Linux; run this command on the WOS server host")
		}
		return nil
	}
	if err := runCommand(ctx, args, stdout, checkHost); err != nil {
		// Database driver errors can contain connection details. The command's
		// public failure is intentionally generic; tests inspect internal errors.
		message := "operation failed; verify approved backup, identity, schema, new destination and connectivity; no routing was changed"
		var refusal operationRefusal
		if errors.As(err, &refusal) {
			message = refusal.Error()
		}
		_ = json.NewEncoder(stderr).Encode(map[string]any{"completed": false, "error": message})
		return 1
	}
	return 0
}
