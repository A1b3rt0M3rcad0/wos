package application

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// Required nullable fields are replacement setters: null explicitly clears the
// value. Public codecs reject omission instead of interpreting it as a clear.
type SetWorkItemExecutionSpecCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version        `json:"expected_version,string"`
	ExecutionSpec   *domain.ExecutionSpec `wos:"required"`
}

type SetWorkItemNotBeforeCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version `json:"expected_version,string"`
	NotBefore       *time.Time     `wos:"required"`
}

type SetWorkItemObjectiveCommand struct {
	Scope                   domain.Scope
	WorkItemID              domain.ID
	ExpectedVersion         domain.Version          `json:"expected_version,string"`
	ExpectedOutcomeRevision *domain.OutcomeRevision `json:"expected_outcome_revision,omitempty,string" wos:"optional"`
	ObjectiveID             *domain.ID              `wos:"required"`
}

type SetObjectiveParentCommand struct {
	Scope                   domain.Scope
	ObjectiveID             domain.ID
	ExpectedVersion         domain.Version          `json:"expected_version,string"`
	ExpectedOutcomeRevision *domain.OutcomeRevision `json:"expected_outcome_revision,omitempty,string" wos:"optional"`
	ParentObjectiveID       *domain.ID              `wos:"required"`
}

func (s *Service) SetWorkItemExecutionSpec(ctx context.Context, cc domain.CommandContext, cmd SetWorkItemExecutionSpecCommand) (MutationResult[domain.WorkItem], error) {
	return editWorkDefinition(ctx, s, cc, cmd, cmd.Scope, cmd.WorkItemID, cmd.ExpectedVersion, func(u ports.UnitOfWork, item *domain.WorkItem, now time.Time) (bool, error) {
		if err := validateExecutionSpecRefs(ctx, u, cmd.Scope, cmd.ExecutionSpec); err != nil {
			return false, err
		}
		return item.SetExecutionSpec(cmd.ExecutionSpec, now)
	})
}

func (s *Service) SetWorkItemNotBefore(ctx context.Context, cc domain.CommandContext, cmd SetWorkItemNotBeforeCommand) (MutationResult[domain.WorkItem], error) {
	return editWorkDefinition(ctx, s, cc, cmd, cmd.Scope, cmd.WorkItemID, cmd.ExpectedVersion, func(_ ports.UnitOfWork, item *domain.WorkItem, now time.Time) (bool, error) {
		return item.SetNotBefore(cmd.NotBefore, now)
	})
}

func (s *Service) SetWorkItemObjective(ctx context.Context, cc domain.CommandContext, cmd SetWorkItemObjectiveCommand) (MutationResult[domain.WorkItem], error) {
	return editWorkDefinition(ctx, s, cc, cmd, cmd.Scope, cmd.WorkItemID, cmd.ExpectedVersion, func(u ports.UnitOfWork, item *domain.WorkItem, now time.Time) (bool, error) {
		if err := validateStructurePreview(ctx, u, cmd.Scope, item.Ref(), cmd.ExpectedOutcomeRevision); err != nil {
			return false, err
		}
		if err := validateWorkObjectiveChange(ctx, u, *item, cmd.ObjectiveID); err != nil {
			return false, err
		}
		return item.SetObjective(cmd.ObjectiveID, now)
	})
}

func editWorkDefinition[C any](ctx context.Context, s *Service, cc domain.CommandContext, cmd C, scope domain.Scope, id domain.ID, expected domain.Version, edit func(ports.UnitOfWork, *domain.WorkItem, time.Time) (bool, error)) (MutationResult[domain.WorkItem], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, u, scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		coord, err := u.Coordination().LockOutcome(ctx, scope)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := u.WorkItems().Get(ctx, scope, id)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if item.Version != expected {
			return domain.WorkItem{}, 0, domain.NewError(domain.ErrorCodeVersionConflict, "work item expected_version is stale")
		}
		now, err := s.transactionTime(ctx, u)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err = validateWorkDefinitionChange(ctx, s, u, item, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		changed, err := edit(u, &item, now)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if !changed {
			return item, coord.Revision, nil
		}
		if err := u.WorkItems().Save(ctx, item, expected); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := u.Coordination().AdvanceOutcome(ctx, scope)
		return item, revision, err
	})
}

func (s *Service) SetObjectiveParent(ctx context.Context, cc domain.CommandContext, cmd SetObjectiveParentCommand) (MutationResult[domain.Objective], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (domain.Objective, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, u, cmd.Scope); err != nil {
			return domain.Objective{}, 0, err
		}
		coord, err := u.Coordination().LockOutcome(ctx, cmd.Scope)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		objective, err := u.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if objective.Version != cmd.ExpectedVersion {
			return domain.Objective{}, 0, domain.NewError(domain.ErrorCodeVersionConflict, "objective expected_version is stale")
		}
		if err := validateStructurePreview(ctx, u, cmd.Scope, objective.Ref(), cmd.ExpectedOutcomeRevision); err != nil {
			return domain.Objective{}, 0, err
		}
		now, err := s.transactionTime(ctx, u)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if err := validateObjectiveParentChange(ctx, s, u, objective, cmd.ParentObjectiveID, now); err != nil {
			return domain.Objective{}, 0, err
		}
		changed, err := objective.SetParent(cmd.ParentObjectiveID, now)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if !changed {
			return objective, coord.Revision, nil
		}
		if err := u.Objectives().Save(ctx, objective, cmd.ExpectedVersion); err != nil {
			return domain.Objective{}, 0, err
		}
		revision, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return objective, revision, err
	})
}

func validateStructurePreview(ctx context.Context, u ports.UnitOfWork, scope domain.Scope, subject domain.EntityRef, expected *domain.OutcomeRevision) error {
	if expected == nil {
		return nil
	}
	coord, err := u.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return err
	}
	if coord.Revision != *expected {
		return definitionPrecondition(domain.ErrorCodePreconditionFailed, "structure_preview_changed", "the Outcome changed after the movement preview; review the current placement and impacts before confirming again", subject, []string{"expected_outcome_revision"})
	}
	return nil
}

func definitionPrecondition(code domain.ErrorCode, reason, message string, subject domain.EntityRef, fields []string, related ...domain.EntityRef) error {
	ref := subject
	remedies := make([]domain.ErrorRemedy, 0, len(related))
	for _, target := range related {
		item := target
		remedies = append(remedies, domain.ErrorRemedy{ActionID: "inspect", SubjectRef: &item})
	}
	return domain.NewDetailedError(code, message, domain.ErrorDetails{ReasonCode: reason, SubjectRef: &ref, FieldPaths: fields, RelatedRefs: related, Remedies: remedies})
}

// These predicates are shared with workspace capabilities. They must run in
// the caller's coherent UnitOfWork; a preview is never an authorization token.
func validateWorkDefinitionChange(ctx context.Context, s *Service, u ports.UnitOfWork, item domain.WorkItem, now time.Time) error {
	if err := validateWorkDefinitionLifecycle(item); err != nil {
		return err
	}
	if err := protectWorkContractMutation(ctx, u, item, now, false, false); err != nil {
		return err
	}
	if item.CurrentContractID != nil {
		return definitionPrecondition(domain.ErrorCodeContractProtocolRequired, "definition_frozen", "revoke or reconcile the attached contract before changing its definition", item.Ref(), nil)
	}
	if item.CurrentLease != nil && item.CurrentLease.ValidAt(now) {
		return definitionPrecondition(domain.ErrorCodeWorkAlreadyClaimed, "active_lease", "release the active lease before changing the work definition", item.Ref(), nil)
	}
	return item.ValidateDefinitionEdit(now)
}

func validateWorkDefinitionLifecycle(item domain.WorkItem) error {
	if item.Lifecycle == domain.WorkItemLifecycleDone || item.Lifecycle == domain.WorkItemLifecycleCancelled {
		return definitionPrecondition(domain.ErrorCodePreconditionFailed, "entity_terminal", "reopen terminal work before changing its definition", item.Ref(), nil)
	}
	return nil
}

func validateWorkObjectiveChange(ctx context.Context, u ports.UnitOfWork, item domain.WorkItem, destination *domain.ID) error {
	if destination == nil {
		return definitionPrecondition(domain.ErrorCodePreconditionFailed, "task_objective_required", "a Task must belong to an Objective; choose an Objective instead of placing it directly under the Outcome", item.Ref(), []string{"objective_id"})
	}
	if err := destination.Validate(); err != nil {
		return err
	}
	objective, err := u.Objectives().Get(ctx, item.Scope, *destination)
	if err != nil {
		return err
	}
	if objective.Lifecycle == domain.ObjectiveLifecycleAchieved || objective.Lifecycle == domain.ObjectiveLifecycleCancelled {
		return definitionPrecondition(domain.ErrorCodePreconditionFailed, "objective_terminal", "choose a nonterminal Objective or reopen the destination", item.Ref(), []string{"objective_id"}, objective.Ref())
	}
	relations, err := u.Relations().ListByOutcome(ctx, item.Scope)
	if err != nil {
		return err
	}
	focus := map[domain.EntityRef]struct{}{item.Ref(): {}}
	if !hasRelevantHardDependency(relations, focus) {
		return nil
	}
	candidate := item
	candidate.ObjectiveID = destination
	if err := validateOwnObjectiveHardDependencies(ctx, u, relations, focus, map[domain.ID]domain.WorkItem{candidate.ID: candidate}); err != nil {
		return err
	}
	profile, err := dependencyOutcomeProfile(ctx, u, item.Scope)
	if err != nil {
		return err
	}
	if profile != domain.OperationalProfileOperationalV2 {
		return nil
	}
	if len(relations) > domain.DefaultDependencyGraphLimit {
		return domain.NewError(domain.ErrorCodeGraphLimitExceeded, "work association validation exceeds the relation graph limit")
	}
	objectives, items, err := loadDependencyHierarchy(ctx, u, item.Scope)
	if err != nil {
		return err
	}
	found := false
	for index := range items {
		if items[index].ID == candidate.ID {
			items[index] = candidate
			found = true
			break
		}
	}
	if !found {
		return domain.NewError(domain.ErrorCodeGraphLimitExceeded, "Task is missing from the bounded hierarchy snapshot")
	}
	return validateDependencyHierarchyDeadlocks(relations, objectives, items, focus, "objective_id")
}

func ownObjectiveDependencyError(item domain.WorkItem, relation domain.Relation) error {
	return definitionPrecondition(domain.ErrorCodePreconditionFailed, "own_objective_hard_dependency", "a Task cannot depend hard on its own Objective; use association for belonging or explicitly remove the dependency", item.Ref(), []string{"objective_id"}, relation.TargetRef)
}

func validateOwnObjectiveDependency(ctx context.Context, u ports.UnitOfWork, relation domain.Relation) error {
	if relation.SourceRef.Kind != domain.EntityKindWorkItem || relation.TargetRef.Kind != domain.EntityKindObjective || relation.Strength != domain.DependencyStrengthHard {
		return nil
	}
	item, err := u.WorkItems().Get(ctx, relation.Scope, relation.SourceRef.ID)
	if err != nil {
		return err
	}
	if domain.IsOwnObjectiveHardDependency(item, relation) {
		return ownObjectiveDependencyError(item, relation)
	}
	return nil
}

func validateObjectiveParentChange(ctx context.Context, s *Service, u ports.UnitOfWork, objective domain.Objective, destination *domain.ID, now time.Time) error {
	var objectives []domain.Objective
	var items []domain.WorkItem
	var err error
	bounded := false
	if reads, ok := u.(ports.ContinuityReadUnitOfWork); ok {
		if repo, ok := reads.ContinuityReads().(ports.WorkspaceReadRepository); ok {
			objectives, items, err = repo.WorkspaceStructure(ctx, objective.Scope, domain.DefaultDependencyGraphLimit+1)
			bounded = true
		}
	}
	if !bounded {
		objectives, err = u.Objectives().ListByOutcome(ctx, objective.Scope)
		if err == nil {
			items, err = u.WorkItems().ListByOutcome(ctx, objective.Scope)
		}
	}
	if err != nil {
		return err
	}
	if len(objectives) > domain.DefaultDependencyGraphLimit {
		return domain.NewError(domain.ErrorCodeGraphLimitExceeded, "Objective hierarchy exceeds structural validation limit")
	}
	if len(items) > domain.DefaultDependencyGraphLimit {
		return domain.NewError(domain.ErrorCodeGraphLimitExceeded, "Objective branch work exceeds structural validation limit")
	}
	byID := make(map[domain.ID]domain.Objective, len(objectives))
	children := make(map[domain.ID][]domain.ID, len(objectives))
	for _, value := range objectives {
		byID[value.ID] = value
		if value.ParentObjectiveID != nil {
			children[*value.ParentObjectiveID] = append(children[*value.ParentObjectiveID], value.ID)
		}
	}
	if destination != nil {
		if err := destination.Validate(); err != nil {
			return err
		}
		parent, ok := byID[*destination]
		if !ok {
			return domain.NewError(domain.ErrorCodeNotFound, "destination Objective was not found in this Outcome")
		}
		if parent.Lifecycle == domain.ObjectiveLifecycleAchieved || parent.Lifecycle == domain.ObjectiveLifecycleCancelled {
			return definitionPrecondition(domain.ErrorCodePreconditionFailed, "objective_terminal", "choose a nonterminal parent Objective", objective.Ref(), []string{"parent_objective_id"}, parent.Ref())
		}
		seen := map[domain.ID]bool{}
		for current := destination; current != nil; {
			if *current == objective.ID || seen[*current] {
				return definitionPrecondition(domain.ErrorCodeDependencyCycle, "objective_hierarchy_cycle", "Objective parent would introduce or inherit a hierarchy cycle", objective.Ref(), []string{"parent_objective_id"}, parent.Ref())
			}
			seen[*current] = true
			ancestor, ok := byID[*current]
			if !ok {
				return domain.NewError(domain.ErrorCodeNotFound, "destination Objective ancestry is incomplete")
			}
			current = ancestor.ParentObjectiveID
		}
	}
	branch := map[domain.ID]bool{}
	queue := []domain.ID{objective.ID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if branch[current] {
			return definitionPrecondition(domain.ErrorCodeDependencyCycle, "objective_hierarchy_cycle", "current Objective branch contains a hierarchy cycle", objective.Ref(), nil)
		}
		branch[current] = true
		value := byID[current]
		if value.Lifecycle != domain.ObjectiveLifecyclePlanned {
			return definitionPrecondition(domain.ErrorCodePreconditionFailed, "objective_subtree_not_planned", "only an entirely planned Objective branch can be reorganized", objective.Ref(), []string{"parent_objective_id"}, value.Ref())
		}
		queue = append(queue, children[current]...)
	}
	for _, item := range items {
		if item.ObjectiveID == nil || !branch[*item.ObjectiveID] {
			continue
		}
		if err := validateWorkDefinitionLifecycle(item); err != nil {
			return err
		}
		// The minimal structural projection explicitly includes open review and
		// authority pointers. Clean rows need no per-Task review lookup.
		if !bounded || item.CurrentContractID != nil || item.PendingReviewCaseID != nil || item.CurrentLease != nil {
			if err := validateWorkDefinitionChange(ctx, s, u, item, now); err != nil {
				return err
			}
		}
		if item.Lifecycle != domain.WorkItemLifecycleBacklog && item.Lifecycle != domain.WorkItemLifecycleTodo {
			return definitionPrecondition(domain.ErrorCodePreconditionFailed, "objective_subtree_in_execution", "prepare descendant work before moving its Objective branch", objective.Ref(), []string{"parent_objective_id"}, item.Ref())
		}
	}
	candidateObjectives := append([]domain.Objective(nil), objectives...)
	for index := range candidateObjectives {
		if candidateObjectives[index].ID == objective.ID {
			candidateObjectives[index].ParentObjectiveID = cloneIDPointer(destination)
			break
		}
	}
	changed := make(map[domain.EntityRef]struct{}, len(branch)+len(items))
	for id := range branch {
		changed[domain.EntityRef{Scope: objective.Scope, Kind: domain.EntityKindObjective, ID: id}] = struct{}{}
	}
	for _, item := range items {
		if item.ObjectiveID != nil && branch[*item.ObjectiveID] {
			changed[item.Ref()] = struct{}{}
		}
	}
	// Validate active hard edges touching the moved branch against the proposed
	// parent graph. Advisory-only and unrelated legacy edges cannot make a move
	// unsafe, so skip the relation limit when no hard edge is affected.
	relations, err := u.Relations().ListByOutcome(ctx, objective.Scope)
	if err != nil {
		return err
	}
	profile, err := dependencyOutcomeProfile(ctx, u, objective.Scope)
	if err != nil {
		return err
	}
	if profile != domain.OperationalProfileOperationalV2 {
		return nil
	}
	if !hasRelevantHardDependency(relations, changed) {
		return nil
	}
	if len(relations) > domain.DefaultDependencyGraphLimit {
		return domain.NewError(domain.ErrorCodeGraphLimitExceeded, "Objective placement validation exceeds the relation graph limit")
	}
	if err := validateDependencyHierarchyDeadlocks(relations, candidateObjectives, items, changed, "parent_objective_id"); err != nil {
		return err
	}
	return nil
}

func validateExecutionSpecRefs(ctx context.Context, u ports.UnitOfWork, scope domain.Scope, spec *domain.ExecutionSpec) error {
	if spec == nil {
		return nil
	}
	if err := spec.Validate(); err != nil {
		return err
	}
	for _, ref := range spec.ContextRefs {
		if ref.Scope != scope {
			return domain.NewError(domain.ErrorCodeInvalidScope, "execution context reference must belong to the same Outcome")
		}
		if err := validateDefinitionContextRef(ctx, u, ref); err != nil {
			return err
		}
	}
	return nil
}

func validateDefinitionContextRef(ctx context.Context, u ports.UnitOfWork, ref domain.EntityRef) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	var err error
	switch ref.Kind {
	case domain.EntityKindOutcome:
		if ref.ID != ref.Scope.OutcomeID {
			return domain.NewError(domain.ErrorCodeInvalidEntityRef, "Outcome reference ID must match its scope")
		}
		_, err = u.Outcomes().Get(ctx, ref.Scope.NamespaceID, ref.ID)
	case domain.EntityKindObjective:
		_, err = u.Objectives().Get(ctx, ref.Scope, ref.ID)
	case domain.EntityKindWorkItem:
		_, err = u.WorkItems().Get(ctx, ref.Scope, ref.ID)
	case domain.EntityKindRelation:
		_, err = u.Relations().Get(ctx, ref.Scope, ref.ID)
	case domain.EntityKindIssue, domain.EntityKindBlocker:
		repo, ok := u.(ports.IssueBlockerUnitOfWork)
		if !ok {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "adapter lacks Issue/Blocker references")
		}
		if ref.Kind == domain.EntityKindIssue {
			_, err = repo.Issues().Get(ctx, ref.Scope, ref.ID)
		} else {
			_, err = repo.Blockers().Get(ctx, ref.Scope, ref.ID)
		}
	case domain.EntityKindArtifact, domain.EntityKindEvidence, domain.EntityKindEvidenceLink, domain.EntityKindDecision:
		repo, ok := u.(ports.DocumentaryUnitOfWork)
		if !ok {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "adapter lacks documentary references")
		}
		switch ref.Kind {
		case domain.EntityKindArtifact:
			_, err = repo.Artifacts().Get(ctx, ref.Scope, ref.ID)
		case domain.EntityKindEvidence:
			_, err = repo.Evidence().Get(ctx, ref.Scope, ref.ID)
		case domain.EntityKindEvidenceLink:
			_, err = repo.EvidenceLinks().Get(ctx, ref.Scope, ref.ID)
		case domain.EntityKindDecision:
			_, err = repo.Decisions().Get(ctx, ref.Scope, ref.ID)
		}
	case domain.EntityKindRoadmap:
		repo, ok := u.(ports.PlanningUnitOfWork)
		if !ok {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "adapter lacks Roadmap references")
		}
		_, err = repo.Roadmaps().Get(ctx, ref.Scope, ref.ID)
	default:
		return domain.NewError(domain.ErrorCodeInvalidEntityRef, "unsupported execution context reference kind")
	}
	return err
}
