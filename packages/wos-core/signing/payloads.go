package signing

import (
	"encoding/json"
	"strconv"
)

// Decimal is the exact new-protocol representation for counters and versions.
// Callers validate whether zero is permitted for each particular field.
type Decimal uint64

func (v Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatUint(uint64(v), 10))
}
func (v *Decimal) UnmarshalJSON(raw []byte) error {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return invalid("invalid_payload", "counter must be a decimal string")
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || strconv.FormatUint(n, 10) != s {
		return invalid("invalid_payload", "counter must be a canonical uint64 string")
	}
	*v = Decimal(n)
	return nil
}

// Binding is authenticated inside every WOS payload. PrincipalID names the
// operational subject; SignerKeyID identifies the actual signing identity.
// Application validates registry ownership and server/subject roles per purpose.
type Binding struct {
	ProtocolVersion int    `json:"protocol_version"`
	ServerID        string `json:"server_id"`
	NamespaceID     string `json:"namespace_id"`
	OutcomeID       string `json:"outcome_id,omitempty"`
	PrincipalID     string `json:"principal_id"`
	SignerKeyID     string `json:"signer_key_id"`
}
type SpecPayload[T any] struct {
	Binding
	ContractKind string `json:"contract_kind"`
	WorkItemID   string `json:"work_item_id"`
	ContractID   string `json:"contract_id"`
	SpecDigest   string `json:"spec_digest"`
	Spec         T      `json:"spec"`
}
type AuthorityPayload struct {
	Binding
	ContractKind         string   `json:"contract_kind"`
	ContractID           string   `json:"contract_id"`
	WorkItemID           string   `json:"work_item_id"`
	ReviewCaseID         string   `json:"review_case_id,omitempty"`
	ExecutionID          string   `json:"execution_id"`
	FencingToken         Decimal  `json:"fencing_token"`
	ContractVersion      Decimal  `json:"contract_version"`
	LeaseVersion         Decimal  `json:"lease_version"`
	IssuedAt             string   `json:"issued_at"`
	ExpiresAt            string   `json:"expires_at"`
	SpecDigest           string   `json:"spec_digest"`
	AcceptanceMode       string   `json:"acceptance_mode"`
	PolicyRevision       Decimal  `json:"policy_revision"`
	AllowedSigningKeyIDs []string `json:"allowed_signing_key_ids"`
}
type RequestBinding struct {
	Binding
	OperationKind           string  `json:"operation_kind"`
	RequestID               string  `json:"request_id"`
	IdempotencyKey          string  `json:"idempotency_key"`
	ContractID              string  `json:"contract_id"`
	WorkItemID              string  `json:"work_item_id"`
	ExecutionID             string  `json:"execution_id"`
	FencingToken            Decimal `json:"fencing_token"`
	SpecDigest              string  `json:"spec_digest"`
	AuthorityDigest         string  `json:"authority_digest"`
	ExpectedContractVersion Decimal `json:"expected_contract_version"`
	ExpectedLeaseVersion    Decimal `json:"expected_lease_version"`
	ExpectedWorkItemVersion Decimal `json:"expected_work_item_version"`
	PolicyRevision          Decimal `json:"policy_revision"`
}
type WorkReturnPayload[T any] struct {
	RequestBinding
	CompletionIntent       string `json:"completion_intent"`
	Material               T      `json:"material"`
	SupersedesSubmissionID string `json:"supersedes_submission_id,omitempty"`
}
type ReviewReturnPayload[T any] struct {
	RequestBinding
	ReviewCaseID     string `json:"review_case_id"`
	SubmissionID     string `json:"submission_id"`
	SubmissionDigest string `json:"submission_digest"`
	Decision         string `json:"decision"`
	Material         T      `json:"material"`
}
type ReceiptPayload struct {
	Binding
	RequestID             string  `json:"request_id"`
	IdempotencyKey        string  `json:"idempotency_key"`
	RequestDigest         string  `json:"request_digest"`
	ContractID            string  `json:"contract_id"`
	WorkItemID            string  `json:"work_item_id"`
	SubmissionID          string  `json:"submission_id,omitempty"`
	ReviewCaseID          string  `json:"review_case_id,omitempty"`
	DecisionID            string  `json:"decision_id,omitempty"`
	OutcomeRevision       Decimal `json:"outcome_revision"`
	Accepted              bool    `json:"accepted"`
	Disposition           string  `json:"disposition"`
	ContractStatus        string  `json:"contract_status"`
	WorkItemLifecycle     string  `json:"work_item_lifecycle"`
	LocalObligationClosed bool    `json:"local_obligation_closed"`
	AcceptedAt            string  `json:"accepted_at"`
}
type EnrollmentPayload struct {
	Binding
	EnrollmentID  string `json:"enrollment_id"`
	Nonce         string `json:"nonce"`
	Purpose       string `json:"purpose"`
	PublicKey     string `json:"public_key"`
	ExpiresAt     string `json:"expires_at"`
	PreviousKeyID string `json:"previous_key_id,omitempty"`
}
