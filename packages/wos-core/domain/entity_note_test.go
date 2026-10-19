package domain

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

func TestEntityNoteIsBoundedDocumentaryAndCorrectionRetainsProvenance(t *testing.T) {
	id := MustParseID("019a3100-0000-7000-8000-000000000001")
	other := MustParseID("019a3100-0000-7000-8000-000000000002")
	scope := Scope{NamespaceID: id, OutcomeID: id}
	n := EntityNote{ID: id, Scope: scope, SubjectRef: EntityRef{Scope: scope, Kind: EntityKindOutcome, ID: id}, Type: EntityNoteObservation, Body: "I completed the external checks; this declaration does not accept the Outcome.", PrincipalID: "human-1", Actor: ActorRef{Kind: ActorKindHuman, Provider: "local", ID: "author"}, RecordedAt: time.Now().UTC(), OutcomeRevision: OutcomeRevision(math.MaxUint64)}
	if err := n.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"outcome_revision":"18446744073709551615"`) {
		t.Fatal("note revision lost exact precision")
	}
	correction := n
	correction.ID = other
	correction.SupersedesNoteID = &id
	correction.Body = "Correction: one external check failed."
	if err := correction.ValidateCorrection(n); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*EntityNote){"other_author": func(n *EntityNote) { n.PrincipalID = "human-2" }, "other_subject": func(n *EntityNote) { n.SubjectRef.Kind = EntityKindObjective; n.SubjectRef.ID = other }, "other_criterion": func(n *EntityNote) { n.CriterionID = &other }} {
		t.Run(name, func(t *testing.T) {
			invalid := correction
			mutate(&invalid)
			if invalid.ValidateCorrection(n) == nil {
				t.Fatal("correction rewrote provenance")
			}
		})
	}
	for name, mutate := range map[string]func(*EntityNote){"transition_type": func(n *EntityNote) { n.Type = "completed" }, "empty": func(n *EntityNote) { n.Body = " " }, "too_long": func(n *EntityNote) { n.Body = strings.Repeat("x", MaxEntityNoteBodyBytes+1) }, "invalid_utf8": func(n *EntityNote) { n.Body = string([]byte{0xff}) }, "nul": func(n *EntityNote) { n.Body = "prefix\x00suffix" }, "missing_author": func(n *EntityNote) { n.PrincipalID = "" }, "zero_revision": func(n *EntityNote) { n.OutcomeRevision = 0 }, "duplicate_ref": func(n *EntityNote) { n.EvidenceRefs = []ID{id, id} }, "too_many_refs": func(n *EntityNote) { n.ArtifactRefs = make([]ID, 51) }, "wrong_scope": func(n *EntityNote) { n.SubjectRef.OutcomeID = other }} {
		t.Run(name, func(t *testing.T) {
			invalid := n
			mutate(&invalid)
			if invalid.Validate() == nil {
				t.Fatal("invalid documentary note accepted")
			}
		})
	}
}
