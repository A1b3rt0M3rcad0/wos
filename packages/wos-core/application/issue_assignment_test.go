package application_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type issueAssignmentTestIDs struct {
	mu   sync.Mutex
	next int
}

func (g *issueAssignmentTestIDs) NewID() (domain.ID, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.next++
	return domain.ParseID(fmt.Sprintf("0199fa16-0000-7000-8000-%012x", g.next))
}

type issueAssignmentAllowAuthorizer struct{}

func (issueAssignmentAllowAuthorizer) Authorize(_ context.Context, request ports.AuthorizationRequest) error {
	return request.Validate()
}

type issueAssignmentFixture struct {
	service  *application.Service
	store    *memory.Store
	ids      *issueAssignmentTestIDs
	scope    domain.Scope
	issue    domain.Issue
	identity application.Identity
	ctx      context.Context
}

func newIssueAssignmentFixture(t *testing.T) issueAssignmentFixture {
	t.Helper()
	ids := &issueAssignmentTestIDs{}
	store := memory.New()
	service, err := application.NewAuthorizedService(
		store,
		fixedClock{now: time.Date(2026, 10, 20, 19, 0, 0, 0, time.UTC)},
		ids,
		issueAssignmentAllowAuthorizer{},
	)
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := id("0199fa16-0000-7000-8000-000000000001")
	identity := application.Identity{
		NamespaceID: namespaceID,
		PrincipalID: "principal:alice",
		Actor:       domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "external-agent-runtime", ID: "agent/alice"},
	}
	ctx := application.WithIdentity(context.Background(), identity)
	cc := func() domain.CommandContext { return issueAssignmentCommandContext(t, ids, identity, "") }

	created, err := service.CreateOutcome(ctx, cc(), application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Assignment lifecycle",
		DesiredState: "Issue ownership is auditable and race safe",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.AddCriterion(ctx, cc(), application.AddCriterionCommand{
		Owner: created.Value.Ref(), ExpectedVersion: created.Value.Version,
		Title: "Outcome can be operated", Required: true,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	activated, err := service.ActivateOutcome(ctx, cc(), application.ActivateOutcomeCommand{
		Scope: created.Value.Scope(), ExpectedVersion: created.Value.Version + 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective, err := service.CreateObjective(ctx, cc(), application.CreateObjectiveCommand{
		Scope: activated.Value.Scope(), Title: "Own the Issue", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	issue, err := service.CreateIssue(ctx, cc(), application.CreateIssueCommand{
		Scope: objective.Value.Scope, ObjectiveID: objective.Value.ID,
		Title: "Unassigned operational issue", Severity: domain.IssueSeverityMajor,
	})
	if err != nil {
		t.Fatal(err)
	}
	return issueAssignmentFixture{
		service: service, store: store, ids: ids, scope: objective.Value.Scope,
		issue: issue.Value, identity: identity, ctx: ctx,
	}
}

func issueAssignmentIdentity(namespaceID domain.ID, principal, actorID string) application.Identity {
	return application.Identity{
		NamespaceID: namespaceID,
		PrincipalID: principal,
		Actor:       domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "sso", ID: actorID},
	}
}

func issueAssignmentCommandContext(t testing.TB, ids *issueAssignmentTestIDs, identity application.Identity, idempotencyKey string) domain.CommandContext {
	t.Helper()
	commandID, err := ids.NewID()
	if err != nil {
		t.Fatal(err)
	}
	return domain.CommandContext{
		PrincipalID:    identity.PrincipalID,
		Actor:          identity.Actor,
		CommandID:      commandID,
		IdempotencyKey: idempotencyKey,
	}
}

func issueAssignmentContext(identity application.Identity) context.Context {
	return application.WithIdentity(context.Background(), identity)
}

func TestIssueAssignmentApplicationAssignReassignAndUnassignAuditHistory(t *testing.T) {
	f := newIssueAssignmentFixture(t)
	firstIdentity := f.identity
	firstAssignee := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "agent-runtime", ID: "implementation-agent"}

	assigned, err := f.service.AssignIssue(f.ctx, issueAssignmentCommandContext(t, f.ids, firstIdentity, ""), application.AssignIssueCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: f.issue.Version, AssigneeRef: firstAssignee,
	})
	if err != nil {
		t.Fatal(err)
	}
	if assigned.Value.AssigneeRef == nil || *assigned.Value.AssigneeRef != firstAssignee || assigned.Value.Lifecycle != domain.IssueLifecycleOpen {
		t.Fatalf("assigned Issue = %#v", assigned.Value)
	}
	if assigned.Value.Version != f.issue.Version+1 || assigned.OutcomeRevision == 0 {
		t.Fatalf("assign version/revision = %d/%d", assigned.Value.Version, assigned.OutcomeRevision)
	}
	assertIssueAssignmentEvent(t, f.store, f.scope, assigned.OutcomeRevision, "issue.assigned", nil, &firstAssignee, "", firstIdentity)

	secondIdentity := issueAssignmentIdentity(f.scope.NamespaceID, "principal:bob", "sso-user-bob")
	secondCtx := issueAssignmentContext(secondIdentity)
	secondAssignee := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "directory", ID: "operator-12"}
	reassigned, err := f.service.AssignIssue(secondCtx, issueAssignmentCommandContext(t, f.ids, secondIdentity, ""), application.AssignIssueCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: assigned.Value.Version, AssigneeRef: secondAssignee,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reassigned.Value.AssigneeRef == nil || *reassigned.Value.AssigneeRef != secondAssignee || reassigned.Value.Version != assigned.Value.Version+1 {
		t.Fatalf("reassigned Issue = %#v", reassigned.Value)
	}
	assertIssueAssignmentEvent(t, f.store, f.scope, reassigned.OutcomeRevision, "issue.assigned", &firstAssignee, &secondAssignee, "", secondIdentity)

	unassigned, err := f.service.UnassignIssue(secondCtx, issueAssignmentCommandContext(t, f.ids, secondIdentity, ""), application.UnassignIssueCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: reassigned.Value.Version, Reason: "return to the shared triage queue",
	})
	if err != nil {
		t.Fatal(err)
	}
	if unassigned.Value.AssigneeRef != nil || unassigned.Value.Lifecycle != domain.IssueLifecycleOpen || unassigned.Value.Version != reassigned.Value.Version+1 {
		t.Fatalf("unassigned Issue = %#v", unassigned.Value)
	}
	assertIssueAssignmentEvent(t, f.store, f.scope, unassigned.OutcomeRevision, "issue.unassigned", &secondAssignee, nil, "return to the shared triage queue", secondIdentity)
}

func TestStartIssueInvestigationAssignsAuthenticatedActorInOneRevision(t *testing.T) {
	f := newIssueAssignmentFixture(t)
	firstIdentity := f.identity
	firstAssignee := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "directory", ID: "first-responder"}
	assigned, err := f.service.AssignIssue(f.ctx, issueAssignmentCommandContext(t, f.ids, firstIdentity, ""), application.AssignIssueCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: f.issue.Version, AssigneeRef: firstAssignee,
	})
	if err != nil {
		t.Fatal(err)
	}

	investigator := issueAssignmentIdentity(f.scope.NamespaceID, "principal:investigator", "mapped-actor-77")
	started, err := f.service.StartIssueInvestigation(issueAssignmentContext(investigator), issueAssignmentCommandContext(t, f.ids, investigator, ""), application.StartIssueInvestigationCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: assigned.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if started.Value.Lifecycle != domain.IssueLifecycleInvestigating || started.Value.AssigneeRef == nil || *started.Value.AssigneeRef != investigator.Actor {
		t.Fatalf("investigating Issue = %#v", started.Value)
	}
	if started.Value.Version != assigned.Value.Version+1 || started.OutcomeRevision != assigned.OutcomeRevision+1 {
		t.Fatalf("start must advance Issue version and Outcome revision once: version=%d revision=%d", started.Value.Version, started.OutcomeRevision)
	}
	assertIssueAssignmentEvent(t, f.store, f.scope, started.OutcomeRevision, "issue.investigating", &firstAssignee, &investigator.Actor, "", investigator)

	eventsAtStart := eventsAtRevision(f.store, f.scope, started.OutcomeRevision)
	if len(eventsAtStart) != 1 {
		t.Fatalf("start-investigation events at revision %d = %d, want one atomic event", started.OutcomeRevision, len(eventsAtStart))
	}
	if _, err = f.service.StartIssueInvestigation(issueAssignmentContext(investigator), issueAssignmentCommandContext(t, f.ids, investigator, ""), application.StartIssueInvestigationCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: started.Value.Version,
	}); err == nil {
		t.Fatal("already investigating Issue accepted another start")
	}
}

func TestLegacyInvestigateIssueAlsoAssignsAuthenticatedActor(t *testing.T) {
	f := newIssueAssignmentFixture(t)
	result, err := f.service.InvestigateIssue(f.ctx, issueAssignmentCommandContext(t, f.ids, f.identity, ""), application.InvestigateIssueCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: f.issue.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Value.Lifecycle != domain.IssueLifecycleInvestigating || result.Value.AssigneeRef == nil || *result.Value.AssigneeRef != f.identity.Actor {
		t.Fatalf("legacy InvestigateIssue produced an unassigned investigation: %#v", result.Value)
	}
	assertIssueAssignmentEvent(t, f.store, f.scope, result.OutcomeRevision, "issue.investigating", nil, &f.identity.Actor, "", f.identity)
}

func TestIssueAssignmentNoOpsStaleVersionsAndIdempotentReplay(t *testing.T) {
	f := newIssueAssignmentFixture(t)
	before := len(f.store.SnapshotDomainEvents(f.scope))

	firstCC := issueAssignmentCommandContext(t, f.ids, f.identity, "issue-assign-once")
	assigned, err := f.service.AssignIssue(f.ctx, firstCC, application.AssignIssueCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: f.issue.Version, AssigneeRef: f.identity.Actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := f.service.AssignIssue(f.ctx, issueAssignmentCommandContext(t, f.ids, f.identity, "issue-assign-once"), application.AssignIssueCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: f.issue.Version, AssigneeRef: f.identity.Actor,
	})
	if err != nil || !replayed.IdempotentReplay || replayed.Value.Version != assigned.Value.Version || replayed.CommandID != assigned.CommandID {
		t.Fatalf("assignment replay = %#v, %v; original %#v", replayed, err, assigned)
	}
	if got := len(f.store.SnapshotDomainEvents(f.scope)); got != before+1 {
		t.Fatalf("assignment replay emitted duplicate events: before=%d after=%d", before, got)
	}

	noop, err := f.service.AssignIssue(f.ctx, issueAssignmentCommandContext(t, f.ids, f.identity, ""), application.AssignIssueCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: assigned.Value.Version, AssigneeRef: f.identity.Actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if noop.Value.Version != assigned.Value.Version || noop.OutcomeRevision != assigned.OutcomeRevision || len(f.store.SnapshotDomainEvents(f.scope)) != before+1 {
		t.Fatalf("same-assignee no-op changed state/events: result=%#v events=%d", noop, len(f.store.SnapshotDomainEvents(f.scope)))
	}
	noopKey := "issue-assign-noop"
	keyedNoop, err := f.service.AssignIssue(f.ctx, issueAssignmentCommandContext(t, f.ids, f.identity, noopKey), application.AssignIssueCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: assigned.Value.Version, AssigneeRef: f.identity.Actor,
	})
	if err != nil || keyedNoop.Value.Version != assigned.Value.Version || keyedNoop.OutcomeRevision != assigned.OutcomeRevision {
		t.Fatalf("idempotent no-op = %#v, %v", keyedNoop, err)
	}
	keyedNoopReplay, err := f.service.AssignIssue(f.ctx, issueAssignmentCommandContext(t, f.ids, f.identity, noopKey), application.AssignIssueCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: assigned.Value.Version, AssigneeRef: f.identity.Actor,
	})
	if err != nil || !keyedNoopReplay.IdempotentReplay || keyedNoopReplay.Value.Version != keyedNoop.Value.Version || keyedNoopReplay.OutcomeRevision != keyedNoop.OutcomeRevision {
		t.Fatalf("idempotent no-op replay = %#v, %v; original %#v", keyedNoopReplay, err, keyedNoop)
	}
	if len(f.store.SnapshotDomainEvents(f.scope)) != before+1 {
		t.Fatal("idempotent no-op or replay emitted an event")
	}

	unassigned, err := f.service.UnassignIssue(f.ctx, issueAssignmentCommandContext(t, f.ids, f.identity, ""), application.UnassignIssueCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: assigned.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if unassigned.Value.Version != assigned.Value.Version+1 || len(f.store.SnapshotDomainEvents(f.scope)) != before+2 {
		t.Fatalf("unassignment setup failed: result=%#v events=%d", unassigned, len(f.store.SnapshotDomainEvents(f.scope)))
	}
	noopUnassign, err := f.service.UnassignIssue(f.ctx, issueAssignmentCommandContext(t, f.ids, f.identity, ""), application.UnassignIssueCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: unassigned.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if noopUnassign.Value.Version != unassigned.Value.Version || noopUnassign.OutcomeRevision != unassigned.OutcomeRevision || len(f.store.SnapshotDomainEvents(f.scope)) != before+2 {
		t.Fatalf("already-unassigned no-op changed state/events: result=%#v events=%d", noopUnassign, len(f.store.SnapshotDomainEvents(f.scope)))
	}

	_, err = f.service.AssignIssue(f.ctx, issueAssignmentCommandContext(t, f.ids, f.identity, ""), application.AssignIssueCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: f.issue.Version, AssigneeRef: f.identity.Actor,
	})
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeVersionConflict {
		t.Fatalf("stale same-state assignment error = %v (%s), want version conflict", err, code)
	}
}

func TestIssueAssignmentRejectsCommandIdentityDifferentFromAuthenticatedIdentity(t *testing.T) {
	f := newIssueAssignmentFixture(t)
	forged := issueAssignmentCommandContext(t, f.ids, f.identity, "")
	forged.Actor = domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "sso", ID: "someone-else"}
	_, err := f.service.StartIssueInvestigation(f.ctx, forged, application.StartIssueInvestigationCommand{
		Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: f.issue.Version,
	})
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeForbidden {
		t.Fatalf("mismatched command Actor error = %v (%s), want forbidden", err, code)
	}
	if len(f.store.SnapshotDomainEvents(f.scope)) == 0 {
		t.Fatal("test fixture should already contain its Issue creation event")
	}
}

func TestConcurrentIssueInvestigationHasOneVersionedWinner(t *testing.T) {
	f := newIssueAssignmentFixture(t)
	identities := []application.Identity{
		issueAssignmentIdentity(f.scope.NamespaceID, "principal:first", "actor-first"),
		issueAssignmentIdentity(f.scope.NamespaceID, "principal:second", "actor-second"),
	}
	start := make(chan struct{})
	results := make(chan struct {
		identity application.Identity
		result   application.MutationResult[domain.Issue]
		err      error
	}, len(identities))
	commands := make([]domain.CommandContext, len(identities))
	for i, identity := range identities {
		commands[i] = issueAssignmentCommandContext(t, f.ids, identity, "")
	}
	for i, identity := range identities {
		identity := identity
		command := commands[i]
		go func() {
			<-start
			result, err := f.service.StartIssueInvestigation(issueAssignmentContext(identity), command, application.StartIssueInvestigationCommand{
				Scope: f.scope, IssueID: f.issue.ID, ExpectedVersion: f.issue.Version,
			})
			results <- struct {
				identity application.Identity
				result   application.MutationResult[domain.Issue]
				err      error
			}{identity: identity, result: result, err: err}
		}()
	}
	close(start)

	successes, conflicts := 0, 0
	var winner application.Identity
	var winningResult application.MutationResult[domain.Issue]
	for range identities {
		got := <-results
		if got.err == nil {
			successes++
			winner, winningResult = got.identity, got.result
			continue
		}
		if code, _ := domain.ErrorCodeOf(got.err); code == domain.ErrorCodeVersionConflict {
			conflicts++
			continue
		}
		t.Fatalf("unexpected concurrent start error = %v", got.err)
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent starts: successes=%d conflicts=%d", successes, conflicts)
	}
	if winningResult.Value.AssigneeRef == nil || *winningResult.Value.AssigneeRef != winner.Actor || winningResult.Value.Version != f.issue.Version+1 {
		t.Fatalf("winning assignment = %#v for actor %#v", winningResult.Value, winner.Actor)
	}
	events := eventsAtRevision(f.store, f.scope, winningResult.OutcomeRevision)
	if len(events) != 1 || events[0].EventType != "issue.investigating" || events[0].PrincipalID != winner.PrincipalID || events[0].Actor != winner.Actor {
		t.Fatalf("winning investigation audit events = %#v", events)
	}
}

func assertIssueAssignmentEvent(
	t *testing.T,
	store *memory.Store,
	scope domain.Scope,
	revision domain.OutcomeRevision,
	eventType string,
	previous, next *domain.ActorRef,
	reason string,
	identity application.Identity,
) {
	t.Helper()
	events := eventsAtRevision(store, scope, revision)
	if len(events) != 1 {
		t.Fatalf("events at revision %d = %d, want one: %#v", revision, len(events), events)
	}
	event := events[0]
	if event.EventType != eventType || event.PrincipalID != identity.PrincipalID || event.Actor != identity.Actor {
		t.Fatalf("audit identity/type = %s %q %#v, want %s %q %#v", event.EventType, event.PrincipalID, event.Actor, eventType, identity.PrincipalID, identity.Actor)
	}
	if event.AggregateVersionBefore == nil || event.AggregateVersionAfter == nil || *event.AggregateVersionAfter != *event.AggregateVersionBefore+1 {
		t.Fatalf("audit aggregate versions = %v -> %v", event.AggregateVersionBefore, event.AggregateVersionAfter)
	}
	var payload struct {
		PreviousAssigneeRef *domain.ActorRef `json:"previous_assignee_ref"`
		AssigneeRef         *domain.ActorRef `json:"assignee_ref"`
		Reason              string           `json:"reason"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if !sameIssueActorRef(payload.PreviousAssigneeRef, previous) || !sameIssueActorRef(payload.AssigneeRef, next) || payload.Reason != reason {
		t.Fatalf("audit assignee payload = %#v, want previous=%#v next=%#v reason=%q", payload, previous, next, reason)
	}
}

func sameIssueActorRef(a, b *domain.ActorRef) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func eventsAtRevision(store *memory.Store, scope domain.Scope, revision domain.OutcomeRevision) []domain.DomainEvent {
	all := store.SnapshotDomainEvents(scope)
	matched := make([]domain.DomainEvent, 0, 1)
	for _, event := range all {
		if event.OutcomeRevision == revision {
			matched = append(matched, event)
		}
	}
	return matched
}
