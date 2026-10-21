package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestLegacyCredentialRouteAcceptsAndReplaysIssuanceMetadata(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	store := memory.New()
	service, err := application.NewService(store, fixedClock{value: now}, &sequenceIDs{prefix: "0199ec01", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	security := &application.SecurityService{Store: store, Clock: fixedClock{value: now}, IDs: &sequenceIDs{prefix: "0199ec02", next: 1}}
	ns := domain.MustParseID("0199ec03-0000-7000-8000-000000000001")
	adminToken := "http-credential-route-admin-token-at-least-32"
	if err := security.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "http credentials"}, "admin", adminToken); err != nil {
		t.Fatal(err)
	}
	admin, err := security.Authenticate(ctx, adminToken)
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := application.WithIdentity(ctx, admin)
	if err := security.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "worker", Permissions: []ports.Permission{ports.PermissionStateRead}}); err != nil {
		t.Fatal(err)
	}
	resolver := func(r *http.Request) (application.Identity, error) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			return application.Identity{}, domain.NewError(domain.ErrorCodeForbidden, "credential required")
		}
		return security.Authenticate(r.Context(), strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
	}
	handler, err := New(Options{
		Prefix: "/api/v1", RequestTimeout: 2 * time.Second, Service: service,
		IDs: &sequenceIDs{prefix: "0199ec04", next: 1}, Security: security, ResolveIdentity: resolver,
	})
	if err != nil {
		t.Fatal(err)
	}
	version, err := store.ReadSecurityAdminVersion(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"expected_namespace_version": version,
		"principal_id":               "worker",
		"actor_ref":                  domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker"},
		"expires_at":                 now.Add(time.Hour),
		"credential_name":            "HTTP worker key",
		"credential_description":     "Issued by the legacy route with a receipt.",
		"permissions":                []ports.Permission{ports.PermissionStateRead},
		"allowed_outcome_ids":        []domain.ID{},
		"allowed_signing_key_ids":    []domain.ID{},
	})
	if err != nil {
		t.Fatal(err)
	}
	endpoint := "/api/v1/namespaces/" + ns.String() + "/credentials"
	req := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Idempotency-Key", "http-credential-route-issue-0001")
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, req)
	if first.Code != http.StatusCreated {
		t.Fatalf("issue status=%d body=%s", first.Code, first.Body.String())
	}
	var issued struct {
		Result struct {
			Credential         *ports.Credential         `json:"credential"`
			CredentialMetadata *ports.CredentialMetadata `json:"credential_metadata"`
			IdempotentReplay   bool                      `json:"idempotent_replay"`
			TokenOmitted       bool                      `json:"token_omitted"`
		} `json:"result"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	if issued.Token == "" || issued.Result.Credential == nil || issued.Result.CredentialMetadata == nil || issued.Result.CredentialMetadata.Name != "HTTP worker key" || issued.Result.CredentialMetadata.Description != "Issued by the legacy route with a receipt." {
		t.Fatalf("route did not return issuance metadata and initial token: %+v", issued)
	}
	if err := security.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "worker", Permissions: []ports.Permission{}}); err != nil {
		t.Fatal(err)
	}

	replayRequest := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	replayRequest.Header.Set("Authorization", "Bearer "+adminToken)
	replayRequest.Header.Set("Idempotency-Key", "http-credential-route-issue-0001")
	replay := httptest.NewRecorder()
	handler.ServeHTTP(replay, replayRequest)
	if replay.Code != http.StatusCreated {
		t.Fatalf("replay status=%d body=%s", replay.Code, replay.Body.String())
	}
	var replayed struct {
		Result struct {
			Credential       *ports.Credential `json:"credential"`
			IdempotentReplay bool              `json:"idempotent_replay"`
			TokenOmitted     bool              `json:"token_omitted"`
		} `json:"result"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(replay.Body.Bytes(), &replayed); err != nil {
		t.Fatal(err)
	}
	if replayed.Token != "" || !replayed.Result.IdempotentReplay || !replayed.Result.TokenOmitted || replayed.Result.Credential == nil || replayed.Result.Credential.ID != issued.Result.Credential.ID {
		t.Fatalf("route replay exposed a token or duplicated issuance: %+v", replayed)
	}
	if strings.Contains(replay.Body.String(), issued.Token) {
		t.Fatal("replay response contains the bearer token")
	}
}
