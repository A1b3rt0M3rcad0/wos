package memory_test

import (
	"context"
	"encoding/json"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type portClock struct{ now time.Time }

func (c portClock) Now() time.Time { return c.now }

type portIDs struct{ next int }

func (g *portIDs) NewID() (d.ID, error) { g.next++; return portID(10000 + g.next), nil }
func portID(n int) d.ID                 { return d.MustParseID(fmt.Sprintf("0199d180-0000-7000-8000-%012x", n)) }
func TestContinuityMetadataPagesMatchLegacySerializedSections(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	out, e := d.NewOutcome(portID(2), portID(1), "Port parity", "", "bounded pages", d.PriorityNormal, now)
	must(e)
	u, e := store.Begin(ctx)
	must(e)
	must(u.Outcomes().Insert(ctx, out))
	scope := out.Scope()
	actor := d.ActorRef{Kind: d.ActorKindService, Provider: "test", ID: "reader"}
	docs := u.(ports.DocumentaryUnitOfWork)
	issues := u.(ports.IssueBlockerUnitOfWork)
	works := []d.WorkItem{}
	providers := []string{"a\"", "a&", "a<", "a\\", "é", "\u2028x", "0", "z"}
	for i := 0; i < 8; i++ {
		title := fmt.Sprintf("%d & %s", i, strings.Repeat("界", 180))
		base := 100 + i*20
		work, e := d.NewWorkItem(portID(base), scope, title, "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
		must(e)
		must(u.WorkItems().Insert(ctx, work))
		works = append(works, work)
		objective, e := d.NewObjective(portID(base+1), scope, title, "", d.PriorityNormal, false, now)
		must(e)
		must(u.Objectives().Insert(ctx, objective))
		issue, e := d.NewIssue(portID(base+2), scope, title, "observed", d.IssueSeverityMinor, []d.EntityRef{work.Ref()}, actor, now)
		must(e)
		must(issues.Issues().Insert(ctx, issue))
		cause := issue.Ref()
		blocker, e := d.NewBlocker(portID(base+3), work.Ref(), &cause, nil, title, d.BlockerPropagationDirect, now)
		must(e)
		must(issues.Blockers().Insert(ctx, blocker))
		artifact, e := d.NewArtifact(portID(base+4), scope, "report", title, "https://example.test/report", "application/json", "", "v1", actor, nil, now)
		must(e)
		must(docs.Artifacts().Insert(ctx, artifact))
		evidence, e := d.NewEvidence(portID(base+5), scope, d.EvidenceTypeSource, title, d.SourceReference{Provider: "fixture", ID: fmt.Sprint(i)}, actor, now, &artifact.ID, nil, "v1", "", now)
		must(e)
		must(docs.Evidence().Insert(ctx, evidence))
		link, e := d.NewEvidenceLink(portID(base+6), scope, evidence.ID, work.Ref(), nil, d.EvidenceStanceSupports, "observed", now)
		must(e)
		must(docs.EvidenceLinks().Insert(ctx, link))
		decision, e := d.NewDecision(portID(base+7), scope, title, "choose", []string{"A", "B"}, "A", "reason", actor, now)
		must(e)
		must(docs.Decisions().Insert(ctx, decision))
		must(u.(ports.ExternalContextUnitOfWork).ExternalContexts().Add(ctx, scope, d.ExternalReference{Provider: providers[i], Kind: "ticket", ExternalID: fmt.Sprint(i), URL: "https://example.test/?a=1&b=2"}))
	}
	for i := 1; i < len(works); i++ {
		relation, e := d.NewDependencyRelation(portID(1000+i), works[i].Ref(), works[i-1].Ref(), d.DependencyStrengthHard, now)
		must(e)
		must(u.Relations().Insert(ctx, relation))
	}
	_, e = u.Coordination().AdvanceOutcome(ctx, scope)
	must(e)
	must(u.Commit())
	service, e := a.NewService(store, portClock{now}, &portIDs{})
	must(e)
	snapshot, e := service.GetContinuity(ctx, scope, 100)
	must(e)
	goldenRaw, e := os.ReadFile(filepath.Join("..", "..", "..", "..", "testdata", "continuity", "v042-snapshot.json"))
	must(e)
	actualRaw, e := json.Marshal(snapshot)
	must(e)
	var want, actual any
	must(json.Unmarshal(goldenRaw, &want))
	must(json.Unmarshal(actualRaw, &actual))
	normalizeGoldenEvaluation(want)
	normalizeGoldenEvaluation(actual)
	if !reflect.DeepEqual(want, actual) {
		t.Fatal("snapshot differs from immutable v0.4.2 golden")
	}
	for _, section := range []string{"work_items", "objectives", "issues", "blockers", "artifacts", "evidence", "decisions", "relations", "evidence_links", "external_references"} {
		t.Run(section, func(t *testing.T) {
			after, cursor := "", ""
			seen := 0
			for {
				legacy, e := service.GetContinuitySection(ctx, scope, section, 2, cursor)
				must(e)
				u, e := store.Begin(ctx)
				must(e)
				page, e := u.(ports.ContinuityReadUnitOfWork).ContinuityReads().Page(ctx, scope, section, after, 2)
				must(e)
				must(u.Rollback())
				values := make([]json.RawMessage, 0, len(page.Items))
				for _, item := range page.Items {
					values = append(values, item.Value)
				}
				if !reflect.DeepEqual(values, legacy.Items) || page.Count != legacy.Count || page.Remaining != legacy.Omitted {
					t.Fatalf("legacy mismatch: count=%d/%d remaining=%d/%d", page.Count, legacy.Count, page.Remaining, legacy.Omitted)
				}
				seen += len(page.Items)
				if page.Remaining == 0 {
					if seen != page.Count {
						t.Fatal("omitted/duplicate records")
					}
					break
				}
				after = page.Items[len(page.Items)-1].Key
				cursor = legacy.NextCursor
			}
		})
	}
}

// Evaluation time belongs to each database snapshot. Compare all other bytes
// semantically against the immutable published-source fixture; lease/cursor
// time consistency is tested separately rather than replacing persisted dates.
func normalizeGoldenEvaluation(value any) {
	switch v := value.(type) {
	case map[string]any:
		// The owner-authorized Tenant contract changes only the scope spelling.
		// Keep the published v0.4.2 fixture immutable; compare its known Scope
		// and EntityRef shapes after that explicit field projection.
		if _, scope := v["outcome_id"]; scope && (len(v) == 2 || len(v) == 4) {
			if id, old := v["namespace_id"]; old {
				v["tenant_id"] = id
				delete(v, "namespace_id")
			}
		}
		for key, item := range v {
			if key == "evaluated_at" {
				delete(v, key)
			} else {
				normalizeGoldenEvaluation(item)
			}
		}
	case []any:
		for _, item := range v {
			normalizeGoldenEvaluation(item)
		}
	}
}
