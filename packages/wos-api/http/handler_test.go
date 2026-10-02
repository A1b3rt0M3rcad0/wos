package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/authentication/local"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type fixedClock struct{ value time.Time }

func (c fixedClock) Now() time.Time { return c.value }

type sequenceIDs struct {
	prefix string
	next   uint64
}

func (g *sequenceIDs) NewID() (domain.ID, error) {
	value := domain.MustParseID(fmt.Sprintf("%s-0000-7000-8000-%012x", g.prefix, g.next))
	g.next++
	return value, nil
}

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	store := memory.New()
	appIDs := &sequenceIDs{prefix: "0199ea00", next: 1}
	service, err := application.NewService(
		store,
		fixedClock{value: time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)},
		appIDs,
	)
	if err != nil {
		t.Fatal(err)
	}
	auth, err := local.New("http-human")
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(Options{
		Prefix:         "/api/v1",
		RequestTimeout: 2 * time.Second,
		Service:        service,
		IDs:            &sequenceIDs{prefix: "0199eb00", next: 1},
		LocalAuth:      auth,
	})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestHumanHTTPVerticalSlice(t *testing.T) {
	handler := newTestHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()

	namespaceID := "0199ec00-0000-7000-8000-000000000001"
	base := server.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes"

	createOutcomeBody := []byte(`{
		"title":"HTTP outcome",
		"description":"created without any agent runtime",
		"desired_state":"durable state can be coordinated through HTTP",
		"priority":"normal"
	}`)
	createOutcomeResponse := doJSON[application.MutationResult[domain.Outcome]](
		t,
		client,
		http.MethodPost,
		base,
		createOutcomeBody,
		map[string]string{
			"Idempotency-Key":  "http-outcome-create-0001",
			"X-Correlation-ID": "http-test-correlation",
		},
		http.StatusCreated,
	)
	outcome := createOutcomeResponse.Value.Value
	if outcome.NamespaceID.String() != namespaceID {
		t.Fatalf("outcome namespace = %s, want %s", outcome.NamespaceID, namespaceID)
	}
	if createOutcomeResponse.Header.Get("X-Correlation-ID") != "http-test-correlation" {
		t.Fatal("correlation id was not preserved")
	}
	outcomeURL := base + "/" + outcome.ID.String()
	expectedLocation := "/api/v1/namespaces/" + namespaceID + "/outcomes/" + outcome.ID.String()
	if createOutcomeResponse.Header.Get("Location") != expectedLocation {
		t.Fatalf("Location = %q, want %q", createOutcomeResponse.Header.Get("Location"), expectedLocation)
	}
	outcomeETagV1 := createOutcomeResponse.Header.Get("ETag")
	if outcomeETagV1 == "" {
		t.Fatal("create outcome did not return ETag")
	}
	initialOutcomeETag := outcomeETagV1

	patchedOutcome := doJSON[application.MutationResult[domain.Outcome]](
		t,
		client,
		http.MethodPatch,
		outcomeURL,
		[]byte(`{"title":"HTTP outcome updated"}`),
		map[string]string{
			"Idempotency-Key": "http-outcome-patch-0001",
			"If-Match":        outcomeETagV1,
		},
		http.StatusOK,
	)
	if patchedOutcome.Value.Value.Title != "HTTP outcome updated" {
		t.Fatalf("patched title = %q", patchedOutcome.Value.Value.Title)
	}
	outcomeETagV1 = patchedOutcome.Header.Get("ETag")

	stalePatch := doRequest(
		t,
		client,
		http.MethodPatch,
		outcomeURL,
		[]byte(`{"description":"stale update"}`),
		map[string]string{
			"Idempotency-Key": "http-outcome-patch-stale-0001",
			"If-Match":        initialOutcomeETag,
		},
	)
	if stalePatch.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("stale patch status = %d, want 412", stalePatch.StatusCode)
	}
	stalePatch.Body.Close()

	noOpPatch := doJSON[application.MutationResult[domain.Outcome]](
		t,
		client,
		http.MethodPatch,
		outcomeURL,
		[]byte(`{"title":"HTTP outcome updated"}`),
		map[string]string{
			"Idempotency-Key": "http-outcome-patch-noop-0001",
			"If-Match":        outcomeETagV1,
		},
		http.StatusOK,
	)
	if noOpPatch.Value.Value.Version != patchedOutcome.Value.Value.Version {
		t.Fatalf("no-op patch version = %d, want %d", noOpPatch.Value.Value.Version, patchedOutcome.Value.Value.Version)
	}
	if noOpPatch.Value.OutcomeRevision != patchedOutcome.Value.OutcomeRevision {
		t.Fatalf("no-op patch outcome revision = %d, want %d", noOpPatch.Value.OutcomeRevision, patchedOutcome.Value.OutcomeRevision)
	}
	if noOpPatch.Header.Get("ETag") != outcomeETagV1 {
		t.Fatalf("no-op patch ETag = %q, want %q", noOpPatch.Header.Get("ETag"), outcomeETagV1)
	}

	criterionResponse := doJSON[application.MutationResult[domain.SuccessCriterion]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/criteria",
		[]byte(`{
			"title":"Human verifies completion",
			"required":true,
			"verification_mode":"attestation"
		}`),
		map[string]string{
			"Idempotency-Key": "http-outcome-criterion-0001",
			"If-Match":        outcomeETagV1,
		},
		http.StatusCreated,
	)
	outcomeETagV2 := criterionResponse.Header.Get("ETag")
	if outcomeETagV2 == "" || outcomeETagV2 == outcomeETagV1 {
		t.Fatalf("criterion mutation ETag = %q, previous = %q", outcomeETagV2, outcomeETagV1)
	}

	stale := doRequest(
		t,
		client,
		http.MethodPost,
		outcomeURL+"/actions/activate",
		[]byte(`{}`),
		map[string]string{
			"Idempotency-Key": "http-stale-activate-0001",
			"If-Match":        outcomeETagV1,
		},
	)
	if stale.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("stale activation status = %d, want 412", stale.StatusCode)
	}
	stale.Body.Close()

	activateResponse := doJSON[application.MutationResult[domain.Outcome]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/actions/activate",
		[]byte(`{}`),
		map[string]string{
			"Idempotency-Key": "http-activate-outcome-0001",
			"If-Match":        outcomeETagV2,
		},
		http.StatusOK,
	)
	if activateResponse.Value.Value.Lifecycle != domain.OutcomeLifecycleActive {
		t.Fatalf("outcome lifecycle = %q, want active", activateResponse.Value.Value.Lifecycle)
	}

	objectiveResponse := doJSON[application.MutationResult[domain.Objective]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/objectives",
		[]byte(`{
			"title":"Prepare HTTP flow",
			"priority":"normal",
			"required_for_outcome":true
		}`),
		map[string]string{"Idempotency-Key": "http-objective-create-0001"},
		http.StatusCreated,
	)
	objective := objectiveResponse.Value.Value
	objectiveURL := outcomeURL + "/objectives/" + objective.ID.String()
	patchedObjective := doJSON[application.MutationResult[domain.Objective]](
		t,
		client,
		http.MethodPatch,
		objectiveURL,
		[]byte(`{"title":"Prepare HTTP flow updated"}`),
		map[string]string{
			"Idempotency-Key": "http-objective-patch-0001",
			"If-Match":        objectiveResponse.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if patchedObjective.Value.Value.Title != "Prepare HTTP flow updated" {
		t.Fatalf("patched objective title = %q", patchedObjective.Value.Value.Title)
	}

	objectiveCriterion := doJSON[application.MutationResult[domain.SuccessCriterion]](
		t,
		client,
		http.MethodPost,
		objectiveURL+"/criteria",
		[]byte(`{
			"title":"Preparation attested",
			"required":true,
			"verification_mode":"attestation"
		}`),
		map[string]string{
			"Idempotency-Key": "http-objective-criterion-0001",
			"If-Match":        patchedObjective.Header.Get("ETag"),
		},
		http.StatusCreated,
	)
	startedObjective := doJSON[application.MutationResult[domain.Objective]](
		t,
		client,
		http.MethodPost,
		objectiveURL+"/actions/start",
		[]byte(`{}`),
		map[string]string{
			"Idempotency-Key": "http-objective-start-0001",
			"If-Match":        objectiveCriterion.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if startedObjective.Value.Value.Lifecycle != domain.ObjectiveLifecycleInProgress {
		t.Fatalf("objective lifecycle = %q, want in_progress", startedObjective.Value.Value.Lifecycle)
	}

	workBody := []byte(`{
		"title":"Execute HTTP work",
		"priority":"normal",
		"lifecycle":"todo"
	}`)
	workResponse := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/work-items",
		workBody,
		map[string]string{"Idempotency-Key": "http-work-create-replay-0001"},
		http.StatusCreated,
	)
	replayedWork := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		client,
		http.MethodPost,
		outcomeURL+"/work-items",
		workBody,
		map[string]string{"Idempotency-Key": "http-work-create-replay-0001"},
		http.StatusCreated,
	)
	if !replayedWork.Value.IdempotentReplay {
		t.Fatal("repeated HTTP create was not marked as idempotent replay")
	}
	if replayedWork.Value.Value.ID != workResponse.Value.Value.ID {
		t.Fatalf("replayed work id = %s, want %s", replayedWork.Value.Value.ID, workResponse.Value.Value.ID)
	}

	work := workResponse.Value.Value
	workURL := outcomeURL + "/work-items/" + work.ID.String()
	patchedWork := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		client,
		http.MethodPatch,
		workURL,
		[]byte(`{"title":"Execute HTTP work updated"}`),
		map[string]string{
			"Idempotency-Key": "http-work-patch-0001",
			"If-Match":        workResponse.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if patchedWork.Value.Value.Title != "Execute HTTP work updated" {
		t.Fatalf("patched work title = %q", patchedWork.Value.Value.Title)
	}
	work = patchedWork.Value.Value

	claimResponse := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		client,
		http.MethodPost,
		workURL+"/actions/claim",
		[]byte(`{"lease_ttl_seconds":300}`),
		map[string]string{
			"Idempotency-Key": "http-work-claim-0001",
			"If-Match":        patchedWork.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if claimResponse.Value.Value.CurrentLease == nil {
		t.Fatal("claim did not return a lease")
	}

	completeBody, err := json.Marshal(completeRequest{
		ClaimID:       claimResponse.Value.Value.CurrentLease.ClaimID.String(),
		FencingToken:  claimResponse.Value.Value.CurrentLease.FencingToken,
		ResultSummary: "HTTP work completed",
		Reason:        "completed through local HTTP transport",
	})
	if err != nil {
		t.Fatal(err)
	}
	completed := doJSON[application.MutationResult[domain.WorkItem]](
		t,
		client,
		http.MethodPost,
		workURL+"/actions/complete",
		completeBody,
		map[string]string{
			"Idempotency-Key": "http-work-complete-0001",
			"If-Match":        claimResponse.Header.Get("ETag"),
		},
		http.StatusOK,
	)
	if completed.Value.Value.Lifecycle != domain.WorkItemLifecycleDone {
		t.Fatalf("completed lifecycle = %q, want done", completed.Value.Value.Lifecycle)
	}

	stateResponse := doJSON[application.OutcomeState](
		t,
		client,
		http.MethodGet,
		outcomeURL+"/state",
		nil,
		nil,
		http.StatusOK,
	)
	if stateResponse.Value.Outcome.ID != outcome.ID {
		t.Fatalf("state outcome id = %s, want %s", stateResponse.Value.Outcome.ID, outcome.ID)
	}
	if len(stateResponse.Value.Objectives) != 1 {
		t.Fatalf("state objectives = %d, want 1", len(stateResponse.Value.Objectives))
	}
	if len(stateResponse.Value.WorkItems) != 1 || stateResponse.Value.WorkItems[0].Lifecycle != domain.WorkItemLifecycleDone {
		t.Fatalf("state work items = %#v", stateResponse.Value.WorkItems)
	}
	if stateResponse.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("state Cache-Control = %q, want no-store", stateResponse.Header.Get("Cache-Control"))
	}

	invalidPayload := doRequest(
		t,
		client,
		http.MethodPost,
		outcomeURL+"/objectives",
		[]byte(`{"title":"bad","priority":"normal","unknown":true}`),
		map[string]string{"Idempotency-Key": "http-invalid-payload-0001"},
	)
	if invalidPayload.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid payload status = %d, want 400", invalidPayload.StatusCode)
	}
	invalidPayload.Body.Close()

	semanticInvalid := doRequest(
		t,
		client,
		http.MethodPost,
		outcomeURL+"/work-items",
		[]byte(`{"title":"bad priority","priority":"unsupported"}`),
		map[string]string{"Idempotency-Key": "http-semantic-invalid-0001"},
	)
	if semanticInvalid.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("semantic invalid status = %d, want 422", semanticInvalid.StatusCode)
	}
	semanticInvalid.Body.Close()

	largeDescription := bytes.Repeat([]byte("x"), maxJSONBodyBytes+1)
	largePayload := append([]byte(`{"title":"large","priority":"normal","description":"`), largeDescription...)
	largePayload = append(largePayload, []byte(`"}`)...)
	tooLarge := doRequest(
		t,
		client,
		http.MethodPost,
		outcomeURL+"/objectives",
		largePayload,
		map[string]string{"Idempotency-Key": "http-payload-too-large-0001"},
	)
	if tooLarge.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("payload too large status = %d, want 413", tooLarge.StatusCode)
	}
	tooLarge.Body.Close()

	missingVersion := doRequest(
		t,
		client,
		http.MethodPost,
		objectiveURL+"/actions/reopen",
		[]byte(`{"reason":"missing precondition"}`),
		map[string]string{"Idempotency-Key": "http-missing-version-0001"},
	)
	if missingVersion.StatusCode != http.StatusPreconditionRequired {
		t.Fatalf("missing version status = %d, want 428", missingVersion.StatusCode)
	}
	missingVersion.Body.Close()

	otherNamespace := "0199ec00-0000-7000-8000-000000000002"
	wrongScope := doRequest(
		t,
		client,
		http.MethodGet,
		server.URL+"/api/v1/namespaces/"+otherNamespace+"/outcomes/"+outcome.ID.String(),
		nil,
		nil,
	)
	if wrongScope.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-namespace read status = %d, want 404", wrongScope.StatusCode)
	}
	wrongScope.Body.Close()
}

type decodedResponse[T any] struct {
	Value  T
	Header http.Header
}

func doJSON[T any](
	t *testing.T,
	client *http.Client,
	method string,
	url string,
	body []byte,
	headers map[string]string,
	wantStatus int,
) decodedResponse[T] {
	t.Helper()
	response := doRequest(t, client, method, url, body, headers)
	defer response.Body.Close()
	if response.StatusCode != wantStatus {
		var payload any
		_ = json.NewDecoder(response.Body).Decode(&payload)
		t.Fatalf("%s %s status = %d, want %d, body=%#v", method, url, response.StatusCode, wantStatus, payload)
	}
	var value T
	if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return decodedResponse[T]{Value: value, Header: response.Header.Clone()}
}

func doRequest(
	t *testing.T,
	client *http.Client,
	method string,
	url string,
	body []byte,
	headers map[string]string,
) *http.Response {
	t.Helper()
	var payload *bytes.Reader
	if body == nil {
		payload = bytes.NewReader(nil)
	} else {
		payload = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, url, payload)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
