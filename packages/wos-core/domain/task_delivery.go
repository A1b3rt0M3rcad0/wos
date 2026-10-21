package domain

import (
	"reflect"
	"slices"
	"strings"
	"time"
)

// TaskDeliveryAuthorityKind identifies the protocol that authorized a delivery.
// Signed v2 deliberately has its own delivery and review aggregates; it cannot
// be represented by this unsigned delivery model.
type TaskDeliveryAuthorityKind string

const (
	TaskDeliveryAuthorityLegacyLease TaskDeliveryAuthorityKind = "legacy_lease"
	TaskDeliveryAuthorityContractsV1 TaskDeliveryAuthorityKind = "contracts_v1"
	TaskDeliveryAuthoritySignedV2    TaskDeliveryAuthorityKind = "signed_v2"
)

// TaskDeliveryAuthority records the execution authority that produced a
// delivery. Credential IDs are canonical roots; session IDs are provenance
// only and never transfer ownership.
type TaskDeliveryAuthority struct {
	Kind                TaskDeliveryAuthorityKind `json:"kind"`
	PrincipalID         string                    `json:"principal_id"`
	Actor               ActorRef                  `json:"actor_ref"`
	ClaimID             *ID                       `json:"claim_id,omitempty"`
	ContractID          *ID                       `json:"contract_id,omitempty"`
	ExecutionID         *ID                       `json:"execution_id,omitempty"`
	CredentialID        *ID                       `json:"credential_id,omitempty"`
	SessionCredentialID *ID                       `json:"session_credential_id,omitempty"`
	FencingToken        FencingToken              `json:"fencing_token,string"`
}

func (a TaskDeliveryAuthority) Validate() error {
	if strings.TrimSpace(a.PrincipalID) == "" || a.FencingToken == 0 || a.CredentialID == nil {
		return NewError(ErrorCodeInvalidArgument, "delivery authority requires principal, canonical credential and fencing token")
	}
	if err := a.Actor.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidArgument, "delivery authority actor is invalid", err)
	}
	for _, id := range []*ID{a.ClaimID, a.ContractID, a.ExecutionID, a.CredentialID, a.SessionCredentialID} {
		if id != nil {
			if err := id.Validate(); err != nil {
				return WrapError(ErrorCodeInvalidArgument, "delivery authority reference is invalid", err)
			}
		}
	}
	switch a.Kind {
	case TaskDeliveryAuthorityLegacyLease:
		if a.ClaimID == nil || a.ContractID != nil || a.ExecutionID != nil {
			return NewError(ErrorCodeInvalidArgument, "legacy delivery requires only a lease claim binding")
		}
	case TaskDeliveryAuthorityContractsV1:
		if a.ClaimID != nil || a.ContractID == nil || a.ExecutionID == nil {
			return NewError(ErrorCodeInvalidArgument, "contracts_v1 delivery requires a contract and execution binding")
		}
	case TaskDeliveryAuthoritySignedV2:
		return NewError(ErrorCodeSignedProtocolRequired, "signed-v2 deliveries use the signed review protocol")
	default:
		return NewError(ErrorCodeInvalidArgument, "unknown task delivery authority kind")
	}
	return nil
}

type TaskDeliveryCriterionSnapshot struct {
	CriterionID       ID                `json:"criterion_id"`
	CriterionRevision CriterionRevision `json:"criterion_revision"`
	Required          bool              `json:"required"`
	VerificationMode  VerificationMode  `json:"verification_mode"`
}

// TaskDeliveryMaterial is unsigned, immutable material attached to a legacy
// lease delivery. Contracts v1 deliveries reference their validated
// WorkSubmission instead, preserving that protocol's exact digest.
type TaskDeliveryMaterial struct {
	Summary           string                        `json:"summary"`
	Artifacts         []SubmissionArtifact          `json:"artifacts"`
	EvidenceIDs       []ID                          `json:"evidence_ids"`
	CriterionEvidence []SubmissionCriterionEvidence `json:"criterion_evidence"`
}

func NormalizeTaskDeliveryMaterial(m TaskDeliveryMaterial) TaskDeliveryMaterial {
	if m.Artifacts == nil {
		m.Artifacts = []SubmissionArtifact{}
	}
	if m.EvidenceIDs == nil {
		m.EvidenceIDs = []ID{}
	}
	if m.CriterionEvidence == nil {
		m.CriterionEvidence = []SubmissionCriterionEvidence{}
	}
	for i := range m.CriterionEvidence {
		if m.CriterionEvidence[i].EvidenceIDs == nil {
			m.CriterionEvidence[i].EvidenceIDs = []ID{}
		}
	}
	return m
}

func (m TaskDeliveryMaterial) validate(snapshot []TaskDeliveryCriterionSnapshot) error {
	if strings.TrimSpace(m.Summary) == "" || len(m.Summary) > 16384 || len(m.Artifacts) > 100 || len(m.EvidenceIDs) > 500 || len(m.CriterionEvidence) > 100 {
		return NewError(ErrorCodeInvalidArgument, "delivery material requires a bounded summary and bounded references")
	}
	seenArtifacts := make(map[ID]struct{}, len(m.Artifacts))
	for _, artifact := range m.Artifacts {
		if err := artifact.ArtifactID.Validate(); err != nil {
			return err
		}
		if strings.TrimSpace(artifact.SourceVersion) == "" && strings.TrimSpace(artifact.Checksum) == "" {
			return NewError(ErrorCodeInvalidArgument, "delivery artifact needs an exact source version or checksum")
		}
		if _, exists := seenArtifacts[artifact.ArtifactID]; exists {
			return NewError(ErrorCodeInvalidArgument, "delivery contains a duplicate artifact")
		}
		seenArtifacts[artifact.ArtifactID] = struct{}{}
	}
	seenEvidence := make(map[ID]struct{}, len(m.EvidenceIDs))
	for _, id := range m.EvidenceIDs {
		if err := id.Validate(); err != nil {
			return err
		}
		if _, exists := seenEvidence[id]; exists {
			return NewError(ErrorCodeInvalidArgument, "delivery contains duplicate evidence")
		}
		seenEvidence[id] = struct{}{}
	}
	byCriterion := make(map[ID]TaskDeliveryCriterionSnapshot, len(snapshot))
	for _, criterion := range snapshot {
		byCriterion[criterion.CriterionID] = criterion
	}
	seenCriteria := make(map[ID]struct{}, len(m.CriterionEvidence))
	for _, link := range m.CriterionEvidence {
		criterion, exists := byCriterion[link.CriterionID]
		if !exists || criterion.CriterionRevision != link.CriterionRevision {
			return NewError(ErrorCodeCriterion, "delivery evidence does not match the criterion snapshot")
		}
		if _, duplicate := seenCriteria[link.CriterionID]; duplicate {
			return NewError(ErrorCodeCriterion, "delivery contains duplicate criterion evidence")
		}
		seenCriteria[link.CriterionID] = struct{}{}
		if len(link.EvidenceIDs) == 0 {
			return NewError(ErrorCodeInvalidArgument, "criterion evidence requires at least one delivery evidence reference")
		}
		seenEvidenceForCriterion := make(map[ID]struct{}, len(link.EvidenceIDs))
		for _, id := range link.EvidenceIDs {
			if err := id.Validate(); err != nil {
				return err
			}
			if _, included := seenEvidence[id]; !included {
				return NewError(ErrorCodeInvalidArgument, "criterion evidence must reference evidence included in the delivery")
			}
			if _, duplicate := seenEvidenceForCriterion[id]; duplicate {
				return NewError(ErrorCodeInvalidArgument, "criterion evidence contains a duplicate evidence reference")
			}
			seenEvidenceForCriterion[id] = struct{}{}
		}
	}
	return nil
}

type TaskDeliverySubmissionRef struct {
	ID         ID     `json:"id"`
	ContractID ID     `json:"contract_id"`
	Digest     string `json:"digest"`
}

// TaskDelivery is an immutable handoff of exact work material for review. It
// never records acceptance; TaskDeliveryDecision and Conclusion do that.
type TaskDelivery struct {
	ID                   ID                              `json:"id"`
	Scope                Scope                           `json:"scope"`
	WorkItemID           ID                              `json:"work_item_id"`
	Authority            TaskDeliveryAuthority           `json:"authority"`
	Round                uint64                          `json:"round,string"`
	SupersedesDeliveryID *ID                             `json:"supersedes_delivery_id,omitempty"`
	DefinitionDigest     string                          `json:"definition_digest"`
	CriterionSnapshot    []TaskDeliveryCriterionSnapshot `json:"criterion_snapshot"`
	Material             *TaskDeliveryMaterial           `json:"material,omitempty"`
	SubmissionRef        *TaskDeliverySubmissionRef      `json:"submission_ref,omitempty"`
	Summary              string                          `json:"summary"`
	DeliveredAt          time.Time                       `json:"delivered_at"`
	Digest               string                          `json:"delivery_digest"`
}

// TaskDefinitionDigest identifies the exact task definition currently stored
// on the aggregate. It intentionally excludes mutable execution state,
// assessments, leases, and version counters.
func TaskDefinitionDigest(w WorkItem) (string, error) {
	criteria := append([]SuccessCriterion(nil), w.Criteria.Items...)
	return SemanticDigest(struct {
		Scope         Scope              `json:"scope"`
		WorkItemID    ID                 `json:"work_item_id"`
		ObjectiveID   *ID                `json:"objective_id,omitempty"`
		Title         string             `json:"title"`
		Description   string             `json:"description"`
		Priority      Priority           `json:"priority"`
		ExecutionSpec *ExecutionSpec     `json:"execution_spec,omitempty"`
		Criteria      []SuccessCriterion `json:"criteria"`
	}{w.Scope, w.ID, cloneID(w.ObjectiveID), w.Title, w.Description, w.Priority, cloneExecutionSpec(w.ExecutionSpec), criteria})
}

func taskDeliveryCriterionSnapshot(w WorkItem) []TaskDeliveryCriterionSnapshot {
	snapshot := make([]TaskDeliveryCriterionSnapshot, 0, len(w.Criteria.Items))
	for _, criterion := range w.Criteria.Items {
		if criterion.Status != CriterionStatusActive {
			continue
		}
		snapshot = append(snapshot, TaskDeliveryCriterionSnapshot{
			CriterionID:       criterion.ID,
			CriterionRevision: criterion.Revision,
			Required:          criterion.Required,
			VerificationMode:  criterion.VerificationMode,
		})
	}
	return snapshot
}

func (d TaskDelivery) Validate() error {
	for _, id := range []ID{d.ID, d.WorkItemID} {
		if err := id.Validate(); err != nil {
			return err
		}
	}
	if err := d.Scope.Validate(); err != nil {
		return err
	}
	if err := d.Authority.Validate(); err != nil {
		return err
	}
	if d.Round == 0 || d.DeliveredAt.IsZero() || strings.TrimSpace(d.Summary) == "" || !ValidSignedDigest(d.DefinitionDigest) {
		return NewError(ErrorCodeInvalidArgument, "delivery requires round, definition digest, summary and timestamp")
	}
	if d.Round == 1 {
		if d.SupersedesDeliveryID != nil {
			return NewError(ErrorCodeInvalidArgument, "first TaskDelivery round cannot supersede another delivery")
		}
	} else {
		if d.SupersedesDeliveryID == nil || d.SupersedesDeliveryID.Validate() != nil || *d.SupersedesDeliveryID == d.ID {
			return NewError(ErrorCodeInvalidArgument, "correction TaskDelivery round requires a valid prior delivery reference")
		}
	}
	if len(d.CriterionSnapshot) > 100 {
		return NewError(ErrorCodeInvalidArgument, "delivery criterion snapshot exceeds 100 items")
	}
	seen := make(map[ID]struct{}, len(d.CriterionSnapshot))
	for _, criterion := range d.CriterionSnapshot {
		if criterion.CriterionID.Validate() != nil || criterion.CriterionRevision < InitialCriterionRevision || !criterion.VerificationMode.Valid() {
			return NewError(ErrorCodeInvalidArgument, "delivery criterion snapshot is invalid")
		}
		if _, duplicate := seen[criterion.CriterionID]; duplicate {
			return NewError(ErrorCodeInvalidArgument, "delivery criterion snapshot contains duplicates")
		}
		seen[criterion.CriterionID] = struct{}{}
	}
	if (d.Material == nil) == (d.SubmissionRef == nil) {
		return NewError(ErrorCodeInvalidArgument, "delivery must contain either inline material or one submission reference")
	}
	switch d.Authority.Kind {
	case TaskDeliveryAuthorityLegacyLease:
		if d.Material == nil {
			return NewError(ErrorCodeInvalidArgument, "legacy lease delivery requires inline material")
		}
		if err := NormalizeTaskDeliveryMaterial(*d.Material).validate(d.CriterionSnapshot); err != nil {
			return err
		}
	case TaskDeliveryAuthorityContractsV1:
		if d.SubmissionRef == nil || d.SubmissionRef.ID.Validate() != nil || d.SubmissionRef.ContractID.Validate() != nil || !ValidSignedDigest(d.SubmissionRef.Digest) {
			return NewError(ErrorCodeInvalidArgument, "contracts_v1 delivery requires an exact WorkSubmission reference")
		}
		if d.Authority.ContractID == nil || d.SubmissionRef.ContractID != *d.Authority.ContractID {
			return NewError(ErrorCodeContractSpecMismatch, "contracts_v1 SubmissionRef must identify its exact contract")
		}
	case TaskDeliveryAuthoritySignedV2:
		return NewError(ErrorCodeSignedProtocolRequired, "signed-v2 deliveries use the signed review protocol")
	}
	if d.Material != nil && NormalizeTaskDeliveryMaterial(*d.Material).Summary != d.Summary {
		return NewError(ErrorCodeContractSpecMismatch, "delivery summary differs from its material")
	}
	digest, err := d.semanticDigest()
	if err != nil {
		return err
	}
	if digest != d.Digest {
		return NewError(ErrorCodeContractSpecMismatch, "task delivery digest mismatch")
	}
	return nil
}

func (d TaskDelivery) semanticDigest() (string, error) {
	return SemanticDigest(struct {
		ID                   ID                              `json:"id"`
		Scope                Scope                           `json:"scope"`
		WorkItemID           ID                              `json:"work_item_id"`
		Authority            TaskDeliveryAuthority           `json:"authority"`
		Round                uint64                          `json:"round,string"`
		SupersedesDeliveryID *ID                             `json:"supersedes_delivery_id,omitempty"`
		DefinitionDigest     string                          `json:"definition_digest"`
		CriterionSnapshot    []TaskDeliveryCriterionSnapshot `json:"criterion_snapshot"`
		Material             *TaskDeliveryMaterial           `json:"material,omitempty"`
		SubmissionRef        *TaskDeliverySubmissionRef      `json:"submission_ref,omitempty"`
		Summary              string                          `json:"summary"`
		DeliveredAt          time.Time                       `json:"delivered_at"`
	}{d.ID, d.Scope, d.WorkItemID, d.Authority, d.Round, cloneID(d.SupersedesDeliveryID), d.DefinitionDigest, d.CriterionSnapshot, normalizeDeliveryMaterialPointer(d.Material), d.SubmissionRef, d.Summary, d.DeliveredAt.UTC()})
}

func normalizeDeliveryMaterialPointer(m *TaskDeliveryMaterial) *TaskDeliveryMaterial {
	if m == nil {
		return nil
	}
	normalized := NormalizeTaskDeliveryMaterial(*m)
	return &normalized
}

func (d *TaskDelivery) seal() error {
	// Keep the canonical no-criteria snapshot as an empty slice. ValidateForWork
	// compares the frozen snapshot with the current active criteria, whose
	// zero-value representation is non-nil empty.
	d.CriterionSnapshot = append([]TaskDeliveryCriterionSnapshot{}, d.CriterionSnapshot...)
	if d.Material != nil {
		material := NormalizeTaskDeliveryMaterial(*d.Material)
		d.Material = &material
	}
	digest, err := d.semanticDigest()
	if err != nil {
		return err
	}
	d.Digest = digest
	return d.Validate()
}

func (d TaskDelivery) ValidateForWork(w WorkItem) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if d.Scope != w.Scope || d.WorkItemID != w.ID {
		return NewError(ErrorCodeInvalidScope, "delivery does not belong to this Task")
	}
	definitionDigest, err := TaskDefinitionDigest(w)
	if err != nil {
		return err
	}
	if definitionDigest != d.DefinitionDigest || !reflect.DeepEqual(taskDeliveryCriterionSnapshot(w), d.CriterionSnapshot) {
		return NewError(ErrorCodeContractSpecMismatch, "Task definition or criterion snapshot changed after delivery")
	}
	return nil
}

func sameOptionalID(a, b *ID) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

func taskDeliveryAuthorityMatchesLease(a TaskDeliveryAuthority, l *WorkLease) bool {
	if l == nil || l.CredentialBindingVersion != WorkLeaseCredentialBindingV1 || a.Kind != TaskDeliveryAuthorityLegacyLease || a.ClaimID == nil || *a.ClaimID != l.ClaimID || uint64(a.FencingToken) != l.FencingToken || a.PrincipalID != l.PrincipalID || a.Actor != l.Actor {
		return false
	}
	return a.CredentialID != nil && *a.CredentialID == l.HolderCredentialID && sameOptionalID(a.SessionCredentialID, l.SessionCredentialID)
}

func taskDeliveryAuthorityMatchesContract(a TaskDeliveryAuthority, c WorkContract) bool {
	if a.Kind != TaskDeliveryAuthorityContractsV1 || a.ContractID == nil || *a.ContractID != c.ID || a.ExecutionID == nil || *a.ExecutionID != c.ExecutionID || a.FencingToken != c.FencingToken || a.PrincipalID != c.HolderPrincipalID || a.Actor != c.Actor {
		return false
	}
	if c.CredentialBindingVersion != WorkContractCredentialBindingV1 {
		return false
	}
	return a.CredentialID != nil && *a.CredentialID == c.HolderCredentialID && sameOptionalID(a.SessionCredentialID, c.SessionCredentialID)
}

func validateContractV1Submission(c WorkContract, p WorkSubmission, delivery TaskDelivery) error {
	if c.SignedBinding != nil {
		return NewError(ErrorCodeSignedProtocolRequired, "signed-v2 delivery cannot use the unsigned TaskDelivery model")
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if c.LatestSubmissionID == nil || *c.LatestSubmissionID != p.ID || delivery.SubmissionRef == nil || delivery.SubmissionRef.ID != p.ID || delivery.SubmissionRef.ContractID != c.ID || delivery.SubmissionRef.Digest != p.Digest || p.ProtocolVersion == 2 || p.Scope != c.Scope || p.Material.ContractID != c.ID || p.Material.WorkItemID != c.WorkItemID || p.Material.SpecDigest != c.SpecDigest || p.Material.Summary != delivery.Summary || p.PrincipalID != c.HolderPrincipalID || p.Actor != c.Actor {
		return NewError(ErrorCodeContractSpecMismatch, "delivery does not reference the exact current contracts_v1 submission")
	}
	return nil
}

func contractV1DefinitionMatchesWorkItem(c WorkContract, w WorkItem) bool {
	if c.Spec.Title != w.Title || c.Spec.Description != w.Description || !sameOptionalID(c.Spec.ObjectiveID, w.ObjectiveID) || !slices.Equal(c.Spec.Criteria, w.Criteria.Items) {
		return false
	}
	expected := ExecutionSpec{}
	if w.ExecutionSpec != nil {
		expected = *cloneExecutionSpec(w.ExecutionSpec)
	}
	actual := c.Spec.ExecutionSpec
	return executionSpecsEqual(&actual, &expected)
}

func taskDeliveryStatusError() error {
	return NewError(ErrorCodeInvalidTransition, "Task must be in progress without another pending delivery or review")
}

// RecordTaskDelivery removes the executor's legacy lease or contracts_v1
// authority and records the exact pending handoff. A contracts_v1 contract
// must first be closed by DeliverV1; signed-v2 records cannot enter this path.
func (w *WorkItem) RecordTaskDelivery(d TaskDelivery, contract *WorkContract, submission *WorkSubmission, expected Version, now time.Time) error {
	if w.Version != expected {
		return NewError(ErrorCodeVersionConflict, "Task version changed before delivery")
	}
	if w.Lifecycle != WorkItemLifecycleInProgress || w.PendingDeliveryID != nil || w.PendingReviewCaseID != nil || w.DeliveryRound == ^uint64(0) {
		return taskDeliveryStatusError()
	}
	if d.Round != w.DeliveryRound+1 || !sameOptionalID(d.SupersedesDeliveryID, w.LatestDeliveryID) || d.DeliveredAt.After(now) {
		return NewError(ErrorCodeVersionConflict, "delivery round, predecessor, or timestamp does not match current Task state")
	}
	if err := d.ValidateForWork(*w); err != nil {
		return err
	}
	switch d.Authority.Kind {
	case TaskDeliveryAuthorityLegacyLease:
		if w.ContractsEnabled || contract != nil || submission != nil || !taskDeliveryAuthorityMatchesLease(d.Authority, w.CurrentLease) {
			return NewError(ErrorCodeLease, "delivery does not match the current legacy lease")
		}
		if w.CurrentLease == nil || w.CurrentLease.CredentialBindingVersion != WorkLeaseCredentialBindingV1 || !w.CurrentLease.ValidAt(now) {
			return NewError(ErrorCodeLease, "legacy lease expired before delivery")
		}
		if d.Authority.CredentialID == nil {
			return NewError(ErrorCodeLease, "credential-bound lease delivery lost its holder binding")
		}
		if err := w.CheckCompletionAuthorityForCredential(d.Authority.PrincipalID, *d.Authority.CredentialID, *d.Authority.ClaimID, uint64(d.Authority.FencingToken), now); err != nil {
			return err
		}
	case TaskDeliveryAuthorityContractsV1:
		if !w.ContractsEnabled || contract == nil || submission == nil || w.CurrentLease != nil || w.CurrentContractID == nil || d.Authority.ContractID == nil || *w.CurrentContractID != *d.Authority.ContractID {
			return NewError(ErrorCodeContractSpecMismatch, "contracts_v1 delivery does not match the current Task contract")
		}
		if contract.CredentialBindingVersion != WorkContractCredentialBindingV1 || contract.Status != ContractCompleted || contract.CloseReason != ContractV1DeliveryCloseReason || contract.ClosedAt == nil || !contract.ClosedAt.Equal(d.DeliveredAt) || contract.ClosedBy != d.Authority.PrincipalID || !taskDeliveryAuthorityMatchesContract(d.Authority, *contract) {
			return NewError(ErrorCodeContractSpecMismatch, "contracts_v1 authority was not closed for this delivery")
		}
		if !contractV1DefinitionMatchesWorkItem(*contract, *w) {
			return NewError(ErrorCodeContractSpecMismatch, "contracts_v1 snapshot differs from the delivered Task definition")
		}
		if err := validateContractV1Submission(*contract, *submission, d); err != nil {
			return err
		}
	default:
		return NewError(ErrorCodeSignedProtocolRequired, "signed-v2 deliveries use the signed review protocol")
	}
	nextVersion, err := w.Version.Next()
	if err != nil {
		return err
	}
	next := *w
	next.PendingDeliveryID = cloneID(&d.ID)
	next.LatestDeliveryID = cloneID(&d.ID)
	next.LatestDeliveryDecisionID = nil
	next.LatestDeliveryWithdrawalID = nil
	next.TaskDeliveryReplanAuthorized = false
	next.DeliveryRound = d.Round
	next.CurrentLease = nil
	next.CurrentContractID = nil
	next.Version = nextVersion
	next.UpdatedAt = now.UTC()
	if err := next.Validate(); err != nil {
		return err
	}
	*w = next
	return nil
}

func (w WorkItem) validatePendingTaskDelivery(d TaskDelivery, decision TaskDeliveryDecision, expected Version) error {
	if w.Version != expected {
		return NewError(ErrorCodeVersionConflict, "Task version changed before delivery decision")
	}
	if w.Lifecycle != WorkItemLifecycleInProgress || w.PendingDeliveryID == nil || *w.PendingDeliveryID != d.ID || w.LatestDeliveryID == nil || *w.LatestDeliveryID != d.ID || d.Round != w.DeliveryRound {
		return NewError(ErrorCodeSubmissionNotAccepted, "decision does not target the pending TaskDelivery")
	}
	if err := d.ValidateForWork(w); err != nil {
		return err
	}
	return decision.ValidateForDelivery(d)
}

// AcceptTaskDelivery records review approval and the Task conclusion without
// reacquiring the executor's closed lease. Criteria still require explicit,
// current assessments; review approval alone does not synthesize them.
func (w *WorkItem) AcceptTaskDelivery(d TaskDelivery, decision TaskDeliveryDecision, submission *WorkSubmission, acceptedAssessmentRefs []CriterionAssessmentRef, expected Version, conclusion Conclusion, now time.Time) error {
	if err := w.validatePendingTaskDelivery(d, decision, expected); err != nil {
		return err
	}
	if decision.Result != TaskDeliveryDecisionAccepted {
		return NewError(ErrorCodeInvalidTransition, "only an accepted delivery can conclude a Task")
	}
	if !decision.DecidedAt.Equal(now) || conclusion.PrincipalID != decision.ReviewerPrincipalID || conclusion.Actor != decision.ReviewerActor || conclusion.Reason != decision.Reason || !conclusion.ConcludedAt.Equal(decision.DecidedAt) {
		return NewError(ErrorCodeInvalidArgument, "Task conclusion must match the accepted delivery decision")
	}
	assessmentRefs := append([]CriterionAssessmentRef(nil), acceptedAssessmentRefs...)
	if err := validateDeliveryAssessmentRefs(*w, assessmentRefs); err != nil {
		return err
	}
	if !w.Criteria.HasRequiredActive() && strings.TrimSpace(d.Summary) == "" {
		return NewError(ErrorCodePreconditionFailed, "Task without required criteria needs a delivery summary")
	}
	if d.SubmissionRef != nil {
		if submission == nil {
			return NewError(ErrorCodeSubmissionNotAccepted, "contracts_v1 acceptance requires its exact WorkSubmission")
		}
		if err := submission.Validate(); err != nil {
			return err
		}
		if submission.ID != d.SubmissionRef.ID || submission.Material.ContractID != d.SubmissionRef.ContractID || submission.Digest != d.SubmissionRef.Digest || submission.Material.Summary != d.Summary || submission.Scope != w.Scope || submission.Material.WorkItemID != w.ID {
			return NewError(ErrorCodeSubmissionNotAccepted, "acceptance material differs from the reviewed contracts_v1 submission")
		}
		id := submission.ID
		conclusion.SubmissionID = &id
		conclusion.SubmissionDigest = submission.Digest
	} else {
		if submission != nil || d.Material == nil {
			return NewError(ErrorCodeSubmissionNotAccepted, "legacy acceptance must target its inline delivery material")
		}
		if conclusion.SubmissionID != nil || conclusion.SubmissionDigest != "" {
			return NewError(ErrorCodeInvalidArgument, "legacy TaskDelivery is not a WorkSubmission")
		}
	}
	conclusion.Assessments = assessmentRefs
	nextVersion, err := w.Version.Next()
	if err != nil {
		return err
	}
	if err := conclusion.Bind(w.Ref(), nextVersion, string(WorkItemLifecycleDone), ConclusionObligations{RequiredCriteria: w.Criteria.RequiredObligations(), ResultSummaryRequired: !w.Criteria.HasRequiredActive()}); err != nil {
		return err
	}
	next := *w
	next.Version = nextVersion
	next.Lifecycle = WorkItemLifecycleDone
	next.ResultSummary = d.Summary
	next.PendingDeliveryID = nil
	next.LatestDeliveryDecisionID = cloneID(&decision.ID)
	next.LatestDeliveryWithdrawalID = nil
	next.TaskDeliveryReplanAuthorized = false
	next.CurrentLease = nil
	next.CurrentContractID = nil
	next.CurrentConclusion = &conclusion
	next.UpdatedAt = now.UTC()
	if err := next.Validate(); err != nil {
		return err
	}
	*w = next
	return nil
}

// RequestTaskDeliveryChanges returns the Task to To do for a new, separately
// authorized execution generation. It does not hand the previous lease back.
func (w *WorkItem) RequestTaskDeliveryChanges(d TaskDelivery, decision TaskDeliveryDecision, expected Version, now time.Time) error {
	if err := w.validatePendingTaskDelivery(d, decision, expected); err != nil {
		return err
	}
	if decision.Result != TaskDeliveryDecisionChangesRequested {
		return NewError(ErrorCodeInvalidTransition, "only a changes_requested decision can return a Task to To do")
	}
	if decision.DecidedAt.After(now) {
		return NewError(ErrorCodeInvalidArgument, "delivery decision cannot be in the future")
	}
	nextVersion, err := w.Version.Next()
	if err != nil {
		return err
	}
	next := *w
	next.Version = nextVersion
	next.Lifecycle = WorkItemLifecycleTodo
	next.PendingDeliveryID = nil
	next.LatestDeliveryDecisionID = cloneID(&decision.ID)
	next.LatestDeliveryWithdrawalID = nil
	next.TaskDeliveryReplanAuthorized = false
	next.CurrentLease = nil
	next.CurrentContractID = nil
	next.UpdatedAt = now.UTC()
	if err := next.Validate(); err != nil {
		return err
	}
	*w = next
	return nil
}

// WithdrawTaskDelivery closes only the exact current unsigned delivery and
// returns its Task to To do. It records no reviewer decision and grants no
// execution authority; replanning still requires an explicit defer to Backlog.
func (w *WorkItem) WithdrawTaskDelivery(d TaskDelivery, withdrawal TaskDeliveryWithdrawal, expected Version, now time.Time) error {
	if w.Version != expected {
		return NewError(ErrorCodeVersionConflict, "Task version changed before delivery withdrawal")
	}
	if w.Lifecycle != WorkItemLifecycleInProgress || w.PendingDeliveryID == nil || *w.PendingDeliveryID != d.ID || w.LatestDeliveryID == nil || *w.LatestDeliveryID != d.ID || w.DeliveryRound != d.Round || w.LatestDeliveryDecisionID != nil || w.LatestDeliveryWithdrawalID != nil {
		return NewError(ErrorCodeSubmissionNotAccepted, "withdrawal does not target the current pending TaskDelivery")
	}
	if err := d.ValidateForWork(*w); err != nil {
		return err
	}
	if err := withdrawal.ValidateForDelivery(d); err != nil {
		return err
	}
	if withdrawal.WithdrawnAt.After(now) {
		return NewError(ErrorCodeInvalidArgument, "TaskDelivery withdrawal cannot be in the future")
	}
	nextVersion, err := w.Version.Next()
	if err != nil {
		return err
	}
	next := *w
	next.Version = nextVersion
	next.Lifecycle = WorkItemLifecycleTodo
	next.PendingDeliveryID = nil
	next.LatestDeliveryDecisionID = nil
	next.LatestDeliveryWithdrawalID = cloneID(&withdrawal.ID)
	next.TaskDeliveryReplanAuthorized = false
	next.CurrentLease = nil
	next.CurrentContractID = nil
	next.UpdatedAt = now.UTC()
	if err := next.Validate(); err != nil {
		return err
	}
	*w = next
	return nil
}

func validateDeliveryAssessmentRefs(w WorkItem, refs []CriterionAssessmentRef) error {
	seen := make(map[ID]struct{}, len(refs))
	seenAssessments := make(map[ID]struct{}, len(refs))
	for _, ref := range refs {
		if err := ref.AssessmentID.Validate(); err != nil {
			return err
		}
		if err := ref.CriterionID.Validate(); err != nil {
			return err
		}
		if ref.CriterionRevision < InitialCriterionRevision || ref.Result != AssessmentResultMet {
			return NewError(ErrorCodeAssessment, "accepted assessment reference must be current and met")
		}
		if _, duplicate := seen[ref.CriterionID]; duplicate {
			return NewError(ErrorCodeAssessment, "accepted assessment references contain a duplicate criterion")
		}
		seen[ref.CriterionID] = struct{}{}
		if _, duplicate := seenAssessments[ref.AssessmentID]; duplicate {
			return NewError(ErrorCodeAssessment, "accepted assessment references contain a duplicate assessment")
		}
		seenAssessments[ref.AssessmentID] = struct{}{}
		criterion, err := w.Criteria.Find(ref.CriterionID)
		if err != nil {
			return err
		}
		if criterion.Status != CriterionStatusActive || criterion.Revision != ref.CriterionRevision {
			return NewError(ErrorCodeAssessment, "accepted assessment reference is stale or inactive")
		}
	}
	for _, criterion := range w.Criteria.Items {
		if criterion.Status != CriterionStatusActive || !criterion.Required {
			continue
		}
		if _, present := seen[criterion.ID]; !present {
			return NewError(ErrorCodeAssessment, "accepted assessment references must include every required active criterion")
		}
	}
	return nil
}

// NewLegacyTaskDelivery snapshots material while the legacy lease is still
// active. The application layer remains responsible for independent review
// authorization and idempotency.
func NewLegacyTaskDelivery(id ID, w WorkItem, material TaskDeliveryMaterial, principal string, credentialID *ID, now time.Time) (TaskDelivery, error) {
	var zero TaskDelivery
	if w.ContractsEnabled {
		return zero, NewError(ErrorCodeContractProtocolRequired, "use contracts_v1 or signed-v2 delivery")
	}
	if w.Lifecycle != WorkItemLifecycleInProgress || w.CurrentLease == nil {
		return zero, taskDeliveryStatusError()
	}
	lease := *w.CurrentLease
	if lease.CredentialBindingVersion != WorkLeaseCredentialBindingV1 || credentialID == nil || *credentialID != lease.HolderCredentialID {
		return zero, NewError(ErrorCodeLease, "TaskDelivery requires a credential-bound legacy lease")
	}
	if err := w.CheckCompletionAuthorityForCredential(principal, *credentialID, lease.ClaimID, lease.FencingToken, now); err != nil {
		return zero, err
	}
	definitionDigest, err := TaskDefinitionDigest(w)
	if err != nil {
		return zero, err
	}
	authority := TaskDeliveryAuthority{Kind: TaskDeliveryAuthorityLegacyLease, PrincipalID: lease.PrincipalID, Actor: lease.Actor, ClaimID: cloneID(&lease.ClaimID), FencingToken: FencingToken(lease.FencingToken), SessionCredentialID: cloneID(lease.SessionCredentialID)}
	authority.CredentialID = cloneID(&lease.HolderCredentialID)
	d := TaskDelivery{ID: id, Scope: w.Scope, WorkItemID: w.ID, Authority: authority, Round: w.DeliveryRound + 1, SupersedesDeliveryID: cloneID(w.LatestDeliveryID), DefinitionDigest: definitionDigest, CriterionSnapshot: taskDeliveryCriterionSnapshot(w), Material: &material, Summary: strings.TrimSpace(material.Summary), DeliveredAt: now.UTC()}
	if err := d.seal(); err != nil {
		return zero, err
	}
	return d, nil
}

// NewContractsV1TaskDelivery freezes the exact unsigned submission before the
// contract is closed for review. Signed-v2 records are rejected explicitly.
func NewContractsV1TaskDelivery(id ID, w WorkItem, c WorkContract, p WorkSubmission, principal string, credentialID *ID, now time.Time) (TaskDelivery, error) {
	var zero TaskDelivery
	if c.SignedBinding != nil || p.ProtocolVersion == 2 {
		return zero, NewError(ErrorCodeSignedProtocolRequired, "signed-v2 delivery uses signed review records")
	}
	if !w.ContractsEnabled || w.Lifecycle != WorkItemLifecycleInProgress || w.PendingReviewCaseID != nil || w.CurrentContractID == nil || *w.CurrentContractID != c.ID || c.Status != ContractActive || c.CredentialBindingVersion != WorkContractCredentialBindingV1 {
		return zero, taskDeliveryStatusError()
	}
	if credentialID == nil || *credentialID != c.HolderCredentialID {
		return zero, NewError(ErrorCodeLease, "credential-bound contract requires credential identity")
	}
	if err := c.AuthorizeForCredential(principal, *credentialID, c.ExecutionID, c.FencingToken, now); err != nil {
		return zero, err
	}
	if err := validateContractV1SubmissionForConstruction(c, p); err != nil {
		return zero, err
	}
	if !contractV1DefinitionMatchesWorkItem(c, w) {
		return zero, NewError(ErrorCodeContractSpecMismatch, "contracts_v1 snapshot differs from the current Task definition")
	}
	definitionDigest, err := TaskDefinitionDigest(w)
	if err != nil {
		return zero, err
	}
	authority := TaskDeliveryAuthority{Kind: TaskDeliveryAuthorityContractsV1, PrincipalID: c.HolderPrincipalID, Actor: c.Actor, ContractID: cloneID(&c.ID), ExecutionID: cloneID(&c.ExecutionID), FencingToken: c.FencingToken, SessionCredentialID: cloneID(c.SessionCredentialID)}
	authority.CredentialID = cloneID(&c.HolderCredentialID)
	d := TaskDelivery{ID: id, Scope: w.Scope, WorkItemID: w.ID, Authority: authority, Round: w.DeliveryRound + 1, SupersedesDeliveryID: cloneID(w.LatestDeliveryID), DefinitionDigest: definitionDigest, CriterionSnapshot: taskDeliveryCriterionSnapshot(w), SubmissionRef: &TaskDeliverySubmissionRef{ID: p.ID, ContractID: c.ID, Digest: p.Digest}, Summary: p.Material.Summary, DeliveredAt: now.UTC()}
	if err := d.seal(); err != nil {
		return zero, err
	}
	return d, nil
}

func validateContractV1SubmissionForConstruction(c WorkContract, p WorkSubmission) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if c.LatestSubmissionID == nil || *c.LatestSubmissionID != p.ID || p.ProtocolVersion == 2 || p.Scope != c.Scope || p.Material.ContractID != c.ID || p.Material.WorkItemID != c.WorkItemID || p.Material.SpecDigest != c.SpecDigest || p.PrincipalID != c.HolderPrincipalID || p.Actor != c.Actor {
		return NewError(ErrorCodeContractSpecMismatch, "delivery requires the exact current contracts_v1 submission")
	}
	return nil
}

// ValidateTaskDeliveryMutation enforces the delivery's immutable record
// contract for persistence adapters.
func ValidateTaskDeliveryMutation(old, next TaskDelivery) error {
	if !reflect.DeepEqual(old, next) {
		return NewError(ErrorCodeInvalidTransition, "TaskDelivery is immutable")
	}
	return next.Validate()
}
