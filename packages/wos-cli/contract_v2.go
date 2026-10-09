package woscli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"reflect"
)

type IssuedDocumentsV2 struct {
	Specification signing.Document `json:"specification"`
	Authority     signing.Document `json:"authority"`
}
type ProgressV2 struct {
	Summary    string   `json:"summary"`
	Completed  []string `json:"completed"`
	Pending    []string `json:"pending"`
	NextAction string   `json:"next_action"`
}
type ExecutionDraftV2 struct {
	Progress         ProgressV2             `json:"progress"`
	CompletionIntent string                 `json:"completion_intent"`
	Material         a.SignedReturnMaterial `json:"material"`
}
type ReviewDraftV2 struct {
	Progress ProgressV2             `json:"progress"`
	Decision string                 `json:"decision"`
	Material a.SignedReviewMaterial `json:"material"`
}

// Only editable material uses public human command names. Issued documents and
// frozen envelopes remain exact and never pass through command normalization.
func (v ExecutionDraftV2) MarshalJSON() ([]byte, error) {
	type plain ExecutionDraftV2
	return commands.Encode(plain(v))
}
func (v *ExecutionDraftV2) UnmarshalJSON(raw []byte) error {
	type plain ExecutionDraftV2
	normalized, e := commands.Normalize(raw, reflect.TypeFor[plain]())
	if e != nil {
		return e
	}
	if e = commands.Decode(normalized, (*plain)(v)); e != nil {
		return e
	}
	return validateDraftEvidenceV2(v.Material.Evidence)
}
func (v ReviewDraftV2) MarshalJSON() ([]byte, error) {
	type plain ReviewDraftV2
	return commands.Encode(plain(v))
}
func (v *ReviewDraftV2) UnmarshalJSON(raw []byte) error {
	type plain ReviewDraftV2
	normalized, e := commands.Normalize(raw, reflect.TypeFor[plain]())
	if e != nil {
		return e
	}
	if e = commands.Decode(normalized, (*plain)(v)); e != nil {
		return e
	}
	if v.Decision != "approved" && len(v.Material.Assessments) > 0 {
		return fmt.Errorf("non-approval cannot certify criteria; use findings for changes_requested")
	}
	for _, finding := range v.Material.Findings {
		if e = finding.Validate(); e != nil {
			return fmt.Errorf("finding %q: %w", finding.ID, e)
		}
	}
	return validateDraftEvidenceV2(v.Material.Evidence)
}

func validateDraftEvidenceV2(inputs []a.SyncEvidenceInput) error {
	for _, input := range inputs {
		if e := d.ValidateEvidenceMeasurement(input.Evidence.EvidenceType, input.Evidence.Measurement); e != nil {
			return fmt.Errorf("evidence %q: %w", input.LocalKey, e)
		}
	}
	return nil
}

type FrozenReturnV2 struct {
	CredentialID    d.ID             `json:"credential_id"`
	SignerPublicKey string           `json:"signer_public_key"`
	PendingMAC      string           `json:"pending_mac"`
	State           string           `json:"state"`
	Envelope        signing.Envelope `json:"envelope"`
	DraftDigest     string           `json:"draft_digest"`
	RequestDigest   string           `json:"request_digest"`
	Response        string           `json:"response,omitempty"`
}
type ContractLocalV2 struct {
	SchemaVersion     int               `json:"schema_version"`
	Profile           string            `json:"profile"`
	ReviewCaseVersion signing.Decimal   `json:"review_case_version,omitempty"`
	WorkItemVersion   signing.Decimal   `json:"work_item_version"`
	Pending           *FrozenReturnV2   `json:"pending,omitempty"`
	AcceptanceReceipt *signing.Document `json:"acceptance_receipt,omitempty"`
}
type ContractFileV2 struct {
	SchemaVersion int               `json:"schema_version"`
	Kind          string            `json:"kind"`
	Issued        IssuedDocumentsV2 `json:"issued"`
	Execution     *ExecutionDraftV2 `json:"execution,omitempty"`
	Review        *ReviewDraftV2    `json:"review,omitempty"`
	Local         ContractLocalV2   `json:"_local"`
}
type ContractViewV2 struct {
	Authority           signing.AuthorityPayload
	WorkSpecification   *signing.SpecPayload[d.SignedWorkSpec]
	ReviewSpecification *signing.SpecPayload[a.SignedReviewSpec]
}

func verifyProfileDocumentV2(profile ProfileV2, document signing.Document, purpose signing.PayloadType) ([]byte, error) {
	if document.Proof.PayloadType != purpose {
		return nil, fmt.Errorf("issued document purpose differs")
	}
	for _, key := range profile.Binding.IssuerKeys {
		if key.KeyID.String() != document.Proof.KeyID {
			continue
		}
		public, e := base64.StdEncoding.Strict().DecodeString(key.PublicKey)
		if e != nil || signing.Digest(public) != key.Fingerprint {
			return nil, fmt.Errorf("profile issuer pin differs")
		}
		envelope, e := document.Envelope()
		if e != nil {
			return nil, e
		}
		return signing.Verify(envelope, purpose, key.KeyID.String(), public)
	}
	return nil, fmt.Errorf("issued signing key is outside the pinned trust set")
}
func (file ContractFileV2) VerifyIssued(profile ProfileV2) (ContractViewV2, error) {
	var view ContractViewV2
	if file.SchemaVersion != 2 || file.Kind != "WOSContractFile" || file.Local.SchemaVersion != 1 || file.Local.Profile != profile.Name || (file.Execution == nil) == (file.Review == nil) {
		return view, fmt.Errorf("invalid local contract discriminator/profile")
	}
	authority, e := verifyProfileDocumentV2(profile, file.Issued.Authority, signing.ContractAuthority)
	if e != nil {
		return view, e
	}
	if e = signing.DecodeStrict(authority, &view.Authority, signing.MaxPayloadBytes); e != nil {
		return view, e
	}
	grant := view.Authority
	if grant.ServerID != profile.Binding.ServerID.String() || grant.NamespaceID != profile.Binding.NamespaceID.String() || grant.PrincipalID != profile.Binding.PrincipalID || d.ID(grant.ContractID).Validate() != nil || d.ID(grant.WorkItemID).Validate() != nil || d.ID(grant.OutcomeID).Validate() != nil {
		return view, fmt.Errorf("issuer authority destination/identity binding differs")
	}
	specification, e := verifyProfileDocumentV2(profile, file.Issued.Specification, signing.ContractSpec)
	if e != nil {
		return view, e
	}
	if signing.Digest(specification) != grant.SpecDigest {
		return view, fmt.Errorf("issued specification/grant digest differs")
	}
	var binding signing.SpecPayload[json.RawMessage]
	if e = signing.DecodeStrict(specification, &binding, 128<<10); e != nil {
		return view, e
	}
	if binding.ServerID != grant.ServerID || binding.NamespaceID != grant.NamespaceID || binding.OutcomeID != grant.OutcomeID || binding.PrincipalID != grant.PrincipalID || binding.ContractID != grant.ContractID || binding.WorkItemID != grant.WorkItemID || binding.ContractKind != grant.ContractKind {
		return view, fmt.Errorf("issued specification/authority targets differ")
	}
	switch grant.ContractKind {
	case "execution":
		if file.Execution == nil {
			return view, fmt.Errorf("execution draft required")
		}
		var spec signing.SpecPayload[d.SignedWorkSpec]
		if e = signing.DecodeStrict(specification, &spec, 128<<10); e != nil {
			return view, e
		}
		view.WorkSpecification = &spec
	case "review":
		if file.Review == nil {
			return view, fmt.Errorf("review draft required")
		}
		var spec signing.SpecPayload[a.SignedReviewSpec]
		if e = signing.DecodeStrict(specification, &spec, 128<<10); e != nil {
			return view, e
		}
		view.ReviewSpecification = &spec
	default:
		return view, fmt.Errorf("unknown signed contract kind")
	}
	return view, nil
}
func (file ContractFileV2) DraftDigest() (string, error) {
	var material any
	if file.Execution != nil && file.Review == nil {
		type plain ExecutionDraftV2
		material = plain(*file.Execution)
	} else if file.Review != nil && file.Execution == nil {
		type plain ReviewDraftV2
		material = plain(*file.Review)
	} else {
		return "", fmt.Errorf("exactly one editable draft required")
	}
	raw, e := signing.Canonical(material)
	if e != nil {
		return "", e
	}
	return signing.Digest(raw), nil
}
