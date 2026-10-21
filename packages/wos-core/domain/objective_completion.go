package domain

import (
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

// ObjectiveCompletionMode controls whether Objective achievement is always an
// explicit operation or may be derived from a separately authorized rule.
// An empty value preserves the historical manual behavior.
type ObjectiveCompletionMode string

const (
	ObjectiveCompletionModeManual    ObjectiveCompletionMode = "manual"
	ObjectiveCompletionModeAutomatic ObjectiveCompletionMode = "automatic"
)

// ObjectiveCompletionAuthority records one explicit authorization of the
// Objective completion mode. It is append-only history; the Objective's
// CompletionMode remains the current-state projection. Automatic mode is
// valid only when this exact authority is bound to the current completion
// cycle and to a credential policy revision.
type ObjectiveCompletionAuthority struct {
	ID                       ID                      `json:"id"`
	Scope                    Scope                   `json:"scope"`
	ObjectiveID              ID                      `json:"objective_id"`
	PolicyVersion            uint64                  `json:"policy_version,string"`
	Mode                     ObjectiveCompletionMode `json:"mode"`
	CompletionCycle          uint64                  `json:"completion_cycle,string"`
	DelegatorPrincipalID     string                  `json:"delegator_principal_id"`
	DelegatorActor           ActorRef                `json:"delegator_actor"`
	DelegatorCredentialID    *ID                     `json:"delegator_credential_id,omitempty"`
	CredentialPolicyRevision Version                 `json:"credential_policy_revision,string"`
	OperationalProfile       OperationalProfile      `json:"operational_profile"`
	CompletionPolicy         CompletionPolicy        `json:"completion_policy"`
	AutomationActor          ActorRef                `json:"automation_actor"`
	Reason                   string                  `json:"reason"`
	AuthorizedAt             time.Time               `json:"authorized_at"`
}

// ObjectiveCompletionDerivation records the exact delegated rule that produced
// an Objective conclusion. It is attached to the conclusion itself so the
// provenance remains available with the aggregate's current and historical
// conclusions, independently of event or receipt retention.
type ObjectiveCompletionDerivation struct {
	AuthorityID              ID        `json:"authority_id"`
	Scope                    Scope     `json:"scope"`
	PolicyVersion            uint64    `json:"policy_version,string"`
	CompletionCycle          uint64    `json:"completion_cycle,string"`
	DelegatorPrincipalID     string    `json:"delegator_principal_id"`
	DelegatorCredentialID    ID        `json:"delegator_credential_id"`
	CredentialPolicyRevision Version   `json:"credential_policy_revision,string"`
	EvaluatedAt              time.Time `json:"evaluated_at"`
}

const ObjectiveCompletionAutomationPrincipalID = "wos:objective_completion"

func (d ObjectiveCompletionDerivation) Validate() error {
	if err := d.AuthorityID.Validate(); err != nil {
		return err
	}
	if err := d.Scope.Validate(); err != nil {
		return err
	}
	if d.PolicyVersion == 0 || d.CompletionCycle == 0 || d.CredentialPolicyRevision == 0 || d.EvaluatedAt.IsZero() {
		return NewError(ErrorCodeInvalidArgument, "Objective completion derivation provenance is incomplete")
	}
	if strings.TrimSpace(d.DelegatorPrincipalID) == "" || len(d.DelegatorPrincipalID) > 256 {
		return NewError(ErrorCodeInvalidArgument, "Objective completion derivation delegator is required")
	}
	return d.DelegatorCredentialID.Validate()
}

func (a ObjectiveCompletionAuthority) Validate() error {
	if err := a.ID.Validate(); err != nil {
		return err
	}
	if err := a.Scope.Validate(); err != nil {
		return err
	}
	if err := a.ObjectiveID.Validate(); err != nil {
		return err
	}
	if a.PolicyVersion == 0 || a.CompletionCycle == 0 || !a.Mode.Valid() || !a.OperationalProfile.Valid() || !a.CompletionPolicy.Valid() {
		return NewError(ErrorCodeInvalidArgument, "invalid Objective completion authority version, cycle, mode, or policy snapshot")
	}
	if strings.TrimSpace(a.DelegatorPrincipalID) == "" || len(a.DelegatorPrincipalID) > 256 {
		return NewError(ErrorCodeInvalidArgument, "Objective completion authority delegator is required")
	}
	if err := a.DelegatorActor.Validate(); err != nil {
		return err
	}
	if a.AutomationActor != ObjectiveCompletionAutomationActor() {
		return NewError(ErrorCodeInvalidArgument, "Objective completion automation actor is not the host-controlled identity")
	}
	if strings.TrimSpace(a.Reason) == "" || !utf8.ValidString(a.Reason) || strings.ContainsRune(a.Reason, 0) || utf8.RuneCountInString(a.Reason) > 1024 {
		return NewError(ErrorCodeInvalidArgument, "Objective completion authority reason is required and must not exceed 1024 characters")
	}
	if a.AuthorizedAt.IsZero() {
		return NewError(ErrorCodeInvalidArgument, "Objective completion authority timestamp is required")
	}
	if a.DelegatorCredentialID == nil {
		if a.CredentialPolicyRevision != 0 {
			return NewError(ErrorCodeInvalidArgument, "Objective completion credential policy revision requires a credential")
		}
	} else if err := a.DelegatorCredentialID.Validate(); err != nil {
		return err
	} else if a.CredentialPolicyRevision == 0 {
		return NewError(ErrorCodeInvalidArgument, "Objective completion credential policy revision is required")
	}
	if a.Mode.Effective() == ObjectiveCompletionModeAutomatic {
		if a.DelegatorCredentialID == nil || a.CredentialPolicyRevision == 0 ||
			a.OperationalProfile.Effective() != OperationalProfileOperationalV2 ||
			a.CompletionPolicy.Effective() != CompletionPolicyStrictWork {
			return NewError(ErrorCodeInvalidArgument, "automatic Objective completion requires a credential-bound strict operational policy")
		}
	}
	return nil
}

func ObjectiveCompletionAutomationActor() ActorRef {
	return ActorRef{Kind: ActorKindAutomation, Provider: "wos", ID: "objective_completion"}
}

func NextObjectiveCompletionPolicyVersion(version uint64) (uint64, error) {
	if version == math.MaxUint64 {
		return 0, NewError(ErrorCodeInvalidVersion, "Objective completion policy version overflow")
	}
	return version + 1, nil
}

func (m ObjectiveCompletionMode) Effective() ObjectiveCompletionMode {
	if m == "" {
		return ObjectiveCompletionModeManual
	}
	return m
}

func (m ObjectiveCompletionMode) Valid() bool {
	switch m.Effective() {
	case ObjectiveCompletionModeManual, ObjectiveCompletionModeAutomatic:
		return true
	default:
		return false
	}
}

// EffectiveCompletionCycle returns the first cycle for legacy Objectives that
// predate explicit cycle persistence.
func (o Objective) EffectiveCompletionCycle() uint64 {
	if o.CompletionCycle == 0 {
		return 1
	}
	return o.CompletionCycle
}

// ValidateCompletionConfiguration checks the policy combination that cannot
// be validated by either aggregate in isolation.
func (o Objective) ValidateCompletionConfiguration(outcome Outcome) error {
	if outcome.Scope() != o.Scope {
		return NewError(ErrorCodeInvalidArgument, "objective completion policy must use its owning outcome")
	}
	if !outcome.OperationalProfile.Valid() || !outcome.CompletionPolicy.Valid() || !o.CompletionMode.Valid() {
		return NewError(ErrorCodeInvalidArgument, "invalid operational completion policy")
	}
	if o.CompletionMode.Effective() == ObjectiveCompletionModeAutomatic &&
		(outcome.OperationalProfile.Effective() != OperationalProfileOperationalV2 ||
			outcome.CompletionPolicy.Effective() != CompletionPolicyStrictWork) {
		return NewError(ErrorCodeInvalidArgument, "automatic objective completion requires operational_v2 and strict_work")
	}
	return nil
}

func nextObjectiveCompletionCycle(cycle uint64) (uint64, error) {
	if cycle == math.MaxUint64 {
		return 0, NewError(ErrorCodeInvalidVersion, "objective completion cycle overflow")
	}
	return cycle + 1, nil
}
