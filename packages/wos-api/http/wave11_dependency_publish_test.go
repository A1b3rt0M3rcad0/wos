package httptransport

import (
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWave11RoadmapDependencyChangeRequestMapping(t *testing.T) {
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199f321-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199f321-0000-7000-8000-000000000010"),
	}
	expected := uint64(3)
	changes, err := roadmapDependencyChanges(scope, []roadmapDependencyChangeRequest{
		{
			Action:    "add",
			SourceRef: relationEndpointRequest{Kind: domain.EntityKindWorkItem, ID: "0199f321-0000-7000-8000-000000000020"},
			TargetRef: relationEndpointRequest{Kind: domain.EntityKindWorkItem, ID: "0199f321-0000-7000-8000-000000000021"},
			Strength:  domain.DependencyStrengthHard,
		},
		{
			Action:          "remove",
			RelationID:      "0199f321-0000-7000-8000-000000000030",
			ExpectedVersion: &expected,
			Reason:          "replanned",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("change count = %d, want 2", len(changes))
	}
	if changes[0].Action != application.RoadmapDependencyChangeAdd ||
		changes[0].SourceRef.Scope != scope ||
		changes[0].TargetRef.Scope != scope {
		t.Fatalf("mapped add change = %#v", changes[0])
	}
	if changes[1].Action != application.RoadmapDependencyChangeRemove ||
		changes[1].ExpectedVersion != 3 {
		t.Fatalf("mapped remove change = %#v", changes[1])
	}
}
