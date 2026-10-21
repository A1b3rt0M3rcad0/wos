package domain_test

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type legacyOutcomeJSON struct {
	CompletionPolicy  domain.CompletionPolicy `json:"completion_policy,omitempty"`
	ExternalContext   domain.ExternalContext  `json:"external_context,omitempty"`
	ID                domain.ID               `json:"id"`
	NamespaceID       domain.ID               `json:"namespace_id"`
	Version           domain.Version          `json:"version"`
	Title             string                  `json:"title"`
	Description       string                  `json:"description,omitempty"`
	DesiredState      string                  `json:"desired_state"`
	Lifecycle         domain.OutcomeLifecycle `json:"lifecycle"`
	Priority          domain.Priority         `json:"priority"`
	OwnerRefs         []domain.ActorRef       `json:"owner_refs,omitempty"`
	ArchivedAt        *time.Time              `json:"archived_at,omitempty"`
	Criteria          domain.CriterionSet     `json:"criteria"`
	CurrentConclusion *domain.Conclusion      `json:"conclusion,omitempty"`
	ConclusionHistory []domain.Conclusion     `json:"conclusion_history,omitempty"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
}

type legacyObjectiveJSON struct {
	ID                 domain.ID                 `json:"id"`
	Scope              domain.Scope              `json:"scope"`
	Version            domain.Version            `json:"version"`
	Title              string                    `json:"title"`
	Description        string                    `json:"description,omitempty"`
	ParentObjectiveID  *domain.ID                `json:"parent_objective_id,omitempty"`
	Lifecycle          domain.ObjectiveLifecycle `json:"lifecycle"`
	Priority           domain.Priority           `json:"priority"`
	OwnerRefs          []domain.ActorRef         `json:"owner_refs,omitempty"`
	RequiredForOutcome bool                      `json:"required_for_outcome"`
	Criteria           domain.CriterionSet       `json:"criteria"`
	CurrentConclusion  *domain.Conclusion        `json:"conclusion,omitempty"`
	ConclusionHistory  []domain.Conclusion       `json:"conclusion_history,omitempty"`
	CreatedAt          time.Time                 `json:"created_at"`
	UpdatedAt          time.Time                 `json:"updated_at"`
}

func TestLegacyAggregateJSONDefaultsPolicyFieldsWithoutAddingZeroValues(t *testing.T) {
	outcome, err := domain.NewOutcome(outID, nsID, "Legacy", "", "Preserve old behavior", domain.PriorityNormal, t0)
	if err != nil {
		t.Fatal(err)
	}
	if got := outcome.OperationalProfile.Effective(); got != domain.OperationalProfileCompatibilityV1 {
		t.Fatalf("effective Outcome profile = %q, want compatibility_v1", got)
	}

	objective, err := domain.NewObjective(objID, outcome.Scope(), "Manual by default", "", domain.PriorityNormal, false, t0)
	if err != nil {
		t.Fatal(err)
	}
	if got := objective.CompletionMode.Effective(); got != domain.ObjectiveCompletionModeManual {
		t.Fatalf("effective Objective completion mode = %q, want manual", got)
	}
	if got := objective.EffectiveCompletionCycle(); got != 1 {
		t.Fatalf("effective Objective completion cycle = %d, want 1", got)
	}

	for name, value := range map[string]any{"Outcome": outcome, "Objective": objective} {
		payload, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(payload, &fields); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		for _, key := range []string{"operational_profile", "completion_mode", "completion_cycle"} {
			if _, exists := fields[key]; exists {
				t.Errorf("legacy %s JSON unexpectedly contains zero-valued %q: %s", name, key, payload)
			}
		}
	}
	legacyOutcomeBytes, err := json.Marshal(legacyOutcomeJSON{
		CompletionPolicy: outcome.CompletionPolicy, ExternalContext: outcome.ExternalContext,
		ID: outcome.ID, NamespaceID: outcome.NamespaceID, Version: outcome.Version,
		Title: outcome.Title, Description: outcome.Description, DesiredState: outcome.DesiredState,
		Lifecycle: outcome.Lifecycle, Priority: outcome.Priority, OwnerRefs: outcome.OwnerRefs,
		ArchivedAt: outcome.ArchivedAt, Criteria: outcome.Criteria, CurrentConclusion: outcome.CurrentConclusion,
		ConclusionHistory: outcome.ConclusionHistory, CreatedAt: outcome.CreatedAt, UpdatedAt: outcome.UpdatedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	currentOutcome, err := json.Marshal(outcome)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(currentOutcome, legacyOutcomeBytes) {
		t.Fatalf("zero-valued new Outcome policy fields changed legacy JSON bytes:\ncurrent: %s\nlegacy:  %s", currentOutcome, legacyOutcomeBytes)
	}
	legacyObjectiveBytes, err := json.Marshal(legacyObjectiveJSON{
		ID: objective.ID, Scope: objective.Scope, Version: objective.Version, Title: objective.Title,
		Description: objective.Description, ParentObjectiveID: objective.ParentObjectiveID,
		Lifecycle: objective.Lifecycle, Priority: objective.Priority, OwnerRefs: objective.OwnerRefs,
		RequiredForOutcome: objective.RequiredForOutcome, Criteria: objective.Criteria,
		CurrentConclusion: objective.CurrentConclusion, ConclusionHistory: objective.ConclusionHistory,
		CreatedAt: objective.CreatedAt, UpdatedAt: objective.UpdatedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	currentObjective, err := json.Marshal(objective)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(currentObjective, legacyObjectiveBytes) {
		t.Fatalf("zero-valued new Objective policy fields changed legacy JSON bytes:\ncurrent: %s\nlegacy:  %s", currentObjective, legacyObjectiveBytes)
	}

	var legacyOutcome domain.Outcome
	outcomeJSON, _ := json.Marshal(outcome)
	if err := json.Unmarshal(outcomeJSON, &legacyOutcome); err != nil {
		t.Fatal(err)
	}
	if legacyOutcome.OperationalProfile.Effective() != domain.OperationalProfileCompatibilityV1 {
		t.Fatal("omitted Outcome profile did not retain compatibility behavior")
	}
	var legacyObjective domain.Objective
	objectiveJSON, _ := json.Marshal(objective)
	if err := json.Unmarshal(objectiveJSON, &legacyObjective); err != nil {
		t.Fatal(err)
	}
	if legacyObjective.CompletionMode.Effective() != domain.ObjectiveCompletionModeManual || legacyObjective.EffectiveCompletionCycle() != 1 {
		t.Fatal("omitted Objective policy fields did not retain manual cycle-one behavior")
	}
}

func TestOperationalPolicyRejectsUnknownValuesAndIncompatibleAutomaticMode(t *testing.T) {
	outcome, err := domain.NewOutcome(outID, nsID, "Policy", "", "Policy is explicit", domain.PriorityNormal, t0)
	if err != nil {
		t.Fatal(err)
	}
	objective, err := domain.NewObjective(objID, outcome.Scope(), "Completion", "", domain.PriorityNormal, false, t0)
	if err != nil {
		t.Fatal(err)
	}

	invalidOutcome := outcome
	invalidOutcome.OperationalProfile = "future_profile"
	if err := invalidOutcome.Validate(); err == nil {
		t.Fatal("unknown operational profile unexpectedly validated")
	}
	invalidObjective := objective
	invalidObjective.CompletionMode = "automatic_if_possible"
	if err := invalidObjective.Validate(); err == nil {
		t.Fatal("unknown Objective completion mode unexpectedly validated")
	}
	invalidCompletionScope := outcome
	invalidCompletionScope.CompletionPolicy = "ignore_work"
	if err := objective.ValidateCompletionConfiguration(invalidCompletionScope); err == nil {
		t.Fatal("unknown Outcome completion policy unexpectedly validated")
	}

	if err := objective.ValidateCompletionConfiguration(outcome); err != nil {
		t.Fatalf("legacy manual configuration should remain valid: %v", err)
	}
	automatic := objective
	automatic.CompletionMode = domain.ObjectiveCompletionModeAutomatic
	if err := automatic.ValidateCompletionConfiguration(outcome); err == nil {
		t.Fatal("automatic mode on the compatibility profile unexpectedly validated")
	}

	v2 := outcome
	v2.OperationalProfile = domain.OperationalProfileOperationalV2
	if err := automatic.ValidateCompletionConfiguration(v2); err == nil {
		t.Fatal("automatic mode with compatible scope unexpectedly validated")
	}
	v2.CompletionPolicy = domain.CompletionPolicyStrictWork
	if err := automatic.ValidateCompletionConfiguration(v2); err != nil {
		t.Fatalf("v2 automatic strict-work configuration should validate: %v", err)
	}
	if err := objective.ValidateCompletionConfiguration(v2); err != nil {
		t.Fatalf("v2 manual configuration should validate: %v", err)
	}

	other, err := domain.NewOutcome(domain.MustParseID("0199e100-0000-7000-8000-000000000099"), nsID, "Other", "", "Different scope", domain.PriorityNormal, t0)
	if err != nil {
		t.Fatal(err)
	}
	if err := objective.ValidateCompletionConfiguration(other); err == nil {
		t.Fatal("completion configuration accepted a non-owning Outcome")
	}
}

func TestOperationalPolicyValuesRoundTrip(t *testing.T) {
	outcome, err := domain.NewOutcome(outID, nsID, "V2 Outcome", "", "Run the v2 policy", domain.PriorityNormal, t0)
	if err != nil {
		t.Fatal(err)
	}
	outcome.OperationalProfile = domain.OperationalProfileOperationalV2
	outcome.CompletionPolicy = domain.CompletionPolicyStrictWork
	outcomeJSON, err := json.Marshal(outcome)
	if err != nil {
		t.Fatal(err)
	}
	var decodedOutcome domain.Outcome
	if err := json.Unmarshal(outcomeJSON, &decodedOutcome); err != nil {
		t.Fatal(err)
	}
	if decodedOutcome.OperationalProfile != domain.OperationalProfileOperationalV2 || decodedOutcome.CompletionPolicy != domain.CompletionPolicyStrictWork {
		t.Fatalf("Outcome policy round trip = %q/%q", decodedOutcome.OperationalProfile, decodedOutcome.CompletionPolicy)
	}

	objective, err := domain.NewObjective(objID, outcome.Scope(), "Delegated completion", "", domain.PriorityNormal, false, t0)
	if err != nil {
		t.Fatal(err)
	}
	objective.CompletionMode = domain.ObjectiveCompletionModeAutomatic
	objective.CompletionCycle = 9007199254740993
	objectiveJSON, err := json.Marshal(objective)
	if err != nil {
		t.Fatal(err)
	}
	var decodedObjective domain.Objective
	if err := json.Unmarshal(objectiveJSON, &decodedObjective); err != nil {
		t.Fatal(err)
	}
	if decodedObjective.CompletionMode != domain.ObjectiveCompletionModeAutomatic || decodedObjective.CompletionCycle != objective.CompletionCycle {
		t.Fatalf("Objective policy round trip = %q/cycle %d", decodedObjective.CompletionMode, decodedObjective.CompletionCycle)
	}
}

func TestObjectiveCompletionCycleRoundTripsUint64AndAdvancesOnReopen(t *testing.T) {
	objective, err := domain.NewObjective(objID, domain.Scope{NamespaceID: nsID, OutcomeID: outID}, "Cycle", "", domain.PriorityNormal, false, t0)
	if err != nil {
		t.Fatal(err)
	}
	objective.CompletionCycle = math.MaxUint64
	payload, err := json.Marshal(objective)
	if err != nil {
		t.Fatal(err)
	}
	var decoded domain.Objective
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.CompletionCycle != math.MaxUint64 {
		t.Fatalf("round-trip completion_cycle = %d, want %d", decoded.CompletionCycle, uint64(math.MaxUint64))
	}

	objective.CompletionCycle = 0
	criterion, err := domain.NewSuccessCriterion(criterion1, objective.Ref(), "Current assessment", "", true, domain.VerificationModeAttestation)
	if err != nil {
		t.Fatal(err)
	}
	if err := objective.AddCriterion(criterion, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	assessmentRecord := domain.CriterionAssessment{
		ID: assessment, CriterionID: criterion.ID, CriterionRevision: criterion.Revision,
		Result: domain.AssessmentResultMet, Rationale: "The existing criterion was assessed.",
		PrincipalID: "human-1", Actor: actor(), AssessedAt: t0.Add(2 * time.Minute),
	}
	if err := objective.AssessCriterionAttestation(assessmentRecord, t0.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := objective.Achieve(conclusion("First cycle concluded", t0.Add(3*time.Minute)), t0.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := objective.Reopen("Re-evaluate in a new completion cycle", t0.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if objective.CompletionCycle != 2 || objective.EffectiveCompletionCycle() != 2 {
		t.Fatalf("completion cycle after first reopen = %d (effective %d), want 2", objective.CompletionCycle, objective.EffectiveCompletionCycle())
	}
	if objective.Lifecycle != domain.ObjectiveLifecyclePlanned || objective.CurrentConclusion != nil || len(objective.ConclusionHistory) != 1 {
		t.Fatalf("reopen changed completion semantics unexpectedly: %#v", objective)
	}
}

func TestObjectiveReopenRejectsCompletionCycleOverflowWithoutMutation(t *testing.T) {
	objective, err := domain.NewObjective(objID, domain.Scope{NamespaceID: nsID, OutcomeID: outID}, "Exhausted cycle", "", domain.PriorityNormal, false, t0)
	if err != nil {
		t.Fatal(err)
	}
	objective.Lifecycle = domain.ObjectiveLifecycleAchieved
	objective.CompletionCycle = math.MaxUint64
	version := objective.Version
	if err := objective.Reopen("Cannot advance exhausted counter", t0.Add(time.Minute)); err == nil {
		t.Fatal("reopen unexpectedly succeeded at uint64 completion cycle limit")
	}
	if objective.Lifecycle != domain.ObjectiveLifecycleAchieved || objective.Version != version || objective.CompletionCycle != math.MaxUint64 {
		t.Fatal("failed reopen mutated the Objective")
	}
}
