package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strings"
)

func contractRepository(uow ports.UnitOfWork) (ports.WorkContractRepository, error) {
	contracts, ok := uow.(ports.WorkContractUnitOfWork)
	if !ok {
		return nil, d.NewError(d.ErrorCodeInvalidConfig, "adapter lacks work contracts")
	}
	return contracts.WorkContracts(), nil
}
func (s *Service) AcquireWorkContract(ctx context.Context, cc d.CommandContext, cmd AcquireWorkContractCommand) (MutationResult[WorkContractResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[WorkContractResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (WorkContractResult, d.OutcomeRevision, error) {
		return s.acquireContract(ctx, uow, cc, cmd)
	})
}
func (s *Service) acquireContract(ctx context.Context, uow ports.UnitOfWork, cc d.CommandContext, cmd AcquireWorkContractCommand) (WorkContractResult, d.OutcomeRevision, error) {
	zero := WorkContractResult{}
	if err := requireActiveOutcome(ctx, uow, cmd.Scope); err != nil {
		return zero, 0, err
	}
	repo, err := contractRepository(uow)
	if err != nil {
		return zero, 0, err
	}
	w, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
	if err != nil {
		return zero, 0, err
	}
	if !w.ContractsEnabled {
		return zero, 0, d.NewError(d.ErrorCodeContractProtocolRequired, "Namespace has not activated contract protocol")
	}
	if cmd.ExpectedWorkItemVersion != w.Version {
		return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "work item version conflict")
	}
	now, err := s.transactionTime(ctx, uow)
	if err != nil {
		return zero, 0, err
	}
	var previous *d.WorkContract
	if w.CurrentContractID != nil {
		old, err := repo.Get(ctx, cmd.Scope, *w.CurrentContractID)
		if err != nil {
			return zero, 0, err
		}
		if old.ValidAt(now) {
			return zero, 0, d.NewError(d.ErrorCodeWorkAlreadyClaimed, "work item has valid contract")
		}
		if old.Status == d.ContractActive {
			v, l := old.Version, old.LeaseVersion
			if err := old.Expire(now); err != nil {
				return zero, 0, err
			}
			if err := repo.Save(ctx, old, v, l); err != nil {
				return zero, 0, err
			}
		}
		previous = &old
		w.CurrentContractID = nil
	}
	// Recoverability is a projection; reuse all readiness gates using a todo view.
	candidate := w
	candidate.Lifecycle = d.WorkItemLifecycleTodo
	candidate.CurrentLease = nil
	if w.Lifecycle != d.WorkItemLifecycleTodo && w.Lifecycle != d.WorkItemLifecycleInProgress {
		return zero, 0, d.NewError(d.ErrorCodePreconditionFailed, "work lifecycle is not acquirable")
	}
	if err := requireWorkItemReady(ctx, uow, candidate, now); err != nil {
		return zero, 0, err
	}
	relations, err := uow.Relations().ListByOutcome(ctx, cmd.Scope)
	if err != nil {
		return zero, 0, err
	}
	spec := d.WorkContractSpec{Title: w.Title, Description: w.Description, ObjectiveID: cloneIDPtr(w.ObjectiveID), Criteria: append([]d.SuccessCriterion{}, w.Criteria.Items...)}
	if w.ExecutionSpec != nil {
		spec.ExecutionSpec = *w.ExecutionSpec
	}
	for _, r := range relations {
		if r.SourceRef == w.Ref() && r.RelationType == d.RelationTypeDependsOn && r.Lifecycle == d.RelationLifecycleActive {
			spec.Dependencies = append(spec.Dependencies, r.TargetRef)
		}
	}
	id, err := s.ids.NewID()
	if err != nil {
		return zero, 0, err
	}
	execution, err := s.ids.NewID()
	if err != nil {
		return zero, 0, err
	}
	coord, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
	if err != nil {
		return zero, 0, err
	}
	c, err := d.NewWorkContract(id, execution, w, cc.PrincipalID, cc.Actor, d.DefaultContractLeasePolicy(), cmd.TTLSeconds, spec, coord.Revision, now)
	if err != nil {
		return zero, 0, err
	}
	if err := w.BindContract(c, now); err != nil {
		return zero, 0, err
	}
	if err := repo.Insert(ctx, c); err != nil {
		return zero, 0, err
	}
	if err := uow.WorkItems().Save(ctx, w, cmd.ExpectedWorkItemVersion); err != nil {
		return zero, 0, err
	}
	rev, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
	return WorkContractResult{EvaluatedAt: now, Contract: c, WorkItem: w, PreviousContract: previous, Recovery: w.Lifecycle == d.WorkItemLifecycleInProgress && previous != nil}, rev, err
}
func validateContractSpec(c d.WorkContract, a ContractAuthority) error {
	if c.SpecDigest != a.SpecDigest {
		return d.NewError(d.ErrorCodeContractSpecMismatch, "contract spec does not match")
	}
	return nil
}
func (s *Service) RenewWorkContract(ctx context.Context, cc d.CommandContext, cmd RenewWorkContractCommand) (MutationResult[WorkContractResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[WorkContractResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (WorkContractResult, d.OutcomeRevision, error) {
		zero := WorkContractResult{}
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return zero, 0, err
		}
		repo, err := contractRepository(uow)
		if err != nil {
			return zero, 0, err
		}
		c, err := repo.Get(ctx, cmd.Scope, cmd.ContractID)
		if err != nil {
			return zero, 0, err
		}
		if err := validateContractSpec(c, cmd.Authority); err != nil {
			return zero, 0, err
		}
		now, err := s.transactionTime(ctx, uow)
		if err != nil {
			return zero, 0, err
		}
		v, l := c.Version, c.LeaseVersion
		if err := c.Renew(cc.PrincipalID, cmd.Authority.ExecutionID, cmd.Authority.FencingToken, cmd.ExpectedLeaseVersion, cmd.TTLSeconds, now); err != nil {
			return zero, 0, err
		}
		if err := repo.Save(ctx, c, v, l); err != nil {
			return zero, 0, err
		}
		w, err := uow.WorkItems().Get(ctx, cmd.Scope, c.WorkItemID)
		if err != nil {
			return zero, 0, err
		}
		rev, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return WorkContractResult{EvaluatedAt: now, Contract: c, WorkItem: w}, rev, err
	})
}
func (s *Service) ResumeWorkContract(ctx context.Context, cc d.CommandContext, cmd ResumeWorkContractCommand) (MutationResult[WorkContractResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[WorkContractResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (WorkContractResult, d.OutcomeRevision, error) {
		zero := WorkContractResult{}
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return zero, 0, err
		}
		repo, err := contractRepository(uow)
		if err != nil {
			return zero, 0, err
		}
		c, err := repo.Get(ctx, cmd.Scope, cmd.ContractID)
		if err != nil {
			return zero, 0, err
		}
		if err := validateContractSpec(c, cmd.Authority); err != nil {
			return zero, 0, err
		}
		w, err := uow.WorkItems().Get(ctx, cmd.Scope, c.WorkItemID)
		if err != nil {
			return zero, 0, err
		}
		now, err := s.transactionTime(ctx, uow)
		if err != nil {
			return zero, 0, err
		}
		execution, err := s.ids.NewID()
		if err != nil {
			return zero, 0, err
		}
		v, l := c.Version, c.LeaseVersion
		if err := c.Resume(cc.PrincipalID, cmd.Authority.ExecutionID, cmd.Authority.FencingToken, cmd.ExpectedLeaseVersion, execution, w.LastFencingToken, now); err != nil {
			return zero, 0, err
		}
		if w.CurrentContractID == nil || *w.CurrentContractID != c.ID {
			return zero, 0, d.NewError(d.ErrorCodeStaleExecution, "work item contract changed")
		}
		expected := w.Version
		next, err := w.Version.Next()
		if err != nil {
			return zero, 0, err
		}
		w.Version = next
		w.LastFencingToken = uint64(c.FencingToken)
		w.UpdatedAt = now
		if err := repo.Save(ctx, c, v, l); err != nil {
			return zero, 0, err
		}
		if err := uow.WorkItems().Save(ctx, w, expected); err != nil {
			return zero, 0, err
		}
		rev, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return WorkContractResult{EvaluatedAt: now, Contract: c, WorkItem: w}, rev, err
	})
}
func (s *Service) RevokeWorkContract(ctx context.Context, cc d.CommandContext, cmd RevokeWorkContractCommand) (MutationResult[WorkContractResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[WorkContractResult]{}, err
	}
	if strings.TrimSpace(cmd.Reason) == "" {
		return MutationResult[WorkContractResult]{}, d.NewError(d.ErrorCodeInvalidArgument, "revocation reason is required")
	}
	if err := s.authorizer.Authorize(ctx, ports.AuthorizationRequest{NamespaceID: cmd.Scope.NamespaceID, PrincipalID: cc.PrincipalID, Permission: ports.PermissionWorkContractRevoke}); err != nil {
		return MutationResult[WorkContractResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (WorkContractResult, d.OutcomeRevision, error) {
		zero := WorkContractResult{}
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return zero, 0, err
		}
		repo, err := contractRepository(uow)
		if err != nil {
			return zero, 0, err
		}
		c, err := repo.Get(ctx, cmd.Scope, cmd.ContractID)
		if err != nil {
			return zero, 0, err
		}
		now, err := s.transactionTime(ctx, uow)
		if err != nil {
			return zero, 0, err
		}
		v, l := c.Version, c.LeaseVersion
		if err := c.Revoke(cc.PrincipalID, cmd.Reason, cmd.ExpectedContractVersion, now); err != nil {
			return zero, 0, err
		}
		w, err := uow.WorkItems().Get(ctx, cmd.Scope, c.WorkItemID)
		if err != nil {
			return zero, 0, err
		}
		expected := w.Version
		if err := w.DetachContract(c, now); err != nil {
			return zero, 0, err
		}
		if err := repo.Save(ctx, c, v, l); err != nil {
			return zero, 0, err
		}
		if err := uow.WorkItems().Save(ctx, w, expected); err != nil {
			return zero, 0, err
		}
		rev, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return WorkContractResult{EvaluatedAt: now, Contract: c, WorkItem: w}, rev, err
	})
}
func contractCommandEvents[T any](s *Service, cc d.CommandContext, meta commandMetadata, value T, rev d.OutcomeRevision) ([]d.DomainEvent, bool, error) {
	if next, ok := any(value).(WorkContractAcquisition); ok {
		if !next.Acquired {
			return nil, true, nil
		}
		meta.Name = "AcquireWorkContract"
		return contractCommandEvents(s, cc, meta, *next.Result, rev)
	}
	result, ok := any(value).(WorkContractResult)
	if !ok {
		return nil, false, nil
	}
	facts := map[string]string{"AcquireWorkContract": "work_contract.acquired", "RenewWorkContract": "work_contract.renewed", "ResumeWorkContract": "work_contract.execution_resumed", "RevokeWorkContract": "work_contract.revoked", "SyncWorkContract": "work_contract.checkpoint_recorded", "SubmitWorkResult": "work_contract.result_submitted", "FinalizeWorkContract": "work_contract.completed"}
	fact, ok := facts[meta.Name]
	if !ok {
		return nil, false, nil
	}
	specs := []compoundEventSpec{}
	if result.PreviousContract != nil {
		old := result.PreviousContract
		specs = append(specs, compoundEventSpec{eventType: "work_contract.expired", ref: old.Ref(), after: versionPtr(result.WorkItem.Version)})
	}
	specs = append(specs, compoundEventSpec{eventType: fact, ref: result.Contract.Ref(), after: versionPtr(result.WorkItem.Version)})
	for _, a := range result.Artifacts {
		specs = append(specs, compoundEventSpec{eventType: "artifact.registered", ref: a.Ref(), after: versionPtr(a.Version)})
	}
	for _, e := range result.Evidence {
		specs = append(specs, compoundEventSpec{eventType: "evidence.registered", ref: e.Ref(), after: versionPtr(e.Version)})
	}
	for _, l := range result.EvidenceLinks {
		specs = append(specs, compoundEventSpec{eventType: "evidence_link.created", ref: l.Ref(), after: versionPtr(l.Version)})
	}
	if meta.Name == "FinalizeWorkContract" {
		specs = append(specs, compoundEventSpec{eventType: "work_item.completed", ref: result.WorkItem.Ref(), after: versionPtr(result.WorkItem.Version)}, compoundEventSpec{eventType: "work_item.conclusion_recorded", ref: result.WorkItem.Ref(), after: versionPtr(result.WorkItem.Version)})
	}
	events, err := buildCompoundEvents(s, cc, meta, rev, specs)
	// Bind emitted facts to server-assigned identities, not only the input command.
	for i := range events {
		payload, err := json.Marshal(map[string]any{"contract_id": result.Contract.ID, "work_item_id": result.WorkItem.ID, "status": result.Contract.Status, "contract_version": result.Contract.Version, "lease_version": result.Contract.LeaseVersion, "fencing_token": result.Contract.FencingToken, "spec_digest": result.Contract.SpecDigest, "expires_at": result.Contract.ExpiresAt, "previous_contract": result.PreviousContract, "checkpoint": result.Checkpoint, "submission_id": result.Contract.LatestSubmissionID})
		if err != nil {
			return nil, true, err
		}
		events[i].Payload = payload
		events[i].RecordedAt = result.EvaluatedAt
		if meta.Name == "RevokeWorkContract" {
			events[i].RecordedAt = *result.Contract.ClosedAt
		}
		if err := events[i].Validate(); err != nil {
			return nil, true, err
		}
	}
	return events, true, err
}
