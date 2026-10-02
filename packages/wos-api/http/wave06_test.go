package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func TestWave06DependencyAndReadyWorkHTTPContract(t *testing.T) {
	handler := newTestHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()

	namespaceID := "0199ee00-0000-7000-8000-000000000001"
	base := server.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes"

	createdOutcome := doJSON[application.MutationResult[domain.Outcome]](
		t, client, http.MethodPost, base,
		[]byte(`{"title":"Wave 06 HTTP","desired_state":"dependency coordination works","priority":"normal"}`),
		map[string]string{"Idempotency-Key": "wave06-outcome-create-0001"},
		http.StatusCreated,
	)
	outcome := createdOutcome.Value.Value
	outcomeURL := base + "/" + outcome.ID.String()

	criterion := doJSON[application.MutationResult[domain.SuccessCriterion]](
		t, client, http.MethodPost, outcomeURL+"/criteria",
		[]byte(`{"title":"Activate","required":true,"verification_mode":"attestation"}`),
		map[string]string{
			"Idempotency-Key": "wave06-outcome-criterion-0001",
			"If-Match":        createdOutcome.Header.Get("ETag"),
		},
		http.StatusCreated,
	)
	activated := doJSON[application.MutationResult[domain.Outcome]](
		t, client, http.MethodPost, outcomeURL+"/actions/activate",
		[]byte(`{}`),
		map[string]string{
			"Idempotency-Key": "wave06-outcome-activate-0001",
			"If-Match":        criterion.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if activated.Value.Value.Lifecycle != domain.OutcomeLifecycleActive {
		t.Fatalf("outcome lifecycle = %q, want active", activated.Value.Value.Lifecycle)
	}

	createWork := func(title, key string) decodedResponse[application.MutationResult[domain.WorkItem]] {
		return doJSON[application.MutationResult[domain.WorkItem]](
			t, client, http.MethodPost, outcomeURL+"/work-items",
			[]byte(`{"title":"`+title+`","priority":"normal","lifecycle":"todo"}`),
			map[string]string{"Idempotency-Key": key},
			http.StatusCreated,
		)
	}
	targetResponse := createWork("target", "wave06-target-create-0001")
	sourceResponse := createWork("source", "wave06-source-create-0001")
	target := targetResponse.Value.Value
	source := sourceResponse.Value.Value

	relationResponse := doJSON[application.MutationResult[domain.Relation]](
		t, client, http.MethodPost, outcomeURL+"/relations",
		[]byte(`{
			"source_ref":{"kind":"work_item","id":"`+source.ID.String()+`"},
			"relation_type":"depends_on",
			"target_ref":{"kind":"work_item","id":"`+target.ID.String()+`"},
			"strength":"hard",
			"satisfaction":"target_completed",
			"reason":"source needs target"
		}`),
		map[string]string{"Idempotency-Key": "wave06-relation-create-0001"},
		http.StatusCreated,
	)
	relation := relationResponse.Value.Value
	if relation.RelationType != domain.RelationTypeDependsOn || relation.Strength != domain.DependencyStrengthHard {
		t.Fatalf("unexpected relation: %#v", relation)
	}
	if relationResponse.Header.Get("ETag") == "" {
		t.Fatal("relation create did not return ETag")
	}

	relations := doJSON[collectionResponse[domain.Relation]](
		t, client, http.MethodGet, outcomeURL+"/relations", nil, nil, http.StatusOK,
	)
	if len(relations.Value.Items) != 1 || relations.Value.Items[0].ID != relation.ID {
		t.Fatalf("relation collection = %#v", relations.Value.Items)
	}

	readRelation := doJSON[application.ReadResult[domain.Relation]](
		t, client, http.MethodGet, outcomeURL+"/relations/"+relation.ID.String(), nil, nil, http.StatusOK,
	)
	if readRelation.Value.Value.ID != relation.ID {
		t.Fatalf("read relation id = %s, want %s", readRelation.Value.Value.ID, relation.ID)
	}

	ready := doJSON[application.ReadyWork](
		t, client, http.MethodGet, outcomeURL+"/ready-work", nil, nil, http.StatusOK,
	)
	if !httpReadyContains(ready.Value.Items, target.ID) {
		t.Fatal("target should be ready")
	}
	if httpReadyContains(ready.Value.Items, source.ID) {
		t.Fatal("hard-dependent source should not be ready")
	}
	if ready.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("ready-work Cache-Control = %q, want no-store", ready.Header.Get("Cache-Control"))
	}

	removed := doJSON[application.MutationResult[domain.Relation]](
		t, client, http.MethodPost, outcomeURL+"/relations/"+relation.ID.String()+"/actions/remove",
		[]byte(`{"reason":"dependency removed"}`),
		map[string]string{
			"Idempotency-Key": "wave06-relation-remove-0001",
			"If-Match":        relationResponse.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if removed.Value.Value.Lifecycle != domain.RelationLifecycleRemoved {
		t.Fatalf("removed relation lifecycle = %q", removed.Value.Value.Lifecycle)
	}

	readyAfterRemoval := doJSON[application.ReadyWork](
		t, client, http.MethodGet, outcomeURL+"/ready-work", nil, nil, http.StatusOK,
	)
	if !httpReadyContains(readyAfterRemoval.Value.Items, source.ID) {
		t.Fatal("source should be ready after dependency removal")
	}

	state := doJSON[application.OutcomeState](
		t, client, http.MethodGet, outcomeURL+"/state", nil, nil, http.StatusOK,
	)
	if len(state.Value.Relations) != 1 || state.Value.Relations[0].Lifecycle != domain.RelationLifecycleRemoved {
		t.Fatalf("state relations = %#v", state.Value.Relations)
	}
}

func httpReadyContains(items []application.ReadyWorkItem, id domain.ID) bool {
	for _, item := range items {
		if item.WorkItem.ID == id {
			return true
		}
	}
	return false
}
