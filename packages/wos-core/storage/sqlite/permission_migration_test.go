package sqlite

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type permissionMigrationFixture struct {
	security   application.SecurityService
	store      ports.SecurityStore
	adminCtx   context.Context
	namespace  domain.ID
	credential ports.Credential
	principal  string
}

func newPermissionMigrationFixture(t *testing.T, backend string) permissionMigrationFixture {
	t.Helper()
	ctx := context.Background()
	var store ports.SecurityStore
	if backend == "memory" {
		store = memory.New()
	} else {
		store = openTestStore(t, filepath.Join(t.TempDir(), "permission-migration.db"))
	}
	clock := sqliteFixedClock{now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)}
	security := application.SecurityService{Store: store, Clock: clock, IDs: &sqliteSequenceIDs{prefix: "0199a300", next: 1}}
	namespace := testID("0199a300-0000-7000-8000-000000000001")
	adminToken := "permission-migration-admin-token-32-chars"
	if err := security.Bootstrap(ctx, ports.Namespace{ID: namespace, Name: "permission migration"}, "admin", adminToken); err != nil {
		t.Fatal(err)
	}
	admin, err := security.Authenticate(ctx, adminToken)
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := application.WithIdentity(ctx, admin)
	principal := "legacy-executor"
	if err = security.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: namespace, PrincipalID: principal, Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionWorkWrite}}); err != nil {
		t.Fatal(err)
	}
	actor := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "legacy-executor"}
	credential, _, err := security.IssueCredential(adminCtx, namespace, principal, actor, clock.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	policy := domain.CredentialPolicy{
		NamespaceID: namespace, CredentialID: credential.ID, PrincipalID: principal,
		PermittedOperations: []string{string(ports.PermissionStateRead), string(ports.PermissionWorkWrite)},
		AcceptanceFloor:     domain.AcceptanceDirect, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1,
	}
	if _, err = security.SigningMutation(adminCtx, "configure-permission-migration-policy", application.SigningSecurityIntent{
		NamespaceID: namespace, ExpectedNamespaceVersion: 1, Operation: "set_credential_policy", CredentialPolicy: &policy,
	}); err != nil {
		t.Fatal(err)
	}
	return permissionMigrationFixture{security: security, store: store, adminCtx: adminCtx, namespace: namespace, credential: credential, principal: principal}
}

func TestPermissionMigrationDryRunApplyReplayAndStaleCAS(t *testing.T) {
	for _, backend := range []string{"memory", "sqlite"} {
		t.Run(backend, func(t *testing.T) {
			f := newPermissionMigrationFixture(t, backend)
			reader := f.store.(ports.PermissionCompatibilityStore)
			request := application.PermissionMigrationPreviewRequest{NamespaceID: f.namespace, PrincipalID: f.principal, CredentialID: f.credential.ID}
			preview, err := f.security.PreviewPermissionMigration(f.adminCtx, request)
			if err != nil {
				t.Fatalf("read-only preview failed: %v", err)
			}
			if preview.MapVersion != application.PermissionCompatibilityMapVersion || !reflect.DeepEqual(preview.EffectiveCapabilitiesBefore, []ports.Permission{ports.PermissionStateRead, ports.PermissionWorkWrite}) {
				t.Fatalf("unexpected dry-run set: %+v", preview)
			}
			before := preview.Snapshot
			executorCtx := application.WithIdentity(context.Background(), application.Identity{
				NamespaceID: f.namespace, PrincipalID: f.principal, Actor: f.credential.Actor,
				CredentialDigest: f.credential.Digest,
			})
			legacyWorkRequest := ports.AuthorizationRequest{
				NamespaceID: f.namespace, PrincipalID: f.principal,
				Permission: ports.PermissionTaskCreate, LegacyPermission: ports.PermissionWorkWrite,
			}
			if err = f.security.Authorize(executorCtx, legacyWorkRequest); err == nil {
				t.Fatal("unmigrated legacy grant/policy authorized a granular command")
			}
			afterDryRun, err := reader.ReadPermissionMigration(context.Background(), ports.PermissionMigrationRead{NamespaceID: f.namespace, PrincipalID: f.principal, CredentialID: f.credential.ID})
			if err != nil || !reflect.DeepEqual(before, afterDryRun) {
				t.Fatalf("dry-run changed persistent grant/policy/version: %+v => %+v (%v)", before, afterDryRun, err)
			}
			if backend == "sqlite" {
				var receipts int
				if err := f.store.(*Store).db.QueryRow(`SELECT COUNT(*) FROM permission_migration_results`).Scan(&receipts); err != nil || receipts != 0 {
					t.Fatalf("dry-run persisted a receipt: count=%d err=%v", receipts, err)
				}
			}

			result, err := f.security.ApplyPermissionMigration(f.adminCtx, "permission-migration-apply-0001", application.PermissionMigrationApplyRequest{
				NamespaceID: f.namespace, PrincipalID: f.principal, CredentialID: f.credential.ID,
				ExpectedNamespaceVersion: before.NamespaceVersion, ExpectedPolicyVersion: before.Policy.Version,
				PolicyVersionPresent: true, Reason: "Preserve reviewed legacy work authority",
			})
			if err != nil {
				t.Fatalf("apply failed: %v", err)
			}
			if result.MapVersion != application.PermissionCompatibilityMapVersion || result.NamespaceVersion != before.NamespaceVersion+1 || !result.PolicyMigrated {
				t.Fatalf("unexpected migration result: %+v", result)
			}
			after, err := reader.ReadPermissionMigration(context.Background(), ports.PermissionMigrationRead{NamespaceID: f.namespace, PrincipalID: f.principal, CredentialID: f.credential.ID})
			if err != nil {
				t.Fatal(err)
			}
			wantGrant := before.Grant
			wantGrant.PermissionCompatibilityVersion = ports.PermissionCompatibilityLegacyV1
			if !reflect.DeepEqual(after.Grant, wantGrant) {
				t.Fatalf("migration rewrote permission grant: got %+v want %+v", after.Grant, wantGrant)
			}
			wantPolicy := *before.Policy
			wantPolicy.Version++
			wantPolicy.PermissionCompatibilityVersion = domain.PermissionCompatibilityLegacyV1
			if !reflect.DeepEqual(*after.Policy, wantPolicy) {
				t.Fatalf("migration changed policy beyond marker/revision: got %+v want %+v", *after.Policy, wantPolicy)
			}
			if after.NamespaceVersion != result.NamespaceVersion {
				t.Fatalf("namespace revision=%d, result=%d", after.NamespaceVersion, result.NamespaceVersion)
			}
			if err = f.security.Authorize(executorCtx, legacyWorkRequest); err != nil {
				t.Fatalf("migrated legacy grant/policy did not authorize its exact mapped command: %v", err)
			}
			postMigrationPreview, err := f.security.PreviewPermissionMigration(f.adminCtx, request)
			if err != nil || !reflect.DeepEqual(postMigrationPreview.EffectiveCapabilitiesBefore, postMigrationPreview.EffectiveCapabilitiesAfter) {
				t.Fatalf("preview did not reflect migrated authority as current: before=%v after=%v err=%v", postMigrationPreview.EffectiveCapabilitiesBefore, postMigrationPreview.EffectiveCapabilitiesAfter, err)
			}
			mappedTaskCreate := false
			for _, capability := range postMigrationPreview.EffectiveCapabilitiesAfter {
				mappedTaskCreate = mappedTaskCreate || capability == ports.PermissionTaskCreate
			}
			if !mappedTaskCreate {
				t.Fatal("migrated effective capability set omitted the mapped task-create capability")
			}

			replayed, err := f.security.ApplyPermissionMigration(f.adminCtx, "permission-migration-apply-0001", application.PermissionMigrationApplyRequest{
				NamespaceID: f.namespace, PrincipalID: f.principal, CredentialID: f.credential.ID,
				ExpectedNamespaceVersion: before.NamespaceVersion, ExpectedPolicyVersion: before.Policy.Version,
				PolicyVersionPresent: true, Reason: "Preserve reviewed legacy work authority",
			})
			if err != nil || !replayed.IdempotentReplay || replayed.NamespaceVersion != result.NamespaceVersion {
				t.Fatalf("idempotent replay=%+v err=%v", replayed, err)
			}

			staleNamespace, err := f.security.ApplyPermissionMigration(f.adminCtx, "permission-migration-stale-ns-0001", application.PermissionMigrationApplyRequest{
				NamespaceID: f.namespace, PrincipalID: f.principal, CredentialID: f.credential.ID,
				ExpectedNamespaceVersion: before.NamespaceVersion, ExpectedPolicyVersion: result.PolicyVersion,
				PolicyVersionPresent: true, Reason: "Stale namespace guard probe",
			})
			if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeVersionConflict || staleNamespace.NamespaceVersion != 0 {
				t.Fatalf("stale namespace was not rejected: result=%+v err=%v", staleNamespace, err)
			}
			stalePolicy, err := f.security.ApplyPermissionMigration(f.adminCtx, "permission-migration-stale-policy-0001", application.PermissionMigrationApplyRequest{
				NamespaceID: f.namespace, PrincipalID: f.principal, CredentialID: f.credential.ID,
				ExpectedNamespaceVersion: result.NamespaceVersion, ExpectedPolicyVersion: before.Policy.Version,
				PolicyVersionPresent: true, Reason: "Stale policy guard probe",
			})
			if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeVersionConflict || stalePolicy.NamespaceVersion != 0 {
				t.Fatalf("stale policy was not rejected: result=%+v err=%v", stalePolicy, err)
			}

			postReplay, err := reader.ReadPermissionMigration(context.Background(), ports.PermissionMigrationRead{NamespaceID: f.namespace, PrincipalID: f.principal, CredentialID: f.credential.ID})
			if err != nil || !reflect.DeepEqual(after, postReplay) {
				t.Fatalf("replay/stale request changed state: %+v => %+v (%v)", after, postReplay, err)
			}
			if backend == "sqlite" {
				adminSnapshot, snapshotErr := f.security.AdministrativeSnapshot(f.adminCtx, f.namespace)
				if snapshotErr != nil {
					t.Fatal(snapshotErr)
				}
				found := false
				for _, audit := range adminSnapshot.Audit {
					found = found || audit.Operation == "permission_compatibility_migrate" && audit.PrincipalID == "admin"
				}
				if !found {
					t.Fatalf("durable migration audit missing: %+v", adminSnapshot.Audit)
				}
			}
		})
	}
}

func TestPermissionMigrationDoesNotAcceptCallerSuppliedMarkerOrExpandEmptyGrant(t *testing.T) {
	f := newPermissionMigrationFixture(t, "memory")
	if err := f.security.SetGrant(f.adminCtx, ports.NamespaceGrant{NamespaceID: f.namespace, PrincipalID: "forged", PermissionCompatibilityVersion: ports.PermissionCompatibilityLegacyV1, Permissions: []ports.Permission{ports.PermissionWorkWrite}}); err == nil {
		t.Fatal("ordinary grant API accepted a migration marker")
	}
	if err := f.security.SetGrant(f.adminCtx, ports.NamespaceGrant{NamespaceID: f.namespace, PrincipalID: "empty"}); err != nil {
		t.Fatal(err)
	}
	newCredential, _, err := f.security.IssueCredential(f.adminCtx, f.namespace, "empty", domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "empty"}, time.Date(2026, 10, 21, 13, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	forgedPolicy := domain.CredentialPolicy{
		NamespaceID: f.namespace, CredentialID: newCredential.ID, PrincipalID: "empty",
		PermissionCompatibilityVersion: domain.PermissionCompatibilityLegacyV1,
		PermittedOperations:            []string{string(ports.PermissionWorkWrite)},
		AcceptanceFloor:                domain.AcceptanceDirect, MaxActiveWorkContracts: 1, MaxActiveReviewContracts: 1,
	}
	policyResult, err := f.security.SigningMutation(f.adminCtx, "permission-migration-marker-forgery", application.SigningSecurityIntent{
		NamespaceID: f.namespace, ExpectedNamespaceVersion: 2, Operation: "set_credential_policy", CredentialPolicy: &forgedPolicy,
	})
	if err != nil || policyResult.CredentialPolicy.PermissionCompatibilityVersion != 0 {
		t.Fatalf("ordinary policy API accepted a migration marker: result=%+v err=%v", policyResult, err)
	}
	reader := f.store.(ports.PermissionCompatibilityStore)
	state, err := reader.ReadPermissionMigration(context.Background(), ports.PermissionMigrationRead{NamespaceID: f.namespace, PrincipalID: "empty"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := f.security.ApplyPermissionMigration(f.adminCtx, "permission-migration-empty-0001", application.PermissionMigrationApplyRequest{
		NamespaceID: f.namespace, PrincipalID: "empty", ExpectedNamespaceVersion: state.NamespaceVersion,
		Reason: "Verify empty grants remain deny-all",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.MapVersion) == 0 {
		t.Fatal("empty grant migration omitted the map version")
	}
	after, err := f.security.PreviewPermissionMigration(f.adminCtx, application.PermissionMigrationPreviewRequest{NamespaceID: f.namespace, PrincipalID: "empty"})
	if err != nil || len(after.GrantCapabilitiesAfter) != 0 || len(after.EffectiveCapabilitiesAfter) != 0 {
		t.Fatalf("empty grant gained effective permissions: %+v err=%v", after, err)
	}
}

func TestPermissionMigrationRejectsConcurrentGrantRevocation(t *testing.T) {
	f := newPermissionMigrationFixture(t, "sqlite")
	preview, err := f.security.PreviewPermissionMigration(f.adminCtx, application.PermissionMigrationPreviewRequest{NamespaceID: f.namespace, PrincipalID: f.principal, CredentialID: f.credential.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = f.security.AdministrativeMutation(f.adminCtx, "permission-migration-revoke-0001", ports.SecurityAdminCommand{
		NamespaceID: f.namespace, ExpectedVersion: preview.Snapshot.NamespaceVersion,
		Operation: "set_grant", Grant: &ports.NamespaceGrant{NamespaceID: f.namespace, PrincipalID: f.principal},
	}); err != nil {
		t.Fatal(err)
	}
	_, err = f.security.ApplyPermissionMigration(f.adminCtx, "permission-migration-after-revoke-0001", application.PermissionMigrationApplyRequest{
		NamespaceID: f.namespace, PrincipalID: f.principal, CredentialID: f.credential.ID,
		ExpectedNamespaceVersion: preview.Snapshot.NamespaceVersion, ExpectedPolicyVersion: preview.Snapshot.Policy.Version,
		PolicyVersionPresent: true, Reason: "Must not restore revoked legacy authority",
	})
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeVersionConflict {
		t.Fatalf("migration after revocation error=%v", err)
	}
	state, err := f.store.(ports.PermissionCompatibilityStore).ReadPermissionMigration(context.Background(), ports.PermissionMigrationRead{NamespaceID: f.namespace, PrincipalID: f.principal, CredentialID: f.credential.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Grant.Permissions) != 0 || state.Grant.PermissionCompatibilityVersion != 0 || state.Policy.PermissionCompatibilityVersion != 0 {
		t.Fatalf("revocation changed by stale migration: %+v", state)
	}
}
