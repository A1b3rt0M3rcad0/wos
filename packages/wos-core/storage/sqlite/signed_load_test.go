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
	mathrand "math/rand/v2"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type signedLoadParticipant struct {
	identity a.Identity
	ctx      context.Context
	key      d.ID
	private  ed25519.PrivateKey
}

// This fixture measures the real signed application/storage path, not a model
// cost estimate or production/network SLA. Every delivery has an independent
// review and a required original attestation criterion.
func TestSignedWorkReviewLoadMatrix(t *testing.T) {
	if os.Getenv("WOS_SIGNED_LOAD") != "1" {
		t.Skip("opt-in signed workload; set WOS_SIGNED_LOAD=1")
	}
	for _, outcomes := range []int{1, 4} {
		for _, history := range []int{0, 16} {
			for _, consumers := range []int{2, 8} {
				t.Run(fmt.Sprintf("outcomes=%d/history=%d/consumers=%d", outcomes, history, consumers), func(t *testing.T) {
					ctx := context.Background()
					now := time.Now().UTC()
					clock := sqliteFixedClock{now}
					ids := &loadIDs{}
					store := openTestStore(t, filepath.Join(t.TempDir(), "signed-load.db"))
					must := func(err error) {
						t.Helper()
						if err != nil {
							t.Fatal(err)
						}
					}
					newID := func() d.ID { id, e := ids.NewID(); must(e); return id }
					sec := a.SecurityService{Store: store, Clock: clock, IDs: ids}
					ns := newID()
					must(sec.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "signed measured workload"}, "load-admin", "signed-load-owned-bootstrap-token-at-least-32"))
					admin, e := sec.Authenticate(ctx, "signed-load-owned-bootstrap-token-at-least-32")
					must(e)
					adminCtx := a.WithIdentity(ctx, admin)
					pub, private, e := ed25519.GenerateKey(rand.Reader)
					must(e)
					fp, e := signing.Fingerprint(pub)
					must(e)
					issuer := d.ServerIdentity{ID: newID(), IssuerKeyID: newID(), PublicKey: base64.StdEncoding.EncodeToString(pub), Fingerprint: fp, CreatedAt: now}
					service, e := a.NewService(store, clock, ids)
					must(e)
					must(service.ConfigureSignedIssuer(ctx, acquisitionIssuer{issuer, private}, d.AcceptanceIndependentReview))
					participant := func(name string, permissions []ports.Permission) signedLoadParticipant {
						must(sec.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: name, Permissions: permissions}))
						cred, token, e := sec.IssueCredential(adminCtx, ns, name, d.ActorRef{Kind: d.ActorKindAgent, Provider: "signed-load", ID: name}, now.Add(time.Hour))
						must(e)
						identity, e := sec.Authenticate(ctx, token)
						must(e)
						public, key, e := ed25519.GenerateKey(rand.Reader)
						must(e)
						fingerprint, e := signing.Fingerprint(public)
						must(e)
						kid := newID()
						u, e := store.Begin(ctx)
						must(e)
						registry := u.(ports.SigningIdentityUnitOfWork).SigningIdentity()
						must(registry.SaveKey(ctx, d.SigningKey{ID: kid, NamespaceID: ns, PrincipalID: name, Purpose: "agent", Algorithm: "Ed25519", PublicKey: base64.StdEncoding.EncodeToString(public), Fingerprint: fingerprint, Version: 1, Status: "active", CreatedAt: now, UpdatedAt: now}, 0))
						ops := []string{}
						for _, op := range permissions {
							ops = append(ops, string(op))
						}
						must(registry.SaveCredentialPolicy(ctx, d.CredentialPolicy{NamespaceID: ns, CredentialID: cred.ID, PrincipalID: name, Version: 1, PermittedOperations: ops, AcceptanceFloor: d.AcceptanceIndependentReview, AllowedSigningKeyIDs: []d.ID{kid}, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1, UpdatedAt: now}, 0))
						must(u.Commit())
						return signedLoadParticipant{identity, a.WithIdentity(ctx, identity), kid, key}
					}
					executors := []signedLoadParticipant{}
					reviewers := []signedLoadParticipant{}
					for i := 0; i < consumers; i++ {
						executors = append(executors, participant(fmt.Sprintf("executor-%d", i), []ports.Permission{ports.PermissionStateRead, ports.PermissionWorkContractAcquire, ports.PermissionWorkContractReturn, ports.PermissionWorkWrite, ports.PermissionAssessmentWrite, ports.PermissionConclusionWrite, ports.PermissionRecordsWrite}))
						reviewers = append(reviewers, participant(fmt.Sprintf("reviewer-%d", i), []ports.Permission{ports.PermissionStateRead, ports.PermissionWorkReviewAcquire, ports.PermissionWorkReviewDecide, ports.PermissionAssessmentWrite, ports.PermissionConclusionWrite, ports.PermissionRecordsWrite}))
					}
					scopes := []d.Scope{}
					u, e := store.Begin(ctx)
					must(e)
					for i := 0; i < outcomes; i++ {
						oid := newID()
						out, e := d.NewOutcome(oid, ns, "Measured signed Outcome", "", "Independent acceptance", d.PriorityNormal, now)
						must(e)
						out.Lifecycle = d.OutcomeLifecycleActive
						must(u.Outcomes().Insert(ctx, out))
						scopes = append(scopes, d.Scope{NamespaceID: ns, OutcomeID: oid})
					}
					protocol := u.(ports.WorkProtocolUnitOfWork).WorkProtocol()
					state, e := protocol.Lock(ctx, ns, true)
					must(e)
					state.Phase = d.WorkProtocolSigned
					state.WriterEpoch = 2
					state.WritersDrained = true
					state.Version++
					state.UpdatedAt = now
					state.UpdatedBy = "load-admin"
					state.Reason = "owned test-only fixture, public cutover separately validated"
					must(protocol.Save(ctx, state, 1))
					must(u.Commit())
					endpoint := ports.WebhookEndpoint{ID: newID(), NamespaceID: ns, URL: "https://example.invalid/signed-load", SecretRef: "test-only-webhook", KeyID: "load-fixture"}
					must(store.InstallEndpoints(ctx, []ports.WebhookEndpoint{endpoint}))
					for _, scope := range scopes {
						id := newID()
						_, e := service.ConfigureTrigger(adminCtx, d.CommandContext{PrincipalID: admin.PrincipalID, Actor: admin.Actor, CommandID: id, IdempotencyKey: "signed-load-trigger-" + id.String()}, a.ConfigureTriggerCommand{Scope: scope, Name: "Committed signed DONE", EventTypes: []string{"work_item.completed"}, TargetEndpointIDs: []d.ID{endpoint.ID}, SignalType: "test.signed_done"})
						must(e)
					}
					createTask := func(scope d.Scope) d.WorkItem {
						w, e := d.NewWorkItem(newID(), scope, "Signed bounded load task", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
						must(e)
						w.ContractsEnabled = true
						criterion, e := d.NewSuccessCriterion(newID(), w.Ref(), "Original obligation checked", "", true, d.VerificationModeAttestation)
						must(e)
						must(w.Criteria.Add(criterion))
						u, e := store.Begin(ctx)
						must(e)
						must(u.WorkItems().Insert(ctx, w))
						must(u.Commit())
						return w
					}
					var conflicts, retries atomic.Uint64
					next := func(p signedLoadParticipant) d.CommandContext {
						id := newID()
						return d.CommandContext{PrincipalID: p.identity.PrincipalID, Actor: p.identity.Actor, CommandID: id, IdempotencyKey: "signed-load-" + id.String()}
					}
					// The caller creates immutable command/envelope once, outside the retry. Only
					// transaction_conflict repeats that exact original CID and payload.
					runJourney := func(w d.WorkItem, ex, rev signedLoadParticipant) (int, error) {
						cc := next(ex)
						acq, e := retrySignedLoadMutation(func() (a.MutationResult[a.WorkContractResult], error) {
							return service.AcquireSignedWorkContract(ex.ctx, cc, a.AcquireSignedWorkContractCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version, SignerKeyID: ex.key, TTLSeconds: 300})
						}, &conflicts, &retries)
						if e != nil {
							return 0, e
						}
						raw, e := json.Marshal(acq.Value)
						if e != nil {
							return 0, e
						}
						size := len(raw)
						c := acq.Value.Contract
						work := acq.Value.WorkItem
						cc = next(ex)
						ret := signing.WorkReturnPayload[a.SignedReturnMaterial]{RequestBinding: signing.RequestBinding{Binding: signing.Binding{ProtocolVersion: 2, ServerID: issuer.ID.String(), NamespaceID: ns.String(), OutcomeID: w.Scope.OutcomeID.String(), PrincipalID: ex.identity.PrincipalID, SignerKeyID: ex.key.String()}, OperationKind: "work_return", RequestID: cc.CommandID.String(), IdempotencyKey: cc.IdempotencyKey, ContractID: c.ID.String(), WorkItemID: work.ID.String(), ExecutionID: c.ExecutionID.String(), FencingToken: signing.Decimal(c.FencingToken), SpecDigest: c.SignedBinding.SpecificationDigest, AuthorityDigest: signing.Digest(acq.Value.IssuedAuthority.Payload), ExpectedContractVersion: signing.Decimal(c.Version), ExpectedLeaseVersion: signing.Decimal(c.LeaseVersion), ExpectedWorkItemVersion: signing.Decimal(work.Version), PolicyRevision: 1}, CompletionIntent: "review", Material: a.SignedReturnMaterial{Result: d.NewSignedResultMaterial(d.WorkResultMaterial{ContractID: c.ID, WorkItemID: work.ID, SpecDigest: c.SignedBinding.SpecificationDigest, Summary: "load fixture submitted original bounded obligation"}), Reason: "explicit independent review"}}
						env, e := signing.Sign(signing.WorkReturn, ret, ex.key.String(), ex.private)
						if e != nil {
							return 0, e
						}
						accepted, e := retrySignedLoadMutation(func() (a.MutationResult[a.SignedReturnResult], error) {
							return service.ReturnSignedWork(ex.ctx, cc, a.ReturnSignedWorkCommand{Envelope: env})
						}, &conflicts, &retries)
						if e != nil {
							return 0, e
						}
						receiptEnv, e := accepted.Value.Receipt.Envelope()
						if e != nil {
							return 0, e
						}
						receipt, _, e := signing.Decode[signing.ReceiptPayload](receiptEnv, signing.AcceptanceReceipt, issuer.IssuerKeyID.String(), pub)
						if e != nil {
							return 0, e
						}
						events, e := store.SnapshotDomainEvents(ctx, w.Scope)
						if e != nil {
							return 0, e
						}
						for _, event := range events {
							if event.AggregateRef.ID == w.ID && event.EventType == "work_item.completed" {
								return 0, fmt.Errorf("DONE event before independent approval")
							}
						}
						cc = next(rev)
						reviewID := d.ID(receipt.ReviewCaseID)
						// A newly delivered case starts at version 1; acquisition returns the exact
						// post-acquisition case/version used by the signed decision.
						assigned, e := retrySignedLoadMutation(func() (a.MutationResult[a.SignedReviewContractResult], error) {
							return service.AcquireSignedReviewContract(rev.ctx, cc, a.AcquireSignedReviewContractCommand{Scope: w.Scope, ReviewCaseID: reviewID, ExpectedReviewCaseVersion: 1, SignerKeyID: rev.key, TTLSeconds: 300})
						}, &conflicts, &retries)
						if e != nil {
							return 0, e
						}
						rc := assigned.Value.Contract
						criterion := work.Criteria.Items[0]
						cc = next(rev)
						decision := signing.ReviewReturnPayload[a.SignedReviewMaterial]{RequestBinding: signing.RequestBinding{Binding: signing.Binding{ProtocolVersion: 2, ServerID: issuer.ID.String(), NamespaceID: ns.String(), OutcomeID: w.Scope.OutcomeID.String(), PrincipalID: rev.identity.PrincipalID, SignerKeyID: rev.key.String()}, OperationKind: "review_return", RequestID: cc.CommandID.String(), IdempotencyKey: cc.IdempotencyKey, ContractID: rc.ID.String(), WorkItemID: rc.WorkItemID.String(), ExecutionID: rc.ExecutionID.String(), FencingToken: signing.Decimal(rc.FencingToken), SpecDigest: rc.Binding.SpecificationDigest, AuthorityDigest: signing.Digest(assigned.Value.IssuedAuthority.Payload), ExpectedContractVersion: signing.Decimal(rc.Version), ExpectedLeaseVersion: signing.Decimal(rc.LeaseVersion), ExpectedWorkItemVersion: assigned.Value.WorkItemVersion, PolicyRevision: 1}, ExpectedReviewCaseVersion: signing.Decimal(assigned.Value.Case.Version), ReviewCaseID: rc.CaseID.String(), SubmissionID: rc.SubmissionID.String(), SubmissionDigest: rc.SubmissionDigest, Decision: "approved", Material: a.SignedReviewMaterial{Reason: "independent fixture assessment of original attestation criterion", Assessments: []a.SignedAssessmentInput{{CriterionID: criterion.ID, CriterionRevision: signing.Decimal(criterion.Revision), Result: d.AssessmentResultMet, Rationale: "verified original fixture requirement"}}}}
						env, e = signing.Sign(signing.ReviewReturn, decision, rev.key.String(), rev.private)
						if e != nil {
							return 0, e
						}
						approved, e := retrySignedLoadMutation(func() (a.MutationResult[a.SignedReturnResult], error) {
							return service.ReturnSignedReview(rev.ctx, cc, a.ReturnSignedReviewCommand{Envelope: env})
						}, &conflicts, &retries)
						if e != nil {
							return 0, e
						}
						proof, e := approved.Value.Receipt.Envelope()
						if e != nil {
							return 0, e
						}
						final, _, e := signing.Decode[signing.ReceiptPayload](proof, signing.AcceptanceReceipt, issuer.IssuerKeyID.String(), pub)
						if e != nil {
							return 0, e
						}
						if final.WorkItemLifecycle != "done" || !final.LocalObligationClosed {
							return 0, fmt.Errorf("unaccepted independent review")
						}
						replay, e := retrySignedLoadMutation(func() (a.MutationResult[a.SignedReturnResult], error) {
							return service.ReturnSignedReview(rev.ctx, cc, a.ReturnSignedReviewCommand{Envelope: env})
						}, &conflicts, &retries)
						if e != nil {
							return 0, e
						}
						if !replay.IdempotentReplay || string(replay.Value.Receipt.Payload) != string(approved.Value.Receipt.Payload) {
							return 0, fmt.Errorf("signed decision replay changed original receipt")
						}
						events, e = store.SnapshotDomainEvents(ctx, w.Scope)
						if e != nil {
							return 0, e
						}
						completed := 0
						for _, event := range events {
							if event.AggregateRef.ID == w.ID && event.EventType == "work_item.completed" {
								completed++
							}
						}
						if completed != 1 {
							return 0, fmt.Errorf("signed DONE event count %d", completed)
						}
						return size, nil
					}
					for _, scope := range scopes {
						for i := 0; i < history; i++ {
							_, e := runJourney(createTask(scope), executors[0], reviewers[0])
							must(e)
						}
					}
					tasks := make([][]d.WorkItem, consumers)
					for i := range tasks {
						for j := 0; j < 2; j++ {
							tasks[i] = append(tasks[i], createTask(scopes[i%outcomes]))
						}
					}
					conflicts.Store(0)
					retries.Store(0)
					observer := &loadObserver{}
					service.SetObserver(observer)
					pool := store.db.Stats()
					started := time.Now()
					var wg sync.WaitGroup
					var mu sync.Mutex
					latencies := []float64{}
					maxBytes := 0
					failures := []string{}
					for i := range tasks {
						wg.Add(1)
						go func(i int) {
							defer wg.Done()
							for _, w := range tasks[i] {
								start := time.Now()
								size, e := runJourney(w, executors[i], reviewers[i])
								mu.Lock()
								if e != nil {
									failures = append(failures, e.Error())
								} else {
									latencies = append(latencies, float64(time.Since(start).Microseconds())/1000)
								}
								if size > maxBytes {
									maxBytes = size
								}
								mu.Unlock()
							}
						}(i)
					}
					wg.Wait()
					elapsed := time.Since(started)
					if len(failures) > 0 {
						t.Fatalf("signed workload failures: %v", failures)
					}
					if len(latencies) != consumers*2 {
						t.Fatal("incomplete accepted delivery count")
					}
					if maxBytes > a.MaxSnapshotBytes {
						t.Fatal("focal signed issuance exceeded bound")
					}
					deliveryCount := 0
					stableIDs := map[string]bool{}
					for _, scope := range scopes {
						page, e := service.ListDeliveries(adminCtx, scope, 100, "")
						must(e)
						for _, delivery := range page.Items {
							if stableIDs[delivery.ID] {
								t.Fatal("duplicate stable delivery identity")
							}
							stableIDs[delivery.ID] = true
							deliveryCount++
						}
					}
					if deliveryCount != outcomes*history+consumers*2 {
						t.Fatalf("signed outbox count %d", deliveryCount)
					}
					waits, guards := []float64{}, []float64{}
					for _, v := range observer.items {
						waits = append(waits, float64(v.TransactionWait.Microseconds())/1000)
						guards = append(guards, float64(v.GuardDuration.Microseconds())/1000)
					}
					report := map[string]any{"outcomes": outcomes, "accepted_historical_tasks_per_outcome": history, "consumers": consumers, "accepted_deliveries": len(latencies), "signed_mutations": len(observer.items), "elapsed_ms": float64(elapsed.Microseconds()) / 1000, "accepted_per_second": float64(len(latencies)) / elapsed.Seconds(), "delivery_p50_ms": loadQuantile(latencies, .5), "delivery_p95_ms": loadQuantile(latencies, .95), "delivery_p99_ms": loadQuantile(latencies, .99), "transaction_wait_p95_ms": loadQuantile(waits, .95), "guard_wait_p95_ms": loadQuantile(guards, .95), "pool_wait_ms": float64((store.db.Stats().WaitDuration - pool.WaitDuration).Microseconds()) / 1000, "conflicts": conflicts.Load(), "retries": retries.Load(), "issuance_max_bytes": maxBytes, "protocol": "signed_contracts_v2", "acceptance": "independent_review", "sql_count_available": false, "verified_outbox_deliveries": deliveryCount, "approval_replay_deduplicated": true}
					raw, e := json.Marshal(report)
					must(e)
					t.Log("SIGNED_LOAD " + string(raw))
				})
			}
		}
	}
}

func retrySignedLoadMutation[T any](fn func() (a.MutationResult[T], error), conflicts, retries *atomic.Uint64) (a.MutationResult[T], error) {
	for attempt := 0; ; attempt++ {
		value, err := fn()
		code, _ := d.ErrorCodeOf(err)
		if err == nil || code != d.ErrorCodeTransactionConflict || attempt >= 100 {
			return value, err
		}
		conflicts.Add(1)
		retries.Add(1)
		time.Sleep(time.Duration(5+min(attempt*4, 200)+mathrand.IntN(80)) * time.Millisecond)
	}
}
