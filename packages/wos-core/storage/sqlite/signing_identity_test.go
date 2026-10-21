package sqlite

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestSigningIdentityEnrollmentRotationPolicyAndRevocation(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			var store interface {
				ports.SecurityStore
				ports.TransactionManager
			}
			if backend == "memory" {
				store = memory.New()
			} else {
				store = openTestStore(t, filepath.Join(t.TempDir(), "signed-identity.db"))
			}
			clock := sqliteFixedClock{now: time.Now().UTC()}
			sec := application.SecurityService{Store: store, Clock: clock, IDs: &sqliteSequenceIDs{prefix: "0199a222", next: 1}, ServerID: "test-persistent-server"}
			ns := testID("0199a221-0000-7000-8000-000000000001")
			ctx := context.Background()
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			must(sec.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "signed"}, "admin", "signed-identity-admin-token-at-least-32"))
			admin, err := sec.Authenticate(ctx, "signed-identity-admin-token-at-least-32")
			must(err)
			adminCtx := application.WithIdentity(ctx, admin)
			permissions := []ports.Permission{ports.PermissionStateRead, ports.PermissionSigningKeyEnroll, ports.PermissionSigningKeyRotate, ports.PermissionWorkContractAcquire}
			must(sec.Store.PutGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "executor", Permissions: permissions}))
			credential, token, err := sec.IssueCredential(adminCtx, ns, "executor", d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}, clock.Now().Add(time.Hour))
			must(err)
			namespaceVersion, err := store.(ports.SecurityAdminVersionReader).ReadSecurityAdminVersion(ctx, ns)
			must(err)
			agent, err := sec.Authenticate(ctx, token)
			must(err)
			agentCtx := application.WithIdentity(ctx, agent)
			// A bearer token cannot create its own administrator-authorized enrollment.
			_, err = sec.SigningMutation(agentCtx, "unauthorized-enrollment-1", application.SigningSecurityIntent{NamespaceID: ns, Operation: "create_enrollment", CredentialID: credential.ID, PrincipalID: agent.PrincipalID, ExpectedNamespaceVersion: namespaceVersion})
			if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeForbidden {
				t.Fatalf("token authorized its own enrollment: %v", err)
			}
			policy := d.CredentialPolicy{NamespaceID: ns, CredentialID: credential.ID, PrincipalID: agent.PrincipalID, AcceptanceFloor: d.AcceptanceIndependentReview, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1}
			for _, p := range permissions {
				policy.PermittedOperations = append(policy.PermittedOperations, string(p))
			}
			configured, err := sec.SigningMutation(adminCtx, "configure-signing-policy-1", application.SigningSecurityIntent{NamespaceID: ns, Operation: "set_credential_policy", ExpectedNamespaceVersion: namespaceVersion, ExpectedVersion: 1, CredentialPolicy: &policy})
			must(err)
			pub, private, err := ed25519.GenerateKey(rand.Reader)
			must(err)
			fp, err := signing.Fingerprint(pub)
			must(err)
			enrolled, err := sec.SigningMutation(adminCtx, "authorize-enrollment-1", application.SigningSecurityIntent{NamespaceID: ns, Operation: "create_enrollment", ExpectedNamespaceVersion: configured.NamespaceVersion, CredentialID: credential.ID, PrincipalID: agent.PrincipalID, ExpectedFingerprint: fp})
			must(err)
			proofFor := func(e d.SigningEnrollment, public ed25519.PublicKey, secret ed25519.PrivateKey) signing.Envelope {
				t.Helper()
				previous := ""
				if e.PreviousKeyID != nil {
					previous = e.PreviousKeyID.String()
				}
				p := signing.EnrollmentPayload{Binding: signing.Binding{ProtocolVersion: 2, ServerID: sec.ServerID, NamespaceID: ns.String(), PrincipalID: agent.PrincipalID, SignerKeyID: e.KeyID.String()}, EnrollmentID: e.ID.String(), Nonce: e.Nonce, Purpose: e.Purpose, PublicKey: base64.StdEncoding.EncodeToString(public), PreviousKeyID: previous, ExpiresAt: e.ExpiresAt.UTC().Format(time.RFC3339Nano)}
				envelope, err := signing.Sign(signing.KeyEnrollment, p, e.KeyID.String(), secret)
				must(err)
				return envelope
			}
			proof := proofFor(*enrolled.Enrollment, pub, private)
			register := application.SigningSecurityIntent{NamespaceID: ns, Operation: "register_signing_key", EnrollmentID: enrolled.Enrollment.ID, Proof: &proof}
			// Changing untrusted signature metadata cannot change the accepted identity.
			forged := proof
			forged.Signatures = append([]signing.Signature(nil), proof.Signatures...)
			forged.Signatures[0].KeyID = testID("0199a221-0000-7000-8000-000000000099").String()
			bad := register
			bad.Proof = &forged
			if _, err = sec.SigningMutation(agentCtx, "forged-enrollment-1", bad); err == nil {
				t.Fatal("forged key hint accepted")
			}

			challengeView, err := sec.SigningIdentity(agentCtx, &enrolled.Enrollment.ID)
			must(err)
			if challengeView.Enrollment.ID != enrolled.Enrollment.ID || challengeView.CredentialID != credential.ID {
				t.Fatal("identity view lost credential challenge binding")
			}
			if _, err = sec.SigningIdentity(adminCtx, &enrolled.Enrollment.ID); err == nil {
				t.Fatal("another principal read enrollment challenge")
			}
			// Even a valid possession signature must match the exact authorized challenge.
			wrongChallenge := *enrolled.Enrollment
			wrongChallenge.Nonce = "wrong-but-well-formed-32-character-challenge"
			wrongProof := proofFor(wrongChallenge, pub, private)
			bad = register
			bad.Proof = &wrongProof
			if _, err = sec.SigningMutation(agentCtx, "wrong-challenge-registration-1", bad); err == nil {
				t.Fatal("valid signature accepted another challenge")
			}
			// Independently valid signatures must not authorize a different enrollment.
			assertEnrollmentDenied := func(t *testing.T, intent string, caller context.Context, challenge d.SigningEnrollment, envelope signing.Envelope) {
				t.Helper()
				if _, verifyErr := signing.Verify(envelope, signing.KeyEnrollment, challenge.KeyID.String(), pub); verifyErr != nil {
					t.Fatal("probe must have a mathematically valid signature", verifyErr)
				}
				request := register
				request.EnrollmentID = challenge.ID
				request.Proof = &envelope
				_, denied := sec.SigningMutation(caller, intent, request)
				if code, _ := d.ErrorCodeOf(denied); code != d.ErrorCodeForbidden {
					t.Fatalf("enrollment authority probe must fail as forbidden: %v", denied)
				}
				unit, beginErr := store.Begin(ctx)
				must(beginErr)
				defer unit.Rollback()
				repository := unit.(ports.SigningIdentityUnitOfWork).SigningIdentity()
				pending, readErr := repository.Enrollment(ctx, ns, challenge.ID)
				must(readErr)
				if pending.Consumed || pending.Version != challenge.Version {
					t.Fatal("rejected proof consumed or changed the enrollment")
				}
				if _, keyErr := repository.Key(ctx, ns, challenge.KeyID); keyErr == nil {
					t.Fatal("rejected proof registered a signing key")
				} else if code, _ := d.ErrorCodeOf(keyErr); code != d.ErrorCodeNotFound {
					t.Fatal("cannot verify key absence", keyErr)
				}
			}
			for _, field := range []string{"purpose", "principal"} {
				t.Run("valid_signature_wrong_"+field, func(t *testing.T) {
					payload, _, decodeErr := signing.Decode[signing.EnrollmentPayload](proof, signing.KeyEnrollment, enrolled.Enrollment.KeyID.String(), pub)
					must(decodeErr)
					if field == "purpose" {
						payload.Purpose = "issuer"
					} else {
						payload.PrincipalID = admin.PrincipalID
					}
					probe, signErr := signing.Sign(signing.KeyEnrollment, payload, enrolled.Enrollment.KeyID.String(), private)
					must(signErr)
					assertEnrollmentDenied(t, "enrollment-wrong-"+field, agentCtx, *enrolled.Enrollment, probe)
				})
			}
			t.Run("other_principal_cannot_consume_challenge", func(t *testing.T) {
				assertEnrollmentDenied(t, "enrollment-other-principal", adminCtx, *enrolled.Enrollment, proof)
			})
			t.Run("expired_matching_challenge", func(t *testing.T) {
				// PostgreSQL uses transaction time, so persist an elapsed deadline
				// in a separate approved fixture instead of faking the Go clock.
				expired := *enrolled.Enrollment
				expired.ID = testID("0199a221-0000-7000-8000-000000000077")
				expired.KeyID = testID("0199a221-0000-7000-8000-000000000078")
				expired.ExpiresAt = clock.Now().Add(-time.Hour)
				unit, beginErr := store.Begin(ctx)
				must(beginErr)
				repository := unit.(ports.SigningIdentityUnitOfWork).SigningIdentity()
				if saveErr := repository.SaveEnrollment(ctx, expired, 0); saveErr != nil {
					unit.Rollback()
					t.Fatal(saveErr)
				}
				must(unit.Commit())
				assertEnrollmentDenied(t, "enrollment-expired-challenge", agentCtx, expired, proofFor(expired, pub, private))
			})

			registered, err := sec.SigningMutation(agentCtx, "register-signing-key-1", register)
			must(err)
			if registered.Key.PrincipalID != agent.PrincipalID || !registered.Enrollment.Consumed || registered.CredentialPolicy.AcceptanceFloor != d.AcceptanceIndependentReview {
				t.Fatal("registration weakened policy or changed principal")
			}
			replay, err := sec.SigningMutation(agentCtx, "register-signing-key-1", register)
			must(err)
			if !replay.IdempotentReplay || replay.Key.ID != registered.Key.ID {
				t.Fatal("same intent not replayed")
			}
			if _, err = sec.SigningMutation(agentCtx, "consume-enrollment-again-1", register); err == nil {
				t.Fatal("one-use challenge reused")
			}
			// Credential policy is an intersection with grants, including unsigned v1 access.
			if err = sec.Require(agentCtx, ns, ports.PermissionNamespaceAdmin); err == nil {
				t.Fatal("key possession granted administrator access")
			}
			pub2, private2, err := ed25519.GenerateKey(rand.Reader)
			must(err)
			rotation := signing.EnrollmentPayload{Binding: signing.Binding{ProtocolVersion: 2, ServerID: sec.ServerID, NamespaceID: ns.String(), PrincipalID: agent.PrincipalID, SignerKeyID: registered.Key.ID.String()}, EnrollmentID: testID("0199a221-0000-7000-8000-000000000020").String(), NewKeyID: testID("0199a221-0000-7000-8000-000000000021").String(), Nonce: "rotation-authorization-bound-to-unique-enrollment", Purpose: "rotation_authorization", PublicKey: base64.StdEncoding.EncodeToString(pub2), PreviousKeyID: registered.Key.ID.String(), ExpiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339Nano)}
			rotationProof, err := signing.Sign(signing.KeyEnrollment, rotation, registered.Key.ID.String(), private)
			must(err)
			rotating, err := sec.SigningMutation(agentCtx, "authorize-key-rotation-1", application.SigningSecurityIntent{NamespaceID: ns, Operation: "create_rotation_enrollment", Proof: &rotationProof})
			must(err)
			proof2 := proofFor(*rotating.Enrollment, pub2, private2)
			rotated, err := sec.SigningMutation(agentCtx, "register-key-rotation-1", application.SigningSecurityIntent{NamespaceID: ns, Operation: "register_signing_key", EnrollmentID: rotating.Enrollment.ID, Proof: &proof2})
			must(err)
			if rotated.Key.ID == registered.Key.ID || rotated.CredentialPolicy.AcceptanceFloor != d.AcceptanceIndependentReview || rotated.Key.PrincipalID != registered.Key.PrincipalID {
				t.Fatal("rotation changed stable subject/floor")
			}
			uow, err := store.Begin(ctx)
			must(err)
			repo := uow.(ports.SigningIdentityUnitOfWork).SigningIdentity()
			old, err := repo.Key(ctx, ns, registered.Key.ID)
			must(err)
			historical, err := old.PublicBytes()
			must(err)
			if old.Status != "retired" {
				t.Fatal("previous key remains active")
			}
			if _, err = signing.Verify(proof, signing.KeyEnrollment, old.ID.String(), historical); err != nil {
				t.Fatal("historical signature lost", err)
			}
			must(uow.Rollback())
			if _, err = sec.SigningMutation(agentCtx, "rotate-with-retired-key-1", application.SigningSecurityIntent{NamespaceID: ns, Operation: "create_rotation_enrollment", Proof: &rotationProof}); err == nil {
				t.Fatal("retired key authorized new rotation")
			}
			revoked, err := sec.SigningMutation(adminCtx, "revoke-registered-key-1", application.SigningSecurityIntent{NamespaceID: ns, Operation: "revoke_signing_key", ExpectedNamespaceVersion: rotated.NamespaceVersion, KeyID: rotated.Key.ID, ExpectedVersion: 1, Reason: "compromised"})
			must(err)
			if revoked.Key.Status != "revoked" {
				t.Fatal("revocation not persisted")
			}
			durable, err := sec.SigningOperation(agentCtx, "register-key-rotation-1")
			must(err)
			if durable.Key.ID != rotated.Key.ID {
				t.Fatal("durable receipt disappeared on key revocation")
			}

			// Grant alone cannot restore a permission removed by credential policy.
			restricted := *rotated.CredentialPolicy
			restricted.PermittedOperations = []string{string(ports.PermissionStateRead), string(ports.PermissionSigningKeyEnroll), string(ports.PermissionSigningKeyRotate)}
			tightened, err := sec.SigningMutation(adminCtx, "tighten-current-policy-1", application.SigningSecurityIntent{NamespaceID: ns, Operation: "set_credential_policy", ExpectedNamespaceVersion: revoked.NamespaceVersion, ExpectedVersion: restricted.Version, CredentialPolicy: &restricted})
			must(err)
			if err = sec.Require(agentCtx, ns, ports.PermissionWorkContractAcquire); err == nil {
				t.Fatal("grant bypassed current credential restriction")
			}
			pub3, private3, err := ed25519.GenerateKey(rand.Reader)
			must(err)
			fresh, err := sec.SigningMutation(adminCtx, "authorize-concurrent-enrollment-1", application.SigningSecurityIntent{NamespaceID: ns, Operation: "create_enrollment", ExpectedNamespaceVersion: tightened.NamespaceVersion, CredentialID: credential.ID, PrincipalID: agent.PrincipalID})
			must(err)
			proof3 := proofFor(*fresh.Enrollment, pub3, private3)
			concurrentIntent := application.SigningSecurityIntent{NamespaceID: ns, Operation: "register_signing_key", EnrollmentID: fresh.Enrollment.ID, Proof: &proof3}
			results := make(chan error, 2)
			var wait sync.WaitGroup
			for _, key := range []string{"competing-registration-one", "competing-registration-two"} {
				wait.Add(1)
				go func(key string) {
					defer wait.Done()
					_, e := sec.SigningMutation(agentCtx, key, concurrentIntent)
					results <- e
				}(key)
			}
			wait.Wait()
			close(results)
			success := 0
			for e := range results {
				if e == nil {
					success++
				}
			}
			if success != 1 {
				t.Fatalf("one-use challenge concurrent commits=%d", success)
			}
			view, err := sec.SigningIdentity(agentCtx, nil)
			must(err)
			if len(view.Keys) != 3 || view.Policy.AcceptanceFloor != d.AcceptanceIndependentReview {
				t.Fatal("concurrent registration duplicated key or weakened floor")
			}
			// A previously authenticated context is revalidated against live credential revocation.
			must(sec.Store.RevokeCredential(adminCtx, credential.ID))
			if _, err = sec.SigningOperation(agentCtx, "register-key-rotation-1"); err == nil {
				t.Fatal("revoked bearer read durable receipt")
			}
		})
	}
}
