package sqlite

import (
	"context"
	"encoding/json"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

type advancingContinuityClock struct{ now time.Time }

func (c *advancingContinuityClock) Now() time.Time { return c.now }
func TestContinuityProjectionLeaseExpiryAndFrozenCursor(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "expiry.db")
			var store ports.TransactionManager = memory.New()
			if backend == "sql" {
				store = openTestStore(t, path)
			}
			clock := &advancingContinuityClock{time.Now().UTC()}
			ids := &sqliteSequenceIDs{prefix: "0199d211", next: 1}
			service, _ := a.NewService(store, clock, ids)
			ctx := context.Background()
			cc := sqliteCommandContext("0199d210-0000-7000-8000-000000000001", "")
			must := func(e error) {
				t.Helper()
				if e != nil {
					t.Fatal(e)
				}
			}
			out, e := service.CreateOutcome(ctx, cc, a.CreateOutcomeCommand{NamespaceID: testID("0199d210-0000-7000-8000-000000000002"), Title: "Projection expiry", DesiredState: "time changes presentation only", Priority: d.PriorityNormal})
			must(e)
			_, e = service.AddCriterion(ctx, cc, a.AddCriterionCommand{Owner: out.Value.Ref(), ExpectedVersion: 1, Title: "Configured completion contract", Required: true, VerificationMode: d.VerificationModeAttestation})
			must(e)
			_, e = service.ActivateOutcome(ctx, cc, a.ActivateOutcomeCommand{Scope: out.Value.Scope(), ExpectedVersion: 2})
			must(e)
			for i := 0; i < 2; i++ {
				w, e := service.CreateWorkItem(ctx, cc, a.CreateWorkItemCommand{Scope: out.Value.Scope(), ObjectiveID: sqliteObjectiveForWorkFixture(t, service, out.Value.Scope()), Title: "claimed", Priority: d.PriorityNormal, Lifecycle: d.WorkItemLifecycleBacklog})
				must(e)
				_, e = service.AddCriterion(ctx, cc, a.AddCriterionCommand{Owner: w.Value.Ref(), ExpectedVersion: 1, Title: "Lease completion contract", Required: true, VerificationMode: d.VerificationModeAttestation})
				must(e)
				_, e = service.ActivateWorkItem(ctx, cc, a.ActivateWorkItemCommand{Scope: out.Value.Scope(), WorkItemID: w.Value.ID, ExpectedVersion: 2})
				must(e)
				_, e = service.ClaimWorkItem(ctx, cc, a.ClaimWorkItemCommand{Scope: out.Value.Scope(), WorkItemID: w.Value.ID, ExpectedVersion: 3, TTL: 30 * time.Second})
				must(e)
			}
			first, e := service.GetContinuitySection(ctx, out.Value.Scope(), "in_progress_work", 1, "")
			must(e)
			if first.Count != 2 || first.NextCursor == "" {
				t.Fatal("active leases not projected", first)
			}
			if sqlStore, ok := store.(*Store); ok {
				store = reopenIntegrationFixture(t, sqlStore, path)
				service, _ = a.NewService(store, clock, ids)
			}
			clock.now = clock.now.Add(31 * time.Second)
			frozen, e := service.GetContinuitySection(ctx, out.Value.Scope(), "in_progress_work", 1, first.NextCursor)
			must(e)
			if frozen.Count != 2 || len(frozen.Items) != 1 || !frozen.EvaluatedAt.Equal(first.EvaluatedAt) || frozen.OutcomeRevision != first.OutcomeRevision {
				t.Fatal("cursor changed its evaluation snapshot", frozen)
			}
			fresh, e := service.GetContinuitySection(ctx, out.Value.Scope(), "attention_needed_work", 10, "")
			must(e)
			if fresh.Count != 2 || fresh.OutcomeRevision != first.OutcomeRevision {
				t.Fatal("expiry needed a write or lost attention state", fresh)
			}
			var item struct {
				State d.WorkItemOperationalState `json:"state"`
			}
			must(json.Unmarshal(fresh.Items[0], &item))
			if item.State.LeaseStatus != d.LeaseStatusExpired {
				t.Fatal("expired lease reported active")
			}
			oldGroup, e := service.GetContinuitySection(ctx, out.Value.Scope(), "in_progress_work", 10, "")
			must(e)
			if oldGroup.Count != 0 {
				t.Fatal("fresh projection kept expired leases")
			}
		})
	}
}
func TestContinuityProjectionRetainsConclusionEvidenceAndCurrentContradiction(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			var store ports.TransactionManager = memory.New()
			if backend == "sql" {
				store = openTestStore(t, filepath.Join(t.TempDir(), "proof.db"))
			}
			ctx := context.Background()
			clock := sqliteFixedClock{time.Now().UTC()}
			ids := &sqliteSequenceIDs{prefix: "0199d221", next: 1}
			service, _ := a.NewService(store, clock, ids)
			cc := sqliteCommandContext("0199d220-0000-7000-8000-000000000001", "")
			must := func(e error) {
				t.Helper()
				if e != nil {
					t.Fatal(e)
				}
			}
			out, e := service.CreateOutcome(ctx, cc, a.CreateOutcomeCommand{NamespaceID: testID("0199d220-0000-7000-8000-000000000002"), Title: "Current proof", DesiredState: "retraction remains visible", Priority: d.PriorityNormal})
			must(e)
			scope := out.Value.Scope()
			owner := out.Value.Ref()
			criterion, e := service.AddCriterion(ctx, cc, a.AddCriterionCommand{Owner: owner, ExpectedVersion: 1, Title: "Reviewed", Required: true, VerificationMode: d.VerificationModeEvidenceReview})
			must(e)
			_, e = service.ActivateOutcome(ctx, cc, a.ActivateOutcomeCommand{Scope: scope, ExpectedVersion: 2})
			must(e)
			evidence, e := service.RegisterEvidence(ctx, cc, a.RegisterEvidenceCommand{Scope: scope, EvidenceType: d.EvidenceTypeSource, Description: "original observation", SourceRef: d.SourceReference{Provider: "fixture", ID: "original"}, CapturedAt: clock.Now()})
			must(e)
			_, e = service.RecordCriterionAssessment(ctx, cc, a.RecordCriterionAssessmentCommand{Owner: owner, ExpectedVersion: 3, CriterionID: criterion.Value.ID, CriterionRevision: 1, Result: d.AssessmentResultMet, Rationale: "original reviewed source", EvidenceIDs: []d.ID{evidence.Value.ID}})
			must(e)
			_, e = service.AchieveOutcome(ctx, cc, a.AchieveOutcomeCommand{Scope: scope, ExpectedVersion: 4, Reason: "explicit conclusion"})
			must(e)
			initial, e := service.GetContinuity(ctx, scope, 25)
			must(e)
			if initial.Progress["required_criteria_met"].Numerator != 1 {
				t.Fatal("current required proof missing")
			}
			_, e = service.RetractEvidence(ctx, cc, a.RetractEvidenceCommand{Scope: scope, EvidenceID: evidence.Value.ID, ExpectedVersion: 1, Reason: "source withdrawn"})
			must(e)
			retracted, e := service.GetContinuity(ctx, scope, 25)
			must(e)
			if retracted.Progress["required_criteria_met"].Numerator != 0 || retracted.Counts["conclusion_contestations"] != 1 {
				t.Fatal("retracted source certified proof or disappeared", retracted.Progress, retracted.Counts)
			}
			replacement, e := service.RegisterEvidence(ctx, cc, a.RegisterEvidenceCommand{Scope: scope, EvidenceType: d.EvidenceTypeSource, Description: "new observation", SourceRef: d.SourceReference{Provider: "fixture", ID: "replacement"}, CapturedAt: clock.Now()})
			must(e)
			_, e = service.RecordCriterionAssessment(ctx, cc, a.RecordCriterionAssessmentCommand{Owner: owner, ExpectedVersion: 5, CriterionID: criterion.Value.ID, CriterionRevision: 1, Result: d.AssessmentResultNotMet, Rationale: "new contradiction", EvidenceIDs: []d.ID{replacement.Value.ID}})
			must(e)
			full, e := service.GetOutcomeState(ctx, scope)
			must(e)
			page, e := service.GetContinuitySection(ctx, scope, "conclusion_contestations", 25, "")
			must(e)
			want := []string{}
			for _, v := range full.ConclusionContestations {
				raw, e := json.Marshal(v)
				must(e)
				want = append(want, string(raw))
			}
			got := []string{}
			for _, v := range page.Items {
				got = append(got, string(v))
			}
			sort.Strings(want)
			sort.Strings(got)
			if len(want) != 2 || len(got) != 2 || want[0] != got[0] || want[1] != got[1] {
				t.Fatal("projection lost historical referenced evidence/current assessment", want, got)
			}
		})
	}
}

func TestContinuityProjectedBlockingLegacyCursorAndByteBudget(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			var store ports.TransactionManager = memory.New()
			if backend == "sql" {
				store = openTestStore(t, filepath.Join(t.TempDir(), "blocking-page.db"))
			}
			ctx := context.Background()
			now := time.Now().UTC()
			must := func(e error) {
				t.Helper()
				if e != nil {
					t.Fatal(e)
				}
			}
			scope := d.Scope{NamespaceID: portID(5000), OutcomeID: portID(5001)}
			out, e := d.NewOutcome(scope.OutcomeID, scope.NamespaceID, "Blocking pages", "", "bounded inherited details", d.PriorityNormal, now)
			must(e)
			u, e := store.Begin(ctx)
			must(e)
			must(u.Outcomes().Insert(ctx, out))
			works := []d.WorkItem{}
			for _, id := range []int{5010, 5011} {
				w, e := d.NewWorkItem(portID(id), scope, "work", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
				must(e)
				must(u.WorkItems().Insert(ctx, w))
				works = append(works, w)
			}
			actor := d.ActorRef{Kind: d.ActorKindService, Provider: "test", ID: "fixture"}
			issue, e := d.NewIssue(portID(5020), scope, "External dependency", "", d.IssueSeverityMinor, []d.EntityRef{out.Ref()}, actor, now)
			must(e)
			must(u.(ports.IssueBlockerUnitOfWork).Issues().Insert(ctx, issue))
			cause := issue.Ref()
			root, e := d.NewBlocker(portID(5030), out.Ref(), &cause, nil, "root inherited detail", d.BlockerPropagationSubtree, now)
			must(e)
			must(u.(ports.IssueBlockerUnitOfWork).Blockers().Insert(ctx, root))
			direct, e := d.NewBlocker(portID(5031), works[0].Ref(), &cause, nil, "direct work detail", d.BlockerPropagationDirect, now)
			must(e)
			must(u.(ports.IssueBlockerUnitOfWork).Blockers().Insert(ctx, direct))
			_, e = u.Coordination().AdvanceOutcome(ctx, scope)
			must(e)
			must(u.Commit())
			service, _ := a.NewService(store, sqliteFixedClock{now}, &portIDs{})
			full, e := service.GetOutcomeState(ctx, scope)
			must(e)
			legacyRows := []json.RawMessage{}
			for _, v := range full.BlockingStates {
				raw, e := json.Marshal(v)
				must(e)
				legacyRows = append(legacyRows, raw)
			}
			sort.Slice(legacyRows, func(i, j int) bool { return string(legacyRows[i]) < string(legacyRows[j]) })
			oldCursor := legacySectionCursor(scope, "blocking_states", full.OutcomeRevision, full.EvaluatedAt, legacyRows[0])
			page, e := service.GetContinuitySection(ctx, scope, "blocking_states", 1, oldCursor)
			must(e)
			if len(page.Items) != 1 || string(page.Items[0]) != string(legacyRows[1]) || page.Omitted != 1 {
				t.Fatal("legacy blocking digest/order/inherited proof changed", page)
			}
			var state a.BlockingState
			must(json.Unmarshal(page.Items[0], &state))
			if len(state.ActiveBlockers) != 2 || !state.ActiveBlockers[0].Inherited || state.ActiveBlockers[0].Blocker.Description != root.Description || state.ActiveBlockers[1].Blocker.Description != direct.Description {
				t.Fatal("selected blocker bodies/order lost", state)
			}
			// A large shared cause is loaded once for selected records; the final byte
			// envelope reduces pages deterministically while keeping counts and proof.
			u, e = store.Begin(ctx)
			must(e)
			r := u.(ports.IssueBlockerUnitOfWork).Blockers()
			current, e := r.Get(ctx, scope, root.ID)
			must(e)
			expected := current.Version
			current.Description = strings.Repeat("b", 200000)
			current.Version++
			must(r.Save(ctx, current, expected))
			_, e = u.Coordination().AdvanceOutcome(ctx, scope)
			must(e)
			must(u.Commit())
			snapshot, e := service.GetContinuity(ctx, scope, 25)
			must(e)
			raw, e := json.Marshal(snapshot)
			must(e)
			if len(raw) > a.MaxSnapshotBytes || !snapshot.Truncated || snapshot.Counts["blocking_states"] != 3 || snapshot.Limit != 1 {
				t.Fatal("large blocker bypassed response bound or lost counts", len(raw), snapshot.Limit)
			}
		})
	}
}
