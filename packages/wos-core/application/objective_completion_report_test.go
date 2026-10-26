package application

import (
	"fmt"
	"strings"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestObjectiveCompletionReportOptionalObjectiveAndMissingCriterion(t *testing.T) {
	f := newIssueCompletionFixture(t)
	optional := f.objective("Optional outcome objective", nil, true)
	got, err := f.s.GetObjectiveCompletionReport(f.ctx, f.scope, optional.ID, 3)
	if err != nil {
		t.Fatalf("GetObjectiveCompletionReport(optional) error = %v", err)
	}
	if !got.Value.FactualEligible || !got.Value.FactsComplete || got.Value.RequiredForOutcome {
		t.Fatalf("optional empty Objective report = %+v", got.Value)
	}
	if got.Value.Tasks.Total != 0 || got.Value.Tasks.Open.Total != 0 || got.Value.Authorization.Status != "not_evaluated" {
		t.Fatalf("empty Task/authentication facts = %+v / %+v", got.Value.Tasks, got.Value.Authorization)
	}
	if got.OutcomeRevision != got.Value.OutcomeRevision || got.Value.CompletionCycle != 1 {
		t.Fatalf("report revision/cycle = %d/%d", got.OutcomeRevision, got.Value.CompletionCycle)
	}

	missing := f.objective("Objective with no current assessment", nil, false)
	missingReport, err := f.s.GetObjectiveCompletionReport(f.ctx, f.scope, missing.ID, 3)
	if err != nil {
		t.Fatalf("GetObjectiveCompletionReport(missing criterion) error = %v", err)
	}
	if missingReport.Value.FactualEligible || !missingReport.Value.FactsComplete || missingReport.Value.Criteria.ActiveRequired != 1 || missingReport.Value.Criteria.Pending != 1 {
		t.Fatalf("missing criterion was not represented as a complete but ineligible fact: %+v", missingReport.Value)
	}
	if !objectiveCompletionHasReason(missingReport.Value, "criterion_assessment_missing") && !objectiveCompletionHasReason(missingReport.Value, "required_criterion_pending") {
		t.Fatalf("missing criterion reason absent: %v", missingReport.Value.EligibilityReasonCodes)
	}
}

func TestObjectiveCompletionReportDoesNotSubstituteTaskCriteriaForObjectiveCriteria(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Objective criterion is separate", nil, false)
	taskResult, err := f.s.CreateWorkItem(f.ctx, f.cc(), CreateWorkItemCommand{
		Scope: f.scope, ObjectiveID: &owner.ID, Title: "Task-local proof", Priority: d.PriorityNormal,
		InitialCriteria: []InitialCriterion{{Title: "Task result is checked", Required: true, VerificationMode: d.VerificationModeAttestation}},
	})
	if err != nil {
		t.Fatalf("CreateWorkItem() error = %v", err)
	}
	criterion := taskResult.Value.Criteria.Items[0]
	if _, err := f.s.RecordCriterionAssessment(f.ctx, f.cc(), RecordCriterionAssessmentCommand{
		Owner: taskResult.Value.Ref(), ExpectedVersion: taskResult.Value.Version,
		CriterionID: criterion.ID, CriterionRevision: criterion.Revision,
		Result: d.AssessmentResultMet, Rationale: "The Task result was checked",
	}); err != nil {
		t.Fatalf("RecordCriterionAssessment(Task) error = %v", err)
	}

	got, err := f.s.GetObjectiveCompletionReport(f.ctx, f.scope, owner.ID, 3)
	if err != nil {
		t.Fatalf("GetObjectiveCompletionReport() error = %v", err)
	}
	if got.Value.Criteria.ActiveRequired != 1 || got.Value.Tasks.IncompleteCriteria != 0 || got.Value.FactualEligible {
		t.Fatalf("Task criteria were confused with Objective obligations: %+v", got.Value)
	}
	if !objectiveCompletionHasReason(got.Value, "criterion_assessment_missing") && !objectiveCompletionHasReason(got.Value, "required_criterion_pending") {
		t.Fatalf("Objective criterion obligation was hidden by Task proof: %v", got.Value.EligibilityReasonCodes)
	}
}

func TestObjectiveCompletionReportScansNestedWorkBeyondOnePage(t *testing.T) {
	f := newIssueCompletionFixture(t)
	root := f.objective("Outcome branch", nil, true)
	child := f.objective("Nested branch", &root.ID, true)
	setCompletionPolicyForReportTest(t, f, d.CompletionPolicyStrictWork)
	const count = 107
	seedBacklogTasksForReportTest(t, f, child, count)

	got, err := f.s.GetObjectiveCompletionReport(f.ctx, f.scope, root.ID, 4)
	if err != nil {
		t.Fatalf("GetObjectiveCompletionReport(large subtree) error = %v", err)
	}
	report := got.Value
	if !report.FactsComplete || report.FactualEligible {
		t.Fatalf("open strict-work subtree facts = complete:%v eligible:%v reasons:%v", report.FactsComplete, report.FactualEligible, report.EligibilityReasonCodes)
	}
	if report.DescendantObjectives.Total != 1 || report.DescendantObjectives.Open.Total != 1 || report.Tasks.Total != count || report.Tasks.Backlog != count || report.Tasks.Open.Total != count {
		t.Fatalf("nested or large Task totals were truncated: descendants=%+v tasks=%+v", report.DescendantObjectives, report.Tasks)
	}
	if report.StrictWork.Open.Total != count+1 || len(report.StrictWork.Open.Samples) != 4 {
		t.Fatalf("strict subtree obligations = %+v, want %d open and 4 samples", report.StrictWork, count+1)
	}
	for i := 1; i < len(report.Tasks.Open.Samples); i++ {
		if strings.Compare(report.Tasks.Open.Samples[i-1].ID.String(), report.Tasks.Open.Samples[i].ID.String()) >= 0 {
			t.Fatalf("Task samples are not stable ID order: %+v", report.Tasks.Open.Samples)
		}
	}
}

func TestObjectiveCompletionReportIssueDependencyAndBlockerFacts(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Operational facts", nil, true)
	target := f.objective("Hard prerequisite", nil, true)
	if _, err := f.s.AddDependency(f.ctx, f.cc(), AddDependencyCommand{
		Scope: f.scope, SourceRef: owner.Ref(), TargetRef: target.Ref(), Strength: d.DependencyStrengthHard,
		Reason: "The Objective depends on this prerequisite",
	}); err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}
	if _, err := f.s.CreateBlocker(f.ctx, f.cc(), CreateBlockerCommand{
		Scope: f.scope, BlockedRef: owner.Ref(), Description: "External approval is still pending",
		Propagation: d.BlockerPropagationDirect,
	}); err != nil {
		t.Fatalf("CreateBlocker() error = %v", err)
	}
	f.issue(owner.ID)

	got, err := f.s.GetObjectiveCompletionReport(f.ctx, f.scope, owner.ID, 3)
	if err != nil {
		t.Fatalf("GetObjectiveCompletionReport(blocked Objective) error = %v", err)
	}
	report := got.Value
	if !report.FactsComplete || report.FactualEligible {
		t.Fatalf("pending business obligations should be complete facts but ineligible: %+v", report)
	}
	if !report.PendingIssues.Complete || report.PendingIssues.Pending.Total != 1 || len(report.PendingIssues.Pending.Samples) != 1 {
		t.Fatalf("pending Issue scan was conflated with its expected gate failure: %+v", report.PendingIssues)
	}
	if report.HardDependencies.Total != 1 || report.HardDependencies.Satisfied != 0 || report.HardDependencies.Unsatisfied.Total != 1 {
		t.Fatalf("hard dependency facts = %+v", report.HardDependencies)
	}
	if report.ActiveBlockers.Total != 1 || report.ActiveBlockers.Blockers.Total != 1 {
		t.Fatalf("blocker facts = %+v", report.ActiveBlockers)
	}
	for _, expected := range []string{"objective_issue_pending", "hard_dependency_unsatisfied", "active_blocker"} {
		if !objectiveCompletionHasReason(report, expected) {
			t.Errorf("reason %q missing from %v", expected, report.EligibilityReasonCodes)
		}
	}
}

func TestObjectiveCompletionReportCancelledWorkAndWaiver(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Cancelled with waiver", nil, false)
	owner = recordObjectiveWaiverForReportTest(t, f, owner)
	taskResult, err := f.s.CreateWorkItem(f.ctx, f.cc(), CreateWorkItemCommand{
		Scope: f.scope, ObjectiveID: &owner.ID, Title: "Explicitly excluded Task", Priority: d.PriorityNormal,
	})
	if err != nil {
		t.Fatalf("CreateWorkItem() error = %v", err)
	}
	setCompletionPolicyForReportTest(t, f, d.CompletionPolicyStrictWork)
	if _, err := f.s.CancelWorkItem(f.ctx, f.cc(), CancelWorkItemCommand{
		Scope: f.scope, WorkItemID: taskResult.Value.ID, ExpectedVersion: taskResult.Value.Version, Reason: "No longer required",
	}); err != nil {
		t.Fatalf("CancelWorkItem() error = %v", err)
	}

	got, err := f.s.GetObjectiveCompletionReport(f.ctx, f.scope, owner.ID, 3)
	if err != nil {
		t.Fatalf("GetObjectiveCompletionReport(cancelled work) error = %v", err)
	}
	if got.Value.Criteria.Waived != 1 || got.Value.Criteria.Pending != 0 {
		t.Fatalf("waived criterion facts = %+v", got.Value.Criteria)
	}
	if got.Value.StrictWork.Cancelled.Total != 1 || !got.Value.StrictWork.AcknowledgementRequired || got.Value.FactualEligible {
		t.Fatalf("cancelled work did not require explicit acknowledgement: %+v", got.Value.StrictWork)
	}
	if !objectiveCompletionHasReason(got.Value, "cancelled_work_requires_ack") {
		t.Fatalf("cancelled acknowledgement reason absent: %v", got.Value.EligibilityReasonCodes)
	}
}

func TestObjectiveCompletionReportFailsClosedForPriorCycleAssessment(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Reopened Objective", nil, true)
	if err := f.achieve(owner); err != nil {
		t.Fatalf("AchieveObjective() error = %v", err)
	}
	achieved, err := f.s.GetObjective(f.ctx, f.scope, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := f.s.ReopenObjective(f.ctx, f.cc(), ReopenObjectiveCommand{
		Scope: f.scope, ObjectiveID: owner.ID, ExpectedVersion: achieved.Value.Version, Reason: "Reopen for a new completion cycle",
	})
	if err != nil {
		t.Fatalf("ReopenObjective() error = %v", err)
	}
	got, err := f.s.GetObjectiveCompletionReport(f.ctx, f.scope, owner.ID, 3)
	if err != nil {
		t.Fatalf("GetObjectiveCompletionReport(reopened Objective) error = %v", err)
	}
	if reopened.Value.CompletionCycle != 2 || got.Value.CompletionCycle != 2 || got.Value.Criteria.Met != 1 || got.Value.FactsComplete || got.Value.FactualEligible {
		t.Fatalf("prior-cycle assessment was treated as current: reopened=%+v report=%+v", reopened.Value, got.Value)
	}
	if !objectiveCompletionHasReason(got.Value, "objective_assessment_cycle_unavailable") {
		t.Fatalf("cycle-unavailable reason absent: %v", got.Value.EligibilityReasonCodes)
	}
}

func TestObjectiveCompletionReportProjectionSeparatesSignedAndNativePendingReview(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Verification projections", nil, true)
	tasks := []d.WorkItem{
		{ID: d.MustParseID("019a0000-0000-7000-8000-000000000001"), Scope: f.scope, ObjectiveID: &owner.ID, Lifecycle: d.WorkItemLifecycleInProgress},
		{ID: d.MustParseID("019a0000-0000-7000-8000-000000000002"), Scope: f.scope, ObjectiveID: &owner.ID, Lifecycle: d.WorkItemLifecycleInProgress},
	}
	report := newObjectiveCompletionReport(owner, 7, time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC), 4, d.CompletionPolicyCompatible)
	addObjectiveCompletionVerification(tasks[0], TaskVerificationView{Pending: true, SourceKind: "task_delivery"}, 4, &report)
	addObjectiveCompletionVerification(tasks[1], TaskVerificationView{Pending: true, SourceKind: "signed_review_case"}, 4, &report)
	if report.Tasks.AwaitingVerification.Total != 2 || report.Tasks.PendingReview.Total != 2 {
		t.Fatalf("native/signed pending verification projection = %+v", report.Tasks)
	}
	report.finish()
	if !report.FactualEligible || len(report.EligibilityReasonCodes) != 0 {
		t.Fatalf("verification status should be informational until another current gate applies: %+v", report)
	}
}

func seedBacklogTasksForReportTest(t *testing.T, f *issueCompletionFixture, owner d.Objective, count int) {
	t.Helper()
	tx, err := f.store.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < count; i++ {
		id, err := f.ids.NewID()
		if err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
		item, err := d.NewWorkItemForObjective(id, f.scope, &owner.ID, fmt.Sprintf("Backlog Task %03d", i), "", d.PriorityNormal, d.WorkItemLifecycleBacklog, time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC))
		if err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
		if err := tx.WorkItems().Insert(f.ctx, item); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func setCompletionPolicyForReportTest(t *testing.T, f *issueCompletionFixture, policy d.CompletionPolicy) {
	t.Helper()
	tx, err := f.store.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	outcome, err := tx.Outcomes().Get(f.ctx, f.scope.NamespaceID, f.scope.OutcomeID)
	if err != nil {
		t.Fatal(err)
	}
	before := outcome.Version
	outcome.CompletionPolicy = policy
	next, err := outcome.Version.Next()
	if err != nil {
		t.Fatal(err)
	}
	outcome.Version = next
	outcome.UpdatedAt = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	if err := tx.Outcomes().Save(f.ctx, outcome, before); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Coordination().AdvanceOutcome(f.ctx, f.scope); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func recordObjectiveWaiverForReportTest(t *testing.T, f *issueCompletionFixture, owner d.Objective) d.Objective {
	t.Helper()
	criterion := owner.Criteria.Items[0]
	id, err := f.ids.NewID()
	if err != nil {
		t.Fatal(err)
	}
	assessment := d.CriterionAssessment{
		ID: id, CriterionID: criterion.ID, CriterionRevision: criterion.Revision,
		Result: d.AssessmentResultWaived, Rationale: "The agreed exception was reviewed",
		PrincipalID: "owner", Actor: d.ActorRef{Kind: d.ActorKindHuman, Provider: "test", ID: "owner"},
		AssessedAt: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
	}
	if err := owner.RecordCriterionAssessment(assessment, true, assessment.AssessedAt); err != nil {
		t.Fatal(err)
	}
	tx, err := f.store.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := tx.Objectives().Save(f.ctx, owner, owner.Version-1); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Coordination().AdvanceOutcome(f.ctx, f.scope); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return owner
}

func objectiveCompletionHasReason(report ObjectiveCompletionReport, expected string) bool {
	for _, reason := range report.EligibilityReasonCodes {
		if reason == expected {
			return true
		}
	}
	return false
}
