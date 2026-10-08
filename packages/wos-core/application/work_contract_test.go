package application_test

import (
	"context"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"sync"
	"testing"
	"time"
)

type contractClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *contractClock) Now() time.Time  { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *contractClock) set(t time.Time) { c.mu.Lock(); defer c.mu.Unlock(); c.now = t }

type contractIDs struct {
	mu sync.Mutex
	n  int
}

func (g *contractIDs) NewID() (d.ID, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
	return d.ParseID(fmt.Sprintf("0199f2ff-0000-7000-8000-%012x", g.n))
}

type contractAdmin struct{}

func (contractAdmin) Authorize(context.Context, ports.AuthorizationRequest) error { return nil }
func contractServiceFixture(t *testing.T) (*a.Service, *memory.Store, *contractClock, d.WorkItem) {
	t.Helper()
	store := memory.New()
	clock := &contractClock{now: time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)}
	s, err := a.NewServiceWithAuthorizer(store, clock, &contractIDs{}, contractAdmin{})
	if err != nil {
		t.Fatal(err)
	}
	scope := d.Scope{NamespaceID: id("0199e200-0000-7000-8000-000000000001"), OutcomeID: id("0199e200-0000-7000-8000-000000000002")}
	o, err := d.NewOutcome(scope.OutcomeID, scope.NamespaceID, "Contract outcome", "", "Verifiable", d.PriorityNormal, clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	o.Lifecycle = d.OutcomeLifecycleActive
	w, err := d.NewWorkItem(id("0199e200-0000-7000-8000-000000000003"), scope, "Work", "Focused context", d.PriorityNormal, d.WorkItemLifecycleTodo, clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	w.ContractsEnabled = true
	tx, err := store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Coordination().LockOutcome(context.Background(), scope); err != nil {
		t.Fatal(err)
	}
	if err := tx.Outcomes().Insert(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if err := tx.WorkItems().Insert(context.Background(), w); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return s, store, clock, w
}
func TestContractAcquireReplayRenewAndRecovery(t *testing.T) {
	s, _, clock, w := contractServiceFixture(t)
	ctx := context.Background()
	cc := commandContext()
	cc.IdempotencyKey = "contract-acquire-0001"
	cmd := a.AcquireWorkContractCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version, TTLSeconds: 30}
	got, err := s.AcquireWorkContract(ctx, cc, cmd)
	if err != nil {
		t.Fatal(err)
	}
	c := got.Value.Contract
	replay, err := s.AcquireWorkContract(ctx, cc, cmd)
	if err != nil || !replay.IdempotentReplay || replay.Value.Contract.ID != c.ID {
		t.Fatalf("replay %v %v", replay, err)
	}
	cc.IdempotencyKey = "contract-acquire-0002"
	cmd.ExpectedWorkItemVersion = got.Value.WorkItem.Version
	if _, err := s.AcquireWorkContract(ctx, cc, cmd); err == nil {
		t.Fatal("second active acquisition accepted")
	}
	renewal := a.RenewWorkContractCommand{Scope: w.Scope, ContractID: c.ID, ContractAuthority: a.ContractAuthority{ExecutionID: c.ExecutionID, FencingToken: c.FencingToken, SpecDigest: c.SpecDigest}, ExpectedLeaseVersion: c.LeaseVersion, TTLSeconds: 60}
	cc.IdempotencyKey = "contract-renewal-0001"
	renewed, err := s.RenewWorkContract(ctx, cc, renewal)
	if err != nil {
		t.Fatal(err)
	}
	if renewed.Value.WorkItem.Version != got.Value.WorkItem.Version || renewed.Value.Contract.Version != c.Version {
		t.Fatal("renew changed progress/work version")
	}
	clock.set(clock.Now().Add(10 * time.Second))
	repeat, err := s.RenewWorkContract(ctx, cc, renewal)
	if err != nil || !repeat.Value.Contract.ExpiresAt.Equal(renewed.Value.Contract.ExpiresAt) {
		t.Fatal("renew replay extended again")
	}
	clock.set(renewed.Value.Contract.ExpiresAt)
	cc.IdempotencyKey = "contract-renewal-0002"
	renewal.ExpectedLeaseVersion = renewed.Value.Contract.LeaseVersion
	if _, err := s.RenewWorkContract(ctx, cc, renewal); err == nil {
		t.Fatal("renew at deadline accepted")
	}
	cc.IdempotencyKey = "contract-recovery-0001"
	next, err := s.AcquireWorkContract(ctx, cc, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if next.Value.Contract.ID == c.ID || next.Value.Contract.FencingToken <= c.FencingToken || !next.Value.Recovery || next.Value.PreviousContract.Status != d.ContractExpired {
		t.Fatal("recovery did not fence/close previous")
	}
	view, err := s.GetWorkContract(ctx, w.Scope, c.ID)
	if err != nil || view.Value.EffectiveStatus != d.ContractExpired {
		t.Fatalf("old view %v %v", view, err)
	}
}
func TestContractConcurrentAcquireAndTakeover(t *testing.T) {
	s, _, clock, w := contractServiceFixture(t)
	var wg sync.WaitGroup
	results := make(chan a.WorkContractResult, 2)
	errors := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cc := commandContext()
			cc.IdempotencyKey = fmt.Sprintf("concurrent-contract-%02d", i)
			v, err := s.AcquireWorkContract(context.Background(), cc, a.AcquireWorkContractCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version, TTLSeconds: 30})
			if err == nil {
				results <- v.Value
			} else {
				errors <- err
			}
		}(i)
	}
	wg.Wait()
	if len(results) != 1 || len(errors) != 1 {
		t.Fatal("multiple/no winners")
	}
	got := <-results
	c := got.Contract
	cc := commandContext()
	cc.IdempotencyKey = "contract-takeover-0001"
	resumed, err := s.ResumeWorkContract(context.Background(), cc, a.ResumeWorkContractCommand{Scope: w.Scope, ContractID: c.ID, ContractAuthority: a.ContractAuthority{ExecutionID: c.ExecutionID, FencingToken: c.FencingToken, SpecDigest: c.SpecDigest}, ExpectedLeaseVersion: c.LeaseVersion})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Value.Contract.ExecutionID == c.ExecutionID || resumed.Value.Contract.FencingToken <= c.FencingToken || !resumed.Value.Contract.ExpiresAt.Equal(c.ExpiresAt) {
		t.Fatal("takeover authority")
	}
	cc.IdempotencyKey = "stale-renewal-intent-1"
	if _, err := s.RenewWorkContract(context.Background(), cc, a.RenewWorkContractCommand{Scope: w.Scope, ContractID: c.ID, ContractAuthority: a.ContractAuthority{ExecutionID: c.ExecutionID, FencingToken: c.FencingToken, SpecDigest: c.SpecDigest}, ExpectedLeaseVersion: resumed.Value.Contract.LeaseVersion}); err == nil {
		t.Fatal("old execution accepted")
	}
	clock.set(c.ExpiresAt)
	view, err := s.GetWorkContract(context.Background(), w.Scope, c.ID)
	if err != nil || view.Value.ExecutionAllowed || view.Value.EffectiveStatus != d.ContractExpired {
		t.Fatal("read granted expired authority")
	}
}
func TestContractRevokeReasonAndHistory(t *testing.T) {
	s, store, _, w := contractServiceFixture(t)
	ctx := context.Background()
	cc := commandContext()
	cc.IdempotencyKey = "contract-acquire-history"
	got, err := s.AcquireWorkContract(ctx, cc, a.AcquireWorkContractCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version})
	if err != nil {
		t.Fatal(err)
	}
	c := got.Value.Contract
	cc.IdempotencyKey = "contract-revoke-history-1"
	cmd := a.RevokeWorkContractCommand{Scope: w.Scope, ContractID: c.ID, ExpectedContractVersion: c.Version}
	if _, err := s.RevokeWorkContract(ctx, cc, cmd); err == nil {
		t.Fatal("reasonless revoke")
	}
	cmd.Reason = "Administrative intervention"
	v, err := s.RevokeWorkContract(ctx, cc, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if v.Value.Contract.Status != d.ContractRevoked || v.Value.WorkItem.Lifecycle != d.WorkItemLifecycleInProgress || v.Value.WorkItem.CurrentContractID != nil {
		t.Fatal("revocation state")
	}
	tx, _ := store.Begin(ctx)
	defer tx.Rollback()
	repo := tx.(ports.WorkContractUnitOfWork).WorkContracts()
	copy, err := repo.Get(ctx, w.Scope, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	copy.Spec.Title = "edited"
	again, err := repo.Get(ctx, w.Scope, c.ID)
	if err != nil || again.Spec.Title != c.Spec.Title {
		t.Fatal("memory did not clone snapshot")
	}
}
