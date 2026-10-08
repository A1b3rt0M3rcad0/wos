package sqlite

import (
	"context"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"math"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func contractStorageFixture(t *testing.T, store *Store, now time.Time) d.WorkItem {
	t.Helper()
	ctx := context.Background()
	scope := d.Scope{NamespaceID: testID("0199ff10-0000-7000-8000-000000000001"), OutcomeID: testID("0199ff10-0000-7000-8000-000000000002")}
	o, err := d.NewOutcome(scope.OutcomeID, scope.NamespaceID, "Contracts", "", "Verified", d.PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	o.Lifecycle = d.OutcomeLifecycleActive
	w, err := d.NewWorkItem(testID("0199ff10-0000-7000-8000-000000000003"), scope, "Precise authority", "Independent task", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
	if err != nil {
		t.Fatal(err)
	}
	w.ContractsEnabled = true
	w.LastFencingToken = 9007199254740993
	w.ExecutionSpec = &d.ExecutionSpec{Instructions: []string{"Run externally"}}
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Coordination().LockOutcome(ctx, scope); err != nil {
		t.Fatal(err)
	}
	if err := tx.Outcomes().Insert(ctx, o); err != nil {
		t.Fatal(err)
	}
	if err := tx.WorkItems().Insert(ctx, w); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return w
}
func TestWorkContractStorageRestartPrecisionAndExpiry(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "contracts.db")
	store := openTestStore(t, path)
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	w := contractStorageFixture(t, store, now)
	s, err := a.NewService(store, sqliteFixedClock{now}, &sqliteSequenceIDs{prefix: "0199ff11", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	cc := sqliteCommandContext("0199ff10-0000-7000-8000-000000000010", "contract-storage-acquire-1")
	cmd := a.AcquireWorkContractCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version, TTLSeconds: 30}
	got, err := s.AcquireWorkContract(ctx, cc, cmd)
	if err != nil {
		t.Fatal(err)
	}
	c := got.Value.Contract
	if uint64(c.FencingToken) != w.LastFencingToken+1 {
		t.Fatal("fencing lost precision")
	}
	store = reopenIntegrationFixture(t, store, path)
	s, err = a.NewService(store, sqliteFixedClock{c.ExpiresAt}, &sqliteSequenceIDs{prefix: "0199ff12", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.AcquireWorkContract(ctx, cc, cmd)
	if err != nil || !replay.IdempotentReplay || replay.Value.Contract.ID != c.ID {
		t.Fatalf("restart replay %v %v", replay, err)
	}
	view, err := s.GetWorkContract(ctx, w.Scope, c.ID)
	if err != nil || view.Value.ExecutionAllowed || view.Value.EffectiveStatus != d.ContractExpired {
		t.Fatalf("restore expiry %v %v", view, err)
	}
	cc.IdempotencyKey = "contract-storage-recover-1"
	cmd.ExpectedWorkItemVersion = got.Value.WorkItem.Version
	recovered, err := s.AcquireWorkContract(ctx, cc, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Value.PreviousContract.Status != d.ContractExpired || recovered.Value.Contract.FencingToken <= c.FencingToken {
		t.Fatal("recovery fencing/history")
	}
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	repo := tx.(ports.WorkContractUnitOfWork).WorkContracts()
	old, err := repo.Get(ctx, w.Scope, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if old.Status != d.ContractExpired || old.SpecDigest != c.SpecDigest {
		t.Fatal("historical spec changed")
	}
	live, err := tx.WorkItems().Get(ctx, w.Scope, w.ID)
	if err != nil || live.ExecutionSpec == nil || live.LastFencingToken != uint64(recovered.Value.Contract.FencingToken) {
		t.Fatal("work metadata/fencing not durable")
	}
}
func TestWorkContractStorageConcurrentIndependentConnections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "concurrent.db")
	first := openTestStore(t, path)
	second := openTestStore(t, path)
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	w := contractStorageFixture(t, first, now)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i, store := range []*Store{first, second} {
		wg.Add(1)
		go func(i int, store *Store) {
			defer wg.Done()
			prefix := "0199ff13"
			key := "contract-concurrent-0001"
			if i == 1 {
				prefix = "0199ff14"
				key = "contract-concurrent-0002"
			}
			s, err := a.NewService(store, sqliteFixedClock{now}, &sqliteSequenceIDs{prefix: prefix, next: 1})
			if err != nil {
				results <- err
				return
			}
			cc := sqliteCommandContext("0199ff10-0000-7000-8000-000000000011", key)
			_, err = s.AcquireWorkContract(context.Background(), cc, a.AcquireWorkContractCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version})
			results <- err
		}(i, store)
	}
	wg.Wait()
	wins := 0
	for i := 0; i < 2; i++ {
		if <-results == nil {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("winners %d", wins)
	}
	tx, _ := first.Begin(context.Background())
	defer tx.Rollback()
	list, err := tx.(ports.WorkContractUnitOfWork).WorkContracts().List(context.Background(), w.Scope, ports.ContractFilter{Limit: 100})
	if err != nil || len(list) != 1 {
		t.Fatalf("open contracts %d %v", len(list), err)
	}
}
func TestWorkContractStorageUnsignedMaximum(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "unsigned.db"))
	now := time.Now().UTC()
	w := contractStorageFixture(t, store, now)
	tx, _ := store.Begin(ctx)
	loaded, err := tx.WorkItems().Get(ctx, w.Scope, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	loaded.LastFencingToken = math.MaxUint64
	loaded.Version++
	if err := tx.WorkItems().Save(ctx, loaded, w.Version); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	tx, _ = store.Begin(ctx)
	defer tx.Rollback()
	roundtrip, err := tx.WorkItems().Get(ctx, w.Scope, w.ID)
	if err != nil || roundtrip.LastFencingToken != math.MaxUint64 {
		t.Fatalf("max unsigned roundtrip %d %v", roundtrip.LastFencingToken, err)
	}
}

func TestWorkContractFinalizationProofSurvivesRestore(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "contract-results.db")
	store := openTestStore(t, path)
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	w := contractStorageFixture(t, store, now)
	s, err := a.NewService(store, sqliteFixedClock{now}, &sqliteSequenceIDs{prefix: "0199ff21", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	cc := sqliteCommandContext("0199ff10-0000-7000-8000-000000000020", "contract-result-criterion")
	if _, err = s.AddCriterion(ctx, cc, a.AddCriterionCommand{Owner: w.Ref(), ExpectedVersion: w.Version, Title: "Checked delivery", Required: true, VerificationMode: d.VerificationModeAttestation}); err != nil {
		t.Fatal(err)
	}
	cc.IdempotencyKey = "contract-result-acquire"
	acquired, err := s.AcquireWorkContract(ctx, cc, a.AcquireWorkContractCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version + 1})
	if err != nil {
		t.Fatal(err)
	}
	c := acquired.Value.Contract
	authority := a.ContractAuthority{ExecutionID: c.ExecutionID, FencingToken: c.FencingToken, SpecDigest: c.SpecDigest}
	cc.IdempotencyKey = "contract-result-submit"
	submitted, err := s.SubmitWorkResult(ctx, cc, a.SubmitWorkResultCommand{Scope: w.Scope, ContractID: c.ID, Authority: authority, ExpectedContractVersion: c.Version, Material: d.WorkResultMaterial{ContractID: c.ID, WorkItemID: w.ID, SpecDigest: c.SpecDigest, Summary: "Exact delivery"}})
	if err != nil {
		t.Fatal(err)
	}
	criterion := acquired.Value.WorkItem.Criteria.Items[0]
	cc.IdempotencyKey = "contract-result-review"
	if _, err = s.AttestCriterion(ctx, cc, a.AttestCriterionCommand{Owner: w.Ref(), CriterionID: criterion.ID, CriterionRevision: criterion.Revision, ExpectedVersion: acquired.Value.WorkItem.Version, Result: d.AssessmentResultMet, Rationale: "Verified material", SubmissionID: &submitted.Value.Submission.ID}); err != nil {
		t.Fatal(err)
	}
	cc.IdempotencyKey = "contract-result-finalize"
	cmd := a.FinalizeWorkContractCommand{Scope: w.Scope, ContractID: c.ID, Authority: authority, ExpectedContractVersion: submitted.Value.Contract.Version, ExpectedWorkItemVersion: acquired.Value.WorkItem.Version + 1, SubmissionID: submitted.Value.Submission.ID, Reason: "Proof accepted"}
	finished, err := s.FinalizeWorkContract(ctx, cc, cmd)
	if err != nil {
		t.Fatal(err)
	}
	store = reopenIntegrationFixture(t, store, path)
	s, err = a.NewService(store, sqliteFixedClock{now}, &sqliteSequenceIDs{prefix: "0199ff22", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.FinalizeWorkContract(ctx, cc, cmd)
	if err != nil || !replay.IdempotentReplay || replay.Value.Contract.ID != finished.Value.Contract.ID {
		t.Fatalf("finalize receipt after restore: %v", err)
	}
	got, err := s.GetWorkItem(ctx, w.Scope, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Value.CurrentConclusion.SubmissionDigest != submitted.Value.Submission.Digest || got.Value.Criteria.CurrentAssessments[criterion.ID].SubmissionDigest != submitted.Value.Submission.Digest {
		t.Fatal("submission proof lost on restore")
	}
	snapshot, err := s.GetOutcomeState(ctx, w.Scope)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.WorkItems[0].CurrentConclusion.SubmissionDigest != submitted.Value.Submission.Digest {
		t.Fatal("bulk snapshot lost binding")
	}
}
