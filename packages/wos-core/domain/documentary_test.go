package domain

import (
	"testing"
	"time"
)

func TestDocumentaryRecordsPreserveHistoricalContent(t *testing.T) {
	scope := Scope{NamespaceID: MustParseID("0199e100-0000-7000-8000-000000000001"), OutcomeID: MustParseID("0199e100-0000-7000-8000-000000000010")}
	actor := ActorRef{Kind: ActorKindService, Provider: "test", ID: "producer"}
	now := time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC)

	artifact, err := NewArtifact(MustParseID("0199e100-0000-7000-8000-000000000020"), scope, "document", "report", "file:///report.pdf", "application/pdf", "sha256:abc", "v1", actor, nil, now)
	if err != nil { t.Fatal(err) }
	uri := artifact.URI
	if err := artifact.Withdraw("obsolete", now.Add(time.Minute)); err != nil { t.Fatal(err) }
	if artifact.URI != uri || artifact.Lifecycle != ArtifactLifecycleWithdrawn || artifact.Version != 2 {
		t.Fatalf("withdrawal rewrote artifact content: %+v", artifact)
	}

	evidence, err := NewEvidence(MustParseID("0199e100-0000-7000-8000-000000000021"), scope, EvidenceTypeSource, "benchmark report", SourceReference{Provider:"test", URI:"file:///report.pdf"}, actor, now, &artifact.ID, nil, "v1", "sha256:abc", now)
	if err != nil { t.Fatal(err) }
	description := evidence.Description
	if err := evidence.Retract("superseded measurement", now.Add(2*time.Minute)); err != nil { t.Fatal(err) }
	if evidence.Description != description || evidence.Lifecycle != EvidenceLifecycleRetracted || evidence.Version != 2 {
		t.Fatalf("retraction rewrote evidence content: %+v", evidence)
	}
}

func TestEvidenceLinkStanceBelongsToLink(t *testing.T) {
	scope := Scope{NamespaceID: MustParseID("0199e100-0000-7000-8000-000000000001"), OutcomeID: MustParseID("0199e100-0000-7000-8000-000000000010")}
	evidenceID := MustParseID("0199e100-0000-7000-8000-000000000021")
	now := time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC)
	supports, err := NewEvidenceLink(MustParseID("0199e100-0000-7000-8000-000000000030"), scope, evidenceID, EntityRef{Scope:scope, Kind:EntityKindObjective, ID:MustParseID("0199e100-0000-7000-8000-000000000040")}, nil, EvidenceStanceSupports, "supports objective", now)
	if err != nil { t.Fatal(err) }
	contradicts, err := NewEvidenceLink(MustParseID("0199e100-0000-7000-8000-000000000031"), scope, evidenceID, EntityRef{Scope:scope, Kind:EntityKindDecision, ID:MustParseID("0199e100-0000-7000-8000-000000000041")}, nil, EvidenceStanceContradicts, "contradicts decision", now)
	if err != nil { t.Fatal(err) }
	if supports.Stance == contradicts.Stance { t.Fatal("stance must belong to each link") }
}

func TestAcceptedDecisionIsImmutableAndRejectionRequiresReason(t *testing.T) {
	scope := Scope{NamespaceID: MustParseID("0199e100-0000-7000-8000-000000000001"), OutcomeID: MustParseID("0199e100-0000-7000-8000-000000000010")}
	actor := ActorRef{Kind: ActorKindHuman, Provider: "test", ID: "u1"}
	now := time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC)
	d, err := NewDecision(MustParseID("0199e100-0000-7000-8000-000000000050"), scope, "Storage", "Use SQLite", []string{"SQLite","PostgreSQL"}, "SQLite", "local-first", actor, now)
	if err != nil { t.Fatal(err) }
	if err := d.Accept(actor, now.Add(time.Minute)); err != nil { t.Fatal(err) }
	title := "changed"
	if _, err := d.Update(&title, nil, nil, nil, nil, now.Add(2*time.Minute)); err == nil {
		t.Fatal("accepted decision must not be editable")
	}

	rejected, err := NewDecision(MustParseID("0199e100-0000-7000-8000-000000000051"), scope, "Other", "Try other", nil, "", "", actor, now)
	if err != nil { t.Fatal(err) }
	if err := rejected.Reject(" ", actor, now); err == nil { t.Fatal("rejection without reason must fail") }
}
