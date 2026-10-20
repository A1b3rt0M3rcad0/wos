package application

import (
	"context"
	"errors"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// Capabilities explain an observed snapshot. The command pipeline always
// revalidates permissions, versions, time and authority before committing.
func (s *Service) workspaceCapabilities(ctx context.Context, f *continuityReadFrame, v *EntityWorkspaceView, selected any, work *WorkContext, outcome d.Outcome) ([]WorkspaceCapability, error) {
	result := []WorkspaceCapability{}
	identity, _ := IdentityFromContext(ctx)
	protocol := d.DefaultWorkProtocol(v.Subject.Ref.NamespaceID)
	if p, ok := f.u.(ports.WorkProtocolUnitOfWork); ok {
		var e error
		protocol, e = p.WorkProtocol().Lock(ctx, v.Subject.Ref.NamespaceID, false)
		if e != nil {
			return nil, e
		}
	}
	permissions := map[ports.Permission]bool{}
	checked := map[ports.Permission]bool{}
	allow := func(command string) (bool, error) {
		p := commandPermission(command)
		if checked[p] {
			return permissions[p], nil
		}
		checked[p] = true
		if !s.requireIdentity {
			permissions[p] = !privilegedPresentationPermission(p)
			return permissions[p], nil
		}
		identity, ok := IdentityFromContext(ctx)
		if !ok {
			return false, nil
		}
		request := ports.AuthorizationRequest{NamespaceID: v.Subject.Ref.NamespaceID, OutcomeID: v.Subject.Ref.OutcomeID, PrincipalID: identity.PrincipalID, Permission: p}
		var e error
		if identity.CredentialDigest != "" {
			snapshot, ok := f.u.(ports.AccessSnapshotUnitOfWork)
			if !ok {
				return false, d.NewError(d.ErrorCodeInvalidConfig, "transactional capability authorization unavailable")
			}
			e = snapshot.AuthorizeAccessSnapshot(ctx, ports.AccessSnapshotRequest{Authorization: request, CredentialDigest: identity.CredentialDigest, Actor: identity.Actor, Now: f.at})
		} else if transactional, ok := s.authorizer.(ports.TransactionalAuthorizer); ok {
			e = transactional.AuthorizeInUnitOfWork(ctx, f.u, request)
		} else {
			e = s.authorizer.Authorize(ctx, request)
		}
		if e == nil {
			permissions[p] = true
			return true, nil
		}
		if code, _ := d.ErrorCodeOf(e); code == d.ErrorCodeForbidden {
			return false, nil
		}
		return false, e
	}
	add := func(command, transport string, priority int, condition error) error {
		ok, e := allow(command)
		if e != nil {
			return e
		}
		if !ok {
			return nil
		}
		c := WorkspaceCapability{ActionID: command, Subject: v.Subject, Transport: transport, Availability: "available", ReasonCodes: []string{}, RelatedRefs: []d.EntityRef{}, ObservedVersion: v.EntityVersion, PresentationPriority: priority}
		if condition != nil {
			code, ok := d.ErrorCodeOf(condition)
			if !ok {
				return condition
			}
			c.Availability = "blocked"
			c.ReasonCodes = append(c.ReasonCodes, string(code))
			var detail *d.Error
			if errors.As(condition, &detail) && detail.Details != nil {
				if detail.Details.ReasonCode != "" {
					c.ReasonCodes = []string{detail.Details.ReasonCode}
				}
				c.RelatedRefs = append(c.RelatedRefs, detail.Details.RelatedRefs...)
				c.Remedies = append(c.Remedies, detail.Details.Remedies...)
				// Opening a confirmation is available; accepting the result is
				// still gated by the exact acknowledgement in the command.
				if detail.Details.ReasonCode == "cancelled_work_requires_ack" && (command == "AchieveOutcome" || command == "AchieveObjective") {
					c.Availability = "available"
					c.ConfirmationKind = "cancelled_scope_exclusion"
				}
			}
		}
		if condition == nil && transport == "signed_host" {
			c.Availability = "unknown"
			c.ReasonCodes = []string{"authorized_host_profile_required"}
		}
		result = append(result, c)
		return nil
	}
	open := func() error {
		if outcome.IsArchived() {
			return d.NewError(d.ErrorCodePreconditionFailed, "Outcome is archived")
		}
		if outcome.Lifecycle == d.OutcomeLifecycleAchieved || outcome.Lifecycle == d.OutcomeLifecycleFailed || outcome.Lifecycle == d.OutcomeLifecycleAbandoned {
			return d.NewError(d.ErrorCodePreconditionFailed, "reopen the Outcome before changing operational work")
		}
		return nil
	}
	documentaryCondition := error(nil)
	if outcome.IsArchived() {
		documentaryCondition = d.NewDetailedError(d.ErrorCodePreconditionFailed, "unarchive the Outcome before adding documentary records", d.ErrorDetails{ReasonCode: "outcome_archived", SubjectRef: refPointer(outcome.Ref())})
	}
	if roadmap, ok := selected.(d.Roadmap); ok && roadmap.ArchivedAt != nil {
		documentaryCondition = d.NewDetailedError(d.ErrorCodePreconditionFailed, "unarchive the Roadmap before adding documentary records", d.ErrorDetails{ReasonCode: "roadmap_archived", SubjectRef: refPointer(roadmap.Ref())})
	}
	for _, command := range []string{"AppendEntityNote", "RegisterArtifactInContext", "ProposeDecisionInContext", "CreateIssue"} {
		if e := add(command, "browser_command", 40, documentaryCondition); e != nil {
			return nil, e
		}
	}
	if d.EvidenceLinkTargetAllowed(v.Subject.Ref.Kind) {
		if e := add("RegisterEvidenceWithLinks", "browser_command", 35, documentaryCondition); e != nil {
			return nil, e
		}
	}
	if v.Subject.Kind == "criterion" {
		transport := "browser_command"
		if work != nil && protocol.Phase == d.WorkProtocolSigned {
			transport = "signed_host"
		}
		if e := add("RecordCriterionAssessment", transport, 20, documentaryCondition); e != nil {
			return nil, e
		}
		return result, nil
	}
	if v.Subject.Kind != "entity" {
		return result, nil
	}
	switch record := selected.(type) {
	case d.WorkItem:
		criterionCondition := record.CheckCriterionDefinitionEdit()
		if criterionCondition == nil {
			criterionCondition = protectWorkContractMutation(ctx, f.u, record, f.at, false, false)
		}
		priority := 25
		if !record.Criteria.HasRequiredActive() {
			priority = 15
		}
		if e := add("AddCriterion", "browser_command", priority, criterionCondition); e != nil {
			return nil, e
		}
		condition := record.CheckPlanningEdit()
		if condition == nil {
			condition = validateWorkDefinitionChange(ctx, s, f.u, record, f.at)
		}
		if condition == nil {
			condition = open()
		}
		for _, command := range []string{"SetWorkItemExecutionSpec", "SetWorkItemNotBefore", "SetWorkItemObjective"} {
			if e := add(command, "browser_command", 30, condition); e != nil {
				return nil, e
			}
		}
		clone := record
		_, editCondition := clone.UpdateDetails(nil, nil, nil, f.at)
		if editCondition == nil {
			editCondition = open()
		}
		if editCondition == nil {
			editCondition = protectWorkContractMutation(ctx, f.u, record, f.at, false, false)
		}
		if e := add("UpdateWorkItem", "browser_command", 25, editCondition); e != nil {
			return nil, e
		}
		clone = record
		assigneeCondition := clone.SetAssignees(record.AssigneeRefs, f.at)
		if assigneeCondition == nil {
			assigneeCondition = open()
		}
		if assigneeCondition == nil {
			assigneeCondition = protectWorkContractMutation(ctx, f.u, record, f.at, false, false)
		}
		if e := add("SetWorkItemAssignees", "browser_command", 30, assigneeCondition); e != nil {
			return nil, e
		}
		if record.Lifecycle == d.WorkItemLifecycleBacklog {
			if e := add("ActivateWorkItem", "browser_command", 10, protectWorkProtocol(protocol, "ActivateWorkItem")); e != nil {
				return nil, e
			}
		}
		if record.Lifecycle == d.WorkItemLifecycleTodo {
			transport := "browser_command"
			command := "ClaimWorkItem"
			if record.ContractsEnabled {
				transport = "signed_host"
				command = "AcquireWorkContract"
				if protocol.Phase == d.WorkProtocolSigned {
					command = "AcquireSignedWorkContract"
				}
			}
			var reason error
			if work != nil && len(work.Operational.ReadinessReasons) > 0 {
				reason = d.NewDetailedError(d.ErrorCodePreconditionFailed, string(work.Operational.ReadinessReasons[0]), d.ErrorDetails{ReasonCode: string(work.Operational.ReadinessReasons[0]), SubjectRef: refPointer(record.Ref())})
			}
			if e := add(command, transport, 10, reason); e != nil {
				return nil, e
			}
		}
		if record.PendingReviewCaseID != nil {
			if e := add("AcquireSignedReviewContract", "signed_host", 5, nil); e != nil {
				return nil, e
			}
		}
		if record.Lifecycle == d.WorkItemLifecycleInProgress && !record.ContractsEnabled && record.CurrentLease != nil {
			lease := record.CurrentLease
			clone := record
			leaseCopy := *lease
			clone.CurrentLease = &leaseCopy
			command := "RenewWorkItemLease"
			condition := clone.RenewLease(identity.PrincipalID, lease.ClaimID, lease.FencingToken, 5*time.Minute, f.at)
			if !lease.ValidAt(f.at) {
				command = "ReclaimWorkItem"
				// Evaluate on a copy with an existing distinct valid ID. No ID is
				// allocated or persisted by this explanatory read.
				condition = clone.Reclaim(record.ID, identity.PrincipalID, identity.Actor, 5*time.Minute, f.at)
			}
			if condition == nil && (outcome.IsArchived() || outcome.Lifecycle != d.OutcomeLifecycleActive) {
				condition = d.NewError(d.ErrorCodePreconditionFailed, "outcome must be active and not archived")
			}
			if condition == nil {
				condition = protectWorkProtocol(protocol, command)
			}
			if e := add(command, "browser_command", 20, condition); e != nil {
				return nil, e
			}
			released := record
			releaseCondition := released.Release(identity.PrincipalID, lease.ClaimID, lease.FencingToken, f.at)
			if releaseCondition == nil {
				releaseCondition = open()
			}
			if releaseCondition == nil {
				releaseCondition = protectWorkProtocol(protocol, "ReleaseWorkItem")
			}
			if e := add("ReleaseWorkItem", "browser_command", 35, releaseCondition); e != nil {
				return nil, e
			}
			condition = record.CheckCompletionAuthority(identity.PrincipalID, lease.ClaimID, lease.FencingToken, f.at)
			if condition == nil {
				condition = open()
			}
			if condition == nil {
				condition = protectWorkProtocol(protocol, "CompleteWorkItem")
			}
			if condition == nil {
				condition = requireNotBlocked(ctx, f.u, record.Ref())
			}
			if condition == nil {
				condition = requireHardDependenciesSatisfied(ctx, f.u, record.Ref())
			}
			if condition == nil && record.Criteria.HasRequiredActive() {
				_, condition = record.Criteria.RequiredSatisfied()
			}
			if condition == nil && record.Criteria.HasRequiredActive() {
				condition = validateRequiredAssessmentProof(ctx, f.u, record.Ref(), record.Criteria)
			}
			if e := add("CompleteWorkItem", "browser_command", 10, condition); e != nil {
				return nil, e
			}
			if len(result) > 0 && result[len(result)-1].ActionID == "CompleteWorkItem" {
				result[len(result)-1].ConfirmationKind = "explicit_result_and_conclusion"
			}
		}
	case d.Objective:
		if record.Lifecycle == d.ObjectiveLifecyclePlanned {
			condition := record.CheckStart()
			if condition == nil && (outcome.IsArchived() || outcome.Lifecycle != d.OutcomeLifecycleActive) {
				condition = d.NewError(d.ErrorCodePreconditionFailed, "outcome must be active and not archived")
			}
			if condition == nil {
				condition = requireObjectiveReady(ctx, f.u, record, f.at)
			}
			if e := add("StartObjective", "browser_command", 10, condition); e != nil {
				return nil, e
			}
		}
		creationCondition := open()
		if creationCondition == nil && (record.Lifecycle == d.ObjectiveLifecycleAchieved || record.Lifecycle == d.ObjectiveLifecycleCancelled) {
			creationCondition = d.NewDetailedError(d.ErrorCodeInvalidTransition, "reopen the parent Objective before adding work", d.ErrorDetails{ReasonCode: "objective_terminal", SubjectRef: refPointer(record.Ref()), Remedies: []d.ErrorRemedy{{ActionID: "reopen_objective", SubjectRef: refPointer(record.Ref())}}})
		}
		clone := record
		_, editCondition := clone.UpdateDetails(nil, nil, nil, nil, f.at)
		if editCondition == nil {
			editCondition = open()
		}
		if editCondition == nil {
			editCondition = inspectContractMutation(ctx, s, f.u, "UpdateObjective", UpdateObjectiveCommand{Scope: record.Scope, ObjectiveID: record.ID}, f.at)
		}
		if e := add("UpdateObjective", "browser_command", 25, editCondition); e != nil {
			return nil, e
		}
		clone = record
		ownerCondition := clone.SetOwners(record.OwnerRefs, f.at)
		if ownerCondition == nil {
			ownerCondition = open()
		}
		if e := add("SetObjectiveOwners", "browser_command", 30, ownerCondition); e != nil {
			return nil, e
		}
		if e := add("CreatePlanItems", "browser_command", 25, creationCondition); e != nil {
			return nil, e
		}
		if e := add("CreateWorkItem", "browser_command", 10, creationCondition); e != nil {
			return nil, e
		}
		if e := add("CreateObjective", "browser_command", 20, creationCondition); e != nil {
			return nil, e
		}
		criterionCondition := inspectContractMutation(ctx, s, f.u, "AddCriterion", AddCriterionCommand{Owner: record.Ref()}, f.at)
		if criterionCondition == nil {
			criterionCondition = record.CheckCriterionDefinitionEdit()
		}
		criterionPriority := 15
		if !record.Criteria.HasRequiredActive() {
			criterionPriority = 5
		}
		if e := add("AddCriterion", "browser_command", criterionPriority, criterionCondition); e != nil {
			return nil, e
		}
		condition := validateObjectiveParentChange(ctx, s, f.u, record, nil, f.at)
		if e := add("SetObjectiveParent", "browser_command", 30, condition); e != nil {
			return nil, e
		}
		if record.Lifecycle == d.ObjectiveLifecyclePlanned || record.Lifecycle == d.ObjectiveLifecycleInProgress {
			condition := requireActiveOutcome(ctx, f.u, record.Scope)
			if condition == nil {
				condition = s.checkObjectiveAchievementWithIssueReport(ctx, f.u, record, identity.PrincipalID, f.objectiveIssues)
			}
			if e := add("AchieveObjective", "browser_command", 15, condition); e != nil {
				return nil, e
			}
		}
	case d.Outcome:
		criterionCondition := inspectContractMutation(ctx, s, f.u, "AddCriterion", AddCriterionCommand{Owner: record.Ref()}, f.at)
		if criterionCondition == nil {
			criterionCondition = record.CheckCriterionDefinitionEdit()
		}
		criterionPriority := 20
		if !record.Criteria.HasRequiredActive() {
			criterionPriority = 5
		}
		if e := add("AddCriterion", "browser_command", criterionPriority, criterionCondition); e != nil {
			return nil, e
		}
		clone := record
		_, editCondition := clone.UpdateDetails(nil, nil, nil, nil, f.at)
		if editCondition == nil {
			editCondition = inspectContractMutation(ctx, s, f.u, "UpdateOutcome", UpdateOutcomeCommand{Scope: record.Scope()}, f.at)
		}
		if e := add("UpdateOutcome", "browser_command", 25, editCondition); e != nil {
			return nil, e
		}
		clone = record
		ownerCondition := clone.SetOwners(record.OwnerRefs, f.at)
		if e := add("SetOutcomeOwners", "browser_command", 30, ownerCondition); e != nil {
			return nil, e
		}
		for _, command := range []string{"CreateObjective", "CreateWorkItem", "CreatePlanItems"} {
			if e := add(command, "browser_command", 20, open()); e != nil {
				return nil, e
			}
		}
		if record.Lifecycle == d.OutcomeLifecycleDraft {
			condition := record.CheckActivation()
			if e := add("ActivateOutcome", "browser_command", 10, condition); e != nil {
				return nil, e
			}
		}
		if record.Lifecycle == d.OutcomeLifecycleActive {
			_, condition := s.checkOutcomeAchievement(ctx, f.u, record, identity.PrincipalID)
			if e := add("AchieveOutcome", "browser_command", 15, condition); e != nil {
				return nil, e
			}
		}
	case d.Issue:
		if record.Lifecycle == d.IssueLifecycleOpen {
			if e := add("InvestigateIssue", "browser_command", 10, documentaryCondition); e != nil {
				return nil, e
			}
		}
		if record.Lifecycle == d.IssueLifecycleOpen || record.Lifecycle == d.IssueLifecycleInvestigating {
			for _, command := range []string{"ResolveIssue", "ResolveIssueAndBlockers", "MarkIssueWontFix", "MarkIssueDuplicate"} {
				priority := 25
				if command == "ResolveIssue" {
					priority = 10
				}
				if e := add(command, "browser_command", priority, documentaryCondition); e != nil {
					return nil, e
				}
			}
		} else {
			if e := add("ReopenIssue", "browser_command", 15, documentaryCondition); e != nil {
				return nil, e
			}
		}
	case d.Blocker:
		if record.Lifecycle == d.BlockerLifecycleActive {
			if e := add("ResolveBlocker", "browser_command", 10, open()); e != nil {
				return nil, e
			}
			if e := add("CancelBlocker", "browser_command", 60, open()); e != nil {
				return nil, e
			}
		}
	case d.Evidence:
		if record.Lifecycle == d.EvidenceLifecycleRegistered {
			if e := add("RetractEvidence", "browser_command", 80, documentaryCondition); e != nil {
				return nil, e
			}
		}
	case d.Artifact:
		if record.Lifecycle == d.ArtifactLifecycleRegistered {
			if e := add("WithdrawArtifact", "browser_command", 80, documentaryCondition); e != nil {
				return nil, e
			}
		}
	case d.Decision:
		if record.Lifecycle == d.DecisionLifecycleProposed {
			if e := add("AcceptDecision", "browser_command", 10, documentaryCondition); e != nil {
				return nil, e
			}
			if e := add("RejectDecision", "browser_command", 60, documentaryCondition); e != nil {
				return nil, e
			}
		}
	case d.Relation:
		if record.Lifecycle == d.RelationLifecycleActive && record.RelationType == d.RelationTypeDependsOn {
			condition := open()
			if condition == nil {
				condition = inspectContractMutation(ctx, s, f.u, "RemoveDependency", RemoveDependencyCommand{Scope: record.Scope, RelationID: record.ID}, f.at)
			}
			if condition == nil {
				state, e := dependencyEndpointState(ctx, f.u, record.SourceRef)
				if e != nil {
					return nil, e
				}
				if state.terminal {
					condition = d.NewDetailedError(d.ErrorCodeInvalidTransition, "reopen the dependency source before changing dependencies", d.ErrorDetails{ReasonCode: "entity_terminal", SubjectRef: refPointer(record.SourceRef)})
				}
			}
			if e := add("RemoveDependency", "browser_command", 80, condition); e != nil {
				return nil, e
			}
		}
	case d.EvidenceLink:
		if record.Lifecycle == d.EvidenceLinkLifecycleActive {
			if e := add("RetractEvidenceLink", "browser_command", 80, documentaryCondition); e != nil {
				return nil, e
			}
		}
	}
	return result, nil
}
