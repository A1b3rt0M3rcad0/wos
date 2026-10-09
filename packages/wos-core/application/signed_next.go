package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// Every bounded search is its own durable intention, including empty results.
// Continuing a cursor requires another intention rather than changing a replay.
type AcquireNextSignedWorkContractCommand struct {
	Scope       d.Scope
	SignerKeyID d.ID
	WorkItemIDs []d.ID       `wos:"optional"`
	ObjectiveID *d.ID        `wos:"optional"`
	Priorities  []d.Priority `wos:"optional"`
	TTLSeconds  int          `wos:"optional"`
	Limit       int          `wos:"optional"`
	Cursor      string       `wos:"optional"`
}

func (s *Service) AcquireNextSignedWorkContract(ctx context.Context, cc d.CommandContext, cmd AcquireNextSignedWorkContractCommand) (MutationResult[WorkContractAcquisition], error) {
	var zero MutationResult[WorkContractAcquisition]
	if e := cc.Validate(); e != nil {
		return zero, e
	}
	if cc.IdempotencyKey == "" || cmd.Scope.Validate() != nil || cmd.SignerKeyID.Validate() != nil || len(cmd.WorkItemIDs) > 100 {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "signed search requires exact scope, key and independent bounded intention")
	}
	for _, id := range cmd.WorkItemIDs {
		if e := id.Validate(); e != nil {
			return zero, e
		}
	}
	if cmd.ObjectiveID != nil {
		if e := cmd.ObjectiveID.Validate(); e != nil {
			return zero, e
		}
	}
	for _, priority := range cmd.Priorities {
		if !priority.Valid() {
			return zero, d.NewError(d.ErrorCodeInvalidArgument, "invalid priority")
		}
	}
	limit, e := queryLimit(cmd.Limit)
	if e != nil {
		return zero, e
	}
	fingerprint := filterHash(struct {
		IDs        []d.ID
		Objective  *d.ID
		Priorities []d.Priority
	}{cmd.WorkItemIDs, cmd.ObjectiveID, cmd.Priorities})
	query := ports.ContractCandidateQuery{Limit: limit + 1}
	if cmd.Cursor != "" {
		cursor, e := decodeCursor(cmd.Cursor, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID, "signed_contract_candidates", fingerprint)
		if e != nil {
			return zero, e
		}
		var last ports.ContractCandidate
		if json.Unmarshal([]byte(cursor.Key), &last) != nil || last.ID.Validate() != nil {
			return zero, d.NewError(d.ErrorCodeInvalidArgument, "invalid signed candidate cursor")
		}
		query.AfterID = last.ID
		query.AfterCreated = last.CreatedAt
		query.AfterPriority = ports.ContractPriorityRank(last.Priority)
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (WorkContractAcquisition, d.OutcomeRevision, error) {
		result := WorkContractAcquisition{Reasons: []string{}}
		if _, _, e := s.signedAccess(ctx, u, cmd.Scope, ports.PermissionWorkContractAcquire, cmd.SignerKeyID); e != nil {
			return result, 0, e
		}
		if _, e := s.requireSignedIssuer(ctx, u, cmd.Scope.NamespaceID); e != nil {
			return result, 0, e
		}
		if e := requireActiveOutcome(ctx, u, cmd.Scope); e != nil {
			return result, 0, e
		}
		protocol, e := protocolRepository(u)
		if e != nil {
			return result, 0, e
		}
		state, e := protocol.Lock(ctx, cmd.Scope.NamespaceID, true)
		if e != nil {
			return result, 0, e
		}
		if _, e = state.LeasePolicy.TTL(cmd.TTLSeconds); e != nil {
			return result, 0, e
		}
		coord, e := u.Coordination().LockOutcome(ctx, cmd.Scope)
		if e != nil {
			return result, 0, e
		}
		indexed, ok := u.WorkItems().(ports.ContractCandidateRepository)
		if !ok {
			return result, 0, d.NewError(d.ErrorCodeInvalidConfig, "adapter lacks bounded signed candidates")
		}
		candidates, e := indexed.ContractCandidates(ctx, cmd.Scope, query)
		if e != nil {
			return result, 0, e
		}
		result.SearchComplete = len(candidates) <= limit
		if len(candidates) > limit {
			candidates = candidates[:limit]
			last, _ := json.Marshal(candidates[len(candidates)-1])
			result.NextCursor = encodeCursor(queryCursor{Namespace: cmd.Scope.NamespaceID, Outcome: cmd.Scope.OutcomeID, Section: "signed_contract_candidates", Filter: fingerprint, Key: string(last)})
		}
		reviews, contracts, e := signedRepository(u)
		if e != nil {
			return result, 0, e
		}
		now, e := securityTransactionTime(ctx, u, s.clock)
		if e != nil {
			return result, 0, e
		}
		for _, candidate := range candidates {
			if len(cmd.WorkItemIDs) > 0 {
				allowed := false
				for _, id := range cmd.WorkItemIDs {
					allowed = allowed || id == candidate.ID
				}
				if !allowed {
					continue
				}
			}
			if len(cmd.Priorities) > 0 {
				allowed := false
				for _, p := range cmd.Priorities {
					allowed = allowed || p == candidate.Priority
				}
				if !allowed {
					continue
				}
			}
			work, e := u.WorkItems().Get(ctx, cmd.Scope, candidate.ID)
			if e != nil {
				return result, 0, e
			}
			if !work.ContractsEnabled || work.CorrectionReviewCaseID != nil {
				continue
			}
			if cmd.ObjectiveID != nil && (work.ObjectiveID == nil || *work.ObjectiveID != *cmd.ObjectiveID) {
				continue
			}
			pending, e := reviews.OpenCase(ctx, cmd.Scope, work.ID)
			if e != nil {
				return result, 0, e
			}
			if pending != nil {
				continue
			}
			if work.CurrentContractID != nil {
				old, e := contracts.Get(ctx, cmd.Scope, *work.CurrentContractID)
				if code, _ := d.ErrorCodeOf(e); code == d.ErrorCodeNotFound {
					continue
				} // Unsigned authority requires explicit reconciliation.
				if e != nil {
					return result, 0, e
				}
				if old.ValidAt(now) || old.Status == d.ContractDelivered {
					continue
				} // Closed review/correction requires explicit acknowledgement.
			}
			view := work
			view.Lifecycle = d.WorkItemLifecycleTodo
			view.CurrentLease = nil
			ready, e := evaluateWorkItemReadiness(ctx, u, view, now)
			if e != nil {
				return result, 0, e
			}
			if !ready.Ready {
				continue
			}
			acquired, revision, e := s.acquireSignedWork(ctx, u, cc, AcquireSignedWorkContractCommand{Scope: cmd.Scope, WorkItemID: work.ID, ExpectedWorkItemVersion: work.Version, SignerKeyID: cmd.SignerKeyID, TTLSeconds: cmd.TTLSeconds})
			if e != nil {
				return result, 0, e
			}
			result.Acquired = true
			result.Result = &acquired
			return result, revision, nil
		}
		if result.SearchComplete {
			result.Reasons = append(result.Reasons, "no_eligible_work")
		} else {
			result.Reasons = append(result.Reasons, "candidate_search_incomplete")
		}
		return result, coord.Revision, nil
	})
}
