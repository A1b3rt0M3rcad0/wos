package woscli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func operationalFixtureV2(t *testing.T) (*Workspace, ProfileV2, []byte, ContractFileV2, d.ID) {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	project, profile, token, file, intent, _ := returnFixtureV2(t)
	file.Local.Pending, profile.Local.PendingOperations = nil, nil
	if e := profile.SealBinding(token); e != nil {
		t.Fatal(e)
	}
	w, e := OpenWorkspace(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { w.Close() })
	id := *intent.ContractID
	for path, value := range map[string]any{".wos/project.yaml": project, profilePathV2(profile.Name): profile, contractPathV2(profile.Name, id): file} {
		if e := w.CreateV2(path, value); e != nil {
			t.Fatal(e)
		}
	}
	return w, profile, token, file, id
}

func reissueOperationalFixtureV2(t *testing.T, profile ProfileV2, file ContractFileV2, id d.ID, title string) ContractFileV2 {
	t.Helper()
	raw, _ := EncodeV2Document(file)
	if e := DecodeV2Document(raw, &file); e != nil {
		t.Fatal(e)
	}
	var spec signing.SpecPayload[d.SignedWorkSpec]
	var authority signing.AuthorityPayload
	if e := json.Unmarshal(file.Issued.Specification.Payload, &spec); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal(file.Issued.Authority.Payload, &authority); e != nil {
		t.Fatal(e)
	}
	spec.ContractID, spec.Spec.Title, authority.ContractID = id.String(), title, id.String()
	private := ed25519.NewKeyFromSeed(make([]byte, 32))
	env, e := signing.Sign(signing.ContractSpec, spec, profile.Binding.IssuerKeys[0].KeyID.String(), private)
	if e != nil {
		t.Fatal(e)
	}
	file.Issued.Specification, e = signing.ToDocument(env)
	if e != nil {
		t.Fatal(e)
	}
	authority.SpecDigest = signing.Digest(file.Issued.Specification.Payload)
	env, e = signing.Sign(signing.ContractAuthority, authority, profile.Binding.IssuerKeys[0].KeyID.String(), private)
	if e != nil {
		t.Fatal(e)
	}
	file.Issued.Authority, e = signing.ToDocument(env)
	if e != nil {
		t.Fatal(e)
	}
	file.Execution.Material.Result.ContractID, file.Execution.Material.Result.SpecDigest = id, authority.SpecDigest
	return file
}

func TestOperationalContractSelectionRejectsAmbiguityAndRetainsUUIDs(t *testing.T) {
	w, profile, _, original, id := operationalFixtureV2(t)
	selected, e := selectedContractV2(w, profile, "execution", "")
	if e != nil || selected != id {
		t.Fatalf("unique selection: %s %v", selected, e)
	}
	otherID := d.ID("0199ac10-0000-7000-8000-000000000018")
	other := reissueOperationalFixtureV2(t, profile, original, otherID, "A different task")
	if e = w.CreateV2(contractPathV2(profile.Name, otherID), other); e != nil {
		t.Fatal(e)
	}
	for _, ref := range []string{"", "0199"} {
		if _, e = selectedContractV2(w, profile, "execution", ref); e == nil {
			t.Fatalf("ambiguous ref %q selected a contract", ref)
		}
	}
	selected, e = selectedContractV2(w, profile, "execution", "a different TASK")
	if e != nil || selected != otherID {
		t.Fatalf("title selection: %s %v", selected, e)
	}
	selected, e = selectedContractV2(w, profile, "execution", id.String())
	if e != nil || selected != id {
		t.Fatalf("exact UUID changed: %s %v", selected, e)
	}
	if _, e = selectedContractV2(w, profile, "review", "A different task"); e == nil {
		t.Fatal("execution contract selected for reviewer")
	}
}

func TestTypedMaterialEditPreservesIssuerProofAndRejectsStaleOrFrozenDraft(t *testing.T) {
	w, profile, token, original, id := operationalFixtureV2(t)
	beforeDigest, _ := original.DraftDigest()
	output, e := materialCommandV2(context.Background(), w, profile, nil, token, "execution", options{args: []string{"work", "result", "set"}, values: map[string]string{"summary": "Implemented and checked", "completion": "review", "expected-draft": beforeDigest}})
	if e != nil || output.ContractID != id {
		t.Fatalf("typed edit: %+v %v", output, e)
	}
	file, _, path, raw, e := loadContractV2(w, profile, id)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(original.Issued, file.Issued) || file.Execution.Material.Result.Summary != "Implemented and checked" || file.Execution.CompletionIntent != "review" {
		t.Fatal("typed edit altered proof or lost material")
	}
	if _, e = mutateMaterialV2(context.Background(), w, profile, token, id, "execution", beforeDigest, func(file *ContractFileV2, _ ContractViewV2) error {
		file.Execution.Progress.Summary = "stale"
		return nil
	}); e == nil {
		t.Fatal("stale draft overwrote material")
	}
	after, _ := w.ReadV2(path)
	if !bytes.Equal(raw, after) {
		t.Fatal("conflicting edit altered material")
	}
	file.Local.Pending = &FrozenReturnV2{State: "prepared_signed", CredentialID: profile.Binding.CredentialID}
	if e = w.WriteV2(path, file, signing.Digest(raw)); e != nil {
		t.Fatal(e)
	}
	if _, e = mutateMaterialV2(context.Background(), w, profile, token, id, "execution", "", func(*ContractFileV2, ContractViewV2) error { t.Fatal("frozen material reached editor"); return nil }); e == nil {
		t.Fatal("frozen return editable")
	}
}

func TestActivityLostResponseRecoversDurablyWithoutSecondWrite(t *testing.T) {
	w, profile, token, original, id := operationalFixtureV2(t)
	actor := d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}
	intent, e := prepareActivityV2(context.Background(), w, profile, token, actor, id, options{args: []string{"work", "log", "add"}, values: map[string]string{"description": "Identified and fixed expiry handling", "type": "correction"}})
	if e != nil {
		t.Fatal(e)
	}
	cmd, e := decodeActivityIntentV2(profile, intent)
	if e != nil {
		t.Fatal(e)
	}
	activityID := d.ID("0199ac10-0000-7000-8000-000000000019")
	commandID := d.ID("0199ac10-0000-7000-8000-000000000020")
	accepted := sdk.CommandResult[a.WorkActivityResult]{CommandID: commandID, OutcomeRevision: 42, Value: a.WorkActivityResult{Activity: d.WorkActivity{ID: activityID, Scope: cmd.Scope, ContractID: id, WorkItemID: original.Execution.Material.Result.WorkItemID, Sequence: 42, Type: cmd.Type, Description: cmd.Description, PrincipalID: profile.Binding.PrincipalID, Actor: actor, RecordedAt: time.Now().UTC()}}}
	response, _ := json.Marshal(accepted)
	var writes, reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, request *http.Request) {
		if request.Method == "POST" {
			writes.Add(1)
			if request.Header.Get("Idempotency-Key") != intent.IdempotencyKey {
				t.Error("activity idempotency changed")
			}
			var body struct {
				Command json.RawMessage `json:"command"`
			}
			if e := json.NewDecoder(request.Body).Decode(&body); e != nil {
				t.Error(e)
				return
			}
			normalized, e := commands.Normalize(body.Command, reflect.TypeFor[a.AppendWorkActivityCommand]())
			if e != nil {
				t.Error(e)
				return
			}
			var actual a.AppendWorkActivityCommand
			if e = commands.Decode(normalized, &actual); e != nil || !reflect.DeepEqual(cmd, actual) {
				t.Errorf("original activity changed: %v", e)
			}
			connection, _, e := rw.(http.Hijacker).Hijack()
			if e != nil {
				t.Error(e)
				return
			}
			connection.Close()
			return
		}
		reads.Add(1)
		if request.URL.Query().Get("idempotency_key") != intent.IdempotencyKey {
			t.Error("reconciliation selected another key")
		}
		value := a.SignedStateResult{NamespaceID: cmd.Scope.NamespaceID, Scope: &cmd.Scope, OperationName: intent.Operation, OperationCommandID: &commandID, OperationFingerprint: intent.RequestFingerprint, OperationPayload: base64.StdEncoding.EncodeToString(response), PayloadDigest: signing.Digest(response)}
		if e := json.NewEncoder(rw).Encode(value); e != nil {
			t.Errorf("JSON at %s: %v", request.URL.Path, e)
		}
	}))
	defer server.Close()
	client, e := sdk.New(server.URL, string(token), server.Client())
	if e != nil {
		t.Fatal(e)
	}
	if _, e = recoverActivityV2(context.Background(), w, profile, client, token, intent); e == nil {
		t.Fatal("lost response appeared accepted")
	}
	current, e := w.LoadProfileV2(profile.Name)
	if e != nil {
		t.Fatal(e)
	}
	if len(current.Local.PendingOperations) != 1 || current.Local.PendingOperations[0].State != "sent_unknown" || current.Local.PendingOperations[0].Payload != intent.Payload {
		t.Fatal("lost response discarded original activity")
	}
	if _, e = mutateMaterialV2(context.Background(), w, profile, token, id, "execution", "", func(*ContractFileV2, ContractViewV2) error {
		t.Fatal("pending activity allowed draft edit")
		return nil
	}); e == nil {
		t.Fatal("activity recovery not enforced")
	}
	result, e := recoverActivityV2(context.Background(), w, profile, client, token, intent)
	if e != nil {
		t.Fatal(e)
	}
	if result.CommandID != commandID || writes.Load() != 1 || reads.Load() != 1 {
		t.Fatalf("activity recovery wrote twice: %s writes=%d reads=%d", result.CommandID, writes.Load(), reads.Load())
	}
	current, e = w.LoadProfileV2(profile.Name)
	if e != nil || len(current.Local.PendingOperations) != 0 {
		t.Fatalf("accepted intention not cleared: %v", e)
	}
	file, _, _, _, e := loadContractV2(w, current, id)
	if e != nil || !reflect.DeepEqual(file.Issued, original.Issued) {
		t.Fatalf("activity changed issued authority: %v", e)
	}
}

func TestActivityMaterialIsIdempotentAndPreservesExactVersions(t *testing.T) {
	_, file := contractFixtureV2(t)
	result := file.Execution.Material.Result
	evidence := d.ID("0199ac10-0000-7000-8000-000000000022")
	artifact := d.ID("0199ac10-0000-7000-8000-000000000023")
	criterion := d.ID("0199ac10-0000-7000-8000-000000000024")
	accepted := a.WorkActivityResult{Activity: d.WorkActivity{ContractID: result.ContractID, WorkItemID: result.WorkItemID, EvidenceRefs: []d.ID{evidence}, ArtifactVersions: []d.SubmissionArtifact{{ArtifactID: artifact, SourceVersion: "commit-a", Checksum: "sha256:source"}}, CriterionRefs: []d.WorkActivityCriterionRef{{CriterionID: criterion, CriterionRevision: 1}}}}
	for range 2 {
		if e := applyActivityMaterialV2(&file, accepted); e != nil {
			t.Fatal(e)
		}
	}
	material := file.Execution.Material.Result
	if len(material.EvidenceIDs) != 1 || len(material.Artifacts) != 1 || len(material.CriterionEvidence) != 1 || len(material.CriterionEvidence[0].EvidenceIDs) != 1 {
		t.Fatal("publication recovery duplicated documentary material")
	}
	accepted.Activity.ArtifactVersions[0].SourceVersion = "commit-b"
	if e := applyActivityMaterialV2(&file, accepted); e == nil || !strings.Contains(e.Error(), "exact version") {
		t.Fatalf("artifact drift silently changed frozen result: %v", e)
	}
}

func TestUnsupportedActivityServerFailsBeforePreparingIntent(t *testing.T) {
	w, profile, token, _, _ := operationalFixtureV2(t)
	var writes atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, request *http.Request) {
		if request.Method != "GET" {
			writes.Add(1)
		}
		fmt.Fprint(rw, `{"protocols":["signed_contracts_v2"]}`)
	}))
	defer server.Close()
	client, _ := sdk.New(server.URL, string(token), server.Client())
	_, e := workCommandV2(context.Background(), w, profile, client, token, a.SigningIdentityView{}, options{args: []string{"work", "log", "add"}, values: map[string]string{"description": "Must not downgrade"}})
	if e == nil || !strings.Contains(e.Error(), "upgrade the server") || writes.Load() != 0 {
		t.Fatalf("unsupported activity fallback: %v", e)
	}
	current, e := w.LoadProfileV2(profile.Name)
	if e != nil || len(current.Local.PendingOperations) != 0 {
		t.Fatal("unsupported server prepared a dead intention")
	}
}

func TestWorkContextIsBoundedReadOnlyAndCarriesRecentHistoryCursor(t *testing.T) {
	w, profile, token, original, id := operationalFixtureV2(t)
	view, e := original.VerifyIssued(profile)
	if e != nil {
		t.Fatal(e)
	}
	g := view.Authority
	scope := d.Scope{NamespaceID: profile.Binding.NamespaceID, OutcomeID: d.ID(g.OutcomeID)}
	work, e := d.NewWorkItem(d.ID(g.WorkItemID), scope, "Current title", strings.Repeat("context ", 600), d.PriorityNormal, d.WorkItemLifecycleTodo, time.Now().UTC())
	if e != nil {
		t.Fatal(e)
	}
	focal := a.WorkContext{Work: work, Operational: d.WorkItemOperationalState{Ref: work.Ref()}, Outcome: a.EntitySummary{Ref: d.EntityRef{Scope: scope, Kind: d.EntityKindOutcome, ID: scope.OutcomeID}, Title: "Current Outcome intent", Lifecycle: "active"}, OutcomeRevision: 80, EvaluatedAt: time.Now().UTC(), Omitted: map[string]int{}}
	for range 12 {
		focal.Dependencies = append(focal.Dependencies, d.Relation{ID: id, Scope: scope, SourceRef: work.Ref(), TargetRef: work.Ref(), RelationType: d.RelationTypeDependsOn})
	}
	var writes, historyReads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, request *http.Request) {
		if request.Method != "GET" {
			writes.Add(1)
			rw.WriteHeader(500)
			return
		}
		var value any
		switch {
		case strings.HasSuffix(request.URL.Path, "/capabilities"):
			value = a.ContractCapabilities()
		case strings.Contains(request.URL.Path, "/signed-state/execution/"):
			value = a.SignedStateResult{NamespaceID: scope.NamespaceID, Scope: &scope, Contract: &a.SignedContractMetadata{ID: id, CredentialID: profile.Binding.CredentialID, Kind: "execution", WorkItemID: work.ID, ExecutionID: d.ID(g.ExecutionID), Current: true, LeaseValid: true, Status: d.ContractActive, EffectiveStatus: d.ContractActive}}
		case strings.HasSuffix(request.URL.Path, "/attention"):
			value = a.WorkAttention{WorkItemID: work.ID, OperationalState: d.WorkItemOperationalState{Ref: work.Ref()}, Obligations: []a.WorkObligation{}}
		case strings.HasSuffix(request.URL.Path, "/context"):
			value = focal
		case strings.HasSuffix(request.URL.Path, "/work-activity"):
			historyReads.Add(1)
			if request.URL.Query().Get("limit") != "3" || request.URL.Query().Get("descending") != "true" || request.URL.Query().Get("work_item_id") != work.ID.String() || request.URL.Query().Get("contract_id") != "" {
				t.Errorf("unbounded or mismatched recent query: %s", request.URL.RawQuery)
			}
			value = a.WorkActivityPage{Order: "desc", Items: []d.WorkActivity{}, NextCursor: "frozen-history-cursor", OutcomeRevision: 81, HistoryThroughRevision: 81}
		default:
			t.Errorf("unexpected context expansion %s", request.URL.Path)
			rw.WriteHeader(500)
			return
		}
		if e := json.NewEncoder(rw).Encode(value); e != nil {
			t.Errorf("JSON at %s: %v", request.URL.Path, e)
		}
	}))
	defer server.Close()
	client, _ := sdk.New(server.URL, string(token), server.Client())
	before, _ := w.ReadV2(contractPathV2(profile.Name, id))
	output, e := queryWorkV2(context.Background(), w, profile, client, a.SigningIdentityView{}, "execution", options{args: []string{"work", "context"}, values: map[string]string{"limit": "3"}})
	if e != nil {
		t.Fatal(e)
	}
	data := output.Data.(map[string]any)
	context := data["context"].(map[string]any)
	omitted := context["omitted"].(map[string]int)
	if !context["truncated"].(bool) || omitted["dependencies"] != 2 || omitted["work.description_bytes"] == 0 {
		t.Fatalf("context omissions hidden: %+v", omitted)
	}
	page := data["recent_activity"].(map[string]any)
	if page["order"] != "desc" || page["next_cursor"] != "frozen-history-cursor" || historyReads.Load() != 1 || writes.Load() != 0 {
		t.Fatalf("context history/side effects: %+v", page)
	}
	if data["history_follow_up"].(map[string]any)["order"] != "desc" {
		t.Fatal("history cursor lost its order binding")
	}
	specification := data["specification"].(d.SignedWorkSpec)
	if !reflect.DeepEqual(specification, view.WorkSpecification.Spec) {
		t.Fatal("bounded focal context truncated immutable instructions")
	}
	after, _ := w.ReadV2(contractPathV2(profile.Name, id))
	if !bytes.Equal(before, after) {
		t.Fatal("read-only context changed contract authority or draft")
	}
}

func TestReviewDraftCanBeCorrectedThroughTypedCommandsWithoutEditingDocuments(t *testing.T) {
	w, profile, token, original, _ := operationalFixtureV2(t)
	view, e := original.VerifyIssued(profile)
	if e != nil {
		t.Fatal(e)
	}
	private := ed25519.NewKeyFromSeed(make([]byte, 32))
	sign := func(purpose signing.PayloadType, value any) signing.Document {
		t.Helper()
		envelope, e := signing.Sign(purpose, value, profile.Binding.IssuerKeys[0].KeyID.String(), private)
		if e != nil {
			t.Fatal(e)
		}
		document, e := signing.ToDocument(envelope)
		if e != nil {
			t.Fatal(e)
		}
		return document
	}
	workSpec := *view.WorkSpecification
	criterion := d.SignedCriterion{ID: d.ID("0199ac10-0000-7000-8000-000000000030"), OwnerRef: d.EntityRef{Scope: d.Scope{NamespaceID: profile.Binding.NamespaceID, OutcomeID: d.ID(view.Authority.OutcomeID)}, Kind: d.EntityKindWorkItem, ID: d.ID(view.Authority.WorkItemID)}, Title: "The change was checked", Required: true, Revision: 1, VerificationMode: d.VerificationModeAttestation, Status: d.CriterionStatusActive}
	workSpec.Spec.Criteria = []d.SignedCriterion{criterion}
	workDocument := sign(signing.ContractSpec, workSpec)
	reviewID := d.ID("0199ac10-0000-7000-8000-000000000031")
	caseID := d.ID("0199ac10-0000-7000-8000-000000000032")
	submissionID := d.ID("0199ac10-0000-7000-8000-000000000033")
	_, _, _, _, _, receipt := returnFixtureV2(t)
	reviewSpec := signing.SpecPayload[a.SignedReviewSpec]{Binding: workSpec.Binding, ContractKind: "review", WorkItemID: view.Authority.WorkItemID, ContractID: reviewID.String(), Spec: a.SignedReviewSpec{ReviewCaseID: caseID, SubmissionID: submissionID, IssuedWorkSpecification: &workDocument, IssuedWorkSpecDigest: signing.Digest(workDocument.Payload), AcceptanceReceipt: receipt}}
	specification := sign(signing.ContractSpec, reviewSpec)
	grant := view.Authority
	grant.ContractID, grant.ContractKind, grant.ReviewCaseID, grant.SpecDigest = reviewID.String(), "review", caseID.String(), signing.Digest(specification.Payload)
	file := ContractFileV2{SchemaVersion: 2, Kind: "WOSContractFile", Issued: IssuedDocumentsV2{Specification: specification, Authority: sign(signing.ContractAuthority, grant)}, Review: &ReviewDraftV2{Decision: "approved", Material: a.SignedReviewMaterial{Reason: "Initial independent inspection"}}, Local: ContractLocalV2{SchemaVersion: 1, Profile: profile.Name, WorkItemVersion: 1, ReviewCaseVersion: 1}}
	if e = w.CreateV2(contractPathV2(profile.Name, reviewID), file); e != nil {
		t.Fatal(e)
	}
	edit := func(section, verb string, values map[string]string) error {
		_, e := materialCommandV2(context.Background(), w, profile, nil, token, "review", options{args: []string{"review", section, verb}, values: values})
		return e
	}
	assessment := map[string]string{"criterion": criterion.Title, "result": "not_met", "rationale": "A check needs correction"}
	if e = edit("assessment", "add", assessment); e != nil {
		t.Fatal(e)
	}
	if e = edit("assessment", "add", assessment); e == nil {
		t.Fatal("duplicate add silently overwrote an assessment")
	}
	assessment["result"], assessment["rationale"] = "met", "Rechecked the original material"
	if e = edit("assessment", "set", assessment); e != nil {
		t.Fatal(e)
	}
	current, _, _, _, e := loadContractV2(w, profile, reviewID)
	if e != nil || len(current.Review.Material.Assessments) != 1 || current.Review.Material.Assessments[0].Result != d.AssessmentResultMet {
		t.Fatalf("assessment set lost exact target: %v", e)
	}
	decision := map[string]string{"decision": "changes_requested", "reason": "The original task title needs correction"}
	if e = edit("decision", "set", decision); e == nil {
		t.Fatal("decision silently discarded assessments")
	}
	if e = edit("assessment", "remove", map[string]string{"criterion": criterion.Title}); e != nil {
		t.Fatal(e)
	}
	if e = edit("decision", "set", decision); e != nil {
		t.Fatal(e)
	}
	if e = edit("finding", "add", map[string]string{"requirement": "task.title", "description": "Missing title detail"}); e != nil {
		t.Fatal(e)
	}
	current, _, _, _, _ = loadContractV2(w, profile, reviewID)
	findingID := current.Review.Material.Findings[0].ID
	if e = edit("finding", "set", map[string]string{"finding": findingID.String(), "description": "Corrected the finding wording"}); e != nil {
		t.Fatal(e)
	}
	current, _, _, _, _ = loadContractV2(w, profile, reviewID)
	if current.Review.Material.Findings[0].ID != findingID || current.Review.Material.Findings[0].Description != "Corrected the finding wording" {
		t.Fatal("finding set replaced identity or failed to correct text")
	}
	if e = edit("finding", "remove", map[string]string{"finding": findingID.String()}); e != nil {
		t.Fatal(e)
	}
	if e = edit("decision", "set", map[string]string{"decision": "approved", "reason": "Corrected the initial review"}); e != nil {
		t.Fatal(e)
	}
	output, e := reviewDocumentCommandV2(context.Background(), w, profile, nil, token, options{args: []string{"review", "evidence", "add"}, values: map[string]string{"description": "Reviewer observation", "uri": "https://example.test/review", "criterion": criterion.Title}})
	if e != nil {
		t.Fatal(e)
	}
	key := output.Data.(map[string]any)["local_key"].(string)
	assessment["evidence"] = "local:" + key
	if e = edit("assessment", "add", assessment); e != nil {
		t.Fatal(e)
	}
	remove := options{args: []string{"review", "evidence", "remove"}, values: map[string]string{"evidence": "local:" + key}}
	if _, e = reviewDocumentCommandV2(context.Background(), w, profile, nil, token, remove); e == nil {
		t.Fatal("staged evidence disappeared beneath its assessment")
	}
	if e = edit("assessment", "remove", map[string]string{"criterion": criterion.Title}); e != nil {
		t.Fatal(e)
	}
	if _, e = reviewDocumentCommandV2(context.Background(), w, profile, nil, token, remove); e != nil {
		t.Fatal(e)
	}
	current, _, _, _, e = loadContractV2(w, profile, reviewID)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(file.Issued, current.Issued) || len(current.Review.Material.Evidence) != 0 || len(current.Review.Material.EvidenceLinks) != 0 || len(current.Review.Material.CriterionLocalEvidence) != 0 {
		t.Fatal("draft correction changed issuer proof or left dangling references")
	}
}

func TestLocalMaterialCursorRejectsChangedDraftAndOtherSection(t *testing.T) {
	_, file := contractFixtureV2(t)
	items := []any{"first", "second"}
	value, e := materialPageV2(file, "assessment", items, options{values: map[string]string{"limit": "1"}})
	if e != nil {
		t.Fatal(e)
	}
	cursor := value.(map[string]any)["next_cursor"].(string)
	page, e := materialPageV2(file, "assessment", items, options{values: map[string]string{"limit": "1", "cursor": cursor}})
	if e != nil || page.(map[string]any)["items"].([]any)[0] != "second" {
		t.Fatalf("material continuation: %v", e)
	}
	if _, e = materialPageV2(file, "finding", items, options{values: map[string]string{"cursor": cursor}}); e == nil {
		t.Fatal("cursor crossed material section")
	}
	file.Execution.Progress.Summary = "Concurrent author edit"
	if _, e = materialPageV2(file, "assessment", items, options{values: map[string]string{"cursor": cursor}}); e == nil {
		t.Fatal("cursor silently traversed an edited draft")
	}
}

func TestActivityCompactHTTPReceiptRecoversLargeOriginalAndLocalConflict(t *testing.T) {
	w, profile, token, original, id := operationalFixtureV2(t)
	actor := d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}
	values := map[string]string{"name": "Large exact-version metadata", "uri": "https://example.test/" + strings.Repeat("u", 40<<10), "artifact-type": "code", "source-version": strings.Repeat("v", 60<<10), "content-hash": strings.Repeat("c", 60<<10)}
	intent, e := prepareActivityV2(context.Background(), w, profile, token, actor, id, options{args: []string{"work", "artifact", "add"}, values: values})
	if e != nil {
		t.Fatal(e)
	}
	cmd, e := decodeActivityIntentV2(profile, intent)
	if e != nil {
		t.Fatal(e)
	}
	artifactID := d.ID("0199ac10-0000-7000-8000-000000000040")
	activityID := d.ID("0199ac10-0000-7000-8000-000000000041")
	commandID := d.ID("0199ac10-0000-7000-8000-000000000042")
	artifact, e := d.NewArtifact(artifactID, cmd.Scope, "code", values["name"], values["uri"], "", values["content-hash"], values["source-version"], actor, nil, time.Now().UTC())
	if e != nil {
		t.Fatal(e)
	}
	accepted := sdk.CommandResult[a.WorkActivityResult]{CommandID: commandID, OutcomeRevision: 42, Value: a.WorkActivityResult{Activity: d.WorkActivity{ID: activityID, Scope: cmd.Scope, ContractID: id, WorkItemID: original.Execution.Material.Result.WorkItemID, Sequence: 42, Type: cmd.Type, Description: cmd.Description, PrincipalID: profile.Binding.PrincipalID, Actor: actor, RecordedAt: time.Now().UTC(), ArtifactRefs: []d.ID{artifactID}, ArtifactVersions: []d.SubmissionArtifact{{ArtifactID: artifactID, SourceVersion: artifact.SourceVersion, Checksum: artifact.Checksum}}}, Artifacts: []d.Artifact{artifact}}}
	response, e := json.Marshal(accepted)
	if e != nil {
		t.Fatal(e)
	}
	if len(response) <= 256<<10 {
		t.Fatal("fixture does not cover oversized original result")
	}
	_, _, _, frozen, _, _ := returnFixtureV2(t)
	var writes, reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, request *http.Request) {
		if request.Method == "POST" {
			writes.Add(1)
			if request.Header.Get("Idempotency-Key") != intent.IdempotencyKey {
				t.Error("append changed original key")
			}
			if e := json.NewEncoder(rw).Encode(map[string]any{"command_id": commandID, "outcome_revision": 42, "result_omitted": true}); e != nil {
				t.Error(e)
			}
			return
		}
		reads.Add(1)
		// Simulate an uncooperative local editor after server commit. Accepted
		// material must survive in a compact authenticated publication receipt.
		_, _, path, raw, e := loadContractV2(w, profile, id)
		if e != nil {
			t.Error(e)
			return
		}
		changed := original
		changed.Local.Pending = frozen.Local.Pending
		if e = w.WriteV2(path, changed, signing.Digest(raw)); e != nil {
			t.Error(e)
			return
		}
		state := a.SignedStateResult{NamespaceID: cmd.Scope.NamespaceID, Scope: &cmd.Scope, OperationName: intent.Operation, OperationCommandID: &commandID, OperationFingerprint: intent.RequestFingerprint, OperationPayload: base64.StdEncoding.EncodeToString(response), PayloadDigest: signing.Digest(response)}
		if e = json.NewEncoder(rw).Encode(state); e != nil {
			t.Error(e)
		}
	}))
	defer server.Close()
	client, _ := sdk.New(server.URL, string(token), server.Client())
	result, e := recoverActivityV2(context.Background(), w, profile, client, token, intent)
	if e == nil || result.CommandID != commandID {
		t.Fatalf("publication conflict lost committed result: %s %v", result.CommandID, e)
	}
	current, e := w.LoadProfileV2(profile.Name)
	if e != nil {
		t.Fatal(e)
	}
	if len(current.Local.PendingOperations) != 1 || current.Local.PendingOperations[0].State != "accepted_unmaterialized" {
		t.Fatal("accepted result was not retained for local publication")
	}
	pending := current.Local.PendingOperations[0]
	cached, e := base64.StdEncoding.Strict().DecodeString(pending.Response)
	if e != nil {
		t.Fatal(e)
	}
	if len(cached) > signing.MaxEnvelopeBytes || pending.Payload != intent.Payload || pending.RequestFingerprint != intent.RequestFingerprint {
		t.Fatal("large receipt changed frozen intention or exceeded protected local bound")
	}
	_, _, path, raw, e := loadContractV2(w, profile, id)
	if e != nil {
		t.Fatal(e)
	}
	if e = w.WriteV2(path, original, signing.Digest(raw)); e != nil {
		t.Fatal(e)
	}
	if _, e = recoverActivityV2(context.Background(), w, profile, client, token, intent); e != nil {
		t.Fatal(e)
	}
	if writes.Load() != 1 || reads.Load() != 1 {
		t.Fatalf("cached recovery rewrote server history: writes=%d reads=%d", writes.Load(), reads.Load())
	}
	file, _, _, _, e := loadContractV2(w, profile, id)
	if e != nil {
		t.Fatal(e)
	}
	refs := file.Execution.Material.Result.Artifacts
	if len(refs) != 1 || refs[0].ArtifactID != artifactID || refs[0].SourceVersion != artifact.SourceVersion || refs[0].Checksum != artifact.Checksum {
		t.Fatal("compact recovery lost exact artifact metadata")
	}
	current, _ = w.LoadProfileV2(profile.Name)
	if len(current.Local.PendingOperations) != 0 {
		t.Fatal("materialized compact receipt was not cleared")
	}
}

func TestContinuationCompactsActivityMetadataAndInnerEvidenceReferences(t *testing.T) {
	_, file := contractFixtureV2(t)
	result := file.Execution.Material.Result
	ids := []d.ID{}
	for i := range 10 {
		ids = append(ids, d.ID(fmt.Sprintf("0199ac10-0000-7000-8000-%012x", i+100)))
	}
	activity := d.WorkActivity{ID: ids[0], ContractID: result.ContractID, WorkItemID: result.WorkItemID, Type: d.WorkActivityProgress, Sequence: 9007199254741003, Description: strings.Repeat("a", 3000), ArtifactRefs: ids, EvidenceRefs: ids, ArtifactVersions: []d.SubmissionArtifact{{ArtifactID: ids[0], SourceVersion: strings.Repeat("v", 60<<10), Checksum: strings.Repeat("c", 60<<10)}}}
	page := compactRecentActivityV2(a.WorkActivityPage{Order: "desc", Items: []d.WorkActivity{activity}, NextCursor: "cursor", OutcomeRevision: 9007199254741003, HistoryThroughRevision: 9007199254741003})
	raw, e := json.Marshal(page)
	if e != nil {
		t.Fatal(e)
	}
	if len(raw) > 8192 || page["next_cursor"] != "cursor" || page["outcome_revision"] != "9007199254741003" {
		t.Fatal("activity context lost its bound, cursor or exact revision")
	}
	item := page["items"].([]any)[0].(map[string]any)
	omitted := item["omitted"].(map[string]int)
	if omitted["artifact_refs"] != 5 || omitted["evidence_refs"] != 5 || omitted["description_bytes"] != 1976 || omitted["artifact.source_version_bytes"] == 0 {
		t.Fatalf("hidden activity omissions: %+v", omitted)
	}
	file.Execution.Material.Result.CriterionEvidence = []d.SignedCriterionEvidence{{CriterionID: ids[0], CriterionRevision: 1, EvidenceIDs: ids}}
	draft := compactDraftV2(file)
	if draft["omitted"].(map[string]int)["criterion_evidence.evidence_ids"] != 5 {
		t.Fatal("inner criterion references escaped compact draft bound")
	}
}

func TestOversizedActivityInputDoesNotCreatePendingIntention(t *testing.T) {
	w, profile, token, _, id := operationalFixtureV2(t)
	_, e := prepareActivityV2(context.Background(), w, profile, token, d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}, id, options{args: []string{"work", "artifact", "add"}, values: map[string]string{"name": "Oversized", "uri": strings.Repeat("u", 300<<10), "artifact-type": "code"}})
	if e == nil {
		t.Fatal("oversized activity accepted locally")
	}
	current, e := w.LoadProfileV2(profile.Name)
	if e != nil || len(current.Local.PendingOperations) != 0 {
		t.Fatal("oversized local input stranded an intention")
	}
}

func TestHistoricalAcceptedResultVerifiesIssuerAndExactMaterial(t *testing.T) {
	_, profile, token, file, id := operationalFixtureV2(t)
	view, e := file.VerifyIssued(profile)
	if e != nil {
		t.Fatal(e)
	}
	scope := d.Scope{NamespaceID: profile.Binding.NamespaceID, OutcomeID: d.ID(view.Authority.OutcomeID)}
	submissionID := d.ID("0199ac10-0000-7000-8000-000000000050")
	factID := d.ID("0199ac10-0000-7000-8000-000000000051")
	material, e := signing.Canonical(file.Execution.Material.Result)
	if e != nil {
		t.Fatal(e)
	}
	digest := signing.Digest(material)
	receipt := signing.ReceiptPayload{Binding: signing.Binding{ProtocolVersion: 2, ServerID: profile.Binding.ServerID.String(), NamespaceID: scope.NamespaceID.String(), OutcomeID: scope.OutcomeID.String(), PrincipalID: profile.Binding.PrincipalID, SignerKeyID: profile.Binding.IssuerKeys[0].KeyID.String()}, ContractID: id.String(), WorkItemID: view.Authority.WorkItemID, SubmissionID: submissionID.String(), SubmissionDigest: digest, Accepted: true, LocalObligationClosed: true}
	var corrupt atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, request *http.Request) {
		state := a.SignedStateResult{NamespaceID: scope.NamespaceID, Scope: &scope}
		if strings.Contains(request.URL.Path, "/submission/") {
			state.SubmissionID, state.SubmissionDigest, state.MaterialPayload, state.AcceptanceFactID = &submissionID, digest, base64.StdEncoding.EncodeToString(material), &factID
		} else {
			copy := receipt
			if corrupt.Load() {
				copy.SubmissionDigest = signing.Digest([]byte("another accepted material"))
			}
			envelope, e := signing.Sign(signing.AcceptanceReceipt, copy, profile.Binding.IssuerKeys[0].KeyID.String(), ed25519.NewKeyFromSeed(make([]byte, 32)))
			if e != nil {
				t.Error(e)
				return
			}
			state.Envelope = &envelope
		}
		if e := json.NewEncoder(rw).Encode(state); e != nil {
			t.Error(e)
		}
	}))
	defer server.Close()
	client, _ := sdk.New(server.URL, string(token), server.Client())
	target := workTargetV2{ID: id, WorkID: d.ID(view.Authority.WorkItemID), Scope: scope, State: a.SignedStateResult{Contract: &a.SignedContractMetadata{SubmissionID: &submissionID, SpecDigest: view.Authority.SpecDigest, HolderPrincipalID: profile.Binding.PrincipalID}}}
	value, e := acceptedWorkResultV2(context.Background(), profile, client, target)
	if e != nil {
		t.Fatal(e)
	}
	result := value.(map[string]any)
	if result["summary"] != file.Execution.Material.Result.Summary || result["issuer_verified"] != true || result["remote_committed"] != true || result["local_staged"] != false {
		t.Fatalf("accepted material continuation lost binding: %+v", result)
	}
	corrupt.Store(true)
	if _, e = acceptedWorkResultV2(context.Background(), profile, client, target); e == nil {
		t.Fatal("valid issuer signature for different material was accepted")
	}
}

func TestProgressSetPreservesHistoricalCompletedMaterial(t *testing.T) {
	w, profile, token, file, id := operationalFixtureV2(t)
	_, _, path, raw, _ := loadContractV2(w, profile, id)
	file.Execution.Progress.Completed, file.Execution.Progress.Pending = []string{"Imported completed observation"}, []string{"Imported advisory note"}
	if e := w.WriteV2(path, file, signing.Digest(raw)); e != nil {
		t.Fatal(e)
	}
	_, e := materialCommandV2(context.Background(), w, profile, nil, token, "execution", options{args: []string{"work", "progress", "set"}, values: map[string]string{"summary": "Current advisory summary", "next-action": "Inspect actual criteria"}})
	if e != nil {
		t.Fatal(e)
	}
	current, _, _, _, e := loadContractV2(w, profile, id)
	if e != nil {
		t.Fatal(e)
	}
	progress := current.Execution.Progress
	if progress.Summary != "Current advisory summary" || len(progress.Completed) != 1 || progress.Completed[0] != "Imported completed observation" || len(progress.Pending) != 1 {
		t.Fatal("summary edit silently erased imported material")
	}
	if _, exposed := compactProgressV2(progress).(map[string]any)["pending"]; exposed {
		t.Fatal("legacy manual pending list became the new agent attention source")
	}
}
