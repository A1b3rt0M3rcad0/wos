package domain_test

import (
	"strings"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func taskDeliveryFixture(t *testing.T, now time.Time) (d.WorkItem, d.ID, d.ID, d.ActorRef) {
	t.Helper()
	scope := d.Scope{
		NamespaceID: d.MustParseID("01a11770-0000-7000-8000-000000000001"),
		OutcomeID:   d.MustParseID("01a11770-0000-7000-8000-000000000002"),
	}
	work, err := d.NewWorkItem(d.MustParseID("01a11770-0000-7000-8000-000000000003"), scope, "Deliver a verified result", "Task delivery fixture", d.PriorityNormal, d.WorkItemLifecycleBacklog, now)
	if err != nil {
		t.Fatal(err)
	}
	criterion, err := d.NewSuccessCriterion(d.MustParseID("01a11770-0000-7000-8000-000000000004"), work.Ref(), "Result is reviewed", "A reviewer confirms the result", true, d.VerificationModeAttestation)
	if err != nil {
		t.Fatal(err)
	}
	if err := work.AddCriterion(criterion, now); err != nil {
		t.Fatal(err)
	}
	if err := work.Activate(now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	credentialID := d.MustParseID("01a11770-0000-7000-8000-000000000005")
	actor := d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}
	return work, criterion.ID, credentialID, actor
}

func claimTaskDeliveryFixture(t *testing.T, work *d.WorkItem, credentialID d.ID, actor d.ActorRef, now time.Time, claimID d.ID) {
	t.Helper()
	sessionID := d.MustParseID("01a11770-0000-7000-8000-000000000006")
	if err := work.ClaimForCredential(claimID, "executor", credentialID, &sessionID, actor, time.Minute, now); err != nil {
		t.Fatal(err)
	}
}

func taskDeliveryDecisionFixture(t *testing.T, delivery d.TaskDelivery, id d.ID, result d.TaskDeliveryDecisionResult, now time.Time) d.TaskDeliveryDecision {
	t.Helper()
	actor := d.ActorRef{Kind: d.ActorKindHuman, Provider: "local", ID: "reviewer"}
	reason := "Reviewed the exact delivery"
	findings := []string(nil)
	if result == d.TaskDeliveryDecisionChangesRequested {
		findings = []string{"Include the missing check"}
	}
	decision, err := d.NewTaskDeliveryDecision(id, delivery, result, "reviewer", actor, reason, findings, now)
	if err != nil {
		t.Fatal(err)
	}
	return decision
}

func taskDeliveryConclusion(decision d.TaskDeliveryDecision) d.Conclusion {
	return d.Conclusion{
		ID:          d.MustParseID("01a11770-0000-7000-8000-000000000009"),
		PrincipalID: decision.ReviewerPrincipalID,
		Actor:       decision.ReviewerActor,
		Reason:      decision.Reason,
		ConcludedAt: decision.DecidedAt,
	}
}

func TestLegacyTaskDeliveryClosesBoundLeaseAndAllowsReviewWithoutExecutorAuthority(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	work, criterionID, credentialID, actor := taskDeliveryFixture(t, now)
	claimTaskDeliveryFixture(t, &work, credentialID, actor, now.Add(2*time.Second), d.MustParseID("01a11770-0000-7000-8000-000000000007"))
	deliveryAt := now.Add(3 * time.Second)
	delivery, err := d.NewLegacyTaskDelivery(d.MustParseID("01a11770-0000-7000-8000-000000000008"), work, d.TaskDeliveryMaterial{Summary: "The requested result is ready"}, "executor", &credentialID, deliveryAt)
	if err != nil {
		t.Fatal(err)
	}
	before := work.Version
	if err := work.RecordTaskDelivery(delivery, nil, nil, before, deliveryAt); err != nil {
		t.Fatal(err)
	}
	if work.CurrentLease != nil || work.PendingDeliveryID == nil || *work.PendingDeliveryID != delivery.ID || work.Lifecycle != d.WorkItemLifecycleInProgress {
		t.Fatal("delivery did not close executor authority while preserving review state")
	}
	if err := work.Validate(); err != nil {
		t.Fatalf("pending delivery without active lease should be valid: %v", err)
	}
	decision := taskDeliveryDecisionFixture(t, delivery, d.MustParseID("01a11770-0000-7000-8000-000000000010"), d.TaskDeliveryDecisionChangesRequested, deliveryAt.Add(time.Second))
	if err := work.RequestTaskDeliveryChanges(delivery, decision, work.Version, decision.DecidedAt); err != nil {
		t.Fatal(err)
	}
	if work.Lifecycle != d.WorkItemLifecycleTodo || work.PendingDeliveryID != nil || work.CurrentLease != nil || work.LatestDeliveryDecisionID == nil || *work.LatestDeliveryDecisionID != decision.ID {
		t.Fatal("changes requested did not return work to To do without restoring the closed lease")
	}
	if err := work.RequestTaskDeliveryChanges(delivery, decision, work.Version, decision.DecidedAt); err == nil {
		t.Fatal("duplicate decision was applied after the pending delivery closed")
	}
	if _, err := work.Criteria.Find(criterionID); err != nil {
		t.Fatal(err)
	}
}

func TestAcceptedTaskDeliveryUsesSuppliedAssessmentRefsAndConclusion(t *testing.T) {
	now := time.Date(2026, 10, 21, 13, 0, 0, 0, time.UTC)
	work, criterionID, credentialID, actor := taskDeliveryFixture(t, now)
	claimTaskDeliveryFixture(t, &work, credentialID, actor, now.Add(2*time.Second), d.MustParseID("01a11771-0000-7000-8000-000000000007"))
	deliveryAt := now.Add(3 * time.Second)
	delivery, err := d.NewLegacyTaskDelivery(d.MustParseID("01a11771-0000-7000-8000-000000000008"), work, d.TaskDeliveryMaterial{Summary: "Reviewed output"}, "executor", &credentialID, deliveryAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := work.RecordTaskDelivery(delivery, nil, nil, work.Version, deliveryAt); err != nil {
		t.Fatal(err)
	}
	decision := taskDeliveryDecisionFixture(t, delivery, d.MustParseID("01a11771-0000-7000-8000-000000000010"), d.TaskDeliveryDecisionAccepted, deliveryAt.Add(time.Second))
	refs := []d.CriterionAssessmentRef{{
		AssessmentID:      d.MustParseID("01a11771-0000-7000-8000-000000000011"),
		CriterionID:       criterionID,
		CriterionRevision: d.InitialCriterionRevision,
		Result:            d.AssessmentResultMet,
	}}
	missingRefs := taskDeliveryConclusion(decision)
	if err := work.AcceptTaskDelivery(delivery, decision, nil, nil, work.Version, missingRefs, decision.DecidedAt); err == nil {
		t.Fatal("accepted a delivery without all required criterion assessment references")
	}
	notMet := append([]d.CriterionAssessmentRef(nil), refs...)
	notMet[0].Result = d.AssessmentResultInconclusive
	if err := work.AcceptTaskDelivery(delivery, decision, nil, notMet, work.Version, taskDeliveryConclusion(decision), decision.DecidedAt); err == nil {
		t.Fatal("accepted a non-met required criterion assessment reference")
	}
	if err := work.AcceptTaskDelivery(delivery, decision, nil, refs, work.Version, taskDeliveryConclusion(decision), decision.DecidedAt); err != nil {
		t.Fatal(err)
	}
	if work.Lifecycle != d.WorkItemLifecycleDone || work.PendingDeliveryID != nil || work.CurrentLease != nil || work.CurrentConclusion == nil || len(work.CurrentConclusion.Assessments) != 1 || work.CurrentConclusion.Assessments[0].AssessmentID != refs[0].AssessmentID {
		t.Fatal("accepted delivery did not create the distinct Task conclusion")
	}
	if err := work.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyTaskDeliveryEvidenceLinksAreUniqueAndBoundToDeliveryEvidence(t *testing.T) {
	now := time.Date(2026, 10, 21, 13, 30, 0, 0, time.UTC)
	work, _, credentialID, actor := taskDeliveryFixture(t, now)
	claimTaskDeliveryFixture(t, &work, credentialID, actor, now.Add(2*time.Second), d.MustParseID("01a11775-0000-7000-8000-000000000007"))
	evidenceID := d.MustParseID("01a11775-0000-7000-8000-000000000030")
	validMaterial := d.TaskDeliveryMaterial{
		Summary: "Evidence-bound result", EvidenceIDs: []d.ID{evidenceID},
		CriterionEvidence: []d.SubmissionCriterionEvidence{{CriterionID: d.MustParseID("01a11770-0000-7000-8000-000000000004"), CriterionRevision: d.InitialCriterionRevision, EvidenceIDs: []d.ID{evidenceID}}},
	}
	if _, err := d.NewLegacyTaskDelivery(d.MustParseID("01a11775-0000-7000-8000-000000000008"), work, validMaterial, "executor", &credentialID, now.Add(3*time.Second)); err != nil {
		t.Fatalf("valid criterion evidence link was rejected: %v", err)
	}
	for _, tc := range []struct {
		name     string
		material d.TaskDeliveryMaterial
	}{
		{name: "criterion evidence missing from delivery", material: d.TaskDeliveryMaterial{
			Summary: "Evidence-bound result", EvidenceIDs: []d.ID{d.MustParseID("01a11775-0000-7000-8000-000000000020")},
			CriterionEvidence: []d.SubmissionCriterionEvidence{{CriterionID: d.MustParseID("01a11770-0000-7000-8000-000000000004"), CriterionRevision: d.InitialCriterionRevision, EvidenceIDs: []d.ID{d.MustParseID("01a11775-0000-7000-8000-000000000021")}}},
		}},
		{name: "duplicate evidence on criterion", material: d.TaskDeliveryMaterial{
			Summary: "Evidence-bound result", EvidenceIDs: []d.ID{d.MustParseID("01a11775-0000-7000-8000-000000000022")},
			CriterionEvidence: []d.SubmissionCriterionEvidence{{CriterionID: d.MustParseID("01a11770-0000-7000-8000-000000000004"), CriterionRevision: d.InitialCriterionRevision, EvidenceIDs: []d.ID{d.MustParseID("01a11775-0000-7000-8000-000000000022"), d.MustParseID("01a11775-0000-7000-8000-000000000022")}}},
		}},
		{name: "duplicate delivery evidence", material: d.TaskDeliveryMaterial{
			Summary: "Evidence-bound result", EvidenceIDs: []d.ID{d.MustParseID("01a11775-0000-7000-8000-000000000023"), d.MustParseID("01a11775-0000-7000-8000-000000000023")},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			work, _, credentialID, actor := taskDeliveryFixture(t, now)
			claimTaskDeliveryFixture(t, &work, credentialID, actor, now.Add(2*time.Second), d.MustParseID("01a11775-0000-7000-8000-000000000007"))
			if _, err := d.NewLegacyTaskDelivery(d.MustParseID("01a11775-0000-7000-8000-000000000008"), work, tc.material, "executor", &credentialID, now.Add(3*time.Second)); err == nil {
				t.Fatal("accepted invalid criterion evidence references")
			}
		})
	}
}

func TestLegacyTaskDeliveryWithNoActiveCriteriaKeepsCanonicalEmptySnapshot(t *testing.T) {
	now := time.Date(2026, 10, 21, 13, 45, 0, 0, time.UTC)
	scope := d.Scope{
		NamespaceID: d.MustParseID("01a11776-0000-7000-8000-000000000001"),
		OutcomeID:   d.MustParseID("01a11776-0000-7000-8000-000000000002"),
	}
	work, err := d.NewWorkItem(d.MustParseID("01a11776-0000-7000-8000-000000000003"), scope, "Task without criteria", "Legacy compatibility", d.PriorityNormal, d.WorkItemLifecycleBacklog, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := work.Activate(now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	credentialID := d.MustParseID("01a11776-0000-7000-8000-000000000004")
	claimID := d.MustParseID("01a11776-0000-7000-8000-000000000005")
	sessionID := d.MustParseID("01a11776-0000-7000-8000-000000000006")
	actor := d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}
	if err := work.ClaimForCredential(claimID, "executor", credentialID, &sessionID, actor, time.Minute, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	delivery, err := d.NewLegacyTaskDelivery(d.MustParseID("01a11776-0000-7000-8000-000000000007"), work, d.TaskDeliveryMaterial{Summary: "Result without criteria"}, "executor", &credentialID, now.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if delivery.CriterionSnapshot == nil || len(delivery.CriterionSnapshot) != 0 {
		t.Fatalf("empty criterion snapshot = %#v, want a non-nil empty slice", delivery.CriterionSnapshot)
	}
	if err := delivery.ValidateForWork(work); err != nil {
		t.Fatalf("delivery with no active criteria did not match its Task: %v", err)
	}
}

func TestTaskDeliveryCorrectionRoundSupersedesPriorDeliveryAndRequiresFreshAuthority(t *testing.T) {
	now := time.Date(2026, 10, 21, 14, 30, 0, 0, time.UTC)
	work, _, credentialID, actor := taskDeliveryFixture(t, now)
	claimTaskDeliveryFixture(t, &work, credentialID, actor, now.Add(2*time.Second), d.MustParseID("01a11777-0000-7000-8000-000000000007"))
	first, err := d.NewLegacyTaskDelivery(d.MustParseID("01a11777-0000-7000-8000-000000000008"), work, d.TaskDeliveryMaterial{Summary: "First immutable delivery"}, "executor", &credentialID, now.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if first.Round != 1 || first.SupersedesDeliveryID != nil {
		t.Fatalf("first delivery chain = round %d supersedes %v, want round 1 without predecessor", first.Round, first.SupersedesDeliveryID)
	}
	if err := work.RecordTaskDelivery(first, nil, nil, work.Version, first.DeliveredAt); err != nil {
		t.Fatal(err)
	}
	decision := taskDeliveryDecisionFixture(t, first, d.MustParseID("01a11777-0000-7000-8000-000000000010"), d.TaskDeliveryDecisionChangesRequested, first.DeliveredAt.Add(time.Second))
	if err := work.RequestTaskDeliveryChanges(first, decision, work.Version, decision.DecidedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := d.NewLegacyTaskDelivery(d.MustParseID("01a11777-0000-7000-8000-000000000011"), work, d.TaskDeliveryMaterial{Summary: "Cannot correct without reacquiring"}, "executor", &credentialID, decision.DecidedAt.Add(time.Second)); err == nil {
		t.Fatal("created a correction delivery without reacquiring execution authority")
	}
	if err := work.Defer(decision.DecidedAt.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	newCriterion, err := d.NewSuccessCriterion(d.MustParseID("01a11777-0000-7000-8000-000000000012"), work.Ref(), "Correction includes a regression check", "The corrected work includes a fresh regression check", true, d.VerificationModeAttestation)
	if err != nil {
		t.Fatal(err)
	}
	if err := work.AddCriterion(newCriterion, decision.DecidedAt.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := first.Validate(); err != nil {
		t.Fatalf("replanning mutated the previous immutable delivery: %v", err)
	}
	if err := first.ValidateForWork(work); err == nil {
		t.Fatal("previous delivery still matches the changed Task definition")
	}
	if err := work.Activate(decision.DecidedAt.Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	claimTaskDeliveryFixture(t, &work, credentialID, actor, decision.DecidedAt.Add(4*time.Second), d.MustParseID("01a11777-0000-7000-8000-000000000013"))
	second, err := d.NewLegacyTaskDelivery(d.MustParseID("01a11777-0000-7000-8000-000000000014"), work, d.TaskDeliveryMaterial{Summary: "Corrected immutable delivery"}, "executor", &credentialID, decision.DecidedAt.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if second.Round != first.Round+1 || second.SupersedesDeliveryID == nil || *second.SupersedesDeliveryID != first.ID {
		t.Fatalf("correction chain = round %d supersedes %v; want round %d superseding %s", second.Round, second.SupersedesDeliveryID, first.Round+1, first.ID)
	}

	wrongChain := work
	wrongPredecessor := d.MustParseID("01a11777-0000-7000-8000-000000000099")
	wrongChain.LatestDeliveryID = &wrongPredecessor
	if err := wrongChain.RecordTaskDelivery(second, nil, nil, wrongChain.Version, second.DeliveredAt); err == nil {
		t.Fatal("recorded a correction whose supersedes link differs from the WorkItem latest delivery")
	}
	if err := work.RecordTaskDelivery(second, nil, nil, work.Version, second.DeliveredAt); err != nil {
		t.Fatalf("record valid correction round: %v", err)
	}
}

func TestTaskDeliveryRoundRequiresAWellFormedSupersedesReference(t *testing.T) {
	now := time.Date(2026, 10, 21, 15, 0, 0, 0, time.UTC)
	work, _, credentialID, actor := taskDeliveryFixture(t, now)
	claimTaskDeliveryFixture(t, &work, credentialID, actor, now.Add(2*time.Second), d.MustParseID("01a11778-0000-7000-8000-000000000007"))
	delivery, err := d.NewLegacyTaskDelivery(d.MustParseID("01a11778-0000-7000-8000-000000000008"), work, d.TaskDeliveryMaterial{Summary: "First delivery"}, "executor", &credentialID, now.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	mutated := delivery
	mutated.SupersedesDeliveryID = ptrDomainID(d.MustParseID("01a11778-0000-7000-8000-000000000009"))
	if err := mutated.Validate(); err == nil {
		t.Fatal("round-one delivery accepted a supersedes reference")
	}
}

func ptrDomainID(id d.ID) *d.ID { return &id }

func TestTaskDeliveryRejectsUnboundAuthoritySignedProtocolAndChangedDefinitions(t *testing.T) {
	now := time.Date(2026, 10, 21, 14, 0, 0, 0, time.UTC)
	work, _, credentialID, actor := taskDeliveryFixture(t, now)
	claimTaskDeliveryFixture(t, &work, credentialID, actor, now.Add(2*time.Second), d.MustParseID("01a11772-0000-7000-8000-000000000007"))
	delivery, err := d.NewLegacyTaskDelivery(d.MustParseID("01a11772-0000-7000-8000-000000000008"), work, d.TaskDeliveryMaterial{Summary: "Result"}, "executor", &credentialID, now.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	changed := work
	changed.Title = "Different definition"
	if err := delivery.ValidateForWork(changed); err == nil {
		t.Fatal("delivery accepted a changed Task definition")
	}
	tampered := delivery
	tampered.Summary = "Edited after delivery"
	if err := tampered.Validate(); err == nil {
		t.Fatal("delivery accepted mutated material with the old digest")
	}
	dualMaterial := delivery
	dualMaterial.SubmissionRef = &d.TaskDeliverySubmissionRef{ID: d.MustParseID("01a11772-0000-7000-8000-000000000012"), ContractID: d.MustParseID("01a11772-0000-7000-8000-000000000013"), Digest: "sha256:" + strings.Repeat("a", 64)}
	if err := dualMaterial.Validate(); err == nil {
		t.Fatal("delivery accepted inline material and submission reference together")
	}
	signed := delivery
	signed.Authority.Kind = d.TaskDeliveryAuthoritySignedV2
	if err := signed.Validate(); err == nil {
		t.Fatal("unsigned TaskDelivery accepted signed-v2 authority")
	}
	decision := taskDeliveryDecisionFixture(t, delivery, d.MustParseID("01a11772-0000-7000-8000-000000000010"), d.TaskDeliveryDecisionChangesRequested, delivery.DeliveredAt.Add(time.Second))
	changedDecision := decision
	changedDecision.Round++
	if err := changedDecision.ValidateForDelivery(delivery); err == nil {
		t.Fatal("decision accepted a different delivery round")
	}
	mutatedDecision := decision
	mutatedDecision.Reason = "rewritten decision"
	if err := d.ValidateTaskDeliveryDecisionMutation(decision, mutatedDecision); err == nil {
		t.Fatal("stored delivery decision was mutable")
	}

	unbound, _, _, unboundActor := taskDeliveryFixture(t, now.Add(time.Hour))
	if err := unbound.Claim(d.MustParseID("01a11772-0000-7000-8000-000000000020"), "executor", unboundActor, time.Minute, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := d.NewLegacyTaskDelivery(d.MustParseID("01a11772-0000-7000-8000-000000000021"), unbound, d.TaskDeliveryMaterial{Summary: "Unbound"}, "executor", nil, now.Add(time.Hour+time.Second)); err == nil {
		t.Fatal("unbound legacy lease created a TaskDelivery")
	}
}

func TestWorkItemDoesNotAllowUnidentifiedInProgressWithoutAuthority(t *testing.T) {
	now := time.Date(2026, 10, 21, 14, 30, 0, 0, time.UTC)
	work, _, _, _ := taskDeliveryFixture(t, now)
	work.ContractsEnabled = true
	work.CurrentContractID = nil
	work.Lifecycle = d.WorkItemLifecycleInProgress
	if err := work.Validate(); err == nil {
		t.Fatal("in-progress Task without an authority or pending operation was accepted")
	}
}

func TestContractsV1TaskDeliveryClosesContractAndAcceptsExactSubmission(t *testing.T) {
	now := time.Date(2026, 10, 21, 15, 0, 0, 0, time.UTC)
	work, criterionID, credentialID, actor := taskDeliveryFixture(t, now)
	work.ContractsEnabled = true
	sessionID := d.MustParseID("01a11773-0000-7000-8000-000000000006")
	contractID := d.MustParseID("01a11773-0000-7000-8000-000000000007")
	executionID := d.MustParseID("01a11773-0000-7000-8000-000000000008")
	contract, err := d.NewWorkContractForCredential(contractID, executionID, work, "executor", credentialID, &sessionID, actor, d.DefaultContractLeasePolicy(), 60, d.WorkContractSpec{Title: work.Title, Description: work.Description, Criteria: append([]d.SuccessCriterion{}, work.Criteria.Items...)}, 1, now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := work.BindContract(contract, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	material := d.NormalizeResultMaterial(d.WorkResultMaterial{ContractID: contract.ID, WorkItemID: work.ID, SpecDigest: contract.SpecDigest, Summary: "Contract delivery output"})
	digest, err := d.SemanticDigest(material)
	if err != nil {
		t.Fatal(err)
	}
	submission := d.WorkSubmission{ID: d.MustParseID("01a11773-0000-7000-8000-000000000009"), Scope: work.Scope, Material: material, Digest: digest, PrincipalID: "executor", Actor: actor, SubmittedAt: now.Add(3 * time.Second)}
	if err := contract.SubmitForCredential(submission, "executor", credentialID, executionID, contract.FencingToken, contract.Version, now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	deliveryAt := now.Add(4 * time.Second)
	delivery, err := d.NewContractsV1TaskDelivery(d.MustParseID("01a11773-0000-7000-8000-000000000010"), work, contract, submission, "executor", &credentialID, deliveryAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := contract.DeliverV1ForCredential(submission, "executor", credentialID, executionID, contract.FencingToken, contract.Version, deliveryAt); err != nil {
		t.Fatal(err)
	}
	if contract.Status != d.ContractCompleted || work.Lifecycle != d.WorkItemLifecycleInProgress {
		t.Fatal("contracts_v1 delivery incorrectly accepted the Task")
	}
	if err := work.RecordTaskDelivery(delivery, &contract, &submission, work.Version, deliveryAt); err != nil {
		t.Fatal(err)
	}
	decision := taskDeliveryDecisionFixture(t, delivery, d.MustParseID("01a11773-0000-7000-8000-000000000011"), d.TaskDeliveryDecisionAccepted, deliveryAt.Add(time.Second))
	refs := []d.CriterionAssessmentRef{{AssessmentID: d.MustParseID("01a11773-0000-7000-8000-000000000012"), CriterionID: criterionID, CriterionRevision: d.InitialCriterionRevision, Result: d.AssessmentResultMet}}
	if err := work.AcceptTaskDelivery(delivery, decision, &submission, refs, work.Version, taskDeliveryConclusion(decision), decision.DecidedAt); err != nil {
		t.Fatal(err)
	}
	if work.Lifecycle != d.WorkItemLifecycleDone || work.CurrentConclusion == nil || work.CurrentConclusion.SubmissionID == nil || *work.CurrentConclusion.SubmissionID != submission.ID || work.CurrentConclusion.SubmissionDigest != submission.Digest {
		t.Fatal("Task Conclusion did not retain the exact accepted contracts_v1 submission")
	}
}
