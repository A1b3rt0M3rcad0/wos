package sqlite

import (
	"context"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"path/filepath"
	"testing"
	"time"
)

func TestDenseAssessmentOwnerHistoryIndexPlan(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "assessment-plan.db"))
	now := time.Now().UTC()
	ids := &sqliteSequenceIDs{prefix: "0199d201", next: 1}
	service, err := a.NewService(store, sqliteFixedClock{now}, ids)
	if err != nil {
		t.Fatal(err)
	}
	cc := sqliteCommandContext("0199d201-0000-7000-8000-000000000011", "")
	out, err := service.CreateOutcome(ctx, cc, a.CreateOutcomeCommand{NamespaceID: testID("0199d201-0000-7000-8000-000000000012"), Title: "Dense owner history", DesiredState: "Indexed omission counts", Priority: d.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	u, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer u.Rollback()
	var selected d.EntityRef
	for i := 0; i < 32; i++ {
		id, err := ids.NewID()
		if err != nil {
			t.Fatal(err)
		}
		work, err := d.NewWorkItem(id, out.Value.Scope(), fmt.Sprint("History owner ", i), "", d.PriorityNormal, d.WorkItemLifecycleBacklog, now)
		if err != nil {
			t.Fatal(err)
		}
		id, err = ids.NewID()
		if err != nil {
			t.Fatal(err)
		}
		criterion, err := d.NewSuccessCriterion(id, work.Ref(), "Current attestation", "", true, d.VerificationModeAttestation)
		if err != nil {
			t.Fatal(err)
		}
		if err = work.AddCriterion(criterion, now); err != nil {
			t.Fatal(err)
		}
		for j := 0; j < 128; j++ {
			id, err = ids.NewID()
			if err != nil {
				t.Fatal(err)
			}
			assessment := d.CriterionAssessment{ID: id, CriterionID: criterion.ID, CriterionRevision: 1, Result: d.AssessmentResultMet, Rationale: "Domain-validated immutable history", PrincipalID: cc.PrincipalID, Actor: cc.Actor, AssessedAt: now.Add(time.Duration(j) * time.Microsecond)}
			if err = work.RecordCriterionAssessment(assessment, false, assessment.AssessedAt); err != nil {
				t.Fatal(err)
			}
		}
		if err = work.Activate(now); err != nil {
			t.Fatal(err)
		}
		if err = insertWorkItemFixture(ctx, u.Objectives(), u.WorkItems(), &work); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			selected = work.Ref()
		}
	}
	if err = u.Commit(); err != nil {
		t.Fatal(err)
	}
	explainAssessmentHistoryPlan(t, store, selected)
}
