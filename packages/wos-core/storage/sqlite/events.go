package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type eventLog struct{ uow *unitOfWork }
type idempotencyStore struct{ uow *unitOfWork }

func (l eventLog) Append(ctx context.Context, events []domain.DomainEvent) error {
	if err := l.uow.ensureOpen(); err != nil {
		return err
	}
	for i, event := range events {
		if err := event.Validate(); err != nil {
			return err
		}
		if i > 0 {
			previous := events[i-1]
			if event.Scope() != previous.Scope() ||
				event.OutcomeRevision != previous.OutcomeRevision ||
				event.EventIndex != previous.EventIndex+1 {
				return domain.NewError(
					domain.ErrorCodeInvalidEvent,
					"events in one append must share scope/revision and use contiguous event_index values",
				)
			}
		}
		if err := ensurePrincipal(ctx, l.uow.tx, event.PrincipalID); err != nil {
			return err
		}
		actorJSON, err := marshalJSON(event.Actor)
		if err != nil {
			return err
		}
		executionJSON, err := marshalJSON(event.ExecutionContext)
		if err != nil {
			return err
		}
		_, err = l.uow.tx.ExecContext(ctx, `
INSERT INTO domain_events (
    id, namespace_id, outcome_id, event_type, schema_version,
    outcome_revision, event_index, aggregate_id, aggregate_kind,
    aggregate_version_before, aggregate_version_after, command_id,
    principal_id, actor_json, recorded_at, correlation_id, causation_id,
    execution_context_json, payload_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			event.EventID.String(),
			event.NamespaceID.String(),
			event.OutcomeID.String(),
			event.EventType,
			int64(event.SchemaVersion),
			int64(event.OutcomeRevision),
			int64(event.EventIndex),
			event.AggregateRef.ID.String(),
			event.AggregateRef.Kind.String(),
			nullableVersion(event.AggregateVersionBefore),
			nullableVersion(event.AggregateVersionAfter),
			event.CommandID.String(),
			event.PrincipalID,
			actorJSON,
			encodeTime(event.RecordedAt),
			nullIfEmpty(event.CorrelationID),
			nullableID(event.CausationID),
			executionJSON,
			string(event.Payload),
		)
		if err != nil {
			return mapSQLError("append domain event", err)
		}
	}
	return nil
}

func (s idempotencyStore) Reserve(
	ctx context.Context,
	identity domain.IdempotencyIdentity,
	fingerprint string,
) (domain.IdempotencyReservation, error) {
	if err := s.uow.ensureOpen(); err != nil {
		return domain.IdempotencyReservation{}, err
	}
	if err := identity.Validate(); err != nil {
		return domain.IdempotencyReservation{}, err
	}
	if fingerprint == "" {
		return domain.IdempotencyReservation{}, domain.NewError(
			domain.ErrorCodeIdempotencyState,
			"fingerprint is required",
		)
	}
	if err := ensureNamespace(ctx, s.uow.tx, identity.NamespaceID); err != nil {
		return domain.IdempotencyReservation{}, err
	}
	if err := ensurePrincipal(ctx, s.uow.tx, identity.PrincipalID); err != nil {
		return domain.IdempotencyReservation{}, err
	}

	now := time.Now().UTC()
	var (
		existingHash, status string
		commandID            sql.NullString
		revision             sql.NullInt64
		response             sql.NullString
		expiresAt            int64
	)
	err := s.uow.tx.QueryRowContext(ctx, `
SELECT request_hash, status, command_id, outcome_revision, response_json, expires_at
FROM idempotency_records
WHERE namespace_id = ? AND principal_id = ? AND command_name = ? AND idempotency_key = ?`,
		identity.NamespaceID.String(),
		identity.PrincipalID,
		identity.CommandName,
		identity.IdempotencyKey,
	).Scan(&existingHash, &status, &commandID, &revision, &response, &expiresAt)

	if err == nil && expiresAt <= encodeTime(now) {
		if _, deleteErr := s.uow.tx.ExecContext(ctx, `
DELETE FROM idempotency_records
WHERE namespace_id = ? AND principal_id = ? AND command_name = ? AND idempotency_key = ?`,
			identity.NamespaceID.String(),
			identity.PrincipalID,
			identity.CommandName,
			identity.IdempotencyKey,
		); deleteErr != nil {
			return domain.IdempotencyReservation{}, mapSQLError("expire idempotency record", deleteErr)
		}
		err = sql.ErrNoRows
	}

	if err == nil {
		if existingHash != fingerprint {
			return domain.IdempotencyReservation{}, domain.NewError(
				domain.ErrorCodeIdempotencyConflict,
				"idempotency key was already used with a different command fingerprint",
			)
		}
		if status != "completed" {
			return domain.IdempotencyReservation{}, domain.NewError(
				domain.ErrorCodeIdempotencyState,
				"idempotency reservation is still processing",
			)
		}
		if !commandID.Valid || !revision.Valid || !response.Valid {
			return domain.IdempotencyReservation{}, domain.NewError(
				domain.ErrorCodeIdempotencyState,
				"completed idempotency record is incomplete",
			)
		}
		parsedCommandID, err := domain.ParseID(commandID.String)
		if err != nil {
			return domain.IdempotencyReservation{}, err
		}
		result := domain.StoredCommandResult{
			CommandID:       parsedCommandID,
			OutcomeRevision: domain.OutcomeRevision(revision.Int64),
			ResponseJSON:    json.RawMessage(response.String),
		}
		if err := result.Validate(); err != nil {
			return domain.IdempotencyReservation{}, err
		}
		return domain.IdempotencyReservation{
			Identity:    identity,
			Fingerprint: fingerprint,
			Replay:      &result,
		}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.IdempotencyReservation{}, mapSQLError("read idempotency record", err)
	}

	if _, err := s.uow.tx.ExecContext(ctx, `
INSERT INTO idempotency_records (
    namespace_id, principal_id, command_name, idempotency_key,
    request_hash, status, created_at, expires_at
) VALUES (?, ?, ?, ?, ?, 'processing', ?, ?)`,
		identity.NamespaceID.String(),
		identity.PrincipalID,
		identity.CommandName,
		identity.IdempotencyKey,
		fingerprint,
		encodeTime(now),
		encodeTime(now.Add(s.uow.idempotencyRetention)),
	); err != nil {
		return domain.IdempotencyReservation{}, mapSQLError("reserve idempotency record", err)
	}
	return domain.IdempotencyReservation{
		Identity:    identity,
		Fingerprint: fingerprint,
	}, nil
}

func (s idempotencyStore) Complete(
	ctx context.Context,
	reservation domain.IdempotencyReservation,
	result domain.StoredCommandResult,
) error {
	if err := s.uow.ensureOpen(); err != nil {
		return err
	}
	if reservation.IsReplay() {
		return domain.NewError(domain.ErrorCodeIdempotencyState, "replay reservation cannot be completed again")
	}
	if err := reservation.Identity.Validate(); err != nil {
		return err
	}
	if err := result.Validate(); err != nil {
		return err
	}
	update, err := s.uow.tx.ExecContext(ctx, `
UPDATE idempotency_records
SET status = 'completed', command_id = ?, outcome_revision = ?, response_json = ?
WHERE namespace_id = ? AND principal_id = ? AND command_name = ?
  AND idempotency_key = ? AND request_hash = ? AND status = 'processing'`,
		result.CommandID.String(),
		int64(result.OutcomeRevision),
		string(result.ResponseJSON),
		reservation.Identity.NamespaceID.String(),
		reservation.Identity.PrincipalID,
		reservation.Identity.CommandName,
		reservation.Identity.IdempotencyKey,
		reservation.Fingerprint,
	)
	if err != nil {
		return mapSQLError("complete idempotency record", err)
	}
	affected, err := update.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return domain.NewError(
			domain.ErrorCodeIdempotencyState,
			"idempotency reservation was not found or changed before completion",
		)
	}
	return nil
}

func (s *Store) SnapshotDomainEvents(ctx context.Context, scope domain.Scope) ([]domain.DomainEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, event_type, schema_version, outcome_revision, event_index,
       aggregate_id, aggregate_kind, aggregate_version_before,
       aggregate_version_after, command_id, principal_id, actor_json,
       recorded_at, correlation_id, causation_id, execution_context_json,
       payload_json
FROM domain_events
WHERE namespace_id = ? AND outcome_id = ?
ORDER BY outcome_revision, event_index`,
		scope.NamespaceID.String(), scope.OutcomeID.String(),
	)
	if err != nil {
		return nil, mapSQLError("read domain event timeline", err)
	}
	defer rows.Close()
	result := make([]domain.DomainEvent, 0)
	for rows.Next() {
		event, err := scanEvent(rows, scope)
		if err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].OutcomeRevision == result[j].OutcomeRevision {
			return result[i].EventIndex < result[j].EventIndex
		}
		return result[i].OutcomeRevision < result[j].OutcomeRevision
	})
	return result, nil
}

func scanEvent(scanner rowScanner, scope domain.Scope) (domain.DomainEvent, error) {
	var (
		rawID, eventType, aggregateID, aggregateKind, commandID, principal string
		schemaVersion, revision, eventIndex, recordedAt                    int64
		before, after                                                      sql.NullInt64
		actorJSON, executionJSON, payload                                  string
		correlation, causation                                             sql.NullString
	)
	if err := scanner.Scan(
		&rawID, &eventType, &schemaVersion, &revision, &eventIndex,
		&aggregateID, &aggregateKind, &before, &after, &commandID, &principal,
		&actorJSON, &recordedAt, &correlation, &causation, &executionJSON, &payload,
	); err != nil {
		return domain.DomainEvent{}, err
	}
	eventID, err := domain.ParseID(rawID)
	if err != nil {
		return domain.DomainEvent{}, err
	}
	aggregateParsed, err := domain.ParseID(aggregateID)
	if err != nil {
		return domain.DomainEvent{}, err
	}
	kind, err := domain.ParseEntityKind(aggregateKind)
	if err != nil {
		return domain.DomainEvent{}, err
	}
	commandParsed, err := domain.ParseID(commandID)
	if err != nil {
		return domain.DomainEvent{}, err
	}
	var actor domain.ActorRef
	if err := unmarshalJSON(actorJSON, &actor); err != nil {
		return domain.DomainEvent{}, err
	}
	var execution domain.ExternalExecutionContext
	if err := unmarshalJSON(executionJSON, &execution); err != nil {
		return domain.DomainEvent{}, err
	}
	event := domain.DomainEvent{
		EventID:         eventID,
		EventType:       eventType,
		SchemaVersion:   uint32(schemaVersion),
		NamespaceID:     scope.NamespaceID,
		OutcomeID:       scope.OutcomeID,
		OutcomeRevision: domain.OutcomeRevision(revision),
		EventIndex:      uint32(eventIndex),
		AggregateRef: domain.EntityRef{
			Scope: scope,
			Kind:  kind,
			ID:    aggregateParsed,
		},
		PrincipalID:      principal,
		Actor:            actor,
		RecordedAt:       decodeTime(recordedAt),
		CommandID:        commandParsed,
		ExecutionContext: execution,
		Payload:          json.RawMessage(payload),
	}
	if before.Valid {
		v := domain.Version(before.Int64)
		event.AggregateVersionBefore = &v
	}
	if after.Valid {
		v := domain.Version(after.Int64)
		event.AggregateVersionAfter = &v
	}
	if correlation.Valid {
		event.CorrelationID = correlation.String
	}
	if causation.Valid {
		id, err := domain.ParseID(causation.String)
		if err != nil {
			return domain.DomainEvent{}, err
		}
		event.CausationID = &id
	}
	return event, event.Validate()
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
