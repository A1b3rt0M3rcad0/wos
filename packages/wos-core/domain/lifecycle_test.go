package domain_test

import (
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

var (
	t0 = time.Date(2026, 10, 1, 21, 0, 0, 0, time.UTC)

	nsID       = domain.MustParseID("0199e100-0000-7000-8000-000000000001")
	outID      = domain.MustParseID("0199e100-0000-7000-8000-000000000010")
	objID      = domain.MustParseID("0199e100-0000-7000-8000-000000000020")
	workID     = domain.MustParseID("0199e100-0000-7000-8000-000000000030")
	criterion1 = domain.MustParseID("0199e100-0000-7000-8000-000000000040")
	criterion2 = domain.MustParseID("0199e100-0000-7000-8000-000000000041")
	assessment = domain.MustParseID("0199e100-0000-7000-8000-000000000050")
	claimID    = domain.MustParseID("0199e100-0000-7000-8000-000000000060")
)

func actor() domain.ActorRef {
	return domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "local", ID: "human-1"}
}

func conclusion(reason string, at time.Time) domain.Conclusion {
	return domain.Conclusion{
		PrincipalID: "human-1",
		Actor:       actor(),
		Reason:      reason,
		ConcludedAt: at,
	}
}

func TestOutcomeRequiresCriterionAndAssessmentBeforeAchievement(t *testing.T) {
	outcome, err := domain.NewOutcome(outID, nsID, "Ship capability", "", "Capability is available", domain.PriorityHigh, t0)
	if err != nil {
		t.Fatalf("NewOutcome() error = %v", err)
	}

	if err := outcome.Activate(t0.Add(time.Minute)); err == nil {
		t.Fatal("Activate() unexpectedly succeeded without required criterion")
	}

	c, err := domain.NewSuccessCriterion(
		criterion1,
		outcome.Ref(),
		"Capability verified",
		"Human confirms capability is usable",
		true,
		domain.VerificationModeAttestation,
	)
	if err != nil {
		t.Fatalf("NewSuccessCriterion() error = %v", err)
	}
	if err := outcome.AddCriterion(c, t0.Add(time.Minute)); err != nil {
		t.Fatalf("AddCriterion() error = %v", err)
	}
	if err := outcome.Activate(t0.Add(2 * time.Minute)); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if err := outcome.Achieve(conclusion("done", t0.Add(3*time.Minute)), t0.Add(3*time.Minute)); err == nil {
		t.Fatal("Achieve() unexpectedly succeeded without assessment")
	}

	a := domain.CriterionAssessment{
		ID:                assessment,
		CriterionID:       criterion1,
		CriterionRevision: domain.InitialCriterionRevision,
		Result:            domain.AssessmentResultMet,
		Rationale:         "verified manually",
		PrincipalID:       "human-1",
		Actor:             actor(),
		AssessedAt:        t0.Add(3 * time.Minute),
	}
	if err := outcome.AssessCriterionAttestation(a, t0.Add(3*time.Minute)); err != nil {
		t.Fatalf("AssessCriterionAttestation() error = %v", err)
	}
	if err := outcome.Achieve(conclusion("all required criteria met", t0.Add(4*time.Minute)), t0.Add(4*time.Minute)); err != nil {
		t.Fatalf("Achieve() error = %v", err)
	}
	if outcome.Lifecycle != domain.OutcomeLifecycleAchieved {
		t.Fatalf("Lifecycle = %q, want achieved", outcome.Lifecycle)
	}
	if outcome.CurrentConclusion == nil || len(outcome.CurrentConclusion.Assessments) != 1 {
		t.Fatalf("achievement did not preserve assessment references: %#v", outcome.CurrentConclusion)
	}
}

func TestOutcomeArchivePreservesLifecycle(t *testing.T) {
	outcome, err := domain.NewOutcome(outID, nsID, "Archive test", "", "State", domain.PriorityNormal, t0)
	if err != nil {
		t.Fatal(err)
	}
	before := outcome.Lifecycle
	if err := outcome.Archive(t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if outcome.Lifecycle != before {
		t.Fatalf("Archive changed lifecycle from %q to %q", before, outcome.Lifecycle)
	}
	if !outcome.IsArchived() {
		t.Fatal("Outcome should be archived")
	}
}

func TestObjectiveCancelAndReopenPreservesConclusionHistory(t *testing.T) {
	scope := domain.Scope{NamespaceID: nsID, OutcomeID: outID}
	obj, err := domain.NewObjective(objID, scope, "Investigate", "", domain.PriorityNormal, false, t0)
	if err != nil {
		t.Fatal(err)
	}
	if err := obj.Cancel(conclusion("no longer needed", t0.Add(time.Minute)), t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if obj.Lifecycle != domain.ObjectiveLifecycleCancelled {
		t.Fatalf("Lifecycle = %q, want cancelled", obj.Lifecycle)
	}
	if err := obj.Reopen("required again", t0.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if obj.Lifecycle != domain.ObjectiveLifecyclePlanned {
		t.Fatalf("Lifecycle = %q, want planned", obj.Lifecycle)
	}
	if obj.CurrentConclusion != nil || len(obj.ConclusionHistory) != 1 {
		t.Fatalf("reopen did not preserve prior conclusion history")
	}
}

func TestCriterionRevisionInvalidatesCurrentAssessment(t *testing.T) {
	outcome, err := domain.NewOutcome(outID, nsID, "Revision test", "", "State", domain.PriorityNormal, t0)
	if err != nil {
		t.Fatal(err)
	}
	c, err := domain.NewSuccessCriterion(criterion1, outcome.Ref(), "Initial", "", true, domain.VerificationModeAttestation)
	if err != nil {
		t.Fatal(err)
	}
	if err := outcome.AddCriterion(c, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	a := domain.CriterionAssessment{
		ID: assessment, CriterionID: criterion1, CriterionRevision: 1,
		Result: domain.AssessmentResultMet, Rationale: "ok", PrincipalID: "human-1",
		Actor: actor(), AssessedAt: t0.Add(2 * time.Minute),
	}
	if err := outcome.AssessCriterionAttestation(a, t0.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, ok := outcome.Criteria.CurrentAssessments[criterion1]; !ok {
		t.Fatal("assessment should be current")
	}
	if err := outcome.Criteria.Revise(criterion1, "Revised", "stronger condition", true, domain.VerificationModeAttestation); err != nil {
		t.Fatal(err)
	}
	if _, ok := outcome.Criteria.CurrentAssessments[criterion1]; ok {
		t.Fatal("criterion revision must invalidate current assessment")
	}
}

func TestWorkItemClaimAndCompleteRequiresCurrentLease(t *testing.T) {
	scope := domain.Scope{NamespaceID: nsID, OutcomeID: outID}
	work, err := domain.NewWorkItem(workID, scope, "Do work", "", domain.PriorityNormal, domain.WorkItemLifecycleTodo, t0)
	if err != nil {
		t.Fatal(err)
	}
	if err := work.Claim(claimID, "human-1", actor(), domain.DefaultLeaseTTL, t0.Add(time.Minute)); err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if work.CurrentLease == nil || work.CurrentLease.FencingToken != 1 {
		t.Fatalf("unexpected lease %#v", work.CurrentLease)
	}

	wrongClaim := domain.MustParseID("0199e100-0000-7000-8000-000000000061")
	if err := work.Complete("human-1", wrongClaim, 1, "done", conclusion("done", t0.Add(2*time.Minute)), t0.Add(2*time.Minute)); err == nil {
		t.Fatal("Complete() unexpectedly accepted wrong claim id")
	}
	if err := work.Complete("human-1", claimID, 1, "done", conclusion("done", t0.Add(2*time.Minute)), t0.Add(2*time.Minute)); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if work.Lifecycle != domain.WorkItemLifecycleDone {
		t.Fatalf("Lifecycle = %q, want done", work.Lifecycle)
	}
	if work.CurrentLease != nil {
		t.Fatal("completed work item must clear lease")
	}
}

func TestWorkItemWithoutCriteriaRequiresResultSummary(t *testing.T) {
	scope := domain.Scope{NamespaceID: nsID, OutcomeID: outID}
	work, err := domain.NewWorkItem(workID, scope, "Do work", "", domain.PriorityNormal, domain.WorkItemLifecycleTodo, t0)
	if err != nil {
		t.Fatal(err)
	}
	if err := work.Claim(claimID, "human-1", actor(), domain.DefaultLeaseTTL, t0); err != nil {
		t.Fatal(err)
	}
	if err := work.Complete("human-1", claimID, 1, "", conclusion("done", t0.Add(time.Minute)), t0.Add(time.Minute)); err == nil {
		t.Fatal("Complete() unexpectedly accepted empty result without criteria")
	}
}

func TestAttestationCannotTargetNonAttestationCriterion(t *testing.T) {
	outcome, err := domain.NewOutcome(outID, nsID, "Review", "", "State", domain.PriorityNormal, t0)
	if err != nil {
		t.Fatal(err)
	}
	c, err := domain.NewSuccessCriterion(criterion2, outcome.Ref(), "External check", "", true, domain.VerificationModeExternalEvaluation)
	if err != nil {
		t.Fatal(err)
	}
	if err := outcome.AddCriterion(c, t0); err != nil {
		t.Fatal(err)
	}
	a := domain.CriterionAssessment{
		ID: assessment, CriterionID: criterion2, CriterionRevision: 1,
		Result: domain.AssessmentResultMet, Rationale: "manual", PrincipalID: "human-1",
		Actor: actor(), AssessedAt: t0,
	}
	if err := outcome.AssessCriterionAttestation(a, t0); err == nil {
		t.Fatal("attestation unexpectedly accepted external_evaluation criterion")
	}
}
