package application_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func observedRevision(t *testing.T, service *application.Service, scope domain.Scope) domain.OutcomeRevision {
	t.Helper()
	state, err := service.GetOutcomeState(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	return state.OutcomeRevision
}

func TestReportIssueWithBlockersIsAtomicVersionPinnedAndIdempotent(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	owner := issueObjectiveForTest(t, service, outcome.Scope())
	first := createTodoWork(t, service, outcome.Scope(), "First affected task")
	second := createTodoWork(t, service, outcome.Scope(), "Second affected task")
	revision := observedRevision(t, service, outcome.Scope())
	cc := commandContext()
	cc.IdempotencyKey = "report-multi-blocker-once"
	command := application.ReportIssueWithBlockersCommand{
		Scope:                   outcome.Scope(),
		ObjectiveID:             owner,
		Title:                   "Shared infrastructure outage",
		Description:             "One outage currently blocks several distinct targets.",
		Severity:                domain.IssueSeverityCritical,
		AffectedRefs:            []domain.EntityRef{first.Ref(), second.Ref()},
		ObservedOutcomeRevision: &revision,
		Blockers: []application.IssueBlockerInput{
			{BlockedRef: first.Ref(), Description: "First task cannot proceed."},
			{BlockedRef: second.Ref(), Description: "Second task cannot proceed."},
			{BlockedRef: domain.EntityRef{Scope: outcome.Scope(), Kind: domain.EntityKindObjective, ID: owner}, Description: "The objective is affected.", Propagation: domain.BlockerPropagationSubtree},
		},
	}

	created, err := service.ReportIssueWithBlockers(ctx, cc, command)
	if err != nil {
		t.Fatal(err)
	}
	if created.Value.Issue.ObjectiveID == nil || *created.Value.Issue.ObjectiveID != owner || len(created.Value.Blockers) != len(command.Blockers) {
		t.Fatalf("multi-target result = %#v", created.Value)
	}
	if created.OutcomeRevision != revision+1 {
		t.Fatalf("revision advanced from %d to %d, want one atomic advance", revision, created.OutcomeRevision)
	}
	for i, blocker := range created.Value.Blockers {
		if blocker.CauseRef == nil || *blocker.CauseRef != created.Value.Issue.Ref() || blocker.BlockedRef != command.Blockers[i].BlockedRef {
			t.Fatalf("blocker %d lost target order or Issue cause: %#v", i, blocker)
		}
	}
	events := store.SnapshotDomainEvents(outcome.Scope())
	var sameRevision []domain.DomainEvent
	for _, event := range events {
		if event.OutcomeRevision == created.OutcomeRevision {
			sameRevision = append(sameRevision, event)
		}
	}
	if len(sameRevision) != 1+len(command.Blockers) || sameRevision[0].EventType != "issue.created" || sameRevision[0].AggregateRef != created.Value.Issue.Ref() {
		t.Fatalf("compound events = %#v", sameRevision)
	}
	for i, blocker := range created.Value.Blockers {
		event := sameRevision[i+1]
		if event.EventType != "blocker.created" || event.AggregateRef != blocker.Ref() || event.CommandID != created.CommandID || event.OutcomeRevision != created.OutcomeRevision {
			t.Fatalf("blocker event %d = %#v", i, event)
		}
	}

	replayed, err := service.ReportIssueWithBlockers(ctx, cc, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.IdempotentReplay || replayed.Value.Issue.ID != created.Value.Issue.ID || !reflect.DeepEqual(replayed.Value.Blockers, created.Value.Blockers) || replayed.OutcomeRevision != created.OutcomeRevision || len(store.SnapshotDomainEvents(outcome.Scope())) != len(events) {
		t.Fatalf("same idempotency key created another compound operation: replay=%#v", replayed)
	}
	conflicting := command
	conflicting.Title = "Different operation under the same key"
	_, err = service.ReportIssueWithBlockers(ctx, cc, conflicting)
	var coded *domain.Error
	if !errors.As(err, &coded) || coded.Code != domain.ErrorCodeIdempotencyConflict {
		t.Fatalf("changed request under the same idempotency key = %v, want idempotency conflict", err)
	}
	if len(store.SnapshotDomainEvents(outcome.Scope())) != len(events) {
		t.Fatal("idempotency conflict created another event")
	}
}

func TestReportIssueWithBlockersRejectsInvalidAndDuplicateTargetsWithoutWrites(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	owner := issueObjectiveForTest(t, service, outcome.Scope())
	work := createTodoWork(t, service, outcome.Scope(), "Existing target")
	revision := observedRevision(t, service, outcome.Scope())
	missing := domain.EntityRef{Scope: outcome.Scope(), Kind: domain.EntityKindWorkItem, ID: id("0199e2ff-0000-7000-8000-00000000feed")}
	valid := application.IssueBlockerInput{BlockedRef: work.Ref(), Description: "Valid impact."}
	base := application.ReportIssueWithBlockersCommand{
		Scope: outcome.Scope(), ObjectiveID: owner, Title: "Broken dependency", Severity: domain.IssueSeverityMajor,
		ObservedOutcomeRevision: &revision,
		Blockers:                []application.IssueBlockerInput{valid, {BlockedRef: missing, Description: "Missing target."}},
	}
	stateBefore, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := store.SnapshotDomainEvents(outcome.Scope())
	if _, err := service.ReportIssueWithBlockers(ctx, commandContext(), base); err == nil {
		t.Fatal("multi-target operation accepted a missing target")
	}
	stateAfter, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stateBefore, stateAfter) || !reflect.DeepEqual(eventsBefore, store.SnapshotDomainEvents(outcome.Scope())) {
		t.Fatal("invalid target left a partial Issue/Blocker write, event or revision")
	}

	duplicate := base
	duplicate.Blockers = []application.IssueBlockerInput{valid, {BlockedRef: work.Ref(), Description: "Same target and default propagation."}}
	if _, err := service.ReportIssueWithBlockers(ctx, commandContext(), duplicate); err == nil {
		t.Fatal("duplicate target/propagation pair was accepted")
	}
	stateAfterDuplicate, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stateBefore, stateAfterDuplicate) || !reflect.DeepEqual(eventsBefore, store.SnapshotDomainEvents(outcome.Scope())) {
		t.Fatal("duplicate target validation mutated state")
	}

	invalidPropagation := base
	invalidPropagation.Blockers = []application.IssueBlockerInput{{BlockedRef: work.Ref(), Propagation: domain.BlockerPropagationSubtree}}
	if _, err := service.ReportIssueWithBlockers(ctx, commandContext(), invalidPropagation); err == nil {
		t.Fatal("work item blocker accepted subtree propagation")
	}
	if !reflect.DeepEqual(stateBefore, mustOutcomeState(t, service, outcome.Scope())) || !reflect.DeepEqual(eventsBefore, store.SnapshotDomainEvents(outcome.Scope())) {
		t.Fatal("invalid propagation validation mutated state")
	}
}

func TestReportIssueWithBlockersRequiresFreshReviewAndCompleteBoundedInput(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	owner := issueObjectiveForTest(t, service, outcome.Scope())
	work := createTodoWork(t, service, outcome.Scope(), "Review target")
	observed := observedRevision(t, service, outcome.Scope())
	command := application.ReportIssueWithBlockersCommand{
		Scope: outcome.Scope(), ObjectiveID: owner, Title: "Reviewed impact", Severity: domain.IssueSeverityMinor,
		ObservedOutcomeRevision: &observed,
		Blockers:                []application.IssueBlockerInput{{BlockedRef: work.Ref(), Description: "Waiting on the reviewed cause."}},
	}
	validationBaseline, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	validationEvents := store.SnapshotDomainEvents(outcome.Scope())
	missingReview := command
	missingReview.ObservedOutcomeRevision = nil
	if _, err := service.ReportIssueWithBlockers(ctx, commandContext(), missingReview); err == nil {
		t.Fatal("multi-target operation accepted an omitted observed revision")
	}
	tooMany := command
	tooMany.Blockers = make([]application.IssueBlockerInput, 65)
	for i := range tooMany.Blockers {
		tooMany.Blockers[i] = application.IssueBlockerInput{BlockedRef: domain.EntityRef{Scope: outcome.Scope(), Kind: domain.EntityKindObjective, ID: owner}, Propagation: domain.BlockerPropagationDirect, Description: "A distinct explicitly reviewed impact."}
		// Use distinct allowed pairs so this case reaches the configured limit.
		if i%2 == 1 {
			tooMany.Blockers[i].Propagation = domain.BlockerPropagationSubtree
		}
	}
	if _, err := service.ReportIssueWithBlockers(ctx, commandContext(), tooMany); err == nil {
		t.Fatal("operation exceeded the configured multi-blocker limit")
	}
	tooLarge := command
	tooLarge.Blockers = make([]application.IssueBlockerInput, 64)
	for i := range tooLarge.Blockers {
		tooLarge.Blockers[i] = application.IssueBlockerInput{BlockedRef: work.Ref(), Description: strings.Repeat("x", 5000)}
	}
	if _, err := service.ReportIssueWithBlockers(ctx, commandContext(), tooLarge); err == nil {
		t.Fatal("operation exceeded the 256 KiB command envelope")
	}
	if !reflect.DeepEqual(validationBaseline, mustOutcomeState(t, service, outcome.Scope())) || !reflect.DeepEqual(validationEvents, store.SnapshotDomainEvents(outcome.Scope())) {
		t.Fatal("rejected multi-target input validation wrote Issue/Blocker records or events")
	}

	changed, err := service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{Scope: outcome.Scope(), ObjectiveID: owner, Title: "Another update", Severity: domain.IssueSeverityMinor})
	if err != nil {
		t.Fatal(err)
	}
	stateBefore, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := store.SnapshotDomainEvents(outcome.Scope())
	_, err = service.ReportIssueWithBlockers(ctx, commandContext(), command)
	var coded *domain.Error
	if !errors.As(err, &coded) || coded.Code != domain.ErrorCodePreconditionFailed || coded.Details == nil || coded.Details.ReasonCode != "blocker_impact_changed" {
		t.Fatalf("stale reviewed revision error = %#v", err)
	}
	if changed.Value.ID == "" {
		t.Fatal("test setup did not create the intervening revision")
	}
	if !reflect.DeepEqual(stateBefore, mustOutcomeState(t, service, outcome.Scope())) || !reflect.DeepEqual(eventsBefore, store.SnapshotDomainEvents(outcome.Scope())) {
		t.Fatal("stale review rejection wrote an Issue/Blocker or advanced the revision")
	}
}

func mustOutcomeState(t *testing.T, service *application.Service, scope domain.Scope) application.OutcomeState {
	t.Helper()
	state, err := service.GetOutcomeState(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	return state
}
