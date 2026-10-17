package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultIdempotencyRetention = 7 * 24 * time.Hour
const DefaultBusyTimeout = 5 * time.Second

type Options struct {
	MigrateOnOpen        bool
	BusyTimeout          time.Duration
	IdempotencyRetention time.Duration
	MaxOpenConnections   int
	Schema               string
	// TenantID binds an isolated schema to one Tenant. TenantName is used only
	// when initializing its singleton metadata row. TenantLease is supplied by
	// the catalog-backed runtime and fences every pooled database connection.
	TenantID    domain.ID
	TenantName  string
	TenantLease *TenantLease
}
type Store struct {
	db                   *sql.DB
	path                 string
	tenantID             domain.ID
	idempotencyRetention time.Duration
}

func Open(dsn string, options Options) (*Store, error) {
	if options.Schema != "" && !regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`).MatchString(options.Schema) {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "invalid PostgreSQL schema")
	}
	if !options.TenantID.IsZero() {
		if err := options.TenantID.Validate(); err != nil {
			return nil, err
		}
		if options.Schema == "" || options.Schema == "public" || strings.HasPrefix(options.Schema, "pg_") {
			return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "a bound Tenant requires an explicit isolated PostgreSQL schema")
		}
	}
	if options.TenantLease != nil {
		p := options.TenantLease.Placement()
		if options.TenantID.IsZero() || p.TenantID != options.TenantID || p.Schema != options.Schema {
			return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "Tenant runtime lease does not match the requested schema and Tenant")
		}
		if !options.TenantLease.initializing && options.MigrateOnOpen {
			return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "active Tenant runtimes cannot migrate schemas; suspend the Tenant and run its explicit migration operation")
		}
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "invalid PostgreSQL configuration")
	}
	if options.BusyTimeout <= 0 {
		options.BusyTimeout = DefaultBusyTimeout
	}
	if options.IdempotencyRetention <= 0 {
		options.IdempotencyRetention = DefaultIdempotencyRetention
	}
	if options.MaxOpenConnections <= 0 {
		options.MaxOpenConnections = 16
	}
	cfg.RuntimeParams["lock_timeout"] = strconv.FormatInt(options.BusyTimeout.Milliseconds(), 10)
	cfg.RuntimeParams["statement_timeout"] = "30000"
	// Migrations execute multiple DDL statements. This pinned pgx version supports parameter-safe simple protocol.
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	if options.Schema != "" {
		// No public or user-schema fallback: a removed/missing Tenant schema
		// must cause failure rather than resolve another installation's tables.
		cfg.RuntimeParams["search_path"] = options.Schema
	}
	var dbOptions []stdlib.OptionOpenDB
	if options.TenantLease != nil {
		dbOptions = append(dbOptions, stdlib.OptionAfterConnect(options.TenantLease.GuardConnection))
	}
	db := stdlib.OpenDB(*cfg, dbOptions...)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect PostgreSQL: %w", err)
	}
	if options.Schema != "" && options.MigrateOnOpen && options.TenantLease == nil {
		if _, err = db.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS `+options.Schema); err != nil {
			db.Close()
			return nil, err
		}
	}
	db.SetMaxOpenConns(options.MaxOpenConnections)
	db.SetMaxIdleConns(options.MaxOpenConnections)
	db.SetConnMaxLifetime(30 * time.Minute)
	store := &Store{db: db, path: options.Schema, tenantID: options.TenantID, idempotencyRetention: options.IdempotencyRetention}
	if options.MigrateOnOpen {
		if err = store.Migrate(ctx); err != nil {
			db.Close()
			return nil, err
		}
	}
	if !options.TenantID.IsZero() {
		var bindingErr error
		if options.TenantLease != nil && !options.TenantLease.initializing {
			bindingErr = store.validateTenantMigrations(ctx)
			if bindingErr == nil {
				bindingErr = store.verifyTenantBinding(ctx, options.TenantID)
			}
		} else {
			bindingErr = store.bindTenant(ctx, options.TenantID, options.TenantName)
		}
		if bindingErr != nil {
			_ = db.Close()
			return nil, bindingErr
		}
	}
	return store, nil
}
func mapSQLError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.NewError(domain.ErrorCodeNotFound, operation)
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return domain.NewError(domain.ErrorCodeAlreadyExists, operation)
		case "23503", "23514", "23502":
			return domain.NewError(domain.ErrorCodeInvalidArgument, operation)
		case "40001", "40P01", "55P03":
			return domain.WrapError(domain.ErrorCodeTransactionConflict, "transaction conflict; retry the same intent and idempotency key", err)
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}
func rebind(query string) string {
	n := 0
	var b strings.Builder
	quoted := false
	for i := 0; i < len(query); i++ {
		c := query[i]
		if c == '\'' {
			quoted = !quoted
		}
		if c == '?' && !quoted {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
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
	return s.begin(ctx, false)
}

func (s *Store) BeginReadSnapshot(ctx context.Context) (ports.UnitOfWork, error) {
	return s.begin(ctx, true)
}

func (s *Store) begin(ctx context.Context, readSnapshot bool) (ports.UnitOfWork, error) {
	if s == nil || s.db == nil {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "postgres store is not open")
	}
	options := &sql.TxOptions{Isolation: sql.LevelSerializable}
	if readSnapshot {
		// Shared row security barriers require a read-write transaction; all
		// domain access remains observational and Commit is prohibited.
		options.Isolation = sql.LevelRepeatableRead
	}
	tx, err := s.db.BeginTx(ctx, options)
	if err != nil {
		return nil, mapSQLError("begin postgres transaction", err)
	}
	uow := &unitOfWork{
		readSnapshot:         readSnapshot,
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
	readSnapshot         bool
	ownedStateMu         sync.Mutex
	ownedState           map[domain.EntityRef]ownedStateSnapshot
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
	if u.readSnapshot {
		return domain.NewError(domain.ErrorCodeInvalidTransition, "read snapshots must be rolled back, never committed")
	}
	if u.closed {
		return domain.NewError(domain.ErrorCodeInvalidTransition, "transaction already closed")
	}
	u.closed = true
	if err := u.tx.Commit(); err != nil {
		if u.ctx != nil && u.ctx.Err() != nil {
			return u.ctx.Err()
		}
		return mapSQLError("commit postgres transaction", err)
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
