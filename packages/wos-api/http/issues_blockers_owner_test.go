package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIssueHTTPCreateRejectsLegacyPayloadWithoutObjectiveOwner(t *testing.T) {
	handler := newTestHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()

	namespaceID := "0199fa28-0000-7000-8000-000000000001"
	outcomeID := "0199fa28-0000-7000-8000-000000000002"
	response := doJSON[errorEnvelope](
		t,
		server.Client(),
		http.MethodPost,
		server.URL+"/api/v1/namespaces/"+namespaceID+"/outcomes/"+outcomeID+"/issues",
		[]byte(`{"title":"An Issue without an owner","severity":"major"}`),
		map[string]string{"Idempotency-Key": "issue-create-missing-objective-0001"},
		http.StatusUnprocessableEntity,
	)

	if response.Value.Error.Code != "invalid_argument" {
		t.Fatalf("error code = %q, want invalid_argument", response.Value.Error.Code)
	}
	if response.Value.Error.Message != "invalid_argument: objective_id is required and must be a valid ID" {
		t.Fatalf("error message = %q, want missing Objective owner guidance", response.Value.Error.Message)
	}
}
