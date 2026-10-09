package application

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"time"
)

type SetNamespaceWorkProtocolCommand struct {
	Scope                   domain.Scope
	ExpectedProtocolVersion domain.Version
	Phase                   domain.WorkProtocolPhase
	WritersDrained          bool                `wos:"optional"`
	LeasePolicy             *domain.LeasePolicy `wos:"optional"`
	Reason                  string
}
type ProtocolOutcomeRevision struct {
	Scope    domain.Scope           `json:"scope"`
	Revision domain.OutcomeRevision `json:"outcome_revision"`
}
type NamespaceWorkProtocolResult struct {
	Protocol         domain.NamespaceWorkProtocol `json:"protocol"`
	AffectedOutcomes []ProtocolOutcomeRevision    `json:"affected_outcomes"`
}

func protocolRepository(u ports.UnitOfWork) (ports.WorkProtocolRepository, error) {
	p, ok := u.(ports.WorkProtocolUnitOfWork)
	if !ok {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "adapter lacks Namespace protocol metadata")
	}
	return p.WorkProtocol(), nil
}
func (s *Service) SetNamespaceWorkProtocol(ctx context.Context, cc domain.CommandContext, cmd SetNamespaceWorkProtocolCommand) (MutationResult[NamespaceWorkProtocolResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[NamespaceWorkProtocolResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (NamespaceWorkProtocolResult, domain.OutcomeRevision, error) {
		zero := NamespaceWorkProtocolResult{}
		repo, err := protocolRepository(u)
		if err != nil {
			return zero, 0, err
		}
		p, err := repo.Lock(ctx, cmd.Scope.NamespaceID, true)
		if err != nil {
			return zero, 0, err
		}
		if p.Version != cmd.ExpectedProtocolVersion {
			return zero, 0, domain.NewError(domain.ErrorCodeVersionConflict, "Namespace protocol version conflict")
		}
		if cmd.Reason == "" {
			return zero, 0, domain.NewError(domain.ErrorCodeInvalidArgument, "protocol change requires reason")
		}
		legal := p.Phase == cmd.Phase || p.Phase == domain.WorkProtocolLegacy && cmd.Phase == domain.WorkProtocolDraining || p.Phase == domain.WorkProtocolDraining && cmd.Phase == domain.WorkProtocolContracts || p.Phase == domain.WorkProtocolContracts && cmd.Phase == domain.WorkProtocolSignedDraining || p.Phase == domain.WorkProtocolSignedDraining && cmd.Phase == domain.WorkProtocolSigned
		if !legal {
			return zero, 0, domain.NewError(domain.ErrorCodeInvalidTransition, "protocol requires legacy -> draining -> contracts_v1 -> draining_to_signed_v2 -> signed_contracts_v2; downgrade is unsupported")
		}
		scopes, err := repo.Scopes(ctx, cmd.Scope.NamespaceID)
		if err != nil {
			return zero, 0, err
		}
		anchor := false
		for _, scope := range scopes {
			if scope == cmd.Scope {
				anchor = true
			}
			if _, err = u.Coordination().LockOutcome(ctx, scope); err != nil {
				return zero, 0, err
			}
		}
		if !anchor {
			return zero, 0, domain.NewError(domain.ErrorCodeNotFound, "protocol command requires an existing anchor Outcome")
		}
		now, err := s.transactionTime(ctx, u)
		if err != nil {
			return zero, 0, err
		}
		next := p
		next.Phase = cmd.Phase
		next.Version++
		next.UpdatedAt = now
		next.UpdatedBy = cc.PrincipalID
		next.Reason = cmd.Reason
		if cmd.LeasePolicy != nil {
			next.LeasePolicy = *cmd.LeasePolicy
			if next.LeasePolicy.Revision <= p.LeasePolicy.Revision {
				return zero, 0, domain.NewError(domain.ErrorCodeVersionConflict, "lease policy revision must increase")
			}
		}
		if next.Phase == domain.WorkProtocolContracts {
			if !cmd.WritersDrained {
				return zero, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "operator must confirm old writers have been stopped and drained")
			}
			next.WriterEpoch = 1
			next.WritersDrained = true
			if p.Phase != domain.WorkProtocolContracts {
				if err = repo.EnableContracts(ctx, p.NamespaceID, now); err != nil {
					return zero, 0, err
				}
			}
		}
		if next.Phase == domain.WorkProtocolSignedDraining {
			next.WriterEpoch = 1
			next.WritersDrained = false
		}
		if next.Phase == domain.WorkProtocolSigned {
			if !cmd.WritersDrained {
				return zero, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "operator must explicitly retire and drain all v1 writers")
			}
			if count, e := repo.ValidLegacyLeases(ctx, p.NamespaceID, now); e != nil {
				return zero, 0, e
			} else if count > 0 {
				return zero, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "valid legacy claims remain")
			}
			if count, e := repo.ValidUnsignedContracts(ctx, p.NamespaceID, now); e != nil {
				return zero, 0, e
			} else if count > 0 {
				return zero, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "valid unsigned contract authority remains")
			}
			if _, e := s.requireSignedIssuer(ctx, u, p.NamespaceID); e != nil {
				return zero, 0, e
			}
			registry, e := signingRepository(u)
			if e != nil {
				return zero, 0, e
			}
			policy, e := registry.AcceptancePolicy(ctx, p.NamespaceID, nil, nil)
			if e != nil {
				return zero, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "explicit Namespace acceptance policy required before signed activation")
			}
			if e = policy.Validate(); e != nil {
				return zero, 0, e
			}
			next.WriterEpoch = 2
			next.WritersDrained = true
			if p.Phase != domain.WorkProtocolSigned {
				if e = repo.EnableContracts(ctx, p.NamespaceID, now); e != nil {
					return zero, 0, e
				}
			}
		}
		if err = repo.Save(ctx, next, p.Version); err != nil {
			return zero, 0, err
		}
		result := NamespaceWorkProtocolResult{Protocol: next, AffectedOutcomes: []ProtocolOutcomeRevision{}}
		var anchorRevision domain.OutcomeRevision
		for _, scope := range scopes {
			rev, e := u.Coordination().AdvanceOutcome(ctx, scope)
			if e != nil {
				return zero, 0, e
			}
			result.AffectedOutcomes = append(result.AffectedOutcomes, ProtocolOutcomeRevision{scope, rev})
			if scope == cmd.Scope {
				anchorRevision = rev
			}
		}
		return result, anchorRevision, nil
	})
}
func (s *Service) GetNamespaceWorkProtocol(ctx context.Context, ns domain.ID) (domain.NamespaceWorkProtocol, error) {
	if err := ns.Validate(); err != nil {
		return domain.NamespaceWorkProtocol{}, err
	}
	if err := s.authorizeNamespaceMetadataRead(ctx, ns); err != nil {
		return domain.NamespaceWorkProtocol{}, err
	}
	u, err := s.tx.Begin(ctx)
	if err != nil {
		return domain.NamespaceWorkProtocol{}, err
	}
	defer u.Rollback()
	if err = s.authorizeScopedReadInUnitOfWork(ctx, u, domain.Scope{NamespaceID: ns}); err != nil {
		return domain.NamespaceWorkProtocol{}, err
	}
	repo, err := protocolRepository(u)
	if err != nil {
		return domain.NamespaceWorkProtocol{}, err
	}
	return repo.Lock(ctx, ns, false)
}
func protectWorkProtocol(p domain.NamespaceWorkProtocol, name string) error {
	if p.Phase == domain.WorkProtocolSigned || p.Phase == domain.WorkProtocolSignedDraining {
		switch name {
		case "AcquireWorkContract", "AcquireNextWorkContract":
			return domain.NewError(domain.ErrorCodeSignedProtocolRequired, "unsigned acquisition disabled by signed protocol or drain")
		}
	}
	if p.Phase == domain.WorkProtocolSigned {
		switch name {
		case "ClaimWorkItem", "ReclaimWorkItem", "RenewWorkItemLease", "ReleaseWorkItem", "CompleteWorkItem", "AdministrativeCompleteWorkItem", "RenewWorkContract", "ResumeWorkContract", "RevokeWorkContract", "SyncWorkContract", "SubmitWorkResult", "FinalizeWorkContract", "ReconcileExpiredWorkContracts":
			return domain.NewError(domain.ErrorCodeSignedProtocolRequired, "signed Namespace requires signed contract operations")
		}
	}

	blocked := false
	switch name {
	case "ClaimWorkItem", "ReclaimWorkItem":
		blocked = p.Phase != domain.WorkProtocolLegacy
	case "RenewWorkItemLease", "ReleaseWorkItem", "CompleteWorkItem", "AdministrativeCompleteWorkItem":
		blocked = p.Phase == domain.WorkProtocolContracts
	}
	if blocked {
		return domain.NewError(domain.ErrorCodeContractProtocolRequired, "legacy command is disabled by Namespace work protocol")
	}
	return nil
}
func protocolCommandEvents(s *Service, cc domain.CommandContext, meta commandMetadata, value any) ([]domain.DomainEvent, bool, error) {
	if _, ok := value.(ReconciledWorkContracts); ok {
		return nil, true, nil
	}
	r, ok := value.(NamespaceWorkProtocolResult)
	if !ok {
		return nil, false, nil
	}
	events := []domain.DomainEvent{}
	for _, affected := range r.AffectedOutcomes {
		outcomeRef := domain.EntityRef{Scope: affected.Scope, Kind: domain.EntityKindOutcome, ID: affected.Scope.OutcomeID}
		es, err := buildCompoundEvents(s, cc, meta, affected.Revision, []compoundEventSpec{{eventType: "outcome.work_protocol_changed", ref: outcomeRef}})
		if err != nil {
			return nil, true, err
		}
		for i := range es {
			es[i].RecordedAt = r.Protocol.UpdatedAt
		}
		events = append(events, es...)
	}
	return events, true, nil
}

type ReconcileExpiredWorkContractsCommand struct {
	Scope domain.Scope
	Limit int       `wos:"optional"`
	After domain.ID `wos:"optional" json:",omitempty"`
}
type ReconciledWorkContracts struct {
	Contracts      []domain.ID `json:"contracts"`
	NextAfter      domain.ID   `json:"next_after,omitempty"`
	SearchComplete bool        `json:"search_complete"`
	EvaluatedAt    time.Time   `json:"evaluated_at"`
}

// Explicit bounded housekeeping. It never launches agents, renews or acquires work.
func (s *Service) ReconcileExpiredWorkContracts(ctx context.Context, cc domain.CommandContext, cmd ReconcileExpiredWorkContractsCommand) (MutationResult[ReconciledWorkContracts], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[ReconciledWorkContracts]{}, err
	}
	if cmd.Limit == 0 {
		cmd.Limit = 25
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		return MutationResult[ReconciledWorkContracts]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "limit must be 1..100")
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (ReconciledWorkContracts, domain.OutcomeRevision, error) {
		result := ReconciledWorkContracts{Contracts: []domain.ID{}}
		coord, err := u.Coordination().LockOutcome(ctx, cmd.Scope)
		if err != nil {
			return result, 0, err
		}
		repo, err := contractRepository(u)
		if err != nil {
			return result, 0, err
		}
		now, err := s.transactionTime(ctx, u)
		if err != nil {
			return result, 0, err
		}
		result.EvaluatedAt = now
		list, err := repo.List(ctx, cmd.Scope, ports.ContractFilter{Status: domain.ContractActive, After: cmd.After, Limit: cmd.Limit + 1})
		if err != nil {
			return result, 0, err
		}
		result.SearchComplete = len(list) <= cmd.Limit
		if !result.SearchComplete {
			list = list[:cmd.Limit]
		}
		if len(list) > 0 {
			result.NextAfter = list[len(list)-1].ID
		}
		for _, header := range list {
			c, e := repo.Get(ctx, cmd.Scope, header.ID)
			if e != nil {
				return result, 0, e
			}
			if c.ValidAt(now) {
				continue
			}
			v, l := c.Version, c.LeaseVersion
			if e = c.Expire(now); e != nil {
				return result, 0, e
			}
			if e = repo.Save(ctx, c, v, l); e != nil {
				return result, 0, e
			}
			w, e := u.WorkItems().Get(ctx, cmd.Scope, c.WorkItemID)
			if e != nil {
				return result, 0, e
			}
			if w.CurrentContractID != nil && *w.CurrentContractID == c.ID {
				before := w.Version
				if e = w.DetachContract(c, now); e != nil {
					return result, 0, e
				}
				if e = u.WorkItems().Save(ctx, w, before); e != nil {
					return result, 0, e
				}
			}
			result.Contracts = append(result.Contracts, c.ID)
		}
		rev := coord.Revision
		if len(result.Contracts) > 0 {
			rev, err = u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
			if err != nil {
				return result, 0, err
			}
			for i, id := range result.Contracts {
				c, e := repo.Get(ctx, cmd.Scope, id)
				if e != nil {
					return result, 0, e
				}
				meta := commandMetadata{Name: "ReconcileExpiredWorkContracts", Scope: cmd.Scope}
				meta.Payload, _ = json.Marshal(map[string]any{"contract_id": id, "cause": "expiry"})
				es, e := buildCompoundEvents(s, cc, meta, rev, []compoundEventSpec{{eventType: "work_contract.expired", ref: c.Ref()}})
				if e != nil {
					return result, 0, e
				}
				es[0].EventIndex = uint32(i)
				es[0].RecordedAt = now
				if e = u.Events().Append(ctx, es); e != nil {
					return result, 0, e
				}
			}
		}
		return result, rev, nil
	})
}
