package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"strings"
	"time"
)

type RevokeSignedContractCommand struct {
	Scope                   d.Scope
	ContractKind            string
	ContractID              d.ID
	ExpectedContractVersion d.Version
	Reason                  string
}
type InterveneSignedReviewCaseCommand struct {
	Scope                     d.Scope
	ReviewCaseID              d.ID
	ExpectedReviewCaseVersion d.Version
	Status                    d.ReviewCaseStatus
	Reason                    string
}
type SignedInterventionResult struct {
	Reason          string           `json:"reason"`
	ContractID      d.ID             `json:"contract_id,omitempty"`
	ContractKind    string           `json:"contract_kind,omitempty"`
	ContractStatus  d.ContractStatus `json:"contract_status,omitempty"`
	ContractVersion signing.Decimal  `json:"contract_version,omitempty"`
	ReviewCase      *d.ReviewCase    `json:"review_case,omitempty"`
	WorkItemID      d.ID             `json:"work_item_id"`
	WorkItemVersion signing.Decimal  `json:"work_item_version"`
	EvaluatedAt     time.Time        `json:"evaluated_at"`
	facts           []string
}

func (s *Service) RevokeSignedContract(ctx context.Context, cc d.CommandContext, cmd RevokeSignedContractCommand) (MutationResult[SignedInterventionResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[SignedInterventionResult]{}, err
	}
	if cc.IdempotencyKey == "" || strings.TrimSpace(cmd.Reason) == "" {
		return MutationResult[SignedInterventionResult]{}, d.NewError(d.ErrorCodeInvalidArgument, "revocation needs reason and stable intent")
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (SignedInterventionResult, d.OutcomeRevision, error) {
		var zero SignedInterventionResult
		if _, _, err := s.signedAccess(ctx, u, cmd.Scope, ports.PermissionWorkContractRevoke, d.ID("")); err != nil {
			return zero, 0, err
		}
		if _, err := u.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return zero, 0, err
		}
		now, err := securityTransactionTime(ctx, u, s.clock)
		if err != nil {
			return zero, 0, err
		}
		repo, workRepo, err := signedRepository(u)
		if err != nil {
			return zero, 0, err
		}
		var work d.WorkItem
		result := SignedInterventionResult{Reason: cmd.Reason, ContractID: cmd.ContractID, ContractKind: cmd.ContractKind, EvaluatedAt: now}
		switch cmd.ContractKind {
		case "execution":
			c, e := workRepo.Get(ctx, cmd.Scope, cmd.ContractID)
			if e != nil {
				return zero, 0, e
			}
			v, l := c.Version, c.LeaseVersion
			if e = c.Revoke(cc.PrincipalID, cmd.Reason, cmd.ExpectedContractVersion, now); e != nil {
				return zero, 0, e
			}
			if e = workRepo.Save(ctx, c, v, l); e != nil {
				return zero, 0, e
			}
			work, e = u.WorkItems().Get(ctx, cmd.Scope, c.WorkItemID)
			if e != nil {
				return zero, 0, e
			}
			if work.CurrentContractID != nil && *work.CurrentContractID == c.ID {
				wv := work.Version
				work.CurrentContractID = nil
				next, e := wv.Next()
				if e != nil {
					return zero, 0, e
				}
				work.Version = next
				work.UpdatedAt = now
				if e = u.WorkItems().Save(ctx, work, wv); e != nil {
					return zero, 0, e
				}
			}
			result.ContractStatus = c.Status
			result.ContractVersion = signing.Decimal(c.Version)
			result.facts = []string{"work_contract.revoked"}
		case "review":
			c, e := repo.ReviewContract(ctx, cmd.Scope, cmd.ContractID)
			if e != nil {
				return zero, 0, e
			}
			v, l := c.Version, c.LeaseVersion
			if e = c.Close(d.ContractRevoked, cc.PrincipalID, cmd.Reason, cmd.ExpectedContractVersion, now); e != nil {
				return zero, 0, e
			}
			r, e := repo.Case(ctx, cmd.Scope, c.CaseID)
			if e != nil {
				return zero, 0, e
			}
			rv := r.Version
			if e = r.Release(c, now); e != nil {
				return zero, 0, e
			}
			if e = repo.SaveReviewContract(ctx, c, v, l); e != nil {
				return zero, 0, e
			}
			if e = repo.SaveCase(ctx, r, rv); e != nil {
				return zero, 0, e
			}
			work, e = u.WorkItems().Get(ctx, cmd.Scope, c.WorkItemID)
			if e != nil {
				return zero, 0, e
			}
			result.ReviewCase = &r
			result.ContractStatus = c.Status
			result.ContractVersion = signing.Decimal(c.Version)
			result.facts = []string{"work_review.revoked"}
		default:
			return zero, 0, d.NewError(d.ErrorCodeInvalidArgument, "contract kind must be execution/review")
		}
		result.WorkItemID = work.ID
		result.WorkItemVersion = signing.Decimal(work.Version)
		revision, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return result, revision, err
	})
}
func (s *Service) InterveneSignedReviewCase(ctx context.Context, cc d.CommandContext, cmd InterveneSignedReviewCaseCommand) (MutationResult[SignedInterventionResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[SignedInterventionResult]{}, err
	}
	if cc.IdempotencyKey == "" || strings.TrimSpace(cmd.Reason) == "" || (cmd.Status != d.ReviewCancelled && cmd.Status != d.ReviewSuperseded) {
		return MutationResult[SignedInterventionResult]{}, d.NewError(d.ErrorCodeInvalidArgument, "explicit review cancellation/supersession needs reason and intent")
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (SignedInterventionResult, d.OutcomeRevision, error) {
		var zero SignedInterventionResult
		if _, _, err := s.signedAccess(ctx, u, cmd.Scope, ports.PermissionNamespaceAdmin, d.ID("")); err != nil {
			return zero, 0, err
		}
		if _, err := u.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return zero, 0, err
		}
		now, err := securityTransactionTime(ctx, u, s.clock)
		if err != nil {
			return zero, 0, err
		}
		repo, _, err := signedRepository(u)
		if err != nil {
			return zero, 0, err
		}
		review, err := repo.Case(ctx, cmd.Scope, cmd.ReviewCaseID)
		if err != nil {
			return zero, 0, err
		}
		if review.Version != cmd.ExpectedReviewCaseVersion || !review.Status.Open() {
			return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "open review case CAS required")
		}
		result := SignedInterventionResult{Reason: cmd.Reason, ReviewCase: &review, WorkItemID: review.WorkItemID, EvaluatedAt: now}
		if review.CurrentContractID != nil {
			c, e := repo.ReviewContract(ctx, cmd.Scope, *review.CurrentContractID)
			if e != nil {
				return zero, 0, e
			}
			v, l, rv := c.Version, c.LeaseVersion, review.Version
			status := d.ContractRevoked
			typ := "work_review.revoked"
			if !now.Before(c.ExpiresAt) {
				status = d.ContractExpired
				typ = "work_review.expired"
			}
			if e = c.Close(status, cc.PrincipalID, cmd.Reason, v, now); e != nil {
				return zero, 0, e
			}
			if e = review.Release(c, now); e != nil {
				return zero, 0, e
			}
			if e = repo.SaveReviewContract(ctx, c, v, l); e != nil {
				return zero, 0, e
			}
			if e = repo.SaveCase(ctx, review, rv); e != nil {
				return zero, 0, e
			}
			result.ContractID = c.ID
			result.ContractKind = "review"
			result.ContractStatus = c.Status
			result.ContractVersion = signing.Decimal(c.Version)
			result.facts = append(result.facts, typ)
		}
		rv := review.Version
		if err = review.Intervene(cmd.Status, cc.PrincipalID, cmd.Reason, rv, now); err != nil {
			return zero, 0, err
		}
		if err = repo.SaveCase(ctx, review, rv); err != nil {
			return zero, 0, err
		}
		work, err := u.WorkItems().Get(ctx, cmd.Scope, review.WorkItemID)
		if err != nil {
			return zero, 0, err
		}
		if work.PendingReviewCaseID == nil || *work.PendingReviewCaseID != review.ID {
			return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "Task pending review navigation differs")
		}
		wv := work.Version
		work.PendingReviewCaseID = nil
		work.CorrectionReviewCaseID = nil
		next, err := wv.Next()
		if err != nil {
			return zero, 0, err
		}
		work.Version = next
		work.UpdatedAt = now
		if err = u.WorkItems().Save(ctx, work, wv); err != nil {
			return zero, 0, err
		}
		result.WorkItemVersion = signing.Decimal(work.Version)
		typ := "work_review.cancelled"
		if cmd.Status == d.ReviewSuperseded {
			typ = "work_review.superseded"
		}
		result.facts = append(result.facts, typ)
		revision, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return result, revision, err
	})
}
func signedInterventionEvents(s *Service, cc d.CommandContext, meta commandMetadata, r SignedInterventionResult, revision d.OutcomeRevision) ([]d.DomainEvent, error) {
	specs := []compoundEventSpec{}
	ref := d.EntityRef{Scope: meta.Scope, Kind: d.EntityKindWorkItem, ID: r.WorkItemID}
	for _, typ := range r.facts {
		specs = append(specs, compoundEventSpec{eventType: typ, ref: ref, after: versionPtr(d.Version(r.WorkItemVersion))})
	}
	events, err := buildCompoundEvents(s, cc, meta, revision, specs)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	for i := range events {
		events[i].Payload = raw
		events[i].RecordedAt = r.EvaluatedAt
		if err = events[i].Validate(); err != nil {
			return nil, err
		}
	}
	return events, nil
}
