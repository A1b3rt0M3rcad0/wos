package application

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

type policyReadFaultUOW struct {
	ports.UnitOfWork
	policyReadFaultRegistry
}

func (u policyReadFaultUOW) WorkProtocol() ports.WorkProtocolRepository {
	return policyReadFaultProtocol{}
}
func (u policyReadFaultUOW) AuthorizeAccessSnapshot(context.Context, ports.AccessSnapshotRequest) error {
	return nil
}
func (u policyReadFaultUOW) SigningIdentity() ports.SigningIdentityRepository {
	return u.policyReadFaultRegistry
}
func (u policyReadFaultUOW) Rollback() error { return nil }

type policyReadFaultProtocol struct{ ports.WorkProtocolRepository }

func (policyReadFaultProtocol) Lock(_ context.Context, ns d.ID, _ bool) (d.TenantWorkProtocol, error) {
	p := d.DefaultWorkProtocol(ns)
	p.Phase = d.WorkProtocolSigned
	return p, nil
}

type policyReadFaultRegistry struct {
	ports.SigningIdentityRepository
	credential ports.Credential
	policy     d.CredentialPolicy
	enrollment d.SigningEnrollment
	err        error
}

func (r policyReadFaultRegistry) CredentialByDigest(context.Context, string) (ports.Credential, error) {
	return r.credential, nil
}
func (r policyReadFaultRegistry) CredentialPolicy(context.Context, d.ID, d.ID) (d.CredentialPolicy, error) {
	return r.policy, r.err
}
func (r policyReadFaultRegistry) LockTenant(context.Context, d.ID) (d.Version, error) {
	return 1, nil
}
func (r policyReadFaultRegistry) Receipt(context.Context, d.ID, string, string) (string, json.RawMessage, error) {
	return "", nil, d.NewError(d.ErrorCodeNotFound, "no preceding security operation")
}
func (r policyReadFaultRegistry) Enrollment(context.Context, d.ID, d.ID) (d.SigningEnrollment, error) {
	return r.enrollment, nil
}

type policyReadFaultSecurityStore struct {
	ports.SecurityStore
	u policyReadFaultUOW
}

func (s policyReadFaultSecurityStore) Begin(context.Context) (ports.UnitOfWork, error) {
	return s.u, nil
}
func (s policyReadFaultSecurityStore) GetCredential(context.Context, string) (ports.Credential, error) {
	return s.u.credential, nil
}
func (s policyReadFaultSecurityStore) GetGrant(context.Context, d.ID, string) (ports.TenantGrant, error) {
	return ports.TenantGrant{Permissions: []ports.Permission{ports.PermissionSigningKeyEnroll}}, nil
}

type policyReadFaultClock struct{}

func (policyReadFaultClock) Now() time.Time { return time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC) }

// Policy absence is an authorization decision. A failed transactional policy
// read is an infrastructure failure and must preserve exact-intent recovery.
func TestSignedAccessPreservesPolicyReadFailures(t *testing.T) {
	ns := d.MustParseID("0199f390-0000-7000-8000-000000000001")
	cid := d.MustParseID("0199f390-0000-7000-8000-000000000002")
	scope := d.Scope{TenantID: ns, OutcomeID: d.MustParseID("0199f390-0000-7000-8000-000000000003")}
	id := Identity{TenantID: ns, PrincipalID: "executor", CredentialDigest: "authenticated-digest"}
	ctx := WithIdentity(context.Background(), id)
	policy := d.CredentialPolicy{TenantID: ns, CredentialID: cid, PrincipalID: id.PrincipalID, Version: 1, PermittedOperations: []string{string(ports.PermissionWorkContractAcquire)}, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1, AcceptanceFloor: d.AcceptanceDirect}
	conflict := d.WrapError(d.ErrorCodeTransactionConflict, "same transaction must be reconciled", errors.New("storage serialization conflict"))
	for _, tc := range []struct {
		name    string
		failure error
		policy  d.CredentialPolicy
		code    d.ErrorCode
	}{
		{"transaction conflict", conflict, policy, d.ErrorCodeTransactionConflict},
		{"request deadline", context.DeadlineExceeded, policy, ""},
		{"missing policy", d.NewError(d.ErrorCodeNotFound, "policy absent"), policy, d.ErrorCodeForbidden},
		{"disallowed operation", nil, d.CredentialPolicy{PrincipalID: id.PrincipalID}, d.ErrorCodeForbidden},
		{"allowed operation", nil, policy, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := policyReadFaultUOW{policyReadFaultRegistry: policyReadFaultRegistry{credential: ports.Credential{ID: cid}, policy: tc.policy, err: tc.failure}}
			s := Service{clock: policyReadFaultClock{}}
			got, _, err := s.signedAccess(ctx, u, scope, ports.PermissionWorkContractAcquire, "")
			if tc.failure != nil && tc.code != d.ErrorCodeForbidden {
				if !errors.Is(err, tc.failure) {
					t.Fatal("policy read failure was replaced", err)
				}
			} else if tc.code != "" {
				if code, _ := d.ErrorCodeOf(err); code != tc.code {
					t.Fatal("policy denial changed", err)
				}
			} else if err != nil || got.CredentialID != cid || got.CredentialPolicyRevision != policy.Version {
				t.Fatal("valid policy no longer grants the original scoped authority", got, err)
			}
		})
	}
}

func TestSigningMutationPreservesEnrollmentPolicyReadFailure(t *testing.T) {
	now := policyReadFaultClock{}.Now()
	ns := d.MustParseID("0199f392-0000-7000-8000-000000000001")
	cid := d.MustParseID("0199f392-0000-7000-8000-000000000002")
	key := d.MustParseID("0199f392-0000-7000-8000-000000000003")
	server := "0199f392-0000-7000-8000-000000000004"
	enrollment := d.SigningEnrollment{ID: d.MustParseID("0199f392-0000-7000-8000-000000000005"), TenantID: ns, CredentialID: cid, PrincipalID: "executor", KeyID: key, Nonce: "policy-read-fault-challenge", Purpose: "agent", ExpiresAt: now.Add(time.Minute), Version: 1}
	pub, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	payload := signing.EnrollmentPayload{Binding: signing.Binding{ProtocolVersion: 3, ServerID: server, TenantID: ns.String(), PrincipalID: "executor", SignerKeyID: key.String()}, EnrollmentID: enrollment.ID.String(), Nonce: enrollment.Nonce, Purpose: enrollment.Purpose, PublicKey: base64.StdEncoding.EncodeToString(pub), ExpiresAt: enrollment.ExpiresAt.UTC().Format(time.RFC3339Nano)}
	proof, err := signing.Sign(signing.KeyEnrollment, payload, key.String(), private)
	if err != nil {
		t.Fatal(err)
	}
	id := Identity{TenantID: ns, PrincipalID: "executor", CredentialDigest: "authenticated-digest"}
	ctx := WithIdentity(context.Background(), id)
	conflict := d.NewError(d.ErrorCodeTransactionConflict, "transactional enrollment policy unavailable")
	for _, failure := range []error{conflict, d.NewError(d.ErrorCodeNotFound, "policy absent")} {
		u := policyReadFaultUOW{policyReadFaultRegistry: policyReadFaultRegistry{credential: ports.Credential{ID: cid, TenantID: ns, PrincipalID: id.PrincipalID, ExpiresAt: now.Add(time.Hour)}, enrollment: enrollment, err: failure}}
		s := SecurityService{Store: policyReadFaultSecurityStore{u: u}, Clock: policyReadFaultClock{}, ServerID: server}
		_, err := s.SigningMutation(ctx, "enroll-policy-fault", SigningSecurityIntent{TenantID: ns, Operation: "register_signing_key", EnrollmentID: enrollment.ID, Proof: &proof})
		if failure == conflict {
			if !errors.Is(err, conflict) {
				t.Fatal("enrollment policy failure was disguised as denial", err)
			}
		} else if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeForbidden {
			t.Fatal("absent enrollment policy no longer denies enrollment", err)
		}
	}
}
