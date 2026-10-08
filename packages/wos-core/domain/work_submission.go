package domain

import (
	"strings"
	"time"
)

type WorkCheckpoint struct {
	ID            ID        `json:"id"`
	Scope         Scope     `json:"scope"`
	ContractID    ID        `json:"contract_id"`
	WorkItemID    ID        `json:"work_item_id"`
	Sequence      uint64    `json:"sequence"`
	PrincipalID   string    `json:"principal_id"`
	AcceptedAt    time.Time `json:"accepted_at"`
	Summary       string    `json:"summary"`
	Completed     []string  `json:"completed"`
	Pending       []string  `json:"pending"`
	NextAction    string    `json:"next_action"`
	ArtifactRefs  []ID      `json:"artifact_refs"`
	EvidenceRefs  []ID      `json:"evidence_refs"`
	IssueRefs     []ID      `json:"issue_refs"`
	RepositoryRef string    `json:"repository_ref"`
	WorkingCommit string    `json:"working_commit"`
	Dirty         bool      `json:"dirty"`
	Unknown       []string  `json:"unknown"`
}

func (p WorkCheckpoint) Validate() error {
	for _, id := range []ID{p.ID, p.ContractID, p.WorkItemID} {
		if err := id.Validate(); err != nil {
			return err
		}
	}
	if err := p.Scope.Validate(); err != nil {
		return err
	}
	if p.Sequence == 0 || strings.TrimSpace(p.PrincipalID) == "" || p.AcceptedAt.IsZero() || strings.TrimSpace(p.Summary) == "" {
		return NewError(ErrorCodeInvalidArgument, "checkpoint requires sequence, provenance and summary")
	}
	for _, ids := range [][]ID{p.ArtifactRefs, p.EvidenceRefs, p.IssueRefs} {
		for _, id := range ids {
			if err := id.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

type SubmissionArtifact struct {
	ArtifactID    ID     `json:"artifact_id"`
	SourceVersion string `json:"source_version"`
	Checksum      string `json:"checksum"`
}
type SubmissionCriterionEvidence struct {
	CriterionID       ID                `json:"criterion_id"`
	CriterionRevision CriterionRevision `json:"criterion_revision"`
	EvidenceIDs       []ID              `json:"evidence_ids"`
}
type WorkResultMaterial struct {
	ContractID        ID                            `json:"contract_id"`
	WorkItemID        ID                            `json:"work_item_id"`
	SpecDigest        string                        `json:"spec_digest"`
	Summary           string                        `json:"summary"`
	Artifacts         []SubmissionArtifact          `json:"artifacts"`
	EvidenceIDs       []ID                          `json:"evidence_ids"`
	CriterionEvidence []SubmissionCriterionEvidence `json:"criterion_evidence"`
}

func NormalizeResultMaterial(m WorkResultMaterial) WorkResultMaterial {
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

type WorkSubmission struct {
	ProtocolVersion        int                `json:"protocol_version,omitempty"`
	ID                     ID                 `json:"id"`
	Scope                  Scope              `json:"scope"`
	Material               WorkResultMaterial `json:"material"`
	Digest                 string             `json:"submission_digest"`
	PrincipalID            string             `json:"principal_id"`
	Actor                  ActorRef           `json:"actor_ref"`
	SubmittedAt            time.Time          `json:"submitted_at"`
	SupersedesSubmissionID *ID                `json:"supersedes_submission_id,omitempty"`
}

func (p WorkSubmission) Validate() error {
	for _, id := range []ID{p.ID, p.Material.ContractID, p.Material.WorkItemID} {
		if err := id.Validate(); err != nil {
			return err
		}
	}
	if err := p.Scope.Validate(); err != nil {
		return err
	}
	if err := p.Actor.Validate(); err != nil {
		return err
	}
	if p.SubmittedAt.IsZero() || strings.TrimSpace(p.PrincipalID) == "" || strings.TrimSpace(p.Material.Summary) == "" {
		return NewError(ErrorCodeInvalidArgument, "submission provenance and summary required")
	}
	for _, a := range p.Material.Artifacts {
		if err := a.ArtifactID.Validate(); err != nil {
			return err
		}
		if strings.TrimSpace(a.SourceVersion) == "" && strings.TrimSpace(a.Checksum) == "" {
			return NewError(ErrorCodeInvalidArgument, "submission artifact needs exact source version or checksum")
		}
	}
	for _, id := range p.Material.EvidenceIDs {
		if err := id.Validate(); err != nil {
			return err
		}
	}
	for _, link := range p.Material.CriterionEvidence {
		if err := link.CriterionID.Validate(); err != nil {
			return err
		}
		if link.CriterionRevision < 1 {
			return NewError(ErrorCodeCriterion, "invalid submission criterion revision")
		}
		for _, id := range link.EvidenceIDs {
			if err := id.Validate(); err != nil {
				return err
			}
		}
	}
	if p.ProtocolVersion != 0 && p.ProtocolVersion != 2 {
		return NewError(ErrorCodeInvalidArgument, "unsupported submission protocol")
	}
	digest, err := SemanticDigest(NormalizeResultMaterial(p.Material))
	if p.ProtocolVersion == 2 {
		digest, err = SignedResultDigest(p.Material)
	}
	if err != nil {
		return err
	}
	if digest != p.Digest {
		return NewError(ErrorCodeContractSpecMismatch, "submission digest mismatch")
	}
	if p.SupersedesSubmissionID != nil {
		if err := p.SupersedesSubmissionID.Validate(); err != nil {
			return err
		}
		if *p.SupersedesSubmissionID == p.ID {
			return NewError(ErrorCodeInvalidArgument, "submission cannot supersede itself")
		}
	}
	return nil
}
func (c *WorkContract) RecordCheckpoint(p WorkCheckpoint, principal string, execution ID, token FencingToken, expected Version, now time.Time) error {
	if err := c.Authorize(principal, execution, token, now); err != nil {
		return err
	}
	if expected != c.Version {
		return NewError(ErrorCodeVersionConflict, "contract version conflict")
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if p.Scope != c.Scope || p.ContractID != c.ID || p.WorkItemID != c.WorkItemID || p.PrincipalID != principal {
		return NewError(ErrorCodeInvalidScope, "checkpoint binding mismatch")
	}
	next, err := c.Version.Next()
	if err != nil {
		return err
	}
	id := p.ID
	c.LatestCheckpointID = &id
	c.Version = next
	return nil
}
func (c *WorkContract) Submit(p WorkSubmission, principal string, execution ID, token FencingToken, expected Version, now time.Time) error {
	if err := c.Authorize(principal, execution, token, now); err != nil {
		return err
	}
	if expected != c.Version {
		return NewError(ErrorCodeVersionConflict, "contract version conflict")
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if p.Scope != c.Scope || p.Material.ContractID != c.ID || p.Material.WorkItemID != c.WorkItemID || p.PrincipalID != principal || p.Material.SpecDigest != c.SpecDigest {
		return NewError(ErrorCodeContractSpecMismatch, "submission binding mismatch")
	}
	if c.LatestSubmissionID != nil && (p.SupersedesSubmissionID == nil || *p.SupersedesSubmissionID != *c.LatestSubmissionID) {
		return NewError(ErrorCodeVersionConflict, "new submission must explicitly supersede latest")
	}
	if c.LatestSubmissionID == nil && p.SupersedesSubmissionID != nil {
		return NewError(ErrorCodeInvalidArgument, "first submission has no predecessor")
	}
	next, err := c.Version.Next()
	if err != nil {
		return err
	}
	id := p.ID
	c.LatestSubmissionID = &id
	c.Version = next
	return nil
}

// Finalize requires Application to validate live blockers, dependencies and evidence.
// The Domain binds every required assessment to the exact submitted material.
func (c *WorkContract) Finalize(w *WorkItem, p WorkSubmission, principal string, execution ID, token FencingToken, expected, expectedWork Version, conclusion Conclusion, now time.Time) error {
	if err := c.Authorize(principal, execution, token, now); err != nil {
		return err
	}
	if c.Version != expected || w.Version != expectedWork {
		return NewError(ErrorCodeVersionConflict, "contract or work version conflict")
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if c.LatestSubmissionID == nil || *c.LatestSubmissionID != p.ID || p.Scope != c.Scope || p.Material.ContractID != c.ID || p.Material.WorkItemID != w.ID || w.Scope != c.Scope || p.Material.SpecDigest != c.SpecDigest || w.CurrentContractID == nil || *w.CurrentContractID != c.ID || w.Lifecycle != WorkItemLifecycleInProgress {
		return NewError(ErrorCodeSubmissionNotAccepted, "submission is not the current contract result")
	}
	refs, err := w.Criteria.RequiredSatisfied()
	if err != nil && w.Criteria.HasRequiredActive() {
		return err
	}
	for _, ref := range refs {
		a := w.Criteria.CurrentAssessments[ref.CriterionID]
		if a.SubmissionID == nil || *a.SubmissionID != p.ID || a.SubmissionDigest != p.Digest {
			return NewError(ErrorCodeSubmissionNotAccepted, "assessment is not bound to submitted material")
		}
	}
	conclusion.Assessments = refs
	id := p.ID
	conclusion.SubmissionID = &id
	conclusion.SubmissionDigest = p.Digest
	if err := conclusion.Validate(); err != nil {
		return err
	}
	workNext, err := w.Version.Next()
	if err != nil {
		return err
	}
	contractNext, err := c.Version.Next()
	if err != nil {
		return err
	}
	if err := conclusion.Bind(w.Ref(), workNext, string(WorkItemLifecycleDone), ConclusionObligations{RequiredCriteria: w.Criteria.RequiredObligations(), ResultSummaryRequired: !w.Criteria.HasRequiredActive()}); err != nil {
		return err
	}
	instant := now.UTC()
	c.Status = ContractCompleted
	c.Version = contractNext
	c.ClosedAt = &instant
	c.CloseReason = p.Material.Summary
	c.ClosedBy = principal
	w.Version = workNext
	w.Lifecycle = WorkItemLifecycleDone
	w.CurrentContractID = nil
	w.CurrentLease = nil
	w.ResultSummary = p.Material.Summary
	w.CurrentConclusion = &conclusion
	w.UpdatedAt = instant
	return nil
}
