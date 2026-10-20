package sqlite

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type noHydrationManager struct{ ports.TransactionManager }

func (m noHydrationManager) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	u, e := m.TransactionManager.Begin(ctx)
	if e != nil {
		return nil, e
	}
	return noHydrationUnit{UnitOfWork: u, read: u.(ports.ContinuityReadUnitOfWork).ContinuityReads()}, nil
}

type noHydrationUnit struct {
	ports.UnitOfWork
	read ports.ContinuityReadRepository
}

func (u noHydrationUnit) ContinuityReads() ports.ContinuityReadRepository { return u.read }
func (u noHydrationUnit) Outcomes() ports.OutcomeRepository {
	return noHydrationOutcome{u.UnitOfWork.Outcomes()}
}
func (u noHydrationUnit) WorkItems() ports.WorkItemRepository {
	return noHydrationWork{u.UnitOfWork.WorkItems()}
}

type noHydrationOutcome struct{ ports.OutcomeRepository }

func (noHydrationOutcome) Get(context.Context, d.ID, d.ID) (d.Outcome, error) {
	panic("paged metadata hydrated Outcome proof/history")
}

type noHydrationWork struct{ ports.WorkItemRepository }

func (noHydrationWork) Get(context.Context, d.Scope, d.ID) (d.WorkItem, error) {
	panic("paged metadata hydrated a WorkItem")
}
func (noHydrationWork) ListByOutcome(context.Context, d.Scope) ([]d.WorkItem, error) {
	panic("paged metadata hydrated every WorkItem")
}
func legacySectionCursor(scope d.Scope, section string, revision d.OutcomeRevision, at time.Time, last json.RawMessage) string {
	hash := sha256.Sum256(last)
	raw, _ := json.Marshal(map[string]any{"ns": scope.NamespaceID, "outcome": scope.OutcomeID, "section": section, "revision": revision, "key": hex.EncodeToString(hash[:]), "evaluated_at": at})
	return base64.RawURLEncoding.EncodeToString(raw)
}
func TestContinuityDirectLegacyCursorAndNoHydration(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "direct.db")
	store := openTestStore(t, path)
	clock := sqliteFixedClock{time.Now().UTC()}
	ids := &sqliteSequenceIDs{prefix: "0199d191", next: 1}
	service, _ := a.NewService(store, clock, ids)
	cc := sqliteCommandContext("0199d190-0000-7000-8000-000000000001", "")
	out, e := service.CreateOutcome(ctx, cc, a.CreateOutcomeCommand{NamespaceID: testID("0199d190-0000-7000-8000-000000000002"), Title: "No hydration", DesiredState: "one page is one page", Priority: d.PriorityNormal})
	if e != nil {
		t.Fatal(e)
	}
	scope := out.Value.Scope()
	for i := 0; i < 7; i++ {
		if _, e = service.CreateWorkItem(ctx, cc, a.CreateWorkItemCommand{Scope: scope, ObjectiveID: sqliteObjectiveForWorkFixture(t, service, scope), Title: "work", Priority: d.PriorityNormal}); e != nil {
			t.Fatal(e)
		}
	}
	u, e := store.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{"0", "1", "2"} {
		v := d.ExternalReference{Provider: strings.Repeat("p", 250) + "&", Kind: "ticket", ExternalID: id, URL: "https://example.test/?s=" + strings.Repeat("v", 1900)}
		if e = v.Validate(); e != nil {
			t.Fatal(e)
		}
		if e = u.(ports.ExternalContextUnitOfWork).ExternalContexts().Add(ctx, scope, v); e != nil {
			t.Fatal(e)
		}
	}
	if e = u.Commit(); e != nil {
		t.Fatal(e)
	}
	snapshot, e := service.GetContinuity(ctx, scope, 2)
	if e != nil {
		t.Fatal(e)
	}
	bounded, _ := a.NewService(noHydrationManager{store}, clock, ids)
	boundedSnapshot, e := bounded.GetContinuity(ctx, scope, 25)
	if e != nil || boundedSnapshot.Counts["work_items"] != 7 {
		t.Fatal("full continuity used aggregate hydration", e)
	}
	legacy := legacySectionCursor(scope, "work_items", snapshot.OutcomeRevision, snapshot.EvaluatedAt, snapshot.Sections["work_items"][1])
	page, e := bounded.GetContinuitySection(ctx, scope, "work_items", 2, legacy)
	if e != nil || len(page.Items) != 2 || page.Count != 7 || page.Omitted != 3 || !page.EvaluatedAt.Equal(snapshot.EvaluatedAt) {
		t.Fatal("legacy cursor was not resumed with bounded reads", page, e)
	}
	raw, e := base64.RawURLEncoding.DecodeString(page.NextCursor)
	if e != nil {
		t.Fatal(e)
	}
	var cursor map[string]any
	if e = json.Unmarshal(raw, &cursor); e != nil || cursor["cursor_version"] != float64(2) {
		t.Fatal("new keyset cursor not emitted", e)
	}
	// Continue both old digest and new keyset cursors after a real clean-file
	// or new-database restore, preserving the original revision/evaluation time.
	expectedNew, e := bounded.GetContinuitySection(ctx, scope, "work_items", 2, page.NextCursor)
	if e != nil {
		t.Fatal(e)
	}
	store = reopenIntegrationFixture(t, store, path)
	service, _ = a.NewService(store, clock, ids)
	bounded, _ = a.NewService(noHydrationManager{store}, clock, ids)
	restoredLegacy, e := bounded.GetContinuitySection(ctx, scope, "work_items", 2, legacy)
	if e != nil {
		t.Fatal(e)
	}
	expectedBytes, _ := json.Marshal(page)
	restoredBytes, _ := json.Marshal(restoredLegacy)
	if !bytes.Equal(expectedBytes, restoredBytes) {
		t.Fatal("restore changed legacy continuation or its frozen frame")
	}
	restoredNew, e := bounded.GetContinuitySection(ctx, scope, "work_items", 2, page.NextCursor)
	if e != nil {
		t.Fatal(e)
	}
	expectedBytes, _ = json.Marshal(expectedNew)
	restoredBytes, _ = json.Marshal(restoredNew)
	if !bytes.Equal(expectedBytes, restoredBytes) {
		t.Fatal("restore changed v2 keyset continuation")
	}
	count := 4
	seen := map[string]bool{}
	for page.NextCursor != "" {
		page, e = bounded.GetContinuitySection(ctx, scope, "work_items", 2, page.NextCursor)
		if e != nil {
			t.Fatal(e)
		}
		for _, item := range page.Items {
			if seen[string(item)] {
				t.Fatal("repeated item")
			}
			seen[string(item)] = true
		}
		count += len(page.Items)
	}
	if count != 7 {
		t.Fatal("cursor omitted rows", count)
	}
	refs, e := bounded.GetContinuitySection(ctx, scope, "external_references", 1, "")
	if e != nil || refs.NextCursor == "" || len(refs.NextCursor) > 4096 {
		t.Fatal("long reference exceeded cursor bound", e)
	}
	decoded, e := base64.RawURLEncoding.DecodeString(refs.NextCursor)
	if e != nil || strings.Contains(string(decoded), "https://example.test") {
		t.Fatal("cursor embedded a receiver/reference URL", e)
	}
	legacyRef := legacySectionCursor(scope, "external_references", refs.OutcomeRevision, refs.EvaluatedAt, refs.Items[0])
	next, e := bounded.GetContinuitySection(ctx, scope, "external_references", 1, legacyRef)
	if e != nil || len(next.Items) != 1 || next.Omitted != 1 {
		t.Fatal("legacy external digest did not resolve", e)
	}
	missing := testID("0199d190-0000-7000-8000-000000000099")
	if _, e = bounded.GetContinuitySection(ctx, d.Scope{NamespaceID: scope.NamespaceID, OutcomeID: missing}, "work_items", 2, ""); securityErrorCode(e) != d.ErrorCodeNotFound {
		t.Fatal("missing Outcome became empty page", e)
	}
	future := legacySectionCursor(scope, "work_items", snapshot.OutcomeRevision, clock.Now().Add(time.Hour), snapshot.Sections["work_items"][1])
	if _, e = bounded.GetContinuitySection(ctx, scope, "work_items", 2, future); securityErrorCode(e) != d.ErrorCodeInvalidArgument {
		t.Fatal("future evaluation time accepted", e)
	}
	_, e = service.CreateWorkItem(ctx, cc, a.CreateWorkItemCommand{Scope: scope, ObjectiveID: sqliteObjectiveForWorkFixture(t, service, scope), Title: "revision changed", Priority: d.PriorityNormal})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = bounded.GetContinuitySection(ctx, scope, "work_items", 2, legacy); securityErrorCode(e) != d.ErrorCodePreconditionFailed {
		t.Fatal("changed revision silently paginated", e)
	}
}
