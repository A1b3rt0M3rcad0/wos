package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewIssueForObjectiveSetsOwnerWithoutChangingAffectedRefs(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	scope := Scope{
		NamespaceID: MustParseID("0199f100-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199f100-0000-7000-8000-000000000002"),
	}
	objectiveID := MustParseID("0199f100-0000-7000-8000-000000000003")
	affectedTask := EntityRef{
		Scope: scope,
		Kind:  EntityKindWorkItem,
		ID:    MustParseID("0199f100-0000-7000-8000-000000000004"),
	}
	issue, err := NewIssueForObjective(
		MustParseID("0199f100-0000-7000-8000-000000000005"),
		scope,
		&objectiveID,
		"Payment service is unavailable",
		"Observed during the integration test",
		IssueSeverityMajor,
		[]EntityRef{affectedTask},
		ActorRef{Kind: ActorKindHuman, Provider: "local", ID: "reporter"},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if issue.ObjectiveID == nil || *issue.ObjectiveID != objectiveID {
		t.Fatalf("Issue ObjectiveID = %v, want %s", issue.ObjectiveID, objectiveID)
	}
	if len(issue.AffectedRefs) != 1 || issue.AffectedRefs[0] != affectedTask {
		t.Fatalf("Issue affected refs = %#v, want only the affected Task", issue.AffectedRefs)
	}
	if err := issue.ValidateForCreation(); err != nil {
		t.Fatalf("new owned Issue did not validate for creation: %v", err)
	}

	encoded, err := json.Marshal(issue)
	if err != nil {
		t.Fatal(err)
	}
	var serialized map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &serialized); err != nil {
		t.Fatal(err)
	}
	var serializedOwner ID
	if err := json.Unmarshal(serialized["objective_id"], &serializedOwner); err != nil {
		t.Fatalf("serialized objective_id is missing or invalid: %s (%v)", serialized["objective_id"], err)
	}
	if serializedOwner != objectiveID {
		t.Fatalf("serialized objective_id = %s, want %s", serializedOwner, objectiveID)
	}
}

func TestNewIssueForObjectiveRejectsMissingOrInvalidOwner(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	scope := Scope{
		NamespaceID: MustParseID("0199f101-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199f101-0000-7000-8000-000000000002"),
	}
	create := func(owner *ID) error {
		_, err := NewIssueForObjective(
			MustParseID("0199f101-0000-7000-8000-000000000004"),
			scope,
			owner,
			"Observed failure",
			"",
			IssueSeverityMinor,
			nil,
			ActorRef{Kind: ActorKindHuman, Provider: "local", ID: "reporter"},
			now,
		)
		return err
	}
	if err := create(nil); err == nil || !strings.Contains(err.Error(), "requires an Objective") {
		t.Fatalf("missing Objective error = %v", err)
	}
	invalid := ID("not-a-uuid")
	if err := create(&invalid); err == nil || !strings.Contains(err.Error(), "Objective ID is invalid") {
		t.Fatalf("invalid Objective error = %v", err)
	}
}

func TestIssueLegacyOwnerlessRecordStillLoadsDuringExpand(t *testing.T) {
	legacyJSON := `{"id":"0199f102-0000-7000-8000-000000000003","scope":{"namespace_id":"0199f102-0000-7000-8000-000000000001","outcome_id":"0199f102-0000-7000-8000-000000000002"},"version":1,"title":"Legacy report","severity":"minor","lifecycle":"open","reported_by":{"kind":"human","provider":"local","id":"reporter"},"created_at":"2026-10-20T12:00:00Z","updated_at":"2026-10-20T12:00:00Z"}`
	var issue Issue
	if err := json.Unmarshal([]byte(legacyJSON), &issue); err != nil {
		t.Fatal(err)
	}
	if issue.ObjectiveID != nil {
		t.Fatalf("legacy Issue unexpectedly has ObjectiveID %v", issue.ObjectiveID)
	}
	if err := issue.Validate(); err != nil {
		t.Fatalf("legacy Issue should remain readable during expand: %v", err)
	}
	if err := issue.ValidateForCreation(); err == nil {
		t.Fatal("ownerless legacy record unexpectedly passed the new-creation validator")
	}
}

func TestIssueRejectsMalformedPersistedObjectiveID(t *testing.T) {
	issue, err := NewIssue(
		MustParseID("0199f103-0000-7000-8000-000000000003"),
		Scope{
			NamespaceID: MustParseID("0199f103-0000-7000-8000-000000000001"),
			OutcomeID:   MustParseID("0199f103-0000-7000-8000-000000000002"),
		},
		"Invalid owner",
		"",
		IssueSeverityMinor,
		nil,
		ActorRef{Kind: ActorKindHuman, Provider: "local", ID: "reporter"},
		time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	invalid := ID("not-a-uuid")
	issue.ObjectiveID = &invalid
	if err := issue.Validate(); err == nil || !strings.Contains(err.Error(), "Objective ID is invalid") {
		t.Fatalf("invalid persisted owner error = %v", err)
	}
}
