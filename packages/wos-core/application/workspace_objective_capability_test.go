package application

import (
	"context"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestWorkspaceObjectiveStartCapabilityTracksRealCommandGates(t *testing.T) {
	ctx := context.Background()
	ids := &workspaceSnapshotIDs{}
	service, err := NewService(memory.New(), workspaceSnapshotClock{time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)}, ids)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(err)
	id := func() d.ID { v, e := ids.NewID(); must(e); return v }
	cc := func() d.CommandContext {
		return d.CommandContext{CommandID: id(), PrincipalID: "owner", Actor: d.ActorRef{Kind: d.ActorKindHuman, Provider: "test", ID: "owner"}}
	}
	out, err := service.CreateOutcome(ctx, cc(), CreateOutcomeCommand{NamespaceID: id(), Title: "Operate legitimate work", DesiredState: "Explicit acceptance", Priority: d.PriorityNormal})
	must(err)
	scope := out.Value.Scope()
	obj, err := service.CreateObjective(ctx, cc(), CreateObjectiveCommand{Scope: scope, Title: "A condition to start", Priority: d.PriorityNormal})
	must(err)
	read := func() WorkspaceCapability {
		t.Helper()
		v, e := service.GetEntityWorkspaceView(ctx, scope, WorkspaceViewQuery{Subject: WorkspaceSubject{Kind: "entity", Ref: obj.Value.Ref()}})
		must(e)
		for _, c := range v.Capabilities {
			if c.ActionID == "StartObjective" {
				return c
			}
		}
		t.Fatal("start capability missing")
		return WorkspaceCapability{}
	}
	if read().Availability != "blocked" {
		t.Fatal("missing criterion must block start")
	}
	_, err = service.AddCriterion(ctx, cc(), AddCriterionCommand{Owner: obj.Value.Ref(), ExpectedVersion: obj.Value.Version, Title: "Agreed success", Required: true, VerificationMode: d.VerificationModeAttestation})
	must(err)
	if read().Availability != "blocked" {
		t.Fatal("inactive Outcome must block start")
	}
	_, err = service.AddCriterion(ctx, cc(), AddCriterionCommand{Owner: out.Value.Ref(), ExpectedVersion: out.Value.Version, Title: "An agreed result", Required: true, VerificationMode: d.VerificationModeAttestation})
	must(err)
	_, err = service.ActivateOutcome(ctx, cc(), ActivateOutcomeCommand{Scope: scope, ExpectedVersion: out.Value.Version + 1})
	must(err)
	cap := read()
	if cap.Availability != "available" {
		t.Fatalf("ready Objective blocked: %+v", cap)
	}
	_, err = service.CreateBlocker(ctx, cc(), CreateBlockerCommand{Scope: scope, BlockedRef: obj.Value.Ref(), Description: "Missing review access", Propagation: d.BlockerPropagationDirect})
	must(err)
	if read().Availability != "blocked" {
		t.Fatal("real blocker must prevent start")
	}
	_, err = service.StartObjective(ctx, cc(), StartObjectiveCommand{Scope: scope, ObjectiveID: obj.Value.ID, ExpectedVersion: obj.Value.Version + 1})
	if err == nil {
		t.Fatal("stale available capability bypassed current blocker")
	}
}

func TestWorkspaceLeaseRecoveryCapabilitiesKeepOwnershipAndExpiry(t *testing.T) {
	clock := &workspaceBoundaryClock{now: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)}
	ids := &workspaceSnapshotIDs{}
	service, err := NewService(memory.New(), clock, ids)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(err)
	id := func() d.ID { v, e := ids.NewID(); must(e); return v }
	actor := d.ActorRef{Kind: d.ActorKindHuman, Provider: "test", ID: "owner"}
	ctx := WithIdentity(context.Background(), Identity{PrincipalID: "owner", Actor: actor})
	cc := func() d.CommandContext { return d.CommandContext{CommandID: id(), PrincipalID: "owner", Actor: actor} }
	out, err := service.CreateOutcome(ctx, cc(), CreateOutcomeCommand{NamespaceID: id(), Title: "Recover work", DesiredState: "Fenced authority", Priority: d.PriorityNormal})
	must(err)
	scope := out.Value.Scope()
	_, err = service.AddCriterion(ctx, cc(), AddCriterionCommand{Owner: out.Value.Ref(), ExpectedVersion: out.Value.Version, Title: "Verified result", Required: true, VerificationMode: d.VerificationModeAttestation})
	must(err)
	_, err = service.ActivateOutcome(ctx, cc(), ActivateOutcomeCommand{Scope: scope, ExpectedVersion: out.Value.Version + 1})
	must(err)
	work, err := service.CreateWorkItem(ctx, cc(), CreateWorkItemCommand{Scope: scope, ObjectiveID: objectiveForWorkFixture(t, service, scope), Title: "Recover real reservation", Priority: d.PriorityNormal, Lifecycle: d.WorkItemLifecycleTodo})
	must(err)
	claimed, err := service.ClaimWorkItem(ctx, cc(), ClaimWorkItemCommand{Scope: scope, WorkItemID: work.Value.ID, ExpectedVersion: work.Value.Version, TTL: 30 * time.Second})
	must(err)
	read := func(context context.Context) map[string]WorkspaceCapability {
		t.Helper()
		v, e := service.GetEntityWorkspaceView(context, scope, WorkspaceViewQuery{Subject: WorkspaceSubject{Kind: "entity", Ref: work.Value.Ref()}})
		must(e)
		result := map[string]WorkspaceCapability{}
		for _, c := range v.Capabilities {
			result[c.ActionID] = c
		}
		return result
	}
	if read(ctx)["RenewWorkItemLease"].Availability != "available" {
		t.Fatal("holder cannot renew live reservation")
	}
	other := WithIdentity(context.Background(), Identity{PrincipalID: "other", Actor: d.ActorRef{Kind: d.ActorKindHuman, Provider: "test", ID: "other"}})
	if read(other)["RenewWorkItemLease"].Availability != "blocked" {
		t.Fatal("non-holder was offered renewal")
	}
	clock.now = clock.now.Add(31 * time.Second)
	cap := read(ctx)
	if cap["ReclaimWorkItem"].Availability != "available" {
		t.Fatalf("expired work cannot be reacquired: %+v", cap["ReclaimWorkItem"])
	}
	if _, ok := cap["RenewWorkItemLease"]; ok {
		t.Fatal("expired reservation offered renewal")
	}
	reclaimed, err := service.ReclaimWorkItem(ctx, cc(), ReclaimWorkItemCommand{Scope: scope, WorkItemID: work.Value.ID, ExpectedVersion: claimed.Value.Version, TTL: time.Minute})
	must(err)
	if reclaimed.Value.CurrentLease.FencingToken <= claimed.Value.CurrentLease.FencingToken {
		t.Fatal("recovery did not advance fencing")
	}
	if reclaimed.Value.CurrentLease.ClaimID == claimed.Value.CurrentLease.ClaimID {
		t.Fatal("recovery reused the old claim")
	}
}
