package domain

import (
	"testing"
	"time"
)

func TestWorkspaceEligibilityMatchesDocumentaryAndBlockingValidation(t *testing.T) {
	ns := MustParseID("0199edc4-0000-7000-8000-000000000001")
	outcome := MustParseID("0199edc4-0000-7000-8000-000000000002")
	record := MustParseID("0199edc4-0000-7000-8000-000000000003")
	source := MustParseID("0199edc4-0000-7000-8000-000000000004")
	scope := Scope{NamespaceID: ns, OutcomeID: outcome}
	at := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	for _, kind := range []EntityKind{EntityKindOutcome, EntityKindObjective, EntityKindWorkItem, EntityKindIssue, EntityKindDecision, EntityKindBlocker, EntityKindArtifact, EntityKindEvidence, EntityKindRoadmap, EntityKindRelation, EntityKindEvidenceLink} {
		target := EntityRef{Scope: scope, Kind: kind, ID: record}
		if kind == EntityKindOutcome {
			target.ID = outcome
		}
		_, err := NewEvidenceLink(source, scope, record, target, nil, EvidenceStanceContext, "Context", at)
		if (err == nil) != EvidenceLinkTargetAllowed(kind) {
			t.Fatalf("EvidenceLink eligibility differs from mutation validation for %s: %v", kind, err)
		}
		for _, propagation := range []BlockerPropagation{BlockerPropagationDirect, BlockerPropagationSubtree} {
			cause := ExternalCause{Provider: "external", ID: "problem", Description: "Awaiting prerequisite"}
			_, err := NewBlocker(source, target, nil, &cause, "Blocked", propagation, at)
			if (err == nil) != BlockerPropagationAllowed(kind, propagation) {
				t.Fatalf("Blocker eligibility differs from mutation validation for %s/%s: %v", kind, propagation, err)
			}
		}
	}
	first := EvidenceLinkTargetKinds()
	first[0] = EntityKindBlocker
	if EvidenceLinkTargetKinds()[0] == EntityKindBlocker {
		t.Fatal("caller mutated canonical eligibility")
	}
}
