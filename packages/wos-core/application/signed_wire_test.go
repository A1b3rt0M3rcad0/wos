package application

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"math"
	"testing"
	"time"
)

func TestSignedMutationWirePreservesPrecisionAndIssuedProof(t *testing.T) {
	id := d.MustParseID("0199a555-0000-7000-8000-000000000001")
	scope := d.Scope{NamespaceID: id, OutcomeID: id}
	work, err := d.NewWorkItem(id, scope, "exact Task", "", d.PriorityNormal, d.WorkItemLifecycleTodo, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	work.Version = d.Version(math.MaxUint64)
	work.LastFencingToken = math.MaxUint64
	candidate := work
	candidate.LastFencingToken = 0
	contract, err := d.NewWorkContract(id, id, candidate, "worker", d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "worker"}, d.DefaultContractLeasePolicy(), 300, d.WorkContractSpec{Title: work.Title}, d.OutcomeRevision(math.MaxUint64), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	contract.Version = d.Version(math.MaxUint64)
	contract.LeaseVersion = 9007199254740993
	contract.SignedBinding = &d.SignedContractBinding{ProtocolVersion: 2, ServerID: id.String(), CredentialID: id, SignerKeyID: id, PolicyRevision: d.Version(math.MaxUint64), AcceptanceFloor: d.AcceptanceDirect, AllowedSigningKeyIDs: []d.ID{id}}

	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := signing.Sign(signing.ContractSpec, signing.SpecPayload[map[string]any]{Binding: signing.Binding{ProtocolVersion: 2, ServerID: id.String(), NamespaceID: id.String(), OutcomeID: id.String(), PrincipalID: "issuer", SignerKeyID: id.String()}, ContractKind: "execution", ContractID: id.String(), WorkItemID: id.String(), Spec: map[string]any{"version": uint64(5)}}, id.String(), private)
	if err != nil {
		t.Fatal(err)
	}
	document, err := signing.ToDocument(envelope)
	if err != nil {
		t.Fatal(err)
	}
	result := MutationResult[WorkContractResult]{Value: WorkContractResult{Contract: contract, WorkItem: work, IssuedSpecification: &document}, CommandID: id, OutcomeRevision: d.OutcomeRevision(math.MaxUint64)}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Revision string `json:"outcome_revision"`
		Value    struct {
			Contract struct {
				Version string `json:"version"`
				Lease   string `json:"lease_version"`
			} `json:"contract"`
			Work struct {
				Version string `json:"version"`
			} `json:"work_item"`
			Document signing.Document `json:"issued_specification"`
		} `json:"value"`
	}
	if err = json.Unmarshal(raw, &wire); err != nil {
		t.Fatal("counter emitted as a JSON number", err)
	}
	if wire.Revision != "18446744073709551615" || wire.Value.Contract.Version != wire.Revision || wire.Value.Work.Version != wire.Revision || wire.Value.Contract.Lease != "9007199254740993" {
		t.Fatal("counter rounded or not quoted")
	}
	rebuilt, err := wire.Value.Document.Envelope()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = signing.Verify(rebuilt, signing.ContractSpec, id.String(), public); err != nil {
		t.Fatal("transport changed immutable signed payload", err)
	}
	var decoded MutationResult[WorkContractResult]
	if err = json.Unmarshal(raw, &decoded); err != nil || decoded.OutcomeRevision != result.OutcomeRevision || decoded.Value.Contract.LeaseVersion != contract.LeaseVersion || decoded.Value.WorkItem.LastFencingToken != work.LastFencingToken {
		t.Fatal("typed SDK-compatible decoding lost precision", err)
	}
	result.Value.Contract.SignedBinding = nil
	legacy, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var historical map[string]json.RawMessage
	if err = json.Unmarshal(legacy, &historical); err != nil || historical["outcome_revision"][0] == '"' {
		t.Fatal("v1 outer encoding changed", err)
	}
}
