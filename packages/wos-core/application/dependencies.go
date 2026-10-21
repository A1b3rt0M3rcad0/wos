package application

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type AddDependencyCommand struct {
	Scope     domain.Scope
	SourceRef domain.EntityRef
	TargetRef domain.EntityRef
	Strength  domain.DependencyStrength
	Reason    string
}

type RemoveDependencyCommand struct {
	Scope           domain.Scope
	RelationID      domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

type DependencyChangeAction string

const (
	DependencyChangeAdd        DependencyChangeAction = "add"
	DependencyChangeRemove     DependencyChangeAction = "remove"
	dependencyChangeMaxItems                          = 128
	dependencyChangeMaxPayload                        = 256 * 1024
)

// DependencyChange describes one proposed graph edit. Add requires source_ref,
// target_ref and strength. Remove requires relation_id, expected_version and
// reason. A batch is validated against its final graph before any write occurs.
type DependencyChange struct {
	Action          DependencyChangeAction    `json:"action"`
	SourceRef       *domain.EntityRef         `json:"source_ref,omitempty"`
	TargetRef       *domain.EntityRef         `json:"target_ref,omitempty"`
	Strength        domain.DependencyStrength `json:"strength,omitempty"`
	RelationID      *domain.ID                `json:"relation_id,omitempty"`
	ExpectedVersion *domain.Version           `json:"expected_version,omitempty,string"`
	Reason          string                    `json:"reason,omitempty"`
}

type DependencySourceVersion struct {
	SourceRef       domain.EntityRef `json:"source_ref"`
	ExpectedVersion domain.Version   `json:"expected_version,string"`
}

type ApplyDependencyChangesCommand struct {
	Scope                   domain.Scope              `json:"scope"`
	ExpectedOutcomeRevision domain.OutcomeRevision    `json:"expected_outcome_revision,string"`
	ExpectedSources         []DependencySourceVersion `json:"expected_sources"`
	Changes                 []DependencyChange        `json:"changes"`
}

type DependencyChangesPreviewQuery struct {
	ExpectedOutcomeRevision domain.OutcomeRevision    `json:"expected_outcome_revision,string"`
	ExpectedSources         []DependencySourceVersion `json:"expected_sources"`
	Changes                 []DependencyChange        `json:"changes"`
}

type DependencyChangePreview struct {
	Action        DependencyChangeAction    `json:"action"`
	SourceRef     domain.EntityRef          `json:"source_ref"`
	TargetRef     domain.EntityRef          `json:"target_ref"`
	Strength      domain.DependencyStrength `json:"strength,omitempty"`
	RelationID    *domain.ID                `json:"relation_id,omitempty"`
	BeforeVersion *domain.Version           `json:"before_version,omitempty,string"`
	AfterVersion  *domain.Version           `json:"after_version,omitempty,string"`
}

type DependencyChangesPreview struct {
	Availability               string                         `json:"availability"`
	OutcomeRevision            domain.OutcomeRevision         `json:"outcome_revision,string"`
	ExpectedOutcomeRevision    domain.OutcomeRevision         `json:"expected_outcome_revision,string"`
	Changes                    []DependencyChangePreview      `json:"changes"`
	FinalActiveDependencyCount int                            `json:"final_active_dependency_count"`
	Error                      *DependencyChangesPreviewError `json:"error,omitempty"`
}

type DependencyChangesPreviewError struct {
	Code    domain.ErrorCode     `json:"code"`
	Message string               `json:"message"`
	Details *domain.ErrorDetails `json:"details,omitempty"`
}

type DependencyChangesResult struct {
	Changes []DependencyChangeResult `json:"changes"`
}

type DependencyChangeResult struct {
	Action        DependencyChangeAction `json:"action"`
	Relation      domain.Relation        `json:"relation"`
	BeforeVersion *domain.Version        `json:"before_version,omitempty,string"`
}

type preparedDependencyChanges struct {
	items                      []DependencyChangeResult
	finalActiveDependencyCount int
}

func (s *Service) AddDependency(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd AddDependencyCommand,
) (MutationResult[domain.Relation], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	if cmd.SourceRef.Scope != cmd.Scope || cmd.TargetRef.Scope != cmd.Scope {
		return MutationResult[domain.Relation]{}, domain.NewError(domain.ErrorCodeInvalidRelation, "dependency endpoints must match command scope")
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	now := s.clock.Now().UTC()
	relation, err := domain.NewDependencyRelation(id, cmd.SourceRef, cmd.TargetRef, cmd.Strength, now)
	if err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	relation.Reason = strings.TrimSpace(cmd.Reason)

	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Relation, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Relation{}, 0, err
		}
		sourceState, err := dependencyEndpointState(ctx, uow, cmd.SourceRef)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		if _, err := dependencyEndpointState(ctx, uow, cmd.TargetRef); err != nil {
			return domain.Relation{}, 0, err
		}
		if sourceState.terminal {
			return domain.Relation{}, 0, domain.NewError(domain.ErrorCodeInvalidTransition, "terminal dependency source must be reopened before changing dependencies")
		}
		if sourceState.inProgress && strings.TrimSpace(cmd.Reason) == "" {
			return domain.Relation{}, 0, domain.NewError(domain.ErrorCodeInvalidArgument, "adding dependency to in-progress source requires reason")
		}
		if err := validateOwnObjectiveDependency(ctx, uow, relation); err != nil {
			return domain.Relation{}, 0, err
		}
		existing, err := uow.Relations().ListByOutcome(ctx, cmd.Scope)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		if err := domain.ValidateDependencyAcyclic(existing, relation, domain.DefaultDependencyGraphLimit); err != nil {
			return domain.Relation{}, 0, err
		}
		if err := uow.Relations().Insert(ctx, relation); err != nil {
			return domain.Relation{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return relation, revision, err
	})
}

func (s *Service) RemoveDependency(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd RemoveDependencyCommand,
) (MutationResult[domain.Relation], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Relation, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Relation{}, 0, err
		}
		relation, err := uow.Relations().Get(ctx, cmd.Scope, cmd.RelationID)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		if relation.RelationType != domain.RelationTypeDependsOn {
			return domain.Relation{}, 0, domain.NewError(domain.ErrorCodeInvalidRelation, "relation is not a dependency")
		}
		sourceState, err := dependencyEndpointState(ctx, uow, relation.SourceRef)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		if sourceState.terminal {
			return domain.Relation{}, 0, domain.NewError(domain.ErrorCodeInvalidTransition, "terminal dependency source must be reopened before changing dependencies")
		}
		if err := relation.Remove(cmd.Reason, now); err != nil {
			return domain.Relation{}, 0, err
		}
		if err := uow.Relations().Save(ctx, relation, cmd.ExpectedVersion); err != nil {
			return domain.Relation{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return relation, revision, err
	})
}

func (s *Service) PreviewDependencyChanges(ctx context.Context, scope domain.Scope, q DependencyChangesPreviewQuery) (DependencyChangesPreview, error) {
	var zero DependencyChangesPreview
	command := ApplyDependencyChangesCommand{Scope: scope, ExpectedOutcomeRevision: q.ExpectedOutcomeRevision, ExpectedSources: q.ExpectedSources, Changes: q.Changes}
	if err := validateDependencyChangesEnvelope(command); err != nil {
		return zero, err
	}
	frame, handled, err := s.openContinuityFrame(ctx, scope)
	if err != nil {
		return zero, err
	}
	if !handled {
		return zero, domain.NewError(domain.ErrorCodeInvalidConfig, "dependency change preview requires a continuity read adapter")
	}
	defer frame.u.Rollback()
	preview := DependencyChangesPreview{
		Availability:            "available",
		OutcomeRevision:         frame.revision,
		ExpectedOutcomeRevision: q.ExpectedOutcomeRevision,
		Changes:                 []DependencyChangePreview{},
	}
	if frame.revision != q.ExpectedOutcomeRevision {
		err = domain.NewError(domain.ErrorCodePreconditionFailed, "snapshot_changed")
		preview.Availability, preview.Error = "blocked", dependencyPreviewError(err)
		return preview, nil
	}
	if err = inspectContractMutation(ctx, s, frame.u, "ApplyDependencyChanges", command, frame.at); err != nil {
		if blocked, ok := dependencyPreviewBlockingError(err); ok {
			preview.Availability, preview.Error = "blocked", blocked
			return preview, nil
		}
		return zero, err
	}
	prepared, err := s.prepareDependencyChanges(ctx, frame.u, command, frame.at)
	if err != nil {
		if blocked, ok := dependencyPreviewBlockingError(err); ok {
			preview.Availability, preview.Error = "blocked", blocked
			return preview, nil
		}
		return zero, err
	}
	preview.FinalActiveDependencyCount = prepared.finalActiveDependencyCount
	for _, change := range prepared.items {
		item := DependencyChangePreview{Action: change.Action, SourceRef: change.Relation.SourceRef, TargetRef: change.Relation.TargetRef, Strength: change.Relation.Strength, AfterVersion: versionPtr(change.Relation.Version)}
		if change.Action == DependencyChangeRemove {
			item.RelationID = cloneIDPointer(&change.Relation.ID)
			item.BeforeVersion = cloneVersionPointer(change.BeforeVersion)
		}
		preview.Changes = append(preview.Changes, item)
	}
	return preview, nil
}

func (s *Service) ApplyDependencyChanges(ctx context.Context, cc domain.CommandContext, cmd ApplyDependencyChangesCommand) (MutationResult[DependencyChangesResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[DependencyChangesResult]{}, err
	}
	if err := validateDependencyChangesEnvelope(cmd); err != nil {
		return MutationResult[DependencyChangesResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (DependencyChangesResult, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return DependencyChangesResult{}, 0, err
		}
		coord, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
		if err != nil {
			return DependencyChangesResult{}, 0, err
		}
		if coord.Revision != cmd.ExpectedOutcomeRevision {
			return DependencyChangesResult{}, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "snapshot_changed")
		}
		prepared, err := s.prepareDependencyChanges(ctx, uow, cmd, s.clock.Now().UTC())
		if err != nil {
			return DependencyChangesResult{}, 0, err
		}
		result := DependencyChangesResult{Changes: append([]DependencyChangeResult(nil), prepared.items...)}
		for _, change := range prepared.items {
			if change.Action == DependencyChangeRemove {
				if change.BeforeVersion == nil {
					return DependencyChangesResult{}, 0, domain.NewError(domain.ErrorCodeInvalidConfig, "dependency removal is missing its expected version")
				}
				if err := uow.Relations().Save(ctx, change.Relation, *change.BeforeVersion); err != nil {
					return DependencyChangesResult{}, 0, err
				}
			}
		}
		for _, change := range prepared.items {
			if change.Action == DependencyChangeAdd {
				if err := uow.Relations().Insert(ctx, change.Relation); err != nil {
					return DependencyChangesResult{}, 0, err
				}
			}
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return result, revision, err
	})
}

func validateDependencyChangesEnvelope(cmd ApplyDependencyChangesCommand) error {
	if err := cmd.Scope.Validate(); err != nil {
		return err
	}
	if cmd.ExpectedOutcomeRevision == 0 {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "expected_outcome_revision is required")
	}
	if len(cmd.Changes) == 0 || len(cmd.Changes) > dependencyChangeMaxItems {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "dependency change batch must contain between 1 and 128 changes")
	}
	if len(cmd.ExpectedSources) == 0 || len(cmd.ExpectedSources) > dependencyChangeMaxItems {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "expected_sources must contain between 1 and 128 source versions")
	}
	encoded, err := json.Marshal(cmd)
	if err != nil {
		return domain.WrapError(domain.ErrorCodeInvalidArgument, "dependency change batch cannot be encoded", err)
	}
	if len(encoded) > dependencyChangeMaxPayload {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "dependency change batch exceeds 256 KiB")
	}
	sources := map[domain.EntityRef]bool{}
	for _, expected := range cmd.ExpectedSources {
		if err := expected.SourceRef.Validate(); err != nil || expected.SourceRef.Scope != cmd.Scope || (expected.SourceRef.Kind != domain.EntityKindObjective && expected.SourceRef.Kind != domain.EntityKindWorkItem) || expected.ExpectedVersion == 0 {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "expected_sources contains an invalid source reference or version")
		}
		if sources[expected.SourceRef] {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "expected_sources contains a duplicate source")
		}
		sources[expected.SourceRef] = true
	}
	removed := map[domain.ID]bool{}
	for _, change := range cmd.Changes {
		switch change.Action {
		case DependencyChangeAdd:
			if change.SourceRef == nil || change.TargetRef == nil || change.RelationID != nil || change.ExpectedVersion != nil || !change.Strength.Valid() {
				return domain.NewError(domain.ErrorCodeInvalidArgument, "add dependency change requires source_ref, target_ref and valid strength only")
			}
			for _, ref := range []*domain.EntityRef{change.SourceRef, change.TargetRef} {
				if err := ref.Validate(); err != nil || ref.Scope != cmd.Scope || (ref.Kind != domain.EntityKindObjective && ref.Kind != domain.EntityKindWorkItem) {
					return domain.NewError(domain.ErrorCodeInvalidRelation, "dependency endpoints must be Objectives or Tasks in this Outcome")
				}
			}
		case DependencyChangeRemove:
			if change.RelationID == nil || change.ExpectedVersion == nil || change.SourceRef != nil || change.TargetRef != nil || change.Strength != "" || *change.ExpectedVersion == 0 || strings.TrimSpace(change.Reason) == "" {
				return domain.NewError(domain.ErrorCodeInvalidArgument, "remove dependency change requires relation_id, expected_version and reason only")
			}
			if err := change.RelationID.Validate(); err != nil {
				return domain.WrapError(domain.ErrorCodeInvalidArgument, "dependency removal relation_id is invalid", err)
			}
			if removed[*change.RelationID] {
				return domain.NewError(domain.ErrorCodeInvalidArgument, "dependency removal relation_id is repeated")
			}
			removed[*change.RelationID] = true
		default:
			return domain.NewError(domain.ErrorCodeInvalidArgument, "dependency change action must be add or remove")
		}
	}
	return nil
}

func (s *Service) prepareDependencyChanges(ctx context.Context, uow ports.UnitOfWork, cmd ApplyDependencyChangesCommand, now time.Time) (preparedDependencyChanges, error) {
	var result preparedDependencyChanges
	relations, err := uow.Relations().ListByOutcome(ctx, cmd.Scope)
	if err != nil {
		return result, err
	}
	byID := make(map[domain.ID]domain.Relation, len(relations))
	final := make([]domain.Relation, 0, len(relations)+len(cmd.Changes))
	for _, relation := range relations {
		byID[relation.ID] = relation
		if relation.Lifecycle == domain.RelationLifecycleActive && relation.RelationType == domain.RelationTypeDependsOn {
			final = append(final, relation)
		}
	}
	result.items = make([]DependencyChangeResult, 0, len(cmd.Changes))
	removed := map[domain.ID]bool{}
	for _, change := range cmd.Changes {
		if change.Action != DependencyChangeRemove {
			continue
		}
		relation, exists := byID[*change.RelationID]
		if !exists {
			return result, domain.NewError(domain.ErrorCodeNotFound, "dependency relation was not found")
		}
		if relation.RelationType != domain.RelationTypeDependsOn || relation.Lifecycle != domain.RelationLifecycleActive {
			return result, domain.NewError(domain.ErrorCodeInvalidRelation, "only an active dependency can be removed")
		}
		if relation.Version != *change.ExpectedVersion {
			return result, domain.NewError(domain.ErrorCodePreconditionFailed, "dependency relation version conflict")
		}
		before := relation.Version
		if err := relation.Remove(change.Reason, now); err != nil {
			return result, err
		}
		removed[relation.ID] = true
		result.items = append(result.items, DependencyChangeResult{Action: DependencyChangeRemove, Relation: relation, BeforeVersion: versionPtr(before)})
	}
	for _, change := range cmd.Changes {
		if change.Action != DependencyChangeAdd {
			continue
		}
		id, err := s.ids.NewID()
		if err != nil {
			return result, err
		}
		relation, err := domain.NewDependencyRelation(id, *change.SourceRef, *change.TargetRef, change.Strength, now)
		if err != nil {
			return result, err
		}
		relation.Reason = strings.TrimSpace(change.Reason)
		sourceState, err := dependencyEndpointState(ctx, uow, relation.SourceRef)
		if err != nil {
			return result, err
		}
		if sourceState.terminal {
			return result, domain.NewError(domain.ErrorCodeInvalidTransition, "terminal dependency source must be reopened before changing dependencies")
		}
		if sourceState.inProgress && relation.Reason == "" {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "adding dependency to in-progress source requires reason")
		}
		if _, err := dependencyEndpointState(ctx, uow, relation.TargetRef); err != nil {
			return result, err
		}
		if err := validateOwnObjectiveDependency(ctx, uow, relation); err != nil {
			return result, err
		}
		result.items = append(result.items, DependencyChangeResult{Action: DependencyChangeAdd, Relation: relation})
	}
	final = final[:0]
	for _, relation := range relations {
		if relation.Lifecycle == domain.RelationLifecycleActive && relation.RelationType == domain.RelationTypeDependsOn && !removed[relation.ID] {
			final = append(final, relation)
		}
	}
	for _, change := range result.items {
		if change.Action == DependencyChangeAdd {
			final = append(final, change.Relation)
		}
	}
	if err := validateDependencyGraphAcyclic(final, cmd.Scope); err != nil {
		return result, err
	}
	result.finalActiveDependencyCount = len(final)
	// Restore the caller's requested ordering so emitted events remain stable.
	removedByID := map[domain.ID]DependencyChangeResult{}
	adds := make([]DependencyChangeResult, 0)
	for _, item := range result.items {
		if item.Action == DependencyChangeRemove {
			removedByID[item.Relation.ID] = item
		} else {
			adds = append(adds, item)
		}
	}
	ordered := make([]DependencyChangeResult, 0, len(result.items))
	addIndex := 0
	for _, change := range cmd.Changes {
		if change.Action == DependencyChangeRemove {
			ordered = append(ordered, removedByID[*change.RelationID])
			continue
		}
		if addIndex < len(adds) {
			ordered = append(ordered, adds[addIndex])
			addIndex++
		}
	}
	if len(ordered) == len(result.items) {
		result.items = ordered
	}
	return result, nil
}

func validateDependencyChangesContract(ctx context.Context, s *Service, uow ports.UnitOfWork, cmd ApplyDependencyChangesCommand, inspectionAt *time.Time) error {
	if err := validateDependencyChangesEnvelope(cmd); err != nil {
		return err
	}
	coord, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
	if err != nil {
		return err
	}
	if coord.Revision != cmd.ExpectedOutcomeRevision {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "snapshot_changed")
	}
	if inspectionAt == nil {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return err
		}
	} else {
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return err
		}
		if err := validateOpenOutcome(outcome); err != nil {
			return err
		}
	}
	relations, err := uow.Relations().ListByOutcome(ctx, cmd.Scope)
	if err != nil {
		return err
	}
	relationByID := make(map[domain.ID]domain.Relation, len(relations))
	sources := map[domain.EntityRef]bool{}
	addsBySource := map[domain.EntityRef][]DependencyChange{}
	for _, relation := range relations {
		relationByID[relation.ID] = relation
	}
	for _, change := range cmd.Changes {
		if change.Action == DependencyChangeAdd {
			source := *change.SourceRef
			sources[source] = true
			addsBySource[source] = append(addsBySource[source], change)
			continue
		}
		relation, exists := relationByID[*change.RelationID]
		if !exists {
			return domain.NewError(domain.ErrorCodeNotFound, "dependency relation was not found")
		}
		if relation.Scope != cmd.Scope || relation.RelationType != domain.RelationTypeDependsOn || relation.Lifecycle != domain.RelationLifecycleActive {
			return domain.NewError(domain.ErrorCodeInvalidRelation, "only an active dependency in this Outcome can be removed")
		}
		if relation.Version != *change.ExpectedVersion {
			return domain.NewError(domain.ErrorCodePreconditionFailed, "dependency relation version conflict")
		}
		sources[relation.SourceRef] = true
	}
	expected := make(map[domain.EntityRef]domain.Version, len(cmd.ExpectedSources))
	for _, value := range cmd.ExpectedSources {
		expected[value.SourceRef] = value.ExpectedVersion
	}
	if len(sources) != len(expected) {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "expected_sources must contain exactly the distinct dependency sources changed by the batch")
	}
	now := s.clock.Now().UTC()
	if inspectionAt != nil {
		now = inspectionAt.UTC()
	} else if nowValue, err := s.transactionTime(ctx, uow); err != nil {
		return err
	} else {
		now = nowValue
	}
	sourceRefs := make([]domain.EntityRef, 0, len(sources))
	for source := range sources {
		if _, exists := expected[source]; !exists {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "expected_sources is missing a changed dependency source")
		}
		sourceRefs = append(sourceRefs, source)
	}
	sort.Slice(sourceRefs, func(i, j int) bool {
		return dependencyImpactRefKey(sourceRefs[i]) < dependencyImpactRefKey(sourceRefs[j])
	})
	for _, source := range sourceRefs {
		state, err := dependencyEndpointState(ctx, uow, source)
		if err != nil {
			return err
		}
		var currentVersion domain.Version
		switch source.Kind {
		case domain.EntityKindObjective:
			value, err := uow.Objectives().Get(ctx, source.Scope, source.ID)
			if err != nil {
				return err
			}
			currentVersion = value.Version
		case domain.EntityKindWorkItem:
			value, err := uow.WorkItems().Get(ctx, source.Scope, source.ID)
			if err != nil {
				return err
			}
			currentVersion = value.Version
		}
		if currentVersion != expected[source] {
			return domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "dependency source version conflict", domain.ErrorDetails{ReasonCode: "source_version_conflict", SubjectRef: &source})
		}
		if state.terminal {
			return domain.NewError(domain.ErrorCodeInvalidTransition, "terminal dependency source must be reopened before changing dependencies")
		}
		for _, add := range addsBySource[source] {
			if state.inProgress && strings.TrimSpace(add.Reason) == "" {
				return domain.NewError(domain.ErrorCodeInvalidArgument, "adding dependency to in-progress source requires reason")
			}
		}
		if source.Kind == domain.EntityKindWorkItem {
			work, err := uow.WorkItems().Get(ctx, source.Scope, source.ID)
			if err != nil {
				return err
			}
			if err := work.CheckPlanningEdit(); err != nil {
				return err
			}
			if err := protectWorkContractMutation(ctx, uow, work, now, false, false); err != nil {
				return err
			}
			continue
		}
		items, err := uow.WorkItems().ListByOutcome(ctx, cmd.Scope)
		if err != nil {
			return err
		}
		for _, work := range items {
			if work.ObjectiveID == nil || *work.ObjectiveID != source.ID {
				continue
			}
			if err := work.CheckPlanningEdit(); err != nil {
				return err
			}
			if err := protectWorkContractMutation(ctx, uow, work, now, true, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateDependencyGraphAcyclic(relations []domain.Relation, scope domain.Scope) error {
	adjacency := map[domain.EntityRef][]domain.EntityRef{}
	indegree := map[domain.EntityRef]int{}
	edges := 0
	for _, relation := range relations {
		if relation.RelationType != domain.RelationTypeDependsOn || relation.Lifecycle != domain.RelationLifecycleActive {
			continue
		}
		if relation.Scope != scope || relation.SourceRef.Scope != scope || relation.TargetRef.Scope != scope || relation.Validate() != nil {
			return domain.NewError(domain.ErrorCodeInvalidRelation, "dependency graph contains an invalid relation")
		}
		edges++
		if edges > domain.DefaultDependencyGraphLimit {
			return domain.NewError(domain.ErrorCodeGraphLimitExceeded, "dependency graph validation limit exceeded")
		}
		adjacency[relation.SourceRef] = append(adjacency[relation.SourceRef], relation.TargetRef)
		if _, ok := indegree[relation.SourceRef]; !ok {
			indegree[relation.SourceRef] = 0
		}
		indegree[relation.TargetRef]++
	}
	ready := make([]domain.EntityRef, 0, len(indegree))
	for ref, degree := range indegree {
		if degree == 0 {
			ready = append(ready, ref)
		}
	}
	sort.Slice(ready, func(i, j int) bool { return dependencyImpactRefKey(ready[i]) < dependencyImpactRefKey(ready[j]) })
	visited := 0
	for len(ready) > 0 {
		current := ready[0]
		ready = ready[1:]
		visited++
		for _, target := range adjacency[current] {
			indegree[target]--
			if indegree[target] == 0 {
				ready = append(ready, target)
			}
		}
	}
	if visited != len(indegree) {
		return domain.NewError(domain.ErrorCodeDependencyCycle, "dependency batch would create a cycle")
	}
	return nil
}

func dependencyPreviewError(err error) *DependencyChangesPreviewError {
	code, ok := domain.ErrorCodeOf(err)
	if !ok {
		code = domain.ErrorCodeInvalidRelation
	}
	var typed *domain.Error
	var details *domain.ErrorDetails
	message := err.Error()
	if errors.As(err, &typed) {
		message, details = typed.Message, typed.Details
	}
	return &DependencyChangesPreviewError{Code: code, Message: message, Details: details}
}

func dependencyPreviewBlockingError(err error) (*DependencyChangesPreviewError, bool) {
	var typed *domain.Error
	if !errors.As(err, &typed) || typed.Code == domain.ErrorCodeInvalidConfig {
		return nil, false
	}
	return dependencyPreviewError(err), true
}

func dependencyChangesCommandEvents(s *Service, cc domain.CommandContext, meta commandMetadata, value any, revision domain.OutcomeRevision) ([]domain.DomainEvent, bool, error) {
	result, ok := value.(DependencyChangesResult)
	if !ok {
		return nil, false, nil
	}
	specs := make([]compoundEventSpec, 0, len(result.Changes))
	payloads := make([]domain.Relation, 0, len(result.Changes))
	for _, change := range result.Changes {
		eventType := "relation.created"
		var before *domain.Version
		if change.Action == DependencyChangeRemove {
			eventType = "relation.removed"
			before = cloneVersionPointer(change.BeforeVersion)
			if before == nil {
				return nil, true, domain.NewError(domain.ErrorCodeInvalidEvent, "removed dependency event is missing its prior version")
			}
		}
		specs = append(specs, compoundEventSpec{eventType: eventType, ref: change.Relation.Ref(), before: before, after: versionPtr(change.Relation.Version)})
		payloads = append(payloads, change.Relation)
	}
	events, err := buildCompoundEvents(s, cc, meta, revision, specs)
	if err != nil {
		return nil, true, err
	}
	for index := range events {
		events[index].Payload, err = json.Marshal(payloads[index])
		if err != nil {
			return nil, true, domain.WrapError(domain.ErrorCodeInvalidEvent, "dependency change event payload cannot be encoded", err)
		}
		if err := events[index].Validate(); err != nil {
			return nil, true, err
		}
	}
	return events, true, nil
}

func cloneVersionPointer(value *domain.Version) *domain.Version {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

type endpointState struct {
	completed  bool
	terminal   bool
	inProgress bool
}

func dependencyEndpointState(ctx context.Context, uow ports.UnitOfWork, ref domain.EntityRef) (endpointState, error) {
	if err := ref.Validate(); err != nil {
		return endpointState{}, err
	}
	switch ref.Kind {
	case domain.EntityKindObjective:
		value, err := uow.Objectives().Get(ctx, ref.Scope, ref.ID)
		if err != nil {
			return endpointState{}, err
		}
		return endpointState{
			completed:  value.Lifecycle == domain.ObjectiveLifecycleAchieved,
			terminal:   value.Lifecycle == domain.ObjectiveLifecycleAchieved || value.Lifecycle == domain.ObjectiveLifecycleCancelled,
			inProgress: value.Lifecycle == domain.ObjectiveLifecycleInProgress,
		}, nil
	case domain.EntityKindWorkItem:
		value, err := uow.WorkItems().Get(ctx, ref.Scope, ref.ID)
		if err != nil {
			return endpointState{}, err
		}
		return endpointState{
			completed:  value.Lifecycle == domain.WorkItemLifecycleDone,
			terminal:   value.Lifecycle == domain.WorkItemLifecycleDone || value.Lifecycle == domain.WorkItemLifecycleCancelled,
			inProgress: value.Lifecycle == domain.WorkItemLifecycleInProgress,
		}, nil
	default:
		return endpointState{}, domain.NewError(domain.ErrorCodeInvalidRelation, "dependency endpoint must be objective or work_item")
	}
}

func dependencyEvaluationsForSource(
	ctx context.Context,
	uow ports.UnitOfWork,
	source domain.EntityRef,
	relations []domain.Relation,
) ([]domain.DependencyEvaluation, error) {
	result := make([]domain.DependencyEvaluation, 0)
	for _, relation := range relations {
		if relation.Lifecycle != domain.RelationLifecycleActive ||
			relation.RelationType != domain.RelationTypeDependsOn ||
			relation.SourceRef != source {
			continue
		}
		state, err := dependencyEndpointState(ctx, uow, relation.TargetRef)
		if err != nil {
			return nil, err
		}
		result = append(result, domain.DependencyEvaluation{
			Relation:  relation,
			Satisfied: state.completed,
		})
	}
	return result, nil
}

func requireHardDependenciesSatisfied(
	ctx context.Context,
	uow ports.UnitOfWork,
	source domain.EntityRef,
) error {
	relations, err := uow.Relations().ListByOutcome(ctx, source.Scope)
	if err != nil {
		return err
	}
	evaluations, err := dependencyEvaluationsForSource(ctx, uow, source, relations)
	if err != nil {
		return err
	}
	for _, evaluation := range evaluations {
		if evaluation.Relation.Strength == domain.DependencyStrengthHard && !evaluation.Satisfied {
			target := evaluation.Relation.TargetRef
			return domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "hard dependency is not satisfied", domain.ErrorDetails{ReasonCode: "hard_dependency_unsatisfied", SubjectRef: &source, RelatedRefs: []domain.EntityRef{target}, Remedies: []domain.ErrorRemedy{{ActionID: "inspect", SubjectRef: &target}}})
		}
	}
	return nil
}

func evaluateObjectiveReadiness(
	ctx context.Context,
	uow ports.UnitOfWork,
	objective domain.Objective,
	now time.Time,
) (domain.Readiness, error) {
	return evaluateObjectiveExecutionGate(ctx, uow, objective, now)
}

func evaluateWorkItemReadiness(
	ctx context.Context,
	uow ports.UnitOfWork,
	item domain.WorkItem,
	now time.Time,
) (domain.Readiness, error) {
	return evaluateTaskAcquisition(ctx, uow, item, now)
}

func requireObjectiveReady(ctx context.Context, uow ports.UnitOfWork, objective domain.Objective, now time.Time) error {
	readiness, err := evaluateObjectiveReadiness(ctx, uow, objective, now)
	if err != nil {
		return err
	}
	if readiness.Ready {
		return nil
	}
	return operationalReadinessError(readiness, "objective is not ready")
}

func requireWorkItemReady(ctx context.Context, uow ports.UnitOfWork, item domain.WorkItem, now time.Time) error {
	readiness, err := evaluateWorkItemReadiness(ctx, uow, item, now)
	if err != nil {
		return err
	}
	if readiness.Ready {
		return nil
	}
	return operationalReadinessError(readiness, "work item is not ready")
}
