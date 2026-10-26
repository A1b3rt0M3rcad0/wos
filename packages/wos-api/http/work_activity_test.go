package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWorkActivityReadHTTPBoundaries(t *testing.T) {
	server := httptest.NewServer(newTestHandler(t))
	defer server.Close()
	client := server.Client()
	base := server.URL + "/api/v1/namespaces/0199ec00-0000-7000-8000-000000000001/outcomes"
	outcome := doJSON[application.MutationResult[domain.Outcome]](t, client, http.MethodPost, base,
		[]byte(`{"title":"Activity read boundary","desired_state":"Scoped observations","priority":"normal"}`),
		map[string]string{"Idempotency-Key": "activity-http-outcome"}, http.StatusCreated).Value.Value
	path := base + "/" + outcome.ID.String() + "/work-activity"
	objective := createHTTPTestObjective(t, client, base+"/"+outcome.ID.String(), "activity-http-owner")
	page := doJSON[application.WorkActivityPage](t, client, http.MethodGet, path, nil, nil, http.StatusOK)
	if len(page.Value.Items) != 0 || page.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("empty activity read contract: %+v", page)
	}
	for _, query := range []string{"?limit=0", "?limit=101", "?cursor=malformed", "?descending=maybe", "?descending=1", "?descending="} {
		response := doRequest(t, client, http.MethodGet, path+query, nil, nil)
		response.Body.Close()
		if response.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s returned %d", query, response.StatusCode)
		}
	}
	for _, query := range []string{"?work_item_id=malformed", "?contract_id=malformed"} {
		response := doRequest(t, client, http.MethodGet, path+query, nil, nil)
		response.Body.Close()
		if response.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s returned %d", query, response.StatusCode)
		}
	}
	work := doJSON[application.MutationResult[domain.WorkItem]](t, client, http.MethodPost, base+"/"+outcome.ID.String()+"/work-items",
		testWorkItemBody(objective.ID, "Inspect attention", "todo"),
		map[string]string{"Idempotency-Key": "activity-http-task"}, http.StatusCreated).Value.Value
	attentionPath := base + "/" + outcome.ID.String() + "/work-items/" + work.ID.String() + "/attention"
	attention := doJSON[application.WorkAttention](t, client, http.MethodGet, attentionPath, nil, nil, http.StatusOK)
	if attention.Value.WorkItemID != work.ID || attention.Header.Get("Cache-Control") != "no-store" || attention.Value.ExecutionAllowed {
		t.Fatalf("attention read unexpectedly grants authority: %+v", attention)
	}
}
