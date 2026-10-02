package application

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWave10TerminalCommandsEmitConclusionRecordedEvent(t *testing.T) {
	cases := map[string]string{
		"AchieveOutcome":                 "outcome.conclusion_recorded",
		"FailOutcome":                    "outcome.conclusion_recorded",
		"AbandonOutcome":                 "outcome.conclusion_recorded",
		"AchieveObjective":               "objective.conclusion_recorded",
		"CancelObjective":                "objective.conclusion_recorded",
		"CompleteWorkItem":               "work_item.conclusion_recorded",
		"AdministrativeCompleteWorkItem": "work_item.conclusion_recorded",
		"AdministrativeCancelWorkItem":   "work_item.conclusion_recorded",
		"CancelWorkItem":                 "work_item.conclusion_recorded",
	}
	for commandName, expected := range cases {
		eventTypes, err := eventTypesForCommand(commandMetadata{Name: commandName})
		if err != nil {
			t.Fatalf("%s: %v", commandName, err)
		}
		var found bool
		for _, eventType := range eventTypes {
			if eventType == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s events = %#v, missing %q", commandName, eventTypes, expected)
		}
	}
}

func TestWave10ConclusionRecordedPayloadCarriesPublicIdentityAndSnapshot(t *testing.T) {
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199efd0-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199efd0-0000-7000-8000-000000000010"),
	}
	owner := domain.EntityRef{Scope: scope, Kind: domain.EntityKindOutcome, ID: scope.OutcomeID}
	version := domain.Version(4)
	conclusion := domain.Conclusion{
		ID:              domain.MustParseID("0199efd0-0000-7000-8000-000000000020"),
		PrincipalID:     "tester",
		Actor:           domain.ActorRef{Kind: domain.ActorKindService, Provider: "test", ID: "validator"},
		Reason:          "verified",
		ConcludedAt:     time.Date(2026, 10, 2, 20, 30, 0, 0, time.UTC),
		OwnerRef:        &owner,
		OwnerVersion:    &version,
		LifecycleResult: string(domain.OutcomeLifecycleAchieved),
		Obligations:     domain.ConclusionObligations{},
	}
	outcome, err := domain.NewOutcome(
		scope.OutcomeID,
		scope.NamespaceID,
		"Conclusion event",
		"",
		"observable conclusion",
		domain.PriorityNormal,
		conclusion.ConcludedAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	outcome.Lifecycle = domain.OutcomeLifecycleAchieved
	outcome.Version = version
	outcome.CurrentConclusion = &conclusion

	payload, err := conclusionRecordedPayload(outcome)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Conclusion domain.Conclusion `json:"conclusion"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Conclusion.ID != conclusion.ID ||
		decoded.Conclusion.OwnerVersion == nil ||
		*decoded.Conclusion.OwnerVersion != version {
		t.Fatalf("conclusion event payload = %#v", decoded.Conclusion)
	}
}
