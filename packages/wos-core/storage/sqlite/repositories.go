package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type outcomeRepository struct{ uow *unitOfWork }
type objectiveRepository struct{ uow *unitOfWork }
type workItemRepository struct{ uow *unitOfWork }
type coordinationStore struct{ uow *unitOfWork }

func (r outcomeRepository) Get(ctx context.Context, namespaceID, outcomeID domain.ID) (domain.Outcome, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return domain.Outcome{}, err
	}
	var (
		version                          int64
		title, description, desiredState string
		externalContext                  string
		lifecycle, priority              string
		archivedAt                       sql.NullInt64
		createdAt, updatedAt             int64
	)
	err := r.uow.tx.QueryRowContext(ctx, `
SELECT version, title, description, desired_state, lifecycle, priority,
       archived_at, created_at, updated_at,external_context_json
FROM outcomes
WHERE namespace_id = ? AND id = ? AND outcome_id = ?`,
		namespaceID.String(), outcomeID.String(), outcomeID.String(),
	).Scan(
		&version, &title, &description, &desiredState, &lifecycle, &priority,
		&archivedAt, &createdAt, &updatedAt, &externalContext,
	)
	if err != nil {
		return domain.Outcome{}, mapSQLError("get outcome", err)
	}

	value := domain.Outcome{
		ID:           outcomeID,
		NamespaceID:  namespaceID,
		Version:      domain.Version(version),
		Title:        title,
		Description:  description,
		DesiredState: desiredState,
		Lifecycle:    domain.OutcomeLifecycle(lifecycle),
		Priority:     domain.Priority(priority),
		CreatedAt:    decodeTime(createdAt),
		UpdatedAt:    decodeTime(updatedAt),
		Criteria:     domain.NewCriterionSet(),
	}
	if err := unmarshalJSON(externalContext, &value.ExternalContext); err != nil {
		return domain.Outcome{}, err
	}
	if archivedAt.Valid {
		t := decodeTime(archivedAt.Int64)
		value.ArchivedAt = &t
	}

	actors, criteria, current, history, err := loadOwnedState(ctx, r.uow.tx, value.Ref(), "owner")
	if err != nil {
		return domain.Outcome{}, err
	}
	value.OwnerRefs = actors
	value.Criteria = criteria
	value.CurrentConclusion = current
	value.ConclusionHistory = history
	if err := value.Validate(); err != nil {
		return domain.Outcome{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "persisted outcome is invalid", err)
	}
	return value, nil
}

func (r outcomeRepository) Insert(ctx context.Context, outcome domain.Outcome) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := outcome.Validate(); err != nil {
		return err
	}
	if err := ensureNamespace(ctx, r.uow.tx, outcome.NamespaceID); err != nil {
		return err
	}
	if err := ensureCoordinationRow(ctx, r.uow.tx, outcome.Scope()); err != nil {
		return err
	}
	if err := insertEntityRef(ctx, r.uow.tx, outcome.Ref()); err != nil {
		return err
	}
	contextJSON, err := marshalJSON(outcome.ExternalContext)
	if err != nil {
		return err
	}
	_, err = r.uow.tx.ExecContext(ctx, `
INSERT INTO outcomes (
    id, namespace_id, outcome_id, kind, version, created_at, updated_at,
    title, description, priority, desired_state, lifecycle, archived_at,external_context_json
) VALUES (?, ?, ?, 'outcome', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		outcome.ID.String(),
		outcome.NamespaceID.String(),
		outcome.ID.String(),
		int64(outcome.Version),
		encodeTime(outcome.CreatedAt),
		encodeTime(outcome.UpdatedAt),
		outcome.Title,
		outcome.Description,
		string(outcome.Priority),
		outcome.DesiredState,
		string(outcome.Lifecycle),
		encodeOptionalTime(outcome.ArchivedAt),
		contextJSON,
	)
	if err != nil {
		return mapSQLError("insert outcome", err)
	}
	if err := syncExternalContextIndex(ctx, r.uow.tx, outcome); err != nil {
		return err
	}
	return syncOwnedState(
		ctx, r.uow.tx, outcome.Ref(), "owner", outcome.OwnerRefs, outcome.Criteria,
		outcome.CurrentConclusion, outcome.ConclusionHistory, outcome.Version,
		string(outcome.Lifecycle), outcome.UpdatedAt,
	)
}

func (r outcomeRepository) Save(ctx context.Context, outcome domain.Outcome, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := outcome.Validate(); err != nil {
		return err
	}
	if outcome.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "outcome version must advance exactly once per save")
	}
	contextJSON, err := marshalJSON(outcome.ExternalContext)
	if err != nil {
		return err
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE outcomes
SET version = ?, updated_at = ?, title = ?, description = ?, priority = ?,
    desired_state = ?, lifecycle = ?, archived_at = ?,external_context_json=?
WHERE namespace_id = ? AND id = ? AND outcome_id = ? AND version = ?`,
		int64(outcome.Version),
		encodeTime(outcome.UpdatedAt),
		outcome.Title,
		outcome.Description,
		string(outcome.Priority),
		outcome.DesiredState,
		string(outcome.Lifecycle),
		encodeOptionalTime(outcome.ArchivedAt),
		contextJSON,
		outcome.NamespaceID.String(),
		outcome.ID.String(),
		outcome.ID.String(),
		int64(expected),
	)
	if err != nil {
		return mapSQLError("save outcome", err)
	}
	if err := requireVersionedUpdate(ctx, r.uow.tx, result, "outcomes",
		outcome.NamespaceID, outcome.ID, outcome.ID, expected); err != nil {
		return err
	}
	if err := syncExternalContextIndex(ctx, r.uow.tx, outcome); err != nil {
		return err
	}
	return syncOwnedState(
		ctx, r.uow.tx, outcome.Ref(), "owner", outcome.OwnerRefs, outcome.Criteria,
		outcome.CurrentConclusion, outcome.ConclusionHistory, outcome.Version,
		string(outcome.Lifecycle), outcome.UpdatedAt,
	)
}

func (r objectiveRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Objective, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return domain.Objective{}, err
	}
	row := r.uow.tx.QueryRowContext(ctx, `
SELECT id, version, title, description, parent_objective_id, lifecycle, priority,
       required_for_outcome, created_at, updated_at
FROM objectives
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`, scope.NamespaceID.String(), scope.OutcomeID.String(), id.String())
	value, err := scanObjective(row, scope)
	if err != nil {
		return domain.Objective{}, err
	}
	actors, criteria, current, history, err := loadOwnedState(ctx, r.uow.tx, value.Ref(), "owner")
	if err != nil {
		return domain.Objective{}, err
	}
	value.OwnerRefs = actors
	value.Criteria = criteria
	value.CurrentConclusion = current
	value.ConclusionHistory = history
	if err := value.Validate(); err != nil {
		return domain.Objective{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "persisted objective is invalid", err)
	}
	return value, nil
}

func scanObjective(scanner rowScanner, scope domain.Scope) (domain.Objective, error) {
	var rawID string
	var (
		version              int64
		title, description   string
		parent               sql.NullString
		lifecycle, priority  string
		required             int
		createdAt, updatedAt int64
	)
	err := scanner.Scan(
		&rawID,
		&version, &title, &description, &parent, &lifecycle, &priority,
		&required, &createdAt, &updatedAt,
	)
	if err != nil {
		return domain.Objective{}, mapSQLError("get objective", err)
	}
	id, err := domain.ParseID(rawID)
	if err != nil {
		return domain.Objective{}, err
	}
	value := domain.Objective{
		ID:                 id,
		Scope:              scope,
		Version:            domain.Version(version),
		Title:              title,
		Description:        description,
		Lifecycle:          domain.ObjectiveLifecycle(lifecycle),
		Priority:           domain.Priority(priority),
		RequiredForOutcome: required == 1,
		CreatedAt:          decodeTime(createdAt),
		UpdatedAt:          decodeTime(updatedAt),
		Criteria:           domain.NewCriterionSet(),
		OwnerRefs:          []domain.ActorRef{}, ConclusionHistory: []domain.Conclusion{},
	}
	if parent.Valid {
		parsed, err := domain.ParseID(parent.String)
		if err != nil {
			return domain.Objective{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "persisted parent objective id is invalid", err)
		}
		value.ParentObjectiveID = &parsed
	}
	return value, nil
}

func (r objectiveRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Objective, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return nil, err
	}
	rows, err := r.uow.tx.QueryContext(ctx, `
SELECT id, version, title, description, parent_objective_id, lifecycle, priority,
       required_for_outcome, created_at, updated_at
FROM objectives
WHERE namespace_id = ? AND outcome_id = ?
ORDER BY id`, scope.NamespaceID.String(), scope.OutcomeID.String())
	if err != nil {
		return nil, mapSQLError("list objective aggregates", err)
	}
	defer rows.Close()
	values := []domain.Objective{}
	for rows.Next() {
		value, err := scanObjective(rows, scope)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return values, nil
	}
	owned, err := ownedAggregateIDs(ctx, r.uow.tx, scope)
	if err != nil {
		return nil, err
	}
	var states map[domain.ID]*aggregateOwnedState
	for _, value := range values {
		if owned[value.ID] {
			states, err = loadOwnedStates(ctx, r.uow.tx, scope)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	for i := range values {
		if state := states[values[i].ID]; state != nil {
			values[i].OwnerRefs = state.actors
			values[i].Criteria = state.criteria
			values[i].CurrentConclusion = state.current
			values[i].ConclusionHistory = state.history
		}
		if err := values[i].Validate(); err != nil {
			return nil, domain.WrapError(domain.ErrorCodeInvalidArgument, "persisted objective is invalid", err)
		}
	}
	return values, nil
}

func (r objectiveRepository) Insert(ctx context.Context, objective domain.Objective) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := objective.Validate(); err != nil {
		return err
	}
	if err := insertEntityRef(ctx, r.uow.tx, objective.Ref()); err != nil {
		return err
	}
	_, err := r.uow.tx.ExecContext(ctx, `
INSERT INTO objectives (
    id, namespace_id, outcome_id, kind, version, created_at, updated_at,
    title, description, priority, parent_objective_id, lifecycle, required_for_outcome
) VALUES (?, ?, ?, 'objective', ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		objective.ID.String(),
		objective.Scope.NamespaceID.String(),
		objective.Scope.OutcomeID.String(),
		int64(objective.Version),
		encodeTime(objective.CreatedAt),
		encodeTime(objective.UpdatedAt),
		objective.Title,
		objective.Description,
		string(objective.Priority),
		nullableID(objective.ParentObjectiveID),
		string(objective.Lifecycle),
		boolInt(objective.RequiredForOutcome),
	)
	if err != nil {
		return mapSQLError("insert objective", err)
	}
	return syncOwnedState(
		ctx, r.uow.tx, objective.Ref(), "owner", objective.OwnerRefs, objective.Criteria,
		objective.CurrentConclusion, objective.ConclusionHistory, objective.Version,
		string(objective.Lifecycle), objective.UpdatedAt,
	)
}

func (r objectiveRepository) Save(ctx context.Context, objective domain.Objective, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := objective.Validate(); err != nil {
		return err
	}
	if objective.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "objective version must advance exactly once per save")
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE objectives
SET version = ?, updated_at = ?, title = ?, description = ?, priority = ?,
    parent_objective_id = ?, lifecycle = ?, required_for_outcome = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(objective.Version),
		encodeTime(objective.UpdatedAt),
		objective.Title,
		objective.Description,
		string(objective.Priority),
		nullableID(objective.ParentObjectiveID),
		string(objective.Lifecycle),
		boolInt(objective.RequiredForOutcome),
		objective.Scope.NamespaceID.String(),
		objective.Scope.OutcomeID.String(),
		objective.ID.String(),
		int64(expected),
	)
	if err != nil {
		return mapSQLError("save objective", err)
	}
	if err := requireVersionedUpdate(ctx, r.uow.tx, result, "objectives",
		objective.Scope.NamespaceID, objective.Scope.OutcomeID, objective.ID, expected); err != nil {
		return err
	}
	return syncOwnedState(
		ctx, r.uow.tx, objective.Ref(), "owner", objective.OwnerRefs, objective.Criteria,
		objective.CurrentConclusion, objective.ConclusionHistory, objective.Version,
		string(objective.Lifecycle), objective.UpdatedAt,
	)
}

func (r workItemRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.WorkItem, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return domain.WorkItem{}, err
	}
	row := r.uow.tx.QueryRowContext(ctx, `
SELECT id, version, title, description, objective_id, lifecycle, priority,
       not_before, result_summary, COALESCE(fencing_token_decimal, CAST(last_fencing_token AS TEXT)), lease_claim_id, lease_principal_id,
       lease_actor_json, lease_acquired_at, lease_expires_at, created_at, updated_at, contracts_enabled, current_contract_id, execution_spec_json
FROM work_items
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`, scope.NamespaceID.String(), scope.OutcomeID.String(), id.String())
	value, err := scanWorkItem(row, scope)
	if err != nil {
		return domain.WorkItem{}, err
	}
	actors, criteria, current, history, err := loadOwnedState(ctx, r.uow.tx, value.Ref(), "assignee")
	if err != nil {
		return domain.WorkItem{}, err
	}
	value.AssigneeRefs = actors
	value.Criteria = criteria
	value.CurrentConclusion = current
	value.ConclusionHistory = history
	if err := value.Validate(); err != nil {
		return domain.WorkItem{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "persisted work item is invalid", err)
	}
	return value, nil
}

func scanWorkItem(scanner rowScanner, scope domain.Scope) (domain.WorkItem, error) {
	var rawID string
	var (
		version                             int64
		title, description                  string
		objectiveID                         sql.NullString
		lifecycle, priority, resultSummary  string
		lastFencing                         string
		contractsEnabled                    int
		contractID, executionSpec           sql.NullString
		claimID, leasePrincipal, leaseActor sql.NullString
		notBefore, acquiredAt, expiresAt    sql.NullInt64
		createdAt, updatedAt                int64
	)
	err := scanner.Scan(
		&rawID,
		&version, &title, &description, &objectiveID, &lifecycle, &priority,
		&notBefore, &resultSummary, &lastFencing, &claimID, &leasePrincipal, &leaseActor,
		&acquiredAt, &expiresAt, &createdAt, &updatedAt, &contractsEnabled, &contractID, &executionSpec,
	)
	if err != nil {
		return domain.WorkItem{}, mapSQLError("get work item", err)
	}
	id, err := domain.ParseID(rawID)
	if err != nil {
		return domain.WorkItem{}, err
	}
	value := domain.WorkItem{
		ID:               id,
		Scope:            scope,
		Version:          domain.Version(version),
		Title:            title,
		Description:      description,
		Lifecycle:        domain.WorkItemLifecycle(lifecycle),
		Priority:         domain.Priority(priority),
		ResultSummary:    resultSummary,
		LastFencingToken: 0,
		ContractsEnabled: contractsEnabled == 1,
		CreatedAt:        decodeTime(createdAt),
		UpdatedAt:        decodeTime(updatedAt),
		Criteria:         domain.NewCriterionSet(),
		AssigneeRefs:     []domain.ActorRef{}, ConclusionHistory: []domain.Conclusion{},
	}
	token, err := strconv.ParseUint(lastFencing, 10, 64)
	if err != nil || lastFencing != strconv.FormatUint(token, 10) {
		return domain.WorkItem{}, domain.NewError(domain.ErrorCodeLease, "persisted unsigned fencing is invalid")
	}
	value.LastFencingToken = token
	if contractID.Valid {
		parsed, err := domain.ParseID(contractID.String)
		if err != nil {
			return domain.WorkItem{}, err
		}
		value.CurrentContractID = &parsed
	}
	if executionSpec.Valid {
		var spec domain.ExecutionSpec
		if err := unmarshalJSON(executionSpec.String, &spec); err != nil {
			return domain.WorkItem{}, err
		}
		value.ExecutionSpec = &spec
	}
	if notBefore.Valid {
		t := decodeTime(notBefore.Int64)
		value.NotBefore = &t
	}
	if objectiveID.Valid {
		parsed, err := domain.ParseID(objectiveID.String)
		if err != nil {
			return domain.WorkItem{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "persisted work objective id is invalid", err)
		}
		value.ObjectiveID = &parsed
	}
	if claimID.Valid {
		parsed, err := domain.ParseID(claimID.String)
		if err != nil {
			return domain.WorkItem{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "persisted claim id is invalid", err)
		}
		var actor domain.ActorRef
		if !leaseActor.Valid || !acquiredAt.Valid || !expiresAt.Valid || !leasePrincipal.Valid {
			return domain.WorkItem{}, domain.NewError(domain.ErrorCodeInvalidArgument, "persisted work lease is incomplete")
		}
		if err := unmarshalJSON(leaseActor.String, &actor); err != nil {
			return domain.WorkItem{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "persisted lease actor is invalid", err)
		}
		value.CurrentLease = &domain.WorkLease{
			ClaimID:      parsed,
			PrincipalID:  leasePrincipal.String,
			Actor:        actor,
			FencingToken: token,
			AcquiredAt:   decodeTime(acquiredAt.Int64),
			ExpiresAt:    decodeTime(expiresAt.Int64),
		}
	}
	return value, nil
}

func (r workItemRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.WorkItem, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return nil, err
	}
	rows, err := r.uow.tx.QueryContext(ctx, `
SELECT id, version, title, description, objective_id, lifecycle, priority,
       not_before, result_summary, COALESCE(fencing_token_decimal, CAST(last_fencing_token AS TEXT)), lease_claim_id, lease_principal_id,
       lease_actor_json, lease_acquired_at, lease_expires_at, created_at, updated_at, contracts_enabled, current_contract_id, execution_spec_json
FROM work_items
WHERE namespace_id = ? AND outcome_id = ?
ORDER BY id`, scope.NamespaceID.String(), scope.OutcomeID.String())
	if err != nil {
		return nil, mapSQLError("list work item aggregates", err)
	}
	defer rows.Close()
	values := []domain.WorkItem{}
	for rows.Next() {
		value, err := scanWorkItem(rows, scope)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return values, nil
	}
	owned, err := ownedAggregateIDs(ctx, r.uow.tx, scope)
	if err != nil {
		return nil, err
	}
	var states map[domain.ID]*aggregateOwnedState
	for _, value := range values {
		if owned[value.ID] {
			states, err = loadOwnedStates(ctx, r.uow.tx, scope)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	for i := range values {
		if state := states[values[i].ID]; state != nil {
			values[i].AssigneeRefs = state.actors
			values[i].Criteria = state.criteria
			values[i].CurrentConclusion = state.current
			values[i].ConclusionHistory = state.history
		}
		if err := values[i].Validate(); err != nil {
			return nil, domain.WrapError(domain.ErrorCodeInvalidArgument, "persisted work item is invalid", err)
		}
	}
	return values, nil
}

func (r workItemRepository) Insert(ctx context.Context, item domain.WorkItem) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := item.Validate(); err != nil {
		return err
	}
	if err := insertEntityRef(ctx, r.uow.tx, item.Ref()); err != nil {
		return err
	}
	lease, err := encodeLease(item.CurrentLease)
	if err != nil {
		return err
	}
	if item.CurrentLease != nil {
		if err := ensurePrincipal(ctx, r.uow.tx, item.CurrentLease.PrincipalID); err != nil {
			return err
		}
	}
	_, err = r.uow.tx.ExecContext(ctx, `
INSERT INTO work_items (
    id, namespace_id, outcome_id, kind, version, created_at, updated_at,
    title, description, priority, objective_id, lifecycle, not_before, result_summary,
    last_fencing_token, lease_claim_id, lease_principal_id, lease_actor_json,
    lease_acquired_at, lease_expires_at, contracts_enabled, current_contract_id, execution_spec_json, fencing_token_decimal
) VALUES (?, ?, ?, 'work_item', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID.String(),
		item.Scope.NamespaceID.String(),
		item.Scope.OutcomeID.String(),
		int64(item.Version),
		encodeTime(item.CreatedAt),
		encodeTime(item.UpdatedAt),
		item.Title,
		item.Description,
		string(item.Priority),
		nullableID(item.ObjectiveID),
		string(item.Lifecycle),
		encodeOptionalTime(item.NotBefore),
		item.ResultSummary,
		legacyFencing(item.LastFencingToken),
		lease.claimID,
		lease.principalID,
		lease.actorJSON,
		lease.acquiredAt,
		lease.expiresAt,
		boolInt(item.ContractsEnabled), nullableID(item.CurrentContractID), executionSpecJSON(item.ExecutionSpec), strconv.FormatUint(item.LastFencingToken, 10),
	)
	if err != nil {
		return mapSQLError("insert work item", err)
	}
	return syncOwnedState(
		ctx, r.uow.tx, item.Ref(), "assignee", item.AssigneeRefs, item.Criteria,
		item.CurrentConclusion, item.ConclusionHistory, item.Version,
		string(item.Lifecycle), item.UpdatedAt,
	)
}

func (r workItemRepository) Save(ctx context.Context, item domain.WorkItem, expected domain.Version) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := item.Validate(); err != nil {
		return err
	}
	if item.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "work item version must advance exactly once per save")
	}
	lease, err := encodeLease(item.CurrentLease)
	if err != nil {
		return err
	}
	if item.CurrentLease != nil {
		if err := ensurePrincipal(ctx, r.uow.tx, item.CurrentLease.PrincipalID); err != nil {
			return err
		}
	}
	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE work_items
SET version = ?, updated_at = ?, title = ?, description = ?, priority = ?,
    objective_id = ?, lifecycle = ?, not_before = ?, result_summary = ?, last_fencing_token = ?,
    lease_claim_id = ?, lease_principal_id = ?, lease_actor_json = ?,
    lease_acquired_at = ?, lease_expires_at = ?, contracts_enabled = ?, current_contract_id = ?, execution_spec_json = ?, fencing_token_decimal = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(item.Version),
		encodeTime(item.UpdatedAt),
		item.Title,
		item.Description,
		string(item.Priority),
		nullableID(item.ObjectiveID),
		string(item.Lifecycle),
		encodeOptionalTime(item.NotBefore),
		item.ResultSummary,
		legacyFencing(item.LastFencingToken),
		lease.claimID,
		lease.principalID,
		lease.actorJSON,
		lease.acquiredAt,
		lease.expiresAt,
		boolInt(item.ContractsEnabled), nullableID(item.CurrentContractID), executionSpecJSON(item.ExecutionSpec), strconv.FormatUint(item.LastFencingToken, 10),
		item.Scope.NamespaceID.String(),
		item.Scope.OutcomeID.String(),
		item.ID.String(),
		int64(expected),
	)
	if err != nil {
		return mapSQLError("save work item", err)
	}
	if err := requireVersionedUpdate(ctx, r.uow.tx, result, "work_items",
		item.Scope.NamespaceID, item.Scope.OutcomeID, item.ID, expected); err != nil {
		return err
	}
	return syncOwnedState(
		ctx, r.uow.tx, item.Ref(), "assignee", item.AssigneeRefs, item.Criteria,
		item.CurrentConclusion, item.ConclusionHistory, item.Version,
		string(item.Lifecycle), item.UpdatedAt,
	)
}

func (s coordinationStore) LockOutcome(ctx context.Context, scope domain.Scope) (ports.OutcomeCoordination, error) {
	started := time.Now()
	defer func() { s.uow.guardDuration += time.Since(started) }()
	if err := s.uow.ensureOpen(); err != nil {
		return ports.OutcomeCoordination{}, err
	}
	if err := scope.Validate(); err != nil {
		return ports.OutcomeCoordination{}, err
	}
	if err := ensureNamespace(ctx, s.uow.tx, scope.NamespaceID); err != nil {
		return ports.OutcomeCoordination{}, err
	}
	if err := ensureCoordinationRow(ctx, s.uow.tx, scope); err != nil {
		return ports.OutcomeCoordination{}, err
	}
	var revision int64
	if err := s.uow.tx.QueryRowContext(ctx, `
SELECT state_revision
FROM outcome_coordination
WHERE namespace_id = ? AND outcome_id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(),
	).Scan(&revision); err != nil {
		return ports.OutcomeCoordination{}, mapSQLError("lock outcome coordination", err)
	}
	return ports.OutcomeCoordination{
		Scope:    scope,
		Revision: domain.OutcomeRevision(revision),
	}, nil
}

func (s coordinationStore) AdvanceOutcome(ctx context.Context, scope domain.Scope) (domain.OutcomeRevision, error) {
	if err := s.uow.ensureOpen(); err != nil {
		return 0, err
	}
	result, err := s.uow.tx.ExecContext(ctx, `
UPDATE outcome_coordination
SET state_revision = state_revision + 1
WHERE namespace_id = ? AND outcome_id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(),
	)
	if err != nil {
		return 0, mapSQLError("advance outcome revision", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if affected != 1 {
		return 0, domain.NewError(domain.ErrorCodeNotFound, "outcome coordination row not found")
	}
	var revision int64
	if err := s.uow.tx.QueryRowContext(ctx, `
SELECT state_revision
FROM outcome_coordination
WHERE namespace_id = ? AND outcome_id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(),
	).Scan(&revision); err != nil {
		return 0, mapSQLError("read advanced outcome revision", err)
	}
	return domain.OutcomeRevision(revision), nil
}

func ensureNamespace(ctx context.Context, tx *sql.Tx, namespaceID domain.ID) error {
	if err := namespaceID.Validate(); err != nil {
		return err
	}
	now := encodeTime(nowUTC())
	_, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO namespaces (
    id, name, lifecycle, version, created_at, updated_at
) VALUES (?, ?, 'active', 1, ?, ?)`,
		namespaceID.String(),
		"namespace:"+namespaceID.String(),
		now,
		now,
	)
	return mapSQLError("ensure namespace", err)
}

func ensurePrincipal(ctx context.Context, tx *sql.Tx, principalID string) error {
	if principalID == "" {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "principal_id is required")
	}
	_, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO principals (
    id, display_name, kind, lifecycle, created_at
) VALUES (?, ?, 'local', 'active', ?)`,
		principalID,
		principalID,
		encodeTime(nowUTC()),
	)
	return mapSQLError("ensure principal", err)
}

func ensureCoordinationRow(ctx context.Context, tx *sql.Tx, scope domain.Scope) error {
	_, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO outcome_coordination (
    namespace_id, outcome_id, state_revision
) VALUES (?, ?, 0)`,
		scope.NamespaceID.String(), scope.OutcomeID.String(),
	)
	return mapSQLError("ensure outcome coordination", err)
}

func insertEntityRef(ctx context.Context, tx *sql.Tx, ref domain.EntityRef) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO entity_refs (id, namespace_id, outcome_id, kind)
VALUES (?, ?, ?, ?)`,
		ref.ID.String(),
		ref.NamespaceID.String(),
		ref.OutcomeID.String(),
		ref.Kind.String(),
	)
	return mapSQLError("insert entity ref", err)
}

func requireVersionedUpdate(
	ctx context.Context,
	tx *sql.Tx,
	result sql.Result,
	table string,
	namespaceID, outcomeID, entityID domain.ID,
	expected domain.Version,
) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 1 {
		return nil
	}
	if affected > 1 {
		return fmt.Errorf("%s versioned update affected %d rows", table, affected)
	}
	var current int64
	query := fmt.Sprintf(
		"SELECT version FROM %s WHERE namespace_id = ? AND outcome_id = ? AND id = ?",
		table,
	)
	err = tx.QueryRowContext(ctx, query,
		namespaceID.String(), outcomeID.String(), entityID.String(),
	).Scan(&current)
	if err == sql.ErrNoRows {
		return domain.NewError(domain.ErrorCodeNotFound, "aggregate not found")
	}
	if err != nil {
		return mapSQLError("inspect version conflict", err)
	}
	return domain.NewError(
		domain.ErrorCodeVersionConflict,
		fmt.Sprintf("expected version %d but current version is %d", expected, current),
	)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

type encodedLease struct {
	claimID     any
	principalID any
	actorJSON   any
	acquiredAt  any
	expiresAt   any
}

func encodeLease(lease *domain.WorkLease) (encodedLease, error) {
	if lease == nil {
		return encodedLease{}, nil
	}
	if err := lease.Validate(); err != nil {
		return encodedLease{}, err
	}
	actorJSON, err := marshalJSON(lease.Actor)
	if err != nil {
		return encodedLease{}, err
	}
	return encodedLease{
		claimID:     lease.ClaimID.String(),
		principalID: lease.PrincipalID,
		actorJSON:   actorJSON,
		acquiredAt:  encodeTime(lease.AcquiredAt),
		expiresAt:   encodeTime(lease.ExpiresAt),
	}, nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

// A bounded set read avoids seven empty owned-state queries per aggregate.
// Owners with criteria, assignments or conclusions keep the full history codec.
func ownedAggregateIDs(ctx context.Context, tx *sql.Tx, scope domain.Scope) (map[domain.ID]bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT entity_id FROM entity_actor_links WHERE namespace_id=? AND outcome_id=?
 UNION SELECT owner_id FROM success_criteria WHERE namespace_id=? AND outcome_id=?
 UNION SELECT owner_id FROM conclusions WHERE namespace_id=? AND outcome_id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), scope.NamespaceID.String(), scope.OutcomeID.String(), scope.NamespaceID.String(), scope.OutcomeID.String())
	if err != nil {
		return nil, mapSQLError("list aggregate owned state", err)
	}
	defer rows.Close()
	owners := map[domain.ID]bool{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		id, err := domain.ParseID(raw)
		if err != nil {
			return nil, err
		}
		owners[id] = true
	}
	return owners, rows.Err()
}

// Legacy signed column remains a compatibility hint; the exact decimal column is authoritative for new writers.
func legacyFencing(v uint64) int64 {
	if v > uint64(1<<63-1) {
		return 0
	}
	return int64(v)
}
func executionSpecJSON(spec *domain.ExecutionSpec) any {
	if spec == nil {
		return nil
	}
	encoded, _ := marshalJSON(spec)
	return encoded
}
