package sqlite

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"path/filepath"
	"testing"
	"time"
)

func TestIndexedExternalContextPreservesTypeAndRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "context.db")
	store := openTestStore(t, path)
	clock := sqliteFixedClock{now: time.Now().UTC()}
	ids := &sqliteSequenceIDs{prefix: "0199d098", next: 1}
	service, _ := application.NewService(store, clock, ids)
	ns := testID("0199d098-0000-7000-8000-000000000010")
	cc := sqliteCommandContext("0199d098-0000-7000-8000-000000000011", "")
	var outcomes []domain.Outcome
	for _, value := range []any{42, "42", true, nil} {
		created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{NamespaceID: ns, Title: "Context", DesiredState: "continues", Priority: domain.PriorityNormal, ExternalContext: domain.ExternalContext{"product": "reference", "user_id": value}})
		if err != nil {
			t.Fatal(err)
		}
		outcomes = append(outcomes, created.Value)
	}
	for _, value := range []any{42.0, "42", true, nil} {
		page, err := service.SearchOutcomes(ctx, ns, ports.OutcomeFilter{ExternalContext: domain.ExternalContext{"product": "reference", "user_id": value}}, 25, "")
		if err != nil || len(page.Items) != 1 {
			t.Fatalf("typed lookup %v: %+v %v", value, page, err)
		}
	}
	changed, err := service.ReplaceExternalContext(ctx, cc, application.ReplaceExternalContextCommand{Scope: outcomes[0].Scope(), ExpectedVersion: outcomes[0].Version, ExternalContext: domain.ExternalContext{"product": "updated", "user_id": 42}})
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.SearchOutcomes(ctx, ns, ports.OutcomeFilter{ExternalContext: domain.ExternalContext{"product": "reference", "user_id": 42}}, 25, "")
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("obsolete index: %+v %v", page, err)
	}
	page, err = service.SearchOutcomes(ctx, ns, ports.OutcomeFilter{CreatorPrincipalID: cc.PrincipalID}, 25, "")
	if err != nil || len(page.Items) != 4 {
		t.Fatalf("authorship: %+v %v", page, err)
	}
	store.Close()
	restarted := openTestStore(t, path)
	service, _ = application.NewService(restarted, clock, ids)
	page, err = service.SearchOutcomes(ctx, ns, ports.OutcomeFilter{ExternalContext: domain.ExternalContext{"product": "updated", "user_id": 42}}, 25, "")
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != changed.Value.ID {
		t.Fatalf("restart lookup: %+v %v", page, err)
	}
	read, err := service.GetOutcome(ctx, changed.Value.Scope())
	if err != nil || read.Value.ExternalContext["product"] != "updated" {
		t.Fatal("context not restored", err)
	}
	for _, invalid := range []domain.ExternalContext{{"wos.admin": true}, {"nested": map[string]any{"tenant_id": "other"}}, {"array": []string{"x"}}} {
		if _, err := service.ReplaceExternalContext(ctx, cc, application.ReplaceExternalContextCommand{Scope: changed.Value.Scope(), ExpectedVersion: changed.Value.Version, ExternalContext: invalid}); err == nil {
			t.Fatal("accepted invalid context")
		}
	}
}
