package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestRuntimeHTTPPersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Storage.SQLitePath = filepath.Join(dir, "wos.db")
	cfg.Storage.MigrateOnStart = true
	cfg.HTTP.RequestTimeout = 2 * time.Second

	first, err := OpenRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	firstServer := httptest.NewServer(first.Handler())
	client := firstServer.Client()

	for _, path := range []string{"/livez", "/readyz"} {
		response, err := client.Get(firstServer.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, response.StatusCode)
		}
	}

	namespaceID := "0199ed00-0000-7000-8000-000000000001"
	createURL := firstServer.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes"
	request, err := http.NewRequest(
		http.MethodPost,
		createURL,
		bytes.NewBufferString(`{"title":"restart","desired_state":"survives runtime restart","priority":"normal"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "runtime-restart-create-0001")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusCreated {
		response.Body.Close()
		t.Fatalf("create status = %d, want 201", response.StatusCode)
	}
	var created application.MutationResult[domain.Outcome]
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	etag := response.Header.Get("ETag")
	response.Body.Close()

	patchRequest, err := http.NewRequest(
		http.MethodPatch,
		firstServer.URL+"/api/v1/namespaces/"+namespaceID+"/outcomes/"+created.Value.ID.String(),
		bytes.NewBufferString(`{"title":"restart updated"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	patchRequest.Header.Set("Content-Type", "application/json")
	patchRequest.Header.Set("Idempotency-Key", "runtime-restart-patch-0001")
	patchRequest.Header.Set("If-Match", etag)
	patchResponse, err := client.Do(patchRequest)
	if err != nil {
		t.Fatal(err)
	}
	if patchResponse.StatusCode != http.StatusOK {
		patchResponse.Body.Close()
		t.Fatalf("patch status = %d, want 200", patchResponse.StatusCode)
	}
	patchResponse.Body.Close()

	firstServer.Close()
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := OpenRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	secondServer := httptest.NewServer(second.Handler())
	defer secondServer.Close()

	getURL := secondServer.URL + "/api/v1/namespaces/" + namespaceID + "/outcomes/" + created.Value.ID.String()
	getResponse, err := secondServer.Client().Get(getURL)
	if err != nil {
		t.Fatal(err)
	}
	defer getResponse.Body.Close()
	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf("get after restart status = %d, want 200", getResponse.StatusCode)
	}
	var persisted application.ReadResult[domain.Outcome]
	if err := json.NewDecoder(getResponse.Body).Decode(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Value.ID != created.Value.ID {
		t.Fatalf("persisted id = %s, want %s", persisted.Value.ID, created.Value.ID)
	}
	if persisted.Value.Title != "restart updated" {
		t.Fatalf("persisted title = %q", persisted.Value.Title)
	}
}
