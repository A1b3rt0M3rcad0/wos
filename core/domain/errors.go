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
	ErrorCodeInvalidID             ErrorCode = "invalid_id"
	ErrorCodeInvalidScope          ErrorCode = "invalid_scope"
	ErrorCodeInvalidEntityKind     ErrorCode = "invalid_entity_kind"
	ErrorCodeInvalidEntityRef      ErrorCode = "invalid_entity_ref"
	ErrorCodeInvalidActorKind      ErrorCode = "invalid_actor_kind"
	ErrorCodeInvalidActorRef       ErrorCode = "invalid_actor_ref"
	ErrorCodeInvalidVersion        ErrorCode = "invalid_version"
	ErrorCodeInvalidConfig         ErrorCode = "invalid_configuration"
	ErrorCodeInvalidCommandContext ErrorCode = "invalid_command_context"
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

// NewError creates a typed WOS error without a wrapped cause.
func NewError(code ErrorCode, message string) error {
	return &Error{Code: code, Message: message}
}

// WrapError creates a typed WOS error while preserving the original cause.
func WrapError(code ErrorCode, message string, cause error) error {
	return &Error{Code: code, Message: message, Cause: cause}
}

// ErrorCodeOf extracts a stable WOS error code from an error chain.
func ErrorCodeOf(err error) (ErrorCode, bool) {
	var target *Error
	if !errors.As(err, &target) || target == nil {
		return "", false
	}
	return target.Code, true
}
