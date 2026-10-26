package application

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
)

type workspaceSnapshotIDs struct{ sequence atomic.Uint64 }

func (g *workspaceSnapshotIDs) NewID() (d.ID, error) {
	return d.ParseID(fmt.Sprintf("019a39b0-0000-7000-8000-%012x", g.sequence.Add(1)))
}

type workspaceSnapshotClock struct{ at time.Time }

func (c workspaceSnapshotClock) Now() time.Time { return c.at }

// PostgreSQL can keep a repeatable read snapshot while another connection
// changes the same Outcome. SQLite's intentionally serialized connection is a
// different deployment contract, so this concurrency case requires real PG.
func TestPostgresWorkspaceFrameKeepsRevisionFactsAndCapabilitiesWhileWriterCommits(t *testing.T) {
	dsn := os.Getenv("WOS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL required: set WOS_TEST_POSTGRES_DSN; CI supplies it")
	}
	schema := fmt.Sprintf("workspace_snapshot_%x", sha256.Sum256([]byte(t.TempDir())))[:48]
	store, err := postgres.Open(dsn, postgres.Options{Schema: schema, MigrateOnOpen: true, MaxOpenConnections: 4, BusyTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	clock := workspaceSnapshotClock{time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)}
	ids := &workspaceSnapshotIDs{}
	service, err := NewService(store, clock, ids)
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	command := func() d.CommandContext {
		id, err := ids.NewID()
		must(err)
		return d.CommandContext{CommandID: id, PrincipalID: "snapshot-owner", Actor: d.ActorRef{Kind: d.ActorKindHuman, Provider: "fixture", ID: "snapshot-owner"}}
	}
	namespace, err := ids.NewID()
	must(err)
	outcome, err := service.CreateOutcome(ctx, command(), CreateOutcomeCommand{NamespaceID: namespace, Title: "Concurrent workspace reads", DesiredState: "A reader keeps a coherent snapshot without holding up writers", Priority: d.PriorityNormal})
	must(err)
	scope := outcome.Value.Scope()
	work, err := service.CreateWorkItem(ctx, command(), CreateWorkItemCommand{Scope: scope, ObjectiveID: objectiveForWorkFixture(t, service, scope), Title: "Before concurrent edit", Priority: d.PriorityNormal, Lifecycle: d.WorkItemLifecycleBacklog})
	must(err)
	frame, handled, err := service.openWorkspaceFrame(ctx, scope)
	must(err)
	if !handled || frame == nil {
		t.Fatal("workspace snapshot adapter was not used")
	}
	defer frame.u.Rollback()
	initialRevision := frame.revision
	initialTime := frame.at
	focal, ok := frame.repo.(ports.FocalReadRepository)
	if !ok {
		t.Fatal("current focal reads unavailable")
	}
	before, err := focal.CurrentWork(ctx, scope, work.Value.ID)
	must(err)
	if before.Work.Title != work.Value.Title {
		t.Fatal("initial frame did not read the stored Task")
	}
	type writeResult struct {
		mutation MutationResult[d.WorkItem]
		err      error
	}
	written := make(chan writeResult, 1)
	writeContext, stopWrite := context.WithTimeout(ctx, 5*time.Second)
	defer stopWrite()
	writeCommand := command()
	newTitle := "Committed while the old workspace remains open"
	go func() {
		result, err := service.UpdateWorkItem(writeContext, writeCommand, UpdateWorkItemCommand{Scope: scope, WorkItemID: work.Value.ID, ExpectedVersion: work.Value.Version, Title: &newTitle})
		written <- writeResult{result, err}
	}()
	var result writeResult
	select {
	case result = <-written:
		if result.err != nil {
			t.Fatalf("an open query frame blocked the real writer: %v", result.err)
		}
	case <-writeContext.Done():
		t.Fatal("the writer could not commit while the read frame remained open")
	}
	if result.mutation.OutcomeRevision <= initialRevision {
		t.Fatal("writer did not advance the Outcome revision")
	}
	pinned, err := service.currentFocalWorkContextAt(ctx, frame.u, focal, scope, work.Value.ID, frame.revision, frame.at)
	must(err)
	if pinned.Work.Title != work.Value.Title || pinned.Work.Version != work.Value.Version || pinned.OutcomeRevision != initialRevision || !pinned.EvaluatedAt.Equal(initialTime) {
		t.Fatalf("facts crossed the snapshot frontier: %+v", pinned)
	}
	repo := frame.repo.(ports.WorkspaceReadRepository)
	stillPinned, err := repo.WorkspaceRevision(ctx, scope)
	must(err)
	if stillPinned != initialRevision {
		t.Fatalf("revision changed inside the old frame: %d -> %d", initialRevision, stillPinned)
	}
	parent, err := focal.FocalOutcome(ctx, scope)
	must(err)
	view := EntityWorkspaceView{Subject: WorkspaceSubject{Kind: "entity", Ref: work.Value.Ref()}, EntityVersion: pinned.Work.Version, OutcomeRevision: frame.revision, EvaluatedAt: frame.at}
	capabilities, err := service.workspaceCapabilities(ctx, frame, &view, pinned.Work, &pinned, parent)
	must(err)
	if len(capabilities) == 0 {
		t.Fatal("snapshot did not evaluate any capabilities")
	}
	for _, capability := range capabilities {
		if capability.ObservedVersion != work.Value.Version {
			t.Fatalf("capability was evaluated against a different entity version: %+v", capability)
		}
	}
	fresh, err := service.GetEntityWorkspaceView(ctx, scope, WorkspaceViewQuery{Subject: WorkspaceSubject{Kind: "entity", Ref: work.Value.Ref()}})
	must(err)
	if fresh.Header.Title != newTitle || fresh.EntityVersion != result.mutation.Value.Version || fresh.OutcomeRevision != result.mutation.OutcomeRevision {
		t.Fatalf("fresh frame did not observe the committed edit: %+v", fresh)
	}
}
