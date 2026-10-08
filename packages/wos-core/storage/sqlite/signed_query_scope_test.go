package sqlite

import (
	"context"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"path/filepath"
	"testing"
	"time"
)

type beforeReadTransaction struct {
	base   ports.TransactionManager
	before func() error
}

func (tx *beforeReadTransaction) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	if tx.before != nil {
		callback := tx.before
		tx.before = nil
		if err := callback(); err != nil {
			return nil, err
		}
	}
	return tx.base.Begin(ctx)
}

func TestOutcomeRestrictionsCoverLegacyReadsDiscoveryAndSnapshotRevocation(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			var store interface {
				ports.SecurityStore
				ports.TransactionManager
			}
			if backend == "memory" {
				store = memory.New()
			} else {
				store = openTestStore(t, filepath.Join(t.TempDir(), "query-scope.db"))
			}
			ctx := context.Background()
			now := time.Now().UTC()
			clock := sqliteFixedClock{now}
			ids := &sqliteSequenceIDs{prefix: "0199a558", next: 1}
			sec := a.SecurityService{Store: store, Clock: clock, IDs: ids}
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			ns := testID("0199a557-0000-7000-8000-000000000001")
			must(sec.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "query-scope"}, "admin", "read-scope-bootstrap-token-at-least-32"))
			admin, err := sec.Authenticate(ctx, "read-scope-bootstrap-token-at-least-32")
			must(err)
			adminCtx := a.WithIdentity(ctx, admin)
			must(sec.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "reader", Permissions: []ports.Permission{ports.PermissionStateRead}}))
			credential, token, err := sec.IssueCredential(adminCtx, ns, "reader", d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "reader"}, now.Add(time.Hour))
			must(err)
			identity, err := sec.Authenticate(ctx, token)
			must(err)
			readerCtx := a.WithIdentity(ctx, identity)
			first := testID("0199a557-0000-7000-8000-000000000020")
			hidden := testID("0199a557-0000-7000-8000-000000000021")
			last := testID("0199a557-0000-7000-8000-000000000022")
			policy := d.CredentialPolicy{NamespaceID: ns, CredentialID: credential.ID, PrincipalID: identity.PrincipalID, Version: 1, PermittedOperations: []string{string(ports.PermissionStateRead)}, AllowedOutcomeIDs: []d.ID{first, last}, AcceptanceFloor: d.AcceptanceDirect, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1, UpdatedAt: now}
			u, err := store.Begin(ctx)
			must(err)
			for _, id := range []d.ID{first, hidden, last} {
				outcome, err := d.NewOutcome(id, ns, "visible only in permitted scope", "", "verified", d.PriorityNormal, now)
				must(err)
				must(u.Outcomes().Insert(ctx, outcome))
			}
			must(u.(ports.SigningIdentityUnitOfWork).SigningIdentity().SaveCredentialPolicy(ctx, policy, 0))
			must(u.Commit())
			service, err := a.NewAuthorizedService(store, clock, ids, &sec)
			must(err)
			_, err = service.GetOutcome(readerCtx, d.Scope{NamespaceID: ns, OutcomeID: first})
			must(err)
			if _, err = service.GetOutcome(readerCtx, d.Scope{NamespaceID: ns, OutcomeID: hidden}); err == nil {
				t.Fatal("Outcome restriction bypassed through legacy entity query")
			}
			if _, err = service.GetContinuity(readerCtx, d.Scope{NamespaceID: ns, OutcomeID: hidden}, 10); err == nil {
				t.Fatal("Outcome restriction bypassed through continuity")
			}
			page, err := service.SearchOutcomes(readerCtx, ns, ports.OutcomeFilter{AllowedOutcomeIDs: []d.ID{hidden}}, 1, "")
			must(err)
			if len(page.Items) != 1 || page.Items[0].ID != first || page.NextCursor == "" {
				t.Fatal("discovery ignored policy, caller widened it, or paging lost permitted candidates")
			}
			next, err := service.SearchOutcomes(readerCtx, ns, ports.OutcomeFilter{}, 1, page.NextCursor)
			must(err)
			if len(next.Items) != 1 || next.Items[0].ID != last || next.NextCursor != "" {
				t.Fatal("discovery exposed hidden Outcome through pagination")
			}
			update := func(allowed []d.ID) error {
				u, err := store.Begin(ctx)
				if err != nil {
					return err
				}
				defer u.Rollback()
				previous := policy.Version
				policy.Version++
				policy.AllowedOutcomeIDs = allowed
				if err = u.(ports.SigningIdentityUnitOfWork).SigningIdentity().SaveCredentialPolicy(ctx, policy, previous); err != nil {
					return err
				}
				return u.Commit()
			}
			must(update([]d.ID{hidden}))
			if _, err = service.SearchOutcomes(readerCtx, ns, ports.OutcomeFilter{}, 1, page.NextCursor); err == nil {
				t.Fatal("cursor reused after its permitted scope changed")
			}
			changed, err := service.SearchOutcomes(readerCtx, ns, ports.OutcomeFilter{}, 10, "")
			must(err)
			if len(changed.Items) != 1 || changed.Items[0].ID != hidden {
				t.Fatal("discovery used stale credential policy")
			}
			if _, err = service.GetCommandReceipt(readerCtx, ns, first); err == nil {
				t.Fatal("restricted reader accessed Namespace-wide receipt path")
			}
			// Narrow after preflight authorization but before the read snapshot opens.
			guarded := &beforeReadTransaction{base: store, before: func() error { return update([]d.ID{first}) }}
			racing, err := a.NewAuthorizedService(guarded, clock, ids, &sec)
			must(err)
			if _, err = racing.GetOutcome(readerCtx, d.Scope{NamespaceID: ns, OutcomeID: hidden}); err == nil {
				t.Fatal("policy hardening between preflight and snapshot leaked state")
			}
			must(update(nil))
			guarded.before = func() error {
				guarded.before = func() error { return update([]d.ID{first}) }
				return nil
			}
			if _, e := racing.GetCommandReceipt(readerCtx, ns, first); e == nil {
				t.Fatal("Namespace receipt ignored policy change before its own snapshot")
			} else if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeForbidden {
				t.Fatalf("receipt snapshot did not enforce restriction: %v", e)
			}
			guarded.before = func() error { return sec.RevokeCredential(adminCtx, ns, credential.ID) }
			if _, err = racing.GetOutcome(readerCtx, d.Scope{NamespaceID: ns, OutcomeID: first}); err == nil {
				t.Fatal("revocation between preflight and snapshot leaked state")
			}
		})
	}
}
