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

func newWave09TestHandler(t *testing.T) *Handler {
	t.Helper()
	store := memory.New()
	service, err := application.NewService(
		store,
		fixedClock{now: time.Date(2026, 10, 2, 21, 0, 0, 0, time.UTC)},
		&sequenceIDs{prefix: "0199eda0", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	auth, err := local.New("http-human")
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(Options{
		Prefix: "/api/v1", RequestTimeout: 2 * time.Second, Service: service,
		IDs: &sequenceIDs{prefix: "0199eda1", next: 1}, LocalAuth: auth,
	})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestWave09DocumentaryHTTPContract(t *testing.T) {
	handler := newWave09TestHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()

	namespaceID := "0199eda2-0000-7000-8000-000000000001"
	base := server.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes"
	created := doJSON[application.MutationResult[domain.Outcome]](
		t, client, http.MethodPost, base,
		[]byte(`{"title":"Wave 09 HTTP","desired_state":"documentary continuity","priority":"normal"}`),
		map[string]string{"Idempotency-Key": "wave09-outcome-create-0001"},
		http.StatusCreated,
	)
	outcome := created.Value.Value
	outcomeURL := base + "/" + outcome.ID.String()

	artifactResponse := doJSON[application.MutationResult[domain.Artifact]](
		t, client, http.MethodPost, outcomeURL+"/artifacts",
		[]byte(`{"artifact_type":"document","name":"report","uri":"file:///report.pdf","media_type":"application/pdf","checksum":"sha256:abc"}`),
		map[string]string{"Idempotency-Key": "wave09-artifact-create-0001"},
		http.StatusCreated,
	)
	artifact := artifactResponse.Value.Value
	if artifactResponse.Header.Get("ETag") == "" {
		t.Fatal("artifact response omitted ETag")
	}

	evidenceResponse := doJSON[application.MutationResult[domain.Evidence]](
		t, client, http.MethodPost, outcomeURL+"/evidence",
		[]byte(`{"evidence_type":"source","description":"report observation","source_ref":{"provider":"test","uri":"file:///report.pdf"},"captured_at":"2026-10-02T21:00:00Z","artifact_id":"`+artifact.ID.String()+`","checksum":"sha256:abc"}`),
		map[string]string{"Idempotency-Key": "wave09-evidence-create-0001"},
		http.StatusCreated,
	)
	evidence := evidenceResponse.Value.Value

	decisionResponse := doJSON[application.MutationResult[domain.Decision]](
		t, client, http.MethodPost, outcomeURL+"/decisions",
		[]byte(`{"title":"Storage","proposal":"Use SQLite","alternatives":["SQLite","PostgreSQL"],"chosen_alternative":"SQLite","rationale":"local-first"}`),
		map[string]string{"Idempotency-Key": "wave09-decision-propose-0001"},
		http.StatusCreated,
	)
	decision := decisionResponse.Value.Value
	decisionURL := outcomeURL + "/decisions/" + decision.ID.String()

	accepted := doJSON[application.MutationResult[domain.Decision]](
		t, client, http.MethodPost, decisionURL+"/actions/accept", []byte(`{}`),
		map[string]string{
			"Idempotency-Key": "wave09-decision-accept-0001",
			"If-Match": decisionResponse.Header.Get("ETag"),
		},
		http.StatusOK,
	)

	linkResponse := doJSON[application.MutationResult[domain.EvidenceLink]](
		t, client, http.MethodPost, outcomeURL+"/evidence-links",
		[]byte(`{"evidence_id":"`+evidence.ID.String()+`","target_ref":{"kind":"decision","id":"`+decision.ID.String()+`"},"stance":"supports","rationale":"report supports the accepted choice"}`),
		map[string]string{"Idempotency-Key": "wave09-link-create-0001"},
		http.StatusCreated,
	)
	if linkResponse.Value.Value.Stance != domain.EvidenceStanceSupports {
		t.Fatalf("link stance = %q", linkResponse.Value.Value.Stance)
	}

	immutable := doRequest(
		t, client, http.MethodPatch, decisionURL,
		[]byte(`{"title":"mutated"}`),
		map[string]string{
			"Idempotency-Key": "wave09-decision-edit-0001",
			"If-Match": accepted.Header.Get("ETag"),
		},
	)
	if immutable.StatusCode != http.StatusConflict {
		t.Fatalf("accepted decision edit status = %d, want 409", immutable.StatusCode)
	}
	immutable.Body.Close()

	superseded := doJSON[application.MutationResult[application.DecisionSupersessionResult]](
		t, client, http.MethodPost, decisionURL+"/actions/supersede",
		[]byte(`{"title":"Storage v2","proposal":"Use PostgreSQL","alternatives":["SQLite","PostgreSQL"],"chosen_alternative":"PostgreSQL","rationale":"remote multi-user operation"}`),
		map[string]string{
			"Idempotency-Key": "wave09-decision-supersede-0001",
			"If-Match": accepted.Header.Get("ETag"),
		},
		http.StatusCreated,
	)
	if superseded.Value.Value.Predecessor.Lifecycle != domain.DecisionLifecycleSuperseded ||
		superseded.Value.Value.Successor.Lifecycle != domain.DecisionLifecycleAccepted {
		t.Fatalf("supersession response = %#v", superseded.Value.Value)
	}

	listed := doJSON[collectionResponse[domain.Decision]](
		t, client, http.MethodGet, outcomeURL+"/decisions", nil, nil, http.StatusOK,
	)
	if len(listed.Value.Items) != 2 {
		t.Fatalf("decision count = %d, want 2", len(listed.Value.Items))
	}

	retracted := doJSON[application.MutationResult[domain.Evidence]](
		t, client, http.MethodPost, outcomeURL+"/evidence/"+evidence.ID.String()+"/actions/retract",
		[]byte(`{"reason":"source replaced"}`),
		map[string]string{
			"Idempotency-Key": "wave09-evidence-retract-0001",
			"If-Match": evidenceResponse.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if retracted.Value.Value.Description != evidence.Description ||
		retracted.Value.Value.Lifecycle != domain.EvidenceLifecycleRetracted {
		t.Fatalf("retracted evidence = %#v", retracted.Value.Value)
	}
}
