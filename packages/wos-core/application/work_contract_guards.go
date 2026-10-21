package application

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"reflect"
	"time"
)

// Guards run after receipt replay and before mutations, under the coordination
// lock. Observations remain writable; signed review obligations reject unsigned assessments.
func protectContractMutation(ctx context.Context, s *Service, u ports.UnitOfWork, name string, command any) error {
	return validateContractMutation(ctx, s, u, name, command, nil)
}

// inspectContractMutation uses the same predicates in an already pinned query
// snapshot. It never substitutes for the mutation pipeline's coordination guard.
func inspectContractMutation(ctx context.Context, s *Service, u ports.UnitOfWork, name string, command any, at time.Time) error {
	return validateContractMutation(ctx, s, u, name, command, &at)
}
func validateContractMutation(ctx context.Context, s *Service, u ports.UnitOfWork, name string, command any, inspectionAt *time.Time) error {
	// Only the inspection entrypoint supplies a time. Commands always acquire
	// their guard and then read the current authoritative clock, as before.
	acquireGuard := inspectionAt == nil
	evaluationTime := func() (time.Time, error) {
		if inspectionAt != nil {
			return inspectionAt.UTC(), nil
		}
		return s.transactionTime(ctx, u)
	}
	var scope domain.Scope
	var workID, objectiveID domain.ID
	var wholeOutcome bool
	var signedOnly bool
	var immutableAssessment bool
	switch cmd := command.(type) {
	case RecordCriterionAssessmentCommand:
		scope = cmd.Owner.Scope
		signedOnly = true
		immutableAssessment = true
		if cmd.Owner.Kind == domain.EntityKindWorkItem {
			workID = cmd.Owner.ID
		} else if cmd.Owner.Kind == domain.EntityKindObjective {
			objectiveID = cmd.Owner.ID
		} else if cmd.Owner.Kind == domain.EntityKindOutcome {
			wholeOutcome = true
		}
	case AttestCriterionCommand:
		scope = cmd.Owner.Scope
		signedOnly = true
		immutableAssessment = true
		if cmd.Owner.Kind == domain.EntityKindWorkItem {
			workID = cmd.Owner.ID
		} else if cmd.Owner.Kind == domain.EntityKindObjective {
			objectiveID = cmd.Owner.ID
		} else if cmd.Owner.Kind == domain.EntityKindOutcome {
			wholeOutcome = true
		}
	case UpdateWorkItemCommand:
		scope, workID = cmd.Scope, cmd.WorkItemID
	case SetWorkItemExecutionSpecCommand:
		scope, workID = cmd.Scope, cmd.WorkItemID
	case SetWorkItemNotBeforeCommand:
		scope, workID = cmd.Scope, cmd.WorkItemID
	case SetWorkItemObjectiveCommand:
		scope, workID = cmd.Scope, cmd.WorkItemID
	case SetObjectiveParentCommand:
		if acquireGuard {
			if _, err := u.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
				return err
			}
		}
		objective, err := u.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return err
		}
		if err := validateStructurePreview(ctx, u, cmd.Scope, objective.Ref(), cmd.ExpectedOutcomeRevision); err != nil {
			return err
		}
		now, err := evaluationTime()
		if err != nil {
			return err
		}
		return validateObjectiveParentChange(ctx, s, u, objective, cmd.ParentObjectiveID, now)
	case UpdateOutcomeCommand:
		scope, wholeOutcome = cmd.Scope, true
		signedOnly = true
	case UpdateObjectiveCommand:
		scope, objectiveID = cmd.Scope, cmd.ObjectiveID
		signedOnly = true
	case AddCriterionCommand:
		scope = cmd.Owner.Scope
		signedOnly = cmd.Owner.Kind != domain.EntityKindWorkItem
		switch cmd.Owner.Kind {
		case domain.EntityKindWorkItem:
			workID = cmd.Owner.ID
		case domain.EntityKindObjective:
			objectiveID = cmd.Owner.ID
		case domain.EntityKindOutcome:
			wholeOutcome = true
		}
	case ReviseCriterionCommand:
		scope = cmd.Owner.Scope
		signedOnly = cmd.Owner.Kind != domain.EntityKindWorkItem
		switch cmd.Owner.Kind {
		case domain.EntityKindWorkItem:
			workID = cmd.Owner.ID
		case domain.EntityKindObjective:
			objectiveID = cmd.Owner.ID
		case domain.EntityKindOutcome:
			wholeOutcome = true
		}
	case RetireCriterionCommand:
		scope = cmd.Owner.Scope
		signedOnly = cmd.Owner.Kind != domain.EntityKindWorkItem
		switch cmd.Owner.Kind {
		case domain.EntityKindWorkItem:
			workID = cmd.Owner.ID
		case domain.EntityKindObjective:
			objectiveID = cmd.Owner.ID
		case domain.EntityKindOutcome:
			wholeOutcome = true
		}
	case AddDependencyCommand:
		scope = cmd.Scope
		signedOnly = cmd.SourceRef.Kind != domain.EntityKindWorkItem
		switch cmd.SourceRef.Kind {
		case domain.EntityKindWorkItem:
			workID = cmd.SourceRef.ID
		case domain.EntityKindObjective:
			objectiveID = cmd.SourceRef.ID
		case domain.EntityKindOutcome:
			wholeOutcome = true
		}
	case ApplyDependencyChangesCommand:
		return validateDependencyChangesContract(ctx, s, u, cmd, inspectionAt)
	case RemoveDependencyCommand:
		if acquireGuard {
			if _, err := u.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
				return err
			}
		}
		rel, err := u.Relations().Get(ctx, cmd.Scope, cmd.RelationID)
		if err != nil {
			return err
		}
		scope = cmd.Scope
		signedOnly = rel.SourceRef.Kind != domain.EntityKindWorkItem
		switch rel.SourceRef.Kind {
		case domain.EntityKindWorkItem:
			workID = rel.SourceRef.ID
		case domain.EntityKindObjective:
			objectiveID = rel.SourceRef.ID
		case domain.EntityKindOutcome:
			wholeOutcome = true
		}
	case AchieveOutcomeCommand:
		scope, wholeOutcome = cmd.Scope, true
	case AbandonOutcomeCommand:
		scope, wholeOutcome = cmd.Scope, true
	case FailOutcomeCommand:
		scope, wholeOutcome = cmd.Scope, true
	case ArchiveOutcomeCommand:
		scope, wholeOutcome = cmd.Scope, true
	case AchieveObjectiveCommand:
		scope, objectiveID = cmd.Scope, cmd.ObjectiveID
	case CancelObjectiveCommand:
		scope, objectiveID = cmd.Scope, cmd.ObjectiveID
	default:
		// All work setters and lifecycle transitions share Scope/WorkItemID.
		switch name {
		case "CompleteWorkItem", "AdministrativeCompleteWorkItem", "SetWorkItemObjective", "SetWorkItemNotBefore", "SetWorkItemAssignees", "CancelWorkItem", "AdministrativeCancelWorkItem", "DeferWorkItem", "ReopenWorkItem":
			v := reflect.ValueOf(command)
			if x := v.FieldByName("Scope"); x.IsValid() {
				scope = x.Interface().(domain.Scope)
			}
			if x := v.FieldByName("WorkItemID"); x.IsValid() {
				workID = x.Interface().(domain.ID)
			}
		}
	}
	if scope.OutcomeID.IsZero() {
		return nil
	}
	if acquireGuard {
		if _, err := u.Coordination().LockOutcome(ctx, scope); err != nil {
			return err
		}
	}
	now, err := evaluationTime()
	if err != nil {
		return err
	}
	check := func(w domain.WorkItem) error {
		return protectWorkContractMutation(ctx, u, w, now, signedOnly, immutableAssessment)
	}
	if !workID.IsZero() {
		w, err := u.WorkItems().Get(ctx, scope, workID)
		if err != nil {
			return err
		}
		if cmd, ok := command.(SetWorkItemObjectiveCommand); ok {
			if err := validateStructurePreview(ctx, u, cmd.Scope, w.Ref(), cmd.ExpectedOutcomeRevision); err != nil {
				return err
			}
		}
		switch name {
		case "UpdateWorkItem", "SetWorkItemExecutionSpec", "SetWorkItemNotBefore", "SetWorkItemObjective", "AddCriterion", "ReviseCriterion", "RetireCriterion", "AddDependency", "RemoveDependency":
			if err := w.CheckPlanningEdit(); err != nil {
				return err
			}
		}
		return check(w)
	}
	if wholeOutcome || !objectiveID.IsZero() {
		var items []domain.WorkItem
		var err error
		if reads, ok := u.(ports.ContinuityReadUnitOfWork); ok {
			if repo, ok := reads.ContinuityReads().(ports.WorkspaceReadRepository); ok {
				var objective *domain.ID
				if !wholeOutcome {
					objective = &objectiveID
				}
				items, err = repo.WorkspaceGuardWork(ctx, scope, objective)
			} else {
				items, err = u.WorkItems().ListByOutcome(ctx, scope)
			}
		} else {
			items, err = u.WorkItems().ListByOutcome(ctx, scope)
		}
		if err != nil {
			return err
		}
		for _, w := range items {
			if wholeOutcome || (w.ObjectiveID != nil && *w.ObjectiveID == objectiveID) {
				if err = check(w); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Share the actual authority check with capability/read projections. It does
// not revoke, reconcile, mutate, or infer authority from declared assignees.
func protectWorkContractMutation(ctx context.Context, u ports.UnitOfWork, w domain.WorkItem, now time.Time, signedOnly, immutableAssessment bool) error {
	if w.PendingReviewCaseID != nil {
		return definitionPrecondition(domain.ErrorCodeSignedProtocolRequired, "open_review", "open review freezes obligations and requires signed independent decision", w.Ref(), nil)
	}
	if signed, ok := u.(ports.SignedContractUnitOfWork); ok {
		pending, err := signed.SignedContracts().OpenCase(ctx, w.Scope, w.ID)
		if err != nil {
			return err
		}
		if pending != nil {
			return definitionPrecondition(domain.ErrorCodeSignedProtocolRequired, "open_review", "open review freezes obligations and requires signed independent decision", w.Ref(), nil)
		}
		if w.CurrentContractID != nil {
			c, err := signed.SignedWorkContracts().Get(ctx, w.Scope, *w.CurrentContractID)
			if err == nil {
				if c.ValidAt(now) || immutableAssessment && c.LatestSubmissionID != nil {
					return definitionPrecondition(domain.ErrorCodeSignedProtocolRequired, "definition_frozen", "active signed authority requires a signed operation", w.Ref(), nil)
				}
				return nil
			}
			if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeNotFound {
				return err
			}
		}
	}
	if signedOnly || w.CurrentContractID == nil {
		return nil
	}
	repo, err := contractRepository(u)
	if err != nil {
		if _, ok := u.(ports.WorkContractUnitOfWork); !ok {
			return nil
		}
		return err
	}
	c, err := repo.Get(ctx, w.Scope, *w.CurrentContractID)
	if err != nil {
		return err
	}
	if c.ValidAt(now) {
		return definitionPrecondition(domain.ErrorCodeWorkAlreadyClaimed, "definition_frozen", "explicitly revoke the valid contract before changing its obligations or lifecycle", w.Ref(), nil)
	}
	return nil
}
