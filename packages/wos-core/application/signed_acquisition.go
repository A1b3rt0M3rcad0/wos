package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"time"
)

type AcquireSignedWorkContractCommand struct {
	PreviousReviewCaseID    *d.ID `wos:"optional"`
	Scope                   d.Scope
	WorkItemID              d.ID
	ExpectedWorkItemVersion d.Version
	SignerKeyID             d.ID
	TTLSeconds              int `wos:"optional"`
}

func (s *Service) AcquireSignedWorkContract(ctx context.Context, cc d.CommandContext, cmd AcquireSignedWorkContractCommand) (MutationResult[WorkContractResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[WorkContractResult]{}, err
	}
	if cc.IdempotencyKey == "" {
		return MutationResult[WorkContractResult]{}, d.NewError(d.ErrorCodeInvalidArgument, "signed acquisition requires independent idempotency intent")
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (WorkContractResult, d.OutcomeRevision, error) {
		return s.acquireSignedWork(ctx, uow, cc, cmd)
	})
}
func (s *Service) acquireSignedWork(ctx context.Context, uow ports.UnitOfWork, cc d.CommandContext, cmd AcquireSignedWorkContractCommand) (WorkContractResult, d.OutcomeRevision, error) {
	var zero WorkContractResult
	if cmd.SignerKeyID.Validate() != nil {
		return zero, 0, d.NewError(d.ErrorCodeInvalidArgument, "agent signing key required")
	}
	identity, credential, err := s.signedAccess(ctx, uow, cmd.Scope, ports.PermissionWorkContractAcquire, cmd.SignerKeyID)
	if err != nil {
		return zero, 0, err
	}
	server, err := s.requireSignedIssuer(ctx, uow, cmd.Scope.NamespaceID)
	if err != nil {
		return zero, 0, err
	}
	if err = requireActiveOutcome(ctx, uow, cmd.Scope); err != nil {
		return zero, 0, err
	}
	coord, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
	if err != nil {
		return zero, 0, err
	}
	repo, workRepo, err := signedRepository(uow)
	if err != nil {
		return zero, 0, err
	}
	work, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
	if err != nil {
		return zero, 0, err
	}
	if !work.ContractsEnabled || work.Version != cmd.ExpectedWorkItemVersion {
		return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "signed work version/protocol differs")
	}
	if pending, e := repo.OpenCase(ctx, cmd.Scope, work.ID); e != nil {
		return zero, 0, e
	} else if pending != nil {
		return zero, 0, d.NewError(d.ErrorCodePreconditionFailed, "Task awaits independent review")
	}
	now, err := securityTransactionTime(ctx, uow, s.clock)
	if err != nil {
		return zero, 0, err
	}
	counts, err := repo.Counts(ctx, cmd.Scope.NamespaceID, identity.PrincipalID, identity.CredentialID, now)
	if err != nil {
		return zero, 0, err
	}
	if counts.PrincipalWork >= 3 || counts.CredentialWork >= credential.MaxActiveWorkContracts {
		return zero, 0, d.NewError(d.ErrorCodePreconditionFailed, "active signed execution quota reached")
	}
	registry, err := signingRepository(uow)
	if err != nil {
		return zero, 0, err
	}
	if nsPolicy, e := registry.AcceptancePolicy(ctx, cmd.Scope.NamespaceID, nil, nil); e == nil {
		if counts.NamespaceWork >= nsPolicy.MaxActiveWorkContracts {
			return zero, 0, d.NewError(d.ErrorCodePreconditionFailed, "Namespace execution quota reached")
		}
	} else if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeNotFound {
		return zero, 0, e
	}
	var previous *d.WorkContract
	previousExpiredNow := false
	if work.CurrentContractID != nil {
		old, e := workRepo.Get(ctx, cmd.Scope, *work.CurrentContractID)
		if code, _ := d.ErrorCodeOf(e); code == d.ErrorCodeNotFound {
			legacy, ok := uow.(ports.WorkContractUnitOfWork)
			if !ok {
				return zero, 0, e
			}
			old, e = legacy.WorkContracts().Get(ctx, cmd.Scope, *work.CurrentContractID)
		}
		if e != nil {
			return zero, 0, e
		}
		if old.ValidAt(now) {
			return zero, 0, d.NewError(d.ErrorCodeWorkAlreadyClaimed, "current authority still valid")
		}
		if old.Status == d.ContractActive {
			if old.SignedBinding == nil {
				return zero, 0, d.NewError(d.ErrorCodeSignedProtocolRequired, "reconcile old v1 authority before signed acquisition")
			}
			v, l := old.Version, old.LeaseVersion
			if e = old.Expire(now); e != nil {
				return zero, 0, e
			}
			if e = workRepo.Save(ctx, old, v, l); e != nil {
				return zero, 0, e
			}
			previousExpiredNow = true
		}
		previous = &old
		work.CurrentContractID = nil
	}
	if work.Lifecycle != d.WorkItemLifecycleTodo && work.Lifecycle != d.WorkItemLifecycleInProgress {
		return zero, 0, d.NewError(d.ErrorCodePreconditionFailed, "Task lifecycle not acquirable")
	}
	candidate := work
	candidate.Lifecycle = d.WorkItemLifecycleTodo
	candidate.CurrentLease = nil
	if err = requireWorkItemReady(ctx, uow, candidate, now); err != nil {
		return zero, 0, err
	}
	spec, err := snapshotContractSpec(ctx, uow, work)
	if err != nil {
		return zero, 0, err
	}
	floor, err := s.signedAcceptanceFloor(ctx, uow, cmd.Scope, work.ID, credential, d.AcceptanceDirect)
	if err != nil {
		return zero, 0, err
	}
	var correction *d.ReviewCase
	var findings []d.SignedFinding
	if work.CorrectionReviewCaseID != nil {
		if cmd.PreviousReviewCaseID == nil || *cmd.PreviousReviewCaseID != *work.CorrectionReviewCaseID {
			return zero, 0, d.NewError(d.ErrorCodePreconditionFailed, "correction must explicitly target the pending prior review")
		}
		prior, e := repo.Case(ctx, cmd.Scope, *cmd.PreviousReviewCaseID)
		if e != nil {
			return zero, 0, e
		}
		if prior.Status != d.ReviewChangesRequested || prior.WorkItemID != work.ID || prior.LatestDecisionID == nil || previous == nil || previous.Status != d.ContractDelivered || previous.ID != prior.WorkContractID {
			return zero, 0, d.NewError(d.ErrorCodeSubmissionNotAccepted, "pending correction target differs")
		}
		decisionFact, e := repo.Fact(ctx, cmd.Scope, *prior.LatestDecisionID)
		if e != nil {
			return zero, 0, e
		}
		decision, e := verifiedReviewDecision(ctx, uow, decisionFact, server.ID.String())
		if e != nil {
			return zero, 0, e
		}
		if decision.Decision != "changes_requested" || decision.ReviewCaseID != prior.ID.String() || decision.SubmissionID != prior.SubmissionID.String() || decision.SubmissionDigest != prior.SubmissionDigest {
			return zero, 0, d.NewError(d.ErrorCodeSubmissionNotAccepted, "signed correction findings differ")
		}
		correction = &prior
		findings = decision.Material.Findings
		for _, finding := range findings {
			if e = finding.ValidateAgainst(spec); e != nil {
				return zero, 0, e
			}
		}
		floor = d.StrongerAcceptance(floor, prior.AcceptanceFloor)
	} else if work.LatestReviewCaseID != nil && previous != nil && previous.Status == d.ContractDelivered {
		if cmd.PreviousReviewCaseID == nil || *cmd.PreviousReviewCaseID != *work.LatestReviewCaseID {
			return zero, 0, d.NewError(d.ErrorCodePreconditionFailed, "replanned execution must acknowledge the prior closed review")
		}
		prior, e := repo.Case(ctx, cmd.Scope, *cmd.PreviousReviewCaseID)
		if e != nil {
			return zero, 0, e
		}
		if (prior.Status != d.ReviewCancelled && prior.Status != d.ReviewSuperseded) || prior.WorkContractID != previous.ID {
			return zero, 0, d.NewError(d.ErrorCodeSubmissionNotAccepted, "prior review was not explicitly cancelled/superseded")
		}
		correction = &prior
	} else if cmd.PreviousReviewCaseID != nil {
		return zero, 0, d.NewError(d.ErrorCodeInvalidArgument, "Task has no acknowledged prior review/correction")
	}

	group, err := registry.PrincipalGroup(ctx, cmd.Scope.NamespaceID, identity.PrincipalID)
	if err != nil {
		return zero, 0, err
	}
	allowed := []d.ID{}
	for _, id := range credential.AllowedSigningKeyIDs {
		key, e := registry.Key(ctx, cmd.Scope.NamespaceID, id)
		if e != nil {
			if code, _ := d.ErrorCodeOf(e); code == d.ErrorCodeNotFound {
				continue
			}
			return zero, 0, e
		}
		if key.Status == "active" && key.Purpose == "agent" && key.PrincipalID == identity.PrincipalID {
			allowed = append(allowed, id)
		}
	}
	binding := d.SignedContractBinding{ProtocolVersion: 2, ServerID: server.ID.String(), CredentialID: identity.CredentialID, SignerKeyID: cmd.SignerKeyID, AcceptanceFloor: floor, PolicyRevision: credential.Version, AllowedSigningKeyIDs: allowed, SeparationGroup: group}
	if correction != nil {
		binding.PreviousReviewCaseID = &correction.ID
		binding.PreviousSubmissionID = &correction.SubmissionID
		binding.CorrectionFindings = findings
	}
	contractID, err := s.ids.NewID()
	if err != nil {
		return zero, 0, err
	}
	executionID, err := s.ids.NewID()
	if err != nil {
		return zero, 0, err
	}
	protocol, err := protocolRepository(uow)
	if err != nil {
		return zero, 0, err
	}
	p, err := protocol.Lock(ctx, cmd.Scope.NamespaceID, true)
	if err != nil {
		return zero, 0, err
	}
	contract, err := d.NewWorkContract(contractID, executionID, work, identity.PrincipalID, identity.Actor, p.LeasePolicy, cmd.TTLSeconds, spec, coord.Revision, now)
	if err != nil {
		return zero, 0, err
	}
	contract.SignedBinding = &binding
	payload := signing.SpecPayload[d.SignedWorkSpec]{Binding: signing.Binding{ProtocolVersion: 2, ServerID: server.ID.String(), NamespaceID: cmd.Scope.NamespaceID.String(), OutcomeID: cmd.Scope.OutcomeID.String(), PrincipalID: identity.PrincipalID, SignerKeyID: server.IssuerKeyID.String()}, ContractKind: "execution", ContractID: contract.ID.String(), WorkItemID: work.ID.String(), Spec: d.NewSignedWorkSpec(contract.Spec, binding)}
	specFact, specDocument, err := s.issueSignedFact(ctx, uow, server, cmd.Scope, contract.ID, "execution", "specification", identity.PrincipalID, signing.ContractSpec, payload, now)
	if err != nil {
		return zero, 0, err
	}
	binding.SpecificationDigest = specFact.PayloadDigest
	contract.IssuedSpecificationID = &specFact.ID
	authorityFact, authorityDocument, err := s.issueWorkAuthority(ctx, uow, server, contract, credential, allowed, now)
	if err != nil {
		return zero, 0, err
	}
	contract.LatestAuthorityID = &authorityFact.ID
	work.CorrectionReviewCaseID = nil
	if err = work.BindContract(contract, now); err != nil {
		return zero, 0, err
	}
	if err = workRepo.Insert(ctx, contract); err != nil {
		return zero, 0, err
	}
	if err = repo.InsertFact(ctx, specFact); err != nil {
		return zero, 0, err
	}
	if err = repo.InsertFact(ctx, authorityFact); err != nil {
		return zero, 0, err
	}
	if err = uow.WorkItems().Save(ctx, work, cmd.ExpectedWorkItemVersion); err != nil {
		return zero, 0, err
	}
	revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
	return WorkContractResult{Contract: contract, WorkItem: work, PreviousContract: previous, previousExpiredNow: previousExpiredNow, Recovery: previous != nil, EvaluatedAt: now, IssuedSpecification: &specDocument, IssuedAuthority: &authorityDocument}, revision, err
}
func snapshotContractSpec(ctx context.Context, uow ports.UnitOfWork, work d.WorkItem) (d.WorkContractSpec, error) {
	spec := d.WorkContractSpec{Title: work.Title, Description: work.Description, ObjectiveID: cloneIDPtr(work.ObjectiveID), Criteria: append([]d.SuccessCriterion{}, work.Criteria.Items...)}
	outcome, err := uow.Outcomes().Get(ctx, work.Scope.NamespaceID, work.Scope.OutcomeID)
	if err != nil {
		return spec, err
	}
	spec.OutcomeIntent = outcome.DesiredState
	if work.ObjectiveID != nil {
		objective, e := uow.Objectives().Get(ctx, work.Scope, *work.ObjectiveID)
		if e != nil {
			return spec, e
		}
		spec.ObjectiveIntent = objective.Title
		if objective.Description != "" {
			spec.ObjectiveIntent += "\n" + objective.Description
		}
	}
	if work.ExecutionSpec != nil {
		spec.ExecutionSpec = *work.ExecutionSpec
	}
	relations, err := uow.Relations().ListByOutcome(ctx, work.Scope)
	if err != nil {
		return spec, err
	}
	for _, relation := range relations {
		if relation.SourceRef == work.Ref() && relation.RelationType == d.RelationTypeDependsOn && relation.Lifecycle == d.RelationLifecycleActive {
			spec.Dependencies = append(spec.Dependencies, relation.TargetRef)
		}
	}
	return d.NormalizeContractSpec(spec), nil
}
func (s *Service) issueWorkAuthority(ctx context.Context, uow ports.UnitOfWork, server d.ServerIdentity, c d.WorkContract, policy d.CredentialPolicy, keys []d.ID, now time.Time) (d.SignedFact, signing.Document, error) {
	allowed := []string{}
	for _, id := range keys {
		allowed = append(allowed, id.String())
	}
	payload := signing.AuthorityPayload{Binding: signing.Binding{ProtocolVersion: 2, ServerID: server.ID.String(), NamespaceID: c.Scope.NamespaceID.String(), OutcomeID: c.Scope.OutcomeID.String(), PrincipalID: c.HolderPrincipalID, SignerKeyID: server.IssuerKeyID.String()}, ContractKind: "execution", ContractID: c.ID.String(), WorkItemID: c.WorkItemID.String(), ExecutionID: c.ExecutionID.String(), FencingToken: signing.Decimal(c.FencingToken), ContractVersion: signing.Decimal(c.Version), LeaseVersion: signing.Decimal(c.LeaseVersion), IssuedAt: now.UTC().Format(time.RFC3339Nano), ExpiresAt: c.ExpiresAt.UTC().Format(time.RFC3339Nano), SpecDigest: c.SignedBinding.SpecificationDigest, AcceptanceMode: string(c.SignedBinding.AcceptanceFloor), PolicyRevision: signing.Decimal(policy.Version), AllowedSigningKeyIDs: allowed}
	return s.issueSignedFact(ctx, uow, server, c.Scope, c.ID, "execution", "authority", c.HolderPrincipalID, signing.ContractAuthority, payload, now)
}
