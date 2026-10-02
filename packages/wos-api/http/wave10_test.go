package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWave10HTTPRecordsEvidenceAwareAssessment(t *testing.T) {
	handler := newWave09TestHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()

	namespaceID := "0199efa0-0000-7000-8000-000000000001"
	base := server.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes"
	created := doJSON[application.MutationResult[domain.Outcome]](
		t, client, http.MethodPost, base,
		[]byte(`{"title":"Wave 10 HTTP","desired_state":"verifiable conclusion","priority":"normal"}`),
		map[string]string{"Idempotency-Key": "wave10-outcome-create-0001"},
		http.StatusCreated,
	)
	outcome := created.Value.Value
	outcomeURL := base + "/" + outcome.ID.String()

	criterionResponse := doJSON[application.MutationResult[domain.SuccessCriterion]](
		t, client, http.MethodPost, outcomeURL+"/criteria",
		[]byte(`{"title":"Reviewed source","required":true,"verification_mode":"evidence_review"}`),
		map[string]string{
			"Idempotency-Key": "wave10-criterion-create-0001",
			"If-Match":        created.Header.Get("ETag"),
		},
		http.StatusCreated,
	)
	criterion := criterionResponse.Value.Value

	evidenceResponse := doJSON[application.MutationResult[domain.Evidence]](
		t, client, http.MethodPost, outcomeURL+"/evidence",
		[]byte(`{"evidence_type":"source","description":"review input","source_ref":{"provider":"http-test","id":"source-1"},"captured_at":"2026-10-02T21:00:00Z"}`),
		map[string]string{"Idempotency-Key": "wave10-evidence-create-0001"},
		http.StatusCreated,
	)
	evidence := evidenceResponse.Value.Value

	assessment := doJSON[application.MutationResult[domain.CriterionAssessment]](
		t, client, http.MethodPost,
		outcomeURL+"/criteria/"+criterion.ID.String()+"/assessments",
		[]byte(`{"criterion_revision":1,"result":"met","rationale":"reviewed source","evidence_ids":["`+evidence.ID.String()+`"]}`),
		map[string]string{
			"Idempotency-Key": "wave10-assessment-create-0001",
			"If-Match":        criterionResponse.Header.Get("ETag"),
		},
		http.StatusCreated,
	)
	if len(assessment.Value.Value.EvidenceIDs) != 1 ||
		assessment.Value.Value.EvidenceIDs[0] != evidence.ID {
		t.Fatalf("assessment evidence ids = %#v", assessment.Value.Value.EvidenceIDs)
	}
	if assessment.Header.Get("ETag") == "" {
		t.Fatal("assessment response omitted owner ETag")
	}
}

func TestWave10HTTPReadsValidationAndConclusionHistory(t *testing.T) {
	handler := newWave09TestHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()

	namespaceID := "0199efc0-0000-7000-8000-000000000001"
	base := server.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes"
	created := doJSON[application.MutationResult[domain.Outcome]](
		t, client, http.MethodPost, base,
		[]byte(`{"title":"Wave 10 history","desired_state":"validation history is addressable","priority":"normal"}`),
		map[string]string{"Idempotency-Key": "wave10-history-outcome-0001"},
		http.StatusCreated,
	)
	outcome := created.Value.Value
	outcomeURL := base + "/" + outcome.ID.String()

	criterionResponse := doJSON[application.MutationResult[domain.SuccessCriterion]](
		t, client, http.MethodPost, outcomeURL+"/criteria",
		[]byte(`{"title":"Verified","required":true,"verification_mode":"attestation"}`),
		map[string]string{
			"Idempotency-Key": "wave10-history-criterion-0001",
			"If-Match":        created.Header.Get("ETag"),
		},
		http.StatusCreated,
	)
	criterion := criterionResponse.Value.Value
	activated := doJSON[application.MutationResult[domain.Outcome]](
		t, client, http.MethodPost, outcomeURL+"/actions/activate",
		[]byte(`{}`),
		map[string]string{
			"Idempotency-Key": "wave10-history-activate-0001",
			"If-Match":        criterionResponse.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	assessment := doJSON[application.MutationResult[domain.CriterionAssessment]](
		t, client, http.MethodPost,
		outcomeURL+"/criteria/"+criterion.ID.String()+"/assessments",
		[]byte(`{"criterion_revision":1,"result":"met","rationale":"verified"}`),
		map[string]string{
			"Idempotency-Key": "wave10-history-assessment-0001",
			"If-Match":        activated.Header.Get("ETag"),
		},
		http.StatusCreated,
	)
	achieved := doJSON[application.MutationResult[domain.Outcome]](
		t, client, http.MethodPost, outcomeURL+"/actions/achieve",
		[]byte(`{"reason":"all requirements verified"}`),
		map[string]string{
			"Idempotency-Key": "wave10-history-achieve-0001",
			"If-Match":        assessment.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	conclusion := achieved.Value.Value.CurrentConclusion
	if conclusion == nil || conclusion.ID.IsZero() {
		t.Fatalf("achieved conclusion = %#v", conclusion)
	}

	history := doJSON[application.ReadResult[application.CriterionValidationHistory]](
		t, client, http.MethodGet,
		outcomeURL+"/criteria/"+criterion.ID.String()+"/history",
		nil, nil, http.StatusOK,
	)
	if len(history.Value.Value.DefinitionRevisions) != 1 ||
		len(history.Value.Value.Assessments) != 1 ||
		history.Value.Value.CurrentAssessment == nil {
		t.Fatalf("criterion history = %#v", history.Value.Value)
	}

	conclusions := doJSON[application.ReadResult[application.ConclusionHistory]](
		t, client, http.MethodGet, outcomeURL+"/conclusions",
		nil, nil, http.StatusOK,
	)
	if conclusions.Value.Value.Current == nil ||
		conclusions.Value.Value.Current.ID != conclusion.ID {
		t.Fatalf("conclusion history = %#v", conclusions.Value.Value)
	}

	fetched := doJSON[application.ReadResult[domain.Conclusion]](
		t, client, http.MethodGet,
		outcomeURL+"/conclusions/"+conclusion.ID.String(),
		nil, nil, http.StatusOK,
	)
	if fetched.Value.Value.ID != conclusion.ID {
		t.Fatalf("fetched conclusion id = %s, want %s", fetched.Value.Value.ID, conclusion.ID)
	}
}
