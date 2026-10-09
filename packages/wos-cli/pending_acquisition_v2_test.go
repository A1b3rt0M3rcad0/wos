package woscli

import (
	"context"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"strings"
	"testing"
)

func TestAcquisitionBatchSnapshotPreservesEditorAndIndependentIntents(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	project, profile, token := profileFixtureV2(t, "executor_a")
	w, e := OpenWorkspace(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	if e = w.CreateV2(".wos/project.yaml", project); e != nil {
		t.Fatal(e)
	}
	if e = w.CreateV2(profilePathV2(profile.Name), profile); e != nil {
		t.Fatal(e)
	}
	actor := d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}
	scope := d.Scope{NamespaceID: profile.Binding.NamespaceID, OutcomeID: project.Scope.DefaultOutcomeID}
	cmd := a.AcquireNextSignedWorkContractCommand{Scope: scope, SignerKeyID: profile.Signing.KeyID, Limit: 25, TTLSeconds: 300}
	intents, e := prepareAcquisitionBatchV2(context.Background(), w, profile, token, "AcquireNextSignedWorkContract", scope, actor, cmd, 5)
	if e != nil {
		t.Fatal(e)
	}
	current, e := w.LoadProfileV2(profile.Name)
	if e != nil {
		t.Fatal(e)
	}
	if len(current.Local.PendingOperations) != 5 {
		t.Fatal("entire batch was not persisted before network")
	}
	seen := map[string]bool{}
	for _, intent := range intents {
		if seen[intent.IdempotencyKey] || intent.State != "prepared" || intent.RequestFingerprint == "" {
			t.Fatal("batch intentions are not independent and frozen")
		}
		seen[intent.IdempotencyKey] = true
	}
	if _, e = prepareAcquisitionBatchV2(context.Background(), w, profile, token, "AcquireNextSignedWorkContract", scope, actor, cmd, 6); e == nil {
		t.Fatal("pending bound ignored")
	}
	e = mutateProfileV2(context.Background(), w, profile, token, func(value *ProfileV2) error {
		raw, e := w.ReadV2(profilePathV2(profile.Name))
		if e != nil {
			return e
		}
		edited := *value
		edited.Output.DefaultFormat = "text"
		if e = w.WriteV2(profilePathV2(profile.Name), edited, signing.Digest(raw)); e != nil {
			return e
		}
		value.Lease.RequestedTTLSeconds = 60
		return nil
	})
	if e == nil || !strings.Contains(e.Error(), "local_version_conflict") {
		t.Fatalf("editor conflict lost: %v", e)
	}
	preserved, e := w.LoadProfileV2(profile.Name)
	if e != nil {
		t.Fatal(e)
	}
	if preserved.Output.DefaultFormat != "text" || preserved.Lease.RequestedTTLSeconds != 300 || len(preserved.Local.PendingOperations) != 5 {
		t.Fatal("edited snapshot overwritten or batch discarded")
	}
}
