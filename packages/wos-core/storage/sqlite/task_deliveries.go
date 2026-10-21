package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type taskDeliveryRepository struct{ uow *unitOfWork }

func (r taskDeliveryRepository) validateWorkItemNavigation(ctx context.Context, item d.WorkItem) error {
	if item.LatestDeliveryID == nil {
		if item.PendingDeliveryID != nil || item.LatestDeliveryDecisionID != nil || item.LatestDeliveryWithdrawalID != nil || item.TaskDeliveryReplanAuthorized || item.DeliveryRound != 0 {
			return d.NewError(d.ErrorCodeInvalidScope, "WorkItem TaskDelivery navigation has no latest delivery")
		}
		return nil
	}
	delivery, err := r.Get(ctx, item.Scope, *item.LatestDeliveryID)
	if err != nil {
		return err
	}
	if delivery.WorkItemID != item.ID || delivery.Round != item.DeliveryRound {
		return d.NewError(d.ErrorCodeInvalidScope, "WorkItem latest delivery reference is missing or mismatched")
	}
	decision, err := r.DecisionForDelivery(ctx, item.Scope, delivery.ID)
	if err != nil {
		return err
	}
	withdrawal, err := r.WithdrawalForDelivery(ctx, item.Scope, delivery.ID)
	if err != nil {
		return err
	}
	if item.PendingDeliveryID != nil {
		if *item.PendingDeliveryID != delivery.ID || item.LatestDeliveryDecisionID != nil || item.LatestDeliveryWithdrawalID != nil || decision != nil || withdrawal != nil {
			return d.NewError(d.ErrorCodeInvalidScope, "WorkItem pending delivery is not its unresolved latest delivery")
		}
		return nil
	}
	if item.LatestDeliveryDecisionID != nil && item.LatestDeliveryWithdrawalID == nil {
		if decision == nil || decision.ID != *item.LatestDeliveryDecisionID || withdrawal != nil {
			return d.NewError(d.ErrorCodeInvalidScope, "WorkItem resolved delivery is missing its exact decision reference")
		}
		return nil
	}
	if item.LatestDeliveryWithdrawalID != nil && item.LatestDeliveryDecisionID == nil {
		stored, err := r.GetWithdrawal(ctx, item.Scope, *item.LatestDeliveryWithdrawalID)
		if err != nil {
			return err
		}
		if withdrawal == nil || withdrawal.ID != stored.ID || stored.WorkItemID != item.ID || stored.DeliveryID != delivery.ID || stored.Round != delivery.Round || stored.DeliveryDigest != delivery.Digest || decision != nil {
			return d.NewError(d.ErrorCodeInvalidScope, "WorkItem resolved delivery is missing its exact withdrawal reference")
		}
		return nil
	}
	return d.NewError(d.ErrorCodeInvalidScope, "WorkItem resolved delivery is missing its exact decision or withdrawal reference")
}

func saveTaskDeliveryWorkState(ctx context.Context, tx *sql.Tx, item d.WorkItem) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO task_delivery_work_state(namespace_id,outcome_id,work_item_id,pending_delivery_id,latest_delivery_id,latest_decision_id,latest_withdrawal_id,delivery_round,replan_authorized)
VALUES(?,?,?,?,?,?,?,?,?)
ON CONFLICT(namespace_id,outcome_id,work_item_id) DO UPDATE SET
 pending_delivery_id=excluded.pending_delivery_id,
 latest_delivery_id=excluded.latest_delivery_id,
 latest_decision_id=excluded.latest_decision_id,
 latest_withdrawal_id=excluded.latest_withdrawal_id,
	delivery_round=excluded.delivery_round,
	replan_authorized=excluded.replan_authorized`,
		item.Scope.NamespaceID.String(), item.Scope.OutcomeID.String(), item.ID.String(),
		nullableID(item.PendingDeliveryID), nullableID(item.LatestDeliveryID), nullableID(item.LatestDeliveryDecisionID), nullableID(item.LatestDeliveryWithdrawalID),
		strconv.FormatUint(item.DeliveryRound, 10), boolInt(item.TaskDeliveryReplanAuthorized))
	return mapSQLError("save TaskDelivery navigation", err)
}

func insertTaskDeliveryReferences(ctx context.Context, tx *sql.Tx, delivery d.TaskDelivery) error {
	for index, criterion := range delivery.CriterionSnapshot {
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_delivery_criteria(namespace_id,outcome_id,work_item_id,delivery_id,snapshot_index,criterion_id,criterion_revision,required,verification_mode)
VALUES(?,?,?,?,?,?,?,?,?)`, delivery.Scope.NamespaceID.String(), delivery.Scope.OutcomeID.String(), delivery.WorkItemID.String(), delivery.ID.String(), index, criterion.CriterionID.String(), int64(criterion.CriterionRevision), boolInt(criterion.Required), string(criterion.VerificationMode)); err != nil {
			return mapSQLError("insert TaskDelivery criterion snapshot", err)
		}
	}
	if delivery.Material == nil {
		return nil
	}
	for index, artifact := range delivery.Material.Artifacts {
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_delivery_artifacts(namespace_id,outcome_id,delivery_id,artifact_index,artifact_id,source_version,checksum)
VALUES(?,?,?,?,?,?,?)`, delivery.Scope.NamespaceID.String(), delivery.Scope.OutcomeID.String(), delivery.ID.String(), index, artifact.ArtifactID.String(), artifact.SourceVersion, artifact.Checksum); err != nil {
			return mapSQLError("insert TaskDelivery artifact reference", err)
		}
	}
	for index, evidenceID := range delivery.Material.EvidenceIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_delivery_evidence(namespace_id,outcome_id,delivery_id,evidence_index,evidence_id)
VALUES(?,?,?,?,?)`, delivery.Scope.NamespaceID.String(), delivery.Scope.OutcomeID.String(), delivery.ID.String(), index, evidenceID.String()); err != nil {
			return mapSQLError("insert TaskDelivery evidence reference", err)
		}
	}
	for _, link := range delivery.Material.CriterionEvidence {
		for index, evidenceID := range link.EvidenceIDs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO task_delivery_criterion_evidence(namespace_id,outcome_id,delivery_id,criterion_id,criterion_revision,evidence_index,evidence_id)
VALUES(?,?,?,?,?,?,?)`, delivery.Scope.NamespaceID.String(), delivery.Scope.OutcomeID.String(), delivery.ID.String(), link.CriterionID.String(), int64(link.CriterionRevision), index, evidenceID.String()); err != nil {
				return mapSQLError("insert TaskDelivery criterion evidence", err)
			}
		}
	}
	return nil
}

func validateTaskDeliveryReferences(ctx context.Context, tx *sql.Tx, delivery d.TaskDelivery) error {
	rows, err := tx.QueryContext(ctx, `SELECT criterion_id,criterion_revision,required,verification_mode FROM task_delivery_criteria
WHERE namespace_id=? AND outcome_id=? AND delivery_id=? ORDER BY snapshot_index`, delivery.Scope.NamespaceID.String(), delivery.Scope.OutcomeID.String(), delivery.ID.String())
	if err != nil {
		return mapSQLError("read TaskDelivery criterion snapshot", err)
	}
	criteria := make([]d.TaskDeliveryCriterionSnapshot, 0, len(delivery.CriterionSnapshot))
	for rows.Next() {
		var id, mode string
		var revision, required int64
		if err := rows.Scan(&id, &revision, &required, &mode); err != nil {
			rows.Close()
			return err
		}
		parsed, err := d.ParseID(id)
		if err != nil {
			rows.Close()
			return err
		}
		criteria = append(criteria, d.TaskDeliveryCriterionSnapshot{CriterionID: parsed, CriterionRevision: d.CriterionRevision(revision), Required: required == 1, VerificationMode: d.VerificationMode(mode)})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !equalTaskDeliveryCriteria(criteria, delivery.CriterionSnapshot) {
		return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery criterion rows differ from immutable payload")
	}

	artifacts := []d.SubmissionArtifact{}
	rows, err = tx.QueryContext(ctx, `SELECT artifact_id,source_version,checksum FROM task_delivery_artifacts
WHERE namespace_id=? AND outcome_id=? AND delivery_id=? ORDER BY artifact_index`, delivery.Scope.NamespaceID.String(), delivery.Scope.OutcomeID.String(), delivery.ID.String())
	if err != nil {
		return mapSQLError("read TaskDelivery artifact references", err)
	}
	for rows.Next() {
		var rawID, sourceVersion, checksum string
		if err := rows.Scan(&rawID, &sourceVersion, &checksum); err != nil {
			rows.Close()
			return err
		}
		id, err := d.ParseID(rawID)
		if err != nil {
			rows.Close()
			return err
		}
		artifacts = append(artifacts, d.SubmissionArtifact{ArtifactID: id, SourceVersion: sourceVersion, Checksum: checksum})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	var expectedArtifacts []d.SubmissionArtifact
	var expectedEvidence []d.ID
	var expectedCriterionEvidence []d.SubmissionCriterionEvidence
	if delivery.Material != nil {
		expectedArtifacts = delivery.Material.Artifacts
		expectedEvidence = delivery.Material.EvidenceIDs
		expectedCriterionEvidence = delivery.Material.CriterionEvidence
	}
	if !equalTaskDeliveryArtifacts(artifacts, expectedArtifacts) {
		return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery artifact rows differ from immutable payload")
	}

	evidence := []d.ID{}
	rows, err = tx.QueryContext(ctx, `SELECT evidence_id FROM task_delivery_evidence
WHERE namespace_id=? AND outcome_id=? AND delivery_id=? ORDER BY evidence_index`, delivery.Scope.NamespaceID.String(), delivery.Scope.OutcomeID.String(), delivery.ID.String())
	if err != nil {
		return mapSQLError("read TaskDelivery evidence references", err)
	}
	for rows.Next() {
		var rawID string
		if err := rows.Scan(&rawID); err != nil {
			rows.Close()
			return err
		}
		id, err := d.ParseID(rawID)
		if err != nil {
			rows.Close()
			return err
		}
		evidence = append(evidence, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !equalTaskDeliveryIDs(evidence, expectedEvidence) {
		return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery evidence rows differ from immutable payload")
	}

	criterionEvidence := make(map[d.ID]d.SubmissionCriterionEvidence)
	rows, err = tx.QueryContext(ctx, `SELECT criterion_id,criterion_revision,evidence_id FROM task_delivery_criterion_evidence
WHERE namespace_id=? AND outcome_id=? AND delivery_id=? ORDER BY criterion_id,evidence_index`, delivery.Scope.NamespaceID.String(), delivery.Scope.OutcomeID.String(), delivery.ID.String())
	if err != nil {
		return mapSQLError("read TaskDelivery criterion evidence", err)
	}
	for rows.Next() {
		var criterionID, evidenceID string
		var revision int64
		if err := rows.Scan(&criterionID, &revision, &evidenceID); err != nil {
			rows.Close()
			return err
		}
		criterion, err := d.ParseID(criterionID)
		if err != nil {
			rows.Close()
			return err
		}
		evidence, err := d.ParseID(evidenceID)
		if err != nil {
			rows.Close()
			return err
		}
		link, ok := criterionEvidence[criterion]
		if !ok {
			link = d.SubmissionCriterionEvidence{CriterionID: criterion, CriterionRevision: d.CriterionRevision(revision), EvidenceIDs: []d.ID{}}
		} else if link.CriterionRevision != d.CriterionRevision(revision) {
			rows.Close()
			return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery criterion evidence revisions differ")
		}
		link.EvidenceIDs = append(link.EvidenceIDs, evidence)
		criterionEvidence[criterion] = link
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	actualCriterionEvidence := make([]d.SubmissionCriterionEvidence, 0, len(criterionEvidence))
	for _, link := range expectedCriterionEvidence {
		actual, ok := criterionEvidence[link.CriterionID]
		if !ok {
			return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery criterion evidence rows differ from immutable payload")
		}
		actualCriterionEvidence = append(actualCriterionEvidence, actual)
		delete(criterionEvidence, link.CriterionID)
	}
	if len(criterionEvidence) != 0 || !equalTaskDeliveryCriterionEvidence(actualCriterionEvidence, expectedCriterionEvidence) {
		return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery criterion evidence rows differ from immutable payload")
	}
	var predecessorID, predecessorRound, predecessorDigest sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT predecessor_delivery_id,predecessor_round,predecessor_digest
FROM task_delivery_predecessors WHERE namespace_id=? AND outcome_id=? AND work_item_id=? AND delivery_id=?`, delivery.Scope.NamespaceID.String(), delivery.Scope.OutcomeID.String(), delivery.WorkItemID.String(), delivery.ID.String()).Scan(&predecessorID, &predecessorRound, &predecessorDigest)
	if delivery.SupersedesDeliveryID == nil {
		if err == nil {
			return d.NewError(d.ErrorCodeInvalidScope, "first TaskDelivery unexpectedly has a predecessor row")
		}
		if err != sql.ErrNoRows {
			return mapSQLError("read TaskDelivery predecessor", err)
		}
		if delivery.Round != 1 {
			return d.NewError(d.ErrorCodeInvalidScope, "noninitial TaskDelivery is missing its predecessor")
		}
	} else {
		if err != nil {
			return mapSQLError("read TaskDelivery predecessor", err)
		}
		parsedID, parseErr := d.ParseID(predecessorID.String)
		parsedRound, roundErr := strconv.ParseUint(predecessorRound.String, 10, 64)
		if parseErr != nil || roundErr != nil || predecessorRound.String != strconv.FormatUint(parsedRound, 10) || parsedID != *delivery.SupersedesDeliveryID || delivery.Round <= 1 || parsedRound != delivery.Round-1 || len(predecessorDigest.String) != 71 || !strings.HasPrefix(predecessorDigest.String, "sha256:") {
			return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery predecessor row differs from its immutable correction chain")
		}
	}
	return nil
}

func equalTaskDeliveryCriteria(left, right []d.TaskDeliveryCriterionSnapshot) bool {
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

func equalTaskDeliveryArtifacts(left, right []d.SubmissionArtifact) bool {
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

func equalTaskDeliveryIDs(left, right []d.ID) bool {
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

func equalTaskDeliveryCriterionEvidence(left, right []d.SubmissionCriterionEvidence) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].CriterionID != right[i].CriterionID || left[i].CriterionRevision != right[i].CriterionRevision || !equalTaskDeliveryIDs(left[i].EvidenceIDs, right[i].EvidenceIDs) {
			return false
		}
	}
	return true
}

func sameOptionalTaskDeliveryID(left, right *d.ID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func (r taskDeliveryRepository) Insert(ctx context.Context, delivery d.TaskDelivery) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := delivery.Validate(); err != nil {
		return err
	}
	work, err := r.uow.workItems.Get(ctx, delivery.Scope, delivery.WorkItemID)
	if err != nil {
		return err
	}
	if err := delivery.ValidateForWork(work); err != nil {
		return err
	}
	if !sameOptionalTaskDeliveryID(delivery.SupersedesDeliveryID, work.LatestDeliveryID) {
		return d.NewError(d.ErrorCodeInvalidTransition, "TaskDelivery must supersede the exact latest delivery")
	}
	if work.PendingDeliveryID != nil || delivery.Round != work.DeliveryRound+1 {
		return d.NewError(d.ErrorCodeInvalidTransition, "Task already has a pending delivery or delivery round is stale")
	}
	var submissionID, submissionContractID, submissionDigest any
	var contractID any
	if delivery.Authority.ContractID != nil {
		contractID = delivery.Authority.ContractID.String()
	}
	if delivery.SubmissionRef != nil {
		if delivery.Authority.ContractID == nil || delivery.SubmissionRef.ContractID != *delivery.Authority.ContractID {
			return d.NewError(d.ErrorCodeContractSpecMismatch, "delivery submission does not reference its authorizing contract")
		}
		submission, getErr := r.uow.WorkContracts().GetSubmission(ctx, delivery.Scope, delivery.SubmissionRef.ID)
		if getErr != nil {
			return getErr
		}
		if submission.Scope != delivery.Scope || submission.Material.WorkItemID != delivery.WorkItemID || submission.Material.ContractID != delivery.SubmissionRef.ContractID || submission.Digest != delivery.SubmissionRef.Digest {
			return d.NewError(d.ErrorCodeContractSpecMismatch, "delivery does not reference the exact scoped WorkSubmission")
		}
		submissionID = delivery.SubmissionRef.ID.String()
		submissionContractID = delivery.SubmissionRef.ContractID.String()
		submissionDigest = delivery.SubmissionRef.Digest
	} else if delivery.Authority.Kind != d.TaskDeliveryAuthorityLegacyLease || delivery.Material == nil {
		return d.NewError(d.ErrorCodeInvalidArgument, "inline delivery material is only valid for a legacy lease")
	}
	if submissionContractID != nil && contractID != submissionContractID {
		return d.NewError(d.ErrorCodeContractSpecMismatch, "delivery submission contract differs from its authority")
	}
	if _, err := r.uow.tx.ExecContext(ctx, `INSERT INTO task_delivery_work_state(namespace_id,outcome_id,work_item_id,pending_delivery_id,latest_delivery_id,latest_decision_id,latest_withdrawal_id,delivery_round)
	VALUES(?,?,?,NULL,NULL,NULL,NULL,'0') ON CONFLICT(namespace_id,outcome_id,work_item_id) DO NOTHING`, delivery.Scope.NamespaceID.String(), delivery.Scope.OutcomeID.String(), delivery.WorkItemID.String()); err != nil {
		return mapSQLError("initialize TaskDelivery navigation", err)
	}
	payload, err := marshalJSON(delivery)
	if err != nil {
		return err
	}
	_, err = r.uow.tx.ExecContext(ctx, `INSERT INTO task_deliveries(id,namespace_id,outcome_id,work_item_id,delivery_round,authority_kind,contract_id,submission_id,submission_digest,delivery_digest,definition_digest,payload_json)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, delivery.ID.String(), delivery.Scope.NamespaceID.String(), delivery.Scope.OutcomeID.String(), delivery.WorkItemID.String(), strconv.FormatUint(delivery.Round, 10), string(delivery.Authority.Kind), contractID, submissionID, submissionDigest, delivery.Digest, delivery.DefinitionDigest, payload)
	if err != nil {
		return mapSQLError("insert TaskDelivery", err)
	}
	return insertTaskDeliveryReferences(ctx, r.uow.tx, delivery)
}

func (r taskDeliveryRepository) Get(ctx context.Context, scope d.Scope, id d.ID) (d.TaskDelivery, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return d.TaskDelivery{}, err
	}
	var raw, workItemID, round, authority, deliveryDigest, definitionDigest string
	var contractID, submissionID, submissionDigest sql.NullString
	err := r.uow.tx.QueryRowContext(ctx, `SELECT payload_json,work_item_id,delivery_round,authority_kind,contract_id,submission_id,submission_digest,delivery_digest,definition_digest
FROM task_deliveries WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), id.String()).Scan(&raw, &workItemID, &round, &authority, &contractID, &submissionID, &submissionDigest, &deliveryDigest, &definitionDigest)
	if err != nil {
		return d.TaskDelivery{}, mapSQLError("get TaskDelivery", err)
	}
	var value d.TaskDelivery
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return value, err
	}
	if err := validateTaskDeliveryColumns(value, scope, id, workItemID, round, authority, contractID, submissionID, submissionDigest, deliveryDigest, definitionDigest); err != nil {
		return value, err
	}
	if err := value.Validate(); err != nil {
		return value, err
	}
	if err := validateTaskDeliveryReferences(ctx, r.uow.tx, value); err != nil {
		return value, err
	}
	return value, nil
}

func validateTaskDeliveryColumns(value d.TaskDelivery, scope d.Scope, id d.ID, workItemID, round, authority string, contractID, submissionID, submissionDigest sql.NullString, deliveryDigest, definitionDigest string) error {
	parsedWorkID, err := d.ParseID(workItemID)
	if err != nil {
		return d.WrapError(d.ErrorCodeInvalidScope, "TaskDelivery work item binding is invalid", err)
	}
	parsedRound, err := strconv.ParseUint(round, 10, 64)
	if err != nil || round != strconv.FormatUint(parsedRound, 10) {
		return d.NewError(d.ErrorCodeInvalidArgument, "persisted TaskDelivery round is invalid")
	}
	if value.Scope != scope || value.ID != id || value.WorkItemID != parsedWorkID || value.Round != parsedRound || string(value.Authority.Kind) != authority || value.Digest != deliveryDigest || value.DefinitionDigest != definitionDigest {
		return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery columns differ from immutable payload")
	}
	if contractID.Valid != (value.Authority.ContractID != nil) || submissionID.Valid != (value.SubmissionRef != nil) || submissionDigest.Valid != (value.SubmissionRef != nil) {
		return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery submission binding columns differ from payload")
	}
	if contractID.Valid && (value.Authority.ContractID.String() != contractID.String || value.SubmissionRef == nil || value.SubmissionRef.ContractID.String() != contractID.String) {
		return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery contract binding differs from payload")
	}
	if submissionID.Valid && (value.SubmissionRef.ID.String() != submissionID.String || value.SubmissionRef.Digest != submissionDigest.String) {
		return d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery WorkSubmission binding differs from payload")
	}
	return nil
}

func (r taskDeliveryRepository) Pending(ctx context.Context, scope d.Scope, workItemID d.ID) (*d.TaskDelivery, error) {
	work, err := r.uow.workItems.Get(ctx, scope, workItemID)
	if err != nil {
		return nil, err
	}
	if work.PendingDeliveryID == nil {
		return nil, nil
	}
	delivery, err := r.Get(ctx, scope, *work.PendingDeliveryID)
	if err != nil {
		return nil, err
	}
	if delivery.WorkItemID != workItemID || work.LatestDeliveryID == nil || *work.LatestDeliveryID != delivery.ID || work.DeliveryRound != delivery.Round {
		return nil, d.NewError(d.ErrorCodeInvalidScope, "pending TaskDelivery navigation differs from its WorkItem")
	}
	return &delivery, nil
}

func (r taskDeliveryRepository) List(ctx context.Context, scope d.Scope, filter ports.TaskDeliveryFilter) ([]d.TaskDelivery, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return nil, err
	}
	if filter.Limit < 1 || filter.Limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "TaskDelivery limit outside 1..101")
	}
	query := `SELECT payload_json FROM task_deliveries WHERE namespace_id=? AND outcome_id=? AND (?='' OR work_item_id=?) AND id>?`
	args := []any{scope.NamespaceID.String(), scope.OutcomeID.String(), filter.WorkItemID.String(), filter.WorkItemID.String(), filter.After.String()}
	if filter.Pending {
		query += ` AND EXISTS (SELECT 1 FROM task_delivery_work_state s WHERE s.namespace_id=task_deliveries.namespace_id AND s.outcome_id=task_deliveries.outcome_id AND s.work_item_id=task_deliveries.work_item_id AND s.pending_delivery_id=task_deliveries.id)`
	}
	query += ` ORDER BY id LIMIT ?`
	args = append(args, filter.Limit)
	rows, err := r.uow.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mapSQLError("list TaskDeliveries", err)
	}
	defer rows.Close()
	values := []d.TaskDelivery{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var value d.TaskDelivery
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			return nil, err
		}
		if err := value.Validate(); err != nil {
			return nil, err
		}
		if value.Scope != scope || (filter.WorkItemID != "" && value.WorkItemID != filter.WorkItemID) {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "persisted TaskDelivery page contains a mismatched binding")
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for _, value := range values {
		if err := validateTaskDeliveryReferences(ctx, r.uow.tx, value); err != nil {
			return nil, err
		}
	}
	return values, nil
}

func (r taskDeliveryRepository) InsertDecision(ctx context.Context, decision d.TaskDeliveryDecision) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := decision.Validate(); err != nil {
		return err
	}
	delivery, err := r.Get(ctx, decision.Scope, decision.DeliveryID)
	if err != nil {
		return err
	}
	if err := decision.ValidateForDelivery(delivery); err != nil {
		return err
	}
	work, err := r.uow.workItems.Get(ctx, decision.Scope, decision.WorkItemID)
	if err != nil {
		return err
	}
	if work.PendingDeliveryID == nil || *work.PendingDeliveryID != delivery.ID || work.LatestDeliveryID == nil || *work.LatestDeliveryID != delivery.ID {
		return d.NewError(d.ErrorCodeSubmissionNotAccepted, "decision target is not the pending TaskDelivery")
	}
	withdrawal, err := r.WithdrawalForDelivery(ctx, decision.Scope, delivery.ID)
	if err != nil {
		return err
	}
	if withdrawal != nil {
		return d.NewError(d.ErrorCodeSubmissionNotAccepted, "TaskDelivery has already been withdrawn")
	}
	payload, err := marshalJSON(decision)
	if err != nil {
		return err
	}
	_, err = r.uow.tx.ExecContext(ctx, `INSERT INTO task_delivery_decisions(id,namespace_id,outcome_id,work_item_id,delivery_id,delivery_round,delivery_digest,result,payload_json)
VALUES(?,?,?,?,?,?,?,?,?)`, decision.ID.String(), decision.Scope.NamespaceID.String(), decision.Scope.OutcomeID.String(), decision.WorkItemID.String(), decision.DeliveryID.String(), strconv.FormatUint(decision.Round, 10), decision.DeliveryDigest, string(decision.Result), payload)
	return mapSQLError("insert TaskDeliveryDecision", err)
}

func (r taskDeliveryRepository) GetDecision(ctx context.Context, scope d.Scope, id d.ID) (d.TaskDeliveryDecision, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return d.TaskDeliveryDecision{}, err
	}
	var raw, workID, deliveryID, round, digest, result string
	err := r.uow.tx.QueryRowContext(ctx, `SELECT payload_json,work_item_id,delivery_id,delivery_round,delivery_digest,result FROM task_delivery_decisions WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), id.String()).Scan(&raw, &workID, &deliveryID, &round, &digest, &result)
	if err != nil {
		return d.TaskDeliveryDecision{}, mapSQLError("get TaskDeliveryDecision", err)
	}
	var value d.TaskDeliveryDecision
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return value, err
	}
	parsedWork, workErr := d.ParseID(workID)
	parsedDelivery, deliveryErr := d.ParseID(deliveryID)
	parsedRound, roundErr := strconv.ParseUint(round, 10, 64)
	if workErr != nil || deliveryErr != nil || roundErr != nil || round != strconv.FormatUint(parsedRound, 10) || value.Scope != scope || value.ID != id || value.WorkItemID != parsedWork || value.DeliveryID != parsedDelivery || value.Round != parsedRound || value.DeliveryDigest != digest || string(value.Result) != result {
		return value, d.NewError(d.ErrorCodeInvalidScope, "TaskDeliveryDecision columns differ from immutable payload")
	}
	if err := value.Validate(); err != nil {
		return value, err
	}
	delivery, err := r.Get(ctx, scope, value.DeliveryID)
	if err != nil {
		return value, err
	}
	return value, value.ValidateForDelivery(delivery)
}

func (r taskDeliveryRepository) DecisionForDelivery(ctx context.Context, scope d.Scope, deliveryID d.ID) (*d.TaskDeliveryDecision, error) {
	var id string
	err := r.uow.tx.QueryRowContext(ctx, `SELECT id FROM task_delivery_decisions WHERE namespace_id=? AND outcome_id=? AND delivery_id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), deliveryID.String()).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, mapSQLError("get TaskDeliveryDecision for delivery", err)
	}
	parsed, err := d.ParseID(id)
	if err != nil {
		return nil, err
	}
	value, err := r.GetDecision(ctx, scope, parsed)
	return &value, err
}

func (r taskDeliveryRepository) InsertWithdrawal(ctx context.Context, withdrawal d.TaskDeliveryWithdrawal) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := withdrawal.Validate(); err != nil {
		return err
	}
	delivery, err := r.Get(ctx, withdrawal.Scope, withdrawal.DeliveryID)
	if err != nil {
		return err
	}
	if err := withdrawal.ValidateForDelivery(delivery); err != nil {
		return err
	}
	work, err := r.uow.workItems.Get(ctx, withdrawal.Scope, withdrawal.WorkItemID)
	if err != nil {
		return err
	}
	if work.PendingDeliveryID == nil || *work.PendingDeliveryID != delivery.ID || work.LatestDeliveryID == nil || *work.LatestDeliveryID != delivery.ID {
		return d.NewError(d.ErrorCodeSubmissionNotAccepted, "withdrawal target is not the pending TaskDelivery")
	}
	decision, err := r.DecisionForDelivery(ctx, withdrawal.Scope, delivery.ID)
	if err != nil {
		return err
	}
	if decision != nil {
		return d.NewError(d.ErrorCodeSubmissionNotAccepted, "TaskDelivery already has a decision")
	}
	prior, err := r.WithdrawalForDelivery(ctx, withdrawal.Scope, delivery.ID)
	if err != nil {
		return err
	}
	if prior != nil {
		return d.NewError(d.ErrorCodeAlreadyExists, "TaskDelivery already has a withdrawal")
	}
	payload, err := marshalJSON(withdrawal)
	if err != nil {
		return err
	}
	_, err = r.uow.tx.ExecContext(ctx, `INSERT INTO task_delivery_withdrawals(id,namespace_id,outcome_id,work_item_id,delivery_id,delivery_round,delivery_digest,payload_json)
VALUES(?,?,?,?,?,?,?,?)`, withdrawal.ID.String(), withdrawal.Scope.NamespaceID.String(), withdrawal.Scope.OutcomeID.String(), withdrawal.WorkItemID.String(), withdrawal.DeliveryID.String(), strconv.FormatUint(withdrawal.Round, 10), withdrawal.DeliveryDigest, payload)
	return mapSQLError("insert TaskDeliveryWithdrawal", err)
}

func (r taskDeliveryRepository) GetWithdrawal(ctx context.Context, scope d.Scope, id d.ID) (d.TaskDeliveryWithdrawal, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return d.TaskDeliveryWithdrawal{}, err
	}
	var raw, workID, deliveryID, round, digest string
	err := r.uow.tx.QueryRowContext(ctx, `SELECT payload_json,work_item_id,delivery_id,delivery_round,delivery_digest
FROM task_delivery_withdrawals WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), id.String()).Scan(&raw, &workID, &deliveryID, &round, &digest)
	if err != nil {
		return d.TaskDeliveryWithdrawal{}, mapSQLError("get TaskDeliveryWithdrawal", err)
	}
	var value d.TaskDeliveryWithdrawal
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return value, err
	}
	parsedWork, workErr := d.ParseID(workID)
	parsedDelivery, deliveryErr := d.ParseID(deliveryID)
	parsedRound, roundErr := strconv.ParseUint(round, 10, 64)
	if workErr != nil || deliveryErr != nil || roundErr != nil || round != strconv.FormatUint(parsedRound, 10) || value.Scope != scope || value.ID != id || value.WorkItemID != parsedWork || value.DeliveryID != parsedDelivery || value.Round != parsedRound || value.DeliveryDigest != digest {
		return value, d.NewError(d.ErrorCodeInvalidScope, "TaskDeliveryWithdrawal columns differ from immutable payload")
	}
	if err := value.Validate(); err != nil {
		return value, err
	}
	delivery, err := r.Get(ctx, scope, value.DeliveryID)
	if err != nil {
		return value, err
	}
	return value, value.ValidateForDelivery(delivery)
}

func (r taskDeliveryRepository) WithdrawalForDelivery(ctx context.Context, scope d.Scope, deliveryID d.ID) (*d.TaskDeliveryWithdrawal, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return nil, err
	}
	var id string
	err := r.uow.tx.QueryRowContext(ctx, `SELECT id FROM task_delivery_withdrawals WHERE namespace_id=? AND outcome_id=? AND delivery_id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), deliveryID.String()).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, mapSQLError("get TaskDeliveryWithdrawal for delivery", err)
	}
	parsed, err := d.ParseID(id)
	if err != nil {
		return nil, err
	}
	value, err := r.GetWithdrawal(ctx, scope, parsed)
	return &value, err
}

func (r taskDeliveryRepository) ListDecisions(ctx context.Context, scope d.Scope, deliveryID, after d.ID, limit int) ([]d.TaskDeliveryDecision, error) {
	if limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "TaskDeliveryDecision limit outside 1..101")
	}
	rows, err := r.uow.tx.QueryContext(ctx, `SELECT id FROM task_delivery_decisions WHERE namespace_id=? AND outcome_id=? AND delivery_id=? AND id>? ORDER BY id LIMIT ?`, scope.NamespaceID.String(), scope.OutcomeID.String(), deliveryID.String(), after.String(), limit)
	if err != nil {
		return nil, mapSQLError("list TaskDeliveryDecisions", err)
	}
	ids := make([]d.ID, 0)
	for rows.Next() {
		var rawID string
		if err := rows.Scan(&rawID); err != nil {
			rows.Close()
			return nil, err
		}
		id, err := d.ParseID(rawID)
		if err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	values := make([]d.TaskDeliveryDecision, 0, len(ids))
	for _, id := range ids {
		value, err := r.GetDecision(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
