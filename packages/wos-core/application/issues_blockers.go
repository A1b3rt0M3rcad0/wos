package application

import (
	"context"
	"sort"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

const maxIssueDuplicateDepth = 10000

type CreateIssueCommand struct {
	Scope        domain.Scope
	Title        string
	Description  string
	Severity     domain.IssueSeverity
	AffectedRefs []domain.EntityRef
}
type InvestigateIssueCommand struct {
	Scope           domain.Scope
	IssueID         domain.ID
	ExpectedVersion domain.Version
}
type ResolveIssueCommand struct {
	Scope             domain.Scope
	IssueID           domain.ID
	ExpectedVersion   domain.Version
	ResolutionSummary string
}
type MarkIssueWontFixCommand struct {
	Scope             domain.Scope
	IssueID           domain.ID
	ExpectedVersion   domain.Version
	ResolutionSummary string
}
type MarkIssueDuplicateCommand struct {
	Scope           domain.Scope
	IssueID         domain.ID
	ExpectedVersion domain.Version
	DuplicateOfID   domain.ID
}
type ReopenIssueCommand struct {
	Scope           domain.Scope
	IssueID         domain.ID
	ExpectedVersion domain.Version
}
type CreateBlockerCommand struct {
	Scope         domain.Scope
	BlockedRef    domain.EntityRef
	CauseRef      *domain.EntityRef
	ExternalCause *domain.ExternalCause
	Description   string
	Propagation   domain.BlockerPropagation
}
type ResolveBlockerCommand struct {
	Scope             domain.Scope
	BlockerID         domain.ID
	ExpectedVersion   domain.Version
	ResolutionSummary string
}
type CancelBlockerCommand struct {
	Scope           domain.Scope
	BlockerID       domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

type AppliedBlocker struct {
	Blocker       domain.Blocker    `json:"blocker"`
	Inherited     bool              `json:"inherited"`
	InheritedFrom *domain.EntityRef `json:"inherited_from,omitempty"`
}
type BlockingState struct {
	Ref            domain.EntityRef `json:"ref"`
	IsBlocked      bool             `json:"is_blocked"`
	ActiveBlockers []AppliedBlocker `json:"active_blockers"`
}

func (s *Service) CreateIssue(ctx context.Context, cc domain.CommandContext, cmd CreateIssueCommand) (MutationResult[domain.Issue], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Issue]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Issue]{}, err
	}
	issue, err := domain.NewIssue(id, cmd.Scope, cmd.Title, cmd.Description, cmd.Severity, cmd.AffectedRefs, cc.Actor, s.clock.Now().UTC())
	if err != nil {
		return MutationResult[domain.Issue]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Issue, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Issue{}, 0, err
		}
		issues, blockers, err := issueBlockerRepositories(uow)
		if err != nil {
			return domain.Issue{}, 0, err
		}
		if err := validateIssueAffectedRefs(ctx, uow, issues, blockers, issue.AffectedRefs); err != nil {
			return domain.Issue{}, 0, err
		}
		if err := issues.Insert(ctx, issue); err != nil {
			return domain.Issue{}, 0, err
		}
		rev, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return issue, rev, err
	})
}
func (s *Service) InvestigateIssue(ctx context.Context, cc domain.CommandContext, cmd InvestigateIssueCommand) (MutationResult[domain.Issue], error) {
	return s.mutateIssue(ctx, cc, cmd, cmd.Scope, cmd.IssueID, cmd.ExpectedVersion, func(v *domain.Issue) error { return v.Investigate(s.clock.Now().UTC()) })
}
func (s *Service) ResolveIssue(ctx context.Context, cc domain.CommandContext, cmd ResolveIssueCommand) (MutationResult[domain.Issue], error) {
	return s.mutateIssue(ctx, cc, cmd, cmd.Scope, cmd.IssueID, cmd.ExpectedVersion, func(v *domain.Issue) error { return v.Resolve(cmd.ResolutionSummary, s.clock.Now().UTC()) })
}
func (s *Service) MarkIssueWontFix(ctx context.Context, cc domain.CommandContext, cmd MarkIssueWontFixCommand) (MutationResult[domain.Issue], error) {
	return s.mutateIssue(ctx, cc, cmd, cmd.Scope, cmd.IssueID, cmd.ExpectedVersion, func(v *domain.Issue) error { return v.MarkWontFix(cmd.ResolutionSummary, s.clock.Now().UTC()) })
}
func (s *Service) ReopenIssue(ctx context.Context, cc domain.CommandContext, cmd ReopenIssueCommand) (MutationResult[domain.Issue], error) {
	return s.mutateIssue(ctx, cc, cmd, cmd.Scope, cmd.IssueID, cmd.ExpectedVersion, func(v *domain.Issue) error { return v.Reopen(s.clock.Now().UTC()) })
}

func (s *Service) mutateIssue(ctx context.Context, cc domain.CommandContext, command any, scope domain.Scope, id domain.ID, expected domain.Version, mutate func(*domain.Issue) error) (MutationResult[domain.Issue], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Issue]{}, err
	}
	return transactCommand(ctx, s, cc, command, func(uow ports.UnitOfWork) (domain.Issue, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, scope); err != nil {
			return domain.Issue{}, 0, err
		}
		issues, _, err := issueBlockerRepositories(uow)
		if err != nil {
			return domain.Issue{}, 0, err
		}
		v, err := issues.Get(ctx, scope, id)
		if err != nil {
			return domain.Issue{}, 0, err
		}
		if err := mutate(&v); err != nil {
			return domain.Issue{}, 0, err
		}
		if err := issues.Save(ctx, v, expected); err != nil {
			return domain.Issue{}, 0, err
		}
		rev, err := uow.Coordination().AdvanceOutcome(ctx, scope)
		return v, rev, err
	})
}

func (s *Service) MarkIssueDuplicate(ctx context.Context, cc domain.CommandContext, cmd MarkIssueDuplicateCommand) (MutationResult[domain.Issue], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Issue]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Issue, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Issue{}, 0, err
		}
		issues, _, err := issueBlockerRepositories(uow)
		if err != nil {
			return domain.Issue{}, 0, err
		}
		v, err := issues.Get(ctx, cmd.Scope, cmd.IssueID)
		if err != nil {
			return domain.Issue{}, 0, err
		}
		if err := validateIssueDuplicateChain(ctx, issues, cmd.Scope, v.ID, cmd.DuplicateOfID); err != nil {
			return domain.Issue{}, 0, err
		}
		if err := v.MarkDuplicate(cmd.DuplicateOfID, s.clock.Now().UTC()); err != nil {
			return domain.Issue{}, 0, err
		}
		if err := issues.Save(ctx, v, cmd.ExpectedVersion); err != nil {
			return domain.Issue{}, 0, err
		}
		rev, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return v, rev, err
	})
}

func (s *Service) CreateBlocker(ctx context.Context, cc domain.CommandContext, cmd CreateBlockerCommand) (MutationResult[domain.Blocker], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Blocker]{}, err
	}
	if cmd.BlockedRef.Scope != cmd.Scope {
		return MutationResult[domain.Blocker]{}, domain.NewError(domain.ErrorCodeBlocker, "blocked_ref must match command scope")
	}
	if cmd.CauseRef != nil && cmd.CauseRef.Scope != cmd.Scope {
		return MutationResult[domain.Blocker]{}, domain.NewError(domain.ErrorCodeBlocker, "cause_ref must match command scope")
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Blocker]{}, err
	}
	v, err := domain.NewBlocker(id, cmd.BlockedRef, cmd.CauseRef, cmd.ExternalCause, cmd.Description, cmd.Propagation, s.clock.Now().UTC())
	if err != nil {
		return MutationResult[domain.Blocker]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Blocker, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Blocker{}, 0, err
		}
		issues, blockers, err := issueBlockerRepositories(uow)
		if err != nil {
			return domain.Blocker{}, 0, err
		}
		if err := validateBlockerTarget(ctx, uow, cmd.BlockedRef); err != nil {
			return domain.Blocker{}, 0, err
		}
		if err := validateBlockerCause(ctx, uow, issues, cmd.CauseRef); err != nil {
			return domain.Blocker{}, 0, err
		}
		if err := blockers.Insert(ctx, v); err != nil {
			return domain.Blocker{}, 0, err
		}
		rev, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return v, rev, err
	})
}
func (s *Service) ResolveBlocker(ctx context.Context, cc domain.CommandContext, cmd ResolveBlockerCommand) (MutationResult[domain.Blocker], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Blocker]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Blocker, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Blocker{}, 0, err
		}
		_, b, err := issueBlockerRepositories(uow)
		if err != nil {
			return domain.Blocker{}, 0, err
		}
		v, err := b.Get(ctx, cmd.Scope, cmd.BlockerID)
		if err != nil {
			return domain.Blocker{}, 0, err
		}
		if err := v.Resolve(cmd.ResolutionSummary, s.clock.Now().UTC()); err != nil {
			return domain.Blocker{}, 0, err
		}
		if err := b.Save(ctx, v, cmd.ExpectedVersion); err != nil {
			return domain.Blocker{}, 0, err
		}
		rev, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return v, rev, err
	})
}
func (s *Service) CancelBlocker(ctx context.Context, cc domain.CommandContext, cmd CancelBlockerCommand) (MutationResult[domain.Blocker], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Blocker]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Blocker, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Blocker{}, 0, err
		}
		_, b, err := issueBlockerRepositories(uow)
		if err != nil {
			return domain.Blocker{}, 0, err
		}
		v, err := b.Get(ctx, cmd.Scope, cmd.BlockerID)
		if err != nil {
			return domain.Blocker{}, 0, err
		}
		if err := v.Cancel(cmd.Reason, s.clock.Now().UTC()); err != nil {
			return domain.Blocker{}, 0, err
		}
		if err := b.Save(ctx, v, cmd.ExpectedVersion); err != nil {
			return domain.Blocker{}, 0, err
		}
		rev, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return v, rev, err
	})
}

func (s *Service) GetIssue(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.Issue], error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Issue]{}, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return ReadResult[domain.Issue]{}, err
	}
	issues, _, err := issueBlockerRepositories(uow)
	if err != nil {
		return ReadResult[domain.Issue]{}, err
	}
	v, err := issues.Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.Issue]{}, err
	}
	c, err := uow.Coordination().LockOutcome(ctx, scope)
	return ReadResult[domain.Issue]{Value: v, OutcomeRevision: c.Revision}, err
}
func (s *Service) ListIssues(ctx context.Context, scope domain.Scope) ([]domain.Issue, domain.OutcomeRevision, error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return nil, 0, err
	}
	issues, _, err := issueBlockerRepositories(uow)
	if err != nil {
		return nil, 0, err
	}
	v, err := issues.ListByOutcome(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	c, err := uow.Coordination().LockOutcome(ctx, scope)
	return v, c.Revision, err
}
func (s *Service) GetBlocker(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.Blocker], error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Blocker]{}, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return ReadResult[domain.Blocker]{}, err
	}
	_, b, err := issueBlockerRepositories(uow)
	if err != nil {
		return ReadResult[domain.Blocker]{}, err
	}
	v, err := b.Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.Blocker]{}, err
	}
	c, err := uow.Coordination().LockOutcome(ctx, scope)
	return ReadResult[domain.Blocker]{Value: v, OutcomeRevision: c.Revision}, err
}
func (s *Service) ListBlockers(ctx context.Context, scope domain.Scope) ([]domain.Blocker, domain.OutcomeRevision, error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return nil, 0, err
	}
	_, b, err := issueBlockerRepositories(uow)
	if err != nil {
		return nil, 0, err
	}
	v, err := b.ListByOutcome(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	c, err := uow.Coordination().LockOutcome(ctx, scope)
	return v, c.Revision, err
}
func (s *Service) GetBlockingState(ctx context.Context, ref domain.EntityRef) (BlockingState, domain.OutcomeRevision, error) {
	if err := ref.Validate(); err != nil {
		return BlockingState{}, 0, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return BlockingState{}, 0, err
	}
	defer uow.Rollback()
	v, err := blockingStateForRef(ctx, uow, ref)
	if err != nil {
		return BlockingState{}, 0, err
	}
	c, err := uow.Coordination().LockOutcome(ctx, ref.Scope)
	return v, c.Revision, err
}

func issueBlockerRepositories(uow ports.UnitOfWork) (ports.IssueRepository, ports.BlockerRepository, error) {
	x, ok := uow.(ports.IssueBlockerUnitOfWork)
	if !ok {
		return nil, nil, domain.NewError(domain.ErrorCodeInvalidConfig, "transaction adapter does not support issues and blockers")
	}
	return x.Issues(), x.Blockers(), nil
}
func blockerRepositoryIfAvailable(uow ports.UnitOfWork) (ports.BlockerRepository, bool) {
	x, ok := uow.(ports.IssueBlockerUnitOfWork)
	if !ok {
		return nil, false
	}
	return x.Blockers(), true
}
func lockExistingOutcome(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope) error {
	if _, err := uow.Coordination().LockOutcome(ctx, scope); err != nil {
		return err
	}
	_, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	return err
}
func validateIssueDuplicateChain(ctx context.Context, issues ports.IssueRepository, scope domain.Scope, source, target domain.ID) error {
	if source == target {
		return domain.NewError(domain.ErrorCodeIssue, "issue duplicate chain cannot reference itself")
	}
	current := target
	for depth := 0; depth < maxIssueDuplicateDepth; depth++ {
		v, err := issues.Get(ctx, scope, current)
		if err != nil {
			return err
		}
		if v.ID == source {
			return domain.NewError(domain.ErrorCodeIssue, "issue duplicate chain would create a cycle")
		}
		if v.DuplicateOfIssueID == nil {
			return nil
		}
		current = *v.DuplicateOfIssueID
	}
	return domain.NewError(domain.ErrorCodeGraphLimitExceeded, "issue duplicate chain validation limit exceeded")
}
func validateBlockerTarget(ctx context.Context, uow ports.UnitOfWork, ref domain.EntityRef) error {
	switch ref.Kind {
	case domain.EntityKindOutcome:
		if ref.ID != ref.Scope.OutcomeID {
			return domain.NewError(domain.ErrorCodeBlocker, "outcome blocked_ref id must equal outcome scope id")
		}
		v, err := uow.Outcomes().Get(ctx, ref.Scope.NamespaceID, ref.Scope.OutcomeID)
		if err != nil {
			return err
		}
		if v.IsArchived() || v.Lifecycle == domain.OutcomeLifecycleAchieved || v.Lifecycle == domain.OutcomeLifecycleFailed || v.Lifecycle == domain.OutcomeLifecycleAbandoned {
			return domain.NewError(domain.ErrorCodeInvalidTransition, "terminal or archived outcome cannot receive an active blocker")
		}
	case domain.EntityKindObjective:
		v, err := uow.Objectives().Get(ctx, ref.Scope, ref.ID)
		if err != nil {
			return err
		}
		if v.Lifecycle == domain.ObjectiveLifecycleAchieved || v.Lifecycle == domain.ObjectiveLifecycleCancelled {
			return domain.NewError(domain.ErrorCodeInvalidTransition, "terminal objective cannot receive an active blocker")
		}
	case domain.EntityKindWorkItem:
		v, err := uow.WorkItems().Get(ctx, ref.Scope, ref.ID)
		if err != nil {
			return err
		}
		if v.Lifecycle == domain.WorkItemLifecycleDone || v.Lifecycle == domain.WorkItemLifecycleCancelled {
			return domain.NewError(domain.ErrorCodeInvalidTransition, "terminal work item cannot receive an active blocker")
		}
	default:
		return domain.NewError(domain.ErrorCodeBlocker, "unsupported blocker target kind")
	}
	return nil
}
func validateBlockerCause(ctx context.Context, uow ports.UnitOfWork, issues ports.IssueRepository, ref *domain.EntityRef) error {
	if ref == nil {
		return nil
	}
	switch ref.Kind {
	case domain.EntityKindIssue:
		_, err := issues.Get(ctx, ref.Scope, ref.ID)
		return err
	case domain.EntityKindObjective:
		_, err := uow.Objectives().Get(ctx, ref.Scope, ref.ID)
		return err
	case domain.EntityKindWorkItem:
		_, err := uow.WorkItems().Get(ctx, ref.Scope, ref.ID)
		return err
	case domain.EntityKindDecision:
		return domain.NewError(domain.ErrorCodeBlocker, "decision blocker causes are unavailable until Decision records are implemented")
	default:
		return domain.NewError(domain.ErrorCodeBlocker, "unsupported blocker cause kind")
	}
}

func validateIssueAffectedRefs(
	ctx context.Context,
	uow ports.UnitOfWork,
	issues ports.IssueRepository,
	blockers ports.BlockerRepository,
	refs []domain.EntityRef,
) error {
	for _, ref := range refs {
		var err error
		switch ref.Kind {
		case domain.EntityKindOutcome:
			if ref.ID != ref.Scope.OutcomeID {
				err = domain.NewError(domain.ErrorCodeIssue, "outcome affected_ref id must equal outcome scope id")
			} else {
				_, err = uow.Outcomes().Get(ctx, ref.Scope.NamespaceID, ref.Scope.OutcomeID)
			}
		case domain.EntityKindObjective:
			_, err = uow.Objectives().Get(ctx, ref.Scope, ref.ID)
		case domain.EntityKindWorkItem:
			_, err = uow.WorkItems().Get(ctx, ref.Scope, ref.ID)
		case domain.EntityKindIssue:
			_, err = issues.Get(ctx, ref.Scope, ref.ID)
		case domain.EntityKindBlocker:
			_, err = blockers.Get(ctx, ref.Scope, ref.ID)
		case domain.EntityKindRelation:
			_, err = uow.Relations().Get(ctx, ref.Scope, ref.ID)
		default:
			err = domain.NewError(domain.ErrorCodeIssue, "affected_ref kind is unavailable in the current implementation")
		}
		if err != nil {
			return domain.WrapError(domain.ErrorCodeIssue, "affected_ref does not resolve to a persisted entity", err)
		}
	}
	return nil
}

func blockingStateForRef(ctx context.Context, uow ports.UnitOfWork, ref domain.EntityRef) (BlockingState, error) {
	state := BlockingState{Ref: ref, ActiveBlockers: []AppliedBlocker{}}
	if !blockerTargetKindForApplication(ref.Kind) {
		return BlockingState{}, domain.NewError(domain.ErrorCodeBlocker, "blocking projection supports outcome, objective and work_item")
	}
	var objectiveID *domain.ID
	switch ref.Kind {
	case domain.EntityKindOutcome:
		if ref.ID != ref.Scope.OutcomeID {
			return BlockingState{}, domain.NewError(domain.ErrorCodeBlocker, "outcome ref id must equal outcome scope id")
		}
		if _, err := uow.Outcomes().Get(ctx, ref.Scope.NamespaceID, ref.Scope.OutcomeID); err != nil {
			return BlockingState{}, err
		}
	case domain.EntityKindObjective:
		if _, err := uow.Objectives().Get(ctx, ref.Scope, ref.ID); err != nil {
			return BlockingState{}, err
		}
		id := ref.ID
		objectiveID = &id
	case domain.EntityKindWorkItem:
		item, err := uow.WorkItems().Get(ctx, ref.Scope, ref.ID)
		if err != nil {
			return BlockingState{}, err
		}
		if item.ObjectiveID != nil {
			id := *item.ObjectiveID
			objectiveID = &id
		}
	}
	blockers, ok := blockerRepositoryIfAvailable(uow)
	if !ok {
		return state, nil
	}
	values, err := blockers.ListByOutcome(ctx, ref.Scope)
	if err != nil {
		return BlockingState{}, err
	}
	objectives, err := uow.Objectives().ListByOutcome(ctx, ref.Scope)
	if err != nil {
		return BlockingState{}, err
	}
	byID := make(map[domain.ID]domain.Objective, len(objectives))
	for _, o := range objectives {
		byID[o.ID] = o
	}
	for _, b := range values {
		if b.Lifecycle != domain.BlockerLifecycleActive {
			continue
		}
		if b.BlockedRef == ref {
			state.ActiveBlockers = append(state.ActiveBlockers, AppliedBlocker{Blocker: b})
			continue
		}
		if b.Propagation != domain.BlockerPropagationSubtree {
			continue
		}
		switch b.BlockedRef.Kind {
		case domain.EntityKindOutcome:
			origin := b.BlockedRef
			state.ActiveBlockers = append(state.ActiveBlockers, AppliedBlocker{Blocker: b, Inherited: true, InheritedFrom: &origin})
		case domain.EntityKindObjective:
			if objectiveID != nil && objectiveDescendsFrom(*objectiveID, b.BlockedRef.ID, byID) {
				origin := b.BlockedRef
				state.ActiveBlockers = append(state.ActiveBlockers, AppliedBlocker{Blocker: b, Inherited: true, InheritedFrom: &origin})
			}
		}
	}
	sort.Slice(state.ActiveBlockers, func(i, j int) bool {
		return state.ActiveBlockers[i].Blocker.ID.String() < state.ActiveBlockers[j].Blocker.ID.String()
	})
	state.IsBlocked = len(state.ActiveBlockers) > 0
	return state, nil
}
func objectiveDescendsFrom(id, ancestor domain.ID, values map[domain.ID]domain.Objective) bool {
	current := id
	for depth := 0; depth < len(values)+1; depth++ {
		if current == ancestor {
			return true
		}
		v, ok := values[current]
		if !ok || v.ParentObjectiveID == nil {
			return false
		}
		current = *v.ParentObjectiveID
	}
	return false
}
func blockerTargetKindForApplication(kind domain.EntityKind) bool {
	return kind == domain.EntityKindOutcome || kind == domain.EntityKindObjective || kind == domain.EntityKindWorkItem
}
func requireNotBlocked(ctx context.Context, uow ports.UnitOfWork, ref domain.EntityRef) error {
	state, err := blockingStateForRef(ctx, uow, ref)
	if err != nil {
		return err
	}
	if !state.IsBlocked {
		return nil
	}
	return domain.NewError(domain.ErrorCodePreconditionFailed, "entity is blocked by active blocker "+state.ActiveBlockers[0].Blocker.ID.String())
}
