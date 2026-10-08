package sqlite

import (
	"context"
	"database/sql"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type aggregateOwnedState struct {
	actors   []domain.ActorRef
	criteria domain.CriterionSet
	current  *domain.Conclusion
	history  []domain.Conclusion
}

// All nested aggregate state is read in nine set queries, independent of the
// number of owners, criteria, assessments and conclusions in the Outcome.
func loadOwnedStates(ctx context.Context, tx *sql.Tx, scope domain.Scope) (map[domain.ID]*aggregateOwnedState, error) {
	states := map[domain.ID]*aggregateOwnedState{}
	get := func(id domain.ID) *aggregateOwnedState {
		if states[id] == nil {
			states[id] = &aggregateOwnedState{actors: []domain.ActorRef{}, criteria: domain.NewCriterionSet(), history: []domain.Conclusion{}}
		}
		return states[id]
	}
	ns, outcome := scope.NamespaceID.String(), scope.OutcomeID.String()
	read := func(query string, consume func(*sql.Rows) error) error {
		rows, err := tx.QueryContext(ctx, query, ns, outcome)
		if err != nil {
			return mapSQLError("load aggregate owned collections", err)
		}
		defer rows.Close()
		for rows.Next() {
			if err := consume(rows); err != nil {
				return err
			}
		}
		return rows.Err()
	}
	if err := read(`SELECT l.entity_id,l.actor_json FROM entity_actor_links l JOIN entity_refs e ON e.namespace_id=l.namespace_id AND e.outcome_id=l.outcome_id AND e.id=l.entity_id WHERE l.namespace_id=? AND l.outcome_id=? AND ((e.kind='work_item' AND l.role='assignee') OR (e.kind IN ('outcome','objective') AND l.role='owner')) ORDER BY l.entity_id,l.actor_kind,l.actor_provider,l.actor_id`, func(rows *sql.Rows) error {
		var id, encoded string
		if err := rows.Scan(&id, &encoded); err != nil {
			return err
		}
		parsed, err := domain.ParseID(id)
		if err != nil {
			return err
		}
		var actor domain.ActorRef
		if err := unmarshalJSON(encoded, &actor); err != nil {
			return err
		}
		if err := actor.Validate(); err != nil {
			return err
		}
		get(parsed).actors = append(get(parsed).actors, actor)
		return nil
	}); err != nil {
		return nil, err
	}
	owners := map[domain.ID]domain.EntityRef{}
	if err := read(`SELECT c.owner_id,c.owner_kind,c.id,c.criterion_revision,c.status,r.definition_json,r.required,r.verification_mode FROM success_criteria c JOIN criterion_revisions r ON r.namespace_id=c.namespace_id AND r.outcome_id=c.outcome_id AND r.criterion_id=c.id AND r.criterion_revision=c.criterion_revision WHERE c.namespace_id=? AND c.outcome_id=? ORDER BY c.created_at,c.id`, func(rows *sql.Rows) error {
		var oid, kind, cid, status, encoded, mode string
		var revision int64
		var required int
		if err := rows.Scan(&oid, &kind, &cid, &revision, &status, &encoded, &required, &mode); err != nil {
			return err
		}
		ownerID, err := domain.ParseID(oid)
		if err != nil {
			return err
		}
		id, err := domain.ParseID(cid)
		if err != nil {
			return err
		}
		owner := domain.EntityRef{Scope: scope, Kind: domain.EntityKind(kind), ID: ownerID}
		owners[id] = owner
		var def criterionDefinition
		if err := unmarshalJSON(encoded, &def); err != nil {
			return err
		}
		state := get(ownerID)
		state.criteria.Items = append(state.criteria.Items, domain.SuccessCriterion{ID: id, OwnerRef: owner, Title: def.Title, Description: def.Description, Required: required == 1, Revision: domain.CriterionRevision(revision), VerificationMode: domain.VerificationMode(mode), Status: domain.CriterionStatus(status)})
		return nil
	}); err != nil {
		return nil, err
	}
	if err := read(`SELECT r.criterion_id,r.criterion_revision,r.definition_json,r.required,r.verification_mode,r.status FROM criterion_revisions r WHERE r.namespace_id=? AND r.outcome_id=? ORDER BY r.criterion_id,r.criterion_revision`, func(rows *sql.Rows) error {
		var cid, encoded, mode, status string
		var revision int64
		var required int
		if err := rows.Scan(&cid, &revision, &encoded, &required, &mode, &status); err != nil {
			return err
		}
		id, err := domain.ParseID(cid)
		if err != nil {
			return err
		}
		owner, ok := owners[id]
		if !ok {
			return domain.NewError(domain.ErrorCodeAssessment, "criterion history owner missing")
		}
		var def criterionDefinition
		if err := unmarshalJSON(encoded, &def); err != nil {
			return err
		}
		v := domain.CriterionDefinitionRevision{CriterionID: id, OwnerRef: owner, Revision: domain.CriterionRevision(revision), Title: def.Title, Description: def.Description, Required: required == 1, VerificationMode: domain.VerificationMode(mode), Status: domain.CriterionStatus(status)}
		if err := v.Validate(); err != nil {
			return err
		}
		state := get(owner.ID)
		state.criteria.DefinitionRevisions = append(state.criteria.DefinitionRevisions, v)
		return nil
	}); err != nil {
		return nil, err
	}
	evidence := map[domain.ID][]domain.ID{}
	if err := read(`SELECT assessment_id,evidence_id FROM criterion_assessment_evidence WHERE namespace_id=? AND outcome_id=? ORDER BY assessment_id,evidence_id`, func(rows *sql.Rows) error {
		var aid, eid string
		if err := rows.Scan(&aid, &eid); err != nil {
			return err
		}
		a, err := domain.ParseID(aid)
		if err != nil {
			return err
		}
		e, err := domain.ParseID(eid)
		if err != nil {
			return err
		}
		evidence[a] = append(evidence[a], e)
		return nil
	}); err != nil {
		return nil, err
	}
	assessments := map[domain.ID]domain.CriterionAssessment{}
	if err := read(`SELECT id,criterion_id,criterion_revision,result,rationale,principal_id,actor_json,assessed_at,evaluator_ref_json,supersedes_assessment_id,submission_id,submission_digest FROM criterion_assessments WHERE namespace_id=? AND outcome_id=? ORDER BY assessed_at,id`, func(rows *sql.Rows) error {
		v, err := scanAssessment(rows)
		if err != nil {
			return err
		}
		v.EvidenceIDs = evidence[v.ID]
		if v.EvidenceIDs == nil {
			v.EvidenceIDs = []domain.ID{}
		}
		if err := v.Validate(); err != nil {
			return err
		}
		owner, ok := owners[v.CriterionID]
		if !ok {
			return domain.NewError(domain.ErrorCodeAssessment, "assessment owner missing")
		}
		state := get(owner.ID)
		state.criteria.Assessments = append(state.criteria.Assessments, v)
		assessments[v.ID] = v
		return nil
	}); err != nil {
		return nil, err
	}
	if err := read(`SELECT criterion_id,assessment_id FROM criterion_current_assessments WHERE namespace_id=? AND outcome_id=? ORDER BY criterion_id`, func(rows *sql.Rows) error {
		var cid, aid string
		if err := rows.Scan(&cid, &aid); err != nil {
			return err
		}
		c, err := domain.ParseID(cid)
		if err != nil {
			return err
		}
		a, err := domain.ParseID(aid)
		if err != nil {
			return err
		}
		v, ok := assessments[a]
		owner, exists := owners[c]
		if !ok || !exists || v.CriterionID != c {
			return domain.NewError(domain.ErrorCodeAssessment, "current assessment history missing")
		}
		get(owner.ID).criteria.CurrentAssessments[c] = v
		return nil
	}); err != nil {
		return nil, err
	}
	refs := map[string][]domain.CriterionAssessmentRef{}
	if err := read(`SELECT ca.conclusion_id,a.id,a.criterion_id,a.criterion_revision,a.result FROM conclusion_assessments ca JOIN criterion_assessments a ON a.namespace_id=ca.namespace_id AND a.outcome_id=ca.outcome_id AND a.id=ca.assessment_id WHERE ca.namespace_id=? AND ca.outcome_id=? ORDER BY a.criterion_id,a.id`, func(rows *sql.Rows) error {
		var conclusion, aid, cid, result string
		var revision int64
		if err := rows.Scan(&conclusion, &aid, &cid, &revision, &result); err != nil {
			return err
		}
		a, err := domain.ParseID(aid)
		if err != nil {
			return err
		}
		c, err := domain.ParseID(cid)
		if err != nil {
			return err
		}
		refs[conclusion] = append(refs[conclusion], domain.CriterionAssessmentRef{AssessmentID: a, CriterionID: c, CriterionRevision: domain.CriterionRevision(revision), Result: domain.AssessmentResult(result)})
		return nil
	}); err != nil {
		return nil, err
	}
	current := map[domain.ID]string{}
	if err := read(`SELECT owner_id,conclusion_id FROM aggregate_current_conclusions WHERE namespace_id=? AND outcome_id=?`, func(rows *sql.Rows) error {
		var oid, cid string
		if err := rows.Scan(&oid, &cid); err != nil {
			return err
		}
		o, err := domain.ParseID(oid)
		if err != nil {
			return err
		}
		current[o] = cid
		return nil
	}); err != nil {
		return nil, err
	}
	if err := read(`SELECT e.id,e.kind,c.id,c.public_id,c.owner_version,c.lifecycle_result,c.principal_id,c.actor_json,c.recorded_at,c.rationale,c.obligations_snapshot_json,c.submission_id,c.submission_digest FROM conclusions c JOIN entity_refs e ON e.namespace_id=c.namespace_id AND e.outcome_id=c.outcome_id AND e.id=c.owner_id WHERE c.namespace_id=? AND c.outcome_id=? ORDER BY c.owner_id,c.ordinal`, func(rows *sql.Rows) error {
		var oid, kind string
		// Owner identity is needed by the shared codec, so buffer the fixed row first.
		var sid, lifecycle, principal, actor, rationale, obligations string
		var publicID, submissionID sql.NullString
		var submissionDigest string
		var version sql.NullInt64
		var at int64
		if err := rows.Scan(&oid, &kind, &sid, &publicID, &version, &lifecycle, &principal, &actor, &at, &rationale, &obligations, &submissionID, &submissionDigest); err != nil {
			return err
		}
		o, err := domain.ParseID(oid)
		if err != nil {
			return err
		}
		owner := domain.EntityRef{Scope: scope, Kind: domain.EntityKind(kind), ID: o}
		storageID, v, err := scanConclusion(conclusionRow{sid, publicID, version, lifecycle, principal, actor, at, rationale, obligations, submissionID, submissionDigest}, owner)
		if err != nil {
			return err
		}
		v.Assessments = refs[storageID]
		if v.Assessments == nil {
			v.Assessments = []domain.CriterionAssessmentRef{}
		}
		if err := v.Validate(); err != nil {
			return err
		}
		state := get(o)
		if current[o] == storageID {
			state.current = &v
		} else {
			state.history = append(state.history, v)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return states, nil
}

// The same conclusion codec handles legacy storage IDs and public identities.
type conclusionRow struct {
	id                          string
	public                      sql.NullString
	version                     sql.NullInt64
	lifecycle, principal, actor string
	at                          int64
	rationale, obligations      string
	submissionID                sql.NullString
	submissionDigest            string
}

func (r conclusionRow) Scan(dest ...any) error {
	*dest[0].(*string) = r.id
	*dest[1].(*sql.NullString) = r.public
	*dest[2].(*sql.NullInt64) = r.version
	*dest[3].(*string) = r.lifecycle
	*dest[4].(*string) = r.principal
	*dest[5].(*string) = r.actor
	*dest[6].(*int64) = r.at
	*dest[7].(*string) = r.rationale
	*dest[8].(*string) = r.obligations
	*dest[9].(*sql.NullString) = r.submissionID
	*dest[10].(*string) = r.submissionDigest
	return nil
}
