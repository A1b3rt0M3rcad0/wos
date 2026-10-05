package server

import (
	"context"
	"errors"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	wossdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestIndependentSDKConsumerContinuesAfterRestart(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Storage.SQLitePath = filepath.Join(t.TempDir(), "sdk.db")
	runtime, err := OpenRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(runtime.Handler())
	client, _ := wossdk.New(srv.URL, "", srv.Client())
	ctx := context.Background()
	key, _ := wossdk.NewIdempotencyKey()
	created, err := client.CreateOutcome(ctx, key, application.CreateOutcomeCommand{NamespaceID: domain.MustParseID("0199d050-0000-7000-8000-000000000001"), Title: "SDK durable result", DesiredState: "Independent new consumer continues", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := client.CreateOutcome(ctx, key, application.CreateOutcomeCommand{NamespaceID: created.Value.NamespaceID, Title: "SDK durable result", DesiredState: "Independent new consumer continues", Priority: domain.PriorityNormal})
	if err != nil || !replay.IdempotentReplay || replay.Value.ID != created.Value.ID {
		t.Fatalf("replay: %v %+v", err, replay)
	}
	criterionKey, _ := wossdk.NewIdempotencyKey()
	if _, err = client.AddCriterion(ctx, criterionKey, application.AddCriterionCommand{Owner: created.Value.Ref(), ExpectedVersion: created.Value.Version, Title: "Verified continuation", Required: true, VerificationMode: domain.VerificationModeAttestation}); err != nil {
		t.Fatal(err)
	}
	staleKey, _ := wossdk.NewIdempotencyKey()
	_, err = client.UpdateOutcome(ctx, staleKey, application.UpdateOutcomeCommand{Scope: created.Value.Scope(), ExpectedVersion: 99, Title: new("Stale update")})
	var typed *wossdk.Error
	if !errors.As(err, &typed) || typed.Code != string(domain.ErrorCodeVersionConflict) {
		t.Fatalf("untyped stale error: %v", err)
	}
	srv.Close()
	runtime.Close()
	runtime, err = OpenRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	srv = httptest.NewServer(runtime.Handler())
	defer srv.Close()
	independent, _ := wossdk.New(srv.URL, "", srv.Client())
	found, err := independent.SearchOutcomes(ctx, created.Value.NamespaceID, "durable", 1, "")
	if err != nil || len(found.Items) != 1 {
		t.Fatalf("discovery after restart: %v %+v", err, found)
	}
	snapshot, err := independent.GetContinuity(ctx, found.Items[0].Scope(), 1)
	if err != nil || snapshot.Outcome.Ref.ID != created.Value.ID || snapshot.OutcomeRevision != created.OutcomeRevision+1 {
		t.Fatalf("continuation: %v %+v", err, snapshot)
	}
	activateKey, _ := wossdk.NewIdempotencyKey()
	activated, err := independent.ActivateOutcome(ctx, activateKey, application.ActivateOutcomeCommand{Scope: found.Items[0].Scope(), ExpectedVersion: found.Items[0].Version})
	if err != nil || activated.Value.Lifecycle != domain.OutcomeLifecycleActive {
		t.Fatalf("new consumer could not continue: %v", err)
	}
}
