package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type mutableCredentialLifecycleClock struct{ now time.Time }

func (c *mutableCredentialLifecycleClock) Now() time.Time { return c.now }

type failingCredentialUseObserver struct{ ports.SecurityStore }

func (f failingCredentialUseObserver) ObserveCredentialUse(context.Context, domain.ID, domain.ID, time.Time, time.Duration) error {
	return errors.New("observation unavailable")
}

func TestCredentialUseObservationFailureDoesNotBlockAuthentication(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	store := New()
	security := application.SecurityService{Store: store, Clock: &mutableCredentialLifecycleClock{now: now}, IDs: &credentialManagementIDs{next: 100}}
	ns := domain.MustParseID("0199f803-0000-7000-8000-000000000001")
	token := "memory-credential-observation-error-token-32"
	if err := security.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "credential observer"}, "admin", token); err != nil {
		t.Fatal(err)
	}
	security.Store = failingCredentialUseObserver{SecurityStore: store}
	if _, err := security.Authenticate(ctx, token); err != nil {
		t.Fatalf("best-effort last-used observation blocked authentication: %v", err)
	}
}

func TestMemoryCredentialLifecycleUpdateRotationAndObservation(t *testing.T) {
	ctx := context.Background()
	clock := &mutableCredentialLifecycleClock{now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)}
	store := New()
	security := application.SecurityService{Store: store, Clock: clock, IDs: &credentialManagementIDs{next: 10}}
	ns := domain.MustParseID("0199f802-0000-7000-8000-000000000001")
	adminToken := "memory-credential-lifecycle-admin-token-32"
	if err := security.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "credential lifecycle"}, "admin", adminToken); err != nil {
		t.Fatal(err)
	}
	admin, err := security.Authenticate(ctx, adminToken)
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := application.WithIdentity(ctx, admin)
	if err = security.Store.PutGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "worker", Permissions: []ports.Permission{ports.PermissionStateRead}}); err != nil {
		t.Fatal(err)
	}
	version, err := store.ReadSecurityAdminVersion(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	issue := ports.SecurityAdminCommand{NamespaceID: ns, ExpectedVersion: version, Operation: "issue_credential", CredentialName: "Worker", Credential: &ports.Credential{NamespaceID: ns, PrincipalID: "worker", Actor: admin.Actor, ExpiresAt: clock.Now().Add(2 * time.Hour)}}
	issued, oldToken, err := security.AdministrativeMutation(adminCtx, "memory-lifecycle-issue-0001", issue)
	if err != nil || oldToken == "" {
		t.Fatalf("issue: result=%+v err=%v", issued, err)
	}
	record, err := security.GetCredential(adminCtx, ns, issued.Credential.ID)
	if err != nil || record.Status != "active" || record.EffectivePolicy.PermittedOperations[0] != string(ports.PermissionStateRead) {
		t.Fatalf("detail: record=%+v err=%v", record, err)
	}
	page, err := security.ListCredentials(adminCtx, ns, ports.CredentialManagementQuery{PrincipalID: "worker", Limit: 1})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("list: page=%+v err=%v", page, err)
	}
	version, err = store.ReadSecurityAdminVersion(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := security.UpdateCredential(adminCtx, "memory-lifecycle-update-0001", application.CredentialUpdateIntent{
		NamespaceID: ns, CredentialID: record.Credential.ID, ExpectedNamespaceVersion: version,
		ExpectedMetadataRevision: record.Metadata.MetadataRevision, ExpectedPolicyRevision: record.EffectivePolicy.Version,
		Name: "Worker updated", Description: "memory adapter parity",
	})
	if err != nil || updated.CredentialMetadata.MetadataRevision != record.Metadata.MetadataRevision+1 {
		t.Fatalf("update: result=%+v err=%v", updated, err)
	}
	current, err := security.GetCredential(adminCtx, ns, record.Credential.ID)
	if err != nil {
		t.Fatal(err)
	}
	version, err = store.ReadSecurityAdminVersion(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	overlap := clock.Now().Add(20 * time.Minute)
	rotation := application.CredentialRotationIntent{NamespaceID: ns, CredentialID: current.Credential.ID, ExpectedNamespaceVersion: version, ExpectedMetadataRevision: current.Metadata.MetadataRevision, ExpectedPolicyRevision: current.EffectivePolicy.Version, ExpiresAt: clock.Now().Add(3 * time.Hour), OverlapUntil: overlap}
	rotated, newToken, err := security.RotateCredential(adminCtx, "memory-lifecycle-rotate-0001", rotation)
	if err != nil || newToken == "" {
		t.Fatalf("rotate: result=%+v err=%v", rotated, err)
	}
	replay, hidden, err := security.RotateCredential(adminCtx, "memory-lifecycle-rotate-0001", rotation)
	if err != nil || hidden != "" || !replay.IdempotentReplay || !replay.TokenOmitted || replay.Credential.ID != rotated.Credential.ID {
		t.Fatalf("rotation replay: result=%+v token_present=%t err=%v", replay, hidden != "", err)
	}
	oldRecord, err := security.GetCredential(adminCtx, ns, current.Credential.ID)
	if err != nil || oldRecord.Metadata.ReplacedByCredentialID != rotated.Credential.ID || oldRecord.Metadata.RotationOverlapUntil == nil {
		t.Fatalf("rotation provenance: record=%+v err=%v", oldRecord, err)
	}
	if _, err = security.Authenticate(ctx, oldToken); err != nil {
		t.Fatalf("old token should work during overlap: %v", err)
	}
	lastUsed, err := security.GetCredential(adminCtx, ns, current.Credential.ID)
	if err != nil || lastUsed.Metadata.LastUsedAt == nil {
		t.Fatalf("last-used observation: record=%+v err=%v", lastUsed, err)
	}
	clock.now = clock.Now().Add(time.Minute)
	if _, err = security.Authenticate(ctx, oldToken); err != nil {
		t.Fatal(err)
	}
	coalesced, err := security.GetCredential(adminCtx, ns, current.Credential.ID)
	if err != nil || coalesced.Metadata.LastUsedAt == nil || !coalesced.Metadata.LastUsedAt.Equal(*lastUsed.Metadata.LastUsedAt) {
		t.Fatalf("last-used observation not coalesced: record=%+v err=%v", coalesced, err)
	}
	if _, err = security.Authenticate(ctx, newToken); err != nil {
		t.Fatalf("replacement token: %v", err)
	}
}
