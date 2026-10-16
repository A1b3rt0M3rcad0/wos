package domain

import (
	"encoding/json"
	"strings"
	"time"
)

// A single immutable record always fits a bounded history page, including its
// response envelope and cursor. Referenced source metadata is included here.
const MaxWorkActivityBytes = 128 << 10

// WorkActivity is an immutable declaration accepted from a live contract holder.
// It does not attest that external execution occurred and never approves criteria.
type WorkActivityType string

const (
	WorkActivityProgress   WorkActivityType = "progress"
	WorkActivityDiscovery  WorkActivityType = "discovery"
	WorkActivityProblem    WorkActivityType = "problem"
	WorkActivityCorrection WorkActivityType = "correction"
	WorkActivityValidation WorkActivityType = "validation"
	WorkActivityResult     WorkActivityType = "result"
)

func (t WorkActivityType) Valid() bool {
	switch t {
	case WorkActivityProgress, WorkActivityDiscovery, WorkActivityProblem, WorkActivityCorrection, WorkActivityValidation, WorkActivityResult:
		return true
	}
	return false
}

type WorkActivityCriterionRef struct {
	CriterionID       ID                `json:"criterion_id"`
	CriterionRevision CriterionRevision `json:"criterion_revision,string"`
}

type WorkActivity struct {
	ID         ID    `json:"id"`
	Scope      Scope `json:"scope"`
	ContractID ID    `json:"contract_id"`
	WorkItemID ID    `json:"work_item_id"`
	// Sequence is the containing Outcome revision, assigned under its guard.
	Sequence         uint64                     `json:"sequence,string"`
	Type             WorkActivityType           `json:"type"`
	Description      string                     `json:"description"`
	PrincipalID      string                     `json:"principal_id"`
	Actor            ActorRef                   `json:"actor_ref"`
	RecordedAt       time.Time                  `json:"recorded_at"`
	ArtifactRefs     []ID                       `json:"artifact_refs,omitempty"`
	ArtifactVersions []SubmissionArtifact       `json:"artifact_versions,omitempty"`
	EvidenceRefs     []ID                       `json:"evidence_refs,omitempty"`
	CriterionRefs    []WorkActivityCriterionRef `json:"criterion_refs,omitempty"`
}

func (a WorkActivity) Validate() error {
	if err := a.Scope.Validate(); err != nil {
		return err
	}
	for _, id := range []ID{a.ID, a.ContractID, a.WorkItemID} {
		if err := id.Validate(); err != nil {
			return err
		}
	}
	if !a.Type.Valid() || a.Sequence == 0 || a.RecordedAt.IsZero() || strings.TrimSpace(a.PrincipalID) == "" || strings.TrimSpace(a.Description) == "" || len(a.Description) > 16384 {
		return NewError(ErrorCodeInvalidArgument, "activity requires type, description (up to 16 KiB), chronology and provenance")
	}
	if err := a.Actor.Validate(); err != nil {
		return err
	}
	if len(a.ArtifactRefs)+len(a.EvidenceRefs)+len(a.CriterionRefs) > 100 {
		return NewError(ErrorCodeInvalidArgument, "activity reference limit is 100")
	}
	for _, ids := range [][]ID{a.ArtifactRefs, a.EvidenceRefs} {
		seen := map[ID]bool{}
		for _, id := range ids {
			if err := id.Validate(); err != nil {
				return err
			}
			if seen[id] {
				return NewError(ErrorCodeInvalidArgument, "duplicate activity reference")
			}
			seen[id] = true
		}
	}
	seen := map[ID]bool{}
	for _, ref := range a.CriterionRefs {
		if ref.CriterionID.Validate() != nil || ref.CriterionRevision < 1 || seen[ref.CriterionID] {
			return NewError(ErrorCodeInvalidArgument, "invalid or duplicate activity criterion revision")
		}
		seen[ref.CriterionID] = true
	}
	if len(a.ArtifactVersions) > len(a.ArtifactRefs) {
		return NewError(ErrorCodeInvalidArgument, "activity artifact versions require artifact references")
	}
	seen = map[ID]bool{}
	for _, ref := range a.ArtifactVersions {
		found := false
		for _, id := range a.ArtifactRefs {
			if id == ref.ArtifactID {
				found = true
			}
		}
		if !found || seen[ref.ArtifactID] {
			return NewError(ErrorCodeInvalidArgument, "invalid or duplicate activity artifact version")
		}
		seen[ref.ArtifactID] = true
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	if len(raw) > MaxWorkActivityBytes {
		return NewError(ErrorCodeInvalidArgument, "complete activity record exceeds 128 KiB; shorten referenced metadata")
	}
	return nil
}
