package domain

import (
	"math"
	"testing"
	"time"
)

func validObjectiveCompletionAuthority() ObjectiveCompletionAuthority {
	ns := MustParseID("0199fa10-0000-7000-8000-000000000001")
	outcome := MustParseID("0199fa10-0000-7000-8000-000000000002")
	objective := MustParseID("0199fa10-0000-7000-8000-000000000003")
	credential := MustParseID("0199fa10-0000-7000-8000-000000000004")
	return ObjectiveCompletionAuthority{
		ID:    MustParseID("0199fa10-0000-7000-8000-000000000005"),
		Scope: Scope{NamespaceID: ns, OutcomeID: outcome}, ObjectiveID: objective,
		PolicyVersion: 1, Mode: ObjectiveCompletionModeAutomatic, CompletionCycle: 1,
		DelegatorPrincipalID:  "principal-1",
		DelegatorActor:        ActorRef{Kind: ActorKindHuman, Provider: "fixture", ID: "operator"},
		DelegatorCredentialID: &credential, CredentialPolicyRevision: 1,
		OperationalProfile: OperationalProfileOperationalV2, CompletionPolicy: CompletionPolicyStrictWork,
		AutomationActor: ObjectiveCompletionAutomationActor(), Reason: "Allow deterministic completion after verified work",
		AuthorizedAt: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
	}
}

func TestObjectiveCompletionAuthorityRequiresExactAutomaticDelegation(t *testing.T) {
	base := validObjectiveCompletionAuthority()
	if err := base.Validate(); err != nil {
		t.Fatalf("valid automatic authority rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*ObjectiveCompletionAuthority)
	}{
		{name: "missing credential", mutate: func(a *ObjectiveCompletionAuthority) { a.DelegatorCredentialID = nil; a.CredentialPolicyRevision = 0 }},
		{name: "missing policy revision", mutate: func(a *ObjectiveCompletionAuthority) { a.CredentialPolicyRevision = 0 }},
		{name: "compatibility profile", mutate: func(a *ObjectiveCompletionAuthority) { a.OperationalProfile = OperationalProfileCompatibilityV1 }},
		{name: "compatible completion policy", mutate: func(a *ObjectiveCompletionAuthority) { a.CompletionPolicy = CompletionPolicyCompatible }},
		{name: "forged host automation identity", mutate: func(a *ObjectiveCompletionAuthority) { a.AutomationActor.ID = "caller-controlled" }},
		{name: "zero policy version", mutate: func(a *ObjectiveCompletionAuthority) { a.PolicyVersion = 0 }},
		{name: "zero completion cycle", mutate: func(a *ObjectiveCompletionAuthority) { a.CompletionCycle = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := base
			test.mutate(&value)
			if err := value.Validate(); err == nil {
				t.Fatal("invalid automatic authority was accepted")
			}
		})
	}
}

func TestObjectiveCompletionManualAuthorityAllowsTrustedLegacyDelegation(t *testing.T) {
	value := validObjectiveCompletionAuthority()
	value.Mode = ObjectiveCompletionModeManual
	value.DelegatorCredentialID = nil
	value.CredentialPolicyRevision = 0
	value.OperationalProfile = OperationalProfileCompatibilityV1
	value.CompletionPolicy = CompletionPolicyCompatible
	if err := value.Validate(); err != nil {
		t.Fatalf("valid manual authority rejected: %v", err)
	}
}

func TestNextObjectiveCompletionPolicyVersionRejectsOverflow(t *testing.T) {
	if got, err := NextObjectiveCompletionPolicyVersion(41); err != nil || got != 42 {
		t.Fatalf("next policy version = %d, err=%v", got, err)
	}
	if _, err := NextObjectiveCompletionPolicyVersion(math.MaxUint64); err == nil {
		t.Fatal("policy version overflow was accepted")
	}
}
