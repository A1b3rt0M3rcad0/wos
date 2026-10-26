package domain_test

import (
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestNewWorkItemForObjectiveRequiresValidOwner(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199e300-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199e300-0000-7000-8000-000000000010"),
	}
	workID := domain.MustParseID("0199e300-0000-7000-8000-000000000020")

	if _, err := domain.NewWorkItemForObjective(workID, scope, nil, "Implement login", "", domain.PriorityNormal, domain.WorkItemLifecycleBacklog, now); err == nil {
		t.Fatal("NewWorkItemForObjective accepted a nil Objective")
	}
	invalid := domain.ID("")
	if _, err := domain.NewWorkItemForObjective(workID, scope, &invalid, "Implement login", "", domain.PriorityNormal, domain.WorkItemLifecycleBacklog, now); err == nil {
		t.Fatal("NewWorkItemForObjective accepted an invalid Objective ID")
	}

	objectiveID := domain.MustParseID("0199e300-0000-7000-8000-000000000030")
	work, err := domain.NewWorkItemForObjective(workID, scope, &objectiveID, "Implement login", "", domain.PriorityNormal, domain.WorkItemLifecycleBacklog, now)
	if err != nil {
		t.Fatal(err)
	}
	if work.ObjectiveID == nil || *work.ObjectiveID != objectiveID {
		t.Fatalf("ObjectiveID = %v, want %s", work.ObjectiveID, objectiveID)
	}
	if err := work.ValidateForCreation(); err != nil {
		t.Fatalf("ValidateForCreation() = %v", err)
	}

	legacy, err := domain.NewWorkItem(workID, scope, "Legacy Task", "", domain.PriorityNormal, domain.WorkItemLifecycleBacklog, now)
	if err != nil {
		t.Fatalf("legacy constructor should remain usable during expand: %v", err)
	}
	if err := legacy.Validate(); err != nil {
		t.Fatalf("legacy state validation rejected historical ownerless Task: %v", err)
	}
	if err := legacy.ValidateForCreation(); err == nil {
		t.Fatal("ValidateForCreation accepted a legacy Task without an Objective")
	}
}
