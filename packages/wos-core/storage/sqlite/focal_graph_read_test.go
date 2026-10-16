package sqlite

import (
	"context"
	"encoding/json"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// Hides only optimized reads. The legacy Application path remains the semantic
// oracle; documentary/blocker data is still available, never silently omitted.
type legacyFocalManager struct{ ports.TransactionManager }

func (m legacyFocalManager) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	u, e := m.TransactionManager.Begin(ctx)
	if e != nil {
		return nil, e
	}
	return legacyFocalUnit{u}, nil
}

type legacyFocalUnit struct{ ports.UnitOfWork }

func (u legacyFocalUnit) Issues() ports.IssueRepository {
	return u.UnitOfWork.(ports.IssueBlockerUnitOfWork).Issues()
}
func (u legacyFocalUnit) Blockers() ports.BlockerRepository {
	return u.UnitOfWork.(ports.IssueBlockerUnitOfWork).Blockers()
}
func (u legacyFocalUnit) Artifacts() ports.ArtifactRepository {
	return u.UnitOfWork.(ports.DocumentaryUnitOfWork).Artifacts()
}
func (u legacyFocalUnit) Evidence() ports.EvidenceRepository {
	return u.UnitOfWork.(ports.DocumentaryUnitOfWork).Evidence()
}
func (u legacyFocalUnit) Decisions() ports.DecisionRepository {
	return u.UnitOfWork.(ports.DocumentaryUnitOfWork).Decisions()
}
func (u legacyFocalUnit) EvidenceLinks() ports.EvidenceLinkRepository {
	return u.UnitOfWork.(ports.DocumentaryUnitOfWork).EvidenceLinks()
}
func assertFocalJSON(t *testing.T, want, got any) {
	t.Helper()
	w, e := json.Marshal(want)
	if e != nil {
		t.Fatal(e)
	}
	g, e := json.Marshal(got)
	if e != nil {
		t.Fatal(e)
	}
	var wm, gm any
	if e = json.Unmarshal(w, &wm); e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(g, &gm); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(wm, gm) {
		t.Fatalf("bounded read changed legacy semantics\nwant %s\ngot  %s", w, g)
	}
}
func TestFocalCurrentProofDenseHistoryAndHiddenDependency(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			var store ports.TransactionManager = memory.New()
			if backend == "sql" {
				store = openTestStore(t, filepath.Join(t.TempDir(), "focal.db"))
			}
			must := func(e error) {
				t.Helper()
				if e != nil {
					t.Fatal(e)
				}
			}
			ctx := context.Background()
			now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
			clock := portClock{now}
			ids := &portIDs{}
			service, e := a.NewService(store, clock, ids)
			must(e)
			cc := sqliteCommandContext(portID(1).String(), "")
			out, e := service.CreateOutcome(ctx, cc, a.CreateOutcomeCommand{NamespaceID: portID(2), Title: "Dense focal", DesiredState: "current proof only", Priority: d.PriorityNormal})
			must(e)
			scope := out.Value.Scope()
			_, e = service.AddCriterion(ctx, cc, a.AddCriterionCommand{Owner: out.Value.Ref(), ExpectedVersion: 1, Title: "Completion", Required: true, VerificationMode: d.VerificationModeAttestation})
			must(e)
			_, e = service.ActivateOutcome(ctx, cc, a.ActivateOutcomeCommand{Scope: scope, ExpectedVersion: 2})
			must(e)
			u, e := store.Begin(ctx)
			must(e)
			defer u.Rollback()
			work, e := d.NewWorkItem(portID(100), scope, "Focal work", "all fields survive", d.PriorityHigh, d.WorkItemLifecycleTodo, now)
			must(e)
			criterion, e := d.NewSuccessCriterion(portID(101), work.Ref(), "Current attestation", "retain definition", true, d.VerificationModeAttestation)
			must(e)
			must(work.AddCriterion(criterion, now))
			for i := 0; i < 1000; i++ {
				assessment := d.CriterionAssessment{ID: portID(1000 + i), CriterionID: criterion.ID, CriterionRevision: 1, Result: d.AssessmentResultMet, Rationale: "Current proof with superseded history", PrincipalID: cc.PrincipalID, Actor: cc.Actor, AssessedAt: now.Add(time.Duration(i) * time.Nanosecond)}
				must(work.RecordCriterionAssessment(assessment, false, now))
			}
			must(u.WorkItems().Insert(ctx, work))
			for i := 0; i < 101; i++ {
				target, e := d.NewWorkItem(portID(3000+i), scope, "Target", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
				must(e)
				must(u.WorkItems().Insert(ctx, target))
				strength := d.DependencyStrengthAdvisory
				if i == 100 {
					strength = d.DependencyStrengthHard
				}
				rel, e := d.NewDependencyRelation(portID(4000+i), work.Ref(), target.Ref(), strength, now)
				must(e)
				must(u.Relations().Insert(ctx, rel))
			}
			_, e = u.Coordination().AdvanceOutcome(ctx, scope)
			must(e)
			must(u.Commit())
			legacy, e := a.NewService(legacyFocalManager{store}, clock, ids)
			must(e)
			want, e := legacy.GetWorkContext(ctx, scope, work.ID)
			must(e)
			bounded, e := a.NewService(noHydrationManager{store}, clock, ids)
			must(e)
			got, e := bounded.GetWorkContext(ctx, scope, work.ID)
			must(e)
			assertFocalJSON(t, want, got)
			if got.Operational.DisplayState != d.WorkItemDisplayStateWaitingDependencies || len(got.Dependencies) != 100 || got.Omitted["dependencies"] != 1 || got.Omitted["assessment_history"] != 1000 || len(got.Work.Criteria.Assessments) != 0 || got.Work.Criteria.CurrentAssessments[criterion.ID].ID != portID(1999) {
				t.Fatal("current proof, counts or hidden hard dependency lost", got.Operational, got.Omitted)
			}
			if _, e = bounded.GetWorkContext(ctx, d.Scope{NamespaceID: portID(99), OutcomeID: scope.OutcomeID}, work.ID); e == nil {
				t.Fatal("cross-namespace focal read succeeded")
			}
		})
	}
}
func TestGraphAdjacencyMatchesLegacyDepthDirectionsKindsAndCursors(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			var store ports.TransactionManager = memory.New()
			if backend == "sql" {
				store = openTestStore(t, filepath.Join(t.TempDir(), "graph.db"))
			}
			must := func(e error) {
				t.Helper()
				if e != nil {
					t.Fatal(e)
				}
			}
			ctx := context.Background()
			now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
			clock := portClock{now}
			ids := &portIDs{}
			service, e := a.NewService(store, clock, ids)
			must(e)
			cc := sqliteCommandContext(portID(1).String(), "")
			out, e := service.CreateOutcome(ctx, cc, a.CreateOutcomeCommand{NamespaceID: portID(2), Title: "Graph parity", DesiredState: "bounded adjacency", Priority: d.PriorityNormal})
			must(e)
			scope := out.Value.Scope()
			u, e := store.Begin(ctx)
			must(e)
			defer u.Rollback()
			works := []d.WorkItem{}
			for i := 0; i < 4; i++ {
				w, e := d.NewWorkItem(portID(100+i), scope, "Node", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
				must(e)
				must(u.WorkItems().Insert(ctx, w))
				works = append(works, w)
			}
			// Cycles are legal for relates_to, and depth-boundary induced edges must stay.
			for i := 0; i < 4; i++ {
				rel := d.Relation{ID: portID(200 + i), Scope: scope, Version: 1, SourceRef: works[i].Ref(), TargetRef: works[(i+1)%4].Ref(), RelationType: d.RelationTypeRelatesTo, Lifecycle: d.RelationLifecycleActive, CreatedAt: now, UpdatedAt: now}
				must(rel.Validate())
				must(u.Relations().Insert(ctx, rel))
			}
			issue, e := d.NewIssue(portID(300), scope, "Cause", "observed", d.IssueSeverityMinor, []d.EntityRef{works[0].Ref()}, cc.Actor, now)
			must(e)
			must(u.(ports.IssueBlockerUnitOfWork).Issues().Insert(ctx, issue))
			cause := issue.Ref()
			blocker, e := d.NewBlocker(portID(301), works[0].Ref(), &cause, nil, "Blocking cause", d.BlockerPropagationDirect, now)
			must(e)
			must(u.(ports.IssueBlockerUnitOfWork).Blockers().Insert(ctx, blocker))
			docs := u.(ports.DocumentaryUnitOfWork)
			artifact, e := d.NewArtifact(portID(302), scope, "report", "Report", "https://example.test/report", "text/plain", "", "v1", cc.Actor, nil, now)
			must(e)
			must(docs.Artifacts().Insert(ctx, artifact))
			evidence, e := d.NewEvidence(portID(303), scope, d.EvidenceTypeSource, "Observed", d.SourceReference{Provider: "fixture", ID: "source"}, cc.Actor, now, &artifact.ID, nil, "v1", "", now)
			must(e)
			must(docs.Evidence().Insert(ctx, evidence))
			link, e := d.NewEvidenceLink(portID(304), scope, evidence.ID, works[0].Ref(), nil, d.EvidenceStanceSupports, "observed", now)
			must(e)
			must(docs.EvidenceLinks().Insert(ctx, link))
			_, e = u.Coordination().AdvanceOutcome(ctx, scope)
			must(e)
			must(u.Commit())
			legacy, e := a.NewService(legacyFocalManager{store}, clock, ids)
			must(e)
			bounded, e := a.NewService(noHydrationManager{store}, clock, ids)
			must(e)
			for _, root := range []d.EntityRef{out.Value.Ref(), works[0].Ref(), issue.Ref(), evidence.Ref()} {
				for _, direction := range []string{"out", "in", "both"} {
					for _, depth := range []int{0, 1, 2, 8} {
						for _, kinds := range [][]string{nil, {"relates_to"}, {"ownership", "supports", "artifact_source"}} {
							q := a.GraphQuery{Root: &root, Direction: direction, Depth: depth, Limit: 3, Kinds: kinds}
							for pages := 0; ; pages++ {
								if pages > 20 {
									t.Fatal("graph cursor did not terminate")
								}
								want, e := legacy.GetOutcomeGraph(ctx, scope, q)
								must(e)
								got, e := bounded.GetOutcomeGraph(ctx, scope, q)
								must(e)
								assertFocalJSON(t, want, got)
								if got.NextCursor == "" {
									break
								}
								q.Cursor = got.NextCursor
							}
						}
					}
				}
			}
			unsupported := link.Ref()
			if _, e = bounded.GetOutcomeGraph(ctx, scope, a.GraphQuery{Root: &unsupported, Depth: 1}); securityErrorCode(e) != d.ErrorCodeNotFound {
				t.Fatal("non-node entity became a graph root", e)
			}
			first, e := bounded.GetOutcomeGraph(ctx, scope, a.GraphQuery{Depth: 1, Limit: 1})
			must(e)
			_, e = service.CreateWorkItem(ctx, cc, a.CreateWorkItemCommand{Scope: scope, Title: "Revision change", Priority: d.PriorityNormal})
			must(e)
			if _, e = bounded.GetOutcomeGraph(ctx, scope, a.GraphQuery{Depth: 1, Limit: 1, Cursor: first.NextCursor}); securityErrorCode(e) != d.ErrorCodePreconditionFailed {
				t.Fatal("stale graph cursor was accepted", e)
			}
		})
	}
}
