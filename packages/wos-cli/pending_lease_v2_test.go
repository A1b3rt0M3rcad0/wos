package woscli

import (
	"context"
	"encoding/base64"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"testing"
)

func TestLeaseIntentionFreezesExactAuthorityAndPausesConflictingReturn(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	project, profile, token := profileFixtureV2(t, "executor_a")
	_, file := contractFixtureV2(t)
	view, e := file.VerifyIssued(profile)
	if e != nil {
		t.Fatal(e)
	}
	id := d.ID(view.Authority.ContractID)
	w, e := OpenWorkspace(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	must(w.CreateV2(".wos/project.yaml", project))
	must(w.CreateV2(profilePathV2(profile.Name), profile))
	must(w.CreateV2(contractPathV2(profile.Name, id), file))
	actor := d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}
	intent, e := prepareLeaseV2(context.Background(), w, profile, token, actor, id, "execution", false, 300)
	must(e)
	raw, e := base64.StdEncoding.Strict().DecodeString(intent.Payload)
	must(e)
	var frozen a.RenewSignedWorkContractCommand
	must(signing.DecodeStrict(raw, &frozen, signing.MaxPayloadBytes))
	if frozen.AuthorityDigest != signing.Digest(file.Issued.Authority.Payload) || frozen.Authority.FencingToken != d.FencingToken(view.Authority.FencingToken) || frozen.ExpectedLeaseVersion != d.Version(view.Authority.LeaseVersion) || intent.ContractID == nil || *intent.ContractID != id {
		t.Fatal("lease intention lost exact authority binding")
	}
	if _, e = prepareLeaseV2(context.Background(), w, profile, token, actor, id, "execution", true, 0); e == nil {
		t.Fatal("unresolved renewal permitted another takeover intention")
	}
	current, e := w.LoadProfileV2(profile.Name)
	must(e)
	if len(current.Local.PendingOperations) != 1 {
		t.Fatal("conflict replaced original intention")
	}
	changed := intent
	other := project.Scope.DefaultOutcomeID
	changed.ContractID = &other
	if _, e = pendingIndexV2(&current, changed); e == nil {
		t.Fatal("pending contract substitution accepted")
	}
	must(mutateProfileV2(context.Background(), w, profile, token, func(p *ProfileV2) error { p.Local.PendingOperations = nil; return nil }))
	old, e := w.ReadV2(contractPathV2(profile.Name, id))
	must(e)
	// Any unresolved/accepted final marker pauses maintenance; it is not treated
	// as permission to delete or as an authenticated acceptance by this test.
	file.Local.AcceptanceReceipt = &file.Issued.Specification
	must(w.WriteV2(contractPathV2(profile.Name, id), file, signing.Digest(old)))
	if _, e = prepareLeaseV2(context.Background(), w, profile, token, actor, id, "execution", false, 300); e == nil {
		t.Fatal("final marker failed to pause renewal")
	}
}
