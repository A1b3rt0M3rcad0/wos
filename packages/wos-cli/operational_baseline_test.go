package woscli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

// A 422 is not proof that an earlier concurrent attempt cannot commit. The
// original signature must survive until an authoritative terminal decision.
func TestRejectedFrozenReturnPreservesOriginalIntent(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	project, profile, token, file, intent, _ := returnFixtureV2(t)
	intent.State, intent.Response = "sent_unknown", ""
	profile.Local.PendingOperations = []PendingOperationV2{intent}
	if err := profile.SealBinding(token); err != nil {
		t.Fatal(err)
	}
	w, err := OpenWorkspace(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	for path, value := range map[string]any{".wos/project.yaml": project, profilePathV2(profile.Name): profile, contractPathV2(profile.Name, *intent.ContractID): file} {
		if err = w.CreateV2(path, value); err != nil {
			t.Fatal(err)
		}
	}
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		code := "not_found"
		status := http.StatusNotFound
		if r.Method == http.MethodPost {
			posts++
			code, status = "invalid_argument", http.StatusUnprocessableEntity
		}
		rw.WriteHeader(status)
		_ = json.NewEncoder(rw).Encode(map[string]any{"error": map[string]any{"code": code, "message": "controlled baseline rejection"}})
	}))
	defer server.Close()
	client, err := sdk.New(server.URL, string(token), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, committed, err := recoverReturnV2(context.Background(), w, profile, client, token, intent, true)
	if err == nil || committed || posts != 1 {
		t.Fatalf("rejection must remain uncertain: committed=%v posts=%d error=%v", committed, posts, err)
	}
	current, err := w.LoadProfileV2(profile.Name)
	if err != nil {
		t.Fatal(err)
	}
	if len(current.Local.PendingOperations) != 1 || !reflect.DeepEqual(current.Local.PendingOperations[0].Payload, intent.Payload) {
		t.Fatal("rejection discarded or changed the original signed intention")
	}
	_, err = prepareReturnV2(context.Background(), w, current, client, token, a.SigningIdentityView{}, *intent.ContractID, "execution", "")
	if err == nil {
		t.Fatal("a rejected but unresolved return was silently re-signed")
	}
}
