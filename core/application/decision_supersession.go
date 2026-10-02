package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
)

const maxDecisionSupersessionDepth = 10000

type SupersedeDecisionCommand struct {
	Scope             domain.Scope
	DecisionID        domain.ID
	ExpectedVersion   domain.Version
	Title             string
	Proposal          string
	Alternatives      []string
	ChosenAlternative string
	Rationale         string
}

type SupersedeDecisionResult struct {
	Superseded domain.Decision `json:"superseded"`
	Successor  domain.Decision `json:"successor"`
}

func (s *Service) SupersedeDecision(
	ctx context.Context,
	cc domain.CommandContext,
	cmd SupersedeDecisionCommand,
) (MutationResult[SupersedeDecisionResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[SupersedeDecisionResult]{}, err
	}
	successorID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[SupersedeDecisionResult]{}, err
	}
	now := s.clock.Now().UTC()

	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (SupersedeDecisionResult, domain.OutcomeRevision, error) {
		repositories, err := documentaryRepositories(uow)
		if err != nil {
			return SupersedeDecisionResult{}, 0, err
		}
		if err := requireDocumentaryOutcome(ctx, uow, cmd.Scope); err != nil {
			return SupersedeDecisionResult{}, 0, err
		}

		current, err := repositories.decisions.Get(ctx, cmd.Scope, cmd.DecisionID)
		if err != nil {
			return SupersedeDecisionResult{}, 0, err
		}
		if current.Lifecycle != domain.DecisionLifecycleAccepted {
			return SupersedeDecisionResult{}, 0, domain.NewError(
				domain.ErrorCodeInvalidTransition,
				"only an accepted decision can be superseded",
			)
		}
		if err := validateDecisionSupersessionChain(ctx, repositories.decisions, cmd.Scope, current); err != nil {
			return SupersedeDecisionResult{}, 0, err
		}

		successor, err := domain.NewDecision(
			successorID,
			cmd.Scope,
			cmd.Title,
			cmd.Proposal,
			"",
			cmd.Alternatives,
			&current.ID,
			now,
		)
		if err != nil {
			return SupersedeDecisionResult{}, 0, err
		}
		if err := successor.Accept(cmd.ChosenAlternative, cmd.Rationale, cc.Actor, now); err != nil {
			return SupersedeDecisionResult{}, 0, err
		}
		if err := current.MarkSuperseded(now); err != nil {
			return SupersedeDecisionResult{}, 0, err
		}

		if err := repositories.decisions.Insert(ctx, successor); err != nil {
			return SupersedeDecisionResult{}, 0, err
		}
		if err := repositories.decisions.Save(ctx, current, cmd.ExpectedVersion); err != nil {
			return SupersedeDecisionResult{}, 0, err
		}

		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		if err != nil {
			return SupersedeDecisionResult{}, 0, err
		}
		return SupersedeDecisionResult{
			Superseded: current,
			Successor:  successor,
		}, revision, nil
	})
}

func validateDecisionSupersessionChain(
	ctx context.Context,
	repository ports.DecisionRepository,
	scope domain.Scope,
	target domain.Decision,
) error {
	values, err := repository.ListByOutcome(ctx, scope)
	if err != nil {
		return err
	}
	for _, value := range values {
		if value.SupersedesDecisionID != nil && *value.SupersedesDecisionID == target.ID {
			return domain.NewError(
				domain.ErrorCodeDecision,
				"decision already has a direct successor",
			)
		}
	}

	seen := make(map[domain.ID]struct{})
	current := target
	for depth := 0; depth < maxDecisionSupersessionDepth; depth++ {
		if _, exists := seen[current.ID]; exists {
			return domain.NewError(domain.ErrorCodeDecision, "decision supersession chain contains a cycle")
		}
		seen[current.ID] = struct{}{}
		if current.SupersedesDecisionID == nil {
			return nil
		}
		next, err := repository.Get(ctx, scope, *current.SupersedesDecisionID)
		if err != nil {
			return domain.WrapError(domain.ErrorCodeDecision, "decision supersession chain is broken", err)
		}
		current = next
	}
	return domain.NewError(domain.ErrorCodeGraphLimitExceeded, "decision supersession chain validation limit exceeded")
}

func compoundDecisionEvents[T any](
	s *Service,
	cc domain.CommandContext,
	meta commandMetadata,
	value T,
	revision domain.OutcomeRevision,
) ([]domain.DomainEvent, bool, error) {
	result, ok := any(value).(SupersedeDecisionResult)
	if !ok {
		return nil, false, nil
	}
	if revision == 0 {
		return nil, true, domain.NewError(domain.ErrorCodeInvalidEvent, "confirmed decision supersession must have an outcome_revision")
	}
	if result.Successor.Version < 2 {
		return nil, true, domain.NewError(domain.ErrorCodeInvalidEvent, "accepted successor must include proposal and acceptance versions")
	}
	beforeSuperseded, err := previousVersionPtr(result.Superseded.Version)
	if err != nil {
		return nil, true, err
	}
	proposedVersion := domain.InitialVersion
	acceptedBefore := proposedVersion
	specs := []compoundEventSpec{
		{
			eventType: "decision.proposed",
			ref:       result.Successor.Ref(),
			after:     versionPtr(proposedVersion),
		},
		{
			eventType: "decision.accepted",
			ref:       result.Successor.Ref(),
			before:    versionPtr(acceptedBefore),
			after:     versionPtr(result.Successor.Version),
		},
		{
			eventType: "decision.superseded",
			ref:       result.Superseded.Ref(),
			before:    beforeSuperseded,
			after:     versionPtr(result.Superseded.Version),
		},
	}
	events, err := buildCompoundEvents(s, cc, meta, revision, specs)
	return events, true, err
}
