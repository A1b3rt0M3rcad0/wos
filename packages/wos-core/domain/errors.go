package domain

import (
	"errors"
	"fmt"
)

// ErrorCode is a stable machine-readable classification for WOS domain and
// application errors. Public adapters should map codes instead of matching
// human-readable error strings.
type ErrorCode string

const (
	ErrorCodeContractProtocolRequired ErrorCode = "contract_protocol_required"
	ErrorCodeContractExpired          ErrorCode = "contract_expired"
	ErrorCodeContractRevoked          ErrorCode = "contract_revoked"
	ErrorCodeStaleExecution           ErrorCode = "stale_execution"
	ErrorCodeContractSpecMismatch     ErrorCode = "contract_spec_mismatch"
	ErrorCodeWorkAlreadyClaimed       ErrorCode = "work_already_claimed"
	ErrorCodeSubmissionNotAccepted    ErrorCode = "submission_not_accepted"

	ErrorCodeTransactionConflict   ErrorCode = "transaction_conflict"
	ErrorCodeInvalidID             ErrorCode = "invalid_id"
	ErrorCodeInvalidScope          ErrorCode = "invalid_scope"
	ErrorCodeInvalidEntityKind     ErrorCode = "invalid_entity_kind"
	ErrorCodeInvalidEntityRef      ErrorCode = "invalid_entity_ref"
	ErrorCodeInvalidActorKind      ErrorCode = "invalid_actor_kind"
	ErrorCodeInvalidActorRef       ErrorCode = "invalid_actor_ref"
	ErrorCodeInvalidVersion        ErrorCode = "invalid_version"
	ErrorCodeInvalidConfig         ErrorCode = "invalid_configuration"
	ErrorCodeInvalidCommandContext ErrorCode = "invalid_command_context"
	ErrorCodeInvalidArgument       ErrorCode = "invalid_argument"
	ErrorCodeInvalidTransition     ErrorCode = "invalid_transition"
	ErrorCodePreconditionFailed    ErrorCode = "precondition_failed"
	ErrorCodeVersionConflict       ErrorCode = "version_conflict"
	ErrorCodeNotFound              ErrorCode = "not_found"
	ErrorCodeAlreadyExists         ErrorCode = "already_exists"
	ErrorCodeForbidden             ErrorCode = "forbidden"
	ErrorCodeCriterion             ErrorCode = "criterion_error"
	ErrorCodeAssessment            ErrorCode = "assessment_error"
	ErrorCodeLease                 ErrorCode = "lease_error"
	ErrorCodeInvalidEvent          ErrorCode = "invalid_event"
	ErrorCodeInvalidIdempotencyKey ErrorCode = "invalid_idempotency_key"
	ErrorCodeIdempotencyConflict   ErrorCode = "idempotency_conflict"
	ErrorCodeIdempotencyState      ErrorCode = "idempotency_state"
	ErrorCodeInvalidRelation       ErrorCode = "invalid_relation"
	ErrorCodeDependencyCycle       ErrorCode = "dependency_cycle"
	ErrorCodeGraphLimitExceeded    ErrorCode = "graph_limit_exceeded"
	ErrorCodeIssue                 ErrorCode = "issue_error"
	ErrorCodeBlocker               ErrorCode = "blocker_error"
	ErrorCodeArtifact              ErrorCode = "artifact_error"
	ErrorCodeEvidence              ErrorCode = "evidence_error"
	ErrorCodeEvidenceLink          ErrorCode = "evidence_link_error"
	ErrorCodeDecision              ErrorCode = "decision_error"
	ErrorCodeRoadmap               ErrorCode = "roadmap_error"
)

// Error is the foundational typed error used by public Core contracts.
// Details may be added by higher layers, but Code remains the stable field
// transports should map to protocol-specific responses.
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Message == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func NewError(code ErrorCode, message string) error {
	return &Error{Code: code, Message: message}
}

func WrapError(code ErrorCode, message string, cause error) error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func ErrorCodeOf(err error) (ErrorCode, bool) {
	var target *Error
	if !errors.As(err, &target) || target == nil {
		return "", false
	}
	return target.Code, true
}
