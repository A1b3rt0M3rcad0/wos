package application

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type SetObjectiveCompletionModeCommand struct {
	Scope                   d.Scope                   `json:"scope"`
	ObjectiveID             d.ID                      `json:"objective_id"`
	Mode                    d.ObjectiveCompletionMode `json:"mode"`
	ExpectedVersion         d.Version                 `json:"expected_version,string"`
	ExpectedOutcomeRevision *d.OutcomeRevision        `wos:"required" wosnullable:"false" json:"expected_outcome_revision,string"`
	Reason                  string                    `json:"reason"`
}

type ObjectiveCompletionModeChange struct {
	Objective d.Objective                    `json:"objective"`
	Authority d.ObjectiveCompletionAuthority `json:"authority"`
}

type ObjectiveCompletionAuthorityHistory struct {
	ObjectiveID d.ID                             `json:"objective_id"`
	Authorities []d.ObjectiveCompletionAuthority `json:"authorities"`
}

func objectiveCompletionModeEvents(s *Service, commandContext d.CommandContext, meta commandMetadata, result ObjectiveCompletionModeChange, revision d.OutcomeRevision) ([]d.DomainEvent, error) {
	if revision == 0 || result.Objective.Ref().Scope != result.Authority.Scope || result.Objective.ID != result.Authority.ObjectiveID {
		return nil, d.NewError(d.ErrorCodeInvalidEvent, "Objective completion authority event has inconsistent aggregate or revision")
	}
	eventID, err := s.ids.NewID()
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(struct {
		Authority d.ObjectiveCompletionAuthority `json:"authority"`
	}{Authority: result.Authority})
	if err != nil {
		return nil, d.WrapError(d.ErrorCodeInvalidEvent, "Objective completion authority event cannot be encoded", err)
	}
	before := meta.ExpectedVersion
	after := result.Objective.Version
	event := d.DomainEvent{
		EventID: eventID, EventType: "objective.completion_policy_authorized", SchemaVersion: d.DomainEventSchemaVersion,
		NamespaceID: result.Objective.Scope.NamespaceID, OutcomeID: result.Objective.Scope.OutcomeID,
		OutcomeRevision: revision, EventIndex: 0, AggregateRef: result.Objective.Ref(),
		AggregateVersionBefore: before, AggregateVersionAfter: &after,
		PrincipalID: commandContext.PrincipalID, Actor: commandContext.Actor, RecordedAt: result.Authority.AuthorizedAt,
		CommandID: commandContext.CommandID, CorrelationID: commandContext.CorrelationID,
		CausationID: cloneIDPointer(commandContext.CausationID), ExecutionContext: commandContext.Execution, Payload: payload,
	}
	if err := event.Validate(); err != nil {
		return nil, err
	}
	return []d.DomainEvent{event}, nil
}

const maxObjectiveCompletionReasonRunes = 1024

func (s *Service) SetObjectiveCompletionMode(ctx context.Context, commandContext d.CommandContext, cmd SetObjectiveCompletionModeCommand) (MutationResult[ObjectiveCompletionModeChange], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[ObjectiveCompletionModeChange]{}, err
	}
	if err := cmd.Scope.Validate(); err != nil {
		return MutationResult[ObjectiveCompletionModeChange]{}, err
	}
	if err := cmd.ObjectiveID.Validate(); err != nil {
		return MutationResult[ObjectiveCompletionModeChange]{}, err
	}
	if err := cmd.ExpectedVersion.Validate(); err != nil {
		return MutationResult[ObjectiveCompletionModeChange]{}, err
	}
	if cmd.ExpectedOutcomeRevision == nil {
		return MutationResult[ObjectiveCompletionModeChange]{}, d.NewError(d.ErrorCodeInvalidArgument, "expected_outcome_revision is required")
	}
	if !cmd.Mode.Valid() {
		return MutationResult[ObjectiveCompletionModeChange]{}, d.NewError(d.ErrorCodeInvalidArgument, "completion mode must be manual or automatic")
	}
	cmd.Mode = cmd.Mode.Effective()
	cmd.Reason = strings.TrimSpace(cmd.Reason)
	if cmd.Reason == "" || !utf8.ValidString(cmd.Reason) || strings.ContainsRune(cmd.Reason, 0) || utf8.RuneCountInString(cmd.Reason) > maxObjectiveCompletionReasonRunes {
		return MutationResult[ObjectiveCompletionModeChange]{}, d.NewError(d.ErrorCodeInvalidArgument, "completion mode change requires a reason of at most 1024 characters")
	}
	if commandContext.IdempotencyKey == "" {
		return MutationResult[ObjectiveCompletionModeChange]{}, d.NewError(d.ErrorCodeInvalidArgument, "completion mode change requires an idempotency key")
	}
	if cmd.Mode == d.ObjectiveCompletionModeAutomatic && s.independentReviewer {
		return MutationResult[ObjectiveCompletionModeChange]{}, d.NewDetailedError(d.ErrorCodePreconditionFailed, "automatic Objective completion is unavailable while independent human review is required", d.ErrorDetails{ReasonCode: "independent_reviewer_required"})
	}
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (ObjectiveCompletionModeChange, d.OutcomeRevision, error) {
		coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
		if err != nil {
			return ObjectiveCompletionModeChange{}, 0, err
		}
		if coordination.Revision != *cmd.ExpectedOutcomeRevision {
			return ObjectiveCompletionModeChange{}, 0, d.NewError(d.ErrorCodeVersionConflict, "outcome expected_revision is stale")
		}
		objective, err := uow.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return ObjectiveCompletionModeChange{}, 0, err
		}
		if objective.Scope != cmd.Scope || objective.ID != cmd.ObjectiveID {
			return ObjectiveCompletionModeChange{}, 0, d.NewError(d.ErrorCodeInvalidScope, "Objective completion mode target differs from command scope")
		}
		if objective.Version != cmd.ExpectedVersion {
			return ObjectiveCompletionModeChange{}, 0, d.NewError(d.ErrorCodeVersionConflict, "objective expected_version is stale")
		}
		if cmd.Mode == d.ObjectiveCompletionModeAutomatic && objective.CompletionMode.Effective() != d.ObjectiveCompletionModeAutomatic {
			configured, err := uow.Objectives().ListByOutcome(ctx, cmd.Scope)
			if err != nil {
				return ObjectiveCompletionModeChange{}, 0, err
			}
			automaticCount := 0
			for _, candidate := range configured {
				if candidate.Scope != cmd.Scope || candidate.ID.Validate() != nil || candidate.Validate() != nil {
					return ObjectiveCompletionModeChange{}, 0, d.NewError(d.ErrorCodeInvalidConfig, "Objective completion configuration scan is inconsistent")
				}
				if candidate.CompletionMode.Effective() == d.ObjectiveCompletionModeAutomatic {
					automaticCount++
				}
			}
			if automaticCount >= maxSynchronousObjectiveAchievements {
				return ObjectiveCompletionModeChange{}, 0, d.NewDetailedError(d.ErrorCodePreconditionFailed, "this Outcome has reached the synchronous automatic completion limit", d.ErrorDetails{ReasonCode: "objective_completion_budget_exceeded"})
			}
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return ObjectiveCompletionModeChange{}, 0, err
		}
		if err := objective.ValidateCompletionConfiguration(outcome); err != nil && cmd.Mode == d.ObjectiveCompletionModeAutomatic {
			return ObjectiveCompletionModeChange{}, 0, d.WrapError(d.ErrorCodePreconditionFailed, "automatic completion requires an operational_v2 Outcome using strict_work", err)
		}
		if cmd.Mode == d.ObjectiveCompletionModeAutomatic && (outcome.Lifecycle != d.OutcomeLifecycleActive || outcome.IsArchived()) {
			return ObjectiveCompletionModeChange{}, 0, d.NewDetailedError(d.ErrorCodePreconditionFailed, "automatic completion can only be delegated for an active, unarchived Outcome", d.ErrorDetails{ReasonCode: "outcome_not_active"})
		}
		completionStore, ok := uow.(ports.ObjectiveCompletionAuthorityUnitOfWork)
		if !ok {
			return ObjectiveCompletionModeChange{}, 0, d.NewError(d.ErrorCodeInvalidConfig, "Objective completion authority storage is unavailable")
		}
		history, err := completionStore.ObjectiveCompletionAuthorities().ListByObjective(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return ObjectiveCompletionModeChange{}, 0, err
		}
		policyVersion := uint64(1)
		if len(history) > 0 {
			latest := history[len(history)-1]
			if latest.Mode.Effective() != objective.CompletionMode.Effective() {
				return ObjectiveCompletionModeChange{}, 0, d.NewError(d.ErrorCodeInvalidScope, "Objective completion policy projection differs from its authority history")
			}
			policyVersion, err = d.NextObjectiveCompletionPolicyVersion(latest.PolicyVersion)
			if err != nil {
				return ObjectiveCompletionModeChange{}, 0, err
			}
		}
		now, err := s.transactionTime(ctx, uow)
		if err != nil {
			return ObjectiveCompletionModeChange{}, 0, err
		}
		credentialID, policyRevision, err := s.objectiveCompletionDelegator(ctx, uow, commandContext, cmd.Scope, now, cmd.Mode)
		if err != nil {
			return ObjectiveCompletionModeChange{}, 0, err
		}
		nextVersion, err := objective.Version.Next()
		if err != nil {
			return ObjectiveCompletionModeChange{}, 0, err
		}
		objective.CompletionMode = cmd.Mode
		objective.Version = nextVersion
		objective.UpdatedAt = now.UTC()
		if err := objective.Validate(); err != nil {
			return ObjectiveCompletionModeChange{}, 0, err
		}
		if err := uow.Objectives().Save(ctx, objective, cmd.ExpectedVersion); err != nil {
			return ObjectiveCompletionModeChange{}, 0, err
		}
		authorityID, err := s.ids.NewID()
		if err != nil {
			return ObjectiveCompletionModeChange{}, 0, err
		}
		authority := d.ObjectiveCompletionAuthority{
			ID: authorityID, Scope: cmd.Scope, ObjectiveID: cmd.ObjectiveID,
			PolicyVersion: policyVersion, Mode: cmd.Mode, CompletionCycle: objective.EffectiveCompletionCycle(),
			DelegatorPrincipalID: commandContext.PrincipalID, DelegatorActor: commandContext.Actor,
			DelegatorCredentialID: credentialID, CredentialPolicyRevision: policyRevision,
			OperationalProfile: outcome.OperationalProfile.Effective(), CompletionPolicy: outcome.CompletionPolicy.Effective(),
			AutomationActor: d.ObjectiveCompletionAutomationActor(), Reason: cmd.Reason, AuthorizedAt: now.UTC(),
		}
		if err := authority.Validate(); err != nil {
			return ObjectiveCompletionModeChange{}, 0, err
		}
		if err := completionStore.ObjectiveCompletionAuthorities().Append(ctx, authority); err != nil {
			return ObjectiveCompletionModeChange{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return ObjectiveCompletionModeChange{Objective: objective, Authority: authority}, revision, err
	})
}

func (s *Service) objectiveCompletionDelegator(ctx context.Context, uow ports.UnitOfWork, commandContext d.CommandContext, scope d.Scope, now time.Time, mode d.ObjectiveCompletionMode) (*d.ID, d.Version, error) {
	authority, err := s.resolveLeaseCredentialAuthority(ctx, uow, commandContext, scope, nil, "SetObjectiveCompletionMode", now)
	if err != nil {
		return nil, 0, err
	}
	if authority == nil {
		if mode == d.ObjectiveCompletionModeAutomatic {
			return nil, 0, d.NewDetailedError(d.ErrorCodeForbidden, "automatic completion requires an authenticated credential delegation", d.ErrorDetails{ReasonCode: "completion_credential_required"})
		}
		return nil, 0, nil
	}
	registry, err := signingRepository(uow)
	if err != nil {
		return nil, 0, err
	}
	policy, err := registry.CredentialPolicy(ctx, scope.NamespaceID, authority.holderID)
	if err != nil {
		if mode == d.ObjectiveCompletionModeAutomatic {
			return nil, 0, d.WrapError(d.ErrorCodeForbidden, "automatic completion requires a current credential policy", err)
		}
		return nil, 0, err
	}
	permission := ports.PermissionObjectiveConfigureCompletion
	if policy.PrincipalID != commandContext.PrincipalID || !policy.PermitsWithCompatibility(string(permission), "", scope.OutcomeID) {
		return nil, 0, d.NewError(d.ErrorCodeForbidden, "delegating credential policy does not authorize Objective completion configuration")
	}
	credentialID := authority.holderID
	return &credentialID, policy.Version, nil
}

func (s *Service) ListObjectiveCompletionAuthorities(ctx context.Context, scope d.Scope, objectiveID d.ID) (ObjectiveCompletionAuthorityHistory, error) {
	if err := scope.Validate(); err != nil {
		return ObjectiveCompletionAuthorityHistory{}, err
	}
	if err := objectiveID.Validate(); err != nil {
		return ObjectiveCompletionAuthorityHistory{}, err
	}
	if err := s.authorizeScopedRead(ctx, scope); err != nil {
		return ObjectiveCompletionAuthorityHistory{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ObjectiveCompletionAuthorityHistory{}, err
	}
	defer uow.Rollback()
	objective, err := uow.Objectives().Get(ctx, scope, objectiveID)
	if err != nil {
		return ObjectiveCompletionAuthorityHistory{}, err
	}
	if objective.Scope != scope {
		return ObjectiveCompletionAuthorityHistory{}, d.NewError(d.ErrorCodeInvalidScope, "Objective completion authority query scope differs")
	}
	repository, ok := uow.(ports.ObjectiveCompletionAuthorityUnitOfWork)
	if !ok {
		return ObjectiveCompletionAuthorityHistory{}, d.NewError(d.ErrorCodeInvalidConfig, "Objective completion authority storage is unavailable")
	}
	authorities, err := repository.ObjectiveCompletionAuthorities().ListByObjective(ctx, scope, objectiveID)
	if err != nil {
		return ObjectiveCompletionAuthorityHistory{}, err
	}
	return ObjectiveCompletionAuthorityHistory{ObjectiveID: objectiveID, Authorities: authorities}, nil
}
