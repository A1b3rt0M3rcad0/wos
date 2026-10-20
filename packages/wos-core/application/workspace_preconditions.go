package application

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (s *Service) checkOutcomeAchievement(ctx context.Context, u ports.UnitOfWork, outcome domain.Outcome, principal string, acknowledgements ...[]domain.EntityRef) ([]domain.ID, error) {
	if err := s.requireIndependentReview(ctx, u, outcome.Ref().Scope, principal); err != nil {
		return nil, err
	}
	if err := requireNotBlocked(ctx, u, outcome.Ref()); err != nil {
		return nil, err
	}
	var objectives []domain.Objective
	var err error
	if reads, ok := u.(ports.ContinuityReadUnitOfWork); ok {
		if repo, ok := reads.ContinuityReads().(ports.WorkspaceReadRepository); ok {
			objectives, err = repo.WorkspaceRequiredObjectives(ctx, outcome.Ref().Scope)
		} else {
			objectives, err = u.Objectives().ListByOutcome(ctx, outcome.Ref().Scope)
		}
	} else {
		objectives, err = u.Objectives().ListByOutcome(ctx, outcome.Ref().Scope)
	}
	if err != nil {
		return nil, err
	}
	ids := make([]domain.ID, 0)
	for _, objective := range objectives {
		if !objective.RequiredForOutcome {
			continue
		}
		if objective.Lifecycle != domain.ObjectiveLifecycleAchieved {
			ref := outcome.Ref()
			return nil, domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "required objective is not achieved", domain.ErrorDetails{ReasonCode: "required_objective_not_achieved", SubjectRef: &ref, RelatedRefs: []domain.EntityRef{objective.Ref()}, Remedies: []domain.ErrorRemedy{{ActionID: "inspect", SubjectRef: refPointer(objective.Ref())}}})
		}
		ids = append(ids, objective.ID)
	}
	if err := outcome.CheckAchievement(); err != nil {
		return nil, err
	}
	if err := validateRequiredAssessmentProof(ctx, u, outcome.Ref(), outcome.Criteria); err != nil {
		return nil, err
	}
	if err := checkStrictCompletion(ctx, u, outcome.Ref(), outcome.CompletionPolicy, completionAcknowledgements(acknowledgements)); err != nil {
		return nil, err
	}
	return ids, nil
}
func (s *Service) checkObjectiveAchievement(ctx context.Context, u ports.UnitOfWork, objective domain.Objective, principal string, acknowledgements ...[]domain.EntityRef) error {
	if err := s.requireIndependentReview(ctx, u, objective.Scope, principal); err != nil {
		return err
	}
	if err := requireNotBlocked(ctx, u, objective.Ref()); err != nil {
		return err
	}
	if err := requireHardDependenciesSatisfied(ctx, u, objective.Ref()); err != nil {
		return err
	}
	if err := requireObjectiveIssuesClosed(ctx, u, objective.Ref()); err != nil {
		return err
	}
	if err := objective.CheckAchievement(); err != nil {
		return err
	}
	parent, err := completionOutcome(ctx, u, objective.Scope)
	if err != nil {
		return err
	}
	if err := validateRequiredAssessmentProof(ctx, u, objective.Ref(), objective.Criteria); err != nil {
		return err
	}
	if err := checkStrictCompletion(ctx, u, objective.Ref(), parent.CompletionPolicy, completionAcknowledgements(acknowledgements)); err != nil {
		return err
	}
	return nil
}
func refPointer(ref domain.EntityRef) *domain.EntityRef { return &ref }
