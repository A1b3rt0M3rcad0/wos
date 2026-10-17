package woscli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

func centralTestEnvironment(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "installation")
	t.Setenv("WOS_DATA_DIR", path)
	t.Setenv("WOS_WORKSPACE", "")
	t.Setenv("WOS_PROFILE", "")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	return path
}

func centralRun(t *testing.T, args ...string) (int, Output) {
	t.Helper()
	args = append(args, "--output", "json")
	var out, diagnostic bytes.Buffer
	code := Run(context.Background(), args, &out, &diagnostic)
	var result Output
	if e := json.Unmarshal(out.Bytes(), &result); e != nil {
		t.Fatalf("decode result for %v: %v; stdout %s; stderr %s", args, e, out.String(), diagnostic.String())
	}
	return code, result
}

func addCentralFixture(t *testing.T, name string, change func(*ProjectV2, *ProfileV2)) {
	t.Helper()
	w, e := openInstallation(false)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	value, before, e := readInstallation(w)
	if e != nil {
		t.Fatal(e)
	}
	if e = w.Mkdir(filepath.Join("profiles", name)); e != nil {
		t.Fatal(e)
	}
	store, e := openCentralProfile(w, name)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	project, profile, token := profileFixtureV2(t, name)
	if change != nil {
		change(&project, &profile)
	}
	if e = profile.SealBinding(token); e != nil {
		t.Fatal(e)
	}
	t.Setenv("TEST_WOS_TOKEN", string(token))
	if e = store.CreateV2(".wos/project.yaml", project); e != nil {
		t.Fatal(e)
	}
	if e = store.CreateV2(profilePathV2(name), profile); e != nil {
		t.Fatal(e)
	}
	value.Profiles = append(value.Profiles, name)
	if e = writeInstallation(w, value, before); e != nil {
		t.Fatal(e)
	}
}

func TestCentralInitNeverDiscoversOrCreatesProjectState(t *testing.T) {
	path := centralTestEnvironment(t)
	project := t.TempDir()
	t.Chdir(project)
	if e := os.Mkdir(".wos", 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(".wos/config.yaml", []byte("untrusted legacy document"), 0600); e != nil {
		t.Fatal(e)
	}
	if code, result := centralRun(t, "init"); code != 0 {
		t.Fatal(code, result.Error)
	}
	if code, result := centralRun(t, "init"); code != 0 {
		t.Fatal("idempotent init", code, result.Error)
	}
	if _, e := os.Stat(filepath.Join(path, "installation.json")); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(".wos/project.yaml"); !os.IsNotExist(e) {
		t.Fatal("central init touched caller's .wos")
	}
	if code, result := centralRun(t, "profile", "list"); code != 0 || !bytes.Contains(mustJSON(t, result.Data), []byte(`"items":[]`)) {
		t.Fatal(code, result)
	}
	t.Chdir(t.TempDir())
	if code, result := centralRun(t, "profile", "list"); code != 0 {
		t.Fatal("different CWD changed installation", code, result.Error)
	}
	if _, e := os.Stat(".wos"); !os.IsNotExist(e) {
		t.Fatal("operation created per-project data")
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, e := json.Marshal(value)
	if e != nil {
		t.Fatal(e)
	}
	return raw
}

func TestCentralProfilePrecedenceActivePersistenceAndDistinctDestinations(t *testing.T) {
	centralTestEnvironment(t)
	if code, result := centralRun(t, "init"); code != 0 {
		t.Fatal(result.Error)
	}
	addCentralFixture(t, "executor", nil)
	addCentralFixture(t, "reviewer", func(project *ProjectV2, profile *ProfileV2) {
		project.Connection.ServerURL = "https://other.example"
		project.Scope.TenantID = d.MustParseID("0199ac10-0000-7000-8000-000000000009")
		project.Scope.DefaultOutcomeID = ""
		profile.Binding.ServerOrigin = project.Connection.ServerURL
		profile.Binding.TenantID = project.Scope.TenantID
	})
	if code, _ := centralRun(t, "profile", "show"); code == 0 {
		t.Fatal("ambiguous profiles selected without authority")
	}
	if code, result := centralRun(t, "profile", "use", "executor"); code != 0 {
		t.Fatal(result.Error)
	}
	w, e := openInstallation(false)
	if e != nil {
		t.Fatal(e)
	}
	value, _, e := readInstallation(w)
	w.Close()
	if e != nil || value.ActiveProfile != "executor" {
		t.Fatal("active selection did not persist", e)
	}
	for _, test := range []struct{ explicit, env, expected string }{{"", "", "executor"}, {"", "reviewer", "reviewer"}, {"executor", "reviewer", "executor"}} {
		name, e := installationSelect(value, test.explicit, test.env)
		if e != nil || name != test.expected {
			t.Fatal(test, name, e)
		}
	}
	if _, e := installationSelect(value, "Executor", "reviewer"); e == nil {
		t.Fatal("invalid explicit selection fell back to environment")
	}
	t.Setenv("WOS_PROFILE", "reviewer")
	if code, result := centralRun(t, "profile", "show"); code != 0 || !bytes.Contains(mustJSON(t, result.Data), []byte("https://other.example")) {
		t.Fatal(code, result)
	}
	if code, result := centralRun(t, "profile", "show", "--profile", "executor"); code != 0 || bytes.Contains(mustJSON(t, result.Data), []byte("https://other.example")) {
		t.Fatal(code, result)
	}
}

func TestCentralInFlightProcessKeepsItsSelectedProfile(t *testing.T) {
	centralTestEnvironment(t)
	centralRun(t, "init")
	project, template, _ := profileFixtureV2(t, "executor")
	identityStarted := make(chan struct{})
	continueIdentity := make(chan struct{})
	capabilitiesRead := make(chan struct{}, 1)
	endpoint := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/security/signing-identity":
			close(identityStarted)
			select {
			case <-continueIdentity:
			case <-r.Context().Done():
				return
			}
			json.NewEncoder(rw).Encode(a.SigningIdentityView{Actor: d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}, ServerID: project.Connection.ExpectedServerID.String(), TenantID: project.Scope.TenantID, PrincipalID: template.Binding.PrincipalID, CredentialID: template.Binding.CredentialID})
		case "/api/v1/capabilities":
			capabilitiesRead <- struct{}{}
			json.NewEncoder(rw).Encode(a.ExecutionCapabilities{})
		default:
			json.NewEncoder(rw).Encode(a.SignedStateResult{TenantID: project.Scope.TenantID, Resource: "trust", ProtocolVersion: 3, Server: &d.ServerIdentity{ID: project.Connection.ExpectedServerID, IssuerKeyID: template.Binding.IssuerKeys[0].KeyID, PublicKey: template.Binding.IssuerKeys[0].PublicKey, Fingerprint: template.Binding.IssuerKeys[0].Fingerprint}})
		}
	}))
	defer endpoint.Close()
	addCentralFixture(t, "executor", func(project *ProjectV2, profile *ProfileV2) {
		project.Connection.ServerURL = endpoint.URL
		profile.Binding.ServerOrigin = endpoint.URL
	})
	addCentralFixture(t, "reviewer", func(project *ProjectV2, profile *ProfileV2) {
		project.Connection.ServerURL = "https://other.example"
		profile.Binding.ServerOrigin = project.Connection.ServerURL
	})
	centralRun(t, "profile", "use", "executor")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan int, 1)
	go func() {
		var out, diagnostic bytes.Buffer
		done <- Run(ctx, []string{"capabilities", "--output", "json"}, &out, &diagnostic)
	}()
	select {
	case <-identityStarted:
	case <-ctx.Done():
		t.Fatal("selected invocation did not reach initial server")
	}
	if code, result := centralRun(t, "profile", "use", "reviewer"); code != 0 {
		close(continueIdentity)
		t.Fatal("active switch blocked by in-flight process", result.Error)
	}
	close(continueIdentity)
	if code := <-done; code != 0 {
		t.Fatal("in-flight command followed changed global profile", code)
	}
	select {
	case <-capabilitiesRead:
	default:
		t.Fatal("query moved away from the initially selected server")
	}
}

func TestCentralProfileRemovalRefusesActivePendingAndContracts(t *testing.T) {
	centralTestEnvironment(t)
	centralRun(t, "init")
	addCentralFixture(t, "executor", nil)
	addCentralFixture(t, "reviewer", nil)
	centralRun(t, "profile", "use", "executor")
	if code, result := centralRun(t, "profile", "remove", "executor"); code == 0 || !strings.Contains(result.Error, "active") {
		t.Fatal(code, result)
	}
	w, e := openInstallation(false)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	store, e := openCentralProfile(w, "reviewer")
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	profile, before, e := store.readProfileSnapshotV2("reviewer")
	if e != nil {
		t.Fatal(e)
	}
	payload := []byte(`{"preserve":"exact intention"}`)
	profile.Local.PendingOperations = []PendingOperationV2{{ID: d.MustParseID("0199ac10-0000-7000-8000-000000000008"), Operation: "AcquireSignedWorkContract", State: "prepared", Scope: ProfilePendingScopeV2{TenantID: profile.Binding.TenantID}, IdempotencyKey: "preserved-pending-idempotency", Payload: "eyJwcmVzZXJ2ZSI6ImV4YWN0IGludGVudGlvbiJ9", PayloadDigest: signing.Digest(payload)}}
	if e = profile.SealBinding([]byte(os.Getenv("TEST_WOS_TOKEN"))); e != nil {
		t.Fatal(e)
	}
	if e = store.WriteV2(profilePathV2("reviewer"), profile, signing.Digest(before)); e != nil {
		t.Fatal(e)
	}
	if code, result := centralRun(t, "profile", "remove", "reviewer"); code == 0 || !strings.Contains(result.Error, "pending") {
		t.Fatal(code, result)
	}
	profile.Local.PendingOperations = []PendingOperationV2{}
	if e = profile.SealBinding([]byte(os.Getenv("TEST_WOS_TOKEN"))); e != nil {
		t.Fatal(e)
	}
	current, e := store.ReadV2(profilePathV2("reviewer"))
	if e != nil {
		t.Fatal(e)
	}
	if e = store.WriteV2(profilePathV2("reviewer"), profile, signing.Digest(current)); e != nil {
		t.Fatal(e)
	}
	if e = store.writeRawV2(".wos/profiles/reviewer/contract/preserved.yaml", []byte("not an authorized contract"), "", true); e != nil {
		t.Fatal(e)
	}
	if code, result := centralRun(t, "profile", "remove", "reviewer"); code == 0 || !strings.Contains(result.Error, "contracts") {
		t.Fatal(code, result)
	}
}

func TestCentralRemovalPreservesHistoryAndFencesOpenedStore(t *testing.T) {
	centralTestEnvironment(t)
	centralRun(t, "init")
	addCentralFixture(t, "executor", nil)
	addCentralFixture(t, "reviewer", nil)
	centralRun(t, "profile", "use", "executor")
	w, e := openInstallation(false)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	store, e := openCentralProfile(w, "reviewer")
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	history := []byte(`{"receipt":"preserved after selector removal"}`)
	historyPath := ".wos/profiles/reviewer/history/confirmed.json"
	if e = store.writeRawV2(historyPath, history, "", true); e != nil {
		t.Fatal(e)
	}
	if code, result := centralRun(t, "profile", "remove", "reviewer"); code != 0 {
		t.Fatal(result.Error)
	}
	actual, e := migrationRead(store, historyPath)
	if e != nil || !bytes.Equal(actual, history) {
		t.Fatal("removing a selector deleted durable local proof", e)
	}
	if _, e = store.LockV2(context.Background(), "reviewer", ""); e == nil {
		t.Fatal("previously opened process wrote a removed profile")
	}
	if code, _ := centralRun(t, "profile", "show", "reviewer"); code == 0 {
		t.Fatal("removed profile remained selectable")
	}
	value, _, e := readInstallation(w)
	if e != nil || len(value.Profiles) != 1 || value.Profiles[0] != "executor" {
		t.Fatal("profile removal damaged another profile", value, e)
	}
}

func centralLegacyFixture(t *testing.T) (*Workspace, string) {
	t.Helper()
	project, profile, token, contract, intent, receipt := returnFixtureV2(t)
	t.Setenv("TEST_WOS_TOKEN", string(token))
	profile.Local.PendingOperations = []PendingOperationV2{intent}
	if e := profile.SealBinding(token); e != nil {
		t.Fatal(e)
	}
	w, e := OpenWorkspace(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	for path, value := range map[string]any{".wos/project.yaml": project, profilePathV2(profile.Name): profile, contractPathV2(profile.Name, *intent.ContractID): contract, filepath.Join(".wos/profiles", profile.Name, "receipts", "preserved.yaml"): receipt} {
		if e = w.CreateV2(path, value); e != nil {
			w.Close()
			t.Fatal(e)
		}
	}
	return w, profile.Name
}

func TestCentralMigrationPreservesSignedPendingReceiptsAndFencesSource(t *testing.T) {
	path := centralTestEnvironment(t)
	source, name := centralLegacyFixture(t)
	defer source.Close()
	manifest, contents, e := centralMigrationInventory(source, path)
	if e != nil {
		t.Fatal(e)
	}
	if code, result := centralRun(t, "init", "--from", source.canonicalPath, "--dry-run"); code != 0 {
		t.Fatal(result.Error)
	}
	if _, e = os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("dry-run created installation")
	}
	if _, e = source.root.Lstat(centralMigrationMarker); !os.IsNotExist(e) {
		t.Fatal("dry-run fenced source")
	}
	if code, _ := centralRun(t, "init", "--from", source.canonicalPath); code == 0 {
		t.Fatal("migration without explicit writer drain accepted")
	}
	if code, result := centralRun(t, "init", "--from", source.canonicalPath, "--writers-drained"); code != 0 || !result.Committed {
		t.Fatal(code, result)
	}
	w, e := openInstallation(false)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	store, e := openCentralProfile(w, name)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	for _, file := range manifest.Files {
		actual, e := migrationRead(store, file.StorePath)
		original, sourceErr := migrationRead(source, file.SourcePath)
		if e != nil || sourceErr != nil || !bytes.Equal(actual, contents[name+"/"+file.StorePath]) || !bytes.Equal(original, actual) {
			t.Fatal("migration changed or lost protected bytes", file, e, sourceErr)
		}
	}
	if _, e = source.LockV2(context.Background(), name, ""); e == nil {
		t.Fatal("legacy source lock remained writable after fence")
	}
	if code, _ := centralRun(t, "profile", "show", "--workspace", source.canonicalPath); code == 0 {
		t.Fatal("explicit legacy source dispatch ignored fence")
	}
	if code, result := centralRun(t, "init", "--from", source.canonicalPath, "--writers-drained"); code != 0 {
		t.Fatal("exact frozen migration did not resume idempotently", result.Error)
	}
}

func TestCentralMigrationRejectsBusySourceCollisionAndUnsafePaths(t *testing.T) {
	centralTestEnvironment(t)
	source, name := centralLegacyFixture(t)
	defer source.Close()
	locked := make(chan struct{})
	finish := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		release, e := source.LockV2(context.Background(), name, "")
		if e != nil {
			return
		}
		close(locked)
		<-finish
		release()
	}()
	select {
	case <-locked:
	case <-time.After(time.Second):
		t.Fatal("could not establish source writer lock")
	}
	code, result := centralRun(t, "init", "--from", source.canonicalPath, "--writers-drained")
	close(finish)
	<-done
	if code == 0 || !strings.Contains(result.Error, "busy") {
		t.Fatal(code, result)
	}
	if _, e := source.root.Lstat(centralMigrationMarker); !os.IsNotExist(e) {
		t.Fatal("busy refusal fenced untouched source")
	}
	centralRun(t, "init")
	addCentralFixture(t, name, nil)
	if code, result = centralRun(t, "init", "--from", source.canonicalPath, "--writers-drained"); code == 0 || !strings.Contains(result.Error, "collision") {
		t.Fatal(code, result)
	}
	t.Setenv("WOS_DATA_DIR", "relative")
	if code, result = centralRun(t, "init"); code == 0 || !strings.Contains(result.Error, "absolute") {
		t.Fatal(code, result)
	}
	if runtime.GOOS != "windows" {
		root := t.TempDir()
		link := filepath.Join(root, "linked")
		if e := os.Symlink(t.TempDir(), link); e != nil {
			t.Fatal(e)
		}
		t.Setenv("WOS_DATA_DIR", filepath.Join(link, "data"))
		if code, result = centralRun(t, "init"); code == 0 || !strings.Contains(result.Error, "real directories") {
			t.Fatal(code, result)
		}
	}
}

func TestCentralMigrationResumePreservesObservedStagingEdits(t *testing.T) {
	path := centralTestEnvironment(t)
	source, name := centralLegacyFixture(t)
	defer source.Close()
	manifest, contents, e := centralMigrationInventory(source, path)
	if e != nil {
		t.Fatal(e)
	}
	raw := mustJSON(t, manifest)
	if e = source.writeRawV2(centralMigrationMarker, raw, "", true); e != nil {
		t.Fatal(e)
	}
	w, e := openInstallation(true)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	stage := filepath.Join("migration-staging", strings.TrimPrefix(signing.Digest(raw), "sha256:"), name)
	if e = w.Mkdir(stage); e != nil {
		t.Fatal(e)
	}
	store, e := OpenWorkspace(filepath.Join(path, stage))
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	file := manifest.Files[0]
	edited := []byte("observed staging edit must survive")
	if e = store.writeRawV2(file.StorePath, edited, "", true); e != nil {
		t.Fatal(e)
	}
	if code, result := centralRun(t, "init", "--from", source.canonicalPath, "--writers-drained"); code == 0 || !strings.Contains(result.Error, "staging edit") {
		t.Fatal(code, result)
	}
	actual, e := migrationRead(store, file.StorePath)
	if e != nil || !bytes.Equal(actual, edited) {
		t.Fatal("migration overwrote staging edit", e)
	}
	// Restore the fixture's original bytes to emulate explicit recovery of the
	// interrupted copy; production migration itself never overwrites that edit.
	if e = store.writeRawV2(file.StorePath, contents[name+"/"+file.StorePath], signing.Digest(edited), false); e != nil {
		t.Fatal(e)
	}
	// Windows directory handles opened by os.OpenRoot do not share delete
	// access. Finish inspecting the fixture before migration publishes it by
	// renaming the staging directory; production closes its own handle too.
	if e = store.Close(); e != nil {
		t.Fatal(e)
	}
	if code, result := centralRun(t, "init", "--from", source.canonicalPath, "--writers-drained"); code != 0 {
		t.Fatal("original frozen copy could not resume", result.Error)
	}
}

func TestCentralMigrationRefusesAliasedFilesWithoutChangingSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix link fixtures; Windows reparse/file identity guards have separate tests")
	}
	for _, linkKind := range []string{"symlink", "hardlink"} {
		t.Run(linkKind, func(t *testing.T) {
			path := centralTestEnvironment(t)
			source, name := centralLegacyFixture(t)
			defer source.Close()
			outside := filepath.Join(t.TempDir(), "protected-original")
			original := []byte("original must not be modified")
			if e := os.WriteFile(outside, original, 0600); e != nil {
				t.Fatal(e)
			}
			directory := filepath.Join(".wos/profiles", name, "history")
			if e := source.Mkdir(directory); e != nil {
				t.Fatal(e)
			}
			alias := filepath.Join(source.canonicalPath, directory, "alias.json")
			var e error
			if linkKind == "symlink" {
				e = os.Symlink(outside, alias)
			} else {
				e = os.Link(outside, alias)
			}
			if e != nil {
				t.Fatal(e)
			}
			if code, _ := centralRun(t, "init", "--from", source.canonicalPath, "--dry-run"); code == 0 {
				t.Fatal("migration accepted an aliased file")
			}
			if _, e = os.Stat(path); !os.IsNotExist(e) {
				t.Fatal("refused inventory created destination")
			}
			actual, e := os.ReadFile(outside)
			if e != nil || !bytes.Equal(actual, original) {
				t.Fatal("refused inventory changed original", e)
			}
		})
	}
}
