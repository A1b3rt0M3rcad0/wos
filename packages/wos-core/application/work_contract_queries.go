package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"time"
)

type WorkContractView struct {
	Contract         d.WorkContract      `json:"contract"`
	EffectiveStatus  d.ContractStatus    `json:"effective_status"`
	EvaluatedAt      time.Time           `json:"evaluated_at"`
	ExecutionAllowed bool                `json:"execution_allowed"`
	Recoverable      bool                `json:"recoverable"`
	Reasons          []d.ReadinessReason `json:"reasons"`
}

func (s *Service) GetWorkContract(ctx context.Context, scope d.Scope, id d.ID) (ReadResult[WorkContractView], error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	if err := scope.Validate(); err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	if err := id.Validate(); err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	defer uow.Rollback()
	coord, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	repo, err := contractRepository(uow)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	c, err := repo.Get(ctx, scope, id)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	now, err := s.transactionTime(ctx, uow)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	w, err := uow.WorkItems().Get(ctx, scope, c.WorkItemID)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	candidate := w
	candidate.CurrentLease = nil
	candidate.Lifecycle = d.WorkItemLifecycleTodo
	ready, err := evaluateWorkItemReadiness(ctx, uow, candidate, now)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	live := w.CurrentContractID != nil && *w.CurrentContractID == c.ID
	v := WorkContractView{Contract: c, EffectiveStatus: c.EffectiveStatus(now), EvaluatedAt: now, ExecutionAllowed: live && c.ValidAt(now) && ready.Ready, Recoverable: w.Lifecycle == d.WorkItemLifecycleInProgress && !c.ValidAt(now) && ready.Ready, Reasons: ready.Reasons}
	return ReadResult[WorkContractView]{Value: v, OutcomeRevision: coord.Revision}, nil
}
func (s *Service) ListWorkContracts(ctx context.Context, scope d.Scope, f ports.ContractFilter) (ReadResult[[]d.WorkContract], error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return ReadResult[[]d.WorkContract]{}, err
	}
	if err := scope.Validate(); err != nil {
		return ReadResult[[]d.WorkContract]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[[]d.WorkContract]{}, err
	}
	defer uow.Rollback()
	coord, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[[]d.WorkContract]{}, err
	}
	repo, err := contractRepository(uow)
	if err != nil {
		return ReadResult[[]d.WorkContract]{}, err
	}
	list, err := repo.List(ctx, scope, f)
	return ReadResult[[]d.WorkContract]{Value: list, OutcomeRevision: coord.Revision}, err
}

type ContractPage struct {
	Items           []d.WorkContract  `json:"items"`
	NextCursor      string            `json:"next_cursor,omitempty"`
	OutcomeRevision d.OutcomeRevision `json:"outcome_revision"`
	Truncated       bool              `json:"truncated"`
	Omitted         map[string]int    `json:"omitted"`
}

func (s *Service) WorkContractHistory(ctx context.Context, scope d.Scope, f ports.ContractFilter, cursor string) (ContractPage, error) {
	limit, err := queryLimit(f.Limit)
	if err != nil {
		return ContractPage{}, err
	}
	f.Limit = 0
	f.After = ""
	fingerprint := filterHash(f)
	if cursor != "" {
		c, err := decodeCursor(cursor, scope.NamespaceID, scope.OutcomeID, "work_contracts", fingerprint)
		if err != nil {
			return ContractPage{}, err
		}
		id, err := d.ParseID(c.Key)
		if err != nil {
			return ContractPage{}, err
		}
		f.After = id
	}
	f.Limit = limit + 1
	result, err := s.ListWorkContracts(ctx, scope, f)
	if err != nil {
		return ContractPage{}, err
	}
	page := ContractPage{Items: result.Value, OutcomeRevision: result.OutcomeRevision, Omitted: map[string]int{}}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		page.Truncated = true
		page.NextCursor = encodeCursor(queryCursor{Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: "work_contracts", Filter: fingerprint, Key: string(page.Items[len(page.Items)-1].ID)})
	}
	// History is a small header projection: specifications expand through their immutable resource.
	for i := range page.Items {
		page.Items[i].Spec = d.WorkContractSpec{}
		page.Omitted["specs"]++
	}
	return page, nil
}
func (s *Service) GetWorkSubmission(ctx context.Context, scope d.Scope, id d.ID) (ReadResult[d.WorkSubmission], error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return ReadResult[d.WorkSubmission]{}, err
	}
	if err := scope.Validate(); err != nil {
		return ReadResult[d.WorkSubmission]{}, err
	}
	if err := id.Validate(); err != nil {
		return ReadResult[d.WorkSubmission]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[d.WorkSubmission]{}, err
	}
	defer uow.Rollback()
	coord, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[d.WorkSubmission]{}, err
	}
	repo, err := contractRepository(uow)
	if err != nil {
		return ReadResult[d.WorkSubmission]{}, err
	}
	v, err := repo.GetSubmission(ctx, scope, id)
	return ReadResult[d.WorkSubmission]{Value: v, OutcomeRevision: coord.Revision}, err
}

type ContractRecordPage struct {
	Checkpoints     []d.WorkCheckpoint `json:"checkpoints,omitempty"`
	Submissions     []d.WorkSubmission `json:"submissions,omitempty"`
	NextCursor      string             `json:"next_cursor,omitempty"`
	OutcomeRevision d.OutcomeRevision  `json:"outcome_revision"`
}

func (s *Service) ContractRecords(ctx context.Context, scope d.Scope, id d.ID, section string, limit int, cursor string) (ContractRecordPage, error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return ContractRecordPage{}, err
	}
	if err := scope.Validate(); err != nil {
		return ContractRecordPage{}, err
	}
	if err := id.Validate(); err != nil {
		return ContractRecordPage{}, err
	}
	if section != "checkpoints" && section != "submissions" {
		return ContractRecordPage{}, d.NewError(d.ErrorCodeInvalidArgument, "unknown contract section")
	}
	limit, err := queryLimit(limit)
	if err != nil {
		return ContractRecordPage{}, err
	}
	fingerprint := filterHash(id)
	after := d.ID("")
	if cursor != "" {
		c, err := decodeCursor(cursor, scope.NamespaceID, scope.OutcomeID, "contract_"+section, fingerprint)
		if err != nil {
			return ContractRecordPage{}, err
		}
		after, err = d.ParseID(c.Key)
		if err != nil {
			return ContractRecordPage{}, err
		}
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ContractRecordPage{}, err
	}
	defer uow.Rollback()
	coord, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ContractRecordPage{}, err
	}
	repo, err := contractRepository(uow)
	if err != nil {
		return ContractRecordPage{}, err
	}
	page := ContractRecordPage{OutcomeRevision: coord.Revision}
	var last d.ID
	if section == "checkpoints" {
		page.Checkpoints, err = repo.ListCheckpoints(ctx, scope, id, after, limit+1)
		if len(page.Checkpoints) > limit {
			page.Checkpoints = page.Checkpoints[:limit]
			last = page.Checkpoints[limit-1].ID
		}
	} else {
		page.Submissions, err = repo.ListSubmissions(ctx, scope, id, after, limit+1)
		if len(page.Submissions) > limit {
			page.Submissions = page.Submissions[:limit]
			last = page.Submissions[limit-1].ID
		}
	}
	if last != "" {
		page.NextCursor = encodeCursor(queryCursor{Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: "contract_" + section, Filter: fingerprint, Key: string(last)})
	}
	raw, marshalErr := json.Marshal(page)
	if marshalErr != nil {
		return page, marshalErr
	}
	if len(raw) > MaxSnapshotBytes {
		return ContractRecordPage{}, d.NewError(d.ErrorCodeGraphLimitExceeded, "record page exceeds 256 KiB; reduce limit")
	}
	return page, err
}

func (s *Service) GetCommandReceipt(ctx context.Context, ns, id d.ID) (d.StoredCommandResult, error) {
	if err := s.authorizeRead(ctx, ns); err != nil {
		return d.StoredCommandResult{}, err
	}
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		return d.StoredCommandResult{}, d.NewError(d.ErrorCodeForbidden, "receipt lookup requires authenticated principal")
	}
	if err := ns.Validate(); err != nil {
		return d.StoredCommandResult{}, err
	}
	if err := id.Validate(); err != nil {
		return d.StoredCommandResult{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return d.StoredCommandResult{}, err
	}
	defer uow.Rollback()
	if dynamic, ok := s.authorizer.(ports.TransactionalAuthorizer); ok && s.requireIdentity {
		if err := dynamic.AuthorizeInUnitOfWork(ctx, uow, ports.AuthorizationRequest{NamespaceID: ns, PrincipalID: identity.PrincipalID, Permission: ports.PermissionStateRead}); err != nil {
			return d.StoredCommandResult{}, err
		}
	}
	store, ok := uow.Idempotency().(ports.CommandReceiptStore)
	if !ok {
		return d.StoredCommandResult{}, d.NewError(d.ErrorCodeInvalidConfig, "adapter lacks receipt lookup")
	}
	return store.LookupReceipt(ctx, ns, identity.PrincipalID, id)
}

type ExecutionCapabilities struct {
	Protocols       []string           `json:"protocols"`
	YAMLVersions    []int              `json:"yaml_schema_versions"`
	MaxCommandBytes int                `json:"max_command_bytes"`
	MaxQueryLimit   int                `json:"max_query_limit"`
	LeasePolicy     d.LeasePolicy      `json:"lease_policy"`
	TerminalCauses  []d.ContractStatus `json:"terminal_causes"`
	AgentExecution  bool               `json:"agent_execution"`
}

func ContractCapabilities() ExecutionCapabilities {
	return ExecutionCapabilities{Protocols: []string{"legacy", "contracts_v1"}, YAMLVersions: []int{1}, MaxCommandBytes: MaxSnapshotBytes, MaxQueryLimit: MaxQueryLimit, LeasePolicy: d.DefaultContractLeasePolicy(), TerminalCauses: []d.ContractStatus{d.ContractExpired, d.ContractRevoked, d.ContractCompleted}, AgentExecution: false}
}
