package httptransport

import "github.com/A1b3rt0M3rcad0/wos/core/domain"

type createIssueRequest struct {
	Title        string                    `json:"title"`
	Description  string                    `json:"description,omitempty"`
	Severity     domain.IssueSeverity      `json:"severity"`
	AffectedRefs []relationEndpointRequest `json:"affected_refs,omitempty"`
}

type updateIssueRequest struct {
	ExpectedVersion *uint64                    `json:"expected_version,omitempty"`
	Title           *string                    `json:"title,omitempty"`
	Description     *string                    `json:"description,omitempty"`
	Severity        *domain.IssueSeverity      `json:"severity,omitempty"`
	AffectedRefs    *[]relationEndpointRequest `json:"affected_refs,omitempty"`
}

type createBlockerRequest struct {
	BlockedRef    relationEndpointRequest  `json:"blocked_ref"`
	CauseRef      *relationEndpointRequest `json:"cause_ref,omitempty"`
	ExternalCause *domain.ExternalCause    `json:"external_cause,omitempty"`
	Description   string                   `json:"description,omitempty"`
	Propagation   domain.BlockerPropagation `json:"propagation,omitempty"`
}

type updateBlockerRequest struct {
	ExpectedVersion *uint64 `json:"expected_version,omitempty"`
	Description     *string `json:"description,omitempty"`
}

type resolutionRequest struct {
	ExpectedVersion   *uint64 `json:"expected_version,omitempty"`
	ResolutionSummary string  `json:"resolution_summary"`
}

type duplicateIssueRequest struct {
	ExpectedVersion    *uint64 `json:"expected_version,omitempty"`
	DuplicateOfIssueID string  `json:"duplicate_of_issue_id"`
}

type reportIssueWithBlockerRequest struct {
	Issue struct {
		Title        string                    `json:"title"`
		Description  string                    `json:"description,omitempty"`
		Severity     domain.IssueSeverity      `json:"severity"`
		AffectedRefs []relationEndpointRequest `json:"affected_refs,omitempty"`
	} `json:"issue"`
	Blocker struct {
		BlockedRef  relationEndpointRequest   `json:"blocked_ref"`
		Description string                    `json:"description,omitempty"`
		Propagation domain.BlockerPropagation `json:"propagation,omitempty"`
	} `json:"blocker"`
}

type resolveIssueAndBlockersRequest struct {
	ExpectedIssueVersion   *uint64                    `json:"expected_issue_version"`
	IssueResolutionSummary string                     `json:"issue_resolution_summary"`
	Blockers               []blockerResolutionRequest `json:"blockers"`
}

type blockerResolutionRequest struct {
	BlockerID         string  `json:"blocker_id"`
	ExpectedVersion   *uint64 `json:"expected_version"`
	ResolutionSummary string  `json:"resolution_summary"`
	ReleaseConfirmed  bool    `json:"release_confirmed"`
}
