package integration_test

import (
	"context"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/integration"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

type ids struct{ next uint64 }

func (g *ids) NewID() (domain.ID, error) {
	g.next++
	return domain.ParseID(fmt.Sprintf("0199d101-0000-7000-8000-%012x", g.next))
}
func TestRealWebhookRetryKeepsEventIdentityAndSignature(t *testing.T) {
	ctx := context.Background()
	secret := strings.Repeat("s", 40)
	clock := &clock{now: time.Now().UTC()}
	ids := &ids{}
	var mu sync.Mutex
	received := []string{}
	invalid := false
	hold := false
	started := make(chan struct{}, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		err := integration.VerifySignature(secret, r.Header.Get("X-WOS-Timestamp"), r.Header.Get("X-WOS-Event-ID"), r.Header.Get("X-WOS-Signature"), body, clock.Now())
		mu.Lock()
		invalid = invalid || err != nil
		received = append(received, r.Header.Get("X-WOS-Event-ID"))
		count, waiting := len(received), hold
		mu.Unlock()
		if waiting {
			started <- struct{}{}
			<-r.Context().Done()
			return
		}
		if count == 1 {
			w.WriteHeader(503)
		} else {
			w.WriteHeader(204)
		}
	}))
	defer target.Close()
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "worker.db"), sqlite.Options{MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service, _ := application.NewService(store, clock, ids)
	ns := domain.MustParseID("0199d102-0000-7000-8000-000000000001")
	commandID, _ := ids.NewID()
	cc := domain.CommandContext{PrincipalID: "operator", Actor: domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "operator"}, CommandID: commandID}
	outcome, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{NamespaceID: ns, Title: "Webhook", DesiredState: "Durable signed delivery", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	endpointID, _ := ids.NewID()
	endpoint := ports.WebhookEndpoint{ID: endpointID, NamespaceID: ns, URL: target.URL, SecretRef: "target-key", KeyID: "key-1"}
	if err = store.InstallEndpoints(ctx, []ports.WebhookEndpoint{endpoint}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.ConfigureTrigger(ctx, cc, application.ConfigureTriggerCommand{Scope: outcome.Value.Scope(), Name: "Notify", EventTypes: []string{"work_item.created"}, TargetEndpointIDs: []domain.ID{endpointID}, SignalType: "product.work_available"}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: outcome.Value.Scope(), Title: "External actor decides", Lifecycle: domain.WorkItemLifecycleTodo, Priority: domain.PriorityNormal}); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(target.URL)
	worker, err := integration.New(store, clock, ids, map[string]string{"target-key": secret}, integration.Policy{AllowedHosts: []string{u.Hostname()}, AllowLoopback: true})
	if err != nil {
		t.Fatal(err)
	}
	worked, err := worker.Tick(ctx)
	if err != nil || !worked {
		t.Fatal("first attempt", err)
	}
	pending, err := service.ListDeliveries(ctx, outcome.Value.Scope(), 25, "")
	if err != nil || pending.Items[0].Status != "pending" || pending.Items[0].Attempts != 1 {
		t.Fatal("unavailable destination lost delivery", err)
	}
	worked, err = worker.Tick(ctx)
	if err != nil || worked {
		t.Fatal("ignored retry backoff", err)
	}
	clock.now = clock.now.Add(3 * time.Second)
	worked, err = worker.Tick(ctx)
	if err != nil || !worked {
		t.Fatal("retry", err)
	}
	completed, err := service.ListDeliveries(ctx, outcome.Value.Scope(), 25, "")
	if err != nil || completed.Items[0].Status != "succeeded" || completed.Items[0].Attempts != 2 {
		t.Fatal("retry did not succeed", err)
	}
	mu.Lock()
	if invalid || len(received) != 2 || received[0] != received[1] {
		t.Fatal("retry changed identity or had invalid signature", received)
	}
	hold = true
	mu.Unlock()
	if _, err = service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: outcome.Value.Scope(), Title: "Shutdown preserves delivery", Lifecycle: domain.WorkItemLifecycleTodo, Priority: domain.PriorityNormal}); err != nil {
		t.Fatal(err)
	}
	interrupted, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { _, err := worker.Tick(interrupted); done <- err }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("delivery did not start")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("interrupted delivery did not finish")
	}
	pending, err = service.ListDeliveries(ctx, outcome.Value.Scope(), 25, "")
	if err != nil {
		t.Fatal(err)
	}
	var retryID domain.ID
	for _, delivery := range pending.Items {
		if delivery.Status == "pending" && delivery.Attempts == 1 {
			retryID = delivery.IntegrationEventID
		}
	}
	if retryID == "" {
		t.Fatal("shutdown lost pending delivery")
	}
	mu.Lock()
	hold = false
	mu.Unlock()
	clock.now = clock.now.Add(3 * time.Second)
	if worked, err = worker.Tick(ctx); err != nil || !worked {
		t.Fatal("resume interrupted worker", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if invalid || len(received) != 4 || received[2] != received[3] || received[3] != retryID.String() {
		t.Fatal("shutdown/resume changed delivery identity", received)
	}
}
