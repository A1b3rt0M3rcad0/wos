package application_test

import (
	"context"
	"reflect"
	"testing"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func operationalV2TaskForAcceptance(t *testing.T) credentialLeaseFixture {
	t.Helper()
	f := newTaskReviewFixture(t)
	setTaskAcceptanceProfile(t, f, d.OperationalProfileOperationalV2)

	owner, err := f.service.GetObjective(context.Background(), f.scope, *f.work.ObjectiveID)
	if err != nil {
		t.Fatalf("GetObjective() = %v", err)
	}
	_, err = f.service.AddCriterion(context.Background(), freshCommandContext(f.cc, "e33-owner-criterion"), a.AddCriterionCommand{
		Owner: owner.Value.Ref(), ExpectedVersion: owner.Value.Version,
		Title: "The parent Objective is verifiable", Required: true, VerificationMode: d.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatalf("AddCriterion(Objective) = %v", err)
	}
	owner, err = f.service.GetObjective(context.Background(), f.scope, owner.Value.ID)
	if err != nil {
		t.Fatalf("GetObjective() after AddCriterion = %v", err)
	}
	if _, err := f.service.StartObjective(context.Background(), freshCommandContext(f.cc, "e33-start-objective"), a.StartObjectiveCommand{
		Scope: f.scope, ObjectiveID: owner.Value.ID, ExpectedVersion: owner.Value.Version,
	}); err != nil {
		t.Fatalf("StartObjective() = %v", err)
	}
	addAndActivateReviewCriterion(t, &f, d.VerificationModeAttestation, "e33-task-criterion")
	return f
}

func setTaskAcceptanceProfile(t *testing.T, f credentialLeaseFixture, profile d.OperationalProfile) {
	t.Helper()
	ctx := context.Background()
	uow, err := f.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := uow.Outcomes().Get(ctx, f.scope.NamespaceID, f.scope.OutcomeID)
	if err != nil {
		_ = uow.Rollback()
		t.Fatal(err)
	}
	version := outcome.Version
	outcome.OperationalProfile = profile
	if profile == d.OperationalProfileOperationalV2 {
		outcome.CompletionPolicy = d.CompletionPolicyStrictWork
	}
	outcome.Version++
	if err := uow.Outcomes().Save(ctx, outcome, version); err != nil {
		_ = uow.Rollback()
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
}

func claimTaskForAcceptance(t *testing.T, f *credentialLeaseFixture, suffix string) ports.Credential {
	t.Helper()
	_, session := f.addCredentialTree(t,
		"0199f2a0-0000-7000-8000-000000000011", "0199f2a0-0000-7000-8000-000000000012",
		"e33-executor-root-"+suffix, "e33-executor-session-"+suffix, f.cc.PrincipalID, f.cc.Actor)
	current, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil {
		t.Fatalf("GetWorkItem() before claim = %v", err)
	}
	claimed, err := f.service.ClaimWorkItem(f.identityContext(session), freshCommandContext(f.cc, "e33-claim-"+suffix), a.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: current.Value.Version, TTL: d.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("ClaimWorkItem() = %v", err)
	}
	f.work = claimed.Value
	return session
}

func enableTaskContracts(t *testing.T, f *credentialLeaseFixture) {
	t.Helper()
	ctx := context.Background()
	uow, err := f.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	item, err := uow.WorkItems().Get(ctx, f.scope, f.work.ID)
	if err != nil {
		_ = uow.Rollback()
		t.Fatal(err)
	}
	version := item.Version
	item.ContractsEnabled = true
	item.Version++
	if err := uow.WorkItems().Save(ctx, item, version); err != nil {
		_ = uow.Rollback()
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
	f.work = item
}

func assertRejectedCompletionDidNotMutateTask(t *testing.T, f credentialLeaseFixture, before d.WorkItem, events []d.DomainEvent, receipts int) {
	t.Helper()
	after, err := f.service.GetWorkItem(context.Background(), f.scope, before.ID)
	if err != nil {
		t.Fatalf("GetWorkItem() after rejected completion = %v", err)
	}
	if !reflect.DeepEqual(after.Value, before) {
		t.Fatalf("rejected completion changed Task\nbefore: %#v\nafter:  %#v", before, after.Value)
	}
	if got := f.store.SnapshotDomainEvents(f.scope); !reflect.DeepEqual(got, events) {
		t.Fatalf("rejected completion wrote domain events: before %d, after %d", len(events), len(got))
	}
	if got := f.store.IdempotencyRecordCount(); got != receipts {
		t.Fatalf("rejected completion wrote a receipt: before %d, after %d", receipts, got)
	}
}

func TestOperationalV2DirectAndAdministrativeCompletionRequireTaskDeliveryAcceptance(t *testing.T) {
	for _, mode := range []string{"direct", "administrative"} {
		t.Run(mode, func(t *testing.T) {
			f := operationalV2TaskForAcceptance(t)
			executor := claimTaskForAcceptance(t, &f, mode)
			before := f.work
			events := f.store.SnapshotDomainEvents(f.scope)
			receipts := f.store.IdempotencyRecordCount()
			commandContext := freshCommandContext(f.cc, "e33-reject-"+mode)
			var err error
			if mode == "direct" {
				_, err = f.service.CompleteWorkItem(f.identityContext(executor), commandContext, a.CompleteWorkItemCommand{
					Scope: f.scope, WorkItemID: before.ID, ExpectedVersion: before.Version,
					ClaimID: before.CurrentLease.ClaimID, FencingToken: before.CurrentLease.FencingToken,
					ResultSummary: "Attempt to bypass independent review", Reason: "direct completion must not accept v2 work",
				})
			} else {
				admin, serviceErr := a.NewServiceWithAuthorizer(f.store, fixedClock{now: f.now}, &sequenceIDs{next: 9900}, permissionAuthorizer{
					allowed: map[ports.Permission]bool{ports.PermissionWorkAdminComplete: true},
				})
				if serviceErr != nil {
					t.Fatal(serviceErr)
				}
				_, err = admin.AdministrativeCompleteWorkItem(f.identityContext(executor), commandContext, a.AdministrativeCompleteWorkItemCommand{
					Scope: f.scope, WorkItemID: before.ID, ExpectedVersion: before.Version,
					ResultSummary: "Attempt to bypass independent review", Reason: "admin authority does not replace review",
				})
			}
			if code, ok := d.ErrorCodeOf(err); !ok || code != d.ErrorCodePreconditionFailed {
				t.Fatalf("completion error = %v, want independent-review precondition", err)
			}
			assertRejectedCompletionDidNotMutateTask(t, f, before, events, receipts)
		})
	}
}

func TestOperationalV2FinalizeWorkContractCannotBypassTaskDeliveryAcceptance(t *testing.T) {
	f := operationalV2TaskForAcceptance(t)
	enableTaskContracts(t, &f)

	commandContext := freshCommandContext(f.cc, "e33-contract-acquire")
	acquired, err := f.service.AcquireWorkContract(context.Background(), commandContext, a.AcquireWorkContractCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: f.work.Version,
	})
	if err != nil {
		t.Fatalf("AcquireWorkContract() = %v", err)
	}
	contract := acquired.Value.Contract
	authority := a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SpecDigest}
	submitted, err := f.service.SubmitWorkResult(context.Background(), freshCommandContext(f.cc, "e33-contract-submit"), a.SubmitWorkResultCommand{
		Scope: f.scope, ContractID: contract.ID, Authority: authority, ExpectedContractVersion: contract.Version,
		Material: d.WorkResultMaterial{ContractID: contract.ID, WorkItemID: f.work.ID, SpecDigest: contract.SpecDigest, Summary: "A result submitted without independent acceptance"},
	})
	if err != nil {
		t.Fatalf("SubmitWorkResult() = %v", err)
	}
	before := submitted.Value.WorkItem
	contractBefore, err := f.service.GetWorkContract(context.Background(), f.scope, contract.ID)
	if err != nil {
		t.Fatalf("GetWorkContract() before finalize = %v", err)
	}
	events := f.store.SnapshotDomainEvents(f.scope)
	receipts := f.store.IdempotencyRecordCount()
	_, err = f.service.FinalizeWorkContract(context.Background(), freshCommandContext(f.cc, "e33-reject-finalize"), a.FinalizeWorkContractCommand{
		Scope: f.scope, ContractID: contract.ID, Authority: authority,
		ExpectedContractVersion: submitted.Value.Contract.Version,
		ExpectedWorkItemVersion: before.Version, SubmissionID: submitted.Value.Submission.ID,
		Reason: "direct contract finalization must not bypass the review decision",
	})
	if code, ok := d.ErrorCodeOf(err); !ok || code != d.ErrorCodePreconditionFailed {
		t.Fatalf("FinalizeWorkContract() error = %v, want independent-review precondition", err)
	}
	assertRejectedCompletionDidNotMutateTask(t, f, before, events, receipts)
	contractAfter, err := f.service.GetWorkContract(context.Background(), f.scope, contract.ID)
	if err != nil {
		t.Fatalf("GetWorkContract() after rejected finalize = %v", err)
	}
	if !reflect.DeepEqual(contractAfter.Value, contractBefore.Value) {
		t.Fatalf("rejected finalize changed contract\nbefore: %#v\nafter:  %#v", contractBefore.Value, contractAfter.Value)
	}
}

func TestCompatibilityDirectCompletionRemainsAvailable(t *testing.T) {
	f, executor, criterion := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	assessment := recordUnboundTaskCriterionAssessment(t, f, executor, f.cc, criterion, f.work.Version, "e33-compat-assessment")
	work, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil {
		t.Fatal(err)
	}
	completed, err := f.service.CompleteWorkItem(f.identityContext(executor), freshCommandContext(f.cc, "e33-compat-direct-complete"), a.CompleteWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: assessment.WorkVersion,
		ClaimID: work.Value.CurrentLease.ClaimID, FencingToken: work.Value.CurrentLease.FencingToken,
		ResultSummary: "Legacy result with its current criterion assessment", Reason: "Compatibility behavior remains supported",
	})
	if err != nil {
		t.Fatalf("compatibility CompleteWorkItem() = %v", err)
	}
	if completed.Value.Lifecycle != d.WorkItemLifecycleDone {
		t.Fatalf("compatibility completion lifecycle = %s", completed.Value.Lifecycle)
	}
}
