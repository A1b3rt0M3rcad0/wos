package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func newOperationalAuthService(t *testing.T) (*application.Service, *memory.Store, *revocablePlanningPolicy, context.Context, domain.CommandContext, *wave09IDs, domain.ID) {
	t.Helper()
	store := memory.New()
	policy := &revocablePlanningPolicy{}
	ids := &wave09IDs{next: 1}
	service, err := application.NewAuthorizedService(store, fixedClock{now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)}, ids, policy)
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	ns := id("0199e200-0000-7000-8000-000000000001")
	ctx := application.WithIdentity(context.Background(), application.Identity{NamespaceID: ns, PrincipalID: cc.PrincipalID, Actor: cc.Actor})
	return service, store, policy, ctx, cc, ids, ns
}

func createOperationalAuthOutcome(t *testing.T, service *application.Service, ctx context.Context, ns domain.ID) domain.Outcome {
	t.Helper()
	created, err := service.CreateOutcome(ctx, planningIntent("e16-operational-outcome"), application.CreateOutcomeCommand{
		NamespaceID: ns, Title: "Authorization scope", DesiredState: "Compound effects stay atomic", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	return created.Value
}

func createOperationalAuthObjective(t *testing.T, service *application.Service, ctx context.Context, scope domain.Scope, key, title string) domain.Objective {
	t.Helper()
	created, err := service.CreateObjective(ctx, planningIntent(key), application.CreateObjectiveCommand{
		Scope: scope, Title: title, Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	return created.Value
}

func TestCreatePlanItemsCriteriaPermissionDenialIsAtomicAndReplayRechecked(t *testing.T) {
	service, store, policy, ctx, _, _, ns := newOperationalAuthService(t)
	outcome := createOperationalAuthOutcome(t, service, ctx, ns)
	before, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	receiptsBefore := store.IdempotencyRecordCount()
	cmdContext := planningIntent("e16-plan-criteria-command")
	cmd := application.CreatePlanItemsCommand{Scope: outcome.Scope(), Items: []application.PlanItemInput{
		{LocalKey: "objective", Kind: domain.EntityKindObjective, Title: "Plan owner", InitialCriteria: []application.InitialCriterion{initialProof("Objective proof")}},
		{LocalKey: "task", Kind: domain.EntityKindWorkItem, Title: "Task with proof", ParentLocalKey: "objective", InitialCriteria: []application.InitialCriterion{initialProof("Task proof")}},
	}}
	policy.denied = ports.PermissionCriterionDefine
	if _, err := service.CreatePlanItems(ctx, cmdContext, cmd); err == nil {
		t.Fatal("plan items with criteria were created without criterion:define")
	}
	afterDenied, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if afterDenied.OutcomeRevision != before.OutcomeRevision || len(afterDenied.Objectives) != len(before.Objectives) || len(afterDenied.WorkItems) != len(before.WorkItems) || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore || store.IdempotencyRecordCount() != receiptsBefore {
		t.Fatal("denied plan creation persisted an entity, criterion, revision, event, or receipt")
	}
	policy.denied = ""
	created, err := service.CreatePlanItems(ctx, cmdContext, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if created.IdempotentReplay || len(created.Value.Items) != 2 || len(created.Value.Items[0].Criteria) != 1 || len(created.Value.Items[1].Criteria) != 1 {
		t.Fatalf("creation did not execute as a fresh atomic command: %#v", created)
	}
	afterCreated, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsAfterCreate := len(store.SnapshotDomainEvents(outcome.Scope()))
	receiptsAfterCreate := store.IdempotencyRecordCount()
	policy.denied = ports.PermissionCriterionDefine
	if _, err := service.CreatePlanItems(ctx, cmdContext, cmd); err == nil {
		t.Fatal("replay disclosed plan result after criterion:define was revoked")
	}
	afterReplayDenied, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if afterReplayDenied.OutcomeRevision != afterCreated.OutcomeRevision || len(afterReplayDenied.Objectives) != len(afterCreated.Objectives) || len(afterReplayDenied.WorkItems) != len(afterCreated.WorkItems) || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsAfterCreate || store.IdempotencyRecordCount() != receiptsAfterCreate {
		t.Fatal("denied replay changed the persisted plan or receipts")
	}
}

func TestIssueBlockerCompoundsRequireBothRightsAndRecheckReplay(t *testing.T) {
	for _, multi := range []bool{false, true} {
		name := "single"
		if multi {
			name = "multiple"
		}
		t.Run(name, func(t *testing.T) {
			service, store, policy, ctx, _, _, ns := newOperationalAuthService(t)
			outcome := createOperationalAuthOutcome(t, service, ctx, ns)
			objective := createOperationalAuthObjective(t, service, ctx, outcome.Scope(), "e16-issue-owner-created", "Issue owner")
			before, err := service.GetOutcomeState(ctx, outcome.Scope())
			if err != nil {
				t.Fatal(err)
			}
			eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
			receiptsBefore := store.IdempotencyRecordCount()
			cc := planningIntent("e16-report-" + name)
			blocked := objective.Ref()
			var invoke func() (bool, error)
			if multi {
				revision := before.OutcomeRevision
				cmd := application.ReportIssueWithBlockersCommand{Scope: outcome.Scope(), ObjectiveID: objective.ID, Title: "Multi-impact issue", Severity: domain.IssueSeverityMajor, Blockers: []application.IssueBlockerInput{{BlockedRef: blocked}}, ObservedOutcomeRevision: &revision}
				invoke = func() (bool, error) {
					result, err := service.ReportIssueWithBlockers(ctx, cc, cmd)
					return result.IdempotentReplay, err
				}
			} else {
				cmd := application.ReportIssueWithBlockerCommand{Scope: outcome.Scope(), ObjectiveID: objective.ID, Title: "Single-impact issue", Severity: domain.IssueSeverityMajor, BlockedRef: blocked, BlockerDescription: "Blocked pending resolution."}
				invoke = func() (bool, error) {
					result, err := service.ReportIssueWithBlocker(ctx, cc, cmd)
					return result.IdempotentReplay, err
				}
			}
			policy.denied = ports.PermissionBlockerCreate
			if _, err := invoke(); err == nil {
				t.Fatal("Issue + Blocker compound succeeded without blocker:create")
			}
			afterDenied, err := service.GetOutcomeState(ctx, outcome.Scope())
			if err != nil {
				t.Fatal(err)
			}
			if afterDenied.OutcomeRevision != before.OutcomeRevision || len(afterDenied.Issues) != len(before.Issues) || len(afterDenied.Blockers) != len(before.Blockers) || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore || store.IdempotencyRecordCount() != receiptsBefore {
				t.Fatal("denied Issue + Blocker compound persisted partial state or receipt")
			}
			policy.denied = ""
			if replay, err := invoke(); err != nil || replay {
				t.Fatalf("denied command left a receipt or did not execute fresh: replay=%v err=%v", replay, err)
			}
			afterCreated, err := service.GetOutcomeState(ctx, outcome.Scope())
			if err != nil {
				t.Fatal(err)
			}
			eventsAfterCreate := len(store.SnapshotDomainEvents(outcome.Scope()))
			receiptsAfterCreate := store.IdempotencyRecordCount()
			if len(afterCreated.Issues) != len(before.Issues)+1 || len(afterCreated.Blockers) != len(before.Blockers)+1 {
				t.Fatal("successful compound did not persist exactly one Issue and Blocker")
			}
			policy.denied = ports.PermissionBlockerCreate
			if _, err := invoke(); err == nil {
				t.Fatal("compound replay disclosed result after blocker:create was revoked")
			}
			afterReplayDenied, err := service.GetOutcomeState(ctx, outcome.Scope())
			if err != nil {
				t.Fatal(err)
			}
			if afterReplayDenied.OutcomeRevision != afterCreated.OutcomeRevision || len(afterReplayDenied.Issues) != len(afterCreated.Issues) || len(afterReplayDenied.Blockers) != len(afterCreated.Blockers) || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsAfterCreate || store.IdempotencyRecordCount() != receiptsAfterCreate {
				t.Fatal("denied compound replay changed issue/blocker state or receipts")
			}
		})
	}
}

func TestIssueBlockerResolutionRequiresBothRightsAndRechecksReplay(t *testing.T) {
	service, store, policy, ctx, _, _, ns := newOperationalAuthService(t)
	outcome := createOperationalAuthOutcome(t, service, ctx, ns)
	objective := createOperationalAuthObjective(t, service, ctx, outcome.Scope(), "e16-issue-owner-resolved", "Issue owner")
	reported, err := service.ReportIssueWithBlocker(ctx, planningIntent("e16-report-before-resolution"), application.ReportIssueWithBlockerCommand{
		Scope: outcome.Scope(), ObjectiveID: objective.ID, Title: "Resolve issue with blocker",
		Severity: domain.IssueSeverityMajor, BlockedRef: objective.Ref(), BlockerDescription: "Work is waiting for the root cause to be fixed.",
	})
	if err != nil {
		t.Fatal(err)
	}
	issue, blocker := reported.Value.Issue, reported.Value.Blocker
	cmdContext := planningIntent("e16-resolve-issue-and-blocker")
	cmd := application.ResolveIssueAndBlockersCommand{
		Scope: outcome.Scope(), IssueID: issue.ID, IssueExpectedVersion: issue.Version,
		IssueResolutionSummary: "Root cause fixed and the blocked work can resume.",
		Blockers: []application.BlockerResolution{{
			BlockerID: blocker.ID, ExpectedVersion: blocker.Version,
			ResolutionSummary: "The blocked work is released.", ReleaseConfirmed: true,
		}},
	}
	before, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	receiptsBefore := store.IdempotencyRecordCount()

	policy.denied = ports.PermissionBlockerRelease
	if _, err := service.ResolveIssueAndBlockers(ctx, cmdContext, cmd); err == nil {
		t.Fatal("Issue + Blocker resolution succeeded without blocker:release")
	}
	afterDenied, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if afterDenied.OutcomeRevision != before.OutcomeRevision || len(afterDenied.Issues) != len(before.Issues) || len(afterDenied.Blockers) != len(before.Blockers) || afterDenied.Issues[0].Lifecycle != domain.IssueLifecycleOpen || afterDenied.Blockers[0].Lifecycle != domain.BlockerLifecycleActive || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore || store.IdempotencyRecordCount() != receiptsBefore {
		t.Fatal("denied Issue + Blocker resolution changed issue, blocker, revision, events, or receipt count")
	}

	policy.denied = ""
	resolved, err := service.ResolveIssueAndBlockers(ctx, cmdContext, cmd)
	if err != nil || resolved.IdempotentReplay {
		t.Fatalf("resolution did not execute fresh after denied attempt: replay=%v err=%v", resolved.IdempotentReplay, err)
	}
	if resolved.Value.Issue.Lifecycle != domain.IssueLifecycleResolved || len(resolved.Value.Blockers) != 1 || resolved.Value.Blockers[0].Lifecycle != domain.BlockerLifecycleResolved {
		t.Fatalf("successful compound resolution = %#v", resolved.Value)
	}
	afterResolved, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsAfterResolve := len(store.SnapshotDomainEvents(outcome.Scope()))
	receiptsAfterResolve := store.IdempotencyRecordCount()
	policy.denied = ports.PermissionBlockerRelease
	if _, err := service.ResolveIssueAndBlockers(ctx, cmdContext, cmd); err == nil {
		t.Fatal("compound replay disclosed result after blocker:release was revoked")
	}
	afterReplayDenied, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if afterReplayDenied.OutcomeRevision != afterResolved.OutcomeRevision || len(afterReplayDenied.Issues) != len(afterResolved.Issues) || len(afterReplayDenied.Blockers) != len(afterResolved.Blockers) || afterReplayDenied.Issues[0].Lifecycle != domain.IssueLifecycleResolved || afterReplayDenied.Blockers[0].Lifecycle != domain.BlockerLifecycleResolved || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsAfterResolve || store.IdempotencyRecordCount() != receiptsAfterResolve {
		t.Fatal("denied compound replay changed issue, blocker, revision, events, or receipts")
	}
}

func TestIssueBlockerCompoundsRejectPersistedCrossOutcomeOwnersAndTargets(t *testing.T) {
	service, store, _, ctx, _, _, ns := newOperationalAuthService(t)
	firstOutcome := createOperationalAuthOutcome(t, service, ctx, ns)
	secondOutcome, err := service.CreateOutcome(ctx, planningIntent("e16-foreign-outcome"), application.CreateOutcomeCommand{
		NamespaceID: ns, Title: "Other authorization scope", DesiredState: "Records stay in their Outcome", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	firstObjective := createOperationalAuthObjective(t, service, ctx, firstOutcome.Scope(), "e16-local-issue-owner", "Local issue owner")
	foreignObjective := createOperationalAuthObjective(t, service, ctx, secondOutcome.Value.Scope(), "e16-foreign-issue-owner", "Foreign issue owner")
	beforeFirst, err := service.GetOutcomeState(ctx, firstOutcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	beforeSecond, err := service.GetOutcomeState(ctx, secondOutcome.Value.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsFirst := len(store.SnapshotDomainEvents(firstOutcome.Scope()))
	eventsSecond := len(store.SnapshotDomainEvents(secondOutcome.Value.Scope()))
	receipts := store.IdempotencyRecordCount()
	observedRevision := beforeFirst.OutcomeRevision

	foreignOwner := application.ReportIssueWithBlockersCommand{
		Scope: firstOutcome.Scope(), ObjectiveID: foreignObjective.ID, Title: "Forged owner", Severity: domain.IssueSeverityMajor,
		ObservedOutcomeRevision: &observedRevision,
		Blockers:                []application.IssueBlockerInput{{BlockedRef: firstObjective.Ref(), Description: "The owner is outside this Outcome."}},
	}
	if _, err := service.ReportIssueWithBlockers(ctx, planningIntent("e16-foreign-owner-compound"), foreignOwner); err == nil {
		t.Fatal("Issue + Blocker compound accepted an Objective persisted in another Outcome")
	}
	foreignTarget := application.ReportIssueWithBlockersCommand{
		Scope: firstOutcome.Scope(), ObjectiveID: firstObjective.ID, Title: "Foreign target", Severity: domain.IssueSeverityMajor,
		ObservedOutcomeRevision: &observedRevision,
		Blockers:                []application.IssueBlockerInput{{BlockedRef: foreignObjective.Ref(), Description: "The target is outside this Outcome."}},
	}
	if _, err := service.ReportIssueWithBlockers(ctx, planningIntent("e16-foreign-target-compound"), foreignTarget); err == nil {
		t.Fatal("Issue + Blocker compound accepted a target persisted in another Outcome")
	}

	afterFirst, err := service.GetOutcomeState(ctx, firstOutcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	afterSecond, err := service.GetOutcomeState(ctx, secondOutcome.Value.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if afterFirst.OutcomeRevision != beforeFirst.OutcomeRevision || len(afterFirst.Issues) != len(beforeFirst.Issues) || len(afterFirst.Blockers) != len(beforeFirst.Blockers) || afterSecond.OutcomeRevision != beforeSecond.OutcomeRevision || len(afterSecond.Issues) != len(beforeSecond.Issues) || len(afterSecond.Blockers) != len(beforeSecond.Blockers) || len(store.SnapshotDomainEvents(firstOutcome.Scope())) != eventsFirst || len(store.SnapshotDomainEvents(secondOutcome.Value.Scope())) != eventsSecond || store.IdempotencyRecordCount() != receipts {
		t.Fatal("cross-Outcome Issue + Blocker denial changed persisted entities, revisions, events, or receipts")
	}
}

func TestIssueBlockerResolutionRejectsLaterBlockerCauseAtomically(t *testing.T) {
	service, store, _, ctx, _, _, ns := newOperationalAuthService(t)
	outcome := createOperationalAuthOutcome(t, service, ctx, ns)
	objective := createOperationalAuthObjective(t, service, ctx, outcome.Scope(), "e16-cause-owner-01", "Issue owner")
	reported, err := service.ReportIssueWithBlocker(ctx, planningIntent("e16-cause-report"), application.ReportIssueWithBlockerCommand{
		Scope: outcome.Scope(), ObjectiveID: objective.ID, Title: "Cause checked after issue update",
		Severity: domain.IssueSeverityMajor, BlockedRef: objective.Ref(), BlockerDescription: "Issue-caused blocker.",
	})
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := service.CreateBlocker(ctx, planningIntent("e16-cause-orphan-blocker"), application.CreateBlockerCommand{
		Scope: outcome.Scope(), BlockedRef: objective.Ref(), Description: "Not caused by the Issue being resolved.",
	})
	if err != nil {
		t.Fatal(err)
	}
	issue, first, second := reported.Value.Issue, reported.Value.Blocker, orphan.Value
	before, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	receiptsBefore := store.IdempotencyRecordCount()
	_, err = service.ResolveIssueAndBlockers(ctx, planningIntent("e16-invalid-later-cause"), application.ResolveIssueAndBlockersCommand{
		Scope: outcome.Scope(), IssueID: issue.ID, IssueExpectedVersion: issue.Version,
		IssueResolutionSummary: "Attempted resolution must roll back as one operation.",
		Blockers: []application.BlockerResolution{
			{BlockerID: first.ID, ExpectedVersion: first.Version, ResolutionSummary: "release first", ReleaseConfirmed: true},
			{BlockerID: second.ID, ExpectedVersion: second.Version, ResolutionSummary: "release unrelated blocker", ReleaseConfirmed: true},
		},
	})
	if err == nil {
		t.Fatal("Issue + Blocker resolution released a later Blocker with a different cause")
	}
	after, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	issueAfter, err := service.GetIssue(ctx, outcome.Scope(), issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	firstAfter, err := service.GetBlocker(ctx, outcome.Scope(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	secondAfter, err := service.GetBlocker(ctx, outcome.Scope(), second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.OutcomeRevision != before.OutcomeRevision || len(after.Issues) != len(before.Issues) || len(after.Blockers) != len(before.Blockers) || issueAfter.Value.Lifecycle != domain.IssueLifecycleOpen || firstAfter.Value.Lifecycle != domain.BlockerLifecycleActive || secondAfter.Value.Lifecycle != domain.BlockerLifecycleActive || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore || store.IdempotencyRecordCount() != receiptsBefore {
		t.Fatal("invalid later Blocker cause partially resolved Issue/Blockers or persisted a revision, event, or receipt")
	}
}

func TestDocumentaryCompoundsRejectPersistedCrossOutcomeReferencesAtomically(t *testing.T) {
	service, store, _, ctx, _, _, ns := newOperationalAuthService(t)
	owner := createOperationalAuthOutcome(t, service, ctx, ns)
	foreign, err := service.CreateOutcome(ctx, planningIntent("e16-document-foreign-outcome"), application.CreateOutcomeCommand{
		NamespaceID: ns, Title: "Foreign documentary scope", DesiredState: "References cannot cross Outcomes", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := service.GetOutcomeState(ctx, owner.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(store.SnapshotDomainEvents(owner.Scope()))
	receiptsBefore := store.IdempotencyRecordCount()

	// The batch proposes an Artifact and Evidence before linking to the foreign
	// Outcome. Validation must roll all of those records and the revision back.
	evidenceAt := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	_, err = service.RegisterEvidenceWithLinks(ctx, planningIntent("e16-document-cross-outcome-link"), application.RegisterEvidenceWithLinksCommand{
		Scope: owner.Scope(),
		Evidence: &application.EvidenceRegistration{
			EvidenceType: domain.EvidenceTypeSource, Description: "Cross-Outcome reference attempt",
			SourceRef: domain.SourceReference{Provider: "external", URI: "https://example.test/report"}, CapturedAt: evidenceAt,
		},
		Artifact: &application.ArtifactRegistration{ArtifactType: "report", Name: "Evidence attachment", URI: "https://example.test/report"},
		Links:    []application.EvidenceLinkInput{{TargetRef: foreign.Value.Ref(), Stance: domain.EvidenceStanceContext, Rationale: "Must remain in the command Outcome."}},
	})
	if err == nil {
		t.Fatal("RegisterEvidenceWithLinks accepted a persisted target from another Outcome")
	}
	_, err = service.RegisterArtifactInContext(ctx, planningIntent("e16-document-cross-outcome-artifact"), application.RegisterArtifactInContextCommand{
		Scope: owner.Scope(), SubjectRef: foreign.Value.Ref(), Relationship: domain.RelationTypeProduces,
		Artifact: application.ArtifactRegistration{ArtifactType: "document", Name: "Foreign subject", URI: "https://example.test/foreign"},
	})
	if err == nil {
		t.Fatal("RegisterArtifactInContext accepted a persisted subject from another Outcome")
	}
	_, err = service.ProposeDecisionInContext(ctx, planningIntent("e16-document-cross-outcome-decision"), application.ProposeDecisionInContextCommand{
		Scope: owner.Scope(), SubjectRef: foreign.Value.Ref(),
		Decision: application.DecisionProposal{Title: "Foreign subject", Proposal: "Keep the source Outcome", Rationale: "The related record must be local."},
	})
	if err == nil {
		t.Fatal("ProposeDecisionInContext accepted a persisted subject from another Outcome")
	}

	after, err := service.GetOutcomeState(ctx, owner.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if after.OutcomeRevision != before.OutcomeRevision || len(after.Artifacts) != len(before.Artifacts) || len(after.Evidence) != len(before.Evidence) || len(after.EvidenceLinks) != len(before.EvidenceLinks) || len(after.Decisions) != len(before.Decisions) || len(after.Relations) != len(before.Relations) || len(store.SnapshotDomainEvents(owner.Scope())) != eventsBefore || store.IdempotencyRecordCount() != receiptsBefore {
		t.Fatal("cross-Outcome documentary denial persisted a record, relation, revision, event, or receipt")
	}
}

func TestDocumentaryEvidenceBatchRollsBackWhenLaterLinkHasForeignOutcome(t *testing.T) {
	service, store, _, ctx, _, _, ns := newOperationalAuthService(t)
	owner := createOperationalAuthOutcome(t, service, ctx, ns)
	foreign, err := service.CreateOutcome(ctx, planningIntent("e16-document-link-foreign"), application.CreateOutcomeCommand{
		NamespaceID: ns, Title: "Foreign Evidence target", DesiredState: "Link remains within one Outcome", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := service.GetOutcomeState(ctx, owner.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(store.SnapshotDomainEvents(owner.Scope()))
	receiptsBefore := store.IdempotencyRecordCount()
	_, err = service.RegisterEvidenceWithLinks(ctx, planningIntent("e16-document-late-cross-link"), application.RegisterEvidenceWithLinksCommand{
		Scope: owner.Scope(),
		Evidence: &application.EvidenceRegistration{
			EvidenceType: domain.EvidenceTypeSource, Description: "Evidence batch with a foreign final target",
			SourceRef: domain.SourceReference{Provider: "external", URI: "https://example.test/evidence"}, CapturedAt: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
		},
		Artifact: &application.ArtifactRegistration{ArtifactType: "report", Name: "Compound attachment", URI: "https://example.test/evidence"},
		Links:    []application.EvidenceLinkInput{{TargetRef: owner.Ref(), Stance: domain.EvidenceStanceContext, Rationale: "First local link."}, {TargetRef: foreign.Value.Ref(), Stance: domain.EvidenceStanceContext, Rationale: "Later foreign link."}},
	})
	if err == nil {
		t.Fatal("evidence batch accepted a later link to a foreign Outcome")
	}
	after, err := service.GetOutcomeState(ctx, owner.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if after.OutcomeRevision != before.OutcomeRevision || len(after.Artifacts) != len(before.Artifacts) || len(after.Evidence) != len(before.Evidence) || len(after.EvidenceLinks) != len(before.EvidenceLinks) || len(store.SnapshotDomainEvents(owner.Scope())) != eventsBefore || store.IdempotencyRecordCount() != receiptsBefore {
		t.Fatal("foreign later link partially persisted Artifact/Evidence/link, revision, event, or receipt")
	}
}

func TestDependencyRemovalValidatesStoredSourceBeforeWritingAndRechecksReplay(t *testing.T) {
	service, store, policy, ctx, _, _, ns := newOperationalAuthService(t)
	outcome := createOperationalAuthOutcome(t, service, ctx, ns)
	source := createOperationalAuthObjective(t, service, ctx, outcome.Scope(), "e16-dependency-source", "Dependency source")
	target := createOperationalAuthObjective(t, service, ctx, outcome.Scope(), "e16-dependency-target", "Dependency target")
	created, err := service.AddDependency(ctx, planningIntent("e16-add-dependency"), application.AddDependencyCommand{Scope: outcome.Scope(), SourceRef: source.Ref(), TargetRef: target.Ref(), Strength: domain.DependencyStrengthHard})
	if err != nil {
		t.Fatal(err)
	}
	before, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	receiptsBefore := store.IdempotencyRecordCount()
	cmdContext := planningIntent("e16-remove-dependency")
	cmd := application.ApplyDependencyChangesCommand{
		Scope: outcome.Scope(), ExpectedOutcomeRevision: before.OutcomeRevision,
		ExpectedSources: []application.DependencySourceVersion{{SourceRef: target.Ref(), ExpectedVersion: target.Version}},
		Changes:         []application.DependencyChange{{Action: application.DependencyChangeRemove, RelationID: &created.Value.ID, ExpectedVersion: &created.Value.Version, Reason: "Remove in bounded test."}},
	}
	if _, err := service.ApplyDependencyChanges(ctx, cmdContext, cmd); err == nil {
		t.Fatal("removal accepted target as source instead of deriving source from stored Relation")
	}
	afterInvalid, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if afterInvalid.OutcomeRevision != before.OutcomeRevision || len(afterInvalid.Relations) != len(before.Relations) || afterInvalid.Relations[0].Lifecycle != domain.RelationLifecycleActive || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore || store.IdempotencyRecordCount() != receiptsBefore {
		t.Fatal("invalid source claim changed relation, revision, events, or receipts")
	}
	cmd.ExpectedSources = []application.DependencySourceVersion{{SourceRef: source.Ref(), ExpectedVersion: source.Version}}
	removed, err := service.ApplyDependencyChanges(ctx, cmdContext, cmd)
	if err != nil || removed.IdempotentReplay {
		t.Fatalf("correct source removal did not execute fresh after invalid request: %#v err=%v", removed, err)
	}
	afterRemoved, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsAfterRemove := len(store.SnapshotDomainEvents(outcome.Scope()))
	policy.denied = ports.PermissionDependencyWrite
	if _, err := service.ApplyDependencyChanges(ctx, cmdContext, cmd); err == nil {
		t.Fatal("dependency replay disclosed result after dependency:write was revoked")
	}
	afterReplayDenied, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if afterReplayDenied.OutcomeRevision != afterRemoved.OutcomeRevision || len(afterReplayDenied.Relations) != len(afterRemoved.Relations) || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsAfterRemove {
		t.Fatal("denied dependency replay changed persisted graph")
	}
}
