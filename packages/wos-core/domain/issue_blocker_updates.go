package domain

import (
	"strings"
	"time"
)

func (i *Issue) UpdateDetails(
	title *string,
	description *string,
	severity *IssueSeverity,
	affectedRefs *[]EntityRef,
	now time.Time,
) (bool, error) {
	if i.Lifecycle != IssueLifecycleOpen && i.Lifecycle != IssueLifecycleInvestigating {
		return false, NewError(ErrorCodeInvalidTransition, "only open or investigating issue can be edited")
	}

	next := *i
	next.AffectedRefs = append([]EntityRef(nil), i.AffectedRefs...)
	changed := false
	if title != nil {
		value := strings.TrimSpace(*title)
		if next.Title != value {
			next.Title = value
			changed = true
		}
	}
	if description != nil {
		value := strings.TrimSpace(*description)
		if next.Description != value {
			next.Description = value
			changed = true
		}
	}
	if severity != nil && next.Severity != *severity {
		next.Severity = *severity
		changed = true
	}
	if affectedRefs != nil && !entityRefsEqual(next.AffectedRefs, *affectedRefs) {
		next.AffectedRefs = append([]EntityRef(nil), (*affectedRefs)...)
		changed = true
	}
	if !changed {
		return false, nil
	}
	version, err := nextVersion(i.Version)
	if err != nil {
		return false, err
	}
	next.Version = version
	next.UpdatedAt = now.UTC()
	if err := next.Validate(); err != nil {
		return false, err
	}
	*i = next
	return true, nil
}

func (b *Blocker) UpdateDescription(description string, now time.Time) (bool, error) {
	if b.Lifecycle != BlockerLifecycleActive {
		return false, NewError(ErrorCodeInvalidTransition, "only active blocker description can be edited")
	}
	description = strings.TrimSpace(description)
	if b.Description == description {
		return false, nil
	}
	next := *b
	next.Description = description
	version, err := nextVersion(b.Version)
	if err != nil {
		return false, err
	}
	next.Version = version
	next.UpdatedAt = now.UTC()
	if err := next.Validate(); err != nil {
		return false, err
	}
	*b = next
	return true, nil
}

func entityRefsEqual(left, right []EntityRef) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
