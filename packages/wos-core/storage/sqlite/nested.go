package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type criterionDefinition struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

func syncOwnedState(
	ctx context.Context,
	tx *sql.Tx,
	owner domain.EntityRef,
	actorRole string,
	actors []domain.ActorRef,
	criteria domain.CriterionSet,
	current *domain.Conclusion,
	history []domain.Conclusion,
	ownerVersion domain.Version,
	lifecycleResult string,
	changedAt time.Time,
) error {
	if err := syncActorLinks(ctx, tx, owner, actorRole, actors); err != nil {
		return err
	}
	if err := syncCriteria(ctx, tx, owner, criteria, changedAt); err != nil {
		return err
	}
	if err := syncConclusions(ctx, tx, owner, current, history, ownerVersion, lifecycleResult); err != nil {
		return err
	}
	return nil
}

func loadOwnedState(
	ctx context.Context,
	tx *sql.Tx,
	owner domain.EntityRef,
	actorRole string,
) ([]domain.ActorRef, domain.CriterionSet, *domain.Conclusion, []domain.Conclusion, error) {
	actors, err := loadActorLinks(ctx, tx, owner, actorRole)
	if err != nil {
		return nil, domain.CriterionSet{}, nil, nil, err
	}
	criteria, err := loadCriteria(ctx, tx, owner)
	if err != nil {
		return nil, domain.CriterionSet{}, nil, nil, err
	}
	current, history, err := loadConclusions(ctx, tx, owner)
	if err != nil {
		return nil, domain.CriterionSet{}, nil, nil, err
	}
	return actors, criteria, current, history, nil
}

func syncActorLinks(
	ctx context.Context,
	tx *sql.Tx,
	owner domain.EntityRef,
	role string,
	actors []domain.ActorRef,
) error {
	if role != "owner" && role != "assignee" {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "invalid actor link role")
	}
	if _, err := tx.ExecContext(ctx, `
DELETE FROM entity_actor_links
WHERE namespace_id = ? AND outcome_id = ? AND entity_id = ? AND role = ?`,
		owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String(), role,
	); err != nil {
		return mapSQLError("replace actor links", err)
	}
	sorted := append([]domain.ActorRef(nil), actors...)
	sort.Slice(sorted, func(i, j int) bool {
		left := string(sorted[i].Kind) + "\x00" + sorted[i].Provider + "\x00" + sorted[i].ID
		right := string(sorted[j].Kind) + "\x00" + sorted[j].Provider + "\x00" + sorted[j].ID
		return left < right
	})
	for _, actor := range sorted {
		if err := actor.Validate(); err != nil {
			return err
		}
		encoded, err := marshalJSON(actor)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO entity_actor_links (
    namespace_id, outcome_id, entity_id, role,
    actor_kind, actor_provider, actor_id, actor_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			owner.NamespaceID.String(),
			owner.OutcomeID.String(),
			owner.ID.String(),
			role,
			actor.Kind.String(),
			actor.Provider,
			actor.ID,
			encoded,
		); err != nil {
			return mapSQLError("insert actor link", err)
		}
	}
	return nil
}

func loadActorLinks(
	ctx context.Context,
	tx *sql.Tx,
	owner domain.EntityRef,
	role string,
) ([]domain.ActorRef, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT actor_json
FROM entity_actor_links
WHERE namespace_id = ? AND outcome_id = ? AND entity_id = ? AND role = ?
ORDER BY actor_kind, actor_provider, actor_id`,
		owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String(), role,
	)
	if err != nil {
		return nil, mapSQLError("load actor links", err)
	}
	defer rows.Close()
	result := make([]domain.ActorRef, 0)
	for rows.Next() {
		var encoded string
		if err := rows.Scan(&encoded); err != nil {
			return nil, err
		}
		var actor domain.ActorRef
		if err := unmarshalJSON(encoded, &actor); err != nil {
			return nil, domain.WrapError(domain.ErrorCodeInvalidArgument, "persisted actor link is invalid", err)
		}
		if err := actor.Validate(); err != nil {
			return nil, err
		}
		result = append(result, actor)
	}
	return result, rows.Err()
}

func syncCriteria(
	ctx context.Context,
	tx *sql.Tx,
	owner domain.EntityRef,
	set domain.CriterionSet,
	changedAt time.Time,
) error {
	if err := set.ValidateForOwner(owner); err != nil {
		return err
	}
	for _, criterion := range set.Items {
		_, err := tx.ExecContext(ctx, `
INSERT INTO success_criteria (
    id, namespace_id, outcome_id, owner_id, owner_kind,
    criterion_revision, status, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    criterion_revision = excluded.criterion_revision,
    status = excluded.status`,
			criterion.ID.String(),
			owner.NamespaceID.String(),
			owner.OutcomeID.String(),
			owner.ID.String(),
			owner.Kind.String(),
			int64(criterion.Revision),
			string(criterion.Status),
			encodeTime(changedAt),
		)
		if err != nil {
			return mapSQLError("upsert success criterion", err)
		}

		if _, err := tx.ExecContext(ctx, `
DELETE FROM criterion_current_assessments
WHERE namespace_id = ? AND outcome_id = ? AND criterion_id = ?`,
			owner.NamespaceID.String(), owner.OutcomeID.String(), criterion.ID.String(),
		); err != nil {
			return mapSQLError("clear current criterion assessment", err)
		}
	}

	for _, revision := range set.DefinitionRevisions {
		definition, err := marshalJSON(criterionDefinition{
			Title:       revision.Title,
			Description: revision.Description,
		})
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO criterion_revisions (
    namespace_id, outcome_id, criterion_id, criterion_revision,
    definition_json, required, verification_mode, status, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			owner.NamespaceID.String(),
			owner.OutcomeID.String(),
			revision.CriterionID.String(),
			int64(revision.Revision),
			definition,
			boolInt(revision.Required),
			string(revision.VerificationMode),
			string(revision.Status),
			encodeTime(changedAt),
		); err != nil {
			return mapSQLError("insert criterion revision", err)
		}
		if err := verifyCriterionRevision(ctx, tx, owner.Scope, revision, definition); err != nil {
			return err
		}
	}

	for _, assessment := range set.Assessments {
		if err := ensurePrincipal(ctx, tx, assessment.PrincipalID); err != nil {
			return err
		}
		actorJSON, err := marshalJSON(assessment.Actor)
		if err != nil {
			return err
		}
		var evaluatorJSON any
		if assessment.EvaluatorRef != nil {
			encoded, err := marshalJSON(assessment.EvaluatorRef)
			if err != nil {
				return err
			}
			evaluatorJSON = encoded
		}
		if _, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO criterion_assessments (
    id, namespace_id, outcome_id, criterion_id, criterion_revision,
    result, rationale, principal_id, actor_json, assessed_at,
    evaluator_ref_json, supersedes_assessment_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			assessment.ID.String(),
			owner.NamespaceID.String(),
			owner.OutcomeID.String(),
			assessment.CriterionID.String(),
			int64(assessment.CriterionRevision),
			string(assessment.Result),
			assessment.Rationale,
			assessment.PrincipalID,
			actorJSON,
			encodeTime(assessment.AssessedAt),
			evaluatorJSON,
			nullableID(assessment.SupersedesAssessmentID),
		); err != nil {
			return mapSQLError("insert criterion assessment", err)
		}
		for _, evidenceID := range assessment.EvidenceIDs {
			if _, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO criterion_assessment_evidence (
    namespace_id, outcome_id, assessment_id, evidence_id
) VALUES (?, ?, ?, ?)`,
				owner.NamespaceID.String(),
				owner.OutcomeID.String(),
				assessment.ID.String(),
				evidenceID.String(),
			); err != nil {
				return mapSQLError("insert criterion assessment evidence", err)
			}
		}
		if err := verifyCriterionAssessment(ctx, tx, owner.Scope, assessment); err != nil {
			return err
		}
	}

	for criterionID, assessment := range set.CurrentAssessments {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO criterion_current_assessments (
    namespace_id, outcome_id, criterion_id, assessment_id
) VALUES (?, ?, ?, ?)`,
			owner.NamespaceID.String(),
			owner.OutcomeID.String(),
			criterionID.String(),
			assessment.ID.String(),
		); err != nil {
			return mapSQLError("set current criterion assessment", err)
		}
	}
	return nil
}

func verifyCriterionRevision(
	ctx context.Context,
	tx *sql.Tx,
	scope domain.Scope,
	revision domain.CriterionDefinitionRevision,
	definition string,
) error {
	var existingDefinition, mode, status string
	var required int
	err := tx.QueryRowContext(ctx, `
SELECT definition_json, required, verification_mode, status
FROM criterion_revisions
WHERE namespace_id = ? AND outcome_id = ? AND criterion_id = ? AND criterion_revision = ?`,
		scope.NamespaceID.String(),
		scope.OutcomeID.String(),
		revision.CriterionID.String(),
		int64(revision.Revision),
	).Scan(&existingDefinition, &required, &mode, &status)
	if err != nil {
		return mapSQLError("verify criterion revision", err)
	}
	if existingDefinition != definition ||
		required != boolInt(revision.Required) ||
		mode != string(revision.VerificationMode) ||
		status != string(revision.Status) {
		return domain.NewError(
			domain.ErrorCodeCriterion,
			"criterion revision is immutable and differs from persisted definition",
		)
	}
	return nil
}

func verifyCriterionAssessment(
	ctx context.Context,
	tx *sql.Tx,
	scope domain.Scope,
	assessment domain.CriterionAssessment,
) error {
	var (
		rawCriterionID, result, rationale, principalID, actorJSON string
		revision, assessedAt                                      int64
		evaluatorJSON, supersedes                                 sql.NullString
	)
	err := tx.QueryRowContext(ctx, `
SELECT criterion_id, criterion_revision, result, rationale, principal_id,
       actor_json, assessed_at, evaluator_ref_json, supersedes_assessment_id
FROM criterion_assessments
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(),
		scope.OutcomeID.String(),
		assessment.ID.String(),
	).Scan(
		&rawCriterionID,
		&revision,
		&result,
		&rationale,
		&principalID,
		&actorJSON,
		&assessedAt,
		&evaluatorJSON,
		&supersedes,
	)
	if err != nil {
		return mapSQLError("verify criterion assessment", err)
	}

	expectedActorJSON, err := marshalJSON(assessment.Actor)
	if err != nil {
		return err
	}
	var expectedEvaluatorJSON string
	if assessment.EvaluatorRef != nil {
		expectedEvaluatorJSON, err = marshalJSON(assessment.EvaluatorRef)
		if err != nil {
			return err
		}
	}
	expectedSupersedes := ""
	if assessment.SupersedesAssessmentID != nil {
		expectedSupersedes = assessment.SupersedesAssessmentID.String()
	}

	if rawCriterionID != assessment.CriterionID.String() ||
		revision != int64(assessment.CriterionRevision) ||
		result != string(assessment.Result) ||
		rationale != assessment.Rationale ||
		principalID != assessment.PrincipalID ||
		actorJSON != expectedActorJSON ||
		assessedAt != encodeTime(assessment.AssessedAt) ||
		evaluatorJSON.Valid != (assessment.EvaluatorRef != nil) ||
		(evaluatorJSON.Valid && evaluatorJSON.String != expectedEvaluatorJSON) ||
		supersedes.Valid != (assessment.SupersedesAssessmentID != nil) ||
		(supersedes.Valid && supersedes.String != expectedSupersedes) {
		return domain.NewError(
			domain.ErrorCodeAssessment,
			"criterion assessment is immutable and differs from persisted history",
		)
	}

	persistedEvidence, err := loadAssessmentEvidence(ctx, tx, scope, assessment.ID)
	if err != nil {
		return err
	}
	expectedEvidence := append([]domain.ID(nil), assessment.EvidenceIDs...)
	sort.Slice(expectedEvidence, func(i, j int) bool {
		return expectedEvidence[i].String() < expectedEvidence[j].String()
	})
	if len(persistedEvidence) != len(expectedEvidence) {
		return domain.NewError(
			domain.ErrorCodeAssessment,
			"criterion assessment Evidence references are immutable",
		)
	}
	for i := range expectedEvidence {
		if persistedEvidence[i] != expectedEvidence[i] {
			return domain.NewError(
				domain.ErrorCodeAssessment,
				"criterion assessment Evidence references are immutable",
			)
		}
	}
	return nil
}

func loadCriteria(
	ctx context.Context,
	tx *sql.Tx,
	owner domain.EntityRef,
) (domain.CriterionSet, error) {
	set := domain.NewCriterionSet()
	rows, err := tx.QueryContext(ctx, `
SELECT c.id, c.criterion_revision, c.status,
       r.definition_json, r.required, r.verification_mode
FROM success_criteria c
JOIN criterion_revisions r
  ON r.namespace_id = c.namespace_id
 AND r.outcome_id = c.outcome_id
 AND r.criterion_id = c.id
 AND r.criterion_revision = c.criterion_revision
WHERE c.namespace_id = ? AND c.outcome_id = ? AND c.owner_id = ? AND c.owner_kind = ?
ORDER BY c.created_at, c.id`,
		owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String(), owner.Kind.String(),
	)
	if err != nil {
		return domain.CriterionSet{}, mapSQLError("load criteria", err)
	}
	for rows.Next() {
		var (
			rawID, status, definition, mode string
			revision                        int64
			required                        int
		)
		if err := rows.Scan(&rawID, &revision, &status, &definition, &required, &mode); err != nil {
			rows.Close()
			return domain.CriterionSet{}, err
		}
		id, err := domain.ParseID(rawID)
		if err != nil {
			rows.Close()
			return domain.CriterionSet{}, err
		}
		var def criterionDefinition
		if err := unmarshalJSON(definition, &def); err != nil {
			rows.Close()
			return domain.CriterionSet{}, err
		}
		set.Items = append(set.Items, domain.SuccessCriterion{
			ID:               id,
			OwnerRef:         owner,
			Title:            def.Title,
			Description:      def.Description,
			Required:         required == 1,
			Revision:         domain.CriterionRevision(revision),
			VerificationMode: domain.VerificationMode(mode),
			Status:           domain.CriterionStatus(status),
		})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return domain.CriterionSet{}, err
	}
	rows.Close()

	revisionRows, err := tx.QueryContext(ctx, `
SELECT r.criterion_id, r.criterion_revision, r.definition_json, r.required, r.verification_mode, r.status
FROM criterion_revisions r
JOIN success_criteria c
  ON c.namespace_id = r.namespace_id
 AND c.outcome_id = r.outcome_id
 AND c.id = r.criterion_id
WHERE c.namespace_id = ? AND c.outcome_id = ? AND c.owner_id = ? AND c.owner_kind = ?
ORDER BY r.criterion_id, r.criterion_revision`,
		owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String(), owner.Kind.String(),
	)
	if err != nil {
		return domain.CriterionSet{}, mapSQLError("load criterion revision history", err)
	}
	for revisionRows.Next() {
		var rawCriterionID, definition, mode, status string
		var revision int64
		var required int
		if err := revisionRows.Scan(&rawCriterionID, &revision, &definition, &required, &mode, &status); err != nil {
			revisionRows.Close()
			return domain.CriterionSet{}, err
		}
		criterionID, err := domain.ParseID(rawCriterionID)
		if err != nil {
			revisionRows.Close()
			return domain.CriterionSet{}, err
		}
		var def criterionDefinition
		if err := unmarshalJSON(definition, &def); err != nil {
			revisionRows.Close()
			return domain.CriterionSet{}, err
		}
		value := domain.CriterionDefinitionRevision{
			CriterionID:      criterionID,
			OwnerRef:         owner,
			Revision:         domain.CriterionRevision(revision),
			Title:            def.Title,
			Description:      def.Description,
			Required:         required == 1,
			VerificationMode: domain.VerificationMode(mode),
			Status:           domain.CriterionStatus(status),
		}
		if err := value.Validate(); err != nil {
			revisionRows.Close()
			return domain.CriterionSet{}, err
		}
		set.DefinitionRevisions = append(set.DefinitionRevisions, value)
	}
	if err := revisionRows.Err(); err != nil {
		revisionRows.Close()
		return domain.CriterionSet{}, err
	}
	revisionRows.Close()

	assessmentRows, err := tx.QueryContext(ctx, `
SELECT a.id, a.criterion_id, a.criterion_revision, a.result, a.rationale,
       a.principal_id, a.actor_json, a.assessed_at, a.evaluator_ref_json,
       a.supersedes_assessment_id
FROM criterion_assessments a
JOIN success_criteria c
  ON c.namespace_id = a.namespace_id
 AND c.outcome_id = a.outcome_id
 AND c.id = a.criterion_id
WHERE c.namespace_id = ? AND c.outcome_id = ? AND c.owner_id = ? AND c.owner_kind = ?
ORDER BY a.assessed_at, a.id`,
		owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String(), owner.Kind.String(),
	)
	if err != nil {
		return domain.CriterionSet{}, mapSQLError("load criterion assessments", err)
	}
	byID := make(map[domain.ID]domain.CriterionAssessment)
	for assessmentRows.Next() {
		assessment, err := scanAssessment(assessmentRows)
		if err != nil {
			assessmentRows.Close()
			return domain.CriterionSet{}, err
		}
		evidenceIDs, err := loadAssessmentEvidence(ctx, tx, owner.Scope, assessment.ID)
		if err != nil {
			assessmentRows.Close()
			return domain.CriterionSet{}, err
		}
		assessment.EvidenceIDs = evidenceIDs
		if err := assessment.Validate(); err != nil {
			assessmentRows.Close()
			return domain.CriterionSet{}, err
		}
		set.Assessments = append(set.Assessments, assessment)
		byID[assessment.ID] = assessment
	}
	if err := assessmentRows.Err(); err != nil {
		assessmentRows.Close()
		return domain.CriterionSet{}, err
	}
	assessmentRows.Close()

	currentRows, err := tx.QueryContext(ctx, `
SELECT criterion_id, assessment_id
FROM criterion_current_assessments
WHERE namespace_id = ? AND outcome_id = ?
  AND criterion_id IN (
      SELECT id FROM success_criteria
      WHERE namespace_id = ? AND outcome_id = ? AND owner_id = ? AND owner_kind = ?
  )
ORDER BY criterion_id`,
		owner.NamespaceID.String(), owner.OutcomeID.String(),
		owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String(), owner.Kind.String(),
	)
	if err != nil {
		return domain.CriterionSet{}, mapSQLError("load current assessments", err)
	}
	defer currentRows.Close()
	for currentRows.Next() {
		var criterionRaw, assessmentRaw string
		if err := currentRows.Scan(&criterionRaw, &assessmentRaw); err != nil {
			return domain.CriterionSet{}, err
		}
		criterionID, err := domain.ParseID(criterionRaw)
		if err != nil {
			return domain.CriterionSet{}, err
		}
		assessmentID, err := domain.ParseID(assessmentRaw)
		if err != nil {
			return domain.CriterionSet{}, err
		}
		assessment, ok := byID[assessmentID]
		if !ok {
			return domain.CriterionSet{}, domain.NewError(
				domain.ErrorCodeAssessment,
				"persisted current assessment is missing from assessment history",
			)
		}
		set.CurrentAssessments[criterionID] = assessment
	}
	if err := currentRows.Err(); err != nil {
		return domain.CriterionSet{}, err
	}
	return set, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAssessment(scanner rowScanner) (domain.CriterionAssessment, error) {
	var (
		rawID, rawCriterion, result, rationale, principal, actorJSON string
		revision, assessedAt                                         int64
		evaluatorJSON, supersedes                                    sql.NullString
	)
	if err := scanner.Scan(
		&rawID, &rawCriterion, &revision, &result, &rationale,
		&principal, &actorJSON, &assessedAt, &evaluatorJSON, &supersedes,
	); err != nil {
		return domain.CriterionAssessment{}, err
	}
	id, err := domain.ParseID(rawID)
	if err != nil {
		return domain.CriterionAssessment{}, err
	}
	criterionID, err := domain.ParseID(rawCriterion)
	if err != nil {
		return domain.CriterionAssessment{}, err
	}
	var actor domain.ActorRef
	if err := unmarshalJSON(actorJSON, &actor); err != nil {
		return domain.CriterionAssessment{}, err
	}
	value := domain.CriterionAssessment{
		ID:                id,
		CriterionID:       criterionID,
		CriterionRevision: domain.CriterionRevision(revision),
		Result:            domain.AssessmentResult(result),
		Rationale:         rationale,
		PrincipalID:       principal,
		Actor:             actor,
		AssessedAt:        decodeTime(assessedAt),
	}
	if evaluatorJSON.Valid {
		var evaluator domain.EvaluatorRef
		if err := unmarshalJSON(evaluatorJSON.String, &evaluator); err != nil {
			return domain.CriterionAssessment{}, err
		}
		value.EvaluatorRef = &evaluator
	}
	if supersedes.Valid {
		parsed, err := domain.ParseID(supersedes.String)
		if err != nil {
			return domain.CriterionAssessment{}, err
		}
		value.SupersedesAssessmentID = &parsed
	}
	return value, value.Validate()
}

func loadAssessmentEvidence(
	ctx context.Context,
	tx *sql.Tx,
	scope domain.Scope,
	assessmentID domain.ID,
) ([]domain.ID, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT evidence_id
FROM criterion_assessment_evidence
WHERE namespace_id = ? AND outcome_id = ? AND assessment_id = ?
ORDER BY evidence_id`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), assessmentID.String(),
	)
	if err != nil {
		return nil, mapSQLError("load criterion assessment evidence", err)
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

func nullableConclusionID(id domain.ID) any {
	if id.IsZero() {
		return nil
	}
	return id.String()
}

func syncConclusions(
	ctx context.Context,
	tx *sql.Tx,
	owner domain.EntityRef,
	current *domain.Conclusion,
	history []domain.Conclusion,
	ownerVersion domain.Version,
	lifecycleResult string,
) error {
	if _, err := tx.ExecContext(ctx, `
DELETE FROM aggregate_current_conclusions
WHERE namespace_id = ? AND outcome_id = ? AND owner_id = ?`,
		owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String(),
	); err != nil {
		return mapSQLError("clear current conclusion", err)
	}

	all := make([]domain.Conclusion, 0, len(history)+1)
	all = append(all, history...)
	if current != nil {
		all = append(all, *current)
	}
	for ordinal, conclusion := range all {
		if err := conclusion.Validate(); err != nil {
			return err
		}
		if err := ensurePrincipal(ctx, tx, conclusion.PrincipalID); err != nil {
			return err
		}
		storageID, err := conclusionStorageID(owner, ordinal, conclusion)
		if err != nil {
			return err
		}
		actorJSON, err := marshalJSON(conclusion.Actor)
		if err != nil {
			return err
		}
		obligationsJSON, err := marshalJSON(conclusion.Obligations)
		if err != nil {
			return err
		}
		result := conclusion.LifecycleResult
		if result == "" {
			result = "historical"
		}
		var persistedOwnerVersion any
		if conclusion.OwnerVersion != nil {
			persistedOwnerVersion = int64(*conclusion.OwnerVersion)
		} else if current != nil && ordinal == len(history) {
			persistedOwnerVersion = int64(ownerVersion)
		}
		if current != nil && ordinal == len(history) && conclusion.LifecycleResult == "" {
			result = lifecycleResult
		}
		if _, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO conclusions (
    id, namespace_id, outcome_id, owner_id, ordinal, owner_version,
    lifecycle_result, principal_id, actor_json, recorded_at, rationale,
    obligations_snapshot_json, public_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			storageID,
			owner.NamespaceID.String(),
			owner.OutcomeID.String(),
			owner.ID.String(),
			int64(ordinal),
			persistedOwnerVersion,
			result,
			conclusion.PrincipalID,
			actorJSON,
			encodeTime(conclusion.ConcludedAt),
			conclusion.Reason,
			obligationsJSON,
			nullableConclusionID(conclusion.ID),
		); err != nil {
			return mapSQLError("insert conclusion", err)
		}
		for _, ref := range conclusion.Assessments {
			if _, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO conclusion_assessments (
    namespace_id, outcome_id, conclusion_id, assessment_id
) VALUES (?, ?, ?, ?)`,
				owner.NamespaceID.String(),
				owner.OutcomeID.String(),
				storageID,
				ref.AssessmentID.String(),
			); err != nil {
				return mapSQLError("insert conclusion assessment", err)
			}
		}
		if !conclusion.ID.IsZero() {
			if _, err := tx.ExecContext(ctx, `
UPDATE conclusions
SET public_id = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND public_id IS NULL`,
				conclusion.ID.String(),
				owner.NamespaceID.String(),
				owner.OutcomeID.String(),
				storageID,
			); err != nil {
				return mapSQLError("backfill conclusion public id", err)
			}
		}
		if err := verifyConclusionRow(
			ctx,
			tx,
			owner,
			storageID,
			conclusion,
			persistedOwnerVersion,
			result,
			obligationsJSON,
		); err != nil {
			return err
		}
		if current != nil && ordinal == len(history) {
			if _, err := tx.ExecContext(ctx, `
INSERT INTO aggregate_current_conclusions (
    namespace_id, outcome_id, owner_id, conclusion_id
) VALUES (?, ?, ?, ?)`,
				owner.NamespaceID.String(),
				owner.OutcomeID.String(),
				owner.ID.String(),
				storageID,
			); err != nil {
				return mapSQLError("set current conclusion", err)
			}
		}
	}
	return nil
}

func verifyConclusionRow(
	ctx context.Context,
	tx *sql.Tx,
	owner domain.EntityRef,
	storageID string,
	conclusion domain.Conclusion,
	expectedOwnerVersion any,
	expectedLifecycleResult string,
	expectedObligationsJSON string,
) error {
	var (
		rawOwnerID, lifecycleResult, principalID, actorJSON, rationale, obligationsJSON string
		recordedAt                                                                      int64
		ownerVersion                                                                    sql.NullInt64
		publicID                                                                        sql.NullString
	)
	err := tx.QueryRowContext(ctx, `
SELECT owner_id, owner_version, lifecycle_result, principal_id, actor_json,
       recorded_at, rationale, obligations_snapshot_json, public_id
FROM conclusions
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		owner.NamespaceID.String(),
		owner.OutcomeID.String(),
		storageID,
	).Scan(
		&rawOwnerID,
		&ownerVersion,
		&lifecycleResult,
		&principalID,
		&actorJSON,
		&recordedAt,
		&rationale,
		&obligationsJSON,
		&publicID,
	)
	if err != nil {
		return mapSQLError("verify conclusion history", err)
	}
	expectedActorJSON, err := marshalJSON(conclusion.Actor)
	if err != nil {
		return err
	}

	var expectedVersion *int64
	switch value := expectedOwnerVersion.(type) {
	case int64:
		copyValue := value
		expectedVersion = &copyValue
	}
	expectedPublicID := ""
	if !conclusion.ID.IsZero() {
		expectedPublicID = conclusion.ID.String()
	}

	if rawOwnerID != owner.ID.String() ||
		lifecycleResult != expectedLifecycleResult ||
		principalID != conclusion.PrincipalID ||
		actorJSON != expectedActorJSON ||
		recordedAt != encodeTime(conclusion.ConcludedAt) ||
		rationale != conclusion.Reason ||
		obligationsJSON != expectedObligationsJSON ||
		ownerVersion.Valid != (expectedVersion != nil) ||
		(ownerVersion.Valid && ownerVersion.Int64 != *expectedVersion) ||
		publicID.Valid != !conclusion.ID.IsZero() ||
		(publicID.Valid && publicID.String != expectedPublicID) {
		return domain.NewError(
			domain.ErrorCodeInvalidArgument,
			"conclusion history is immutable and differs from persisted record",
		)
	}
	return nil
}

func loadConclusions(
	ctx context.Context,
	tx *sql.Tx,
	owner domain.EntityRef,
) (*domain.Conclusion, []domain.Conclusion, error) {
	var currentID sql.NullString
	err := tx.QueryRowContext(ctx, `
SELECT conclusion_id
FROM aggregate_current_conclusions
WHERE namespace_id = ? AND outcome_id = ? AND owner_id = ?`,
		owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String(),
	).Scan(&currentID)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, mapSQLError("load current conclusion id", err)
	}

	rows, err := tx.QueryContext(ctx, `
SELECT id, public_id, owner_version, lifecycle_result, principal_id, actor_json,
       recorded_at, rationale, obligations_snapshot_json
FROM conclusions
WHERE namespace_id = ? AND outcome_id = ? AND owner_id = ?
ORDER BY ordinal`,
		owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String(),
	)
	if err != nil {
		return nil, nil, mapSQLError("load conclusions", err)
	}
	defer rows.Close()

	history := make([]domain.Conclusion, 0)
	var current *domain.Conclusion
	for rows.Next() {
		var storageID, lifecycleResult, principal, actorJSON, rationale, obligationsJSON string
		var publicID sql.NullString
		var recordedAt int64
		var ownerVersion sql.NullInt64
		if err := rows.Scan(
			&storageID, &publicID, &ownerVersion, &lifecycleResult, &principal, &actorJSON,
			&recordedAt, &rationale, &obligationsJSON,
		); err != nil {
			return nil, nil, err
		}
		var actor domain.ActorRef
		if err := unmarshalJSON(actorJSON, &actor); err != nil {
			return nil, nil, err
		}
		refs, err := loadConclusionAssessments(ctx, tx, owner.Scope, storageID)
		if err != nil {
			return nil, nil, err
		}
		var obligations domain.ConclusionObligations
		if err := unmarshalJSON(obligationsJSON, &obligations); err != nil {
			return nil, nil, err
		}
		value := domain.Conclusion{
			PrincipalID: principal,
			Actor:       actor,
			Reason:      rationale,
			ConcludedAt: decodeTime(recordedAt),
			Assessments: refs,
			Obligations: obligations,
		}
		if publicID.Valid {
			parsed, err := domain.ParseID(publicID.String)
			if err != nil {
				return nil, nil, err
			}
			value.ID = parsed
		} else {
			derived, err := legacyConclusionPublicID(storageID, decodeTime(recordedAt))
			if err != nil {
				return nil, nil, err
			}
			value.ID = derived
		}
		if ownerVersion.Valid {
			ownerCopy := owner
			versionCopy := domain.Version(ownerVersion.Int64)
			value.OwnerRef = &ownerCopy
			value.OwnerVersion = &versionCopy
			value.LifecycleResult = lifecycleResult
		}
		if err := value.Validate(); err != nil {
			return nil, nil, err
		}
		if currentID.Valid && storageID == currentID.String {
			copy := value
			current = &copy
		} else {
			history = append(history, value)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return current, history, nil
}

func loadConclusionAssessments(
	ctx context.Context,
	tx *sql.Tx,
	scope domain.Scope,
	conclusionID string,
) ([]domain.CriterionAssessmentRef, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT a.id, a.criterion_id, a.criterion_revision, a.result
FROM conclusion_assessments ca
JOIN criterion_assessments a
  ON a.namespace_id = ca.namespace_id
 AND a.outcome_id = ca.outcome_id
 AND a.id = ca.assessment_id
WHERE ca.namespace_id = ? AND ca.outcome_id = ? AND ca.conclusion_id = ?
ORDER BY a.criterion_id, a.id`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), conclusionID,
	)
	if err != nil {
		return nil, mapSQLError("load conclusion assessments", err)
	}
	defer rows.Close()
	result := make([]domain.CriterionAssessmentRef, 0)
	for rows.Next() {
		var rawAssessment, rawCriterion, rawResult string
		var revision int64
		if err := rows.Scan(&rawAssessment, &rawCriterion, &revision, &rawResult); err != nil {
			return nil, err
		}
		assessmentID, err := domain.ParseID(rawAssessment)
		if err != nil {
			return nil, err
		}
		criterionID, err := domain.ParseID(rawCriterion)
		if err != nil {
			return nil, err
		}
		result = append(result, domain.CriterionAssessmentRef{
			AssessmentID:      assessmentID,
			CriterionID:       criterionID,
			CriterionRevision: domain.CriterionRevision(revision),
			Result:            domain.AssessmentResult(rawResult),
		})
	}
	return result, rows.Err()
}

func actorRefsEqualForStorage(left, right []domain.ActorRef) bool {
	if len(left) != len(right) {
		return false
	}
	l := append([]domain.ActorRef(nil), left...)
	r := append([]domain.ActorRef(nil), right...)
	key := func(a domain.ActorRef) string {
		return fmt.Sprintf("%s\x00%s\x00%s", a.Kind, a.Provider, a.ID)
	}
	sort.Slice(l, func(i, j int) bool { return key(l[i]) < key(l[j]) })
	sort.Slice(r, func(i, j int) bool { return key(r[i]) < key(r[j]) })
	for i := range l {
		if l[i] != r[i] {
			return false
		}
	}
	return true
}
