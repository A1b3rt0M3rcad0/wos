package server

import (
	"context"
	"errors"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDeliveryHealthDisabledBacklogDoesNotChangeAPIReadiness(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		cfg.Auth.BootstrapNamespaceID = "0199d129-0000-7000-8000-000000000010"
		cfg.Auth.BootstrapNamespaceName = "delivery health"
		cfg.Auth.Mode = AuthModeAPIToken
		cfg.Auth.BootstrapToken = strings.Repeat("d", 40)
		ns := d.MustParseID(cfg.Auth.BootstrapNamespaceID)
		endpoint := ports.WebhookEndpoint{ID: d.MustParseID("0199d129-0000-7000-8000-000000000001"), NamespaceID: ns, URL: "https://unavailable.example.test/hook", SecretRef: "not-returned", KeyID: "key"}
		cfg.Integration.Endpoints = []ports.WebhookEndpoint{endpoint}
		cfg.Integration.WorkerEnabled = false
		runtime, e := OpenRuntime(cfg)
		if e != nil {
			t.Fatal(e)
		}
		defer runtime.Close()
		ctx := context.Background()
		seed, _ := a.NewService(runtime.store, systemClock{}, uuidV7Generator{})
		cc := d.CommandContext{CommandID: d.MustParseID("0199d129-0000-7000-8000-000000000002"), PrincipalID: "test", Actor: d.ActorRef{Kind: d.ActorKindService, Provider: "test", ID: "seed"}}
		out, e := seed.CreateOutcome(ctx, cc, a.CreateOutcomeCommand{NamespaceID: ns, Title: "Delivery health", DesiredState: "observable backlog", Priority: d.PriorityNormal})
		if e != nil {
			t.Fatal(e)
		}
		_, e = seed.ConfigureTrigger(ctx, cc, a.ConfigureTriggerCommand{Scope: out.Value.Scope(), Name: "signal", EventTypes: []string{"work_item.created"}, TargetEndpointIDs: []d.ID{endpoint.ID}, SignalType: "test.ready"})
		if e != nil {
			t.Fatal(e)
		}
		_, e = seed.CreateWorkItem(ctx, cc, a.CreateWorkItemCommand{Scope: out.Value.Scope(), Title: "emit signal", Priority: d.PriorityNormal})
		if e != nil {
			t.Fatal(e)
		}
		host := httptest.NewServer(runtime.Handler())
		defer host.Close()
		client, _ := sdk.New(host.URL, cfg.Auth.BootstrapToken, host.Client())
		health, e := client.GetDeliveryHealth(ctx, a.DeliveryHealthQuery{NamespaceID: ns})
		if e != nil {
			t.Fatal(e)
		}
		if health.State != "disabled" || health.Queue.Pending != 1 || health.Worker.Running || health.Queue.LastClaim != nil {
			t.Fatalf("disabled worker claimed delivery health: %+v", health)
		}
		for _, path := range []string{"/livez", "/readyz"} {
			response, e := host.Client().Get(host.URL + path)
			if e != nil {
				t.Fatal(e)
			}
			response.Body.Close()
			if response.StatusCode != 200 {
				t.Fatalf("backlog broke %s", path)
			}
		}
	})
}

type fatalDeliveryStore struct{ ports.DeliveryStore }

func (fatalDeliveryStore) ClaimDelivery(context.Context, time.Time, d.ID, time.Duration) (*ports.Delivery, error) {
	return nil, errors.New("synthetic fatal database fault with private detail")
}
func TestDeliveryFatalWorkerIsVisibleWhileAPIStaysReady(t *testing.T) {
	cfg := transportConfig(t)
	cfg.Integration.WorkerEnabled = true
	runtime, e := OpenRuntime(cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer runtime.Close()
	runtime.deliveryWorker.Store = fatalDeliveryStore{runtime.deliveryWorker.Store}
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runtime.serve(ctx, l) }()
	defer func() {
		cancel()
		if e := <-done; e != nil {
			t.Error(e)
		}
	}()
	deadline := time.After(4 * time.Second)
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for runtime.deliveryDiagnostics.DeliveryWorkerStatus().LastFailureClass == "" {
		select {
		case <-deadline:
			t.Fatal("fatal worker failure remained invisible")
		case <-tick.C:
		}
	}
	status := runtime.deliveryDiagnostics.DeliveryWorkerStatus()
	if status.Running || status.LastFailureClass != "storage_error" {
		t.Fatal(status)
	}
	response, e := http.Get("http://" + l.Addr().String() + "/readyz")
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("worker death broke Core readiness")
	}
}
