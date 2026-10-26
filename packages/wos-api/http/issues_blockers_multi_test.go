package httptransport

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestReportIssueWithBlockersHTTPRequiresFreshOutcomeRevisionAndIsAtomic(t *testing.T) {
	handler := newTestHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()

	const namespaceID = "0199ef00-0000-7000-8000-000000000001"
	base := server.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes"
	createdOutcome := doJSON[application.MutationResult[domain.Outcome]](
		t, client, http.MethodPost, base,
		[]byte(`{"title":"Reviewed multi-impact","desired_state":"review every affected target before creating blockers","priority":"normal"}`),
		map[string]string{"Idempotency-Key": "multi-impact-outcome-0001"}, http.StatusCreated,
	)
	outcomeURL := base + "/" + createdOutcome.Value.Value.ID.String()
	owner := createHTTPTestObjective(t, client, outcomeURL, "multi-impact-objective-0001")
	criterion := doJSON[application.MutationResult[domain.SuccessCriterion]](
		t, client, http.MethodPost, outcomeURL+"/criteria",
		[]byte(`{"title":"Outcome is active","required":true,"verification_mode":"attestation"}`),
		map[string]string{"Idempotency-Key": "multi-impact-criterion-0001", "If-Match": createdOutcome.Header.Get("ETag")},
		http.StatusCreated,
	)
	doJSON[application.MutationResult[domain.Outcome]](
		t, client, http.MethodPost, outcomeURL+"/actions/activate", []byte(`{}`),
		map[string]string{"Idempotency-Key": "multi-impact-activate-0001", "If-Match": criterion.Header.Get("ETag")},
		http.StatusOK,
	)
	first := doJSON[application.MutationResult[domain.WorkItem]](
		t, client, http.MethodPost, outcomeURL+"/work-items", testWorkItemBody(owner.ID, "First affected task", "todo"),
		map[string]string{"Idempotency-Key": "multi-impact-work-0001"}, http.StatusCreated,
	)
	second := doJSON[application.MutationResult[domain.WorkItem]](
		t, client, http.MethodPost, outcomeURL+"/work-items", testWorkItemBody(owner.ID, "Second affected task", "todo"),
		map[string]string{"Idempotency-Key": "multi-impact-work-0002"}, http.StatusCreated,
	)

	state := doJSON[application.OutcomeState](t, client, http.MethodGet, outcomeURL+"/state", nil, nil, http.StatusOK)
	initialRevision := state.Value.OutcomeRevision
	payload := func(revision *domain.OutcomeRevision, missingTarget bool) []byte {
		revisionField := ""
		if revision != nil {
			revisionField = fmt.Sprintf(`"observed_outcome_revision":%d,`, *revision)
		}
		secondID := second.Value.Value.ID.String()
		if missingTarget {
			secondID = "0199ef00-0000-7000-8000-000000000099"
		}
		return []byte(fmt.Sprintf(`{%s"issue":{"objective_id":%q,"title":"Shared infrastructure outage","description":"One outage impacts more than one target.","severity":"critical","affected_refs":[{"kind":"work_item","id":%q},{"kind":"work_item","id":%q}]},"blockers":[{"blocked_ref":{"kind":"work_item","id":%q},"description":"First target is blocked."},{"blocked_ref":{"kind":"work_item","id":%q},"description":"Second target is blocked."}]}`,
			revisionField, owner.ID.String(), first.Value.Value.ID.String(), second.Value.Value.ID.String(), first.Value.Value.ID.String(), secondID))
	}
	url := outcomeURL + "/issues/actions/report-with-blockers"

	missing := doRequest(t, client, http.MethodPost, url, payload(nil, false), map[string]string{"Idempotency-Key": "multi-impact-missing-rev-0001"})
	if missing.StatusCode != http.StatusPreconditionRequired {
		t.Fatalf("missing observed revision status = %d, want 428", missing.StatusCode)
	}
	missing.Body.Close()
	afterMissing := doJSON[application.OutcomeState](t, client, http.MethodGet, outcomeURL+"/state", nil, nil, http.StatusOK)
	if afterMissing.Value.OutcomeRevision != initialRevision || len(afterMissing.Value.Issues) != 0 || len(afterMissing.Value.Blockers) != 0 {
		t.Fatalf("missing revision mutated state: revision %d→%d, issues/blockers %d/%d", initialRevision, afterMissing.Value.OutcomeRevision, len(afterMissing.Value.Issues), len(afterMissing.Value.Blockers))
	}

	// Change the Outcome after the caller's review. The stale reviewed revision
	// must fail under the application's Outcome lock without partial records.
	doJSON[application.MutationResult[domain.WorkItem]](
		t, client, http.MethodPost, outcomeURL+"/work-items", testWorkItemBody(owner.ID, "Concurrent target", "todo"),
		map[string]string{"Idempotency-Key": "multi-impact-work-0003"}, http.StatusCreated,
	)
	stale := doRequest(t, client, http.MethodPost, url, payload(&initialRevision, false), map[string]string{"Idempotency-Key": "multi-impact-stale-rev-0001"})
	if stale.StatusCode != http.StatusConflict {
		t.Fatalf("stale observed revision status = %d, want 409", stale.StatusCode)
	}
	stale.Body.Close()
	afterStale := doJSON[application.OutcomeState](t, client, http.MethodGet, outcomeURL+"/state", nil, nil, http.StatusOK)
	if afterStale.Value.OutcomeRevision != afterMissing.Value.OutcomeRevision+1 || len(afterStale.Value.Issues) != 0 || len(afterStale.Value.Blockers) != 0 {
		t.Fatalf("stale revision left compound records or changed revision unexpectedly: revision=%d issues/blockers=%d/%d", afterStale.Value.OutcomeRevision, len(afterStale.Value.Issues), len(afterStale.Value.Blockers))
	}

	// One missing target invalidates the entire reviewed impact set.
	freshRevision := afterStale.Value.OutcomeRevision
	invalidTarget := doRequest(t, client, http.MethodPost, url, payload(&freshRevision, true), map[string]string{"Idempotency-Key": "multi-impact-invalid-target-0001"})
	if invalidTarget.StatusCode != http.StatusNotFound {
		t.Fatalf("missing target status = %d, want 404", invalidTarget.StatusCode)
	}
	invalidTarget.Body.Close()
	afterInvalidTarget := doJSON[application.OutcomeState](t, client, http.MethodGet, outcomeURL+"/state", nil, nil, http.StatusOK)
	if afterInvalidTarget.Value.OutcomeRevision != freshRevision || len(afterInvalidTarget.Value.Issues) != 0 || len(afterInvalidTarget.Value.Blockers) != 0 {
		t.Fatalf("invalid target left a partial compound write: revision=%d issues/blockers=%d/%d", afterInvalidTarget.Value.OutcomeRevision, len(afterInvalidTarget.Value.Issues), len(afterInvalidTarget.Value.Blockers))
	}

	// The existing single-impact endpoint retains its previous compatibility
	// contract; this new multi-target route requires a fresh reviewed revision.
	created := doJSON[application.MutationResult[application.ReportIssueWithBlockersResult]](
		t, client, http.MethodPost, url, payload(&freshRevision, false),
		map[string]string{"Idempotency-Key": "multi-impact-create-0001"}, http.StatusCreated,
	)
	if len(created.Value.Value.Blockers) != 2 || created.Value.OutcomeRevision != freshRevision+1 {
		t.Fatalf("created multi-impact result = blockers:%d revision:%d, want blockers:2 revision:%d", len(created.Value.Value.Blockers), created.Value.OutcomeRevision, freshRevision+1)
	}
	for i, blocker := range created.Value.Value.Blockers {
		if blocker.CauseRef == nil || *blocker.CauseRef != created.Value.Value.Issue.Ref() {
			t.Fatalf("blocker %d cause = %#v, want newly created Issue", i, blocker.CauseRef)
		}
	}
}
