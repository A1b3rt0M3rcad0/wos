package domain

import (
	"strings"
	"time"
)

type IssueSeverity string

const (
	IssueSeverityCritical      IssueSeverity = "critical"
	IssueSeverityMajor         IssueSeverity = "major"
	IssueSeverityMinor         IssueSeverity = "minor"
	IssueSeverityInformational IssueSeverity = "informational"
)

func (s IssueSeverity) Valid() bool {
	switch s {
	case IssueSeverityCritical, IssueSeverityMajor, IssueSeverityMinor, IssueSeverityInformational:
		return true
	default:
		return false
	}
}

type IssueLifecycle string

const (
	IssueLifecycleOpen          IssueLifecycle = "open"
	IssueLifecycleInvestigating IssueLifecycle = "investigating"
	IssueLifecycleResolved      IssueLifecycle = "resolved"
	IssueLifecycleWontFix       IssueLifecycle = "wont_fix"
	IssueLifecycleDuplicate     IssueLifecycle = "duplicate"
)

func (l IssueLifecycle) Valid() bool {
	switch l {
	case IssueLifecycleOpen, IssueLifecycleInvestigating, IssueLifecycleResolved, IssueLifecycleWontFix, IssueLifecycleDuplicate:
		return true
	default:
		return false
	}
}

type Issue struct {
	ID                 ID             `json:"id"`
	Scope              Scope          `json:"scope"`
	Version            Version        `json:"version"`
	Title              string         `json:"title"`
	Description        string         `json:"description,omitempty"`
	Severity           IssueSeverity  `json:"severity"`
	Lifecycle          IssueLifecycle `json:"lifecycle"`
	AffectedRefs       []EntityRef    `json:"affected_refs,omitempty"`
	ReportedBy         ActorRef       `json:"reported_by"`
	ResolutionSummary  string         `json:"resolution_summary,omitempty"`
	DuplicateOfIssueID *ID            `json:"duplicate_of_issue_id,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

func NewIssue(
	id ID,
	scope Scope,
	title, description string,
	severity IssueSeverity,
	affectedRefs []EntityRef,
	reportedBy ActorRef,
	now time.Time,
) (Issue, error) {
	value := Issue{
		ID:           id,
		Scope:        scope,
		Version:      InitialVersion,
		Title:        strings.TrimSpace(title),
		Description:  strings.TrimSpace(description),
		Severity:     severity,
		Lifecycle:    IssueLifecycleOpen,
		AffectedRefs: append([]EntityRef(nil), affectedRefs...),
		ReportedBy:   reportedBy,
		CreatedAt:    now.UTC(),
		UpdatedAt:    now.UTC(),
	}
	if err := value.Validate(); err != nil {
		return Issue{}, err
	}
	return value, nil
}

func (i Issue) Ref() EntityRef {
	return EntityRef{Scope: i.Scope, Kind: EntityKindIssue, ID: i.ID}
}

func (i Issue) Validate() error {
	if err := i.ID.Validate(); err != nil {
		return err
	}
	if err := i.Scope.Validate(); err != nil {
		return err
	}
	if err := i.Version.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(i.Title) == "" {
		return NewError(ErrorCodeIssue, "issue title is required")
	}
	if !i.Severity.Valid() {
		return NewError(ErrorCodeIssue, "issue severity is invalid")
	}
	if !i.Lifecycle.Valid() {
		return NewError(ErrorCodeIssue, "issue lifecycle is invalid")
	}
	if err := i.ReportedBy.Validate(); err != nil {
		return WrapError(ErrorCodeIssue, "issue reported_by is invalid", err)
	}
	seen := make(map[EntityRef]struct{}, len(i.AffectedRefs))
	for _, ref := range i.AffectedRefs {
		if err := ref.Validate(); err != nil {
			return WrapError(ErrorCodeIssue, "issue affected_ref is invalid", err)
		}
		if ref.Scope != i.Scope {
			return NewError(ErrorCodeIssue, "issue affected_refs must belong to the same Outcome")
		}
		if _, ok := seen[ref]; ok {
			return NewError(ErrorCodeIssue, "issue affected_refs cannot contain duplicates")
		}
		seen[ref] = struct{}{}
	}
	switch i.Lifecycle {
	case IssueLifecycleResolved, IssueLifecycleWontFix:
		if strings.TrimSpace(i.ResolutionSummary) == "" {
			return NewError(ErrorCodeIssue, "closed issue requires resolution_summary")
		}
		if i.DuplicateOfIssueID != nil {
			return NewError(ErrorCodeIssue, "resolved or wont_fix issue cannot reference duplicate_of_issue_id")
		}
	case IssueLifecycleDuplicate:
		if i.DuplicateOfIssueID == nil {
			return NewError(ErrorCodeIssue, "duplicate issue requires duplicate_of_issue_id")
		}
		if err := i.DuplicateOfIssueID.Validate(); err != nil {
			return WrapError(ErrorCodeIssue, "duplicate_of_issue_id is invalid", err)
		}
		if *i.DuplicateOfIssueID == i.ID {
			return NewError(ErrorCodeIssue, "issue cannot be a duplicate of itself")
		}
	default:
		if i.DuplicateOfIssueID != nil {
			return NewError(ErrorCodeIssue, "only duplicate issue may reference duplicate_of_issue_id")
		}
		if strings.TrimSpace(i.ResolutionSummary) != "" {
			return NewError(ErrorCodeIssue, "open issue cannot have resolution_summary")
		}
	}
	if i.CreatedAt.IsZero() || i.UpdatedAt.IsZero() {
		return NewError(ErrorCodeIssue, "issue timestamps are required")
	}
	return nil
}

func (i *Issue) Investigate(now time.Time) error {
	if i.Lifecycle != IssueLifecycleOpen {
		return NewError(ErrorCodeInvalidTransition, "only open issue can enter investigation")
	}
	i.Lifecycle = IssueLifecycleInvestigating
	return i.touch(now)
}

func (i *Issue) Resolve(summary string, now time.Time) error {
	if i.Lifecycle != IssueLifecycleOpen && i.Lifecycle != IssueLifecycleInvestigating {
		return NewError(ErrorCodeInvalidTransition, "issue cannot be resolved from current lifecycle")
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return NewError(ErrorCodeIssue, "issue resolution summary is required")
	}
	i.Lifecycle = IssueLifecycleResolved
	i.ResolutionSummary = summary
	i.DuplicateOfIssueID = nil
	return i.touch(now)
}

func (i *Issue) MarkWontFix(summary string, now time.Time) error {
	if i.Lifecycle != IssueLifecycleOpen && i.Lifecycle != IssueLifecycleInvestigating {
		return NewError(ErrorCodeInvalidTransition, "issue cannot be marked wont_fix from current lifecycle")
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return NewError(ErrorCodeIssue, "wont_fix summary is required")
	}
	i.Lifecycle = IssueLifecycleWontFix
	i.ResolutionSummary = summary
	i.DuplicateOfIssueID = nil
	return i.touch(now)
}

func (i *Issue) MarkDuplicate(target ID, now time.Time) error {
	if i.Lifecycle != IssueLifecycleOpen && i.Lifecycle != IssueLifecycleInvestigating {
		return NewError(ErrorCodeInvalidTransition, "issue cannot be marked duplicate from current lifecycle")
	}
	if err := target.Validate(); err != nil {
		return WrapError(ErrorCodeIssue, "duplicate issue id is invalid", err)
	}
	if target == i.ID {
		return NewError(ErrorCodeIssue, "issue cannot be a duplicate of itself")
	}
	i.Lifecycle = IssueLifecycleDuplicate
	i.ResolutionSummary = ""
	i.DuplicateOfIssueID = &target
	return i.touch(now)
}

func (i *Issue) Reopen(now time.Time) error {
	switch i.Lifecycle {
	case IssueLifecycleResolved, IssueLifecycleWontFix, IssueLifecycleDuplicate:
	default:
		return NewError(ErrorCodeInvalidTransition, "only closed issue can be reopened")
	}
	i.Lifecycle = IssueLifecycleOpen
	i.ResolutionSummary = ""
	i.DuplicateOfIssueID = nil
	return i.touch(now)
}

func (i *Issue) touch(now time.Time) error {
	next, err := nextVersion(i.Version)
	if err != nil {
		return err
	}
	i.Version = next
	i.UpdatedAt = now.UTC()
	return i.Validate()
}
