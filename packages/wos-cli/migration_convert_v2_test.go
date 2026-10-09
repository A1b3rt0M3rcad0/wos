package woscli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

func TestMigrationPartialPublicationReconcilesOriginalDocumentsAndPreservesEdits(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	source, config, contract := journalFixture(t)
	config.Connection.ServerURL = "http://127.0.0.1:8080"
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	must(source.WriteDocument(".wos/config.yaml", config))
	must(source.saveJSON(".wos/trust.json", config.Destination()))
	_, e := source.materialize(config, contract, contract.Contract.ID)
	must(e)
	draftPath := filepath.Join(stateDir(contract.Contract.WorkItemID, contract.Contract.ID), "result.yaml")
	var result ResultDraft
	must(source.ReadDocument(draftPath, &result))
	result.Material.Summary = "Unconfirmed operator draft survives migration"
	must(source.WriteDocument(draftPath, result))
	inventory, _, e := migrationInventoryV2(source)
	must(e)
	files, e := convertedLegacyFilesV2(source, config, inventory)
	must(e)
	if len(files) != 1 || files[0].Protocol != "legacy_unsigned" || files[0].Kind == "WOSContractFile" || files[0].Result.Material.Summary != result.Material.Summary {
		t.Fatal("conversion lost draft or fabricated signed document")
	}
	for _, original := range files[0].Originals {
		raw := []byte{}
		for _, chunk := range original.Chunks {
			b, e := base64.StdEncoding.Strict().DecodeString(chunk)
			must(e)
			raw = append(raw, b...)
		}
		sourceRaw, e := source.ReadV2(filepath.FromSlash(original.Path))
		must(e)
		if string(raw) != string(sourceRaw) || signing.Digest(raw) != original.Digest {
			t.Fatal("original bytes were not retained")
		}
	}
	destination, e := OpenWorkspace(t.TempDir())
	must(e)
	defer destination.Close()
	project, profile, token := profileFixtureV2(t, "executor")
	project.Connection.ServerURL = config.Connection.ServerURL
	project.Scope = config.Scope
	profile.Binding.ServerOrigin = config.Connection.ServerURL
	profile.Binding.NamespaceID = config.Scope.NamespaceID
	manifest := FrozenMigrationV2{Source: source.canonicalPath, InventoryDigest: inventory.InventoryDigest, Targets: []MigrationTargetV2{}}
	raw, e := EncodeV2Document(files[0])
	must(e)
	manifest.Targets = append(manifest.Targets, MigrationTargetV2{files[0].Contract.Contract.ID, signing.Digest(raw)})
	frozen, e := signing.Canonical(manifest)
	must(e)
	id, e := newLocalIDV2()
	must(e)
	intent := PendingOperationV2{ID: id, Operation: "WorkspaceMigration", State: "prepared", Scope: ProfilePendingScopeV2{NamespaceID: profile.Binding.NamespaceID}, IdempotencyKey: "migration-original-intent-0001", Payload: base64.StdEncoding.EncodeToString(frozen), PayloadDigest: signing.Digest(frozen)}
	profile.Local.PendingOperations = append(profile.Local.PendingOperations, intent)
	must(profile.SealBinding(token))
	must(destination.CreateV2(".wos/project.yaml", project))
	must(destination.CreateV2(profilePathV2(profile.Name), profile))
	child := exec.Command(os.Args[0], "-test.run=^TestMigrationPublicationChild$")
	child.Env = append(os.Environ(), "WOS_MIGRATION_SOURCE="+source.canonicalPath, "WOS_MIGRATION_DESTINATION="+destination.canonicalPath)
	output, e := child.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(e, &exit) || exit.ExitCode() != 23 {
		t.Fatalf("child did not die after publication: %v %s", e, output)
	}
	retained, e := destination.LoadProfileV2(profile.Name)
	must(e)
	if len(retained.Local.PendingOperations) != 1 {
		t.Fatal("partial migration lost original intent")
	}
	path := contractPathV2(profile.Name, contract.Contract.ID)
	copied, e := destination.ReadV2(path)
	must(e)
	// A source edit after process death must preserve the frozen intention.
	originalDraft, e := source.ReadV2(draftPath)
	must(e)
	changedSource := append(append([]byte{}, originalDraft...), []byte("\n# changed source after interruption\n")...)
	must(os.WriteFile(filepath.Join(source.canonicalPath, draftPath), changedSource, 0600))
	if e = publishMigrationV2(context.Background(), source, destination, profile, token, intent, manifest, files, nil); e == nil {
		t.Fatal("recovery accepted a changed source inventory")
	}
	preservedSource, e := source.ReadV2(draftPath)
	must(e)
	if string(preservedSource) != string(changedSource) {
		t.Fatal("changed source lost")
	}
	retained, e = destination.LoadProfileV2(profile.Name)
	must(e)
	if len(retained.Local.PendingOperations) != 1 {
		t.Fatal("changed source discarded frozen intention")
	}
	must(os.WriteFile(filepath.Join(source.canonicalPath, draftPath), originalDraft, 0600))
	// An editor can change the migrated draft; recovery must keep both copies.
	edited := append(append([]byte{}, copied...), []byte("\n# operator edits after interruption\n")...)
	must(os.WriteFile(filepath.Join(destination.canonicalPath, path), edited, 0600))
	if e = publishMigrationV2(context.Background(), source, destination, profile, token, intent, manifest, files, nil); e == nil {
		t.Fatal("recovery overwrote converted draft edit")
	}
	preserved, e := destination.ReadV2(path)
	must(e)
	if string(preserved) != string(edited) {
		t.Fatal("edited conversion lost")
	}
	must(os.WriteFile(filepath.Join(destination.canonicalPath, path), copied, 0600))
	must(publishMigrationV2(context.Background(), source, destination, profile, token, intent, manifest, files, nil))
	cleared, e := destination.LoadProfileV2(profile.Name)
	must(e)
	if len(cleared.Local.PendingOperations) != 0 {
		t.Fatal("verified migration retained pending intent")
	}
	after, _, e := migrationInventoryV2(source)
	must(e)
	if after.InventoryDigest != inventory.InventoryDigest {
		t.Fatal("migration changed originals")
	}
	var doc LegacyUnsignedFileV2
	must(DecodeV2Document(copied, &doc))
	encoded, e := json.Marshal(doc)
	must(e)
	if !json.Valid(encoded) {
		t.Fatal("legacy conversion has invalid public representation")
	}
}

func TestMigrationPublicationChild(t *testing.T) {
	sourcePath := os.Getenv("WOS_MIGRATION_SOURCE")
	if sourcePath == "" {
		return
	}
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	source, e := OpenWorkspace(sourcePath)
	must(e)
	defer source.Close()
	destination, e := OpenWorkspace(os.Getenv("WOS_MIGRATION_DESTINATION"))
	must(e)
	defer destination.Close()
	_, fixture, token := profileFixtureV2(t, "executor")
	profile, e := destination.LoadProfileV2(fixture.Name)
	must(e)
	if len(profile.Local.PendingOperations) != 1 {
		t.Fatal("missing original migration intention")
	}
	intent := profile.Local.PendingOperations[0]
	payload, e := base64.StdEncoding.Strict().DecodeString(intent.Payload)
	must(e)
	var manifest FrozenMigrationV2
	must(json.Unmarshal(payload, &manifest))
	inventory, config, e := migrationInventoryV2(source)
	must(e)
	files, e := convertedLegacyFilesV2(source, config, inventory)
	must(e)
	must(publishMigrationV2(context.Background(), source, destination, profile, token, intent, manifest, files, func(int) error { os.Exit(23); return nil }))
	t.Fatal("child did not reach publication")
}

func TestLegacyConversionPreservesDecimalCountersAboveJavaScriptPrecision(t *testing.T) {
	source, config, result := journalFixture(t)
	config.Connection.ServerURL = "http://127.0.0.1:8080"
	result.Contract.Version = d.Version(9007199254740995)
	result.Contract.LeaseVersion = d.Version(9007199254740997)
	result.Contract.WorkItemVersionAtAcquire = d.Version(9007199254740999)
	result.Contract.OutcomeRevisionAtAcquire = d.OutcomeRevision(9007199254741001)
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	must(source.WriteDocument(".wos/config.yaml", config))
	must(source.saveJSON(".wos/trust.json", config.Destination()))
	_, e := source.materialize(config, result, result.Contract.ID)
	must(e)
	inventory, _, e := migrationInventoryV2(source)
	must(e)
	files, e := convertedLegacyFilesV2(source, config, inventory)
	must(e)
	if len(files) != 1 {
		t.Fatal("missing conversion")
	}
	raw, e := EncodeV2Document(files[0])
	must(e)
	var decoded LegacyUnsignedFileV2
	must(DecodeV2Document(raw, &decoded))
	c := decoded.Contract.Contract
	if c.Version != result.Contract.Version || c.LeaseVersion != result.Contract.LeaseVersion || c.WorkItemVersionAtAcquire != result.Contract.WorkItemVersionAtAcquire || c.OutcomeRevisionAtAcquire != result.Contract.OutcomeRevisionAtAcquire {
		t.Fatal("decimal counter changed during conversion")
	}
}
