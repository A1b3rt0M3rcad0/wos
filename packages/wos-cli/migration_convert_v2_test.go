package woscli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

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
	interrupted := errors.New("publication interrupted")
	e = publishMigrationV2(context.Background(), source, destination, profile, token, intent, manifest, files, func(int) error { return interrupted })
	if !errors.Is(e, interrupted) {
		t.Fatal("fault did not interrupt after publication", e)
	}
	retained, e := destination.LoadProfileV2(profile.Name)
	must(e)
	if len(retained.Local.PendingOperations) != 1 {
		t.Fatal("partial migration lost original intent")
	}
	path := contractPathV2(profile.Name, contract.Contract.ID)
	copied, e := destination.ReadV2(path)
	must(e)
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
