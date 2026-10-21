package memory

import (
	"context"
	"sort"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func objectiveCompletionAuthorityKey(scope d.Scope, objectiveID d.ID) string {
	return scopeKey(scope) + "/" + objectiveID.String()
}

func cloneObjectiveCompletionAuthority(value d.ObjectiveCompletionAuthority) d.ObjectiveCompletionAuthority {
	if value.DelegatorCredentialID != nil {
		id := *value.DelegatorCredentialID
		value.DelegatorCredentialID = &id
	}
	return value
}

func cloneObjectiveCompletionAuthorities(source map[string][]d.ObjectiveCompletionAuthority) map[string][]d.ObjectiveCompletionAuthority {
	result := make(map[string][]d.ObjectiveCompletionAuthority, len(source))
	for key, values := range source {
		cloned := make([]d.ObjectiveCompletionAuthority, len(values))
		for index, value := range values {
			cloned[index] = cloneObjectiveCompletionAuthority(value)
		}
		result[key] = cloned
	}
	return result
}

type objectiveCompletionAuthorityRepository struct{ tx *transaction }

func (r objectiveCompletionAuthorityRepository) Append(ctx context.Context, value d.ObjectiveCompletionAuthority) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	objective, ok := r.tx.objectives[entityKey(value.Scope, value.ObjectiveID)]
	if !ok || objective.ID != value.ObjectiveID || objective.Scope != value.Scope {
		return d.NewError(d.ErrorCodeNotFound, "Objective completion authority owner not found")
	}
	key := objectiveCompletionAuthorityKey(value.Scope, value.ObjectiveID)
	history := r.tx.objectiveCompletionAuthorities[key]
	wantVersion := uint64(1)
	if len(history) > 0 {
		latest := history[len(history)-1]
		if latest.Scope != value.Scope || latest.ObjectiveID != value.ObjectiveID {
			return d.NewError(d.ErrorCodeInvalidScope, "Objective completion authority history scope differs")
		}
		var err error
		wantVersion, err = d.NextObjectiveCompletionPolicyVersion(latest.PolicyVersion)
		if err != nil {
			return err
		}
	}
	if value.PolicyVersion != wantVersion {
		return d.NewError(d.ErrorCodeVersionConflict, "Objective completion policy version is not the next history version")
	}
	r.tx.objectiveCompletionAuthorities[key] = append(history, cloneObjectiveCompletionAuthority(value))
	return nil
}

func (r objectiveCompletionAuthorityRepository) ListByObjective(ctx context.Context, scope d.Scope, objectiveID d.ID) ([]d.ObjectiveCompletionAuthority, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if err := objectiveID.Validate(); err != nil {
		return nil, err
	}
	values := r.tx.objectiveCompletionAuthorities[objectiveCompletionAuthorityKey(scope, objectiveID)]
	result := make([]d.ObjectiveCompletionAuthority, len(values))
	for index, value := range values {
		result[index] = cloneObjectiveCompletionAuthority(value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].PolicyVersion < result[j].PolicyVersion })
	return result, nil
}
