package server

import (
	"bytes"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserSessionCatalogAndRevocation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.Mode = AuthModeAPIToken
	cfg.Auth.BootstrapToken = strings.Repeat("a", 40)
	cfg.Auth.BootstrapNamespaceID = "0199d030-0000-7000-8000-000000000001"
	cfg.Auth.BootstrapNamespaceName = "Shared"
	cfg.Storage.SQLitePath = filepath.Join(t.TempDir(), "web.db")
	runtime, err := OpenRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	srv := httptest.NewServer(runtime.Handler())
	defer srv.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	call := func(method, path string, body any, origin string, key string) (int, []byte) {
		t.Helper()
		b, _ := json.Marshal(body)
		r, _ := http.NewRequest(method, srv.URL+path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		resp, err := client.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, raw
	}
	status, _ := call("GET", "/app/", nil, "", "")
	if status != 200 {
		t.Fatal(status)
	}
	status, _ = call("POST", "/session", map[string]string{"token": cfg.Auth.BootstrapToken}, "https://foreign.example", "")
	if status != 403 {
		t.Fatal("accepted foreign session origin", status)
	}
	status, _ = call("POST", "/session", map[string]string{"token": cfg.Auth.BootstrapToken}, srv.URL, "")
	if status != 200 {
		t.Fatal("login", status)
	}
	status, raw := call("GET", "/api/v1/namespaces", nil, "", "")
	if status != 200 || !bytes.Contains(raw, []byte("Shared")) {
		t.Fatalf("discovery: %d %s", status, raw)
	}
	payload := map[string]any{"command": map[string]any{"namespace_id": cfg.Auth.BootstrapNamespaceID, "title": "Human web result", "desired_state": "Continued by independent client", "priority": "normal"}}
	status, _ = call("POST", "/api/v1/commands/create_outcome", payload, "https://foreign.example", "web-create-00000001")
	if status != 403 {
		t.Fatalf("accepted CSRF: %d", status)
	}
	status, raw = call("POST", "/api/v1/commands/create_outcome", payload, srv.URL, "web-create-00000001")
	if status != 200 {
		t.Fatalf("create: %d %s", status, raw)
	}
	var created application.MutationResult[domain.Outcome]
	if err = json.Unmarshal(raw, &created); err != nil || created.Value.ID.IsZero() {
		t.Fatalf("invalid result: %s", raw)
	}
	status, raw = call("POST", "/api/v1/commands/create_outcome", payload, srv.URL, "web-create-00000001")
	var replay application.MutationResult[domain.Outcome]
	json.Unmarshal(raw, &replay)
	if status != 200 || !replay.IdempotentReplay || replay.Value.ID != created.Value.ID {
		t.Fatalf("replay: %d %s", status, raw)
	}
	u, _ := http.NewRequest("GET", srv.URL, nil)
	cookies := jar.Cookies(u.URL)
	if len(cookies) != 1 {
		t.Fatal("missing browser session")
	}
	sessionToken := cookies[0].Value
	status, _ = call("DELETE", "/session", nil, srv.URL, "")
	if status != 204 {
		t.Fatal(status)
	}
	req := httptest.NewRequest("GET", "/api/v1/namespaces", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: sessionToken})
	rec := httptest.NewRecorder()
	runtime.Handler().ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatal("logout did not revoke stored session", rec.Code)
	}
}
