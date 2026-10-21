package application

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (s *Service) FailOutcome(ctx context.Context, commandContext domain.CommandContext, cmd FailOutcomeCommand) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	now := s.clock.Now().UTC()
	conclusionID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	conclusion := conclusionFromContext(conclusionID, commandContext, cmd.Reason, now)
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.Outcome{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := outcome.Fail(conclusion, now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) AbandonOutcome(ctx context.Context, commandContext domain.CommandContext, cmd AbandonOutcomeCommand) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	now := s.clock.Now().UTC()
	conclusionID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	conclusion := conclusionFromContext(conclusionID, commandContext, cmd.Reason, now)
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.Outcome{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := outcome.Abandon(conclusion, now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) ReopenOutcome(ctx context.Context, commandContext domain.CommandContext, cmd ReopenOutcomeCommand) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.Outcome{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := outcome.Reopen(cmd.Reason, now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) ArchiveOutcome(ctx context.Context, commandContext domain.CommandContext, cmd ArchiveOutcomeCommand) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.Outcome{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := outcome.Archive(now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) UnarchiveOutcome(ctx context.Context, commandContext domain.CommandContext, cmd UnarchiveOutcomeCommand) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.Outcome{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := outcome.Unarchive(cmd.Reason, now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) CancelObjective(ctx context.Context, commandContext domain.CommandContext, cmd CancelObjectiveCommand) (MutationResult[domain.Objective], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	now := s.clock.Now().UTC()
	conclusionID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	conclusion := conclusionFromContext(conclusionID, commandContext, cmd.Reason, now)
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Objective, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Objective{}, 0, err
		}
		objective, err := uow.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if err := validateObjectiveCancellationSafe(ctx, uow, objective, now); err != nil {
			return domain.Objective{}, 0, err
		}
		if err := objective.Cancel(conclusion, now); err != nil {
			return domain.Objective{}, 0, err
		}
		if err := uow.Objectives().Save(ctx, objective, cmd.ExpectedVersion); err != nil {
			return domain.Objective{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return objective, revision, err
	})
}

func (s *Service) ReopenObjective(ctx context.Context, commandContext domain.CommandContext, cmd ReopenObjectiveCommand) (MutationResult[domain.Objective], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Objective, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Objective{}, 0, err
		}
		objective, err := uow.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if err := objective.Reopen(cmd.Reason, now); err != nil {
			return domain.Objective{}, 0, err
		}
		if err := uow.Objectives().Save(ctx, objective, cmd.ExpectedVersion); err != nil {
			return domain.Objective{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return objective, revision, err
	})
}

func (s *Service) DeferWorkItem(ctx context.Context, commandContext domain.CommandContext, cmd DeferWorkItemCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := item.Defer(now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func (s *Service) ReleaseWorkItem(ctx context.Context, commandContext domain.CommandContext, cmd ReleaseWorkItemCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		now, err := s.transactionTime(ctx, uow)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		authority, err := s.resolveLeaseCredentialAuthority(ctx, uow, commandContext, cmd.Scope, item.CurrentLease, "ReleaseWorkItem", now)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if authority != nil {
			err = item.ReleaseForCredential(commandContext.PrincipalID, authority.holderID, cmd.ClaimID, cmd.FencingToken, now)
		} else {
			err = item.Release(commandContext.PrincipalID, cmd.ClaimID, cmd.FencingToken, now)
		}
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func (s *Service) CancelWorkItem(ctx context.Context, commandContext domain.CommandContext, cmd CancelWorkItemCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	conclusionID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		now, err := s.transactionTime(ctx, uow)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		conclusion := conclusionFromContext(conclusionID, commandContext, cmd.Reason, now)
		if err := item.Cancel(conclusion, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func (s *Service) ReopenWorkItem(ctx context.Context, commandContext domain.CommandContext, cmd ReopenWorkItemCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := validateWorkItemReopenAncestry(ctx, uow, item, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := item.Reopen(cmd.Reason, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

// validateObjectiveCancellationSafe examines the complete, bounded structural
// snapshot under the caller's Outcome guard. Cancellation does not cascade to
// descendants, so active descendant lifecycles or execution/review authority
// must be resolved explicitly first.
func validateObjectiveCancellationSafe(ctx context.Context, uow ports.UnitOfWork, objective domain.Objective, now time.Time) error {
	reads, ok := uow.(ports.ContinuityReadUnitOfWork)
	if !ok {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "bounded Objective structure reads are required before cancellation")
	}
	repo, ok := reads.ContinuityReads().(ports.WorkspaceReadRepository)
	if !ok {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "bounded Objective structure reads are required before cancellation")
	}
	objectives, items, err := repo.WorkspaceStructure(ctx, objective.Scope, domain.DefaultDependencyGraphLimit+1)
	if err != nil {
		return err
	}
	if len(objectives) > domain.DefaultDependencyGraphLimit || len(items) > domain.DefaultDependencyGraphLimit {
		return domain.NewError(domain.ErrorCodeGraphLimitExceeded, "Objective cancellation exceeds the complete structural safety limit")
	}

	byID := make(map[domain.ID]domain.Objective, len(objectives))
	children := make(map[domain.ID][]domain.ID, len(objectives))
	for _, value := range objectives {
		if value.Scope != objective.Scope || value.ID.Validate() != nil {
			return domain.NewError(domain.ErrorCodeInvalidRelation, "Objective cancellation found an invalid structural reference")
		}
		if _, exists := byID[value.ID]; exists {
			return domain.NewError(domain.ErrorCodeInvalidRelation, "Objective cancellation found duplicate structural references")
		}
		byID[value.ID] = value
	}
	if _, exists := byID[objective.ID]; !exists {
		return domain.NewError(domain.ErrorCodeNotFound, "Objective is missing from the bounded structural snapshot")
	}
	for _, value := range objectives {
		if value.ParentObjectiveID == nil {
			continue
		}
		if value.ParentObjectiveID.Validate() != nil {
			return objectiveCancellationStructureError(objective, value.Ref(), "objective_ancestry_corrupt", "Objective cancellation found an invalid parent reference")
		}
		if _, exists := byID[*value.ParentObjectiveID]; !exists {
			return objectiveCancellationStructureError(objective, value.Ref(), "objective_ancestry_missing", "Objective cancellation cannot prove the hierarchy because a parent is missing")
		}
		children[*value.ParentObjectiveID] = append(children[*value.ParentObjectiveID], value.ID)
	}

	// Validate the target's own parent chain before using it as a subtree root.
	seenAncestors := make(map[domain.ID]bool, len(byID))
	for current := objective.ID; ; {
		if seenAncestors[current] {
			return objectiveCancellationStructureError(objective, objective.Ref(), "objective_ancestry_cycle", "Objective cancellation cannot proceed through a cyclic hierarchy")
		}
		seenAncestors[current] = true
		value := byID[current]
		if value.ParentObjectiveID == nil {
			break
		}
		current = *value.ParentObjectiveID
	}

	descendants := map[domain.ID]bool{objective.ID: true}
	queue := []domain.ID{objective.ID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, child := range children[current] {
			if descendants[child] {
				return objectiveCancellationStructureError(objective, byID[child].Ref(), "objective_ancestry_cycle", "Objective cancellation cannot proceed through a cyclic descendant branch")
			}
			descendants[child] = true
			queue = append(queue, child)
		}
	}

	blockers := make([]domain.EntityRef, 0)
	remedies := make([]domain.ErrorRemedy, 0)
	for _, value := range objectives {
		if value.ID == objective.ID || !descendants[value.ID] || value.Lifecycle != domain.ObjectiveLifecycleInProgress {
			continue
		}
		ref := value.Ref()
		blockers = append(blockers, ref)
		remedies = append(remedies, domain.ErrorRemedy{ActionID: "inspect", SubjectRef: &ref})
	}
	for _, item := range items {
		activeLease := item.CurrentLease != nil && item.CurrentLease.ValidAt(now)
		if item.Scope != objective.Scope || item.ID.Validate() != nil {
			return domain.NewError(domain.ErrorCodeInvalidRelation, "Objective cancellation found an invalid Task reference")
		}
		if item.ObjectiveID == nil {
			continue // legacy Outcome-level work is not a descendant of this Objective
		}
		if item.ObjectiveID.Validate() != nil {
			if item.Lifecycle == domain.WorkItemLifecycleInProgress || activeLease || item.CurrentContractID != nil || item.PendingReviewCaseID != nil {
				return objectiveCancellationStructureError(objective, item.Ref(), "objective_ancestry_corrupt", "Objective cancellation cannot classify active work with an invalid Objective reference")
			}
			continue
		}
		if _, exists := byID[*item.ObjectiveID]; !exists {
			if item.Lifecycle == domain.WorkItemLifecycleInProgress || activeLease || item.CurrentContractID != nil || item.PendingReviewCaseID != nil {
				return objectiveCancellationStructureError(objective, item.Ref(), "objective_ancestry_missing", "Objective cancellation cannot classify active work whose Objective is missing")
			}
			continue
		}
		if !descendants[*item.ObjectiveID] || item.Lifecycle != domain.WorkItemLifecycleInProgress && !activeLease && item.CurrentContractID == nil && item.PendingReviewCaseID == nil {
			continue
		}
		ref := item.Ref()
		blockers = append(blockers, ref)
		action := "inspect"
		if activeLease {
			action = "ReleaseWorkItem"
		} else if item.CurrentContractID != nil {
			action = "RevokeWorkContract"
		} else if item.Lifecycle == domain.WorkItemLifecycleInProgress {
			action = "AdministrativeCancelWorkItem"
		}
		remedies = append(remedies, domain.ErrorRemedy{ActionID: action, SubjectRef: &ref})
	}
	if len(blockers) == 0 {
		return nil
	}
	sort.Slice(blockers, func(i, j int) bool {
		return blockers[i].Kind.String()+":"+blockers[i].ID.String() < blockers[j].Kind.String()+":"+blockers[j].ID.String()
	})
	sort.Slice(remedies, func(i, j int) bool {
		left, right := "", ""
		if remedies[i].SubjectRef != nil {
			left = remedies[i].SubjectRef.Kind.String() + ":" + remedies[i].SubjectRef.ID.String()
		}
		if remedies[j].SubjectRef != nil {
			right = remedies[j].SubjectRef.Kind.String() + ":" + remedies[j].SubjectRef.ID.String()
		}
		return left < right
	})
	ref := objective.Ref()
	return domain.NewDetailedError(domain.ErrorCodePreconditionFailed,
		"resolve descendant execution, lease, contract, or review authority before cancelling this Objective",
		domain.ErrorDetails{ReasonCode: "objective_descendant_active", SubjectRef: &ref, RelatedRefs: blockers, Remedies: remedies})
}

func objectiveCancellationStructureError(objective domain.Objective, related domain.EntityRef, reason, message string) error {
	subject := objective.Ref()
	return domain.NewDetailedError(domain.ErrorCodePreconditionFailed, message, domain.ErrorDetails{
		ReasonCode: reason, SubjectRef: &subject, RelatedRefs: []domain.EntityRef{related},
		Remedies: []domain.ErrorRemedy{{ActionID: "inspect", SubjectRef: &related}},
	})
}

// Reopening a Task makes it executable To do work. operational_v2 therefore
// requires its owner and every ancestor to be active before that transition;
// compatibility_v1 retains the historical reopen behavior.
func validateWorkItemReopenAncestry(ctx context.Context, uow ports.UnitOfWork, item domain.WorkItem, now time.Time) error {
	outcome, err := uow.Outcomes().Get(ctx, item.Scope.NamespaceID, item.Scope.OutcomeID)
	if err != nil {
		return err
	}
	if err := validateOperationalProfile(outcome); err != nil {
		return err
	}
	if outcome.OperationalProfile.Effective() != domain.OperationalProfileOperationalV2 {
		return nil
	}
	readiness := domain.NewReadiness(item.Ref(), now)
	if outcome.Lifecycle != domain.OutcomeLifecycleActive || outcome.IsArchived() {
		addOutcomeEligibility(&readiness, outcome)
	}
	if item.ObjectiveID == nil {
		readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonTaskObjectiveRequired)
		return workItemReopenReadinessError(readiness)
	}
	objective, err := uow.Objectives().Get(ctx, item.Scope, *item.ObjectiveID)
	if err != nil {
		if code, _ := domain.ErrorCodeOf(err); code == domain.ErrorCodeNotFound {
			readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonObjectiveAncestryMissing)
			return workItemReopenReadinessError(readiness)
		}
		return err
	}
	if objective.Scope != item.Scope || objective.Validate() != nil {
		readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonObjectiveAncestryCorrupt)
		return workItemReopenReadinessError(readiness)
	}
	addObjectiveLifecycleReadiness(&readiness, objective, false)
	ancestors, issue, err := loadObjectiveAncestors(ctx, uow, item.Scope, objective)
	if err != nil {
		return err
	}
	if issue != "" {
		readiness.AddReason(domain.ReadinessStageScope, issue)
	}
	for _, ancestor := range ancestors {
		addObjectiveLifecycleReadiness(&readiness, ancestor, true)
	}
	if readiness.Ready {
		return nil
	}
	return workItemReopenReadinessError(readiness)
}

func addObjectiveLifecycleReadiness(readiness *domain.Readiness, objective domain.Objective, ancestor bool) {
	if objective.Lifecycle == domain.ObjectiveLifecycleInProgress {
		return
	}
	if ancestor {
		if objective.Lifecycle == domain.ObjectiveLifecycleAchieved || objective.Lifecycle == domain.ObjectiveLifecycleCancelled {
			readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonAncestorObjectiveTerminal, objective.Ref())
		} else {
			readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonAncestorObjectiveNotStarted, objective.Ref())
		}
		return
	}
	if objective.Lifecycle == domain.ObjectiveLifecycleAchieved || objective.Lifecycle == domain.ObjectiveLifecycleCancelled {
		readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonObjectiveTerminal, objective.Ref())
	} else {
		readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonObjectiveNotStarted, objective.Ref())
	}
}

func workItemReopenReadinessError(readiness domain.Readiness) error {
	err := operationalReadinessError(readiness, "Task cannot be reopened until its Objective and ancestors are in progress")
	var typed *domain.Error
	if len(readiness.Reasons) > 0 && errors.As(err, &typed) && typed != nil && typed.Details != nil {
		switch readiness.Reasons[0] {
		case domain.ReadinessReasonObjectiveTerminal, domain.ReadinessReasonAncestorObjectiveTerminal:
			for _, ref := range readiness.RelatedRefs {
				if ref.Kind == domain.EntityKindObjective {
					typed.Details.Remedies = []domain.ErrorRemedy{{ActionID: "ReopenObjective", SubjectRef: &ref}}
					break
				}
			}
		}
	}
	return err
}

func (s *Service) RetireCriterion(ctx context.Context, commandContext domain.CommandContext, cmd RetireCriterionCommand) (MutationResult[domain.SuccessCriterion], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.SuccessCriterion]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.SuccessCriterion, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Owner.Scope); err != nil {
			return domain.SuccessCriterion{}, 0, err
		}
		criterion, err := retireCriterion(ctx, uow, cmd.Owner, cmd.CriterionID, cmd.ExpectedVersion, now)
		if err != nil {
			return domain.SuccessCriterion{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Owner.Scope)
		return criterion, revision, err
	})
}

func (s *Service) SetOutcomeOwners(ctx context.Context, commandContext domain.CommandContext, cmd SetOutcomeOwnersCommand) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		if outcome.Version != cmd.ExpectedVersion {
			return domain.Outcome{}, 0, domain.NewError(domain.ErrorCodeVersionConflict, "outcome expected_version is stale")
		}
		if actorRefsEqual(outcome.OwnerRefs, cmd.OwnerRefs) {
			return outcome, coordination.Revision, nil
		}
		if err := outcome.SetOwners(cmd.OwnerRefs, now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) SetObjectiveOwners(ctx context.Context, commandContext domain.CommandContext, cmd SetObjectiveOwnersCommand) (MutationResult[domain.Objective], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Objective, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Objective{}, 0, err
		}
		objective, err := uow.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if objective.Version != cmd.ExpectedVersion {
			return domain.Objective{}, 0, domain.NewError(domain.ErrorCodeVersionConflict, "objective expected_version is stale")
		}
		if actorRefsEqual(objective.OwnerRefs, cmd.OwnerRefs) {
			coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
			if err != nil {
				return domain.Objective{}, 0, err
			}
			return objective, coordination.Revision, nil
		}
		if err := objective.SetOwners(cmd.OwnerRefs, now); err != nil {
			return domain.Objective{}, 0, err
		}
		if err := uow.Objectives().Save(ctx, objective, cmd.ExpectedVersion); err != nil {
			return domain.Objective{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return objective, revision, err
	})
}

func (s *Service) SetWorkItemAssignees(ctx context.Context, commandContext domain.CommandContext, cmd SetWorkItemAssigneesCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if item.Version != cmd.ExpectedVersion {
			return domain.WorkItem{}, 0, domain.NewError(domain.ErrorCodeVersionConflict, "work item expected_version is stale")
		}
		if actorRefsEqual(item.AssigneeRefs, cmd.AssigneeRefs) {
			coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
			if err != nil {
				return domain.WorkItem{}, 0, err
			}
			return item, coordination.Revision, nil
		}
		if err := item.SetAssignees(cmd.AssigneeRefs, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func retireCriterion(ctx context.Context, uow ports.UnitOfWork, owner domain.EntityRef, criterionID domain.ID, expected domain.Version, now time.Time) (domain.SuccessCriterion, error) {
	switch owner.Kind {
	case domain.EntityKindOutcome:
		value, err := uow.Outcomes().Get(ctx, owner.Scope.NamespaceID, owner.Scope.OutcomeID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := value.RetireCriterion(criterionID, now); err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := uow.Outcomes().Save(ctx, value, expected); err != nil {
			return domain.SuccessCriterion{}, err
		}
		c, err := value.Criteria.Find(criterionID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		return *c, nil
	case domain.EntityKindObjective:
		value, err := uow.Objectives().Get(ctx, owner.Scope, owner.ID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := value.RetireCriterion(criterionID, now); err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := uow.Objectives().Save(ctx, value, expected); err != nil {
			return domain.SuccessCriterion{}, err
		}
		c, err := value.Criteria.Find(criterionID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		return *c, nil
	case domain.EntityKindWorkItem:
		value, err := uow.WorkItems().Get(ctx, owner.Scope, owner.ID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := value.RetireCriterion(criterionID, now); err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := uow.WorkItems().Save(ctx, value, expected); err != nil {
			return domain.SuccessCriterion{}, err
		}
		c, err := value.Criteria.Find(criterionID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		return *c, nil
	default:
		return domain.SuccessCriterion{}, domain.NewError(domain.ErrorCodeInvalidArgument, "owner kind does not support criteria")
	}
}

func requireOpenOutcome(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope) error {
	if _, err := uow.Coordination().LockOutcome(ctx, scope); err != nil {
		return err
	}
	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return err
	}
	return validateOpenOutcome(outcome)
}

// validateOpenOutcome is the shared state predicate for mutations and pinned
// read previews. Commands still acquire their guard and load canonical state.
func validateOpenOutcome(outcome domain.Outcome) error {
	if outcome.IsArchived() {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "outcome must not be archived")
	}
	switch outcome.Lifecycle {
	case domain.OutcomeLifecycleAchieved, domain.OutcomeLifecycleFailed, domain.OutcomeLifecycleAbandoned:
		return domain.NewError(domain.ErrorCodePreconditionFailed, "outcome must not be terminal")
	default:
		return nil
	}
}
