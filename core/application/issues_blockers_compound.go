package application

import (
	"context"
	"encoding/json"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
)

const maxResolveIssueBlockers = 64

// ReportIssueWithBlockerCommand records a problem and the concrete impediment
// it creates as one atomic domain command. The resulting Blocker always points
// to the newly created Issue as its cause.
type ReportIssueWithBlockerCommand struct {
	Scope              domain.Scope
	Title              string
	Description        string
	Severity           domain.IssueSeverity
	AffectedRefs       []domain.EntityRef
	BlockedRef         domain.EntityRef
	BlockerDescription string
	Propagation        domain.BlockerPropagation
}

type ReportIssueWithBlockerResult struct {
	Issue   domain.Issue   `json:"issue"`
	Blocker domain.Blocker `json:"blocker"`
}

// BlockerResolution makes release confirmation explicit. A compound issue
// resolution never infers which Blockers should be released.
type BlockerResolution struct {
	BlockerID         domain.ID      `json:"blocker_id"`
	ExpectedVersion   domain.Version `json:"expected_version"`
	ResolutionSummary string         `json:"resolution_summary"`
	ReleaseConfirmed  bool           `json:"release_confirmed"`
}

type ResolveIssueAndBlockersCommand struct {
	Scope                  domain.Scope
	IssueID                domain.ID
	IssueExpectedVersion   domain.Version
	IssueResolutionSummary string
	Blockers               []BlockerResolution
}

type ResolveIssueAndBlockersResult struct {
	Issue    domain.Issue     `json:"issue"`
	Blockers []domain.Blocker `json:"blockers"`
}

func (s *Service) ReportIssueWithBlocker(
	ctx context.Context,
	cc domain.CommandContext,
	cmd ReportIssueWithBlockerCommand,
) (MutationResult[ReportIssueWithBlockerResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[ReportIssueWithBlockerResult]{}, err
	}
	if cmd.BlockedRef.Scope != cmd.Scope {
		return MutationResult[ReportIssueWithBlockerResult]{}, domain.NewError(domain.ErrorCodeBlocker, "blocked_ref must match command scope")
	}

	issueID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[ReportIssueWithBlockerResult]{}, err
	}
	blockerID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[ReportIssueWithBlockerResult]{}, err
	}
	now := s.clock.Now().UTC()
	issue, err := domain.NewIssue(
		issueID,
		cmd.Scope,
		cmd.Title,
		cmd.Description,
		cmd.Severity,
		cmd.AffectedRefs,
		cc.Actor,
		now,
	)
	if err != nil {
		return MutationResult[ReportIssueWithBlockerResult]{}, err
	}
	cause := issue.Ref()
	blocker, err := domain.NewBlocker(
		blockerID,
		cmd.BlockedRef,
		&cause,
		nil,
		cmd.BlockerDescription,
		cmd.Propagation,
		now,
	)
	if err != nil {
		return MutationResult[ReportIssueWithBlockerResult]{}, err
	}

	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (ReportIssueWithBlockerResult, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return ReportIssueWithBlockerResult{}, 0, err
		}
		issues, blockers, err := issueBlockerRepositories(uow)
		if err != nil {
			return ReportIssueWithBlockerResult{}, 0, err
		}
		if err := validateBlockerTarget(ctx, uow, cmd.BlockedRef); err != nil {
			return ReportIssueWithBlockerResult{}, 0, err
		}
		if err := issues.Insert(ctx, issue); err != nil {
			return ReportIssueWithBlockerResult{}, 0, err
		}
		if err := blockers.Insert(ctx, blocker); err != nil {
			return ReportIssueWithBlockerResult{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		if err != nil {
			return ReportIssueWithBlockerResult{}, 0, err
		}
		return ReportIssueWithBlockerResult{Issue: issue, Blocker: blocker}, revision, nil
	})
}

func (s *Service) ResolveIssueAndBlockers(
	ctx context.Context,
	cc domain.CommandContext,
	cmd ResolveIssueAndBlockersCommand,
) (MutationResult[ResolveIssueAndBlockersResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[ResolveIssueAndBlockersResult]{}, err
	}
	if err := cmd.Scope.Validate(); err != nil {
		return MutationResult[ResolveIssueAndBlockersResult]{}, err
	}
	if err := cmd.IssueID.Validate(); err != nil {
		return MutationResult[ResolveIssueAndBlockersResult]{}, err
	}
	if err := cmd.IssueExpectedVersion.Validate(); err != nil {
		return MutationResult[ResolveIssueAndBlockersResult]{}, err
	}
	if len(cmd.Blockers) == 0 || len(cmd.Blockers) > maxResolveIssueBlockers {
		return MutationResult[ResolveIssueAndBlockersResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "ResolveIssueAndBlockers requires between 1 and 64 explicit blockers")
	}
	seen := make(map[domain.ID]struct{}, len(cmd.Blockers))
	for _, item := range cmd.Blockers {
		if err := item.BlockerID.Validate(); err != nil {
			return MutationResult[ResolveIssueAndBlockersResult]{}, err
		}
		if err := item.ExpectedVersion.Validate(); err != nil {
			return MutationResult[ResolveIssueAndBlockersResult]{}, err
		}
		if !item.ReleaseConfirmed {
			return MutationResult[ResolveIssueAndBlockersResult]{}, domain.NewError(domain.ErrorCodePreconditionFailed, "every blocker in ResolveIssueAndBlockers requires explicit release confirmation")
		}
		if _, exists := seen[item.BlockerID]; exists {
			return MutationResult[ResolveIssueAndBlockersResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "ResolveIssueAndBlockers cannot contain duplicate blocker ids")
		}
		seen[item.BlockerID] = struct{}{}
	}

	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (ResolveIssueAndBlockersResult, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return ResolveIssueAndBlockersResult{}, 0, err
		}
		issues, blockers, err := issueBlockerRepositories(uow)
		if err != nil {
			return ResolveIssueAndBlockersResult{}, 0, err
		}
		issue, err := issues.Get(ctx, cmd.Scope, cmd.IssueID)
		if err != nil {
			return ResolveIssueAndBlockersResult{}, 0, err
		}
		if err := issue.Resolve(cmd.IssueResolutionSummary, now); err != nil {
			return ResolveIssueAndBlockersResult{}, 0, err
		}
		if err := issues.Save(ctx, issue, cmd.IssueExpectedVersion); err != nil {
			return ResolveIssueAndBlockersResult{}, 0, err
		}

		resolved := make([]domain.Blocker, 0, len(cmd.Blockers))
		issueRef := issue.Ref()
		for _, item := range cmd.Blockers {
			blocker, err := blockers.Get(ctx, cmd.Scope, item.BlockerID)
			if err != nil {
				return ResolveIssueAndBlockersResult{}, 0, err
			}
			if blocker.CauseRef == nil || *blocker.CauseRef != issueRef {
				return ResolveIssueAndBlockersResult{}, 0, domain.NewError(domain.ErrorCodeBlocker, "compound issue resolution may only release blockers explicitly caused by that issue")
			}
			if err := blocker.Resolve(item.ResolutionSummary, now); err != nil {
				return ResolveIssueAndBlockersResult{}, 0, err
			}
			if err := blockers.Save(ctx, blocker, item.ExpectedVersion); err != nil {
				return ResolveIssueAndBlockersResult{}, 0, err
			}
			resolved = append(resolved, blocker)
		}

		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		if err != nil {
			return ResolveIssueAndBlockersResult{}, 0, err
		}
		return ResolveIssueAndBlockersResult{Issue: issue, Blockers: resolved}, revision, nil
	})
}

func compoundIssueBlockerEvents[T any](
	s *Service,
	cc domain.CommandContext,
	meta commandMetadata,
	value T,
	revision domain.OutcomeRevision,
) ([]domain.DomainEvent, bool, error) {
	if revision == 0 {
		return nil, false, nil
	}
	switch result := any(value).(type) {
	case ReportIssueWithBlockerResult:
		specs := []compoundEventSpec{
			{eventType: "issue.created", ref: result.Issue.Ref(), after: versionPtr(result.Issue.Version)},
			{eventType: "blocker.created", ref: result.Blocker.Ref(), after: versionPtr(result.Blocker.Version)},
		}
		events, err := buildCompoundEvents(s, cc, meta, revision, specs)
		return events, true, err
	case ResolveIssueAndBlockersResult:
		beforeIssue, err := previousVersionPtr(result.Issue.Version)
		if err != nil {
			return nil, true, err
		}
		specs := make([]compoundEventSpec, 0, 1+len(result.Blockers))
		specs = append(specs, compoundEventSpec{
			eventType: "issue.resolved",
			ref:       result.Issue.Ref(),
			before:    beforeIssue,
			after:     versionPtr(result.Issue.Version),
		})
		for _, blocker := range result.Blockers {
			before, err := previousVersionPtr(blocker.Version)
			if err != nil {
				return nil, true, err
			}
			specs = append(specs, compoundEventSpec{
				eventType: "blocker.resolved",
				ref:       blocker.Ref(),
				before:    before,
				after:     versionPtr(blocker.Version),
			})
		}
		events, err := buildCompoundEvents(s, cc, meta, revision, specs)
		return events, true, err
	default:
		return nil, false, nil
	}
}

type compoundEventSpec struct {
	eventType string
	ref       domain.EntityRef
	before    *domain.Version
	after     *domain.Version
}

func buildCompoundEvents(
	s *Service,
	cc domain.CommandContext,
	meta commandMetadata,
	revision domain.OutcomeRevision,
	specs []compoundEventSpec,
) ([]domain.DomainEvent, error) {
	if revision == 0 {
		return nil, domain.NewError(domain.ErrorCodeInvalidEvent, "confirmed compound mutation must have an outcome_revision")
	}
	now := s.clock.Now().UTC()
	events := make([]domain.DomainEvent, 0, len(specs))
	for index, spec := range specs {
		eventID, err := s.ids.NewID()
		if err != nil {
			return nil, err
		}
		payload := append(json.RawMessage(nil), meta.Payload...)
		event := domain.DomainEvent{
			EventID:                eventID,
			EventType:              spec.eventType,
			SchemaVersion:          domain.DomainEventSchemaVersion,
			NamespaceID:            spec.ref.NamespaceID,
			OutcomeID:              spec.ref.OutcomeID,
			OutcomeRevision:        revision,
			EventIndex:             uint32(index),
			AggregateRef:           spec.ref,
			AggregateVersionBefore: spec.before,
			AggregateVersionAfter:  spec.after,
			PrincipalID:            cc.PrincipalID,
			Actor:                  cc.Actor,
			RecordedAt:             now,
			CommandID:              cc.CommandID,
			CorrelationID:          cc.CorrelationID,
			CausationID:            cloneIDPointer(cc.CausationID),
			ExecutionContext:       cc.Execution,
			Payload:                payload,
		}
		if err := event.Validate(); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func versionPtr(value domain.Version) *domain.Version {
	copy := value
	return &copy
}

func previousVersionPtr(current domain.Version) (*domain.Version, error) {
	if current <= domain.InitialVersion {
		return nil, domain.NewError(domain.ErrorCodeInvalidEvent, "updated compound aggregate must have version greater than one")
	}
	previous := current - 1
	return &previous, nil
}
