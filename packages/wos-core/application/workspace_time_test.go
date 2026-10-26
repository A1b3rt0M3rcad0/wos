package application

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

type workspaceBoundaryClock struct {
	now   time.Time
	reads int
}

func (c *workspaceBoundaryClock) Now() time.Time { c.reads++; return c.now }

// Expiry can pass while a capability query is being projected. Its predicates
// must use the frame's existing time; a later actual command must check anew.
func TestWorkspaceCapabilitiesPinContractExpiryWhileActualMutationsRecheckTime(t *testing.T) {
	for _, backend := range []string{"memory", "sqlite", "postgres"} {
		t.Run(backend, func(t *testing.T) {
			ctx := context.Background()
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			var store ports.TransactionManager
			switch backend {
			case "memory":
				store = memory.New()
			case "sqlite":
				sql, err := sqlite.Open(filepath.Join(t.TempDir(), "workspace-time.db"), sqlite.Options{MigrateOnOpen: true})
				must(err)
				t.Cleanup(func() { sql.Close() })
				store = sql
			case "postgres":
				dsn := os.Getenv("WOS_TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("real PostgreSQL required; CI supplies WOS_TEST_POSTGRES_DSN")
				}
				schema := fmt.Sprintf("workspace_time_%x", sha256.Sum256([]byte(t.TempDir())))[:48]
				sql, err := postgres.Open(dsn, postgres.Options{Schema: schema, MigrateOnOpen: true})
				must(err)
				t.Cleanup(func() { sql.Close() })
				store = sql
			}
			now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
			clock := &workspaceBoundaryClock{now: now}
			ids := &workspaceSnapshotIDs{}
			id := func() d.ID { v, err := ids.NewID(); must(err); return v }
			service, err := NewService(store, clock, ids)
			must(err)
			outcome, err := d.NewOutcome(id(), id(), "Pinned capability time", "", "One time per query", d.PriorityNormal, now)
			must(err)
			outcome.Lifecycle = d.OutcomeLifecycleActive
			scope := outcome.Scope()
			objective, err := d.NewObjective(id(), scope, "Reserved objective definition", "", d.PriorityNormal, false, now)
			must(err)
			work, err := d.NewWorkItem(id(), scope, "Signed execution freezes the definitions", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
			must(err)
			work.ObjectiveID, work.ContractsEnabled = &objective.ID, true
			contract, err := d.NewWorkContract(id(), id(), work, "executor", d.ActorRef{Kind: d.ActorKindAgent, Provider: "fixture", ID: "executor"}, d.DefaultContractLeasePolicy(), 30, d.WorkContractSpec{Title: work.Title}, 0, now)
			must(err)
			key := id()
			contract.SignedBinding = &d.SignedContractBinding{ProtocolVersion: 2, ServerID: "expiry-fixture", CredentialID: id(), SignerKeyID: key, PolicyRevision: 1, AcceptanceFloor: d.AcceptanceDirect, AllowedSigningKeyIDs: []d.ID{key}}
			must(work.BindContract(contract, now))
			target, err := d.NewObjective(id(), scope, "Dependency target", "", d.PriorityNormal, false, now)
			must(err)
			relation, err := d.NewDependencyRelation(id(), objective.Ref(), target.Ref(), d.DependencyStrengthAdvisory, now)
			must(err)
			u, err := store.Begin(ctx)
			must(err)
			_, err = u.Coordination().LockOutcome(ctx, scope)
			must(err)
			must(u.Outcomes().Insert(ctx, outcome))
			must(u.Objectives().Insert(ctx, objective))
			must(u.Objectives().Insert(ctx, target))
			must(u.WorkItems().Insert(ctx, work))
			must(u.Relations().Insert(ctx, relation))
			must(u.(ports.SignedContractUnitOfWork).SignedWorkContracts().Insert(ctx, contract))
			must(u.Commit())
			frame, handled, err := service.openWorkspaceFrame(ctx, scope)
			must(err)
			if !handled || frame == nil {
				t.Fatal("expected workspace frame")
			}
			defer frame.u.Rollback()
			if !frame.at.Before(contract.ExpiresAt) {
				t.Fatal("fixture did not pin before expiry")
			}
			clock.now = contract.ExpiresAt
			reads := clock.reads
			for _, selected := range []struct {
				ref     d.EntityRef
				value   any
				version d.Version
				actions []string
			}{
				{objective.Ref(), objective, objective.Version, []string{"UpdateObjective", "AddCriterion"}},
				{outcome.Ref(), outcome, outcome.Version, []string{"UpdateOutcome", "AddCriterion"}},
				{relation.Ref(), relation, relation.Version, []string{"RemoveDependency"}},
			} {
				view := EntityWorkspaceView{Subject: WorkspaceSubject{Kind: "entity", Ref: selected.ref}, EntityVersion: selected.version, OutcomeRevision: frame.revision, EvaluatedAt: frame.at}
				capabilities, err := service.workspaceCapabilities(ctx, frame, &view, selected.value, nil, outcome)
				must(err)
				for _, action := range selected.actions {
					found := false
					for _, capability := range capabilities {
						if capability.ActionID != action {
							continue
						}
						found = true
						if capability.Availability != "blocked" || len(capability.ReasonCodes) != 1 || capability.ReasonCodes[0] != "definition_frozen" {
							t.Fatalf("%s crossed the frame's expiry frontier: %+v", action, capability)
						}
					}
					if !found {
						t.Fatalf("missing %s capability", action)
					}
				}
			}
			if clock.reads != reads {
				t.Fatalf("capability projection reread time %d times", clock.reads-reads)
			}
			must(frame.u.Rollback())
			command := func() d.CommandContext {
				return d.CommandContext{CommandID: id(), PrincipalID: "owner", Actor: d.ActorRef{Kind: d.ActorKindHuman, Provider: "fixture", ID: "owner"}}
			}
			title := "Explicitly edited after expiry"
			_, err = service.UpdateOutcome(ctx, command(), UpdateOutcomeCommand{Scope: scope, ExpectedVersion: outcome.Version, Title: &title})
			must(err)
			_, err = service.UpdateObjective(ctx, command(), UpdateObjectiveCommand{Scope: scope, ObjectiveID: objective.ID, ExpectedVersion: objective.Version, Title: &title})
			must(err)
			_, err = service.RemoveDependency(ctx, command(), RemoveDependencyCommand{Scope: scope, RelationID: relation.ID, ExpectedVersion: relation.Version, Reason: "Explicitly remove after authority expires"})
			must(err)
			if clock.reads <= reads {
				t.Fatal("actual commands reused the read frame's old time")
			}
		})
	}
}
