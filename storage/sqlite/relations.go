package sqlite

import (
	"context"
	"database/sql"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

type relationRepository struct{ uow *unitOfWork }

func (r relationRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Relation, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return domain.Relation{}, err
	}
	var (
		version                                int64
		sourceID, sourceKind, relationType     string
		targetID, targetKind, lifecycle        string
		strength, satisfaction, removalReason sql.NullString
		createdAt, updatedAt                   int64
	)
	err := r.uow.tx.QueryRowContext(ctx, `
SELECT version, source_id, source_kind, relation_type, target_id, target_kind,
       strength, satisfaction, lifecycle, removal_reason, created_at, updated_at
FROM relations
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(),
	).Scan(
		&version, &sourceID, &sourceKind, &relationType, &targetID, &targetKind,
		&strength, &satisfaction, &lifecycle, &removalReason, &createdAt, &updatedAt,
	)
	if err != nil {
		return domain.Relation{}, mapSQLError("get relation", err)
	}
	sourceParsed, err := domain.ParseID(sourceID)
	if err != nil {
		return domain.Relation{}, domain.WrapError(domain.ErrorCodeInvalidRelation, "persisted relation source id is invalid", err)
	}
	targetParsed, err := domain.ParseID(targetID)
	if err != nil {
		return domain.Relation{}, domain.WrapError(domain.ErrorCodeInvalidRelation, "persisted relation target id is invalid", err)
	}
	sourceKindParsed, err := domain.ParseEntityKind(sourceKind)
	if err != nil {
		return domain.Relation{}, err
	}
	targetKindParsed, err := domain.ParseEntityKind(targetKind)
	if err != nil {
		return domain.Relation{}, err
	}
	value := domain.Relation{
		ID:            id,
		Scope:         scope,
		Version:       domain.Version(version),
		SourceRef:     domain.EntityRef{Scope: scope, Kind: sourceKindParsed, ID: sourceParsed},
		RelationType:  domain.RelationType(relationType),
		TargetRef:     domain.EntityRef{Scope: scope, Kind: targetKindParsed, ID: targetParsed},
		Strength:      domain.DependencyStrength(strength.String),
		Satisfaction:  domain.DependencySatisfaction(satisfaction.String),
		Lifecycle:     domain.RelationLifecycle(lifecycle),
		RemovalReason: removalReason.String,
		CreatedAt:     decodeTime(createdAt),
		UpdatedAt:     decodeTime(updatedAt),
	}
	if err := value.Validate(); err != nil {
		return domain.Relation{}, domain.WrapError(domain.ErrorCodeInvalidRelation, "persisted relation is invalid", err)
	}
	return value, nil
}

func (r relationRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Relation, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return nil, err
	}
	rows, err := r.uow.tx.QueryContext(ctx, `
SELECT id FROM relations
WHERE namespace_id = ? AND outcome_id = ?
ORDER BY id`, scope.NamespaceID.String(), scope.OutcomeID.String())
	if err != nil {
		return nil, mapSQLError("list relations", err)
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
	result := make([]domain.Relation, 0, len(ids))
	for _, id := range ids {
		value, err := r.Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (r relationRepository) Insert(ctx context.Context, relation domain.Relation) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := relation.Validate(); err != nil {
		return err
	}
	if err := insertEntityRef(ctx, r.uow.tx, relation.Ref()); err != nil {
		return err
	}
	_, err := r.uow.tx.ExecContext(ctx, `
INSERT INTO relations (
    id, namespace_id, outcome_id, kind, version,
    source_id, source_kind, relation_type, target_id, target_kind,
    strength, satisfaction, lifecycle, removal_reason, created_at, updated_at
) VALUES (?, ?, ?, 'relation', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		relation.ID.String(),
		relation.Scope.NamespaceID.String(),
		relation.Scope.OutcomeID.String(),
		int64(relation.Version),
		relation.SourceRef.ID.String(),
		relation.SourceRef.Kind.String(),
		string(relation.RelationType),
		relation.TargetRef.ID.String(),
		relation.TargetRef.Kind.String(),
		nullableString(string(relation.Strength)),
		nullableString(string(relation.Satisfaction)),
		string(relation.Lifecycle),
		relation.RemovalReason,
		encodeTime(relation.CreatedAt),
		encodeTime(relation.UpdatedAt),
	)
	return mapSQLError("insert relation", err)
}

func (r relationRepository) Save(ctx context.Context, relation domain.Relation, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := relation.Validate(); err != nil {
		return err
	}
	if relation.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "relation version must advance exactly once per save")
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE relations
SET version = ?, lifecycle = ?, removal_reason = ?, updated_at = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(relation.Version),
		string(relation.Lifecycle),
		relation.RemovalReason,
		encodeTime(relation.UpdatedAt),
		relation.Scope.NamespaceID.String(),
		relation.Scope.OutcomeID.String(),
		relation.ID.String(),
		int64(expected),
	)
	if err != nil {
		return mapSQLError("save relation", err)
	}
	return requireVersionedUpdate(
		ctx, r.uow.tx, result, "relations",
		relation.Scope.NamespaceID, relation.Scope.OutcomeID, relation.ID, expected,
	)
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
