package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/authentication/local"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestOperationalQueriesAreScopedAndReadOnly(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	store := memory.New()
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199ec10-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199ec10-0000-7000-8000-000000000002"),
	}
	outcome, err := domain.NewOutcome(scope.OutcomeID, scope.NamespaceID, "Operational query HTTP test", "", "Verified work is observable", domain.PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	objectiveID := domain.MustParseID("0199ec10-0000-7000-8000-000000000003")
	objective, err := domain.NewObjective(objectiveID, scope, "Ship the verified slice", "", domain.PriorityNormal, true, now)
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the public quoted-decimal wire format above JavaScript's exact
	// integer range; the read must preserve this revision without float coercion.
	objective.Version = domain.Version(9007199254740993)
	workID := domain.MustParseID("0199ec10-0000-7000-8000-000000000004")
	work, err := domain.NewWorkItemForObjective(workID, scope, &objective.ID, "Implement the slice", "Return exact proof", domain.PriorityNormal, domain.WorkItemLifecycleBacklog, now)
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "http-test", ID: "worker"}
	credentialID := domain.MustParseID("0199ec10-0000-7000-8000-000000000005")
	claimID := domain.MustParseID("0199ec10-0000-7000-8000-000000000006")
	if err := work.Activate(now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := work.ClaimForCredential(claimID, "http-query-worker", credentialID, nil, actor, domain.MinLeaseTTL, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Coordination().LockOutcome(ctx, scope); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Outcomes().Insert(ctx, outcome); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Objectives().Insert(ctx, objective); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.WorkItems().Insert(ctx, work); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	deliveryID := domain.MustParseID("0199ec10-0000-7000-8000-000000000007")
	deliveryTime := now.Add(3 * time.Second)
	delivery, err := domain.NewLegacyTaskDelivery(deliveryID, work, domain.TaskDeliveryMaterial{Summary: "The implementation and proof are ready"}, "http-query-worker", &credentialID, deliveryTime)
	if err != nil {
		t.Fatal(err)
	}
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	current, err := tx.WorkItems().Get(ctx, scope, workID)
	if err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	before := current.Version
	if err := current.RecordTaskDelivery(delivery, nil, nil, before, deliveryTime); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries().Insert(ctx, delivery); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.WorkItems().Save(ctx, current, before); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if _, err := tx.Coordination().AdvanceOutcome(ctx, scope); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	service, err := application.NewService(store, fixedClock{value: now.Add(4 * time.Second)}, &sequenceIDs{prefix: "0199ec20", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	auth, err := local.New("http-human")
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(Options{Prefix: "/api/v1", RequestTimeout: 2 * time.Second, Service: service, IDs: &sequenceIDs{prefix: "0199ec30", next: 1}, LocalAuth: auth})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()
	base := server.URL + "/api/v1/namespaces/" + scope.NamespaceID.String() + "/outcomes/" + scope.OutcomeID.String()

	completion := doJSON[application.ReadResult[application.ObjectiveCompletionReport]](t, client, http.MethodGet, base+"/objectives/"+objective.ID.String()+"/completion?limit=1", nil, nil, http.StatusOK)
	if completion.Value.Value.Objective.ID != objective.ID || completion.Value.Value.ObjectiveVersion != objective.Version {
		t.Fatalf("completion report identity/version = %s/v%d, want %s/v%d", completion.Value.Value.Objective.ID, completion.Value.Value.ObjectiveVersion, objective.ID, objective.Version)
	}
	wantObjectiveETag := `"objective:` + objective.ID.String() + `:v9007199254740993"`
	if completion.Header.Get("Cache-Control") != "no-store" || completion.Header.Get("ETag") != wantObjectiveETag {
		t.Fatalf("completion headers missing no-store or objective ETag: %#v", completion.Header)
	}
	wireResponse := doRequest(t, client, http.MethodGet, base+"/objectives/"+objective.ID.String()+"/completion?limit=1", nil, nil)
	var wire struct {
		Value struct {
			ObjectiveVersion json.RawMessage `json:"objective_version"`
		} `json:"value"`
	}
	if wireResponse.StatusCode != http.StatusOK {
		wireResponse.Body.Close()
		t.Fatalf("completion decimal wire read status = %d, want 200", wireResponse.StatusCode)
	}
	if err := json.NewDecoder(wireResponse.Body).Decode(&wire); err != nil {
		wireResponse.Body.Close()
		t.Fatal(err)
	}
	wireResponse.Body.Close()
	if string(wire.Value.ObjectiveVersion) != `"9007199254740993"` {
		t.Fatalf("Objective version JSON = %s, want exact quoted decimal", wire.Value.ObjectiveVersion)
	}

	verification := doJSON[application.TaskVerificationView](t, client, http.MethodGet, base+"/work-items/"+work.ID.String()+"/verification", nil, nil, http.StatusOK)
	if !verification.Value.Pending || verification.Value.SourceKind != "task_delivery" || verification.Value.SourceID == nil || *verification.Value.SourceID != delivery.ID {
		t.Fatalf("verification did not project the actual pending native delivery: %#v", verification.Value)
	}
	if verification.Header.Get("Cache-Control") != "no-store" || verification.Header.Get("ETag") == "" {
		t.Fatalf("verification headers missing no-store or Task ETag: %#v", verification.Header)
	}

	detail := doJSON[application.ReadResult[application.TaskDeliveryDetail]](t, client, http.MethodGet, base+"/task-deliveries/"+delivery.ID.String(), nil, nil, http.StatusOK)
	if detail.Value.Value.Delivery.ID != delivery.ID || !detail.Value.Value.Pending || !detail.Value.Value.Latest || detail.Value.Value.Decision != nil || detail.Value.Value.Withdrawal != nil {
		t.Fatalf("TaskDelivery detail did not preserve the pending immutable handoff: %#v", detail.Value.Value)
	}
	if detail.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("TaskDelivery detail is cacheable: %#v", detail.Header)
	}

	wrongOutcome := domain.MustParseID("0199ec10-0000-7000-8000-000000000008")
	crossScope := doRequest(t, client, http.MethodGet, server.URL+"/api/v1/namespaces/"+scope.NamespaceID.String()+"/outcomes/"+wrongOutcome.String()+"/task-deliveries/"+delivery.ID.String(), nil, nil)
	defer crossScope.Body.Close()
	if crossScope.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-Outcome TaskDelivery read status = %d, want 404", crossScope.StatusCode)
	}

	badLimit := doRequest(t, client, http.MethodGet, base+"/objectives/"+objective.ID.String()+"/completion?limit=21", nil, nil)
	defer badLimit.Body.Close()
	if badLimit.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("out-of-range completion limit status = %d, want 422", badLimit.StatusCode)
	}
}
