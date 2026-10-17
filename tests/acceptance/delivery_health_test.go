package acceptance_test

import (
	"context"
	"encoding/json"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Called by the real signed transport journey for memory, SQLite and PostgreSQL.
func deliveryHealthParity(t *testing.T, ctx context.Context, server *httptest.Server, operator, reader *sdk.Client, scope d.Scope, store ports.TransactionManager, adminToken, readerToken string) {
	t.Helper()
	q := a.DeliveryHealthQuery{TenantID: scope.TenantID, OutcomeID: &scope.OutcomeID}
	value, e := operator.GetDeliveryHealth(ctx, q)
	if _, supported := store.(ports.DeliveryStore); supported {
		if e != nil || value.State != "unknown" || value.Queue.Pending != 0 || value.Queue.AttemptsTotal != 0 {
			t.Fatal("SDK health", value, e)
		}
	} else if e == nil {
		t.Fatal("unsupported memory queue certified healthy")
	}
	if _, e = reader.GetDeliveryHealth(ctx, q); e == nil {
		t.Fatal("HTTP non-admin got health")
	}
	for _, test := range []struct {
		token  string
		denied bool
	}{{adminToken, false}, {readerToken, true}} {
		client := *server.Client()
		client.Transport = credentialTransport{base: server.Client().Transport, token: test.token}
		session, e := mcp.NewClient(&mcp.Implementation{Name: "health-parity", Version: "1"}, nil).Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp", HTTPClient: &client, DisableStandaloneSSE: true}, nil)
		if e != nil {
			t.Fatal(e)
		}
		result, e := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_delivery_health", Arguments: q})
		session.Close()
		if e != nil {
			t.Fatal(e)
		}
		_, supported := store.(ports.DeliveryStore)
		if result.IsError != (test.denied || !supported) {
			t.Fatalf("MCP auth/availability mismatch: %+v", result)
		}
		if !result.IsError {
			raw, e := json.Marshal(result.StructuredContent)
			if e != nil {
				t.Fatal(e)
			}
			var got a.DeliveryHealthResult
			if e = json.Unmarshal(raw, &got); e != nil {
				t.Fatal(e)
			}
			if got.State != value.State || got.Queue.Pending != value.Queue.Pending || got.Queue.TenantID != scope.TenantID {
				t.Fatal("HTTP/MCP mismatch")
			}
		}
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/v1/tenants/"+scope.TenantID.String()+"/operations/delivery-health", nil)
	response, e := server.Client().Do(req)
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatal("unauthenticated health allowed", response.StatusCode)
	}
}
