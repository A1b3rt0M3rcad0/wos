package domain

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

func TestWorkActivityValidatesBoundedDeclarationAndExactChronology(t *testing.T) {
	id := MustParseID("0199a555-0000-7000-8000-000000000001")
	activity := WorkActivity{ID: id, Scope: Scope{TenantID: id, OutcomeID: id}, ContractID: id, WorkItemID: id, Sequence: math.MaxUint64, Type: WorkActivityProblem, Description: "A failed check remains in history after a later correction.", PrincipalID: "executor", Actor: ActorRef{Kind: ActorKindAgent, Provider: "test", ID: "executor"}, RecordedAt: time.Now().UTC(), CriterionRefs: []WorkActivityCriterionRef{{CriterionID: id, CriterionRevision: CriterionRevision(math.MaxUint64)}}}
	if err := activity.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(activity)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"sequence":"18446744073709551615"`) || !strings.Contains(string(raw), `"criterion_revision":"18446744073709551615"`) {
		t.Fatal("activity counters lose JSON precision")
	}
	var roundtrip WorkActivity
	if err = json.Unmarshal(raw, &roundtrip); err != nil || roundtrip.Sequence != math.MaxUint64 {
		t.Fatalf("activity counter roundtrip: %v", err)
	}
	for _, mutate := range []func(*WorkActivity){func(a *WorkActivity) { a.Type = "approved" }, func(a *WorkActivity) { a.Description = " " }, func(a *WorkActivity) { a.Description = strings.Repeat("x", 16385) }, func(a *WorkActivity) { a.Sequence = 0 }, func(a *WorkActivity) { a.Actor = ActorRef{} }, func(a *WorkActivity) { a.EvidenceRefs = []ID{id, id} }, func(a *WorkActivity) { a.CriterionRefs = append(a.CriterionRefs, a.CriterionRefs[0]) }} {
		invalid := activity
		mutate(&invalid)
		if invalid.Validate() == nil {
			t.Fatal("invalid activity accepted")
		}
	}
	activity.ArtifactRefs = []ID{id}
	activity.ArtifactVersions = []SubmissionArtifact{{ArtifactID: id, SourceVersion: strings.Repeat("v", MaxWorkActivityBytes)}}
	if activity.Validate() == nil {
		t.Fatal("unreadable oversized activity reference metadata accepted")
	}
}
