package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type objectiveAutoFixture struct {
	workspace *operationalProfileFixture
	objective d.Objective
	task      d.WorkItem
	human     ports.Credential
	agent     ports.Credential
}

type objectiveAutoAcceptance struct {
	result                a.MutationResult[a.TaskDeliveryReviewResult]
	outcomeRevisionBefore d.OutcomeRevision
}

func newObjectiveAutoFixture(t *testing.T) objectiveAutoFixture {
	t.Helper()
	f := newOperationalProfileFixture(t)
	setCompletionOutcomeProfile(t, f, d.OperationalProfileOperationalV2, d.CompletionPolicyStrictWork)

	objective := createObjectiveFixture(t, f.Service, f.Command, f.Scope, "Delegated completion target", d.ObjectiveLifecycleInProgress)
	task := createObjectiveAutoReadyTask(t, f, objective.ID, "Verified work for automatic Objective", "auto-completion-task-definition")
	human := f.Credentials["operator_human"]
	grantObjectiveCompletionPermission(t, f, human, true)
	ctx, cc, cmd := completionModeRequest(t, f, objective, human, "auto-completion-delegate-"+objective.ID.String(), d.ObjectiveCompletionModeAutomatic)
	if _, err := f.Service.SetObjectiveCompletionMode(ctx, cc, cmd); err != nil {
		t.Fatalf("SetObjectiveCompletionMode(automatic) error = %v", err)
	}
	return objectiveAutoFixture{workspace: f, objective: objective, task: task, human: human, agent: f.Credentials["operator_agent"]}
}

func createObjectiveAutoReadyTask(t *testing.T, f *operationalProfileFixture, objectiveID d.ID, title, key string) d.WorkItem {
	t.Helper()
	created, err := f.Service.CreateWorkItem(f.Context, objectiveAutoCommandContext(f, f.Credentials["operator_human"], key+"-create"), a.CreateWorkItemCommand{
		Scope: f.Scope, ObjectiveID: &objectiveID, Title: title, Priority: d.PriorityNormal,
		Lifecycle: d.WorkItemLifecycleBacklog,
		InitialCriteria: []a.InitialCriterion{{
			Title: "The requested Task work is complete", Required: true,
			VerificationMode: d.VerificationModeAttestation,
		}},
	})
	if err != nil {
		t.Fatalf("CreateWorkItem(Backlog with required criterion) error = %v", err)
	}
	activated, err := f.Service.ActivateWorkItem(f.Context, objectiveAutoCommandContext(f, f.Credentials["operator_human"], key+"-activate"), a.ActivateWorkItemCommand{
		Scope: f.Scope, WorkItemID: created.Value.ID, ExpectedVersion: created.Value.Version,
	})
	if err != nil {
		t.Fatalf("ActivateWorkItem() error = %v", err)
	}
	current := activated.Value
	criterion := current.Criteria.Items[0]
	if _, err := f.Service.AttestCriterion(objectiveAutoIdentityContext(f.Context, f.Credentials["operator_human"]), objectiveAutoCommandContext(f, f.Credentials["operator_human"], key+"-attest"), a.AttestCriterionCommand{
		Owner: current.Ref(), CriterionID: criterion.ID, CriterionRevision: criterion.Revision,
		ExpectedVersion: current.Version, Result: d.AssessmentResultMet,
		Rationale: "The Task acceptance criteria were verified before delivery review.",
	}); err != nil {
		t.Fatalf("AttestCriterion(Task) error = %v", err)
	}
	loaded, err := f.Service.GetWorkItem(f.Context, f.Scope, current.ID)
	if err != nil {
		t.Fatalf("GetWorkItem(after Task criterion assessment) error = %v", err)
	}
	return loaded.Value
}

func objectiveAutoIdentityContext(ctx context.Context, credential ports.Credential) context.Context {
	return a.WithIdentity(ctx, a.Identity{
		CredentialID: credential.ID, CredentialDigest: credential.Digest,
		PrincipalID: credential.PrincipalID, NamespaceID: credential.NamespaceID, Actor: credential.Actor,
	})
}

func objectiveAutoCommandContext(f *operationalProfileFixture, credential ports.Credential, key string) d.CommandContext {
	cc := freshCommandContext(f.Command, key)
	cc.PrincipalID = credential.PrincipalID
	cc.Actor = credential.Actor
	return cc
}

func attestObjectiveAutoCriterion(t *testing.T, f objectiveAutoFixture) (d.CriterionAssessment, d.ID) {
	t.Helper()
	loaded, err := f.workspace.Service.GetObjective(f.workspace.Context, f.workspace.Scope, f.objective.ID)
	if err != nil {
		t.Fatalf("GetObjective(before attestation) error = %v", err)
	}
	if len(loaded.Value.Criteria.Items) != 1 {
		t.Fatalf("automatic-completion Objective criteria = %d, want one", len(loaded.Value.Criteria.Items))
	}
	criterion := loaded.Value.Criteria.Items[0]
	cc := objectiveAutoCommandContext(f.workspace, f.human, "auto-completion-attest-"+criterion.ID.String())
	result, err := f.workspace.Service.AttestCriterion(objectiveAutoIdentityContext(f.workspace.Context, f.human), cc, a.AttestCriterionCommand{
		Owner: loaded.Value.Ref(), CriterionID: criterion.ID, CriterionRevision: criterion.Revision,
		ExpectedVersion: loaded.Value.Version, Result: d.AssessmentResultMet,
		Rationale: "The current Objective criterion was verified by its authorized reviewer.",
	})
	if err != nil {
		t.Fatalf("AttestCriterion(Objective) error = %v", err)
	}
	return result.Value, result.CommandID
}

func submitObjectiveAutoTask(t *testing.T, f objectiveAutoFixture, service *a.Service, task d.WorkItem, key string) a.TaskDeliveryResult {
	t.Helper()
	ctx := objectiveAutoIdentityContext(f.workspace.Context, f.agent)
	agentCC := objectiveAutoCommandContext(f.workspace, f.agent, key+"-claim")
	claimed, err := service.ClaimWorkItem(ctx, agentCC, a.ClaimWorkItemCommand{
		Scope: f.workspace.Scope, WorkItemID: task.ID, ExpectedVersion: task.Version, TTL: d.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("ClaimWorkItem() error = %v", err)
	}
	if claimed.Value.CurrentLease == nil {
		t.Fatal("ClaimWorkItem() returned no lease")
	}
	lease := claimed.Value.CurrentLease
	submission, err := service.SubmitTaskForReview(ctx, objectiveAutoCommandContext(f.workspace, f.agent, key+"-submit"), a.SubmitTaskForReviewCommand{
		Scope: f.workspace.Scope, WorkItemID: task.ID, ExpectedWorkItemVersion: claimed.Value.Version,
		ClaimID: &lease.ClaimID, FencingToken: lease.FencingToken,
		LegacyMaterial: ptrTaskDeliveryMaterial(d.TaskDeliveryMaterial{Summary: "The requested Task work is complete and ready for review."}),
	})
	if err != nil {
		t.Fatalf("SubmitTaskForReview() error = %v", err)
	}
	work, _ := assessObjectiveAutoTaskDelivery(t, f, service, task.ID, submission.Value, key+"-assessment")
	result := submission.Value
	result.WorkItem = work
	return result
}

func assessObjectiveAutoTaskDelivery(t *testing.T, f objectiveAutoFixture, service *a.Service, taskID d.ID, delivery a.TaskDeliveryResult, key string) (d.WorkItem, []d.CriterionAssessmentRef) {
	t.Helper()
	work, err := service.GetWorkItem(f.workspace.Context, f.workspace.Scope, taskID)
	if err != nil {
		t.Fatalf("GetWorkItem(before delivery assessment) error = %v", err)
	}
	deliveryRef := &d.TaskDeliveryAssessmentRef{ID: delivery.Delivery.ID, Digest: delivery.Delivery.Digest, Round: delivery.Delivery.Round}
	for _, criterion := range work.Value.Criteria.Items {
		if !criterion.Required || criterion.Status != d.CriterionStatusActive {
			continue
		}
		if _, err := service.RecordCriterionAssessment(objectiveAutoIdentityContext(f.workspace.Context, f.human), objectiveAutoCommandContext(f.workspace, f.human, key+"-"+criterion.ID.String()), a.RecordCriterionAssessmentCommand{
			Owner: work.Value.Ref(), ExpectedVersion: work.Value.Version, CriterionID: criterion.ID,
			CriterionRevision: criterion.Revision, Result: d.AssessmentResultMet,
			Rationale: "The exact pending Task delivery satisfies this criterion.", TaskDeliveryRef: deliveryRef,
		}); err != nil {
			t.Fatalf("RecordCriterionAssessment(exact Task delivery) error = %v", err)
		}
		work, err = service.GetWorkItem(f.workspace.Context, f.workspace.Scope, taskID)
		if err != nil {
			t.Fatalf("GetWorkItem(after delivery assessment) error = %v", err)
		}
	}
	refs := make([]d.CriterionAssessmentRef, 0, len(work.Value.Criteria.Items))
	for _, criterion := range work.Value.Criteria.Items {
		if !criterion.Required || criterion.Status != d.CriterionStatusActive {
			continue
		}
		assessment, ok := work.Value.Criteria.CurrentAssessments[criterion.ID]
		if !ok || assessment.TaskDeliveryRef == nil || assessment.TaskDeliveryRef.ID != delivery.Delivery.ID || assessment.TaskDeliveryRef.Digest != delivery.Delivery.Digest || assessment.TaskDeliveryRef.Round != delivery.Delivery.Round {
			t.Fatalf("current assessment for criterion %s is not bound to the exact pending delivery: %+v", criterion.ID, assessment)
		}
		refs = append(refs, d.CriterionAssessmentRef{AssessmentID: assessment.ID, CriterionID: assessment.CriterionID, CriterionRevision: assessment.CriterionRevision, Result: assessment.Result})
	}
	return work.Value, refs
}

func acceptPreparedObjectiveAutoTask(t *testing.T, f objectiveAutoFixture, service *a.Service, task d.WorkItem, delivery a.TaskDeliveryResult, key string) objectiveAutoAcceptance {
	t.Helper()
	stateBeforeAcceptance, err := service.GetOutcomeState(f.workspace.Context, f.workspace.Scope)
	if err != nil {
		t.Fatalf("GetOutcomeState(before acceptance) error = %v", err)
	}
	currentWork, assessmentRefs := objectiveAutoAssessmentRefs(t, service, f, task.ID, delivery)
	reviewerCC := objectiveAutoCommandContext(f.workspace, f.human, key+"-accept")
	returned, err := service.AcceptTaskDelivery(objectiveAutoIdentityContext(f.workspace.Context, f.human), reviewerCC, a.AcceptTaskDeliveryCommand{
		Scope: f.workspace.Scope, WorkItemID: task.ID, ExpectedWorkItemVersion: currentWork.Version,
		DeliveryID: delivery.Delivery.ID, ExpectedDeliveryDigest: delivery.Delivery.Digest,
		ExpectedDeliveryRound: delivery.Delivery.Round, Reason: "The delivery satisfies the Task request.", AssessmentRefs: assessmentRefs,
	})
	if err != nil {
		t.Fatalf("AcceptTaskDelivery() error = %v", err)
	}
	return objectiveAutoAcceptance{result: returned, outcomeRevisionBefore: stateBeforeAcceptance.OutcomeRevision}
}

func objectiveAutoAssessmentRefs(t *testing.T, service *a.Service, f objectiveAutoFixture, taskID d.ID, delivery a.TaskDeliveryResult) (d.WorkItem, []d.CriterionAssessmentRef) {
	t.Helper()
	work, err := service.GetWorkItem(f.workspace.Context, f.workspace.Scope, taskID)
	if err != nil {
		t.Fatalf("GetWorkItem(before acceptance) error = %v", err)
	}
	refs := make([]d.CriterionAssessmentRef, 0, len(work.Value.Criteria.Items))
	for _, criterion := range work.Value.Criteria.Items {
		if !criterion.Required || criterion.Status != d.CriterionStatusActive {
			continue
		}
		assessment, ok := work.Value.Criteria.CurrentAssessments[criterion.ID]
		if !ok || assessment.TaskDeliveryRef == nil || assessment.TaskDeliveryRef.ID != delivery.Delivery.ID || assessment.TaskDeliveryRef.Digest != delivery.Delivery.Digest || assessment.TaskDeliveryRef.Round != delivery.Delivery.Round {
			t.Fatalf("current assessment for criterion %s is not bound to exact delivery %s", criterion.ID, delivery.Delivery.ID)
		}
		refs = append(refs, d.CriterionAssessmentRef{AssessmentID: assessment.ID, CriterionID: assessment.CriterionID, CriterionRevision: assessment.CriterionRevision, Result: assessment.Result})
	}
	return work.Value, refs
}

func acceptObjectiveAutoTask(t *testing.T, f objectiveAutoFixture, service *a.Service, key string) objectiveAutoAcceptance {
	t.Helper()
	delivery := submitObjectiveAutoTask(t, f, service, f.task, key)
	return acceptPreparedObjectiveAutoTask(t, f, service, f.task, delivery, key)
}

func TestAutoObjectiveAchievementRunsAfterLastTaskAcceptanceAndReplaysOnce(t *testing.T) {
	f := newObjectiveAutoFixture(t)
	assessment, _ := attestObjectiveAutoCriterion(t, f)
	beforeEvents := f.workspace.Store.SnapshotDomainEvents(f.workspace.Scope)
	acceptance := acceptObjectiveAutoTask(t, f, f.workspace.Service, "auto-completion-last-task")
	if len(acceptance.result.DerivedObjectiveAchievements) != 1 {
		t.Fatalf("derived achievements = %+v, want exactly one", acceptance.result.DerivedObjectiveAchievements)
	}
	assertAutoObjectiveAchievement(t, f, acceptance, assessment, beforeEvents)
	eventCountAfterAcceptance := len(f.workspace.Store.SnapshotDomainEvents(f.workspace.Scope))

	// A lost response is recovered from the original protected receipt. It must
	// return the derived Objective result without emitting another event.
	stateAfter, err := f.workspace.Service.GetOutcomeState(f.workspace.Context, f.workspace.Scope)
	if err != nil {
		t.Fatal(err)
	}
	work, err := f.workspace.Service.GetWorkItem(f.workspace.Context, f.workspace.Scope, f.task.ID)
	if err != nil {
		t.Fatal(err)
	}
	replayCC := objectiveAutoCommandContext(f.workspace, f.human, "auto-completion-last-task-accept")
	replayDelivery := a.TaskDeliveryResult{WorkItem: acceptance.result.Value.WorkItem, Delivery: acceptance.result.Value.Delivery}
	_, assessmentRefs := objectiveAutoAssessmentRefs(t, f.workspace.Service, f, f.task.ID, replayDelivery)
	replay, err := f.workspace.Service.AcceptTaskDelivery(objectiveAutoIdentityContext(f.workspace.Context, f.human), replayCC, a.AcceptTaskDeliveryCommand{
		Scope: f.workspace.Scope, WorkItemID: f.task.ID, ExpectedWorkItemVersion: acceptance.result.Value.WorkItem.Version - 1,
		DeliveryID: acceptance.result.Value.Delivery.ID, ExpectedDeliveryDigest: acceptance.result.Value.Delivery.Digest, ExpectedDeliveryRound: acceptance.result.Value.Delivery.Round,
		Reason: "The delivery satisfies the Task request.", AssessmentRefs: assessmentRefs,
	})
	if err != nil || !replay.IdempotentReplay || !reflect.DeepEqual(replay.DerivedObjectiveAchievements, acceptance.result.DerivedObjectiveAchievements) {
		t.Fatalf("acceptance replay = %+v, err=%v", replay, err)
	}
	if after := len(f.workspace.Store.SnapshotDomainEvents(f.workspace.Scope)); after != eventCountAfterAcceptance {
		t.Fatalf("idempotent acceptance replay appended events: before=%d after=%d", eventCountAfterAcceptance, after)
	}
	if stateAfter.Outcome.Lifecycle != d.OutcomeLifecycleActive || stateAfter.OutcomeRevision != acceptance.result.OutcomeRevision {
		t.Fatalf("automatic Objective closure changed or re-advanced Outcome: lifecycle=%s revision=%d acceptance revision=%d", stateAfter.Outcome.Lifecycle, stateAfter.OutcomeRevision, acceptance.result.OutcomeRevision)
	}
	if work.Value.Lifecycle != d.WorkItemLifecycleDone {
		t.Fatalf("accepted Task lifecycle = %s, want done", work.Value.Lifecycle)
	}
}

func TestAutoObjectiveAchievementRunsAfterLastRequiredObjectiveAssessment(t *testing.T) {
	f := newObjectiveAutoFixture(t)
	accepted := acceptObjectiveAutoTask(t, f, f.workspace.Service, "auto-completion-last-criterion-task")
	if len(accepted.result.DerivedObjectiveAchievements) != 0 {
		t.Fatalf("Objective closed before its required criterion was assessed: %+v", accepted.result.DerivedObjectiveAchievements)
	}
	stateBefore, err := f.workspace.Service.GetOutcomeState(f.workspace.Context, f.workspace.Scope)
	if err != nil {
		t.Fatal(err)
	}
	assessment, assessmentCommandID := attestObjectiveAutoCriterion(t, f)
	objective, err := f.workspace.Service.GetObjective(f.workspace.Context, f.workspace.Scope, f.objective.ID)
	if err != nil {
		t.Fatal(err)
	}
	if objective.Value.Lifecycle != d.ObjectiveLifecycleAchieved || objective.Value.CurrentConclusion == nil || len(objective.Value.CurrentConclusion.Assessments) != 1 || objective.Value.CurrentConclusion.Assessments[0].AssessmentID != assessment.ID {
		t.Fatalf("final assessment did not produce a conclusion bound to the real assessment: %+v", objective.Value)
	}
	if objective.Value.CurrentConclusion.DerivedCompletion == nil || objective.Value.CurrentConclusion.DerivedCompletion.DelegatorCredentialID != f.human.ID {
		t.Fatalf("derived conclusion is missing its explicit delegation provenance: %+v", objective.Value.CurrentConclusion)
	}
	stateAfter, err := f.workspace.Service.GetOutcomeState(f.workspace.Context, f.workspace.Scope)
	if err != nil || stateAfter.Outcome.Lifecycle != d.OutcomeLifecycleActive || stateAfter.OutcomeRevision != stateBefore.OutcomeRevision+1 {
		t.Fatalf("last-criterion completion advanced incorrect state: state=%+v err=%v", stateAfter, err)
	}
	var derivedEvents int
	for _, event := range f.workspace.Store.SnapshotDomainEvents(f.workspace.Scope) {
		if event.CommandID == assessmentCommandID && (event.EventType == "objective.achieved" || event.EventType == "objective.conclusion_recorded") {
			derivedEvents++
		}
	}
	if derivedEvents != 2 {
		t.Fatalf("last-assessment derived Objective events = %d, want achievement and conclusion records", derivedEvents)
	}
}

func TestConcurrentLastTaskAcceptancesDeriveOneObjectiveConclusion(t *testing.T) {
	f := newObjectiveAutoFixture(t)
	attestObjectiveAutoCriterion(t, f)
	secondTask := createObjectiveAutoReadyTask(t, f.workspace, f.objective.ID, "Second task in the automatic Objective", "auto-completion-second-task")
	firstDelivery := submitObjectiveAutoTask(t, f, f.workspace.Service, f.task, "auto-completion-concurrent-first")
	secondDelivery := submitObjectiveAutoTask(t, f, f.workspace.Service, secondTask, "auto-completion-concurrent-second")
	before, err := f.workspace.Service.GetOutcomeState(f.workspace.Context, f.workspace.Scope)
	if err != nil {
		t.Fatal(err)
	}
	keys := []string{"auto-completion-concurrent-first", "auto-completion-concurrent-second"}
	tasks := []d.WorkItem{f.task, secondTask}
	deliveries := []a.TaskDeliveryResult{firstDelivery, secondDelivery}
	currentWorks := make([]d.WorkItem, len(tasks))
	assessmentRefs := make([][]d.CriterionAssessmentRef, len(tasks))
	for index := range tasks {
		currentWorks[index], assessmentRefs[index] = objectiveAutoAssessmentRefs(t, f.workspace.Service, f, tasks[index].ID, deliveries[index])
	}
	type response struct {
		result a.MutationResult[a.TaskDeliveryReviewResult]
		err    error
	}
	responses := make(chan response, 2)
	for index := range tasks {
		index := index
		go func() {
			cc := objectiveAutoCommandContext(f.workspace, f.human, keys[index]+"-accept")
			delivery := deliveries[index]
			result, err := f.workspace.Service.AcceptTaskDelivery(objectiveAutoIdentityContext(f.workspace.Context, f.human), cc, a.AcceptTaskDeliveryCommand{
				Scope: f.workspace.Scope, WorkItemID: tasks[index].ID, ExpectedWorkItemVersion: currentWorks[index].Version,
				DeliveryID: delivery.Delivery.ID, ExpectedDeliveryDigest: delivery.Delivery.Digest,
				ExpectedDeliveryRound: delivery.Delivery.Round, Reason: "The delivery satisfies the Task request.", AssessmentRefs: assessmentRefs[index],
			})
			responses <- response{result: result, err: err}
		}()
	}
	var derivedCount int
	commandIDs := make([]d.ID, 0, 2)
	for range tasks {
		response := <-responses
		if response.err != nil {
			t.Fatalf("concurrent AcceptTaskDelivery() error = %v", response.err)
		}
		derivedCount += len(response.result.DerivedObjectiveAchievements)
		commandIDs = append(commandIDs, response.result.CommandID)
	}
	if derivedCount != 1 {
		t.Fatalf("concurrent final Task acceptances derived %d achievements, want exactly one", derivedCount)
	}
	state, err := f.workspace.Service.GetOutcomeState(f.workspace.Context, f.workspace.Scope)
	if err != nil || state.OutcomeRevision != before.OutcomeRevision+2 || state.Outcome.Lifecycle != d.OutcomeLifecycleActive {
		t.Fatalf("concurrent acceptance revisions/outcome = %d/%s, want two Task revisions and active Outcome; err=%v", state.OutcomeRevision, state.Outcome.Lifecycle, err)
	}
	objective, err := f.workspace.Service.GetObjective(f.workspace.Context, f.workspace.Scope, f.objective.ID)
	if err != nil || objective.Value.Lifecycle != d.ObjectiveLifecycleAchieved || objective.Value.CurrentConclusion == nil {
		t.Fatalf("concurrent final acceptance did not persist one Objective conclusion: %+v err=%v", objective.Value, err)
	}
	var achievements int
	for _, event := range f.workspace.Store.SnapshotDomainEvents(f.workspace.Scope) {
		if event.EventType == "objective.achieved" {
			achievements++
		}
	}
	if achievements != 1 {
		t.Fatalf("Objective achievement event count after concurrent final acceptances = %d, want one (commands %v)", achievements, commandIDs)
	}
}

func TestAutoObjectiveAchievementDoesNotCloseWhileTaskDeliveryIsPending(t *testing.T) {
	f := newObjectiveAutoFixture(t)
	attestObjectiveAutoCriterion(t, f)
	issue, err := f.workspace.Service.CreateIssue(f.workspace.Context, f.workspace.Command, a.CreateIssueCommand{
		Scope: f.workspace.Scope, ObjectiveID: f.objective.ID, Title: "Issue that remains open during review",
		Severity: d.IssueSeverityMinor,
	})
	if err != nil {
		t.Fatal(err)
	}
	agentCtx := objectiveAutoIdentityContext(f.workspace.Context, f.agent)
	claim, err := f.workspace.Service.ClaimWorkItem(agentCtx, objectiveAutoCommandContext(f.workspace, f.agent, "auto-completion-pending-claim"), a.ClaimWorkItemCommand{
		Scope: f.workspace.Scope, WorkItemID: f.task.ID, ExpectedVersion: f.task.Version, TTL: d.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	lease := claim.Value.CurrentLease
	_, err = f.workspace.Service.SubmitTaskForReview(agentCtx, objectiveAutoCommandContext(f.workspace, f.agent, "auto-completion-pending-submit"), a.SubmitTaskForReviewCommand{
		Scope: f.workspace.Scope, WorkItemID: f.task.ID, ExpectedWorkItemVersion: claim.Value.Version,
		ClaimID: &lease.ClaimID, FencingToken: lease.FencingToken,
		LegacyMaterial: ptrTaskDeliveryMaterial(d.TaskDeliveryMaterial{Summary: "Submitted, but not accepted."}),
	})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := f.workspace.Service.ResolveIssue(f.workspace.Context, freshCommandContext(f.workspace.Command, "auto-completion-resolve-while-pending"), a.ResolveIssueCommand{
		Scope: f.workspace.Scope, IssueID: issue.Value.ID, ExpectedVersion: issue.Value.Version,
		ResolutionSummary: "Resolved the Issue while the Task still awaits review.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.DerivedObjectiveAchievements) != 0 {
		t.Fatalf("pending Task delivery was treated as accepted work: %+v", resolved.DerivedObjectiveAchievements)
	}
	objective, err := f.workspace.Service.GetObjective(f.workspace.Context, f.workspace.Scope, f.objective.ID)
	if err != nil || objective.Value.Lifecycle != d.ObjectiveLifecycleInProgress || objective.Value.CurrentConclusion != nil {
		t.Fatalf("pending review closed its Objective: objective=%+v err=%v", objective.Value, err)
	}
}

func TestAutoObjectiveAchievementRunsAfterLastOwnedIssueResolution(t *testing.T) {
	f := newObjectiveAutoFixture(t)
	assessment, _ := attestObjectiveAutoCriterion(t, f)
	created, err := f.workspace.Service.CreateIssue(f.workspace.Context, f.workspace.Command, a.CreateIssueCommand{
		Scope: f.workspace.Scope, ObjectiveID: f.objective.ID, Title: "A pending operational issue",
		Severity: d.IssueSeverityMinor,
	})
	if err != nil {
		t.Fatalf("CreateIssue() error = %v", err)
	}
	accepted := acceptObjectiveAutoTask(t, f, f.workspace.Service, "auto-completion-last-issue-task")
	if len(accepted.result.DerivedObjectiveAchievements) != 0 {
		t.Fatalf("Objective closed while an owned Issue remained open: %+v", accepted.result.DerivedObjectiveAchievements)
	}
	before, err := f.workspace.Service.GetOutcomeState(f.workspace.Context, f.workspace.Scope)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := f.workspace.Store.SnapshotDomainEvents(f.workspace.Scope)
	resolved, err := f.workspace.Service.ResolveIssue(f.workspace.Context, freshCommandContext(f.workspace.Command, "auto-completion-resolve-last-issue"), a.ResolveIssueCommand{
		Scope: f.workspace.Scope, IssueID: created.Value.ID, ExpectedVersion: created.Value.Version,
		ResolutionSummary: "The operational issue has been resolved.",
	})
	if err != nil {
		t.Fatalf("ResolveIssue() error = %v", err)
	}
	if resolved.Value.Lifecycle != d.IssueLifecycleResolved {
		t.Fatalf("Issue lifecycle = %s, want resolved", resolved.Value.Lifecycle)
	}
	objective, err := f.workspace.Service.GetObjective(f.workspace.Context, f.workspace.Scope, f.objective.ID)
	if err != nil || objective.Value.Lifecycle != d.ObjectiveLifecycleAchieved || objective.Value.CurrentConclusion == nil || objective.Value.CurrentConclusion.Assessments[0].AssessmentID != assessment.ID {
		t.Fatalf("last-Issue resolution did not derive the Objective conclusion: objective=%+v err=%v", objective.Value, err)
	}
	if len(resolved.DerivedObjectiveAchievements) != 1 || resolved.DerivedObjectiveAchievements[0].Objective.ID != f.objective.ID {
		t.Fatalf("Issue resolution mutation omitted its derived achievement: %+v", resolved.DerivedObjectiveAchievements)
	}
	if resolved.OutcomeRevision != before.OutcomeRevision+1 {
		t.Fatalf("Issue resolution plus derived achievement advanced Outcome revision from %d to %d", before.OutcomeRevision, resolved.OutcomeRevision)
	}
	assertDerivedEventsCausal(t, f.workspace.Store.SnapshotDomainEvents(f.workspace.Scope), resolved.CommandID, eventsBefore)
}

func TestAutoObjectiveAchievementCanRunWhenAutomaticObjectiveStarts(t *testing.T) {
	f := newOperationalProfileFixture(t)
	created, err := f.Service.CreateObjective(f.Context, f.Command, a.CreateObjectiveCommand{
		Scope: f.Scope, Title: "Preconfigured automatic Objective", Priority: d.PriorityNormal,
		InitialCriteria: []a.InitialCriterion{{Title: "The Objective is complete", Required: true, VerificationMode: d.VerificationModeAttestation}},
	})
	if err != nil {
		t.Fatal(err)
	}
	task := createObjectiveAutoReadyTask(t, f, created.Value.ID, "Task completed before Objective start", "auto-start-task-definition")
	human := f.Credentials["operator_human"]
	agent := f.Credentials["operator_agent"]
	grantObjectiveCompletionPermission(t, f, human, true)
	criterion := created.Value.Criteria.Items[0]
	attest, err := f.Service.AttestCriterion(objectiveAutoIdentityContext(f.Context, human), objectiveAutoCommandContext(f, human, "auto-start-pre-attest"), a.AttestCriterionCommand{
		Owner: created.Value.Ref(), CriterionID: criterion.ID, CriterionRevision: criterion.Revision,
		ExpectedVersion: created.Value.Version, Result: d.AssessmentResultMet, Rationale: "Verified before start.",
	})
	if err != nil {
		t.Fatalf("AttestCriterion(planned Objective) error = %v", err)
	}
	prepared := objectiveAutoFixture{workspace: f, objective: created.Value, task: task, human: human, agent: agent}
	// Native delivery and review can be persisted while this compatibility
	// Objective is planned. Its automation is delegated only after the Outcome
	// profile is moved to its strict current policy.
	accepted := acceptObjectiveAutoTask(t, prepared, f.Service, "auto-start-accepted-task")
	if len(accepted.result.DerivedObjectiveAchievements) != 0 {
		t.Fatalf("pre-start Objective unexpectedly closed before explicit start: %+v", accepted.result.DerivedObjectiveAchievements)
	}
	setCompletionOutcomeProfile(t, f, d.OperationalProfileOperationalV2, d.CompletionPolicyStrictWork)
	currentObjective, err := f.Service.GetObjective(f.Context, f.Scope, created.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cc, modeCommand := completionModeRequest(t, f, currentObjective.Value, human, "auto-start-delegate", d.ObjectiveCompletionModeAutomatic)
	modeChange, err := f.Service.SetObjectiveCompletionMode(ctx, cc, modeCommand)
	if err != nil {
		t.Fatalf("SetObjectiveCompletionMode(planned automatic) error = %v", err)
	}
	if len(modeChange.DerivedObjectiveAchievements) != 0 {
		t.Fatalf("planned automatic Objective closed before it started: %+v", modeChange.DerivedObjectiveAchievements)
	}
	started, err := f.Service.StartObjective(f.Context, freshCommandContext(f.Command, "auto-start-objective"), a.StartObjectiveCommand{
		Scope: f.Scope, ObjectiveID: created.Value.ID, ExpectedVersion: modeChange.Value.Objective.Version,
	})
	if err != nil {
		t.Fatalf("StartObjective() error = %v", err)
	}
	if started.Value.Lifecycle != d.ObjectiveLifecycleInProgress || len(started.DerivedObjectiveAchievements) != 1 || started.DerivedObjectiveAchievements[0].Objective.ID != created.Value.ID {
		t.Fatalf("StartObjective did not run the completion effect after enabling eligibility: %+v", started)
	}
	objective, err := f.Service.GetObjective(f.Context, f.Scope, created.Value.ID)
	if err != nil || objective.Value.Lifecycle != d.ObjectiveLifecycleAchieved || objective.Value.CurrentConclusion == nil || objective.Value.CurrentConclusion.Assessments[0].AssessmentID != attest.Value.ID {
		t.Fatalf("StartObjective-derived conclusion did not retain the earlier real assessment: %+v err=%v", objective.Value, err)
	}
}

func TestAutoObjectiveAchievementFailsClosedAfterDelegationPolicyRevocation(t *testing.T) {
	f := newObjectiveAutoFixture(t)
	_, _ = attestObjectiveAutoCriterion(t, f)
	grantObjectiveCompletionPermission(t, f.workspace, f.human, false)
	before, err := f.workspace.Service.GetOutcomeState(f.workspace.Context, f.workspace.Scope)
	if err != nil {
		t.Fatal(err)
	}
	accepted := acceptObjectiveAutoTask(t, f, f.workspace.Service, "auto-completion-revoked-policy")
	if len(accepted.result.DerivedObjectiveAchievements) != 0 {
		t.Fatalf("revoked completion authority still derived an achievement: %+v", accepted.result.DerivedObjectiveAchievements)
	}
	objective, err := f.workspace.Service.GetObjective(f.workspace.Context, f.workspace.Scope, f.objective.ID)
	if err != nil || objective.Value.Lifecycle != d.ObjectiveLifecycleInProgress || objective.Value.CurrentConclusion != nil {
		t.Fatalf("revoked delegation closed the Objective: objective=%+v err=%v", objective.Value, err)
	}
	after, err := f.workspace.Service.GetOutcomeState(f.workspace.Context, f.workspace.Scope)
	if err != nil || after.OutcomeRevision != accepted.result.OutcomeRevision {
		t.Fatalf("unexpected Outcome revision after accepted work with revoked delegation: before=%d after=%d err=%v", before.OutcomeRevision, after.OutcomeRevision, err)
	}
}

func TestAutoObjectiveAchievementRequiresOperationalV2AndStrictWork(t *testing.T) {
	for _, test := range []struct {
		name       string
		profile    d.OperationalProfile
		completion d.CompletionPolicy
	}{
		{name: "compatibility-profile", profile: d.OperationalProfileCompatibilityV1, completion: d.CompletionPolicyStrictWork},
		{name: "compatible-completion-policy", profile: d.OperationalProfileOperationalV2, completion: d.CompletionPolicyCompatible},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newOperationalProfileFixture(t)
			setCompletionOutcomeProfile(t, f, test.profile, test.completion)
			objective := createObjectiveFixture(t, f.Service, f.Command, f.Scope, "Policy-gated Objective", d.ObjectiveLifecycleInProgress)
			human := f.Credentials["operator_human"]
			grantObjectiveCompletionPermission(t, f, human, true)
			ctx, cc, cmd := completionModeRequest(t, f, objective, human, "auto-completion-policy-"+test.name, d.ObjectiveCompletionModeAutomatic)
			before, err := f.Service.GetOutcomeState(f.Context, f.Scope)
			if err != nil {
				t.Fatal(err)
			}
			eventsBefore := f.Store.SnapshotDomainEvents(f.Scope)
			receiptsBefore := f.Store.IdempotencyRecordCount()
			if _, err := f.Service.SetObjectiveCompletionMode(ctx, cc, cmd); err == nil {
				t.Fatal("automatic completion delegation succeeded without operational_v2 and strict_work")
			}
			after, err := f.Service.GetOutcomeState(f.Context, f.Scope)
			if err != nil {
				t.Fatal(err)
			}
			stored, err := f.Service.GetObjective(f.Context, f.Scope, objective.ID)
			if err != nil {
				t.Fatal(err)
			}
			if after.OutcomeRevision != before.OutcomeRevision || stored.Value.CompletionMode.Effective() != d.ObjectiveCompletionModeManual || !reflect.DeepEqual(eventsBefore, f.Store.SnapshotDomainEvents(f.Scope)) || receiptsBefore != f.Store.IdempotencyRecordCount() {
				t.Fatal("rejected automatic completion policy changed the Objective, Outcome, events, or receipts")
			}
		})
	}
}

type objectiveAutoCommitFailureManager struct{ inner ports.TransactionManager }

func (m objectiveAutoCommitFailureManager) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	uow, err := m.inner.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return objectiveAutoCommitFailureUOW{UnitOfWork: uow}, nil
}

type objectiveAutoCommitFailureUOW struct{ ports.UnitOfWork }

func (u objectiveAutoCommitFailureUOW) Commit() error { return errors.New("injected commit failure") }

func TestAutoObjectiveAchievementCommitFailureRollsBackPrimaryAndDerivedWrites(t *testing.T) {
	f := newObjectiveAutoFixture(t)
	attestObjectiveAutoCriterion(t, f)
	// Prepare the native delivery before injecting a commit failure into the
	// final acceptance transaction.
	agentCtx := objectiveAutoIdentityContext(f.workspace.Context, f.agent)
	claim, err := f.workspace.Service.ClaimWorkItem(agentCtx, objectiveAutoCommandContext(f.workspace, f.agent, "auto-completion-rollback-claim"), a.ClaimWorkItemCommand{
		Scope: f.workspace.Scope, WorkItemID: f.task.ID, ExpectedVersion: f.task.Version, TTL: d.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	lease := claim.Value.CurrentLease
	submission, err := f.workspace.Service.SubmitTaskForReview(agentCtx, objectiveAutoCommandContext(f.workspace, f.agent, "auto-completion-rollback-submit"), a.SubmitTaskForReviewCommand{
		Scope: f.workspace.Scope, WorkItemID: f.task.ID, ExpectedWorkItemVersion: claim.Value.Version,
		ClaimID: &lease.ClaimID, FencingToken: lease.FencingToken,
		LegacyMaterial: ptrTaskDeliveryMaterial(d.TaskDeliveryMaterial{Summary: "Ready for transactional acceptance."}),
	})
	if err != nil {
		t.Fatal(err)
	}
	currentWork, assessmentRefs := assessObjectiveAutoTaskDelivery(t, f, f.workspace.Service, f.task.ID, submission.Value, "auto-completion-rollback-assessment")
	acceptCC := objectiveAutoCommandContext(f.workspace, f.human, "auto-completion-rollback-accept")
	acceptCommand := a.AcceptTaskDeliveryCommand{
		Scope: f.workspace.Scope, WorkItemID: f.task.ID, ExpectedWorkItemVersion: currentWork.Version,
		DeliveryID: submission.Value.Delivery.ID, ExpectedDeliveryDigest: submission.Value.Delivery.Digest,
		ExpectedDeliveryRound: submission.Value.Delivery.Round, Reason: "Accept after independent verification.", AssessmentRefs: assessmentRefs,
	}
	beforeState, err := f.workspace.Service.GetOutcomeState(f.workspace.Context, f.workspace.Scope)
	if err != nil {
		t.Fatal(err)
	}
	beforeObjective, err := f.workspace.Service.GetObjective(f.workspace.Context, f.workspace.Scope, f.objective.ID)
	if err != nil {
		t.Fatal(err)
	}
	beforeWork, err := f.workspace.Service.GetWorkItem(f.workspace.Context, f.workspace.Scope, f.task.ID)
	if err != nil {
		t.Fatal(err)
	}
	beforeEvents := f.workspace.Store.SnapshotDomainEvents(f.workspace.Scope)
	beforeReceipts := f.workspace.Store.IdempotencyRecordCount()
	failingService, err := a.NewService(objectiveAutoCommitFailureManager{inner: f.workspace.Store}, fixedClock{now: time.Date(2026, 10, 1, 21, 30, 0, 0, time.UTC)}, &sequenceIDs{next: 70000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := failingService.AcceptTaskDelivery(objectiveAutoIdentityContext(f.workspace.Context, f.human), acceptCC, acceptCommand); err == nil {
		t.Fatal("acceptance reported success when the combined transaction commit failed")
	}
	afterState, err := f.workspace.Service.GetOutcomeState(f.workspace.Context, f.workspace.Scope)
	if err != nil {
		t.Fatal(err)
	}
	afterObjective, err := f.workspace.Service.GetObjective(f.workspace.Context, f.workspace.Scope, f.objective.ID)
	if err != nil {
		t.Fatal(err)
	}
	afterWork, err := f.workspace.Service.GetWorkItem(f.workspace.Context, f.workspace.Scope, f.task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterObjective.Value.Lifecycle != beforeObjective.Value.Lifecycle || afterObjective.Value.Version != beforeObjective.Value.Version || afterWork.Value.Lifecycle != beforeWork.Value.Lifecycle || afterWork.Value.Version != beforeWork.Value.Version || afterWork.Value.PendingDeliveryID == nil || *afterWork.Value.PendingDeliveryID != submission.Value.Delivery.ID {
		t.Fatalf("failed combined commit partially changed Task or Objective: objective=%+v task=%+v", afterObjective.Value, afterWork.Value)
	}
	if afterState.OutcomeRevision != beforeState.OutcomeRevision || !reflect.DeepEqual(beforeEvents, f.workspace.Store.SnapshotDomainEvents(f.workspace.Scope)) || beforeReceipts != f.workspace.Store.IdempotencyRecordCount() {
		t.Fatal("failed combined commit changed Outcome revision, events, or idempotency receipts")
	}
	// A normal retry with the same command succeeds, proving the rollback did
	// not leave a terminal acceptance receipt or a partial Objective write.
	retried, err := f.workspace.Service.AcceptTaskDelivery(objectiveAutoIdentityContext(f.workspace.Context, f.human), acceptCC, acceptCommand)
	if err != nil || len(retried.DerivedObjectiveAchievements) != 1 {
		t.Fatalf("acceptance retry after failed commit = %+v err=%v", retried, err)
	}
}

func assertAutoObjectiveAchievement(
	t *testing.T,
	f objectiveAutoFixture,
	acceptance objectiveAutoAcceptance,
	assessment d.CriterionAssessment,
	priorEvents []d.DomainEvent,
) {
	t.Helper()
	result := acceptance.result
	if result.OutcomeRevision != acceptance.outcomeRevisionBefore+1 {
		t.Fatalf("Task acceptance plus derived achievement advanced revision %d -> %d, want one revision", acceptance.outcomeRevisionBefore, result.OutcomeRevision)
	}
	objective, err := f.workspace.Service.GetObjective(f.workspace.Context, f.workspace.Scope, f.objective.ID)
	if err != nil {
		t.Fatal(err)
	}
	if objective.Value.Lifecycle != d.ObjectiveLifecycleAchieved || objective.Value.CurrentConclusion == nil || objective.Value.CurrentConclusion.DerivedCompletion == nil {
		t.Fatalf("eligible delegated Objective was not derived: %+v", objective.Value)
	}
	conclusion := objective.Value.CurrentConclusion
	if conclusion.DerivedCompletion.AuthorityID.IsZero() || conclusion.DerivedCompletion.DelegatorCredentialID != f.human.ID || conclusion.DerivedCompletion.PolicyVersion == 0 || conclusion.DerivedCompletion.CredentialPolicyRevision == 0 {
		t.Fatalf("derived conclusion provenance does not identify its live delegation: %+v", conclusion.DerivedCompletion)
	}
	if len(conclusion.Assessments) != 1 || conclusion.Assessments[0] != (d.CriterionAssessmentRef{
		AssessmentID: assessment.ID, CriterionID: assessment.CriterionID,
		CriterionRevision: assessment.CriterionRevision, Result: assessment.Result,
	}) {
		t.Fatalf("derived conclusion did not reference only the real current assessment: %+v", conclusion.Assessments)
	}
	state, err := f.workspace.Service.GetOutcomeState(f.workspace.Context, f.workspace.Scope)
	if err != nil || state.Outcome.Lifecycle != d.OutcomeLifecycleActive || state.OutcomeRevision != result.OutcomeRevision {
		t.Fatalf("derived Objective achievement affected Outcome lifecycle/revision: state=%+v err=%v", state, err)
	}
	assertDerivedEventsCausal(t, f.workspace.Store.SnapshotDomainEvents(f.workspace.Scope), result.CommandID, priorEvents)
}

func assertDerivedEventsCausal(t *testing.T, all []d.DomainEvent, commandID d.ID, prior []d.DomainEvent) {
	t.Helper()
	priorIDs := make(map[d.ID]bool, len(prior))
	for _, event := range prior {
		priorIDs[event.EventID] = true
	}
	commandEvents := make([]d.DomainEvent, 0)
	for _, event := range all {
		if event.CommandID == commandID {
			commandEvents = append(commandEvents, event)
		}
	}
	causes := make(map[d.ID]bool)
	var derived int
	for _, event := range commandEvents {
		if event.AggregateRef.Kind == d.EntityKindObjective {
			if event.EventType != "objective.achieved" && event.EventType != "objective.conclusion_recorded" {
				t.Fatalf("unexpected derived Objective event type %q", event.EventType)
			}
			if event.CausationID == nil || event.OutcomeRevision == 0 || priorIDs[event.EventID] {
				t.Fatalf("derived event lacks same-command causation or is duplicated: %+v", event)
			}
			derived++
		} else {
			causes[event.EventID] = true
		}
	}
	if derived != 2 {
		t.Fatalf("derived Objective events for command %s = %d, want 2", commandID, derived)
	}
	for _, event := range commandEvents {
		if event.AggregateRef.Kind == d.EntityKindObjective && !causes[*event.CausationID] {
			t.Fatalf("derived Objective event causation %s does not name a primary event in its command", *event.CausationID)
		}
	}
}
