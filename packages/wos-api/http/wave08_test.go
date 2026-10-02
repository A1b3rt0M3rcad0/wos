package httptransport

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/authentication/local"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type mutableClock struct {
	value time.Time
}

func (c *mutableClock) Now() time.Time { return c.value }

func newWave08TestHandler(t *testing.T, clock *mutableClock) *Handler {
	t.Helper()
	store := memory.New()
	service, err := application.NewService(
		store,
		clock,
		&sequenceIDs{prefix: "0199ec80", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	auth, err := local.New("http-human")
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(Options{
		Prefix:         "/api/v1",
		RequestTimeout: 2 * time.Second,
		Service:        service,
		IDs:            &sequenceIDs{prefix: "0199ec81", next: 1},
		LocalAuth:      auth,
	})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestWave08LeaseHTTPContract(t *testing.T) {
	clock := &mutableClock{value: time.Date(2026, 10, 3, 1, 0, 0, 0, time.UTC)}
	handler := newWave08TestHandler(t, clock)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()

	namespaceID := "0199ec82-0000-7000-8000-000000000001"
	base := server.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes"
	created := doJSON[application.MutationResult[domain.Outcome]](
		t,
		client,
		http.MethodPost,
		base,
		[]byte(`{"title":"Wave 08 HTTP","desired_state":"leases are fenced","priority":"normal"}`),
		map[string]string{"Idempotency-Key": "wave08-outcome-create-0001"},
		http.StatusCreated,
	)
	outcome := created.Value.Value
	outcomeURL := base + "/" + outcome.ID.String()

	criterion := doJSON[application.MutationResult[domain.SuccessCriterion]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/criteria",
		[]byte(`{"title":"Activate","required":true,"verification_mode":"attestation"}`),
		map[string]string{
			"Idempotency-Key": "wave08-outcome-criterion-0001",
			"If-Match":        created.Header.Get("ETag"),
		},
		http.StatusCreated,
	)
	doJSON[application.MutationResult[domain.Outcome]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/actions/activate",
		[]byte(`{}`),
		map[string]string{
			"Idempotency-Key": "wave08-outcome-activate-0001",
			"If-Match":        criterion.Header.Get("ETag"),
		},
		http.StatusOK,
	)

	workResponse := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/work-items",
		[]byte(`{"title":"Lease work","priority":"normal","lifecycle":"todo"}`),
		map[string]string{"Idempotency-Key": "wave08-work-create-0001"},
		http.StatusCreated,
	)
	work := workResponse.Value.Value
	workURL := outcomeURL + "/work-items/" + work.ID.String()

	claimed := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		client,
		http.MethodPost,
		workURL+"/actions/claim",
		[]byte(`{"lease_ttl_seconds":30}`),
		map[string]string{
			"Idempotency-Key": "wave08-work-claim-0001",
			"If-Match":        workResponse.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	oldClaim := claimed.Value.Value.CurrentLease
	if oldClaim == nil {
		t.Fatal("claim did not return lease")
	}

	clock.value = clock.value.Add(10 * time.Second)
	renewed := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		client,
		http.MethodPost,
		workURL+"/actions/renew-lease",
		[]byte(`{"claim_id":"`+oldClaim.ClaimID.String()+`","fencing_token":`+itoaUint64(oldClaim.FencingToken)+`,"lease_ttl_seconds":30}`),
		map[string]string{
			"Idempotency-Key": "wave08-work-renew-0001",
			"If-Match":        claimed.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if renewed.Value.Value.CurrentLease == nil ||
		renewed.Value.Value.CurrentLease.ClaimID != oldClaim.ClaimID ||
		renewed.Value.Value.CurrentLease.FencingToken != oldClaim.FencingToken {
		t.Fatalf("renew rotated claim identity: %#v", renewed.Value.Value.CurrentLease)
	}

	clock.value = renewed.Value.Value.CurrentLease.ExpiresAt
	expired := doJSON[application.WorkItemOperationalReadResult](
		t,
		client,
		http.MethodGet,
		workURL+"/operational-state",
		nil,
		nil,
		http.StatusOK,
	)
	if expired.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("operational state cache control = %q", expired.Header.Get("Cache-Control"))
	}
	if expired.Value.State.LeaseStatus != domain.LeaseStatusExpired ||
		expired.Value.State.DisplayState != domain.WorkItemDisplayStateAttentionNeeded {
		t.Fatalf("expired operational state = %#v", expired.Value.State)
	}

	reclaimed := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		client,
		http.MethodPost,
		workURL+"/actions/reclaim",
		[]byte(`{"lease_ttl_seconds":300}`),
		map[string]string{
			"Idempotency-Key": "wave08-work-reclaim-0001",
			"If-Match":        renewed.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	newLease := reclaimed.Value.Value.CurrentLease
	if newLease == nil ||
		newLease.ClaimID == oldClaim.ClaimID ||
		newLease.FencingToken != oldClaim.FencingToken+1 {
		t.Fatalf("reclaim did not rotate claim/fence: %#v", newLease)
	}

	stale := doRequest(
		t,
		client,
		http.MethodPost,
		workURL+"/actions/complete",
		[]byte(`{"claim_id":"`+oldClaim.ClaimID.String()+`","fencing_token":`+itoaUint64(oldClaim.FencingToken)+`,"result_summary":"stale","reason":"old worker"}`),
		map[string]string{
			"Idempotency-Key": "wave08-stale-complete-0001",
			"If-Match":        reclaimed.Header.Get("ETag"),
		},
	)
	if stale.StatusCode != http.StatusConflict {
		t.Fatalf("stale completion status = %d, want 409", stale.StatusCode)
	}
	stale.Body.Close()

	active := doJSON[application.WorkItemOperationalReadResult](
		t,
		client,
		http.MethodGet,
		workURL+"/operational-state",
		nil,
		nil,
		http.StatusOK,
	)
	if active.Value.State.LeaseStatus != domain.LeaseStatusActive ||
		active.Value.State.DisplayState != domain.WorkItemDisplayStateInProgress {
		t.Fatalf("reclaimed operational state = %#v", active.Value.State)
	}

	state := doJSON[application.OutcomeState](
		t,
		client,
		http.MethodGet,
		outcomeURL+"/state",
		nil,
		nil,
		http.StatusOK,
	)
	if len(state.Value.WorkItemOperationalStates) != 1 ||
		state.Value.WorkItemOperationalStates[0].Ref.ID != work.ID ||
		state.Value.WorkItemOperationalStates[0].LeaseStatus != domain.LeaseStatusActive {
		t.Fatalf("outcome operational projection = %#v", state.Value.WorkItemOperationalStates)
	}
}

func itoaUint64(value uint64) string {
	return fmt.Sprintf("%d", value)
}
