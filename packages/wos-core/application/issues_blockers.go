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
	ObjectiveID  domain.ID
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
	issue, err := domain.NewIssueForObjective(id, cmd.Scope, &cmd.ObjectiveID, cmd.Title, cmd.Description, cmd.Severity, cmd.AffectedRefs, cc.Actor, s.clock.Now().UTC())
	if err != nil {
		return MutationResult[domain.Issue]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Issue, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Issue{}, 0, err
		}
		if err := validateIssueObjective(ctx, uow, cmd.Scope, cmd.ObjectiveID); err != nil {
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

func validateIssueObjective(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope, id domain.ID) error {
	if err := id.Validate(); err != nil {
		return domain.WrapError(domain.ErrorCodeInvalidArgument, "issue Objective ID is required and must be valid", err)
	}
	objective, err := uow.Objectives().Get(ctx, scope, id)
	if err != nil {
		return err
	}
	if objective.Scope != scope || objective.ID != id {
		return domain.NewError(domain.ErrorCodeInvalidScope, "issue Objective must belong to the command Outcome")
	}
	if objective.Lifecycle == domain.ObjectiveLifecycleAchieved || objective.Lifecycle == domain.ObjectiveLifecycleCancelled {
		return definitionPrecondition(domain.ErrorCodePreconditionFailed, "objective_terminal", "choose a nonterminal Objective to own this Issue", objective.Ref(), []string{"objective_id"})
	}
	return nil
}
func (s *Service) InvestigateIssue(ctx context.Context, cc domain.CommandContext, cmd InvestigateIssueCommand) (MutationResult[domain.Issue], error) {
	result, err := s.mutateIssueAssignment(ctx, cc, cmd, cmd.Scope, cmd.IssueID, cmd.ExpectedVersion, "", func(v *domain.Issue) error {
		return v.StartInvestigationWithAssignee(cc.Actor, s.clock.Now().UTC())
	})
	return publicIssueAssignmentResult(result), err
}
func (s *Service) ResolveIssue(ctx context.Context, cc domain.CommandContext, cmd ResolveIssueCommand) (MutationResult[domain.Issue], error) {
	return s.mutateIssue(ctx, cc, cmd, cmd.Scope, cmd.IssueID, cmd.ExpectedVersion, func(v *domain.Issue) error { return v.Resolve(cmd.ResolutionSummary, s.clock.Now().UTC()) })
}
func (s *Service) MarkIssueWontFix(ctx context.Context, cc domain.CommandContext, cmd MarkIssueWontFixCommand) (MutationResult[domain.Issue], error) {
	return s.mutateIssue(ctx, cc, cmd, cmd.Scope, cmd.IssueID, cmd.ExpectedVersion, func(v *domain.Issue) error { return v.MarkWontFix(cmd.ResolutionSummary, s.clock.Now().UTC()) })
}
func (s *Service) ReopenIssue(ctx context.Context, cc domain.CommandContext, cmd ReopenIssueCommand) (MutationResult[domain.Issue], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Issue]{}, err
	}
	if err := cmd.Scope.Validate(); err != nil {
		return MutationResult[domain.Issue]{}, err
	}
	if err := cmd.IssueID.Validate(); err != nil {
		return MutationResult[domain.Issue]{}, err
	}
	if err := cmd.ExpectedVersion.Validate(); err != nil {
		return MutationResult[domain.Issue]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Issue, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Issue{}, 0, err
		}
		issues, _, err := issueBlockerRepositories(uow)
		if err != nil {
			return domain.Issue{}, 0, err
		}
		issue, err := issues.Get(ctx, cmd.Scope, cmd.IssueID)
		if err != nil {
			return domain.Issue{}, 0, err
		}
		if issue.Version != cmd.ExpectedVersion {
			return domain.Issue{}, 0, domain.NewError(domain.ErrorCodeVersionConflict, "issue expected_version is stale")
		}
		reopened := issue
		if err := reopened.Reopen(now); err != nil {
			return domain.Issue{}, 0, err
		}
		if issue.ObjectiveID != nil {
			objective, err := uow.Objectives().Get(ctx, cmd.Scope, *issue.ObjectiveID)
			if err != nil {
				return domain.Issue{}, 0, err
			}
			if objective.Scope != cmd.Scope || objective.ID != *issue.ObjectiveID {
				return domain.Issue{}, 0, domain.NewError(domain.ErrorCodeInvalidScope, "issue Objective must belong to the command Outcome")
			}
			if objective.Lifecycle == domain.ObjectiveLifecycleAchieved {
				issueRef := issue.Ref()
				objectiveRef := objective.Ref()
				return domain.Issue{}, 0, domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "reopen the Objective before reopening its Issue", domain.ErrorDetails{
					ReasonCode:  "issue_objective_achieved",
					SubjectRef:  &issueRef,
					FieldPaths:  []string{"objective_id"},
					RelatedRefs: []domain.EntityRef{objectiveRef},
					Remedies:    []domain.ErrorRemedy{{ActionID: "reopen_objective", SubjectRef: &objectiveRef}},
				})
			}
		}
		if err := issues.Save(ctx, reopened, cmd.ExpectedVersion); err != nil {
			return domain.Issue{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return reopened, revision, err
	})
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
	if err := s.authorizeScopedRead(ctx, scope); err != nil {
		return ReadResult[domain.Issue]{}, err
	}

	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Issue]{}, err
	}
	defer uow.Rollback()
	if err := s.authorizeScopedReadInUnitOfWork(ctx, uow, scope); err != nil {
		return ReadResult[domain.Issue]{}, err
	}
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
	if err := s.authorizeScopedRead(ctx, scope); err != nil {
		return nil, 0, err
	}

	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()
	if err := s.authorizeScopedReadInUnitOfWork(ctx, uow, scope); err != nil {
		return nil, 0, err
	}
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
	if err := s.authorizeScopedRead(ctx, scope); err != nil {
		return ReadResult[domain.Blocker]{}, err
	}

	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Blocker]{}, err
	}
	defer uow.Rollback()
	if err := s.authorizeScopedReadInUnitOfWork(ctx, uow, scope); err != nil {
		return ReadResult[domain.Blocker]{}, err
	}
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
	if err := s.authorizeScopedRead(ctx, scope); err != nil {
		return nil, 0, err
	}

	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()
	if err := s.authorizeScopedReadInUnitOfWork(ctx, uow, scope); err != nil {
		return nil, 0, err
	}
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
	if err := s.authorizeScopedRead(ctx, ref.Scope); err != nil {
		return BlockingState{}, 0, err
	}

	if err := ref.Validate(); err != nil {
		return BlockingState{}, 0, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return BlockingState{}, 0, err
	}
	defer uow.Rollback()
	if err := s.authorizeScopedReadInUnitOfWork(ctx, uow, ref.Scope); err != nil {
		return BlockingState{}, 0, err
	}
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
	_, err := lockExistingOutcomeRevision(ctx, uow, scope)
	return err
}

func lockExistingOutcomeRevision(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope) (domain.OutcomeRevision, error) {
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return 0, err
	}
	if _, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID); err != nil {
		return 0, err
	}
	return coordination.Revision, nil
}
func validateIssueDuplicateChain(ctx context.Context, issues ports.IssueRepository, scope domain.Scope, source, target domain.ID) error {
	if source == target {
		return domain.NewError(domain.ErrorCodeIssue, "issue duplicate chain cannot reference itself")
	}
	_, err := canonicalIssueDuplicateTarget(ctx, issues, scope, target, &source)
	return err
}

// issueDuplicateChainClosed reports whether an Issue is closed for the
// Objective gate. A duplicate is closed only when its finite, valid chain ends
// at an Issue resolved or marked wont_fix; the latter is a triage decision, not
// evidence that any Task or Objective criteria were met.
func issueDuplicateChainClosed(ctx context.Context, issues ports.IssueRepository, scope domain.Scope, issue domain.Issue) (bool, domain.Issue, error) {
	switch issue.Lifecycle {
	case domain.IssueLifecycleResolved, domain.IssueLifecycleWontFix:
		return true, issue, nil
	case domain.IssueLifecycleOpen, domain.IssueLifecycleInvestigating:
		return false, issue, nil
	case domain.IssueLifecycleDuplicate:
		if issue.DuplicateOfIssueID == nil {
			return false, domain.Issue{}, domain.NewError(domain.ErrorCodeIssue, "duplicate issue has no canonical target")
		}
		canonical, err := canonicalIssueDuplicateTarget(ctx, issues, scope, *issue.DuplicateOfIssueID, nil)
		if err != nil {
			return false, domain.Issue{}, err
		}
		return canonical.Lifecycle == domain.IssueLifecycleResolved || canonical.Lifecycle == domain.IssueLifecycleWontFix, canonical, nil
	default:
		return false, domain.Issue{}, domain.NewError(domain.ErrorCodeIssue, "issue lifecycle is invalid for the completion gate")
	}
}

// canonicalIssueDuplicateTarget follows duplicate links to their canonical
// Issue. A forbidden ID is supplied while assigning a duplicate edge so a
// chain that reaches its source cannot introduce a cycle. Every read stays
// within the command scope, and missing or corrupt links fail closed.
func canonicalIssueDuplicateTarget(ctx context.Context, issues ports.IssueRepository, scope domain.Scope, start domain.ID, forbidden *domain.ID) (domain.Issue, error) {
	if err := start.Validate(); err != nil {
		return domain.Issue{}, domain.WrapError(domain.ErrorCodeIssue, "duplicate target id is invalid", err)
	}
	seen := make(map[domain.ID]struct{})
	current := start
	for depth := 0; depth < maxIssueDuplicateDepth; depth++ {
		if forbidden != nil && current == *forbidden {
			return domain.Issue{}, domain.NewError(domain.ErrorCodeIssue, "issue duplicate chain would create a cycle")
		}
		if _, exists := seen[current]; exists {
			return domain.Issue{}, domain.NewError(domain.ErrorCodeIssue, "issue duplicate chain contains a cycle")
		}
		seen[current] = struct{}{}

		issue, err := issues.Get(ctx, scope, current)
		if err != nil {
			return domain.Issue{}, err
		}
		if issue.ID != current || issue.Scope != scope {
			return domain.Issue{}, domain.NewError(domain.ErrorCodeInvalidScope, "issue duplicate target does not match the requested scope and id")
		}
		if err := issue.Validate(); err != nil {
			return domain.Issue{}, domain.WrapError(domain.ErrorCodeIssue, "issue duplicate chain contains an invalid Issue", err)
		}
		if issue.DuplicateOfIssueID == nil {
			return issue, nil
		}
		if issue.Lifecycle != domain.IssueLifecycleDuplicate {
			return domain.Issue{}, domain.NewError(domain.ErrorCodeIssue, "issue duplicate link is inconsistent with its lifecycle")
		}
		current = *issue.DuplicateOfIssueID
	}
	return domain.Issue{}, domain.NewError(domain.ErrorCodeGraphLimitExceeded, "issue duplicate chain validation limit exceeded")
}
func validateBlockerTarget(ctx context.Context, uow ports.UnitOfWork, ref domain.EntityRef) error {
	switch ref.Kind {
	case domain.EntityKindOutcome:
		if ref.ID != ref.Scope.OutcomeID {
			return domain.NewError(domain.ErrorCodeBlocker, "outcome blocked_ref id must equal outcome scope id")
		}
		value, err := uow.Outcomes().Get(ctx, ref.Scope.NamespaceID, ref.Scope.OutcomeID)
		if err != nil {
			return err
		}
		return validateBlockerTargetState(ref, string(value.Lifecycle), value.IsArchived())
	case domain.EntityKindObjective:
		value, err := uow.Objectives().Get(ctx, ref.Scope, ref.ID)
		if err != nil {
			return err
		}
		return validateBlockerTargetState(ref, string(value.Lifecycle), false)
	case domain.EntityKindWorkItem:
		value, err := uow.WorkItems().Get(ctx, ref.Scope, ref.ID)
		if err != nil {
			return err
		}
		return validateBlockerTargetState(ref, string(value.Lifecycle), false)
	default:
		return validateBlockerTargetState(ref, "", false)
	}
}

// The target's canonical lifecycle is enough for a coherent preview; reading a
// preview never hydrates aggregate history or grants permission to block work.
func validateBlockerTargetState(ref domain.EntityRef, lifecycle string, archived bool) error {
	switch ref.Kind {
	case domain.EntityKindOutcome:
		if ref.ID != ref.Scope.OutcomeID {
			return domain.NewError(domain.ErrorCodeBlocker, "outcome blocked_ref id must equal outcome scope id")
		}
		if archived || lifecycle == string(domain.OutcomeLifecycleAchieved) || lifecycle == string(domain.OutcomeLifecycleFailed) || lifecycle == string(domain.OutcomeLifecycleAbandoned) {
			return domain.NewError(domain.ErrorCodeInvalidTransition, "terminal or archived outcome cannot receive an active blocker")
		}
	case domain.EntityKindObjective:
		if lifecycle == string(domain.ObjectiveLifecycleAchieved) || lifecycle == string(domain.ObjectiveLifecycleCancelled) {
			return domain.NewError(domain.ErrorCodeInvalidTransition, "terminal objective cannot receive an active blocker")
		}
	case domain.EntityKindWorkItem:
		if lifecycle == string(domain.WorkItemLifecycleDone) || lifecycle == string(domain.WorkItemLifecycleCancelled) {
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
		documentary, ok := uow.(ports.DocumentaryUnitOfWork)
		if !ok {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "transaction adapter does not support decisions")
		}
		_, err := documentary.Decisions().Get(ctx, ref.Scope, ref.ID)
		return err
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
	return projectBlockingState(ref, objectiveID, values, byID), nil
}

// Projection over an already coherent read avoids reloading every aggregate
// and the complete blocker collection for every snapshot entity.
func projectBlockingState(ref domain.EntityRef, objectiveID *domain.ID, values []domain.Blocker, byID map[domain.ID]domain.Objective) BlockingState {
	state := BlockingState{Ref: ref, ActiveBlockers: []AppliedBlocker{}}
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
	return state
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
	blockerRef := state.ActiveBlockers[0].Blocker.Ref()
	return domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "entity is blocked by active blocker "+state.ActiveBlockers[0].Blocker.ID.String(), domain.ErrorDetails{ReasonCode: "active_blocker", SubjectRef: &ref, RelatedRefs: []domain.EntityRef{blockerRef}, Remedies: []domain.ErrorRemedy{{ActionID: "inspect", SubjectRef: &blockerRef}}})
}
