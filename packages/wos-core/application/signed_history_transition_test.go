package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

type historicalFactRegistry struct {
	ports.SigningIdentityRepository
	key d.SigningKey
}

func (r historicalFactRegistry) Key(_ context.Context, tenant, id d.ID) (d.SigningKey, error) {
	if tenant != r.key.TenantID || id != r.key.ID {
		return d.SigningKey{}, d.NewError(d.ErrorCodeNotFound, "key outside Tenant")
	}
	return r.key, nil
}

type historicalFactServer struct {
	ports.ServerIdentityRepository
	server d.ServerIdentity
}

func (r historicalFactServer) Server(context.Context) (d.ServerIdentity, error) { return r.server, nil }

type historicalFactUOW struct {
	ports.UnitOfWork
	registry historicalFactRegistry
	server   historicalFactServer
}

func (u historicalFactUOW) SigningIdentity() ports.SigningIdentityRepository { return u.registry }
func (u historicalFactUOW) ServerIdentity() ports.ServerIdentityRepository   { return u.server }

// A persisted v2 fact is verified under its original DSSE purpose and spelling.
// Reading it must neither re-sign it nor turn it into a current mutation intent.
func TestHistoricalSignedFactSurvivesTenantContractTransition(t *testing.T) {
	now := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	tenant := d.MustParseID("0199f390-0000-7000-8000-000000000001")
	keyID := d.MustParseID("0199f390-0000-7000-8000-000000000002")
	serverID := d.MustParseID("0199f390-0000-7000-8000-000000000003")
	contractID := d.MustParseID("0199f390-0000-7000-8000-000000000004")
	scope := d.Scope{TenantID: tenant, OutcomeID: d.MustParseID("0199f390-0000-7000-8000-000000000005")}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := signing.Fingerprint(public)
	if err != nil {
		t.Fatal(err)
	}
	server := d.ServerIdentity{ID: serverID, IssuerKeyID: keyID, PublicKey: base64.StdEncoding.EncodeToString(public), Fingerprint: fingerprint, CreatedAt: now}
	key := d.SigningKey{ID: keyID, TenantID: tenant, PrincipalID: server.PrincipalID(), Purpose: "issuer", Algorithm: "Ed25519", PublicKey: server.PublicKey, Fingerprint: fingerprint, Version: 1, Status: "retired", CreatedAt: now, UpdatedAt: now}
	u := historicalFactUOW{registry: historicalFactRegistry{key: key}, server: historicalFactServer{server: server}}
	raw, err := signing.Canonical(map[string]any{
		"protocol_version": 2, "namespace_id": tenant.String(), "outcome_id": scope.OutcomeID.String(),
		"server_id": serverID.String(), "signer_key_id": keyID.String(), "principal_id": "executor",
		"contract_kind": "execution", "contract_id": contractID.String(), "work_item_id": contractID.String(),
		"execution_id": contractID.String(), "fencing_token": "1", "contract_version": "1", "lease_version": "1",
		"issued_at": now.Format(time.RFC3339Nano), "expires_at": now.Add(time.Minute).Format(time.RFC3339Nano),
		"spec_digest": signing.Digest([]byte("historical specification")), "acceptance_mode": "direct", "policy_revision": "1", "allowed_signing_key_ids": []string{keyID.String()},
	})
	if err != nil {
		t.Fatal(err)
	}
	purpose := signing.PayloadType("application/vnd.wos.contract-authority.v2+json")
	proof := signing.Proof{PayloadType: purpose, KeyID: keyID.String(), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, signing.PAE(purpose, raw)))}
	proofRaw, err := json.Marshal(proof)
	if err != nil {
		t.Fatal(err)
	}
	fact := d.SignedFact{ID: contractID, Scope: scope, ContractID: contractID, ContractKind: "execution", Kind: "authority", PrincipalID: "executor", KeyID: keyID, Payload: raw, PayloadDigest: signing.Digest(raw), Proof: proofRaw, RecordedAt: now}
	if err = fact.Validate(); err != nil {
		t.Fatal(err)
	}
	doc, err := verifyIssuedFact(context.Background(), u, fact, signing.ContractAuthority, serverID.String(), "executor")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(doc.Payload, raw) || doc.Proof != proof {
		t.Fatal("historical document bytes or proof changed")
	}
	var view SignedStateResult
	if err = fillSignedFactView(context.Background(), u, SignedStateQuery{Scope: scope}, fact, &view); err != nil {
		t.Fatal(err)
	}
	if view.Envelope == nil || view.Envelope.PayloadType != purpose || view.PayloadDigest != fact.PayloadDigest {
		t.Fatal("historical read changed proof purpose or digest")
	}
	verified, err := base64.StdEncoding.DecodeString(view.Envelope.Payload)
	if err != nil || !bytes.Equal(verified, raw) {
		t.Fatal("historical read changed payload", err)
	}
	if _, err = signing.Verify(*view.Envelope, signing.ContractAuthority, keyID.String(), public); err == nil {
		t.Fatal("historical authority became a valid current protocol intent")
	}
	wrongScope := fact
	wrongScope.Scope.OutcomeID = contractID
	if _, err = verifyIssuedFact(context.Background(), u, wrongScope, signing.ContractAuthority, serverID.String(), "executor"); err == nil {
		t.Fatal("historical verification accepted different authenticated scope")
	}
	tampered := fact
	tampered.Payload = bytes.ReplaceAll(raw, []byte("executor"), []byte("intruder"))
	tampered.PayloadDigest = signing.Digest(tampered.Payload)
	if _, err = verifyIssuedFact(context.Background(), u, tampered, signing.ContractAuthority, serverID.String(), "intruder"); err == nil {
		t.Fatal("historical verification trusted tampered bytes")
	}
	current := d.WorkContract{SignedBinding: &d.SignedContractBinding{ProtocolVersion: 2}, LatestAuthorityID: &contractID, IssuedSpecificationID: &contractID}
	if err = requireIssuedWorkAuthority(context.Background(), u, current, ContractAuthority{}, signing.Digest(raw)); err == nil {
		t.Fatal("historical authority can be renewed as current work")
	}
}
