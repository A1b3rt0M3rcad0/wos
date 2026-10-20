package application

import (
	"context"
	"encoding/json"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

const maxResolveIssueBlockers = 64
const maxReportIssueBlockers = 64

type blockerImpactKey struct {
	target      domain.EntityRef
	propagation domain.BlockerPropagation
}

// IssueBlockerInput describes one explicitly reviewed operational impact for
// an Issue. A target and propagation pair may appear only once per request.
type IssueBlockerInput struct {
	BlockedRef  domain.EntityRef          `json:"blocked_ref"`
	Description string                    `json:"description,omitempty"`
	Propagation domain.BlockerPropagation `json:"propagation,omitempty"`
}

// ReportIssueWithBlockersCommand creates one Objective-owned Issue and all of
// its explicitly confirmed Blockers as one version-pinned operation.
type ReportIssueWithBlockersCommand struct {
	Scope                   domain.Scope
	ObjectiveID             domain.ID
	Title                   string
	Description             string
	Severity                domain.IssueSeverity
	AffectedRefs            []domain.EntityRef
	Blockers                []IssueBlockerInput
	ObservedOutcomeRevision *domain.OutcomeRevision `wos:"optional" json:"observed_outcome_revision,omitempty,string"`
}

type ReportIssueWithBlockersResult struct {
	Issue    domain.Issue     `json:"issue"`
	Blockers []domain.Blocker `json:"blockers"`
}

// ReportIssueWithBlockerCommand records a problem and the concrete impediment
// it creates as one atomic domain command. The resulting Blocker always points
// to the newly created Issue as its cause.
type ReportIssueWithBlockerCommand struct {
	Scope                   domain.Scope
	ObjectiveID             domain.ID
	Title                   string
	Description             string
	Severity                domain.IssueSeverity
	AffectedRefs            []domain.EntityRef
	BlockedRef              domain.EntityRef
	BlockerDescription      string
	Propagation             domain.BlockerPropagation
	ObservedOutcomeRevision *domain.OutcomeRevision `wos:"optional" json:"observed_outcome_revision,omitempty,string"`
}

type ReportIssueWithBlockerResult struct {
	Issue   domain.Issue   `json:"issue"`
	Blocker domain.Blocker `json:"blocker"`
}

func (s *Service) ReportIssueWithBlockers(
	ctx context.Context,
	cc domain.CommandContext,
	cmd ReportIssueWithBlockersCommand,
) (MutationResult[ReportIssueWithBlockersResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[ReportIssueWithBlockersResult]{}, err
	}
	if err := cmd.Scope.Validate(); err != nil {
		return MutationResult[ReportIssueWithBlockersResult]{}, err
	}
	if cmd.ObservedOutcomeRevision == nil {
		return MutationResult[ReportIssueWithBlockersResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "multi-target Issue reporting requires observed_outcome_revision")
	}
	if len(cmd.Blockers) == 0 || len(cmd.Blockers) > maxReportIssueBlockers {
		return MutationResult[ReportIssueWithBlockersResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "ReportIssueWithBlockers requires between 1 and 64 explicit blockers")
	}
	payload, err := json.Marshal(cmd)
	if err != nil {
		return MutationResult[ReportIssueWithBlockersResult]{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "ReportIssueWithBlockers payload cannot be encoded", err)
	}
	if len(payload) > domain.MaxDomainEventPayloadBytes {
		return MutationResult[ReportIssueWithBlockersResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "ReportIssueWithBlockers payload exceeds the 256 KiB command envelope")
	}
	for _, item := range cmd.Blockers {
		if item.BlockedRef.Scope != cmd.Scope {
			return MutationResult[ReportIssueWithBlockersResult]{}, domain.NewError(domain.ErrorCodeBlocker, "every blocked_ref must match command scope")
		}
	}

	issueID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[ReportIssueWithBlockersResult]{}, err
	}
	blockerIDs := make([]domain.ID, len(cmd.Blockers))
	for i := range blockerIDs {
		blockerIDs[i], err = s.ids.NewID()
		if err != nil {
			return MutationResult[ReportIssueWithBlockersResult]{}, err
		}
	}
	now := s.clock.Now().UTC()
	issue, err := domain.NewIssueForObjective(
		issueID,
		cmd.Scope,
		&cmd.ObjectiveID,
		cmd.Title,
		cmd.Description,
		cmd.Severity,
		cmd.AffectedRefs,
		cc.Actor,
		now,
	)
	if err != nil {
		return MutationResult[ReportIssueWithBlockersResult]{}, err
	}
	issueRef := issue.Ref()
	createdBlockers := make([]domain.Blocker, 0, len(cmd.Blockers))
	uniqueImpacts := make(map[blockerImpactKey]struct{}, len(cmd.Blockers))
	for index, item := range cmd.Blockers {
		blocker, err := domain.NewBlocker(blockerIDs[index], item.BlockedRef, &issueRef, nil, item.Description, item.Propagation, now)
		if err != nil {
			return MutationResult[ReportIssueWithBlockersResult]{}, err
		}
		impactKey := blockerImpactKey{target: blocker.BlockedRef, propagation: blocker.Propagation}
		if _, exists := uniqueImpacts[impactKey]; exists {
			return MutationResult[ReportIssueWithBlockersResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "ReportIssueWithBlockers cannot repeat a target and propagation")
		}
		uniqueImpacts[impactKey] = struct{}{}
		createdBlockers = append(createdBlockers, blocker)
	}

	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (ReportIssueWithBlockersResult, domain.OutcomeRevision, error) {
		coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
		if err != nil {
			return ReportIssueWithBlockersResult{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return ReportIssueWithBlockersResult{}, 0, err
		}
		if err := validateOpenOutcome(outcome); err != nil {
			return ReportIssueWithBlockersResult{}, 0, err
		}
		if coordination.Revision != *cmd.ObservedOutcomeRevision {
			subject := cmd.Blockers[0].BlockedRef
			related := make([]domain.EntityRef, 0, min(5, len(cmd.Blockers)))
			for _, item := range cmd.Blockers[:min(5, len(cmd.Blockers))] {
				related = append(related, item.BlockedRef)
			}
			return ReportIssueWithBlockersResult{}, 0, domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "the Outcome changed after the multi-target blocking impact review; review affected work before confirming again", domain.ErrorDetails{
				ReasonCode: "blocker_impact_changed", SubjectRef: &subject, RelatedRefs: related, FieldPaths: []string{"observed_outcome_revision"},
				Remedies: []domain.ErrorRemedy{{ActionID: "review_blocker_impact", SubjectRef: &subject}},
			})
		}
		if err := validateIssueObjective(ctx, uow, cmd.Scope, cmd.ObjectiveID); err != nil {
			return ReportIssueWithBlockersResult{}, 0, err
		}
		issues, blockers, err := issueBlockerRepositories(uow)
		if err != nil {
			return ReportIssueWithBlockersResult{}, 0, err
		}
		if err := validateIssueAffectedRefs(ctx, uow, issues, blockers, issue.AffectedRefs); err != nil {
			return ReportIssueWithBlockersResult{}, 0, err
		}
		// Validate every target before the first write. The transaction remains
		// atomic if an adapter rejects any later insert or event append.
		for _, blocker := range createdBlockers {
			if err := validateBlockerTarget(ctx, uow, blocker.BlockedRef); err != nil {
				return ReportIssueWithBlockersResult{}, 0, err
			}
		}
		if err := issues.Insert(ctx, issue); err != nil {
			return ReportIssueWithBlockersResult{}, 0, err
		}
		for _, blocker := range createdBlockers {
			if err := blockers.Insert(ctx, blocker); err != nil {
				return ReportIssueWithBlockersResult{}, 0, err
			}
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		if err != nil {
			return ReportIssueWithBlockersResult{}, 0, err
		}
		return ReportIssueWithBlockersResult{Issue: issue, Blockers: createdBlockers}, revision, nil
	})
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
	issue, err := domain.NewIssueForObjective(
		issueID,
		cmd.Scope,
		&cmd.ObjectiveID,
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
		if err := validateIssueObjective(ctx, uow, cmd.Scope, cmd.ObjectiveID); err != nil {
			return ReportIssueWithBlockerResult{}, 0, err
		}
		issues, blockers, err := issueBlockerRepositories(uow)
		if err != nil {
			return ReportIssueWithBlockerResult{}, 0, err
		}
		if err := validateBlockerTarget(ctx, uow, cmd.BlockedRef); err != nil {
			return ReportIssueWithBlockerResult{}, 0, err
		}
		if err := validateIssueAffectedRefs(ctx, uow, issues, blockers, issue.AffectedRefs); err != nil {
			return ReportIssueWithBlockerResult{}, 0, err
		}
		if cmd.ObservedOutcomeRevision != nil {
			current, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
			if err != nil {
				return ReportIssueWithBlockerResult{}, 0, err
			}
			if current.Revision != *cmd.ObservedOutcomeRevision {
				return ReportIssueWithBlockerResult{}, 0, domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "the Outcome changed after the blocking impact review; review current affected work before confirming again", domain.ErrorDetails{
					ReasonCode: "blocker_impact_changed", SubjectRef: &cmd.BlockedRef, FieldPaths: []string{"observed_outcome_revision"},
					Remedies: []domain.ErrorRemedy{{ActionID: "review_blocker_impact", SubjectRef: &cmd.BlockedRef}},
				})
			}
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
	case ReportIssueWithBlockersResult:
		specs := make([]compoundEventSpec, 0, 1+len(result.Blockers))
		specs = append(specs, compoundEventSpec{eventType: "issue.created", ref: result.Issue.Ref(), after: versionPtr(result.Issue.Version)})
		for _, blocker := range result.Blockers {
			specs = append(specs, compoundEventSpec{eventType: "blocker.created", ref: blocker.Ref(), after: versionPtr(blocker.Version)})
		}
		events, err := buildCompoundEvents(s, cc, meta, revision, specs)
		return events, true, err
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
