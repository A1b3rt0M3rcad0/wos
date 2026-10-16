package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func activityStorageID(n int) d.ID {
	return d.MustParseID(fmt.Sprintf("019a0001-0000-7000-8000-%012x", n))
}

func TestWorkActivityStorageChronologyImmutabilityScopeAndRestore(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			ctx := context.Background()
			now := time.Date(2026, 10, 16, 15, 0, 0, 0, time.UTC)
			path := filepath.Join(t.TempDir(), "activity.db")
			var store ports.TransactionManager
			var sqlStore *Store
			if backend == "memory" {
				store = memory.New()
			} else {
				sqlStore = openTestStore(t, path)
				store = sqlStore
			}
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			begin := func() ports.UnitOfWork {
				t.Helper()
				tx, err := store.Begin(ctx)
				must(err)
				return tx
			}
			list := func(scope d.Scope, filter ports.WorkActivityFilter) []d.WorkActivity {
				t.Helper()
				tx := begin()
				defer tx.Rollback()
				got, err := tx.(ports.WorkActivityUnitOfWork).WorkActivities().List(ctx, scope, filter)
				must(err)
				return got
			}
			scope := d.Scope{NamespaceID: activityStorageID(1), OutcomeID: activityStorageID(2)}
			otherOutcome := d.Scope{NamespaceID: scope.NamespaceID, OutcomeID: activityStorageID(3)}
			otherNamespace := d.Scope{NamespaceID: activityStorageID(4), OutcomeID: activityStorageID(5)}
			tx := begin()
			for _, boundary := range []d.Scope{scope, otherOutcome, otherNamespace} {
				outcome, err := d.NewOutcome(boundary.OutcomeID, boundary.NamespaceID, "Activity history", "", "Auditable delivery", d.PriorityNormal, now)
				must(err)
				must(tx.Outcomes().Insert(ctx, outcome))
			}
			var contracts []d.WorkContract
			for i, boundary := range []d.Scope{scope, scope, otherOutcome, otherNamespace} {
				work, err := d.NewWorkItem(activityStorageID(10+i), boundary, "External implementation", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
				must(err)
				work.ContractsEnabled = true
				must(tx.WorkItems().Insert(ctx, work))
				contract, err := d.NewWorkContract(activityStorageID(20+i), activityStorageID(30+i), work, "activity-executor", d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}, d.DefaultContractLeasePolicy(), 30, d.WorkContractSpec{Title: work.Title}, 0, now)
				must(err)
				keyID := activityStorageID(40)
				contract.SignedBinding = &d.SignedContractBinding{ProtocolVersion: 2, ServerID: "activity-server", CredentialID: activityStorageID(41), SignerKeyID: keyID, PolicyRevision: 1, AcceptanceFloor: d.AcceptanceIndependentReview, AllowedSigningKeyIDs: []d.ID{keyID}}
				must(tx.(ports.SignedContractUnitOfWork).SignedWorkContracts().Insert(ctx, contract))
				contracts = append(contracts, contract)
			}
			unsigned := contracts[0]
			unsigned.ID, unsigned.ExecutionID, unsigned.SignedBinding = activityStorageID(25), activityStorageID(35), nil
			must(tx.(ports.WorkContractUnitOfWork).WorkContracts().Insert(ctx, unsigned))
			must(tx.Commit())
			newActivity := func(id int, contract d.WorkContract, sequence uint64) d.WorkActivity {
				return d.WorkActivity{ID: activityStorageID(id), Scope: contract.Scope, ContractID: contract.ID, WorkItemID: contract.WorkItemID, Sequence: sequence, Type: d.WorkActivityProgress, Description: "Implementation progress recorded externally", PrincipalID: contract.HolderPrincipalID, Actor: contract.Actor, RecordedAt: now.Add(-time.Duration(id) * time.Second)}
			}
			first := newActivity(101, contracts[0], 2)
			first.ArtifactRefs = []d.ID{activityStorageID(301)}
			first.ArtifactVersions = []d.SubmissionArtifact{{ArtifactID: first.ArtifactRefs[0], SourceVersion: "commit-original", Checksum: "sha256:original"}}
			first.EvidenceRefs = []d.ID{activityStorageID(302)}
			first.CriterionRefs = []d.WorkActivityCriterionRef{{CriterionID: activityStorageID(303), CriterionRevision: 1}}
			expected := first
			// Copies below deliberately keep caller-owned slices, which storage must detach.
			tx = begin()
			repo := tx.(ports.WorkActivityUnitOfWork).WorkActivities()
			must(repo.Insert(ctx, first))
			first.ArtifactRefs[0] = activityStorageID(399)
			first.ArtifactVersions[0].SourceVersion = "caller-mutated"
			first.EvidenceRefs[0] = activityStorageID(398)
			first.CriterionRefs[0].CriterionRevision = 9
			for _, activity := range []d.WorkActivity{
				newActivity(102, contracts[0], 10),
				newActivity(103, contracts[0], 9007199254740993),
				newActivity(104, contracts[0], uint64(1)<<63+3),
				newActivity(105, contracts[0], math.MaxUint64),
				newActivity(106, contracts[1], 3),
				newActivity(107, contracts[2], 2),
				newActivity(108, contracts[3], 2),
			} {
				must(repo.Insert(ctx, activity))
			}
			must(tx.Commit())
			if sqlStore != nil {
				sqlStore = reopenIntegrationFixture(t, sqlStore, path)
				store = sqlStore
			}
			all := list(scope, ports.WorkActivityFilter{ThroughSequence: math.MaxUint64, Limit: 101})
			var sequences []uint64
			for _, activity := range all {
				sequences = append(sequences, activity.Sequence)
			}
			if !reflect.DeepEqual(sequences, []uint64{2, 3, 10, 9007199254740993, uint64(1)<<63 + 3, math.MaxUint64}) {
				t.Fatalf("numeric chronological order/precision lost: %v", sequences)
			}
			if all[0].ArtifactRefs[0] != activityStorageID(301) || all[0].ArtifactVersions[0].SourceVersion != "commit-original" || all[0].EvidenceRefs[0] != activityStorageID(302) || all[0].CriterionRefs[0].CriterionRevision != 1 {
				t.Fatal("insert retained caller-owned references")
			}
			expected.ArtifactRefs = []d.ID{activityStorageID(301)}
			expected.ArtifactVersions = []d.SubmissionArtifact{{ArtifactID: activityStorageID(301), SourceVersion: "commit-original", Checksum: "sha256:original"}}
			expected.EvidenceRefs = []d.ID{activityStorageID(302)}
			expected.CriterionRefs = []d.WorkActivityCriterionRef{{CriterionID: activityStorageID(303), CriterionRevision: 1}}
			if !reflect.DeepEqual(all[0], expected) {
				t.Fatalf("activity changed through restore: %#v", all[0])
			}
			all[0].Description = "read mutation"
			all[0].ArtifactRefs[0] = activityStorageID(397)
			all[0].ArtifactVersions[0].SourceVersion = "read mutation"
			all[0].EvidenceRefs[0] = activityStorageID(396)
			all[0].CriterionRefs[0].CriterionRevision = 8
			if got := list(scope, ports.WorkActivityFilter{ThroughSequence: 2, Limit: 1}); !reflect.DeepEqual(got[0], expected) {
				t.Fatal("list returned mutable references to persisted history")
			}
			// A fixed upper bound excludes later appends, even across page reads.
			page1 := list(scope, ports.WorkActivityFilter{ThroughSequence: 10, Limit: 2})
			page2 := list(scope, ports.WorkActivityFilter{AfterSequence: page1[1].Sequence, ThroughSequence: 10, Limit: 2})
			if len(page1) != 2 || len(page2) != 1 || page2[0].Sequence != 10 {
				t.Fatal("frozen traversal omitted or repeated activity")
			}
			// Newest-first traversal uses zero only as the initial cursor, and
			// strict numeric '<' thereafter without decrementing uint64 values.
			latest := list(scope, ports.WorkActivityFilter{ThroughSequence: math.MaxUint64, Descending: true, Limit: 1})
			if len(latest) != 1 || latest[0].Sequence != math.MaxUint64 {
				t.Fatal("latest activity lost the unsigned maximum or zero cursor sentinel")
			}
			older := list(scope, ports.WorkActivityFilter{AfterSequence: latest[0].Sequence, ThroughSequence: math.MaxUint64, Descending: true, Limit: 2})
			if len(older) != 2 || older[0].Sequence != uint64(1)<<63+3 || older[1].Sequence != 9007199254740993 {
				t.Fatal("descending continuation lost numeric order/precision")
			}
			newestPage := list(scope, ports.WorkActivityFilter{ThroughSequence: 10, Descending: true, Limit: 2})
			if len(newestPage) != 2 || newestPage[0].Sequence != 10 || newestPage[1].Sequence != 3 {
				t.Fatal("newest-first page did not honor its frozen upper bound")
			}
			tx = begin()
			must(tx.(ports.WorkActivityUnitOfWork).WorkActivities().Insert(ctx, newActivity(109, contracts[0], 11)))
			must(tx.Commit())
			olderPage := list(scope, ports.WorkActivityFilter{AfterSequence: newestPage[1].Sequence, ThroughSequence: 10, Descending: true, Limit: 2})
			if len(olderPage) != 1 || olderPage[0].Sequence != 2 {
				t.Fatal("concurrent append above frozen history changed the next older page")
			}
			if got := list(scope, ports.WorkActivityFilter{AfterSequence: 1, ThroughSequence: math.MaxUint64, Descending: true, Limit: 1}); len(got) != 0 {
				t.Fatal("descending empty range wrapped around to newest records")
			}
			for _, filter := range []ports.WorkActivityFilter{
				{WorkItemID: contracts[1].WorkItemID, ThroughSequence: 10, Limit: 101},
				{ContractID: contracts[1].ID, ThroughSequence: 10, Limit: 101},
				{WorkItemID: contracts[1].WorkItemID, ContractID: contracts[1].ID, ThroughSequence: 10, Limit: 101},
			} {
				got := list(scope, filter)
				if len(got) != 1 || got[0].ID != activityStorageID(106) {
					t.Fatalf("task/contract scope filter failed: %#v", got)
				}
			}
			for _, boundary := range []d.Scope{otherOutcome, otherNamespace} {
				if got := list(boundary, ports.WorkActivityFilter{ThroughSequence: 10, Limit: 101}); len(got) != 1 || got[0].Scope != boundary {
					t.Fatal("activity crossed Outcome/Namespace boundaries")
				}
			}
			for name, invalid := range map[string]ports.WorkActivityFilter{
				"zero limit":          {ThroughSequence: 10},
				"large limit":         {ThroughSequence: 10, Limit: 102},
				"missing upper bound": {Limit: 1},
				"inverted bounds":     {AfterSequence: 11, ThroughSequence: 10, Limit: 1},
			} {
				tx = begin()
				_, err := tx.(ports.WorkActivityUnitOfWork).WorkActivities().List(ctx, scope, invalid)
				must(tx.Rollback())
				if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeInvalidArgument {
					t.Fatalf("%s: %v", name, err)
				}
			}
			if got := list(scope, ports.WorkActivityFilter{AfterSequence: 10, ThroughSequence: 10, Limit: 1}); len(got) != 0 {
				t.Fatal("exhausted range did not return an empty list")
			}
			// Every failed write gets a separate transaction: PostgreSQL correctly
			// aborts a transaction after a database constraint violation.
			for name, mutate := range map[string]func(*d.WorkActivity){
				"identical repeat":      func(a *d.WorkActivity) { *a = expected },
				"duplicate identity":    func(a *d.WorkActivity) { a.ID = expected.ID; a.Description = "rewrite" },
				"global identity":       func(a *d.WorkActivity) { *a = newActivity(101, contracts[2], 4) },
				"duplicate sequence":    func(a *d.WorkActivity) { a.Sequence = expected.Sequence },
				"sequence across tasks": func(a *d.WorkActivity) { *a = newActivity(201, contracts[1], expected.Sequence) },
				"unsigned contract":     func(a *d.WorkActivity) { a.ContractID = unsigned.ID },
				"wrong task":            func(a *d.WorkActivity) { a.WorkItemID = contracts[1].WorkItemID },
				"wrong Outcome":         func(a *d.WorkActivity) { a.Scope = otherOutcome },
				"wrong Namespace":       func(a *d.WorkActivity) { a.Scope = otherNamespace },
			} {
				activity := newActivity(201, contracts[0], 4)
				mutate(&activity)
				tx = begin()
				err := tx.(ports.WorkActivityUnitOfWork).WorkActivities().Insert(ctx, activity)
				must(tx.Rollback())
				expectedCode := d.ErrorCodeAlreadyExists
				switch name {
				case "wrong task":
					expectedCode = d.ErrorCodeInvalidScope
				case "wrong Outcome", "wrong Namespace", "unsigned contract":
					expectedCode = d.ErrorCodeNotFound
				}
				if code, _ := d.ErrorCodeOf(err); code != expectedCode {
					t.Fatalf("%s: got %v, want %v", name, err, expectedCode)
				}
			}
			if sqlStore != nil {
				// Exercise the relational composite FK directly as well as the
				// repository precheck: scope and task must belong to that exact
				// signed contract even for another database writer.
				for name, mutate := range map[string]func(*d.WorkActivity){
					"task":              func(a *d.WorkActivity) { a.WorkItemID = contracts[1].WorkItemID },
					"Outcome":           func(a *d.WorkActivity) { a.Scope = otherOutcome },
					"Namespace":         func(a *d.WorkActivity) { a.Scope = otherNamespace },
					"unsigned contract": func(a *d.WorkActivity) { a.ContractID = unsigned.ID },
				} {
					activity := newActivity(201, contracts[0], 4)
					mutate(&activity)
					raw, err := json.Marshal(activity)
					must(err)
					tx = begin()
					_, err = tx.(*unitOfWork).tx.ExecContext(ctx, `INSERT INTO work_activities (id,namespace_id,outcome_id,contract_id,work_item_id,sequence,payload_json) VALUES (?,?,?,?,?,?,?)`, activity.ID.String(), activity.Scope.NamespaceID.String(), activity.Scope.OutcomeID.String(), activity.ContractID.String(), activity.WorkItemID.String(), activitySequenceKey(activity.Sequence), string(raw))
					must(tx.Rollback())
					if code, _ := d.ErrorCodeOf(mapSQLError("foreign-key test", err)); code != d.ErrorCodeInvalidArgument {
						t.Fatalf("database accepted wrong %s binding: %v", name, err)
					}
				}
			}
			tx = begin()
			repo = tx.(ports.WorkActivityUnitOfWork).WorkActivities()
			must(repo.Insert(ctx, newActivity(202, contracts[0], 4)))
			must(tx.Rollback())
			if got := list(scope, ports.WorkActivityFilter{ThroughSequence: 10, Limit: 101}); len(got) != 3 || !reflect.DeepEqual(got[0], expected) {
				t.Fatal("rollback or duplicate write changed immutable history")
			}
			tx = begin()
			repo = tx.(ports.WorkActivityUnitOfWork).WorkActivities()
			must(tx.Rollback())
			if err := repo.Insert(ctx, newActivity(203, contracts[0], 4)); err == nil {
				t.Fatal("closed transaction accepted an activity")
			}
		})
	}
}
