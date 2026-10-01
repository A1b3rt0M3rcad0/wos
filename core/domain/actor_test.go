package domain_test

import (
	"errors"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func TestActorRefValidation(t *testing.T) {
	valid := domain.ActorRef{
		Kind:     domain.ActorKindAgent,
		Provider: "woobe",
		ID:       "architecture-agent",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("ActorRef.Validate() error = %v", err)
	}

	invalid := valid
	invalid.Provider = "  "
	err := invalid.Validate()
	if err == nil {
		t.Fatal("ActorRef.Validate() unexpectedly accepted blank provider")
	}
	code, ok := domain.ErrorCodeOf(err)
	if !ok || code != domain.ErrorCodeInvalidActorRef {
		t.Fatalf("ErrorCodeOf() = %q, %v; want %q, true", code, ok, domain.ErrorCodeInvalidActorRef)
	}
}

func TestTypedErrorPreservesCause(t *testing.T) {
	cause := errors.New("cause")
	err := domain.WrapError(domain.ErrorCodeInvalidScope, "scope failed", cause)
	if !errors.Is(err, cause) {
		t.Fatal("wrapped error does not preserve its cause")
	}
}
