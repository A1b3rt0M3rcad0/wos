package wossdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWorkActivityOriginalResponseBoundAllowsInlineRecordExpansion(t *testing.T) {
	scope := d.Scope{NamespaceID: d.MustParseID("0199a555-0000-7000-8000-000000000001"), OutcomeID: d.MustParseID("0199a555-0000-7000-8000-000000000002")}
	now := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	actor := d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "worker"}
	activity := d.WorkActivity{ID: d.MustParseID("0199a555-0000-7000-8000-000000000003"), Scope: scope, ContractID: d.MustParseID("0199a555-0000-7000-8000-000000000004"), WorkItemID: d.MustParseID("0199a555-0000-7000-8000-000000000005"), Sequence: 1, Type: d.WorkActivityProgress, Description: "Registered bounded artifact observations", PrincipalID: "worker", Actor: actor, RecordedAt: now}
	command := a.AppendWorkActivityCommand{Scope: scope, ContractID: activity.ContractID, SignerKeyID: scope.NamespaceID, ExpectedContractVersion: 1, Type: activity.Type, Description: activity.Description, Authority: a.ContractAuthority{ExecutionID: scope.OutcomeID, FencingToken: 1, SpecDigest: "sha256:" + strings.Repeat("a", 64)}, AuthorityDigest: "sha256:" + strings.Repeat("b", 64)}
	result := a.WorkActivityResult{Activity: activity}
	for i := 0; i < 10; i++ {
		id := d.MustParseID(fmt.Sprintf("0199a555-0000-7000-8000-%012x", i+10))
		version := strings.Repeat("v", 10<<10)
		artifact, err := d.NewArtifact(id, scope, "report", fmt.Sprintf("Report %d", i), "file:///workspace/"+strings.Repeat("r", 8<<10), "text/plain", "sha256:original", version, actor, nil, now)
		if err != nil {
			t.Fatal(err)
		}
		command.Artifacts = append(command.Artifacts, a.SyncArtifactInput{LocalKey: fmt.Sprintf("report-%d", i), Artifact: a.RegisterArtifactCommand{Scope: scope, ArtifactType: artifact.ArtifactType, Name: artifact.Name, URI: artifact.URI, MediaType: artifact.MediaType, Checksum: artifact.Checksum, SourceVersion: artifact.SourceVersion}})
		result.Artifacts = append(result.Artifacts, artifact)
		result.Activity.ArtifactRefs = append(result.Activity.ArtifactRefs, id)
		result.Activity.ArtifactVersions = append(result.Activity.ArtifactVersions, d.SubmissionArtifact{ArtifactID: id, SourceVersion: version, Checksum: artifact.Checksum})
	}
	if err := result.Activity.Validate(); err != nil {
		t.Fatal(err)
	}
	payload, err := commands.Encode(command)
	if err != nil || len(payload) > 256<<10 {
		t.Fatalf("command fixture exceeds request bound: %d %v", len(payload), err)
	}
	body, err := json.Marshal(CommandResult[a.WorkActivityResult]{Value: result, CommandID: activity.ID, OutcomeRevision: 1})
	if err != nil || len(body) <= 256<<10 || len(body) > d.MaxSignedOperationResultBytes {
		t.Fatalf("invalid response expansion fixture: %d %v", len(body), err)
	}
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(body) }))
	defer endpoint.Close()
	client, err := New(endpoint.URL, "", endpoint.Client())
	if err != nil {
		t.Fatal(err)
	}
	observed, err := client.AppendWorkActivity(context.Background(), "bounded-activity-original", command)
	if err != nil || len(observed.Value.Artifacts) != 10 || observed.Value.Activity.ArtifactVersions[9].SourceVersion != result.Activity.ArtifactVersions[9].SourceVersion {
		t.Fatalf("activity original response truncated: %v", err)
	}
	if _, err = client.ExecuteCommand(context.Background(), "append_work_activity", "bounded-activity-raw", payload); err != nil {
		t.Fatalf("raw activity original response truncated: %v", err)
	}
	if _, err = commandResultLegacyBound(context.Background(), client, scope); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("legacy typed bound widened: %v", err)
	}
	if _, err = client.ExecuteCommand(context.Background(), "create_work_item", "bounded-legacy-raw", json.RawMessage(`{}`)); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("legacy raw bound widened: %v", err)
	}
	body = []byte(`{"value":"` + strings.Repeat("x", d.MaxSignedOperationResultBytes+(64<<10)) + `"}`)
	if _, err = client.AppendWorkActivity(context.Background(), "oversized-activity-original", command); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized activity response accepted: %v", err)
	}
}

func commandResultLegacyBound(ctx context.Context, client *Client, scope d.Scope) (CommandResult[d.WorkItem], error) {
	return command[d.WorkItem](ctx, client, "create_work_item", "bounded-legacy-typed", a.CreateWorkItemCommand{Scope: scope, ObjectiveID: &scope.OutcomeID, Title: "Legacy bound", Priority: d.PriorityNormal, Lifecycle: d.WorkItemLifecycleTodo})
}
