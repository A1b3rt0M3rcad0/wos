package domain_test

import (
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestCommandContextSeparatesPrincipalActorAndExecution(t *testing.T) {
	ctx := domain.CommandContext{
		PrincipalID: "service-account-product",
		Actor: domain.ActorRef{
			Kind:        domain.ActorKindAgent,
			Provider:    "woobe",
			ID:          "agent-architecture",
			DisplayName: "Architecture Agent",
		},
		Execution: domain.ExternalExecutionContext{
			Provider:  "woobe",
			ProjectID: "project-01",
			NetworkID: "network-02",
			AgentID:   "agent-03",
			SessionID: "session-04",
			RunID:     "run-05",
			TraceID:   "trace-06",
		},
		CorrelationID: "request-07",
		CommandID:     domain.MustParseID("0199e100-0000-7000-8000-000000000030"),
	}

	if err := ctx.Validate(); err != nil {
		t.Fatalf("CommandContext.Validate() error = %v", err)
	}
	if ctx.PrincipalID == ctx.Actor.ID {
		t.Fatal("test fixture must demonstrate that principal and declared actor are independent identities")
	}
	if ctx.Execution.Empty() {
		t.Fatal("execution context unexpectedly reported empty")
	}
}

func TestCommandContextAllowsNoExternalExecutionContext(t *testing.T) {
	ctx := domain.CommandContext{
		PrincipalID: "local-user",
		Actor: domain.ActorRef{
			Kind:     domain.ActorKindHuman,
			Provider: "local",
			ID:       "local-user",
		},
		CommandID: domain.MustParseID("0199e100-0000-7000-8000-000000000031"),
	}

	if !ctx.Execution.Empty() {
		t.Fatal("zero execution context must be optional")
	}
	if err := ctx.Validate(); err != nil {
		t.Fatalf("CommandContext.Validate() error = %v", err)
	}
}

func TestCommandContextRejectsMissingAuthenticatedPrincipal(t *testing.T) {
	ctx := domain.CommandContext{
		Actor: domain.ActorRef{
			Kind:     domain.ActorKindHuman,
			Provider: "local",
			ID:       "local-user",
		},
		CommandID: domain.MustParseID("0199e100-0000-7000-8000-000000000032"),
	}

	err := ctx.Validate()
	if err == nil {
		t.Fatal("CommandContext.Validate() unexpectedly accepted a missing principal")
	}
	code, ok := domain.ErrorCodeOf(err)
	if !ok || code != domain.ErrorCodeInvalidCommandContext {
		t.Fatalf("ErrorCodeOf() = %q, %v; want %q, true", code, ok, domain.ErrorCodeInvalidCommandContext)
	}
}
