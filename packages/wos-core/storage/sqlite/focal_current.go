package sqlite

import (
	"context"
	"database/sql"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

const focalWorkColumns = `id,version,title,description,objective_id,lifecycle,priority,not_before,result_summary,COALESCE(fencing_token_decimal,CAST(last_fencing_token AS TEXT)),lease_claim_id,lease_principal_id,lease_actor_json,lease_acquired_at,lease_expires_at,created_at,updated_at,contracts_enabled,current_contract_id,execution_spec_json,pending_review_case_id,latest_review_case_id,correction_review_case_id,lease_credential_binding_version,lease_holder_credential_id,lease_session_credential_id`

func (r continuityReadRepository) CurrentWork(ctx context.Context, scope d.Scope, id d.ID) (ports.FocalCurrentWork, error) {
	var result ports.FocalCurrentWork
	if e := scope.Validate(); e != nil {
		return result, e
	}
	if e := id.Validate(); e != nil {
		return result, e
	}
	work, e := scanWorkItem(r.row(ctx, "SELECT "+focalWorkColumns+" FROM work_items WHERE namespace_id=? AND outcome_id=? AND id=?", scope.NamespaceID.String(), scope.OutcomeID.String(), id.String()), scope)
	if e != nil {
		return result, e
	}
	criteria, current, history, e := r.currentOwnerProof(ctx, work.Ref())
	if e != nil {
		return result, e
	}
	actors, e := loadActorLinks(ctx, r.u.tx, work.Ref(), "assignee")
	if e != nil {
		return result, e
	}
	work.Criteria = criteria
	work.CurrentConclusion = current
	work.AssigneeRefs = actors
	work.ConclusionHistory = nil
	return ports.FocalCurrentWork{Work: work, History: history}, nil
}
func (r continuityReadRepository) CurrentObjective(ctx context.Context, scope d.Scope, id d.ID) (ports.FocalCurrentObjective, error) {
	var result ports.FocalCurrentObjective
	if e := scope.Validate(); e != nil {
		return result, e
	}
	if e := id.Validate(); e != nil {
		return result, e
	}
	v, e := scanObjective(r.row(ctx, `SELECT id,version,title,description,parent_objective_id,lifecycle,priority,required_for_outcome,created_at,updated_at,completion_mode,completion_cycle FROM objectives WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), id.String()), scope)
	if e != nil {
		return result, e
	}
	owner, e := r.u.outcomes.Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if e != nil {
		return result, e
	}
	if e = v.ValidateCompletionConfiguration(owner); e != nil {
		return result, d.WrapError(d.ErrorCodeInvalidArgument, "persisted objective completion configuration is invalid", e)
	}
	criteria, current, history, e := r.currentOwnerProof(ctx, v.Ref())
	if e != nil {
		return result, e
	}
	actors, e := loadActorLinks(ctx, r.u.tx, v.Ref(), "owner")
	if e != nil {
		return result, e
	}
	v.Criteria = criteria
	v.CurrentConclusion = current
	v.OwnerRefs = actors
	v.ConclusionHistory = nil
	return ports.FocalCurrentObjective{Objective: v, History: history}, nil
}
func (r continuityReadRepository) FocalOutcome(ctx context.Context, scope d.Scope) (d.Outcome, error) {
	var result d.Outcome
	if e := scope.Validate(); e != nil {
		return result, e
	}
	var version d.Version
	var title, life, priority, completionPolicy, operationalProfile string
	var archived sql.NullInt64
	e := r.row(ctx, `SELECT version,substr(title,1,512),lifecycle,priority,archived_at,completion_policy,operational_profile FROM outcomes WHERE namespace_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String()).Scan(&version, &title, &life, &priority, &archived, &completionPolicy, &operationalProfile)
	if e != nil {
		return result, mapSQLError("read focal Outcome metadata", e)
	}
	result = d.Outcome{CompletionPolicy: d.CompletionPolicy(completionPolicy), OperationalProfile: d.OperationalProfile(operationalProfile), ID: scope.OutcomeID, NamespaceID: scope.NamespaceID, Version: version, Title: title, Lifecycle: d.OutcomeLifecycle(life), Priority: d.Priority(priority)}
	if archived.Valid {
		at := decodeTime(archived.Int64)
		result.ArchivedAt = &at
	}
	return result, nil
}
func (r continuityReadRepository) currentOwnerProof(ctx context.Context, owner d.EntityRef) (d.CriterionSet, *d.Conclusion, ports.FocalHistoryCounts, error) {
	set := d.NewCriterionSet()
	counts := ports.FocalHistoryCounts{}
	var current *d.Conclusion
	args := []any{owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String(), owner.Kind.String()}
	rows, e := r.rows(ctx, `SELECT c.id,c.criterion_revision,c.status,r.definition_json,r.required,r.verification_mode FROM success_criteria c JOIN criterion_revisions r ON r.namespace_id=c.namespace_id AND r.outcome_id=c.outcome_id AND r.criterion_id=c.id AND r.criterion_revision=c.criterion_revision WHERE c.namespace_id=? AND c.outcome_id=? AND c.owner_id=? AND c.owner_kind=? ORDER BY c.created_at,c.id`, args...)
	if e != nil {
		return set, nil, counts, e
	}
	for rows.Next() {
		var id, status, definition, mode string
		var revision d.CriterionRevision
		var required int
		if e = rows.Scan(&id, &revision, &status, &definition, &required, &mode); e != nil {
			rows.Close()
			return set, nil, counts, e
		}
		cid, e := d.ParseID(id)
		if e != nil {
			rows.Close()
			return set, nil, counts, e
		}
		var def criterionDefinition
		if e = unmarshalJSON(definition, &def); e != nil {
			rows.Close()
			return set, nil, counts, e
		}
		c := d.SuccessCriterion{ID: cid, OwnerRef: owner, Revision: revision, Title: def.Title, Description: def.Description, Required: required == 1, Status: d.CriterionStatus(status), VerificationMode: d.VerificationMode(mode)}
		if e = c.Validate(); e != nil {
			rows.Close()
			return set, nil, counts, e
		}
		set.Items = append(set.Items, c)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return set, nil, counts, e
	}
	// Owner selection is resolved before touching assessment history. IN uses the
	// indexed criterion/current-assessment IDs even before planner statistics warm.
	scopedArgs := []any{owner.NamespaceID.String(), owner.OutcomeID.String(), owner.NamespaceID.String(), owner.OutcomeID.String(), owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String(), owner.Kind.String()}
	rows, e = r.rows(ctx, `SELECT a.id,a.criterion_id,a.criterion_revision,a.result,a.rationale,a.principal_id,a.actor_json,a.assessed_at,a.evaluator_ref_json,a.supersedes_assessment_id,a.submission_id,a.submission_digest FROM criterion_assessments a WHERE a.namespace_id=? AND a.outcome_id=? AND a.id IN(SELECT p.assessment_id FROM criterion_current_assessments p WHERE p.namespace_id=? AND p.outcome_id=? AND p.criterion_id IN(SELECT c.id FROM success_criteria c WHERE c.namespace_id=? AND c.outcome_id=? AND c.owner_id=? AND c.owner_kind=?)) ORDER BY a.assessed_at,a.id`, scopedArgs...)
	if e != nil {
		return set, nil, counts, e
	}
	currentByID := map[d.ID]d.CriterionAssessment{}
	for rows.Next() {
		a, e := scanAssessment(rows)
		if e != nil {
			rows.Close()
			return set, nil, counts, e
		}
		currentByID[a.ID] = a
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return set, nil, counts, e
	}
	rows, e = r.rows(ctx, `SELECT p.assessment_id,p.evidence_id FROM criterion_assessment_evidence p WHERE p.namespace_id=? AND p.outcome_id=? AND p.assessment_id IN(SELECT a.assessment_id FROM criterion_current_assessments a WHERE a.namespace_id=? AND a.outcome_id=? AND a.criterion_id IN(SELECT c.id FROM success_criteria c WHERE c.namespace_id=? AND c.outcome_id=? AND c.owner_id=? AND c.owner_kind=?)) ORDER BY p.assessment_id,p.evidence_id`, scopedArgs...)
	if e != nil {
		return set, nil, counts, e
	}
	for rows.Next() {
		var rawID, rawEvidence string
		if e = rows.Scan(&rawID, &rawEvidence); e != nil {
			rows.Close()
			return set, nil, counts, e
		}
		aid, e := d.ParseID(rawID)
		if e != nil {
			rows.Close()
			return set, nil, counts, e
		}
		eid, e := d.ParseID(rawEvidence)
		if e != nil {
			rows.Close()
			return set, nil, counts, e
		}
		a, ok := currentByID[aid]
		if !ok {
			rows.Close()
			return set, nil, counts, d.NewError(d.ErrorCodeInvalidArgument, "current assessment evidence has no owner projection")
		}
		a.EvidenceIDs = append(a.EvidenceIDs, eid)
		currentByID[aid] = a
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return set, nil, counts, e
	}
	for _, a := range currentByID {
		set.CurrentAssessments[a.CriterionID] = a
	}
	ownerArgs := []any{owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String()}
	storageID, conclusion, e := scanConclusion(r.row(ctx, `SELECT c.id,c.public_id,c.owner_version,c.lifecycle_result,c.principal_id,c.actor_json,c.recorded_at,c.rationale,c.obligations_snapshot_json,c.submission_id,c.submission_digest FROM aggregate_current_conclusions p JOIN conclusions c ON c.namespace_id=p.namespace_id AND c.outcome_id=p.outcome_id AND c.id=p.conclusion_id WHERE p.namespace_id=? AND p.outcome_id=? AND p.owner_id=?`, ownerArgs...), owner)
	if e != nil {
		if code, _ := d.ErrorCodeOf(e); e != sql.ErrNoRows && code != d.ErrorCodeNotFound {
			return set, nil, counts, e
		}
	} else {
		current = &conclusion
		rows, e = r.rows(ctx, `SELECT a.id,a.criterion_id,a.criterion_revision,a.result FROM conclusion_assessments p JOIN criterion_assessments a ON a.namespace_id=p.namespace_id AND a.outcome_id=p.outcome_id AND a.id=p.assessment_id WHERE p.namespace_id=? AND p.outcome_id=? AND p.conclusion_id=? ORDER BY a.criterion_id,a.id`, owner.NamespaceID.String(), owner.OutcomeID.String(), storageID)
		if e != nil {
			return set, nil, counts, e
		}
		current.Assessments = []d.CriterionAssessmentRef{}
		for rows.Next() {
			var id, criterion, result string
			var revision d.CriterionRevision
			if e = rows.Scan(&id, &criterion, &revision, &result); e != nil {
				rows.Close()
				return set, nil, counts, e
			}
			aid, e := d.ParseID(id)
			if e != nil {
				rows.Close()
				return set, nil, counts, e
			}
			cid, e := d.ParseID(criterion)
			if e != nil {
				rows.Close()
				return set, nil, counts, e
			}
			current.Assessments = append(current.Assessments, d.CriterionAssessmentRef{AssessmentID: aid, CriterionID: cid, CriterionRevision: revision, Result: d.AssessmentResult(result)})
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return set, nil, counts, e
		}
		if e = current.Validate(); e != nil {
			return set, nil, counts, e
		}
	}
	// Exact omission counts are scalar owner-indexed reads, not loaded histories.
	queryArgs := []any{owner.NamespaceID.String(), owner.OutcomeID.String(), owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String(), owner.Kind.String()}
	if e = r.row(ctx, `SELECT COUNT(*) FROM criterion_assessments a WHERE a.namespace_id=? AND a.outcome_id=? AND a.criterion_id IN(SELECT c.id FROM success_criteria c WHERE c.namespace_id=? AND c.outcome_id=? AND c.owner_id=? AND c.owner_kind=?)`, queryArgs...).Scan(&counts.Assessments); e != nil {
		return set, nil, counts, e
	}
	if e = r.row(ctx, `SELECT COUNT(*) FROM criterion_revisions a WHERE a.namespace_id=? AND a.outcome_id=? AND a.criterion_id IN(SELECT c.id FROM success_criteria c WHERE c.namespace_id=? AND c.outcome_id=? AND c.owner_id=? AND c.owner_kind=?)`, queryArgs...).Scan(&counts.Definitions); e != nil {
		return set, nil, counts, e
	}
	if e = r.row(ctx, `SELECT COUNT(*) FROM conclusions WHERE namespace_id=? AND outcome_id=? AND owner_id=?`, ownerArgs...).Scan(&counts.Conclusions); e != nil {
		return set, nil, counts, e
	}
	if current != nil {
		counts.Conclusions--
	}
	return set, current, counts, nil
}
