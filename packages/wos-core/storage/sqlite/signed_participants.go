package sqlite

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func (r signedContractRepository) ExecutionParticipants(ctx context.Context, scope d.Scope, work d.ID, limit int) ([]d.ExecutionParticipant, error) {
	if scope.Validate() != nil || work.Validate() != nil || limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid bounded independence query")
	}
	rows, err := r.u.tx.QueryContext(ctx, `SELECT DISTINCT principal_id,separation_group FROM (
 SELECT holder_principal_id AS principal_id,COALESCE(json_extract(state_json,'$.signed_binding.separation_group'),'') AS separation_group FROM signed_work_contracts WHERE namespace_id=? AND outcome_id=? AND work_item_id=?
 UNION ALL SELECT holder_principal_id,'' FROM work_contracts WHERE namespace_id=? AND outcome_id=? AND work_item_id=?
 UNION ALL SELECT principal_id,'' FROM domain_events WHERE namespace_id=? AND outcome_id=? AND aggregate_id=? AND event_type IN ('work_item.claimed','work_item.reclaimed','work_item.admin_completed')
 ) participants ORDER BY principal_id,separation_group LIMIT ?`, scope.NamespaceID.String(), scope.OutcomeID.String(), work.String(), scope.NamespaceID.String(), scope.OutcomeID.String(), work.String(), scope.NamespaceID.String(), scope.OutcomeID.String(), work.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []d.ExecutionParticipant{}
	for rows.Next() {
		var p d.ExecutionParticipant
		if err = rows.Scan(&p.PrincipalID, &p.SeparationGroup); err != nil {
			return nil, mapSQLError("scan signed state", err)
		}
		out = append(out, p)
	}
	return out, mapSQLError("iterate signed state", rows.Err())
}
