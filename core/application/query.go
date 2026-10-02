package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

type OutcomeState struct {
	Outcome         domain.Outcome         `json:"outcome"`
	Objectives      []domain.Objective     `json:"objectives"`
	WorkItems       []domain.WorkItem      `json:"work_items"`
	Relations       []domain.Relation      `json:"relations"`
	Issues          []domain.Issue         `json:"issues"`
	Blockers        []domain.Blocker       `json:"blockers"`
	BlockingStates  []BlockingState        `json:"blocking_states"`
	OutcomeRevision domain.OutcomeRevision `json:"outcome_revision"`
}

type ReadResult[T any] struct {
	Value           T                      `json:"value"`
	OutcomeRevision domain.OutcomeRevision `json:"outcome_revision"`
}

func (s *Service) GetOutcome(ctx context.Context, scope domain.Scope) (ReadResult[domain.Outcome], error) {
	if err := scope.Validate(); err != nil {
		return ReadResult[domain.Outcome]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Outcome]{}, err
	}
	defer uow.Rollback()

	value, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return ReadResult[domain.Outcome]{}, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[domain.Outcome]{}, err
	}
	return ReadResult[domain.Outcome]{
		Value:           value,
		OutcomeRevision: coordination.Revision,
	}, nil
}

func (s *Service) GetObjective(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.Objective], error) {
	if err := scope.Validate(); err != nil {
		return ReadResult[domain.Objective]{}, err
	}
	if err := id.Validate(); err != nil {
		return ReadResult[domain.Objective]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Objective]{}, err
	}
	defer uow.Rollback()

	value, err := uow.Objectives().Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.Objective]{}, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[domain.Objective]{}, err
	}
	return ReadResult[domain.Objective]{
		Value:           value,
		OutcomeRevision: coordination.Revision,
	}, nil
}

func (s *Service) GetWorkItem(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.WorkItem], error) {
	if err := scope.Validate(); err != nil {
		return ReadResult[domain.WorkItem]{}, err
	}
	if err := id.Validate(); err != nil {
		return ReadResult[domain.WorkItem]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.WorkItem]{}, err
	}
	defer uow.Rollback()

	value, err := uow.WorkItems().Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.WorkItem]{}, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[domain.WorkItem]{}, err
	}
	return ReadResult[domain.WorkItem]{
		Value:           value,
		OutcomeRevision: coordination.Revision,
	}, nil
}

func (s *Service) GetRelation(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.Relation], error) {
	if err := scope.Validate(); err != nil {
		return ReadResult[domain.Relation]{}, err
	}
	if err := id.Validate(); err != nil {
		return ReadResult[domain.Relation]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Relation]{}, err
	}
	defer uow.Rollback()

	value, err := uow.Relations().Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.Relation]{}, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[domain.Relation]{}, err
	}
	return ReadResult[domain.Relation]{
		Value:           value,
		OutcomeRevision: coordination.Revision,
	}, nil
}

func (s *Service) ListRelations(ctx context.Context, scope domain.Scope) ([]domain.Relation, domain.OutcomeRevision, error) {
	if err := scope.Validate(); err != nil {
		return nil, 0, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()

	values, err := uow.Relations().ListByOutcome(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	return values, coordination.Revision, nil
}

func (s *Service) GetOutcomeState(ctx context.Context, scope domain.Scope) (OutcomeState, error) {
	if err := scope.Validate(); err != nil {
		return OutcomeState{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return OutcomeState{}, err
	}
	defer uow.Rollback()

	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return OutcomeState{}, err
	}
	objectives, err := uow.Objectives().ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	workItems, err := uow.WorkItems().ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	relations, err := uow.Relations().ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	issuesRepo, blockersRepo, err := issueBlockerRepositories(uow)
	if err != nil {
		return OutcomeState{}, err
	}
	issues, err := issuesRepo.ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	blockers, err := blockersRepo.ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}

	blockingStates := make([]BlockingState, 0, 1+len(objectives)+len(workItems))
	refs := make([]domain.EntityRef, 0, 1+len(objectives)+len(workItems))
	refs = append(refs, outcome.Ref())
	for _, objective := range objectives {
		refs = append(refs, objective.Ref())
	}
	for _, workItem := range workItems {
		refs = append(refs, workItem.Ref())
	}
	for _, ref := range refs {
		state, err := blockingStateForRef(ctx, uow, ref)
		if err != nil {
			return OutcomeState{}, err
		}
		blockingStates = append(blockingStates, state)
	}

	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	return OutcomeState{
		Outcome:         outcome,
		Objectives:      objectives,
		WorkItems:       workItems,
		Relations:       relations,
		Issues:          issues,
		Blockers:        blockers,
		BlockingStates:  blockingStates,
		OutcomeRevision: coordination.Revision,
	}, nil
}
