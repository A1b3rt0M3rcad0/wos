package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestWorkContractCredentialBindingPersistsAndIsCrossChecked(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "contracts.db"))
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	work := contractStorageFixture(t, store, now)
	service, err := a.NewService(store, sqliteFixedClock{now}, &sqliteSequenceIDs{prefix: "0199ff29", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	actor := d.ActorRef{Kind: d.ActorKindHuman, Provider: "local", ID: "sqlite-human"}
	cc := sqliteCommandContext("0199ff10-0000-7000-8000-000000000010", "contracts-v1-storage-binding")
	credential := ports.Credential{ID: testID("0199f282-0000-7000-8000-000000000001"), NamespaceID: work.Scope.NamespaceID, PrincipalID: cc.PrincipalID, Actor: actor, Digest: "sqlite-contract-root", ExpiresAt: now.Add(time.Hour)}
	if err := store.PutCredential(ctx, credential); err != nil {
		t.Fatal(err)
	}
	identity := a.WithIdentity(ctx, a.Identity{CredentialID: credential.ID, CredentialDigest: credential.Digest, NamespaceID: credential.NamespaceID, PrincipalID: credential.PrincipalID, Actor: credential.Actor})
	result, err := service.AcquireWorkContract(identity, cc, a.AcquireWorkContractCommand{
		Scope: work.Scope, WorkItemID: work.ID, ExpectedWorkItemVersion: work.Version, TTLSeconds: 30,
	})
	if err != nil {
		t.Fatalf("AcquireWorkContract() error = %v", err)
	}
	want := result.Value.Contract
	if want.CredentialBindingVersion != d.WorkContractCredentialBindingV1 || want.HolderCredentialID != credential.ID {
		t.Fatalf("acquired credential binding = version %d holder %s", want.CredentialBindingVersion, want.HolderCredentialID)
	}

	uow, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	repo := uow.(ports.WorkContractUnitOfWork).WorkContracts()
	stored, err := repo.Get(ctx, work.Scope, want.ID)
	if err != nil {
		t.Fatal(err)
	}
	page, err := repo.List(ctx, work.Scope, ports.ContractFilter{Limit: 100})
	if err != nil || len(page) != 1 || page[0].HolderCredentialID != credential.ID {
		t.Fatalf("bound contract list = %+v err=%v", page, err)
	}
	if stored.CredentialBindingVersion != want.CredentialBindingVersion || stored.HolderCredentialID != credential.ID || stored.SessionCredentialID != nil {
		t.Fatalf("restart state lost binding: %+v", stored)
	}
	if err := uow.Rollback(); err != nil {
		t.Fatal(err)
	}

	if _, err := store.db.ExecContext(ctx, `UPDATE work_contracts SET holder_credential_id=? WHERE namespace_id=? AND outcome_id=? AND id=?`, "0199f282-0000-7000-8000-000000000099", work.Scope.NamespaceID.String(), work.Scope.OutcomeID.String(), want.ID.String()); err != nil {
		t.Fatal(err)
	}
	uow, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	if _, err := uow.(ports.WorkContractUnitOfWork).WorkContracts().Get(ctx, work.Scope, want.ID); err == nil {
		t.Fatal("repository accepted credential columns that differed from contract state")
	}
}
