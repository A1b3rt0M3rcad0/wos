package application

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"reflect"
)

// Guards run after receipt replay and before mutations, under the coordination
// lock. Observations remain writable; signed review obligations reject unsigned assessments.
func protectContractMutation(ctx context.Context, s *Service, u ports.UnitOfWork, name string, command any) error {
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
	case RemoveDependencyCommand:
		if _, err := u.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return err
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
	if _, err := u.Coordination().LockOutcome(ctx, scope); err != nil {
		return err
	}
	now, err := s.transactionTime(ctx, u)
	if err != nil {
		return err
	}
	repo, err := contractRepository(u)
	if err != nil { // Legacy adapters retain their existing behavior.
		if _, ok := u.(ports.WorkContractUnitOfWork); !ok {
			return nil
		}
		return err
	}
	check := func(w domain.WorkItem) error {
		if signed, ok := u.(ports.SignedContractUnitOfWork); ok {
			pending, err := signed.SignedContracts().OpenCase(ctx, scope, w.ID)
			if err != nil {
				return err
			}
			if pending != nil {
				return domain.NewError(domain.ErrorCodeSignedProtocolRequired, "open review freezes obligations and requires signed independent decision")
			}
			if w.CurrentContractID != nil {
				c, err := signed.SignedWorkContracts().Get(ctx, scope, *w.CurrentContractID)
				if err == nil {
					if c.ValidAt(now) || immutableAssessment && c.LatestSubmissionID != nil {
						return domain.NewError(domain.ErrorCodeSignedProtocolRequired, "active signed authority requires a signed operation")
					}
					return nil
				}
				if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeNotFound {
					return err
				}
			}
		}
		if signedOnly {
			return nil
		}
		if w.CurrentContractID == nil {
			return nil
		}
		c, err := repo.Get(ctx, scope, *w.CurrentContractID)
		if err != nil {
			return err
		}
		if c.ValidAt(now) {
			return domain.NewError(domain.ErrorCodeWorkAlreadyClaimed, "explicitly revoke the valid contract before changing its obligations or lifecycle")
		}
		return nil
	}
	if !workID.IsZero() {
		w, err := u.WorkItems().Get(ctx, scope, workID)
		if err != nil {
			return err
		}
		return check(w)
	}
	if wholeOutcome || !objectiveID.IsZero() {
		items, err := u.WorkItems().ListByOutcome(ctx, scope)
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
