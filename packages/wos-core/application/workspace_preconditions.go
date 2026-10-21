package application

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strings"
	"time"
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
	return s.checkObjectiveAchievementWithIssueReport(ctx, u, objective, principal, nil, acknowledgements...)
}

func (s *Service) checkObjectiveAchievementWithIssueReport(ctx context.Context, u ports.UnitOfWork, objective domain.Objective, principal string, issues *objectiveIssueReport, acknowledgements ...[]domain.EntityRef) error {
	if err := s.requireIndependentReview(ctx, u, objective.Scope, principal); err != nil {
		return err
	}
	if err := requireNotBlocked(ctx, u, objective.Ref()); err != nil {
		return err
	}
	if err := requireHardDependenciesSatisfied(ctx, u, objective.Ref()); err != nil {
		return err
	}
	issueReport := objectiveIssueReport{}
	if issues != nil && issues.owner == objective.Ref() {
		issueReport = *issues
	} else {
		issueReport = evaluateObjectiveIssues(ctx, u, objective.Ref(), defaultObjectiveCompletionSampleLimit, true)
	}
	if err := issueReport.requireClosed(); err != nil {
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
	coordination, err := u.Coordination().LockOutcome(ctx, objective.Scope)
	if err != nil {
		return err
	}
	at, err := s.transactionTime(ctx, u)
	if err != nil {
		return err
	}
	return s.checkObjectiveCompletionEvaluation(ctx, u, objective, parent, coordination.Revision, at, &issueReport, completionAcknowledgements(acknowledgements))
}

func (s *Service) checkObjectiveAchievementInFrame(ctx context.Context, frame *continuityReadFrame, objective domain.Objective, principal string, issues *objectiveIssueReport, acknowledgements ...[]domain.EntityRef) error {
	if err := s.requireIndependentReview(ctx, frame.u, objective.Scope, principal); err != nil {
		return err
	}
	if err := requireNotBlocked(ctx, frame.u, objective.Ref()); err != nil {
		return err
	}
	if err := requireHardDependenciesSatisfied(ctx, frame.u, objective.Ref()); err != nil {
		return err
	}
	issueReport := objectiveIssueReport{}
	if issues != nil && issues.owner == objective.Ref() {
		issueReport = *issues
	} else {
		issueReport = evaluateObjectiveIssues(ctx, frame.u, objective.Ref(), defaultObjectiveCompletionSampleLimit, true)
	}
	if err := issueReport.requireClosed(); err != nil {
		return err
	}
	if err := objective.CheckAchievement(); err != nil {
		return err
	}
	parent, err := completionOutcome(ctx, frame.u, objective.Scope)
	if err != nil {
		return err
	}
	if err := validateRequiredAssessmentProof(ctx, frame.u, objective.Ref(), objective.Criteria); err != nil {
		return err
	}
	return s.checkObjectiveCompletionEvaluation(ctx, frame.u, objective, parent, frame.revision, frame.at, &issueReport, completionAcknowledgements(acknowledgements))
}

func (s *Service) checkObjectiveCompletionEvaluation(ctx context.Context, u ports.UnitOfWork, objective domain.Objective, parent domain.Outcome, revision domain.OutcomeRevision, at time.Time, issues *objectiveIssueReport, acknowledgements []domain.EntityRef) error {
	// Focal Objective projections intentionally omit append-only assessment and
	// definition history. The completion evaluator validates the canonical
	// aggregate, so resolve it from this same UoW and reject a projection that
	// changed underneath the caller's expected version.
	current, err := u.Objectives().Get(ctx, objective.Scope, objective.ID)
	if err != nil {
		return err
	}
	if current.Scope != objective.Scope || current.ID != objective.ID || current.Version != objective.Version {
		return domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "Objective changed while its completion facts were being evaluated; reload before concluding", domain.ErrorDetails{ReasonCode: "objective_completion_snapshot_mismatch", SubjectRef: refPointer(objective.Ref())})
	}
	objective = current
	evaluation, err := s.evaluateObjectiveCompletionInUnit(ctx, u, objective, revision, at, maxObjectiveCompletionSampleLimit, issues)
	if err != nil {
		return err
	}
	if err := checkStrictCompletionFacts(objective.Ref(), parent.CompletionPolicy, evaluation.StrictOpen, evaluation.StrictCancelled, acknowledgements); err != nil {
		return err
	}
	if !evaluation.Report.FactsComplete {
		reason := "objective_completion_facts_incomplete"
		for _, candidate := range evaluation.Report.EligibilityReasonCodes {
			if strings.Contains(candidate, "incomplete") || strings.Contains(candidate, "unavailable") || strings.Contains(candidate, "invalid") || strings.Contains(candidate, "mismatch") {
				reason = candidate
				break
			}
		}
		return domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "Objective completion facts are incomplete; reload the current Objective state before concluding", domain.ErrorDetails{ReasonCode: reason, SubjectRef: refPointer(objective.Ref())})
	}
	for _, reason := range evaluation.Report.EligibilityReasonCodes {
		if reason == "cancelled_work_requires_ack" {
			// checkStrictCompletionFacts above validates the complete cancellation
			// set, not the bounded public samples.
			continue
		}
		return domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "Objective is not eligible for conclusion", domain.ErrorDetails{ReasonCode: reason, SubjectRef: refPointer(objective.Ref())})
	}
	return nil
}
func refPointer(ref domain.EntityRef) *domain.EntityRef { return &ref }
