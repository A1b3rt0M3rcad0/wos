package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// RelateIssueWorkItemCommand records an informational link between an Issue
// and a corrective Task. It does not create a dependency or change readiness.
type RelateIssueWorkItemCommand struct {
	Scope                   domain.Scope   `json:"scope"`
	IssueID                 domain.ID      `json:"issue_id"`
	ExpectedIssueVersion    domain.Version `json:"expected_issue_version,string"`
	WorkItemID              domain.ID      `json:"work_item_id"`
	ExpectedWorkItemVersion domain.Version `json:"expected_work_item_version,string"`
}

type RemoveIssueWorkItemRelationCommand struct {
	Scope           domain.Scope   `json:"scope"`
	RelationID      domain.ID      `json:"relation_id"`
	ExpectedVersion domain.Version `json:"expected_version,string"`
	Reason          string         `json:"reason"`
}

func (s *Service) RelateIssueToWorkItem(ctx context.Context, cc domain.CommandContext, cmd RelateIssueWorkItemCommand) (MutationResult[domain.Relation], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	if err := cmd.Scope.Validate(); err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	if err := cmd.IssueID.Validate(); err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	if err := cmd.WorkItemID.Validate(); err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	if cmd.ExpectedIssueVersion == 0 || cmd.ExpectedWorkItemVersion == 0 {
		return MutationResult[domain.Relation]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "expected Issue and Task versions are required")
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Relation, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Relation{}, 0, err
		}
		issues, _, err := issueBlockerRepositories(uow)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		issue, err := issues.Get(ctx, cmd.Scope, cmd.IssueID)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		if issue.Version != cmd.ExpectedIssueVersion {
			return domain.Relation{}, 0, domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "Issue version conflict", domain.ErrorDetails{ReasonCode: "issue_version_conflict", SubjectRef: refPointer(issue.Ref())})
		}
		work, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		if work.Version != cmd.ExpectedWorkItemVersion {
			return domain.Relation{}, 0, domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "Task version conflict", domain.ErrorDetails{ReasonCode: "task_version_conflict", SubjectRef: refPointer(work.Ref())})
		}
		relations, err := uow.Relations().ListByOutcome(ctx, cmd.Scope)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		for _, relation := range relations {
			if relation.Lifecycle == domain.RelationLifecycleActive && relation.RelationType == domain.RelationTypeRelatesTo && sameRelationEndpoints(relation, issue.Ref(), work.Ref()) {
				return domain.Relation{}, 0, domain.NewError(domain.ErrorCodeInvalidRelation, "Issue and Task are already related")
			}
		}
		at, err := s.transactionTime(ctx, uow)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		relation, err := s.insertContextualRelation(ctx, uow, cmd.Scope, issue.Ref(), work.Ref(), domain.RelationTypeRelatesTo, at)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return relation, revision, err
	})
}

func (s *Service) RemoveIssueWorkItemRelation(ctx context.Context, cc domain.CommandContext, cmd RemoveIssueWorkItemRelationCommand) (MutationResult[domain.Relation], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	if err := cmd.Scope.Validate(); err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	if err := cmd.RelationID.Validate(); err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	if cmd.ExpectedVersion == 0 {
		return MutationResult[domain.Relation]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "expected relation version is required")
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Relation, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Relation{}, 0, err
		}
		relation, err := uow.Relations().Get(ctx, cmd.Scope, cmd.RelationID)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		if relation.RelationType != domain.RelationTypeRelatesTo || !issueTaskRelation(relation) {
			return domain.Relation{}, 0, domain.NewError(domain.ErrorCodeInvalidRelation, "relation is not an Issue↔Task association")
		}
		if err := relation.Remove(cmd.Reason, s.clock.Now().UTC()); err != nil {
			return domain.Relation{}, 0, err
		}
		if err := uow.Relations().Save(ctx, relation, cmd.ExpectedVersion); err != nil {
			return domain.Relation{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return relation, revision, err
	})
}

func sameRelationEndpoints(relation domain.Relation, first, second domain.EntityRef) bool {
	return relation.SourceRef == first && relation.TargetRef == second || relation.SourceRef == second && relation.TargetRef == first
}

func issueTaskRelation(relation domain.Relation) bool {
	return relation.SourceRef.Kind == domain.EntityKindIssue && relation.TargetRef.Kind == domain.EntityKindWorkItem ||
		relation.SourceRef.Kind == domain.EntityKindWorkItem && relation.TargetRef.Kind == domain.EntityKindIssue
}
