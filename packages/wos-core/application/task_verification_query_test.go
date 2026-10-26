package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func verificationTestID(n byte) d.ID {
	return d.ID(fmt.Sprintf("019a4500-0000-7000-8000-%012d", n))
}
func verificationTestScope() d.Scope {
	return d.Scope{NamespaceID: verificationTestID(1), OutcomeID: verificationTestID(2)}
}
func verificationTestActor() d.ActorRef {
	return d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}
}
func verificationTestDigest(c string) string { return "sha256:" + strings.Repeat(c, 64) }

func verificationNativeFixture(at time.Time) ports.TaskVerificationFacts {
	scope, deliveryID, claimID, credentialID, predecessorID := verificationTestScope(), verificationTestID(4), verificationTestID(5), verificationTestID(6), verificationTestID(19)
	return ports.TaskVerificationFacts{WorkItemID: verificationTestID(3), Version: 7, Lifecycle: d.WorkItemLifecycleInProgress, PendingDeliveryID: &deliveryID, LatestDeliveryID: &deliveryID, DeliveryRound: 3,
		NativeDelivery: &ports.TaskVerificationDeliveryFacts{ID: deliveryID, Scope: scope, WorkItemID: verificationTestID(3), Round: 3, SupersedesDeliveryID: &predecessorID, Digest: verificationTestDigest("a"), DefinitionDigest: verificationTestDigest("b"), Summary: "An exact delivery", DeliveredAt: at.Add(-time.Minute), ArtifactCount: 21, EvidenceCount: 25, CriterionCount: 2,
			Authority: d.TaskDeliveryAuthority{Kind: d.TaskDeliveryAuthorityLegacyLease, PrincipalID: "executor", Actor: verificationTestActor(), ClaimID: &claimID, CredentialID: &credentialID, FencingToken: 2}}}
}

func verificationSignedFixture(at time.Time) ports.TaskVerificationFacts {
	scope, id := verificationTestScope(), verificationTestID(7)
	r := &d.ReviewCase{ID: id, Scope: scope, WorkItemID: verificationTestID(3), WorkContractID: verificationTestID(8), SubmissionID: verificationTestID(9), SubmissionDigest: verificationTestDigest("a"), IssuedSpecDigest: verificationTestDigest("b"), PolicyRevision: 1, AcceptanceFloor: d.AcceptanceIndependentReview, Round: 4, Version: 8, Status: d.ReviewPending, ExecutionPrincipals: []string{"executor"}, CreatedAt: at.Add(-time.Minute), UpdatedAt: at.Add(-time.Minute)}
	return ports.TaskVerificationFacts{WorkItemID: r.WorkItemID, Version: 7, Lifecycle: d.WorkItemLifecycleInProgress, ContractsEnabled: true, PendingReviewCaseID: &id, LatestReviewCaseID: &id, SignedReview: r}
}

func verificationDecision(f *ports.TaskVerificationFacts, result d.TaskDeliveryDecisionResult, at time.Time) {
	id := verificationTestID(10)
	f.PendingDeliveryID, f.LatestDeliveryDecisionID = nil, &id
	f.NativeDecision = &d.TaskDeliveryDecision{ID: id, Scope: verificationTestScope(), WorkItemID: f.WorkItemID, DeliveryID: f.NativeDelivery.ID, DeliveryDigest: f.NativeDelivery.Digest, Round: f.NativeDelivery.Round, Result: result, ReviewerPrincipalID: "reviewer", ReviewerActor: d.ActorRef{Kind: d.ActorKindHuman, Provider: "test", ID: "reviewer"}, Reason: "Reviewed exact round", DecidedAt: at.Add(-time.Second)}
	if result == d.TaskDeliveryDecisionChangesRequested {
		f.NativeDecision.Findings = []string{"Correct the output"}
	}
}

func TestTaskVerificationActualSourceStageMatrix(t *testing.T) {
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name                  string
		fixture               func() ports.TaskVerificationFacts
		stage, status, source string
		pending               bool
	}{
		{"native_legacy_pending", func() ports.TaskVerificationFacts { return verificationNativeFixture(at) }, "verification", "awaiting_review", "task_delivery", true},
		{"native_contracts_pending", func() ports.TaskVerificationFacts {
			f := verificationNativeFixture(at)
			c, e := verificationTestID(8), verificationTestID(9)
			f.ContractsEnabled = true
			f.NativeDelivery.Authority.Kind = d.TaskDeliveryAuthorityContractsV1
			f.NativeDelivery.Authority.ClaimID = nil
			f.NativeDelivery.Authority.ContractID = &c
			f.NativeDelivery.Authority.ExecutionID = &e
			f.NativeDelivery.SubmissionRef = &d.TaskDeliverySubmissionRef{ID: verificationTestID(11), ContractID: c, Digest: verificationTestDigest("c")}
			return f
		}, "verification", "awaiting_review", "task_delivery", true},
		{"signed_pending", func() ports.TaskVerificationFacts { return verificationSignedFixture(at) }, "verification", "awaiting_review", "signed_review_case", true},
		{"signed_changes_todo", func() ports.TaskVerificationFacts {
			f := verificationSignedFixture(at)
			f.PendingReviewCaseID = nil
			f.Lifecycle = d.WorkItemLifecycleTodo
			closed := at.Add(-time.Second)
			f.SignedReview.Status = d.ReviewChangesRequested
			f.SignedReview.ClosedAt = &closed
			f.SignedReview.CloseReason = "Address findings"
			f.SignedReview.ClosedBy = "reviewer"
			f.CorrectionReviewCaseID = f.LatestReviewCaseID
			f.SignedCorrection = f.SignedReview
			return f
		}, "todo", "corrections_requested", "signed_review_case", false},
		{"accepted_signed", func() ports.TaskVerificationFacts {
			f := verificationSignedFixture(at)
			f.PendingReviewCaseID = nil
			f.Lifecycle = d.WorkItemLifecycleDone
			closed := at.Add(-time.Second)
			f.SignedReview.Status = d.ReviewApproved
			f.SignedReview.ClosedAt = &closed
			f.SignedReview.CloseReason = "Accepted exact result"
			f.SignedReview.ClosedBy = "reviewer"
			return f
		}, "accepted", "done", "signed_review_case", false},
		{"native_changes_todo", func() ports.TaskVerificationFacts {
			f := verificationNativeFixture(at)
			verificationDecision(&f, d.TaskDeliveryDecisionChangesRequested, at)
			f.Lifecycle = d.WorkItemLifecycleTodo
			return f
		}, "todo", "corrections_requested", "task_delivery", false},
		{"correction_active_executor", func() ports.TaskVerificationFacts {
			f := verificationNativeFixture(at)
			verificationDecision(&f, d.TaskDeliveryDecisionChangesRequested, at)
			f.CurrentLease = &d.WorkLease{ClaimID: verificationTestID(12), PrincipalID: "executor", Actor: verificationTestActor(), AcquiredAt: at.Add(-time.Second), ExpiresAt: at.Add(time.Minute), FencingToken: 3}
			return f
		}, "in_progress", "in_progress", "task_delivery", false},
		{"correction_expired_executor", func() ports.TaskVerificationFacts {
			f := verificationNativeFixture(at)
			verificationDecision(&f, d.TaskDeliveryDecisionChangesRequested, at)
			f.CurrentLease = &d.WorkLease{ClaimID: verificationTestID(12), PrincipalID: "executor", Actor: verificationTestActor(), AcquiredAt: at.Add(-time.Minute), ExpiresAt: at, FencingToken: 3}
			return f
		}, "in_progress", "attention_needed", "task_delivery", false},
		{"native_withdrawn", func() ports.TaskVerificationFacts {
			f := verificationNativeFixture(at)
			id := verificationTestID(13)
			f.PendingDeliveryID = nil
			f.LatestDeliveryWithdrawalID = &id
			f.Lifecycle = d.WorkItemLifecycleTodo
			f.NativeWithdrawal = &d.TaskDeliveryWithdrawal{ID: id, Scope: verificationTestScope(), WorkItemID: f.WorkItemID, DeliveryID: f.NativeDelivery.ID, DeliveryDigest: f.NativeDelivery.Digest, Round: f.DeliveryRound, PrincipalID: "executor", Actor: verificationTestActor(), Reason: "Replan", WithdrawnAt: at.Add(-time.Second)}
			return f
		}, "todo", "withdrawn", "task_delivery", false},
		{"accepted_native", func() ports.TaskVerificationFacts {
			f := verificationNativeFixture(at)
			verificationDecision(&f, d.TaskDeliveryDecisionAccepted, at)
			f.Lifecycle = d.WorkItemLifecycleDone
			return f
		}, "accepted", "done", "task_delivery", false},
		{"terminal_historical_pending", func() ports.TaskVerificationFacts {
			f := verificationSignedFixture(at)
			f.Lifecycle = d.WorkItemLifecycleDone
			return f
		}, "accepted", "done", "signed_review_case", false},
		{"cancelled_historical_review", func() ports.TaskVerificationFacts {
			f := verificationSignedFixture(at)
			f.Lifecycle = d.WorkItemLifecycleCancelled
			return f
		}, "cancelled", "cancelled", "signed_review_case", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, err := projectTaskVerification(verificationTestScope(), tc.fixture(), 99, at)
			if err != nil {
				t.Fatal(err)
			}
			if v.Stage != tc.stage || v.Status != tc.status || v.SourceKind != tc.source || v.Pending != tc.pending {
				t.Fatalf("wrong source/stage: %+v", v)
			}
			if v.Pending && v.ExecutorAuthority != "ended" {
				t.Fatal("review retained executor authority")
			}
			if v.SourceKind == "task_delivery" && v.ReviewCaseVersion != 0 {
				t.Fatal("native delivery fabricated a mutable review version")
			}
			if v.SourceKind == "signed_review_case" && v.DeliveryDigest != "" {
				t.Fatal("signed review fabricated a native delivery")
			}
			if v.EvaluatedAt != at || v.OutcomeRevision != 99 {
				t.Fatal("projection changed snapshot")
			}
		})
	}
}

func TestTaskVerificationSignedReviewerExpiryRemainsVerification(t *testing.T) {
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	for _, expired := range []bool{false, true} {
		t.Run(map[bool]string{false: "active", true: "expired"}[expired], func(t *testing.T) {
			f := verificationSignedFixture(at)
			key := verificationTestID(16)
			binding := d.SignedContractBinding{ProtocolVersion: 2, ServerID: "server", CredentialID: verificationTestID(15), SignerKeyID: key, PolicyRevision: 1, AcceptanceFloor: d.AcceptanceIndependentReview, AllowedSigningKeyIDs: []d.ID{key}}
			c, err := d.NewReviewContract(verificationTestID(14), verificationTestID(17), *f.SignedReview, "reviewer", d.ActorRef{Kind: d.ActorKindHuman, Provider: "test", ID: "reviewer"}, "review", binding, d.DefaultContractLeasePolicy(), 30, at.Add(-time.Second))
			if err != nil {
				t.Fatal(err)
			}
			if expired {
				c.ExpiresAt = at
			}
			if err := f.SignedReview.Bind(c, at.Add(-time.Second)); err != nil {
				t.Fatal(err)
			}
			f.ReviewContract = &c
			v, err := projectTaskVerification(verificationTestScope(), f, 99, at)
			if err != nil {
				t.Fatal(err)
			}
			if !v.Pending || v.Stage != "verification" || v.ReviewCaseVersion != f.SignedReview.Version || v.ReviewerAuthority != map[bool]string{false: "active", true: "expired"}[expired] {
				t.Fatalf("review source/expiry mismatch: %+v", v)
			}
			state := applyTaskVerificationOperational(d.WorkItemOperationalState{DisplayState: d.WorkItemDisplayStateAttentionNeeded, LeaseStatus: d.LeaseStatusNone, ReadinessReasons: []d.ReadinessReason{d.ReadinessReasonLeaseExpired}}, &v)
			if string(state.DisplayState) != "awaiting_review" || state.LeaseStatus != d.LeaseStatusNone || len(state.ReadinessReasons) != 0 {
				t.Fatalf("review mistaken for executor recovery: %+v", state)
			}
		})
	}
}

func TestTaskVerificationSignedCorrectionWithEndedAuthorityIsTodo(t *testing.T) {
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	f := verificationSignedFixture(at)
	closed, decisionID, keyID := at.Add(-time.Second), verificationTestID(18), verificationTestID(16)
	f.PendingReviewCaseID = nil
	f.CorrectionReviewCaseID, f.SignedCorrection = f.LatestReviewCaseID, f.SignedReview
	f.SignedReview.Status = d.ReviewChangesRequested
	f.SignedReview.ClosedAt, f.SignedReview.ClosedBy, f.SignedReview.CloseReason = &closed, "reviewer", "Address findings"
	f.SignedReview.LatestDecisionID = &decisionID
	w := d.WorkItem{ID: f.WorkItemID, Scope: verificationTestScope(), Version: f.Version, Lifecycle: f.Lifecycle, ContractsEnabled: true, Criteria: d.NewCriterionSet(), CorrectionReviewCaseID: f.CorrectionReviewCaseID}
	c, err := d.NewWorkContract(f.SignedReview.WorkContractID, verificationTestID(17), w, "executor", verificationTestActor(), d.DefaultContractLeasePolicy(), 30, d.WorkContractSpec{Title: "Actual signed correction"}, 98, at.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	c.SignedBinding = &d.SignedContractBinding{ProtocolVersion: 2, ServerID: "server", CredentialID: verificationTestID(15), SignerKeyID: keyID, PolicyRevision: 1, AcceptanceFloor: d.AcceptanceIndependentReview, AllowedSigningKeyIDs: []d.ID{keyID}}
	c.Status, c.ClosedAt, c.ClosedBy, c.CloseReason = d.ContractDelivered, &closed, "executor", "Submitted actual result"
	c.LatestSubmissionID = &f.SignedReview.SubmissionID
	f.SignedReview.IssuedSpecDigest = c.SubmissionSpecDigest()
	f.ExecutionContract, f.CurrentContractID, w.CurrentContractID = &c, &c.ID, &c.ID
	v, err := projectTaskVerification(w.Scope, f, 99, at)
	if err != nil {
		t.Fatal(err)
	}
	if v.Stage != "todo" || v.Status != "corrections_requested" || v.Pending || !v.CorrectionRequired || v.ExecutorAuthority != "ended" || v.Lifecycle != d.WorkItemLifecycleInProgress {
		t.Fatalf("ended signed execution was presented as live/expired work: %+v", v)
	}
	if len(v.ReasonCodes) != 0 || v.SourceStatus != string(d.ReviewChangesRequested) || v.ReviewCaseVersion != f.SignedReview.Version || v.Round != f.SignedReview.Round {
		t.Fatalf("projection lost the actual immutable review source: %+v", v)
	}
	candidate, ok := taskVerificationCorrectionCandidate(w, &v)
	if !ok || candidate.Lifecycle != d.WorkItemLifecycleTodo || candidate.CurrentContractID != nil || candidate.CurrentLease != nil || w.Lifecycle != d.WorkItemLifecycleInProgress || w.CurrentContractID == nil {
		t.Fatal("signed correction handoff must use a readiness copy without changing persisted authority/lifecycle")
	}
	for _, corrupt := range []func(*d.WorkContract){
		func(c *d.WorkContract) { c.Status = d.ContractRevoked },
		func(c *d.WorkContract) { c.LatestSubmissionID = nil },
		func(c *d.WorkContract) { c.SpecDigest = verificationTestDigest("c") },
	} {
		changed := c
		corrupt(&changed)
		invalid := v
		invalid.executionContract = &changed
		if _, ok := taskVerificationCorrectionCandidate(w, &invalid); ok {
			t.Fatal("handoff ignored the actual delivered contract/submission binding")
		}
	}
	state := d.WorkItemOperationalState{Ref: w.Ref(), Lifecycle: w.Lifecycle, DisplayState: d.WorkItemDisplayStateAttentionNeeded, LeaseStatus: d.LeaseStatusExpired, ReadinessReasons: []d.ReadinessReason{d.ReadinessReasonLeaseExpired}}
	a := projectWorkAttention(WorkContext{Work: w, Operational: state, Verification: &v, Contract: &WorkContractView{Contract: c, EffectiveStatus: d.ContractDelivered, EvaluatedAt: at, Reasons: []d.ReadinessReason{d.ReadinessReasonLeaseExpired}}, OutcomeRevision: 99, EvaluatedAt: at})
	if a.Status != "corrections_requested" || a.ExecutionAllowed || a.AwaitingReview || !a.CorrectionRequired || a.OperationalState.LeaseStatus != d.LeaseStatusNone || len(a.OperationalState.ReadinessReasons) != 0 {
		t.Fatalf("attention invented expired executor authority: %+v", a)
	}
	for _, authority := range []string{"active", "expired"} {
		live := v
		live.ExecutorAuthority = authority
		if taskVerificationCorrectionPending(live) || taskVerificationStage(string(w.Lifecycle), live) != "in_progress" {
			t.Fatalf("%s correction execution lost its real stage", authority)
		}
		if _, ok := taskVerificationCorrectionCandidate(w, &live); ok {
			t.Fatal("correction acquisition was offered over active/expired execution")
		}
	}
}

func TestTaskVerificationCapabilitiesFollowTheActualProtocolGuard(t *testing.T) {
	for _, phase := range []d.WorkProtocolPhase{d.WorkProtocolLegacy, d.WorkProtocolContracts, d.WorkProtocolSignedDraining, d.WorkProtocolSigned} {
		p := d.DefaultWorkProtocol(verificationTestScope().NamespaceID)
		p.Phase = phase
		for _, command := range []string{"AcquireSignedWorkContract", "AcquireSignedReviewContract", "ReturnSignedWork", "ReturnSignedReview"} {
			err := taskVerificationProtocolCondition(p, command)
			if (err == nil) != (phase == d.WorkProtocolSigned) {
				t.Fatalf("%s was advertised for incompatible protocol %s: %v", command, phase, err)
			}
		}
		for _, command := range []string{"ClaimWorkItem", "AcquireWorkContract", "SubmitTaskForReview", "AcceptTaskDelivery", "RequestTaskDeliveryChanges", "WithdrawTaskDelivery"} {
			actual, projected := protectWorkProtocol(p, command), taskVerificationProtocolCondition(p, command)
			actualCode, _ := d.ErrorCodeOf(actual)
			projectedCode, _ := d.ErrorCodeOf(projected)
			if (actual == nil) != (projected == nil) || actualCode != projectedCode {
				t.Fatalf("%s capability diverged from mutation protocol %s", command, phase)
			}
		}
	}
}

func TestTaskVerificationRejectsMismatchedActualSources(t *testing.T) {
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name   string
		mutate func(*ports.TaskVerificationFacts)
	}{
		{"missing_delivery", func(f *ports.TaskVerificationFacts) { f.NativeDelivery = nil }},
		{"wrong_round", func(f *ports.TaskVerificationFacts) { f.NativeDelivery.Round++ }},
		{"wrong_scope", func(f *ports.TaskVerificationFacts) { f.NativeDelivery.Scope.OutcomeID = verificationTestID(20) }},
		{"wrong_task", func(f *ports.TaskVerificationFacts) { f.NativeDelivery.WorkItemID = verificationTestID(20) }},
		{"wrong_pointer", func(f *ports.TaskVerificationFacts) { id := verificationTestID(20); f.PendingDeliveryID = &id }},
		{"future_source", func(f *ports.TaskVerificationFacts) { f.NativeDelivery.DeliveredAt = at.Add(time.Second) }},
		{"executor_retained", func(f *ports.TaskVerificationFacts) { f.CurrentLease = &d.WorkLease{ExpiresAt: at.Add(time.Minute)} }},
		{"stale_decision", func(f *ports.TaskVerificationFacts) {
			verificationDecision(f, d.TaskDeliveryDecisionChangesRequested, at)
			f.NativeDecision.Round++
		}},
		{"double_protocol", func(f *ports.TaskVerificationFacts) { id := verificationTestID(7); f.PendingReviewCaseID = &id }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := verificationNativeFixture(at)
			tc.mutate(&f)
			_, err := projectTaskVerification(verificationTestScope(), f, 99, at)
			if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodePreconditionFailed {
				t.Fatalf("source did not fail closed: %v", err)
			}
		})
	}
}

type verificationFakeRepository struct {
	facts []ports.TaskVerificationFacts
	calls int
	ids   []d.ID
}

func (r *verificationFakeRepository) TaskVerificationFacts(_ context.Context, _ d.Scope, ids []d.ID) ([]ports.TaskVerificationFacts, error) {
	r.calls++
	r.ids = append([]d.ID{}, ids...)
	return r.facts, nil
}

type verificationFakeUnit struct {
	ports.UnitOfWork
	repo *verificationFakeRepository
}

func (u verificationFakeUnit) TaskVerificationReads() ports.TaskVerificationReadRepository {
	return u.repo
}

func TestTaskVerificationBatchIsBoundedAndChecksSnapshot(t *testing.T) {
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	f := verificationNativeFixture(at)
	w := d.WorkItem{ID: f.WorkItemID, Scope: verificationTestScope(), Version: f.Version, Lifecycle: f.Lifecycle, PendingDeliveryID: f.PendingDeliveryID, LatestDeliveryID: f.LatestDeliveryID, DeliveryRound: f.DeliveryRound, Criteria: d.NewCriterionSet()}
	digest, err := d.TaskDefinitionDigest(w)
	if err != nil {
		t.Fatal(err)
	}
	f.NativeDelivery.DefinitionDigest = digest
	r := &verificationFakeRepository{facts: []ports.TaskVerificationFacts{f}}
	u := verificationFakeUnit{repo: r}
	v, err := (&Service{}).taskVerificationAt(context.Background(), u, w, 101, at)
	if err != nil {
		t.Fatal(err)
	}
	if r.calls != 1 || len(r.ids) != 1 || v.OutcomeRevision != 101 || v.EvaluatedAt != at {
		t.Fatal("query did not reuse bounded transaction source")
	}
	w.Version++
	if _, err := (&Service{}).taskVerificationAt(context.Background(), u, w, 101, at); err == nil {
		t.Fatal("accepted mixed Task/source revisions")
	}
	r.calls = 0
	if _, err := loadTaskVerificationFacts(context.Background(), u, w.Scope, []d.ID{w.ID, w.ID}); err == nil || r.calls != 0 {
		t.Fatal("duplicate batch reached storage")
	}
	if _, err := loadTaskVerificationFacts(context.Background(), u, w.Scope, make([]d.ID, 101)); err == nil || r.calls != 0 {
		t.Fatal("oversized batch reached storage")
	}
	r.facts = append(r.facts, f)
	if _, err := loadTaskVerificationFacts(context.Background(), u, w.Scope, []d.ID{w.ID}); err == nil {
		t.Fatal("duplicate source response accepted")
	}
}

func TestTaskVerificationOneBatchServesOneHundredCards(t *testing.T) {
	ids := make([]d.ID, 100)
	r := &verificationFakeRepository{}
	for i := range ids {
		ids[i] = verificationTestID(byte(i + 20))
		r.facts = append(r.facts, ports.TaskVerificationFacts{WorkItemID: ids[i], Version: 1, Lifecycle: d.WorkItemLifecycleBacklog})
	}
	facts, err := loadTaskVerificationFacts(context.Background(), verificationFakeUnit{repo: r}, verificationTestScope(), ids)
	if err != nil {
		t.Fatal(err)
	}
	if r.calls != 1 || len(r.ids) != 100 || len(facts) != 100 {
		t.Fatal("card batch became per-Task reads")
	}
}

func TestTaskVerificationAttentionUsesSourcesAndKeepsTerminalPrecedence(t *testing.T) {
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	for _, lifecycle := range []d.WorkItemLifecycle{d.WorkItemLifecycleInProgress, d.WorkItemLifecycleDone, d.WorkItemLifecycleCancelled} {
		f := verificationNativeFixture(at)
		f.Lifecycle = lifecycle
		v, err := projectTaskVerification(verificationTestScope(), f, 101, at)
		if err != nil {
			t.Fatal(err)
		}
		w := d.WorkItem{ID: f.WorkItemID, Scope: verificationTestScope(), Lifecycle: lifecycle, Criteria: d.NewCriterionSet()}
		state := d.WorkItemOperationalState{Ref: w.Ref(), Lifecycle: lifecycle, DisplayState: d.WorkItemDisplayStateAttentionNeeded, ReadinessReasons: []d.ReadinessReason{d.ReadinessReasonLeaseExpired}, EvaluatedAt: at}
		state = applyTaskVerificationOperational(state, &v)
		a := projectWorkAttention(WorkContext{Work: w, Operational: state, Verification: &v, OutcomeRevision: 101, EvaluatedAt: at})
		if lifecycle == d.WorkItemLifecycleInProgress {
			if !a.AwaitingReview || a.Status != "awaiting_review" || a.ExecutionAllowed || a.OperationalState.LeaseStatus != d.LeaseStatusNone {
				t.Fatalf("pending review looks executable: %+v", a)
			}
		} else if a.AwaitingReview || a.CorrectionRequired || a.Status != string(lifecycle) || a.ExecutionAllowed {
			t.Fatalf("history overrode terminal Task: %+v", a)
		}
	}
	if taskVerificationStage("in_progress", TaskVerificationView{Pending: true, SourceKind: "none"}) == "verification" {
		t.Fatal("pointer/display inference fabricated verification")
	}
}

func TestTaskVerificationAssessmentIsBoundToCurrentExactRound(t *testing.T) {
	id, submission := verificationTestID(4), verificationTestID(11)
	v := TaskVerificationView{SourceKind: "task_delivery", SourceID: &id, DeliveryDigest: verificationTestDigest("a"), Round: 3, Current: true, SubmissionID: &submission, SubmissionDigest: verificationTestDigest("b")}
	a := d.CriterionAssessment{TaskDeliveryRef: &d.TaskDeliveryAssessmentRef{ID: id, Digest: v.DeliveryDigest, Round: 3}, SubmissionID: &submission, SubmissionDigest: v.SubmissionDigest}
	if !verificationAssessmentApplies(&v, a) {
		t.Fatal("current exact proof was rejected")
	}
	for _, mutate := range []func(*d.CriterionAssessment){func(a *d.CriterionAssessment) { a.TaskDeliveryRef = nil }, func(a *d.CriterionAssessment) { r := *a.TaskDeliveryRef; r.Round--; a.TaskDeliveryRef = &r }, func(a *d.CriterionAssessment) { a.SubmissionDigest = verificationTestDigest("c") }} {
		old := a
		mutate(&old)
		if verificationAssessmentApplies(&v, old) {
			t.Fatal("historical delivery/submission proof inherited")
		}
	}
	v.Current = false
	if verificationAssessmentApplies(&v, a) {
		t.Fatal("changes-requested/withdrawn history became current proof")
	}
}

func TestTaskVerificationQuotesNativeRoundAndSignedVersionWithoutConflation(t *testing.T) {
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	for _, signed := range []bool{false, true} {
		f := verificationNativeFixture(at)
		big := uint64(1<<53) + 1
		f.Version = d.Version(big)
		if signed {
			f = verificationSignedFixture(at)
			f.SignedReview.Version = d.Version(big)
			f.SignedReview.Round = big
		} else {
			f.DeliveryRound = big
			f.NativeDelivery.Round = big
		}
		v, err := projectTaskVerification(verificationTestScope(), f, d.OutcomeRevision(big), at)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]any
		if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatal(err)
		}
		if result["round"] != "9007199254740993" || result["outcome_revision"] != "9007199254740993" {
			t.Fatalf("lost exact counters: %s", raw)
		}
		_, version := result["review_case_version"]
		_, digest := result["delivery_digest"]
		if version != signed || digest == signed {
			t.Fatalf("protocol sources conflated: %s", raw)
		}
	}
}
