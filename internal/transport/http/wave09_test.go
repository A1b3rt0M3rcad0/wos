package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func TestWave09DocumentaryHTTPContract(t *testing.T) {
	handler := newTestHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()

	namespaceID := "0199ed90-0000-7000-8000-000000000001"
	base := server.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes"
	created := doJSON[application.MutationResult[domain.Outcome]](
		t,
		client,
		http.MethodPost,
		base,
		[]byte(`{"title":"Wave 09 HTTP","desired_state":"facts and choices are explicit","priority":"normal"}`),
		map[string]string{"Idempotency-Key": "wave09-outcome-create-0001"},
		http.StatusCreated,
	)
	outcome := created.Value.Value
	outcomeURL := base + "/" + outcome.ID.String()

	artifact := doJSON[application.MutationResult[domain.Artifact]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/artifacts",
		[]byte(`{
			"artifact_type":"report",
			"name":"benchmark report",
			"uri":"https://example.test/report",
			"media_type":"application/json",
			"checksum":"sha256:artifact",
			"source_version":"commit-42",
			"producer_ref":{"kind":"service","provider":"ci","id":"benchmark"}
		}`),
		map[string]string{"Idempotency-Key": "wave09-artifact-create-0001"},
		http.StatusCreated,
	)
	if artifact.Header.Get("ETag") == "" || artifact.Header.Get("Location") == "" {
		t.Fatal("artifact response did not include ETag/Location")
	}

	evidence := doJSON[application.MutationResult[domain.Evidence]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/evidence",
		[]byte(`{
			"evidence_type":"measurement",
			"description":"p95 latency was 180ms",
			"source_ref":{"provider":"benchmark-ci","id":"run-42"},
			"producer_ref":{"kind":"service","provider":"ci","id":"benchmark"},
			"captured_at":"2026-10-02T16:00:00Z",
			"artifact_id":"`+artifact.Value.Value.ID.String()+`",
			"measurement":{"value":180,"unit":"ms","method":"p95","conditions":{"requests":10000}},
			"source_version":"scenario-c1",
			"checksum":"sha256:evidence"
		}`),
		map[string]string{"Idempotency-Key": "wave09-evidence-create-0001"},
		http.StatusCreated,
	)

	link := doJSON[application.MutationResult[domain.EvidenceLink]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/evidence-links",
		[]byte(`{
			"evidence_id":"`+evidence.Value.Value.ID.String()+`",
			"target_ref":{"kind":"outcome","id":"`+outcome.ID.String()+`"},
			"stance":"supports",
			"rationale":"benchmark supports the current outcome"
		}`),
		map[string]string{"Idempotency-Key": "wave09-evidence-link-create-0001"},
		http.StatusCreated,
	)

	proposed := doJSON[application.MutationResult[domain.Decision]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/decisions",
		[]byte(`{
			"title":"Storage engine",
			"proposal":"Choose durable storage",
			"alternatives":["SQLite","PostgreSQL"]
		}`),
		map[string]string{"Idempotency-Key": "wave09-decision-propose-0001"},
		http.StatusCreated,
	)
	decisionURL := outcomeURL + "/decisions/" + proposed.Value.Value.ID.String()

	revised := doJSON[application.MutationResult[domain.Decision]](
		t,
		client,
		http.MethodPatch,
		decisionURL,
		[]byte(`{
			"title":"Storage engine",
			"proposal":"Choose durable storage for standalone release",
			"rationale":"initial deployment profile",
			"alternatives":["SQLite","PostgreSQL"]
		}`),
		map[string]string{
			"Idempotency-Key": "wave09-decision-revise-0001",
			"If-Match":        proposed.Header.Get("ETag"),
		},
		http.StatusOK,
	)

	accepted := doJSON[application.MutationResult[domain.Decision]](
		t,
		client,
		http.MethodPost,
		decisionURL+"/actions/accept",
		[]byte(`{"chosen_alternative":"SQLite","rationale":"standalone-first"}`),
		map[string]string{
			"Idempotency-Key": "wave09-decision-accept-0001",
			"If-Match":        revised.Header.Get("ETag"),
		},
		http.StatusOK,
	)

	superseded := doJSON[application.MutationResult[application.SupersedeDecisionResult]](
		t,
		client,
		http.MethodPost,
		decisionURL+"/actions/supersede",
		[]byte(`{
			"title":"Storage engine after scale-out",
			"proposal":"Choose shared durable storage",
			"alternatives":["SQLite","PostgreSQL"],
			"chosen_alternative":"PostgreSQL",
			"rationale":"multi-node requires shared durability"
		}`),
		map[string]string{
			"Idempotency-Key": "wave09-decision-supersede-0001",
			"If-Match":        accepted.Header.Get("ETag"),
		},
		http.StatusCreated,
	)
	if superseded.Value.Value.Superseded.Lifecycle != domain.DecisionLifecycleSuperseded ||
		superseded.Value.Value.Successor.Lifecycle != domain.DecisionLifecycleAccepted {
		t.Fatalf("supersession response = %#v", superseded.Value.Value)
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
	if len(state.Value.Documentary.Artifacts) != 1 ||
		len(state.Value.Documentary.RegisteredEvidence) != 1 ||
		len(state.Value.Documentary.ActiveEvidenceLinks) != 1 ||
		len(state.Value.Documentary.CurrentDecisions) != 1 {
		t.Fatalf("documentary state = %#v", state.Value.Documentary)
	}
	if state.Value.Documentary.CurrentDecisions[0].ID != superseded.Value.Value.Successor.ID {
		t.Fatalf("current decision = %#v", state.Value.Documentary.CurrentDecisions)
	}

	decisionList := doJSON[collectionResponse[domain.Decision]](
		t,
		client,
		http.MethodGet,
		outcomeURL+"/decisions",
		nil,
		nil,
		http.StatusOK,
	)
	if len(decisionList.Value.Items) != 2 {
		t.Fatalf("decision list count = %d, want 2", len(decisionList.Value.Items))
	}

	retractedLink := doJSON[application.MutationResult[domain.EvidenceLink]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/evidence-links/"+link.Value.Value.ID.String()+"/actions/retract",
		[]byte(`{}`),
		map[string]string{
			"Idempotency-Key": "wave09-evidence-link-retract-0001",
			"If-Match":        link.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if retractedLink.Value.Value.Lifecycle != domain.EvidenceLinkLifecycleRetracted {
		t.Fatalf("retracted link lifecycle = %q", retractedLink.Value.Value.Lifecycle)
	}

	retractedEvidence := doJSON[application.MutationResult[domain.Evidence]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/evidence/"+evidence.Value.Value.ID.String()+"/actions/retract",
		[]byte(`{"reason":"benchmark invalidated"}`),
		map[string]string{
			"Idempotency-Key": "wave09-evidence-retract-0001",
			"If-Match":        evidence.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if retractedEvidence.Value.Value.Lifecycle != domain.EvidenceLifecycleRetracted {
		t.Fatalf("retracted evidence lifecycle = %q", retractedEvidence.Value.Value.Lifecycle)
	}

	withdrawn := doJSON[application.MutationResult[domain.Artifact]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/artifacts/"+artifact.Value.Value.ID.String()+"/actions/withdraw",
		[]byte(`{"reason":"replaced report"}`),
		map[string]string{
			"Idempotency-Key": "wave09-artifact-withdraw-0001",
			"If-Match":        artifact.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if withdrawn.Value.Value.Lifecycle != domain.ArtifactLifecycleWithdrawn {
		t.Fatalf("withdrawn artifact lifecycle = %q", withdrawn.Value.Value.Lifecycle)
	}
}
