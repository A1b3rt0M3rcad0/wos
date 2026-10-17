package domain_test

import (
	"bytes"
	"encoding/json"
	"testing"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestHistoricalContractDigestPreservesOriginalScopeSpelling(t *testing.T) {
	w, contract, _ := contractFixture(t)
	criterion, err := d.NewSuccessCriterion(d.MustParseID("0199f202-0000-7000-8000-000000000010"), w.Ref(), "Verify delivery", "tenant_id is user text", true, d.VerificationModeEvidenceReview)
	if err != nil {
		t.Fatal(err)
	}
	contract.Spec.Criteria = []d.SuccessCriterion{criterion}
	contract.Spec.Dependencies = []d.EntityRef{w.Ref()}
	contract.Spec.ExecutionSpec.ContextRefs = []d.EntityRef{w.Ref()}
	current, err := json.Marshal(d.NormalizeContractSpec(contract.Spec))
	if err != nil {
		t.Fatal(err)
	}
	// Build the historical fixture using the old wire field only. No signed or
	// persisted production bytes are rewritten by the compatibility reader.
	historical := bytes.ReplaceAll(current, []byte(`"tenant_id":`), []byte(`"namespace_id":`))
	contract.SpecDigest, err = d.SemanticDigest(json.RawMessage(historical))
	if err != nil {
		t.Fatal(err)
	}
	binding := reviewBinding()
	binding.ProtocolVersion = 2
	contract.SignedBinding = &binding
	if err = contract.Validate(); err != nil {
		t.Fatal("historical specification cannot be read", err)
	}
	contract.SignedBinding = nil
	if err = contract.Validate(); err != nil {
		t.Fatal("historical unsigned specification cannot be read", err)
	}
	contract.SignedBinding = &binding
	contract.SignedBinding.ProtocolVersion = 3
	if err = contract.Validate(); err == nil {
		t.Fatal("current contract accepted historical namespace digest")
	}
	contract.SignedBinding.ProtocolVersion = 2
	contract.Spec.Criteria[0].Description = "altered historical user text"
	if err = contract.Validate(); err == nil {
		t.Fatal("historical compatibility concealed changed specification")
	}
}

func TestTenantProtocolRejectsRetiredWriterEpoch(t *testing.T) {
	p := d.DefaultWorkProtocol(d.MustParseID("0199f202-0000-7000-8000-000000000011"))
	p.Phase, p.WriterEpoch, p.WritersDrained = d.WorkProtocolSigned, 3, true
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	p.WriterEpoch = 2
	if err := p.Validate(); err == nil {
		t.Fatal("retired signed writer epoch was accepted")
	}
	p.WriterEpoch, p.Phase = 3, "signed_contracts_v2"
	if err := p.Validate(); err == nil {
		t.Fatal("retired signed protocol phase was accepted")
	}
}
