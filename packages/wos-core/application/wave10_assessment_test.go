package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestRecordCriterionAssessmentUsesActiveEvidence(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	outcome := setupActiveOutcome(t, service)
	scope := outcome.Scope()
	cc := commandContext()

	current, err := service.GetOutcome(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	added, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  current.Value.Version,
		Title:            "Evidence reviewed",
		Required:         false,
		VerificationMode: domain.VerificationModeEvidenceReview,
	})
	if err != nil {
		t.Fatal(err)
	}
	criterion := added.Value

	current, err = service.GetOutcome(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RecordCriterionAssessment(ctx, cc, application.RecordCriterionAssessmentCommand{
		Owner:             outcome.Ref(),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		ExpectedVersion:   current.Value.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "missing evidence",
	}); err == nil {
		t.Fatal("evidence_review assessment without Evidence must fail")
	}

	evidenceResult, err := service.RegisterEvidence(ctx, cc, application.RegisterEvidenceCommand{
		Scope:        scope,
		EvidenceType: domain.EvidenceTypeSource,
		Description:  "review source",
		SourceRef:    domain.SourceReference{Provider: "test", ID: "wave10-source"},
		CapturedAt:   time.Date(2026, 10, 2, 20, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	recorded, err := service.RecordCriterionAssessment(ctx, cc, application.RecordCriterionAssessmentCommand{
		Owner:             outcome.Ref(),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		ExpectedVersion:   current.Value.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "reviewed active Evidence",
		EvidenceIDs:       []domain.ID{evidenceResult.Value.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(recorded.Value.EvidenceIDs) != 1 || recorded.Value.EvidenceIDs[0] != evidenceResult.Value.ID {
		t.Fatalf("assessment evidence_ids = %#v", recorded.Value.EvidenceIDs)
	}

	retracted, err := service.RegisterEvidence(ctx, cc, application.RegisterEvidenceCommand{
		Scope:        scope,
		EvidenceType: domain.EvidenceTypeSource,
		Description:  "retracted source",
		SourceRef:    domain.SourceReference{Provider: "test", ID: "wave10-retracted"},
		CapturedAt:   time.Date(2026, 10, 2, 20, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RetractEvidence(ctx, cc, application.RetractEvidenceCommand{
		Scope:           scope,
		EvidenceID:      retracted.Value.ID,
		ExpectedVersion: retracted.Value.Version,
		Reason:          "invalid source",
	}); err != nil {
		t.Fatal(err)
	}
	current, err = service.GetOutcome(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RecordCriterionAssessment(ctx, cc, application.RecordCriterionAssessmentCommand{
		Owner:             outcome.Ref(),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		ExpectedVersion:   current.Value.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "must reject retracted Evidence",
		EvidenceIDs:       []domain.ID{retracted.Value.ID},
	}); err == nil {
		t.Fatal("retracted Evidence unexpectedly supported a new assessment")
	}
}

func TestRecordCriterionAssessmentWaiverRequiresPermission(t *testing.T) {
	ctx := context.Background()
	service, store := newWave09Service(t)
	outcome := setupActiveOutcome(t, service)
	scope := outcome.Scope()
	cc := commandContext()

	current, err := service.GetOutcome(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	added, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  current.Value.Version,
		Title:            "Waivable attestation",
		Required:         false,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	criterion := added.Value
	current, err = service.GetOutcome(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	cmd := application.RecordCriterionAssessmentCommand{
		Owner:             outcome.Ref(),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		ExpectedVersion:   current.Value.Version,
		Result:            domain.AssessmentResultWaived,
		Rationale:         "approved exception",
	}
	if _, err := service.RecordCriterionAssessment(ctx, cc, cmd); err == nil {
		t.Fatal("default authorizer unexpectedly allowed waiver")
	} else if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodeForbidden {
		t.Fatalf("waiver error = %v, want forbidden", err)
	}

	authorized, err := application.NewServiceWithAuthorizer(
		store,
		fixedClock{now: time.Date(2026, 10, 2, 20, 0, 0, 0, time.UTC)},
		&wave09IDs{next: 500},
		permissionAuthorizer{allowed: map[ports.Permission]bool{
			ports.PermissionAssessmentWaive: true,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authorized.RecordCriterionAssessment(ctx, cc, cmd); err != nil {
		t.Fatal(err)
	}
}
