package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type AcquireNextWorkContractCommand struct {
	Scope       d.Scope
	WorkItemIDs []d.ID `wos:"optional"`
	ObjectiveID *d.ID
	Priorities  []d.Priority `wos:"optional"`
	TTLSeconds  int          `wos:"optional"`
	Limit       int          `wos:"optional"`
	Cursor      string       `wos:"optional"`
}
type WorkContractAcquisition struct {
	Acquired       bool                `json:"acquired"`
	Result         *WorkContractResult `json:"result,omitempty"`
	SearchComplete bool                `json:"search_complete"`
	NextCursor     string              `json:"next_cursor,omitempty"`
	Reasons        []string            `json:"reasons"`
}

func (s *Service) AcquireNextWorkContract(ctx context.Context, cc d.CommandContext, cmd AcquireNextWorkContractCommand) (MutationResult[WorkContractAcquisition], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[WorkContractAcquisition]{}, err
	}
	if len(cmd.WorkItemIDs) > 100 {
		return MutationResult[WorkContractAcquisition]{}, d.NewError(d.ErrorCodeInvalidArgument, "candidate allowlist exceeds 100")
	}
	for _, id := range cmd.WorkItemIDs {
		if err := id.Validate(); err != nil {
			return MutationResult[WorkContractAcquisition]{}, err
		}
	}
	for _, p := range cmd.Priorities {
		if !p.Valid() {
			return MutationResult[WorkContractAcquisition]{}, d.NewError(d.ErrorCodeInvalidArgument, "invalid priority")
		}
	}
	limit, err := queryLimit(cmd.Limit)
	if err != nil {
		return MutationResult[WorkContractAcquisition]{}, err
	}
	fingerprint := filterHash(struct {
		IDs        []d.ID
		Objective  *d.ID
		Priorities []d.Priority
	}{cmd.WorkItemIDs, cmd.ObjectiveID, cmd.Priorities})
	q := ports.ContractCandidateQuery{Limit: limit + 1}
	if cmd.Cursor != "" {
		cursor, err := decodeCursor(cmd.Cursor, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID, "contract_candidates", fingerprint)
		if err != nil {
			return MutationResult[WorkContractAcquisition]{}, err
		}
		var last ports.ContractCandidate
		if json.Unmarshal([]byte(cursor.Key), &last) != nil {
			return MutationResult[WorkContractAcquisition]{}, d.NewError(d.ErrorCodeInvalidArgument, "invalid candidate cursor")
		}
		q.AfterID = last.ID
		q.AfterCreated = last.CreatedAt
		q.AfterPriority = ports.ContractPriorityRank(last.Priority)
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (WorkContractAcquisition, d.OutcomeRevision, error) {
		zero := WorkContractAcquisition{}
		if err := requireActiveOutcome(ctx, uow, cmd.Scope); err != nil {
			return zero, 0, err
		}
		indexed, ok := uow.WorkItems().(ports.ContractCandidateRepository)
		if !ok {
			return zero, 0, d.NewError(d.ErrorCodeInvalidConfig, "adapter lacks bounded candidates")
		}
		candidates, err := indexed.ContractCandidates(ctx, cmd.Scope, q)
		if err != nil {
			return zero, 0, err
		}
		result := WorkContractAcquisition{SearchComplete: len(candidates) <= limit, Reasons: []string{}}
		if len(candidates) > limit {
			candidates = candidates[:limit]
			last, _ := json.Marshal(candidates[len(candidates)-1])
			result.NextCursor = encodeCursor(queryCursor{Namespace: cmd.Scope.NamespaceID, Outcome: cmd.Scope.OutcomeID, Section: "contract_candidates", Filter: fingerprint, Key: string(last)})
		}
		repo, err := contractRepository(uow)
		if err != nil {
			return zero, 0, err
		}
		now, err := s.transactionTime(ctx, uow)
		if err != nil {
			return zero, 0, err
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
			w, err := uow.WorkItems().Get(ctx, cmd.Scope, candidate.ID)
			if err != nil {
				return zero, 0, err
			}
			if cmd.ObjectiveID != nil && (w.ObjectiveID == nil || *w.ObjectiveID != *cmd.ObjectiveID) {
				continue
			}
			if w.CurrentContractID != nil {
				c, err := repo.Get(ctx, cmd.Scope, *w.CurrentContractID)
				if err != nil {
					return zero, 0, err
				}
				if c.ValidAt(now) {
					continue
				}
			}
			view := w
			view.Lifecycle = d.WorkItemLifecycleTodo
			view.CurrentLease = nil
			ready, err := evaluateWorkItemReadiness(ctx, uow, view, now)
			if err != nil {
				return zero, 0, err
			}
			if !ready.Ready {
				continue
			}
			acquired, revision, err := s.acquireContract(ctx, uow, cc, AcquireWorkContractCommand{Scope: cmd.Scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version, TTLSeconds: cmd.TTLSeconds})
			if err != nil {
				return zero, 0, err
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
		coord, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
		return result, coord.Revision, err
	})
}

type AvailableWork struct {
	Candidate ports.ContractCandidate `json:"candidate"`
	Title     string                  `json:"title"`
	Recovery  bool                    `json:"recovery"`
}
type AvailableWorkPage struct {
	Items           []AvailableWork   `json:"items"`
	SearchComplete  bool              `json:"search_complete"`
	NextCursor      string            `json:"next_cursor,omitempty"`
	OutcomeRevision d.OutcomeRevision `json:"outcome_revision"`
}

func (s *Service) ListAvailableWork(ctx context.Context, scope d.Scope, limit int, cursor string) (AvailableWorkPage, error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return AvailableWorkPage{}, err
	}
	if err := scope.Validate(); err != nil {
		return AvailableWorkPage{}, err
	}
	limit, err := queryLimit(limit)
	if err != nil {
		return AvailableWorkPage{}, err
	}
	q := ports.ContractCandidateQuery{Limit: limit + 1}
	if cursor != "" {
		c, err := decodeCursor(cursor, scope.NamespaceID, scope.OutcomeID, "available_work", "")
		if err != nil {
			return AvailableWorkPage{}, err
		}
		var last ports.ContractCandidate
		if json.Unmarshal([]byte(c.Key), &last) != nil {
			return AvailableWorkPage{}, d.NewError(d.ErrorCodeInvalidArgument, "invalid available cursor")
		}
		q.AfterID = last.ID
		q.AfterCreated = last.CreatedAt
		q.AfterPriority = ports.ContractPriorityRank(last.Priority)
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return AvailableWorkPage{}, err
	}
	defer uow.Rollback()
	coord, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return AvailableWorkPage{}, err
	}
	indexed, ok := uow.WorkItems().(ports.ContractCandidateRepository)
	if !ok {
		return AvailableWorkPage{}, d.NewError(d.ErrorCodeInvalidConfig, "adapter lacks candidates")
	}
	repo, err := contractRepository(uow)
	if err != nil {
		return AvailableWorkPage{}, err
	}
	candidates, err := indexed.ContractCandidates(ctx, scope, q)
	if err != nil {
		return AvailableWorkPage{}, err
	}
	page := AvailableWorkPage{Items: []AvailableWork{}, SearchComplete: len(candidates) <= limit, OutcomeRevision: coord.Revision}
	if len(candidates) > limit {
		candidates = candidates[:limit]
		last, _ := json.Marshal(candidates[len(candidates)-1])
		page.NextCursor = encodeCursor(queryCursor{Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: "available_work", Key: string(last)})
	}
	now, err := s.transactionTime(ctx, uow)
	if err != nil {
		return AvailableWorkPage{}, err
	}
	for _, candidate := range candidates {
		w, err := uow.WorkItems().Get(ctx, scope, candidate.ID)
		if err != nil {
			return page, err
		}
		if w.CurrentContractID != nil {
			c, err := repo.Get(ctx, scope, *w.CurrentContractID)
			if err != nil {
				return page, err
			}
			if c.ValidAt(now) {
				continue
			}
		}
		view := w
		view.CurrentLease = nil
		view.Lifecycle = d.WorkItemLifecycleTodo
		ready, err := evaluateWorkItemReadiness(ctx, uow, view, now)
		if err != nil {
			return page, err
		}
		if ready.Ready {
			page.Items = append(page.Items, AvailableWork{Candidate: candidate, Title: w.Title, Recovery: w.Lifecycle == d.WorkItemLifecycleInProgress})
		}
	}
	return page, nil
}
