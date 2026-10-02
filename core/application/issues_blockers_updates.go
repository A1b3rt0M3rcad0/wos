package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
)

type UpdateIssueCommand struct {
	Scope           domain.Scope
	IssueID         domain.ID
	ExpectedVersion domain.Version
	Title           *string
	Description     *string
	Severity        *domain.IssueSeverity
	AffectedRefs    *[]domain.EntityRef
}

type UpdateBlockerDescriptionCommand struct {
	Scope           domain.Scope
	BlockerID       domain.ID
	ExpectedVersion domain.Version
	Description     string
}

func (s *Service) UpdateIssue(
	ctx context.Context,
	cc domain.CommandContext,
	cmd UpdateIssueCommand,
) (MutationResult[domain.Issue], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Issue]{}, err
	}
	if cmd.Title == nil && cmd.Description == nil && cmd.Severity == nil && cmd.AffectedRefs == nil {
		return MutationResult[domain.Issue]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "issue patch has no editable fields")
	}

	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Issue, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Issue{}, 0, err
		}
		coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
		if err != nil {
			return domain.Issue{}, 0, err
		}
		issues, _, err := issueBlockerRepositories(uow)
		if err != nil {
			return domain.Issue{}, 0, err
		}
		issue, err := issues.Get(ctx, cmd.Scope, cmd.IssueID)
		if err != nil {
			return domain.Issue{}, 0, err
		}
		if issue.Version != cmd.ExpectedVersion {
			return domain.Issue{}, 0, domain.NewError(domain.ErrorCodeVersionConflict, "issue expected_version is stale")
		}
		changed, err := issue.UpdateDetails(cmd.Title, cmd.Description, cmd.Severity, cmd.AffectedRefs, now)
		if err != nil {
			return domain.Issue{}, 0, err
		}
		if !changed {
			return issue, coordination.Revision, nil
		}
		if err := issues.Save(ctx, issue, cmd.ExpectedVersion); err != nil {
			return domain.Issue{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return issue, revision, err
	})
}

func (s *Service) UpdateBlockerDescription(
	ctx context.Context,
	cc domain.CommandContext,
	cmd UpdateBlockerDescriptionCommand,
) (MutationResult[domain.Blocker], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Blocker]{}, err
	}

	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Blocker, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Blocker{}, 0, err
		}
		coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
		if err != nil {
			return domain.Blocker{}, 0, err
		}
		_, blockers, err := issueBlockerRepositories(uow)
		if err != nil {
			return domain.Blocker{}, 0, err
		}
		blocker, err := blockers.Get(ctx, cmd.Scope, cmd.BlockerID)
		if err != nil {
			return domain.Blocker{}, 0, err
		}
		if blocker.Version != cmd.ExpectedVersion {
			return domain.Blocker{}, 0, domain.NewError(domain.ErrorCodeVersionConflict, "blocker expected_version is stale")
		}
		changed, err := blocker.UpdateDescription(cmd.Description, now)
		if err != nil {
			return domain.Blocker{}, 0, err
		}
		if !changed {
			return blocker, coordination.Revision, nil
		}
		if err := blockers.Save(ctx, blocker, cmd.ExpectedVersion); err != nil {
			return domain.Blocker{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return blocker, revision, err
	})
}
