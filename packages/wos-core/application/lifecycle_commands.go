package application

import "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"

type FailOutcomeCommand struct {
	Scope           domain.Scope
	ExpectedVersion domain.Version
	Reason          string
}

type AbandonOutcomeCommand struct {
	Scope           domain.Scope
	ExpectedVersion domain.Version
	Reason          string
}

type ReopenOutcomeCommand struct {
	Scope           domain.Scope
	ExpectedVersion domain.Version
	Reason          string
}

type ArchiveOutcomeCommand struct {
	Scope           domain.Scope
	ExpectedVersion domain.Version
}

type UnarchiveOutcomeCommand struct {
	Scope           domain.Scope
	ExpectedVersion domain.Version
	Reason          string
}

type CancelObjectiveCommand struct {
	Scope           domain.Scope
	ObjectiveID     domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

type ReopenObjectiveCommand struct {
	Scope           domain.Scope
	ObjectiveID     domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

type DeferWorkItemCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version
}

type ReleaseWorkItemCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version
	ClaimID         domain.ID
	FencingToken    uint64
}

type CancelWorkItemCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

type ReopenWorkItemCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

type RetireCriterionCommand struct {
	Owner           domain.EntityRef
	CriterionID     domain.ID
	ExpectedVersion domain.Version
}

type SetOutcomeOwnersCommand struct {
	Scope           domain.Scope
	ExpectedVersion domain.Version
	OwnerRefs       []domain.ActorRef
}

type SetObjectiveOwnersCommand struct {
	Scope           domain.Scope
	ObjectiveID     domain.ID
	ExpectedVersion domain.Version
	OwnerRefs       []domain.ActorRef
}

type SetWorkItemAssigneesCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version
	AssigneeRefs    []domain.ActorRef
}
