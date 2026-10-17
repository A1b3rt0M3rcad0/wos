package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strings"
)

// Projection reads headers and the current conclusion's relevant proof, never
// aggregate histories or documentary descriptions unrelated to classification.
func (r continuityReadRepository) Projection(ctx context.Context, scope d.Scope, options ports.ContinuityProjectionOptions) (ports.ContinuityProjectionInputs, error) {
	data := ports.ContinuityProjectionInputs{Objectives: []d.Objective{}, WorkItems: []d.WorkItem{}, Relations: []d.Relation{}, Blockers: []d.Blocker{}, Evidence: []d.Evidence{}, ActivePlans: []ports.ContinuityActivePlan{}}
	if e := scope.Validate(); e != nil {
		return data, e
	}
	var title, lifecycle, priority, external string
	var version d.Version
	var archived sql.NullInt64
	e := r.row(ctx, `SELECT version,substr(title,1,512),lifecycle,priority,archived_at,external_context_json FROM outcomes WHERE tenant_id=? AND id=?`, scope.TenantID.String(), scope.OutcomeID.String()).Scan(&version, &title, &lifecycle, &priority, &archived, &external)
	if e != nil {
		return data, mapSQLError("read operational Outcome header", e)
	}
	data.Outcome = d.Outcome{ID: scope.OutcomeID, TenantID: scope.TenantID, Version: version, Title: title, Lifecycle: d.OutcomeLifecycle(lifecycle), Priority: d.Priority(priority), Criteria: d.NewCriterionSet()}
	if e = unmarshalJSON(external, &data.Outcome.ExternalContext); e != nil {
		return data, e
	}
	if archived.Valid {
		at := decodeTime(archived.Int64)
		data.Outcome.ArchivedAt = &at
	}
	if options.Plans {
		data.ActivePlans, e = r.projectionPlans(ctx, scope)
		if e != nil {
			return data, e
		}
	}
	needHeaders := options.Operational || options.Conclusions || len(data.ActivePlans) > 0
	if needHeaders {
		data.Objectives, e = r.projectionObjectiveHeaders(ctx, scope)
		if e != nil {
			return data, e
		}
		data.WorkItems, e = r.projectionWorkHeaders(ctx, scope, options.Operational || options.Progress, options.Conclusions, data.ActivePlans)
		if e != nil {
			return data, e
		}
	}
	if options.Operational || len(data.ActivePlans) > 0 {
		sourceIDs := []d.ID{}
		for _, v := range data.WorkItems {
			sourceIDs = append(sourceIDs, v.ID)
		}
		data.Relations, e = r.projectionDependencies(ctx, scope, sourceIDs, options.Operational)
		if e != nil {
			return data, e
		}
		// A global operational group needs every dependency endpoint's lifecycle.
		// Plans alone retain only referenced work plus dependency endpoint headers.
		if !options.Operational && len(data.ActivePlans) > 0 {
			data.WorkItems, e = r.addProjectionDependencyTargets(ctx, scope, data.WorkItems, data.Relations)
			if e != nil {
				return data, e
			}
		}
		data.Blockers, e = r.projectionBlockers(ctx, scope, false)
		if e != nil {
			return data, e
		}
	}
	if options.Conclusions {
		proof, e := r.projectionConclusionProof(ctx, scope)
		if e != nil {
			return data, e
		}
		if p := proof[data.Outcome.Ref()]; p != nil {
			data.Outcome.Criteria = p.criteria
			data.Outcome.CurrentConclusion = p.current
		}
		for i := range data.Objectives {
			if p := proof[data.Objectives[i].Ref()]; p != nil {
				data.Objectives[i].Criteria = p.criteria
				data.Objectives[i].CurrentConclusion = p.current
			}
		}
		for i := range data.WorkItems {
			if p := proof[data.WorkItems[i].Ref()]; p != nil {
				data.WorkItems[i].Criteria = p.criteria
				data.WorkItems[i].CurrentConclusion = p.current
			}
		}
		data.Evidence, e = r.projectionConclusionEvidence(ctx, scope)
		if e != nil {
			return data, e
		}
	}
	if options.Progress {
		data.RequiredCriteria, data.MetCriteria, data.WaivedCriteria, e = r.projectionProofCounts(ctx, scope)
		if e != nil {
			return data, e
		}
	}
	return data, nil
}
func (r continuityReadRepository) projectionObjectiveHeaders(ctx context.Context, scope d.Scope) ([]d.Objective, error) {
	rows, e := r.rows(ctx, `SELECT id,version,substr(title,1,512),lifecycle,priority,parent_objective_id,required_for_outcome FROM objectives WHERE tenant_id=? AND outcome_id=? ORDER BY id`, scope.TenantID.String(), scope.OutcomeID.String())
	if e != nil {
		return nil, mapSQLError("read operational Objective headers", e)
	}
	defer rows.Close()
	values := []d.Objective{}
	for rows.Next() {
		var id, title, life, priority string
		var parent sql.NullString
		var version d.Version
		var required int
		if e = rows.Scan(&id, &version, &title, &life, &priority, &parent, &required); e != nil {
			return nil, e
		}
		parsed, e := d.ParseID(id)
		if e != nil {
			return nil, e
		}
		v := d.Objective{ID: parsed, Scope: scope, Version: version, Title: title, Lifecycle: d.ObjectiveLifecycle(life), Priority: d.Priority(priority), RequiredForOutcome: required == 1, Criteria: d.NewCriterionSet()}
		if parent.Valid {
			value, e := d.ParseID(parent.String)
			if e != nil {
				return nil, e
			}
			v.ParentObjectiveID = &value
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

const projectionWorkColumns = `id,version,substr(title,1,512),lifecycle,priority,objective_id,not_before,lease_acquired_at,lease_expires_at`

func scanProjectionWork(row rowScanner, scope d.Scope) (d.WorkItem, error) {
	var id, title, life, priority string
	var version d.Version
	var objective sql.NullString
	var before, acquired, expires sql.NullInt64
	if e := row.Scan(&id, &version, &title, &life, &priority, &objective, &before, &acquired, &expires); e != nil {
		return d.WorkItem{}, e
	}
	parsed, e := d.ParseID(id)
	if e != nil {
		return d.WorkItem{}, e
	}
	v := d.WorkItem{ID: parsed, Scope: scope, Version: version, Title: title, Lifecycle: d.WorkItemLifecycle(life), Priority: d.Priority(priority), Criteria: d.NewCriterionSet()}
	if objective.Valid {
		id, e := d.ParseID(objective.String)
		if e != nil {
			return v, e
		}
		v.ObjectiveID = &id
	}
	if before.Valid {
		at := decodeTime(before.Int64)
		v.NotBefore = &at
	}
	if acquired.Valid && expires.Valid {
		v.CurrentLease = &d.WorkLease{AcquiredAt: decodeTime(acquired.Int64), ExpiresAt: decodeTime(expires.Int64)}
	}
	return v, nil
}
func (r continuityReadRepository) projectionWorkHeaders(ctx context.Context, scope d.Scope, all, concluded bool, plans []ports.ContinuityActivePlan) ([]d.WorkItem, error) {
	values := []d.WorkItem{}
	seen := map[d.ID]bool{}
	read := func(condition string, args ...any) error {
		query := "SELECT " + projectionWorkColumns + " FROM work_items WHERE tenant_id=? AND outcome_id=?" + condition + " ORDER BY id"
		params := []any{scope.TenantID.String(), scope.OutcomeID.String()}
		params = append(params, args...)
		rows, e := r.rows(ctx, query, params...)
		if e != nil {
			return mapSQLError("read operational WorkItem headers", e)
		}
		defer rows.Close()
		for rows.Next() {
			v, e := scanProjectionWork(rows, scope)
			if e != nil {
				return e
			}
			if !seen[v.ID] {
				values = append(values, v)
				seen[v.ID] = true
			}
		}
		return rows.Err()
	}
	if all {
		if e := read(""); e != nil {
			return nil, e
		}
		return values, nil
	}
	if concluded {
		if e := read(` AND EXISTS(SELECT 1 FROM aggregate_current_conclusions a WHERE a.tenant_id=work_items.tenant_id AND a.outcome_id=work_items.outcome_id AND a.owner_id=work_items.id)`); e != nil {
			return nil, e
		}
	}
	refs := []d.ID{}
	for _, p := range plans {
		for _, node := range p.Revision.Nodes {
			if node.TargetRef != nil && node.TargetRef.Scope == scope && node.TargetRef.Kind == d.EntityKindWorkItem && !seen[node.TargetRef.ID] {
				refs = append(refs, node.TargetRef.ID)
			}
		}
	}
	for start := 0; start < len(refs); start += 100 {
		end := min(start+100, len(refs))
		q := " AND id IN (" + strings.TrimSuffix(strings.Repeat("?,", end-start), ",") + ")"
		args := []any{}
		for _, id := range refs[start:end] {
			args = append(args, id.String())
		}
		if e := read(q, args...); e != nil {
			return nil, e
		}
	}
	return values, nil
}
func (r continuityReadRepository) addProjectionDependencyTargets(ctx context.Context, scope d.Scope, values []d.WorkItem, relations []d.Relation) ([]d.WorkItem, error) {
	seen := map[d.ID]bool{}
	for _, v := range values {
		seen[v.ID] = true
	}
	refs := []d.ID{}
	for _, v := range relations {
		if v.TargetRef.Kind == d.EntityKindWorkItem && !seen[v.TargetRef.ID] {
			refs = append(refs, v.TargetRef.ID)
			seen[v.TargetRef.ID] = true
		}
	}
	for start := 0; start < len(refs); start += 100 {
		end := min(start+100, len(refs))
		q := "SELECT " + projectionWorkColumns + " FROM work_items WHERE tenant_id=? AND outcome_id=? AND id IN (" + strings.TrimSuffix(strings.Repeat("?,", end-start), ",") + ") ORDER BY id"
		args := []any{scope.TenantID.String(), scope.OutcomeID.String()}
		for _, id := range refs[start:end] {
			args = append(args, id.String())
		}
		rows, e := r.rows(ctx, q, args...)
		if e != nil {
			return nil, e
		}
		for rows.Next() {
			v, e := scanProjectionWork(rows, scope)
			if e != nil {
				rows.Close()
				return nil, e
			}
			values = append(values, v)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
	}
	return values, nil
}
func (r continuityReadRepository) projectionDependencies(ctx context.Context, scope d.Scope, sourceIDs []d.ID, all bool) ([]d.Relation, error) {
	values := []d.Relation{}
	read := func(condition string, args ...any) error {
		query := `SELECT id,source_id,source_kind,target_id,target_kind,strength FROM relations WHERE tenant_id=? AND outcome_id=? AND lifecycle='active' AND relation_type='depends_on'` + condition + ` ORDER BY id`
		params := []any{scope.TenantID.String(), scope.OutcomeID.String()}
		params = append(params, args...)
		rows, e := r.rows(ctx, query, params...)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var id, source, sourceKind, target, targetKind, strength string
			if e = rows.Scan(&id, &source, &sourceKind, &target, &targetKind, &strength); e != nil {
				return e
			}
			rid, e := d.ParseID(id)
			if e != nil {
				return e
			}
			sid, e := d.ParseID(source)
			if e != nil {
				return e
			}
			tid, e := d.ParseID(target)
			if e != nil {
				return e
			}
			values = append(values, d.Relation{ID: rid, Scope: scope, SourceRef: d.EntityRef{Scope: scope, Kind: d.EntityKind(sourceKind), ID: sid}, TargetRef: d.EntityRef{Scope: scope, Kind: d.EntityKind(targetKind), ID: tid}, Strength: d.DependencyStrength(strength), RelationType: d.RelationTypeDependsOn, Lifecycle: d.RelationLifecycleActive})
		}
		return rows.Err()
	}
	if all {
		if e := read(""); e != nil {
			return nil, e
		}
		return values, nil
	}
	for start := 0; start < len(sourceIDs); start += 100 {
		end := min(start+100, len(sourceIDs))
		q := ` AND source_kind='work_item' AND source_id IN (` + strings.TrimSuffix(strings.Repeat("?,", end-start), ",") + ")"
		args := []any{}
		for _, id := range sourceIDs[start:end] {
			args = append(args, id.String())
		}
		if e := read(q, args...); e != nil {
			return nil, e
		}
	}
	return values, nil
}

func (r continuityReadRepository) projectionBlockers(ctx context.Context, scope d.Scope, details bool) ([]d.Blocker, error) {
	columns := "id,blocked_id,blocked_kind,propagation"
	if details {
		columns = blockerReadColumns
	}
	rows, e := r.rows(ctx, "SELECT "+columns+" FROM blockers WHERE tenant_id=? AND outcome_id=? AND lifecycle='active' ORDER BY id", scope.TenantID.String(), scope.OutcomeID.String())
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	values := []d.Blocker{}
	for rows.Next() {
		if details {
			v, e := scanBlocker(rows, scope)
			if e != nil {
				return nil, e
			}
			values = append(values, v)
			continue
		}
		var id, target, kind, propagation string
		if e = rows.Scan(&id, &target, &kind, &propagation); e != nil {
			return nil, e
		}
		bid, e := d.ParseID(id)
		if e != nil {
			return nil, e
		}
		tid, e := d.ParseID(target)
		if e != nil {
			return nil, e
		}
		values = append(values, d.Blocker{ID: bid, Scope: scope, BlockedRef: d.EntityRef{Scope: scope, Kind: d.EntityKind(kind), ID: tid}, Propagation: d.BlockerPropagation(propagation), Lifecycle: d.BlockerLifecycleActive})
	}
	return values, rows.Err()
}

type projectionProof struct {
	criteria d.CriterionSet
	current  *d.Conclusion
}

func (r continuityReadRepository) projectionConclusionProof(ctx context.Context, scope d.Scope) (map[d.EntityRef]*projectionProof, error) {
	result := map[d.EntityRef]*projectionProof{}
	byID := map[d.ID]d.EntityRef{}
	read := func(query string, consume func(*sql.Rows) error) error {
		rows, e := r.rows(ctx, query, scope.TenantID.String(), scope.OutcomeID.String())
		if e != nil {
			return mapSQLError("read current conclusion proof projection", e)
		}
		defer rows.Close()
		for rows.Next() {
			if e = consume(rows); e != nil {
				return e
			}
		}
		return rows.Err()
	}
	e := read(`SELECT a.owner_id,e.kind,c.obligations_snapshot_json FROM aggregate_current_conclusions a JOIN entity_refs e ON e.tenant_id=a.tenant_id AND e.outcome_id=a.outcome_id AND e.id=a.owner_id JOIN conclusions c ON c.tenant_id=a.tenant_id AND c.outcome_id=a.outcome_id AND c.id=a.conclusion_id WHERE a.tenant_id=? AND a.outcome_id=?`, func(rows *sql.Rows) error {
		var owner, kind, raw string
		if e := rows.Scan(&owner, &kind, &raw); e != nil {
			return e
		}
		id, e := d.ParseID(owner)
		if e != nil {
			return e
		}
		ref := d.EntityRef{Scope: scope, Kind: d.EntityKind(kind), ID: id}
		c := &d.Conclusion{}
		if e = unmarshalJSON(raw, &c.Obligations); e != nil {
			return e
		}
		if e = c.Obligations.Validate(); e != nil {
			return e
		}
		result[ref] = &projectionProof{criteria: d.NewCriterionSet(), current: c}
		byID[id] = ref
		return nil
	})
	if e != nil {
		return nil, e
	}
	if len(result) == 0 {
		return result, nil
	}
	e = read(`SELECT c.owner_id,c.id,c.criterion_revision,c.status,r.required,r.verification_mode FROM success_criteria c JOIN aggregate_current_conclusions a ON a.tenant_id=c.tenant_id AND a.outcome_id=c.outcome_id AND a.owner_id=c.owner_id JOIN criterion_revisions r ON r.tenant_id=c.tenant_id AND r.outcome_id=c.outcome_id AND r.criterion_id=c.id AND r.criterion_revision=c.criterion_revision WHERE c.tenant_id=? AND c.outcome_id=? ORDER BY c.created_at,c.id`, func(rows *sql.Rows) error {
		var owner, id, status, mode string
		var revision d.CriterionRevision
		var required int
		if e := rows.Scan(&owner, &id, &revision, &status, &required, &mode); e != nil {
			return e
		}
		oid, e := d.ParseID(owner)
		if e != nil {
			return e
		}
		cid, e := d.ParseID(id)
		if e != nil {
			return e
		}
		ref := byID[oid]
		result[ref].criteria.Items = append(result[ref].criteria.Items, d.SuccessCriterion{ID: cid, OwnerRef: ref, Revision: revision, Required: required == 1, Status: d.CriterionStatus(status), VerificationMode: d.VerificationMode(mode)})
		return nil
	})
	if e != nil {
		return nil, e
	}
	e = read(`SELECT c.owner_id,a.id,a.criterion_id,a.criterion_revision,a.result FROM success_criteria c JOIN aggregate_current_conclusions x ON x.tenant_id=c.tenant_id AND x.outcome_id=c.outcome_id AND x.owner_id=c.owner_id JOIN criterion_current_assessments p ON p.tenant_id=c.tenant_id AND p.outcome_id=c.outcome_id AND p.criterion_id=c.id JOIN criterion_assessments a ON a.tenant_id=p.tenant_id AND a.outcome_id=p.outcome_id AND a.id=p.assessment_id WHERE c.tenant_id=? AND c.outcome_id=?`, func(rows *sql.Rows) error {
		var owner, id, criterion, resultValue string
		var revision d.CriterionRevision
		if e := rows.Scan(&owner, &id, &criterion, &revision, &resultValue); e != nil {
			return e
		}
		oid, e := d.ParseID(owner)
		if e != nil {
			return e
		}
		aid, e := d.ParseID(id)
		if e != nil {
			return e
		}
		cid, e := d.ParseID(criterion)
		if e != nil {
			return e
		}
		result[byID[oid]].criteria.CurrentAssessments[cid] = d.CriterionAssessment{ID: aid, CriterionID: cid, CriterionRevision: revision, Result: d.AssessmentResult(resultValue)}
		return nil
	})
	if e != nil {
		return nil, e
	}
	type selected struct {
		owner d.EntityRef
		index int
	}
	assessments := map[d.ID]selected{}
	e = read(`SELECT x.owner_id,a.id,a.criterion_id,a.criterion_revision,a.result FROM aggregate_current_conclusions x JOIN conclusion_assessments p ON p.tenant_id=x.tenant_id AND p.outcome_id=x.outcome_id AND p.conclusion_id=x.conclusion_id JOIN criterion_assessments a ON a.tenant_id=p.tenant_id AND a.outcome_id=p.outcome_id AND a.id=p.assessment_id WHERE x.tenant_id=? AND x.outcome_id=? ORDER BY a.criterion_id,a.id`, func(rows *sql.Rows) error {
		var owner, id, criterion, resultValue string
		var revision d.CriterionRevision
		if e := rows.Scan(&owner, &id, &criterion, &revision, &resultValue); e != nil {
			return e
		}
		oid, e := d.ParseID(owner)
		if e != nil {
			return e
		}
		aid, e := d.ParseID(id)
		if e != nil {
			return e
		}
		cid, e := d.ParseID(criterion)
		if e != nil {
			return e
		}
		ref := byID[oid]
		p := result[ref]
		p.current.Assessments = append(p.current.Assessments, d.CriterionAssessmentRef{AssessmentID: aid, CriterionID: cid, CriterionRevision: revision, Result: d.AssessmentResult(resultValue)})
		assessments[aid] = selected{ref, len(p.criteria.Assessments)}
		p.criteria.Assessments = append(p.criteria.Assessments, d.CriterionAssessment{ID: aid, CriterionID: cid, CriterionRevision: revision, Result: d.AssessmentResult(resultValue)})
		return nil
	})
	if e != nil {
		return nil, e
	}
	e = read(`SELECT a.assessment_id,a.evidence_id FROM criterion_assessment_evidence a JOIN conclusion_assessments p ON p.tenant_id=a.tenant_id AND p.outcome_id=a.outcome_id AND p.assessment_id=a.assessment_id JOIN aggregate_current_conclusions x ON x.tenant_id=p.tenant_id AND x.outcome_id=p.outcome_id AND x.conclusion_id=p.conclusion_id WHERE a.tenant_id=? AND a.outcome_id=? ORDER BY a.assessment_id,a.evidence_id`, func(rows *sql.Rows) error {
		var aid, eid string
		if e := rows.Scan(&aid, &eid); e != nil {
			return e
		}
		id, e := d.ParseID(aid)
		if e != nil {
			return e
		}
		evidence, e := d.ParseID(eid)
		if e != nil {
			return e
		}
		where, ok := assessments[id]
		if !ok {
			return d.NewError(d.ErrorCodeInvalidArgument, "current conclusion references missing assessment projection")
		}
		set := &result[where.owner].criteria
		set.Assessments[where.index].EvidenceIDs = append(set.Assessments[where.index].EvidenceIDs, evidence)
		return nil
	})
	return result, e
}
func (r continuityReadRepository) projectionConclusionEvidence(ctx context.Context, scope d.Scope) ([]d.Evidence, error) {
	rows, e := r.rows(ctx, `SELECT DISTINCT e.id,e.lifecycle FROM evidence e JOIN criterion_assessment_evidence a ON a.tenant_id=e.tenant_id AND a.outcome_id=e.outcome_id AND a.evidence_id=e.id JOIN conclusion_assessments p ON p.tenant_id=a.tenant_id AND p.outcome_id=a.outcome_id AND p.assessment_id=a.assessment_id JOIN aggregate_current_conclusions x ON x.tenant_id=p.tenant_id AND x.outcome_id=p.outcome_id AND x.conclusion_id=p.conclusion_id WHERE e.tenant_id=? AND e.outcome_id=? ORDER BY e.id`, scope.TenantID.String(), scope.OutcomeID.String())
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	values := []d.Evidence{}
	for rows.Next() {
		var id, life string
		if e = rows.Scan(&id, &life); e != nil {
			return nil, e
		}
		parsed, e := d.ParseID(id)
		if e != nil {
			return nil, e
		}
		values = append(values, d.Evidence{ID: parsed, Scope: scope, Lifecycle: d.EvidenceLifecycle(life)})
	}
	return values, rows.Err()
}
func (r continuityReadRepository) projectionProofCounts(ctx context.Context, scope d.Scope) (int, int, int, error) {
	query := `SELECT COUNT(*),COALESCE(SUM(CASE WHEN a.result='met' AND a.criterion_revision=c.criterion_revision AND NOT EXISTS(
 SELECT 1 FROM criterion_assessment_evidence p LEFT JOIN evidence e ON e.tenant_id=p.tenant_id AND e.outcome_id=p.outcome_id AND e.id=p.evidence_id
 WHERE p.tenant_id=a.tenant_id AND p.outcome_id=a.outcome_id AND p.assessment_id=a.id AND (e.id IS NULL OR e.lifecycle!='registered')
) THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN a.result='waived' AND a.criterion_revision=c.criterion_revision THEN 1 ELSE 0 END),0)
FROM success_criteria c
JOIN criterion_revisions r ON r.tenant_id=c.tenant_id AND r.outcome_id=c.outcome_id AND r.criterion_id=c.id AND r.criterion_revision=c.criterion_revision
LEFT JOIN work_items w ON w.tenant_id=c.tenant_id AND w.outcome_id=c.outcome_id AND w.id=c.owner_id AND c.owner_kind='work_item'
LEFT JOIN objectives o ON o.tenant_id=c.tenant_id AND o.outcome_id=c.outcome_id AND o.id=c.owner_id AND c.owner_kind='objective'
LEFT JOIN criterion_current_assessments p ON p.tenant_id=c.tenant_id AND p.outcome_id=c.outcome_id AND p.criterion_id=c.id
LEFT JOIN criterion_assessments a ON a.tenant_id=p.tenant_id AND a.outcome_id=p.outcome_id AND a.id=p.assessment_id
WHERE c.tenant_id=? AND c.outcome_id=? AND c.status='active' AND r.required=1
AND (c.owner_kind='outcome' OR (c.owner_kind='work_item' AND w.lifecycle!='cancelled') OR (c.owner_kind='objective' AND o.lifecycle!='cancelled'))`
	var required, met, waived int
	e := r.row(ctx, query, scope.TenantID.String(), scope.OutcomeID.String()).Scan(&required, &met, &waived)
	return required, met, waived, mapSQLError("aggregate current required proof", e)
}
func (r continuityReadRepository) projectionPlans(ctx context.Context, scope d.Scope) ([]ports.ContinuityActivePlan, error) {
	rows, e := r.rows(ctx, `SELECT s.plan_scope_kind,s.plan_scope_id,s.roadmap_id,s.revision_number,s.activated_by_json,s.activated_at,r.content_hash FROM roadmap_active_slots s JOIN roadmaps m ON m.tenant_id=s.tenant_id AND m.outcome_id=s.outcome_id AND m.id=s.roadmap_id AND m.plan_scope_kind=s.plan_scope_kind AND m.plan_scope_id=s.plan_scope_id JOIN roadmap_revisions r ON r.tenant_id=s.tenant_id AND r.outcome_id=s.outcome_id AND r.roadmap_id=s.roadmap_id AND r.revision_number=s.revision_number WHERE s.tenant_id=? AND s.outcome_id=? ORDER BY s.roadmap_id`, scope.TenantID.String(), scope.OutcomeID.String())
	if e != nil {
		return nil, e
	}
	values := []ports.ContinuityActivePlan{}
	byPlan := map[d.ID]int{}
	for rows.Next() {
		var kind, pid, rid, actor, hash string
		var revision uint64
		var at int64
		if e = rows.Scan(&kind, &pid, &rid, &revision, &actor, &at, &hash); e != nil {
			rows.Close()
			return nil, e
		}
		planScope, e := d.ParseID(pid)
		if e != nil {
			rows.Close()
			return nil, e
		}
		roadmap, e := d.ParseID(rid)
		if e != nil {
			rows.Close()
			return nil, e
		}
		slot := d.RoadmapActiveSlot{Scope: scope, PlanScope: d.RoadmapPlanScope{Kind: d.RoadmapScopeKind(kind), ID: planScope}, RoadmapID: roadmap, RevisionNumber: revision, ActivatedAt: decodeTime(at)}
		if e = unmarshalJSON(actor, &slot.ActivatedBy); e != nil {
			rows.Close()
			return nil, e
		}
		if e = slot.Validate(); e != nil {
			rows.Close()
			return nil, e
		}
		byPlan[roadmap] = len(values)
		values = append(values, ports.ContinuityActivePlan{Slot: slot, Revision: d.RoadmapRevision{RevisionNumber: revision, ContentHash: hash, Nodes: []d.RoadmapNode{}}})
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	if len(values) == 0 {
		return values, nil
	}
	rows, e = r.rows(ctx, `SELECT n.roadmap_id,n.node_key,n.title,n.target_ref_json,n.reference_snapshot_json FROM roadmap_revision_nodes n JOIN roadmap_active_slots s ON s.tenant_id=n.tenant_id AND s.outcome_id=n.outcome_id AND s.roadmap_id=n.roadmap_id AND s.revision_number=n.revision_number WHERE n.tenant_id=? AND n.outcome_id=? ORDER BY n.roadmap_id,n.node_key`, scope.TenantID.String(), scope.OutcomeID.String())
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var rid, key, title string
		var target, snapshot sql.NullString
		if e = rows.Scan(&rid, &key, &title, &target, &snapshot); e != nil {
			return nil, e
		}
		roadmap, e := d.ParseID(rid)
		if e != nil {
			return nil, e
		}
		index, ok := byPlan[roadmap]
		if !ok {
			continue
		}
		node := d.RoadmapNode{NodeKey: key, Title: title}
		if target.Valid {
			var ref d.EntityRef
			if e = unmarshalJSON(target.String, &ref); e != nil {
				return nil, e
			}
			if e = ref.Validate(); e != nil || ref.Scope != scope {
				return nil, d.NewError(d.ErrorCodeInvalidEntityRef, "active plan target outside Outcome")
			}
			node.TargetRef = &ref
		}
		if snapshot.Valid {
			var ref d.RoadmapReferenceSnapshot
			if e = unmarshalJSON(snapshot.String, &ref); e != nil {
				return nil, e
			}
			node.ReferenceSnapshot = &ref
		}
		values[index].Revision.Nodes = append(values[index].Revision.Nodes, node)
	}
	return values, rows.Err()
}

var _ ports.ContinuityProjectionRepository = continuityReadRepository{}

func (r continuityReadRepository) BlockerDetails(ctx context.Context, scope d.Scope, ids []d.ID, byteBudget int) ([]d.Blocker, error) {
	if e := scope.Validate(); e != nil {
		return nil, e
	}
	if byteBudget < 1 || byteBudget > 1<<20 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid blocker detail byte budget")
	}
	used := 0
	result := []d.Blocker{}
	seen := map[d.ID]bool{}
	unique := []d.ID{}
	for _, id := range ids {
		if e := id.Validate(); e != nil {
			return nil, e
		}
		if !seen[id] {
			unique = append(unique, id)
			seen[id] = true
		}
	}
	for start := 0; start < len(unique); start += 100 {
		end := min(start+100, len(unique))
		q := "SELECT " + blockerReadColumns + " FROM blockers WHERE tenant_id=? AND outcome_id=? AND id IN (" + strings.TrimSuffix(strings.Repeat("?,", end-start), ",") + ") ORDER BY id"
		args := []any{scope.TenantID.String(), scope.OutcomeID.String()}
		for _, id := range unique[start:end] {
			args = append(args, id.String())
		}
		rows, e := r.rows(ctx, q, args...)
		if e != nil {
			return nil, e
		}
		for rows.Next() {
			v, e := scanBlocker(rows, scope)
			if e != nil {
				rows.Close()
				return nil, e
			}
			raw, e := json.Marshal(v)
			if e != nil {
				return nil, e
			}
			used += len(raw)
			if used > byteBudget {
				rows.Close()
				return nil, d.NewError(d.ErrorCodeGraphLimitExceeded, "projected blocker details exceed byte budget; reduce limit")
			}
			result = append(result, v)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
	}
	if len(result) != len(unique) {
		return nil, d.NewError(d.ErrorCodeNotFound, "projected blocker missing")
	}
	return result, nil
}
