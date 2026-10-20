package memory_test

import (
	"context"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestListPendingIssuesByObjectiveUsesStableScopedPages(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	outcome, err := d.NewOutcome(portID(101), portID(100), "Issue query scope", "", "Pending Issues have an Objective", d.PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := d.NewObjective(portID(102), outcome.Scope(), "Selected owner", "", d.PriorityNormal, false, now)
	if err != nil {
		t.Fatal(err)
	}
	otherOwner, err := d.NewObjective(portID(103), outcome.Scope(), "Other owner", "", d.PriorityNormal, false, now)
	if err != nil {
		t.Fatal(err)
	}
	actor := d.ActorRef{Kind: d.ActorKindHuman, Provider: "test", ID: "reporter"}
	newIssue := func(id int, objectiveID d.ID) d.Issue {
		t.Helper()
		issue, err := d.NewIssueForObjective(portID(id), outcome.Scope(), &objectiveID,
			"Operational issue", "", d.IssueSeverityMinor, []d.EntityRef{owner.Ref()}, actor, now)
		if err != nil {
			t.Fatal(err)
		}
		return issue
	}
	first := newIssue(110, owner.ID)
	assignee := d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "issue-worker"}
	first.AssigneeRef = &assignee
	if err := first.Investigate(now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	second := newIssue(111, owner.ID)
	closed := newIssue(112, owner.ID)
	if err := closed.Resolve("accepted resolution", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	other := newIssue(113, otherOwner.ID)

	write, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = write.Outcomes().Insert(ctx, outcome); err == nil {
		err = write.Objectives().Insert(ctx, owner)
	}
	if err == nil {
		err = write.Objectives().Insert(ctx, otherOwner)
	}
	if err != nil {
		_ = write.Rollback()
		t.Fatal(err)
	}
	issues := write.(ports.IssueBlockerUnitOfWork).Issues()
	for _, issue := range []d.Issue{first, second, closed, other} {
		if err := issues.Insert(ctx, issue); err != nil {
			_ = write.Rollback()
			t.Fatal(err)
		}
	}
	if err := write.Commit(); err != nil {
		t.Fatal(err)
	}

	read, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer read.Rollback()
	repo := read.(ports.IssueBlockerUnitOfWork).Issues()
	page, total, err := repo.ListPendingIssuesByObjective(ctx, outcome.Scope(), owner.ID, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(page) != 1 || page[0].ID != first.ID || page[0].Lifecycle != d.IssueLifecycleInvestigating {
		t.Fatalf("first pending page = %#v total=%d, want first investigating Issue and total 2", page, total)
	}
	if len(page[0].AffectedRefs) != 1 || page[0].AffectedRefs[0] != owner.Ref() {
		t.Fatalf("affected references were not preserved: %#v", page[0].AffectedRefs)
	}
	if page[0].AssigneeRef == nil || *page[0].AssigneeRef != assignee {
		t.Fatalf("assignee was not preserved: %#v", page[0].AssigneeRef)
	}
	stored, err := repo.Get(ctx, outcome.Scope(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.AssigneeRef == nil || *stored.AssigneeRef != assignee || stored.ReportedBy != actor {
		t.Fatalf("Issue Get did not preserve assignee and reporter: %#v", stored)
	}
	previousVersion := stored.Version
	stored.AssigneeRef = nil
	stored.Version++
	stored.UpdatedAt = now.Add(2 * time.Minute)
	if err := repo.Save(ctx, stored, previousVersion); err != nil {
		t.Fatal(err)
	}
	stored, err = repo.Get(ctx, outcome.Scope(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.AssigneeRef != nil || stored.ReportedBy != actor {
		t.Fatalf("Issue Save did not clear assignee while preserving reporter: %#v", stored)
	}
	page, total, err = repo.ListPendingIssuesByObjective(ctx, outcome.Scope(), owner.ID, first.ID.String(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(page) != 1 || page[0].ID != second.ID {
		t.Fatalf("second pending page = %#v total=%d, want second open Issue and total 2", page, total)
	}
	page, total, err = repo.ListPendingIssuesByObjective(ctx, outcome.Scope(), otherOwner.ID, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(page) != 1 || page[0].ID != other.ID {
		t.Fatalf("other Objective page = %#v total=%d, want only its Issue", page, total)
	}
	page, total, err = repo.ListPendingIssuesByObjective(ctx, outcome.Scope(), portID(999), "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(page) != 0 {
		t.Fatalf("unknown Objective page = %#v total=%d, want empty", page, total)
	}
	if _, _, err := repo.ListPendingIssuesByObjective(ctx, outcome.Scope(), owner.ID, "bad-cursor", 1); err == nil {
		t.Fatal("invalid cursor was accepted")
	}
	if _, _, err := repo.ListPendingIssuesByObjective(ctx, outcome.Scope(), owner.ID, "", 0); err == nil {
		t.Fatal("invalid page limit was accepted")
	}
}
