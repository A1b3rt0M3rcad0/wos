package httptransport

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWave07IssueBlockerHTTPContract(t *testing.T) {
	handler := newTestHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()

	namespaceID := "0199ef00-0000-7000-8000-000000000001"
	base := server.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes"
	createdOutcome := doJSON[application.MutationResult[domain.Outcome]](
		t, client, http.MethodPost, base,
		[]byte(`{"title":"Wave 07 HTTP","desired_state":"issues and blockers coordinate work","priority":"normal"}`),
		map[string]string{"Idempotency-Key": "wave07-outcome-create-0001"},
		http.StatusCreated,
	)
	outcome := createdOutcome.Value.Value
	outcomeURL := base + "/" + outcome.ID.String()

	criterion := doJSON[application.MutationResult[domain.SuccessCriterion]](
		t, client, http.MethodPost, outcomeURL+"/criteria",
		[]byte(`{"title":"Activate","required":true,"verification_mode":"attestation"}`),
		map[string]string{
			"Idempotency-Key": "wave07-outcome-criterion-0001",
			"If-Match":        createdOutcome.Header.Get("ETag"),
		},
		http.StatusCreated,
	)
	doJSON[application.MutationResult[domain.Outcome]](
		t, client, http.MethodPost, outcomeURL+"/actions/activate",
		[]byte(`{}`),
		map[string]string{
			"Idempotency-Key": "wave07-outcome-activate-0001",
			"If-Match":        criterion.Header.Get("ETag"),
		},
		http.StatusOK,
	)

	workResponse := doJSON[application.MutationResult[domain.WorkItem]](
		t, client, http.MethodPost, outcomeURL+"/work-items",
		[]byte(`{"title":"Coordinated work","priority":"normal","lifecycle":"todo"}`),
		map[string]string{"Idempotency-Key": "wave07-work-create-0001"},
		http.StatusCreated,
	)
	work := workResponse.Value.Value

	independentIssue := doJSON[application.MutationResult[domain.Issue]](
		t, client, http.MethodPost, outcomeURL+"/issues",
		[]byte(`{
			"title":"Observed but non-blocking",
			"description":"A problem can exist without being an impediment.",
			"severity":"minor",
			"affected_refs":[{"kind":"work_item","id":"`+work.ID.String()+`"}]
		}`),
		map[string]string{"Idempotency-Key": "wave07-issue-independent-0001"},
		http.StatusCreated,
	)
	if independentIssue.Header.Get("ETag") == "" {
		t.Fatal("issue create did not return ETag")
	}
	stateBeforeBlocker := doJSON[application.OutcomeState](
		t, client, http.MethodGet, outcomeURL+"/state", nil, nil, http.StatusOK,
	)
	blockingBefore, ok := httpBlockingState(stateBeforeBlocker.Value.BlockingStates, work.Ref())
	if !ok || blockingBefore.IsBlocked {
		t.Fatalf("independent issue unexpectedly blocked work: %#v, found=%v", blockingBefore, ok)
	}

	patchedIssue := doJSON[application.MutationResult[domain.Issue]](
		t, client, http.MethodPatch, outcomeURL+"/issues/"+independentIssue.Value.Value.ID.String(),
		[]byte(`{"title":"Observed and triaged"}`),
		map[string]string{
			"Idempotency-Key": "wave07-issue-patch-0001",
			"If-Match":        independentIssue.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if patchedIssue.Value.Value.Title != "Observed and triaged" {
		t.Fatalf("patched issue title = %q", patchedIssue.Value.Value.Title)
	}

	compoundBody := []byte(`{
		"issue":{
			"title":"Dependency incident",
			"description":"The dependency is unavailable.",
			"severity":"major",
			"affected_refs":[{"kind":"work_item","id":"` + work.ID.String() + `"}]
		},
		"blocker":{
			"blocked_ref":{"kind":"work_item","id":"` + work.ID.String() + `"},
			"description":"Work cannot proceed."
		}
	}`)
	reported := doJSON[application.MutationResult[application.ReportIssueWithBlockerResult]](
		t, client, http.MethodPost, outcomeURL+"/issues/actions/report-with-blocker",
		compoundBody,
		map[string]string{"Idempotency-Key": "wave07-report-with-blocker-0001"},
		http.StatusCreated,
	)
	replay := doJSON[application.MutationResult[application.ReportIssueWithBlockerResult]](
		t, client, http.MethodPost, outcomeURL+"/issues/actions/report-with-blocker",
		compoundBody,
		map[string]string{"Idempotency-Key": "wave07-report-with-blocker-0001"},
		http.StatusCreated,
	)
	if !replay.Value.IdempotentReplay ||
		replay.Value.Value.Issue.ID != reported.Value.Value.Issue.ID ||
		replay.Value.Value.Blocker.ID != reported.Value.Value.Blocker.ID {
		t.Fatalf("compound replay = %#v, original = %#v", replay.Value, reported.Value)
	}
	issue := reported.Value.Value.Issue
	firstBlocker := reported.Value.Value.Blocker

	secondBlockerResponse := doJSON[application.MutationResult[domain.Blocker]](
		t, client, http.MethodPost, outcomeURL+"/blockers",
		[]byte(`{
			"blocked_ref":{"kind":"work_item","id":"`+work.ID.String()+`"},
			"cause_ref":{"kind":"issue","id":"`+issue.ID.String()+`"},
			"description":"A second independent release confirmation is required."
		}`),
		map[string]string{"Idempotency-Key": "wave07-blocker-second-0001"},
		http.StatusCreated,
	)
	secondBlocker := secondBlockerResponse.Value.Value

	patchedBlocker := doJSON[application.MutationResult[domain.Blocker]](
		t, client, http.MethodPatch, outcomeURL+"/blockers/"+secondBlocker.ID.String(),
		[]byte(`{"description":"Second release confirmation still required."}`),
		map[string]string{
			"Idempotency-Key": "wave07-blocker-patch-0001",
			"If-Match":        secondBlockerResponse.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	secondBlocker = patchedBlocker.Value.Value

	stateBlocked := doJSON[application.OutcomeState](
		t, client, http.MethodGet, outcomeURL+"/state", nil, nil, http.StatusOK,
	)
	blocking, ok := httpBlockingState(stateBlocked.Value.BlockingStates, work.Ref())
	if !ok || !blocking.IsBlocked || len(blocking.ActiveBlockers) != 2 {
		t.Fatalf("blocking projection = %#v, found=%v", blocking, ok)
	}
	if len(stateBlocked.Value.Issues) != 2 || len(stateBlocked.Value.Blockers) != 2 {
		t.Fatalf("snapshot issue/blocker counts = %d/%d", len(stateBlocked.Value.Issues), len(stateBlocked.Value.Blockers))
	}

	resolvedIssue := doJSON[application.MutationResult[domain.Issue]](
		t, client, http.MethodPost, outcomeURL+"/issues/"+issue.ID.String()+"/actions/resolve",
		[]byte(`{"expected_version":`+itoaVersion(issue.Version)+`,"resolution_summary":"dependency recovered"}`),
		map[string]string{"Idempotency-Key": "wave07-issue-resolve-only-0001"},
		http.StatusOK,
	)
	if resolvedIssue.Value.Value.Lifecycle != domain.IssueLifecycleResolved {
		t.Fatalf("resolved issue lifecycle = %q", resolvedIssue.Value.Value.Lifecycle)
	}
	stillBlocked := doJSON[application.OutcomeState](
		t, client, http.MethodGet, outcomeURL+"/state", nil, nil, http.StatusOK,
	)
	blocking, ok = httpBlockingState(stillBlocked.Value.BlockingStates, work.Ref())
	if !ok || !blocking.IsBlocked || len(blocking.ActiveBlockers) != 2 {
		t.Fatalf("resolving issue silently changed blockers: %#v, found=%v", blocking, ok)
	}

	reopened := doJSON[application.MutationResult[domain.Issue]](
		t, client, http.MethodPost, outcomeURL+"/issues/"+issue.ID.String()+"/actions/reopen",
		[]byte(`{}`),
		map[string]string{
			"Idempotency-Key": "wave07-issue-reopen-0001",
			"If-Match":        resolvedIssue.Header.Get("ETag"),
		},
		http.StatusOK,
	)

	compoundResolveBody := []byte(`{
		"expected_issue_version":` + itoaVersion(reopened.Value.Value.Version) + `,
		"issue_resolution_summary":"fix verified for the first impediment",
		"blockers":[{
			"blocker_id":"` + firstBlocker.ID.String() + `",
			"expected_version":` + itoaVersion(firstBlocker.Version) + `,
			"resolution_summary":"first release confirmed",
			"release_confirmed":true
		}]
	}`)
	compoundResolved := doJSON[application.MutationResult[application.ResolveIssueAndBlockersResult]](
		t, client, http.MethodPost, outcomeURL+"/issues/"+issue.ID.String()+"/actions/resolve-with-blockers",
		compoundResolveBody,
		map[string]string{"Idempotency-Key": "wave07-resolve-with-blockers-0001"},
		http.StatusOK,
	)
	if len(compoundResolved.Value.Value.Blockers) != 1 ||
		compoundResolved.Value.Value.Blockers[0].ID != firstBlocker.ID {
		t.Fatalf("compound resolved blockers = %#v", compoundResolved.Value.Value.Blockers)
	}

	oneRemaining := doJSON[application.OutcomeState](
		t, client, http.MethodGet, outcomeURL+"/state", nil, nil, http.StatusOK,
	)
	blocking, ok = httpBlockingState(oneRemaining.Value.BlockingStates, work.Ref())
	if !ok || !blocking.IsBlocked || len(blocking.ActiveBlockers) != 1 ||
		blocking.ActiveBlockers[0].Blocker.ID != secondBlocker.ID {
		t.Fatalf("remaining blocking projection = %#v, found=%v", blocking, ok)
	}

	resolvedSecond := doJSON[application.MutationResult[domain.Blocker]](
		t, client, http.MethodPost, outcomeURL+"/blockers/"+secondBlocker.ID.String()+"/actions/resolve",
		[]byte(`{"resolution_summary":"second release independently confirmed"}`),
		map[string]string{
			"Idempotency-Key": "wave07-blocker-second-resolve-0001",
			"If-Match":        patchedBlocker.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if resolvedSecond.Value.Value.Lifecycle != domain.BlockerLifecycleResolved {
		t.Fatalf("second blocker lifecycle = %q", resolvedSecond.Value.Value.Lifecycle)
	}

	ready := doJSON[application.ReadyWork](
		t, client, http.MethodGet, outcomeURL+"/ready-work", nil, nil, http.StatusOK,
	)
	if !httpReadyContains(ready.Value.Items, work.ID) {
		t.Fatal("work should be ready after every active blocker is resolved")
	}

	externalWork := doJSON[application.MutationResult[domain.WorkItem]](
		t, client, http.MethodPost, outcomeURL+"/work-items",
		[]byte(`{"title":"External dependency work","priority":"normal","lifecycle":"todo"}`),
		map[string]string{"Idempotency-Key": "wave07-external-work-0001"},
		http.StatusCreated,
	)
	externalBlocker := doJSON[application.MutationResult[domain.Blocker]](
		t, client, http.MethodPost, outcomeURL+"/blockers",
		[]byte(`{
			"blocked_ref":{"kind":"work_item","id":"`+externalWork.Value.Value.ID.String()+`"},
			"external_cause":{"provider":"vendor","id":"incident-42","description":"Vendor outage"},
			"description":"Waiting for vendor recovery."
		}`),
		map[string]string{"Idempotency-Key": "wave07-external-blocker-0001"},
		http.StatusCreated,
	)
	if externalBlocker.Value.Value.ExternalCause == nil ||
		externalBlocker.Value.Value.ExternalCause.Provider != "vendor" {
		t.Fatalf("external blocker cause = %#v", externalBlocker.Value.Value.ExternalCause)
	}

	issues := doJSON[collectionResponse[domain.Issue]](
		t, client, http.MethodGet, outcomeURL+"/issues", nil, nil, http.StatusOK,
	)
	blockers := doJSON[collectionResponse[domain.Blocker]](
		t, client, http.MethodGet, outcomeURL+"/blockers", nil, nil, http.StatusOK,
	)
	if len(issues.Value.Items) != 2 {
		t.Fatalf("issue collection size = %d, want 2", len(issues.Value.Items))
	}
	if len(blockers.Value.Items) != 3 {
		t.Fatalf("blocker collection size = %d, want 3", len(blockers.Value.Items))
	}

	missingMultiVersion := doRequest(
		t, client, http.MethodPost, outcomeURL+"/issues/"+independentIssue.Value.Value.ID.String()+"/actions/resolve-with-blockers",
		[]byte(`{"issue_resolution_summary":"missing version","blockers":[]}`),
		map[string]string{"Idempotency-Key": "wave07-missing-multi-version-0001"},
	)
	if missingMultiVersion.StatusCode != http.StatusPreconditionRequired {
		t.Fatalf("missing multi-aggregate version status = %d, want 428", missingMultiVersion.StatusCode)
	}
	missingMultiVersion.Body.Close()
}

func httpBlockingState(states []application.BlockingState, ref domain.EntityRef) (application.BlockingState, bool) {
	for _, state := range states {
		if state.Ref == ref {
			return state, true
		}
	}
	return application.BlockingState{}, false
}

func itoaVersion(version domain.Version) string {
	return fmt.Sprintf("%d", version)
}
