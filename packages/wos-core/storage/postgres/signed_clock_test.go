package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"path/filepath"
	"testing"
	"time"
)

func TestSignedContractsUseDatabaseAuthorityDespiteReplicaClockSkew(t *testing.T) {
	requirePostgres(t)
	store := openTestStore(t, filepath.Join(t.TempDir(), "signed-replica-clock.db"))
	ctx := context.Background()
	now := time.Now().UTC()
	clock := sqliteFixedClock{now}
	ids := &sqliteSequenceIDs{prefix: "0199a556", next: 1}
	sec := a.SecurityService{Store: store, Clock: clock, IDs: ids}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	ns := testID("0199a555-0000-7000-8000-000000000001")
	scope := d.Scope{NamespaceID: ns, OutcomeID: testID("0199a555-0000-7000-8000-000000000002")}
	must(sec.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "signed-acquisition"}, "admin", "signed-acquisition-bootstrap-token-at-least-32"))
	admin, err := sec.Authenticate(ctx, "signed-acquisition-bootstrap-token-at-least-32")
	must(err)
	adminCtx := a.WithIdentity(ctx, admin)
	must(sec.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "executor", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionWorkContractAcquire, ports.PermissionWorkWrite}}))
	credential, token, err := sec.IssueCredential(adminCtx, ns, "executor", d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}, now.Add(time.Hour))
	must(err)
	identity, err := sec.Authenticate(ctx, token)
	must(err)
	agentCtx := a.WithIdentity(ctx, identity)
	pub, private, err := ed25519.GenerateKey(rand.Reader)
	must(err)
	fp, err := signing.Fingerprint(pub)
	must(err)
	server := d.ServerIdentity{ID: testID("0199a555-0000-7000-8000-000000000003"), IssuerKeyID: testID("0199a555-0000-7000-8000-000000000004"), PublicKey: base64.StdEncoding.EncodeToString(pub), Fingerprint: fp, CreatedAt: now}
	service, err := a.NewService(store, clock, ids)
	must(err)
	must(service.ConfigureSignedIssuer(ctx, acquisitionIssuer{server, private}, d.AcceptanceDirect))
	agentPub, _, err := ed25519.GenerateKey(rand.Reader)
	must(err)
	agentFP, err := signing.Fingerprint(agentPub)
	must(err)
	agentKey := testID("0199a555-0000-7000-8000-000000000005")
	uow, err := store.Begin(ctx)
	must(err)
	registry := uow.(ports.SigningIdentityUnitOfWork).SigningIdentity()
	must(registry.SaveKey(ctx, d.SigningKey{ID: agentKey, NamespaceID: ns, PrincipalID: "executor", Purpose: "agent", Algorithm: "Ed25519", PublicKey: base64.StdEncoding.EncodeToString(agentPub), Fingerprint: agentFP, Version: 1, Status: "active", CreatedAt: now, UpdatedAt: now}, 0))
	policy := d.CredentialPolicy{NamespaceID: ns, CredentialID: credential.ID, PrincipalID: "executor", Version: 1, PermittedOperations: []string{string(ports.PermissionStateRead), string(ports.PermissionWorkContractAcquire), string(ports.PermissionWorkWrite)}, AcceptanceFloor: d.AcceptanceIndependentReview, AllowedSigningKeyIDs: []d.ID{agentKey}, MaxActiveWorkContracts: 10, MaxActiveReviewContracts: 1, UpdatedAt: now}
	must(registry.SaveCredentialPolicy(ctx, policy, 0))
	outcome, err := d.NewOutcome(scope.OutcomeID, ns, "signed workload", "", "verified", d.PriorityNormal, now)
	must(err)
	outcome.Lifecycle = d.OutcomeLifecycleActive
	must(uow.Outcomes().Insert(ctx, outcome))
	works := []d.WorkItem{}
	for i := 0; i < 4; i++ {
		w, e := d.NewWorkItem(testID(fmt.Sprintf("0199a555-0000-7000-8000-%012x", 20+i)), scope, "precise signed task", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
		must(e)
		w.ContractsEnabled = true
		must(uow.WorkItems().Insert(ctx, w))
		works = append(works, w)
	}
	protocol := uow.(ports.WorkProtocolUnitOfWork).WorkProtocol()
	state, err := protocol.Lock(ctx, ns, true)
	must(err)
	state.Phase = d.WorkProtocolSigned
	state.WriterEpoch = 2
	state.WritersDrained = true
	state.Version++
	state.UpdatedAt = now
	state.UpdatedBy = "operator"
	state.Reason = "test-only operator activation; public cutover remains separately gated"
	must(protocol.Save(ctx, state, 1))
	must(uow.Commit())
	replica := func(skew time.Duration) *a.Service {
		s, e := a.NewService(store, sqliteFixedClock{now: now.Add(skew)}, ids)
		must(e)
		must(s.ConfigureSignedIssuer(ctx, acquisitionIssuer{server, private}, d.AcceptanceDirect))
		return s
	}
	future, past := replica(365*24*time.Hour), replica(-365*24*time.Hour)
	next := func(key string) d.CommandContext {
		id, e := ids.NewID()
		must(e)
		return d.CommandContext{PrincipalID: identity.PrincipalID, Actor: identity.Actor, CommandID: id, IdempotencyKey: key}
	}
	before := time.Now().UTC()
	acquired, e := future.AcquireSignedWorkContract(agentCtx, next("future-signed-acquire"), a.AcquireSignedWorkContractCommand{Scope: scope, WorkItemID: works[0].ID, ExpectedWorkItemVersion: 1, SignerKeyID: agentKey, TTLSeconds: 30})
	must(e)
	c := acquired.Value.Contract
	if c.AcquiredAt.Before(before.Add(-time.Second)) || c.AcquiredAt.After(time.Now().Add(time.Second)) || c.ExpiresAt.Sub(c.AcquiredAt) != 30*time.Second {
		t.Fatal("signed lease used skewed replica time")
	}
	envelope, e := acquired.Value.IssuedAuthority.Envelope()
	must(e)
	grant, _, e := signing.Decode[signing.AuthorityPayload](envelope, signing.ContractAuthority, server.IssuerKeyID.String(), pub)
	must(e)
	expires, e := time.Parse(time.RFC3339Nano, grant.ExpiresAt)
	must(e)
	if !expires.Equal(c.ExpiresAt) {
		t.Fatal("signed authority deadline differs from database lease")
	}
	authority := a.ContractAuthority{ExecutionID: c.ExecutionID, FencingToken: c.FencingToken, SpecDigest: c.SignedBinding.SpecificationDigest}
	renewed, e := past.RenewSignedWorkContract(agentCtx, next("past-signed-renew"), a.RenewSignedWorkContractCommand{Scope: scope, ContractID: c.ID, SignerKeyID: agentKey, Authority: authority, AuthorityDigest: signing.Digest(acquired.Value.IssuedAuthority.Payload), ExpectedLeaseVersion: c.LeaseVersion, TTLSeconds: 30})
	must(e)
	if renewed.Value.Contract.ExpiresAt.Before(before) || renewed.Value.Contract.ExpiresAt.After(time.Now().Add(31*time.Second)) {
		t.Fatal("renewal used skewed replica deadline")
	}
	rc := renewed.Value.Contract
	resumed, e := future.ResumeSignedWorkContract(agentCtx, next("future-signed-takeover"), a.ResumeSignedWorkContractCommand{Scope: scope, ContractID: rc.ID, SignerKeyID: agentKey, Authority: a.ContractAuthority{ExecutionID: rc.ExecutionID, FencingToken: rc.FencingToken, SpecDigest: rc.SignedBinding.SpecificationDigest}, AuthorityDigest: signing.Digest(renewed.Value.IssuedAuthority.Payload), ExpectedLeaseVersion: rc.LeaseVersion})
	must(e)
	if resumed.Value.Contract.ExecutionID == rc.ExecutionID || resumed.Value.Contract.FencingToken <= rc.FencingToken || !resumed.Value.Contract.ExpiresAt.Equal(rc.ExpiresAt) {
		t.Fatal("skewed takeover lost fencing or extended deadline")
	}
	_, e = past.RenewSignedWorkContract(agentCtx, next("stale-signed-clock-authority"), a.RenewSignedWorkContractCommand{Scope: scope, ContractID: c.ID, SignerKeyID: agentKey, Authority: authority, AuthorityDigest: signing.Digest(acquired.Value.IssuedAuthority.Payload), ExpectedLeaseVersion: c.LeaseVersion, TTLSeconds: 30})
	if e == nil {
		t.Fatal("replica clock revived old fenced authority")
	}
}
