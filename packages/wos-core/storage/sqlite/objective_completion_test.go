package sqlite

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestSQLiteObjectiveCompletionAuthorityRestartAppendOnlyAndAutomaticConstraint(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "objective-completion-authority.db")
	store, _, outcome, objective := newOperationalProfileFixtureAt(t, path)
	actor := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "sqlite-test", ID: "operator"}
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	makeAuthority := func(id string, version uint64) domain.ObjectiveCompletionAuthority {
		return domain.ObjectiveCompletionAuthority{
			ID: testID(id), Scope: outcome.Scope(), ObjectiveID: objective.ID,
			PolicyVersion: version, Mode: domain.ObjectiveCompletionModeManual, CompletionCycle: objective.EffectiveCompletionCycle(),
			DelegatorPrincipalID: "sqlite-operator", DelegatorActor: actor,
			OperationalProfile: outcome.OperationalProfile.Effective(), CompletionPolicy: outcome.CompletionPolicy.Effective(),
			AutomationActor: domain.ObjectiveCompletionAutomationActor(), Reason: "Persist explicit completion mode authority",
			AuthorizedAt: now,
		}
	}
	first := makeAuthority("019afe10-0000-7000-8000-000000000010", 1)
	second := makeAuthority("019afe10-0000-7000-8000-000000000011", 2)
	write, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	repository := write.(ports.ObjectiveCompletionAuthorityUnitOfWork).ObjectiveCompletionAuthorities()
	if err := repository.Append(ctx, first); err != nil {
		t.Fatalf("append first authority: %v", err)
	}
	if err := repository.Append(ctx, second); err != nil {
		t.Fatalf("append second authority: %v", err)
	}
	if err := write.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path, Options{BusyTimeout: 500 * time.Millisecond, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	read, err := reopened.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	history, err := read.(ports.ObjectiveCompletionAuthorityUnitOfWork).ObjectiveCompletionAuthorities().ListByObjective(ctx, outcome.Scope(), objective.ID)
	if err != nil || !reflect.DeepEqual(history, []domain.ObjectiveCompletionAuthority{first, second}) {
		t.Fatalf("authority history after restart = %+v err=%v", history, err)
	}
	if err := read.Rollback(); err != nil {
		t.Fatal(err)
	}

	if _, err := reopened.db.ExecContext(ctx, `UPDATE objective_completion_authority_history SET reason='tampered' WHERE id=?`, first.ID.String()); err == nil {
		t.Fatal("immutable completion authority history allowed update")
	}
	if _, err := reopened.db.ExecContext(ctx, `DELETE FROM objective_completion_authority_history WHERE id=?`, first.ID.String()); err == nil {
		t.Fatal("immutable completion authority history allowed deletion")
	}
	// A direct SQL writer cannot create automatic mode without an exact
	// credential and policy revision, even if the profile snapshot is strict.
	_, err = reopened.db.ExecContext(ctx, `INSERT INTO objective_completion_authority_history (
 id, namespace_id, outcome_id, objective_id, policy_version, mode, completion_cycle,
 delegator_principal_id, delegator_actor_json, delegator_credential_id, credential_policy_revision,
 operational_profile, completion_policy, automation_actor_json, reason, authorized_at, authority_json
) VALUES (?,?,?,?,?,?,?,?,?,NULL,'0','operational_v2','strict_work',?,?,?,?)`,
		testID("019afe10-0000-7000-8000-000000000012").String(), outcome.NamespaceID.String(), outcome.ID.String(), objective.ID.String(),
		"3", "automatic", "1", "sqlite-operator", `{"kind":"human","provider":"sqlite-test","id":"operator"}`,
		`{"kind":"automation","provider":"wos","id":"objective_completion"}`, "unbound automatic authority", encodeTime(now), `{}`,
	)
	if err == nil {
		t.Fatal("SQLite accepted automatic completion authority without credential provenance")
	}
}
