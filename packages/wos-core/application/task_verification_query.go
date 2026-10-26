package application

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// TaskVerificationView adapts actual delivery protocols without creating a
// second persisted review state. Native deliveries are identified by immutable
// digest and round; only an actual signed ReviewCase has a case version.
type TaskVerificationView struct {
	signedReview          *d.ReviewCase
	executionContract     *d.WorkContract
	Ref                   d.EntityRef                  `json:"ref"`
	TaskVersion           d.Version                    `json:"task_version,string"`
	OutcomeRevision       d.OutcomeRevision            `json:"outcome_revision,string"`
	EvaluatedAt           time.Time                    `json:"evaluated_at"`
	SourceKind            string                       `json:"source_kind"`
	SourceStatus          string                       `json:"source_status,omitempty"`
	Protocol              string                       `json:"protocol"`
	SourceID              *d.ID                        `json:"source_id,omitempty"`
	Round                 uint64                       `json:"round,omitempty,string"`
	DeliveryDigest        string                       `json:"delivery_digest,omitempty"`
	ReviewCaseVersion     d.Version                    `json:"review_case_version,omitempty,string"`
	SubmissionID          *d.ID                        `json:"submission_id,omitempty"`
	SubmissionDigest      string                       `json:"submission_digest,omitempty"`
	WorkContractID        *d.ID                        `json:"work_contract_id,omitempty"`
	ReviewContractID      *d.ID                        `json:"review_contract_id,omitempty"`
	DecisionID            *d.ID                        `json:"decision_id,omitempty"`
	WithdrawalID          *d.ID                        `json:"withdrawal_id,omitempty"`
	Lifecycle             d.WorkItemLifecycle          `json:"lifecycle"`
	Stage                 string                       `json:"stage"`
	Status                string                       `json:"status"`
	Pending               bool                         `json:"pending"`
	Current               bool                         `json:"current"`
	CorrectionRequired    bool                         `json:"correction_required"`
	ExecutorAuthority     string                       `json:"executor_authority"`
	ReviewerAuthority     string                       `json:"reviewer_authority"`
	Executor              *WorkspacePerson             `json:"executor,omitempty"`
	ExecutionPrincipalIDs []string                     `json:"execution_principal_ids,omitempty"`
	Reviewer              *WorkspacePerson             `json:"reviewer,omitempty"`
	Material              TaskVerificationMaterial     `json:"material"`
	Requirements          *WorkspaceCriteriaSummary    `json:"requirements,omitempty"`
	ReasonCodes           []string                     `json:"reason_codes"`
	Actions               []WorkspaceCapability        `json:"actions"`
	Coverage              map[string]WorkspaceCoverage `json:"coverage"`
}

type TaskVerificationMaterial struct {
	Summary           string                          `json:"summary,omitempty"`
	ArtifactCount     int                             `json:"artifact_count"`
	EvidenceCount     int                             `json:"evidence_count"`
	CriterionCount    int                             `json:"criterion_count"`
	Artifacts         []d.SubmissionArtifact          `json:"artifacts,omitempty"`
	EvidenceIDs       []d.ID                          `json:"evidence_ids,omitempty"`
	CriterionEvidence []d.SubmissionCriterionEvidence `json:"criterion_evidence,omitempty"`
}

func (v TaskVerificationView) MarshalJSON() ([]byte, error) {
	type plain TaskVerificationView
	return marshalWorkspaceProjection(plain(v))
}

func taskVerificationSourceError(scope d.Scope, workID d.ID, message string) error {
	ref := d.EntityRef{Scope: scope, Kind: d.EntityKindWorkItem, ID: workID}
	return d.NewDetailedError(d.ErrorCodePreconditionFailed, message, d.ErrorDetails{
		ReasonCode: "task_verification_source_mismatch", SubjectRef: &ref,
		Remedies: []d.ErrorRemedy{{ActionID: "refresh_workspace", SubjectRef: &ref}},
	})
}

// loadTaskVerificationFacts never opens a transaction or resamples its clock.
// All callers compose it inside their existing authenticated read frame.
func loadTaskVerificationFacts(ctx context.Context, u ports.UnitOfWork, scope d.Scope, ids []d.ID) (map[d.ID]ports.TaskVerificationFacts, error) {
	if len(ids) > ports.MaxTaskVerificationBatch {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "Task verification batch exceeds 100 Tasks")
	}
	result := make(map[d.ID]ports.TaskVerificationFacts, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	wanted := make(map[d.ID]bool, len(ids))
	for _, id := range ids {
		if id.Validate() != nil || wanted[id] {
			return nil, d.NewError(d.ErrorCodeInvalidArgument, "Task verification requires distinct valid Task IDs")
		}
		wanted[id] = true
	}
	reads, ok := u.(ports.TaskVerificationReadUnitOfWork)
	if !ok {
		return nil, d.NewError(d.ErrorCodeInvalidConfig, "actual-source Task verification reads are unavailable")
	}
	items, err := reads.TaskVerificationReads().TaskVerificationFacts(ctx, scope, ids)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if !wanted[item.WorkItemID] {
			return nil, taskVerificationSourceError(scope, item.WorkItemID, "Task verification returned an unrequested source")
		}
		if _, duplicate := result[item.WorkItemID]; duplicate {
			return nil, taskVerificationSourceError(scope, item.WorkItemID, "Task verification returned a duplicate source")
		}
		result[item.WorkItemID] = item
	}
	if len(result) != len(wanted) {
		return nil, d.NewError(d.ErrorCodeNotFound, "one or more Task verification sources were not found")
	}
	return result, nil
}

func verificationIDEqual(a, b *d.ID) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

func verificationFactsMatchWork(f ports.TaskVerificationFacts, w d.WorkItem) bool {
	return f.WorkItemID == w.ID && f.Version == w.Version && f.Lifecycle == w.Lifecycle &&
		reflect.DeepEqual(f.CurrentLease, w.CurrentLease) &&
		f.ContractsEnabled == w.ContractsEnabled && f.DeliveryRound == w.DeliveryRound &&
		verificationIDEqual(f.CurrentContractID, w.CurrentContractID) &&
		verificationIDEqual(f.PendingDeliveryID, w.PendingDeliveryID) &&
		verificationIDEqual(f.LatestDeliveryID, w.LatestDeliveryID) &&
		verificationIDEqual(f.LatestDeliveryDecisionID, w.LatestDeliveryDecisionID) &&
		verificationIDEqual(f.LatestDeliveryWithdrawalID, w.LatestDeliveryWithdrawalID) &&
		verificationIDEqual(f.PendingReviewCaseID, w.PendingReviewCaseID) &&
		verificationIDEqual(f.LatestReviewCaseID, w.LatestReviewCaseID) &&
		verificationIDEqual(f.CorrectionReviewCaseID, w.CorrectionReviewCaseID)
}

func projectTaskVerification(scope d.Scope, f ports.TaskVerificationFacts, revision d.OutcomeRevision, at time.Time) (TaskVerificationView, error) {
	v := TaskVerificationView{Ref: d.EntityRef{Scope: scope, Kind: d.EntityKindWorkItem, ID: f.WorkItemID}, TaskVersion: f.Version, OutcomeRevision: revision, EvaluatedAt: at,
		SourceKind: "none", Protocol: "none", Lifecycle: f.Lifecycle, Status: string(f.Lifecycle), ExecutorAuthority: "none", ReviewerAuthority: "none", ReasonCodes: []string{}, Actions: []WorkspaceCapability{},
		Coverage: map[string]WorkspaceCoverage{"source": {Status: "complete"}, "material": {Status: "not_applicable"}, "criteria": {Status: "not_loaded"}, "actions": {Status: "not_loaded"}, "readiness": {Status: "not_loaded"}},
	}
	fail := func(message string) (TaskVerificationView, error) {
		return TaskVerificationView{}, taskVerificationSourceError(scope, f.WorkItemID, message)
	}
	if v.Ref.Validate() != nil || f.Version.Validate() != nil || at.IsZero() {
		return fail("Task verification header is invalid")
	}
	switch f.Lifecycle {
	case d.WorkItemLifecycleBacklog, d.WorkItemLifecycleTodo, d.WorkItemLifecycleInProgress, d.WorkItemLifecycleDone, d.WorkItemLifecycleCancelled:
	default:
		return fail("Task verification lifecycle is invalid")
	}
	terminal := f.Lifecycle == d.WorkItemLifecycleDone || f.Lifecycle == d.WorkItemLifecycleCancelled
	if f.PendingDeliveryID != nil && f.PendingReviewCaseID != nil {
		return fail("a Task cannot have both native and signed pending verification")
	}
	if f.CurrentContractID != nil {
		c := f.ExecutionContract
		if c == nil || c.ID != *f.CurrentContractID || c.Scope != scope || c.WorkItemID != f.WorkItemID || c.ValidateHeader() != nil {
			return fail("current execution contract does not match the Task")
		}
		v.executionContract = c
		v.ExecutorAuthority = "expired"
		if c.Status != d.ContractActive {
			v.ExecutorAuthority = "ended"
		}
		v.Protocol = string(d.WorkProtocolContracts)
		if c.SignedBinding != nil {
			v.Protocol = string(d.WorkProtocolSigned)
		}
		if c.ValidAt(at) {
			v.ExecutorAuthority = "active"
		}
		actor, acquired := c.Actor, c.AcquiredAt
		v.Executor = &WorkspacePerson{Role: "executor", Actor: &actor, PrincipalID: c.HolderPrincipalID, Source: "work_contract", AuthorityState: v.ExecutorAuthority, ContractID: &c.ID, At: &acquired}
	} else if f.ExecutionContract != nil {
		return fail("unreferenced execution contract in Task verification")
	} else if f.CurrentLease != nil {
		v.Protocol = string(d.WorkProtocolLegacy)
		v.ExecutorAuthority = "expired"
		if f.CurrentLease.ValidAt(at) {
			v.ExecutorAuthority = "active"
		}
		actor, acquired := f.CurrentLease.Actor, f.CurrentLease.AcquiredAt
		v.Executor = &WorkspacePerson{Role: "executor", Actor: &actor, PrincipalID: f.CurrentLease.PrincipalID, Source: "work_lease", AuthorityState: v.ExecutorAuthority, At: &acquired}
	}
	if f.LatestDeliveryID != nil {
		n := f.NativeDelivery
		if n == nil || n.ID != *f.LatestDeliveryID || n.Scope != scope || n.WorkItemID != f.WorkItemID || n.Round == 0 || n.Round != f.DeliveryRound || !d.ValidSignedDigest(n.Digest) || !d.ValidSignedDigest(n.DefinitionDigest) || n.Authority.Validate() != nil || n.DeliveredAt.IsZero() || n.DeliveredAt.After(at) {
			return fail("latest native delivery does not match the exact Task round")
		}
		if n.Round == 1 && n.SupersedesDeliveryID != nil || n.Round > 1 && (n.SupersedesDeliveryID == nil || n.SupersedesDeliveryID.Validate() != nil) {
			return fail("native delivery predecessor does not match its round")
		}
		if n.Authority.Kind == d.TaskDeliveryAuthorityLegacyLease && n.SubmissionRef != nil {
			return fail("legacy delivery has a fabricated contract submission")
		}
		if n.Authority.Kind == d.TaskDeliveryAuthorityContractsV1 && (n.SubmissionRef == nil || n.SubmissionRef.ID.Validate() != nil || n.SubmissionRef.ContractID != *n.Authority.ContractID || !d.ValidSignedDigest(n.SubmissionRef.Digest)) {
			return fail("contracts_v1 delivery lacks its exact submission")
		}
		if f.PendingDeliveryID == nil && f.LatestDeliveryDecisionID == nil && f.LatestDeliveryWithdrawalID == nil {
			return fail("native delivery has no current resolution")
		}
		if f.PendingDeliveryID != nil && *f.PendingDeliveryID != n.ID {
			return fail("pending native delivery is not the latest delivery")
		}
		if f.LatestDeliveryDecisionID != nil && f.LatestDeliveryWithdrawalID != nil {
			return fail("native delivery has both a decision and withdrawal")
		}
		if f.LatestDeliveryDecisionID != nil {
			x := f.NativeDecision
			if x == nil || x.ID != *f.LatestDeliveryDecisionID || x.Validate() != nil || x.Scope != scope || x.WorkItemID != f.WorkItemID || x.DeliveryID != n.ID || x.DeliveryDigest != n.Digest || x.Round != n.Round || x.DecidedAt.Before(n.DeliveredAt) || x.DecidedAt.After(at) {
				return fail("native decision does not match the exact delivery")
			}
		} else if f.NativeDecision != nil {
			return fail("unreferenced native decision")
		}
		if f.LatestDeliveryWithdrawalID != nil {
			x := f.NativeWithdrawal
			if x == nil || x.ID != *f.LatestDeliveryWithdrawalID || x.Validate() != nil || x.Scope != scope || x.WorkItemID != f.WorkItemID || x.DeliveryID != n.ID || x.DeliveryDigest != n.Digest || x.Round != n.Round || x.WithdrawnAt.Before(n.DeliveredAt) || x.WithdrawnAt.After(at) {
				return fail("native withdrawal does not match the exact delivery")
			}
		} else if f.NativeWithdrawal != nil {
			return fail("unreferenced native withdrawal")
		}
		if f.PendingDeliveryID != nil && (f.NativeDecision != nil || f.NativeWithdrawal != nil) && !terminal {
			return fail("pending native delivery already has a terminal record")
		}
	} else if f.NativeDelivery != nil || f.PendingDeliveryID != nil || f.LatestDeliveryDecisionID != nil || f.LatestDeliveryWithdrawalID != nil || f.DeliveryRound != 0 {
		return fail("native delivery navigation has no latest delivery")
	}
	if f.SignedCorrection != nil || f.CorrectionReviewCaseID != nil {
		r := f.SignedCorrection
		if r == nil || f.CorrectionReviewCaseID == nil || r.ID != *f.CorrectionReviewCaseID || r.Scope != scope || r.WorkItemID != f.WorkItemID || r.Validate() != nil || r.Status != d.ReviewChangesRequested {
			return fail("correction pointer does not identify an actual changes-requested case")
		}
	}
	if f.PendingDeliveryID == nil && (f.SignedReview != nil || f.PendingReviewCaseID != nil || f.LatestReviewCaseID != nil) {
		r := f.SignedReview
		target := f.PendingReviewCaseID
		if target == nil {
			target = f.LatestReviewCaseID
		}
		if r == nil || target == nil || r.ID != *target || r.Scope != scope || r.WorkItemID != f.WorkItemID || r.Validate() != nil || r.CreatedAt.After(at) {
			return fail("signed review pointer does not identify its actual case")
		}
		if f.PendingReviewCaseID != nil && f.LatestReviewCaseID != nil && *f.PendingReviewCaseID != *f.LatestReviewCaseID {
			return fail("pending signed review is not the latest case")
		}
		v.SourceKind, v.SourceID, v.Round, v.ReviewCaseVersion = "signed_review_case", &r.ID, r.Round, r.Version
		v.SourceStatus = string(r.Status)
		v.Protocol, v.signedReview = string(d.WorkProtocolSigned), r
		v.ExecutionPrincipalIDs = append([]string{}, r.ExecutionPrincipals[:min(20, len(r.ExecutionPrincipals))]...)
		v.Coverage["executors"] = WorkspaceCoverage{Status: "complete", ReturnedCount: workspaceCount(len(v.ExecutionPrincipalIDs)), TotalCount: workspaceCount(len(r.ExecutionPrincipals))}
		if len(r.ExecutionPrincipals) > 20 {
			v.Coverage["executors"] = WorkspaceCoverage{Status: "partial", ReturnedCount: workspaceCount(len(v.ExecutionPrincipalIDs)), TotalCount: workspaceCount(len(r.ExecutionPrincipals)), Reason: "review_case_execution_principals_preview"}
		}
		v.SubmissionID, v.SubmissionDigest, v.WorkContractID = &r.SubmissionID, r.SubmissionDigest, &r.WorkContractID
		v.DecisionID = r.LatestDecisionID
		v.Coverage["material"] = WorkspaceCoverage{Status: "not_loaded", Reason: "read_exact_signed_submission"}
		v.Pending = f.PendingReviewCaseID != nil && r.Status.Open() && !terminal
		if f.PendingReviewCaseID != nil && !r.Status.Open() && !terminal {
			return fail("pending signed pointer identifies a closed review case")
		}
		v.Current = v.Pending || terminal && r.Status == d.ReviewApproved
		v.CorrectionRequired = r.Status == d.ReviewChangesRequested || f.SignedCorrection != nil
		if v.Executor == nil && len(r.ExecutionPrincipals) == 1 {
			v.Executor = &WorkspacePerson{Role: "executor", PrincipalID: r.ExecutionPrincipals[0], Source: "review_case_execution_principals", AuthorityState: "ended"}
		}
		if r.ClosedAt != nil && r.LatestDecisionID != nil && (r.Status == d.ReviewApproved || r.Status == d.ReviewChangesRequested) {
			v.Reviewer = &WorkspacePerson{Role: "reviewer", PrincipalID: r.ClosedBy, Source: "review_case_closure", AuthorityState: "ended", At: r.ClosedAt}
		}
		if v.Pending {
			v.Status = "awaiting_review"
			v.ExecutorAuthority = "ended"
			if f.CurrentLease != nil {
				return fail("pending signed review still has executor authority")
			}
			if c := f.ExecutionContract; c != nil && (c.Status != d.ContractDelivered || c.SignedBinding == nil || c.ID != r.WorkContractID || c.LatestSubmissionID == nil || *c.LatestSubmissionID != r.SubmissionID || c.SubmissionSpecDigest() != r.IssuedSpecDigest) {
				return fail("pending signed review does not bind its ended execution contract")
			}
			if r.Status == d.ReviewInReview {
				c := f.ReviewContract
				if c == nil || r.CurrentContractID == nil || c.ID != *r.CurrentContractID || c.CaseID != r.ID || c.WorkItemID != f.WorkItemID || c.Scope != scope || c.SubmissionID != r.SubmissionID || c.SubmissionDigest != r.SubmissionDigest || c.Validate() != nil {
					return fail("signed reviewer authority does not match the actual case")
				}
				v.ReviewContractID = &c.ID
				v.ReviewerAuthority = "expired"
				if c.ValidAt(at) {
					v.ReviewerAuthority, v.Status = "active", "in_review"
				} else {
					v.ReasonCodes = append(v.ReasonCodes, "review_authority_expired")
				}
				actor, acquired := c.Actor, c.AcquiredAt
				v.Reviewer = &WorkspacePerson{Role: "reviewer", Actor: &actor, PrincipalID: c.HolderPrincipalID, Source: "review_contract", AuthorityState: v.ReviewerAuthority, ContractID: &c.ID, At: &acquired}
			} else if f.ReviewContract != nil {
				return fail("pending case has an unreferenced reviewer authority")
			}
		}
	} else if f.NativeDelivery != nil {
		n := f.NativeDelivery
		v.SourceKind, v.SourceID, v.Round, v.DeliveryDigest = "task_delivery", &n.ID, n.Round, n.Digest
		v.SourceStatus = "pending"
		v.ExecutionPrincipalIDs = []string{n.Authority.PrincipalID}
		v.Coverage["executors"] = WorkspaceCoverage{Status: "complete", ReturnedCount: workspaceCount(1), TotalCount: workspaceCount(1)}
		v.Protocol = string(d.WorkProtocolLegacy)
		if n.Authority.Kind == d.TaskDeliveryAuthorityContractsV1 {
			v.Protocol = string(d.WorkProtocolContracts)
		}
		v.Material = TaskVerificationMaterial{Summary: workspaceText(n.Summary, 512), ArtifactCount: n.ArtifactCount, EvidenceCount: n.EvidenceCount, CriterionCount: n.CriterionCount}
		v.Coverage["material"] = WorkspaceCoverage{Status: "partial", Reason: "immutable_delivery_header; expand canonical delivery or submission for complete material"}
		if n.SubmissionRef != nil {
			v.SubmissionID, v.SubmissionDigest, v.WorkContractID = &n.SubmissionRef.ID, n.SubmissionRef.Digest, &n.SubmissionRef.ContractID
		}
		v.Pending = f.PendingDeliveryID != nil && f.NativeDecision == nil && f.NativeWithdrawal == nil && !terminal
		v.Current = v.Pending || f.Lifecycle == d.WorkItemLifecycleDone && f.NativeDecision != nil && f.NativeDecision.Result == d.TaskDeliveryDecisionAccepted
		if f.NativeDecision != nil {
			v.SourceStatus = string(f.NativeDecision.Result)
			v.DecisionID = &f.NativeDecision.ID
			v.CorrectionRequired = f.NativeDecision.Result == d.TaskDeliveryDecisionChangesRequested
			actor, decided := f.NativeDecision.ReviewerActor, f.NativeDecision.DecidedAt
			v.Reviewer = &WorkspacePerson{Role: "reviewer", Actor: &actor, PrincipalID: f.NativeDecision.ReviewerPrincipalID, Source: "task_delivery_decision", AuthorityState: "ended", At: &decided}
		}
		if f.NativeWithdrawal != nil {
			v.WithdrawalID = &f.NativeWithdrawal.ID
			v.SourceStatus = "withdrawn"
			v.Status = "withdrawn"
		}
		if v.Executor == nil {
			actor, delivered := n.Authority.Actor, n.DeliveredAt
			v.Executor = &WorkspacePerson{Role: "executor", Actor: &actor, PrincipalID: n.Authority.PrincipalID, Source: "task_delivery", AuthorityState: "ended", ContractID: n.Authority.ContractID, At: &delivered}
		}
		if v.Pending {
			if f.CurrentLease != nil || f.CurrentContractID != nil {
				return fail("pending native delivery still has executor authority")
			}
			v.Status, v.ExecutorAuthority = "awaiting_review", "ended"
		}
	}
	if terminal {
		v.Pending, v.CorrectionRequired = false, false
		v.ExecutorAuthority, v.ReviewerAuthority = "ended", "ended"
		if v.Executor != nil {
			v.Executor.AuthorityState = "ended"
		}
		if v.Reviewer != nil {
			v.Reviewer.AuthorityState = "ended"
		}
		v.Status = string(f.Lifecycle)
	} else if !v.Pending {
		if f.Lifecycle == d.WorkItemLifecycleBacklog {
			v.Status = "backlog"
		} else if taskVerificationCorrectionPending(v) {
			v.Status = "corrections_requested"
		} else if f.Lifecycle == d.WorkItemLifecycleInProgress {
			v.Status = "in_progress"
			if v.ExecutorAuthority != "active" {
				v.Status = "attention_needed"
				v.ReasonCodes = append(v.ReasonCodes, "execution_authority_expired")
			}
		} else if v.CorrectionRequired {
			v.Status = "corrections_requested"
		}
	}
	v.Stage = taskVerificationStage(string(f.Lifecycle), v)
	return v, nil
}

func verificationDisplayState(v TaskVerificationView) string {
	if v.Pending {
		return "awaiting_review"
	}
	return v.Status
}

func (s *Service) taskVerificationAt(ctx context.Context, u ports.UnitOfWork, w d.WorkItem, revision d.OutcomeRevision, at time.Time) (TaskVerificationView, error) {
	facts, err := loadTaskVerificationFacts(ctx, u, w.Scope, []d.ID{w.ID})
	if err != nil {
		return TaskVerificationView{}, err
	}
	f := facts[w.ID]
	if !verificationFactsMatchWork(f, w) {
		return TaskVerificationView{}, taskVerificationSourceError(w.Scope, w.ID, "Task verification and current Task are from different snapshots")
	}
	v, err := projectTaskVerification(w.Scope, f, revision, at)
	if err != nil {
		return TaskVerificationView{}, err
	}
	if v.Pending && v.SourceKind == "task_delivery" {
		digest, err := d.TaskDefinitionDigest(w)
		if err != nil {
			return TaskVerificationView{}, err
		}
		if f.NativeDelivery.DefinitionDigest != digest {
			return TaskVerificationView{}, taskVerificationSourceError(w.Scope, w.ID, "pending delivery does not match the current Task definition")
		}
	}
	return v, nil
}

// Existing custom adapters can continue reading Tasks without delivery history.
// An adapter must implement the actual-source port before it can describe any
// delivery/review navigation; raw pointers never become a verification fallback.
func (s *Service) optionalTaskVerificationAt(ctx context.Context, u ports.UnitOfWork, w d.WorkItem, revision d.OutcomeRevision, at time.Time) (*TaskVerificationView, error) {
	if _, ok := u.(ports.TaskVerificationReadUnitOfWork); !ok {
		if w.LatestDeliveryID != nil || w.PendingReviewCaseID != nil || w.LatestReviewCaseID != nil || w.CorrectionReviewCaseID != nil {
			return nil, d.NewError(d.ErrorCodeInvalidConfig, "actual-source Task verification reads are unavailable")
		}
		return nil, nil
	}
	v, err := s.taskVerificationAt(ctx, u, w, revision, at)
	return &v, err
}

func applyTaskVerificationOperational(state d.WorkItemOperationalState, v *TaskVerificationView) d.WorkItemOperationalState {
	if v == nil || v.Lifecycle == d.WorkItemLifecycleDone || v.Lifecycle == d.WorkItemLifecycleCancelled {
		return state
	}
	if v.Pending || taskVerificationCorrectionPending(*v) {
		state.DisplayState = d.WorkItemDisplayState(v.Status)
		if v.Pending {
			state.DisplayState = d.WorkItemDisplayState("awaiting_review")
		}
		state.LeaseStatus = d.LeaseStatusNone
		filtered := make([]d.ReadinessReason, 0, len(state.ReadinessReasons))
		for _, r := range state.ReadinessReasons {
			if r != d.ReadinessReasonLeaseExpired {
				filtered = append(filtered, r)
			}
		}
		state.ReadinessReasons = filtered
	} else if v.Lifecycle == d.WorkItemLifecycleInProgress && v.ExecutorAuthority == "active" {
		// Contract authority is not a legacy lease. Keep real blocking/scope
		// conditions, while removing the false missing-legacy-lease diagnosis.
		state.LeaseStatus = d.LeaseStatusActive
		filtered := make([]d.ReadinessReason, 0, len(state.ReadinessReasons))
		for _, r := range state.ReadinessReasons {
			if r != d.ReadinessReasonLeaseExpired {
				filtered = append(filtered, r)
			}
		}
		state.ReadinessReasons = filtered
		if len(filtered) == 0 {
			state.DisplayState = d.WorkItemDisplayStateInProgress
		}
	}
	return state
}

func annotateTaskVerificationReadiness(v *TaskVerificationView, state d.WorkItemOperationalState) {
	if v == nil {
		return
	}
	seen := map[string]bool{}
	for _, reason := range v.ReasonCodes {
		seen[reason] = true
	}
	for _, reason := range state.ReadinessReasons {
		if !seen[string(reason)] {
			v.ReasonCodes = append(v.ReasonCodes, string(reason))
			seen[string(reason)] = true
		}
	}
	v.Coverage["readiness"] = WorkspaceCoverage{Status: "complete"}
}

// GetTaskVerificationView is an authenticated Core query. Protocol adapters are
// registered separately; this method never acquires or creates review authority.
func (s *Service) GetTaskVerificationView(ctx context.Context, scope d.Scope, id d.ID) (TaskVerificationView, error) {
	if id.Validate() != nil {
		return TaskVerificationView{}, d.NewError(d.ErrorCodeInvalidArgument, "invalid Task ID")
	}
	f, handled, err := s.openWorkspaceFrame(ctx, scope)
	if err != nil {
		return TaskVerificationView{}, err
	}
	if !handled {
		return TaskVerificationView{}, d.NewError(d.ErrorCodeInvalidConfig, "coherent Task verification reads are unavailable")
	}
	defer f.u.Rollback()
	focal, ok := f.repo.(ports.FocalReadRepository)
	if !ok {
		return TaskVerificationView{}, d.NewError(d.ErrorCodeInvalidConfig, "current Task reads are unavailable")
	}
	c, err := s.currentFocalWorkContextAt(ctx, f.u, focal, scope, id, f.revision, f.at)
	if err != nil {
		return TaskVerificationView{}, err
	}
	if c.Verification == nil {
		return TaskVerificationView{}, d.NewError(d.ErrorCodeInvalidConfig, "actual-source Task verification reads are unavailable")
	}
	v := *c.Verification
	v.Requirements = verificationCriteriaSummary(c)
	v.Coverage["criteria"] = WorkspaceCoverage{Status: "complete"}
	if err := populateTaskVerificationMaterial(ctx, f.u, &v); err != nil {
		return TaskVerificationView{}, err
	}
	outcome, err := focal.FocalOutcome(ctx, scope)
	if err != nil {
		return TaskVerificationView{}, err
	}
	workspace := EntityWorkspaceView{Subject: WorkspaceSubject{Kind: "entity", Ref: c.Work.Ref()}, EntityVersion: c.Work.Version, Verification: &v}
	actions, err := s.workspaceCapabilities(ctx, f, &workspace, c.Work, &c, outcome)
	if err != nil {
		return TaskVerificationView{}, err
	}
	for _, action := range actions {
		if taskVerificationAction(action.ActionID) {
			v.Actions = append(v.Actions, action)
		}
	}
	v.Coverage["actions"] = WorkspaceCoverage{Status: "complete"}
	if raw, err := json.Marshal(v); err != nil {
		return TaskVerificationView{}, err
	} else if len(raw) > MaxSnapshotBytes {
		return TaskVerificationView{}, d.NewError(d.ErrorCodeGraphLimitExceeded, "Task verification view exceeds byte limit")
	}
	return v, nil
}

func taskVerificationAction(command string) bool {
	switch command {
	case "SubmitTaskForReview", "AcceptTaskDelivery", "RequestTaskDeliveryChanges", "WithdrawTaskDelivery", "AcquireSignedWorkContract", "AcquireSignedReviewContract", "ReturnSignedWork", "ReturnSignedReview", "AcquireWorkContract", "ClaimWorkItem", "ReclaimWorkItem", "ResumeWorkContract":
		return true
	}
	return false
}

func verificationCriteriaSummary(c WorkContext) *WorkspaceCriteriaSummary {
	counts := &WorkspaceCriteriaSummary{}
	for _, criterion := range c.Work.Criteria.Items {
		if criterion.Status != d.CriterionStatusActive {
			continue
		}
		counts.Active++
		if criterion.Required {
			counts.Required++
		}
		a, exists := c.Work.Criteria.CurrentAssessments[criterion.ID]
		if exists && a.CriterionRevision == criterion.Revision && attentionAssessmentApplies(c, a) {
			if a.Result == d.AssessmentResultMet {
				counts.Met++
				continue
			}
			if a.Result == d.AssessmentResultWaived {
				counts.Waived++
				continue
			}
		}
		counts.Pending++
	}
	return counts
}

func populateTaskVerificationMaterial(ctx context.Context, u ports.UnitOfWork, v *TaskVerificationView) error {
	if v.SourceID == nil {
		return nil
	}
	var summary string
	var artifacts []d.SubmissionArtifact
	var evidence []d.ID
	var criteria []d.SubmissionCriterionEvidence
	if v.SourceKind == "task_delivery" {
		reads, ok := u.(ports.TaskDeliveryUnitOfWork)
		if !ok {
			return d.NewError(d.ErrorCodeInvalidConfig, "native delivery material is unavailable")
		}
		delivery, err := reads.TaskDeliveries().Get(ctx, v.Ref.Scope, *v.SourceID)
		if err != nil {
			return err
		}
		if delivery.WorkItemID != v.Ref.ID || delivery.Digest != v.DeliveryDigest || delivery.Round != v.Round {
			return taskVerificationSourceError(v.Ref.Scope, v.Ref.ID, "expanded delivery differs from the verification snapshot")
		}
		if delivery.Material != nil {
			summary, artifacts, evidence, criteria = delivery.Summary, delivery.Material.Artifacts, delivery.Material.EvidenceIDs, delivery.Material.CriterionEvidence
		}
	}
	if v.SubmissionID != nil {
		var repo ports.WorkContractRepository
		var err error
		if v.SourceKind == "signed_review_case" {
			_, repo, err = signedRepository(u)
		} else {
			repo, err = contractRepository(u)
		}
		if err != nil {
			return err
		}
		submission, err := repo.GetSubmission(ctx, v.Ref.Scope, *v.SubmissionID)
		if err != nil {
			return err
		}
		if submission.Scope != v.Ref.Scope || submission.Material.WorkItemID != v.Ref.ID || submission.Digest != v.SubmissionDigest || v.WorkContractID == nil || submission.Material.ContractID != *v.WorkContractID {
			return taskVerificationSourceError(v.Ref.Scope, v.Ref.ID, "expanded submission differs from the verification snapshot")
		}
		summary, artifacts, evidence, criteria = submission.Material.Summary, submission.Material.Artifacts, submission.Material.EvidenceIDs, submission.Material.CriterionEvidence
	}
	v.Material = TaskVerificationMaterial{Summary: workspaceText(summary, 512), ArtifactCount: len(artifacts), EvidenceCount: len(evidence), CriterionCount: len(criteria), Artifacts: append([]d.SubmissionArtifact{}, artifacts[:min(20, len(artifacts))]...), EvidenceIDs: append([]d.ID{}, evidence[:min(20, len(evidence))]...)}
	partial := len(summary) > 512 || len(artifacts) > 20 || len(evidence) > 20 || len(criteria) > 20
	for _, entry := range criteria[:min(20, len(criteria))] {
		if len(entry.EvidenceIDs) > 20 {
			partial = true
		}
		entry.EvidenceIDs = append([]d.ID{}, entry.EvidenceIDs[:min(20, len(entry.EvidenceIDs))]...)
		v.Material.CriterionEvidence = append(v.Material.CriterionEvidence, entry)
	}
	v.Coverage["material"] = WorkspaceCoverage{Status: "complete"}
	if partial {
		v.Coverage["material"] = WorkspaceCoverage{Status: "partial", Reason: "material_preview; expand exact canonical delivery or submission"}
	}
	return nil
}

func verificationAssessmentApplies(v *TaskVerificationView, a d.CriterionAssessment) bool {
	if v == nil || v.SourceKind == "none" {
		return true
	}
	if !v.Current || v.SourceID == nil {
		return false
	}
	if v.SourceKind == "task_delivery" {
		return a.TaskDeliveryRef != nil && a.TaskDeliveryRef.ID == *v.SourceID && a.TaskDeliveryRef.Digest == v.DeliveryDigest && a.TaskDeliveryRef.Round == v.Round &&
			(v.SubmissionID == nil && a.SubmissionID == nil || v.SubmissionID != nil && a.SubmissionID != nil && *v.SubmissionID == *a.SubmissionID && v.SubmissionDigest == a.SubmissionDigest)
	}
	return v.SubmissionID != nil && a.SubmissionID != nil && *v.SubmissionID == *a.SubmissionID && v.SubmissionDigest == a.SubmissionDigest
}
