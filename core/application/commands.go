package application

import (
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

type MutationResult[T any] struct {
	Value            T                      `json:"value"`
	OutcomeRevision  domain.OutcomeRevision `json:"outcome_revision"`
	CommandID        domain.ID              `json:"command_id"`
	IdempotentReplay bool                   `json:"idempotent_replay"`
}

type CreateOutcomeCommand struct {
	NamespaceID  domain.ID
	Title        string
	Description  string
	DesiredState string
	Priority     domain.Priority
}

type AddCriterionCommand struct {
	Owner            domain.EntityRef
	ExpectedVersion  domain.Version
	Title            string
	Description      string
	Required         bool
	VerificationMode domain.VerificationMode
}

type ReviseCriterionCommand struct {
	Owner            domain.EntityRef
	CriterionID      domain.ID
	ExpectedVersion  domain.Version
	Title            string
	Description      string
	Required         bool
	VerificationMode domain.VerificationMode
}

type AttestCriterionCommand struct {
	Owner             domain.EntityRef
	CriterionID       domain.ID
	CriterionRevision domain.CriterionRevision
	ExpectedVersion   domain.Version
	Result            domain.AssessmentResult
	Rationale         string
}

type ActivateOutcomeCommand struct {
	Scope           domain.Scope
	ExpectedVersion domain.Version
}

type AchieveOutcomeCommand struct {
	Scope           domain.Scope
	ExpectedVersion domain.Version
	Reason          string
}

type CreateObjectiveCommand struct {
	Scope              domain.Scope
	Title              string
	Description        string
	Priority           domain.Priority
	RequiredForOutcome bool
	ParentObjectiveID  *domain.ID
}

type StartObjectiveCommand struct {
	Scope           domain.Scope
	ObjectiveID     domain.ID
	ExpectedVersion domain.Version
}

type AchieveObjectiveCommand struct {
	Scope           domain.Scope
	ObjectiveID     domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

type CreateWorkItemCommand struct {
	Scope       domain.Scope
	Title       string
	Description string
	Priority    domain.Priority
	Lifecycle   domain.WorkItemLifecycle
	ObjectiveID *domain.ID
}

type ActivateWorkItemCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version
}

type ClaimWorkItemCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version
	TTL             time.Duration
}

type CompleteWorkItemCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version
	ClaimID         domain.ID
	FencingToken    uint64
	ResultSummary   string
	Reason          string
}
