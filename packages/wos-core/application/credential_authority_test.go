package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestLegacyLeaseUsesRootCredentialAcrossSessionsAndRejectsDifferentRoot(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	root := f.addRoot(t, "0199f261-0000-7000-8000-000000000001", "root-token-a", f.cc.PrincipalID, f.cc.Actor)
	firstSession := f.addSession(t, "0199f262-0000-7000-8000-000000000001", "session-a", root, f.cc.PrincipalID, f.cc.Actor)
	secondSession := f.addSession(t, "0199f262-0000-7000-8000-000000000002", "session-b", root, f.cc.PrincipalID, f.cc.Actor)
	otherRoot := f.addRoot(t, "0199f261-0000-7000-8000-000000000003", "other-root-token", f.cc.PrincipalID, f.cc.Actor)

	claimed, err := f.service.ClaimWorkItem(f.identityContext(firstSession), f.cc, application.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: domain.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("ClaimWorkItem() error = %v", err)
	}
	lease := claimed.Value.CurrentLease
	if lease.CredentialBindingVersion != domain.WorkLeaseCredentialBindingV1 || lease.HolderCredentialID != root.ID || lease.SessionCredentialID == nil || *lease.SessionCredentialID != firstSession.ID {
		t.Fatalf("lease credential binding = %+v", lease)
	}

	renewed, err := f.service.RenewWorkItemLease(f.identityContext(secondSession), f.cc, application.RenewWorkItemLeaseCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: claimed.Value.Version,
		ClaimID: lease.ClaimID, FencingToken: lease.FencingToken, TTL: domain.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("same-root session could not renew lease: %v", err)
	}
	if renewed.Value.CurrentLease.HolderCredentialID != root.ID || renewed.Value.CurrentLease.SessionCredentialID == nil || *renewed.Value.CurrentLease.SessionCredentialID != firstSession.ID {
		t.Fatalf("renew changed stable holder/session provenance: %+v", renewed.Value.CurrentLease)
	}

	otherContext := f.identityContext(otherRoot)
	_, err = f.service.ReleaseWorkItem(otherContext, f.cc, application.ReleaseWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: renewed.Value.Version,
		ClaimID: lease.ClaimID, FencingToken: lease.FencingToken,
	})
	if code, _ := domain.ErrorCodeOf(err); err == nil || code != domain.ErrorCodeLease {
		t.Fatalf("same Principal with a different root released the lease: code=%s err=%v", code, err)
	}
	_, err = f.service.CompleteWorkItem(otherContext, f.cc, application.CompleteWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: renewed.Value.Version,
		ClaimID: lease.ClaimID, FencingToken: lease.FencingToken, ResultSummary: "must not complete", Reason: "wrong credential",
	})
	if code, _ := domain.ErrorCodeOf(err); err == nil || code != domain.ErrorCodeLease {
		t.Fatalf("same Principal with a different root completed the lease: code=%s err=%v", code, err)
	}

	view, err := f.service.GetEntityWorkspaceView(otherContext, f.scope, application.WorkspaceViewQuery{
		Subject: application.WorkspaceSubject{Kind: "entity", Ref: f.work.Ref()},
	})
	if err != nil {
		t.Fatalf("GetEntityWorkspaceView() error = %v", err)
	}
	capability := findLeaseCapability(t, view.Capabilities, "RenewWorkItemLease")
	if capability.Availability != "blocked" || len(capability.ReasonCodes) != 1 || capability.ReasonCodes[0] != "lease_owned_by_other_credential" {
		t.Fatalf("different root capability = %+v", capability)
	}
}

func TestLeaseCredentialResolverRejectsRevokedExpiredAndMismatchedIdentity(t *testing.T) {
	cases := []struct {
		name           string
		mutate         func(*credentialLeaseFixture, ports.Credential, ports.Credential)
		mutateIdentity func(*application.Identity)
	}{
		{
			name: "revoked parent",
			mutate: func(f *credentialLeaseFixture, root, _ ports.Credential) {
				if err := f.store.RevokeCredential(context.Background(), root.ID); err != nil {
					f.t.Fatal(err)
				}
			},
		},
		{name: "expiration boundary", mutate: func(f *credentialLeaseFixture, _, _ ports.Credential) { f.now = f.now.Add(time.Hour) }},
		{name: "namespace mismatch", mutateIdentity: func(identity *application.Identity) {
			identity.NamespaceID = id("0199f260-0000-7000-8000-000000000099")
		}},
		{name: "principal mismatch", mutate: func(f *credentialLeaseFixture, _, _ ports.Credential) { f.cc.PrincipalID = "different-principal" }},
		{name: "actor mismatch", mutate: func(f *credentialLeaseFixture, _, _ ports.Credential) { f.cc.Actor.ID = "different-actor" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCredentialLeaseFixture(t)
			root, session := f.addCredentialTree(t, "0199f261-0000-7000-8000-000000000010", "0199f262-0000-7000-8000-000000000010", "root-invalid", "session-invalid", f.cc.PrincipalID, f.cc.Actor)
			// A bound lease requires authenticated identity even when it was
			// acquired by an earlier compatible application composition.
			claimed, err := f.service.ClaimWorkItem(f.identityContext(session), f.cc, application.ClaimWorkItemCommand{
				Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: domain.DefaultLeaseTTL,
			})
			if err != nil {
				t.Fatal(err)
			}
			if tc.mutate != nil {
				tc.mutate(&f, root, session)
			}
			// A fresh service takes its transaction time from this fixed clock.
			service, err := application.NewService(f.store, fixedClock{now: f.now}, &sequenceIDs{next: 9500})
			if err != nil {
				t.Fatal(err)
			}
			identity := application.Identity{CredentialID: session.ID, PrincipalID: session.PrincipalID, NamespaceID: session.NamespaceID, CredentialDigest: session.Digest, Actor: session.Actor}
			if tc.mutateIdentity != nil {
				tc.mutateIdentity(&identity)
			}
			ctx := application.WithIdentity(context.Background(), identity)
			_, err = service.RenewWorkItemLease(ctx, f.cc, application.RenewWorkItemLeaseCommand{
				Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: claimed.Value.Version,
				ClaimID: claimed.Value.CurrentLease.ClaimID, FencingToken: claimed.Value.CurrentLease.FencingToken, TTL: domain.DefaultLeaseTTL,
			})
			if err == nil {
				t.Fatal("invalid credential identity renewed a lease")
			}
			if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeForbidden {
				t.Fatalf("credential failure code = %s, want forbidden: %v", code, err)
			}
		})
	}
}

func TestLeaseCredentialResolverRejectsMultiLevelSessionParent(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	root, session := f.addCredentialTree(t, "0199f261-0000-7000-8000-000000000040", "0199f262-0000-7000-8000-000000000040", "root-chain", "session-chain", f.cc.PrincipalID, f.cc.Actor)
	grandchild := f.addSession(t, "0199f262-0000-7000-8000-000000000041", "session-grandchild", session, f.cc.PrincipalID, f.cc.Actor)
	if root.ParentDigest != "" {
		t.Fatal("fixture root unexpectedly has a parent")
	}
	_, err := f.service.ClaimWorkItem(f.identityContext(grandchild), f.cc, application.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: domain.DefaultLeaseTTL,
	})
	if code, _ := domain.ErrorCodeOf(err); err == nil || code != domain.ErrorCodeForbidden {
		t.Fatalf("multi-level session chain was accepted: code=%s err=%v", code, err)
	}
}

func TestUnboundLegacyLeaseCapabilitiesDoNotClaimCredentialOwnership(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	_, session := f.addCredentialTree(t, "0199f261-0000-7000-8000-000000000020", "0199f262-0000-7000-8000-000000000020", "root-compat", "session-compat", f.cc.PrincipalID, f.cc.Actor)
	claimed, err := f.service.ClaimWorkItem(context.Background(), f.cc, application.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: domain.MinLeaseTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := f.identityContext(session)
	read := func() map[string]application.WorkspaceCapability {
		t.Helper()
		view, e := f.service.GetEntityWorkspaceView(ctx, f.scope, application.WorkspaceViewQuery{
			Subject: application.WorkspaceSubject{Kind: "entity", Ref: f.work.Ref()},
		})
		if e != nil {
			t.Fatal(e)
		}
		caps := map[string]application.WorkspaceCapability{}
		for _, capability := range view.Capabilities {
			caps[capability.ActionID] = capability
		}
		return caps
	}
	caps := read()
	if got := caps["RenewWorkItemLease"]; got.Availability != "blocked" || len(got.ReasonCodes) != 1 || got.ReasonCodes[0] != "lease_credential_binding_unavailable" {
		t.Fatalf("unbound lease renewal capability = %+v", got)
	}
	if claimed.Value.CurrentLease.CredentialBindingVersion != 0 {
		t.Fatal("legacy compatibility claim unexpectedly inferred credential ownership")
	}
	f.now = claimed.Value.CurrentLease.ExpiresAt
	f.service, err = application.NewService(f.store, fixedClock{now: f.now}, &sequenceIDs{next: 9800})
	if err != nil {
		t.Fatal(err)
	}
	caps = read()
	if got := caps["ReclaimWorkItem"]; got.Availability != "available" {
		t.Fatalf("expired unbound lease cannot be explicitly adopted: %+v", got)
	}
	unchanged, err := f.service.GetWorkItem(ctx, f.scope, f.work.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Value.CurrentLease.CredentialBindingVersion != 0 || unchanged.Value.CurrentLease.FencingToken != claimed.Value.CurrentLease.FencingToken {
		t.Fatalf("capability evaluation mutated legacy lease: %+v", unchanged.Value.CurrentLease)
	}
	reclaimed, err := f.service.ReclaimWorkItem(ctx, f.cc, application.ReclaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: unchanged.Value.Version, TTL: domain.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("credential did not explicitly adopt the expired unbound lease: %v", err)
	}
	if reclaimed.Value.CurrentLease.CredentialBindingVersion != domain.WorkLeaseCredentialBindingV1 || reclaimed.Value.CurrentLease.HolderCredentialID == "" || reclaimed.Value.CurrentLease.SessionCredentialID == nil || *reclaimed.Value.CurrentLease.SessionCredentialID != session.ID {
		t.Fatalf("adopted lease binding = %+v", reclaimed.Value.CurrentLease)
	}
}

func TestAuthorizedLeaseClaimRequiresCredentialBoundAccessSnapshot(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	root, _ := f.addCredentialTree(t, "0199f261-0000-7000-8000-000000000030", "0199f262-0000-7000-8000-000000000030", "root-authorized", "session-authorized", f.cc.PrincipalID, f.cc.Actor)
	if err := f.store.PutGrant(context.Background(), ports.NamespaceGrant{
		NamespaceID: f.scope.NamespaceID, PrincipalID: f.cc.PrincipalID,
		Permissions: []ports.Permission{ports.PermissionTaskClaim},
	}); err != nil {
		t.Fatal(err)
	}
	clock := fixedClock{now: f.now}
	security := application.SecurityService{Store: f.store, Clock: clock, IDs: &sequenceIDs{next: 9900}}
	service, err := application.NewAuthorizedService(f.store, clock, &sequenceIDs{next: 9910}, security)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := service.ClaimWorkItem(f.identityContext(root), f.cc, application.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: domain.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("authorized credential could not claim work: %v", err)
	}
	if claimed.Value.CurrentLease.HolderCredentialID != root.ID {
		t.Fatalf("authorized lease holder = %s, want %s", claimed.Value.CurrentLease.HolderCredentialID, root.ID)
	}

	missingDigest := application.WithIdentity(context.Background(), application.Identity{
		CredentialID: root.ID, PrincipalID: root.PrincipalID, NamespaceID: root.NamespaceID, Actor: root.Actor,
	})
	_, err = service.RenewWorkItemLease(missingDigest, f.cc, application.RenewWorkItemLeaseCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: claimed.Value.Version,
		ClaimID: claimed.Value.CurrentLease.ClaimID, FencingToken: claimed.Value.CurrentLease.FencingToken, TTL: domain.DefaultLeaseTTL,
	})
	if code, _ := domain.ErrorCodeOf(err); err == nil || code != domain.ErrorCodeForbidden {
		t.Fatalf("authorized host accepted missing credential digest: code=%s err=%v", code, err)
	}
}

type credentialLeaseFixture struct {
	t       *testing.T
	store   *memory.Store
	service *application.Service
	now     time.Time
	scope   domain.Scope
	work    domain.WorkItem
	cc      domain.CommandContext
}

func newCredentialLeaseFixture(t *testing.T) credentialLeaseFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	store := memory.New()
	namespace := id("0199f260-0000-7000-8000-000000000001")
	if err := store.PutNamespace(ctx, ports.Namespace{ID: namespace, Name: "lease tests"}); err != nil {
		t.Fatal(err)
	}
	service, err := application.NewService(store, fixedClock{now: now}, &sequenceIDs{next: 9000})
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID: namespace, Title: "Credential-bound lease", DesiredState: "lease ownership follows root credential", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner: created.Value.Ref(), ExpectedVersion: created.Value.Version, Title: "work is verified", Required: true, VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	activated, err := service.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{Scope: created.Value.Scope(), ExpectedVersion: created.Value.Version + 1})
	if err != nil {
		t.Fatal(err)
	}
	objectiveID := objectiveForWorkFixture(t, service, activated.Value.Scope())
	createdTask, err := service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{
		Scope: activated.Value.Scope(), ObjectiveID: objectiveID, Title: "leased task", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	return credentialLeaseFixture{t: t, store: store, service: service, now: now, scope: activated.Value.Scope(), work: createdTask.Value, cc: cc}
}

func (f credentialLeaseFixture) addRoot(t *testing.T, credentialID, digest, principal string, actor domain.ActorRef) ports.Credential {
	t.Helper()
	credential := ports.Credential{
		ID: id(credentialID), NamespaceID: f.scope.NamespaceID,
		PrincipalID: principal, Actor: actor, Digest: digest, ExpiresAt: f.now.Add(time.Hour),
	}
	if err := f.store.PutCredential(context.Background(), credential); err != nil {
		t.Fatal(err)
	}
	return credential
}

func (f credentialLeaseFixture) addCredentialTree(t *testing.T, rootID, sessionID, rootSuffix, sessionSuffix, principal string, actor domain.ActorRef) (ports.Credential, ports.Credential) {
	t.Helper()
	rootDigest := "root-token-" + rootSuffix
	root := f.addRoot(t, rootID, rootDigest, principal, actor)
	session := f.addSession(t, sessionID, sessionSuffix, root, principal, actor)
	return root, session
}

func (f credentialLeaseFixture) addSession(t *testing.T, credentialID, suffix string, root ports.Credential, principal string, actor domain.ActorRef) ports.Credential {
	t.Helper()
	session := ports.Credential{
		ID: id(credentialID), NamespaceID: f.scope.NamespaceID,
		PrincipalID: principal, Actor: actor, Digest: "session-token-" + suffix,
		ParentDigest: root.Digest, ExpiresAt: f.now.Add(time.Hour),
	}
	if err := f.store.PutCredential(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	return session
}

func (f credentialLeaseFixture) identityContext(credential ports.Credential) context.Context {
	return application.WithIdentity(context.Background(), application.Identity{
		CredentialID: credential.ID, PrincipalID: credential.PrincipalID, NamespaceID: credential.NamespaceID,
		CredentialDigest: credential.Digest, Actor: credential.Actor,
	})
}

func findLeaseCapability(t *testing.T, capabilities []application.WorkspaceCapability, action string) application.WorkspaceCapability {
	t.Helper()
	for _, capability := range capabilities {
		if capability.ActionID == action {
			return capability
		}
	}
	t.Fatalf("capability %q not found", action)
	return application.WorkspaceCapability{}
}
