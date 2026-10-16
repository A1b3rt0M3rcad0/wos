package sqlite

import (
	"context"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPrincipalQuotasAdminCASAndConcurrentCredentials(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			f := newResolutionFixture(t, backend)
			ctx := context.Background()
			must := func(e error) {
				t.Helper()
				if e != nil {
					t.Fatal(e)
				}
			}
			ids := &loadIDs{}
			clock := sqliteFixedClock{time.Now().UTC()}
			ns := d.ID(f.request.NamespaceID)
			scope := d.Scope{NamespaceID: ns, OutcomeID: d.ID(f.request.OutcomeID)}
			sec := a.SecurityService{Store: f.store, Clock: clock, IDs: ids}
			admin, e := sec.Authenticate(ctx, "signed-acquisition-bootstrap-token-at-least-32")
			must(e)
			adminCtx := a.WithIdentity(ctx, admin)
			service, e := a.NewService(f.store, clock, ids)
			must(e)
			must(service.ConfigureSignedIssuer(ctx, acquisitionIssuer{f.server, f.issuerPrivate}, d.AcceptanceDirect))
			first, e := sec.SigningIdentity(f.ctx, nil)
			must(e)
			if first.Quotas.MaxActiveWorkPerPrincipal != 3 || first.Quotas.MaxActiveReviewPerPrincipal != 1 || first.Quotas.ActivePrincipalWork != 1 || first.Quotas.EffectiveWorkCeiling != 3 {
				t.Fatal("old defaults/activity changed", first.Quotas)
			}
			cred, token, e := sec.IssueCredential(adminCtx, ns, f.cc.PrincipalID, f.cc.Actor, clock.Now().Add(time.Hour))
			must(e)
			other, e := sec.Authenticate(ctx, token)
			must(e)
			otherCtx := a.WithIdentity(ctx, other)
			u, e := f.store.Begin(ctx)
			must(e)
			reg := u.(ports.SigningIdentityUnitOfWork).SigningIdentity()
			cp := *first.Policy
			cp.CredentialID = cred.ID
			must(reg.SaveCredentialPolicy(ctx, cp, 0))
			out, e := d.NewOutcome(portID(90000), ns, "Other quota Outcome", "", "same Principal is same quota", d.PriorityNormal, clock.Now())
			must(e)
			criterion, e := d.NewSuccessCriterion(portID(90001), out.Ref(), "Configured outcome proof", "", true, d.VerificationModeAttestation)
			must(e)
			must(out.AddCriterion(criterion, clock.Now()))
			must(out.Activate(clock.Now()))
			must(u.Outcomes().Insert(ctx, out))
			works := []d.WorkItem{}
			for i := 0; i < 52; i++ {
				target := scope
				if i%2 == 1 {
					target = out.Scope()
				}
				w, e := d.NewWorkItem(portID(5000+i), target, "Concurrent quota target", "", d.PriorityNormal, d.WorkItemLifecycleTodo, clock.Now())
				must(e)
				w.ContractsEnabled = true
				must(u.WorkItems().Insert(ctx, w))
				works = append(works, w)
			}
			must(u.Commit())
			command := func(i int) (d.CommandContext, a.AcquireSignedWorkContractCommand) {
				return d.CommandContext{PrincipalID: f.cc.PrincipalID, Actor: f.cc.Actor, CommandID: portID(6000 + i), IdempotencyKey: fmt.Sprintf("principal-quota-acquire-%d", i)}, a.AcquireSignedWorkContractCommand{Scope: works[i].Scope, WorkItemID: works[i].ID, ExpectedWorkItemVersion: works[i].Version, SignerKeyID: d.ID(f.request.SignerKeyID), TTLSeconds: 300}
			}
			for i, caller := range []context.Context{f.ctx, otherCtx} {
				cc, cmd := command(i)
				_, e := service.AcquireSignedWorkContract(caller, cc, cmd)
				must(e)
			}
			cc, cmd := command(2)
			before, e := service.GetWorkContext(otherCtx, cmd.Scope, cmd.WorkItemID)
			must(e)
			if _, e = service.AcquireSignedWorkContract(otherCtx, cc, cmd); e == nil || !strings.Contains(e.Error(), "Principal") {
				t.Fatal("second CID bypassed default quota", e)
			}
			after, e := service.GetWorkContext(otherCtx, cmd.Scope, cmd.WorkItemID)
			must(e)
			if before.OutcomeRevision != after.OutcomeRevision || after.Work.Version != before.Work.Version || after.Work.CurrentContractID != nil {
				t.Fatal("denied acquisition consumed state")
			}
			seven, one := 7, 1
			policy := d.WorkAcceptancePolicy{NamespaceID: ns, AcceptanceFloor: d.AcceptanceDirect, MaxActiveWorkContracts: 100, MaxActiveReviewContracts: 100, MaxActiveWorkPerPrincipal: &seven, MaxActiveReviewPerPrincipal: &one}
			view, e := sec.SigningIdentity(adminCtx, nil)
			must(e)
			intent := a.SigningSecurityIntent{NamespaceID: ns, Operation: "set_acceptance_policy", ExpectedNamespaceVersion: view.NamespaceVersion, AcceptancePolicy: &policy}
			if _, e = sec.SigningMutation(f.ctx, "quota-unauthorized", intent); securityErrorCode(e) != d.ErrorCodeForbidden {
				t.Fatal("agent elevated its quota", e)
			}
			scoped := policy
			scoped.OutcomeID = &scope.OutcomeID
			invalid := intent
			invalid.AcceptancePolicy = &scoped
			if _, e = sec.SigningMutation(adminCtx, "quota-scoped-override", invalid); securityErrorCode(e) != d.ErrorCodeInvalidArgument {
				t.Fatal("scoped override accepted", e)
			}
			configured, e := sec.SigningMutation(adminCtx, "quota-admin-raise", intent)
			must(e)
			if _, e = sec.SigningMutation(adminCtx, "quota-stale-admin", intent); securityErrorCode(e) != d.ErrorCodeVersionConflict {
				t.Fatal("stale Namespace CAS accepted", e)
			}
			var success, conflicts, retries atomic.Uint64
			var wg sync.WaitGroup
			failures := make(chan error, 50)
			start := make(chan struct{})
			for i := 2; i < 52; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					<-start
					caller := f.ctx
					if i%2 == 1 {
						caller = otherCtx
					}
					cc, cmd := command(i)
					_, e := retrySignedLoadMutation(func() (a.MutationResult[a.WorkContractResult], error) {
						return service.AcquireSignedWorkContract(caller, cc, cmd)
					}, &conflicts, &retries)
					if e == nil {
						success.Add(1)
						return
					}
					if securityErrorCode(e) != d.ErrorCodePreconditionFailed || !strings.Contains(e.Error(), "Principal") {
						failures <- e
					}
				}(i)
			}
			close(start)
			wg.Wait()
			close(failures)
			for e := range failures {
				t.Error(e)
			}
			if success.Load() != 4 {
				t.Fatal("concurrent credentials exceeded or failed configured quota", success.Load())
			}
			status, e := sec.SigningIdentity(otherCtx, nil)
			must(e)
			if status.Quotas.ActivePrincipalWork != 7 || status.Quotas.MaxActiveWorkPerPrincipal != 7 || status.Quotas.EffectiveWorkCeiling != 7 {
				t.Fatal("quota diagnostics lost aggregate Principal", status.Quotas)
			}
			// Lowering a ceiling keeps already-issued authority intact and blocks new work.
			two := 2
			policy.MaxActiveWorkPerPrincipal = &two
			changed, e := sec.SigningMutation(adminCtx, "quota-lower-active", a.SigningSecurityIntent{NamespaceID: ns, Operation: "set_acceptance_policy", ExpectedNamespaceVersion: configured.NamespaceVersion, ExpectedVersion: 1, AcceptancePolicy: &policy})
			must(e)
			_ = changed
			original, e := service.ReadSignedState(f.ctx, a.SignedStateQuery{Scope: scope, Resource: "execution", ID: d.ID(f.request.ContractID)})
			must(e)
			if original.Contract == nil || original.Contract.Status != d.ContractActive || original.Contract.FencingToken != f.request.FencingToken {
				t.Fatal("lowering ceiling removed existing authority")
			}
			status, e = sec.SigningIdentity(otherCtx, nil)
			must(e)
			if status.Quotas.ActivePrincipalWork != 7 || status.Quotas.EffectiveWorkCeiling != 2 {
				t.Fatal("lowered quota not applied", status.Quotas)
			}
			t.Logf("50 consumers, two credentials, two Outcomes: newly acquired=%d conflicts=%d exact retries=%d", success.Load(), conflicts.Load(), retries.Load())
		})
	}
}

func TestPrincipalQuotasKeepCredentialNamespaceAndRevocationIndependent(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			f := newResolutionFixture(t, backend)
			ctx := context.Background()
			must := func(e error) {
				t.Helper()
				if e != nil {
					t.Fatal(e)
				}
			}
			clock := sqliteFixedClock{time.Now().UTC()}
			sec := a.SecurityService{Store: f.store, Clock: clock, IDs: &loadIDs{}}
			ns := d.ID(f.request.NamespaceID)
			scope := d.Scope{NamespaceID: ns, OutcomeID: d.ID(f.request.OutcomeID)}
			admin, e := sec.Authenticate(ctx, "signed-acquisition-bootstrap-token-at-least-32")
			must(e)
			adminCtx := a.WithIdentity(ctx, admin)
			serial := 0
			mutate := func(p *d.WorkAcceptancePolicy, cp *d.CredentialPolicy, expected d.Version) {
				serial++
				v, e := sec.SigningIdentity(adminCtx, nil)
				must(e)
				operation := "set_acceptance_policy"
				if cp != nil {
					operation = "set_credential_policy"
				}
				_, e = sec.SigningMutation(adminCtx, fmt.Sprintf("independent-ceiling-%d", serial), a.SigningSecurityIntent{NamespaceID: ns, Operation: operation, ExpectedNamespaceVersion: v.NamespaceVersion, ExpectedVersion: expected, AcceptancePolicy: p, CredentialPolicy: cp})
				must(e)
			}
			identity, e := sec.SigningIdentity(f.ctx, nil)
			must(e)
			cp := *identity.Policy
			cp.MaxActiveWorkContracts = 1
			mutate(nil, &cp, 1)
			five := 5
			policy := d.WorkAcceptancePolicy{NamespaceID: ns, AcceptanceFloor: d.AcceptanceDirect, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 3, MaxActiveWorkPerPrincipal: &five}
			mutate(&policy, nil, 0)
			acquire := func(caller context.Context, principal string, actor d.ActorRef, key d.ID, index int) (a.MutationResult[a.WorkContractResult], error) {
				serial++
				return f.service.AcquireSignedWorkContract(caller, d.CommandContext{PrincipalID: principal, Actor: actor, CommandID: portID(80000 + serial), IdempotencyKey: fmt.Sprintf("independent-acquire-%d", serial)}, a.AcquireSignedWorkContractCommand{Scope: scope, WorkItemID: testID(fmt.Sprintf("0199a555-0000-7000-8000-%012x", 20+index)), ExpectedWorkItemVersion: 1, SignerKeyID: key, TTLSeconds: 300})
			}
			if _, e = acquire(f.ctx, f.cc.PrincipalID, f.cc.Actor, d.ID(f.request.SignerKeyID), 1); e == nil || !strings.Contains(e.Error(), "credential") {
				t.Fatal("Principal increase bypassed CID ceiling", e)
			}
			participant := func(principal string, key d.ID, template d.CredentialPolicy) (a.Identity, context.Context) {
				cred, token, e := sec.IssueCredential(adminCtx, ns, principal, d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: principal}, clock.Now().Add(time.Hour))
				must(e)
				id, e := sec.Authenticate(ctx, token)
				must(e)
				template.CredentialID = cred.ID
				template.PrincipalID = principal
				template.AllowedSigningKeyIDs = []d.ID{key}
				template.MaxActiveWorkContracts = 4
				mutate(nil, &template, 0)
				return id, a.WithIdentity(ctx, id)
			}
			second, secondCtx := participant(f.cc.PrincipalID, d.ID(f.request.SignerKeyID), cp)
			_, e = acquire(secondCtx, second.PrincipalID, second.Actor, d.ID(f.request.SignerKeyID), 1)
			must(e)
			must(sec.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "another-executor", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionWorkContractAcquire, ports.PermissionWorkWrite}}))
			foreignKey := testID("0199a555-0000-7000-8000-000000000099")
			foreign, foreignCtx := participant("another-executor", foreignKey, cp)
			_, e = acquire(foreignCtx, foreign.PrincipalID, foreign.Actor, foreignKey, 2)
			must(e)
			if _, e = acquire(secondCtx, second.PrincipalID, second.Actor, d.ID(f.request.SignerKeyID), 3); e == nil || !strings.Contains(e.Error(), "Namespace") {
				t.Fatal("two Principals bypassed Namespace ceiling", e)
			}
			must(sec.RevokeCredential(adminCtx, ns, second.CredentialID))
			if _, e = acquire(secondCtx, second.PrincipalID, second.Actor, d.ID(f.request.SignerKeyID), 3); securityErrorCode(e) != d.ErrorCodeForbidden {
				t.Fatal("stale authenticated context bypassed CID revocation", e)
			}
			one := 1
			policy.MaxActiveWorkContracts = 10
			policy.MaxActiveWorkPerPrincipal = &one
			mutate(&policy, nil, 1)
			third, thirdCtx := participant(f.cc.PrincipalID, d.ID(f.request.SignerKeyID), cp)
			if _, e = acquire(thirdCtx, third.PrincipalID, third.Actor, d.ID(f.request.SignerKeyID), 3); e == nil || !strings.Contains(e.Error(), "Principal") {
				t.Fatal("new CID bypassed lowered Principal ceiling", e)
			}
		})
	}
}
