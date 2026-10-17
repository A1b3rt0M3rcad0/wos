package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestTenantAdmissionIsolationAuthenticationAndRelease(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	resolve := func(r *http.Request) (application.Identity, error) {
		if r.Header.Get("Authorization") == "bad" {
			return application.Identity{}, errors.New("invalid")
		}
		id := domain.MustParseID("0199df20-0000-7000-8000-000000000001")
		if r.Header.Get("Authorization") == "other" {
			id = domain.MustParseID("0199df20-0000-7000-8000-000000000002")
		}
		return application.Identity{NamespaceID: id}, nil
	}
	h := tenantAdmissionHTTP(1, resolve, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := application.IdentityFromContext(r.Context()); !ok {
			t.Error("authenticated identity lost")
		}
		if r.URL.Path == "/block" {
			close(entered)
			<-release
		}
		if r.URL.Path == "/panic" {
			panic("test")
		}
		w.WriteHeader(204)
	}))
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/block", nil))
	}()
	<-entered
	request := func(token, path string) int {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Authorization", token)
		// A supplied tenant header never selects an admission bucket.
		r.Header.Set("X-Tenant-ID", "invented")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	if code := request("same", "/"); code != 429 {
		t.Fatalf("tenant cap = %d", code)
	}
	if code := request("other", "/"); code != 204 {
		t.Fatalf("other tenant blocked = %d", code)
	}
	if code := request("bad", "/"); code != 401 {
		t.Fatalf("unauthenticated request admitted = %d", code)
	}
	close(release)
	<-done
	if code := request("same", "/"); code != 204 {
		t.Fatal("slot leaked")
	}
	func() { defer func() { _ = recover() }(); request("same", "/panic") }()
	if code := request("same", "/"); code != 204 {
		t.Fatal("panic leaked tenant slot")
	}
}
