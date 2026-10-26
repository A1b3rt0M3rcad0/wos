package application_test

import (
	"context"
	"testing"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestTaskDeliveryCorrectionCreatesFreshSupersedingRoundAndKeepsAssessmentHistory(t *testing.T) {
	f, executor, criterion := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	first := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "First immutable delivery"}, "e31-first-round-delivery")
	if first.Delivery.Round != 1 || first.Delivery.SupersedesDeliveryID != nil {
		t.Fatalf("first delivery chain = round %d supersedes %v; want round 1 with no predecessor", first.Delivery.Round, first.Delivery.SupersedesDeliveryID)
	}
	reviewer, reviewerCC := newReviewPrincipal(t, f, "e31-correction-reviewer", "reviewer-e31-correction")
	priorAssessment := recordTaskCriterionAssessment(t, f, reviewer, reviewerCC, criterion, first.Delivery, first.WorkItem.Version, nil, nil, "e31-first-round-assessment")
	changes, err := f.service.RequestTaskDeliveryChanges(f.identityContext(reviewer), freshCommandContext(reviewerCC, "e31-first-round-changes"), a.RequestTaskDeliveryChangesCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: priorAssessment.WorkVersion,
		DeliveryID: first.Delivery.ID, ExpectedDeliveryDigest: first.Delivery.Digest, ExpectedDeliveryRound: first.Delivery.Round,
		Reason: "The first result needs a correction", Findings: []string{"Provide an updated result"},
	})
	if err != nil {
		t.Fatalf("RequestTaskDeliveryChanges() = %v", err)
	}
	if changes.Value.WorkItem.Lifecycle != d.WorkItemLifecycleTodo || changes.Value.WorkItem.PendingDeliveryID != nil || changes.Value.WorkItem.CurrentLease != nil || changes.Value.WorkItem.CurrentContractID != nil {
		t.Fatalf("changes request did not close the review and execution authority: %+v", changes.Value.WorkItem)
	}
	if changes.Value.WorkItem.LatestDeliveryID == nil || *changes.Value.WorkItem.LatestDeliveryID != first.Delivery.ID || changes.Value.WorkItem.DeliveryRound != first.Delivery.Round {
		t.Fatalf("changes request lost prior delivery history: %+v", changes.Value.WorkItem)
	}
	if _, err := d.NewLegacyTaskDelivery(id("0199f2b1-0000-7000-8000-000000000001"), changes.Value.WorkItem, d.TaskDeliveryMaterial{Summary: "Attempt without reacquisition"}, f.cc.PrincipalID, nil, f.now); err == nil {
		t.Fatal("created a correction delivery before acquiring fresh execution authority")
	}

	claimed, err := f.service.ClaimWorkItem(f.identityContext(executor), freshCommandContext(f.cc, "e31-second-round-claim"), a.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: changes.Value.WorkItem.Version, TTL: d.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("ClaimWorkItem() for correction = %v", err)
	}
	f.work = claimed.Value
	second := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "Corrected immutable delivery"}, "e31-second-round-delivery")
	if second.Delivery.Round != first.Delivery.Round+1 || second.Delivery.SupersedesDeliveryID == nil || *second.Delivery.SupersedesDeliveryID != first.Delivery.ID {
		t.Fatalf("correction chain = round %d supersedes %v; want round %d superseding %s", second.Delivery.Round, second.Delivery.SupersedesDeliveryID, first.Delivery.Round+1, first.Delivery.ID)
	}
	if err := first.Delivery.Validate(); err != nil {
		t.Fatalf("first delivery changed after correction: %v", err)
	}

	current, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !hasAssessment(current.Value.Criteria.Assessments, priorAssessment.Assessment.ID) {
		t.Fatalf("previous-round assessment was deleted instead of retained as history: %+v", current.Value.Criteria.Assessments)
	}
	if current.Value.Criteria.CurrentAssessments[criterion.ID].ID != priorAssessment.Assessment.ID {
		t.Fatal("test precondition failed: prior assessment is not current, so only history behavior would be exercised")
	}
	oldAssessmentCommand := makeTaskDeliveryAcceptCommand(f, second.Delivery, second.WorkItem.Version, priorAssessment.Assessment, "Reject approval from the previous round")
	if _, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), freshCommandContext(reviewerCC, "e31-reject-old-round-proof"), oldAssessmentCommand); errorCode(err) != d.ErrorCodeSubmissionNotAccepted {
		t.Fatalf("acceptance with a previous-round assessment error = %v, want submission not accepted", err)
	}
	currentAssessment := recordTaskCriterionAssessment(t, f, reviewer, reviewerCC, criterion, second.Delivery, second.WorkItem.Version, nil, nil, "e31-second-round-assessment")
	accept := makeTaskDeliveryAcceptCommand(f, second.Delivery, currentAssessment.WorkVersion, currentAssessment.Assessment, "Accept the corrected delivery")
	accepted, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), freshCommandContext(reviewerCC, "e31-accept-correction"), accept)
	if err != nil {
		t.Fatalf("acceptance with current correction assessment = %v", err)
	}
	if accepted.Value.WorkItem.Lifecycle != d.WorkItemLifecycleDone || accepted.Value.Decision.Round != second.Delivery.Round {
		t.Fatalf("acceptance did not decide the current correction round: %+v", accepted.Value)
	}
}

func TestTaskDeliveryDefinitionCanChangeOnlyAfterReviewClosesAndTaskReturnsToBacklog(t *testing.T) {
	f, executor, _ := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	delivery := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "Pending delivery"}, "e31-definition-first-delivery")
	newCriterion := a.AddCriterionCommand{
		Owner: f.work.Ref(), Title: "Add a correction criterion", Description: "Only editable during explicit replanning", Required: true,
		VerificationMode: d.VerificationModeAttestation,
	}
	newCriterion.ExpectedVersion = delivery.WorkItem.Version
	if _, err := f.service.AddCriterion(context.Background(), freshCommandContext(f.cc, "e31-reject-edit-pending"), newCriterion); err == nil {
		t.Fatal("changed Task criteria while a delivery was under review")
	}
	reviewer, reviewerCC := newReviewPrincipal(t, f, "e31-planning-reviewer", "reviewer-e31-planning")
	changes, err := f.service.RequestTaskDeliveryChanges(f.identityContext(reviewer), freshCommandContext(reviewerCC, "e31-close-pending-review"), a.RequestTaskDeliveryChangesCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: delivery.WorkItem.Version,
		DeliveryID: delivery.Delivery.ID, ExpectedDeliveryDigest: delivery.Delivery.Digest, ExpectedDeliveryRound: delivery.Delivery.Round,
		Reason: "Return work for planned correction", Findings: []string{"Revise the task definition before another execution"},
	})
	if err != nil {
		t.Fatalf("RequestTaskDeliveryChanges() = %v", err)
	}
	newCriterion.ExpectedVersion = changes.Value.WorkItem.Version
	if _, err := f.service.AddCriterion(context.Background(), freshCommandContext(f.cc, "e31-reject-edit-todo"), newCriterion); err == nil {
		t.Fatal("changed Task criteria in To do without returning it to Backlog")
	}
	deferred, err := f.service.DeferWorkItem(context.Background(), freshCommandContext(f.cc, "e31-defer-for-replanning"), a.DeferWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: changes.Value.WorkItem.Version,
	})
	if err != nil {
		t.Fatalf("DeferWorkItem() for explicit replanning = %v", err)
	}
	newCriterion.ExpectedVersion = deferred.Value.Version
	added, err := f.service.AddCriterion(context.Background(), freshCommandContext(f.cc, "e31-add-replanned-criterion"), newCriterion)
	if err != nil {
		t.Fatalf("AddCriterion() after explicit return to Backlog = %v", err)
	}
	if added.Value.Title != newCriterion.Title {
		t.Fatalf("added criterion title = %q, want %q", added.Value.Title, newCriterion.Title)
	}
	if deferred.Value.Lifecycle != d.WorkItemLifecycleBacklog || deferred.Value.CurrentLease != nil || deferred.Value.PendingDeliveryID != nil {
		t.Fatalf("replanning state retained execution/review authority: %+v", deferred.Value)
	}
}

func hasAssessment(assessments []d.CriterionAssessment, id d.ID) bool {
	for _, assessment := range assessments {
		if assessment.ID == id {
			return true
		}
	}
	return false
}
