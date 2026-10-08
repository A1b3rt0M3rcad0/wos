package acceptance_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/authentication/local"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	ht "github.com/A1b3rt0M3rcad0/wos/packages/wos-api/http"
	mt "github.com/A1b3rt0M3rcad0/wos/packages/wos-api/mcp"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type ids struct{ n atomic.Uint64 }

func (g *ids) NewID() (d.ID, error) {
	return d.ParseID(fmt.Sprintf("0199ff20-0000-7000-8000-%012x", g.n.Add(1)))
}

type clock struct{}

func (clock) Now() time.Time { return time.Now().UTC() }
func TestHTTPAcquisitionMCPResumeAndSDKReceipts(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	generator := &ids{}
	service, err := a.NewService(store, clock{}, generator)
	if err != nil {
		t.Fatal(err)
	}
	scope := d.Scope{NamespaceID: d.MustParseID("0199ff21-0000-7000-8000-000000000001"), OutcomeID: d.MustParseID("0199ff21-0000-7000-8000-000000000002")}
	now := time.Now().UTC()
	o, err := d.NewOutcome(scope.OutcomeID, scope.NamespaceID, "Protocol parity", "", "Independent client", d.PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	o.Lifecycle = d.OutcomeLifecycleActive
	w, err := d.NewWorkItem(d.MustParseID("0199ff21-0000-7000-8000-000000000003"), scope, "Focused task", "Execute only required work", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
	if err != nil {
		t.Fatal(err)
	}
	w.ContractsEnabled = true
	w.LastFencingToken = 9007199254740993
	tx, _ := store.Begin(ctx)
	if _, err := tx.Coordination().LockOutcome(ctx, scope); err != nil {
		t.Fatal(err)
	}
	if err := tx.Outcomes().Insert(ctx, o); err != nil {
		t.Fatal(err)
	}
	if err := tx.WorkItems().Insert(ctx, w); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Coordination().AdvanceOutcome(ctx, scope); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	auth, _ := local.New("contract-holder")
	identity := a.Identity{PrincipalID: auth.Principal(), Actor: auth.Actor(), NamespaceID: scope.NamespaceID}
	handler, err := ht.New(ht.Options{Prefix: "/api/v1", RequestTimeout: time.Second, Service: service, IDs: generator, LocalAuth: auth})
	if err != nil {
		t.Fatal(err)
	}
	protocol, err := mt.New(service, generator, mt.Options{LocalIdentity: &identity})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", handler)
	mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return protocol }, &mcp.StreamableHTTPOptions{JSONResponse: true}))
	server := httptest.NewServer(mux)
	defer server.Close()
	client, _ := sdk.New(server.URL, "", server.Client())
	capabilities, err := client.Capabilities(ctx)
	if err != nil || capabilities.AgentExecution || len(capabilities.TerminalCauses) != 3 {
		t.Fatalf("capabilities %v %v", capabilities, err)
	}
	got, err := client.AcquireWorkContract(ctx, "protocol-acquisition-0001", a.AcquireWorkContractCommand{Scope: scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version, TTLSeconds: 60})
	if err != nil {
		t.Fatal(err)
	}
	c := got.Value.Contract
	session, err := mcp.NewClient(&mcp.Implementation{Name: "contract-consumer", Version: "1"}, nil).Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	input, err := commands.Encode(a.ResumeWorkContractCommand{Scope: scope, ContractID: c.ID, Authority: a.ContractAuthority{ExecutionID: c.ExecutionID, FencingToken: c.FencingToken, SpecDigest: c.SpecDigest}, ExpectedLeaseVersion: c.LeaseVersion})
	if err != nil {
		t.Fatal(err)
	}
	response, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_resume_work_contract", Arguments: map[string]any{"idempotency_key": "protocol-takeover-0001", "command": json.RawMessage(input)}})
	if err != nil || response.IsError {
		t.Fatalf("MCP resume %v %v", response, err)
	}
	raw, _ := json.Marshal(response.StructuredContent)
	var resumed a.MutationResult[a.WorkContractResult]
	if err := json.Unmarshal(raw, &resumed); err != nil {
		t.Fatal(err)
	}
	if resumed.Value.Contract.FencingToken <= c.FencingToken || resumed.Value.Contract.ExecutionID == c.ExecutionID {
		t.Fatal("transport takeover lost precision")
	}
	if _, err := client.RenewWorkContract(ctx, "protocol-stale-renew-0001", a.RenewWorkContractCommand{Scope: scope, ContractID: c.ID, Authority: a.ContractAuthority{ExecutionID: c.ExecutionID, FencingToken: c.FencingToken, SpecDigest: c.SpecDigest}, ExpectedLeaseVersion: resumed.Value.Contract.LeaseVersion, TTLSeconds: 60}); err == nil {
		t.Fatal("stale HTTP execution accepted after MCP takeover")
	}
	receipt, err := client.GetCommandReceipt(ctx, scope.NamespaceID, got.CommandID)
	if err != nil || receipt.CommandID != got.CommandID {
		t.Fatalf("receipt %v %v", receipt, err)
	}
	history, err := client.WorkContractHistory(ctx, scope, ports.ContractFilter{Limit: 1}, "")
	if err != nil || len(history.Items) != 1 || history.Omitted["specs"] != 1 {
		t.Fatalf("history %v %v", history, err)
	}
	view, err := client.GetWorkContract(ctx, scope, c.ID)
	if err != nil || view.Value.Contract.LeaseVersion != resumed.Value.Contract.LeaseVersion {
		t.Fatal("HTTP/MCP state mismatch")
	}
	none, err := client.AcquireNextWorkContract(ctx, "protocol-next-empty-0001", a.AcquireNextWorkContractCommand{Scope: scope, Limit: 1})
	if err != nil || none.Value.Acquired || !none.Value.SearchComplete {
		t.Fatalf("next empty %v %v", none, err)
	}
	replay, err := client.AcquireNextWorkContract(ctx, "protocol-next-empty-0001", a.AcquireNextWorkContractCommand{Scope: scope, Limit: 1})
	if err != nil || !replay.IdempotentReplay || replay.Value.Acquired {
		t.Fatalf("empty intent replay %v %v", replay, err)
	}
	// A bounded scan cannot claim absence when the next candidate lies outside its window.
	second := w
	second.ID = d.MustParseID("0199ff21-0000-7000-8000-000000000004")
	second.CreatedAt = now.Add(time.Second)
	second.UpdatedAt = second.CreatedAt
	tx, _ = store.Begin(ctx)
	if err := tx.WorkItems().Insert(ctx, second); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Coordination().AdvanceOutcome(ctx, scope); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	originalEmpty, err := client.AcquireNextWorkContract(ctx, "protocol-next-empty-0001", a.AcquireNextWorkContractCommand{Scope: scope, Limit: 1})
	if err != nil || originalEmpty.Value.Acquired || !originalEmpty.IdempotentReplay {
		t.Fatal("empty intent silently changed after new work")
	}
	incomplete, err := client.AcquireNextWorkContract(ctx, "protocol-next-page-0001", a.AcquireNextWorkContractCommand{Scope: scope, Limit: 1})
	if err != nil || incomplete.Value.Acquired || incomplete.Value.SearchComplete || incomplete.Value.NextCursor == "" {
		t.Fatalf("bounded search %v %v", incomplete, err)
	}
	acquiredNext, err := client.AcquireNextWorkContract(ctx, "protocol-next-page-0002", a.AcquireNextWorkContractCommand{Scope: scope, Limit: 1, Cursor: incomplete.Value.NextCursor})
	if err != nil || !acquiredNext.Value.Acquired || acquiredNext.Value.Result.WorkItem.ID != second.ID {
		t.Fatalf("next candidate %v %v", acquiredNext, err)
	}

}
