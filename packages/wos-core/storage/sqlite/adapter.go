package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	_ "github.com/ncruces/go-sqlite3/driver"
)

const (
	DefaultBusyTimeout          = 5 * time.Second
	DefaultStartupTimeout       = 30 * time.Second
	DefaultIdempotencyRetention = 7 * 24 * time.Hour
)

type Options struct {
	// StartupTimeout bounds connection setup and migrations independently of writer contention.
	StartupTimeout       time.Duration
	BusyTimeout          time.Duration
	IdempotencyRetention time.Duration
	MigrateOnOpen        bool
}

type Store struct {
	db                   *sql.DB
	path                 string
	busyTimeout          time.Duration
	idempotencyRetention time.Duration
}

type ConnectionSettings struct {
	ForeignKeys bool
	JournalMode string
	BusyTimeout time.Duration
}

func Open(path string, opts Options) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "sqlite path is required")
	}
	if opts.StartupTimeout < 0 {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "sqlite startup timeout must be positive")
	}
	if opts.StartupTimeout == 0 {
		opts.StartupTimeout = DefaultStartupTimeout
		// Preserve the older bootstrap allowance of an explicitly longer busy timeout.
		if opts.BusyTimeout > opts.StartupTimeout {
			opts.StartupTimeout = opts.BusyTimeout
		}
	}
	if opts.BusyTimeout <= 0 {
		opts.BusyTimeout = DefaultBusyTimeout
	}
	if opts.IdempotencyRetention <= 0 {
		opts.IdempotencyRetention = DefaultIdempotencyRetention
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, domain.WrapError(domain.ErrorCodeInvalidConfig, "resolve sqlite path", err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, domain.WrapError(domain.ErrorCodeInvalidConfig, "create sqlite directory", err)
	}

	dsn := sqliteDSN(abs, opts.BusyTimeout)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, domain.WrapError(domain.ErrorCodeInvalidConfig, "open sqlite database", err)
	}

	// SQLite serializes writers. Keep one writer connection so every command
	// gets the same explicit per-connection PRAGMA contract and an immediate
	// transaction without pretending parallel writers can make progress.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	store := &Store{
		db:                   db,
		path:                 abs,
		busyTimeout:          opts.BusyTimeout,
		idempotencyRetention: opts.IdempotencyRetention,
	}
	ctx, cancel := context.WithTimeout(context.Background(), opts.StartupTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, domain.WrapError(domain.ErrorCodeInvalidConfig, "ping sqlite database", err)
	}
	settings, err := store.ConnectionSettings(ctx)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if !settings.ForeignKeys {
		_ = db.Close()
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "sqlite foreign_keys pragma is disabled")
	}
	if !strings.EqualFold(settings.JournalMode, "wal") {
		_ = db.Close()
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "sqlite WAL mode was not enabled")
	}

	if opts.MigrateOnOpen {
		if err := store.Migrate(ctx); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return store, nil
}

func sqliteDSN(path string, busyTimeout time.Duration) string {
	uriPath := filepath.ToSlash(path)
	u := &url.URL{Scheme: "file", Path: uriPath}
	// This Go VFS receives SQLite's parsed name directly. A Windows drive
	// must remain C:/..., rather than an authority or the invalid /C:/... .
	// Opaque file:C:/... preserves that native name and still escapes bytes.
	if len(uriPath) >= 3 && uriPath[1] == ':' && uriPath[2] == '/' {
		u.Path = ""
		u.Opaque = (&url.URL{Path: uriPath}).EscapedPath()
	}
	q := u.Query()
	q.Set("_txlock", "immediate")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", busyTimeout.Milliseconds()))
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(NORMAL)")
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *Store) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	if s == nil || s.db == nil {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "sqlite store is not open")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, mapSQLError("begin sqlite transaction", err)
	}
	uow := &unitOfWork{
		tx:                   tx,
		ctx:                  ctx,
		idempotencyRetention: s.idempotencyRetention,
	}
	uow.outcomes = outcomeRepository{uow: uow}
	uow.objectives = objectiveRepository{uow: uow}
	uow.workItems = workItemRepository{uow: uow}
	uow.relations = relationRepository{uow: uow}
	uow.issues = issueRepository{uow: uow}
	uow.blockers = blockerRepository{uow: uow}
	uow.artifacts = artifactRepository{uow: uow}
	uow.evidence = evidenceRepository{uow: uow}
	uow.evidenceLinks = evidenceLinkRepository{uow: uow}
	uow.decisions = decisionRepository{uow: uow}
	uow.roadmaps = roadmapRepository{uow: uow}
	uow.roadmapActivations = roadmapActivationStore{uow: uow}
	uow.coordination = coordinationStore{uow: uow}
	uow.events = eventLog{uow: uow}
	uow.idempotency = idempotencyStore{uow: uow}
	return uow, nil
}

type unitOfWork struct {
	guardDuration        time.Duration
	tx                   *sql.Tx
	ctx                  context.Context
	closed               bool
	idempotencyRetention time.Duration

	outcomes           outcomeRepository
	objectives         objectiveRepository
	workItems          workItemRepository
	relations          relationRepository
	issues             issueRepository
	blockers           blockerRepository
	artifacts          artifactRepository
	evidence           evidenceRepository
	evidenceLinks      evidenceLinkRepository
	decisions          decisionRepository
	roadmaps           roadmapRepository
	roadmapActivations roadmapActivationStore
	coordination       coordinationStore
	events             eventLog
	idempotency        idempotencyStore
}

func (u *unitOfWork) Outcomes() ports.OutcomeRepository           { return u.outcomes }
func (u *unitOfWork) Objectives() ports.ObjectiveRepository       { return u.objectives }
func (u *unitOfWork) WorkItems() ports.WorkItemRepository         { return u.workItems }
func (u *unitOfWork) Relations() ports.RelationRepository         { return u.relations }
func (u *unitOfWork) Issues() ports.IssueRepository               { return u.issues }
func (u *unitOfWork) Blockers() ports.BlockerRepository           { return u.blockers }
func (u *unitOfWork) Artifacts() ports.ArtifactRepository         { return u.artifacts }
func (u *unitOfWork) Evidence() ports.EvidenceRepository          { return u.evidence }
func (u *unitOfWork) EvidenceLinks() ports.EvidenceLinkRepository { return u.evidenceLinks }
func (u *unitOfWork) Decisions() ports.DecisionRepository         { return u.decisions }
func (u *unitOfWork) Roadmaps() ports.RoadmapRepository           { return u.roadmaps }
func (u *unitOfWork) RoadmapActivations() ports.RoadmapActivationStore {
	return u.roadmapActivations
}
func (u *unitOfWork) Coordination() ports.CoordinationStore { return u.coordination }
func (u *unitOfWork) Events() ports.DomainEventLog          { return u.events }
func (u *unitOfWork) Idempotency() ports.IdempotencyStore   { return u.idempotency }

func (u *unitOfWork) Commit() error {
	if u.closed {
		return domain.NewError(domain.ErrorCodeInvalidTransition, "transaction already closed")
	}
	u.closed = true
	if err := u.tx.Commit(); err != nil {
		if u.ctx != nil && u.ctx.Err() != nil {
			return u.ctx.Err()
		}
		return mapSQLError("commit sqlite transaction", err)
	}
	return nil
}

func (u *unitOfWork) Rollback() error {
	if u.closed {
		return nil
	}
	u.closed = true
	err := u.tx.Rollback()
	if errors.Is(err, sql.ErrTxDone) {
		return nil
	}
	return err
}

func (u *unitOfWork) ensureOpen() error {
	if u.closed {
		return domain.NewError(domain.ErrorCodeInvalidTransition, "transaction is closed")
	}
	if u.ctx != nil {
		if err := u.ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ConnectionSettings(ctx context.Context) (ConnectionSettings, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return ConnectionSettings{}, mapSQLError("open sqlite connection", err)
	}
	defer conn.Close()

	var foreignKeys int
	if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		return ConnectionSettings{}, mapSQLError("read sqlite foreign_keys pragma", err)
	}
	var journalMode string
	if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return ConnectionSettings{}, mapSQLError("read sqlite journal_mode pragma", err)
	}
	var timeoutMS int64
	if err := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&timeoutMS); err != nil {
		return ConnectionSettings{}, mapSQLError("read sqlite busy_timeout pragma", err)
	}
	return ConnectionSettings{
		ForeignKeys: foreignKeys == 1,
		JournalMode: journalMode,
		BusyTimeout: time.Duration(timeoutMS) * time.Millisecond,
	}, nil
}

func (s *Store) Backup(ctx context.Context, destination string) error {
	if s == nil || s.db == nil {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "sqlite store is not open")
	}
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "backup destination is required")
	}
	abs, err := filepath.Abs(destination)
	if err != nil {
		return domain.WrapError(domain.ErrorCodeInvalidArgument, "resolve backup destination", err)
	}
	if _, err := os.Stat(abs); err == nil {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "backup destination already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}

	conn, err := s.db.Conn(ctx)
	if err != nil {
		return mapSQLError("open sqlite backup connection", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "VACUUM main INTO ?", abs); err != nil {
		return mapSQLError("create sqlite backup", err)
	}
	return nil
}

func RestoreFile(source, destination string) error {
	source = strings.TrimSpace(source)
	destination = strings.TrimSpace(destination)
	if source == "" || destination == "" {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "restore source and destination are required")
	}
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "restore source must be a regular file")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	dst, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return domain.NewError(domain.ErrorCodeAlreadyExists, "restore destination already exists")
		}
		return err
	}
	ok := false
	defer func() {
		_ = dst.Close()
		if !ok {
			_ = os.Remove(destination)
		}
	}()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	if err := dst.Sync(); err != nil {
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func mapSQLError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.NewError(domain.ErrorCodeNotFound, operation)
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unique constraint failed"),
		strings.Contains(message, "primary key"):
		return domain.WrapError(domain.ErrorCodeAlreadyExists, operation, err)
	case strings.Contains(message, "foreign key constraint failed"),
		strings.Contains(message, "check constraint failed"):
		return domain.WrapError(domain.ErrorCodeInvalidArgument, operation, err)
	default:
		return fmt.Errorf("%s: %w", operation, err)
	}
}

func nullableID(value *domain.ID) any {
	if value == nil {
		return nil
	}
	return value.String()
}

func nullableVersion(value *domain.Version) any {
	if value == nil {
		return nil
	}
	return int64(*value)
}

func (u *unitOfWork) GuardDuration() time.Duration { return u.guardDuration }
