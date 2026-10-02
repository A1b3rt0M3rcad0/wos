package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
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
		producedAt, withdrawnAt                                     sql.NullInt64
		registeredAt, updatedAt                                     int64
	)
	err := r.uow.tx.QueryRowContext(ctx, `
SELECT version, artifact_type, name, uri, media_type, checksum, source_version,
       producer_ref_json, produced_at, registered_at, lifecycle, withdrawal_reason,
       withdrawn_at, updated_at
FROM artifacts
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(),
	).Scan(&version, &artifactType, &name, &uri, &mediaType, &checksum, &sourceVersion,
		&producerJSON, &producedAt, &registeredAt, &lifecycle, &withdrawalReason,
		&withdrawnAt, &updatedAt)
	if err != nil {
		return domain.Artifact{}, mapSQLError("get artifact", err)
	}
	var producer domain.ActorRef
	if err := unmarshalJSON(producerJSON, &producer); err != nil {
		return domain.Artifact{}, domain.WrapError(domain.ErrorCodeArtifact, "persisted artifact producer_ref is invalid", err)
	}
	value := domain.Artifact{
		ID: id, Scope: scope, Version: domain.Version(version), ArtifactType: artifactType,
		Name: name, URI: uri, MediaType: mediaType, Checksum: checksum, SourceVersion: sourceVersion,
		ProducerRef: producer, RegisteredAt: decodeTime(registeredAt), Lifecycle: domain.ArtifactLifecycle(lifecycle),
		WithdrawalReason: withdrawalReason, UpdatedAt: decodeTime(updatedAt),
	}
	if producedAt.Valid {
		at := decodeTime(producedAt.Int64)
		value.ProducedAt = &at
	}
	if withdrawnAt.Valid {
		at := decodeTime(withdrawnAt.Int64)
		value.WithdrawnAt = &at
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
	producer, err := marshalJSON(value.ProducerRef)
	if err != nil {
		return err
	}
	_, err = r.uow.tx.ExecContext(ctx, `
INSERT INTO artifacts (
    id, namespace_id, outcome_id, kind, version, artifact_type, name, uri, media_type,
    checksum, source_version, producer_ref_json, produced_at, registered_at, lifecycle,
    withdrawal_reason, withdrawn_at, updated_at
) VALUES (?, ?, ?, 'artifact', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		value.ID.String(), value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(),
		int64(value.Version), value.ArtifactType, value.Name, value.URI, value.MediaType,
		value.Checksum, value.SourceVersion, producer, encodeOptionalTime(value.ProducedAt),
		encodeTime(value.RegisteredAt), string(value.Lifecycle), value.WithdrawalReason,
		encodeOptionalTime(value.WithdrawnAt), encodeTime(value.UpdatedAt),
	)
	if err != nil {
		return mapSQLError("insert artifact", err)
	}
	return nil
}

func (r artifactRepository) Save(ctx context.Context, value domain.Artifact, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "artifact version must advance exactly once per save")
	}
	current, err := r.Get(ctx, value.Scope, value.ID)
	if err != nil {
		return err
	}
	if !sameArtifactContent(current, value) {
		return domain.NewError(domain.ErrorCodeArtifact, "artifact content is immutable")
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE artifacts
SET version = ?, lifecycle = ?, withdrawal_reason = ?, withdrawn_at = ?, updated_at = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(value.Version), string(value.Lifecycle), value.WithdrawalReason,
		encodeOptionalTime(value.WithdrawnAt), encodeTime(value.UpdatedAt),
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
		version, capturedAt, registeredAt, updatedAt                                                              int64
		evidenceType, description, sourceJSON, producerJSON, sourceVersion, checksum, lifecycle, retractionReason string
		artifactRaw, measurementRaw                                                                               sql.NullString
		retractedAt                                                                                               sql.NullInt64
	)
	err := r.uow.tx.QueryRowContext(ctx, `
SELECT version, evidence_type, description, source_ref_json, producer_ref_json, captured_at,
       registered_at, artifact_id, measurement_json, source_version, checksum, lifecycle,
       retraction_reason, retracted_at, updated_at
FROM evidence
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(),
	).Scan(&version, &evidenceType, &description, &sourceJSON, &producerJSON, &capturedAt,
		&registeredAt, &artifactRaw, &measurementRaw, &sourceVersion, &checksum, &lifecycle,
		&retractionReason, &retractedAt, &updatedAt)
	if err != nil {
		return domain.Evidence{}, mapSQLError("get evidence", err)
	}
	var source domain.SourceReference
	if err := unmarshalJSON(sourceJSON, &source); err != nil {
		return domain.Evidence{}, err
	}
	var producer domain.ActorRef
	if err := unmarshalJSON(producerJSON, &producer); err != nil {
		return domain.Evidence{}, err
	}
	value := domain.Evidence{
		ID: id, Scope: scope, Version: domain.Version(version), EvidenceType: domain.EvidenceType(evidenceType),
		Description: description, SourceRef: source, ProducerRef: producer, CapturedAt: decodeTime(capturedAt),
		RegisteredAt: decodeTime(registeredAt), SourceVersion: sourceVersion, Checksum: checksum,
		Lifecycle: domain.EvidenceLifecycle(lifecycle), RetractionReason: retractionReason, UpdatedAt: decodeTime(updatedAt),
	}
	if artifactRaw.Valid {
		parsed, err := domain.ParseID(artifactRaw.String)
		if err != nil {
			return domain.Evidence{}, err
		}
		value.ArtifactID = &parsed
	}
	if measurementRaw.Valid {
		var measurement domain.Measurement
		if err := unmarshalJSON(measurementRaw.String, &measurement); err != nil {
			return domain.Evidence{}, err
		}
		value.Measurement = &measurement
	}
	if retractedAt.Valid {
		at := decodeTime(retractedAt.Int64)
		value.RetractedAt = &at
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
	source, err := marshalJSON(value.SourceRef)
	if err != nil {
		return err
	}
	producer, err := marshalJSON(value.ProducerRef)
	if err != nil {
		return err
	}
	var measurement any
	if value.Measurement != nil {
		encoded, err := marshalJSON(value.Measurement)
		if err != nil {
			return err
		}
		measurement = encoded
	}
	_, err = r.uow.tx.ExecContext(ctx, `
INSERT INTO evidence (
    id, namespace_id, outcome_id, kind, version, evidence_type, description, source_ref_json,
    producer_ref_json, captured_at, registered_at, artifact_id, measurement_json, source_version,
    checksum, lifecycle, retraction_reason, retracted_at, updated_at
) VALUES (?, ?, ?, 'evidence', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		value.ID.String(), value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(),
		int64(value.Version), string(value.EvidenceType), value.Description, source, producer,
		encodeTime(value.CapturedAt), encodeTime(value.RegisteredAt), nullableID(value.ArtifactID),
		measurement, value.SourceVersion, value.Checksum, string(value.Lifecycle),
		value.RetractionReason, encodeOptionalTime(value.RetractedAt), encodeTime(value.UpdatedAt),
	)
	if err != nil {
		return mapSQLError("insert evidence", err)
	}
	return nil
}

func (r evidenceRepository) Save(ctx context.Context, value domain.Evidence, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "evidence version must advance exactly once per save")
	}
	current, err := r.Get(ctx, value.Scope, value.ID)
	if err != nil {
		return err
	}
	if !sameEvidenceContent(current, value) {
		return domain.NewError(domain.ErrorCodeEvidence, "evidence content is immutable")
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE evidence
SET version = ?, lifecycle = ?, retraction_reason = ?, retracted_at = ?, updated_at = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(value.Version), string(value.Lifecycle), value.RetractionReason, encodeOptionalTime(value.RetractedAt),
		encodeTime(value.UpdatedAt), value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(),
		value.ID.String(), int64(expected),
	)
	if err != nil {
		return mapSQLError("save evidence", err)
	}
	return requireVersionedUpdate(ctx, r.uow.tx, result, "evidence", value.Scope.NamespaceID, value.Scope.OutcomeID, value.ID, expected)
}

type evidenceLinkRepository struct{ uow *unitOfWork }

func (r evidenceLinkRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.EvidenceLink, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return domain.EvidenceLink{}, err
	}
	var (
		version, createdAt, updatedAt                                                      int64
		evidenceRaw, targetRaw, targetKind, stance, rationale, lifecycle, retractionReason string
		criterionRaw                                                                       sql.NullString
		retractedAt                                                                        sql.NullInt64
	)
	err := r.uow.tx.QueryRowContext(ctx, `
SELECT version, evidence_id, target_id, target_kind, criterion_id, stance, rationale,
       lifecycle, retraction_reason, retracted_at, created_at, updated_at
FROM evidence_links
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(),
	).Scan(&version, &evidenceRaw, &targetRaw, &targetKind, &criterionRaw, &stance, &rationale,
		&lifecycle, &retractionReason, &retractedAt, &createdAt, &updatedAt)
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
		ID: id, Scope: scope, Version: domain.Version(version), EvidenceID: evidenceID,
		TargetRef: domain.EntityRef{Scope: scope, Kind: kind, ID: targetID}, Stance: domain.EvidenceStance(stance),
		Rationale: rationale, Lifecycle: domain.EvidenceLinkLifecycle(lifecycle), RetractionReason: retractionReason,
		CreatedAt: decodeTime(createdAt), UpdatedAt: decodeTime(updatedAt),
	}
	if criterionRaw.Valid {
		parsed, err := domain.ParseID(criterionRaw.String)
		if err != nil {
			return domain.EvidenceLink{}, err
		}
		value.CriterionID = &parsed
	}
	if retractedAt.Valid {
		at := decodeTime(retractedAt.Int64)
		value.RetractedAt = &at
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
    criterion_id, stance, rationale, lifecycle, retraction_reason, retracted_at, created_at, updated_at
) VALUES (?, ?, ?, 'evidence_link', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		value.ID.String(), value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(),
		int64(value.Version), value.EvidenceID.String(), value.TargetRef.ID.String(), value.TargetRef.Kind.String(),
		nullableID(value.CriterionID), string(value.Stance), value.Rationale, string(value.Lifecycle),
		value.RetractionReason, encodeOptionalTime(value.RetractedAt), encodeTime(value.CreatedAt), encodeTime(value.UpdatedAt),
	)
	if err != nil {
		return mapSQLError("insert evidence link", err)
	}
	return nil
}

func (r evidenceLinkRepository) Save(ctx context.Context, value domain.EvidenceLink, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "evidence link version must advance exactly once per save")
	}
	current, err := r.Get(ctx, value.Scope, value.ID)
	if err != nil {
		return err
	}
	if !sameEvidenceLinkContent(current, value) {
		return domain.NewError(domain.ErrorCodeEvidenceLink, "evidence link content is immutable")
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE evidence_links
SET version = ?, lifecycle = ?, retraction_reason = ?, retracted_at = ?, updated_at = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(value.Version), string(value.Lifecycle), value.RetractionReason, encodeOptionalTime(value.RetractedAt),
		encodeTime(value.UpdatedAt), value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(),
		value.ID.String(), int64(expected),
	)
	if err != nil {
		return mapSQLError("save evidence link", err)
	}
	return requireVersionedUpdate(ctx, r.uow.tx, result, "evidence_links", value.Scope.NamespaceID, value.Scope.OutcomeID, value.ID, expected)
}

type decisionRepository struct{ uow *unitOfWork }

func (r decisionRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Decision, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return domain.Decision{}, err
	}
	var (
		version, createdAt, updatedAt                                                                    int64
		title, proposal, chosen, rationale, alternativesJSON, lifecycle, proposedByJSON, rejectionReason string
		decidedByJSON, supersedesRaw                                                                     sql.NullString
		decidedAt                                                                                        sql.NullInt64
	)
	err := r.uow.tx.QueryRowContext(ctx, `
SELECT version, title, proposal, chosen_alternative, rationale, alternatives_json, lifecycle,
       proposed_by_json, decided_by_json, decided_at, rejection_reason, supersedes_decision_id,
       created_at, updated_at
FROM decisions
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(),
	).Scan(&version, &title, &proposal, &chosen, &rationale, &alternativesJSON, &lifecycle,
		&proposedByJSON, &decidedByJSON, &decidedAt, &rejectionReason, &supersedesRaw, &createdAt, &updatedAt)
	if err != nil {
		return domain.Decision{}, mapSQLError("get decision", err)
	}
	var proposedBy domain.ActorRef
	if err := unmarshalJSON(proposedByJSON, &proposedBy); err != nil {
		return domain.Decision{}, err
	}
	var alternatives []string
	if err := unmarshalJSON(alternativesJSON, &alternatives); err != nil {
		return domain.Decision{}, err
	}
	value := domain.Decision{
		ID: id, Scope: scope, Version: domain.Version(version), Title: title, Proposal: proposal,
		ChosenAlternative: chosen, Rationale: rationale, Alternatives: alternatives,
		Lifecycle: domain.DecisionLifecycle(lifecycle), ProposedBy: proposedBy, RejectionReason: rejectionReason,
		CreatedAt: decodeTime(createdAt), UpdatedAt: decodeTime(updatedAt),
	}
	if decidedByJSON.Valid {
		var actor domain.ActorRef
		if err := unmarshalJSON(decidedByJSON.String, &actor); err != nil {
			return domain.Decision{}, err
		}
		value.DecidedBy = &actor
	}
	if decidedAt.Valid {
		at := decodeTime(decidedAt.Int64)
		value.DecidedAt = &at
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
	proposedBy, err := marshalJSON(value.ProposedBy)
	if err != nil {
		return err
	}
	alternatives, err := marshalJSON(value.Alternatives)
	if err != nil {
		return err
	}
	var decidedBy any
	if value.DecidedBy != nil {
		encoded, err := marshalJSON(value.DecidedBy)
		if err != nil {
			return err
		}
		decidedBy = encoded
	}
	_, err = r.uow.tx.ExecContext(ctx, `
INSERT INTO decisions (
    id, namespace_id, outcome_id, kind, version, title, proposal, chosen_alternative, rationale,
    alternatives_json, lifecycle, proposed_by_json, decided_by_json, decided_at, rejection_reason,
    supersedes_decision_id, created_at, updated_at
) VALUES (?, ?, ?, 'decision', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		value.ID.String(), value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(), int64(value.Version),
		value.Title, value.Proposal, value.ChosenAlternative, value.Rationale, alternatives, string(value.Lifecycle),
		proposedBy, decidedBy, encodeOptionalTime(value.DecidedAt), value.RejectionReason,
		nullableID(value.SupersedesDecisionID), encodeTime(value.CreatedAt), encodeTime(value.UpdatedAt))
	if err != nil {
		return mapSQLError("insert decision", err)
	}
	return nil
}

func (r decisionRepository) Save(ctx context.Context, value domain.Decision, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "decision version must advance exactly once per save")
	}
	current, err := r.Get(ctx, value.Scope, value.ID)
	if err != nil {
		return err
	}
	if current.Lifecycle != domain.DecisionLifecycleProposed && !sameDecisionContent(current, value) {
		return domain.NewError(domain.ErrorCodeDecision, "accepted/rejected/superseded decision content is immutable")
	}
	proposedBy, err := marshalJSON(value.ProposedBy)
	if err != nil {
		return err
	}
	alternatives, err := marshalJSON(value.Alternatives)
	if err != nil {
		return err
	}
	var decidedBy any
	if value.DecidedBy != nil {
		encoded, err := marshalJSON(value.DecidedBy)
		if err != nil {
			return err
		}
		decidedBy = encoded
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE decisions
SET version = ?, title = ?, proposal = ?, chosen_alternative = ?, rationale = ?,
    alternatives_json = ?, lifecycle = ?, proposed_by_json = ?, decided_by_json = ?,
    decided_at = ?, rejection_reason = ?, supersedes_decision_id = ?, updated_at = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(value.Version), value.Title, value.Proposal, value.ChosenAlternative, value.Rationale,
		alternatives, string(value.Lifecycle), proposedBy, decidedBy, encodeOptionalTime(value.DecidedAt),
		value.RejectionReason, nullableID(value.SupersedesDecisionID), encodeTime(value.UpdatedAt),
		value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(), value.ID.String(), int64(expected))
	if err != nil {
		return mapSQLError("save decision", err)
	}
	return requireVersionedUpdate(ctx, r.uow.tx, result, "decisions", value.Scope.NamespaceID, value.Scope.OutcomeID, value.ID, expected)
}

func listDocumentaryIDs(ctx context.Context, uow *unitOfWork, table string, scope domain.Scope) ([]domain.ID, error) {
	if err := uow.ensureOpen(); err != nil {
		return nil, err
	}
	query := "SELECT id FROM " + table + " WHERE namespace_id = ? AND outcome_id = ? ORDER BY id"
	rows, err := uow.tx.QueryContext(ctx, query, scope.NamespaceID.String(), scope.OutcomeID.String())
	if err != nil {
		return nil, mapSQLError("list "+table, err)
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

func sameArtifactContent(a, b domain.Artifact) bool {
	return a.ID == b.ID && a.Scope == b.Scope && a.ArtifactType == b.ArtifactType && a.Name == b.Name && a.URI == b.URI &&
		a.MediaType == b.MediaType && a.Checksum == b.Checksum && a.SourceVersion == b.SourceVersion && a.ProducerRef == b.ProducerRef &&
		optionalTimeEqual(a.ProducedAt, b.ProducedAt) && a.RegisteredAt.Equal(b.RegisteredAt)
}

func sameEvidenceContent(a, b domain.Evidence) bool {
	return a.ID == b.ID && a.Scope == b.Scope && a.EvidenceType == b.EvidenceType && a.Description == b.Description &&
		a.SourceRef == b.SourceRef && a.ProducerRef == b.ProducerRef && a.CapturedAt.Equal(b.CapturedAt) && a.RegisteredAt.Equal(b.RegisteredAt) &&
		optionalIDEqual(a.ArtifactID, b.ArtifactID) && optionalMeasurementEqual(a.Measurement, b.Measurement) &&
		a.SourceVersion == b.SourceVersion && a.Checksum == b.Checksum
}

func sameEvidenceLinkContent(a, b domain.EvidenceLink) bool {
	return a.ID == b.ID && a.Scope == b.Scope && a.EvidenceID == b.EvidenceID && a.TargetRef == b.TargetRef &&
		optionalIDEqual(a.CriterionID, b.CriterionID) && a.Stance == b.Stance && a.Rationale == b.Rationale && a.CreatedAt.Equal(b.CreatedAt)
}

func sameDecisionContent(a, b domain.Decision) bool {
	if a.ID != b.ID || a.Scope != b.Scope || a.Title != b.Title || a.Proposal != b.Proposal || a.ChosenAlternative != b.ChosenAlternative ||
		a.Rationale != b.Rationale || a.ProposedBy != b.ProposedBy || !optionalIDEqual(a.SupersedesDecisionID, b.SupersedesDecisionID) || len(a.Alternatives) != len(b.Alternatives) {
		return false
	}
	for i := range a.Alternatives {
		if a.Alternatives[i] != b.Alternatives[i] {
			return false
		}
	}
	return true
}

func optionalIDEqual(a, b *domain.ID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
func optionalTimeEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}
func optionalMeasurementEqual(a, b *domain.Measurement) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
