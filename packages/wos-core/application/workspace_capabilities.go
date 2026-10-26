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
	add := func(command, transport string, priority int, condition error, targetLifecycle ...d.WorkItemLifecycle) error {
		ok, e := allow(command)
		if e != nil {
			return e
		}
		if !ok {
			return nil
		}
		c := WorkspaceCapability{ActionID: command, Subject: v.Subject, Transport: transport, Availability: "available", ReasonCodes: []string{}, RelatedRefs: []d.EntityRef{}, ObservedVersion: v.EntityVersion, PresentationPriority: priority}
		if len(targetLifecycle) > 0 {
			c.TargetLifecycle = targetLifecycle[0]
		}
		if condition != nil {
			code, ok := d.ErrorCodeOf(condition)
			if !ok {
				return condition
			}
			c.Availability = "blocked"
			c.ReasonCodes = append(c.ReasonCodes, string(code))
			var detail *d.Error
			if errors.As(condition, &detail) && detail.Details != nil {
				if len(detail.Details.ReasonCodes) > 0 {
					c.ReasonCodes = append([]string(nil), detail.Details.ReasonCodes...)
				} else if detail.Details.ReasonCode != "" {
					c.ReasonCodes = []string{detail.Details.ReasonCode}
				}
				c.RelatedRefs = append(c.RelatedRefs, authorizedReadinessRefs(v.Subject.Ref.Scope, detail.Details.RelatedRefs)...)
				for _, remedy := range detail.Details.Remedies {
					if remedyAllowedInScope(v.Subject.Ref.Scope, remedy) {
						c.Remedies = append(c.Remedies, remedy)
					}
				}
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
			activationCondition := protectWorkProtocol(protocol, "ActivateWorkItem")
			if activationCondition == nil {
				readiness, e := evaluateTaskActivation(ctx, f.u, record, f.at)
				if e != nil {
					return nil, e
				}
				activationCondition = operationalReadinessError(readiness, "Task cannot be activated")
			}
			if e := add("ActivateWorkItem", "browser_command", 10, activationCondition); e != nil {
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
			readiness, e := evaluateTaskAcquisition(ctx, f.u, record, f.at)
			if e != nil {
				return nil, e
			}
			condition := taskVerificationProtocolCondition(protocol, command)
			if condition == nil {
				condition = operationalReadinessError(readiness, "Task cannot be acquired")
			}
			if e := add(command, transport, 10, condition); e != nil {
				return nil, e
			}
		}
		if work != nil {
			if candidate, correction := taskVerificationCorrectionCandidate(record, work.Verification); correction {
				condition := taskVerificationProtocolCondition(protocol, "AcquireSignedWorkContract")
				if condition == nil {
					readiness, e := evaluateTaskAcquisition(ctx, f.u, candidate, f.at)
					if e != nil {
						return nil, e
					}
					condition = operationalReadinessError(readiness, "Task correction cannot be acquired")
				}
				if e := add("AcquireSignedWorkContract", "signed_host", 10, condition); e != nil {
					return nil, e
				}
			}
		}
		if work != nil && work.Verification != nil && work.Verification.Pending {
			verification := work.Verification
			if verification.SourceKind == "task_delivery" {
				canAccept, e := allow("AcceptTaskDelivery")
				if e != nil {
					return nil, e
				}
				canChange, e := allow("RequestTaskDeliveryChanges")
				if e != nil {
					return nil, e
				}
				canWithdraw, e := allow("WithdrawTaskDelivery")
				if e != nil {
					return nil, e
				}
				accept := taskVerificationProtocolCondition(protocol, "AcceptTaskDelivery")
				changes := taskVerificationProtocolCondition(protocol, "RequestTaskDeliveryChanges")
				withdraw := taskVerificationProtocolCondition(protocol, "WithdrawTaskDelivery")
				checkAccept, checkChange, checkWithdraw := canAccept && accept == nil, canChange && changes == nil, canWithdraw && withdraw == nil
				if checkAccept || checkChange || checkWithdraw {
					a, c, w := s.nativeTaskVerificationConditions(ctx, f.u, record, verification, f.at, checkAccept, checkChange, checkWithdraw)
					if checkAccept {
						accept = a
					}
					if checkChange {
						changes = c
					}
					if checkWithdraw {
						withdraw = w
					}
				}
				for _, action := range []struct {
					command   string
					condition error
				}{{"AcceptTaskDelivery", accept}, {"RequestTaskDeliveryChanges", changes}, {"WithdrawTaskDelivery", withdraw}} {
					if e := add(action.command, "browser_command", 5, action.condition); e != nil {
						return nil, e
					}
				}
			} else if verification.SourceKind == "signed_review_case" {
				condition := taskVerificationProtocolCondition(protocol, "AcquireSignedReviewContract")
				if condition == nil {
					condition = open()
				}
				if condition == nil && verification.signedReview != nil {
					condition = verification.signedReview.RequireIndependent(identity.PrincipalID, "")
				}
				if condition == nil && verification.ReviewerAuthority == "active" {
					condition = d.NewError(d.ErrorCodePreconditionFailed, "current signed reviewer authority must finish or expire before reacquisition")
				}
				if e := add("AcquireSignedReviewContract", "signed_host", 5, condition); e != nil {
					return nil, e
				}
				if verification.ReviewerAuthority == "active" {
					condition = taskVerificationProtocolCondition(protocol, "ReturnSignedReview")
					if condition == nil {
						condition = open()
					}
					if condition == nil && (verification.Reviewer == nil || verification.Reviewer.PrincipalID != identity.PrincipalID) {
						condition = d.NewError(d.ErrorCodeForbidden, "current review authority belongs to a different Principal")
					}
					if e := add("ReturnSignedReview", "signed_host", 5, condition); e != nil {
						return nil, e
					}
				}
			}
		}
		if record.Lifecycle == d.WorkItemLifecycleInProgress && record.CurrentContractID != nil && work != nil && work.Contract != nil && (work.Verification == nil || !work.Verification.Pending) {
			command := "SubmitTaskForReview"
			if work.Contract.Contract.SignedBinding != nil {
				command = "ReturnSignedWork"
			}
			condition := taskVerificationProtocolCondition(protocol, command)
			if condition == nil {
				condition = open()
			}
			if condition == nil && !work.Contract.ExecutionAllowed {
				condition = d.NewError(d.ErrorCodePreconditionFailed, "current execution authority is expired or blocked")
			}
			if e := add(command, "signed_host", 10, condition); e != nil {
				return nil, e
			}
		}
		if record.Lifecycle == d.WorkItemLifecycleInProgress && !record.ContractsEnabled && record.CurrentLease != nil {
			lease := record.CurrentLease
			clone := record
			leaseCopy := *lease
			clone.CurrentLease = &leaseCopy
			authority, authorityErr := s.resolveLeaseCredentialAuthority(ctx, f.u, d.CommandContext{PrincipalID: identity.PrincipalID, Actor: identity.Actor}, v.Subject.Ref.Scope, lease, "", f.at)
			command := "RenewWorkItemLease"
			condition := error(nil)
			if authorityErr != nil {
				condition = authorityErr
			} else if authority != nil {
				if lease.CredentialBindingVersion == 0 {
					condition = leaseCredentialBindingUnavailable()
				} else if authority.holderID != lease.HolderCredentialID {
					condition = leaseOwnerMismatch()
				} else {
					condition = clone.RenewLeaseForCredential(identity.PrincipalID, authority.holderID, lease.ClaimID, lease.FencingToken, 5*time.Minute, f.at)
				}
			} else {
				condition = clone.RenewLease(identity.PrincipalID, lease.ClaimID, lease.FencingToken, 5*time.Minute, f.at)
			}
			if !lease.ValidAt(f.at) {
				command = "ReclaimWorkItem"
				// Evaluate on a copy with an existing distinct valid ID. No ID is
				// allocated or persisted by this explanatory read.
				if authorityErr != nil {
					condition = authorityErr
				} else if authority != nil {
					condition = clone.ReclaimForCredential(record.ID, identity.PrincipalID, authority.holderID, authority.sessionID, identity.Actor, 5*time.Minute, f.at)
				} else {
					condition = clone.Reclaim(record.ID, identity.PrincipalID, identity.Actor, 5*time.Minute, f.at)
				}
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
			var releaseCondition error
			if authorityErr != nil {
				releaseCondition = authorityErr
			} else if authority != nil {
				if lease.CredentialBindingVersion == 0 {
					releaseCondition = leaseCredentialBindingUnavailable()
				} else if authority.holderID != lease.HolderCredentialID {
					releaseCondition = leaseOwnerMismatch()
				} else {
					releaseCondition = released.ReleaseForCredential(identity.PrincipalID, authority.holderID, lease.ClaimID, lease.FencingToken, f.at)
				}
			} else {
				releaseCondition = released.Release(identity.PrincipalID, lease.ClaimID, lease.FencingToken, f.at)
			}
			if releaseCondition == nil {
				releaseCondition = open()
			}
			if releaseCondition == nil {
				releaseCondition = protectWorkProtocol(protocol, "ReleaseWorkItem")
			}
			if e := add("ReleaseWorkItem", "browser_command", 35, releaseCondition); e != nil {
				return nil, e
			}
			if authorityErr != nil {
				condition = authorityErr
			} else if authority != nil {
				if lease.CredentialBindingVersion == 0 {
					condition = leaseCredentialBindingUnavailable()
				} else if authority.holderID != lease.HolderCredentialID {
					condition = leaseOwnerMismatch()
				} else {
					condition = record.CheckCompletionAuthorityForCredential(identity.PrincipalID, authority.holderID, lease.ClaimID, lease.FencingToken, f.at)
				}
			} else {
				condition = record.CheckCompletionAuthority(identity.PrincipalID, lease.ClaimID, lease.FencingToken, f.at)
			}
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
			submitCondition := condition
			if submitCondition == nil && authority == nil {
				submitCondition = leaseCredentialBindingUnavailable()
			}
			if e := add("SubmitTaskForReview", "browser_command", 5, submitCondition); e != nil {
				return nil, e
			}
			if condition == nil {
				condition = requireTaskDeliveryReviewForCompletion(ctx, f.u, record.Scope, record.ID)
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
		if e := add("CreateWorkItem", "browser_command", 10, creationCondition, d.WorkItemLifecycleBacklog); e != nil {
			return nil, e
		}
		todoCondition := creationCondition
		if todoCondition == nil {
			profileOutcome, err := f.u.Outcomes().Get(ctx, record.Scope.NamespaceID, record.Scope.OutcomeID)
			if err != nil {
				return nil, err
			}
			if err := validateOperationalProfile(profileOutcome); err != nil {
				todoCondition = err
			} else if profileOutcome.OperationalProfile.Effective() == d.OperationalProfileOperationalV2 {
				operationalObjective, err := f.u.Objectives().Get(ctx, record.Scope, record.ID)
				if err != nil {
					return nil, err
				}
				readiness := d.NewReadiness(record.Ref(), f.at)
				addOutcomeEligibility(&readiness, profileOutcome)
				if operationalObjective.Lifecycle == d.ObjectiveLifecyclePlanned {
					readiness.AddReason(d.ReadinessStageScope, d.ReadinessReasonObjectiveNotStarted, operationalObjective.Ref())
				}
				if err := addObjectiveAncestorReadiness(ctx, f.u, record.Scope, operationalObjective, &readiness); err != nil {
					return nil, err
				}
				todoCondition = operationalReadinessError(readiness, "Task cannot be created directly in To Do")
			}
		}
		// Direct creation in To Do requires a complete Task definition, so its
		// target is explicit while availability remains unknown until the form
		// supplies that definition. The write runs the shared activation gate.
		if e := add("CreateWorkItem", "browser_command", 11, todoCondition, d.WorkItemLifecycleTodo); e != nil {
			return nil, e
		}
		if todoCondition == nil {
			for i := len(result) - 1; i >= 0; i-- {
				if result[i].ActionID == "CreateWorkItem" && result[i].TargetLifecycle == d.WorkItemLifecycleTodo {
					result[i].Availability = "unknown"
					result[i].ReasonCodes = []string{string(d.ReadinessReasonTargetDefinitionRequired)}
					break
				}
			}
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
				condition = s.checkObjectiveAchievementInFrame(ctx, f, record, identity.PrincipalID, f.objectiveIssues)
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
		for _, command := range []string{"CreateObjective", "CreatePlanItems"} {
			if e := add(command, "browser_command", 20, open()); e != nil {
				return nil, e
			}
		}
		if e := add("CreateWorkItem", "browser_command", 20, open(), d.WorkItemLifecycleBacklog); e != nil {
			return nil, e
		}
		todoCondition := open()
		if todoCondition == nil {
			if err := validateOperationalProfile(record); err != nil {
				todoCondition = err
			} else if record.OperationalProfile.Effective() == d.OperationalProfileOperationalV2 {
				readiness := d.NewReadiness(record.Ref(), f.at)
				addOutcomeEligibility(&readiness, record)
				todoCondition = operationalReadinessError(readiness, "Task cannot be created directly in To Do")
			}
		}
		if e := add("CreateWorkItem", "browser_command", 21, todoCondition, d.WorkItemLifecycleTodo); e != nil {
			return nil, e
		}
		if todoCondition == nil {
			for i := len(result) - 1; i >= 0; i-- {
				if result[i].ActionID == "CreateWorkItem" && result[i].TargetLifecycle == d.WorkItemLifecycleTodo {
					result[i].Availability = "unknown"
					result[i].ReasonCodes = []string{string(d.ReadinessReasonTargetDefinitionRequired)}
					break
				}
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
