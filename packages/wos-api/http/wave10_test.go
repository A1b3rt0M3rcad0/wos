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
