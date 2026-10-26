package application

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// The first implementation keeps automatic completion bounded to 64
// explicitly delegated Objectives in one Outcome. E39 adds durable
// reconciliation for installations that need a larger cascade budget.
const maxSynchronousObjectiveAchievements = 64

type objectiveCompletionEffect struct {
	Objective     d.Objective
	VersionBefore d.Version
	Authority     d.ObjectiveCompletionAuthority
}

// objectiveCompletionEffectAllowed is intentionally a closed allowlist. The
// result checks matter: a command name alone does not prove that it accepted
// work, resolved an issue, or recorded an applicable assessment.
func objectiveCompletionEffectAllowed(name string, command, value any) bool {
	switch name {
	case "AcceptTaskDelivery":
		result, ok := value.(TaskDeliveryReviewResult)
		return ok && result.Decision.Result == d.TaskDeliveryDecisionAccepted
	case "ReturnSignedReview":
		result, ok := value.(SignedReturnResult)
		return ok && !result.IsDurableReplay() && result.reviewDecision == "approved"
	case "RecordCriterionAssessment":
		cmd, ok := command.(RecordCriterionAssessmentCommand)
		if !ok || cmd.Owner.Kind != d.EntityKindObjective || cmd.Result != d.AssessmentResultMet {
			return false
		}
		assessment, ok := value.(d.CriterionAssessment)
		return ok && assessment.Result == d.AssessmentResultMet
	case "AttestCriterion":
		cmd, ok := command.(AttestCriterionCommand)
		if !ok || cmd.Owner.Kind != d.EntityKindObjective || cmd.Result != d.AssessmentResultMet {
			return false
		}
		assessment, ok := value.(d.CriterionAssessment)
		return ok && assessment.Result == d.AssessmentResultMet
	case "ResolveIssue", "MarkIssueDuplicate":
		issue, ok := value.(d.Issue)
		return ok && (issue.Lifecycle == d.IssueLifecycleResolved || issue.Lifecycle == d.IssueLifecycleDuplicate)
	case "ResolveBlocker", "CancelBlocker":
		blocker, ok := value.(d.Blocker)
		return ok && (blocker.Lifecycle == d.BlockerLifecycleResolved || blocker.Lifecycle == d.BlockerLifecycleCancelled)
	case "ResolveIssueAndBlockers":
		result, ok := value.(ResolveIssueAndBlockersResult)
		return ok && result.Issue.Lifecycle == d.IssueLifecycleResolved
	case "SetObjectiveCompletionMode":
		result, ok := value.(ObjectiveCompletionModeChange)
		return ok && result.Authority.Mode.Effective() == d.ObjectiveCompletionModeAutomatic
	case "AchieveObjective":
		objective, ok := value.(d.Objective)
		return ok && objective.Lifecycle == d.ObjectiveLifecycleAchieved
	case "StartObjective":
		objective, ok := value.(d.Objective)
		return ok && objective.Lifecycle == d.ObjectiveLifecycleInProgress
	default:
		return false
	}
}

func (s *Service) deriveObjectiveCompletionEffects(
	ctx context.Context,
	uow ports.UnitOfWork,
	commandName string,
	command any,
	value any,
	revision d.OutcomeRevision,
) ([]objectiveCompletionEffect, error) {
	if !objectiveCompletionEffectAllowed(commandName, command, value) {
		return nil, nil
	}
	if s.independentReviewer {
		return nil, nil
	}
	var scope d.Scope
	switch typed := command.(type) {
	case SetObjectiveCompletionModeCommand:
		scope = typed.Scope
	case RecordCriterionAssessmentCommand:
		scope = typed.Owner.Scope
	case AttestCriterionCommand:
		scope = typed.Owner.Scope
	case ResolveIssueCommand:
		scope = typed.Scope
	case MarkIssueDuplicateCommand:
		scope = typed.Scope
	case ResolveBlockerCommand:
		scope = typed.Scope
	case CancelBlockerCommand:
		scope = typed.Scope
	case ResolveIssueAndBlockersCommand:
		scope = typed.Scope
	case AchieveObjectiveCommand:
		scope = typed.Scope
	case StartObjectiveCommand:
		scope = typed.Scope
	case AcceptTaskDeliveryCommand:
		scope = typed.Scope
	case ReturnSignedReviewCommand:
		result, ok := value.(SignedReturnResult)
		if !ok || result.work == nil {
			return nil, nil
		}
		scope = result.work.WorkItem.Scope
	default:
		return nil, nil
	}

	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return nil, err
	}
	if coordination.Revision != revision {
		return nil, d.NewError(d.ErrorCodeVersionConflict, "Objective completion effect observed a different Outcome revision")
	}
	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return nil, err
	}
	if outcome.Scope() != scope || outcome.Lifecycle != d.OutcomeLifecycleActive || outcome.IsArchived() ||
		outcome.OperationalProfile.Effective() != d.OperationalProfileOperationalV2 || outcome.CompletionPolicy.Effective() != d.CompletionPolicyStrictWork {
		return nil, nil
	}
	objectives, err := uow.Objectives().ListByOutcome(ctx, scope)
	if err != nil {
		return nil, err
	}
	if err := validateObjectiveCompletionCandidates(scope, objectives); err != nil {
		return nil, err
	}
	sort.Slice(objectives, func(i, j int) bool { return objectives[i].ID.String() < objectives[j].ID.String() })
	candidates := make([]d.Objective, 0, len(objectives))
	for _, objective := range objectives {
		if objective.CompletionMode.Effective() == d.ObjectiveCompletionModeAutomatic {
			candidates = append(candidates, objective)
		}
	}
	if len(candidates) > maxSynchronousObjectiveAchievements {
		return nil, d.NewDetailedError(d.ErrorCodeInvalidConfig, "automatic Objective completion exceeds the synchronous Outcome limit", d.ErrorDetails{ReasonCode: "objective_completion_budget_exceeded"})
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	completionStore, ok := uow.(ports.ObjectiveCompletionAuthorityUnitOfWork)
	if !ok {
		return nil, d.NewError(d.ErrorCodeInvalidConfig, "Objective completion authority storage is unavailable")
	}
	now, err := s.transactionTime(ctx, uow)
	if err != nil {
		return nil, err
	}
	var derived []objectiveCompletionEffect
	for pass := 0; pass <= len(candidates); pass++ {
		changed := false
		for _, candidate := range candidates {
			current, err := uow.Objectives().Get(ctx, scope, candidate.ID)
			if err != nil {
				return nil, err
			}
			if current.Scope != scope || current.ID != candidate.ID || current.CompletionMode.Effective() != d.ObjectiveCompletionModeAutomatic || current.Lifecycle != d.ObjectiveLifecycleInProgress {
				continue
			}
			authority, valid, err := currentAutomaticObjectiveCompletionAuthority(ctx, uow, completionStore, current, outcome, now, s.requireIdentity)
			if err != nil {
				return nil, err
			}
			if !valid {
				continue
			}
			evaluation, err := s.evaluateObjectiveCompletionInUnit(ctx, uow, current, revision, now, maxObjectiveCompletionSampleLimit, nil)
			if err != nil {
				return nil, err
			}
			report := evaluation.Report
			if !report.FactsComplete || !report.FactualEligible || report.Tasks.Total+report.DescendantObjectives.Total == 0 || report.Criteria.Waived != 0 {
				continue
			}
			assessments, err := current.Criteria.RequiredSatisfied()
			if err != nil {
				continue
			}
			allCurrentCriteriaMet := len(assessments) > 0
			for _, assessment := range assessments {
				if assessment.Result != d.AssessmentResultMet {
					allCurrentCriteriaMet = false
					break
				}
			}
			if !allCurrentCriteriaMet {
				continue
			}
			before := current.Version
			conclusionID, err := s.ids.NewID()
			if err != nil {
				return nil, err
			}
			conclusion := d.Conclusion{
				ID: conclusionID, PrincipalID: d.ObjectiveCompletionAutomationPrincipalID,
				Actor: d.ObjectiveCompletionAutomationActor(), Reason: "Deterministic Objective completion policy satisfied all current obligations.",
				ConcludedAt: now,
				DerivedCompletion: &d.ObjectiveCompletionDerivation{
					AuthorityID: authority.ID, Scope: scope, PolicyVersion: authority.PolicyVersion,
					CompletionCycle: current.EffectiveCompletionCycle(), DelegatorPrincipalID: authority.DelegatorPrincipalID,
					DelegatorCredentialID: *authority.DelegatorCredentialID, CredentialPolicyRevision: authority.CredentialPolicyRevision,
					EvaluatedAt: now,
				},
			}
			if err := achieveObjectiveInUnit(ctx, uow, &current, before, conclusion, now); err != nil {
				return nil, err
			}
			derived = append(derived, objectiveCompletionEffect{Objective: current, VersionBefore: before, Authority: authority})
			changed = true
		}
		if !changed {
			return derived, nil
		}
		if len(derived) > maxSynchronousObjectiveAchievements {
			return nil, d.NewDetailedError(d.ErrorCodeInvalidConfig, "automatic Objective completion exceeded its synchronous Outcome limit", d.ErrorDetails{ReasonCode: "objective_completion_budget_exceeded"})
		}
	}
	return nil, d.NewError(d.ErrorCodeInvalidConfig, "automatic Objective completion did not converge within the bounded Outcome scan")
}

func validateObjectiveCompletionCandidates(scope d.Scope, objectives []d.Objective) error {
	seen := make(map[d.ID]bool, len(objectives))
	for _, objective := range objectives {
		if objective.Scope != scope || objective.ID.Validate() != nil || seen[objective.ID] || objective.Validate() != nil {
			return d.NewError(d.ErrorCodeInvalidConfig, "automatic Objective completion candidate scan is inconsistent")
		}
		seen[objective.ID] = true
	}
	return nil
}

func currentAutomaticObjectiveCompletionAuthority(
	ctx context.Context,
	uow ports.UnitOfWork,
	store ports.ObjectiveCompletionAuthorityUnitOfWork,
	objective d.Objective,
	outcome d.Outcome,
	now time.Time,
	requireIdentity bool,
) (d.ObjectiveCompletionAuthority, bool, error) {
	history, err := store.ObjectiveCompletionAuthorities().ListByObjective(ctx, objective.Scope, objective.ID)
	if err != nil {
		return d.ObjectiveCompletionAuthority{}, false, err
	}
	if len(history) == 0 {
		return d.ObjectiveCompletionAuthority{}, false, d.NewError(d.ErrorCodeInvalidConfig, "automatic Objective has no completion authority history")
	}
	for index := range history {
		if err := history[index].Validate(); err != nil || history[index].Scope != objective.Scope || history[index].ObjectiveID != objective.ID ||
			(index > 0 && history[index].PolicyVersion <= history[index-1].PolicyVersion) {
			return d.ObjectiveCompletionAuthority{}, false, d.NewError(d.ErrorCodeInvalidConfig, "Objective completion authority history is inconsistent")
		}
	}
	authority := history[len(history)-1]
	if authority.Mode.Effective() != d.ObjectiveCompletionModeAutomatic || authority.CompletionCycle != objective.EffectiveCompletionCycle() ||
		authority.OperationalProfile.Effective() != outcome.OperationalProfile.Effective() || authority.CompletionPolicy.Effective() != outcome.CompletionPolicy.Effective() ||
		objective.ValidateCompletionConfiguration(outcome) != nil || authority.DelegatorCredentialID == nil {
		return authority, false, nil
	}
	registry, err := signingRepository(uow)
	if err != nil {
		return d.ObjectiveCompletionAuthority{}, false, err
	}
	credential, err := registry.Credential(ctx, objective.Scope.NamespaceID, *authority.DelegatorCredentialID)
	if err != nil {
		if code, _ := d.ErrorCodeOf(err); code == d.ErrorCodeNotFound {
			return authority, false, nil
		}
		return d.ObjectiveCompletionAuthority{}, false, err
	}
	if credential.ID != *authority.DelegatorCredentialID || credential.NamespaceID != objective.Scope.NamespaceID || credential.PrincipalID != authority.DelegatorPrincipalID ||
		credential.ParentDigest != "" || credential.Revoked || !now.Before(credential.ExpiresAt) || credential.Digest == "" {
		return authority, false, nil
	}
	policy, err := registry.CredentialPolicy(ctx, objective.Scope.NamespaceID, credential.ID)
	if err != nil {
		if code, _ := d.ErrorCodeOf(err); code == d.ErrorCodeNotFound {
			return authority, false, nil
		}
		return d.ObjectiveCompletionAuthority{}, false, err
	}
	if err := policy.Validate(); err != nil {
		return d.ObjectiveCompletionAuthority{}, false, d.WrapError(d.ErrorCodeInvalidConfig, "Objective completion credential policy is invalid", err)
	}
	if policy.PrincipalID != authority.DelegatorPrincipalID || policy.Version != authority.CredentialPolicyRevision ||
		!policy.PermitsWithCompatibility(string(ports.PermissionObjectiveConfigureCompletion), "", objective.Scope.OutcomeID) {
		return authority, false, nil
	}
	if requireIdentity {
		access, ok := uow.(ports.AccessSnapshotUnitOfWork)
		if !ok {
			return d.ObjectiveCompletionAuthority{}, false, d.NewError(d.ErrorCodeInvalidConfig, "Objective completion requires transactional credential authorization")
		}
		if err := access.AuthorizeAccessSnapshot(ctx, ports.AccessSnapshotRequest{
			Authorization:    ports.AuthorizationRequest{NamespaceID: objective.Scope.NamespaceID, OutcomeID: objective.Scope.OutcomeID, PrincipalID: authority.DelegatorPrincipalID, Permission: ports.PermissionObjectiveConfigureCompletion},
			CredentialDigest: credential.Digest, Actor: credential.Actor, Now: now.UTC(),
		}); err != nil {
			if code, _ := d.ErrorCodeOf(err); code == d.ErrorCodeForbidden {
				return authority, false, nil
			}
			return d.ObjectiveCompletionAuthority{}, false, err
		}
	}
	return authority, true, nil
}

func achieveObjectiveInUnit(ctx context.Context, uow ports.UnitOfWork, objective *d.Objective, expected d.Version, conclusion d.Conclusion, now time.Time) error {
	if objective == nil || objective.Version != expected {
		return d.NewError(d.ErrorCodeVersionConflict, "Objective expected_version is stale")
	}
	if err := objective.Achieve(conclusion, now); err != nil {
		return err
	}
	return uow.Objectives().Save(ctx, *objective, expected)
}

func appendObjectiveCompletionEvents(
	s *Service,
	commandContext d.CommandContext,
	primary []d.DomainEvent,
	effects []objectiveCompletionEffect,
	revision d.OutcomeRevision,
) ([]d.DomainEvent, error) {
	if len(effects) == 0 {
		return primary, nil
	}
	if len(primary) == 0 {
		return nil, d.NewError(d.ErrorCodeInvalidEvent, "derived Objective achievement requires a causal mutation event")
	}
	cause := primary[0].EventID
	for _, effect := range effects {
		conclusion := effect.Objective.CurrentConclusion
		if conclusion == nil || conclusion.DerivedCompletion == nil {
			return nil, d.NewError(d.ErrorCodeInvalidEvent, "derived Objective achievement has no persisted provenance")
		}
		derivationPayload, err := json.Marshal(struct {
			Objective    d.EntityRef                     `json:"objective"`
			ConclusionID d.ID                            `json:"conclusion_id"`
			Assessments  []d.CriterionAssessmentRef      `json:"assessments"`
			Derivation   d.ObjectiveCompletionDerivation `json:"derivation"`
		}{effect.Objective.Ref(), conclusion.ID, conclusion.Assessments, *conclusion.DerivedCompletion})
		if err != nil {
			return nil, d.WrapError(d.ErrorCodeInvalidEvent, "derived Objective achievement event cannot be encoded", err)
		}
		conclusionPayload, err := conclusionRecordedPayload(effect.Objective)
		if err != nil {
			return nil, err
		}
		for _, spec := range []struct {
			eventType string
			payload   json.RawMessage
		}{{"objective.achieved", derivationPayload}, {"objective.conclusion_recorded", conclusionPayload}} {
			eventID, err := s.ids.NewID()
			if err != nil {
				return nil, err
			}
			before, after := effect.VersionBefore, effect.Objective.Version
			event := d.DomainEvent{
				EventID: eventID, EventType: spec.eventType, SchemaVersion: d.DomainEventSchemaVersion,
				NamespaceID: effect.Objective.Scope.NamespaceID, OutcomeID: effect.Objective.Scope.OutcomeID,
				OutcomeRevision: revision, EventIndex: uint32(len(primary)), AggregateRef: effect.Objective.Ref(),
				AggregateVersionBefore: &before, AggregateVersionAfter: &after,
				PrincipalID: d.ObjectiveCompletionAutomationPrincipalID, Actor: d.ObjectiveCompletionAutomationActor(),
				RecordedAt: conclusion.ConcludedAt, CommandID: commandContext.CommandID, CorrelationID: commandContext.CorrelationID,
				CausationID: &cause, ExecutionContext: commandContext.Execution, Payload: spec.payload,
			}
			if err := event.Validate(); err != nil {
				return nil, err
			}
			primary = append(primary, event)
		}
	}
	return primary, nil
}
