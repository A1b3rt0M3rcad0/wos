package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

type artifactRepository struct{ uow *unitOfWork }

func (r artifactRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Artifact, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return domain.Artifact{}, err
	}
	var (
		version                                                     int64
		artifactType, name, uri, mediaType, checksum, sourceVersion string
		producerJSON, lifecycle, withdrawalReason                   string
		producedAt                                                  sql.NullInt64
		registeredAt, updatedAt                                     int64
	)
	err := r.uow.tx.QueryRowContext(ctx, `
SELECT version, artifact_type, name, uri, media_type, checksum, source_version,
       producer_json, produced_at, registered_at, lifecycle, withdrawal_reason, updated_at
FROM artifacts
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(),
	).Scan(
		&version, &artifactType, &name, &uri, &mediaType, &checksum, &sourceVersion,
		&producerJSON, &producedAt, &registeredAt, &lifecycle, &withdrawalReason, &updatedAt,
	)
	if err != nil {
		return domain.Artifact{}, mapSQLError("get artifact", err)
	}
	var producer domain.ActorRef
	if err := unmarshalJSON(producerJSON, &producer); err != nil {
		return domain.Artifact{}, err
	}
	value := domain.Artifact{
		ID:               id,
		Scope:            scope,
		Version:          domain.Version(version),
		ArtifactType:     artifactType,
		Name:             name,
		URI:              uri,
		MediaType:        mediaType,
		Checksum:         checksum,
		SourceVersion:    sourceVersion,
		ProducerRef:      producer,
		RegisteredAt:     decodeTime(registeredAt),
		Lifecycle:        domain.ArtifactLifecycle(lifecycle),
		WithdrawalReason: withdrawalReason,
		UpdatedAt:        decodeTime(updatedAt),
	}
	if producedAt.Valid {
		t := decodeTime(producedAt.Int64)
		value.ProducedAt = &t
	}
	if err := value.Validate(); err != nil {
		return domain.Artifact{}, domain.WrapError(domain.ErrorCodeArtifact, "persisted artifact is invalid", err)
	}
	return value, nil
}

func (r artifactRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Artifact, error) {
	ids, err := listDocumentaryIDs(ctx, r.uow, "artifacts", scope)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Artifact, 0, len(ids))
	for _, id := range ids {
		value, err := r.Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (r artifactRepository) Insert(ctx context.Context, value domain.Artifact) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if err := insertEntityRef(ctx, r.uow.tx, value.Ref()); err != nil {
		return err
	}
	producerJSON, err := marshalJSON(value.ProducerRef)
	if err != nil {
		return err
	}
	_, err = r.uow.tx.ExecContext(ctx, `
INSERT INTO artifacts (
    id, namespace_id, outcome_id, kind, version, artifact_type, name, uri,
    media_type, checksum, source_version, producer_json, produced_at,
    registered_at, lifecycle, withdrawal_reason, updated_at
) VALUES (?, ?, ?, 'artifact', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		value.ID.String(), value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(),
		int64(value.Version), value.ArtifactType, value.Name, value.URI, value.MediaType,
		value.Checksum, value.SourceVersion, producerJSON, encodeOptionalTime(value.ProducedAt),
		encodeTime(value.RegisteredAt), string(value.Lifecycle), value.WithdrawalReason,
		encodeTime(value.UpdatedAt),
	)
	return mapSQLError("insert artifact", err)
}

func (r artifactRepository) Save(ctx context.Context, value domain.Artifact, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	current, err := r.Get(ctx, value.Scope, value.ID)
	if err != nil {
		return err
	}
	if !sameArtifactContent(current, value) {
		return domain.NewError(domain.ErrorCodeArtifact, "artifact content is immutable")
	}
	if value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "artifact version must advance exactly once per save")
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE artifacts
SET version = ?, lifecycle = ?, withdrawal_reason = ?, updated_at = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(value.Version), string(value.Lifecycle), value.WithdrawalReason, encodeTime(value.UpdatedAt),
		value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(), value.ID.String(), int64(expected),
	)
	if err != nil {
		return mapSQLError("save artifact", err)
	}
	return requireVersionedUpdate(ctx, r.uow.tx, result, "artifacts", value.Scope.NamespaceID, value.Scope.OutcomeID, value.ID, expected)
}

type evidenceRepository struct{ uow *unitOfWork }

func (r evidenceRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Evidence, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return domain.Evidence{}, err
	}
	var (
		version                                              int64
		evidenceType, description, sourceJSON, producerJSON  string
		capturedAt, registeredAt, updatedAt                  int64
		artifactRaw, measurementJSON                         sql.NullString
		sourceVersion, checksum, lifecycle, retractionReason string
	)
	err := r.uow.tx.QueryRowContext(ctx, `
SELECT version, evidence_type, description, source_ref_json, producer_json,
       captured_at, registered_at, artifact_id, measurement_json,
       source_version, checksum, lifecycle, retraction_reason, updated_at
FROM evidence
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(),
	).Scan(
		&version, &evidenceType, &description, &sourceJSON, &producerJSON,
		&capturedAt, &registeredAt, &artifactRaw, &measurementJSON,
		&sourceVersion, &checksum, &lifecycle, &retractionReason, &updatedAt,
	)
	if err != nil {
		return domain.Evidence{}, mapSQLError("get evidence", err)
	}
	var source domain.ExternalReference
	if err := unmarshalJSON(sourceJSON, &source); err != nil {
		return domain.Evidence{}, err
	}
	var producer domain.ActorRef
	if err := unmarshalJSON(producerJSON, &producer); err != nil {
		return domain.Evidence{}, err
	}
	value := domain.Evidence{
		ID:               id,
		Scope:            scope,
		Version:          domain.Version(version),
		EvidenceType:     domain.EvidenceType(evidenceType),
		Description:      description,
		SourceRef:        source,
		ProducerRef:      producer,
		CapturedAt:       decodeTime(capturedAt),
		RegisteredAt:     decodeTime(registeredAt),
		SourceVersion:    sourceVersion,
		Checksum:         checksum,
		Lifecycle:        domain.EvidenceLifecycle(lifecycle),
		RetractionReason: retractionReason,
		UpdatedAt:        decodeTime(updatedAt),
	}
	if artifactRaw.Valid {
		artifactID, err := domain.ParseID(artifactRaw.String)
		if err != nil {
			return domain.Evidence{}, err
		}
		value.ArtifactID = &artifactID
	}
	if measurementJSON.Valid {
		var measurement domain.Measurement
		if err := json.Unmarshal([]byte(measurementJSON.String), &measurement); err != nil {
			return domain.Evidence{}, err
		}
		value.Measurement = &measurement
	}
	if err := value.Validate(); err != nil {
		return domain.Evidence{}, domain.WrapError(domain.ErrorCodeEvidence, "persisted evidence is invalid", err)
	}
	return value, nil
}

func (r evidenceRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Evidence, error) {
	ids, err := listDocumentaryIDs(ctx, r.uow, "evidence", scope)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Evidence, 0, len(ids))
	for _, id := range ids {
		value, err := r.Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (r evidenceRepository) Insert(ctx context.Context, value domain.Evidence) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if err := insertEntityRef(ctx, r.uow.tx, value.Ref()); err != nil {
		return err
	}
	sourceJSON, err := marshalJSON(value.SourceRef)
	if err != nil {
		return err
	}
	producerJSON, err := marshalJSON(value.ProducerRef)
	if err != nil {
		return err
	}
	var measurementJSON any
	if value.Measurement != nil {
		encoded, err := marshalJSON(value.Measurement)
		if err != nil {
			return err
		}
		measurementJSON = encoded
	}
	_, err = r.uow.tx.ExecContext(ctx, `
INSERT INTO evidence (
    id, namespace_id, outcome_id, kind, version, evidence_type, description,
    source_ref_json, producer_json, captured_at, registered_at, artifact_id,
    measurement_json, source_version, checksum, lifecycle, retraction_reason, updated_at
) VALUES (?, ?, ?, 'evidence', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		value.ID.String(), value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(),
		int64(value.Version), string(value.EvidenceType), value.Description,
		sourceJSON, producerJSON, encodeTime(value.CapturedAt), encodeTime(value.RegisteredAt),
		nullableDocumentaryID(value.ArtifactID), measurementJSON, value.SourceVersion, value.Checksum,
		string(value.Lifecycle), value.RetractionReason, encodeTime(value.UpdatedAt),
	)
	return mapSQLError("insert evidence", err)
}

func (r evidenceRepository) Save(ctx context.Context, value domain.Evidence, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	current, err := r.Get(ctx, value.Scope, value.ID)
	if err != nil {
		return err
	}
	if !sameEvidenceContent(current, value) {
		return domain.NewError(domain.ErrorCodeEvidence, "evidence observation is immutable")
	}
	if value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "evidence version must advance exactly once per save")
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE evidence
SET version = ?, lifecycle = ?, retraction_reason = ?, updated_at = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(value.Version), string(value.Lifecycle), value.RetractionReason, encodeTime(value.UpdatedAt),
		value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(), value.ID.String(), int64(expected),
	)
	if err != nil {
		return mapSQLError("save evidence", err)
	}
	return requireVersionedUpdate(ctx, r.uow.tx, result, "evidence", value.Scope.NamespaceID, value.Scope.OutcomeID, value.ID, expected)
}

type decisionRepository struct{ uow *unitOfWork }

func (r decisionRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Decision, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return domain.Decision{}, err
	}
	var (
		version                                              int64
		title, proposal, chosen, rationale, alternativesJSON string
		lifecycle                                            string
		decidedByJSON, supersedesRaw                         sql.NullString
		decidedAt                                            sql.NullInt64
		createdAt, updatedAt                                 int64
	)
	err := r.uow.tx.QueryRowContext(ctx, `
SELECT version, title, proposal, chosen_alternative, rationale, alternatives_json,
       lifecycle, decided_by_json, decided_at, supersedes_decision_id, created_at, updated_at
FROM decisions
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(),
	).Scan(
		&version, &title, &proposal, &chosen, &rationale, &alternativesJSON,
		&lifecycle, &decidedByJSON, &decidedAt, &supersedesRaw, &createdAt, &updatedAt,
	)
	if err != nil {
		return domain.Decision{}, mapSQLError("get decision", err)
	}
	var alternatives []string
	if err := unmarshalJSON(alternativesJSON, &alternatives); err != nil {
		return domain.Decision{}, err
	}
	value := domain.Decision{
		ID:                id,
		Scope:             scope,
		Version:           domain.Version(version),
		Title:             title,
		Proposal:          proposal,
		ChosenAlternative: chosen,
		Rationale:         rationale,
		Alternatives:      alternatives,
		Lifecycle:         domain.DecisionLifecycle(lifecycle),
		CreatedAt:         decodeTime(createdAt),
		UpdatedAt:         decodeTime(updatedAt),
	}
	if decidedByJSON.Valid {
		var actor domain.ActorRef
		if err := unmarshalJSON(decidedByJSON.String, &actor); err != nil {
			return domain.Decision{}, err
		}
		value.DecidedBy = &actor
	}
	if decidedAt.Valid {
		t := decodeTime(decidedAt.Int64)
		value.DecidedAt = &t
	}
	if supersedesRaw.Valid {
		parsed, err := domain.ParseID(supersedesRaw.String)
		if err != nil {
			return domain.Decision{}, err
		}
		value.SupersedesDecisionID = &parsed
	}
	if err := value.Validate(); err != nil {
		return domain.Decision{}, domain.WrapError(domain.ErrorCodeDecision, "persisted decision is invalid", err)
	}
	return value, nil
}

func (r decisionRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Decision, error) {
	ids, err := listDocumentaryIDs(ctx, r.uow, "decisions", scope)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Decision, 0, len(ids))
	for _, id := range ids {
		value, err := r.Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (r decisionRepository) Insert(ctx context.Context, value domain.Decision) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if err := insertEntityRef(ctx, r.uow.tx, value.Ref()); err != nil {
		return err
	}
	alternativesJSON, err := marshalJSON(value.Alternatives)
	if err != nil {
		return err
	}
	decidedByJSON, err := nullableActorJSON(value.DecidedBy)
	if err != nil {
		return err
	}
	_, err = r.uow.tx.ExecContext(ctx, `
INSERT INTO decisions (
    id, namespace_id, outcome_id, kind, version, title, proposal, chosen_alternative,
    rationale, alternatives_json, lifecycle, decided_by_json, decided_at,
    supersedes_decision_id, created_at, updated_at
) VALUES (?, ?, ?, 'decision', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		value.ID.String(), value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(),
		int64(value.Version), value.Title, value.Proposal, value.ChosenAlternative,
		value.Rationale, alternativesJSON, string(value.Lifecycle), decidedByJSON,
		encodeOptionalTime(value.DecidedAt), nullableDocumentaryID(value.SupersedesDecisionID),
		encodeTime(value.CreatedAt), encodeTime(value.UpdatedAt),
	)
	return mapSQLError("insert decision", err)
}

func (r decisionRepository) Save(ctx context.Context, value domain.Decision, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	current, err := r.Get(ctx, value.Scope, value.ID)
	if err != nil {
		return err
	}
	if current.Lifecycle != domain.DecisionLifecycleProposed && !sameDecisionContent(current, value) {
		return domain.NewError(domain.ErrorCodeDecision, "final decision content is immutable")
	}
	if value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "decision version must advance exactly once per save")
	}
	alternativesJSON, err := marshalJSON(value.Alternatives)
	if err != nil {
		return err
	}
	decidedByJSON, err := nullableActorJSON(value.DecidedBy)
	if err != nil {
		return err
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE decisions
SET version = ?, title = ?, proposal = ?, chosen_alternative = ?, rationale = ?,
    alternatives_json = ?, lifecycle = ?, decided_by_json = ?, decided_at = ?,
    supersedes_decision_id = ?, updated_at = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(value.Version), value.Title, value.Proposal, value.ChosenAlternative, value.Rationale,
		alternativesJSON, string(value.Lifecycle), decidedByJSON, encodeOptionalTime(value.DecidedAt),
		nullableDocumentaryID(value.SupersedesDecisionID), encodeTime(value.UpdatedAt),
		value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(), value.ID.String(), int64(expected),
	)
	if err != nil {
		return mapSQLError("save decision", err)
	}
	return requireVersionedUpdate(ctx, r.uow.tx, result, "decisions", value.Scope.NamespaceID, value.Scope.OutcomeID, value.ID, expected)
}

type evidenceLinkRepository struct{ uow *unitOfWork }

func (r evidenceLinkRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.EvidenceLink, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return domain.EvidenceLink{}, err
	}
	var (
		version                            int64
		evidenceRaw, targetRaw, targetKind string
		criterionRaw                       sql.NullString
		stance, rationale, lifecycle       string
		createdAt, updatedAt               int64
	)
	err := r.uow.tx.QueryRowContext(ctx, `
SELECT version, evidence_id, target_id, target_kind, criterion_id,
       stance, rationale, lifecycle, created_at, updated_at
FROM evidence_links
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(),
	).Scan(
		&version, &evidenceRaw, &targetRaw, &targetKind, &criterionRaw,
		&stance, &rationale, &lifecycle, &createdAt, &updatedAt,
	)
	if err != nil {
		return domain.EvidenceLink{}, mapSQLError("get evidence link", err)
	}
	evidenceID, err := domain.ParseID(evidenceRaw)
	if err != nil {
		return domain.EvidenceLink{}, err
	}
	targetID, err := domain.ParseID(targetRaw)
	if err != nil {
		return domain.EvidenceLink{}, err
	}
	kind, err := domain.ParseEntityKind(targetKind)
	if err != nil {
		return domain.EvidenceLink{}, err
	}
	value := domain.EvidenceLink{
		ID:         id,
		Scope:      scope,
		Version:    domain.Version(version),
		EvidenceID: evidenceID,
		TargetRef:  domain.EntityRef{Scope: scope, Kind: kind, ID: targetID},
		Stance:     domain.EvidenceStance(stance),
		Rationale:  rationale,
		Lifecycle:  domain.EvidenceLinkLifecycle(lifecycle),
		CreatedAt:  decodeTime(createdAt),
		UpdatedAt:  decodeTime(updatedAt),
	}
	if criterionRaw.Valid {
		criterionID, err := domain.ParseID(criterionRaw.String)
		if err != nil {
			return domain.EvidenceLink{}, err
		}
		value.CriterionID = &criterionID
	}
	if err := value.Validate(); err != nil {
		return domain.EvidenceLink{}, domain.WrapError(domain.ErrorCodeEvidenceLink, "persisted evidence link is invalid", err)
	}
	return value, nil
}

func (r evidenceLinkRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.EvidenceLink, error) {
	ids, err := listDocumentaryIDs(ctx, r.uow, "evidence_links", scope)
	if err != nil {
		return nil, err
	}
	result := make([]domain.EvidenceLink, 0, len(ids))
	for _, id := range ids {
		value, err := r.Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (r evidenceLinkRepository) Insert(ctx context.Context, value domain.EvidenceLink) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if err := insertEntityRef(ctx, r.uow.tx, value.Ref()); err != nil {
		return err
	}
	_, err := r.uow.tx.ExecContext(ctx, `
INSERT INTO evidence_links (
    id, namespace_id, outcome_id, kind, version, evidence_id, target_id, target_kind,
    criterion_id, stance, rationale, lifecycle, created_at, updated_at
) VALUES (?, ?, ?, 'evidence_link', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		value.ID.String(), value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(),
		int64(value.Version), value.EvidenceID.String(), value.TargetRef.ID.String(),
		value.TargetRef.Kind.String(), nullableDocumentaryID(value.CriterionID),
		string(value.Stance), value.Rationale, string(value.Lifecycle),
		encodeTime(value.CreatedAt), encodeTime(value.UpdatedAt),
	)
	return mapSQLError("insert evidence link", err)
}

func (r evidenceLinkRepository) Save(ctx context.Context, value domain.EvidenceLink, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	current, err := r.Get(ctx, value.Scope, value.ID)
	if err != nil {
		return err
	}
	if !sameEvidenceLinkContent(current, value) {
		return domain.NewError(domain.ErrorCodeEvidenceLink, "evidence link content is immutable")
	}
	if value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "evidence link version must advance exactly once per save")
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE evidence_links
SET version = ?, lifecycle = ?, updated_at = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(value.Version), string(value.Lifecycle), encodeTime(value.UpdatedAt),
		value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(), value.ID.String(), int64(expected),
	)
	if err != nil {
		return mapSQLError("save evidence link", err)
	}
	return requireVersionedUpdate(ctx, r.uow.tx, result, "evidence_links", value.Scope.NamespaceID, value.Scope.OutcomeID, value.ID, expected)
}

func listDocumentaryIDs(ctx context.Context, uow *unitOfWork, table string, scope domain.Scope) ([]domain.ID, error) {
	if err := uow.ensureOpen(); err != nil {
		return nil, err
	}
	query := "SELECT id FROM " + table + " WHERE namespace_id = ? AND outcome_id = ? ORDER BY id"
	rows, err := uow.tx.QueryContext(ctx, query, scope.NamespaceID.String(), scope.OutcomeID.String())
	if err != nil {
		return nil, mapSQLError("list documentary records", err)
	}
	defer rows.Close()
	result := make([]domain.ID, 0)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		id, err := domain.ParseID(raw)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func sameArtifactContent(left, right domain.Artifact) bool {
	return left.ArtifactType == right.ArtifactType &&
		left.Name == right.Name &&
		left.URI == right.URI &&
		left.MediaType == right.MediaType &&
		left.Checksum == right.Checksum &&
		left.SourceVersion == right.SourceVersion &&
		left.ProducerRef == right.ProducerRef &&
		sameOptionalTime(left.ProducedAt, right.ProducedAt) &&
		left.RegisteredAt.Equal(right.RegisteredAt)
}

func sameEvidenceContent(left, right domain.Evidence) bool {
	return left.EvidenceType == right.EvidenceType &&
		left.Description == right.Description &&
		left.SourceRef == right.SourceRef &&
		left.ProducerRef == right.ProducerRef &&
		left.CapturedAt.Equal(right.CapturedAt) &&
		left.RegisteredAt.Equal(right.RegisteredAt) &&
		sameOptionalID(left.ArtifactID, right.ArtifactID) &&
		sameMeasurement(left.Measurement, right.Measurement) &&
		left.SourceVersion == right.SourceVersion &&
		left.Checksum == right.Checksum
}

func sameDecisionContent(left, right domain.Decision) bool {
	return left.Title == right.Title &&
		left.Proposal == right.Proposal &&
		left.ChosenAlternative == right.ChosenAlternative &&
		left.Rationale == right.Rationale &&
		sameStrings(left.Alternatives, right.Alternatives) &&
		sameOptionalActor(left.DecidedBy, right.DecidedBy) &&
		sameOptionalTime(left.DecidedAt, right.DecidedAt) &&
		sameOptionalID(left.SupersedesDecisionID, right.SupersedesDecisionID) &&
		left.CreatedAt.Equal(right.CreatedAt)
}

func sameEvidenceLinkContent(left, right domain.EvidenceLink) bool {
	return left.EvidenceID == right.EvidenceID &&
		left.TargetRef == right.TargetRef &&
		sameOptionalID(left.CriterionID, right.CriterionID) &&
		left.Stance == right.Stance &&
		left.Rationale == right.Rationale &&
		left.CreatedAt.Equal(right.CreatedAt)
}

func sameOptionalID(left, right *domain.ID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func sameOptionalActor(left, right *domain.ActorRef) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sameMeasurement(left, right *domain.Measurement) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return string(left.Value) == string(right.Value) &&
		left.Unit == right.Unit &&
		left.Method == right.Method &&
		string(left.Conditions) == string(right.Conditions)
}

func nullableDocumentaryID(value *domain.ID) any {
	if value == nil {
		return nil
	}
	return value.String()
}

func nullableActorJSON(value *domain.ActorRef) (any, error) {
	if value == nil {
		return nil, nil
	}
	return marshalJSON(*value)
}
