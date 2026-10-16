package signing

// A separate signed intention refers to the exact original envelope. The
// original is carried beside it, avoiding recursive base64 growth in DSSE.
type ReturnResolutionPayload struct {
	Binding
	OperationKind          string  `json:"operation_kind"`
	RequestID              string  `json:"request_id"`
	IdempotencyKey         string  `json:"idempotency_key"`
	CredentialID           string  `json:"credential_id"`
	OriginalCommandName    string  `json:"original_command_name"`
	OriginalRequestID      string  `json:"original_request_id"`
	OriginalIdempotencyKey string  `json:"original_idempotency_key"`
	OriginalRequestDigest  string  `json:"original_request_digest"`
	OriginalEnvelopeDigest string  `json:"original_envelope_digest"`
	ContractID             string  `json:"contract_id"`
	PolicyRevision         Decimal `json:"policy_revision"`
	Reason                 string  `json:"reason"`
}

type ReturnResolutionReceiptPayload struct {
	Binding
	CredentialID           string  `json:"credential_id"`
	RequestID              string  `json:"request_id"`
	IdempotencyKey         string  `json:"idempotency_key"`
	RequestDigest          string  `json:"request_digest"`
	OriginalCommandName    string  `json:"original_command_name"`
	OriginalRequestID      string  `json:"original_request_id"`
	OriginalIdempotencyKey string  `json:"original_idempotency_key"`
	OriginalRequestDigest  string  `json:"original_request_digest"`
	OriginalEnvelopeDigest string  `json:"original_envelope_digest"`
	ContractID             string  `json:"contract_id"`
	Decision               string  `json:"decision"`
	Reason                 string  `json:"reason"`
	OutcomeRevision        Decimal `json:"outcome_revision"`
	ResolvedAt             string  `json:"resolved_at"`
}
