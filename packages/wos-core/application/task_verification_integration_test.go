package application_test

import (
	"context"
	"testing"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func assertNativeVerificationReadParity(t *testing.T, f credentialLeaseFixture, ctx context.Context, delivery d.TaskDelivery, expectedStage string, pending bool) a.TaskVerificationView {
	t.Helper()
	beforeEvents, beforeReceipts, beforeRevision := len(f.store.SnapshotDomainEvents(f.scope)), f.store.IdempotencyRecordCount(), readTaskDeliveryRevision(t, f)
	view, err := f.service.GetTaskVerificationView(ctx, f.scope, f.work.ID)
	if err != nil {
		t.Fatalf("verification query: %v", err)
	}
	if view.SourceKind != "task_delivery" || view.SourceID == nil || *view.SourceID != delivery.ID || view.DeliveryDigest != delivery.Digest || view.Round != delivery.Round || view.ReviewCaseVersion != 0 || view.Stage != expectedStage || view.Pending != pending {
		t.Fatalf("verification source mismatch: %+v", view)
	}
	cards, err := f.service.GetWorkspaceCards(ctx, f.scope, a.WorkspaceCardsQuery{Refs: []d.EntityRef{f.work.Ref()}})
	if err != nil || len(cards.Items) != 1 {
		t.Fatalf("workspace cards: %+v, %v", cards, err)
	}
	card := cards.Items[0]
	if card.Verification == nil || card.Stage != view.Stage || card.Verification.Pending != view.Pending || card.Verification.Round != view.Round || card.Verification.DeliveryDigest != view.DeliveryDigest {
		t.Fatalf("card/source disagreement: %+v vs %+v", card, view)
	}
	inspector, err := f.service.GetEntityWorkspaceView(ctx, f.scope, a.WorkspaceViewQuery{Subject: a.WorkspaceSubject{Kind: "entity", Ref: f.work.Ref()}})
	if err != nil {
		t.Fatalf("workspace inspector: %v", err)
	}
	attention, err := f.service.GetWorkAttention(ctx, f.scope, f.work.ID)
	if err != nil {
		t.Fatalf("attention query: %v", err)
	}
	operational, err := f.service.GetWorkItemOperationalState(ctx, f.scope, f.work.ID)
	if err != nil {
		t.Fatalf("operational query: %v", err)
	}
	for name, value := range map[string]*a.TaskVerificationView{"inspector": inspector.Verification, "attention": attention.Verification, "operational": operational.Verification} {
		if value == nil || value.SourceID == nil || *value.SourceID != delivery.ID || value.Stage != view.Stage || value.Pending != pending || value.Round != delivery.Round || value.DeliveryDigest != delivery.Digest {
			t.Fatalf("%s source disagreement: %+v", name, value)
		}
	}
	if pending && (attention.Status != "awaiting_review" || !attention.AwaitingReview || attention.ExecutionAllowed || string(operational.State.DisplayState) != "awaiting_review" || operational.State.LeaseStatus != d.LeaseStatusNone) {
		t.Fatalf("pending delivery looks executable/expired: attention=%+v operational=%+v", attention, operational)
	}
	if pending {
		ready, err := f.service.ListReadyWork(ctx, f.scope)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range ready.Items {
			if item.WorkItem.ID == f.work.ID {
				t.Fatal("pending native review advertised as ready execution")
			}
		}
	}
	if len(f.store.SnapshotDomainEvents(f.scope)) != beforeEvents || f.store.IdempotencyRecordCount() != beforeReceipts || readTaskDeliveryRevision(t, f) != beforeRevision {
		t.Fatal("verification reads wrote authority, events, revision or receipts")
	}
	return view
}

func TestNativeTaskVerificationReadsAgreeAcrossPendingCorrectionsAndAcceptance(t *testing.T) {
	f, executor, criterion := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	first := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "Initial result"}, "e34-first-delivery")
	assertNativeVerificationReadParity(t, f, f.identityContext(executor), first.Delivery, "verification", true)
	reviewer, reviewCC := newReviewPrincipal(t, f, "e34-round-reviewer", "e34-reviewer")
	assessment := recordTaskCriterionAssessment(t, f, reviewer, reviewCC, criterion, first.Delivery, first.WorkItem.Version, nil, nil, "e34-first-assessment")
	view := assertNativeVerificationReadParity(t, f, f.identityContext(reviewer), first.Delivery, "verification", true)
	if view.Requirements == nil || view.Requirements.Met != 1 || view.Requirements.Pending != 0 {
		t.Fatalf("current exact-round assessment not applicable: %+v", view.Requirements)
	}
	changed, err := f.service.RequestTaskDeliveryChanges(f.identityContext(reviewer), freshCommandContext(reviewCC, "e34-request-changes"), a.RequestTaskDeliveryChangesCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: assessment.WorkVersion, DeliveryID: first.Delivery.ID, ExpectedDeliveryDigest: first.Delivery.Digest, ExpectedDeliveryRound: first.Delivery.Round, Reason: "Correction required", Findings: []string{"Clarify the result"}})
	if err != nil {
		t.Fatal(err)
	}
	view = assertNativeVerificationReadParity(t, f, f.identityContext(reviewer), first.Delivery, "todo", false)
	if !view.CorrectionRequired || view.Current || view.Requirements == nil || view.Requirements.Met != 0 {
		t.Fatalf("changes-requested history remained applicable: %+v", view)
	}
	claimed, err := f.service.ClaimWorkItem(f.identityContext(executor), freshCommandContext(f.cc, "e34-correction-claim"), a.ClaimWorkItemCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: changed.Value.WorkItem.Version, TTL: d.DefaultLeaseTTL})
	if err != nil {
		t.Fatal(err)
	}
	f.work = claimed.Value
	view = assertNativeVerificationReadParity(t, f, f.identityContext(executor), first.Delivery, "in_progress", false)
	if view.ExecutorAuthority != "active" {
		t.Fatalf("correction authority missing: %+v", view)
	}
	second := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "Corrected result"}, "e34-second-delivery")
	view = assertNativeVerificationReadParity(t, f, f.identityContext(reviewer), second.Delivery, "verification", true)
	if view.Requirements == nil || view.Requirements.Met != 0 || view.Requirements.Pending != 1 {
		t.Fatalf("old-round assessment inherited by corrected delivery: %+v", view.Requirements)
	}
	assessment = recordTaskCriterionAssessment(t, f, reviewer, reviewCC, criterion, second.Delivery, second.WorkItem.Version, nil, nil, "e34-second-assessment")
	accepted, err := f.service.AcceptTaskDelivery(f.identityContext(reviewer), freshCommandContext(reviewCC, "e34-accept-current-round"), makeTaskDeliveryAcceptCommand(f, second.Delivery, assessment.WorkVersion, assessment.Assessment, "Accept exact corrected result"))
	if err != nil {
		t.Fatal(err)
	}
	f.work = accepted.Value.WorkItem
	view = assertNativeVerificationReadParity(t, f, f.identityContext(reviewer), second.Delivery, "accepted", false)
	if !view.Current || view.Status != "done" {
		t.Fatalf("accepted actual source not current: %+v", view)
	}
}

func TestContractsV1PendingNativeVerificationIsNotAvailableOrRecoverableExecution(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	markWorkContractEnabled(t, &f)
	_, executor := f.addCredentialTree(t, "0199f2e0-0000-7000-8000-000000000001", "0199f2e1-0000-7000-8000-000000000001", "e34-contract-root", "e34-contract-session", f.cc.PrincipalID, f.cc.Actor)
	ctx := f.identityContext(executor)
	acquired, err := f.service.AcquireWorkContract(ctx, freshCommandContext(f.cc, "e34-contract-acquire"), a.AcquireWorkContractCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: f.work.Version, TTLSeconds: 60})
	if err != nil {
		t.Fatal(err)
	}
	c := acquired.Value.Contract
	submitted, err := f.service.SubmitTaskForReview(ctx, freshCommandContext(f.cc, "e34-contract-submit"), a.SubmitTaskForReviewCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: acquired.Value.WorkItem.Version, ContractID: &c.ID, Authority: &a.ContractAuthority{ExecutionID: c.ExecutionID, FencingToken: c.FencingToken, SpecDigest: c.SpecDigest}, ExpectedContractVersion: &c.Version, SubmissionMaterial: &d.WorkResultMaterial{ContractID: c.ID, WorkItemID: f.work.ID, SpecDigest: c.SpecDigest, Summary: "Exact contract result awaiting review"}})
	if err != nil {
		t.Fatal(err)
	}
	view := assertNativeVerificationReadParity(t, f, ctx, submitted.Value.Delivery, "verification", true)
	if view.Protocol != string(d.WorkProtocolContracts) || view.SubmissionID == nil || submitted.Value.Submission == nil || *view.SubmissionID != submitted.Value.Submission.ID {
		t.Fatalf("contracts_v1 source lost exact submission: %+v", view)
	}
	available, err := f.service.ListAvailableWork(ctx, f.scope, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range available.Items {
		if item.Candidate.ID == f.work.ID {
			t.Fatal("pending native review advertised as available/recovery work")
		}
	}
	next, err := f.service.AcquireNextWorkContract(ctx, freshCommandContext(f.cc, "e34-next-must-not-recover-review"), a.AcquireNextWorkContractCommand{Scope: f.scope, WorkItemIDs: []d.ID{f.work.ID}, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if next.Value.Acquired {
		t.Fatal("acquired pending native verification as execution recovery")
	}
	history, err := f.service.GetWorkContract(ctx, f.scope, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if history.Value.Recoverable || history.Value.ExecutionAllowed {
		t.Fatal("closed delivery authority advertised as recoverable/executable")
	}
}
