package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/authentication/local"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func newAdministrativeOverrideHandler(t *testing.T, allow bool) *Handler {
	t.Helper()
	store := memory.New()
	auth, err := local.New("local-admin")
	if err != nil {
		t.Fatal(err)
	}
	authorizer, err := local.NewAuthorizer("local-admin", allow)
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewServiceWithAuthorizer(
		store,
		fixedClock{value: time.Date(2026, 10, 3, 4, 0, 0, 0, time.UTC)},
		&sequenceIDs{prefix: "0199ed80", next: 1},
		authorizer,
	)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(Options{
		Prefix:         "/api/v1",
		RequestTimeout: 2 * time.Second,
		Service:        service,
		IDs:            &sequenceIDs{prefix: "0199ed81", next: 1},
		LocalAuth:      auth,
	})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestAdministrativeOverrideHTTPRequiresExplicitLocalOptIn(t *testing.T) {
	deniedServer := httptest.NewServer(newAdministrativeOverrideHandler(t, false))
	defer deniedServer.Close()
	deniedClient := deniedServer.Client()

	namespaceID := "0199ed82-0000-7000-8000-000000000001"
	base := deniedServer.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes"
	created := doJSON[application.MutationResult[domain.Outcome]](
		t,
		deniedClient,
		http.MethodPost,
		base,
		[]byte(`{"title":"Admin override denied","desired_state":"explicit authorization","priority":"normal"}`),
		map[string]string{"Idempotency-Key": "admin-denied-outcome-0001"},
		http.StatusCreated,
	)
	outcomeURL := base + "/" + created.Value.Value.ID.String()
	criterion := doJSON[application.MutationResult[domain.SuccessCriterion]](
		t,
		deniedClient,
		http.MethodPost,
		outcomeURL+"/criteria",
		[]byte(`{"title":"Activate","required":true,"verification_mode":"attestation"}`),
		map[string]string{
			"Idempotency-Key": "admin-denied-criterion-0001",
			"If-Match":        created.Header.Get("ETag"),
		},
		http.StatusCreated,
	)
	doJSON[application.MutationResult[domain.Outcome]](
		t,
		deniedClient,
		http.MethodPost,
		outcomeURL+"/actions/activate",
		[]byte(`{}`),
		map[string]string{
			"Idempotency-Key": "admin-denied-activate-0001",
			"If-Match":        criterion.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	workResponse := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		deniedClient,
		http.MethodPost,
		outcomeURL+"/work-items",
		[]byte(`{"title":"Leased work","priority":"normal","lifecycle":"todo"}`),
		map[string]string{"Idempotency-Key": "admin-denied-work-0001"},
		http.StatusCreated,
	)
	workURL := outcomeURL + "/work-items/" + workResponse.Value.Value.ID.String()
	claimed := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		deniedClient,
		http.MethodPost,
		workURL+"/actions/claim",
		[]byte(`{"lease_ttl_seconds":300}`),
		map[string]string{
			"Idempotency-Key": "admin-denied-claim-0001",
			"If-Match":        workResponse.Header.Get("ETag"),
		},
		http.StatusOK,
	)

	denied := doRequest(
		t,
		deniedClient,
		http.MethodPost,
		workURL+"/actions/admin-cancel",
		[]byte(`{"reason":"operator override"}`),
		map[string]string{
			"Idempotency-Key": "admin-denied-cancel-0001",
			"If-Match":        claimed.Header.Get("ETag"),
		},
	)
	if denied.StatusCode != http.StatusForbidden {
		t.Fatalf("disabled admin override status = %d, want 403", denied.StatusCode)
	}
	denied.Body.Close()
}

func TestAdministrativeOverrideHTTPCompletesLeasedWorkWhenEnabled(t *testing.T) {
	server := httptest.NewServer(newAdministrativeOverrideHandler(t, true))
	defer server.Close()
	client := server.Client()

	namespaceID := "0199ed83-0000-7000-8000-000000000001"
	base := server.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes"
	created := doJSON[application.MutationResult[domain.Outcome]](
		t,
		client,
		http.MethodPost,
		base,
		[]byte(`{"title":"Admin override enabled","desired_state":"authorized intervention","priority":"normal"}`),
		map[string]string{"Idempotency-Key": "admin-enabled-outcome-0001"},
		http.StatusCreated,
	)
	outcomeURL := base + "/" + created.Value.Value.ID.String()
	criterion := doJSON[application.MutationResult[domain.SuccessCriterion]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/criteria",
		[]byte(`{"title":"Activate","required":true,"verification_mode":"attestation"}`),
		map[string]string{
			"Idempotency-Key": "admin-enabled-criterion-0001",
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
			"Idempotency-Key": "admin-enabled-activate-0001",
			"If-Match":        criterion.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	workResponse := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/work-items",
		[]byte(`{"title":"Externally completed work","priority":"normal","lifecycle":"todo"}`),
		map[string]string{"Idempotency-Key": "admin-enabled-work-0001"},
		http.StatusCreated,
	)
	workURL := outcomeURL + "/work-items/" + workResponse.Value.Value.ID.String()
	claimed := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		client,
		http.MethodPost,
		workURL+"/actions/claim",
		[]byte(`{"lease_ttl_seconds":300}`),
		map[string]string{
			"Idempotency-Key": "admin-enabled-claim-0001",
			"If-Match":        workResponse.Header.Get("ETag"),
		},
		http.StatusOK,
	)

	completed := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		client,
		http.MethodPost,
		workURL+"/actions/admin-complete",
		[]byte(`{"result_summary":"operator verified external result","reason":"worker unavailable after producing result"}`),
		map[string]string{
			"Idempotency-Key": "admin-enabled-complete-0001",
			"If-Match":        claimed.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if completed.Value.Value.Lifecycle != domain.WorkItemLifecycleDone ||
		completed.Value.Value.CurrentLease != nil {
		t.Fatalf("admin completion = %#v", completed.Value.Value)
	}
}
