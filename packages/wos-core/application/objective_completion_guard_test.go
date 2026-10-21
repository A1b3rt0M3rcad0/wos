package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type objectiveCompletionCountingTransactions struct {
	delegate ports.TransactionManager
	begins   int
}

func (m *objectiveCompletionCountingTransactions) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	m.begins++
	return m.delegate.Begin(ctx)
}

func TestObjectiveCompletionReportCapabilityAndManualCommandUseOneUnitOfWork(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("One coherent completion frame", nil, true)
	manager := &objectiveCompletionCountingTransactions{delegate: f.store}
	f.s.tx = manager

	if _, err := f.s.GetObjectiveCompletionReport(f.ctx, f.scope, owner.ID, 3); err != nil {
		t.Fatalf("GetObjectiveCompletionReport() error = %v", err)
	}
	if manager.begins != 1 {
		t.Fatalf("report opened %d transactions, want exactly its one read frame", manager.begins)
	}

	manager.begins = 0
	view := workspaceIssueImpactView(f, WorkspaceSubject{Kind: "entity", Ref: owner.Ref()}, 3)
	capability := workspaceIssueAchieveCapability(t, view)
	if capability.Availability != "available" || manager.begins != 1 {
		t.Fatalf("Objective capability did not use its existing workspace frame: capability=%+v transactions=%d", capability, manager.begins)
	}

	manager.begins = 0
	if err := f.achieve(owner); err != nil {
		t.Fatalf("manual AchieveObjective() error = %v", err)
	}
	if manager.begins != 1 {
		t.Fatalf("manual achievement opened %d transactions, want its one guarded command UoW", manager.begins)
	}
}

func TestObjectiveCompletionCapabilityAndManualCommandShareFirstGate(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Blocked Objective", nil, true)
	prerequisite := f.objective("Unfinished prerequisite", nil, true)
	if _, err := f.s.CreateBlocker(f.ctx, f.cc(), CreateBlockerCommand{
		Scope: f.scope, BlockedRef: owner.Ref(), Description: "Approval is pending",
		Propagation: d.BlockerPropagationDirect,
	}); err != nil {
		t.Fatalf("CreateBlocker() error = %v", err)
	}
	if _, err := f.s.AddDependency(f.ctx, f.cc(), AddDependencyCommand{
		Scope: f.scope, SourceRef: owner.Ref(), TargetRef: prerequisite.Ref(), Strength: d.DependencyStrengthHard,
		Reason: "The prerequisite must be achieved first",
	}); err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}

	report, err := f.s.GetObjectiveCompletionReport(f.ctx, f.scope, owner.ID, 3)
	if err != nil {
		t.Fatalf("GetObjectiveCompletionReport() error = %v", err)
	}
	if !objectiveCompletionHasReason(report.Value, "active_blocker") || !objectiveCompletionHasReason(report.Value, "hard_dependency_unsatisfied") {
		t.Fatalf("report omitted factual blockers: %v", report.Value.EligibilityReasonCodes)
	}
	view := workspaceIssueImpactView(f, WorkspaceSubject{Kind: "entity", Ref: owner.Ref()}, 3)
	capability := workspaceIssueAchieveCapability(t, view)
	workspaceIssueAssertCommandParity(f, owner, capability, false)
	if !reflect.DeepEqual(capability.ReasonCodes, []string{"active_blocker"}) {
		t.Fatalf("capability did not preserve the command's first gate: %+v", capability)
	}
}

func TestObjectiveCompletionManualAcknowledgementUsesFullCancelledSet(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Complete with explicit exclusions", nil, true)
	setCompletionPolicyForReportTest(t, f, d.CompletionPolicyStrictWork)

	const cancelledCount = 24
	cancelled := make([]d.EntityRef, 0, cancelledCount)
	for i := 0; i < cancelledCount; i++ {
		created, err := f.s.CreateWorkItem(f.ctx, f.cc(), CreateWorkItemCommand{
			Scope: f.scope, ObjectiveID: &owner.ID, Title: "Excluded Task", Priority: d.PriorityNormal,
		})
		if err != nil {
			t.Fatalf("CreateWorkItem(%d) error = %v", i, err)
		}
		ended, err := f.s.CancelWorkItem(f.ctx, f.cc(), CancelWorkItemCommand{
			Scope: f.scope, WorkItemID: created.Value.ID, ExpectedVersion: created.Value.Version, Reason: "No longer required",
		})
		if err != nil {
			t.Fatalf("CancelWorkItem(%d) error = %v", i, err)
		}
		cancelled = append(cancelled, ended.Value.Ref())
	}

	report, err := f.s.GetObjectiveCompletionReport(f.ctx, f.scope, owner.ID, 3)
	if err != nil {
		t.Fatalf("GetObjectiveCompletionReport() error = %v", err)
	}
	if report.Value.StrictWork.Cancelled.Total != cancelledCount || len(report.Value.StrictWork.Cancelled.Samples) != 3 {
		t.Fatalf("public report did not keep complete total and bounded sample: %+v", report.Value.StrictWork.Cancelled)
	}
	view := workspaceIssueImpactView(f, WorkspaceSubject{Kind: "entity", Ref: owner.Ref()}, 3)
	capability := workspaceIssueAchieveCapability(t, view)
	if capability.ConfirmationKind != "cancelled_scope_exclusion" || len(capability.RelatedRefs) != cancelledCount {
		t.Fatalf("capability did not retain complete cancellation references: %+v", capability)
	}

	observed := report.OutcomeRevision
	_, err = f.s.AchieveObjective(f.ctx, f.cc(), AchieveObjectiveCommand{
		Scope: f.scope, ObjectiveID: owner.ID, ExpectedVersion: owner.Version, Reason: "Accept with reviewed exclusions",
		ObservedOutcomeRevision: &observed, AcknowledgedCancelledWork: report.Value.StrictWork.Cancelled.Samples,
	})
	if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodePreconditionFailed {
		t.Fatalf("bounded sample incorrectly acknowledged the full cancellation set: error=%v code=%s", err, code)
	}
	if reason := objectiveCompletionErrorReason(err); reason != "cancelled_work_requires_ack" {
		t.Fatalf("bounded sample failure reason = %q, want cancelled_work_requires_ack", reason)
	}

	_, err = f.s.AchieveObjective(f.ctx, f.cc(), AchieveObjectiveCommand{
		Scope: f.scope, ObjectiveID: owner.ID, ExpectedVersion: owner.Version, Reason: "Accept with reviewed exclusions",
		ObservedOutcomeRevision: &observed, AcknowledgedCancelledWork: cancelled,
	})
	if err != nil {
		t.Fatalf("full exact cancellation acknowledgement was rejected: %v", err)
	}
}

func objectiveCompletionErrorReason(err error) string {
	var typed *d.Error
	if !errors.As(err, &typed) || typed.Details == nil {
		return ""
	}
	return typed.Details.ReasonCode
}
