package sqlite

import (
	"context"
	"database/sql"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

type issueRepository struct{ uow *unitOfWork }

func (r issueRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Issue, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return domain.Issue{}, err
	}
	var (
		version                                 int64
		title, description, severity, lifecycle string
		reportedByJSON, resolutionSummary       string
		duplicateRaw                            sql.NullString
		createdAt, updatedAt                    int64
	)
	err := r.uow.tx.QueryRowContext(ctx, `
SELECT version, title, description, severity, lifecycle, reported_by_json,
       resolution_summary, duplicate_of_issue_id, created_at, updated_at
FROM issues
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(),
	).Scan(
		&version, &title, &description, &severity, &lifecycle, &reportedByJSON,
		&resolutionSummary, &duplicateRaw, &createdAt, &updatedAt,
	)
	if err != nil {
		return domain.Issue{}, mapSQLError("get issue", err)
	}
	var reportedBy domain.ActorRef
	if err := unmarshalJSON(reportedByJSON, &reportedBy); err != nil {
		return domain.Issue{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "persisted issue reported_by is invalid", err)
	}
	value := domain.Issue{
		ID:                id,
		Scope:             scope,
		Version:           domain.Version(version),
		Title:             title,
		Description:       description,
		Severity:          domain.IssueSeverity(severity),
		Lifecycle:         domain.IssueLifecycle(lifecycle),
		ReportedBy:        reportedBy,
		ResolutionSummary: resolutionSummary,
		CreatedAt:         decodeTime(createdAt),
		UpdatedAt:         decodeTime(updatedAt),
	}
	if duplicateRaw.Valid {
		parsed, err := domain.ParseID(duplicateRaw.String)
		if err != nil {
			return domain.Issue{}, err
		}
		value.DuplicateOfIssueID = &parsed
	}
	refs, err := r.loadAffectedRefs(ctx, scope, id)
	if err != nil {
		return domain.Issue{}, err
	}
	value.AffectedRefs = refs
	if err := value.Validate(); err != nil {
		return domain.Issue{}, domain.WrapError(domain.ErrorCodeIssue, "persisted issue is invalid", err)
	}
	return value, nil
}

func (r issueRepository) loadAffectedRefs(ctx context.Context, scope domain.Scope, id domain.ID) ([]domain.EntityRef, error) {
	rows, err := r.uow.tx.QueryContext(ctx, `
SELECT affected_id, affected_kind
FROM issue_affected_refs
WHERE namespace_id = ? AND outcome_id = ? AND issue_id = ?
ORDER BY affected_kind, affected_id`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(),
	)
	if err != nil {
		return nil, mapSQLError("load issue affected refs", err)
	}
	defer rows.Close()
	result := make([]domain.EntityRef, 0)
	for rows.Next() {
		var rawID, rawKind string
		if err := rows.Scan(&rawID, &rawKind); err != nil {
			return nil, err
		}
		parsedID, err := domain.ParseID(rawID)
		if err != nil {
			return nil, err
		}
		kind, err := domain.ParseEntityKind(rawKind)
		if err != nil {
			return nil, err
		}
		result = append(result, domain.EntityRef{Scope: scope, Kind: kind, ID: parsedID})
	}
	return result, rows.Err()
}

func (r issueRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Issue, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return nil, err
	}
	rows, err := r.uow.tx.QueryContext(ctx, `
SELECT id FROM issues
WHERE namespace_id = ? AND outcome_id = ?
ORDER BY id`, scope.NamespaceID.String(), scope.OutcomeID.String())
	if err != nil {
		return nil, mapSQLError("list issues", err)
	}
	defer rows.Close()
	ids := make([]domain.ID, 0)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		id, err := domain.ParseID(raw)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]domain.Issue, 0, len(ids))
	for _, id := range ids {
		value, err := r.Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (r issueRepository) Insert(ctx context.Context, issue domain.Issue) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := issue.Validate(); err != nil {
		return err
	}
	if err := insertEntityRef(ctx, r.uow.tx, issue.Ref()); err != nil {
		return err
	}
	reportedByJSON, err := marshalJSON(issue.ReportedBy)
	if err != nil {
		return err
	}
	_, err = r.uow.tx.ExecContext(ctx, `
INSERT INTO issues (
    id, namespace_id, outcome_id, kind, version, title, description, severity,
    lifecycle, reported_by_json, resolution_summary, duplicate_of_issue_id,
    created_at, updated_at
) VALUES (?, ?, ?, 'issue', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		issue.ID.String(), issue.Scope.NamespaceID.String(), issue.Scope.OutcomeID.String(),
		int64(issue.Version), issue.Title, issue.Description, string(issue.Severity),
		string(issue.Lifecycle), reportedByJSON, issue.ResolutionSummary,
		nullableID(issue.DuplicateOfIssueID), encodeTime(issue.CreatedAt), encodeTime(issue.UpdatedAt),
	)
	if err != nil {
		return mapSQLError("insert issue", err)
	}
	for _, ref := range issue.AffectedRefs {
		if _, err := r.uow.tx.ExecContext(ctx, `
INSERT INTO issue_affected_refs (
    namespace_id, outcome_id, issue_id, affected_id, affected_kind
) VALUES (?, ?, ?, ?, ?)`,
			issue.Scope.NamespaceID.String(), issue.Scope.OutcomeID.String(), issue.ID.String(),
			ref.ID.String(), ref.Kind.String(),
		); err != nil {
			return mapSQLError("insert issue affected ref", err)
		}
	}
	return nil
}

func (r issueRepository) Save(ctx context.Context, issue domain.Issue, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := issue.Validate(); err != nil {
		return err
	}
	if issue.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "issue version must advance exactly once per save")
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE issues
SET version = ?, lifecycle = ?, resolution_summary = ?, duplicate_of_issue_id = ?, updated_at = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(issue.Version), string(issue.Lifecycle), issue.ResolutionSummary,
		nullableID(issue.DuplicateOfIssueID), encodeTime(issue.UpdatedAt),
		issue.Scope.NamespaceID.String(), issue.Scope.OutcomeID.String(), issue.ID.String(), int64(expected),
	)
	if err != nil {
		return mapSQLError("save issue", err)
	}
	return requireVersionedUpdate(
		ctx, r.uow.tx, result, "issues",
		issue.Scope.NamespaceID, issue.Scope.OutcomeID, issue.ID, expected,
	)
}

type blockerRepository struct{ uow *unitOfWork }

func (r blockerRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Blocker, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return domain.Blocker{}, err
	}
	var (
		version                                                     int64
		blockedID, blockedKind, description, lifecycle, propagation string
		causeID, causeKind, externalJSON                            sql.NullString
		resolvedAt                                                  sql.NullInt64
		resolutionSummary                                           string
		createdAt, updatedAt                                        int64
	)
	err := r.uow.tx.QueryRowContext(ctx, `
SELECT version, blocked_id, blocked_kind, cause_id, cause_kind, external_cause_json,
       description, lifecycle, propagation, resolved_at, resolution_summary, created_at, updated_at
FROM blockers
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(),
	).Scan(
		&version, &blockedID, &blockedKind, &causeID, &causeKind, &externalJSON,
		&description, &lifecycle, &propagation, &resolvedAt, &resolutionSummary, &createdAt, &updatedAt,
	)
	if err != nil {
		return domain.Blocker{}, mapSQLError("get blocker", err)
	}
	blockedParsed, err := domain.ParseID(blockedID)
	if err != nil {
		return domain.Blocker{}, err
	}
	blockedKindParsed, err := domain.ParseEntityKind(blockedKind)
	if err != nil {
		return domain.Blocker{}, err
	}
	value := domain.Blocker{
		ID:                id,
		Scope:             scope,
		Version:           domain.Version(version),
		BlockedRef:        domain.EntityRef{Scope: scope, Kind: blockedKindParsed, ID: blockedParsed},
		Description:       description,
		Lifecycle:         domain.BlockerLifecycle(lifecycle),
		Propagation:       domain.BlockerPropagation(propagation),
		ResolutionSummary: resolutionSummary,
		CreatedAt:         decodeTime(createdAt),
		UpdatedAt:         decodeTime(updatedAt),
	}
	if causeID.Valid {
		parsed, err := domain.ParseID(causeID.String)
		if err != nil {
			return domain.Blocker{}, err
		}
		kind, err := domain.ParseEntityKind(causeKind.String)
		if err != nil {
			return domain.Blocker{}, err
		}
		ref := domain.EntityRef{Scope: scope, Kind: kind, ID: parsed}
		value.CauseRef = &ref
	}
	if externalJSON.Valid {
		var external domain.ExternalCause
		if err := unmarshalJSON(externalJSON.String, &external); err != nil {
			return domain.Blocker{}, err
		}
		value.ExternalCause = &external
	}
	if resolvedAt.Valid {
		t := decodeTime(resolvedAt.Int64)
		value.ResolvedAt = &t
	}
	if err := value.Validate(); err != nil {
		return domain.Blocker{}, domain.WrapError(domain.ErrorCodeBlocker, "persisted blocker is invalid", err)
	}
	return value, nil
}

func (r blockerRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Blocker, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return nil, err
	}
	rows, err := r.uow.tx.QueryContext(ctx, `
SELECT id FROM blockers
WHERE namespace_id = ? AND outcome_id = ?
ORDER BY id`, scope.NamespaceID.String(), scope.OutcomeID.String())
	if err != nil {
		return nil, mapSQLError("list blockers", err)
	}
	defer rows.Close()
	ids := make([]domain.ID, 0)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		id, err := domain.ParseID(raw)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]domain.Blocker, 0, len(ids))
	for _, id := range ids {
		value, err := r.Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (r blockerRepository) Insert(ctx context.Context, blocker domain.Blocker) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := blocker.Validate(); err != nil {
		return err
	}
	if err := insertEntityRef(ctx, r.uow.tx, blocker.Ref()); err != nil {
		return err
	}
	var causeID, causeKind, externalJSON any
	if blocker.CauseRef != nil {
		causeID = blocker.CauseRef.ID.String()
		causeKind = blocker.CauseRef.Kind.String()
	}
	if blocker.ExternalCause != nil {
		encoded, err := marshalJSON(blocker.ExternalCause)
		if err != nil {
			return err
		}
		externalJSON = encoded
	}
	_, err := r.uow.tx.ExecContext(ctx, `
INSERT INTO blockers (
    id, namespace_id, outcome_id, kind, version, blocked_id, blocked_kind,
    cause_id, cause_kind, external_cause_json, description, lifecycle, propagation,
    resolved_at, resolution_summary, created_at, updated_at
) VALUES (?, ?, ?, 'blocker', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		blocker.ID.String(), blocker.Scope.NamespaceID.String(), blocker.Scope.OutcomeID.String(),
		int64(blocker.Version), blocker.BlockedRef.ID.String(), blocker.BlockedRef.Kind.String(),
		causeID, causeKind, externalJSON, blocker.Description, string(blocker.Lifecycle),
		string(blocker.Propagation), encodeOptionalTime(blocker.ResolvedAt), blocker.ResolutionSummary,
		encodeTime(blocker.CreatedAt), encodeTime(blocker.UpdatedAt),
	)
	return mapSQLError("insert blocker", err)
}

func (r blockerRepository) Save(ctx context.Context, blocker domain.Blocker, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := blocker.Validate(); err != nil {
		return err
	}
	if blocker.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "blocker version must advance exactly once per save")
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE blockers
SET version = ?, lifecycle = ?, resolved_at = ?, resolution_summary = ?, updated_at = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(blocker.Version), string(blocker.Lifecycle), encodeOptionalTime(blocker.ResolvedAt),
		blocker.ResolutionSummary, encodeTime(blocker.UpdatedAt),
		blocker.Scope.NamespaceID.String(), blocker.Scope.OutcomeID.String(), blocker.ID.String(), int64(expected),
	)
	if err != nil {
		return mapSQLError("save blocker", err)
	}
	return requireVersionedUpdate(
		ctx, r.uow.tx, result, "blockers",
		blocker.Scope.NamespaceID, blocker.Scope.OutcomeID, blocker.ID, expected,
	)
}
