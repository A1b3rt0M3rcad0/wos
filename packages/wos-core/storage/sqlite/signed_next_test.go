package sqlite

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSignedNextSearchDurablyReplaysEmptyAndPreservesScanCursor(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			var store interface {
				ports.SecurityStore
				ports.TransactionManager
			}
			path := filepath.Join(t.TempDir(), "signed-acquisition.db")
			if backend == "memory" {
				store = memory.New()
			} else {
				store = openTestStore(t, path)
			}
			ctx := context.Background()
			now := time.Now().UTC()
			clock := sqliteFixedClock{now}
			ids := &sqliteSequenceIDs{prefix: "0199a556", next: 1}
			sec := a.SecurityService{Store: store, Clock: clock, IDs: ids}
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			ns := testID("0199a555-0000-7000-8000-000000000001")
			scope := d.Scope{NamespaceID: ns, OutcomeID: testID("0199a555-0000-7000-8000-000000000002")}
			must(sec.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "signed-acquisition"}, "admin", "signed-acquisition-bootstrap-token-at-least-32"))
			admin, err := sec.Authenticate(ctx, "signed-acquisition-bootstrap-token-at-least-32")
			must(err)
			adminCtx := a.WithIdentity(ctx, admin)
			must(sec.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "executor", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionWorkContractAcquire, ports.PermissionWorkWrite}}))
			credential, token, err := sec.IssueCredential(adminCtx, ns, "executor", d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}, now.Add(time.Hour))
			must(err)
			identity, err := sec.Authenticate(ctx, token)
			must(err)
			agentCtx := a.WithIdentity(ctx, identity)
			pub, private, err := ed25519.GenerateKey(rand.Reader)
			must(err)
			fp, err := signing.Fingerprint(pub)
			must(err)
			server := d.ServerIdentity{ID: testID("0199a555-0000-7000-8000-000000000003"), IssuerKeyID: testID("0199a555-0000-7000-8000-000000000004"), PublicKey: base64.StdEncoding.EncodeToString(pub), Fingerprint: fp, CreatedAt: now}
			service, err := a.NewService(store, clock, ids)
			must(err)
			must(service.ConfigureSignedIssuer(ctx, acquisitionIssuer{server, private}, d.AcceptanceDirect))
			agentPub, _, err := ed25519.GenerateKey(rand.Reader)
			must(err)
			agentFP, err := signing.Fingerprint(agentPub)
			must(err)
			agentKey := testID("0199a555-0000-7000-8000-000000000005")
			uow, err := store.Begin(ctx)
			must(err)
			registry := uow.(ports.SigningIdentityUnitOfWork).SigningIdentity()
			must(registry.SaveKey(ctx, d.SigningKey{ID: agentKey, NamespaceID: ns, PrincipalID: "executor", Purpose: "agent", Algorithm: "Ed25519", PublicKey: base64.StdEncoding.EncodeToString(agentPub), Fingerprint: agentFP, Version: 1, Status: "active", CreatedAt: now, UpdatedAt: now}, 0))
			policy := d.CredentialPolicy{NamespaceID: ns, CredentialID: credential.ID, PrincipalID: "executor", Version: 1, PermittedOperations: []string{string(ports.PermissionStateRead), string(ports.PermissionWorkContractAcquire), string(ports.PermissionWorkWrite)}, AcceptanceFloor: d.AcceptanceIndependentReview, AllowedSigningKeyIDs: []d.ID{agentKey}, MaxActiveWorkContracts: 10, MaxActiveReviewContracts: 1, UpdatedAt: now}
			must(registry.SaveCredentialPolicy(ctx, policy, 0))
			outcome, err := d.NewOutcome(scope.OutcomeID, ns, "signed workload", "", "verified", d.PriorityNormal, now)
			must(err)
			outcome.Lifecycle = d.OutcomeLifecycleActive
			must(uow.Outcomes().Insert(ctx, outcome))
			works := []d.WorkItem{}
			for i := 0; i < 4; i++ {
				w, e := d.NewWorkItem(testID(fmt.Sprintf("0199a555-0000-7000-8000-%012x", 20+i)), scope, "precise signed task", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
				must(e)
				w.ContractsEnabled = true
				must(uow.WorkItems().Insert(ctx, w))
				works = append(works, w)
			}
			protocol := uow.(ports.WorkProtocolUnitOfWork).WorkProtocol()
			state, err := protocol.Lock(ctx, ns, true)
			must(err)
			state.Phase = d.WorkProtocolSigned
			state.WriterEpoch = 2
			state.WritersDrained = true
			state.Version++
			state.UpdatedAt = now
			state.UpdatedBy = "operator"
			state.Reason = "test-only operator activation; public cutover remains separately gated"
			must(protocol.Save(ctx, state, 1))
			_, err = uow.Coordination().AdvanceOutcome(ctx, scope)
			must(err)
			must(uow.Commit())

			future := testID("0199a555-0000-7000-8000-000000000099")
			commandID, e := ids.NewID()
			must(e)
			cc := d.CommandContext{PrincipalID: identity.PrincipalID, Actor: identity.Actor, CommandID: commandID, IdempotencyKey: "signed-empty-search-original"}
			cmd := a.AcquireNextSignedWorkContractCommand{Scope: scope, SignerKeyID: agentKey, WorkItemIDs: []d.ID{future}, TTLSeconds: 300, Limit: 100}
			empty, e := service.AcquireNextSignedWorkContract(agentCtx, cc, cmd)
			must(e)
			if empty.Value.Acquired || !empty.Value.SearchComplete || empty.Value.Result != nil {
				t.Fatal("empty search fabricated acquisition")
			}
			u, e := store.Begin(ctx)
			must(e)
			work, e := d.NewWorkItem(future, scope, "new task after original empty intention", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
			must(e)
			work.ContractsEnabled = true
			must(u.WorkItems().Insert(ctx, work))
			must(u.Commit())
			noCache, e := a.NewService(signedNoCacheManager{base: store}, clock, ids)
			must(e)
			cc.CommandID, e = ids.NewID()
			must(e)
			replay, e := noCache.AcquireNextSignedWorkContract(agentCtx, cc, cmd)
			must(e)
			if !replay.IdempotentReplay || replay.CommandID != empty.CommandID || replay.OutcomeRevision != empty.OutcomeRevision || !reflect.DeepEqual(replay.Value, empty.Value) {
				t.Fatal("empty intention became a new search")
			}

			// A technical expansion has a separate response bound, unlike ordinary views.
			bigResponse, e := json.Marshal(map[string]any{"value": []string{strings.Repeat("x", 60<<10), strings.Repeat("x", 60<<10), strings.Repeat("x", 60<<10), strings.Repeat("x", 60<<10), strings.Repeat("x", 60<<10)}})
			must(e)
			bigID, e := ids.NewID()
			must(e)
			operationFingerprint, e := a.SignedCommandFingerprint(identity.Actor, cmd)
			must(e)
			unit, e := store.Begin(ctx)
			must(e)
			must(unit.(ports.SignedOperationUnitOfWork).SignedOperations().Insert(ctx, d.SignedOperationResult{Scope: scope, PrincipalID: identity.PrincipalID, CredentialID: credential.ID, CommandName: "AcquireNextSignedWorkContract", IdempotencyKey: "large-technical-expansion", Fingerprint: operationFingerprint, RecordedAt: now, Result: d.StoredCommandResult{CommandID: bigID, OutcomeRevision: empty.OutcomeRevision, ResponseJSON: bigResponse}}))
			must(unit.Commit())
			bigView, e := service.ReadSignedState(agentCtx, a.SignedStateQuery{Scope: scope, Resource: "operation", IdempotencyKey: "large-technical-expansion"})
			must(e)
			decoded, e := base64.StdEncoding.Strict().DecodeString(bigView.OperationPayload)
			must(e)
			if string(decoded) != string(bigResponse) || bigView.PayloadDigest != signing.Digest(bigResponse) || bigView.OperationFingerprint != operationFingerprint {
				t.Fatal("server truncated or changed bounded technical result")
			}
			// A changed cursor/payload cannot reinterpret that intention.
			changed := cmd
			changed.Limit = 1
			if _, e = service.AcquireNextSignedWorkContract(agentCtx, cc, changed); e == nil {
				t.Fatal("changed scan reused original intention")
			}
			cc.IdempotencyKey = "signed-incomplete-first-page"
			cc.CommandID, e = ids.NewID()
			must(e)
			page, e := service.AcquireNextSignedWorkContract(agentCtx, cc, changed)
			must(e)
			if page.Value.Acquired || page.Value.SearchComplete || page.Value.NextCursor == "" || !reflect.DeepEqual(page.Value.Reasons, []string{"candidate_search_incomplete"}) {
				t.Fatal("incomplete empty page declared global absence")
			}
			cc.IdempotencyKey = "signed-next-new-intention"
			cc.CommandID, e = ids.NewID()
			must(e)
			fresh, e := service.AcquireNextSignedWorkContract(agentCtx, cc, cmd)
			must(e)
			if !fresh.Value.Acquired || fresh.Value.Result == nil || fresh.Value.Result.WorkItem.ID != future || fresh.Value.Result.Contract.SignedBinding == nil || fresh.Value.Result.IssuedAuthority == nil {
				t.Fatal("fresh intention did not acquire signed task")
			}
			cc.CommandID, e = ids.NewID()
			must(e)
			acquiredReplay, e := noCache.AcquireNextSignedWorkContract(agentCtx, cc, cmd)
			must(e)
			originalJSON, _ := json.Marshal(fresh.Value)
			replayJSON, _ := json.Marshal(acquiredReplay.Value)
			if !acquiredReplay.IdempotentReplay || acquiredReplay.CommandID != fresh.CommandID || string(originalJSON) != string(replayJSON) {
				t.Fatal("acquired search replay changed original grant")
			}
			// Other two same-page candidates remain available; the Principal cap is three.
			unrestricted := cmd
			unrestricted.WorkItemIDs = nil
			for i := 0; i < 3; i++ {
				cc.IdempotencyKey = fmt.Sprintf("signed-next-independent-%d", i)
				cc.CommandID, e = ids.NewID()
				must(e)
				next, e := service.AcquireNextSignedWorkContract(agentCtx, cc, unrestricted)
				if i == 2 {
					if e == nil {
						t.Fatal("search bypassed Principal quota")
					}
					break
				}
				must(e)
				if !next.Value.Acquired || next.Value.Result.Contract.ID == fresh.Value.Result.Contract.ID {
					t.Fatal("search skipped or duplicated same-page work")
				}
			}
		})
	}
}
