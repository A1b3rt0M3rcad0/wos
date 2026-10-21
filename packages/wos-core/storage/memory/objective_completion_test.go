package memory

import (
	"context"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestMemoryObjectiveCompletionAuthorityIsVersionedCloneSafeAndTransactional(t *testing.T) {
	ctx := context.Background()
	store := New()
	now := time.Date(2026, 10, 21, 18, 0, 0, 0, time.UTC)
	scope := d.Scope{NamespaceID: d.MustParseID("019afa10-0000-7000-8000-000000000001"), OutcomeID: d.MustParseID("019afa10-0000-7000-8000-000000000002")}
	objectiveID := d.MustParseID("019afa10-0000-7000-8000-000000000003")
	principal := "completion-policy-operator"
	actor := d.ActorRef{Kind: d.ActorKindHuman, Provider: "memory-test", ID: principal}
	outcome, err := d.NewOutcome(scope.OutcomeID, scope.NamespaceID, "Completion authority", "", "Persist explicit completion authority", d.PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	objective, err := d.NewObjective(objectiveID, scope, "Persist authority", "", d.PriorityNormal, false, now)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.Outcomes().Insert(ctx, outcome); err != nil {
		t.Fatal(err)
	}
	if err := seed.Objectives().Insert(ctx, objective); err != nil {
		t.Fatal(err)
	}
	if err := seed.Commit(); err != nil {
		t.Fatal(err)
	}
	makeAuthority := func(id string, version uint64) d.ObjectiveCompletionAuthority {
		return d.ObjectiveCompletionAuthority{
			ID: d.MustParseID(id), Scope: scope, ObjectiveID: objectiveID,
			PolicyVersion: version, Mode: d.ObjectiveCompletionModeManual, CompletionCycle: 1,
			DelegatorPrincipalID: principal, DelegatorActor: actor,
			OperationalProfile: d.OperationalProfileCompatibilityV1, CompletionPolicy: d.CompletionPolicyCompatible,
			AutomationActor: d.ObjectiveCompletionAutomationActor(), Reason: "Record explicit completion policy authorization",
			AuthorizedAt: now.Add(time.Minute),
		}
	}

	first := makeAuthority("019afa10-0000-7000-8000-000000000010", 1)
	write, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := write.(ports.ObjectiveCompletionAuthorityUnitOfWork).ObjectiveCompletionAuthorities().Append(ctx, first); err != nil {
		t.Fatalf("append first authority: %v", err)
	}
	if err := write.Commit(); err != nil {
		t.Fatal(err)
	}

	// The repository owns pointer-valued fields rather than retaining caller memory.
	credentialID := d.MustParseID("019afa10-0000-7000-8000-000000000011")
	first.DelegatorCredentialID = &credentialID
	read, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	history, err := read.(ports.ObjectiveCompletionAuthorityUnitOfWork).ObjectiveCompletionAuthorities().ListByObjective(ctx, scope, objectiveID)
	if err != nil || len(history) != 1 || history[0].DelegatorCredentialID != nil {
		t.Fatalf("caller pointer mutated stored authority: %+v err=%v", history, err)
	}
	if err := read.Rollback(); err != nil {
		t.Fatal(err)
	}

	badVersion := makeAuthority("019afa10-0000-7000-8000-000000000012", 3)
	invalid, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := invalid.(ports.ObjectiveCompletionAuthorityUnitOfWork).ObjectiveCompletionAuthorities().Append(ctx, badVersion); err == nil {
		t.Fatal("non-contiguous policy history version was accepted")
	}
	_ = invalid.Rollback()

	second := makeAuthority("019afa10-0000-7000-8000-000000000013", 2)
	rolledBack, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := rolledBack.(ports.ObjectiveCompletionAuthorityUnitOfWork).ObjectiveCompletionAuthorities().Append(ctx, second); err != nil {
		t.Fatal(err)
	}
	if err := rolledBack.Rollback(); err != nil {
		t.Fatal(err)
	}
	final, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer final.Rollback()
	history, err = final.(ports.ObjectiveCompletionAuthorityUnitOfWork).ObjectiveCompletionAuthorities().ListByObjective(ctx, scope, objectiveID)
	if err != nil || len(history) != 1 || history[0].PolicyVersion != 1 {
		t.Fatalf("rolled-back authority persisted: %+v err=%v", history, err)
	}
}
