package application_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func setupLegacyTaskForReview(t *testing.T, mode d.VerificationMode) (credentialLeaseFixture, ports.Credential, d.SuccessCriterion) {
	t.Helper()
	f := newTaskReviewFixture(t)
	criterion := addAndActivateReviewCriterion(t, &f, mode, "review-add-task-criterion")
	_, session := f.addCredentialTree(t,
		"0199f290-0000-7000-8000-000000000001", "0199f291-0000-7000-8000-000000000001",
		"e30-executor-root", "e30-executor-session", f.cc.PrincipalID, f.cc.Actor)
	claimed, err := f.service.ClaimWorkItem(f.identityContext(session), freshCommandContext(f.cc, "review-task-claim"), a.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: d.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("ClaimWorkItem() error = %v", err)
	}
	f.work = claimed.Value
	return f, session, criterion
}

func newTaskReviewFixture(t *testing.T) credentialLeaseFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	store := memory.New()
	namespace := id("0199f2a0-0000-7000-8000-000000000001")
	if err := store.PutNamespace(ctx, ports.Namespace{ID: namespace, Name: "Task delivery review"}); err != nil {
		t.Fatal(err)
	}
	service, err := a.NewService(store, fixedClock{now: now}, &sequenceIDs{next: 9000})
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, a.CreateOutcomeCommand{
		NamespaceID: namespace, Title: "Delivery review outcome", DesiredState: "A verified Task result", Priority: d.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.AddCriterion(ctx, cc, a.AddCriterionCommand{
		Owner: created.Value.Ref(), ExpectedVersion: created.Value.Version, Title: "The result can be verified", Required: true, VerificationMode: d.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	activatedOutcome, err := service.ActivateOutcome(ctx, cc, a.ActivateOutcomeCommand{Scope: created.Value.Scope(), ExpectedVersion: created.Value.Version + 1})
	if err != nil {
		t.Fatal(err)
	}
	objectiveID := objectiveForWorkFixture(t, service, activatedOutcome.Value.Scope())
	createdTask, err := service.CreateWorkItem(ctx, cc, a.CreateWorkItemCommand{
		Scope: activatedOutcome.Value.Scope(), ObjectiveID: objectiveID, Title: "Task pending verified delivery", Priority: d.PriorityNormal, Lifecycle: d.WorkItemLifecycleBacklog,
	})
	if err != nil {
		t.Fatal(err)
	}
	return credentialLeaseFixture{t: t, store: store, service: service, now: now, scope: activatedOutcome.Value.Scope(), work: createdTask.Value, cc: cc}
}

func addAndActivateReviewCriterion(t *testing.T, f *credentialLeaseFixture, mode d.VerificationMode, key string) d.SuccessCriterion {
	t.Helper()
	added, err := f.service.AddCriterion(context.Background(), freshCommandContext(f.cc, key), a.AddCriterionCommand{
		Owner: f.work.Ref(), ExpectedVersion: f.work.Version, Title: "the delivered result satisfies its criterion", Required: true, VerificationMode: mode,
	})
	if err != nil {
		t.Fatalf("AddCriterion() error = %v", err)
	}
	current, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil {
		t.Fatalf("GetWorkItem() after AddCriterion = %v", err)
	}
	activated, err := f.service.ActivateWorkItem(context.Background(), freshCommandContext(f.cc, key+"-activate"), a.ActivateWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: current.Value.Version,
	})
	if err != nil {
		t.Fatalf("ActivateWorkItem() after AddCriterion = %v", err)
	}
	f.work = activated.Value
	return added.Value
}

func submitLegacyTaskDelivery(t *testing.T, f credentialLeaseFixture, executor ports.Credential, material d.TaskDeliveryMaterial, key string) a.TaskDeliveryResult {
	t.Helper()
	work, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil {
		t.Fatalf("GetWorkItem() before submit = %v", err)
	}
	command := a.SubmitTaskForReviewCommand{
		Scope: f.scope, WorkItemID: work.Value.ID, ExpectedWorkItemVersion: work.Value.Version,
		ClaimID: &work.Value.CurrentLease.ClaimID, FencingToken: work.Value.CurrentLease.FencingToken,
		LegacyMaterial: ptrTaskDeliveryMaterial(material),
	}
	result, err := f.service.SubmitTaskForReview(f.identityContext(executor), freshCommandContext(f.cc, key), command)
	if err != nil {
		t.Fatalf("SubmitTaskForReview() error = %v", err)
	}
	return result.Value
}

func newReviewPrincipal(t *testing.T, f credentialLeaseFixture, suffix, principal string) (ports.Credential, d.CommandContext) {
	t.Helper()
	actor := d.ActorRef{Kind: d.ActorKindHuman, Provider: "local", ID: principal}
	credential := f.addRoot(t, "0199f292-0000-7000-8000-"+fmt.Sprintf("%012x", commandOrdinal(suffix)), "e30-reviewer-"+suffix, principal, actor)
	cc := f.cc
	cc.PrincipalID = principal
	cc.Actor = actor
	return credential, cc
}

type recordedTaskAssessment struct {
	Assessment  d.CriterionAssessment
	WorkVersion d.Version
}

func recordTaskCriterionAssessment(t *testing.T, f credentialLeaseFixture, reviewer ports.Credential, cc d.CommandContext, criterion d.SuccessCriterion, delivery d.TaskDelivery, expected d.Version, submissionID *d.ID, evidenceIDs []d.ID, key string) recordedTaskAssessment {
	t.Helper()
	deliveryRef := d.TaskDeliveryAssessmentRef{ID: delivery.ID, Digest: delivery.Digest, Round: delivery.Round}
	assessment, err := f.service.RecordCriterionAssessment(f.identityContext(reviewer), freshCommandContext(cc, key), a.RecordCriterionAssessmentCommand{
		Owner: d.EntityRef{Scope: f.scope, Kind: d.EntityKindWorkItem, ID: f.work.ID}, ExpectedVersion: expected,
		CriterionID: criterion.ID, CriterionRevision: criterion.Revision, Result: d.AssessmentResultMet,
		Rationale: "Reviewed the current delivery against this criterion", SubmissionID: submissionID, TaskDeliveryRef: &deliveryRef, EvidenceIDs: evidenceIDs,
	})
	if err != nil {
		t.Fatalf("RecordCriterionAssessment() error = %v", err)
	}
	work, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil {
		t.Fatalf("GetWorkItem() after assessment = %v", err)
	}
	return recordedTaskAssessment{Assessment: assessment.Value, WorkVersion: work.Value.Version}
}

func recordUnboundTaskCriterionAssessment(t *testing.T, f credentialLeaseFixture, reviewer ports.Credential, cc d.CommandContext, criterion d.SuccessCriterion, expected d.Version, key string) recordedTaskAssessment {
	t.Helper()
	assessment, err := f.service.RecordCriterionAssessment(f.identityContext(reviewer), freshCommandContext(cc, key), a.RecordCriterionAssessmentCommand{
		Owner: d.EntityRef{Scope: f.scope, Kind: d.EntityKindWorkItem, ID: f.work.ID}, ExpectedVersion: expected,
		CriterionID: criterion.ID, CriterionRevision: criterion.Revision, Result: d.AssessmentResultMet,
		Rationale: "Reviewed the Task criterion before a delivery was pending",
	})
	if err != nil {
		t.Fatalf("RecordCriterionAssessment() before delivery error = %v", err)
	}
	work, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil {
		t.Fatalf("GetWorkItem() after pre-delivery assessment = %v", err)
	}
	return recordedTaskAssessment{Assessment: assessment.Value, WorkVersion: work.Value.Version}
}

func makeTaskDeliveryAcceptCommand(f credentialLeaseFixture, delivery d.TaskDelivery, workVersion d.Version, assessment d.CriterionAssessment, reason string) a.AcceptTaskDeliveryCommand {
	return a.AcceptTaskDeliveryCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: workVersion,
		DeliveryID: delivery.ID, ExpectedDeliveryDigest: delivery.Digest, ExpectedDeliveryRound: delivery.Round,
		Reason: reason, AssessmentRefs: []d.CriterionAssessmentRef{{
			AssessmentID: assessment.ID, CriterionID: assessment.CriterionID,
			CriterionRevision: assessment.CriterionRevision, Result: assessment.Result,
		}},
	}
}

func TestAcceptTaskDeliveryRequiresAuthenticatedReviewerAndCurrentAssessment(t *testing.T) {
	f, executor, criterion := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	deliveryResult := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "Implemented the requested result"}, "review-delivery-accept")
	reviewer, reviewerCC := newReviewPrincipal(t, f, "accept-reviewer", "reviewer-accept")
	assessment := recordTaskCriterionAssessment(t, f, reviewer, reviewerCC, criterion, deliveryResult.Delivery, deliveryResult.WorkItem.Version, nil, nil, "review-assess-accept")
	command := makeTaskDeliveryAcceptCommand(f, deliveryResult.Delivery, assessment.WorkVersion, assessment.Assessment, "The delivered work satisfies the required criterion")
	commandContext := freshCommandContext(reviewerCC, "review-accept-task")

	if _, err := f.service.AcceptTaskDelivery(context.Background(), commandContext, command); errorCode(err) != d.ErrorCodeForbidden {
		t.Fatalf("unauthenticated acceptance error = %v, want forbidden", err)
	}
	accepted, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), commandContext, command)
	if err != nil {
		t.Fatalf("AcceptTaskDelivery() error = %v", err)
	}
	if accepted.Value.Decision.Result != d.TaskDeliveryDecisionAccepted || accepted.Value.Decision.DeliveryID != deliveryResult.Delivery.ID || accepted.Value.WorkItem.Lifecycle != d.WorkItemLifecycleDone || accepted.Value.WorkItem.PendingDeliveryID != nil || accepted.Value.WorkItem.CurrentConclusion == nil {
		t.Fatalf("accepted delivery did not atomically conclude the Task: %+v", accepted.Value)
	}
	if accepted.Value.WorkItem.CurrentConclusion.PrincipalID != reviewerCC.PrincipalID || accepted.Value.WorkItem.CurrentConclusion.Reason != command.Reason {
		t.Fatalf("Task conclusion did not preserve reviewer decision provenance: %+v", accepted.Value.WorkItem.CurrentConclusion)
	}
	replayed, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), commandContext, command)
	if err != nil || !replayed.IdempotentReplay || replayed.Value.Decision.ID != accepted.Value.Decision.ID {
		t.Fatalf("acceptance replay = (replay=%t, err=%v)", replayed.IdempotentReplay, err)
	}
	duplicate := command
	duplicate.ExpectedWorkItemVersion = accepted.Value.WorkItem.Version
	duplicateContext := freshCommandContext(reviewerCC, "review-accept-duplicate")
	if _, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), duplicateContext, duplicate); err == nil {
		t.Fatal("a second decision for the same delivery was accepted")
	}
}

func TestRequestTaskDeliveryChangesReturnsTaskToTodoWithoutLease(t *testing.T) {
	f, executor, _ := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	deliveryResult := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "Needs correction"}, "review-delivery-changes")
	reviewer, reviewerCC := newReviewPrincipal(t, f, "changes-reviewer", "reviewer-changes")
	command := a.RequestTaskDeliveryChangesCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: deliveryResult.WorkItem.Version,
		DeliveryID: deliveryResult.Delivery.ID, ExpectedDeliveryDigest: deliveryResult.Delivery.Digest, ExpectedDeliveryRound: deliveryResult.Delivery.Round,
		Reason: "The submitted result needs a correction", Findings: []string{"The response is missing the requested behavior"},
	}
	result, err := f.service.RequestTaskDeliveryChanges(f.identityContext(reviewer), freshCommandContext(reviewerCC, "review-changes-request"), command)
	if err != nil {
		t.Fatalf("RequestTaskDeliveryChanges() error = %v", err)
	}
	if result.Value.Decision.Result != d.TaskDeliveryDecisionChangesRequested || result.Value.WorkItem.Lifecycle != d.WorkItemLifecycleTodo || result.Value.WorkItem.PendingDeliveryID != nil || result.Value.WorkItem.CurrentLease != nil || len(result.Value.Decision.Findings) != 1 {
		t.Fatalf("changes request did not preserve findings and return the Task without authority: %+v", result.Value)
	}
}

func TestTaskDeliveryDecisionsRequireIdempotencyKeysWithoutMutation(t *testing.T) {
	f, executor, criterion := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	deliveryResult := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "Awaiting a keyed decision"}, "review-delivery-no-decision-key")
	reviewer, reviewerCC := newReviewPrincipal(t, f, "no-decision-key-reviewer", "reviewer-no-decision-key")
	assessment := recordTaskCriterionAssessment(t, f, reviewer, reviewerCC, criterion, deliveryResult.Delivery, deliveryResult.WorkItem.Version, nil, nil, "review-assess-no-decision-key")
	accept := makeTaskDeliveryAcceptCommand(f, deliveryResult.Delivery, assessment.WorkVersion, assessment.Assessment, "Accept the delivery")
	changes := a.RequestTaskDeliveryChangesCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: assessment.WorkVersion,
		DeliveryID: deliveryResult.Delivery.ID, ExpectedDeliveryDigest: deliveryResult.Delivery.Digest, ExpectedDeliveryRound: deliveryResult.Delivery.Round,
		Reason: "Ask for a correction", Findings: []string{"Provide a clearer result"},
	}
	beforeEvents := len(f.store.SnapshotDomainEvents(f.scope))
	beforeReceipts := f.store.IdempotencyRecordCount()
	beforeRevision := readTaskDeliveryRevision(t, f)
	missingKey := freshCommandContext(reviewerCC, "review-empty-decision-key")
	missingKey.IdempotencyKey = "  \t"
	if _, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), missingKey, accept); errorCode(err) != d.ErrorCodeInvalidArgument {
		t.Fatalf("acceptance without idempotency key error = %v, want invalid argument", err)
	}
	if _, err := f.service.RequestTaskDeliveryChanges(f.identityContext(reviewer), missingKey, changes); errorCode(err) != d.ErrorCodeInvalidArgument {
		t.Fatalf("changes request without idempotency key error = %v, want invalid argument", err)
	}
	work, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil || work.Value.Version != assessment.WorkVersion || work.Value.Lifecycle != d.WorkItemLifecycleInProgress || work.Value.PendingDeliveryID == nil || *work.Value.PendingDeliveryID != deliveryResult.Delivery.ID {
		t.Fatalf("missing-key decisions mutated pending Task: value=%+v err=%v", work.Value, err)
	}
	uow, err := f.store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	decision, decisionErr := uow.(ports.TaskDeliveryUnitOfWork).TaskDeliveries().DecisionForDelivery(context.Background(), f.scope, deliveryResult.Delivery.ID)
	if rollbackErr := uow.Rollback(); rollbackErr != nil {
		t.Fatal(rollbackErr)
	}
	if decisionErr != nil || decision != nil {
		t.Fatalf("missing-key decision was persisted: decision=%+v err=%v", decision, decisionErr)
	}
	if events := f.store.SnapshotDomainEvents(f.scope); len(events) != beforeEvents {
		t.Fatalf("missing-key decisions emitted events: before=%d after=%d", beforeEvents, len(events))
	}
	if revision := readTaskDeliveryRevision(t, f); revision != beforeRevision {
		t.Fatalf("missing-key decisions advanced Outcome revision: before=%d after=%d", beforeRevision, revision)
	}
	if receipts := f.store.IdempotencyRecordCount(); receipts != beforeReceipts {
		t.Fatalf("missing-key decisions wrote receipts: before=%d after=%d", beforeReceipts, receipts)
	}
}

func TestTaskDeliveryIndependentReviewRejectsSelfReview(t *testing.T) {
	f, executor, criterion := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	deliveryResult := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "Ready for independent review"}, "review-delivery-self")
	reviewer, reviewerCC := newReviewPrincipal(t, f, "self-review-assessor", "reviewer-independent")
	assessment := recordTaskCriterionAssessment(t, f, reviewer, reviewerCC, criterion, deliveryResult.Delivery, deliveryResult.WorkItem.Version, nil, nil, "review-assess-self")
	f.service.SetIndependentReviewer(true)
	command := makeTaskDeliveryAcceptCommand(f, deliveryResult.Delivery, assessment.WorkVersion, assessment.Assessment, "accept")
	_, err := f.service.AcceptTaskDelivery(f.identityContext(executor), freshCommandContext(f.cc, "review-self-accept"), command)
	if errorCode(err) != d.ErrorCodeForbidden {
		t.Fatalf("executor self-review error = %v, want forbidden", err)
	}
	if _, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), freshCommandContext(reviewerCC, "review-independent-accept"), command); err != nil {
		t.Fatalf("independent reviewer acceptance failed: %v", err)
	}
}

func TestAcceptTaskDeliveryRejectsUnresolvedRequiredCriterion(t *testing.T) {
	f, executor, _ := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	deliveryResult := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "No assessment exists yet"}, "review-delivery-unresolved")
	reviewer, reviewerCC := newReviewPrincipal(t, f, "unresolved-reviewer", "reviewer-unresolved")
	command := a.AcceptTaskDeliveryCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: deliveryResult.WorkItem.Version,
		DeliveryID: deliveryResult.Delivery.ID, ExpectedDeliveryDigest: deliveryResult.Delivery.Digest, ExpectedDeliveryRound: deliveryResult.Delivery.Round,
		Reason: "Attempting acceptance with an unresolved criterion",
	}
	if _, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), freshCommandContext(reviewerCC, "review-unresolved-accept"), command); err == nil {
		t.Fatal("Task was accepted while a required active criterion remained unresolved")
	}
	work, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil || work.Value.Lifecycle != d.WorkItemLifecycleInProgress || work.Value.PendingDeliveryID == nil || *work.Value.PendingDeliveryID != deliveryResult.Delivery.ID {
		t.Fatalf("failed acceptance changed the pending Task: value=%+v err=%v", work.Value, err)
	}
}

func TestAcceptTaskDeliveryRejectsAssessmentRecordedBeforeDelivery(t *testing.T) {
	f, executor, criterion := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	reviewer, reviewerCC := newReviewPrincipal(t, f, "predelivery-reviewer", "reviewer-predelivery")
	stale := recordUnboundTaskCriterionAssessment(t, f, reviewer, reviewerCC, criterion, f.work.Version, "review-assess-before-delivery")
	delivery := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "The work is now delivered"}, "review-delivery-after-assessment")
	command := makeTaskDeliveryAcceptCommand(f, delivery.Delivery, delivery.WorkItem.Version, stale.Assessment, "Accept the current delivery")
	if _, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), freshCommandContext(reviewerCC, "review-reject-predelivery-assessment"), command); errorCode(err) != d.ErrorCodeSubmissionNotAccepted {
		t.Fatalf("acceptance with pre-delivery assessment error = %v, want submission not accepted", err)
	}
	work, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil || work.Value.PendingDeliveryID == nil || *work.Value.PendingDeliveryID != delivery.Delivery.ID || work.Value.Lifecycle != d.WorkItemLifecycleInProgress {
		t.Fatalf("rejected stale assessment changed the pending delivery: work=%+v err=%v", work.Value, err)
	}
}

func TestAcceptTaskDeliveryRejectsAssessmentFromPriorCorrectionRound(t *testing.T) {
	f, executor, criterion := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	first := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "First delivery"}, "review-round-one-delivery")
	reviewer, reviewerCC := newReviewPrincipal(t, f, "round-reviewer", "reviewer-rounds")
	firstAssessment := recordTaskCriterionAssessment(t, f, reviewer, reviewerCC, criterion, first.Delivery, first.WorkItem.Version, nil, nil, "review-round-one-assessment")
	changed, err := f.service.RequestTaskDeliveryChanges(f.identityContext(reviewer), freshCommandContext(reviewerCC, "review-round-one-changes"), a.RequestTaskDeliveryChangesCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: firstAssessment.WorkVersion,
		DeliveryID: first.Delivery.ID, ExpectedDeliveryDigest: first.Delivery.Digest, ExpectedDeliveryRound: first.Delivery.Round,
		Reason: "The first result needs a correction", Findings: []string{"Revise the delivered result"},
	})
	if err != nil {
		t.Fatalf("RequestTaskDeliveryChanges() = %v", err)
	}
	claimed, err := f.service.ClaimWorkItem(f.identityContext(executor), freshCommandContext(f.cc, "review-round-two-claim"), a.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: changed.Value.WorkItem.Version, TTL: d.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("ClaimWorkItem() for correction round = %v", err)
	}
	f.work = claimed.Value
	second := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "Corrected delivery"}, "review-round-two-delivery")
	staleCommand := makeTaskDeliveryAcceptCommand(f, second.Delivery, second.WorkItem.Version, firstAssessment.Assessment, "Accept the corrected delivery")
	if _, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), freshCommandContext(reviewerCC, "review-reject-prior-round-assessment"), staleCommand); errorCode(err) != d.ErrorCodeSubmissionNotAccepted {
		t.Fatalf("acceptance with prior-round assessment error = %v, want submission not accepted", err)
	}
	currentAssessment := recordTaskCriterionAssessment(t, f, reviewer, reviewerCC, criterion, second.Delivery, second.WorkItem.Version, nil, nil, "review-round-two-assessment")
	accept := makeTaskDeliveryAcceptCommand(f, second.Delivery, currentAssessment.WorkVersion, currentAssessment.Assessment, "Accept the corrected result")
	accepted, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), freshCommandContext(reviewerCC, "review-accept-round-two"), accept)
	if err != nil {
		t.Fatalf("acceptance with current correction-round assessment = %v", err)
	}
	if accepted.Value.WorkItem.Lifecycle != d.WorkItemLifecycleDone || accepted.Value.Decision.Round != second.Delivery.Round {
		t.Fatalf("accepted result does not match correction round: %+v", accepted.Value)
	}
}

func TestAcceptTaskDeliveryRejectsEvidenceFromPreviousDeliveryRound(t *testing.T) {
	f, executor, criterion := setupLegacyTaskForReview(t, d.VerificationModeEvidenceReview)
	oldEvidence, err := f.service.RegisterEvidence(context.Background(), freshCommandContext(f.cc, "review-old-evidence"), a.RegisterEvidenceCommand{
		Scope: f.scope, EvidenceType: d.EvidenceTypeInspection, Description: "Proof from first delivery",
		SourceRef: d.SourceReference{Provider: "test", URI: "https://example.test/first-delivery"}, CapturedAt: f.now,
	})
	if err != nil {
		t.Fatal(err)
	}
	newEvidence, err := f.service.RegisterEvidence(context.Background(), freshCommandContext(f.cc, "review-new-evidence"), a.RegisterEvidenceCommand{
		Scope: f.scope, EvidenceType: d.EvidenceTypeInspection, Description: "Proof from correction delivery",
		SourceRef: d.SourceReference{Provider: "test", URI: "https://example.test/correction-delivery"}, CapturedAt: f.now,
	})
	if err != nil {
		t.Fatal(err)
	}
	firstMaterial := d.TaskDeliveryMaterial{Summary: "First delivery", EvidenceIDs: []d.ID{oldEvidence.Value.ID}, CriterionEvidence: []d.SubmissionCriterionEvidence{{CriterionID: criterion.ID, CriterionRevision: criterion.Revision, EvidenceIDs: []d.ID{oldEvidence.Value.ID}}}}
	first := submitLegacyTaskDelivery(t, f, executor, firstMaterial, "review-first-evidence-delivery")
	reviewer, reviewerCC := newReviewPrincipal(t, f, "previous-round-reviewer", "reviewer-previous-round")
	changed, err := f.service.RequestTaskDeliveryChanges(f.identityContext(reviewer), freshCommandContext(reviewerCC, "review-first-evidence-changes"), a.RequestTaskDeliveryChangesCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: first.WorkItem.Version,
		DeliveryID: first.Delivery.ID, ExpectedDeliveryDigest: first.Delivery.Digest, ExpectedDeliveryRound: first.Delivery.Round,
		Reason: "The proof needs to be refreshed", Findings: []string{"Attach proof from the corrected result"},
	})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := f.service.ClaimWorkItem(f.identityContext(executor), freshCommandContext(f.cc, "review-second-round-claim"), a.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: changed.Value.WorkItem.Version, TTL: d.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("reacquire after changes request = %v", err)
	}
	f.work = claimed.Value
	secondMaterial := d.TaskDeliveryMaterial{Summary: "Corrected delivery", EvidenceIDs: []d.ID{newEvidence.Value.ID}, CriterionEvidence: []d.SubmissionCriterionEvidence{{CriterionID: criterion.ID, CriterionRevision: criterion.Revision, EvidenceIDs: []d.ID{newEvidence.Value.ID}}}}
	second := submitLegacyTaskDelivery(t, f, executor, secondMaterial, "review-second-evidence-delivery")
	assessment := recordTaskCriterionAssessment(t, f, reviewer, reviewerCC, criterion, second.Delivery, second.WorkItem.Version, nil, []d.ID{oldEvidence.Value.ID}, "review-assess-old-evidence")
	command := makeTaskDeliveryAcceptCommand(f, second.Delivery, assessment.WorkVersion, assessment.Assessment, "Accept corrected result")
	if _, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), freshCommandContext(reviewerCC, "review-reject-old-evidence"), command); errorCode(err) != d.ErrorCodeSubmissionNotAccepted {
		t.Fatalf("acceptance with evidence from an earlier delivery round error = %v, want submission not accepted", err)
	}
}

func TestAcceptTaskDeliveryBindsContractsV1AssessmentToPendingSubmission(t *testing.T) {
	f := newTaskReviewFixture(t)
	criterion := addAndActivateReviewCriterion(t, &f, d.VerificationModeAttestation, "review-contract-add-task-criterion")
	markWorkContractEnabled(t, &f)
	executorRoot, executor := f.addCredentialTree(t,
		"0199f293-0000-7000-8000-000000000001", "0199f294-0000-7000-8000-000000000001",
		"e30-contract-executor", "e30-contract-session", f.cc.PrincipalID, f.cc.Actor)
	_ = executorRoot
	acquired, err := f.service.AcquireWorkContract(f.identityContext(executor), freshCommandContext(f.cc, "review-contract-acquire"), a.AcquireWorkContractCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: f.work.Version, TTLSeconds: 60,
	})
	if err != nil {
		t.Fatalf("AcquireWorkContract() error = %v", err)
	}
	contract := acquired.Value.Contract
	submitted, err := f.service.SubmitTaskForReview(f.identityContext(executor), freshCommandContext(f.cc, "review-contract-delivery"), a.SubmitTaskForReviewCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: acquired.Value.WorkItem.Version,
		ContractID: &contract.ID, Authority: &a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SpecDigest},
		ExpectedContractVersion: ptrVersion(contract.Version),
		SubmissionMaterial:      &d.WorkResultMaterial{ContractID: contract.ID, WorkItemID: f.work.ID, SpecDigest: contract.SpecDigest, Summary: "Completed with a current submission"},
	})
	if err != nil {
		t.Fatalf("SubmitTaskForReview() error = %v", err)
	}
	reviewer, reviewerCC := newReviewPrincipal(t, f, "contract-reviewer", "reviewer-contract")
	submissionID := submitted.Value.Submission.ID
	assessment := recordTaskCriterionAssessment(t, f, reviewer, reviewerCC, criterion, submitted.Value.Delivery, submitted.Value.WorkItem.Version, &submissionID, nil, "review-contract-assess")
	command := makeTaskDeliveryAcceptCommand(f, submitted.Value.Delivery, assessment.WorkVersion, assessment.Assessment, "The exact submitted result satisfies the criterion")
	accepted, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), freshCommandContext(reviewerCC, "review-contract-accept"), command)
	if err != nil {
		t.Fatalf("AcceptTaskDelivery() with exact contracts_v1 assessment error = %v", err)
	}
	if accepted.Value.WorkItem.Lifecycle != d.WorkItemLifecycleDone || accepted.Value.WorkItem.CurrentConclusion == nil || accepted.Value.WorkItem.CurrentConclusion.SubmissionID == nil || *accepted.Value.WorkItem.CurrentConclusion.SubmissionID != submissionID {
		t.Fatalf("contracts_v1 acceptance did not conclude against the exact submission: %+v", accepted.Value.WorkItem)
	}
}

func TestConcurrentReviewersProduceOnlyOneTaskDeliveryDecision(t *testing.T) {
	f, executor, criterion := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	deliveryResult := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "Ready for competing review"}, "review-delivery-race")
	assessor, assessorCC := newReviewPrincipal(t, f, "race-assessor", "reviewer-race-assessor")
	assessment := recordTaskCriterionAssessment(t, f, assessor, assessorCC, criterion, deliveryResult.Delivery, deliveryResult.WorkItem.Version, nil, nil, "review-race-assessment")
	reviewerOne, reviewerOneCC := newReviewPrincipal(t, f, "race-reviewer-one", "reviewer-race-one")
	reviewerTwo, reviewerTwoCC := newReviewPrincipal(t, f, "race-reviewer-two", "reviewer-race-two")
	serviceOne, err := a.NewService(f.store, fixedClock{now: f.now}, &sequenceIDs{next: 31000})
	if err != nil {
		t.Fatal(err)
	}
	serviceTwo, err := a.NewService(f.store, fixedClock{now: f.now}, &sequenceIDs{next: 32000})
	if err != nil {
		t.Fatal(err)
	}
	accept := makeTaskDeliveryAcceptCommand(f, deliveryResult.Delivery, assessment.WorkVersion, assessment.Assessment, "Approve this exact delivery")
	changes := a.RequestTaskDeliveryChangesCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: assessment.WorkVersion,
		DeliveryID: deliveryResult.Delivery.ID, ExpectedDeliveryDigest: deliveryResult.Delivery.Digest, ExpectedDeliveryRound: deliveryResult.Delivery.Round,
		Reason: "Request a correction", Findings: []string{"One item needs adjustment"},
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		<-start
		_, err := serviceOne.AcceptTaskDelivery(f.identityContext(reviewerOne), freshCommandContext(reviewerOneCC, "review-race-accept"), accept)
		results <- err
	}()
	go func() {
		defer group.Done()
		<-start
		_, err := serviceTwo.RequestTaskDeliveryChanges(f.identityContext(reviewerTwo), freshCommandContext(reviewerTwoCC, "review-race-changes"), changes)
		results <- err
	}()
	close(start)
	group.Wait()
	close(results)
	successes, failures := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("concurrent decisions = %d successes, %d failures; want one each", successes, failures)
	}
	tx, err := f.store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	repository := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
	decision, err := repository.DecisionForDelivery(context.Background(), f.scope, deliveryResult.Delivery.ID)
	if err != nil || decision == nil {
		t.Fatalf("persisted decision = %+v, error = %v", decision, err)
	}
}
