package application

import (
	"context"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

const defaultObjectiveCompletionSampleLimit = 8
const maxObjectiveCompletionSampleLimit = 20

// GetObjectiveCompletionReport evaluates current factual obligations for one
// Objective in a single authenticated read frame. The caller's permission to
// execute AchieveObjective is intentionally not inferred here.
func (s *Service) GetObjectiveCompletionReport(ctx context.Context, scope d.Scope, objectiveID d.ID, requestedLimit int) (ReadResult[ObjectiveCompletionReport], error) {
	if err := scope.Validate(); err != nil {
		return ReadResult[ObjectiveCompletionReport]{}, err
	}
	if err := objectiveID.Validate(); err != nil {
		return ReadResult[ObjectiveCompletionReport]{}, d.NewError(d.ErrorCodeInvalidArgument, "invalid Objective ID")
	}
	limit := requestedLimit
	if limit <= 0 {
		limit = defaultObjectiveCompletionSampleLimit
	}
	limit = min(limit, maxObjectiveCompletionSampleLimit)

	frame, handled, err := s.openWorkspaceFrame(ctx, scope)
	if err != nil {
		return ReadResult[ObjectiveCompletionReport]{}, err
	}
	if !handled {
		return ReadResult[ObjectiveCompletionReport]{}, d.NewError(d.ErrorCodeInvalidConfig, "coherent Objective completion reads are unavailable")
	}
	defer frame.u.Rollback()

	objective, err := frame.u.Objectives().Get(ctx, scope, objectiveID)
	if err != nil {
		return ReadResult[ObjectiveCompletionReport]{}, err
	}
	if objective.Scope != scope || objective.ID != objectiveID {
		return ReadResult[ObjectiveCompletionReport]{}, d.NewError(d.ErrorCodeInvalidScope, "Objective completion report resolved a different Objective")
	}
	evaluation, err := s.evaluateObjectiveCompletionInUnit(ctx, frame.u, objective, frame.revision, frame.at, limit, nil)
	if err != nil {
		return ReadResult[ObjectiveCompletionReport]{}, err
	}
	return ReadResult[ObjectiveCompletionReport]{Value: evaluation.Report, OutcomeRevision: frame.revision}, nil
}

// evaluateObjectiveCompletionInUnit is the single source of factual
// completion state. The caller supplies its authenticated read frame or
// guarded write UoW so no nested transaction can observe stale facts.
func (s *Service) evaluateObjectiveCompletionInUnit(ctx context.Context, u ports.UnitOfWork, objective d.Objective, revision d.OutcomeRevision, at time.Time, requestedLimit int, cachedIssues *objectiveIssueReport) (objectiveCompletionEvaluation, error) {
	limit := requestedLimit
	if limit <= 0 {
		limit = defaultObjectiveCompletionSampleLimit
	}
	limit = min(limit, maxObjectiveCompletionSampleLimit)
	scope := objective.Scope
	if err := scope.Validate(); err != nil {
		return objectiveCompletionEvaluation{}, err
	}
	if err := objective.Validate(); err != nil {
		return objectiveCompletionEvaluation{}, d.WrapError(d.ErrorCodeInvalidConfig, "Objective completion facts are invalid", err)
	}
	outcome, err := completionOutcome(ctx, u, scope)
	if err != nil {
		return objectiveCompletionEvaluation{}, err
	}
	evaluation := objectiveCompletionEvaluation{Report: newObjectiveCompletionReport(objective, revision, at, limit, outcome.CompletionPolicy)}
	report := &evaluation.Report

	if objective.Lifecycle != d.ObjectiveLifecyclePlanned && objective.Lifecycle != d.ObjectiveLifecycleInProgress {
		report.addReason("invalid_lifecycle")
	}
	if err := objective.ValidateCompletionConfiguration(outcome); err != nil {
		report.addReason(objectiveCompletionReason(err, "invalid_completion_configuration"))
	}

	fullTasks, err := u.WorkItems().ListByOutcome(ctx, scope)
	if err != nil {
		return objectiveCompletionEvaluation{}, err
	}
	fullObjectives, err := u.Objectives().ListByOutcome(ctx, scope)
	if err != nil {
		return objectiveCompletionEvaluation{}, err
	}
	if err := validateObjectiveCompletionScan(scope, fullTasks, fullObjectives); err != nil {
		report.markIncomplete("completion_scan_inconsistent")
	} else {
		if err := validateObjectiveCompletionRepositoryScan(ctx, u, scope, fullTasks, fullObjectives); err != nil {
			report.markIncomplete("completion_scan_incomplete")
		}
	}
	objectiveByID := objectiveCompletionDescendants(fullObjectives, objective, limit, report)
	for _, item := range fullTasks {
		if item.ObjectiveID != nil {
			if _, ok := objectiveByID[*item.ObjectiveID]; !ok {
				report.markIncomplete("task_objective_link_incomplete")
			}
		}
	}
	tasks := objectiveCompletionTaskScope(fullTasks, objectiveByID, objective)
	report.Tasks.Total = len(tasks)
	for _, item := range tasks {
		switch item.Lifecycle {
		case d.WorkItemLifecycleBacklog:
			report.Tasks.Backlog++
			addCompletionRef(&report.Tasks.Open, item.Ref(), limit)
		case d.WorkItemLifecycleTodo:
			report.Tasks.Todo++
			addCompletionRef(&report.Tasks.Open, item.Ref(), limit)
		case d.WorkItemLifecycleInProgress:
			report.Tasks.InProgress++
			addCompletionRef(&report.Tasks.Open, item.Ref(), limit)
		case d.WorkItemLifecycleDone:
			report.Tasks.Done++
		case d.WorkItemLifecycleCancelled:
			report.Tasks.Cancelled++
		default:
			report.markIncomplete("task_lifecycle_invalid")
		}
	}

	if report.StrictWork.Applies {
		evaluation.StrictOpen, evaluation.StrictCancelled = strictCompletionWorkFromSnapshot(objective.Ref(), fullTasks, fullObjectives)
		for _, ref := range evaluation.StrictOpen {
			addCompletionRef(&report.StrictWork.Open, ref, limit)
		}
		for _, ref := range evaluation.StrictCancelled {
			addCompletionRef(&report.StrictWork.Cancelled, ref, limit)
		}
		report.StrictWork.AcknowledgementRequired = len(evaluation.StrictCancelled) > 0
		if len(evaluation.StrictOpen) > 0 {
			report.addReason("required_child_work_open")
		}
		if len(evaluation.StrictCancelled) > 0 {
			report.addReason("cancelled_work_requires_ack")
		}
	}

	if err := s.objectiveCompletionTaskVerification(ctx, u, scope, tasks, limit, report); err != nil {
		return objectiveCompletionEvaluation{}, err
	}
	objectiveCompletionRequiredCriteria(objective, limit, report)
	if objectiveCompletionCycleProofUnverifiable(objective) {
		report.markIncomplete("objective_assessment_cycle_unavailable")
	}
	if report.Criteria.ActiveRequired > 0 && report.Criteria.Pending == 0 {
		if err := validateRequiredAssessmentProof(ctx, u, objective.Ref(), objective.Criteria); err != nil {
			if objectiveCompletionBusinessFailure(err) {
				report.Criteria.ProofInvalid++
				report.addReason("current_assessment_proof_invalid")
			} else {
				report.markIncomplete("assessment_proof_unavailable")
			}
		}
	}
	if cachedIssues != nil && cachedIssues.owner == objective.Ref() {
		evaluation.Issues = *cachedIssues
	} else {
		evaluation.Issues = evaluateObjectiveIssues(ctx, u, objective.Ref(), limit, true)
	}
	if err := objectiveCompletionIssuesFromReport(ctx, evaluation.Issues, report); err != nil {
		return objectiveCompletionEvaluation{}, err
	}
	if err := objectiveCompletionDependencies(ctx, u, objective.Ref(), limit, report); err != nil {
		return objectiveCompletionEvaluation{}, err
	}
	if err := objectiveCompletionBlockers(ctx, u, objective.Ref(), objectiveByID, limit, report); err != nil {
		return objectiveCompletionEvaluation{}, err
	}
	if report.HardDependencies.Unsatisfied.Total > 0 {
		report.addReason("hard_dependency_unsatisfied")
	}
	if report.ActiveBlockers.Blockers.Total > 0 {
		report.addReason("active_blocker")
	}
	if !report.PendingIssues.Complete {
		report.markIncomplete("objective_issue_scan_incomplete")
	} else if report.PendingIssues.Pending.Total > 0 {
		report.addReason("objective_issue_pending")
	}
	if err := ctx.Err(); err != nil {
		return objectiveCompletionEvaluation{}, err
	}
	report.finish()
	return evaluation, nil
}

func validateObjectiveCompletionScan(scope d.Scope, tasks []d.WorkItem, objectives []d.Objective) error {
	seenTasks := make(map[d.ID]bool, len(tasks))
	for _, task := range tasks {
		if task.Scope != scope || task.ID.Validate() != nil || seenTasks[task.ID] || task.Validate() != nil {
			return d.NewError(d.ErrorCodeInvalidConfig, "completion Task scan is inconsistent")
		}
		seenTasks[task.ID] = true
	}
	seenObjectives := make(map[d.ID]bool, len(objectives))
	for _, objective := range objectives {
		if objective.Scope != scope || objective.ID.Validate() != nil || seenObjectives[objective.ID] || objective.Validate() != nil {
			return d.NewError(d.ErrorCodeInvalidConfig, "completion Objective scan is inconsistent")
		}
		seenObjectives[objective.ID] = true
	}
	for _, objective := range objectives {
		if objective.ParentObjectiveID != nil && !seenObjectives[*objective.ParentObjectiveID] {
			return d.NewError(d.ErrorCodeInvalidConfig, "completion Objective hierarchy is incomplete")
		}
	}
	for _, task := range tasks {
		if task.ObjectiveID != nil && !seenObjectives[*task.ObjectiveID] {
			return d.NewError(d.ErrorCodeInvalidConfig, "completion Task has a missing Objective")
		}
	}
	return nil
}

func validateObjectiveCompletionRepositoryScan(ctx context.Context, u ports.UnitOfWork, scope d.Scope, tasks []d.WorkItem, objectives []d.Objective) error {
	reads, ok := u.(ports.ContinuityReadUnitOfWork)
	if !ok {
		return nil
	}
	completion, ok := reads.ContinuityReads().(ports.CompletionWorkRepository)
	if !ok {
		return nil
	}
	summaryTasks, summaryObjectives, err := completion.CompletionWork(ctx, scope)
	if err != nil {
		return err
	}
	if len(summaryTasks) != len(tasks) || len(summaryObjectives) != len(objectives) {
		return d.NewError(d.ErrorCodeInvalidConfig, "complete acceptance scan omitted current work")
	}
	taskByID := make(map[d.ID]d.WorkItem, len(tasks))
	for _, task := range tasks {
		taskByID[task.ID] = task
	}
	for _, item := range summaryTasks {
		full, ok := taskByID[item.ID]
		if !ok || item.Scope != scope || item.Lifecycle != full.Lifecycle || !completionIDEqual(item.ObjectiveID, full.ObjectiveID) {
			return d.NewError(d.ErrorCodeInvalidConfig, "complete acceptance Task scan does not match current Task state")
		}
		delete(taskByID, item.ID)
	}
	if len(taskByID) != 0 {
		return d.NewError(d.ErrorCodeInvalidConfig, "complete acceptance Task scan omitted current Tasks")
	}
	objectiveByID := make(map[d.ID]d.Objective, len(objectives))
	for _, objective := range objectives {
		objectiveByID[objective.ID] = objective
	}
	for _, item := range summaryObjectives {
		full, ok := objectiveByID[item.ID]
		if !ok || item.Scope != scope || item.Lifecycle != full.Lifecycle || !completionIDEqual(item.ParentObjectiveID, full.ParentObjectiveID) || item.CompletionMode.Effective() != full.CompletionMode.Effective() || item.EffectiveCompletionCycle() != full.EffectiveCompletionCycle() {
			return d.NewError(d.ErrorCodeInvalidConfig, "complete acceptance Objective scan does not match current Objective state")
		}
		delete(objectiveByID, item.ID)
	}
	if len(objectiveByID) != 0 {
		return d.NewError(d.ErrorCodeInvalidConfig, "complete acceptance Objective scan omitted current Objectives")
	}
	return nil
}

func completionIDEqual(a, b *d.ID) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

func (s *Service) objectiveCompletionTaskVerification(ctx context.Context, u ports.UnitOfWork, scope d.Scope, tasks []d.WorkItem, limit int, report *ObjectiveCompletionReport) error {
	// A workspace may expose only summary/task repositories for an Objective
	// with no Tasks. In that case there are no verification facts to read; do
	// not turn an empty, complete set into an unavailable-storage condition.
	if len(tasks) == 0 {
		return nil
	}
	_, ok := u.(ports.TaskVerificationReadUnitOfWork)
	if !ok {
		report.markIncomplete("task_verification_facts_unavailable")
		return nil
	}
	for start := 0; start < len(tasks); start += ports.MaxTaskVerificationBatch {
		batch := tasks[start:min(start+ports.MaxTaskVerificationBatch, len(tasks))]
		ids := make([]d.ID, 0, len(batch))
		for _, item := range batch {
			ids = append(ids, item.ID)
		}
		facts, err := loadTaskVerificationFacts(ctx, u, scope, ids)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			report.markIncomplete("task_verification_facts_incomplete")
			return nil
		}
		for _, item := range batch {
			fact, ok := facts[item.ID]
			if !ok || !verificationFactsMatchWork(fact, item) {
				report.markIncomplete("task_verification_snapshot_mismatch")
				continue
			}
			verification, err := projectTaskVerification(scope, fact, report.OutcomeRevision, report.EvaluatedAt)
			if err != nil {
				report.markIncomplete("task_verification_source_invalid")
				continue
			}
			if verification.Pending && verification.SourceKind == "task_delivery" {
				digest, digestErr := d.TaskDefinitionDigest(item)
				if digestErr != nil || fact.NativeDelivery == nil || fact.NativeDelivery.DefinitionDigest != digest {
					report.markIncomplete("task_delivery_definition_mismatch")
					continue
				}
			}
			addObjectiveCompletionVerification(item, verification, limit, report)
			objectiveCompletionTaskCriteria(item, &verification, limit, report)
		}
	}
	return nil
}

func addObjectiveCompletionVerification(item d.WorkItem, verification TaskVerificationView, limit int, report *ObjectiveCompletionReport) {
	if verification.Pending {
		addCompletionRef(&report.Tasks.AwaitingVerification, item.Ref(), limit)
		if verification.SourceKind == "task_delivery" || verification.SourceKind == "signed_review_case" {
			addCompletionRef(&report.Tasks.PendingReview, item.Ref(), limit)
		}
	}
	if verification.CorrectionRequired {
		addCompletionRef(&report.Tasks.CorrectionRequired, item.Ref(), limit)
	}
	if verification.ExecutorAuthority == "active" || verification.ReviewerAuthority == "active" {
		addCompletionRef(&report.Tasks.ActiveAuthority, item.Ref(), limit)
	}
}

func objectiveCompletionIssuesFromReport(ctx context.Context, issues objectiveIssueReport, report *ObjectiveCompletionReport) error {
	// gateError may be the expected business failure for a pending Issue. Scan
	// completion is an independent fact and must remain true in that case.
	report.PendingIssues.Complete = issues.complete
	report.PendingIssues.Pending.Total = issues.pendingCount
	report.PendingIssues.Pending.Samples = append([]d.EntityRef{}, issues.pendingRefs...)
	if !report.PendingIssues.Complete {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		report.markIncomplete("objective_issue_scan_incomplete")
		return nil
	}
	return nil
}

func objectiveCompletionDependencies(ctx context.Context, u ports.UnitOfWork, owner d.EntityRef, limit int, report *ObjectiveCompletionReport) error {
	relations, err := u.Relations().ListByOutcome(ctx, owner.Scope)
	if err != nil {
		return err
	}
	for _, relation := range relations {
		if relation.Scope != owner.Scope || relation.Validate() != nil {
			report.markIncomplete("dependency_scan_inconsistent")
			continue
		}
		if relation.Lifecycle != d.RelationLifecycleActive || relation.RelationType != d.RelationTypeDependsOn || relation.SourceRef != owner || relation.Strength != d.DependencyStrengthHard {
			continue
		}
		report.HardDependencies.Total++
		state, err := dependencyEndpointState(ctx, u, relation.TargetRef)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			report.markIncomplete("dependency_target_unavailable")
			continue
		}
		if state.completed {
			report.HardDependencies.Satisfied++
		} else {
			addCompletionRef(&report.HardDependencies.Unsatisfied, relation.TargetRef, limit)
		}
	}
	return nil
}

func objectiveCompletionBlockers(ctx context.Context, u ports.UnitOfWork, owner d.EntityRef, objectives map[d.ID]d.Objective, limit int, report *ObjectiveCompletionReport) error {
	issues, blockers, err := issueBlockerRepositories(u)
	_ = issues
	if err != nil {
		return err
	}
	values, err := blockers.ListByOutcome(ctx, owner.Scope)
	if err != nil {
		return err
	}
	for _, blocker := range values {
		if blocker.Scope != owner.Scope || blocker.Validate() != nil {
			report.markIncomplete("blocker_scan_inconsistent")
		}
	}
	objectiveID := owner.ID
	state := projectBlockingState(owner, &objectiveID, values, objectives)
	for _, applied := range state.ActiveBlockers {
		addCompletionRef(&report.ActiveBlockers.Blockers, applied.Blocker.Ref(), limit)
	}
	report.ActiveBlockers.Total = report.ActiveBlockers.Blockers.Total
	return nil
}
