package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type commandMetadata struct {
	Name            string
	NamespaceID     domain.ID
	Scope           domain.Scope
	Owner           *domain.EntityRef
	ExpectedVersion *domain.Version
	Payload         json.RawMessage
	Fingerprint     string
}

func transactCommand[T any, C any](
	ctx context.Context,
	service *Service,
	commandContext domain.CommandContext,
	command C,
	fn func(ports.UnitOfWork) (T, domain.OutcomeRevision, error),
) (MutationResult[T], error) {
	var zero T

	meta, err := buildCommandMetadata(commandContext, command)
	if err != nil {
		return MutationResult[T]{}, err
	}

	uow, err := service.tx.Begin(ctx)
	if err != nil {
		return MutationResult[T]{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	var reservation domain.IdempotencyReservation
	if commandContext.IdempotencyKey != "" {
		identity := domain.IdempotencyIdentity{
			NamespaceID:    meta.NamespaceID,
			PrincipalID:    commandContext.PrincipalID,
			CommandName:    meta.Name,
			IdempotencyKey: commandContext.IdempotencyKey,
		}
		reservation, err = uow.Idempotency().Reserve(ctx, identity, meta.Fingerprint)
		if err != nil {
			return MutationResult[T]{Value: zero}, err
		}
		if reservation.IsReplay() {
			var replay MutationResult[T]
			if err := json.Unmarshal(reservation.Replay.ResponseJSON, &replay); err != nil {
				return MutationResult[T]{Value: zero}, domain.WrapError(domain.ErrorCodeIdempotencyState, "stored command result cannot be decoded", err)
			}
			replay.CommandID = reservation.Replay.CommandID
			replay.OutcomeRevision = reservation.Replay.OutcomeRevision
			replay.IdempotentReplay = true
			return replay, nil
		}
	}

	value, revision, err := fn(uow)
	if err != nil {
		return MutationResult[T]{Value: zero}, err
	}

	events, err := eventsForCommand(service, commandContext, meta, value, revision)
	if err != nil {
		return MutationResult[T]{Value: zero}, err
	}
	if len(events) > 0 {
		if err := uow.Events().Append(ctx, events); err != nil {
			return MutationResult[T]{Value: zero}, err
		}
	}

	result := MutationResult[T]{
		Value:           value,
		OutcomeRevision: revision,
		CommandID:       commandContext.CommandID,
	}
	if commandContext.IdempotencyKey != "" {
		response, err := json.Marshal(result)
		if err != nil {
			return MutationResult[T]{Value: zero}, domain.WrapError(domain.ErrorCodeIdempotencyState, "command result cannot be serialized", err)
		}
		if err := uow.Idempotency().Complete(ctx, reservation, domain.StoredCommandResult{
			CommandID:       commandContext.CommandID,
			OutcomeRevision: revision,
			ResponseJSON:    response,
		}); err != nil {
			return MutationResult[T]{Value: zero}, err
		}
	}

	if err := uow.Commit(); err != nil {
		return MutationResult[T]{Value: zero}, err
	}
	committed = true
	return result, nil
}

func buildCommandMetadata[C any](commandContext domain.CommandContext, command C) (commandMetadata, error) {
	value := reflect.ValueOf(command)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return commandMetadata{}, domain.NewError(domain.ErrorCodeInvalidArgument, "command cannot be nil")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return commandMetadata{}, domain.NewError(domain.ErrorCodeInvalidArgument, "command must be a struct")
	}

	name := value.Type().Name()
	name = strings.TrimSuffix(name, "Command")
	if name == "" {
		return commandMetadata{}, domain.NewError(domain.ErrorCodeInvalidArgument, "command name cannot be empty")
	}

	meta := commandMetadata{Name: name}
	if field := value.FieldByName("NamespaceID"); field.IsValid() {
		meta.NamespaceID = field.Interface().(domain.ID)
	}
	if field := value.FieldByName("Scope"); field.IsValid() {
		meta.Scope = field.Interface().(domain.Scope)
		meta.NamespaceID = meta.Scope.NamespaceID
	}
	if field := value.FieldByName("Owner"); field.IsValid() {
		owner := field.Interface().(domain.EntityRef)
		meta.Owner = &owner
		meta.Scope = owner.Scope
		meta.NamespaceID = owner.NamespaceID
	}
	if field := value.FieldByName("ExpectedVersion"); field.IsValid() {
		expected := field.Interface().(domain.Version)
		meta.ExpectedVersion = &expected
	}
	if err := meta.NamespaceID.Validate(); err != nil {
		return commandMetadata{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "command namespace cannot be resolved", err)
	}

	normalized, err := normalizedCommandJSON(command)
	if err != nil {
		return commandMetadata{}, err
	}
	meta.Payload = normalized

	fingerprintInput := struct {
		CommandName string          `json:"command_name"`
		Actor       domain.ActorRef `json:"actor_ref"`
		Payload     json.RawMessage `json:"payload"`
	}{
		CommandName: meta.Name,
		Actor:       commandContext.Actor,
		Payload:     normalized,
	}
	encoded, err := json.Marshal(fingerprintInput)
	if err != nil {
		return commandMetadata{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "command fingerprint cannot be encoded", err)
	}
	sum := sha256.Sum256(encoded)
	meta.Fingerprint = hex.EncodeToString(sum[:])
	return meta, nil
}

func normalizedCommandJSON[C any](command C) (json.RawMessage, error) {
	// Go's encoding/json provides deterministic struct field order and sorted map
	// keys. For declarative actor sets, sort a copy so ordering alone does not
	// change the idempotency fingerprint.
	switch value := any(command).(type) {
	case SetOutcomeOwnersCommand:
		value.OwnerRefs = sortedActorRefs(value.OwnerRefs)
		return marshalNormalized(value)
	case SetObjectiveOwnersCommand:
		value.OwnerRefs = sortedActorRefs(value.OwnerRefs)
		return marshalNormalized(value)
	case SetWorkItemAssigneesCommand:
		value.AssigneeRefs = sortedActorRefs(value.AssigneeRefs)
		return marshalNormalized(value)
	default:
		return marshalNormalized(command)
	}
}

func marshalNormalized(value any) (json.RawMessage, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, domain.WrapError(domain.ErrorCodeInvalidArgument, "command payload cannot be encoded", err)
	}
	return encoded, nil
}

func sortedActorRefs(values []domain.ActorRef) []domain.ActorRef {
	result := append([]domain.ActorRef(nil), values...)
	sort.Slice(result, func(i, j int) bool {
		left := fmt.Sprintf("%s\x00%s\x00%s", result[i].Kind, result[i].Provider, result[i].ID)
		right := fmt.Sprintf("%s\x00%s\x00%s", result[j].Kind, result[j].Provider, result[j].ID)
		return left < right
	})
	return result
}

func eventsForCommand[T any](
	s *Service,
	commandContext domain.CommandContext,
	meta commandMetadata,
	value T,
	revision domain.OutcomeRevision,
) ([]domain.DomainEvent, error) {
	if events, handled, err := compoundIssueBlockerEvents(s, commandContext, meta, value, revision); handled || err != nil {
		return events, err
	}
	if events, handled, err := compoundDocumentaryEvents(s, commandContext, meta, value, revision); handled || err != nil {
		return events, err
	}

	ref, before, after, noOp, err := commandAggregate(meta, value)
	if err != nil {
		return nil, err
	}
	if noOp {
		return nil, nil
	}
	if revision == 0 {
		return nil, domain.NewError(domain.ErrorCodeInvalidEvent, "confirmed mutation must have an outcome_revision")
	}

	eventTypes, err := eventTypesForCommand(meta)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now().UTC()
	events := make([]domain.DomainEvent, 0, len(eventTypes))
	for index, eventType := range eventTypes {
		eventID, err := s.ids.NewID()
		if err != nil {
			return nil, err
		}
		payload := append(json.RawMessage(nil), meta.Payload...)
		if meta.Name == "CompleteWorkItem" && eventType == "work_item.released" {
			payload = json.RawMessage(`{"reason":"completion"}`)
		}
		if strings.HasSuffix(eventType, ".conclusion_recorded") {
			payload, err = conclusionRecordedPayload(value)
			if err != nil {
				return nil, err
			}
		}
		event := domain.DomainEvent{
			EventID:                eventID,
			EventType:              eventType,
			SchemaVersion:          domain.DomainEventSchemaVersion,
			NamespaceID:            ref.NamespaceID,
			OutcomeID:              ref.OutcomeID,
			OutcomeRevision:        revision,
			EventIndex:             uint32(index),
			AggregateRef:           ref,
			AggregateVersionBefore: before,
			AggregateVersionAfter:  after,
			PrincipalID:            commandContext.PrincipalID,
			Actor:                  commandContext.Actor,
			RecordedAt:             now,
			CommandID:              commandContext.CommandID,
			CorrelationID:          commandContext.CorrelationID,
			CausationID:            cloneIDPointer(commandContext.CausationID),
			ExecutionContext:       commandContext.Execution,
			Payload:                payload,
		}
		if err := event.Validate(); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func commandAggregate[T any](meta commandMetadata, value T) (domain.EntityRef, *domain.Version, *domain.Version, bool, error) {
	switch result := any(value).(type) {
	case domain.Outcome:
		return aggregateVersions(meta, result.Ref(), result.Version)
	case domain.Objective:
		return aggregateVersions(meta, result.Ref(), result.Version)
	case domain.WorkItem:
		return aggregateVersions(meta, result.Ref(), result.Version)
	case domain.Relation:
		return aggregateVersions(meta, result.Ref(), result.Version)
	case domain.Issue:
		return aggregateVersions(meta, result.Ref(), result.Version)
	case domain.Blocker:
		return aggregateVersions(meta, result.Ref(), result.Version)
	case domain.Artifact:
		return aggregateVersions(meta, result.Ref(), result.Version)
	case domain.Evidence:
		return aggregateVersions(meta, result.Ref(), result.Version)
	case domain.EvidenceLink:
		return aggregateVersions(meta, result.Ref(), result.Version)
	case domain.Decision:
		return aggregateVersions(meta, result.Ref(), result.Version)
	case domain.Roadmap:
		return aggregateVersions(meta, result.Ref(), result.Version)
	case domain.RoadmapActiveSlot:
		return resultRoadmapCoordinationAggregate(meta, result.Scope, result.RoadmapID)
	case domain.RoadmapActivationRecord:
		return resultRoadmapCoordinationAggregate(meta, result.Scope, result.RoadmapID)
	case domain.SuccessCriterion:
		if meta.Owner == nil || meta.ExpectedVersion == nil {
			return domain.EntityRef{}, nil, nil, false, domain.NewError(domain.ErrorCodeInvalidEvent, "criterion command is missing owner/version metadata")
		}
		before := *meta.ExpectedVersion
		after := before + 1
		return *meta.Owner, &before, &after, false, nil
	case domain.CriterionAssessment:
		if meta.Owner == nil || meta.ExpectedVersion == nil {
			return domain.EntityRef{}, nil, nil, false, domain.NewError(domain.ErrorCodeInvalidEvent, "assessment command is missing owner/version metadata")
		}
		before := *meta.ExpectedVersion
		after := before + 1
		return *meta.Owner, &before, &after, false, nil
	default:
		return domain.EntityRef{}, nil, nil, false, domain.NewError(domain.ErrorCodeInvalidEvent, "command result does not expose an auditable aggregate")
	}
}

func resultRoadmapCoordinationAggregate(
	meta commandMetadata,
	scope domain.Scope,
	roadmapID domain.ID,
) (domain.EntityRef, *domain.Version, *domain.Version, bool, error) {
	ref := domain.EntityRef{Scope: scope, Kind: domain.EntityKindRoadmap, ID: roadmapID}
	if err := ref.Validate(); err != nil {
		return domain.EntityRef{}, nil, nil, false, err
	}
	return ref, nil, nil, false, nil
}

func aggregateVersions(meta commandMetadata, ref domain.EntityRef, current domain.Version) (domain.EntityRef, *domain.Version, *domain.Version, bool, error) {
	if meta.ExpectedVersion == nil {
		after := current
		return ref, nil, &after, false, nil
	}
	before := *meta.ExpectedVersion
	if current == before {
		return ref, &before, &current, true, nil
	}
	after := current
	return ref, &before, &after, false, nil
}

func eventTypesForCommand(meta commandMetadata) ([]string, error) {
	ownerPrefix := ""
	if meta.Owner != nil {
		ownerPrefix = meta.Owner.Kind.String()
	}
	switch meta.Name {
	case "CreateOutcome":
		return []string{"outcome.created"}, nil
	case "ActivateOutcome":
		return []string{"outcome.activated"}, nil
	case "AchieveOutcome":
		return []string{"outcome.achieved", "outcome.conclusion_recorded"}, nil
	case "FailOutcome":
		return []string{"outcome.failed", "outcome.conclusion_recorded"}, nil
	case "AbandonOutcome":
		return []string{"outcome.abandoned", "outcome.conclusion_recorded"}, nil
	case "ReopenOutcome":
		return []string{"outcome.reopened"}, nil
	case "ArchiveOutcome":
		return []string{"outcome.archived"}, nil
	case "UnarchiveOutcome":
		return []string{"outcome.unarchived"}, nil
	case "SetOutcomeOwners", "UpdateOutcome":
		return []string{"outcome.updated"}, nil
	case "CreateObjective":
		return []string{"objective.created"}, nil
	case "StartObjective":
		return []string{"objective.started"}, nil
	case "AchieveObjective":
		return []string{"objective.achieved", "objective.conclusion_recorded"}, nil
	case "CancelObjective":
		return []string{"objective.cancelled", "objective.conclusion_recorded"}, nil
	case "ReopenObjective":
		return []string{"objective.reopened"}, nil
	case "SetObjectiveOwners", "UpdateObjective":
		return []string{"objective.updated"}, nil
	case "CreateWorkItem":
		return []string{"work_item.created"}, nil
	case "ActivateWorkItem":
		return []string{"work_item.activated"}, nil
	case "DeferWorkItem", "SetWorkItemAssignees", "UpdateWorkItem":
		return []string{"work_item.updated"}, nil
	case "ClaimWorkItem":
		return []string{"work_item.claimed"}, nil
	case "RenewWorkItemLease":
		return []string{"work_item.lease_renewed"}, nil
	case "ReclaimWorkItem":
		return []string{"work_item.reclaimed"}, nil
	case "ReleaseWorkItem":
		return []string{"work_item.released"}, nil
	case "CompleteWorkItem":
		return []string{"work_item.released", "work_item.completed", "work_item.conclusion_recorded"}, nil
	case "AdministrativeCompleteWorkItem":
		return []string{"work_item.admin_completed", "work_item.conclusion_recorded"}, nil
	case "AdministrativeCancelWorkItem":
		return []string{"work_item.admin_cancelled", "work_item.conclusion_recorded"}, nil
	case "CancelWorkItem":
		return []string{"work_item.cancelled", "work_item.conclusion_recorded"}, nil
	case "ReopenWorkItem":
		return []string{"work_item.reopened"}, nil
	case "AddDependency":
		return []string{"relation.created"}, nil
	case "RemoveDependency":
		return []string{"relation.removed"}, nil
	case "CreateIssue":
		return []string{"issue.created"}, nil
	case "UpdateIssue":
		return []string{"issue.updated"}, nil
	case "InvestigateIssue":
		return []string{"issue.investigating"}, nil
	case "ResolveIssue":
		return []string{"issue.resolved"}, nil
	case "MarkIssueWontFix":
		return []string{"issue.wont_fix"}, nil
	case "MarkIssueDuplicate":
		return []string{"issue.duplicate"}, nil
	case "ReopenIssue":
		return []string{"issue.reopened"}, nil
	case "CreateBlocker":
		return []string{"blocker.created"}, nil
	case "UpdateBlockerDescription":
		return []string{"blocker.updated"}, nil
	case "ResolveBlocker":
		return []string{"blocker.resolved"}, nil
	case "CancelBlocker":
		return []string{"blocker.cancelled"}, nil
	case "RegisterArtifact":
		return []string{"artifact.registered"}, nil
	case "WithdrawArtifact":
		return []string{"artifact.withdrawn"}, nil
	case "RegisterEvidence":
		return []string{"evidence.registered"}, nil
	case "RetractEvidence":
		return []string{"evidence.retracted"}, nil
	case "CreateEvidenceLink":
		return []string{"evidence.link_created"}, nil
	case "RetractEvidenceLink":
		return []string{"evidence.link_retracted"}, nil
	case "ProposeDecision":
		return []string{"decision.proposed"}, nil
	case "UpdateDecision":
		return []string{"decision.updated"}, nil
	case "AcceptDecision":
		return []string{"decision.accepted"}, nil
	case "RejectDecision":
		return []string{"decision.rejected"}, nil
	case "CreateRoadmap":
		return []string{"roadmap.created"}, nil
	case "OpenRoadmapDraft":
		return []string{"roadmap.draft_opened"}, nil
	case "ReplaceRoadmapDraft":
		return []string{"roadmap.draft_updated"}, nil
	case "DiscardRoadmapDraft":
		return []string{"roadmap.draft_discarded"}, nil
	case "PublishRoadmapDraft":
		return []string{"roadmap.revision_published"}, nil
	case "ActivateRoadmapRevision":
		return []string{"roadmap.revision_activated"}, nil
	case "DeactivateRoadmapRevision":
		return []string{"roadmap.revision_deactivated"}, nil
	case "ArchiveRoadmap":
		return []string{"roadmap.archived"}, nil
	case "ReopenRoadmap":
		return []string{"roadmap.reopened"}, nil
	case "AddCriterion":
		return []string{ownerPrefix + ".criterion_added"}, nil
	case "ReviseCriterion":
		return []string{ownerPrefix + ".criterion_revised"}, nil
	case "RetireCriterion":
		return []string{ownerPrefix + ".criterion_retired"}, nil
	case "RecordCriterionAssessment", "AttestCriterion":
		return []string{ownerPrefix + ".assessment_recorded"}, nil
	default:
		return nil, domain.NewError(domain.ErrorCodeInvalidEvent, "no event mapping exists for command "+meta.Name)
	}
}

func conclusionRecordedPayload[T any](value T) (json.RawMessage, error) {
	var conclusion *domain.Conclusion
	switch aggregate := any(value).(type) {
	case domain.Outcome:
		conclusion = aggregate.CurrentConclusion
	case domain.Objective:
		conclusion = aggregate.CurrentConclusion
	case domain.WorkItem:
		conclusion = aggregate.CurrentConclusion
	default:
		return nil, domain.NewError(domain.ErrorCodeInvalidEvent, "conclusion event requires outcome, objective or work item result")
	}
	if conclusion == nil || conclusion.ID.IsZero() {
		return nil, domain.NewError(domain.ErrorCodeInvalidEvent, "conclusion event requires a public conclusion identity")
	}
	payload, err := json.Marshal(struct {
		Conclusion domain.Conclusion `json:"conclusion"`
	}{
		Conclusion: *conclusion,
	})
	if err != nil {
		return nil, domain.WrapError(domain.ErrorCodeInvalidEvent, "conclusion event payload cannot be encoded", err)
	}
	return payload, nil
}

func cloneIDPointer(value *domain.ID) *domain.ID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func actorRefsEqual(left, right []domain.ActorRef) bool {
	if len(left) != len(right) {
		return false
	}
	leftSorted := sortedActorRefs(left)
	rightSorted := sortedActorRefs(right)
	for i := range leftSorted {
		if leftSorted[i] != rightSorted[i] {
			return false
		}
	}
	return true
}
