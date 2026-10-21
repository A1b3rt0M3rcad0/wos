package application_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

type dependencySourcePolicy struct {
	mu               sync.Mutex
	denied           *domain.EntityRef
	resources        []domain.EntityRef
	commandHook      func()
	resourceAuthHook func(domain.EntityRef)
}

func (p *dependencySourcePolicy) Authorize(_ context.Context, request ports.AuthorizationRequest) error {
	return p.check(request, false)
}

func (p *dependencySourcePolicy) AuthorizeInUnitOfWork(_ context.Context, _ ports.UnitOfWork, request ports.AuthorizationRequest) error {
	return p.check(request, true)
}

func (p *dependencySourcePolicy) check(request ports.AuthorizationRequest, transactional bool) error {
	if err := request.Validate(); err != nil {
		return err
	}
	if request.Permission == ports.PermissionDependencyWrite && request.ResourceRef != nil {
		p.mu.Lock()
		p.resources = append(p.resources, *request.ResourceRef)
		denied := p.denied != nil && *p.denied == *request.ResourceRef
		hook := p.resourceAuthHook
		p.mu.Unlock()
		if hook != nil {
			hook(*request.ResourceRef)
		}
		if denied {
			return domain.NewDetailedError(domain.ErrorCodeForbidden, "principal cannot change dependencies for this source", domain.ErrorDetails{SubjectRef: request.ResourceRef})
		}
	} else if request.Permission == ports.PermissionDependencyWrite && !transactional {
		p.mu.Lock()
		hook := p.commandHook
		p.mu.Unlock()
		if hook != nil {
			hook()
		}
	}
	return nil
}

type dependencyBatchTestResult struct {
	mutation application.MutationResult[application.DependencyChangesResult]
	err      error
}

func (p *dependencySourcePolicy) reset(denied *domain.EntityRef) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.denied = denied
	p.resources = nil
}

func (p *dependencySourcePolicy) requests() []domain.EntityRef {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]domain.EntityRef(nil), p.resources...)
}

type dependencySourceFixture struct {
	service *application.Service
	store   *memory.Store
	policy  *dependencySourcePolicy
	ctx     context.Context
	ids     *sequenceIDs
	scope   domain.Scope
}

func newDependencySourceFixture(t *testing.T) dependencySourceFixture {
	t.Helper()
	store := memory.New()
	ids := &sequenceIDs{}
	clock := fixedClock{now: time.Date(2026, 10, 21, 19, 0, 0, 0, time.UTC)}
	trusted, err := application.NewService(store, clock, ids)
	if err != nil {
		t.Fatal(err)
	}
	outcome := setupActiveOutcome(t, trusted)
	policy := &dependencySourcePolicy{}
	service, err := application.NewAuthorizedService(store, clock, ids, policy)
	if err != nil {
		t.Fatal(err)
	}
	identity := application.Identity{NamespaceID: outcome.Scope().NamespaceID, PrincipalID: "human-1", Actor: commandContext().Actor}
	return dependencySourceFixture{service: service, store: store, policy: policy, ctx: application.WithIdentity(context.Background(), identity), ids: ids, scope: outcome.Scope()}
}

func (f dependencySourceFixture) createObjective(t *testing.T, title string) domain.Objective {
	t.Helper()
	created, err := f.service.CreateObjective(f.ctx, commandContext(), application.CreateObjectiveCommand{
		Scope: f.scope, Title: title, Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	return created.Value
}

func (f dependencySourceFixture) commandContext(t *testing.T, key string) domain.CommandContext {
	t.Helper()
	cc := commandContext()
	if key != "" {
		cc.IdempotencyKey = key
	}
	return cc
}

func TestDependencyBatchAuthorizesEachSourceAndRollsBackAtomically(t *testing.T) {
	f := newDependencySourceFixture(t)
	first := f.createObjective(t, "First dependency source")
	second := f.createObjective(t, "Second dependency source")
	dependent := f.createObjective(t, "Dependent target")
	before, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(f.store.SnapshotDomainEvents(f.scope))
	receiptsBefore := f.store.IdempotencyRecordCount()
	cmd := application.ApplyDependencyChangesCommand{
		Scope: f.scope, ExpectedOutcomeRevision: before.OutcomeRevision,
		ExpectedSources: []application.DependencySourceVersion{
			{SourceRef: first.Ref(), ExpectedVersion: first.Version},
			{SourceRef: second.Ref(), ExpectedVersion: second.Version},
		},
		Changes: []application.DependencyChange{
			{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(first.Ref()), TargetRef: dependencyRefPointer(dependent.Ref()), Strength: domain.DependencyStrengthAdvisory},
			{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(second.Ref()), TargetRef: dependencyRefPointer(dependent.Ref()), Strength: domain.DependencyStrengthAdvisory},
		},
	}
	key := "dependency-source-denial-rollback"
	secondRef := second.Ref()
	f.policy.reset(&secondRef)
	_, err = f.service.ApplyDependencyChanges(f.ctx, f.commandContext(t, key), cmd)
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeForbidden {
		t.Fatalf("batch denial error = %v, want forbidden", err)
	}
	requests := f.policy.requests()
	if len(requests) != 2 || requests[0] != first.Ref() || requests[1] != second.Ref() {
		t.Fatalf("source authorization requests = %#v, want ordered exact sources %#v and %#v", requests, first.Ref(), second.Ref())
	}
	afterDenied, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	if afterDenied.OutcomeRevision != before.OutcomeRevision || len(afterDenied.Relations) != len(before.Relations) || len(f.store.SnapshotDomainEvents(f.scope)) != eventsBefore || f.store.IdempotencyRecordCount() != receiptsBefore {
		t.Fatal("source denial persisted part of the graph, revision, events, or idempotency receipt")
	}

	f.policy.reset(nil)
	added, err := f.service.ApplyDependencyChanges(f.ctx, f.commandContext(t, key), cmd)
	if err != nil {
		t.Fatal(err)
	}
	if added.OutcomeRevision != before.OutcomeRevision+1 || len(added.Value.Changes) != 2 {
		t.Fatalf("successful batch revision/result = %d/%#v", added.OutcomeRevision, added.Value)
	}
	events := f.store.SnapshotDomainEvents(f.scope)
	var batchEvents int
	for _, event := range events {
		if event.OutcomeRevision == added.OutcomeRevision && event.CommandID == added.CommandID {
			batchEvents++
		}
	}
	if batchEvents != 2 {
		t.Fatalf("successful batch emitted %d events at its single revision, want 2", batchEvents)
	}
}

func TestDependencyBatchReplayReauthorizesStoredSourcesBeforeDisclosure(t *testing.T) {
	f := newDependencySourceFixture(t)
	first := f.createObjective(t, "Replay source one")
	second := f.createObjective(t, "Replay source two")
	dependent := f.createObjective(t, "Replay dependent")
	before, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	cmd := application.ApplyDependencyChangesCommand{
		Scope: f.scope, ExpectedOutcomeRevision: before.OutcomeRevision,
		ExpectedSources: []application.DependencySourceVersion{
			{SourceRef: first.Ref(), ExpectedVersion: first.Version},
			{SourceRef: second.Ref(), ExpectedVersion: second.Version},
		},
		Changes: []application.DependencyChange{
			{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(first.Ref()), TargetRef: dependencyRefPointer(dependent.Ref()), Strength: domain.DependencyStrengthAdvisory},
			{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(second.Ref()), TargetRef: dependencyRefPointer(dependent.Ref()), Strength: domain.DependencyStrengthAdvisory},
		},
	}
	cc := f.commandContext(t, "dependency-source-replay")
	created, err := f.service.ApplyDependencyChanges(f.ctx, cc, cmd)
	if err != nil || created.IdempotentReplay {
		t.Fatalf("initial dependency batch = replay:%v err:%v", created.IdempotentReplay, err)
	}
	expectedSources := []domain.EntityRef{first.Ref(), second.Ref()}
	f.policy.reset(nil)
	replayed, err := f.service.ApplyDependencyChanges(f.ctx, cc, cmd)
	if err != nil || !replayed.IdempotentReplay {
		t.Fatalf("authorized dependency replay = replay:%v err:%v", replayed.IdempotentReplay, err)
	}
	if requests := f.policy.requests(); len(requests) != 2 || requests[0] != expectedSources[0] || requests[1] != expectedSources[1] {
		t.Fatalf("replay authorization sources = %#v, want %#v", requests, expectedSources)
	}
	afterReplay, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	eventsAfterReplay := len(f.store.SnapshotDomainEvents(f.scope))
	receiptsAfterReplay := f.store.IdempotencyRecordCount()
	denied := second.Ref()
	f.policy.reset(&denied)
	_, err = f.service.ApplyDependencyChanges(f.ctx, cc, cmd)
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeForbidden {
		t.Fatalf("revoked-source replay error = %v, want forbidden", err)
	}
	if requests := f.policy.requests(); len(requests) != 2 || requests[0] != expectedSources[0] || requests[1] != expectedSources[1] {
		t.Fatalf("denied replay authorization sources = %#v, want %#v", requests, expectedSources)
	}
	afterDeniedReplay, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	if afterDeniedReplay.OutcomeRevision != afterReplay.OutcomeRevision || len(afterDeniedReplay.Relations) != len(afterReplay.Relations) || len(f.store.SnapshotDomainEvents(f.scope)) != eventsAfterReplay || f.store.IdempotencyRecordCount() != receiptsAfterReplay {
		t.Fatal("denied replay disclosed or changed the persisted graph, revision, events, or receipt")
	}
}

func TestDependencyRemovalReplayUsesStoredRelationSource(t *testing.T) {
	f := newDependencySourceFixture(t)
	source := f.createObjective(t, "Removal replay source")
	target := f.createObjective(t, "Removal replay target")
	created, err := f.service.AddDependency(f.ctx, f.commandContext(t, "dependency-remove-replay-seed"), application.AddDependencyCommand{
		Scope: f.scope, SourceRef: source.Ref(), TargetRef: target.Ref(), Strength: domain.DependencyStrengthAdvisory,
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	cmd := application.ApplyDependencyChangesCommand{
		Scope: f.scope, ExpectedOutcomeRevision: before.OutcomeRevision,
		ExpectedSources: []application.DependencySourceVersion{{SourceRef: source.Ref(), ExpectedVersion: source.Version}},
		Changes:         []application.DependencyChange{{Action: application.DependencyChangeRemove, RelationID: idPointer(created.Value.ID), ExpectedVersion: versionPointer(created.Value.Version), Reason: "remove from required by view"}},
	}
	cc := f.commandContext(t, "dependency-remove-replay")
	removed, err := f.service.ApplyDependencyChanges(f.ctx, cc, cmd)
	if err != nil || removed.IdempotentReplay {
		t.Fatalf("initial dependency removal = replay:%v err:%v", removed.IdempotentReplay, err)
	}
	f.policy.reset(nil)
	replayed, err := f.service.ApplyDependencyChanges(f.ctx, cc, cmd)
	if err != nil || !replayed.IdempotentReplay {
		t.Fatalf("authorized dependency removal replay = replay:%v err:%v", replayed.IdempotentReplay, err)
	}
	if requests := f.policy.requests(); len(requests) != 1 || requests[0] != source.Ref() {
		t.Fatalf("removal replay authorization sources = %#v, want stored source %v", requests, source.Ref())
	}
	denied := source.Ref()
	f.policy.reset(&denied)
	_, err = f.service.ApplyDependencyChanges(f.ctx, cc, cmd)
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeForbidden {
		t.Fatalf("denied dependency removal replay error = %v, want forbidden", err)
	}
}

func TestDependencyRemovalAuthorizesPersistedSourceNotInitiatingTarget(t *testing.T) {
	f := newDependencySourceFixture(t)
	source := f.createObjective(t, "Persisted relation source")
	initiatingTarget := f.createObjective(t, "Target that initiated removal")
	created, err := f.service.AddDependency(f.ctx, f.commandContext(t, "dependency-source-removal-seed"), application.AddDependencyCommand{
		Scope: f.scope, SourceRef: source.Ref(), TargetRef: initiatingTarget.Ref(), Strength: domain.DependencyStrengthAdvisory,
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(f.store.SnapshotDomainEvents(f.scope))
	receiptsBefore := f.store.IdempotencyRecordCount()
	cmd := application.ApplyDependencyChangesCommand{
		Scope: f.scope, ExpectedOutcomeRevision: before.OutcomeRevision,
		ExpectedSources: []application.DependencySourceVersion{{SourceRef: source.Ref(), ExpectedVersion: source.Version}},
		Changes:         []application.DependencyChange{{Action: application.DependencyChangeRemove, RelationID: idPointer(created.Value.ID), ExpectedVersion: versionPointer(created.Value.Version), Reason: "dependency is no longer required"}},
	}
	sourceRef := source.Ref()
	f.policy.reset(&sourceRef)
	_, err = f.service.ApplyDependencyChanges(f.ctx, f.commandContext(t, "dependency-source-removal-denied"), cmd)
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeForbidden {
		t.Fatalf("source-side removal denial error = %v, want forbidden", err)
	}
	requests := f.policy.requests()
	if len(requests) != 1 || requests[0] != source.Ref() || requests[0] == initiatingTarget.Ref() {
		t.Fatalf("removal authorized refs = %#v, want persisted source %v (not target %v)", requests, source.Ref(), initiatingTarget.Ref())
	}
	after, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	if after.OutcomeRevision != before.OutcomeRevision || len(f.store.SnapshotDomainEvents(f.scope)) != eventsBefore || f.store.IdempotencyRecordCount() != receiptsBefore {
		t.Fatal("denied source-side removal persisted a relation change, revision, event, or receipt")
	}
	if len(after.Relations) != len(before.Relations) {
		t.Fatal("denied removal changed the persisted dependency lifecycle")
	}
	for _, relation := range after.Relations {
		if relation.ID == created.Value.ID && relation.Lifecycle != domain.RelationLifecycleActive {
			t.Fatal("denied removal changed the persisted dependency lifecycle")
		}
	}
}

func TestDependencyBatchRejectsRepeatedAddEdge(t *testing.T) {
	f := newDependencySourceFixture(t)
	source := f.createObjective(t, "Repeated edge source")
	target := f.createObjective(t, "Repeated edge target")
	change := application.DependencyChange{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(source.Ref()), TargetRef: dependencyRefPointer(target.Ref()), Strength: domain.DependencyStrengthAdvisory}
	cmd := application.ApplyDependencyChangesCommand{
		Scope: f.scope, ExpectedOutcomeRevision: 1,
		ExpectedSources: []application.DependencySourceVersion{{SourceRef: source.Ref(), ExpectedVersion: source.Version}},
		Changes:         []application.DependencyChange{change, change},
	}
	_, err := f.service.ApplyDependencyChanges(f.ctx, f.commandContext(t, "dependency-repeated-add"), cmd)
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeInvalidArgument {
		t.Fatalf("repeated edge error = %v, want invalid_argument", err)
	}
}

func TestDependencySourceAuthorizationSQLiteDenialRollsBackBatch(t *testing.T) {
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "dependency-source-auth.db"), sqlite.Options{MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ids := &sequenceIDs{}
	clock := fixedClock{now: time.Date(2026, 10, 21, 20, 0, 0, 0, time.UTC)}
	trusted, err := application.NewService(store, clock, ids)
	if err != nil {
		t.Fatal(err)
	}
	outcome := setupActiveOutcome(t, trusted)
	policy := &dependencySourcePolicy{}
	service, err := application.NewAuthorizedService(store, clock, ids, policy)
	if err != nil {
		t.Fatal(err)
	}
	identity := application.Identity{NamespaceID: outcome.Scope().NamespaceID, PrincipalID: "human-1", Actor: commandContext().Actor}
	ctx := application.WithIdentity(context.Background(), identity)
	createObjective := func(title string) domain.Objective {
		t.Helper()
		created, createErr := service.CreateObjective(ctx, commandContext(), application.CreateObjectiveCommand{Scope: outcome.Scope(), Title: title, Priority: domain.PriorityNormal})
		if createErr != nil {
			t.Fatal(createErr)
		}
		return created.Value
	}
	source := createObjective("SQLite dependency source")
	second := createObjective("SQLite denied source")
	target := createObjective("SQLite dependency target")
	before, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	denied := second.Ref()
	policy.reset(&denied)
	cmd := application.ApplyDependencyChangesCommand{
		Scope: outcome.Scope(), ExpectedOutcomeRevision: before.OutcomeRevision,
		ExpectedSources: []application.DependencySourceVersion{
			{SourceRef: source.Ref(), ExpectedVersion: source.Version},
			{SourceRef: second.Ref(), ExpectedVersion: second.Version},
		},
		Changes: []application.DependencyChange{
			{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(source.Ref()), TargetRef: dependencyRefPointer(target.Ref()), Strength: domain.DependencyStrengthAdvisory},
			{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(second.Ref()), TargetRef: dependencyRefPointer(target.Ref()), Strength: domain.DependencyStrengthAdvisory},
		},
	}
	_, err = service.ApplyDependencyChanges(ctx, commandContext(), cmd)
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeForbidden {
		t.Fatalf("SQLite source authorization error = %v, want forbidden", err)
	}
	after, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if after.OutcomeRevision != before.OutcomeRevision || len(after.Relations) != len(before.Relations) {
		t.Fatalf("SQLite denial partially committed the dependency batch: revision %d -> %d, relations %d -> %d", before.OutcomeRevision, after.OutcomeRevision, len(before.Relations), len(after.Relations))
	}
}

func TestDependencyBatchSQLiteSerializesStaleConcurrentEdit(t *testing.T) {
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "dependency-source-race.db"), sqlite.Options{MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ids := &sequenceIDs{}
	clock := fixedClock{now: time.Date(2026, 10, 21, 21, 0, 0, 0, time.UTC)}
	trusted, err := application.NewService(store, clock, ids)
	if err != nil {
		t.Fatal(err)
	}
	outcome := setupActiveOutcome(t, trusted)
	policy := &dependencySourcePolicy{}
	service, err := application.NewAuthorizedService(store, clock, ids, policy)
	if err != nil {
		t.Fatal(err)
	}
	identity := application.Identity{NamespaceID: outcome.Scope().NamespaceID, PrincipalID: "human-1", Actor: commandContext().Actor}
	ctx := application.WithIdentity(context.Background(), identity)
	createObjective := func(title string) domain.Objective {
		t.Helper()
		created, createErr := service.CreateObjective(ctx, commandContext(), application.CreateObjectiveCommand{Scope: outcome.Scope(), Title: title, Priority: domain.PriorityNormal})
		if createErr != nil {
			t.Fatal(createErr)
		}
		return created.Value
	}
	source := createObjective("Concurrent dependency source")
	firstTarget := createObjective("First concurrent target")
	secondTarget := createObjective("Second concurrent target")
	before, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	cmdFor := func(target domain.Objective) application.ApplyDependencyChangesCommand {
		return application.ApplyDependencyChangesCommand{
			Scope: outcome.Scope(), ExpectedOutcomeRevision: before.OutcomeRevision,
			ExpectedSources: []application.DependencySourceVersion{{SourceRef: source.Ref(), ExpectedVersion: source.Version}},
			Changes:         []application.DependencyChange{{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(source.Ref()), TargetRef: dependencyRefPointer(target.Ref()), Strength: domain.DependencyStrengthAdvisory}},
		}
	}
	firstCmd, secondCmd := cmdFor(firstTarget), cmdFor(secondTarget)
	firstContext, secondContext := commandContext(), commandContext()
	firstContext.CommandID = id("0199e2ff-0000-7000-8000-000000000201")
	secondContext.CommandID = id("0199e2ff-0000-7000-8000-000000000202")
	firstAuthorized := make(chan struct{}, 2)
	resourceReached := make(chan struct{})
	releaseResource := make(chan struct{})
	var firstResource sync.Once
	var releaseOnce sync.Once
	policy.mu.Lock()
	policy.commandHook = func() { firstAuthorized <- struct{}{} }
	policy.resourceAuthHook = func(domain.EntityRef) {
		firstResource.Do(func() {
			close(resourceReached)
			<-releaseResource
		})
	}
	policy.mu.Unlock()
	release := func() { releaseOnce.Do(func() { close(releaseResource) }) }
	defer release()
	firstDone := make(chan dependencyBatchTestResult, 1)
	go func() {
		mutation, applyErr := service.ApplyDependencyChanges(ctx, firstContext, firstCmd)
		firstDone <- dependencyBatchTestResult{mutation: mutation, err: applyErr}
	}()
	waitForDependencySignal(t, firstAuthorized, "first command authorization")
	waitForDependencySignal(t, resourceReached, "first source authorization")
	secondDone := make(chan dependencyBatchTestResult, 1)
	go func() {
		mutation, applyErr := service.ApplyDependencyChanges(ctx, secondContext, secondCmd)
		secondDone <- dependencyBatchTestResult{mutation: mutation, err: applyErr}
	}()
	waitForDependencySignal(t, firstAuthorized, "second command authorization")
	release()
	firstResult := waitForDependencyResult(t, firstDone, "first dependency batch")
	secondResult := waitForDependencyResult(t, secondDone, "second dependency batch")
	if firstResult.err != nil || firstResult.mutation.OutcomeRevision != before.OutcomeRevision+1 {
		t.Fatalf("first serialized batch = revision %d, err=%v", firstResult.mutation.OutcomeRevision, firstResult.err)
	}
	if code, _ := domain.ErrorCodeOf(secondResult.err); code != domain.ErrorCodePreconditionFailed {
		t.Fatalf("stale concurrent batch error = %v, want precondition_failed", secondResult.err)
	}
	after, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if after.OutcomeRevision != before.OutcomeRevision+1 || len(after.Relations) != len(before.Relations)+1 {
		t.Fatalf("serialized SQLite graph has revision/relations %d/%d, want one committed edge", after.OutcomeRevision, len(after.Relations))
	}
}

func waitForDependencySignal(t *testing.T, ch <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", label)
	}
}

func waitForDependencyResult(t *testing.T, ch <-chan dependencyBatchTestResult, label string) dependencyBatchTestResult {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", label)
		return dependencyBatchTestResult{}
	}
}
