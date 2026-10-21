package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type taskVerificationReadRepository struct{ u *unitOfWork }

func (u *unitOfWork) TaskVerificationReads() ports.TaskVerificationReadRepository {
	return taskVerificationReadRepository{u: u}
}

func (r taskVerificationReadRepository) rows(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return r.u.tx.QueryContext(ctx, query, args...)
}

func verificationIN(ids []d.ID) (string, []any) {
	marks := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		marks[i], args[i] = "?", id.String()
	}
	return "(" + strings.Join(marks, ",") + ")", args
}

func (r taskVerificationReadRepository) TaskVerificationFacts(ctx context.Context, scope d.Scope, ids []d.ID) ([]ports.TaskVerificationFacts, error) {
	if err := r.u.ensureOpen(); err != nil {
		return nil, err
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if len(ids) > ports.MaxTaskVerificationBatch {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "Task verification batch exceeds 100 Tasks")
	}
	if len(ids) == 0 {
		return []ports.TaskVerificationFacts{}, nil
	}
	seen := make(map[d.ID]bool, len(ids))
	for _, id := range ids {
		if id.Validate() != nil || seen[id] {
			return nil, d.NewError(d.ErrorCodeInvalidArgument, "Task verification requires distinct valid Task IDs")
		}
		seen[id] = true
	}
	in, idArgs := verificationIN(ids)
	args := append([]any{scope.NamespaceID.String(), scope.OutcomeID.String()}, idArgs...)
	rows, err := r.rows(ctx, `SELECT work_items.id,work_items.version,work_items.title,work_items.description,work_items.objective_id,work_items.lifecycle,work_items.priority,
work_items.not_before,work_items.result_summary,COALESCE(work_items.fencing_token_decimal,CAST(work_items.last_fencing_token AS TEXT)),work_items.lease_claim_id,work_items.lease_principal_id,
work_items.lease_actor_json,work_items.lease_acquired_at,work_items.lease_expires_at,work_items.created_at,work_items.updated_at,work_items.contracts_enabled,work_items.current_contract_id,
work_items.execution_spec_json,work_items.pending_review_case_id,work_items.latest_review_case_id,work_items.correction_review_case_id,
work_items.lease_credential_binding_version,work_items.lease_holder_credential_id,work_items.lease_session_credential_id,
delivery_state.pending_delivery_id,delivery_state.latest_delivery_id,delivery_state.latest_decision_id,delivery_state.latest_withdrawal_id,delivery_state.delivery_round,delivery_state.replan_authorized
FROM work_items LEFT JOIN task_delivery_work_state AS delivery_state
ON delivery_state.namespace_id=work_items.namespace_id AND delivery_state.outcome_id=work_items.outcome_id AND delivery_state.work_item_id=work_items.id
WHERE work_items.namespace_id=? AND work_items.outcome_id=? AND work_items.id IN `+in, args...)
	if err != nil {
		return nil, mapSQLError("read Task verification WorkItems", err)
	}
	items := map[d.ID]d.WorkItem{}
	for rows.Next() {
		item, scanErr := scanWorkItem(rows, scope)
		if scanErr != nil {
			rows.Close()
			return nil, scanErr
		}
		if _, duplicate := items[item.ID]; duplicate {
			rows.Close()
			return nil, d.NewError(d.ErrorCodeInvalidScope, "duplicate Task verification WorkItem")
		}
		items[item.ID] = item
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, mapSQLError("iterate Task verification WorkItems", err)
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	if len(items) != len(ids) {
		return nil, d.NewError(d.ErrorCodeNotFound, "one or more Task verification WorkItems were not found")
	}
	pendingTaskIDs := make([]d.ID, 0, len(items))
	for _, id := range ids {
		if items[id].PendingDeliveryID != nil {
			pendingTaskIDs = append(pendingTaskIDs, id)
		}
	}
	if err = r.loadPendingTaskVerificationCriteria(ctx, scope, pendingTaskIDs, items); err != nil {
		return nil, err
	}

	latestIDs := make([]d.ID, 0, len(ids))
	contractIDs := make([]d.ID, 0, len(ids)*2)
	caseIDs := make([]d.ID, 0, len(ids)*3)
	for _, id := range ids {
		item := items[id]
		if item.ID != id || item.Scope != scope {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "Task verification WorkItem scope differs")
		}
		if item.CurrentContractID != nil {
			contractIDs = append(contractIDs, *item.CurrentContractID)
		}
		if item.LatestDeliveryID != nil {
			latestIDs = append(latestIDs, *item.LatestDeliveryID)
		}
		for _, cid := range []*d.ID{item.PendingReviewCaseID, item.LatestReviewCaseID, item.CorrectionReviewCaseID} {
			if cid != nil {
				caseIDs = append(caseIDs, *cid)
			}
		}
	}
	latestIDs = uniqueIDs(latestIDs)
	caseIDs = uniqueIDs(caseIDs)
	deliveries := map[d.ID]d.TaskDelivery{}
	if len(latestIDs) > 0 {
		deliveryIn, deliveryArgs := verificationIN(latestIDs)
		queryArgs := append([]any{scope.NamespaceID.String(), scope.OutcomeID.String()}, deliveryArgs...)
		rows, err = r.rows(ctx, `SELECT id,work_item_id,delivery_round,authority_kind,contract_id,submission_id,submission_digest,delivery_digest,definition_digest,payload_json
FROM task_deliveries WHERE namespace_id=? AND outcome_id=? AND id IN `+deliveryIn, queryArgs...)
		if err != nil {
			return nil, mapSQLError("read Task verification deliveries", err)
		}
		for rows.Next() {
			var rawID, workID, roundRaw, authority, deliveryDigest, definitionDigest, payload string
			var contractID, submissionID, submissionDigest sql.NullString
			if err = rows.Scan(&rawID, &workID, &roundRaw, &authority, &contractID, &submissionID, &submissionDigest, &deliveryDigest, &definitionDigest, &payload); err != nil {
				rows.Close()
				return nil, err
			}
			id, parseErr := d.ParseID(rawID)
			if parseErr != nil {
				rows.Close()
				return nil, parseErr
			}
			var value d.TaskDelivery
			if err = json.Unmarshal([]byte(payload), &value); err != nil {
				rows.Close()
				return nil, err
			}
			if err = value.Validate(); err != nil {
				rows.Close()
				return nil, err
			}
			if err = validateTaskDeliveryColumns(value, scope, id, workID, roundRaw, authority, contractID, submissionID, submissionDigest, deliveryDigest, definitionDigest); err != nil {
				rows.Close()
				return nil, err
			}
			if _, duplicate := deliveries[id]; duplicate {
				rows.Close()
				return nil, d.NewError(d.ErrorCodeInvalidScope, "duplicate TaskDelivery source")
			}
			deliveries[id] = value
			if value.Authority.ContractID != nil {
				contractIDs = append(contractIDs, *value.Authority.ContractID)
			}
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return nil, mapSQLError("iterate Task verification deliveries", err)
		}
		if err = rows.Close(); err != nil {
			return nil, err
		}
		if len(deliveries) != len(latestIDs) {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "WorkItem latest delivery is missing")
		}
		if err = r.validateDeliveryMaterialBatch(ctx, scope, latestIDs, deliveries); err != nil {
			return nil, err
		}
		// The normalized predecessor relation is the durable correction chain.
		predRows, e := r.rows(ctx, `SELECT delivery_id,delivery_round,predecessor_delivery_id,predecessor_round,predecessor_digest FROM task_delivery_predecessors WHERE namespace_id=? AND outcome_id=? AND delivery_id IN `+deliveryIn, queryArgs...)
		if e != nil {
			return nil, mapSQLError("read TaskDelivery predecessor facts", e)
		}
		predecessors := map[d.ID]struct {
			id     d.ID
			round  uint64
			digest string
		}{}
		for predRows.Next() {
			var deliveryID, roundRaw, predecessorID, predecessorRound, digest string
			if e = predRows.Scan(&deliveryID, &roundRaw, &predecessorID, &predecessorRound, &digest); e != nil {
				predRows.Close()
				return nil, e
			}
			id, e1 := d.ParseID(deliveryID)
			predecessor, e2 := d.ParseID(predecessorID)
			round, e3 := strconv.ParseUint(roundRaw, 10, 64)
			priorRound, e4 := strconv.ParseUint(predecessorRound, 10, 64)
			if e1 != nil || e2 != nil || e3 != nil || e4 != nil || roundRaw != strconv.FormatUint(round, 10) || predecessorRound != strconv.FormatUint(priorRound, 10) {
				predRows.Close()
				return nil, d.NewError(d.ErrorCodeInvalidScope, "malformed TaskDelivery predecessor facts")
			}
			predecessors[id] = struct {
				id     d.ID
				round  uint64
				digest string
			}{predecessor, priorRound, digest}
		}
		if e = predRows.Err(); e != nil {
			predRows.Close()
			return nil, e
		}
		predRows.Close()
		for _, value := range deliveries {
			p, ok := predecessors[value.ID]
			if value.SupersedesDeliveryID == nil {
				if ok || value.Round != 1 {
					return nil, d.NewError(d.ErrorCodeInvalidScope, "first delivery has unexpected predecessor")
				}
				continue
			}
			if !ok || p.id != *value.SupersedesDeliveryID || value.Round < 2 || p.round != value.Round-1 || !d.ValidSignedDigest(p.digest) {
				return nil, d.NewError(d.ErrorCodeInvalidScope, "delivery predecessor relation differs from payload")
			}
		}
	}

	caseMap := map[d.ID]d.ReviewCase{}
	if len(caseIDs) > 0 {
		caseIn, caseArgs := verificationIN(caseIDs)
		queryArgs := append([]any{scope.NamespaceID.String(), scope.OutcomeID.String()}, caseArgs...)
		rows, err = r.rows(ctx, `SELECT id,work_item_id,work_contract_id,submission_id,status,version,state_json FROM work_review_cases WHERE namespace_id=? AND outcome_id=? AND id IN `+caseIn, queryArgs...)
		if err != nil {
			return nil, mapSQLError("read Task verification review cases", err)
		}
		for rows.Next() {
			var idRaw, workRaw, contractRaw, submissionRaw, statusRaw, versionRaw, state string
			if err = rows.Scan(&idRaw, &workRaw, &contractRaw, &submissionRaw, &statusRaw, &versionRaw, &state); err != nil {
				rows.Close()
				return nil, err
			}
			var value d.ReviewCase
			if err = json.Unmarshal([]byte(state), &value); err != nil {
				rows.Close()
				return nil, err
			}
			id, e1 := d.ParseID(idRaw)
			workID, e2 := d.ParseID(workRaw)
			contractID, e3 := d.ParseID(contractRaw)
			submissionID, e4 := d.ParseID(submissionRaw)
			version, e5 := strconv.ParseUint(versionRaw, 10, 64)
			if e1 != nil || e2 != nil || e3 != nil || e4 != nil || e5 != nil || id != value.ID || workID != value.WorkItemID || contractID != value.WorkContractID || submissionID != value.SubmissionID || statusRaw != string(value.Status) || version != uint64(value.Version) || value.Scope != scope || value.Validate() != nil {
				rows.Close()
				return nil, d.NewError(d.ErrorCodeInvalidScope, "review case typed columns differ from immutable state")
			}
			caseMap[id] = value
			contractIDs = append(contractIDs, value.WorkContractID)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return nil, mapSQLError("iterate Task verification review cases", err)
		}
		rows.Close()
		if len(caseMap) != len(caseIDs) {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "WorkItem review case pointer is missing")
		}
	}

	contractIDs = uniqueIDs(contractIDs)
	contracts, err := r.loadVerificationContracts(ctx, scope, contractIDs)
	if err != nil {
		return nil, err
	}
	// A review case's signed submission and execution contract are an exact pair.
	submissionIDs := make([]d.ID, 0, len(latestIDs)+len(caseMap))
	for _, delivery := range deliveries {
		if delivery.SubmissionRef != nil {
			submissionIDs = append(submissionIDs, delivery.SubmissionRef.ID)
		}
	}
	for _, c := range caseMap {
		submissionIDs = append(submissionIDs, c.SubmissionID)
	}
	submissionIDs = uniqueIDs(submissionIDs)
	submissions, err := r.loadVerificationSubmissions(ctx, scope, submissionIDs)
	if err != nil {
		return nil, err
	}
	for _, delivery := range deliveries {
		if delivery.SubmissionRef == nil {
			continue
		}
		sub, ok := submissions[delivery.SubmissionRef.ID]
		contract, cok := contracts[delivery.SubmissionRef.ContractID]
		if !ok || !cok || sub.ProtocolVersion == 2 || sub.Material.ContractID != delivery.SubmissionRef.ContractID || sub.Material.WorkItemID != delivery.WorkItemID || sub.Digest != delivery.SubmissionRef.Digest || contract.WorkItemID != delivery.WorkItemID || contract.SignedBinding != nil || contract.LatestSubmissionID == nil || *contract.LatestSubmissionID != sub.ID {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "contracts_v1 delivery does not bind its exact current submission")
		}
	}
	for _, c := range caseMap {
		contract, ok := contracts[c.WorkContractID]
		sub, sok := submissions[c.SubmissionID]
		if !ok || !sok || contract.SignedBinding == nil || contract.Status != d.ContractDelivered || contract.WorkItemID != c.WorkItemID || contract.LatestSubmissionID == nil || *contract.LatestSubmissionID != c.SubmissionID || sub.ProtocolVersion != 2 || sub.Material.ContractID != c.WorkContractID || sub.Material.WorkItemID != c.WorkItemID || sub.Digest != c.SubmissionDigest {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "signed review case does not bind its immutable execution submission")
		}
	}

	decisionMap, withdrawalMap, err := r.loadVerificationResolutions(ctx, scope, latestIDs, deliveries)
	if err != nil {
		return nil, err
	}
	reviewContracts, err := r.loadVerificationReviewContracts(ctx, scope, caseMap)
	if err != nil {
		return nil, err
	}

	out := make([]ports.TaskVerificationFacts, 0, len(ids))
	for _, id := range ids {
		item := items[id]
		if item.LatestDeliveryID == nil {
			if item.DeliveryRound != 0 || item.PendingDeliveryID != nil || item.LatestDeliveryDecisionID != nil || item.LatestDeliveryWithdrawalID != nil {
				return nil, d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery navigation has no latest delivery")
			}
		} else {
			delivery, ok := deliveries[*item.LatestDeliveryID]
			if !ok || delivery.WorkItemID != item.ID || delivery.Round != item.DeliveryRound {
				return nil, d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery navigation differs from latest delivery")
			}
			decision := decisionMap[delivery.ID]
			withdrawal := withdrawalMap[delivery.ID]
			if item.PendingDeliveryID != nil {
				if *item.PendingDeliveryID != delivery.ID || item.LatestDeliveryDecisionID != nil || item.LatestDeliveryWithdrawalID != nil || decision.ID != "" || withdrawal.ID != "" {
					return nil, d.NewError(d.ErrorCodeInvalidScope, "pending delivery is not unresolved latest delivery")
				}
			} else if item.LatestDeliveryDecisionID != nil {
				if decision.ID != *item.LatestDeliveryDecisionID || withdrawal.ID != "" {
					return nil, d.NewError(d.ErrorCodeInvalidScope, "latest delivery decision pointer is inconsistent")
				}
			} else if item.LatestDeliveryWithdrawalID != nil {
				if withdrawal.ID != *item.LatestDeliveryWithdrawalID || decision.ID != "" {
					return nil, d.NewError(d.ErrorCodeInvalidScope, "latest delivery withdrawal pointer is inconsistent")
				}
			} else {
				return nil, d.NewError(d.ErrorCodeInvalidScope, "latest delivery has no resolution")
			}
		}
		facts := ports.TaskVerificationFacts{WorkItemID: item.ID, Version: item.Version, Lifecycle: item.Lifecycle, ContractsEnabled: item.ContractsEnabled,
			CurrentLease: cloneWorkLease(item.CurrentLease), CurrentContractID: item.CurrentContractID, PendingDeliveryID: item.PendingDeliveryID, LatestDeliveryID: item.LatestDeliveryID,
			LatestDeliveryDecisionID: item.LatestDeliveryDecisionID, LatestDeliveryWithdrawalID: item.LatestDeliveryWithdrawalID, DeliveryRound: item.DeliveryRound,
			PendingReviewCaseID: item.PendingReviewCaseID, LatestReviewCaseID: item.LatestReviewCaseID, CorrectionReviewCaseID: item.CorrectionReviewCaseID}
		if item.CurrentContractID != nil {
			c, ok := contracts[*item.CurrentContractID]
			if !ok || c.Scope != scope || c.WorkItemID != item.ID {
				return nil, d.NewError(d.ErrorCodeInvalidScope, "current contract does not match Task")
			}
			copy := c
			facts.ExecutionContract = &copy
		}
		if item.LatestDeliveryID != nil {
			delivery := deliveries[*item.LatestDeliveryID]
			if item.PendingDeliveryID != nil {
				if workErr := delivery.ValidateForWork(item); workErr != nil {
					return nil, d.WrapError(d.ErrorCodeInvalidScope, "pending delivery does not match the current Task definition", workErr)
				}
			}
			f := ports.TaskVerificationDeliveryFacts{ID: delivery.ID, Scope: delivery.Scope, WorkItemID: delivery.WorkItemID, Authority: delivery.Authority, Round: delivery.Round, SupersedesDeliveryID: delivery.SupersedesDeliveryID, DefinitionDigest: delivery.DefinitionDigest, Digest: delivery.Digest, Summary: delivery.Summary, DeliveredAt: delivery.DeliveredAt, SubmissionRef: delivery.SubmissionRef}
			if delivery.Material != nil {
				f.ArtifactCount = len(delivery.Material.Artifacts)
				f.EvidenceCount = len(delivery.Material.EvidenceIDs)
				f.CriterionCount = len(delivery.Material.CriterionEvidence)
			} else if delivery.SubmissionRef != nil {
				sub := submissions[delivery.SubmissionRef.ID]
				f.ArtifactCount = len(sub.Material.Artifacts)
				f.EvidenceCount = len(sub.Material.EvidenceIDs)
				f.CriterionCount = len(sub.Material.CriterionEvidence)
			}
			facts.NativeDelivery = &f
			if decision := decisionMap[delivery.ID]; decision.ID != "" {
				copy := decision
				facts.NativeDecision = &copy
			}
			if withdrawal := withdrawalMap[delivery.ID]; withdrawal.ID != "" {
				copy := withdrawal
				facts.NativeWithdrawal = &copy
			}
		}
		selected := item.PendingReviewCaseID
		if selected == nil {
			selected = item.LatestReviewCaseID
		}
		if selected != nil {
			c := caseMap[*selected]
			copy := c
			facts.SignedReview = &copy
			if c.CurrentContractID != nil {
				rc, ok := reviewContracts[*c.CurrentContractID]
				if !ok {
					return nil, d.NewError(d.ErrorCodeInvalidScope, "review case current contract is missing")
				}
				rcopy := rc
				facts.ReviewContract = &rcopy
			}
		}
		if item.CorrectionReviewCaseID != nil {
			c := caseMap[*item.CorrectionReviewCaseID]
			copy := c
			facts.SignedCorrection = &copy
		}
		out = append(out, facts)
	}
	return out, nil
}

func (r taskVerificationReadRepository) loadPendingTaskVerificationCriteria(ctx context.Context, scope d.Scope, ids []d.ID, items map[d.ID]d.WorkItem) error {
	for _, id := range ids {
		item := items[id]
		item.Criteria = d.NewCriterionSet()
		items[id] = item
	}
	if len(ids) == 0 {
		return nil
	}
	in, idArgs := verificationIN(ids)
	args := append([]any{scope.NamespaceID.String(), scope.OutcomeID.String(), d.EntityKindWorkItem.String()}, idArgs...)
	rows, err := r.rows(ctx, `SELECT c.owner_id,c.id,c.criterion_revision,c.status,r.definition_json,r.required,r.verification_mode
FROM success_criteria c LEFT JOIN criterion_revisions r
ON r.namespace_id=c.namespace_id AND r.outcome_id=c.outcome_id AND r.criterion_id=c.id AND r.criterion_revision=c.criterion_revision
WHERE c.namespace_id=? AND c.outcome_id=? AND c.owner_kind=? AND c.owner_id IN `+in+` ORDER BY c.owner_id,c.created_at,c.id`, args...)
	if err != nil {
		return mapSQLError("read pending Task verification criteria", err)
	}
	for rows.Next() {
		var ownerRaw, criterionRaw, status string
		var revision int64
		var definitionRaw, mode sql.NullString
		var required sql.NullInt64
		if err = rows.Scan(&ownerRaw, &criterionRaw, &revision, &status, &definitionRaw, &required, &mode); err != nil {
			rows.Close()
			return err
		}
		ownerID, parseErr := d.ParseID(ownerRaw)
		if parseErr != nil {
			rows.Close()
			return parseErr
		}
		criterionID, parseErr := d.ParseID(criterionRaw)
		if parseErr != nil {
			rows.Close()
			return parseErr
		}
		item, ok := items[ownerID]
		if !ok || item.Scope != scope || item.PendingDeliveryID == nil || !definitionRaw.Valid || !required.Valid || !mode.Valid || (required.Int64 != 0 && required.Int64 != 1) {
			rows.Close()
			return d.NewError(d.ErrorCodeInvalidScope, "pending Task criterion definition is incomplete or out of scope")
		}
		var definition criterionDefinition
		if err = unmarshalJSON(definitionRaw.String, &definition); err != nil {
			rows.Close()
			return err
		}
		criterion := d.SuccessCriterion{ID: criterionID, OwnerRef: item.Ref(), Revision: d.CriterionRevision(revision), Title: definition.Title, Description: definition.Description, Required: required.Int64 == 1, VerificationMode: d.VerificationMode(mode.String), Status: d.CriterionStatus(status)}
		if err = criterion.Validate(); err != nil {
			rows.Close()
			return d.WrapError(d.ErrorCodeInvalidScope, "pending Task criterion definition is invalid", err)
		}
		item.Criteria.Items = append(item.Criteria.Items, criterion)
		items[ownerID] = item
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return mapSQLError("iterate pending Task verification criteria", err)
	}
	return rows.Close()
}

func uniqueIDs(ids []d.ID) []d.ID {
	seen := map[d.ID]bool{}
	out := make([]d.ID, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
func cloneWorkLease(value *d.WorkLease) *d.WorkLease {
	if value == nil {
		return nil
	}
	copy := *value
	if value.SessionCredentialID != nil {
		id := *value.SessionCredentialID
		copy.SessionCredentialID = &id
	}
	return &copy
}

func (r taskVerificationReadRepository) validateDeliveryMaterialBatch(ctx context.Context, scope d.Scope, ids []d.ID, deliveries map[d.ID]d.TaskDelivery) error {
	in, idArgs := verificationIN(ids)
	base := append([]any{scope.NamespaceID.String(), scope.OutcomeID.String()}, idArgs...)
	type materialRows struct {
		criteria         []d.TaskDeliveryCriterionSnapshot
		artifacts        []d.SubmissionArtifact
		evidence         []d.ID
		links            map[d.ID]d.SubmissionCriterionEvidence
		predecessorCount int
	}
	actual := map[d.ID]*materialRows{}
	for _, id := range ids {
		actual[id] = &materialRows{criteria: []d.TaskDeliveryCriterionSnapshot{}, artifacts: []d.SubmissionArtifact{}, evidence: []d.ID{}, links: map[d.ID]d.SubmissionCriterionEvidence{}}
	}
	rows, err := r.rows(ctx, `SELECT delivery_id,snapshot_index,criterion_id,criterion_revision,required,verification_mode FROM task_delivery_criteria WHERE namespace_id=? AND outcome_id=? AND delivery_id IN `+in+` ORDER BY delivery_id,snapshot_index`, base...)
	if err != nil {
		return mapSQLError("read verification delivery criteria", err)
	}
	for rows.Next() {
		var did, cid, mode string
		var index int
		var rev int64
		var required int
		if err = rows.Scan(&did, &index, &cid, &rev, &required, &mode); err != nil {
			rows.Close()
			return err
		}
		id, e1 := d.ParseID(did)
		criterion, e2 := d.ParseID(cid)
		if e1 != nil || e2 != nil {
			rows.Close()
			return d.NewError(d.ErrorCodeInvalidScope, "malformed delivery criterion reference")
		}
		actual[id].criteria = append(actual[id].criteria, d.TaskDeliveryCriterionSnapshot{CriterionID: criterion, CriterionRevision: d.CriterionRevision(rev), Required: required == 1, VerificationMode: d.VerificationMode(mode)})
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	rows, err = r.rows(ctx, `SELECT delivery_id,artifact_index,artifact_id,source_version,checksum FROM task_delivery_artifacts WHERE namespace_id=? AND outcome_id=? AND delivery_id IN `+in+` ORDER BY delivery_id,artifact_index`, base...)
	if err != nil {
		return mapSQLError("read verification delivery artifacts", err)
	}
	for rows.Next() {
		var did, aid, version, checksum string
		var index int
		if err = rows.Scan(&did, &index, &aid, &version, &checksum); err != nil {
			rows.Close()
			return err
		}
		id, e1 := d.ParseID(did)
		artifact, e2 := d.ParseID(aid)
		if e1 != nil || e2 != nil {
			rows.Close()
			return d.NewError(d.ErrorCodeInvalidScope, "malformed delivery artifact reference")
		}
		actual[id].artifacts = append(actual[id].artifacts, d.SubmissionArtifact{ArtifactID: artifact, SourceVersion: version, Checksum: checksum})
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	rows, err = r.rows(ctx, `SELECT te.delivery_id,te.evidence_id FROM task_delivery_evidence te JOIN evidence e ON e.namespace_id=te.namespace_id AND e.outcome_id=te.outcome_id AND e.id=te.evidence_id WHERE te.namespace_id=? AND te.outcome_id=? AND te.delivery_id IN `+in+` ORDER BY te.delivery_id,te.evidence_index`, base...)
	if err != nil {
		return mapSQLError("read verification delivery evidence", err)
	}
	for rows.Next() {
		var did, eid string
		if err = rows.Scan(&did, &eid); err != nil {
			rows.Close()
			return err
		}
		id, e1 := d.ParseID(did)
		evidence, e2 := d.ParseID(eid)
		if e1 != nil || e2 != nil {
			rows.Close()
			return d.NewError(d.ErrorCodeInvalidScope, "delivery evidence is unavailable or malformed")
		}
		actual[id].evidence = append(actual[id].evidence, evidence)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	rows, err = r.rows(ctx, `SELECT delivery_id,criterion_id,criterion_revision,evidence_id FROM task_delivery_criterion_evidence WHERE namespace_id=? AND outcome_id=? AND delivery_id IN `+in+` ORDER BY delivery_id,criterion_id,evidence_index`, base...)
	if err != nil {
		return mapSQLError("read verification criterion evidence", err)
	}
	for rows.Next() {
		var did, cid, eid string
		var revision int64
		if err = rows.Scan(&did, &cid, &revision, &eid); err != nil {
			rows.Close()
			return err
		}
		id, e1 := d.ParseID(did)
		criterion, e2 := d.ParseID(cid)
		evidence, e3 := d.ParseID(eid)
		if e1 != nil || e2 != nil || e3 != nil {
			rows.Close()
			return d.NewError(d.ErrorCodeInvalidScope, "malformed criterion evidence reference")
		}
		link, ok := actual[id].links[criterion]
		if !ok {
			link = d.SubmissionCriterionEvidence{CriterionID: criterion, CriterionRevision: d.CriterionRevision(revision), EvidenceIDs: []d.ID{}}
		} else if link.CriterionRevision != d.CriterionRevision(revision) {
			rows.Close()
			return d.NewError(d.ErrorCodeInvalidScope, "criterion evidence revision differs")
		}
		link.EvidenceIDs = append(link.EvidenceIDs, evidence)
		actual[id].links[criterion] = link
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	rows, err = r.rows(ctx, `SELECT delivery_id,predecessor_delivery_id,predecessor_round,predecessor_digest FROM task_delivery_predecessors WHERE namespace_id=? AND outcome_id=? AND delivery_id IN `+in, base...)
	if err != nil {
		return mapSQLError("read verification delivery predecessor", err)
	}
	for rows.Next() {
		var did, pid, round, digest string
		if err = rows.Scan(&did, &pid, &round, &digest); err != nil {
			rows.Close()
			return err
		}
		id, e1 := d.ParseID(did)
		if e1 != nil {
			rows.Close()
			return e1
		}
		actual[id].predecessorCount++
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, id := range ids {
		delivery := deliveries[id]
		rows := actual[id]
		var wantCriteria []d.TaskDeliveryCriterionSnapshot
		var wantArtifacts []d.SubmissionArtifact
		var wantEvidence []d.ID
		var wantLinks []d.SubmissionCriterionEvidence
		if delivery.Material != nil {
			wantCriteria = delivery.CriterionSnapshot
			wantArtifacts = delivery.Material.Artifacts
			wantEvidence = delivery.Material.EvidenceIDs
			wantLinks = delivery.Material.CriterionEvidence
		} else {
			wantCriteria = delivery.CriterionSnapshot
		}
		if !reflect.DeepEqual(rows.criteria, wantCriteria) || !reflect.DeepEqual(rows.artifacts, wantArtifacts) || !reflect.DeepEqual(rows.evidence, wantEvidence) {
			return d.NewError(d.ErrorCodeInvalidScope, "typed TaskDelivery material differs from immutable payload")
		}
		actualLinks := make([]d.SubmissionCriterionEvidence, 0, len(rows.links))
		for _, link := range wantLinks {
			got, ok := rows.links[link.CriterionID]
			if !ok {
				return d.NewError(d.ErrorCodeInvalidScope, "typed TaskDelivery criterion evidence is missing")
			}
			actualLinks = append(actualLinks, got)
			delete(rows.links, link.CriterionID)
		}
		if len(rows.links) != 0 || !reflect.DeepEqual(actualLinks, wantLinks) {
			return d.NewError(d.ErrorCodeInvalidScope, "typed TaskDelivery criterion evidence differs from payload")
		}
		if delivery.SupersedesDeliveryID == nil {
			if rows.predecessorCount != 0 || delivery.Round != 1 {
				return d.NewError(d.ErrorCodeInvalidScope, "unexpected first delivery predecessor")
			}
		} else if rows.predecessorCount != 1 {
			return d.NewError(d.ErrorCodeInvalidScope, "delivery predecessor reference is missing")
		}
	}
	return nil
}

func (r taskVerificationReadRepository) loadVerificationContracts(ctx context.Context, scope d.Scope, ids []d.ID) (map[d.ID]d.WorkContract, error) {
	out := map[d.ID]d.WorkContract{}
	if len(ids) == 0 {
		return out, nil
	}
	in, vals := verificationIN(ids)
	args := append([]any{scope.NamespaceID.String(), scope.OutcomeID.String()}, vals...)
	rows, err := r.rows(ctx, `SELECT id,work_item_id,state_json,spec_json,credential_binding_version,holder_credential_id,session_credential_id FROM work_contracts WHERE namespace_id=? AND outcome_id=? AND id IN `+in, args...)
	if err != nil {
		return nil, mapSQLError("read verification contracts", err)
	}
	for rows.Next() {
		var idRaw, workRaw, state, spec string
		var binding int
		var holder, session sql.NullString
		if err = rows.Scan(&idRaw, &workRaw, &state, &spec, &binding, &holder, &session); err != nil {
			rows.Close()
			return nil, err
		}
		var c d.WorkContract
		if err = json.Unmarshal([]byte(state), &c); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal([]byte(spec), &c.Spec); err != nil {
			rows.Close()
			return nil, err
		}
		id, e1 := d.ParseID(idRaw)
		work, e2 := d.ParseID(workRaw)
		if e1 != nil || e2 != nil || id != c.ID || work != c.WorkItemID || c.Scope != scope || c.Validate() != nil || validateContractCredentialColumns(c, binding, holder, session) != nil {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "contracts_v1 contract columns differ from state")
		}
		out[id] = c
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, mapSQLError("iterate verification contracts", err)
	}
	rows.Close()
	rows, err = r.rows(ctx, `SELECT id,work_item_id,state_json,spec_json,status,version,lease_version FROM signed_work_contracts WHERE namespace_id=? AND outcome_id=? AND id IN `+in, args...)
	if err != nil {
		return nil, mapSQLError("read signed verification contracts", err)
	}
	for rows.Next() {
		var idRaw, workRaw, state, spec, status, version, lease string
		if err = rows.Scan(&idRaw, &workRaw, &state, &spec, &status, &version, &lease); err != nil {
			rows.Close()
			return nil, err
		}
		var c d.WorkContract
		if err = json.Unmarshal([]byte(state), &c); err != nil {
			rows.Close()
			return nil, err
		}
		var signed d.SignedWorkSpec
		if err = json.Unmarshal([]byte(spec), &signed); err != nil {
			rows.Close()
			return nil, err
		}
		c.Spec = signed.WorkSpec()
		id, e1 := d.ParseID(idRaw)
		work, e2 := d.ParseID(workRaw)
		v, e3 := strconv.ParseUint(version, 10, 64)
		lv, e4 := strconv.ParseUint(lease, 10, 64)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || id != c.ID || work != c.WorkItemID || c.Scope != scope || status != string(c.Status) || v != uint64(c.Version) || lv != uint64(c.LeaseVersion) || c.Validate() != nil {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "signed contract columns differ from state")
		}
		if _, duplicate := out[id]; duplicate {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "contract ID exists in multiple protocols")
		}
		out[id] = c
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, mapSQLError("iterate signed verification contracts", err)
	}
	rows.Close()
	for _, id := range ids {
		if _, ok := out[id]; !ok {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "referenced execution contract is missing")
		}
	}
	return out, nil
}

func (r taskVerificationReadRepository) loadVerificationSubmissions(ctx context.Context, scope d.Scope, ids []d.ID) (map[d.ID]d.WorkSubmission, error) {
	out := map[d.ID]d.WorkSubmission{}
	if len(ids) == 0 {
		return out, nil
	}
	in, vals := verificationIN(ids)
	args := append([]any{scope.NamespaceID.String(), scope.OutcomeID.String()}, vals...)
	for _, table := range []string{"work_contract_submissions", "signed_work_contract_submissions"} {
		rows, err := r.rows(ctx, `SELECT id,work_item_id,contract_id,submission_digest,payload_json FROM `+table+` WHERE namespace_id=? AND outcome_id=? AND id IN `+in, args...)
		if err != nil {
			return nil, mapSQLError("read verification submissions", err)
		}
		for rows.Next() {
			var idRaw, workRaw, contractRaw, digest, payload string
			if err = rows.Scan(&idRaw, &workRaw, &contractRaw, &digest, &payload); err != nil {
				rows.Close()
				return nil, err
			}
			var s d.WorkSubmission
			if err = json.Unmarshal([]byte(payload), &s); err != nil {
				rows.Close()
				return nil, err
			}
			id, e1 := d.ParseID(idRaw)
			work, e2 := d.ParseID(workRaw)
			contract, e3 := d.ParseID(contractRaw)
			if e1 != nil || e2 != nil || e3 != nil || id != s.ID || work != s.Material.WorkItemID || contract != s.Material.ContractID || digest != s.Digest || s.Scope != scope || s.Validate() != nil {
				return nil, d.NewError(d.ErrorCodeInvalidScope, "submission columns differ from immutable state")
			}
			if _, duplicate := out[id]; duplicate {
				return nil, d.NewError(d.ErrorCodeInvalidScope, "submission ID exists in multiple protocols")
			}
			out[id] = s
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return nil, mapSQLError("iterate verification submissions", err)
		}
		rows.Close()
	}
	for _, id := range ids {
		if _, ok := out[id]; !ok {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "referenced immutable submission is missing")
		}
	}
	return out, nil
}

func (r taskVerificationReadRepository) loadVerificationResolutions(ctx context.Context, scope d.Scope, ids []d.ID, deliveries map[d.ID]d.TaskDelivery) (map[d.ID]d.TaskDeliveryDecision, map[d.ID]d.TaskDeliveryWithdrawal, error) {
	decisions := map[d.ID]d.TaskDeliveryDecision{}
	withdrawals := map[d.ID]d.TaskDeliveryWithdrawal{}
	if len(ids) == 0 {
		return decisions, withdrawals, nil
	}
	in, vals := verificationIN(ids)
	args := append([]any{scope.NamespaceID.String(), scope.OutcomeID.String()}, vals...)
	rows, err := r.rows(ctx, `SELECT id,work_item_id,delivery_id,delivery_round,delivery_digest,result,payload_json FROM task_delivery_decisions WHERE namespace_id=? AND outcome_id=? AND delivery_id IN `+in, args...)
	if err != nil {
		return nil, nil, mapSQLError("read verification delivery decisions", err)
	}
	for rows.Next() {
		var idRaw, workRaw, deliveryRaw, roundRaw, digest, result, payload string
		if err = rows.Scan(&idRaw, &workRaw, &deliveryRaw, &roundRaw, &digest, &result, &payload); err != nil {
			rows.Close()
			return nil, nil, err
		}
		var value d.TaskDeliveryDecision
		if err = json.Unmarshal([]byte(payload), &value); err != nil {
			rows.Close()
			return nil, nil, err
		}
		id, e1 := d.ParseID(idRaw)
		work, e2 := d.ParseID(workRaw)
		deliveryID, e3 := d.ParseID(deliveryRaw)
		round, e4 := strconv.ParseUint(roundRaw, 10, 64)
		delivery, ok := deliveries[deliveryID]
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || roundRaw != strconv.FormatUint(round, 10) || id != value.ID || work != value.WorkItemID || deliveryID != value.DeliveryID || round != value.Round || digest != value.DeliveryDigest || result != string(value.Result) || !ok || value.ValidateForDelivery(delivery) != nil {
			return nil, nil, d.NewError(d.ErrorCodeInvalidScope, "decision columns differ from exact delivery")
		}
		decisions[deliveryID] = value
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, nil, mapSQLError("iterate verification decisions", err)
	}
	rows.Close()
	rows, err = r.rows(ctx, `SELECT id,work_item_id,delivery_id,delivery_round,delivery_digest,payload_json FROM task_delivery_withdrawals WHERE namespace_id=? AND outcome_id=? AND delivery_id IN `+in, args...)
	if err != nil {
		return nil, nil, mapSQLError("read verification delivery withdrawals", err)
	}
	for rows.Next() {
		var idRaw, workRaw, deliveryRaw, roundRaw, digest, payload string
		if err = rows.Scan(&idRaw, &workRaw, &deliveryRaw, &roundRaw, &digest, &payload); err != nil {
			rows.Close()
			return nil, nil, err
		}
		var value d.TaskDeliveryWithdrawal
		if err = json.Unmarshal([]byte(payload), &value); err != nil {
			rows.Close()
			return nil, nil, err
		}
		id, e1 := d.ParseID(idRaw)
		work, e2 := d.ParseID(workRaw)
		deliveryID, e3 := d.ParseID(deliveryRaw)
		round, e4 := strconv.ParseUint(roundRaw, 10, 64)
		delivery, ok := deliveries[deliveryID]
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || roundRaw != strconv.FormatUint(round, 10) || id != value.ID || work != value.WorkItemID || deliveryID != value.DeliveryID || round != value.Round || digest != value.DeliveryDigest || !ok || value.ValidateForDelivery(delivery) != nil {
			return nil, nil, d.NewError(d.ErrorCodeInvalidScope, "withdrawal columns differ from exact delivery")
		}
		withdrawals[deliveryID] = value
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, nil, mapSQLError("iterate verification withdrawals", err)
	}
	rows.Close()
	for id := range decisions {
		if _, ok := withdrawals[id]; ok {
			return nil, nil, d.NewError(d.ErrorCodeInvalidScope, "delivery has both a decision and withdrawal")
		}
	}
	return decisions, withdrawals, nil
}

func (r taskVerificationReadRepository) loadVerificationReviewContracts(ctx context.Context, scope d.Scope, cases map[d.ID]d.ReviewCase) (map[d.ID]d.ReviewContract, error) {
	ids := []d.ID{}
	for _, c := range cases {
		if c.CurrentContractID != nil {
			ids = append(ids, *c.CurrentContractID)
		}
	}
	ids = uniqueIDs(ids)
	out := map[d.ID]d.ReviewContract{}
	if len(ids) == 0 {
		return out, nil
	}
	in, vals := verificationIN(ids)
	args := append([]any{scope.NamespaceID.String(), scope.OutcomeID.String()}, vals...)
	rows, err := r.rows(ctx, `SELECT id,case_id,work_item_id,status,version,lease_version,state_json FROM work_review_contracts WHERE namespace_id=? AND outcome_id=? AND id IN `+in, args...)
	if err != nil {
		return nil, mapSQLError("read verification review contracts", err)
	}
	for rows.Next() {
		var idRaw, caseRaw, workRaw, status, versionRaw, leaseRaw, state string
		if err = rows.Scan(&idRaw, &caseRaw, &workRaw, &status, &versionRaw, &leaseRaw, &state); err != nil {
			rows.Close()
			return nil, err
		}
		var c d.ReviewContract
		if err = json.Unmarshal([]byte(state), &c); err != nil {
			rows.Close()
			return nil, err
		}
		id, e1 := d.ParseID(idRaw)
		caseID, e2 := d.ParseID(caseRaw)
		workID, e3 := d.ParseID(workRaw)
		version, e4 := strconv.ParseUint(versionRaw, 10, 64)
		lease, e5 := strconv.ParseUint(leaseRaw, 10, 64)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || e5 != nil || id != c.ID || caseID != c.CaseID || workID != c.WorkItemID || status != string(c.Status) || version != uint64(c.Version) || lease != uint64(c.LeaseVersion) || c.Scope != scope || c.Validate() != nil {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "review contract columns differ from state")
		}
		out[id] = c
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, mapSQLError("iterate verification review contracts", err)
	}
	rows.Close()
	for _, id := range ids {
		if _, ok := out[id]; !ok {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "referenced review contract is missing")
		}
	}
	for _, caseValue := range cases {
		if caseValue.CurrentContractID != nil {
			contract := out[*caseValue.CurrentContractID]
			if contract.CaseID != caseValue.ID || contract.WorkItemID != caseValue.WorkItemID || contract.SubmissionID != caseValue.SubmissionID || contract.SubmissionDigest != caseValue.SubmissionDigest {
				return nil, d.NewError(d.ErrorCodeInvalidScope, "review contract does not match its case")
			}
		}
	}
	return out, nil
}

var _ ports.TaskVerificationReadUnitOfWork = (*unitOfWork)(nil)
