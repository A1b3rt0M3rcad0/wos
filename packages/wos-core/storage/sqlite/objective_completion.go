package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type objectiveCompletionAuthorityRepository struct{ uow *unitOfWork }

func (r objectiveCompletionAuthorityRepository) Append(ctx context.Context, value d.ObjectiveCompletionAuthority) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	actorJSON, err := marshalJSON(value.DelegatorActor)
	if err != nil {
		return d.WrapError(d.ErrorCodeInvalidArgument, "encode Objective completion delegator actor", err)
	}
	automationActorJSON, err := marshalJSON(value.AutomationActor)
	if err != nil {
		return d.WrapError(d.ErrorCodeInvalidArgument, "encode Objective completion automation actor", err)
	}
	authorityJSON, err := marshalJSON(value)
	if err != nil {
		return d.WrapError(d.ErrorCodeInvalidArgument, "encode Objective completion authority", err)
	}
	_, err = r.uow.tx.ExecContext(ctx, `INSERT INTO objective_completion_authority_history (
 id, namespace_id, outcome_id, objective_id, policy_version, mode, completion_cycle,
 delegator_principal_id, delegator_actor_json, delegator_credential_id, credential_policy_revision,
 operational_profile, completion_policy, automation_actor_json, reason, authorized_at, authority_json
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		value.ID.String(), value.Scope.NamespaceID.String(), value.Scope.OutcomeID.String(), value.ObjectiveID.String(),
		strconv.FormatUint(value.PolicyVersion, 10), string(value.Mode.Effective()), strconv.FormatUint(value.CompletionCycle, 10),
		value.DelegatorPrincipalID, actorJSON, nullableID(value.DelegatorCredentialID), strconv.FormatUint(uint64(value.CredentialPolicyRevision), 10),
		string(value.OperationalProfile.Effective()), string(value.CompletionPolicy.Effective()), automationActorJSON,
		value.Reason, encodeTime(value.AuthorizedAt), authorityJSON,
	)
	if err != nil {
		return mapSQLError("append Objective completion authority", err)
	}
	return nil
}

func (r objectiveCompletionAuthorityRepository) ListByObjective(ctx context.Context, scope d.Scope, objectiveID d.ID) ([]d.ObjectiveCompletionAuthority, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return nil, err
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if err := objectiveID.Validate(); err != nil {
		return nil, err
	}
	rows, err := r.uow.tx.QueryContext(ctx, `SELECT authority_json, policy_version
FROM objective_completion_authority_history
WHERE namespace_id = ? AND outcome_id = ? AND objective_id = ?
ORDER BY length(policy_version), policy_version`, scope.NamespaceID.String(), scope.OutcomeID.String(), objectiveID.String())
	if err != nil {
		return nil, mapSQLError("list Objective completion authorities", err)
	}
	defer rows.Close()
	values := make([]d.ObjectiveCompletionAuthority, 0)
	for rows.Next() {
		var raw, versionText string
		if err := rows.Scan(&raw, &versionText); err != nil {
			return nil, mapSQLError("scan Objective completion authority", err)
		}
		var value d.ObjectiveCompletionAuthority
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			return nil, d.WrapError(d.ErrorCodeInvalidScope, "persisted Objective completion authority is malformed", err)
		}
		version, err := strconv.ParseUint(versionText, 10, 64)
		if err != nil || version != value.PolicyVersion {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "persisted Objective completion authority version differs from its index")
		}
		if err := value.Validate(); err != nil {
			return nil, d.WrapError(d.ErrorCodeInvalidScope, "persisted Objective completion authority is invalid", err)
		}
		if value.Scope != scope || value.ObjectiveID != objectiveID {
			return nil, d.NewError(d.ErrorCodeInvalidScope, "persisted Objective completion authority scope differs")
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, mapSQLError("iterate Objective completion authorities", err)
	}
	if err := rows.Close(); err != nil {
		return nil, mapSQLError("close Objective completion authority rows", err)
	}
	for index, value := range values {
		want := uint64(index + 1)
		if value.PolicyVersion != want {
			return nil, d.NewError(d.ErrorCodeInvalidScope, fmt.Sprintf("Objective completion authority history is not contiguous at policy version %d", want))
		}
	}
	return values, nil
}

var _ interface {
	Append(context.Context, d.ObjectiveCompletionAuthority) error
	ListByObjective(context.Context, d.Scope, d.ID) ([]d.ObjectiveCompletionAuthority, error)
} = objectiveCompletionAuthorityRepository{}
