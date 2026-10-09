package boundary_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

func TestAuthorizedReferenceSearchStores(t *testing.T) {
	for _, name := range []string{"memory", "sqlite", "postgres"} {
		t.Run(name, func(t *testing.T) {
			var tx ports.TransactionManager
			switch name {
			case "memory":
				tx = memory.New()
			case "sqlite":
				s, err := sqlite.Open(filepath.Join(t.TempDir(), "refs.db"), sqlite.Options{MigrateOnOpen: true})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { s.Close() })
				tx = s
			case "postgres":
				dsn := os.Getenv("WOS_TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("real PostgreSQL requires WOS_TEST_POSTGRES_DSN")
				}
				s, err := postgres.Open(dsn, postgres.Options{MigrateOnOpen: true, Schema: fmt.Sprintf("wos_refs_%d", time.Now().UnixNano())})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { s.Close() })
				tx = s
			}
			runReferenceSearch(t, tx, 1001)
		})
	}
}

func runReferenceSearch(t *testing.T, tx ports.TransactionManager, count int) {
	t.Helper()
	ctx := context.Background()
	clock := &readinessClock{now: time.Now().UTC()}
	ids := &readinessIDs{}
	cmds := &readinessCommands{}
	security := application.SecurityService{Store: tx.(ports.SecurityStore), Clock: clock, IDs: ids}
	ns := domain.MustParseID("0199ed02-0000-7000-8000-000000000001")
	otherNS := domain.MustParseID("0199ed02-0000-7000-8000-000000000002")
	for _, x := range []struct {
		id          domain.ID
		name, token string
	}{{ns, "readiness-human", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, {otherNS, "other", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}} {
		if err := security.Bootstrap(ctx, ports.Namespace{ID: x.id, Name: x.name}, x.name, x.token); err != nil {
			t.Fatal(err)
		}
	}
	identity, err := security.Authenticate(ctx, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	ctx = application.WithIdentity(ctx, identity)
	commandContext := func() domain.CommandContext {
		c := cmds.Context()
		c.Actor = identity.Actor
		c.PrincipalID = identity.PrincipalID
		return c
	}
	s, err := application.NewAuthorizedService(tx, clock, ids, security)
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.CreateOutcome(ctx, commandContext(), application.CreateOutcomeCommand{NamespaceID: ns, Title: "Search scope", DesiredState: "Find every authorized reference", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	scope := out.Value.Scope()
	objective, err := s.CreateObjective(ctx, commandContext(), application.CreateObjectiveCommand{Scope: scope, Title: "Duplicate objective", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	var lastTask domain.WorkItem
	var firstTask domain.WorkItem
	var lastEvidence domain.Evidence
	for i := 0; i < count; i++ {
		title := "Duplicate task"
		description := "Duplicate observation"
		if i == count-1 {
			title = "Off-page target"
			description = "Off-page observation"
		}
		priority := domain.PriorityNormal
		if i == count-1 {
			priority = domain.PriorityHigh
		}
		v, e := s.CreateWorkItem(ctx, commandContext(), application.CreateWorkItemCommand{Scope: scope, Title: title, Priority: priority, Lifecycle: domain.WorkItemLifecycleTodo, ObjectiveID: &objective.Value.ID})
		if e != nil {
			t.Fatal(e)
		}
		lastTask = v.Value
		if i == 0 {
			firstTask = v.Value
		}
		ev, e := s.RegisterEvidence(ctx, commandContext(), application.RegisterEvidenceCommand{Scope: scope, EvidenceType: domain.EvidenceTypeTestResult, Description: description, SourceRef: domain.SourceReference{Provider: "test", ID: fmt.Sprint(i)}, CapturedAt: clock.Now()})
		if e != nil {
			t.Fatal(e)
		}
		lastEvidence = ev.Value
	}
	high, priorityErr := s.SearchReferences(ctx, scope, application.ReferenceQuery{Kinds: []domain.EntityKind{domain.EntityKindWorkItem}, Priority: domain.PriorityHigh, Query: "Off-page"})
	if priorityErr != nil || len(high.Items) != 1 || high.Items[0].Ref.ID != lastTask.ID {
		t.Fatalf("priority off-page query: %+v %v", high, priorityErr)
	}
	if _, e := s.SearchReferences(ctx, scope, application.ReferenceQuery{Kinds: []domain.EntityKind{domain.EntityKindEvidence}, Priority: domain.PriorityHigh}); e == nil {
		t.Fatal("priority accepted for Evidence")
	}
	normal, priorityErr := s.SearchReferences(ctx, scope, application.ReferenceQuery{Kinds: []domain.EntityKind{domain.EntityKindWorkItem}, Priority: domain.PriorityNormal, Limit: 1})
	if priorityErr != nil {
		t.Fatal(priorityErr)
	}
	if normal.NextCursor != "" {
		if _, e := s.SearchReferences(ctx, scope, application.ReferenceQuery{Kinds: []domain.EntityKind{domain.EntityKindWorkItem}, Priority: domain.PriorityHigh, Cursor: normal.NextCursor}); e == nil {
			t.Fatal("cursor accepted with changed priority")
		}
	}
	query := application.ReferenceQuery{Kinds: []domain.EntityKind{domain.EntityKindWorkItem}, Limit: 37}
	seen := map[domain.ID]bool{}
	cursor := ""
	for {
		query.Cursor = cursor
		p, e := s.SearchReferences(ctx, scope, query)
		if e != nil {
			t.Fatal(e)
		}
		if len(p.Items) > 37 {
			t.Fatal("unbounded search")
		}
		for _, v := range p.Items {
			if seen[v.Ref.ID] {
				t.Fatal("duplicate cursor item")
			}
			seen[v.Ref.ID] = true
			if v.Ref.Scope != scope || v.DisplayContext != "Duplicate objective" {
				t.Fatalf("scope/context mismatch: %+v", v)
			}
		}
		cursor = p.NextCursor
		if cursor == "" {
			break
		}
	}
	if len(seen) != count {
		t.Fatalf("only %d candidates", len(seen))
	}
	for _, x := range []struct {
		kind  domain.EntityKind
		id    domain.ID
		query string
	}{{domain.EntityKindWorkItem, lastTask.ID, "off-page"}, {domain.EntityKindEvidence, lastEvidence.ID, "off-page"}} {
		p, e := s.SearchReferences(ctx, scope, application.ReferenceQuery{Kinds: []domain.EntityKind{x.kind}, Query: x.query, Limit: 1})
		if e != nil || len(p.Items) != 1 || p.Items[0].Ref.ID != x.id {
			t.Fatalf("off-page search %+v %v", p, e)
		}
		p, e = s.SearchReferences(ctx, scope, application.ReferenceQuery{Kinds: []domain.EntityKind{x.kind}, ID: x.id, Limit: 1})
		if e != nil || len(p.Items) != 1 {
			t.Fatalf("selected ref resolution %+v %v", p, e)
		}
	}
	var total, max time.Duration
	for i := 0; i < 20; i++ {
		started := time.Now()
		v, e := s.SearchReferences(ctx, scope, application.ReferenceQuery{Kinds: []domain.EntityKind{domain.EntityKindWorkItem}, Query: "off-page", Limit: 20})
		elapsed := time.Since(started)
		total += elapsed
		if elapsed > max {
			max = elapsed
		}
		if e != nil || len(v.Items) != 1 {
			t.Fatalf("measured search failed %+v %v", v, e)
		}
	}
	t.Logf("reference-search dataset=%d queries=20 average=%s maximum=%s (automated latency, not usability)", count, total/20, max)
	p, err := s.SearchReferences(ctx, scope, application.ReferenceQuery{Kinds: []domain.EntityKind{domain.EntityKindEvidence}, Limit: 2})
	if err != nil || p.NextCursor == "" {
		t.Fatal("missing evidence page")
	}
	for _, bad := range []application.ReferenceQuery{{Kinds: []domain.EntityKind{domain.EntityKindEvidence}, Query: "changed", Cursor: p.NextCursor}, {Kinds: []domain.EntityKind{domain.EntityKindWorkItem}, Cursor: p.NextCursor}, {Kinds: []domain.EntityKind{"table; DROP TABLE evidence"}}, {Kinds: []domain.EntityKind{domain.EntityKindEvidence}, Limit: 101}, {Kinds: []domain.EntityKind{domain.EntityKindEvidence}, Limit: -1}, {Kinds: []domain.EntityKind{domain.EntityKindEvidence, domain.EntityKindEvidence}}, {Kinds: []domain.EntityKind{domain.EntityKindEvidence}, Cursor: "garbage"}, {Kinds: []domain.EntityKind{domain.EntityKindEvidence}, Query: "invalid\x00text"}} {
		if _, e := s.SearchReferences(ctx, scope, bad); e == nil {
			t.Fatalf("invalid query accepted: %+v", bad)
		}
	}
	if _, e := s.SearchReferences(ctx, domain.Scope{NamespaceID: otherNS, OutcomeID: scope.OutcomeID}, application.ReferenceQuery{Kinds: []domain.EntityKind{domain.EntityKindEvidence}, Cursor: p.NextCursor}); e == nil {
		t.Fatal("cross namespace cursor/read accepted")
	}
	if _, e := s.CreateWorkItem(ctx, commandContext(), application.CreateWorkItemCommand{Scope: scope, Title: "Revision change", Priority: domain.PriorityNormal}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.SearchReferences(ctx, scope, application.ReferenceQuery{Kinds: []domain.EntityKind{domain.EntityKindEvidence}, Cursor: p.NextCursor}); e == nil {
		t.Fatal("stale revision accepted")
	}
	for _, title := range []string{"Écho", "écho", "İnfo", "Info"} {
		if _, e := s.CreateWorkItem(ctx, commandContext(), application.CreateWorkItemCommand{Scope: scope, Title: title, Priority: domain.PriorityNormal}); e != nil {
			t.Fatal(e)
		}
	}
	for _, text := range []string{"Écho", "écho", "info"} {
		v, e := s.SearchReferences(ctx, scope, application.ReferenceQuery{Kinds: []domain.EntityKind{domain.EntityKindWorkItem}, Query: text})
		if e != nil || len(v.Items) != 1 {
			t.Fatalf("ASCII folding / Unicode preservation mismatch for %q: %+v %v", text, v, e)
		}
	}
	issue, e := s.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{Scope: scope, Title: "Issue target", Severity: domain.IssueSeverityMinor})
	if e != nil {
		t.Fatal(e)
	}
	cause := issue.Value.Ref()
	if _, e = s.CreateBlocker(ctx, commandContext(), application.CreateBlockerCommand{Scope: scope, BlockedRef: lastTask.Ref(), CauseRef: &cause, Description: "Blocker target", Propagation: domain.BlockerPropagationDirect}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.RegisterArtifact(ctx, commandContext(), application.RegisterArtifactCommand{Scope: scope, Name: "Artifact target", ArtifactType: "report", URI: "https://example.test/report"}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ProposeDecision(ctx, commandContext(), application.ProposeDecisionCommand{Scope: scope, Title: "Decision target", Proposal: "Choose the scoped action"}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateRoadmap(ctx, commandContext(), application.CreateRoadmapCommand{Scope: scope, Title: "Roadmap target", PlanScope: domain.RoadmapPlanScope{Kind: domain.RoadmapScopeOutcome, ID: scope.OutcomeID}}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateEvidenceLink(ctx, commandContext(), application.CreateEvidenceLinkCommand{Scope: scope, EvidenceID: lastEvidence.ID, TargetRef: lastTask.Ref(), Stance: domain.EvidenceStanceSupports, Rationale: "Linked observation"}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.AddDependency(ctx, commandContext(), application.AddDependencyCommand{Scope: scope, SourceRef: lastTask.Ref(), TargetRef: firstTask.Ref(), Strength: domain.DependencyStrengthHard}); e != nil {
		t.Fatal(e)
	}
	for _, kind := range []domain.EntityKind{domain.EntityKindOutcome, domain.EntityKindObjective, domain.EntityKindWorkItem, domain.EntityKindEvidence, domain.EntityKindIssue, domain.EntityKindBlocker, domain.EntityKindArtifact, domain.EntityKindDecision, domain.EntityKindRoadmap, domain.EntityKindRelation, domain.EntityKindEvidenceLink} {
		v, e := s.SearchReferences(ctx, scope, application.ReferenceQuery{Kinds: []domain.EntityKind{kind}, Limit: 1})
		if e != nil || len(v.Items) != 1 {
			t.Fatalf("reference kind %s missing: %+v %v", kind, v, e)
		}
	}
	if e = security.SetGrant(ctx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "viewer", Permissions: []ports.Permission{ports.PermissionStateRead}}); e != nil {
		t.Fatal(e)
	}
	cred, token, e := security.IssueCredential(ctx, ns, "viewer", domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "viewer"}, clock.Now().Add(time.Hour))
	if e != nil {
		t.Fatal(e)
	}
	viewer, e := security.Authenticate(ctx, token)
	if e != nil {
		t.Fatal(e)
	}
	viewerCtx := application.WithIdentity(context.Background(), viewer)
	q := application.ReferenceQuery{Kinds: []domain.EntityKind{domain.EntityKindWorkItem}, ID: lastTask.ID}
	if _, e = s.SearchReferences(viewerCtx, scope, q); e != nil {
		t.Fatal(e)
	}
	if e = security.RevokeCredential(ctx, ns, cred.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.SearchReferences(viewerCtx, scope, q); e == nil {
		t.Fatal("cached identity survived revocation")
	}
	if _, e = s.SearchReferences(context.Background(), scope, q); e == nil {
		t.Fatal("missing identity accepted")
	}
}

func TestReferenceCursorFiltersQuickRegression(t *testing.T) { runReferenceSearch(t, memory.New(), 9) }

func TestReferencePageEscapedLabelsRemainBounded(t *testing.T) {
	ctx := context.Background()
	clock := &readinessClock{now: time.Now().UTC()}
	s, err := application.NewService(memory.New(), clock, &readinessIDs{})
	if err != nil {
		t.Fatal(err)
	}
	cmds := &readinessCommands{}
	out, err := s.CreateOutcome(ctx, cmds.Context(), application.CreateOutcomeCommand{NamespaceID: domain.MustParseID("0199ed02-0000-7000-8000-000000000001"), Title: "Bounded labels", DesiredState: "Metadata is bounded", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	objective, err := s.CreateObjective(ctx, cmds.Context(), application.CreateObjectiveCommand{Scope: out.Value.Scope(), Title: strings.Repeat("<", 256), Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		if _, err = s.CreateWorkItem(ctx, cmds.Context(), application.CreateWorkItemCommand{Scope: out.Value.Scope(), Title: strings.Repeat("<", 256), Priority: domain.PriorityNormal, ObjectiveID: &objective.Value.ID}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.SearchReferences(ctx, out.Value.Scope(), application.ReferenceQuery{Kinds: []domain.EntityKind{domain.EntityKindWorkItem}, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(page)
	if err != nil || len(raw) > application.MaxSnapshotBytes || len(page.Items) != 100 {
		t.Fatalf("unbounded metadata response: items=%d bytes=%d err=%v", len(page.Items), len(raw), err)
	}
}
