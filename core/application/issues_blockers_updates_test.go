package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func TestIssueAndBlockerEditableFieldsUseVersionedNoOpSemantics(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	work := createTodoWork(t, service, outcome.Scope(), "editable issue target")
	reported, err := service.ReportIssueWithBlocker(ctx, commandContext(), application.ReportIssueWithBlockerCommand{
		Scope:              outcome.Scope(),
		Title:              "Original issue",
		Description:        "original details",
		Severity:           domain.IssueSeverityMinor,
		AffectedRefs:       []domain.EntityRef{work.Ref()},
		BlockedRef:         work.Ref(),
		BlockerDescription: "original blocker",
	})
	if err != nil {
		t.Fatal(err)
	}
	issue := reported.Value.Issue
	blocker := reported.Value.Blocker

	title := "Updated issue"
	severity := domain.IssueSeverityMajor
	updatedIssue, err := service.UpdateIssue(ctx, commandContext(), application.UpdateIssueCommand{
		Scope: outcome.Scope(), IssueID: issue.ID, ExpectedVersion: issue.Version,
		Title: &title, Severity: &severity,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updatedIssue.Value.Version != issue.Version+1 || updatedIssue.Value.Title != title || updatedIssue.Value.Severity != severity {
		t.Fatalf("updated issue = %#v", updatedIssue.Value)
	}

	updatedBlocker, err := service.UpdateBlockerDescription(ctx, commandContext(), application.UpdateBlockerDescriptionCommand{
		Scope: outcome.Scope(), BlockerID: blocker.ID, ExpectedVersion: blocker.Version, Description: "updated blocker",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updatedBlocker.Value.Version != blocker.Version+1 || updatedBlocker.Value.Description != "updated blocker" {
		t.Fatalf("updated blocker = %#v", updatedBlocker.Value)
	}

	beforeEvents := len(store.SnapshotDomainEvents(outcome.Scope()))
	noop, err := service.UpdateBlockerDescription(ctx, commandContext(), application.UpdateBlockerDescriptionCommand{
		Scope: outcome.Scope(), BlockerID: blocker.ID, ExpectedVersion: updatedBlocker.Value.Version, Description: "updated blocker",
	})
	if err != nil {
		t.Fatal(err)
	}
	if noop.Value.Version != updatedBlocker.Value.Version {
		t.Fatalf("no-op version = %d, want %d", noop.Value.Version, updatedBlocker.Value.Version)
	}
	afterEvents := len(store.SnapshotDomainEvents(outcome.Scope()))
	if afterEvents != beforeEvents {
		t.Fatalf("no-op emitted event: before=%d after=%d", beforeEvents, afterEvents)
	}
}
