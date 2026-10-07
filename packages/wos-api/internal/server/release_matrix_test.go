package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type releaseTransport struct {
	t         *testing.T
	url, kind string
	session   *mcp.ClientSession
	next      int
}

func (c *releaseTransport) call(name string, cmd any, key string) ([]byte, bool) {
	c.t.Helper()
	raw, err := commands.Encode(cmd)
	if err != nil {
		c.t.Fatal(err)
	}
	if c.kind == "mcp" {
		r, err := c.session.CallTool(context.Background(), &mcp.CallToolParams{Name: "wos_" + name, Arguments: map[string]any{"idempotency_key": key, "command": json.RawMessage(raw)}})
		if err != nil {
			c.t.Fatal(err)
		}
		data, err := json.Marshal(r.StructuredContent)
		if err != nil {
			c.t.Fatal(err)
		}
		return data, !r.IsError
	}
	body, _ := json.Marshal(map[string]any{"command": json.RawMessage(raw)})
	req, _ := http.NewRequest("POST", c.url+"/api/v1/commands/"+name, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	var value json.RawMessage
	if err = json.NewDecoder(resp.Body).Decode(&value); err != nil {
		c.t.Fatal(err)
	}
	return value, resp.StatusCode == 200
}
func releaseValue[T any](c *releaseTransport, name string, cmd any) T {
	c.t.Helper()
	c.next++
	raw, ok := c.call(name, cmd, fmt.Sprintf("release-%s-%s-%04d", c.kind, name, c.next))
	if !ok {
		c.t.Fatalf("%s failed: %s", name, raw)
	}
	var r application.MutationResult[T]
	if err := json.Unmarshal(raw, &r); err != nil {
		c.t.Fatal(err)
	}
	return r.Value
}
func releaseConnect(t *testing.T, url, kind string) *releaseTransport {
	t.Helper()
	c := &releaseTransport{t: t, url: url, kind: kind}
	if kind == "mcp" {
		client := mcp.NewClient(&mcp.Implementation{Name: "release-validation", Version: "1"}, nil)
		session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: url + "/mcp"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		c.session = session
		t.Cleanup(func() { session.Close() })
	}
	return c
}
func TestReleaseVerificationModesTransportStorageMatrix(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		cfg.MCP.Enabled = true
		r, err := OpenRuntime(cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		server := httptest.NewServer(r.Handler())
		defer server.Close()
		for _, kind := range []string{"http", "mcp"} {
			t.Run(kind, func(t *testing.T) {
				c := releaseConnect(t, server.URL, kind)
				outcome := releaseValue[domain.Outcome](c, "create_outcome", application.CreateOutcomeCommand{NamespaceID: domain.MustParseID("01a11750-0000-7000-8000-000000000001"), Title: "Verification " + kind, DesiredState: "same proof semantics", Priority: domain.PriorityNormal})
				for _, mode := range []domain.VerificationMode{domain.VerificationModeAttestation, domain.VerificationModeEvidenceReview, domain.VerificationModeExternalEvaluation} {
					live := releaseValue[domain.SuccessCriterion](c, "add_criterion", application.AddCriterionCommand{Owner: outcome.Ref(), ExpectedVersion: outcome.Version, Title: string(mode), Required: true, VerificationMode: mode})
					outcome.Version++
					command := application.RecordCriterionAssessmentCommand{Owner: outcome.Ref(), ExpectedVersion: outcome.Version, CriterionID: live.ID, CriterionRevision: 1, Result: domain.AssessmentResultMet, Rationale: "Review recorded by authenticated participant"}
					if mode != domain.VerificationModeAttestation {
						if _, ok := c.call("record_criterion_assessment", command, fmt.Sprintf("release-%s-invalid-%s", kind, mode)); ok {
							t.Fatalf("%s accepted missing proof", mode)
						}
						evidence := releaseValue[domain.Evidence](c, "register_evidence", application.RegisterEvidenceCommand{Scope: outcome.Scope(), EvidenceType: domain.EvidenceTypeTestResult, Description: "Test execution observed", SourceRef: domain.SourceReference{Provider: "test", URI: "https://example.test/result"}, CapturedAt: time.Now().UTC()})
						command.EvidenceIDs = []domain.ID{evidence.ID}
					}
					if mode == domain.VerificationModeExternalEvaluation {
						command.EvaluatorRef = &domain.EvaluatorRef{Provider: "release", ID: "independent", Version: "1"}
					}
					_ = releaseValue[domain.CriterionAssessment](c, "record_criterion_assessment", command)
					outcome.Version++
				}
				activated := releaseValue[domain.Outcome](c, "activate_outcome", application.ActivateOutcomeCommand{Scope: outcome.Scope(), ExpectedVersion: outcome.Version})
				outcome = activated
				work := releaseValue[domain.WorkItem](c, "create_work_item", application.CreateWorkItemCommand{Scope: outcome.Scope(), Title: "Explicit work", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo})
				claimed := releaseValue[domain.WorkItem](c, "claim_work_item", application.ClaimWorkItemCommand{Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version, TTL: time.Minute})
				_ = releaseValue[domain.WorkItem](c, "complete_work_item", application.CompleteWorkItemCommand{Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: claimed.Version, ClaimID: claimed.CurrentLease.ClaimID, FencingToken: claimed.CurrentLease.FencingToken, ResultSummary: "Delivered", Reason: "Done"})
				conclusion := releaseValue[domain.Outcome](c, "achieve_outcome", application.AchieveOutcomeCommand{Scope: outcome.Scope(), ExpectedVersion: outcome.Version, Reason: "All three verification modes satisfied"})
				if conclusion.Lifecycle != domain.OutcomeLifecycleAchieved || conclusion.CurrentConclusion == nil || len(conclusion.CurrentConclusion.Assessments) != 3 {
					t.Fatalf("incomplete proof snapshot: %+v", conclusion.CurrentConclusion)
				}
			})
		}
	})
}

// Simulate a transport losing the response only AFTER the real handler commits.
// A second transport must reconcile it without another write or another key.
func TestReleaseLostHTTPResponseReconcilesThroughMCP(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		cfg.MCP.Enabled = true
		r, err := OpenRuntime(cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		lost := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			committed := httptest.NewRecorder()
			r.Handler().ServeHTTP(committed, req)
			if committed.Code != 200 {
				t.Errorf("expected commit before response loss: %s", committed.Body.String())
			}
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
		}))
		defer lost.Close()
		cmd := application.CreateOutcomeCommand{NamespaceID: domain.MustParseID("01a11751-0000-7000-8000-000000000001"), Title: "Lost response", DesiredState: "one durable intent", Priority: domain.PriorityNormal}
		raw, _ := commands.Encode(cmd)
		body, _ := json.Marshal(map[string]any{"command": json.RawMessage(raw)})
		req, _ := http.NewRequest("POST", lost.URL+"/api/v1/commands/create_outcome", bytes.NewReader(body))
		req.Header.Set("Idempotency-Key", "release-lost-response-0001")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			t.Fatal("client should observe lost response")
		}
		healthy := httptest.NewServer(r.Handler())
		defer healthy.Close()
		c := releaseConnect(t, healthy.URL, "mcp")
		data, ok := c.call("create_outcome", cmd, "release-lost-response-0001")
		if !ok {
			t.Fatalf("reconciliation failed: %s", data)
		}
		var replay application.MutationResult[domain.Outcome]
		if err = json.Unmarshal(data, &replay); err != nil {
			t.Fatal(err)
		}
		if !replay.IdempotentReplay {
			t.Fatal("response loss created a second mutation")
		}
		again, ok := c.call("create_outcome", cmd, "release-lost-response-0001")
		var second application.MutationResult[domain.Outcome]
		json.Unmarshal(again, &second)
		if !ok || second.Value.ID != replay.Value.ID || second.OutcomeRevision != replay.OutcomeRevision {
			t.Fatal("replay changed identity/revision")
		}
		cmd.Title = "different intent"
		if _, ok = c.call("create_outcome", cmd, "release-lost-response-0001"); ok {
			t.Fatal("same key accepted changed intent")
		}
	})
}

func TestReleaseEveryCatalogMutationRequiresIdempotencyAndSharesSchema(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		cfg.MCP.Enabled = true
		r, err := OpenRuntime(cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		server := httptest.NewServer(r.Handler())
		defer server.Close()
		response, err := server.Client().Get(server.URL + "/api/v1/commands")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var catalog struct {
			Commands []commands.Descriptor `json:"commands"`
		}
		if err = json.NewDecoder(response.Body).Decode(&catalog); err != nil {
			t.Fatal(err)
		}
		client := releaseConnect(t, server.URL, "mcp")
		listed, err := client.session.ListTools(context.Background(), &mcp.ListToolsParams{})
		if err != nil {
			t.Fatal(err)
		}
		schemas := map[string]json.RawMessage{}
		for _, tool := range listed.Tools {
			raw, _ := json.Marshal(tool.InputSchema)
			var shape struct {
				Properties map[string]json.RawMessage `json:"properties"`
			}
			if err = json.Unmarshal(raw, &shape); err != nil {
				t.Fatal(err)
			}
			schemas[tool.Name] = shape.Properties["command"]
		}
		if len(catalog.Commands) < 78 {
			t.Fatalf("command catalog shrank to %d", len(catalog.Commands))
		}
		httpClient := releaseConnect(t, server.URL, "http")
		for _, descriptor := range catalog.Commands {
			t.Run(descriptor.Name, func(t *testing.T) {
				expected, _ := json.Marshal(descriptor.Schema)
				var a, b any
				json.Unmarshal(expected, &a)
				if err = json.Unmarshal(schemas["wos_"+descriptor.Name], &b); err != nil {
					t.Fatal("MCP command missing schema", err)
				}
				canonicalA, _ := json.Marshal(a)
				canonicalB, _ := json.Marshal(b)
				if !bytes.Equal(canonicalA, canonicalB) {
					t.Fatalf("HTTP/MCP schema differs for %s", descriptor.Name)
				}
				if _, ok := httpClient.call(descriptor.Name, struct{}{}, ""); ok {
					t.Fatal("HTTP mutation accepted missing idempotency key")
				}
				result, err := client.session.CallTool(context.Background(), &mcp.CallToolParams{Name: "wos_" + descriptor.Name, Arguments: map[string]any{"command": map[string]any{}}})
				if err == nil && !result.IsError {
					t.Fatal("MCP mutation accepted missing idempotency key")
				}
			})
		}
	})
}
