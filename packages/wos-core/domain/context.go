package domain

import "strings"

// ExternalExecutionContext carries optional correlation with the runtime or
// system that caused a WOS command. It is observational context only: these
// fields do not grant authorization and are not the persistent address of an
// Outcome.
type ExternalExecutionContext struct {
	Provider  string `json:"provider,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	NetworkID string `json:"network_id,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	RunID     string `json:"run_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
}

// Empty reports whether no external execution correlation was supplied.
func (c ExternalExecutionContext) Empty() bool {
	return c.Provider == "" &&
		c.ProjectID == "" &&
		c.NetworkID == "" &&
		c.AgentID == "" &&
		c.SessionID == "" &&
		c.RunID == "" &&
		c.TraceID == ""
}

// CommandContext is the transport-neutral context attached to an application
// command. PrincipalID represents the authenticated identity; Actor is the
// authorized declared author. They intentionally remain separate.
type CommandContext struct {
	PrincipalID    string                   `json:"principal_id"`
	Actor          ActorRef                 `json:"actor"`
	Execution      ExternalExecutionContext `json:"execution,omitempty"`
	CorrelationID  string                   `json:"correlation_id,omitempty"`
	CausationID    *ID                      `json:"causation_id,omitempty"`
	CommandID      ID                       `json:"command_id"`
	IdempotencyKey string                   `json:"idempotency_key,omitempty"`
}

// Validate checks only invariants that apply to every command. Whether an
// idempotency key, execution field, or correlation ID is required is decided
// by the concrete application operation and transport contract.
func (c CommandContext) Validate() error {
	if strings.TrimSpace(c.PrincipalID) == "" {
		return NewError(ErrorCodeInvalidCommandContext, "principal_id is required")
	}
	if err := c.Actor.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidCommandContext, "actor is invalid", err)
	}
	if err := c.CommandID.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidCommandContext, "command_id is invalid", err)
	}
	if c.CausationID != nil {
		if err := c.CausationID.Validate(); err != nil {
			return WrapError(ErrorCodeInvalidCommandContext, "causation_id is invalid", err)
		}
	}
	return nil
}
