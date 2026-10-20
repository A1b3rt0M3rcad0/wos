package application

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

const (
	MaxPlanItems             = 50
	MaxInitialCriteria       = 100
	maxWorkspaceCommandBytes = 256 << 10
)

// InitialCriterion is a definition, never an assessment or an implicit gate.
// Callers explicitly choose its requirement and verification mode.
type InitialCriterion struct {
	Title            string                  `json:"title"`
	Description      string                  `json:"description,omitempty"`
	Required         bool                    `json:"required"`
	VerificationMode domain.VerificationMode `json:"verification_mode"`
}

// PlanItemInput only creates new operational entities. Both parent fields
// address an Objective; for a Task that is its Objective association.
type PlanItemInput struct {
	LocalKey           string             `json:"local_key"`
	Kind               domain.EntityKind  `json:"kind"`
	Title              string             `json:"title"`
	Description        string             `json:"description,omitempty"`
	Priority           domain.Priority    `wos:"optional" json:"priority,omitempty"`
	RequiredForOutcome bool               `wos:"optional" json:"required_for_outcome"`
	ParentObjectiveID  *domain.ID         `wos:"optional" json:"parent_objective_id,omitempty"`
	ParentLocalKey     string             `wos:"optional" json:"parent_local_key,omitempty"`
	InitialCriteria    []InitialCriterion `wos:"optional" json:"initial_criteria,omitempty"`
}

type CreatePlanItemsCommand struct {
	Scope domain.Scope    `json:"scope"`
	Items []PlanItemInput `wos:"required" json:"items"`
}

type CreatedPlanItem struct {
	LocalKey string                    `json:"local_key"`
	Ref      domain.EntityRef          `json:"ref"`
	Version  domain.Version            `json:"version,string"`
	Criteria []domain.SuccessCriterion `json:"criteria,omitempty"`
}

type CreatePlanItemsResult struct {
	Items []CreatedPlanItem           `json:"items"`
	Refs  map[string]domain.EntityRef `json:"refs"`
}

var planLocalKey = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

func boundedWorkspaceCommand(value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return domain.WrapError(domain.ErrorCodeInvalidArgument, "workspace command cannot be encoded", err)
	}
	if len(raw) > maxWorkspaceCommandBytes {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "workspace command exceeds 256 KiB")
	}
	return nil
}

func requireWorkspaceIntent(cc domain.CommandContext) error {
	if cc.IdempotencyKey == "" {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "workspace creation requires a durable idempotency intent")
	}
	return nil
}

func validateInitialCriteria(criteria []InitialCriterion) error {
	if len(criteria) > MaxInitialCriteria {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "initial criteria exceed 100 definitions")
	}
	for i, criterion := range criteria {
		if !utf8.ValidString(criterion.Title) || strings.ContainsRune(criterion.Title, 0) || utf8.RuneCountInString(strings.TrimSpace(criterion.Title)) < 1 || utf8.RuneCountInString(criterion.Title) > 240 || !utf8.ValidString(criterion.Description) || strings.ContainsRune(criterion.Description, 0) || len(criterion.Description) > 64<<10 || !criterion.VerificationMode.Valid() {
			return domain.NewError(domain.ErrorCodeInvalidArgument, fmt.Sprintf("initial_criteria[%d] requires a title up to 240 characters, description up to 64 KiB and valid verification_mode", i))
		}
	}
	return nil
}

func (s *Service) authorizeCreationPermissions(ctx context.Context, uow ports.UnitOfWork, cc domain.CommandContext, scope domain.Scope, permissions ...ports.Permission) error {
	if !s.requireIdentity {
		return nil
	}
	seen := make(map[ports.Permission]bool, len(permissions))
	for _, permission := range permissions {
		if seen[permission] {
			continue
		}
		seen[permission] = true
		request := ports.AuthorizationRequest{NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID, PrincipalID: cc.PrincipalID, Permission: permission}
		if auth, ok := s.authorizer.(ports.TransactionalAuthorizer); ok {
			if err := auth.AuthorizeInUnitOfWork(ctx, uow, request); err != nil {
				return err
			}
		} else if err := s.authorizer.Authorize(ctx, request); err != nil {
			return err
		}
	}
	return nil
}

// The pipeline also calls this before receipt replay. Retaining only the
// primary permission must not disclose a compound result after a constituent
// planning/work permission has been revoked.
func (s *Service) authorizePlanningCommandPermissions(ctx context.Context, uow ports.UnitOfWork, cc domain.CommandContext, command any) error {
	var scope domain.Scope
	var permissions []ports.Permission
	switch cmd := command.(type) {
	case CreatePlanItemsCommand:
		scope = cmd.Scope
		for _, item := range cmd.Items {
			if item.Kind == domain.EntityKindObjective || len(item.InitialCriteria) > 0 {
				permissions = append(permissions, ports.PermissionOutcomeWrite)
			}
			if item.Kind == domain.EntityKindWorkItem {
				permissions = append(permissions, ports.PermissionWorkWrite)
			}
		}
	case CreateWorkItemCommand:
		scope = cmd.Scope
		if len(cmd.InitialCriteria) > 0 {
			permissions = append(permissions, ports.PermissionOutcomeWrite)
		}
	default:
		return nil
	}
	return s.authorizeCreationPermissions(ctx, uow, cc, scope, permissions...)
}

func (s *Service) initialCriteria(ctx context.Context, uow ports.UnitOfWork, cc domain.CommandContext, owner domain.EntityRef, definitions []InitialCriterion, now time.Time, add func(domain.SuccessCriterion, time.Time) error) error {
	if len(definitions) == 0 {
		return nil
	}
	if err := validateInitialCriteria(definitions); err != nil {
		return err
	}
	if err := s.authorizeCreationPermissions(ctx, uow, cc, owner.Scope, ports.PermissionOutcomeWrite); err != nil {
		return err
	}
	for _, definition := range definitions {
		id, err := s.ids.NewID()
		if err != nil {
			return err
		}
		criterion, err := domain.NewSuccessCriterion(id, owner, definition.Title, definition.Description, definition.Required, definition.VerificationMode)
		if err != nil {
			return err
		}
		if err := add(criterion, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) insertObjectiveCreation(ctx context.Context, uow ports.UnitOfWork, cc domain.CommandContext, cmd CreateObjectiveCommand, value *domain.Objective, now time.Time) error {
	if err := requireCreationOutcome(ctx, uow, cmd.Scope, "objective"); err != nil {
		return err
	}
	if cmd.ParentObjectiveID != nil {
		if err := requireOpenCreationParent(ctx, uow, cmd.Scope, *cmd.ParentObjectiveID, "objective"); err != nil {
			return err
		}
	}
	if err := s.initialCriteria(ctx, uow, cc, value.Ref(), cmd.InitialCriteria, now, value.AddCriterion); err != nil {
		return err
	}
	return uow.Objectives().Insert(ctx, *value)
}

func (s *Service) insertWorkItemCreation(ctx context.Context, uow ports.UnitOfWork, cc domain.CommandContext, cmd CreateWorkItemCommand, value *domain.WorkItem, now time.Time) error {
	if err := requireCreationOutcome(ctx, uow, cmd.Scope, "work"); err != nil {
		return err
	}
	if cmd.ObjectiveID != nil {
		if err := requireOpenCreationParent(ctx, uow, cmd.Scope, *cmd.ObjectiveID, "work"); err != nil {
			return err
		}
	}
	if repo, ok := uow.(ports.WorkProtocolUnitOfWork); ok {
		protocol, err := repo.WorkProtocol().Lock(ctx, cmd.Scope.NamespaceID, false)
		if err != nil {
			return err
		}
		value.ContractsEnabled = protocol.Phase == domain.WorkProtocolContracts || protocol.Phase == domain.WorkProtocolSigned || protocol.Phase == domain.WorkProtocolSignedDraining
	}
	if err := validateExecutionSpecRefs(ctx, uow, cmd.Scope, cmd.ExecutionSpec); err != nil {
		return err
	}
	if cmd.ExecutionSpec != nil {
		spec := *cmd.ExecutionSpec
		value.ExecutionSpec = &spec
	}
	initialLifecycle := value.Lifecycle
	value.Lifecycle = domain.WorkItemLifecycleBacklog
	if err := s.initialCriteria(ctx, uow, cc, value.Ref(), cmd.InitialCriteria, now, value.AddCriterion); err != nil {
		return err
	}
	value.Lifecycle = initialLifecycle
	return uow.WorkItems().Insert(ctx, *value)
}

func requireOpenCreationParent(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope, id domain.ID, kind string) error {
	objective, err := uow.Objectives().Get(ctx, scope, id)
	if err != nil {
		return err
	}
	if objective.Lifecycle == domain.ObjectiveLifecycleAchieved || objective.Lifecycle == domain.ObjectiveLifecycleCancelled {
		return domain.NewError(domain.ErrorCodeInvalidTransition, "cannot attach new "+kind+" to terminal objective")
	}
	return nil
}

func requireCreationOutcome(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope, kind string) error {
	if _, err := uow.Coordination().LockOutcome(ctx, scope); err != nil {
		return err
	}
	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return err
	}
	if outcome.IsArchived() || outcome.Lifecycle == domain.OutcomeLifecycleAchieved || outcome.Lifecycle == domain.OutcomeLifecycleFailed || outcome.Lifecycle == domain.OutcomeLifecycleAbandoned {
		return domain.NewError(domain.ErrorCodeInvalidTransition, "cannot create "+kind+" in archived or terminal outcome")
	}
	return nil
}

func validatePlanItems(cmd CreatePlanItemsCommand) ([]int, error) {
	if err := cmd.Scope.Validate(); err != nil {
		return nil, err
	}
	if len(cmd.Items) == 0 || len(cmd.Items) > MaxPlanItems {
		return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "CreatePlanItems requires between 1 and 50 new items")
	}
	indices := make(map[string]int, len(cmd.Items))
	criteriaCount := 0
	for i, item := range cmd.Items {
		if !planLocalKey.MatchString(item.LocalKey) {
			return nil, domain.NewError(domain.ErrorCodeInvalidArgument, fmt.Sprintf("items[%d].local_key must be 1..64 ASCII letters, digits, dots, underscores or hyphens", i))
		}
		if _, exists := indices[item.LocalKey]; exists {
			return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "duplicate local_key: "+item.LocalKey)
		}
		indices[item.LocalKey] = i
		if item.Kind != domain.EntityKindObjective && item.Kind != domain.EntityKindWorkItem {
			return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "plan item kind must be objective or work_item")
		}
		if strings.TrimSpace(item.Title) == "" || !utf8.ValidString(item.Title) || strings.ContainsRune(item.Title, 0) || utf8.RuneCountInString(item.Title) > 240 || !utf8.ValidString(item.Description) || strings.ContainsRune(item.Description, 0) || len(item.Description) > 64<<10 {
			return nil, domain.NewError(domain.ErrorCodeInvalidArgument, fmt.Sprintf("items[%d] has invalid or oversized title/description", i))
		}
		if item.Priority != "" && !item.Priority.Valid() {
			return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid plan item priority")
		}
		if item.Kind == domain.EntityKindWorkItem && item.RequiredForOutcome {
			return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "required_for_outcome applies only to Objectives")
		}
		if item.Kind == domain.EntityKindWorkItem && item.ParentObjectiveID == nil && item.ParentLocalKey == "" {
			return nil, domain.NewDetailedError(domain.ErrorCodeInvalidArgument, "every Task in a plan must belong to an Objective", domain.ErrorDetails{
				ReasonCode: "task_objective_required", FieldPaths: []string{fmt.Sprintf("items[%d].parent_objective_id", i)},
			})
		}
		if item.ParentObjectiveID != nil {
			if err := item.ParentObjectiveID.Validate(); err != nil {
				return nil, err
			}
			if item.ParentLocalKey != "" {
				return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "parent_objective_id and parent_local_key are mutually exclusive")
			}
		}
		if err := validateInitialCriteria(item.InitialCriteria); err != nil {
			return nil, err
		}
		criteriaCount += len(item.InitialCriteria)
	}
	if criteriaCount > MaxInitialCriteria {
		return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "plan batch exceeds 100 initial criteria")
	}
	if err := boundedWorkspaceCommand(cmd); err != nil {
		return nil, err
	}
	state := make([]uint8, len(cmd.Items))
	ordered := make([]int, 0, len(cmd.Items))
	var visit func(int) error
	visit = func(i int) error {
		if state[i] == 1 {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "plan local parent cycle at "+cmd.Items[i].LocalKey)
		}
		if state[i] == 2 {
			return nil
		}
		state[i] = 1
		if key := cmd.Items[i].ParentLocalKey; key != "" {
			parent, exists := indices[key]
			if !exists {
				return domain.NewError(domain.ErrorCodeInvalidArgument, "unknown parent_local_key: "+key)
			}
			if cmd.Items[parent].Kind != domain.EntityKindObjective {
				return domain.NewError(domain.ErrorCodeInvalidArgument, "local parent must be an Objective")
			}
			if err := visit(parent); err != nil {
				return err
			}
		}
		state[i] = 2
		ordered = append(ordered, i)
		return nil
	}
	for i := range cmd.Items {
		if err := visit(i); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

func (s *Service) CreatePlanItems(ctx context.Context, cc domain.CommandContext, cmd CreatePlanItemsCommand) (MutationResult[CreatePlanItemsResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[CreatePlanItemsResult]{}, err
	}
	if err := requireWorkspaceIntent(cc); err != nil {
		return MutationResult[CreatePlanItemsResult]{}, err
	}
	ordered, err := validatePlanItems(cmd)
	if err != nil {
		return MutationResult[CreatePlanItemsResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (CreatePlanItemsResult, domain.OutcomeRevision, error) {
		var zero CreatePlanItemsResult
		if err := requireCreationOutcome(ctx, uow, cmd.Scope, "plan items"); err != nil {
			return zero, 0, err
		}
		permissions := []ports.Permission{}
		for _, item := range cmd.Items {
			if item.Kind == domain.EntityKindObjective || len(item.InitialCriteria) > 0 {
				permissions = append(permissions, ports.PermissionOutcomeWrite)
			}
			if item.Kind == domain.EntityKindWorkItem {
				permissions = append(permissions, ports.PermissionWorkWrite)
			}
		}
		if err := s.authorizeCreationPermissions(ctx, uow, cc, cmd.Scope, permissions...); err != nil {
			return zero, 0, err
		}
		result := CreatePlanItemsResult{Items: make([]CreatedPlanItem, len(cmd.Items)), Refs: make(map[string]domain.EntityRef, len(cmd.Items))}
		now := s.clock.Now().UTC()
		for _, i := range ordered {
			input := cmd.Items[i]
			parent := cloneIDPtr(input.ParentObjectiveID)
			if input.ParentLocalKey != "" {
				ref := result.Refs[input.ParentLocalKey]
				parent = &ref.ID
			}
			priority := input.Priority
			if priority == "" {
				priority = domain.PriorityNormal
			}
			id, err := s.ids.NewID()
			if err != nil {
				return zero, 0, err
			}
			var created CreatedPlanItem
			if input.Kind == domain.EntityKindObjective {
				value, err := domain.NewObjective(id, cmd.Scope, input.Title, input.Description, priority, input.RequiredForOutcome, now)
				if err != nil {
					return zero, 0, err
				}
				value.ParentObjectiveID = parent
				create := CreateObjectiveCommand{Scope: cmd.Scope, ParentObjectiveID: parent, InitialCriteria: input.InitialCriteria}
				if err := s.insertObjectiveCreation(ctx, uow, cc, create, &value, now); err != nil {
					return zero, 0, err
				}
				created = CreatedPlanItem{LocalKey: input.LocalKey, Ref: value.Ref(), Version: value.Version, Criteria: value.Criteria.Items}
			} else {
				value, err := domain.NewWorkItemForObjective(id, cmd.Scope, parent, input.Title, input.Description, priority, domain.WorkItemLifecycleBacklog, now)
				if err != nil {
					return zero, 0, err
				}
				create := CreateWorkItemCommand{Scope: cmd.Scope, ObjectiveID: parent, InitialCriteria: input.InitialCriteria}
				if err := s.insertWorkItemCreation(ctx, uow, cc, create, &value, now); err != nil {
					return zero, 0, err
				}
				created = CreatedPlanItem{LocalKey: input.LocalKey, Ref: value.Ref(), Version: value.Version, Criteria: value.Criteria.Items}
			}
			result.Items[i] = created
			result.Refs[input.LocalKey] = created.Ref
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return result, revision, err
	})
}

func planningCreationEvents[T any](s *Service, cc domain.CommandContext, meta commandMetadata, value T, revision domain.OutcomeRevision) ([]domain.DomainEvent, bool, error) {
	items := []CreatedPlanItem{}
	switch result := any(value).(type) {
	case CreatePlanItemsResult:
		items = result.Items
	case domain.Outcome:
		if meta.Name != "CreateOutcome" || len(result.Criteria.Items) == 0 {
			return nil, false, nil
		}
		items = append(items, CreatedPlanItem{Ref: result.Ref(), Version: result.Version, Criteria: result.Criteria.Items})
	case domain.Objective:
		if meta.Name != "CreateObjective" || len(result.Criteria.Items) == 0 {
			return nil, false, nil
		}
		items = append(items, CreatedPlanItem{Ref: result.Ref(), Version: result.Version, Criteria: result.Criteria.Items})
	case domain.WorkItem:
		if meta.Name != "CreateWorkItem" || len(result.Criteria.Items) == 0 {
			return nil, false, nil
		}
		items = append(items, CreatedPlanItem{Ref: result.Ref(), Version: result.Version, Criteria: result.Criteria.Items})
	default:
		return nil, false, nil
	}
	specs := []compoundEventSpec{}
	payloads := []any{}
	for _, item := range items {
		specs = append(specs, compoundEventSpec{eventType: item.Ref.Kind.String() + ".created", ref: item.Ref, after: versionPtr(domain.InitialVersion)})
		payloads = append(payloads, map[string]any{"ref": item.Ref, "local_key": item.LocalKey})
		for i, criterion := range item.Criteria {
			before := domain.InitialVersion + domain.Version(i)
			specs = append(specs, compoundEventSpec{eventType: item.Ref.Kind.String() + ".criterion_added", ref: item.Ref, before: versionPtr(before), after: versionPtr(before + 1)})
			payloads = append(payloads, map[string]any{"criterion_id": criterion.ID, "criterion": criterion})
		}
	}
	events, err := buildCompoundEvents(s, cc, meta, revision, specs)
	if err != nil {
		return nil, true, err
	}
	for i := range events {
		events[i].Payload, err = json.Marshal(payloads[i])
		if err != nil {
			return nil, true, err
		}
		if err := events[i].Validate(); err != nil {
			return nil, true, err
		}
	}
	return events, true, nil
}
