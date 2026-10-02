package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWave11HTTPPublishesAndActivatesRoadmapRevision(t *testing.T) {
	handler := newWave09TestHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()

	namespaceID := "0199f500-0000-7000-8000-000000000001"
	base := server.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes"
	created := doJSON[application.MutationResult[domain.Outcome]](
		t, client, http.MethodPost, base,
		[]byte(`{"title":"Wave 11 HTTP","desired_state":"published plan","priority":"normal"}`),
		map[string]string{"Idempotency-Key": "wave11-outcome-create-0001"},
		http.StatusCreated,
	)
	outcome := created.Value.Value
	outcomeURL := base + "/" + outcome.ID.String()

	roadmap := doJSON[application.MutationResult[domain.Roadmap]](
		t, client, http.MethodPost, outcomeURL+"/roadmaps",
		[]byte(`{"plan_scope":{"kind":"outcome","id":"`+outcome.ID.String()+`"},"title":"Main plan"}`),
		map[string]string{"Idempotency-Key": "wave11-roadmap-create-0001"},
		http.StatusCreated,
	)
	roadmapID := roadmap.Value.Value.ID
	roadmapURL := outcomeURL + "/roadmaps/" + roadmapID.String()

	opened := doJSON[application.MutationResult[domain.Roadmap]](
		t, client, http.MethodPost, roadmapURL+"/draft/actions/open",
		[]byte(`{}`),
		map[string]string{
			"Idempotency-Key": "wave11-roadmap-open-draft-0001",
			"If-Match":        roadmap.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	edited := doJSON[application.MutationResult[domain.Roadmap]](
		t, client, http.MethodPut, roadmapURL+"/draft",
		[]byte(`{"expected_draft_version":1,"nodes":[{"node_key":"phase-1","node_type":"phase","title":"Phase 1","position":0}]}`),
		map[string]string{
			"Idempotency-Key": "wave11-roadmap-edit-draft-0001",
			"If-Match":        opened.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	published := doJSON[application.MutationResult[domain.Roadmap]](
		t, client, http.MethodPost, roadmapURL+"/draft/actions/publish",
		[]byte(`{"expected_draft_version":2}`),
		map[string]string{
			"Idempotency-Key": "wave11-roadmap-publish-0001",
			"If-Match":        edited.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if len(published.Value.Value.Revisions) != 1 ||
		published.Value.Value.Revisions[0].ContentHash == "" {
		t.Fatalf("published roadmap = %#v", published.Value.Value)
	}

	revision := doJSON[application.ReadResult[domain.RoadmapRevision]](
		t, client, http.MethodGet, roadmapURL+"/revisions/1",
		nil, nil, http.StatusOK,
	)
	if revision.Value.Value.RevisionNumber != 1 {
		t.Fatalf("revision number = %d", revision.Value.Value.RevisionNumber)
	}

	activated := doJSON[application.MutationResult[domain.RoadmapActiveSlot]](
		t, client, http.MethodPost, roadmapURL+"/revisions/1/actions/activate",
		[]byte(`{}`),
		map[string]string{
			"Idempotency-Key": "wave11-roadmap-activate-0001",
			"If-Match":        published.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if activated.Value.Value.RoadmapID != roadmapID {
		t.Fatalf("activated slot = %#v", activated.Value.Value)
	}

	slotURL := outcomeURL + "/roadmap-slots/outcome/" + outcome.ID.String()
	slot := doJSON[roadmapSlotReadResponse](
		t, client, http.MethodGet, slotURL,
		nil, nil, http.StatusOK,
	)
	if slot.Value.Value == nil ||
		slot.Value.Value.RoadmapID != roadmapID ||
		slot.Value.Value.RevisionNumber != 1 {
		t.Fatalf("slot response = %#v", slot.Value)
	}
	history := doJSON[collectionResponse[domain.RoadmapActivationRecord]](
		t, client, http.MethodGet, slotURL+"/history",
		nil, nil, http.StatusOK,
	)
	if len(history.Value.Items) != 1 ||
		history.Value.Items[0].Action != domain.RoadmapActivationActivated {
		t.Fatalf("activation history = %#v", history.Value.Items)
	}
}
