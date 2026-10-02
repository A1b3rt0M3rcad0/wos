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

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
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

	events, err := service.eventsForCommand(commandContext, meta, value, revision)
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

func (s *Service) eventsForCommand[T any](
	commandContext domain.CommandContext,
	meta commandMetadata,
	value T,
	revision domain.OutcomeRevision,
) ([]domain.DomainEvent, error) {
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
			Payload:                 payload,
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
		return []string{"outcome.achieved"}, nil
	case "FailOutcome":
		return []string{"outcome.failed"}, nil
	case "AbandonOutcome":
		return []string{"outcome.abandoned"}, nil
	case "ReopenOutcome":
		return []string{"outcome.reopened"}, nil
	case "ArchiveOutcome":
		return []string{"outcome.archived"}, nil
	case "UnarchiveOutcome":
		return []string{"outcome.unarchived"}, nil
	case "SetOutcomeOwners":
		return []string{"outcome.updated"}, nil
	case "CreateObjective":
		return []string{"objective.created"}, nil
	case "StartObjective":
		return []string{"objective.started"}, nil
	case "AchieveObjective":
		return []string{"objective.achieved"}, nil
	case "CancelObjective":
		return []string{"objective.cancelled"}, nil
	case "ReopenObjective":
		return []string{"objective.reopened"}, nil
	case "SetObjectiveOwners":
		return []string{"objective.updated"}, nil
	case "CreateWorkItem":
		return []string{"work_item.created"}, nil
	case "ActivateWorkItem":
		return []string{"work_item.activated"}, nil
	case "DeferWorkItem", "SetWorkItemAssignees":
		return []string{"work_item.updated"}, nil
	case "ClaimWorkItem":
		return []string{"work_item.claimed"}, nil
	case "ReleaseWorkItem":
		return []string{"work_item.released"}, nil
	case "CompleteWorkItem":
		return []string{"work_item.released", "work_item.completed"}, nil
	case "CancelWorkItem":
		return []string{"work_item.cancelled"}, nil
	case "ReopenWorkItem":
		return []string{"work_item.reopened"}, nil
	case "AddCriterion":
		return []string{ownerPrefix + ".criterion_added"}, nil
	case "ReviseCriterion":
		return []string{ownerPrefix + ".criterion_revised"}, nil
	case "RetireCriterion":
		return []string{ownerPrefix + ".criterion_retired"}, nil
	case "AttestCriterion":
		return []string{ownerPrefix + ".assessment_recorded"}, nil
	default:
		return nil, domain.NewError(domain.ErrorCodeInvalidEvent, "no event mapping exists for command "+meta.Name)
	}
}

func cloneIDPointer(value *domain.ID) *domain.ID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
