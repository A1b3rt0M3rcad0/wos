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
	"strconv"
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
	historySizes, consumerSizes := []int{0, 16}, []int{2, 8}
	if os.Getenv("WOS_SHARED_LOAD") == "1" {
		historySizes = []int{0}
		consumerSizes = []int{2, 8, 50}
	}
	for _, outcomes := range []int{1, 4} {
		for _, history := range historySizes {
			for _, consumers := range consumerSizes {
				t.Run(fmt.Sprintf("outcomes=%d/history=%d/consumers=%d", outcomes, history, consumers), func(t *testing.T) {
					runSignedLoadCase(t, outcomes, history, consumers, 1, 1, false)
				})
			}
		}
	}
}

// Independent Store/Service instances share only the SQL database. This
// measures the application/storage path and per-replica pools, not HTTP, process
// scheduling, ingress or the delivery worker. Transport/failure pilots qualify
// those boundaries separately.
func TestScalableSignedWorkMatrix(t *testing.T) {
	if os.Getenv("WOS_SCALE_QUALIFICATION") != "1" {
		t.Skip("opt-in scalability fixture; set WOS_SCALE_QUALIFICATION=1")
	}
	integer := func(name string, fallback, minimum, maximum int) int {
		t.Helper()
		raw := os.Getenv(name)
		if raw == "" {
			return fallback
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value < minimum || value > maximum {
			t.Fatalf("%s must be %d..%d", name, minimum, maximum)
		}
		return value
	}
	outcomes := integer("WOS_SCALE_OUTCOMES", 4, 1, 16)
	tenants := integer("WOS_SCALE_TENANTS", 1, 1, 16)
	replicas := integer("WOS_SCALE_REPLICAS", 1, 1, 4)
	consumers := integer("WOS_SCALE_CONSUMERS", 4, 2, 50)
	history := integer("WOS_SCALE_HISTORY", 0, 0, 1000)
	if tenants > outcomes || outcomes%tenants != 0 || consumers*2 < outcomes {
		t.Fatal("Outcomes must divide equally across Tenants and all Outcomes must receive timed work")
	}
	t.Run(fmt.Sprintf("tenants=%d/replicas=%d/outcomes=%d/history=%d/consumers=%d", tenants, replicas, outcomes, history, consumers), func(t *testing.T) {
		runSignedLoadCase(t, outcomes, history, consumers, tenants, replicas, true)
	})
}

func runSignedLoadCase(t *testing.T, outcomes, history, consumers, tenantCount, replicaCount int, measuredReads bool) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := sqliteFixedClock{now}
	ids := &loadIDs{}
	path := filepath.Join(t.TempDir(), "signed-load.db")
	store := openTestStore(t, path)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	newID := func() d.ID { id, e := ids.NewID(); must(e); return id }
	sec := a.SecurityService{Store: store, Clock: clock, IDs: ids}
	tenants := []d.ID{}
	admins := map[d.ID]a.Identity{}
	adminContexts := map[d.ID]context.Context{}
	tenantIndexes := map[d.ID]int{}
	for i := 0; i < tenantCount; i++ {
		var ns d.ID
		token := ""
		if i == 0 {
			ns = newID()
			token = "signed-load-owned-bootstrap-token-at-least-32-0"
			must(sec.Bootstrap(ctx, ports.Tenant{ID: ns, Name: "signed measured workload"}, "load-admin", token))
		} else {
			// Bootstrap is one-shot. Provision subsequent tenant scopes with
			// the same public administrative use case available to operators.
			rootNS := tenants[0]
			snapshot, err := sec.AdministrativeSnapshot(adminContexts[rootNS], rootNS)
			must(err)
			result, secret, err := sec.AdministrativeMutation(adminContexts[rootNS], fmt.Sprintf("signed-load-create-tenant-%d", i), ports.SecurityAdminCommand{Operation: "create_tenant", TenantID: rootNS, ExpectedVersion: snapshot.TenantVersion, TenantName: fmt.Sprintf("Measured tenant %d", i)})
			must(err)
			if result.CreatedTenant == nil || secret == "" {
				t.Fatal("Tenant provisioning did not return its initial credential")
			}
			ns = result.CreatedTenant.ID
			token = secret
		}
		admin, err := sec.Authenticate(ctx, token)
		must(err)
		tenants = append(tenants, ns)
		admins[ns] = admin
		adminContexts[ns] = a.WithIdentity(ctx, admin)
		tenantIndexes[ns] = i
	}

	pub, private, e := ed25519.GenerateKey(rand.Reader)
	must(e)
	fp, e := signing.Fingerprint(pub)
	must(e)
	issuer := d.ServerIdentity{ID: newID(), IssuerKeyID: newID(), PublicKey: base64.StdEncoding.EncodeToString(pub), Fingerprint: fp, CreatedAt: now}
	service, e := a.NewService(store, clock, ids)
	must(e)
	must(service.ConfigureSignedIssuer(ctx, acquisitionIssuer{issuer, private}, d.AcceptanceIndependentReview))
	participant := func(ns d.ID, name string, permissions []ports.Permission) signedLoadParticipant {
		must(sec.SetGrant(adminContexts[ns], ports.TenantGrant{TenantID: ns, PrincipalID: name, Permissions: permissions}))
		cred, token, e := sec.IssueCredential(adminContexts[ns], ns, name, d.ActorRef{Kind: d.ActorKindAgent, Provider: "signed-load", ID: name}, now.Add(time.Hour))
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
		must(registry.SaveKey(ctx, d.SigningKey{ID: kid, TenantID: ns, PrincipalID: name, Purpose: "agent", Algorithm: "Ed25519", PublicKey: base64.StdEncoding.EncodeToString(public), Fingerprint: fingerprint, Version: 1, Status: "active", CreatedAt: now, UpdatedAt: now}, 0))
		ops := []string{}
		for _, op := range permissions {
			ops = append(ops, string(op))
		}
		must(registry.SaveCredentialPolicy(ctx, d.CredentialPolicy{TenantID: ns, CredentialID: cred.ID, PrincipalID: name, Version: 1, PermittedOperations: ops, AcceptanceFloor: d.AcceptanceIndependentReview, AllowedSigningKeyIDs: []d.ID{kid}, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1, UpdatedAt: now}, 0))
		must(u.Commit())
		return signedLoadParticipant{identity, a.WithIdentity(ctx, identity), kid, key}
	}
	executors := []signedLoadParticipant{}
	reviewers := []signedLoadParticipant{}
	for _, ns := range tenants {
		for i := 0; i < consumers; i++ {
			executors = append(executors, participant(ns, fmt.Sprintf("executor-%d", i), []ports.Permission{ports.PermissionStateRead, ports.PermissionWorkContractAcquire, ports.PermissionWorkContractReturn, ports.PermissionWorkWrite, ports.PermissionAssessmentWrite, ports.PermissionConclusionWrite, ports.PermissionRecordsWrite}))
			reviewers = append(reviewers, participant(ns, fmt.Sprintf("reviewer-%d", i), []ports.Permission{ports.PermissionStateRead, ports.PermissionWorkReviewAcquire, ports.PermissionWorkReviewDecide, ports.PermissionAssessmentWrite, ports.PermissionConclusionWrite, ports.PermissionRecordsWrite}))
		}
	}
	scopes := []d.Scope{}
	u, e := store.Begin(ctx)
	must(e)
	for i := 0; i < outcomes; i++ {
		ns := tenants[i%tenantCount]
		oid := newID()
		out, e := d.NewOutcome(oid, ns, "Measured signed Outcome", "", "Independent acceptance", d.PriorityNormal, now)
		must(e)
		completion, e := d.NewSuccessCriterion(newID(), out.Ref(), "Explicit Outcome conclusion remains required", "", true, d.VerificationModeAttestation)
		must(e)
		must(out.AddCriterion(completion, now))
		must(out.Activate(now))
		must(u.Outcomes().Insert(ctx, out))
		scopes = append(scopes, d.Scope{TenantID: ns, OutcomeID: oid})
	}
	for _, ns := range tenants {
		protocol := u.(ports.WorkProtocolUnitOfWork).WorkProtocol()
		state, e := protocol.Lock(ctx, ns, true)
		must(e)
		state.Phase = d.WorkProtocolSigned
		state.WriterEpoch = 3
		state.WritersDrained = true
		state.Version++
		state.UpdatedAt = now
		state.UpdatedBy = admins[ns].PrincipalID
		state.Reason = "owned test-only fixture, public cutover separately validated"
		must(protocol.Save(ctx, state, 1))
	}
	must(u.Commit())
	endpoints := map[d.ID]d.ID{}
	for _, ns := range tenants {
		endpoint := ports.WebhookEndpoint{ID: newID(), TenantID: ns, URL: "https://example.invalid/signed-load", SecretRef: "test-only-webhook", KeyID: "load-fixture"}
		must(store.InstallEndpoints(ctx, []ports.WebhookEndpoint{endpoint}))
		endpoints[ns] = endpoint.ID
	}
	for _, scope := range scopes {
		id := newID()
		admin := admins[scope.TenantID]
		_, err := service.ConfigureTrigger(adminContexts[scope.TenantID], d.CommandContext{PrincipalID: admin.PrincipalID, Actor: admin.Actor, CommandID: id, IdempotencyKey: "signed-load-trigger-" + id.String()}, a.ConfigureTriggerCommand{Scope: scope, Name: "Committed signed DONE", EventTypes: []string{"work_item.completed"}, TargetEndpointIDs: []d.ID{endpoints[scope.TenantID]}, SignalType: "test.signed_done"})
		must(err)
	}
	stores := []*Store{store}
	services := []*a.Service{service}
	for i := 1; i < replicaCount; i++ {
		replica := openTestStore(t, path)
		svc, err := a.NewService(replica, clock, ids)
		must(err)
		must(svc.ConfigureSignedIssuer(ctx, acquisitionIssuer{issuer, private}, d.AcceptanceIndependentReview))
		stores = append(stores, replica)
		services = append(services, svc)
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
	runJourney := func(service *a.Service, w d.WorkItem, ex, rev signedLoadParticipant) (issuedBytes int, journeyErr error) {
		stage := "acquire_work"
		defer func() {
			if journeyErr != nil {
				journeyErr = fmt.Errorf("%s tenant=%s: %w", stage, w.Scope.TenantID, journeyErr)
			}
		}()
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
		ret := signing.WorkReturnPayload[a.SignedReturnMaterial]{RequestBinding: signing.RequestBinding{Binding: signing.Binding{ProtocolVersion: 3, ServerID: issuer.ID.String(), TenantID: w.Scope.TenantID.String(), OutcomeID: w.Scope.OutcomeID.String(), PrincipalID: ex.identity.PrincipalID, SignerKeyID: ex.key.String()}, OperationKind: "work_return", RequestID: cc.CommandID.String(), IdempotencyKey: cc.IdempotencyKey, ContractID: c.ID.String(), WorkItemID: work.ID.String(), ExecutionID: c.ExecutionID.String(), FencingToken: signing.Decimal(c.FencingToken), SpecDigest: c.SignedBinding.SpecificationDigest, AuthorityDigest: signing.Digest(acq.Value.IssuedAuthority.Payload), ExpectedContractVersion: signing.Decimal(c.Version), ExpectedLeaseVersion: signing.Decimal(c.LeaseVersion), ExpectedWorkItemVersion: signing.Decimal(work.Version), PolicyRevision: 1}, CompletionIntent: "review", Material: a.SignedReturnMaterial{Result: d.NewSignedResultMaterial(d.WorkResultMaterial{ContractID: c.ID, WorkItemID: work.ID, SpecDigest: c.SignedBinding.SpecificationDigest, Summary: "load fixture submitted original bounded obligation"}), Reason: "explicit independent review"}}
		env, e := signing.Sign(signing.WorkReturn, ret, ex.key.String(), ex.private)
		if e != nil {
			return 0, e
		}
		stage = "return_work"
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
		stage = "audit_events_before_review"
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
		stage = "acquire_review"
		assigned, e := retrySignedLoadMutation(func() (a.MutationResult[a.SignedReviewContractResult], error) {
			return service.AcquireSignedReviewContract(rev.ctx, cc, a.AcquireSignedReviewContractCommand{Scope: w.Scope, ReviewCaseID: reviewID, ExpectedReviewCaseVersion: 1, SignerKeyID: rev.key, TTLSeconds: 300})
		}, &conflicts, &retries)
		if e != nil {
			return 0, e
		}
		rc := assigned.Value.Contract
		criterion := work.Criteria.Items[0]
		cc = next(rev)
		decision := signing.ReviewReturnPayload[a.SignedReviewMaterial]{RequestBinding: signing.RequestBinding{Binding: signing.Binding{ProtocolVersion: 3, ServerID: issuer.ID.String(), TenantID: w.Scope.TenantID.String(), OutcomeID: w.Scope.OutcomeID.String(), PrincipalID: rev.identity.PrincipalID, SignerKeyID: rev.key.String()}, OperationKind: "review_return", RequestID: cc.CommandID.String(), IdempotencyKey: cc.IdempotencyKey, ContractID: rc.ID.String(), WorkItemID: rc.WorkItemID.String(), ExecutionID: rc.ExecutionID.String(), FencingToken: signing.Decimal(rc.FencingToken), SpecDigest: rc.Binding.SpecificationDigest, AuthorityDigest: signing.Digest(assigned.Value.IssuedAuthority.Payload), ExpectedContractVersion: signing.Decimal(rc.Version), ExpectedLeaseVersion: signing.Decimal(rc.LeaseVersion), ExpectedWorkItemVersion: assigned.Value.WorkItemVersion, PolicyRevision: 1}, ExpectedReviewCaseVersion: signing.Decimal(assigned.Value.Case.Version), ReviewCaseID: rc.CaseID.String(), SubmissionID: rc.SubmissionID.String(), SubmissionDigest: rc.SubmissionDigest, Decision: "approved", Material: a.SignedReviewMaterial{Reason: "independent fixture assessment of original attestation criterion", Assessments: []a.SignedAssessmentInput{{CriterionID: criterion.ID, CriterionRevision: signing.Decimal(criterion.Revision), Result: d.AssessmentResultMet, Rationale: "verified original fixture requirement"}}}}
		env, e = signing.Sign(signing.ReviewReturn, decision, rev.key.String(), rev.private)
		if e != nil {
			return 0, e
		}
		stage = "return_review"
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
		stage = "replay_review"
		replay, e := retrySignedLoadMutation(func() (a.MutationResult[a.SignedReturnResult], error) {
			return service.ReturnSignedReview(rev.ctx, cc, a.ReturnSignedReviewCommand{Envelope: env})
		}, &conflicts, &retries)
		if e != nil {
			return 0, e
		}
		if !replay.IdempotentReplay || string(replay.Value.Receipt.Payload) != string(approved.Value.Receipt.Payload) {
			return 0, fmt.Errorf("signed decision replay changed original receipt")
		}
		stage = "audit_events_after_review"
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
			_, e := runJourney(service, createTask(scope), executors[tenantIndexes[scope.TenantID]*consumers], reviewers[tenantIndexes[scope.TenantID]*consumers])
			must(e)
		}
	}
	// Every consumer collides on one exact expected version. Only
	// transaction conflicts retry the unchanged command; semantic
	// version conflicts never advance the caller's expectation.
	collision := createTask(scopes[0])
	var collisionWinners, versionRejections, collisionConflicts, collisionRetries atomic.Uint64
	var collisionWG sync.WaitGroup
	collisionStart := make(chan struct{})
	collisionErrors := make(chan error, consumers)
	for i := range consumers {
		collisionWG.Add(1)
		go func(i int) {
			defer collisionWG.Done()
			ex := executors[i]
			cc := next(ex)
			cmd := a.AcquireSignedWorkContractCommand{Scope: collision.Scope, WorkItemID: collision.ID, ExpectedWorkItemVersion: collision.Version, SignerKeyID: ex.key, TTLSeconds: 300}
			<-collisionStart
			_, e := retrySignedLoadMutation(func() (a.MutationResult[a.WorkContractResult], error) {
				return services[i%replicaCount].AcquireSignedWorkContract(ex.ctx, cc, cmd)
			}, &collisionConflicts, &collisionRetries)
			if e == nil {
				collisionWinners.Add(1)
			} else if code, _ := d.ErrorCodeOf(e); code == d.ErrorCodeVersionConflict {
				versionRejections.Add(1)
			} else {
				collisionErrors <- e
			}
		}(i)
	}
	close(collisionStart)
	collisionWG.Wait()
	close(collisionErrors)
	for e := range collisionErrors {
		must(e)
	}
	if collisionWinners.Load() != 1 || versionRejections.Load() != uint64(consumers-1) {
		t.Fatal("version collision did not retain one winner", collisionWinners.Load(), versionRejections.Load())
	}
	tasks := make([][]d.WorkItem, consumers)
	for i := range tasks {
		for j := 0; j < 2; j++ {
			tasks[i] = append(tasks[i], createTask(scopes[(i*2+j)%outcomes]))
		}
	}
	activeScopes := map[d.Scope]bool{}
	for _, group := range tasks {
		for _, task := range group {
			activeScopes[task.Scope] = true
		}
	}
	if len(activeScopes) != outcomes {
		t.Fatal("declared Outcome count was not exercised", len(activeScopes), outcomes)
	}
	conflicts.Store(0)
	retries.Store(0)
	observer := &loadObserver{}
	for _, svc := range services {
		svc.SetObserver(observer)
	}
	poolWaitBefore := time.Duration(0)
	poolWaitCountBefore := int64(0)
	for _, replica := range stores {
		stats := replica.db.Stats()
		poolWaitBefore += stats.WaitDuration
		poolWaitCountBefore += stats.WaitCount
	}
	started := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	latencies := []float64{}
	readLatencies := []float64{}
	readFailures := []string{}
	var readConflicts atomic.Uint64
	maxReadBytes := 0
	stopReads := make(chan struct{})
	var readWG sync.WaitGroup
	if measuredReads {
		for i := 0; i < consumers; i++ {
			readWG.Add(1)
			go func(i int) {
				defer readWG.Done()
				scope := scopes[i%len(scopes)]
				reader := executors[tenantIndexes[scope.TenantID]*consumers+i]
				for {
					select {
					case <-stopReads:
						return
					default:
					}
					began := time.Now()
					value, err := services[i%replicaCount].GetContinuitySection(reader.ctx, scope, "work_items", 25, "")
					raw, marshalErr := json.Marshal(value)
					mu.Lock()
					if code, _ := d.ErrorCodeOf(err); code == d.ErrorCodeTransactionConflict {
						readConflicts.Add(1)
					} else if err != nil {
						readFailures = append(readFailures, err.Error())
					} else if marshalErr != nil || len(raw) > a.MaxSnapshotBytes {
						readFailures = append(readFailures, "read response bound")
					} else {
						readLatencies = append(readLatencies, float64(time.Since(began).Microseconds())/1000)
						maxReadBytes = max(maxReadBytes, len(raw))
					}
					mu.Unlock()
					select {
					case <-stopReads:
						return
					case <-time.After(10 * time.Millisecond):
					}
				}
			}(i)
		}
	}
	maxBytes := 0
	failures := []string{}
	for i := range tasks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for _, w := range tasks[i] {
				start := time.Now()
				participantIndex := tenantIndexes[w.Scope.TenantID]*consumers + i
				size, e := runJourney(services[i%replicaCount], w, executors[participantIndex], reviewers[participantIndex])
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
	close(stopReads)
	readWG.Wait()
	if len(readFailures) > 0 {
		t.Fatalf("concurrent read failures: %v", readFailures)
	}
	if measuredReads && len(readLatencies) == 0 {
		t.Fatal("concurrent reads were not exercised")
	}
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
		cursor := ""
		for {
			page, e := service.ListDeliveries(adminContexts[scope.TenantID], scope, 100, cursor)
			must(e)
			for _, delivery := range page.Items {
				if stableIDs[delivery.ID] {
					t.Fatal("duplicate stable delivery identity")
				}
				stableIDs[delivery.ID] = true
				deliveryCount++
			}
			if page.NextCursor == "" {
				break
			}
			cursor = page.NextCursor
		}
	}
	if deliveryCount != outcomes*history+consumers*2 {
		t.Fatalf("signed outbox count %d", deliveryCount)
	}

	poolWait := time.Duration(0)
	poolWaitCount := int64(0)
	for _, replica := range stores {
		stats := replica.db.Stats()
		poolWait += stats.WaitDuration
		poolWaitCount += stats.WaitCount
	}
	poolWait -= poolWaitBefore
	poolWaitCount -= poolWaitCountBefore
	recoveryExperiment := !measuredReads && outcomes == 1 && history == 0 && consumers == 2
	if recoveryExperiment {
		claimAt := time.Now().UTC()
		first, e := store.ClaimDelivery(ctx, claimAt, newID(), 30*time.Second)
		must(e)
		if first == nil {
			t.Fatal("signed DONE signal missing before worker crash")
		}
		// Other deliveries are acknowledged; this one models a worker
		// dying after its durable claim and before acknowledgement.
		for {
			next, e := store.ClaimDelivery(ctx, claimAt, newID(), 30*time.Second)
			must(e)
			if next == nil {
				break
			}
			must(store.FinishDelivery(ctx, *next, claimAt, "http_200", true, claimAt))
		}
		must(store.Close())
		store = openTestStore(t, path)
		afterExpiry := claimAt.Add(31 * time.Second)
		replacement, e := store.ClaimDelivery(ctx, afterExpiry, newID(), 30*time.Second)
		must(e)
		if replacement == nil || replacement.ID != first.ID || string(replacement.Body) != string(first.Body) || replacement.FencingToken <= first.FencingToken {
			t.Fatal("restart lost exact signed completion signal or worker fencing")
		}
		if e := store.FinishDelivery(ctx, *first, afterExpiry, "http_200", true, afterExpiry); e == nil {
			t.Fatal("crashed worker acknowledged with stale fence")
		}
		must(store.FinishDelivery(ctx, *replacement, afterExpiry, "http_200", true, afterExpiry))
		pending, e := store.ClaimDelivery(ctx, afterExpiry, newID(), 30*time.Second)
		must(e)
		if pending != nil {
			t.Fatal("completion replay/recovery duplicated signal")
		}
	}
	waits, guards := []float64{}, []float64{}
	for _, v := range observer.items {
		waits = append(waits, float64(v.TransactionWait.Microseconds())/1000)
		guards = append(guards, float64(v.GuardDuration.Microseconds())/1000)
	}
	report := map[string]any{"tenants": tenantCount, "service_replicas": replicaCount, "concurrent_reads": len(readLatencies), "read_transaction_conflicts": readConflicts.Load(), "read_p50_ms": loadQuantile(readLatencies, .5), "read_p95_ms": loadQuantile(readLatencies, .95), "read_p99_ms": loadQuantile(readLatencies, .99), "read_max_bytes": maxReadBytes, "pool_wait_count": poolWaitCount, "transport_measured": false, "outcomes": outcomes, "active_outcomes": len(activeScopes), "accepted_historical_tasks_per_outcome": history, "consumers": consumers, "accepted_deliveries": len(latencies), "signed_mutations": len(observer.items), "elapsed_ms": float64(elapsed.Microseconds()) / 1000, "accepted_per_second": float64(len(latencies)) / elapsed.Seconds(), "delivery_p50_ms": loadQuantile(latencies, .5), "delivery_p95_ms": loadQuantile(latencies, .95), "delivery_p99_ms": loadQuantile(latencies, .99), "transaction_wait_p95_ms": loadQuantile(waits, .95), "guard_wait_p95_ms": loadQuantile(guards, .95), "pool_wait_ms": float64(poolWait.Microseconds()) / 1000, "conflicts": conflicts.Load(), "retries": retries.Load(), "issuance_max_bytes": maxBytes, "protocol": "signed_contracts_v3", "acceptance": "independent_review", "sql_count_available": false, "verified_outbox_deliveries": deliveryCount, "approval_replay_deduplicated": true, "worker_recovery_experiment": recoveryExperiment, "version_collision_winners": collisionWinners.Load(), "version_collision_rejections": versionRejections.Load(), "collision_transaction_conflicts": collisionConflicts.Load(), "collision_exact_retries": collisionRetries.Load()}
	raw, e := json.Marshal(report)
	must(e)
	t.Log("SIGNED_LOAD " + string(raw))
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
