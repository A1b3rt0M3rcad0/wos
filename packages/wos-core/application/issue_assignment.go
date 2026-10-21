package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type AssignIssueCommand struct {
	Scope           domain.Scope
	IssueID         domain.ID
	ExpectedVersion domain.Version
	AssigneeRef     domain.ActorRef
}

type UnassignIssueCommand struct {
	Scope           domain.Scope
	IssueID         domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

type StartIssueInvestigationCommand struct {
	Scope           domain.Scope
	IssueID         domain.ID
	ExpectedVersion domain.Version
}

// issueAssignmentResult keeps the prior assignee observed inside the command's
// transaction available to the event pipeline. The public application result
// remains MutationResult[domain.Issue].
type issueAssignmentResult struct {
	Issue               domain.Issue     `json:"issue"`
	PreviousAssigneeRef *domain.ActorRef `json:"previous_assignee_ref"`
	AssigneeRef         *domain.ActorRef `json:"assignee_ref"`
	Reason              string           `json:"reason,omitempty"`
}

func (s *Service) AssignIssue(ctx context.Context, cc domain.CommandContext, cmd AssignIssueCommand) (MutationResult[domain.Issue], error) {
	result, err := s.mutateIssueAssignment(ctx, cc, cmd, cmd.Scope, cmd.IssueID, cmd.ExpectedVersion, "", func(issue *domain.Issue) error {
		return issue.Assign(cmd.AssigneeRef, s.clock.Now().UTC())
	})
	return publicIssueAssignmentResult(result), err
}

func (s *Service) UnassignIssue(ctx context.Context, cc domain.CommandContext, cmd UnassignIssueCommand) (MutationResult[domain.Issue], error) {
	result, err := s.mutateIssueAssignment(ctx, cc, cmd, cmd.Scope, cmd.IssueID, cmd.ExpectedVersion, cmd.Reason, func(issue *domain.Issue) error {
		return issue.Unassign(cmd.Reason, s.clock.Now().UTC())
	})
	return publicIssueAssignmentResult(result), err
}

// StartIssueInvestigation assigns the Issue to the authenticated command actor
// and moves it into investigation as one single-version, single-revision
// operation. The actor comes only from trusted adapter identity, never from a
// request-selected assignee.
func (s *Service) StartIssueInvestigation(ctx context.Context, cc domain.CommandContext, cmd StartIssueInvestigationCommand) (MutationResult[domain.Issue], error) {
	result, err := s.mutateIssueAssignment(ctx, cc, cmd, cmd.Scope, cmd.IssueID, cmd.ExpectedVersion, "", func(issue *domain.Issue) error {
		return issue.StartInvestigationWithAssignee(cc.Actor, s.clock.Now().UTC())
	})
	return publicIssueAssignmentResult(result), err
}

func (s *Service) mutateIssueAssignment(
	ctx context.Context,
	cc domain.CommandContext,
	command any,
	scope domain.Scope,
	id domain.ID,
	expected domain.Version,
	reason string,
	mutate func(*domain.Issue) error,
) (MutationResult[issueAssignmentResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[issueAssignmentResult]{}, err
	}
	if err := scope.Validate(); err != nil {
		return MutationResult[issueAssignmentResult]{}, err
	}
	if err := id.Validate(); err != nil {
		return MutationResult[issueAssignmentResult]{}, err
	}
	if err := expected.Validate(); err != nil {
		return MutationResult[issueAssignmentResult]{}, err
	}

	return transactCommand(ctx, s, cc, command, func(uow ports.UnitOfWork) (issueAssignmentResult, domain.OutcomeRevision, error) {
		currentRevision, err := lockExistingOutcomeRevision(ctx, uow, scope)
		if err != nil {
			return issueAssignmentResult{}, 0, err
		}
		issues, _, err := issueBlockerRepositories(uow)
		if err != nil {
			return issueAssignmentResult{}, 0, err
		}
		issue, err := issues.Get(ctx, scope, id)
		if err != nil {
			return issueAssignmentResult{}, 0, err
		}
		// Check expected version before invoking a domain operation. This keeps
		// even same-state assignments from masking stale callers as successful
		// no-ops.
		if issue.Version != expected {
			return issueAssignmentResult{}, 0, domain.NewError(domain.ErrorCodeVersionConflict, "issue version conflict")
		}
		previousAssignee := cloneIssueActorRef(issue.AssigneeRef)
		if err := mutate(&issue); err != nil {
			return issueAssignmentResult{}, 0, err
		}
		result := issueAssignmentResult{
			Issue:               issue,
			PreviousAssigneeRef: previousAssignee,
			AssigneeRef:         cloneIssueActorRef(issue.AssigneeRef),
			Reason:              reason,
		}
		if issue.Version == expected {
			// Domain Assign/Unassign deliberately preserve state for no-op calls.
			// Do not save or advance the Outcome revision in that case.
			return result, currentRevision, nil
		}
		if err := issues.Save(ctx, issue, expected); err != nil {
			return issueAssignmentResult{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, scope)
		return result, revision, err
	})
}

func publicIssueAssignmentResult(result MutationResult[issueAssignmentResult]) MutationResult[domain.Issue] {
	return MutationResult[domain.Issue]{
		Value:            result.Value.Issue,
		OutcomeRevision:  result.OutcomeRevision,
		CommandID:        result.CommandID,
		IdempotentReplay: result.IdempotentReplay,
	}
}

func cloneIssueActorRef(actor *domain.ActorRef) *domain.ActorRef {
	if actor == nil {
		return nil
	}
	copy := *actor
	return &copy
}
