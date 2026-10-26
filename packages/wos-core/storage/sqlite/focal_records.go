package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strings"
)

func qualifyReadColumns(alias, columns string) string {
	parts := strings.Split(columns, ",")
	for i, p := range parts {
		parts[i] = alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ",")
}
func (r continuityReadRepository) FocalRecords(ctx context.Context, scope d.Scope, workID d.ID, objectiveID *d.ID) (ports.FocalContextRecords, error) {
	result := ports.FocalContextRecords{Dependencies: []d.Relation{}, DependencyEvaluations: []d.DependencyEvaluation{}, Blockers: []d.Blocker{}, ObjectiveAncestors: []d.Objective{}, Decisions: []d.Decision{}, EvidenceLinks: []d.EvidenceLink{}}
	if e := scope.Validate(); e != nil {
		return result, e
	}
	if e := workID.Validate(); e != nil {
		return result, e
	}
	if objectiveID != nil {
		if e := objectiveID.Validate(); e != nil {
			return result, e
		}
	}
	args := []any{scope.NamespaceID.String(), scope.OutcomeID.String(), workID.String()}
	if e := r.row(ctx, `SELECT COUNT(*) FROM relations WHERE namespace_id=? AND outcome_id=? AND source_kind='work_item' AND source_id=? AND lifecycle='active'`, args...).Scan(&result.DependencyCount); e != nil {
		return result, e
	}
	rows, e := r.rows(ctx, "SELECT "+relationReadColumns+" FROM relations WHERE namespace_id=? AND outcome_id=? AND source_kind='work_item' AND source_id=? AND lifecycle='active' ORDER BY id LIMIT 101", args...)
	if e != nil {
		return result, e
	}
	for rows.Next() {
		v, e := scanRelation(rows, scope)
		if e != nil {
			rows.Close()
			return result, e
		}
		result.Dependencies = append(result.Dependencies, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return result, e
	}
	joins := ` FROM relations r LEFT JOIN work_items w ON w.namespace_id=r.namespace_id AND w.outcome_id=r.outcome_id AND w.id=r.target_id AND r.target_kind='work_item' LEFT JOIN objectives o ON o.namespace_id=r.namespace_id AND o.outcome_id=r.outcome_id AND o.id=r.target_id AND r.target_kind='objective' WHERE r.namespace_id=? AND r.outcome_id=? AND r.source_kind='work_item' AND r.source_id=? AND r.lifecycle='active' AND r.relation_type='depends_on'`
	var missing int
	if e = r.row(ctx, "SELECT COUNT(*)"+joins+` AND ((r.target_kind='work_item' AND w.id IS NULL) OR (r.target_kind='objective' AND o.id IS NULL) OR r.target_kind NOT IN('work_item','objective'))`, args...).Scan(&missing); e != nil {
		return result, e
	}
	if missing > 0 {
		return result, d.NewError(d.ErrorCodeNotFound, "dependency target is missing from focal snapshot")
	}
	witness, e := scanRelation(r.row(ctx, "SELECT "+qualifyReadColumns("r", relationReadColumns)+joins+` AND r.strength='hard' AND ((r.target_kind='work_item' AND w.lifecycle!='done') OR (r.target_kind='objective' AND o.lifecycle!='achieved')) ORDER BY r.id LIMIT 1`, args...), scope)
	if e != nil {
		if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeNotFound && e != sql.ErrNoRows {
			return result, e
		}
	} else {
		result.DependencyEvaluations = append(result.DependencyEvaluations, d.DependencyEvaluation{Relation: witness, Satisfied: false})
	}
	// The witness can be outside the displayed 100 relations: operational
	// readiness never truncates the hard-dependency predicate to the UI page.
	if objectiveID != nil {
		result.ObjectiveAncestors, e = r.focalAncestors(ctx, scope, *objectiveID)
		if e != nil {
			return result, e
		}
	}
	result.Blockers, e = r.focalBlockers(ctx, scope, workID, objectiveID)
	if e != nil {
		return result, e
	}
	if e = r.row(ctx, `SELECT COUNT(*) FROM decisions WHERE namespace_id=? AND outcome_id=? AND lifecycle='accepted'`, scope.NamespaceID.String(), scope.OutcomeID.String()).Scan(&result.DecisionCount); e != nil {
		return result, e
	}
	rows, e = r.rows(ctx, "SELECT "+decisionReadColumns+" FROM decisions WHERE namespace_id=? AND outcome_id=? AND lifecycle='accepted' ORDER BY id LIMIT 26", scope.NamespaceID.String(), scope.OutcomeID.String())
	if e != nil {
		return result, e
	}
	for rows.Next() {
		v, e := scanDecision(rows, scope)
		if e != nil {
			rows.Close()
			return result, e
		}
		result.Decisions = append(result.Decisions, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return result, e
	}
	predicate := ` AND ((target_kind='outcome' AND target_id=?) OR (target_kind='work_item' AND target_id=?)`
	linkArgs := []any{scope.NamespaceID.String(), scope.OutcomeID.String(), scope.OutcomeID.String(), workID.String()}
	if objectiveID != nil {
		predicate += ` OR (target_kind='objective' AND target_id=?)`
		linkArgs = append(linkArgs, objectiveID.String())
	}
	predicate += ")"
	if e = r.row(ctx, `SELECT COUNT(*) FROM evidence_links WHERE namespace_id=? AND outcome_id=? AND lifecycle='active'`+predicate, linkArgs...).Scan(&result.EvidenceLinkCount); e != nil {
		return result, e
	}
	rows, e = r.rows(ctx, "SELECT "+evidenceLinkReadColumns+" FROM evidence_links WHERE namespace_id=? AND outcome_id=? AND lifecycle='active'"+predicate+" ORDER BY id LIMIT 26", linkArgs...)
	if e != nil {
		return result, e
	}
	for rows.Next() {
		v, e := scanEvidenceLink(rows, scope)
		if e != nil {
			rows.Close()
			return result, e
		}
		result.EvidenceLinks = append(result.EvidenceLinks, v)
	}
	e = rows.Err()
	rows.Close()
	return result, e
}

const focalAncestorCTE = `WITH RECURSIVE ancestors(id,parent_objective_id,completion_mode,completion_cycle) AS (
 SELECT id,parent_objective_id,completion_mode,completion_cycle FROM objectives WHERE namespace_id=? AND outcome_id=? AND id=?
 UNION
 SELECT o.id,o.parent_objective_id,o.completion_mode,o.completion_cycle FROM objectives o JOIN ancestors a ON o.id=a.parent_objective_id WHERE o.namespace_id=? AND o.outcome_id=?
) `

func (r continuityReadRepository) focalAncestors(ctx context.Context, scope d.Scope, id d.ID) ([]d.Objective, error) {
	rows, e := r.rows(ctx, focalAncestorCTE+`SELECT id,parent_objective_id,completion_mode,completion_cycle FROM ancestors ORDER BY id`, scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(), scope.NamespaceID.String(), scope.OutcomeID.String())
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	result := []d.Objective{}
	for rows.Next() {
		var rawID string
		var mode, cycle string
		var parent sql.NullString
		if e = rows.Scan(&rawID, &parent, &mode, &cycle); e != nil {
			return nil, e
		}
		parsed, e := d.ParseID(rawID)
		if e != nil {
			return nil, e
		}
		completionCycle, e := parseObjectiveCompletionCycle(cycle)
		if e != nil {
			return nil, e
		}
		v := d.Objective{ID: parsed, Scope: scope, CompletionMode: d.ObjectiveCompletionMode(mode), CompletionCycle: completionCycle}
		if parent.Valid {
			pid, e := d.ParseID(parent.String)
			if e != nil {
				return nil, e
			}
			v.ParentObjectiveID = &pid
		}
		result = append(result, v)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	if len(result) == 0 {
		return nil, d.NewError(d.ErrorCodeNotFound, "focal Objective ancestry missing")
	}
	return r.validateObjectiveCompletionList(ctx, result)
}
func (r continuityReadRepository) focalBlockers(ctx context.Context, scope d.Scope, workID d.ID, objectiveID *d.ID) ([]d.Blocker, error) {
	prefix := ""
	args := []any{}
	if objectiveID != nil {
		prefix = focalAncestorCTE
		args = append(args, scope.NamespaceID.String(), scope.OutcomeID.String(), objectiveID.String(), scope.NamespaceID.String(), scope.OutcomeID.String())
	}
	query := prefix + "SELECT " + blockerReadColumns + ` FROM blockers WHERE namespace_id=? AND outcome_id=? AND lifecycle='active' AND ((blocked_kind='work_item' AND blocked_id=?) OR (propagation='subtree' AND ((blocked_kind='outcome' AND blocked_id=?)`
	args = append(args, scope.NamespaceID.String(), scope.OutcomeID.String(), workID.String(), scope.OutcomeID.String())
	if objectiveID != nil {
		query += ` OR (blocked_kind='objective' AND blocked_id IN(SELECT id FROM ancestors))`
	}
	query += "))) ORDER BY id"
	rows, e := r.rows(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	result := []d.Blocker{}
	bytes := 0
	for rows.Next() {
		v, e := scanBlocker(rows, scope)
		if e != nil {
			return nil, e
		}
		raw, e := json.Marshal(v)
		if e != nil {
			return nil, e
		}
		bytes += len(raw)
		if bytes > 256<<10 {
			return nil, d.NewError(d.ErrorCodeGraphLimitExceeded, "work context blocking proof exceeds byte limit")
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

var _ ports.FocalReadRepository = continuityReadRepository{}
